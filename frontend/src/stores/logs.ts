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

  // When a search result is opened, the viewer scrolls to and highlights this
  // line number; it is cleared once consumed / after a moment.
  const targetLine = ref<number | null>(null)

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

  async function openLog(path: string, line: number | null = null) {
    // Re-opening the same log to jump to a line: keep entries, just retarget.
    if (path === selectedPath.value && entries.value.length > 0) {
      targetLine.value = line
      return
    }
    selectedPath.value = path
    loadingEntries.value = true
    error.value = null
    filterText.value = ''
    targetLine.value = null
    try {
      entries.value = await GetLogEntries(path)
      targetLine.value = line
    } catch (e: unknown) {
      error.value = String(e)
      entries.value = []
    } finally {
      loadingEntries.value = false
    }
  }

  function clearTarget() {
    targetLine.value = null
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
    targetLine,
    refreshList,
    rescan,
    openLog,
    clearTarget,
  }
})
