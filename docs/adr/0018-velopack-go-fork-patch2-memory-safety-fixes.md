# ADR-0018: Retag velopack-go fork to patch2 for further native-memory bugs

- **Status:** Accepted
- **Date:** 2026-07-01

## Context

ADR-0017 forked `quaadgras/velopack-go` v0.0.1358 to `github.com/Shuro/velopack-go`
(tag `v0.0.1358-patch1`) to stop `(*UpdateInfo).load()` from walking the
native `DeltasToTarget` array as if it were NULL-terminated, which it isn't. Patch1's
fix was to leave `DeltasToTarget` unpopulated, since our code never read it.

Further review of the same binding surfaced additional native-memory bugs, unrelated
to the original crash but in the same risk class (cgo boundary code with no Go-side
safety net):

1. **Double-free-prone finalizers.** `toAsset()` registered a `vpkc_free_asset`
   cleanup on every asset it converted, including `TargetFullRelease`, `BaseRelease`,
   and `DeltasToTarget` entries. Those are sub-objects owned by their parent
   `vpkc_update_info_t` and already freed together via `vpkc_free_update_info` — so
   they could be freed twice: once by their own finalizer, once as part of the parent.
2. **Use-after-free in restart args.** `assetSilentRestart()` did
   `defer C.free(arg_cstr)` on the C strings it allocated for restart arguments. That
   defer fires when `assetSilentRestart` itself returns — before its caller,
   `WaitForExitThenApplyUpdates`, ever passes those pointers to the native
   `vpkc_unsafe_apply_updates` / `vpkc_wait_exit_then_apply_updates` calls. Any call
   with `Restart` args was passing already-freed pointers to native code.
3. **`size_t` underflow.** The restart-args count was computed as `restart-1`
   unconditionally, including when `restart == 0` (no restart args). `restart` is a Go
   int; `0-1` cast to `C.size_t` wraps to the maximum `size_t` value, handed straight
   to the native call as an array length.
4. **Off-by-one in `AppID()` / `CurrentlyInstalledVersion()`.** The native length
   already includes the null terminator; the binding over-allocated by one more byte
   and returned the string including the embedded terminator instead of stripping it.

Given the original patch1 fix ("leave the array nil") no longer applies once
`DeltasToTarget` is being properly read via its count field, and these are new,
distinct bugs, this warrants a new fork tag and a new ADR rather than silently
editing ADR-0017's historical record.

## Decision

We retag the fork `v0.0.1358-patch2`, which:

- Bounds the `DeltasToTarget` walk with the native `DeltasToTargetCount` field via
  `unsafe.Slice` instead of leaving it nil — the array is now both safe and populated.
- Adds an `owned bool` parameter to `toAsset()` so only genuinely standalone assets
  (currently just `UpdatePendingRestart`'s return value) get a free-on-GC finalizer;
  assets embedded in a parent struct no longer get a competing one.
- Moves the restart-arg `C.free` calls out of `assetSilentRestart` and into a
  closure that `WaitForExitThenApplyUpdates` defers itself, after the native calls
  that use the pointers.
- Guards the restart-args count computation so it's only computed when `restart > 0`.
- Fixes the `AppID`/`CurrentlyInstalledVersion` buffer sizing and terminator handling.

`go.mod`'s `replace` directive now points at `v0.0.1358-patch2`. ADR-0017 is marked
superseded by this ADR for the `DeltasToTarget` handling specifically; its account of
the original crash and root cause remains accurate and is left unedited.

## Consequences

- **Positive:** Removes a double-free/use-after-free class of native crash risk that
  existed independently of the original delta-update crash, and fixes visibly wrong
  string output from `AppID`/`CurrentlyInstalledVersion`.
- **Negative / risks:** Same as ADR-0017 — this is still a single-maintainer fork with
  no automated regression coverage for these native-memory bugs (reproducing a
  use-after-free or double-free deterministically in a Go test would require faking
  the C allocator/finalizer timing through cgo). Verification here relied on reading
  the fix rather than reproducing each bug beforehand.
- **Follow-up:** File the same upstream issue as ADR-0017 covers this ground too;
  retire the fork once fixed upstream. Before the next release, exercise a real
  update with `Restart` args (not just a bare delta update) to validate fix #2/#3
  end-to-end, since that path previously used freed memory.
