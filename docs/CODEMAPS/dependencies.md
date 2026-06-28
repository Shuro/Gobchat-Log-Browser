<!-- Generated: 2026-06-19 | Files scanned: go.mod + package.json | Token estimate: ~550 -->

# Dependencies

Single static binary. Go backend + Vue frontend embedded via `//go:embed
all:frontend/dist`. No runtime services or databases.

## Backend (go.mod, Go 1.23)

```
github.com/wailsapp/wails/v2   v2.12.0   desktop shell, bindings, events, dialogs
github.com/fsnotify/fsnotify   v1.10.1   live log-directory watching (logstore/watcher.go)
```

Everything else in go.mod is an indirect dependency pulled in by Wails (echo,
go-webview2, gorilla/websocket, golang.org/x/{crypto,net,sys,text}, …). The app
imports only the two direct modules above plus the Go stdlib.

## Frontend (package.json)

```
vue                    ^3.2.37    UI framework (Composition API)
pinia                  ^3.0.4     state management (logs/search/config stores)
vue-i18n               ^9.14.5    i18n; merges backend GetLocaleMessages at runtime
vue-virtual-scroller   ^2.0.0-b8  virtualized log/thread rendering (LogViewer)
@vueuse/core           ^14.3.0    composable utilities
-- dev --
vite ^3 · @vitejs/plugin-vue ^3 · typescript ^5.9 · vue-tsc ^2.2  (build: vue-tsc --noEmit && vite build)
```

## External services

```
GitHub Releases API   opt-in update check only (internal/update, ADR-0012).
                      Default off; dev builds skip the network call entirely.
                      No telemetry, no other network access.
```

## Internal shared packages

```
internal/parser       imported by reassemble, logstore, search, api
internal/highlight    imported by api, config (DefaultMarkerSet)
internal/config       imported by api, main (paths + Config)
internal/search       imported by logstore, api
internal/i18n         embedded en/de locale data (go:embed)
```

## Toolchain (not on PATH in fresh shells — prefix per CLAUDE.md)

```
Go 1.23 · Wails v2 CLI (wails dev|build|generate module) · Node/npm (frontend)
Release: push semver tag vX.Y.Z → CI builds installer + portable zip (ADR-0011)
```
