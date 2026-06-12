<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '../stores/logs'
import PlayerFilter from './PlayerFilter.vue'

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

    <PlayerFilter />

    <div v-if="store.loadingList" class="placeholder">{{ t('nav.scanning') }}</div>
    <div v-else-if="store.summaries.length === 0" class="placeholder">
      {{ t('nav.noLogs') }}
    </div>
    <div v-else-if="store.visibleSummaries.length === 0" class="placeholder">
      {{ t('nav.noFilterMatches') }}
    </div>

    <ul v-else class="list">
      <li
        v-for="log in store.visibleSummaries"
        :key="log.file_path"
        :class="{ active: log.file_path === store.selectedPath }"
        @click="store.openLog(log.file_path)"
      >
        <div class="row-top">
          <span class="date">{{ formatDate(log.log_date) }}</span>
          <span class="row-top-right">
            <span v-if="log.note" class="note-ind" :title="log.note">📝</span>
            <span class="count">{{ log.message_count }}</span>
          </span>
        </div>
        <div v-if="log.participants && log.participants.length" class="participants">
          {{ log.participants.join(', ') }}
        </div>
        <div v-if="log.tags && log.tags.length" class="item-tags">
          <span
            v-for="tg in log.tags"
            :key="tg"
            class="tag clickable"
            :title="t('nav.filterByTag')"
            @click.stop="store.addFilter({ type: 'tag', value: tg })"
            >{{ tg }}</span
          >
        </div>
        <div v-if="log.duration" class="row-bottom">
          <span class="duration">{{ log.duration }}</span>
        </div>
      </li>
    </ul>
  </aside>
</template>
