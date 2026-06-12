# Possible Modifications

Deferred candidates from a UX review of the whole app (2026-06-12). The review's items
A1–A5, B7, B8, B10, C13, and D19 (viewer state on live updates, tag/note autosave,
search-jump highlight, persistent search results, skip-version banner, search truncation
notice, snippet highlighting, search channel/sender filters, note indicator, viewer channel
filter) have been implemented; everything below is **not committed work** — each point is to
be discussed before implementation. Numbering follows the original review.

The scrollbar match-tick accuracy issue is tracked separately in
[known-issues.md](known-issues.md).

## B. Global search

6. **Exact-token AND matching is surprising.** The index matches whole tokens only — "wave"
   does not find "waves" ([search.go](../internal/search/search.go)) — while find-in-log is
   substring-based. Either prefix-match in the index or add a hint about word-based search.
   Related: sender names are not indexed, only message text — searching a player name
   globally finds textual mentions only (the sender *filter* now covers part of this gap).
9. **No date/time context on hits.** Results show only the technical filename + sender +
   snippet; show the log's date and the hit's time (line number / timestamp are already
   available in the backend entry).
11. **No shortcut to focus global search** (e.g. Ctrl+Shift+F, mirroring Ctrl+F for
    find-in-log).
12. **Hits always open in raw mode** (`targetLine` only works there). Map line→thread via
    `ThreadDTO.lines` so jumps also work in reassembled view, or at least restore the
    user's previous mode afterwards.

## C. Log list (overview sidebar)

14. **No log/filter count** — show "240 logs" / "5 of 240 match".
15. **No date navigation.** Always newest-first with no sort toggle, date-range filter, or
    month grouping; finding "that scene three months ago" still means a lot of scrolling
    when the player filter isn't enough.
16. **Date formatting is raw `toLocaleString()`** (includes seconds). Friendlier: weekday +
    short date, or relative day headers (Today / Yesterday / March 2026).
17. **No keyboard navigation / a11y.** Rows are click-only `<li>` without tabindex;
    arrow-key selection (and Enter to open) would help heavy users and accessibility.
18. **List is not virtualized** (plain `v-for` `<ul>` in
    [LogList.vue](../frontend/src/components/LogList.vue)); with years of logs the sidebar
    renders thousands of nodes eagerly. `vue-virtual-scroller` is already a dependency.

## D. Viewer

20. **No font-size setting** — accessibility; long RP reading sessions benefit from size
    control (WebView2 Ctrl+wheel zoom isn't reliably available under Wails).
21. **Cryptic emoji toggles** (🌐 hide-realm, 💬 message-only, ▽ filter-mode) rely entirely
    on hover tooltips and mix emoji with SVG iconography. Clearer icons or text toggles;
    the find group is also quite dense.
22. **No human-readable log date in the header** — only the technical filename
    (`chatlog_2026-05-16_20-09.log`) and message count; show date + duration like the
    list row.
23. **No copy/export.** Text selection across virtualized rows is unreliable, and sharing a
    scene means opening the raw file. A per-row "copy message" and an "export view as
    text/Markdown" (raw or reassembled, honoring the current filter) would fit ADR-0007 —
    logs stay read-only, exports are written elsewhere.
24. **Match navigation needs input focus.** Enter/Shift+Enter only work inside the find
    field; F3 / Ctrl+G (and Shift variants) should work globally while a query is active.
25. **Reassembly's heuristic nature is not communicated.** A small info tooltip on the
    "Reassembled" toggle ("best-effort stitching of split posts; original files untouched")
    would set expectations when stitching guesses wrong (ADR-0006/0007 spirit).

## E. Tags & notes

26. **Note is a single-line `<input>`**
    ([TagEditor.vue](../frontend/src/components/TagEditor.vue)) — multi-line notes are
    impossible; use an auto-growing textarea.
27. **Notes are not searchable anywhere** — neither global search nor the list filter sees
    note text; include notes in search or add a note-text filter.

## F. Settings, wizard, app shell

28. **Esc doesn't close the settings panel; backdrop click silently discards edits.** Add
    Esc handling and an unsaved-changes guard (or make backdrop-close not discard
    silently).
29. **No "System" theme option** — follow the OS dark/light preference in addition to the
    explicit dark/light choice.
30. **Window state isn't persisted** — always starts 1200×800 ([main.go](../main.go)), no
    MinWidth/MinHeight set. Remember size/position/maximized across sessions.
31. **No drag-and-drop** of a log folder onto the window (to add a directory), and no
    "open file location" action per log for users who want the raw file.
