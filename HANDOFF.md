# Session Handoff — Gobchat Log Browser

> Purpose: give a fresh Claude Code session everything needed to continue building this project. Read this first, then the plan and the ADRs.

## What this project is

A cross-platform desktop GUI to open and display Final Fantasy 14 roleplay chat logs from the **Gobchat** tool (https://github.com/MarbleBag/Gobchat). It provides a structured log overview, a readable viewer with roleplay (RP) highlighting, and global + per-log search. Primary target Windows; Linux/macOS later. Multilingual (DE/EN) and modular.

**Stack:** Go + Wails v2 (backend) + Vue 3 + Vite + TypeScript + Pinia (frontend), shipped as a single static binary.

## Source of truth — read these

1. **Approved plan:** `C:\Users\Shuro\.claude\plans\ich-plane-ein-gui-programm-purring-sundae.md` — full architecture, data models, API surface, phases, verification. **Follow this plan.**
2. **ADRs:** [docs/adr/](docs/adr/) — 0001–0007 capture the *why* behind the structure. New architectural decisions MUST be recorded as a new ADR (Nygard template, `docs/adr/0000-template.md`), added *with* the code change, not after.
3. **Sample log:** `chatlog_2026-05-16_20-01.log` — real Gobchat output for parser/highlight tests.
4. **Project rules:** [CLAUDE.md](CLAUDE.md) — 12-rule template applies to all work.

## Hard constraints (do not violate)

- **Log files are STRICTLY read-only.** Never write, rename, move, or modify them. Any reassembly/reordering is **in-memory display only** (see ADR-0007).
- **All `{message}` content is player-authored**, not Gobchat. Split markers (`(1/2)`, `1/3`, trailing ` >`, leading `> `/`"> `, or none), OOC `((…))`, speech/emote quotes are player RP conventions — treat as **best-effort heuristics**, never guaranteed structure. RP delimiters must stay **configurable** (see ADR-0006).
- **Code and comments in English.**

## Log format quick reference

```
Chatlogger Id: CCLv1
Chatlogger format:{channel} [{date} {time-full}] {sender}: {message}
Say [2026-05-16 20:09:30+02:00] ★M'iqo Tester [Shiva]: "Hello..." (1/2)
```
- Line 1: `Chatlogger Id: FCLv1|CCLv1`. Line 2 (CCLv1 only): `Chatlogger format:{...}`.
- Default format: `{channel} [{date} {time-full}] {sender}: {message}`.
- Filename: `chatlog_YYYY-MM-DD_HH-mm.log`. Default dir: `%APPDATA%\Gobchat\log`.
- Sender may carry a leading status symbol (`★`, `♥`, …) and a `[Realm]` suffix. A line that fails to match must still surface as `ChannelUnknown` with raw text — never drop lines.

## Current state (what's done)

- ✅ Toolchain installed: **Go 1.26.4**, **Node 24.16.0**, **npm 11.13.0**, **Wails CLI v2.12.0**. `wails doctor` clean (WebView2 present).
- ✅ Seven ADRs + template written under [docs/adr/](docs/adr/).
- ✅ Wails Vue-TS project scaffolded and merged into repo root (module `gobchat-log-browser`). Standard scaffold: [main.go](main.go), [app.go](app.go) (still has placeholder `Greet`), [frontend/](frontend/), [wails.json](wails.json).
- ✅ Git hygiene: [.gitignore](.gitignore) (sensitive `*.log` excluded; `testdata/*.log` exempt) and [.gitattributes](.gitattributes) (LF). History is committed in logical commits.
- ✅ **Entire Go backend implemented, tested (`go test ./...` green), vetted, and committed:**
  - [internal/highlight/](internal/highlight/) — configurable RP tokenizer → flat `[]Span` (speech/emote/ooc/mention).
  - [internal/parser/](internal/parser/) — CCLv1/FCLv1 format→regex, `Parse`→`ParsedLog`; sender split (symbol/name/realm); heuristic part/continuation; unmatched lines → `ChannelUnknown` (never dropped). Synthetic fixture in `internal/parser/testdata/`.
  - [internal/config/](internal/config/) — `Config` + atomic load/save + platform paths + `DefaultConfig()` seeding `MarkerSet`.
  - [internal/reassemble/](internal/reassemble/) — in-memory interrupted-thread reassembly.
  - [internal/tags/](internal/tags/) — filename-keyed JSON sidecar (tags + notes).
  - [internal/search/](internal/search/) — lazy in-memory inverted index, AND queries.
  - [internal/logstore/](internal/logstore/) — registry + scanner + fsnotify watcher (`github.com/fsnotify/fsnotify`).
  - [internal/i18n/](internal/i18n/) — embedded backend localizer (en/de).
  - [api/](api/) — Wails binding layer: `App` + DTOs in [api/dto.go](api/dto.go); methods `GetConfig`/`SaveConfig`, `ScanLogs`, `GetLogList`, `GetLogEntries`, `GetLogThreads`, `Search`, `GetTags`/`SetTags`/`GetAllTagNames`, `GetLocaleMessages`. Emits `log:new`/`log:updated`/`log:removed` events. [main.go](main.go) binds `api.App` (scaffold placeholder removed).
- ✅ The real sample log was removed (personal/sensitive). The committed synthetic fixtures cover the format patterns.
- ✅ `go build ./...`, `go vet ./...`, `gofmt -l` all clean.

## Next steps (frontend)

Backend is feature-complete for v1. Remaining work is the Vue 3 frontend:

1. **Regenerate Wails bindings** — `wails generate module` (or just `wails dev`) refreshes `frontend/wailsjs/` from the new `api.App` (currently still the scaffold `App.Greet`).
2. **Install frontend deps** — `pinia`, `vue-i18n`, `vue-virtual-scroller` (in `frontend/`, `npm install <pkg>`).
3. **Components** (`frontend/src/components/`): `LogList.vue` (overview: date, participants, message count, duration, tags), `LogViewer.vue` (virtual-scrolled), `EntryRow.vue` (renders `spans` by type; channel styling; Raw/Reassembled toggle calling `GetLogThreads`), `SearchBar.vue` + `SearchResults.vue` (per-log linear filter vs. global `Search`), `TagEditor.vue`, `SettingsPanel.vue` (directories, language, mention names, markers, theme).
4. **i18n** — vue-i18n with `frontend/src/locales/{en,de}.json`; merge backend strings from `GetLocaleMessages()` at startup.
5. **Listen** for `log:new`/`log:updated`/`log:removed` runtime events to refresh the list/open log live.
6. **Build & smoke test** — `wails dev` to iterate, `wails build` for the Windows binary.

Backend call contract is stable; the frontend only consumes the DTOs in [api/dto.go](api/dto.go).

## Toolchain gotchas for the next session

- **PATH does not persist** between tool calls, and freshly-installed binaries aren't on PATH in new PowerShell sessions. Prefix PowerShell commands that need Go/Wails/npm with:
  ```powershell
  $env:Path = [Environment]::GetEnvironmentVariable("Path","Machine")+";"+[Environment]::GetEnvironmentVariable("Path","User")+";C:\Users\Shuro\go\bin"
  ```
- `GOPATH` = `C:\Users\Shuro\go`; Wails CLI at `C:\Users\Shuro\go\bin\wails.exe`.
- Dev run: `wails dev`. Production build: `wails build`. Go tests: `go test ./...`.
- The **Bash** tool is a separate (POSIX) environment where `go`/`node` are NOT on PATH — use the **PowerShell** tool for toolchain commands.

## Memory (already persisted)

Project memory in `…\memory\`: `gobchat-log-browser-project.md` (goal/stack/constraints) and `document-decisions-as-adrs.md` (ADR workflow). Indexed in `MEMORY.md`.
