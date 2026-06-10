<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '../stores/config'
import { useLogsStore } from '../stores/logs'
import { PickDirectory } from '../../wailsjs/go/api/App'
import { config } from '../../wailsjs/go/models'

const { t } = useI18n()
const emit = defineEmits<{ (e: 'close'): void }>()
const configStore = useConfigStore()
const logsStore = useLogsStore()

// All edits go into a deep copy; the store is only updated on Save, so closing
// the panel any other way discards the changes.
const draft = ref<config.Config | null>(null)

onMounted(async () => {
  if (!configStore.cfg) await configStore.load()
  if (configStore.cfg) {
    draft.value = config.Config.createFrom(JSON.parse(JSON.stringify(configStore.cfg)))
  }
})

const markerCategories: Array<'speech' | 'emote' | 'ooc'> = ['speech', 'emote', 'ooc']

// Mention names edited as one-per-line text.
const mentionsText = computed<string>({
  get: () => draft.value?.mention_names.join('\n') ?? '',
  set: (v: string) => {
    if (draft.value) {
      draft.value.mention_names = v
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
    }
  },
})

// Roleplay characters are picked from the indexed player list via a small
// autocomplete (same filtering as PlayerFilter, minus keyboard list nav).
const rpInput = ref('')
const rpOpen = ref(false)
const rpInputEl = ref<HTMLInputElement | null>(null)
const rpSuggestions = computed(() => {
  const d = draft.value
  if (!d) return []
  const q = rpInput.value.trim().toLowerCase()
  const pool = logsStore.allPlayers.filter((p) => !d.roleplay_characters.includes(p))
  return q ? pool.filter((p) => p.toLowerCase().includes(q)) : pool
})

function addCharacter(name: string) {
  draft.value?.roleplay_characters.push(name)
  rpInput.value = ''
  rpOpen.value = false
  // Same as PlayerFilter: the input keeps focus after a suggestion click, and
  // re-clicking a focused input would never reopen the dropdown.
  rpInputEl.value?.blur()
}

function addCharacterFromInput() {
  const list = rpSuggestions.value
  const pick = list.find((p) => p.toLowerCase() === rpInput.value.trim().toLowerCase()) ?? list[0]
  if (pick) addCharacter(pick)
}

function removeCharacter(name: string) {
  if (draft.value) {
    draft.value.roleplay_characters = draft.value.roleplay_characters.filter((n) => n !== name)
  }
}

function addPair(key: 'speech' | 'emote' | 'ooc') {
  draft.value?.markers[key].push({ open: '', close: '' } as any)
}
function removePair(key: 'speech' | 'emote' | 'ooc', i: number) {
  draft.value?.markers[key].splice(i, 1)
}

async function addDirectory() {
  const dir = await PickDirectory()
  if (dir && draft.value && !draft.value.log_directories.includes(dir)) {
    draft.value.log_directories.push(dir)
  }
}

function removeDirectory(dir: string) {
  if (draft.value) {
    draft.value.log_directories = draft.value.log_directories.filter((d) => d !== dir)
  }
}

async function save() {
  if (!draft.value) return
  // A pair missing either delimiter can't delimit anything; drop it instead of
  // letting an empty close match at position 0 in the highlighter.
  for (const cat of markerCategories) {
    draft.value.markers[cat] = draft.value.markers[cat].filter(
      (p) => p.open !== '' && p.close !== '',
    )
  }
  configStore.cfg = draft.value
  await configStore.save()
  emit('close')
}
</script>

