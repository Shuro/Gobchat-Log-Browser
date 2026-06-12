<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '../stores/config'
import { useLogsStore } from '../stores/logs'
import { CheckForUpdate, GetVersion, PickDirectory } from '../../wailsjs/go/api/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { config } from '../../wailsjs/go/models'
import {
  DEFAULT_COLORS,
  normalizeTheme,
  type ColorCategory,
  type ThemeName,
} from '../composables/theme'

const GITHUB_URL = 'https://github.com/Shuro/Gobchat-Log-Browser'

const { t } = useI18n()
const emit = defineEmits<{ (e: 'close'): void }>()
const configStore = useConfigStore()
const logsStore = useLogsStore()

// All edits go into a deep copy; the store is only updated on Save, so closing
// the panel any other way discards the changes.
const draft = ref<config.Config | null>(null)

const appVersion = ref('')

// Sections are split across tabs; v-show keeps tab-local state (update-check
// result, autocomplete input) alive while switching.
const activeTab = ref<'general' | 'design' | 'roleplay'>('general')

onMounted(async () => {
  appVersion.value = await GetVersion()
  if (!configStore.cfg) await configStore.load()
  if (configStore.cfg) {
    draft.value = config.Config.createFrom(JSON.parse(JSON.stringify(configStore.cfg)))
  }
})

const markerCategories: Array<'speech' | 'emote' | 'ooc'> = ['speech', 'emote', 'ooc']

// Highlighter names edited as one comma-separated line; whitespace around the
// names is trimmed, empty pieces dropped.
const mentionsText = computed<string>({
  get: () => draft.value?.mention_names.join(', ') ?? '',
  set: (v: string) => {
    if (draft.value) {
      draft.value.mention_names = v
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    }
  },
})

// --- Color overrides (Design tab) -----------------------------------------
// Pickers edit the overrides of the theme currently selected in the draft;
// an absent entry means "theme default" and disables the reset button.
const colorCategories: ColorCategory[] = ['speech', 'emote', 'ooc', 'mention-fg', 'mention-bg']

const draftTheme = computed<ThemeName>(() => normalizeTheme(draft.value?.theme))

function colorValue(cat: ColorCategory): string {
  return draft.value?.colors?.[draftTheme.value]?.[cat] ?? DEFAULT_COLORS[draftTheme.value][cat]
}

function hasOverride(cat: ColorCategory): boolean {
  return Boolean(draft.value?.colors?.[draftTheme.value]?.[cat])
}

function setColor(cat: ColorCategory, value: string) {
  const d = draft.value
  if (!d) return
  // colors arrives as null from a Go nil map; build the nesting lazily.
  if (!d.colors) d.colors = {}
  if (!d.colors[draftTheme.value]) d.colors[draftTheme.value] = {}
  d.colors[draftTheme.value][cat] = value
}

function resetColor(cat: ColorCategory) {
  const themeColors = draft.value?.colors?.[draftTheme.value]
  if (themeColors) delete themeColors[cat]
}

function colorLabel(cat: ColorCategory): string {
  switch (cat) {
    case 'mention-fg':
      return `${t('settings.highlighter')} (${t('settings.colorText')})`
    case 'mention-bg':
      return `${t('settings.highlighter')} (${t('settings.colorBackground')})`
    default:
      return t('settings.' + cat)
  }
}

// Preview base colors per theme; must match --bg-panel / --color-plain in
// style.css. Inline values so previewing the non-active theme looks right.
const previewBase: Record<ThemeName, { bg: string; fg: string }> = {
  dark: { bg: '#202c40', fg: '#d7deea' },
  light: { bg: '#e9edf4', fg: '#1b2330' },
}

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

// Manual update check: acts immediately, independent of the draft/save flow.
const updateState = ref<'idle' | 'checking' | 'uptodate' | 'available' | 'dev' | 'error'>('idle')
const latestVersion = ref('')
const releaseUrl = ref('')

