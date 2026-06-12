<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { BrowserOpenURL, EventsOn } from '../wailsjs/runtime/runtime'
import { CheckForUpdate, GetSetupState } from '../wailsjs/go/api/App'
import type { api } from '../wailsjs/go/models'
import { useLogsStore } from './stores/logs'
import { useSearchStore } from './stores/search'
import { useConfigStore } from './stores/config'
import LogList from './components/LogList.vue'
import LogViewer from './components/LogViewer.vue'
import SearchBar from './components/SearchBar.vue'
import SearchResults from './components/SearchResults.vue'
import SettingsPanel from './components/SettingsPanel.vue'
import SetupWizard from './components/SetupWizard.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const store = useLogsStore()
const search = useSearchStore()
const config = useConfigStore()
const showSettings = ref(false)
const setupState = ref<api.SetupState | null>(null)
const updateNotice = ref<api.UpdateCheckResult | null>(null)
const unsubscribers: Array<() => void> = []

// Startup update check: opt-in, and silent on any failure — being offline must
// never produce an error popup. The manual check in Settings surfaces errors.
function checkForUpdateQuietly() {
  if (!config.cfg?.check_updates_on_start) return
  CheckForUpdate()
    .then((res) => {
      if (res.status === 'update_available') updateNotice.value = res
    })
    .catch(() => {})
}

onMounted(async () => {
  // Load config first so the theme is applied immediately.
  await config.load()

  // First-run check: show the setup wizard if there is no config yet, no
  // usable log directory, or the wizard gained new content since completion.
  setupState.value = await GetSetupState()

  // Don't pop a banner behind the wizard; onSetupDone re-checks instead.
  if (!setupState.value?.needs_setup) checkForUpdateQuietly()

  // Subscribe before the first fetch so a fast initial scan that finishes in
  // between cannot slip through unnoticed…
  unsubscribers.push(EventsOn('logs:scanned', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:new', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:removed', () => store.refreshList()))
  unsubscribers.push(
    EventsOn('log:updated', (path: string) => {
      if (path === store.selectedPath) store.openLog(path, null, true)
    }),
  )
  // …then pull whatever the backend's startup scan has ready now.
  store.refreshList()
  store.loadAllTagNames()
})

onUnmounted(() => {
  unsubscribers.forEach((off) => off())
})

function onSetupDone() {
  if (setupState.value) setupState.value.needs_setup = false
  store.refreshList()
  checkForUpdateQuietly()
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>{{ t('app.title') }}</h1>
      <SearchBar />
      <button class="icon-btn settings-btn" :title="t('app.settings')" @click="showSettings = true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.09a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.09a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
      </button>
    </header>
    <div v-if="updateNotice" class="update-banner">
      <span>{{ t('update.available', { version: updateNotice.latest_version }) }}</span>
      <button class="ghost" @click="BrowserOpenURL(updateNotice.release_url)">
        {{ t('update.open') }}
      </button>
      <span class="spacer"></span>
      <button class="ghost" :title="t('update.dismiss')" @click="updateNotice = null">✕</button>
    </div>
    <main class="app-body">
      <LogList />
      <div class="main-pane">
        <LogViewer />
        <SearchResults v-if="search.ran" />
      </div>
    </main>
    <SettingsPanel v-if="showSettings" @close="showSettings = false" />
    <SetupWizard
      v-if="setupState && setupState.needs_setup"
      :state="setupState"
      @done="onSetupDone"
    />
  </div>
</template>
