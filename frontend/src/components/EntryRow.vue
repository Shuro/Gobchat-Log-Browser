<script setup lang="ts">
import { computed } from 'vue'
import type { api } from '../../wailsjs/go/models'
import { splitMatches } from '../utils/findMatches'

const props = defineProps<{
  entry: api.EntryDTO
  highlight?: boolean
  query?: string
  messageOnly?: boolean
  hideRealm?: boolean
}>()

const time = computed(() => {
  if (!props.entry.timestamp) return ''
  const d = new Date(props.entry.timestamp)
  return isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
})

const channelClass = computed(() => `channel-${(props.entry.channel || 'unknown').toLowerCase()}`)
const isUnknown = computed(() => props.entry.channel === 'Unknown')
// System lines (e.g. the Error channel) have no sender at all; skip the empty
// sender column instead of rendering a gap.
const hasSender = computed(
  () => !isUnknown.value && Boolean(props.entry.display_name || props.entry.sender),
)

const spanSegments = computed(() =>
  props.entry.spans.map((s) => splitMatches(s.text, props.query ?? '')),
)
const nameSegments = computed(() =>
  splitMatches(
    props.entry.display_name || props.entry.sender,
    props.messageOnly ? '' : props.query ?? '',
  ),
)
</script>

<template>
  <div class="entry" :class="[channelClass, { target: highlight }]">
    <span class="time">{{ time }}</span>
    <span class="channel-tag">{{ entry.channel }}</span>
    <span v-if="hasSender" class="sender" :title="entry.sender">
      <span v-if="entry.status_symbol" class="symbol">{{ entry.status_symbol }}</span>
      <span class="name"
        ><template v-for="(seg, j) in nameSegments" :key="j"
          ><span v-if="seg.match" class="find-match">{{ seg.text }}</span
          ><template v-else>{{ seg.text }}</template></template
        ></span
      >
      <span v-if="entry.realm && !hideRealm" class="realm">{{ entry.realm }}</span>
    </span>
    <span class="message">
      <span
        v-for="(span, i) in entry.spans"
        :key="i"
        :class="`span-${span.type}`"
        ><template v-for="(seg, j) in spanSegments[i]" :key="j"
          ><span v-if="seg.match" class="find-match">{{ seg.text }}</span
          ><template v-else>{{ seg.text }}</template></template
        ></span
      >
    </span>
    <span v-if="entry.part_total > 0" class="part">{{ entry.part_index }}/{{ entry.part_total }}</span>
  </div>
</template>
