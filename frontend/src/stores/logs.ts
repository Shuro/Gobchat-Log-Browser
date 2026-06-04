import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  GetLogList,
  ScanLogs,
  GetLogEntries,
  GetLogThreads,
  GetTags,
  SetTags,
  GetAllTagNames,
} from '../../wailsjs/go/api/App'
import type { api, tags } from '../../wailsjs/go/models'

function baseName(path: string): string {
  const parts = path.split(/[\\/]/)
  return parts[parts.length - 1] ?? path
}

// The logs store owns the overview list, the currently opened log (raw entries
// or the optional in-memory reassembled threads), and that log's tags/note.
export const useLogsStore = defineStore('logs', () => {
  const summaries = ref<api.LogSummary[]>([])
  const loadingList = ref(false)

  const selectedPath = ref<string | null>(null)
  const entries = ref<api.EntryDTO[]>([])
  const loadingEntries = ref(false)
  const error = ref<string | null>(null)

  // Raw vs. reassembled view (reassembly is in-memory only; see ADR-0007).
  const viewMode = ref<'raw' | 'reassembled'>('raw')
  const threads = ref<api.ThreadDTO[]>([])
  const loadingThreads = ref(false)

  // When a search result is opened, the viewer scrolls to and highlights this
  // line number; it is cleared once consumed / after a moment.
  const targetLine = ref<number | null>(null)

  // Per-log filter text (matches message/combined or sender, case-insensitive).
  const filterText = ref('')

  // Tags/note for the selected log, plus all known tag names for autocomplete.
  const currentTags = ref<tags.FileTags>({ file_name: '', tags: [], note: '' })
  const allTagNames = ref<string[]>([])

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

  const visibleThreads = computed(() => {
    const q = filterText.value.trim().toLowerCase()
    if (!q) return threads.value
    return threads.value.filter(
      (t) =>
        t.combined.toLowerCase().includes(q) || t.sender.toLowerCase().includes(q),
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

  async function openLog(path: string, line: number | null = null, force = false) {
    // Re-opening the same log to jump to a line: keep entries, just retarget.
    if (!force && path === selectedPath.value && entries.value.length > 0) {
      viewMode.value = 'raw' // a line target only makes sense in the raw view
      targetLine.value = line
      return
    }
    selectedPath.value = path
    loadingEntries.value = true
    error.value = null
    filterText.value = ''
    targetLine.value = null
    viewMode.value = 'raw'
    threads.value = []
    try {
      entries.value = await GetLogEntries(path)
      targetLine.value = line
    } catch (e: unknown) {
      error.value = String(e)
      entries.value = []
    } finally {
      loadingEntries.value = false
    }
    await loadTags(baseName(path))
  }

  async function setViewMode(mode: 'raw' | 'reassembled') {
    viewMode.value = mode
    if (mode === 'reassembled' && threads.value.length === 0 && selectedPath.value) {
      loadingThreads.value = true
      try {
        threads.value = await GetLogThreads(selectedPath.value)
      } catch (e: unknown) {
        error.value = String(e)
      } finally {
        loadingThreads.value = false
      }
    }
  }

  // reloadCurrent re-fetches the open log, bypassing the cache-skip, so new
  // highlight markers / mention names take effect after a settings change.
  async function reloadCurrent() {
    if (!selectedPath.value) return
    const wasReassembled = viewMode.value === 'reassembled'
    threads.value = []
    await openLog(selectedPath.value, null, true)
    if (wasReassembled) await setViewMode('reassembled')
  }

  function clearTarget() {
    targetLine.value = null
  }

  async function loadTags(fileName: string) {
    currentTags.value = await GetTags(fileName)
  }

  async function loadAllTagNames() {
    allTagNames.value = await GetAllTagNames()
  }

  async function saveTags(tagList: string[], note: string) {
    const fn = currentTags.value.file_name || (selectedPath.value ? baseName(selectedPath.value) : '')
    if (!fn) return
    await SetTags(fn, tagList, note)
    currentTags.value = { file_name: fn, tags: tagList, note }
    // Reflect in the overview list immediately.
    const s = summaries.value.find((x) => x.file_name === fn)
    if (s) {
      s.tags = tagList
      s.note = note
    }
    await loadAllTagNames()
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
    viewMode,
    threads,
    visibleThreads,
    loadingThreads,
    filterText,
    targetLine,
    currentTags,
    allTagNames,
    refreshList,
    rescan,
    openLog,
    reloadCurrent,
    setViewMode,
    clearTarget,
    loadTags,
    loadAllTagNames,
    saveTags,
  }
})
