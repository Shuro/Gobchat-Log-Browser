import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetLogList, ScanLogs, GetLogEntries } from '../../wailsjs/go/api/App'
import type { api } from '../../wailsjs/go/models'

// The logs store owns the overview list and the currently opened log's entries.
// Per-log search is a client-side filter over the loaded entries; global search
// will be added in a later step.
export const useLogsStore = defineStore('logs', () => {
  const summaries = ref<api.LogSummary[]>([])
  const loadingList = ref(false)

  const selectedPath = ref<string | null>(null)
  const entries = ref<api.EntryDTO[]>([])
  const loadingEntries = ref(false)
  const error = ref<string | null>(null)

  // Per-log filter text (matches message or sender, case-insensitive).
  const filterText = ref('')

  const selectedSummary = computed(() =>
    summaries.value.find((s) => s.file_path === selectedPath.value) ?? null,
  )

  const visibleEntries = computed(() => {
    const q = filterText.value.trim().toLowerCase()
    if (!q) return entries.value
    return entries.value.filter(
      (e) =>
        e.message.toLowerCase().includes(q) ||
        e.display_name.toLowerCase().includes(q),
    )
  })

  async function refreshList() {
    loadingList.value = true
    try {
      summaries.value = await GetLogList()
    } finally {
      loadingList.value = false
    }
  }

  async function rescan() {
    loadingList.value = true
    try {
      summaries.value = await ScanLogs()
    } finally {
      loadingList.value = false
    }
  }

  async function openLog(path: string) {
    selectedPath.value = path
    loadingEntries.value = true
    error.value = null
    filterText.value = ''
    try {
      entries.value = await GetLogEntries(path)
    } catch (e: unknown) {
      error.value = String(e)
      entries.value = []
    } finally {
      loadingEntries.value = false
    }
  }

  return {
    summaries,
    loadingList,
    selectedPath,
    selectedSummary,
    entries,
    visibleEntries,
    loadingEntries,
    error,
    filterText,
    refreshList,
    rescan,
    openLog,
  }
})
