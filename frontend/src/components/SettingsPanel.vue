<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useConfigStore } from '../stores/config'

const emit = defineEmits<{ (e: 'close'): void }>()
const config = useConfigStore()

onMounted(() => {
  if (!config.cfg) config.load()
})

const markerCategories: Array<{ key: 'speech' | 'emote' | 'ooc'; label: string }> = [
  { key: 'speech', label: 'Speech' },
  { key: 'emote', label: 'Emote' },
  { key: 'ooc', label: 'Out-of-character' },
]

// Mention names edited as one-per-line text.
const mentionsText = computed<string>({
  get: () => config.cfg?.mention_names.join('\n') ?? '',
  set: (v: string) => {
    if (config.cfg) {
      config.cfg.mention_names = v
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
    }
  },
})

function addPair(key: 'speech' | 'emote' | 'ooc') {
  config.cfg?.markers[key].push({ open: '', close: '' } as any)
}
function removePair(key: 'speech' | 'emote' | 'ooc', i: number) {
  config.cfg?.markers[key].splice(i, 1)
}

async function save() {
  await config.save()
  emit('close')
}
</script>

<template>
  <div class="settings-backdrop" @click.self="emit('close')">
    <div class="settings-panel">
      <header class="settings-header">
        <strong>Settings</strong>
        <button class="ghost" @click="emit('close')">✕</button>
      </header>

      <div v-if="!config.cfg" class="placeholder">Loading…</div>

      <div v-else class="settings-body">
        <!-- Directories -->
        <section>
          <h3>Log directories</h3>
          <label class="check">
            <input type="checkbox" v-model="config.cfg.auto_detect_appdata" />
            Auto-detect the Gobchat log folder
          </label>
          <ul class="dir-list">
            <li v-for="d in config.cfg.log_directories" :key="d">
              <span class="dir-path">{{ d }}</span>
              <button class="ghost" @click="config.removeDirectory(d)">Remove</button>
            </li>
            <li v-if="config.cfg.log_directories.length === 0" class="muted">
              No extra directories. Auto-detect covers the default location.
            </li>
          </ul>
          <button @click="config.addDirectory()">Add directory…</button>
        </section>

        <!-- Appearance & language -->
        <section class="grid-2">
          <div>
            <h3>Theme</h3>
            <select v-model="config.cfg.theme">
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </div>
          <div>
            <h3>Language</h3>
            <select v-model="config.cfg.language">
              <option value="en">English</option>
              <option value="de">Deutsch</option>
            </select>
          </div>
        </section>

        <!-- Mentions -->
        <section>
          <h3>Highlight my names (mentions)</h3>
          <textarea
            v-model="mentionsText"
            class="mentions"
            rows="3"
            placeholder="One name per line…"
          ></textarea>
        </section>

        <!-- RP markers -->
        <section>
          <h3>Roleplay markers</h3>
          <p class="muted">
            Delimiters used to detect speech, emotes and OOC text. Defaults match
            Gobchat; add your own as needed.
          </p>
          <div v-for="cat in markerCategories" :key="cat.key" class="marker-group">
            <div class="marker-group-head">
              <span>{{ cat.label }}</span>
              <button class="ghost" @click="addPair(cat.key)">+ Add</button>
            </div>
            <div
              v-for="(pair, i) in config.cfg.markers[cat.key]"
              :key="i"
              class="marker-row"
            >
              <input v-model="pair.open" class="marker-input" placeholder="open" />
              <input v-model="pair.close" class="marker-input" placeholder="close" />
              <button class="ghost" @click="removePair(cat.key, i)">✕</button>
            </div>
          </div>
        </section>
      </div>

      <footer class="settings-footer">
        <button class="ghost" @click="emit('close')">Cancel</button>
        <button :disabled="config.saving || !config.cfg" @click="save">
          {{ config.saving ? 'Saving…' : 'Save' }}
        </button>
      </footer>
    </div>
  </div>
</template>
