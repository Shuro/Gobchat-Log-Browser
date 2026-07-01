# ADR-0017: Fork velopack-go to fix a delta-update crash

- **Status:** Superseded by ADR-0018 (DeltasToTarget handling only; root cause below still accurate)
- **Date:** 2026-07-01

## Context

Diagnosing the v0.3.2 → v0.3.3 auto-update (a delta update — the first release with
a prior version to diff against) with a locally instrumented build surfaced a crash risk
inside `github.com/quaadgras/velopack-go` v0.0.1358 itself, not in our code.

`(*UpdateInfo).load()` converts the native `vpkc_update_info_t` into Go and walks
`DeltasToTarget` — a `**C.vpkc_asset_t` — as if it were NULL-terminated:

```go
for ptr := update_info.DeltasToTarget; *ptr != nil; ptr = ptr+unsafe.Sizeof(*ptr) {
    deltas = append(deltas, toAsset(*ptr))
}
```

The native Velopack library does not actually NULL-terminate that array. The walk reads
past the real entries into unrelated memory and keeps going until it dereferences garbage
as a `vpkc_asset_t*`, which is a native memory fault — not a Go panic, so `recover()` cannot
catch it and the whole process goes down. This triggers whenever `CheckForUpdates` or
`DownloadUpdates` returns an `UpdateInfo` with a delta package present, which is most
updates after the first release (full-only updates have an empty/absent delta list and
never exercise the walk).

Our own code never reads `UpdateInfo.DeltasToTarget` — `internal/velopackupd/updater.go`
only touches `TargetFullRelease` — and the delta-vs-full download decision happens
natively inside `DownloadUpdates`/`vpkc_download_updates` against the raw C handle, not
through this Go-side slice. So the array is parsed only to populate a field nothing reads.

velopack-go is a single-maintainer community binding; ADR-0013 already flagged this as a
risk and named "call velopack_libc directly or revert to NSIS" as the exit path if it
became unhealthy. A one-function fork is a lighter-touch fix than either.

## Decision

We fork `quaadgras/velopack-go` to `github.com/Shuro/velopack-go`, patch `load()` to skip
the `DeltasToTarget` walk entirely (leaving the slice `nil`), tag the fork
`v0.0.1358-patch1`, and point `go.mod` at it via a `replace` directive:

```
replace github.com/quaadgras/velopack-go => github.com/Shuro/velopack-go v0.0.1358-patch1
```

The fork stays otherwise identical to upstream `v0.0.1358` — same module path declared
internally, same vendored `binaries/`, same everything except this one function — so the
`replace` is a drop-in swap with no other code changes required. The fix was validated by
re-running the real v0.3.2 → v0.3.3 delta update against the instrumented build: the
captured `update-diagnostic.log` shows a clean, successful delta apply with no crash,
where the unpatched binding was expected to fault.

An issue against upstream `quaadgras/velopack-go` is a recommended follow-up so the bug can
be fixed there and this fork retired.

## Consequences

- **Positive:** Delta updates (the common case after the first release) no longer risk
  crashing the update process. The fix is behaviorally inert otherwise — `DeltasToTarget`
  was never consumed on the Go side, so nothing else changes.
- **Negative / risks:** Future `velopack-go` version bumps require manually re-forking or
  re-applying this patch; there is no automated regression test for the native
  out-of-bounds read itself (reproducing it would require faking a non-NULL-terminated C
  array through cgo), so this relies on the manual delta-update verification described
  above plus this ADR to stop a future maintainer from "restoring" the walk.
- **Follow-up:** File the bug upstream; once/if fixed there, drop the `replace` directive
  and return to the stock module.
