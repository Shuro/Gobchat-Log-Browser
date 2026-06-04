# ADR-0006: Treat split/continuation/RP markers as best-effort player heuristics

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

Everything inside a log line's `{message}` is authored by the player, not Gobchat. FFXIV's per-message character limit leads players to split long posts and to mark roleplay segments, but conventions vary widely between people and communities. Observed in a single sample log: multi-part markers as both `(1/2)` (with parentheses) and `1/3` (without); continuations marked by a trailing ` >`, a leading `> ` / `"> `, or no marker at all. Speech/emote/OOC delimiters likewise differ.

If we treated any of these as guaranteed structure, the parser would misread or drop content.

## Decision

We will treat all such markers as **best-effort, low-confidence heuristics**. The parser always preserves the raw message and never drops a line because a marker failed to parse (unmatched lines surface as `ChannelUnknown` with raw text). Multi-part/continuation fields on `LogEntry` are explicitly documented as heuristic. The RP delimiter set (speech/emote/OOC) is a **configurable `MarkerSet` in Config**, seeded with Gobchat defaults but user-editable — no convention is hardcoded.

## Consequences

- **Positive:** Robust against the wide variety of real RP styling; rendering never blocked by a bad guess; users can tune markers to their community.
- **Negative / risks:** Heuristic grouping (e.g. reassembly, ADR-0007) will sometimes be wrong; the UI must make Raw the faithful default and treat grouped/combined views as convenience.
- **Follow-up:** Surface confidence in the UI where grouping is uncertain; keep marker config in Settings.
