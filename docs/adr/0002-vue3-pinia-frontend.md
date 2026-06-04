# ADR-0002: Vue 3 + Vite + Pinia frontend with virtual scrolling

- **Status:** Accepted
- **Date:** 2026-06-04

## Context

The Wails frontend (ADR-0001) needs a UI framework. The core view renders potentially long roleplay logs (sessions can exceed 500 entries, sometimes far more), each entry composed of multiple styled text spans. We also need reactive state for channel filters, search, tags, and runtime language switching. The maintainer writes both markup and logic.

Alternatives considered: plain JS (manual DOM diffing for span rendering and filtering gets messy), Svelte (great reactivity but weaker virtual-scroll ecosystem), React (viable, but JSX vs. co-located SFCs).

## Decision

We will use **Vue 3 with TypeScript, Vite, and Pinia**. We will use **`vue-virtual-scroller`** for the log viewer and **`vue-i18n` v9** for UI translations. Components are authored as `<script setup>` Single File Components.

## Consequences

- **Positive:** SFCs co-locate template/logic/scoped styles; Pinia gives low-boilerplate reactive state; virtual scrolling keeps large logs smooth; mature i18n.
- **Negative / risks:** Adds a Node/npm toolchain dependency for frontend builds; framework lock-in for UI code.
- **Follow-up:** Keep RP highlighting out of the frontend (see ADR-0003) so components stay presentational.
