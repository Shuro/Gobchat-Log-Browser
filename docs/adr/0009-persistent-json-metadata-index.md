# ADR-0009: Persistent JSON metadata index (index.json)

- **Status:** Accepted
- **Date:** 2026-06-10

## Context

Every launch fully parses every log file just to rebuild the overview metadata (`LogMeta`: participants, channels, timestamps, message count) — the known O(files) cost noted in ADR-0008. A realistic corpus (~410 files, ~42 MB) makes startup noticeably slow, and the participant lists are now the basis for the player filter in the log list. The metadata itself is tiny (~1 KB per file), and per-file player extraction already happens during parsing.

A database (SQLite) was considered for the index. It would reintroduce the dependency/CGO question rejected in ADR-0001/0004 — for a few hundred rows of small, derived data.

## Decision

Persist `LogMeta` in a **versioned JSON sidecar** `index.json` in the app data dir (next to `config.json` and `tags.json`), implemented as `logstore.MetaCache`. Entries are keyed by absolute file path and validated by **mtime + size**: scans reuse cached metadata for unchanged files and parse only new/changed ones; deleted files are pruned after each full scan; watcher refreshes update the entry. Writes are atomic (`.tmp` + rename), matching the ADR-0005 pattern.

Unlike `tags.json` (user-authored, preserved as `.corrupt` on parse failure), the index is **derived data**: a missing, corrupt, or version-mismatched file silently yields an empty cache and the next scan rebuilds it. A failed save never fails a scan.

The UI folder badge (per-profile subfolder shown per log row) was dropped at the same time — the player filter supersedes it as the way to find whose logs are whose. `LogMeta.Folder` itself remains (ADR-0008 still stands).

## Consequences

- **Positive:** Near-instant second launch (one stat per file instead of a full parse); zero new dependencies; player names per file survive restarts without re-parsing; tolerant load means no new failure modes at startup.
- **Negative / risks:** mtime+size is a heuristic — a same-size in-place edit within mtime granularity would be served stale (not a realistic pattern for append-only chat logs); the whole index is rewritten on each save (fine at ~hundreds of KB).
- **Follow-up:** The search index (ADR-0004) is still rebuilt lazily per launch and is unchanged in scope. If corpora grow to where JSON rewriting hurts, a SQLite-backed index would warrant a superseding ADR.
