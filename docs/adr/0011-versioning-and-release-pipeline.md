# ADR-0011: Versioning, release pipeline, and update-check groundwork

- **Status:** Accepted
- **Date:** 2026-06-11

## Context

The app has had no version: no git tags, no version constant in the code, and the Windows
file metadata fell back to the Wails default of 1.0.0. Users need installable releases —
without admin rights, since the target audience runs the app on personal gaming machines —
and a future in-app update check needs a machine-readable version plus predictably named
release assets to compare against and download. User data (config.json, tags.json,
index.json) lives in `%APPDATA%\GobchatLogBrowser` with the disposable WebView2 cache as a
`webview2` subfolder (ADR-0005/0009/0010); an uninstaller must never take the user data
with it. The stock Wails NSIS template installs to Program Files with admin elevation and
its uninstall section deletes a folder directly next to our user data.

## Decision

We will version the app with semver git tags (`vX.Y.Z`, starting at v0.1.0) as the single
source of truth. Tags are plain numeric three-part versions because NSIS
`VIProductVersion` accepts nothing else.

The version reaches the Go binary via
`-ldflags "-X gobchat-log-browser/internal/version.Version=x.y.z"`; local builds report
`"dev"`. Windows file metadata gets the version by patching `wails.json`'s
`info.productVersion` in CI from the tag — the committed value stays `0.0.0` as a dev
marker, so there are no version-bump commits and no second source to drift.

A GitHub Actions workflow (`.github/workflows/release.yml`) triggers on tag push, builds
on `windows-latest` with `wails build -nsis`, and publishes a GitHub Release containing a
zipped portable exe and an NSIS installer. Asset names follow the fixed convention
`gobchat-log-browser-v<ver>-windows-amd64.zip` /
`gobchat-log-browser-v<ver>-windows-amd64-installer.exe`, which the future update check
will resolve without extra metadata.

The installer is per-user: `RequestExecutionLevel user`, install dir
`$LOCALAPPDATA\GobchatLogBrowser`, HKCU registry and per-user shortcuts — no UAC prompt.
Uninstall removes the install dir, shortcuts, and only the
`%APPDATA%\GobchatLogBrowser\webview2` cache; config, tags, and the metadata index survive
uninstall and are picked up again on reinstall.

The frontend shows the version (via a `GetVersion` binding) in the Settings footer.

**Planned update check (not implemented yet):** on startup or manual trigger, GET
`https://api.github.com/repos/Shuro/Gobchat-Log-Browser/releases/latest` (unauthenticated;
the rate limit is irrelevant at one request per launch), semver-compare `tag_name` against
`version.Version` — skipped entirely when the version is `"dev"` — and show a notice in
the UI. Future auto-update paths: installed copies download and run the `-installer.exe`
asset (per-user, later silently via `/S`); portable copies either open the release page or
self-swap the exe (e.g. `minio/selfupdate`) from the zip asset.

## Consequences

- **Positive:** Reproducible releases from a tag push with nothing else to edit; no UAC
  anywhere (install, run, update); the binary knows its own version; asset naming and
  version comparison are ready for the update checker.
- **Negative / risks:** Binaries are unsigned, so SmartScreen will warn on first run;
  per-user installs are invisible to other Windows accounts on the same machine; local
  builds carry `dev`/`0.0.0` metadata, which must not confuse the future update check
  (hence the explicit dev skip).
- **Follow-up:** Implement the update check; consider code signing if SmartScreen warnings
  become a support burden.
