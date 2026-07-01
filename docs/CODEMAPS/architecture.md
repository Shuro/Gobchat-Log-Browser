<!-- Generated: 2026-07-01 | Files scanned: 34 Go + 20 frontend | Token estimate: ~660 -->

# Architecture

Cross-platform desktop app (Wails v2) for browsing FFXIV Gobchat RP chat logs.
Go backend does all heavy lifting; Vue 3 frontend is a thin virtualized UI.
Single static binary. Logs are strictly read-only (ADR-0007).

## System diagram

```
            Wails v2 binding (JSON DTOs + events)
 ┌───────────────────────────┐        ┌──────────────────────────────┐
 │ Frontend (Vue 3 + Pinia)  │  call  │ api.App  (api/app.go)        │
 │  App.vue                  │ ─────► │  exported methods = the only │
 │  stores: logs/search/cfg  │        │  surface the frontend calls  │
 │  vue-virtual-scroller     │ ◄───── │  emits logs:* / log:* events │
 └───────────────────────────┘ events └──────────────┬───────────────┘
                                                      │ orchestrates
   ┌──────────────┬──────────────┬───────────────┬────┴─────┬───────────┐
   ▼              ▼              ▼               ▼          ▼           ▼
 parser       highlight     reassemble        search    logstore     tags
 (format→     (RP token →   (in-memory       (inverted  (registry,   (JSON
  regex,       []Span)       thread join)     index)     scan,        sidecars)
  entries)                                               watch,
                                                         metacache)
   ▲                                                      ▲
   └──────────── config (settings + paths) ── i18n ── version ── update
```

## Data flow (open a log)

```
LogList click → logsStore.openLog(path)
  → App.GetLogEntries(path)
      → store.GetEntries → parser.Parse (read-only, cached)
      → highlight.Tokenize(message, markers, mentions) per entry
  → []EntryDTO → EntryRow (virtual scroll)
Reassembled view → App.GetLogThreads → reassemble.Reassemble → []ThreadDTO
```

## Data flow (search)

```
SearchBar → searchStore.run() → App.Search(text, filePath, channels, sender)
  → ensure files parsed+indexed → index.Query (pool 1000)
  → channel/sender post-filter → cap 200 → SearchResponse{Results, Truncated}
```

## Startup / live updates

```
main.go → wails.Run(Bind: App) → App.Startup
  → load config+tags, build index+store, LoadMetaCache
  → goroutine: store.ScanAll(effectiveDirs) → startWatcher → emit "logs:scanned"
  → goroutine: migrate.CleanLegacyNSISInstall (one-shot, Windows only, ADR-0013)
fsnotify Write/Create/Remove → onFileChange (300ms debounce on Write)
  → store.Refresh → emit log:new | log:updated | log:removed
```

## Boundaries

- **Wails binding** (`api/`): the *only* backend surface the frontend sees. DTOs
  in `dto.go`; times are ISO 8601 strings, durations pre-formatted.
- **internal/** packages: pure, dependency-light, unit-tested with `testdata/`.
- **Persistence**: JSON files only (config, tag sidecars, metadata cache) — no DB.

See `docs/adr/` for rationale. Read the relevant ADR before changing parsing,
reassembly, search, or storage behavior.
