# ADR-0015: GobchatEx log directory support and duplicate-log dedup

- **Status:** Accepted
- **Date:** 2026-06-28

## Context

GobchatEx is a fork of Gobchat that writes its chat logs to its own app-data
folder — `%APPDATA%\GobchatEx\log` on Windows (and the analogous
`~/.local/share/GobchatEx/log` elsewhere) — keeping Gobchat's `log` subfolder name
and the identical `chatlog_YYYY-MM-DD_HH-mm.log` filename scheme. Users moving from
Gobchat to GobchatEx end up with both folders present, and the same session is
frequently captured in both: sometimes a byte-identical copy (the file was copied
across), sometimes a divergent one (one client kept logging, or one copy is
truncated). The app already auto-detects Gobchat's default log dir when
`auto_detect_appdata` is on. Two needs follow: detect GobchatEx the same way, and
avoid showing the same log twice in the overview and in search results.

## Decision

**We will auto-detect the GobchatEx default log dir under the existing
`auto_detect_appdata` toggle, and collapse duplicate logs by filename on the read
boundary.** No separate opt-in: a new `config.GobchatExDefaultLogDir()` mirrors
`GobchatDefaultLogDir()`, and `effectiveDirs` adds GobchatEx *before* Gobchat so its
copies take priority on identical-content ties. Detection is runtime-only and never
stored in config.

**Dedup lives in `LogStore.List`, not in `ScanAll`.** Because `Search` builds its
candidate pool from `List()`, deduping there keeps the overview *and* search
consistent in one place, and any path the fsnotify watcher updates via `Refresh` is
re-evaluated on the next `List` with no special-casing. The full metadata set is
retained internally; only `List` filters, so `Get`/`GetEntries` by explicit path are
unaffected.

**Resolution rule per same-filename collision:**

- identical content (equal size *and* equal SHA-256) → keep the lowest
  `SourcePriority` (GobchatEx, scanned first);
- differing content → keep the newer file by OS **mtime**;
- any remaining tie → keep the lexicographically smaller path, for determinism.

`LogMeta` gains two transient (`json:"-"`, never persisted, never sent to the
frontend) fields: `ModTime`, stamped from the file's stat on every scan, and
`SourcePriority`, the index of the scan root the file came from. Content hashes are
computed lazily — only for same-name, same-size collisions — and memoized on the
store keyed by path and validated by mtime+size, so repeated `List` calls do not
re-read unchanged files. Dedup spans **all** scanned directories, not just the two
app-data defaults, so copies in custom-configured folders collapse too.

## Consequences

- **Positive:** GobchatEx users get their logs with zero configuration. The same
  session never appears twice; identical copies resolve to the GobchatEx original,
  divergent copies resolve to the freshest. Search never double-counts a hidden
  duplicate because it is never listed, opened, or indexed. Hashing cost is paid only
  on rare collisions and is memoized.
- **Negative / risks:** "Newer = mtime" is wrong if a file copy reset mtimes such that
  the *less* complete copy looks newer; the user picked mtime over message-count as the
  more literal, lower-surprise definition. Filename is the dedup key, so two genuinely
  different sessions that somehow share a filename across dirs would collapse — the
  Gobchat naming scheme makes this effectively impossible in practice. Hashing reads
  full file bytes on the `List` read path for collisions; acceptable because collisions
  are rare and cached.
- **Follow-up:** If `GobchatEx` ever changes its subfolder name (e.g. to `logs`), it is
  a one-line change in `paths.go`. If mtime proves a poor "newer" signal in the field,
  the resolution rule can switch to last-entry timestamp without changing where dedup
  runs.
