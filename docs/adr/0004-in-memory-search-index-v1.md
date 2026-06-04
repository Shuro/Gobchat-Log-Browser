# ADR-0004: Lazy in-memory inverted index for search (v1)

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

The app needs global search across all logs and per-log search. A typical user has at most a few hundred RP session files, each a few thousand lines. We want fast search without heavy dependencies or a complex build (e.g. CGO-based SQLite FTS, or a large pure-Go engine like Bleve), and without breaking the static-binary guarantee from ADR-0001.

## Decision

For v1 we will build a **lazy, in-memory inverted index** in `internal/search`. The index is populated when a log is first opened or searched (goroutine-per-file parsing capped at 4 concurrent). Tokens are Unicode word-split and lowercased, no stemming or stopwords (RP text contains many proper/fictional nouns). Global search hits the index; per-log search is a linear scan over the already-cached `[]LogEntry` with no backend round-trip. Results are capped (default 200) and ranked by term-match count.

## Consequences

- **Positive:** Zero external dependencies; no CGO; instant search for already-opened files; simple to reason about and test.
- **Negative / risks:** Whole index lives in RAM (acceptable at expected scale: ~200k entries < ~100 MB); index is rebuilt each launch (no persistence in v1).
- **Follow-up:** If users accumulate very large corpora, a v2 can persist via `encoding/gob` or swap in a Bleve backend behind the same `Index` interface — that would warrant a superseding ADR.
