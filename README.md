<div align="center">

<img src="build/appicon.png" alt="Gobchat Log Browser icon" width="128" height="128">

# Gobchat Log Browser

**Browse and search your Final Fantasy XIV roleplay chat logs from [Gobchat](https://github.com/MarbleBag/Gobchat).**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2-DF0000?logo=wails&logoColor=white)](https://wails.io/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue)](LICENSE)

🇩🇪 [Deutsche Version](README.de.md)

</div>

---

## About

[Gobchat](https://github.com/MarbleBag/Gobchat) is a chat overlay for Final Fantasy XIV that can log your conversations to plain-text files. Over time those `chatlog_*.log` files pile up — and finding *that one scene* from three months ago becomes a chore.

**Gobchat Log Browser** is a desktop app that turns those log files into a searchable, filterable archive:

- It auto-detects Gobchat's log folder (`%APPDATA%\Gobchat\log`) or any folders you add yourself.
- Your log files are treated as **strictly read-only** — the app never modifies, moves, or rewrites them.
- All extras (tags, notes, settings, metadata cache) are stored separately in `%APPDATA%\GobchatLogBrowser`.

## Screenshots

| Log list & filters | Log viewer |
| --- | --- |
| ![Log list](docs/screenshots/log-list.png) | ![Log viewer](docs/screenshots/log-viewer.png) |
| **Search** | **Settings** |
| ![Search](docs/screenshots/search.png) | ![Settings](docs/screenshots/settings.png) |

*Captured with the AI-generated [mock logs](docs/examples/mock-logs/) — no real player data.*

## Features

- **Log overview** — all logs at a glance with date, participants, message count, and duration.
- **Roleplay highlighting** — dialogue, emotes, and out-of-character text are color-coded; the marker characters are configurable and default to Gobchat's conventions.
- **Raw & reassembled views** — view a log in its original file order, or let the app stitch interrupted multi-part messages (`(1/2)`, trailing `>`, `->`, `>>`, `+`, …) back together, with each post's start and end time. Reassembly is a best-effort heuristic and happens purely in memory — files are never changed.
- **Search everywhere** — full-text search across all logs, plus find-in-log with match navigation (Enter / Shift+Enter) and match ticks on the scrollbar.
- **Player & tag filter** — filter the log list by participants and `#tags` (combined as AND); your own roleplay characters stay pinned to the top, and tag chips in the list are clickable.
- **Highlighter** — highlight lines that mention your character names.
- **Tags & notes** — tag logs and attach notes; stored as JSON sidecars, never inside the log files.
- **Live updates** — the log list refreshes automatically while Gobchat writes new logs.
- **Fast startup** — a persistent metadata index means even large log collections open quickly.
- **Opt-in update check** — get notified about new releases.
- **First-run wizard, dark & light themes with customizable highlight colors, English & German UI.**

## Getting started

> **Platform support:** Windows is the supported and tested platform. The code base is platform-agnostic and Linux/macOS builds should work, but they are currently untested.

Download the latest version from the [releases page](https://github.com/Shuro/Gobchat-Log-Browser/releases/latest):

- **Installer (recommended):** `Gobchat-Log-Browser-win-Setup.exe` — installs per-user to `%LOCALAPPDATA%\Gobchat-Log-Browser` (no admin rights needed), creates Start Menu and desktop shortcuts, and installs the Microsoft Edge WebView2 runtime if it is missing. Once installed, the app updates itself in place (opt-in, from the Settings → About section). Uninstalling via Windows Settings → Apps keeps your tags, notes, and settings.
- **Portable:** `Gobchat-Log-Browser-win-Portable.zip` — unzip anywhere and run the exe directly (no auto-update).

> **SmartScreen warning:** the binaries are not code-signed, so Windows may show *"Windows protected your PC"* on first run. Click **More info → Run anyway**. This is expected for small open-source tools without a (paid) signing certificate.

Building from source (see below) works too, but is optional.

1. Run the application. On first launch a short setup wizard asks for your language, theme, and log folder.
2. If Gobchat is installed, its log folder is detected automatically — just confirm it.
3. Pick a log from the list and start reading. That's it.

**Requirements:** Windows 10/11 with the [WebView2 runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (preinstalled on Windows 11 and most up-to-date Windows 10 systems).

## Building from source

Prerequisites:

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) (with npm)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

```bash
# Development with hot reload
wails dev

# Production build → build/bin/gobchat-log-browser.exe
wails build
```

Running the tests:

```bash
go test ./...                      # backend
cd frontend && npm run build       # type-check + build the frontend
```

## Architecture

The backend (Go) handles everything heavy: parsing the Gobchat log format, roleplay-span tokenization, search indexing, file watching, and metadata caching. The frontend (Vue 3 + TypeScript + Pinia) is a thin, virtualized UI on top, connected through Wails bindings.

Design decisions are recorded as ADRs in [docs/adr/](docs/adr/) — including why logs are read-only and why message reassembly is heuristic and display-only.

## License

Licensed under the [Apache License 2.0](LICENSE).

*Gobchat Log Browser is a fan-made tool and is not affiliated with Square Enix or with the Gobchat project. Final Fantasy XIV is a registered trademark of Square Enix Holdings Co., Ltd.*
