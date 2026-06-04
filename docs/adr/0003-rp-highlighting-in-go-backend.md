# ADR-0003: Perform RP highlighting in the Go backend, return typed spans

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

Roleplay messages contain styled segments: spoken dialogue (various quote styles including reversed pairs like `»…«` vs `«…»`), emotes (`*…*`, `<…>`), out-of-character text (`((…))`), and configured name mentions. Matching these correctly is stateful (paired/reversed delimiters, nested mentions inside speech) — fragile with a single regex. The styling must be unit-testable and the delimiter set is user-configurable (see ADR-0006).

The choice is where to compute the segmentation: in Go, or in the frontend with JavaScript.

## Decision

We will compute highlighting in **Go**, in `internal/highlight`, as a left-to-right stateful tokenizer returning a **flat, non-overlapping `[]Span`**. When a mention falls inside a speech/emote span, that span is split (`[outer, mention, outer]`) so the result stays flat. `GetLogEntries` returns spans pre-computed inside each entry DTO; the frontend only renders colored spans by type.

## Consequences

- **Positive:** Tokenizer is testable with `go test` (no browser); one implementation, not duplicated in JS; frontend stays presentational; configurable markers handled in one place.
- **Negative / risks:** Changing mention names or markers requires re-fetching/re-tokenizing rather than a pure client re-render. Per-entry tokenization cost scales with log size (mitigated by caching parsed+tokenized results in the log store).
- **Follow-up:** Cache tokenized entries alongside parsed entries; benchmark on large sessions.
