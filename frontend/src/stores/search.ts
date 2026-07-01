import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Search } from '../../wailsjs/go/api/App'
import type { api } from '../../wailsjs/go/models'
import { useLogsStore } from './logs'

// The search store drives the global (index-backed) search. Per-log live
// filtering stays in the logs store; this is the cross-file search.
export const useSearchStore = defineStore('search', () => {
  const query = ref('')
  const scope = ref<'all' | 'current'>('all')
  // Optional backend post-filters: restrict hits to one channel and/or to
  // senders whose display name contains this text (case-insensitive).
  const channel = ref('')
  const sender = ref('')
  const results = ref<api.SearchResultDTO[]>([])
  // True when the backend cut the result list off (more matches exist).
  const truncated = ref(false)
  const loading = ref(false)
  const ran = ref(false)
  // Opening a hit hides the results overlay instead of clearing it, so the
  // user can come back and open the next hit without re-running the search.
  const visible = ref(false)
  const openedHit = ref<{ filePath: string; lineNumber: number } | null>(null)
  // Bumped on every run() so a slower, older request can recognize it's been
  // superseded and discard its result instead of clobbering a newer one.
  let requestId = 0

  async function run() {
    const logs = useLogsStore()
    const q = query.value.trim()
    if (!q) {
      clear()
      return
    }
    const thisRequest = ++requestId
    loading.value = true
    ran.value = true
    visible.value = true
    openedHit.value = null
    try {
      const filePath = scope.value === 'current' ? logs.selectedPath ?? '' : ''
      const channels = channel.value ? [channel.value] : []
      const res = await Search(q, filePath, channels, sender.value.trim())
      if (thisRequest !== requestId) return // a newer search superseded this one
      results.value = res.results ?? []
      truncated.value = res.truncated
    } finally {
      if (thisRequest === requestId) loading.value = false
    }
  }

  function hide() {
    visible.value = false
  }

  function clear() {
    results.value = []
    truncated.value = false
    ran.value = false
    visible.value = false
    openedHit.value = null
  }

  function reset() {
    query.value = ''
    clear()
  }

  return {
    query,
    scope,
    channel,
    sender,
    results,
    truncated,
    loading,
    ran,
    visible,
    openedHit,
    run,
    hide,
    clear,
    reset,
  }
})
