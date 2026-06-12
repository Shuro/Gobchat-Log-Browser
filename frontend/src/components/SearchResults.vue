<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSearchStore } from '../stores/search'
import { useLogsStore } from '../stores/logs'
import { splitMatchesAny } from '../utils/findMatches'
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

// Mirror of the Go index tokenizer (internal/search): split on anything that
// is not a letter or digit, lowercase. Highlights each AND term in snippets.
const queryTerms = computed(() =>
  search.query.toLowerCase().split(/[^\p{L}\p{N}]+/u).filter(Boolean),
)

function snippetSegments(hit: api.SearchResultDTO) {
  return splitMatchesAny(hit.snippet, queryTerms.value)
}

function isOpened(hit: api.SearchResultDTO): boolean {
  return (
    search.openedHit?.filePath === hit.file_path &&
    search.openedHit?.lineNumber === hit.line_number
  )
}

// Opening a hit hides the overlay but keeps the results; the "Results (N)"
// button in the search bar brings them back.
function open(hit: api.SearchResultDTO) {
  search.openedHit = { filePath: hit.file_path, lineNumber: hit.line_number }
  logs.openLog(hit.file_path, hit.line_number)
  search.hide()
}
</script>

<template>
  <div class="search-results">
    <header class="results-header">
      <span v-if="search.loading">{{ t('search.searching') }}</span>
      <template v-else>
        <span>{{ t('search.results', { count: search.results.length }) }}</span>
        <span v-if="search.truncated" class="truncated-hint">{{ t('search.truncated') }}</span>
      </template>
      <button class="ghost results-close" @click="search.clear()">{{ t('search.close') }}</button>
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
          <li
            v-for="hit in g.hits"
            :key="hit.line_number"
            :class="{ active: isOpened(hit) }"
            @click="open(hit)"
          >
            <span class="hit-sender">{{ hit.sender || hit.channel }}</span>
            <span class="hit-snippet"
              ><template v-for="(seg, j) in snippetSegments(hit)" :key="j"
                ><span v-if="seg.match" class="find-match">{{ seg.text }}</span
                ><template v-else>{{ seg.text }}</template></template
              ></span
            >
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