<template>
  <div class="settings-backdrop" @click.self="emit('close')">
    <div class="settings-panel">
      <header class="settings-header">
        <strong>{{ t('settings.title') }}</strong>
        <button class="ghost" @click="emit('close')">✕</button>
      </header>

      <div v-if="!draft" class="placeholder">{{ t('viewer.loading') }}</div>

      <div v-else class="settings-body">
        <!-- Directories -->
        <section>
          <h3>{{ t('settings.logDirs') }}</h3>
          <label class="check">
            <input type="checkbox" v-model="draft.auto_detect_appdata" />
            {{ t('settings.autoDetect') }}
          </label>
          <ul class="dir-list">
            <li v-for="d in draft.log_directories" :key="d">
              <span class="dir-path">{{ d }}</span>
              <button class="ghost" @click="removeDirectory(d)">{{ t('settings.remove') }}</button>
            </li>
            <li v-if="draft.log_directories.length === 0" class="muted">
              {{ t('settings.noDirs') }}
            </li>
          </ul>
          <button @click="addDirectory()">{{ t('settings.addDir') }}</button>
        </section>

        <!-- Appearance & language -->
        <section class="grid-2">
          <div>
            <h3>{{ t('settings.theme') }}</h3>
            <select v-model="draft.theme">
              <option value="dark">{{ t('settings.dark') }}</option>
              <option value="light">{{ t('settings.light') }}</option>
            </select>
          </div>
          <div>
            <h3>{{ t('settings.language') }}</h3>
            <select v-model="draft.language">
              <option value="en">English</option>
              <option value="de">Deutsch</option>
            </select>
          </div>
        </section>

        <!-- Roleplay characters -->
        <section>
          <h3>{{ t('settings.rpCharacters') }}</h3>
          <p class="muted">{{ t('settings.rpCharactersHint') }}</p>
          <ul class="dir-list">
            <li v-for="name in draft.roleplay_characters" :key="name">
              <span class="dir-path">
                {{ name }}
                <span
                  v-if="logsStore.allPlayers.length > 0 && !logsStore.allPlayers.includes(name)"
                  class="muted"
                  >⚠ {{ t('settings.rpCharacterNotFound') }}</span
                >
              </span>
              <button class="ghost" @click="removeCharacter(name)">{{ t('settings.remove') }}</button>
            </li>
          </ul>
          <div class="suggest-wrap">
            <input
              ref="rpInputEl"
              v-model="rpInput"
              class="player-input"
              :placeholder="t('settings.rpCharactersPlaceholder')"
              @focus="rpOpen = true"
              @input="rpOpen = true"
              @keydown.enter.prevent="addCharacterFromInput()"
              @keydown.esc="rpOpen = false"
              @blur="rpOpen = false"
            />
            <ul v-if="rpOpen && rpSuggestions.length" class="suggest-list">
              <li v-for="p in rpSuggestions" :key="p" @mousedown.prevent="addCharacter(p)">
                {{ p }}
              </li>
            </ul>
          </div>
        </section>

        <!-- Mentions -->
        <section>
          <h3>{{ t('settings.mentions') }}</h3>
          <textarea
            v-model="mentionsText"
            class="mentions"
            rows="3"
            :placeholder="t('settings.mentionsPlaceholder')"
          ></textarea>
        </section>

        <!-- RP markers -->
        <section>
          <h3>{{ t('settings.markers') }}</h3>
          <p class="muted">{{ t('settings.markersHint') }}</p>
          <div v-for="cat in markerCategories" :key="cat" class="marker-group">
            <div class="marker-group-head">
              <span>{{ t('settings.' + cat) }}</span>
              <button class="ghost" @click="addPair(cat)">{{ t('settings.add') }}</button>
            </div>
            <div
              v-for="(pair, i) in draft.markers[cat]"
              :key="i"
              class="marker-row"
            >
              <input v-model="pair.open" class="marker-input" :placeholder="t('settings.open')" />
              <input v-model="pair.close" class="marker-input" :placeholder="t('settings.close')" />
              <button class="ghost" @click="removePair(cat, i)">✕</button>
            </div>
          </div>
        </section>
      </div>

      <footer class="settings-footer">
        <button class="ghost" @click="emit('close')">{{ t('settings.cancel') }}</button>
        <button :disabled="configStore.saving || !draft" @click="save">
          {{ configStore.saving ? t('settings.saving') : t('settings.save') }}
        </button>
      </footer>
    </div>
  </div>
</template>
