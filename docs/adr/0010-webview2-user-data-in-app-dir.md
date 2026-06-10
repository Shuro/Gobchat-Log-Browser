# ADR-0010: WebView2 user data lives in the app data dir

- **Status:** Accepted
- **Date:** 2026-06-10

## Context

On Windows the app left two folders in `%APPDATA%`: the intentional `GobchatLogBrowser` (config.json, tags.json, index.json — ADR-0005/0009) and an unexpected `Gobchat-log-browser.exe`. The latter is the WebView2 user data folder: Wails v2 defaults it to `%APPDATA%\<binaryname>.exe` when `windows.Options.WebviewUserDataPath` is unset, and `main.go` passed no Windows options. Two app folders with inconsistent naming is confusing and pollutes `%APPDATA%`.

The frontend stores nothing in browser storage (no localStorage/sessionStorage/indexedDB), so the WebView2 folder is disposable cache and can be relocated without data loss.

## Decision

Set `WebviewUserDataPath` to `%APPDATA%\GobchatLogBrowser\webview2`, derived from the existing `config.AppDataDir()`. If `AppDataDir()` fails, the path is left empty and Wails falls back to its default — startup is never blocked by this setting.

No in-app cleanup of the stale `Gobchat-log-browser.exe` folder: the app stays free of delete-other-directories code; existing users remove the orphan folder manually once.

## Consequences

- **Positive:** All app-created state sits under one consistently named folder; fresh installs never create the `.exe`-suffixed folder.
- **Negative / risks:** Existing installs keep an orphaned `Gobchat-log-browser.exe` folder until manually deleted; on first launch after the change WebView2 rebuilds its cache (one-time, cosmetic).
- **Follow-up:** None.
