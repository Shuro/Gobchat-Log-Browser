import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetConfig, SaveConfig, GetLocaleMessages } from '../../wailsjs/go/api/App'
import type { config } from '../../wailsjs/go/models'
import { useLogsStore } from './logs'
import { applyTheme } from '../composables/theme'
import { setLocale, mergeBackend } from '../i18n'

async function applyLocale(lang: string) {
  setLocale(lang)
  try {
    mergeBackend(lang, await GetLocaleMessages())
  } catch {
    // Backend strings are optional; UI keeps its own translations.
  }
}

// The config store mirrors the backend Config and persists changes. Saving
// re-applies the theme, refreshes the log list (directories may have changed),
// and reloads the open log so new markers / mention names re-highlight.
export const useConfigStore = defineStore('config', () => {
  const cfg = ref<config.Config | null>(null)
  const loading = ref(false)
  const saving = ref(false)

  async function load() {
    loading.value = true
    try {
      cfg.value = await GetConfig()
      applyTheme(cfg.value.theme, cfg.value.colors)
      await applyLocale(cfg.value.language)
    } finally {
      loading.value = false
    }
  }

  async function save() {
    if (!cfg.value) return
    saving.value = true
    try {
      await SaveConfig(cfg.value)
      applyTheme(cfg.value.theme, cfg.value.colors)
      await applyLocale(cfg.value.language)
      const logs = useLogsStore()
      await logs.refreshList()
      await logs.reloadCurrent()
    } finally {
      saving.value = false
    }
  }

  return { cfg, loading, saving, load, save }
})