async function checkForUpdates() {
  updateState.value = 'checking'
  try {
    const res = await CheckForUpdate()
    if (res.status === 'update_available') {
      latestVersion.value = res.latest_version
      releaseUrl.value = res.release_url
      updateState.value = 'available'
    } else if (res.status === 'dev') {
      updateState.value = 'dev'
    } else {
      updateState.value = 'uptodate'
    }
  } catch {
    updateState.value = 'error'
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

      <nav class="settings-tabs">
        <button :class="{ active: activeTab === 'general' }" @click="activeTab = 'general'">
          {{ t('settings.tabGeneral') }}
        </button>
        <button :class="{ active: activeTab === 'design' }" @click="activeTab = 'design'">
          {{ t('settings.tabDesign') }}
        </button>
        <button :class="{ active: activeTab === 'roleplay' }" @click="activeTab = 'roleplay'">
          {{ t('settings.tabRoleplay') }}
        </button>
      </nav>

      <div v-if="!draft" class="placeholder">{{ t('viewer.loading') }}</div>

      <div v-else class="settings-body">
        <!-- ============================== General ============================== -->
        <div v-show="activeTab === 'general'">
          <!-- Language -->
          <section>
            <h3>{{ t('settings.language') }}</h3>
            <select v-model="draft.language">
              <option value="en">English</option>
              <option value="de">Deutsch</option>
            </select>
          </section>

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

          <!-- About -->
          <section>
            <h3>{{ t('settings.about') }}</h3>
            <p class="muted">
              {{ t('app.title') }} — {{ t('settings.version', { version: appVersion }) }}
            </p>
            <label class="check">
              <input type="checkbox" v-model="draft.check_updates_on_start" />
              {{ t('settings.checkUpdatesOnStart') }}
            </label>
            <div class="about-actions">
              <button class="ghost" @click="BrowserOpenURL(GITHUB_URL)">
                {{ t('settings.github') }}
              </button>
              <button :disabled="updateState === 'checking'" @click="checkForUpdates()">
                {{ updateState === 'checking' ? t('settings.checking') : t('settings.checkNow') }}
              </button>
            </div>
            <p v-if="updateState === 'uptodate'" class="muted">{{ t('settings.upToDate') }}</p>
            <p v-else-if="updateState === 'dev'" class="muted">{{ t('settings.devBuild') }}</p>
            <p v-else-if="updateState === 'error'" class="muted">{{ t('settings.updateError') }}</p>
            <p v-else-if="updateState === 'available'">
              {{ t('settings.updateAvailable', { version: latestVersion }) }}
              <button class="ghost" @click="BrowserOpenURL(releaseUrl)">
                {{ t('settings.openRelease') }}
              </button>
            </p>
          </section>
        </div>

        <!-- ============================== Design =============================== -->
        <div v-show="activeTab === 'design'">
          <section>
            <h3>{{ t('settings.theme') }}</h3>
            <select v-model="draft.theme">
              <option value="dark">{{ t('settings.dark') }}</option>
              <option value="light">{{ t('settings.light') }}</option>
            </select>
          </section>

          <section>
            <h3>{{ t('settings.colors') }}</h3>
            <div v-for="cat in colorCategories" :key="cat" class="color-row">
              <span class="color-label">{{ colorLabel(cat) }}</span>
              <input
                type="color"
                :value="colorValue(cat)"
                @input="setColor(cat, ($event.target as HTMLInputElement).value)"
              />
              <button class="ghost" :disabled="!hasOverride(cat)" @click="resetColor(cat)">
                {{ t('settings.resetDefault') }}
              </button>
            </div>

            <h3>{{ t('settings.preview') }}</h3>
            <div
              class="color-preview"
              :style="{ background: previewBase[draftTheme].bg, color: previewBase[draftTheme].fg }"
            >
              <span :style="{ color: colorValue('speech') }">{{ t('settings.previewSpeech') }}</span>
              {{ ' ' }}
              <span :style="{ color: colorValue('emote'), fontStyle: 'italic' }">{{
                t('settings.previewEmote')
              }}</span>
              {{ ' ' }}
              <span :style="{ color: colorValue('ooc') }">{{ t('settings.previewOoc') }}</span>
              {{ ' ' }}
              <span
                :style="{
                  color: colorValue('mention-fg'),
                  background: colorValue('mention-bg'),
                  borderRadius: '3px',
                  padding: '0 2px',
                  fontWeight: 700,
                }"
                >{{ t('settings.previewMention') }}</span
              >
            </div>
          </section>
        </div>

        <!-- ============================= Roleplay ============================== -->
        <div v-show="activeTab === 'roleplay'">
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

          <!-- Highlighter (mention names) -->
          <section>
            <h3>{{ t('settings.highlighter') }}</h3>
            <input
              v-model="mentionsText"
              class="mentions"
              type="text"
              :placeholder="t('settings.highlighterPlaceholder')"
            />
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
