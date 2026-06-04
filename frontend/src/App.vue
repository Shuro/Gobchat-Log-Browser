<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { useLogsStore } from './stores/logs'
import { useSearchStore } from './stores/search'
import LogList from './components/LogList.vue'
import LogViewer from './components/LogViewer.vue'
import SearchBar from './components/SearchBar.vue'
import SearchResults from './components/SearchResults.vue'

const store = useLogsStore()
const search = useSearchStore()
const unsubscribers: Array<() => void> = []

onMounted(() => {
  // The backend scans on startup; pull whatever is ready now…
  store.refreshList()
  store.loadAllTagNames()
  // …and refresh again when the (async) initial scan finishes.
  unsubscribers.push(EventsOn('logs:scanned', () => store.refreshList()))
  // A new or removed log changes the list.
  unsubscribers.push(EventsOn('log:new', () => store.refreshList()))
  unsubscribers.push(EventsOn('log:removed', () => store.refreshList()))
  // A growing, currently-open log should reload its entries.
  unsubscribers.push(
    EventsOn('log:updated', (path: string) => {
      if (path === store.selectedPath) store.openLog(path)
    }),
  )
})

onUnmounted(() => {
  unsubscribers.forEach((off) => off())
})
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>Gobchat Log Browser</h1>
      <SearchBar />
    </header>
    <main class="app-body">
      <LogList />
      <div class="main-pane">
        <LogViewer />
        <SearchResults v-if="search.ran" />
      </div>
    </main>
  </div>
</template>
