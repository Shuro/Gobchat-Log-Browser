# Contributing

Thanks for your interest in Gobchat Log Browser — a cross-platform desktop app
for browsing Final Fantasy XIV roleplay chat logs from
[Gobchat](https://github.com/MarbleBag/Gobchat). It's a Go + Wails v2 backend
with a Vue 3 + TypeScript + Pinia frontend, shipped as a single static binary.

Primary target is **Windows**; the code stays platform-agnostic (Linux/macOS
builds should work but are currently untested).

## Prerequisites

<!-- AUTO-GENERATED: sourced from go.mod, wails.json, frontend/package.json, README.md -->

| Tool | Version | Notes |
|------|---------|-------|
| [Go](https://go.dev/dl/) | 1.24+ | backend (go.mod toolchain: 1.24.3) |
| [Node.js](https://nodejs.org/) + npm | current LTS | frontend build |
| [Wails CLI v2](https://wails.io/docs/gettingstarted/installation) | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| [WebView2 runtime](https://developer.microsoft.com/microsoft-edge/webview2/) | — | runtime only (preinstalled on Win 11 / most Win 10) |

> **Windows note:** Go, Wails, and npm are often not on `PATH` in a fresh shell.
> If a command isn't found, restart the shell after install, or prepend the
> machine + user `PATH` (see [CLAUDE.md](../CLAUDE.md) for the exact one-liner).

## Development

<!-- AUTO-GENERATED: sourced from wails.json + frontend/package.json scripts + CLAUDE.md -->

| Command | Description |
|---------|-------------|
| `wails dev` | Run the app with hot reload (backend + frontend) |
| `wails build` | Production build → `build/bin/gobchat-log-browser.exe` |
| `wails generate module` | Regenerate frontend bindings — **required** after changing `config.Config` or any DTO, and must run *before* the frontend type-check |
| `go test ./...` | Run all backend tests |
| `go test ./internal/parser -run TestName` | Run a single backend test |
| `go vet ./...` | Backend static analysis |
| `gofmt -l .` | List unformatted Go files (should print nothing) |
| `cd frontend && npm run build` | Frontend type-check + build (`vue-tsc --noEmit && vite build`) |
| `cd frontend && npm run dev` | Vite dev server (normally driven by `wails dev`) |

> **Windows build note:** set `CGO_LDFLAGS=-lntdll` (an env var, not a `#cgo`
> directive) for every `wails dev` / `wails build` / `go test` / `go build` on
> Windows — the velopack-go binding links Velopack's Rust libs, which reference
> ntdll's `Nt*` syscalls (ADR-0013).

<!-- END AUTO-GENERATED -->

## Testing

- **Backend:** Go table-driven unit tests live next to the code they cover
  (`internal/**/*_test.go`, `api/setup_test.go`), with synthetic fixtures under
  `internal/parser/testdata/`. Run `go test ./...`. Tests should encode *why* a
  behavior matters (e.g. unparseable lines must still surface, never be dropped),
  not just what it does.
- **Frontend:** there is no JS test runner configured. The frontend is verified
  by type-check + build (`cd frontend && npm run build`); keep it green.

Add tests with any backend behavior change. Use real fixtures over mocks where
the parser/reassembly/search heuristics are involved.

## Code style

- **Go:** keep `gofmt -l .` clean and `go vet ./...` green.
- **Naming & comments:** UI and code comments in English.
- **Localization:** `frontend/src/locales/en.json` and `de.json` must always
  change **together** with the same keys; backend strings live in `internal/i18n`.
- **Generated code:** never hand-edit `frontend/wailsjs/**` — regenerate via
  `wails generate module`.
- **Small, focused files** and explicit error handling are preferred (see
  [.claude/rules/ecc/common/coding-style.md](../.claude/rules/ecc/common/coding-style.md)).

## Hard constraints

These are non-negotiable; PRs that violate them won't be merged:

- **Log files are strictly read-only** — never write, rename, move, or modify
  them. Reassembly/reordering is in-memory display only (ADR-0007).
- **All `{message}` content is player-authored.** Split markers, continuation
  markers (`>`, `->`, `>>`, `+`), OOC `((…))`, and speech/emote quotes are
  best-effort RP heuristics, never guaranteed structure, and stay configurable
  (ADR-0006).
- **Never drop a line** — lines that fail to parse surface as `ChannelUnknown`
  with their raw text.

## Architectural decisions (ADRs)

Architectural decisions must be recorded as ADRs in [docs/adr/](adr/) using the
[Nygard template](adr/0000-template.md), committed **with** the code change —
not after. Read the relevant ADR before changing parsing, reassembly, search, or
storage behavior.

## Commit & PR

- Use [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`, `perf:`, `ci:`.
- Before opening a PR, confirm:
  - [ ] `go test ./...` passes
  - [ ] `go vet ./...` and `gofmt -l .` are clean
  - [ ] `cd frontend && npm run build` succeeds (after `wails generate module` if DTOs/Config changed)
  - [ ] locale files changed together (en + de) when UI strings changed
  - [ ] an ADR is included for any architectural change
  - [ ] no log files were written or modified

## Releases

Releases are automated: push a semver tag `vX.Y.Z` and CI
([.github/workflows/release.yml](../.github/workflows/release.yml)) runs `vpk pack`
(Velopack) and publishes the GitHub release with `Setup.exe`, full/delta `.nupkg`
packages, a portable zip, and `releases.win.json` — the feed the in-app updater
reads (ADR-0013).

## License

By contributing you agree your contributions are licensed under the
[Apache License 2.0](../LICENSE).
