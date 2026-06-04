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

onMounted(async () => {
  if (!config.cfg) await config.load()
  if (config.cfg) {
    language.value = config.cfg.language || 'en'
    theme.value = config.cfg.theme || 'dark'
  }
})

// Live previews while choosing.
watch(theme, (val) => applyTheme(val))
watch(language, (val) => {
  locale.value = val === 'de' ? 'de' : 'en'
})

const hasLogDir = computed(
  () => (useDetected.value && props.state.default_log_dir_exists) || chosenDir.value !== '',
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

      <footer class="wizard-footer">
        <button :disabled="!hasLogDir || config.saving" @click="finish">
          {{ config.saving ? t('setup.saving') : t('setup.getStarted') }}
        </button>
      </footer>
    </div>
  </div>
</template>
