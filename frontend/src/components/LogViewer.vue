<script setup lang="ts">
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { useLogsStore } from '../stores/logs'
import EntryRow from './EntryRow.vue'

const store = useLogsStore()
</script>

<template>
  <section class="viewer">
    <header v-if="store.selectedSummary" class="viewer-header">
      <div class="viewer-title">
        <strong>{{ store.selectedSummary.file_name }}</strong>
        <span class="muted">{{ store.selectedSummary.message_count }} messages</span>
      </div>
      <input
        v-model="store.filterText"
        class="filter"
        type="search"
        placeholder="Filter this log…"
      />
    </header>

    <div v-if="store.loadingEntries" class="placeholder">Loading…</div>
    <div v-else-if="store.error" class="placeholder error">{{ store.error }}</div>
    <div v-else-if="!store.selectedPath" class="placeholder">
      Select a log on the left to view it.
    </div>
    <div v-else-if="store.visibleEntries.length === 0" class="placeholder">
      No entries match the filter.
    </div>

    <DynamicScroller
      v-else
      :items="store.visibleEntries"
      :min-item-size="28"
      key-field="line_number"
      class="scroller"
    >
      <template #default="{ item, index, active }">
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :data-index="index"
          :size-dependencies="[item.message]"
        >
          <EntryRow :entry="item" />
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>
  </section>
</template>
