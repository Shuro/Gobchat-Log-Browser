<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '../stores/config'
import { PickDirectory } from '../../wailsjs/go/api/App'
import type { api } from '../../wailsjs/go/models'
import { applyTheme } from '../composables/theme'

const props = defineProps<{ state: api.SetupState }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const { t, locale } = useI18n()
const config = useConfigStore()

const language = ref('en')
const theme = ref('dark')
const useDetected = ref(props.state.default_log_dir_exists)
const chosenDir = ref('')
// Free text: nothing is scanned yet, so there is no name list to pick from.
const characterName = ref('')
const checkUpdates = ref(false)

onMounted(async () => {
  if (!config.cfg) await config.load()
  if (config.cfg) {
    language.value = config.cfg.language || 'en'
    theme.value = config.cfg.theme || 'dark'
    // Re-shown wizards (version bump) must respect an existing "auto-detect
    // off" choice instead of silently re-enabling it.
    if (props.state.config_exists) {
      useDetected.value = config.cfg.auto_detect_appdata && props.state.default_log_dir_exists
    }
    // The installer's one-shot seed wins on first run; otherwise the existing
    // config value pre-fills (false for true first runs).
    checkUpdates.value = props.state.installer_seed_found
      ? props.state.installer_check_updates
      : config.cfg.check_updates_on_start
  }
})

// Live previews while choosing.
watch(theme, (val) => applyTheme(val))
watch(language, (val) => {
  locale.value = val === 'de' ? 'de' : 'en'
})

// Existing configured directories also count, or a re-shown wizard would lock
// out users whose detected default is gone but who added their own folders.
const hasLogDir = computed(
  () =>
    (useDetected.value && props.state.default_log_dir_exists) ||
    chosenDir.value !== '' ||
    (config.cfg?.log_directories.length ?? 0) > 0,
)

async function pick() {
  const dir = await PickDirectory()
  if (dir) chosenDir.value = dir
}

async function finish() {
  if (!config.cfg) return
  config.cfg.language = language.value
  config.cfg.theme = theme.value
  config.cfg.auto_detect_appdata = useDetected.value
  if (chosenDir.value && !config.cfg.log_directories.includes(chosenDir.value)) {
    config.cfg.log_directories.push(chosenDir.value)
  }
  const name = characterName.value.trim()
  if (name && !config.cfg.roleplay_characters.includes(name)) {
    config.cfg.roleplay_characters.push(name)
  }
  config.cfg.check_updates_on_start = checkUpdates.value
  // Stamp the version delivered by the backend, so Go stays the single source
  // of the current wizard version.
  config.cfg.setup_wizard_version = props.state.wizard_version
  await config.save()
  emit('done')
}
</script>

<template>
  <div class="wizard-backdrop">
    <div class="wizard">
      <h2>{{ t('setup.welcome') }}</h2>
      <p class="muted">{{ t('setup.intro') }}</p>

      <section>
        <h3>{{ t('setup.language') }}</h3>
        <select v-model="language">
          <option value="en">English</option>
          <option value="de">Deutsch</option>
        </select>
      </section>

      <section>
        <h3>{{ t('setup.theme') }}</h3>
        <select v-model="theme">
          <option value="dark">{{ t('settings.dark') }}</option>
          <option value="light">{{ t('settings.light') }}</option>
        </select>
      </section>

      <section>
        <h3>{{ t('setup.logFolder') }}</h3>
        <label v-if="state.default_log_dir_exists" class="check">
          <input type="checkbox" v-model="useDetected" />
          {{ t('setup.useDetected') }}
        </label>
        <p v-if="state.default_log_dir" class="detected-path muted">
          {{ state.default_log_dir }}
          <span v-if="!state.default_log_dir_exists">{{ t('setup.notFound') }}</span>
        </p>

        <div class="chosen">
          <button @click="pick">{{ t('setup.chooseFolder') }}</button>
          <span v-if="chosenDir" class="dir-path">{{ chosenDir }}</span>
        </div>
        <p v-if="!hasLogDir" class="muted hint">{{ t('setup.hint') }}</p>
      </section>

      <section>
        <h3>{{ t('setup.rpCharacter') }}</h3>
        <p class="muted">{{ t('setup.rpCharacterHint') }}</p>
        <input
          v-model="characterName"
          class="player-input"
          :placeholder="t('setup.rpCharacterPlaceholder')"
        />
      </section>

      <section>
        <h3>{{ t('setup.updates') }}</h3>
        <label class="check">
          <input type="checkbox" v-model="checkUpdates" />
          {{ t('setup.checkUpdates') }}
        </label>
        <p class="muted">{{ t('setup.updatesHint') }}</p>
      </section>

      <footer class="wizard-footer">
        <button :disabled="!hasLogDir || config.saving" @click="finish">
          {{ config.saving ? t('setup.saving') : t('setup.getStarted') }}
        </button>
      </footer>
    </div>
  </div>
</template>
