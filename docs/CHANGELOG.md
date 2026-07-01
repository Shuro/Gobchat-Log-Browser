# Changelog

All notable changes to Gobchat Log Browser are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.4] - 2026-07-01

### Fixed

- Delta updates (the common case once you already have a prior version
  installed) could crash the app outright while checking for or downloading
  an update, due to a bug in the underlying Velopack update library reading
  past the end of an internal list. Full updates were unaffected. Fixed by
  patching the library (docs/adr/0017).
- Further hardening of the same library: a potential double-free in asset
  cleanup, a use-after-free when applying an update with a restart, and a bug
  that could leave a stray character on the end of the reported installed
  version (docs/adr/0018).
- **Note for existing installs (v0.3.0–v0.3.3):** the crash above happens in
  your *currently installed* app while it checks for updates, before it ever
  receives this fix — so the in-app updater can't reliably deliver v0.3.4 to
  an install that hits it. If "check for updates" crashes the app, download
  v0.3.4 manually from the releases page and reinstall over the existing
  install (your tags, notes, and settings are preserved).

## [0.3.3] - 2026-07-01

### Fixed

- Search results could occasionally show stale results: if an older, slower
  search request finished after a newer one, its results could overwrite the
  results of the query you actually typed last. Superseded requests are now
  discarded.
- The one-shot legacy NSIS uninstall (used to clean up leftover pre-Velopack
  installs) trusted the `InstallLocation` from an ordinary, unprotected
  registry value for a destructive, unattended operation. It now refuses to
  run unless that value matches the one directory the legacy installer could
  actually have used.

## [0.3.2] - 2026-06-29

### Added

- New setting "Hide logs with no detected players" (Settings → General). When
  enabled, the overview hides log files that contain only system/info or
  unparseable lines and no actual participants. Off by default, so existing
  logs stay visible until you opt in.

## [0.3.1] - 2026-06-29

### Fixed

- The app could flash on screen and immediately exit on launch once a newer
  release existed: Velopack was configured to auto-apply updates on startup
  (`AutoApplyOnStartup`), which also contacted the release feed on every launch
  regardless of the "check for updates on start" opt-in. Updates are now strictly
  user-initiated via "Update & restart" again (ADR-0016). Note: installs of
  v0.2.0–v0.3.0 carry the old behavior and may need a manual reinstall of a fixed
  build to recover.

## [0.3.0] - 2026-06-29

### Added

- Auto-detection of GobchatEx logs (`%APPDATA%\GobchatEx\log`) alongside Gobchat,
  under the existing "auto-detect app data" option (ADR-0015).
- Settings now lists the auto-detected log folders (Gobchat / GobchatEx) so you
  can see exactly which locations auto-detect covers.
- Config schema versioning (`config_version`) with a migration runner, so future
  updates have a real upgrade path beyond zero-value backfill (ADR-0014).

### Changed

- When the same chat log exists in both the Gobchat and GobchatEx folders, the
  overview and search now show a single entry: the GobchatEx copy when the two are
  identical, otherwise the newer file (ADR-0015).
- Renamed the "Dark" theme to "Blue" to match its actual color and distinguish it
  from "GobchatEx Dark"; existing configs are migrated automatically (theme value
  and saved color overrides) via the new config schema migration.

## [0.2.2] - 2026-06-28

### Fixed

- Removing the legacy NSIS install during first-run migration also deleted the
  new app's Start Menu shortcut, because both used the name "Gobchat Log
  Browser". The migration now preserves the current shortcut across the legacy
  uninstall, and fully removes the leftover legacy install folder.

## [0.2.1] - 2026-06-28

### Fixed

- The first-run cleanup of a leftover legacy NSIS install (introduced in 0.2.0)
  could silently never run, leaving the old "Gobchat Log Browser 0.1.x" entry in
  Windows "Installed apps" beside the new one. The cleanup latched completion in
  a sentinel file under shared app data that a development run could write
  prematurely; it now keys solely off the legacy install's presence (so it also
  self-heals if the old version is reinstalled) and runs only in release builds.

## [0.2.0] - 2026-06-28

### Changed

- Replaced the per-user NSIS installer and the notify-only update check with
  [Velopack](https://docs.velopack.io/): the app now downloads and applies
  updates in place and relaunches into the new version from the UI, with full +
  delta packages so updates stay small (ADR-0013). The Settings → About update
  control and the banner now offer a single "Update & restart" action with a
  progress bar instead of opening the GitHub release page.
- The installer now installs per-user to `%LOCALAPPDATA%\Gobchat-Log-Browser`
  (previously `%LOCALAPPDATA%\GobchatLogBrowser`) and installs the Microsoft Edge
  WebView2 runtime if it is missing.
- Release assets are now `Gobchat-Log-Browser-win-Setup.exe` and
  `Gobchat-Log-Browser-win-Portable.zip` (plus full/delta `.nupkg` and
  `releases.win.json` for the update feed).

### Added

- "GobchatEx Dark" theme — a third theme option alongside Dark and Light, tuned
  to GobchatEx's FFXIV-modern palette (selectable in the setup wizard and
  Settings).
- One-shot first-run migration that silently uninstalls a leftover legacy NSIS
  install on Windows. User data in `%APPDATA%\GobchatLogBrowser` (tags, notes,
  settings, metadata cache) is preserved untouched.

### Removed

- The notify-only GitHub-API update checker (`internal/update`) and the NSIS
  installer finish-page update opt-in seed; the update opt-in is now asked only
  in the first-run wizard.

## [0.1.4] - 2026-06-12

### Fixed

- Register the per-user install in Windows "Installed apps" so it can be
  uninstalled from Settings.

### Changed

- Trimmed the README feature list (dropped the realm toggle, shortened the
  update-check note).

## [0.1.3] - 2026-06-12

### Added

- Viewer channel filter.

### Changed

- Search result polish and tag autosave.
- Release notes are now generated from program-affecting commit subjects.

### Fixed

- Live-update reliability fixes.

## [0.1.2] - 2026-06-12

### Added

- Settings tabs with custom highlight colors.
- Player and tag filters for the log list.
- More roleplay continuation markers.

### Fixed

- Log display fixes.

## [0.1.1] - 2026-06-11

### Added

- Opt-in update check with an About section and first-run wizard versioning
  (ADR-0012).
- AI-generated mock log folder for screenshots and examples.

## [0.1.0] - 2026-06-11

Initial public release.

### Added

- Log overview listing all logs with date, participants, message count, and
  duration.
- Gobchat CCLv1/FCLv1 log parsing with sender split (status symbol / name /
  realm) and heuristic multi-part/continuation detection; unparseable lines
  surface as raw text rather than being dropped.
- Configurable roleplay highlighting for dialogue, emotes, and out-of-character
  text.
- Raw and reassembled views — in-memory stitching of interrupted multi-part
  messages with per-post start/end times; log files are never modified.
- Full-text search across all logs plus find-in-log with match navigation and
  scrollbar match ticks.
- Player and `#tag` filtering with your own roleplay characters pinned to the top.
- Tags and notes stored as JSON sidecars, never inside the log files.
- Live updates while Gobchat writes new logs (fsnotify watcher).
- Persistent metadata index for fast startup on large log collections.
- First-run setup wizard, dark/light themes, and English/German UI.
- Per-user NSIS installer and portable zip; tag-triggered release pipeline
  (ADR-0011).

[0.3.3]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/Shuro/Gobchat-Log-Browser/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Shuro/Gobchat-Log-Browser/releases/tag/v0.1.0
