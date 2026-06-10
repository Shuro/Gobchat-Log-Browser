<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSearchStore } from '../stores/search'
import { useLogsStore } from '../stores/logs'
import type { api } from '../../wailsjs/go/models'

const { t } = useI18n()
const search = useSearchStore()
const logs = useLogsStore()

interface Group {
  filePath: string
  fileName: string
  hits: api.SearchResultDTO[]
}

// Results arrive sorted by file then line; group them by file for display.
const groups = computed<Group[]>(() => {
  const map = new Map<string, Group>()
  for (const r of search.results) {
    let g = map.get(r.file_path)
    if (!g) {
      g = {
        filePath: r.file_path,
        fileName: r.file_name,
        hits: [],
      }
      map.set(r.file_path, g)
    }
    g.hits.push(r)
  }
  return [...map.values()]
})

function open(hit: api.SearchResultDTO) {
  logs.openLog(hit.file_path, hit.line_number)
  search.clear()
}
</script>

<template>
  <div class="search-results">
    <header class="results-header">
      <span v-if="search.loading">{{ t('search.searching') }}</span>
      <span v-else>{{ t('search.results', { count: search.results.length }) }}</span>
      <button class="ghost" @click="search.clear()">{{ t('search.close') }}</button>
    </header>

    <div v-if="!search.loading && search.results.length === 0" class="placeholder">
      {{ t('search.noMatches') }}
    </div>

    <div v-else class="results-body">
      <div v-for="g in groups" :key="g.filePath" class="result-group">
        <div class="result-file">
          <strong>{{ g.fileName }}</strong>
          <span class="muted">{{ g.hits.length }}</span>
        </div>
        <ul>
          <li v-for="hit in g.hits" :key="hit.line_number" @click="open(hit)">
            <span class="hit-sender">{{ hit.sender || hit.channel }}</span>
            <span class="hit-snippet">{{ hit.snippet }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
