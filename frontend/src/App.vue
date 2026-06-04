<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { GetSetupState } from '../wailsjs/go/api/App'
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
const unsubscribers: Array<() => void> = []

onMounted(async () => {
  // Load config first so the theme is applied immediately.
  await config.load()

  // First-run check: show the setup wizard if there is no config yet or no
  // usable log directory.
  setupState.value = await GetSetupState()

  // The backend scans on startup; pull whatever is ready now…
  store.refreshList()
  store.loadAllTagNames()
  // …and refresh again when the (async) initial scan finishes.
  unsubscribers.push(EventsOn('logs:scanned', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:new', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:removed', () => store.refreshList()))
  unsubscribers.push(
    EventsOn('log:updated', (path: string) => {
      if (path === store.selectedPath) store.openLog(path, null, true)
    }),
  )
})

onUnmounted(() => {
  unsubscribers.forEach((off) => off())
})

function onSetupDone() {
  if (setupState.value) setupState.value.needs_setup = false
  store.refreshList()
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>{{ t('app.title') }}</h1>
      <SearchBar />
      <button class="icon-btn" :title="t('app.settings')" @click="showSettings = true">⚙</button>
    </header>
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
