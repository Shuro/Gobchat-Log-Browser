<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore, type FilterSelection } from '../stores/logs'
import { useConfigStore } from '../stores/config'

const { t } = useI18n()
const store = useLogsStore()
const configStore = useConfigStore()

const input = ref('')
// Custom dropdown instead of a native <datalist>: Chromium truncates long
// datalist popups, which cut off the player list partway through.
const open = ref(false)
const activeIdx = ref(-1)
const listEl = ref<HTMLUListElement | null>(null)
const inputEl = ref<HTMLInputElement | null>(null)

// The user's own characters (settings) are pinned above the other names.
const pinned = computed(() => new Set(configStore.cfg?.roleplay_characters ?? []))

function label(sel: FilterSelection): string {
  return sel.type === 'tag' ? `#${sel.value}` : sel.value
}

// Group order: pinned characters, then tags, then the (long) player list —
// nobody would scroll past hundreds of players to reach the tags.
const suggestions = computed<FilterSelection[]>(() => {
  const q = input.value.trim().toLowerCase()
  const tagQ = q.startsWith('#') ? q.slice(1) : q

  const selected = (sel: FilterSelection) =>
    store.selectedFilters.some((f) => f.type === sel.type && f.value === sel.value)

  let players = store.allPlayers
    .map((p): FilterSelection => ({ type: 'player', value: p }))
    .filter((s) => !selected(s))
  if (q) players = players.filter((s) => s.value.toLowerCase().includes(q))

  let tags = store.allTags
    .map((v): FilterSelection => ({ type: 'tag', value: v }))
    .filter((s) => !selected(s))
  if (tagQ) tags = tags.filter((s) => s.value.toLowerCase().includes(tagQ))

  return [
    ...players.filter((s) => pinned.value.has(s.value)),
    ...tags,
    ...players.filter((s) => !pinned.value.has(s.value)),
  ]
})

// Indexes of the first tag / first unpinned player; dividers are drawn above them.
const pinnedCount = computed(
  () => suggestions.value.filter((s) => s.type === 'player' && pinned.value.has(s.value)).length,
)
const tagCount = computed(() => suggestions.value.filter((s) => s.type === 'tag').length)

function isGroupStart(i: number): boolean {
  return (
    (pinnedCount.value > 0 && i === pinnedCount.value) ||
    (tagCount.value > 0 && i === pinnedCount.value + tagCount.value && i < suggestions.value.length)
  )
}

function add(sel: FilterSelection) {
  store.addFilter(sel)
  if (store.selectedFilters.some((f) => f.type === sel.type && f.value === sel.value)) {
    input.value = ''
  }
  activeIdx.value = -1
  open.value = false
  // Suggestion clicks keep focus on the input (mousedown.prevent), and a
  // re-click on a focused input fires neither focus nor input, so the
  // dropdown could never reopen. Blur so the next click is a fresh focus.
  inputEl.value?.blur()
}

function onEnter() {
  const list = suggestions.value
  const q = input.value.trim().toLowerCase()
  const pick =
    activeIdx.value >= 0
      ? list[activeIdx.value]
      : list.find((s) => s.value.toLowerCase() === q || label(s).toLowerCase() === q) ?? list[0]
  if (pick) add(pick)
}

function move(delta: number) {
  if (!open.value) {
    open.value = true
    return
  }
  const n = suggestions.value.length
  if (n > 0) activeIdx.value = (activeIdx.value + delta + n) % n
}

watch(activeIdx, async () => {
  await nextTick()
  listEl.value?.querySelector('li.active')?.scrollIntoView({ block: 'nearest' })
})
</script>

<template>
  <div class="player-filter">
    <div class="filter-row">
      <div class="suggest-wrap">
        <input
          ref="inputEl"
          v-model="input"
          class="player-input"
          :placeholder="t('nav.filterPlayers')"
          @focus="open = true"
          @input="open = true; activeIdx = -1"
          @keydown.enter="onEnter"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.esc="open = false"
          @blur="open = false"
        />
        <ul v-if="open && suggestions.length" ref="listEl" class="suggest-list">
          <li
            v-for="(s, i) in suggestions"
            :key="s.type + ':' + s.value"
            :class="{ active: i === activeIdx, 'group-sep': isGroupStart(i) }"
            @mousedown.prevent="add(s)"
          >
            {{ label(s) }}
          </li>
        </ul>
      </div>
      <button v-if="store.selectedFilters.length > 0" class="clear" @click="store.clearFilters()">
        {{ t('nav.clearFilter') }}
      </button>
    </div>
    <div v-if="store.selectedFilters.length > 0" class="filter-badges">
      <span
        v-for="f in store.selectedFilters"
        :key="f.type + ':' + f.value"
        class="tag removable"
        @click="store.removeFilter(f)"
      >
        {{ label(f) }} <span class="x">✕</span>
      </span>
    </div>
  </div>
</template>
