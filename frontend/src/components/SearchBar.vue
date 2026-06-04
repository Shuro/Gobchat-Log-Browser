<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useSearchStore } from '../stores/search'
import { useLogsStore } from '../stores/logs'

const { t } = useI18n()
const search = useSearchStore()
const logs = useLogsStore()
</script>

<template>
  <div class="search-bar">
    <select v-model="search.scope" class="scope">
      <option value="all">{{ t('search.scopeAll') }}</option>
      <option value="current" :disabled="!logs.selectedPath">{{ t('search.scopeCurrent') }}</option>
    </select>
    <input
      v-model="search.query"
      class="search-input"
      type="search"
      :placeholder="t('search.placeholder')"
      @keyup.enter="search.run()"
      @search="search.run()"
    />
    <button :disabled="search.loading" @click="search.run()">
      {{ search.loading ? '…' : t('search.button') }}
    </button>
    <button v-if="search.ran" class="ghost" @click="search.reset()">{{ t('search.clear') }}</button>
  </div>
</template>
