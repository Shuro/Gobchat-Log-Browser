<!-- Generated: 2026-07-01 | Files scanned: 20 frontend | Token estimate: ~740 -->

# Frontend (Vue 3 + TS + Pinia)

Thin virtualized UI. All data comes from Wails bindings (`wailsjs/go/api/App`);
backend locale strings are merged into vue-i18n at runtime via `GetLocaleMessages`.
`vue-tsc --noEmit && vite build` (`npm run build`).

## Entry

```
main.ts → createApp(App) + Pinia + i18n + theme composable
i18n.ts → vue-i18n; merges en/de locales + backend GetLocaleMessages
```

## Component tree

```
App.vue (root: header + update banner + setup gate)
├─ SearchBar.vue            (header: query, scope all|current, channel/sender filters)
├─ main-pane
│  ├─ LogList.vue           (overview list, list events log:new/updated/removed)
│  │  └─ PlayerFilter.vue   (filter by player / #tag — FilterSelection)
│  ├─ LogViewer.vue         (vue-virtual-scroller; raw vs reassembled)
│  │  ├─ EntryRow.vue       (one EntryDTO; renders highlight Spans)
│  │  ├─ ThreadRow.vue      (one reassembled ThreadDTO)
│  │  └─ TagEditor.vue      (tags + note for current log)
│  └─ SearchResults.vue     (overlay; SearchResultDTO list, truncated notice)
├─ SettingsPanel.vue        (v-if showSettings; dirs incl. DetectedLogDirs() auto-detect
│                            list, language, theme, markers, colors, update opt-in,
│                            hide-empty-player-logs toggle)
└─ SetupWizard.vue          (v-if setupState.needs_setup; first-run / re-shown on version bump)
```

## State (Pinia stores)

```
stores/logs.ts    summaries[], selectedPath, entries[] (raw), threads[] (reassembled),
                  viewMode 'raw'|'reassembled', targetLine (scroll-to from search),
                  tags/note; openLog(), refreshList(), setTags(); FilterSelection.
                  visibleSummaries getter applies hide_empty_player_logs (config) and
                  player/tag filter. Persists view.excludedChannels to localStorage.
stores/search.ts  query, scope 'all'|'current', channel, sender, results[], truncated,
                  visible, openedHit; run() tags each request and drops the response if
                  a newer run() has since started (stale-response guard) → App.Search;
                  openHit() sets logs.targetLine.
stores/config.ts  cfg (api.Config), load/save → App.GetConfig/SaveConfig.
```

## State flow

```
Open log:   LogList → logsStore.openLog → App.GetLogEntries → entries → EntryRow
Reassemble: viewMode='reassembled' → App.GetLogThreads → threads → ThreadRow
Search:     SearchBar → searchStore.run → App.Search → SearchResults
            click hit → openHit → logsStore.openLog + targetLine → LogViewer scrolls
Live:       EventsOn('logs:scanned'|'log:new'|'log:updated'|'log:removed') → refresh
Settings:   SettingsPanel → configStore.save → App.SaveConfig (may rescan/re-i18n)
```

## Utils / composables

```
composables/theme.ts   light|dark + per-theme highlight color overrides → CSS vars
utils/rowHeights.ts     virtual-scroller row-height estimation
utils/findMatches.ts    in-view match highlighting for the current query
```

## Conventions

- `wailsjs/**` is generated — never hand-edit; run `wails generate module`.
- `locales/en.json` + `de.json` must change together (same keys).
- DTO field names are snake_case (Go json tags); TS types from `wailsjs/go/models`.
