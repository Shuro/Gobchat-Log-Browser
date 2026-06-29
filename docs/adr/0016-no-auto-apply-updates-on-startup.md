# ADR-0016: Updates are never auto-applied on startup

- **Status:** Accepted
- **Date:** 2026-06-29

## Context

ADR-0013 moved updates to Velopack and established the model: the update **check**
is gated on the existing `check_updates_on_start` opt-in, and applying an update is
a deliberate, user-driven action — the "Update & restart" button calls
`internal/velopackupd.DownloadAndApply`, which stages the package and asks Velopack
to `WaitForExitThenApplyUpdates(..., Restart{})` after the app quits.

The runtime wiring, however, called `velopack.Run(velopack.App{AutoApplyOnStartup:
true})` as the first statement in `main()`. `AutoApplyOnStartup` is a velopack-go
flag that makes the Velopack runtime contact the release feed and apply a newer
release **on every launch, before the GUI**, with no consent check. While `v0.2.2`
was the latest release this lay dormant. Publishing `v0.3.0` activated it in the
field: every `v0.2.2` launch ran the startup auto-apply path
(`releases.win.json` → "0.2.2 → 0.3.0" → delta), the window flashed, and the process
exited to hand off to the updater — the app was effectively unusable, and it phoned
the feed regardless of the user's `check_updates_on_start` choice.

This is a direct contradiction of ADR-0013's consent model that was never noticed
because no newer release existed to trigger it.

## Decision

**`velopack.Run` is called with `AutoApplyOnStartup: false`.** `velopack.Run`
(`vpkc_app_run`) still services the install/update/uninstall lifecycle hooks before
the GUI — that behavior is independent of the flag and is the only reason the call
must come first in `main()`. Dropping the auto-apply restores the single intended
update path: a check gated on `check_updates_on_start`, and an apply that only ever
happens when the user clicks "Update & restart". The in-app flow does not need
`AutoApplyOnStartup` — it applies explicitly via `WaitForExitThenApplyUpdates`.

This supersedes the runtime-wiring portion of ADR-0013; the rest of ADR-0013 (feed,
`UpdateManager` wrapper, one-click apply, NSIS migration) is unchanged.

## Consequences

- **Positive:** The app no longer contacts the release feed or restarts itself on
  launch; updates are once again strictly opt-in and user-initiated, matching the
  documented consent model. The flash-and-exit on launch is gone for builds that
  carry this fix.
- **Negative / risks:** The faulty `AutoApplyOnStartup: true` is compiled into every
  already-shipped binary (`v0.2.0`–`v0.3.0`); those installs cannot be fixed
  remotely. Each will keep attempting the startup auto-apply toward the latest
  release. Whether they self-heal onto a fixed build or need a manual reinstall
  depends on whether that auto-apply completes, which is exactly the unreliable
  behavior being removed — so a manual reinstall of a fixed build is the dependable
  recovery for an affected install.
- **Follow-up:** Ship the fix in the next tagged release. Treat any future use of
  `AutoApplyOnStartup` as a deliberate, separately-recorded decision, not a default.
