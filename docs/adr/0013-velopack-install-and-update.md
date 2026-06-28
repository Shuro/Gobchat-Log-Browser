# ADR-0013: Velopack installer and in-app auto-update

- **Status:** Accepted
- **Date:** 2026-06-28

## Context

ADR-0011 shipped a per-user **NSIS** installer plus a portable zip, and ADR-0012 added an
**opt-in, notify-only** update check: `internal/update` polled the GitHub
`releases/latest` API, compared semver, and showed a banner whose only action was to open
the GitHub release page in a browser. There was no in-app download or apply — every update
was a manual re-download and re-install by the user.

The goal is real auto-updates: the app should fetch a new version and relaunch into it from
inside the UI, with full + delta packages so updates stay small. NSIS has no update story
of its own, and hand-rolling self-update (download installer, run `/S`, swap exe) on top of
it would reimplement what mature updater frameworks already do.

The target audience runs the app on personal gaming machines without admin rights, so any
solution must stay per-user and UAC-free (the ADR-0011 constraint still holds). User data
(`config.json`, `tags.json`, `index.json`) lives in `%APPDATA%\GobchatLogBrowser`
(ADR-0005/0009/0010), separate from the install location, and must survive any
install/uninstall/migration.

## Decision

We will replace NSIS **and** the notify-only checker with **[Velopack](https://docs.velopack.io/)**
(Rust-core installer/updater with full + delta packages), driven from Go via the community
CGO binding **[quaadgras/velopack-go](https://github.com/quaadgras/velopack-go)** (pinned
`v0.0.1358`, MIT, links Velopack's official `velopack_libc` static libs).

**Packaging.** `vpk pack` (run in CI) produces `Setup.exe`, `*-full.nupkg`,
`*-delta.nupkg`, a `*-Portable.zip`, and `releases.win.json`. We pack with
`--packId Gobchat-Log-Browser`, installing per-user to
`%LocalAppData%\Gobchat-Log-Browser\current\` with `Update.exe` alongside — no UAC, the
same no-elevation model as before. The packId is deliberately **not** `GobchatLogBrowser`:
that is the exact folder the legacy NSIS build installed into (ADR-0011), and Velopack
derives its install folder from the packId, so reusing it would collide — the migration's
silent NSIS uninstall (`rmdir /S` on the legacy folder) would then delete the new install.
The hyphenated `Gobchat-Log-Browser` keeps the two side by side. CI computes deltas by first
running `vpk download github` to fetch the previous release's packages.

**WebView2.** `vpk pack --framework webview2` makes `Setup.exe` install the Evergreen
WebView2 runtime if it is missing, replacing the `MicrosoftEdgeWebview2Setup.exe`
bootstrapper that the old NSIS template bundled. No bootstrapper is shipped in the package;
the runtime is fetched only when absent.

**Update feed.** The GitHub Release is the feed: the updater points at
`https://github.com/Shuro/Gobchat-Log-Browser/releases/latest/download`, where Velopack
reads `releases.win.json` and the nupkgs uploaded by CI.

**Runtime wiring.** `velopack.Run(...)` is the **first statement in `main()`** so Velopack
services install/update hooks before the GUI starts. `internal/velopackupd` wraps the
`UpdateManager`: `Check()` and `DownloadAndApply(progress)`. The check is gated on the
existing `check_updates_on_start` opt-in (ADR-0012) and short-circuits to a "dev" status
when `internal/version.Version == "dev"` (uninstalled local builds report not-installed).

**Bindings/UX.** `CheckForUpdate()` returns the velopack-derived status;
`DownloadAndApplyUpdate()` downloads (emitting `update:progress` 0–100), applies, and calls
`wruntime.Quit` so Velopack relaunches the new version. The banner and Settings present a
single **"Update & restart"** action with a progress bar — there is no separate restart
step and no more "open the release page" flow.

**NSIS migration.** Velopack only auto-migrates from Squirrel, not NSIS, so an existing
NSIS install would otherwise sit side-by-side in the old folder. `internal/migrate`
(`nsis_windows.go`, no-op stub elsewhere) does a **detect + silent uninstall** on every
startup of a **release** build: it reads HKCU
`…\Uninstall\ShuroGobchat Log Browser`, verifies the recorded `InstallLocation` is not the
current Velopack dir, and runs the legacy uninstaller. The uninstaller is launched
**directly** (not through `cmd /C`, whose quote handling mangled the already-quoted
`QuietUninstallString`) and **synchronously** via NSIS's `_?=<dir>` in-place flag, so we wait
for it to finish instead of firing and forgetting. User data in `%APPDATA%` is untouched.

The presence of the legacy HKCU key is the **only** state the migration acts on: it
no-ops once the key is gone and self-heals if a legacy copy is later reinstalled. An
earlier design (shipped in v0.2.0) latched completion in an `nsis-migrated` sentinel file
under the version-shared `%APPDATA%\GobchatLogBrowser`. That marker could be written
**prematurely** — by a `wails dev` run, or by any launch that preceded the legacy install —
and then permanently disabled the migration, leaving the old install behind. Two fixes
(v0.2.1): the marker was removed entirely, and the migration is now **gated to release
builds** (`version.Version != "dev"`) so dev runs, which execute from a temp/build dir that
`isCurrentInstall` cannot recognise, never silently uninstall the developer's real copy.

**Build requirement.** Velopack's static Rust libs reference ntdll's `Nt*` syscalls and the
binding omits the link flag, so every Go/Wails build on Windows must set
`CGO_LDFLAGS=-lntdll` (env var only — a per-package `#cgo` directive lands in the wrong link
order). This is set in `wails dev`/`wails build` locally and in `release.yml`.

This **supersedes the NSIS packaging of ADR-0011 and the notify-only mechanism of
ADR-0012.** The parts of those ADRs that still hold are retained: semver tags as the single
source of truth, the `-ldflags` version injection with `dev` for local builds, the
`check_updates_on_start` opt-in, and setup-wizard versioning. The ADR-0012 **installer
seed** is removed — Velopack has no installer finish page, so the update opt-in is asked
only in the wizard.

## Consequences

- **Positive:** Real in-app updates with small delta downloads; one-click "Update &
  restart"; WebView2 handled by the installer again; per-user/no-UAC preserved; old NSIS
  installs are cleaned up automatically; user data carries over untouched.
- **Negative / risks:** velopack-go is an immature single-maintainer binding (pinned;
  `binaries/` vendored; exit path = call `velopack_libc` C directly or revert to NSIS).
  The `-lntdll` requirement is a non-obvious build footgun (recorded here and in project
  docs). Setup.exe is still **unsigned**, so SmartScreen will warn on first run — no
  regression from ADR-0011; signing remains separate future work. The packId/folder change
  (legacy `GobchatLogBrowser` → `Gobchat-Log-Browser`) is what the migration step handles.
- **Follow-up:** Code signing if SmartScreen becomes a support burden; revisit the binding's
  health before depending on newer Velopack features.
