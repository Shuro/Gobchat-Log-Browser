# ADR-0005: Store tags and notes in a filename-keyed JSON sidecar

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

Users want to manually categorize logs with tags and free-text notes. This metadata must not modify the log files themselves (see ADR-0007, read-only logs) and should survive if the user moves or re-points their log directory. The overview panel displays tags and notes per log.

## Decision

We will store tags and notes in a single **JSON sidecar** at `%APPDATA%\GobchatLogBrowser\tags.json`, **keyed by log filename** (not full path). Writes use atomic replace (write `.tmp`, then rename). `internal/tags` owns CRUD plus a distinct-tag list for autocomplete.

## Consequences

- **Positive:** Log files stay untouched; tags survive directory moves/renames; trivial to back up or hand-edit; no database.
- **Negative / risks:** Filename collisions across different directories would share tags (Gobchat filenames embed a timestamp, so collisions are unlikely but possible). Deleting/renaming a log orphans its entry until cleaned up.
- **Follow-up:** If collisions become a real problem, key by filename + a content hash and migrate via a superseding ADR.
