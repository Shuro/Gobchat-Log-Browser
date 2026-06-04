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
  const results = ref<api.SearchResultDTO[]>([])
  const loading = ref(false)
  const ran = ref(false)

  async function run() {
    const logs = useLogsStore()
    const q = query.value.trim()
    if (!q) {
      clear()
      return
    }
    loading.value = true
    ran.value = true
    try {
      const filePath = scope.value === 'current' ? logs.selectedPath ?? '' : ''
      results.value = await Search(q, filePath, [], '')
    } finally {
      loading.value = false
    }
  }

  function clear() {
    results.value = []
    ran.value = false
  }

  function reset() {
    query.value = ''
    clear()
  }

  return { query, scope, results, loading, ran, run, clear, reset }
})
