<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { api } from '../../wailsjs/go/models'
import { splitMatches } from '../utils/findMatches'

const { t } = useI18n()
const props = defineProps<{
  thread: api.ThreadDTO
  highlight?: boolean
  query?: string
  messageOnly?: boolean
  hideRealm?: boolean
}>()

function fmtTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const startTime = computed(() => fmtTime(props.thread.start_time))
// End time only for actual multi-line threads, and only when it differs.
const endTime = computed(() => {
  if (props.thread.lines.length < 2) return ''
  const end = fmtTime(props.thread.end_time)
  return end !== startTime.value ? end : ''
})

const channelClass = computed(
  () => `channel-${(props.thread.channel || 'unknown').toLowerCase()}`,
)

// Display-only realm strip; the thread keeps the full sender for matching.
const displaySender = computed(() =>
  props.hideRealm ? props.thread.sender.replace(/\s*\[[^\]]*\]$/, '') : props.thread.sender,
)

const spanSegments = computed(() =>
  props.thread.spans.map((s) => splitMatches(s.text, props.query ?? '')),
)
const nameSegments = computed(() =>
  splitMatches(displaySender.value, props.messageOnly ? '' : props.query ?? ''),
)
</script>

<template>
  <div class="entry thread" :class="[channelClass, { target: highlight }]">
    <span v-if="startTime" class="time thread-time">
      <span>{{ startTime }}</span>
      <span v-if="endTime" class="thread-time-sep">⋮</span>
      <span v-if="endTime">{{ endTime }}</span>
    </span>
    <span class="channel-tag">{{ thread.channel }}</span>
    <span class="sender"
      ><template v-for="(seg, j) in nameSegments" :key="j"
        ><span v-if="seg.match" class="find-match">{{ seg.text }}</span
        ><template v-else>{{ seg.text }}</template></template
      ></span
    >
    <span class="message">
      <span
        v-for="(span, i) in thread.spans"
        :key="i"
        :class="`span-${span.type}`"
        ><template v-for="(seg, j) in spanSegments[i]" :key="j"
          ><span v-if="seg.match" class="find-match">{{ seg.text }}</span
          ><template v-else>{{ seg.text }}</template></template
        ></span
      >
    </span>
    <span v-if="thread.lines.length > 1" class="part">{{ t('viewer.parts', { count: thread.lines.length }) }}</span>
  </div>
</template>
