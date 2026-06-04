<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '../stores/logs'

const { t } = useI18n()
const store = useLogsStore()

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '' : d.toLocaleString()
}
</script>

<template>
  <aside class="log-list">
    <header class="list-header">
      <strong>{{ t('nav.logs') }}</strong>
      <button :disabled="store.loadingList" @click="store.rescan()">{{ t('nav.rescan') }}</button>
    </header>

    <div v-if="store.loadingList" class="placeholder">{{ t('nav.scanning') }}</div>
    <div v-else-if="store.summaries.length === 0" class="placeholder">
      {{ t('nav.noLogs') }}
    </div>

    <ul v-else class="list">
      <li
        v-for="log in store.summaries"
        :key="log.file_path"
        :class="{ active: log.file_path === store.selectedPath }"
        @click="store.openLog(log.file_path)"
      >
        <div class="row-top">
          <span class="date">{{ formatDate(log.log_date) }}</span>
          <span class="count">{{ log.message_count }}</span>
        </div>
        <div v-if="log.folder" class="folder">{{ log.folder }}</div>
        <div v-if="log.participants && log.participants.length" class="participants">
          {{ log.participants.join(', ') }}
        </div>
        <div class="row-bottom">
          <span v-if="log.duration" class="duration">{{ log.duration }}</span>
          <span v-for="t in log.tags" :key="t" class="tag">{{ t }}</span>
        </div>
      </li>
    </ul>
  </aside>
</template>
