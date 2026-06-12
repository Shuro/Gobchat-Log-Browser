import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
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

// One entry in the overview filter; type-aware so tag names may collide with
// player names without ambiguity. The "#" shown on tags is display-only.
export type FilterSelection = { type: 'player' | 'tag'; value: string }

function loadExcludedChannels(): string[] {
  try {
    const v = JSON.parse(localStorage.getItem('view.excludedChannels') ?? '[]')
    return Array.isArray(v) ? v.filter((x) => typeof x === 'string') : []
  } catch {
    return []
  }
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

  // Per-log find text (matches message/combined or sender, case-insensitive).
  // 'highlight' keeps all lines visible and marks matches; 'filter' hides
  // non-matching lines (the legacy behavior). The mode is session-sticky.
  // Both toggle states persist across restarts (WebView2 keeps localStorage
  // in the app's user data dir).
  const filterText = ref('')
  const filterMode = ref<'highlight' | 'filter'>(
    localStorage.getItem('find.filterMode') === 'filter' ? 'filter' : 'highlight',
  )
  // When true, only the message text is searched, not the sender name.
  const messageOnly = ref(localStorage.getItem('find.messageOnly') === '1')
  watch([filterMode, messageOnly], ([mode, msgOnly]) => {
    localStorage.setItem('find.filterMode', mode)
    localStorage.setItem('find.messageOnly', msgOnly ? '1' : '0')
  })
  // When true, "[Realm]" suffixes are hidden in both viewer modes (display
  // only; the underlying entries keep the full sender).
  const hideRealm = ref(localStorage.getItem('view.hideRealm') === '1')
  watch(hideRealm, (v) => localStorage.setItem('view.hideRealm', v ? '1' : '0'))
  // Channels hidden in the viewer (both modes). Sticky across logs and
  // restarts so "always hide System noise" works like hideRealm.
  const excludedChannels = ref<string[]>(loadExcludedChannels())
  watch(excludedChannels, (v) =>
    localStorage.setItem('view.excludedChannels', JSON.stringify(v)),
  )
  // 0-based position within matchIndexes for Enter/Shift+Enter navigation.
  const currentMatch = ref(0)

  // Tags/note for the selected log, plus all known tag names for autocomplete.
  const currentTags = ref<tags.FileTags>({ file_name: '', tags: [], note: '' })
  const allTagNames = ref<string[]>([])

  const selectedSummary = computed(() =>
    summaries.value.find((s) => s.file_path === selectedPath.value) ?? null,
  )

  // Player/tag filter over the overview list: logs must contain ALL selected
  // players AND carry all selected tags, so picking two names finds the scenes
  // between them. Selections are type-aware because a tag may share its name
  // with a player.
  const selectedFilters = ref<FilterSelection[]>([])

  const allPlayers = computed(() => {
    const set = new Set<string>()
    for (const s of summaries.value) for (const p of s.participants ?? []) set.add(p)
    return [...set].sort()
  })

  // Distinct tags currently present on logs (allTagNames covers every tag ever
  // saved and feeds the TagEditor datalist instead).
  const allTags = computed(() => {
    const set = new Set<string>()
    for (const s of summaries.value) for (const t of s.tags ?? []) set.add(t)
    return [...set].sort()
  })

  const visibleSummaries = computed(() => {
    if (selectedFilters.value.length === 0) return summaries.value
    return summaries.value.filter((s) =>
      selectedFilters.value.every((f) =>
        f.type === 'player' ? s.participants?.includes(f.value) : s.tags?.includes(f.value),
      ),
    )
  })

  function hasFilter(sel: FilterSelection): boolean {
    return selectedFilters.value.some((f) => f.type === sel.type && f.value === sel.value)
  }

  function addFilter(sel: FilterSelection) {
    const value = sel.value.trim()
    if (!value || hasFilter({ ...sel, value })) return
    const known = sel.type === 'player' ? allPlayers.value : allTags.value
    if (known.includes(value)) selectedFilters.value.push({ type: sel.type, value })
  }

  function removeFilter(sel: FilterSelection) {
    selectedFilters.value = selectedFilters.value.filter(
      (f) => f.type !== sel.type || f.value !== sel.value,
    )
  }

  function clearFilters() {
    selectedFilters.value = []
  }

  function entryMatches(e: api.EntryDTO, q: string): boolean {
    return (
      e.message.toLowerCase().includes(q) ||
      (!messageOnly.value && (e.display_name || e.sender).toLowerCase().includes(q))
    )
  }

  function threadMatches(t: api.ThreadDTO, q: string): boolean {
    return (
      t.combined.toLowerCase().includes(q) ||
      (!messageOnly.value && t.sender.toLowerCase().includes(q))
    )
  }

  // Channel exclusion applies before the find filter in both modes.
  const channelVisibleEntries = computed(() => {
    const excluded = new Set(excludedChannels.value)
    if (excluded.size === 0) return entries.value
    return entries.value.filter((e) => !excluded.has(e.channel))
  })

  const channelVisibleThreads = computed(() => {
    const excluded = new Set(excludedChannels.value)
    if (excluded.size === 0) return threads.value
    return threads.value.filter((t) => !excluded.has(t.channel))
  })

  function toggleChannel(channel: string) {
    if (excludedChannels.value.includes(channel)) {
      excludedChannels.value = excludedChannels.value.filter((c) => c !== channel)
    } else {
      excludedChannels.value = [...excludedChannels.value, channel]
    }
  }

  const visibleEntries = computed(() => {
    const q = filterText.value.trim().toLowerCase()
    if (!q || filterMode.value === 'highlight') return channelVisibleEntries.value
    return channelVisibleEntries.value.filter((e) => entryMatches(e, q))
  })

  const visibleThreads = computed(() => {
    const q = filterText.value.trim().toLowerCase()
    if (!q || filterMode.value === 'highlight') return channelVisibleThreads.value
    return channelVisibleThreads.value.filter((t) => threadMatches(t, q))
  })

  // Indexes of matching rows within the currently visible list, used for the
  // match counter, next/prev navigation, and the scrollbar marker track.
  const matchIndexes = computed<number[]>(() => {
    const q = filterText.value.trim().toLowerCase()
    if (!q) return []
    const out: number[] = []
    if (viewMode.value === 'raw') {
      visibleEntries.value.forEach((e, i) => {
        if (entryMatches(e, q)) out.push(i)
      })
    } else {
      visibleThreads.value.forEach((t, i) => {
        if (threadMatches(t, q)) out.push(i)
      })
    }
    return out
  })

  function nextMatch() {
    const n = matchIndexes.value.length
    if (n > 0) currentMatch.value = (currentMatch.value + 1) % n
  }

  function prevMatch() {
    const n = matchIndexes.value.length
    if (n > 0) currentMatch.value = (currentMatch.value - 1 + n) % n
  }

  watch([filterText, viewMode, filterMode, messageOnly, excludedChannels], () => {
    currentMatch.value = 0
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

  // reloadCurrent re-fetches the open log in place — after a settings change
  // (new markers / mention names) or a live file update. View mode, find text,
  // and scroll position stay untouched; the scroller keeps its place because
  // the row keys (line numbers) are stable.
  async function reloadCurrent() {
    if (!selectedPath.value) return
    error.value = null
    try {
      entries.value = await GetLogEntries(selectedPath.value)
      if (viewMode.value === 'reassembled') {
        threads.value = await GetLogThreads(selectedPath.value)
      } else {
        // Drop stale threads so the next switch to reassembled refetches.
        threads.value = []
      }
    } catch (e: unknown) {
      error.value = String(e)
    }
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

  // saveTags takes an explicit file name so the TagEditor can still save a
  // dirty draft for the *previous* log after the user switched to another one.
  async function saveTags(fileName: string, tagList: string[], note: string) {
    if (!fileName) return
    await SetTags(fileName, tagList, note)
    if (currentTags.value.file_name === fileName) {
      currentTags.value = { file_name: fileName, tags: tagList, note }
    }
    // Reflect in the overview list immediately.
    const s = summaries.value.find((x) => x.file_name === fileName)
    if (s) {
      s.tags = tagList
      s.note = note
    }
    await loadAllTagNames()
  }

  return {
    summaries,
    loadingList,
    selectedFilters,
    allPlayers,
    allTags,
    visibleSummaries,
    addFilter,
    removeFilter,
    clearFilters,
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
    filterMode,
    messageOnly,
    hideRealm,
    excludedChannels,
    toggleChannel,
    currentMatch,
    matchIndexes,
    nextMatch,
    prevMatch,
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
