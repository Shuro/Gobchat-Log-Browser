<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsStore } from '../stores/logs'

const { t } = useI18n()
const store = useLogsStore()

const draftTags = ref<string[]>([])
const draftNote = ref('')
const newTag = ref('')
const dirty = ref(false)

// Reset the draft whenever the selected log's tags change (e.g. after openLog).
watch(
  () => store.currentTags,
  (ft) => {
    draftTags.value = [...(ft.tags ?? [])]
    draftNote.value = ft.note ?? ''
    dirty.value = false
  },
  { immediate: true, deep: true },
)

function addTag() {
  const t = newTag.value.trim()
  if (t && !draftTags.value.includes(t)) {
    draftTags.value.push(t)
    dirty.value = true
  }
  newTag.value = ''
}

function removeTag(t: string) {
  draftTags.value = draftTags.value.filter((x) => x !== t)
  dirty.value = true
}

async function save() {
  await store.saveTags(draftTags.value, draftNote.value.trim())
  dirty.value = false
}
</script>

<template>
  <div class="tag-editor">
    <div class="tag-chips">
      <span v-for="t in draftTags" :key="t" class="tag removable" @click="removeTag(t)">
        {{ t }} <span class="x">✕</span>
      </span>
      <!-- Blur commits the typed text as a tag too; it fires before a click on
           Save, so a pending tag is included in that save. (If Save was still
           disabled, the blur-added chip enables it for a second click.) -->
      <input
        v-model="newTag"
        class="tag-input"
        list="all-tags"
        :placeholder="t('tags.addTag')"
        @keyup.enter="addTag"
        @blur="addTag"
      />
      <datalist id="all-tags">
        <option v-for="t in store.allTagNames" :key="t" :value="t" />
      </datalist>
    </div>
    <input
      v-model="draftNote"
      class="note-input"
      :placeholder="t('tags.note')"
      @input="dirty = true"
    />
    <button :disabled="!dirty" @click="save">{{ t('tags.save') }}</button>
  </div>
</template>
