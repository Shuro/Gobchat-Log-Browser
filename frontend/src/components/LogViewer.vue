<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { useLogsStore } from '../stores/logs'
import EntryRow from './EntryRow.vue'
import ThreadRow from './ThreadRow.vue'
import TagEditor from './TagEditor.vue'
import { estimateEntryHeight, estimateThreadHeight } from '../utils/rowHeights'

const { t } = useI18n()
const store = useLogsStore()
const scroller = ref<any>(null)
const filterInput = ref<HTMLInputElement | null>(null)

// Threads have no single unique field; key by their first original line number.
const threadItems = computed(() =>
  store.visibleThreads.map((t, i) => ({ ...t, _id: t.lines[0] ?? i })),
)

// Ctrl+F focuses the find field (no-op while no log is open: ref is null).
function onKeydown(e: KeyboardEvent) {
  if (e.ctrlKey && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'f') {
    e.preventDefault()
    filterInput.value?.focus()
    filterInput.value?.select()
  }
  if (e.key === 'Escape') channelsOpen.value = false
}

// --- Channel visibility dropdown ------------------------------------------
const channelsOpen = ref(false)
const channelsWrap = ref<HTMLElement | null>(null)
const logChannels = computed(() => [...(store.selectedSummary?.channels ?? [])].sort())
// Exclusions are sticky across logs; the badge only counts channels that are
// actually hidden in the open log.
const hiddenChannelCount = computed(
  () => logChannels.value.filter((c) => store.excludedChannels.includes(c)).length,
)

// The popover holds checkboxes, so it must survive clicks inside itself;
// close only on clicks outside the wrap (and on Escape via onKeydown).
function onDocMousedown(e: MouseEvent) {
  if (channelsOpen.value && !channelsWrap.value?.contains(e.target as Node)) {
    channelsOpen.value = false
  }
}
// Pre-fill the scroller's reactive size map with estimated heights for all
// rows it hasn't measured yet. The scroller assumes min-item-size for
// unmeasured rows, which makes the scrollbar thumb and the match ticks drift
// and jump as real measurements come in; good estimates up front keep the
// geometry stable. Rendered rows are re-measured by the scroller's
// ResizeObserver, overwriting these estimates with exact values.
const predicted = new Map<string | number, number>()
let lastScroller: unknown = null

function prefillSizes(repredict = false) {
  const sc = scroller.value
  const el = sc?.$el as HTMLElement | undefined
  if (!sc?.vscrollData || !el || el.clientWidth === 0) return
  if (sc !== lastScroller) {
    // New scroller instance (mode switch / log change) → fresh size map.
    predicted.clear()
    lastScroller = sc
  }
  const width = el.clientWidth
  const sizes = sc.vscrollData.sizes as Record<string | number, number>
  if (store.viewMode === 'raw') {
    for (const e of store.visibleEntries) {
      const id = e.line_number
      if (sizes[id] == null || (repredict && sizes[id] === predicted.get(id))) {
        const h = estimateEntryHeight(e, width)
        sizes[id] = h
        predicted.set(id, h)
      }
    }
  } else {
    for (const t of threadItems.value) {
      const id = t._id
      if (sizes[id] == null || (repredict && sizes[id] === predicted.get(id))) {
        const h = estimateThreadHeight(t, width)
        sizes[id] = h
        predicted.set(id, h)
      }
    }
  }
}

watch(
  () => [scroller.value, store.viewMode, store.visibleEntries, threadItems.value] as const,
  async () => {
    await nextTick()
    prefillSizes()
  },
  { flush: 'post', immediate: true },
)

// Window resizes change text wrapping: re-estimate rows that were never
// really measured (rendered rows re-measure themselves).
let resizeRaf = 0
function onWindowResize() {
  if (resizeRaf) return
  resizeRaf = requestAnimationFrame(() => {
    resizeRaf = 0
    prefillSizes(true)
  })
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onWindowResize)
  document.addEventListener('mousedown', onDocMousedown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onWindowResize)
  document.removeEventListener('mousedown', onDocMousedown)
  if (targetClearTimer) clearTimeout(targetClearTimer)
})

