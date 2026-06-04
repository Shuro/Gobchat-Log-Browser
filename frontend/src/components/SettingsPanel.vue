<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '../stores/config'

const { t } = useI18n()
const emit = defineEmits<{ (e: 'close'): void }>()
const config = useConfigStore()

onMounted(() => {
  if (!config.cfg) config.load()
})

const markerCategories: Array<'speech' | 'emote' | 'ooc'> = ['speech', 'emote', 'ooc']

// Mention names edited as one-per-line text.
const mentionsText = computed<string>({
  get: () => config.cfg?.mention_names.join('\n') ?? '',
  set: (v: string) => {
    if (config.cfg) {
      config.cfg.mention_names = v
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
    }
  },
})

function addPair(key: 'speech' | 'emote' | 'ooc') {
  config.cfg?.markers[key].push({ open: '', close: '' } as any)
}
function removePair(key: 'speech' | 'emote' | 'ooc', i: number) {
  config.cfg?.markers[key].splice(i, 1)
}

async function save() {
  await config.save()
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

      <div v-if="!config.cfg" class="placeholder">{{ t('viewer.loading') }}</div>

      <div v-else class="settings-body">
        <!-- Directories -->
        <section>
          <h3>{{ t('settings.logDirs') }}</h3>
          <label class="check">
            <input type="checkbox" v-model="config.cfg.auto_detect_appdata" />
            {{ t('settings.autoDetect') }}
          </label>
          <ul class="dir-list">
            <li v-for="d in config.cfg.log_directories" :key="d">
              <span class="dir-path">{{ d }}</span>
              <button class="ghost" @click="config.removeDirectory(d)">{{ t('settings.remove') }}</button>
            </li>
            <li v-if="config.cfg.log_directories.length === 0" class="muted">
              {{ t('settings.noDirs') }}
            </li>
          </ul>
          <button @click="config.addDirectory()">{{ t('settings.addDir') }}</button>
        </section>

        <!-- Appearance & language -->
        <section class="grid-2">
          <div>
            <h3>{{ t('settings.theme') }}</h3>
            <select v-model="config.cfg.theme">
              <option value="dark">{{ t('settings.dark') }}</option>
              <option value="light">{{ t('settings.light') }}</option>
            </select>
          </div>
          <div>
            <h3>{{ t('settings.language') }}</h3>
            <select v-model="config.cfg.language">
              <option value="en">English</option>
              <option value="de">Deutsch</option>
            </select>
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
              v-for="(pair, i) in config.cfg.markers[cat]"
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
        <button :disabled="config.saving || !config.cfg" @click="save">
          {{ config.saving ? t('settings.saving') : t('settings.save') }}
        </button>
      </footer>
    </div>
  </div>
</template>
