# Known Issues

## Find-in-log scrollbar match ticks are approximate

**Status:** open (improved, not pixel-perfect) — 2026-06-10

**Symptom:** The gold tick marks next to the log scrollbar (find-in-log
matches) do not always line up exactly with where the matching line sits when
scrolled into view. Most visible in non-maximized windows with long, wrapped
RP paragraphs.

**Architecture involved:**

- `frontend/src/components/LogViewer.vue` — `tickPercents` computes tick
  positions; `prefillSizes()` pre-fills the virtual scroller's size map.
- `frontend/src/utils/rowHeights.ts` — canvas-based row height estimation.
- `vue-virtual-scroller@2.0.0-beta.8` (`DynamicScroller`) — only measures rows
  that have actually been rendered; everything else is an estimate.

**What was done so far (in order):**

1. Index-fraction ticks (`matchIndex / totalItems`) — badly wrong with
   variable row heights.
2. Ticks computed from the scroller's reactive `itemsWithSize` (measured
   height or `min-item-size` fallback) — consistent with the thumb, but both
   drifted/jumped as rows got measured during scrolling.
3. Pre-filling `vscrollData.sizes` with per-row height estimates
   (`rowHeights.ts`: greedy word-wrap with cached canvas glyph widths,
   mirroring the `.entry` CSS metrics) for every row on log load and window
   resize. Rendered rows get corrected by the scroller's ResizeObserver.
4. Removed the native scrollbar arrow buttons (`::-webkit-scrollbar-button`)
   so the thumb's travel range spans the full track height, matching the
   tick overlay's coordinate space.

**Remaining known error sources:**

- Height estimates measure regular-weight glyphs; bold mention spans (and any
  future styling that changes advance widths) can make a paragraph wrap one
  line more than predicted until the row has been rendered once.
- Greedy word-wrap is not identical to the browser's line breaker (hyphens,
  break-word edge cases, kerning).
- A tick marks the match row's document midpoint; the thumb marks the
  viewport, which is taller than one row — up to a thumb-height of perceived
  offset is inherent.
- Rows measured at one window width keep that measurement after a resize
  unless re-rendered (library behavior); estimates are recomputed, real
  measurements are not.

**Ideas if this needs to be exact later:**

- Measure all rows for real by rendering them offscreen in batches per
  log/width (cost: one-off layout pass per log open).
- Or replace the estimate with the measured average of rendered rows as a
  correction factor per log.
- Or render ticks only for the measured region and fade the rest.