const hasQuery = computed(() => store.filterText.trim().length > 0)

// Index of the active match within the currently visible list.
const activeIndex = computed(() => store.matchIndexes[store.currentMatch] ?? -1)

function goToMatch(e: KeyboardEvent) {
  if (e.shiftKey) store.prevMatch()
  else store.nextMatch()
}

// Escape clears the find text and gives focus back to the page.
function clearFind() {
  store.filterText = ''
  filterInput.value?.blur()
}

// Scrollbar marker positions in percent, deduplicated to half-percent steps
// so the track renders at most ~200 ticks no matter how many matches exist.
// Positions mirror the scroller's own geometry via its reactive itemsWithSize
// (measured row height, or the prefilled estimate from prefillSizes) so ticks
// line up with the scrollbar even with wildly varying row heights. Falls back
// to a plain index fraction before the scroller is mounted.
const tickPercents = computed<number[]>(() => {
  const set = new Set<number>()
  const items = scroller.value?.itemsWithSize as { size?: number }[] | undefined
  if (items?.length) {
    // Keep in sync with :min-item-size on the scrollers below.
    const minSize = store.viewMode === 'raw' ? 28 : 32
    const matches = new Set(store.matchIndexes)
    const mids: number[] = []
    let acc = 0
    for (let i = 0; i < items.length; i++) {
      const size = items[i].size || minSize
      if (matches.has(i)) mids.push(acc + size / 2)
      acc += size
    }
    if (acc > 0) for (const m of mids) set.add(Math.round((m / acc) * 200) / 2)
  } else {
    const total =
      store.viewMode === 'raw' ? store.visibleEntries.length : store.visibleThreads.length
    if (total === 0) return []
    for (const i of store.matchIndexes) {
      set.add(Math.round(((i + 0.5) / total) * 200) / 2)
    }
  }
  return [...set].sort((a, b) => a - b)
})

// When a search result targets a line, scroll it into view (raw mode only).
// The target highlight clears itself after a moment instead of sticking to
// the row until another log is opened.
let targetClearTimer: ReturnType<typeof setTimeout> | undefined
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
    if (targetClearTimer) clearTimeout(targetClearTimer)
    targetClearTimer = setTimeout(() => store.clearTarget(), 2500)
  },
  { flush: 'post' },
)

