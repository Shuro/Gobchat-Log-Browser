# ADR-0001: Use Go + Wails v2 with a single static binary

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

We are building a cross-platform desktop GUI to browse Gobchat FFXIV roleplay chat logs. Primary target is Windows, with Linux and macOS planned later. The app must parse text logs, render them with rich roleplay highlighting, search across files, and ship as an easy-to-install desktop application. The language was not pre-decided; Go was a candidate.

Requirements that shape the choice:
- Rich, styled text rendering (colored speech/emote/OOC spans) — easier with web tech than native toolkits.
- Cross-platform with minimal per-platform UI work.
- Simple distribution, ideally a single executable.
- Strong text/file-processing story for the parser and search.

## Decision

We will build the application with **Go (backend) + Wails v2 (desktop shell)**, compiling to a **single self-contained binary** per platform. The frontend is a web UI embedded into the binary by Wails; the Go side owns file I/O, parsing, highlighting, search, and persistence.

We will build with `CGO_ENABLED=0` to keep the binary fully static and dependency-free.

## Consequences

- **Positive:** One binary to distribute; Go's standard library covers file/text/concurrency needs well; web frontend enables flexible RP styling; cross-compilation is straightforward.
- **Negative / risks:** On Windows the WebView2 runtime is required (ships with Windows 11, so acceptable for the primary target; older systems may need the runtime installer). Wails v2 ties us to its build tooling and conventions.
- **Follow-up:** Keep all platform-specific paths behind `internal/config/paths.go`. Revisit if a feature needs CGO (would break the static-binary guarantee).
