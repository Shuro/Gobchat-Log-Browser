import type { api } from '../../wailsjs/go/models'

// Estimates rendered row heights for EntryRow/ThreadRow without rendering
// them, by word-wrapping the message text with canvas-measured glyph widths.
// The virtual scroller only measures rows it has rendered and assumes
// min-item-size for the rest, which makes the scrollbar (and the find match
// ticks) jump around; pre-filling its size map with these estimates keeps the
// geometry stable. Estimates are replaced by exact measurements once a row is
// rendered, so they only need to be close, not perfect.
//
// All constants below mirror style.css (.entry and its children) — keep them
// in sync.

const FAMILY = "'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
const MSG_FONT = `14px ${FAMILY}`
const SENDER_FONT = `700 14px ${FAMILY}`
const TIME_FONT = `12px ${FAMILY}`
const CHANNEL_FONT = `10px ${FAMILY}`
const SMALL_FONT = `11px ${FAMILY}`
const LINE_HEIGHT = 14 * 1.45 // .entry font-size * line-height
const PAD_X = 24 // .entry horizontal padding
const GAP = 8 // .entry flex gap

let canvasCtx: CanvasRenderingContext2D | null = null
function ctx(): CanvasRenderingContext2D {
  canvasCtx ??= document.createElement('canvas').getContext('2d')!
  return canvasCtx
}

// Per-font glyph width cache; summing cached glyph widths (ignoring kerning)
// is far cheaper than calling measureText per word.
const glyphCaches = new Map<string, Map<string, number>>()

function textWidth(text: string, font: string): number {
  let cache = glyphCaches.get(font)
  if (!cache) {
    cache = new Map()
    glyphCaches.set(font, cache)
  }
  let w = 0
  for (const ch of text) {
    let cw = cache.get(ch)
    if (cw === undefined) {
      const c = ctx()
      c.font = font
      cw = c.measureText(ch).width
      cache.set(ch, cw)
    }
    w += cw
  }
  return w
}

// Greedy word wrap matching `white-space: pre-wrap; word-break: break-word`.
function wrappedLines(text: string, width: number, font: string): number {
  if (!text || width <= 0) return 1
  const spaceW = textWidth(' ', font)
  let lines = 0
  for (const para of text.split('\n')) {
    lines++
    let x = 0
    let first = true
    for (const word of para.split(' ')) {
      const w = textWidth(word, font)
      const need = first ? w : spaceW + w
      if (x + need <= width) {
        x += need
      } else if (w <= width) {
        lines++
        x = w
      } else {
        // Overlong word: break-word lets it fill and wrap mid-word.
        const total = x + need
        lines += Math.floor(total / width)
        x = total % width
      }
      first = false
    }
  }
  return Math.max(lines, 1)
}

// The time column renders a fixed-format locale time; measure a sample once.
let timeColWidth: number | null = null
function timeWidth(timestamp: string): number {
  if (!timestamp || isNaN(new Date(timestamp).getTime())) return 0
  timeColWidth ??= textWidth(
    new Date().toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }),
    TIME_FONT,
  )
  return timeColWidth
}

function channelWidth(channel: string): number {
  return Math.max(52, textWidth((channel || 'unknown').toUpperCase(), CHANNEL_FONT))
}

export function estimateEntryHeight(e: api.EntryDTO, containerWidth: number): number {
  let fixed = timeWidth(e.timestamp) + channelWidth(e.channel)
  let children = 3 // time, channel-tag, message
  if (e.channel !== 'Unknown') {
    let s = textWidth(e.display_name || e.sender, SENDER_FONT)
    if (e.status_symbol) s += textWidth(e.status_symbol, SENDER_FONT) + 2
    if (e.realm) s += textWidth(e.realm, SMALL_FONT) + 2
    fixed += s
    children++
  }
  if (e.part_total > 0) {
    fixed += textWidth(`${e.part_index}/${e.part_total}`, SMALL_FONT)
    children++
  }
  const msgWidth = containerWidth - PAD_X - fixed - GAP * (children - 1)
  const lines = wrappedLines(e.message, msgWidth, MSG_FONT)
  return Math.round(lines * LINE_HEIGHT + 6) // + .entry vertical padding
}

export function estimateThreadHeight(t: api.ThreadDTO, containerWidth: number): number {
  let fixed = channelWidth(t.channel) + textWidth(t.sender, SENDER_FONT)
  let children = 3 // channel-tag, sender, message
  if (t.lines.length > 1) {
    fixed += textWidth(`${t.lines.length} parts`, SMALL_FONT) // i18n approximation
    children++
  }
  const msgWidth = containerWidth - PAD_X - fixed - GAP * (children - 1)
  const lines = wrappedLines(t.combined, msgWidth, MSG_FONT)
  // + .entry.thread vertical padding and bottom border
  return Math.round(lines * LINE_HEIGHT + 13)
}
