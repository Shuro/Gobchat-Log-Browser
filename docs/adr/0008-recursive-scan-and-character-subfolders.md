# ADR-0008: Scan log directories recursively and capture character subfolders

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

Inspecting a real Gobchat log directory (`%APPDATA%\Gobchat\log`) showed two coexisting layouts: many logs sit flat at the top level (~179 files) while others live in subfolders (e.g. `Nevio/`, `Citrine/`, ~218 files across subfolders). These subfolders are **user-created**: Gobchat's log path (`behaviour.chatlog.path`) is configurable **per profile**, so a user commonly points each profile (often one per character) at its own subfolder. This is a user/profile choice, not a Gobchat version behaviour. The same directory also contains Electron/Chromium runtime folders (`DawnCache`, `GPUCache`, etc.) that hold no logs.

The original scanner globbed only `*.log` at the top level (non-recursive), which would miss the majority of a real user's logs. The directory also held ~397 logs total, enough that parsing them all synchronously at startup would visibly delay the window.

## Decision

- **Scan recursively.** `ScanDirectory` walks each configured root with `filepath.WalkDir`, collecting every `*.log` file at any depth.
- **Skip Electron cache and hidden directories** (`DawnCache`, `GPUCache`, `Cache`, `Code Cache`, `blob_storage`, `Local Storage`, `Session Storage`, `IndexedDB`, `Network`, and any dotfile dir) to avoid wasted IO; they contain no logs.
- **Capture the subfolder** as `LogMeta.Folder` (relative to the scan root, "" for top-level). Because users typically name these per-profile folders after a character, the UI can group/label logs by that folder.
- **Watch recursively.** The scan returns every discovered directory; the fsnotify watcher (which is not recursive) is given all of them, and newly created subdirectories are added on the fly.
- **Scan off the startup path.** The initial scan runs in a goroutine and emits a `logs:scanned` event so the window appears immediately and the frontend refreshes when the scan finishes.

## Consequences

- **Positive:** All of a real user's logs are found regardless of layout; cache folders are ignored; per-character grouping becomes possible; startup stays responsive.
- **Negative / risks:** Full parsing of every file for metadata is still O(files) work — acceptable at observed scale (~400 files) but a candidate for a lighter header-only meta pass if libraries grow much larger. Recursive watching adds one fsnotify watch per subfolder.
- **Follow-up:** Consider a streaming/quick metadata extractor if scan time becomes noticeable; expose character-folder grouping in the overview UI.
