<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '../stores/logs'

const { t } = useI18n()
const store = useLogsStore()

const input = ref('')

const suggestions = computed(() =>
  store.allPlayers.filter((p) => !store.selectedPlayers.includes(p)),
)

function add() {
  store.addPlayer(input.value)
  if (store.selectedPlayers.includes(input.value.trim())) input.value = ''
}
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
    <input
      v-model="input"
      class="player-input"
      list="all-players"
      :placeholder="t('nav.filterPlayers')"
      @keyup.enter="add"
      @change="add"
    />
    <datalist id="all-players">
      <option v-for="p in suggestions" :key="p" :value="p" />
    </datalist>
    <button v-if="store.selectedPlayers.length > 0" class="clear" @click="store.clearPlayers()">
      {{ t('nav.clearFilter') }}
    </button>
  </div>
</template>
