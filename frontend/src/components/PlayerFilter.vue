<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '../stores/logs'
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

const suggestions = computed(() => {
  const q = input.value.trim().toLowerCase()
  let pool = store.allPlayers.filter((p) => !store.selectedPlayers.includes(p))
  if (q) pool = pool.filter((p) => p.toLowerCase().includes(q))
  return [...pool.filter((p) => pinned.value.has(p)), ...pool.filter((p) => !pinned.value.has(p))]
})

// Index of the first unpinned suggestion; the divider is drawn above it.
const pinnedCount = computed(() => suggestions.value.filter((p) => pinned.value.has(p)).length)

function add(name: string) {
  store.addPlayer(name)
  if (store.selectedPlayers.includes(name)) input.value = ''
  activeIdx.value = -1
  open.value = false
  // Suggestion clicks keep focus on the input (mousedown.prevent), and a
  // re-click on a focused input fires neither focus nor input, so the
  // dropdown could never reopen. Blur so the next click is a fresh focus.
  inputEl.value?.blur()
}

function onEnter() {
  const list = suggestions.value
  const pick =
    activeIdx.value >= 0
      ? list[activeIdx.value]
      : list.find((p) => p.toLowerCase() === input.value.trim().toLowerCase()) ?? list[0]
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
    <span
      v-for="p in store.selectedPlayers"
      :key="p"
      class="tag removable"
      @click="store.removePlayer(p)"
    >
      {{ p }} <span class="x">✕</span>
    </span>
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
          v-for="(p, i) in suggestions"
          :key="p"
          :class="{ active: i === activeIdx, 'group-sep': pinnedCount > 0 && i === pinnedCount }"
          @mousedown.prevent="add(p)"
        >
          {{ p }}
        </li>
      </ul>
    </div>
    <button v-if="store.selectedPlayers.length > 0" class="clear" @click="store.clearPlayers()">
      {{ t('nav.clearFilter') }}
    </button>
  </div>
</template>
