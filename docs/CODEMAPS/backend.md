<!-- Generated: 2026-06-19 | Files scanned: 34 Go | Token estimate: ~850 -->

# Backend (Go)

No HTTP routes — the "API" is the Wails-bound `api.App` method set. Wails
serializes each method's args/return to JSON and generates TS bindings
(`frontend/wailsjs/**`, never hand-edited).

## Bound methods → service chain

```
GetVersion()                       → version.Version
CheckForUpdate()                   → velopackupd.Check (dev → "dev"; else UpdateManager)
DownloadAndApplyUpdate()           → velopackupd.DownloadAndApply(emit update:progress) → wruntime.Quit
GetConfig()                        → cfg (RLock)
SaveConfig(cfg)                    → config.Save → (lang) i18n.New → (dirs) store.ScanAll + startWatcher
ScanLogs()                         → store.ScanAll(effectiveDirs) → GetLogList
GetLogList()                       → store.List → summary(meta)+tags
GetLogEntries(path)                → store.GetEntries → parser.Parse → toEntryDTO + highlight.Tokenize
GetLogThreads(path)                → store.GetEntries → reassemble.Reassemble → highlight.Tokenize
Search(text,path,channels,sender)  → store.GetEntries(ensure) → index.Query(pool=1000) → post-filter → cap=200
GetTags(file) / SetTags / GetAllTagNames → tags.TagStore
GetSetupState()                    → config.Gobchat*Dir → needsSetup()
PickDirectory()                    → wails OpenDirectoryDialog
GetLocaleMessages()                → i18n.Localizer.Messages
Startup(ctx) / Shutdown(ctx)       → Wails lifecycle (load, scan goroutine, watcher,
                                     migrate.CleanLegacyNSISInstall goroutine)
```

Emitted events: `logs:scanned`, `log:new` (LogSummary), `log:updated` (path),
`log:removed` (path), `update:progress` (uint percent).

## Package map (`internal/`)

```
parser/       format.go  CCLv1/FCLv1 format-line → regex (FormatVersion)
              parser.go  Parse → ParsedLog{Entries, ParseErrors}; never drops a line
              entry.go   LogEntry{Channel, Sender/DisplayName/Realm/StatusSymbol,
                         Message, PartIndex/Total, Is/HasContinuation}; StripPrivateUse
highlight/    highlight.go  Tokenize(msg, MarkerSet, mentions) → []Span
                         (plain/speech/emote/ooc/mention); DefaultMarkerSet
reassemble/   reassemble.go  Reassemble([]LogEntry) → []Thread; sender-keyed,
                         maxGap=15m; in-memory only (ADR-0007)
search/       search.go  Index (inverted, RWMutex): AddEntries, Query(text,file,limit),
                         HasFile; tokenize = lowercased letter/digit runs, no stopwords
logstore/     store.go     LogStore: ScanAll, List, GetEntries(lazy+cache), Refresh,
                           Get, WatchDirs; feeds search.Index
              scanner.go   ScanDirectory (recursive), LogMeta{counts, participants,
                           channels, duration, folder}
              watcher.go   Watch(dirs, cb) fsnotify → WatchEvent Created/Modified/Removed
              metacache.go MetaCache: persistent JSON index for fast startup (ADR-0009)
tags/         tags.go      TagStore: filename-keyed JSON sidecars (FileTags{Tags,Note})
config/       config.go    Config + Load/Save (atomic JSON); SetupWizardCurrentVersion=2
              paths.go     platform paths (AppDataDir, ConfigFilePath, GobchatDefaultLogDir…)
i18n/         i18n.go      embedded en/de localizer: New(lang), T(key), Messages()
version/      version.go   Version (ldflags at release; "dev" locally)
velopackupd/  updater.go   Check / DownloadAndApply(progress) over GitHub releases feed;
                           not-installed → "dev" (ADR-0013)
migrate/      nsis_windows.go  one-shot detect + silent uninstall of legacy NSIS install;
                           nsis_other.go no-op stub (ADR-0013)
```

## Concurrency

- `App.mu` (RWMutex) guards cfg/index/store/tags/loc; `watcherMu` serializes
  watcher swap; `debounceMu` + per-path timers coalesce Write bursts (300ms).
- `LogStore.parseMu` prevents double parse/index of one path; `search.Index.mu`
  guards postings. Scan + initial watcher start run off the Startup goroutine so
  the window paints immediately.

## Key files

```
api/app.go (orchestrator, ~650 lines)   api/dto.go (wire contract)
internal/parser/parser.go               internal/logstore/store.go
main.go (velopack.Run first, wails.Run, embeds frontend/dist, webview2 data → app dir ADR-0010)
```

## Commands

`go test ./...` · `go vet ./...` · `gofmt -l .` · `wails build` ·
`wails generate module` (after any DTO/Config change, before frontend type-check).
PATH not persisted — prefix per CLAUDE.md.