// Scroll the active find match into view (also fires when typing changes the
// match list, giving "jump to first match while typing" for free).
watch(
  () => [store.currentMatch, store.matchIndexes] as const,
  async () => {
    const idx = activeIndex.value
    if (idx < 0) return
    await nextTick()
    scroller.value?.scrollToItem?.(idx)
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
          <span class="muted">{{ t('viewer.messages', { count: store.selectedSummary.message_count }) }}</span>
        </div>
        <div class="viewer-controls">
          <div class="mode-toggle">
            <button
              :class="{ active: store.viewMode === 'raw' }"
              @click="store.setViewMode('raw')"
            >
              {{ t('viewer.raw') }}
            </button>
            <button
              :class="{ active: store.viewMode === 'reassembled' }"
              @click="store.setViewMode('reassembled')"
            >
              {{ t('viewer.reassembled') }}
            </button>
          </div>
          <div v-if="logChannels.length" ref="channelsWrap" class="channels-wrap">
            <button
              class="channels-btn"
              :class="{ active: hiddenChannelCount > 0 }"
              :title="t('viewer.channels')"
              @click="channelsOpen = !channelsOpen"
            >
              {{ t('viewer.channels') }}<span v-if="hiddenChannelCount" class="channels-badge">−{{ hiddenChannelCount }}</span>
            </button>
            <div v-if="channelsOpen" class="channels-pop">
              <label v-for="ch in logChannels" :key="ch" class="check">
                <input
                  type="checkbox"
                  :checked="!store.excludedChannels.includes(ch)"
                  @change="store.toggleChannel(ch)"
                />
                {{ ch }}
              </label>
            </div>
          </div>
          <button
            class="icon-btn toggle"
            :class="{ active: store.hideRealm }"
            :title="t('viewer.hideRealm')"
            @click="store.hideRealm = !store.hideRealm"
          >
            🌐
          </button>
          <div class="find-group">
            <input
              ref="filterInput"
              v-model="store.filterText"
              class="filter"
              type="search"
              :placeholder="t('viewer.filter')"
              @keydown.enter="goToMatch"
              @keydown.esc="clearFind"
            />
            <span v-if="hasQuery" class="match-count">
              {{ t('viewer.matchCount', {
                current: store.matchIndexes.length ? store.currentMatch + 1 : 0,
                total: store.matchIndexes.length,
              }) }}
            </span>
            <button
              class="icon-btn"
              :disabled="!store.matchIndexes.length"
              :title="t('viewer.prevMatch')"
              @click="store.prevMatch()"
            >
              ↑
            </button>
            <button
              class="icon-btn"
              :disabled="!store.matchIndexes.length"
              :title="t('viewer.nextMatch')"
              @click="store.nextMatch()"
            >
              ↓
            </button>
            <button
              class="icon-btn toggle"
              :class="{ active: store.messageOnly }"
              :title="t('viewer.messageOnly')"
              @click="store.messageOnly = !store.messageOnly"
            >
              💬
            </button>
            <button
              class="icon-btn toggle"
              :class="{ active: store.filterMode === 'filter' }"
              :title="t('viewer.filterMode')"
              @click="store.filterMode = store.filterMode === 'filter' ? 'highlight' : 'filter'"
            >
              ▽
            </button>
          </div>
        </div>
      </header>
      <TagEditor />
    </template>

    <div v-if="store.loadingEntries" class="placeholder">{{ t('viewer.loading') }}</div>
    <div v-else-if="store.error" class="placeholder error">{{ store.error }}</div>
    <div v-else-if="!store.selectedPath" class="placeholder">
      {{ t('viewer.selectPrompt') }}
    </div>

    <!-- Raw view -->
    <template v-else-if="store.viewMode === 'raw'">
      <div v-if="store.visibleEntries.length === 0" class="placeholder">
        {{ t('viewer.noEntries') }}
      </div>
      <div v-else class="scroller-wrap">
        <DynamicScroller
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
              <EntryRow
                :entry="item"
                :query="store.filterText"
                :message-only="store.messageOnly"
                :hide-realm="store.hideRealm"
                :highlight="item.line_number === store.targetLine || index === activeIndex"
              />
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>
        <div v-if="hasQuery && store.filterMode === 'highlight'" class="match-track">
          <div v-for="p in tickPercents" :key="p" class="match-tick" :style="{ top: p + '%' }" />
        </div>
      </div>
    </template>

    <!-- Reassembled view -->
    <template v-else>
      <div v-if="store.loadingThreads" class="placeholder">{{ t('viewer.reassembling') }}</div>
      <div v-else-if="threadItems.length === 0" class="placeholder">
        {{ t('viewer.noThreads') }}
      </div>
      <div v-else class="scroller-wrap">
        <DynamicScroller
          ref="scroller"
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
              <ThreadRow
                :thread="item"
                :query="store.filterText"
                :message-only="store.messageOnly"
                :hide-realm="store.hideRealm"
                :highlight="index === activeIndex"
              />
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>
        <div v-if="hasQuery && store.filterMode === 'highlight'" class="match-track">
          <div v-for="p in tickPercents" :key="p" class="match-tick" :style="{ top: p + '%' }" />
        </div>
      </div>
    </template>
  </section>
</template>
