<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { CheckForUpdate, DownloadAndApplyUpdate, GetSetupState } from '../wailsjs/go/api/App'
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
const updating = ref(false)
const updateProgress = ref(0)
const updateFailed = ref(false)
const unsubscribers: Array<() => void> = []

// Startup update check: opt-in, and silent on any failure — being offline must
// never produce an error popup. The manual check in Settings surfaces errors.
// A version skipped via the banner stays silent until a newer one appears.
function checkForUpdateQuietly() {
  if (!config.cfg?.check_updates_on_start) return
  CheckForUpdate()
    .then((res) => {
      if (res.status !== 'update_available') return
      if (localStorage.getItem('update.skipVersion') === res.latest_version) return
      updateNotice.value = res
    })
    .catch(() => {})
}

function skipUpdate() {
  if (updateNotice.value) {
    localStorage.setItem('update.skipVersion', updateNotice.value.latest_version)
    updateNotice.value = null
  }
}

// Download the pending update and restart into it. On success the backend quits
// the app (Velopack applies the update and relaunches us), so this call never
// resolves visibly; a rejection means the download failed and we surface it.
function startUpdate() {
  updating.value = true
  updateFailed.value = false
  updateProgress.value = 0
  DownloadAndApplyUpdate().catch(() => {
    updating.value = false
    updateFailed.value = true
  })
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
  unsubscribers.push(EventsOn('update:progress', (p: number) => (updateProgress.value = p)))
  unsubscribers.push(EventsOn('logs:scanned', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:new', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:removed', () => store.refreshList()))
  unsubscribers.push(
    EventsOn('log:updated', (path: string) => {
      // In-place refresh: keeps view mode, find text, and scroll position
      // while Gobchat is still writing to the open log.
      if (path === store.selectedPath) store.reloadCurrent()
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
      <template v-if="updating">
        <span>{{ t('update.downloading', { percent: updateProgress }) }}</span>
        <progress class="update-progress" :value="updateProgress" max="100"></progress>
      </template>
      <template v-else>
        <span>{{ t('update.available', { version: updateNotice.latest_version }) }}</span>
        <button class="ghost" @click="startUpdate">{{ t('update.install') }}</button>
        <button class="ghost" @click="skipUpdate">{{ t('update.skip') }}</button>
        <span v-if="updateFailed" class="muted">{{ t('update.failed') }}</span>
        <span class="spacer"></span>
        <button class="ghost" :title="t('update.dismiss')" @click="updateNotice = null">✕</button>
      </template>
    </div>
    <main class="app-body">
      <LogList />
      <div class="main-pane">
        <LogViewer />
        <SearchResults v-if="search.ran && search.visible" />
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
