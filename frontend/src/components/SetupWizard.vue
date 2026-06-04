<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useConfigStore } from '../stores/config'
import { PickDirectory } from '../../wailsjs/go/api/App'
import type { api } from '../../wailsjs/go/models'
import { applyTheme } from '../composables/theme'

const props = defineProps<{ state: api.SetupState }>()
const emit = defineEmits<{ (e: 'done'): void }>()

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

// Live theme preview.
watch(theme, (t) => applyTheme(t))

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
      <h2>Welcome to Gobchat Log Browser</h2>
      <p class="muted">Let's set things up. You can change all of this later in Settings.</p>

      <section>
        <h3>Language</h3>
        <select v-model="language">
          <option value="en">English</option>
          <option value="de">Deutsch</option>
        </select>
      </section>

      <section>
        <h3>Theme</h3>
        <select v-model="theme">
          <option value="dark">Dark</option>
          <option value="light">Light</option>
        </select>
      </section>

      <section>
        <h3>Log folder</h3>
        <label v-if="state.default_log_dir_exists" class="check">
          <input type="checkbox" v-model="useDetected" />
          Use the detected Gobchat folder
        </label>
        <p v-if="state.default_log_dir" class="detected-path muted">
          {{ state.default_log_dir }}
          <span v-if="!state.default_log_dir_exists"> (not found)</span>
        </p>

        <div class="chosen">
          <button @click="pick">Choose a folder…</button>
          <span v-if="chosenDir" class="dir-path">{{ chosenDir }}</span>
        </div>
        <p v-if="!hasLogDir" class="muted hint">
          Pick the folder where your Gobchat chat logs are stored.
        </p>
      </section>

      <footer class="wizard-footer">
        <button :disabled="!hasLogDir || config.saving" @click="finish">
          {{ config.saving ? 'Saving…' : 'Get started' }}
        </button>
      </footer>
    </div>
  </div>
</template>
