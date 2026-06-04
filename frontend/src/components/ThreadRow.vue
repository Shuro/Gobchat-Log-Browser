<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { api } from '../../wailsjs/go/models'

const { t } = useI18n()
const props = defineProps<{ thread: api.ThreadDTO }>()

const channelClass = computed(
  () => `channel-${(props.thread.channel || 'unknown').toLowerCase()}`,
)
</script>

<template>
  <div class="entry thread" :class="channelClass">
    <span class="channel-tag">{{ thread.channel }}</span>
    <span class="sender">{{ thread.sender }}</span>
    <span class="message">
      <span
        v-for="(span, i) in thread.spans"
        :key="i"
        :class="`span-${span.type}`"
        >{{ span.text }}</span
      >
    </span>
    <span v-if="thread.lines.length > 1" class="part">{{ t('viewer.parts', { count: thread.lines.length }) }}</span>
  </div>
</template>
