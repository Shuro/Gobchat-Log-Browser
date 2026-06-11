# ADR-0012: Opt-in update check, setup wizard versioning, installer seed

- **Status:** Accepted
- **Date:** 2026-06-11

## Context

ADR-0011 prepared everything an update check needs — semver tags, the version baked into
the binary, predictably named release assets — but the check itself was left as follow-up.
The target audience runs the app on personal gaming machines; an app that silently phones
home on every launch is not acceptable there, so the check must be consensual. Existing
users completed the first-run wizard before an update-check question existed, and the
Windows installer is a natural place to ask, but the installer knows nothing about the
app's config schema or its atomic-write discipline, and Go code stays platform-agnostic.

## Decision

**Opt-in, default off.** A new `check_updates_on_start` config field (default `false`)
gates the startup check; no network call ever happens without consent. The check GETs
`https://api.github.com/repos/Shuro/Gobchat-Log-Browser/releases/latest` (unauthenticated;
the 60/h/IP rate limit is irrelevant at one request per launch) and compares `tag_name`
against `internal/version.Version` with a hand-rolled three-part comparison in
`internal/update` — ADR-0011 guarantees plain numeric tags, so no semver dependency is
needed. Builds whose version does not parse (`dev`) skip the network call entirely.

**One binding, two callers.** A single `CheckForUpdate()` Wails binding serves both the
frontend-initiated startup check (which swallows rejections — being offline must be
silent) and the manual "Check for updates" button in the new Settings About section (which
surfaces them). No backend goroutine or event: the check is request/response shaped,
unlike the filesystem-fact events (`logs:scanned`, `log:*`).

**Notice only.** An update shows a dismissible banner / an inline Settings notice with a
button that opens the GitHub release page in the browser. Downloading or running the
installer from inside the app remains the ADR-0011 follow-up.

**Wizard versioning.** Config gains `setup_wizard_version`; Go holds
`SetupWizardCurrentVersion` (now 2) and `GetSetupState` re-shows the wizard when the saved
value is behind. Configs written before this feature lack the field and load as 0, which
is exactly the desired "show once more" semantics. The re-shown wizard is pre-filled from
the existing config and stamps the version delivered by the backend on save.

**Installer seed, read-once.** The NSIS finish page gains a second checkbox (the
repurposed MUI2 readme checkbox) that writes a one-shot
`%APPDATA%\GobchatLogBrowser\installer-defaults.json` containing
`{"check_updates_on_start": true}`. The app consumes it in `GetSetupState` — read, then
delete, even when malformed — and uses it only to pre-check the wizard toggle. The
installer never touches `config.json`, so it stays ignorant of the schema and cannot
corrupt an atomic-write file; the uninstaller deletes a leftover seed (installer artifact,
not user data). Because MUI2 processes the Run checkbox before the readme one, the run
function writes the seed itself first when the box is ticked, so a launched app cannot
race past it.

## Consequences

- **Positive:** No network traffic without explicit consent; existing users get asked the
  new question exactly once; the installer choice survives into the wizard without the
  installer knowing the config format; offline launches stay completely silent.
- **Negative / risks:** If the app is killed between seed consumption and wizard save, the
  installer choice is lost (cosmetic — the toggle just defaults to off). Reinstalling over
  an up-to-date config consumes the seed without effect, since the wizard does not
  re-show. The unauthenticated API call can be rate-limited or blocked; the manual check
  surfaces that as an error, the startup check stays silent.
- **Follow-up:** In-app download/run of the installer asset (ADR-0011); bump
  `SetupWizardCurrentVersion` whenever the wizard gains content existing users should see.
