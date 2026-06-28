<!-- Generated: 2026-06-19 | Files scanned: go.mod + package.json | Token estimate: ~550 -->

# Dependencies

Single static binary. Go backend + Vue frontend embedded via `//go:embed
all:frontend/dist`. No runtime services or databases.

## Backend (go.mod, Go 1.24.3)

```
github.com/wailsapp/wails/v2     v2.12.0   desktop shell, bindings, events, dialogs
github.com/fsnotify/fsnotify     v1.10.1   live log-directory watching (logstore/watcher.go)
github.com/quaadgras/velopack-go v0.0.1358 install + in-app update (cgo; needs -lntdll, ADR-0013)
golang.org/x/sys                 v0.30.0   Windows registry read for NSIS migration (migrate/)
```

Everything else in go.mod is an indirect dependency pulled in by Wails (echo,
go-webview2, gorilla/websocket, golang.org/x/{crypto,net,text}, …). The app
imports only the four direct modules above plus the Go stdlib.

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
GitHub Releases       opt-in Velopack update feed (internal/velopackupd, ADR-0013):
                      releases/latest/download → releases.win.json + nupkgs.
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
Windows builds: CGO_LDFLAGS=-lntdll (velopack-go links Velopack Rust libs, ADR-0013)
Release: push semver tag vX.Y.Z → CI runs vpk pack → Setup.exe + nupkgs + portable (ADR-0013)
```
