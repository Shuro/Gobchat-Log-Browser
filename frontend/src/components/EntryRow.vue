<script setup lang="ts">
import { computed } from 'vue'
import type { api } from '../../wailsjs/go/models'

const props = defineProps<{ entry: api.EntryDTO }>()

const time = computed(() => {
  if (!props.entry.timestamp) return ''
  const d = new Date(props.entry.timestamp)
  return isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
})

const channelClass = computed(() => `channel-${(props.entry.channel || 'unknown').toLowerCase()}`)
const isUnknown = computed(() => props.entry.channel === 'Unknown')
</script>

<template>
  <div class="entry" :class="channelClass">
    <span class="time">{{ time }}</span>
    <span class="channel-tag">{{ entry.channel }}</span>
    <span v-if="!isUnknown" class="sender" :title="entry.sender">
      <span v-if="entry.status_symbol" class="symbol">{{ entry.status_symbol }}</span>
      <span class="name">{{ entry.display_name || entry.sender }}</span>
      <span v-if="entry.realm" class="realm">{{ entry.realm }}</span>
    </span>
    <span class="message">
      <span
        v-for="(span, i) in entry.spans"
        :key="i"
        :class="`span-${span.type}`"
        >{{ span.text }}</span
      >
    </span>
    <span v-if="entry.part_total > 0" class="part">{{ entry.part_index }}/{{ entry.part_total }}</span>
  </div>
</template>
