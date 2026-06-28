# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Cross-platform desktop app for browsing Final Fantasy XIV roleplay chat logs produced by
[Gobchat](https://github.com/MarbleBag/Gobchat). Go + Wails v2 backend, Vue 3 + TypeScript +
Pinia frontend, shipped as a single static binary. Primary target is Windows; code stays
platform-agnostic. UI and code comments in English; UI is localized en/de.

## Commands

PATH does not persist between tool calls and Go/Wails/npm are not on PATH in fresh shells.
Use the **PowerShell** tool (not Bash) for toolchain commands, prefixed with:

```powershell
$env:Path = [Environment]::GetEnvironmentVariable("Path","Machine")+";"+[Environment]::GetEnvironmentVariable("Path","User")+";C:\Users\Shuro\go\bin"
```

- Dev with hot reload: `wails dev`
- Production build: `wails build` → `build/bin/gobchat-log-browser.exe`
- Backend tests: `go test ./...` — single test: `go test ./internal/parser -run TestName`
- Lint/vet: `go vet ./...`, `gofmt -l .`
- Frontend type-check + build: `cd frontend && npm run build` (runs `vue-tsc --noEmit && vite build`)
- Regenerate frontend bindings: `wails generate module` — required after changing
  `config.Config` or any DTO, and it must run *before* the frontend type-check.
- **Windows builds need `CGO_LDFLAGS=-lntdll`** (env var, not a `#cgo` directive) — the
  velopack-go binding links Velopack's Rust libs which reference ntdll. Set it for every
  `wails dev` / `wails build` / `go test` / `go build` on Windows (ADR-0013).
- Release: push a semver tag `vX.Y.Z` → CI runs `vpk pack` (Velopack) and publishes the
  GitHub release with `Setup.exe`, full/delta `.nupkg`, a portable zip, and
  `releases.win.json` (`.github/workflows/release.yml`, ADR-0013).

## Hard constraints

- **Log files are strictly read-only.** Never write, rename, move, or modify them. Reassembly
  and reordering are in-memory display only (ADR-0007).
- **All `{message}` content is player-authored.** Split markers (`(N/M)`) and the
  leading/trailing continuation markers (`>`, `->`, `>>`, `+` — see ADR-0006), OOC `((…))`,
  and speech/emote quotes are player RP conventions — treat them as best-effort heuristics,
  never guaranteed structure. RP delimiters stay configurable (ADR-0006).
- Lines that fail to parse must still surface as `ChannelUnknown` with raw text — never drop lines.
- **Architectural decisions must be recorded as ADRs** in `docs/adr/` (Nygard template,
  `docs/adr/0000-template.md`), committed *with* the code change, not after.
- `frontend/wailsjs/**` is generated — never hand-edit; regenerate via `wails generate module`.
- The frontend locale files `frontend/src/locales/en.json` and `de.json` must always change
  together (same keys in both).

## Log format

```
Chatlogger Id: CCLv1
Chatlogger format:{channel} [{date} {time-full}] {sender}: {message}
Say [2026-05-16 20:09:30+02:00] ★Max Mustermiqote [Shiva]: "Hello..." (1/2)
```

Line 1 is `Chatlogger Id: FCLv1|CCLv1`; CCLv1 adds a format line. Sender may carry a leading
status symbol (`★`, `♥`, …) and a `[Realm]` suffix. Filenames: `chatlog_YYYY-MM-DD_HH-mm.log`,
default dir `%APPDATA%\Gobchat\log`. App data (tags, notes, config, metadata cache) lives
separately in `%APPDATA%\GobchatLogBrowser`.

## Architecture

The Go backend does everything heavy; the frontend is a thin virtualized UI connected through
Wails bindings.

- `internal/parser` — CCLv1/FCLv1 format→regex, sender split (symbol/name/realm), heuristic
  part/continuation detection; synthetic fixtures in `testdata/`
- `internal/highlight` — configurable RP tokenizer → flat `[]Span` (speech/emote/ooc/mention)
- `internal/reassemble` — in-memory interrupted-thread reassembly
- `internal/search` — lazy in-memory inverted index, AND queries
- `internal/logstore` — registry, recursive scanner, fsnotify watcher, persistent JSON
  metadata cache for fast startup (ADR-0009)
- `internal/tags` — filename-keyed JSON sidecars (tags + notes)
- `internal/config` — config + atomic load/save + platform paths
- `internal/i18n` — embedded backend localizer (en/de)
- `internal/version` — app version, injected at release time via ldflags (dev builds: "dev")
- `internal/velopackupd` — Velopack-backed update check + in-app download/apply, gated on the
  `check_updates_on_start` opt-in; no-op on `dev` builds (ADR-0013)
- `internal/migrate` — one-shot first-run detect + silent uninstall of the legacy NSIS
  install (Windows only; no-op stub elsewhere) (ADR-0013)
- `api/` — the Wails binding layer (`App` + DTOs in `dto.go`); the only surface the frontend
  calls. Emits `logs:scanned`, `log:new|updated|removed`, and `update:progress` events.
- `frontend/src` — Vue 3 + Pinia + vue-i18n + vue-virtual-scroller; backend locale strings
  are merged at runtime via `GetLocaleMessages`.

Design rationale lives in `docs/adr/` — read the relevant ADR before changing
parsing, reassembly, search, or storage behavior.
