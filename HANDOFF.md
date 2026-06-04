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
- ✅ `go build ./...` passes.

## Next steps (plan order, Phase 1 onward)

The plan's Phase-1 step 0 (ADRs) is done. Continue:

1. **`internal/config`** — `Config` struct, atomic load/save, platform paths (`paths.go`), `DefaultConfig()` seeding the configurable `MarkerSet`. (Next up.)
2. **`internal/parser`** — `entry.go` (LogEntry + Channel consts incl. `ChannelUnknown`), `format.go` (CCLv1 format string → named-capture regex), `parser.go` (`Parse` → `ParsedLog`, never drops lines). Table-driven tests against the sample log.
3. **`internal/highlight`** — stateful RP tokenizer → flat `[]Span`; `MarkerSet`/`MarkerPair` types; mention-splitting. Tests for all default styles + a custom marker set.
4. **`internal/reassemble`** — in-memory thread reassembly (optional Raw/Reassembled view).
5. `internal/logstore`, `internal/search`, `internal/tags`, `internal/i18n`.
6. **`api/app.go`** — replace placeholder `App.Greet` with the bound methods/DTOs from the plan (`ScanLogs`, `GetLogEntries`, `GetLogThreads`, `Search`, tags, `GetLocaleMessages`); update `Bind` in `main.go`.
7. Frontend components (LogList, LogViewer, EntryRow, Search, Tags, Settings) + vue-i18n.
8. Build & smoke test.

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
