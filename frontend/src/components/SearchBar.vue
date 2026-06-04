<script setup lang="ts">
import { useSearchStore } from '../stores/search'
import { useLogsStore } from '../stores/logs'

const search = useSearchStore()
const logs = useLogsStore()
</script>

<template>
  <div class="search-bar">
    <select v-model="search.scope" class="scope" :title="'Search scope'">
      <option value="all">All logs</option>
      <option value="current" :disabled="!logs.selectedPath">Current log</option>
    </select>
    <input
      v-model="search.query"
      class="search-input"
      type="search"
      placeholder="Search…"
      @keyup.enter="search.run()"
      @search="search.run()"
    />
    <button :disabled="search.loading" @click="search.run()">
      {{ search.loading ? '…' : 'Search' }}
    </button>
    <button v-if="search.ran" class="ghost" @click="search.reset()">Clear</button>
  </div>
</template>
