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
}>()

const channelClass = computed(
  () => `channel-${(props.thread.channel || 'unknown').toLowerCase()}`,
)

const spanSegments = computed(() =>
  props.thread.spans.map((s) => splitMatches(s.text, props.query ?? '')),
)
const nameSegments = computed(() =>
  splitMatches(props.thread.sender, props.messageOnly ? '' : props.query ?? ''),
)
</script>

<template>
  <div class="entry thread" :class="[channelClass, { target: highlight }]">
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
