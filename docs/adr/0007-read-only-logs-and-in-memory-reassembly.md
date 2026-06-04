# ADR-0007: Logs are strictly read-only; reassembly is in-memory and optional

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

Gobchat log files are the user's irreplaceable record of roleplay sessions. A multi-part post from one sender is often interrupted by another player's message (e.g. `A(1/3)`, B speaks, `A(2/3)`; or `A …>`, B speaks, `A > …`; or even `A …>`, B speaks, `A …` with no resume marker). Users want an optional view that stitches such interrupted posts back into a single readable thread.

The risk: a "reorder" or "merge" feature could be misread as something that rewrites the files.

## Decision

The application will open log files **read-only** and MUST NEVER write, rename, move, or otherwise modify them. Any reassembly/reordering/merging is a **pure in-memory display transformation** in `internal/reassemble`, computed over already-parsed entries, referencing original line numbers and copying — never mutating. The viewer exposes a toggle: **Raw** (every line as-is, file order — the default, faithful representation) and **Reassembled** (threads merged in memory). Reassembly uses the heuristics from ADR-0006 (same sender + channel, part/`>` markers, bounded interruption window) and falls back to single-line threads when unsure.

## Consequences

- **Positive:** The source of truth on disk is guaranteed safe; users get readability without risk; the transformation is testable and deterministic.
- **Negative / risks:** Reassembly can mis-group (acceptable because Raw is the default and always available).
- **Follow-up:** Never add a "save reassembled" or "edit log" path without a superseding ADR and explicit user intent.
