<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { useLogsStore } from '../stores/logs'
import EntryRow from './EntryRow.vue'
import ThreadRow from './ThreadRow.vue'
import TagEditor from './TagEditor.vue'

const store = useLogsStore()
const scroller = ref<any>(null)

// Threads have no single unique field; key by their first original line number.
const threadItems = computed(() =>
  store.visibleThreads.map((t, i) => ({ ...t, _id: t.lines[0] ?? i })),
)

// When a search result targets a line, scroll it into view (raw mode only).
watch(
  () => [store.targetLine, store.entries] as const,
  async () => {
    const line = store.targetLine
    if (line == null || store.viewMode !== 'raw') return
    await nextTick()
    const idx = store.visibleEntries.findIndex((e) => e.line_number === line)
    if (idx >= 0 && scroller.value?.scrollToItem) {
      scroller.value.scrollToItem(idx)
    }
  },
  { flush: 'post' },
)
</script>

<template>
  <section class="viewer">
    <template v-if="store.selectedSummary">
      <header class="viewer-header">
        <div class="viewer-title">
          <strong>{{ store.selectedSummary.file_name }}</strong>
          <span class="muted">{{ store.selectedSummary.message_count }} messages</span>
        </div>
        <div class="viewer-controls">
          <div class="mode-toggle">
            <button
              :class="{ active: store.viewMode === 'raw' }"
              @click="store.setViewMode('raw')"
            >
              Raw
            </button>
            <button
              :class="{ active: store.viewMode === 'reassembled' }"
              @click="store.setViewMode('reassembled')"
            >
              Reassembled
            </button>
          </div>
          <input
            v-model="store.filterText"
            class="filter"
            type="search"
            placeholder="Filter this log…"
          />
        </div>
      </header>
      <TagEditor />
    </template>

    <div v-if="store.loadingEntries" class="placeholder">Loading…</div>
    <div v-else-if="store.error" class="placeholder error">{{ store.error }}</div>
    <div v-else-if="!store.selectedPath" class="placeholder">
      Select a log on the left to view it.
    </div>

    <!-- Raw view -->
    <template v-else-if="store.viewMode === 'raw'">
      <div v-if="store.visibleEntries.length === 0" class="placeholder">
        No entries match the filter.
      </div>
      <DynamicScroller
        v-else
        ref="scroller"
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
            <EntryRow :entry="item" :highlight="item.line_number === store.targetLine" />
          </DynamicScrollerItem>
        </template>
      </DynamicScroller>
    </template>

    <!-- Reassembled view -->
    <template v-else>
      <div v-if="store.loadingThreads" class="placeholder">Reassembling…</div>
      <div v-else-if="threadItems.length === 0" class="placeholder">
        No threads match the filter.
      </div>
      <DynamicScroller
        v-else
        :items="threadItems"
        :min-item-size="32"
        key-field="_id"
        class="scroller"
      >
        <template #default="{ item, index, active }">
          <DynamicScrollerItem
            :item="item"
            :active="active"
            :data-index="index"
            :size-dependencies="[item.combined]"
          >
            <ThreadRow :thread="item" />
          </DynamicScrollerItem>
        </template>
      </DynamicScroller>
    </template>
  </section>
</template>
