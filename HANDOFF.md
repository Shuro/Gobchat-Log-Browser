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
- ✅ **Backend core packages implemented, tested (`go test ./internal/...` green), and committed:**
  - [internal/highlight/](internal/highlight/) — configurable RP tokenizer → flat `[]Span` (speech/emote/ooc/mention).
  - [internal/parser/](internal/parser/) — CCLv1/FCLv1 format→regex, `Parse`→`ParsedLog`; sender split (symbol/name/realm); heuristic part/continuation; unmatched lines → `ChannelUnknown` (never dropped). Synthetic fixture in `internal/parser/testdata/sample.log`.
  - [internal/config/](internal/config/) — `Config` + atomic load/save + platform paths + `DefaultConfig()` seeding `MarkerSet`.
  - [internal/reassemble/](internal/reassemble/) — in-memory interrupted-thread reassembly.
- ✅ The real sample log was removed (personal/sensitive). The committed synthetic fixture covers the format patterns.

## Next steps (plan order)

1. **`internal/logstore`** — `store.go` (central registry, `LogMeta`, cache), `scanner.go` (directory scan + quick meta), `watcher.go` (fsnotify; needs `github.com/fsnotify/fsnotify`). (Next up.)
2. **`internal/search`** — lazy in-memory inverted index + query.
3. **`internal/tags`** — filename-keyed JSON sidecar CRUD (path from `config.TagsFilePath`).
4. **`internal/i18n`** — `embed.FS` locale loader for backend strings; `locales/en.json`, `de.json`.
5. **`api/app.go`** — replace placeholder `App.Greet` with bound methods/DTOs from the plan (`GetConfig`/`SaveConfig`, `ScanLogs`, `GetLogList`, `GetLogEntries`, `GetLogThreads`, `Search`, tags, `GetLocaleMessages`); wire the DTO mapping (LogEntry→EntryDTO with `highlight.Tokenize`); update `Bind` in `main.go`.
6. Frontend components (LogList, LogViewer, EntryRow, Search, Tags, Settings) + vue-i18n.
7. Build & smoke test (`wails build`).

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
