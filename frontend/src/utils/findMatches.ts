export interface MatchSegment {
  text: string
  match: boolean
}

// splitMatches cuts text into match/non-match segments for the given query,
// case-insensitive. An empty/whitespace query yields the whole text unmatched.
export function splitMatches(text: string, query: string): MatchSegment[] {
  const q = query.trim().toLowerCase()
  if (!q || !text) return [{ text, match: false }]
  const lower = text.toLowerCase()
  const segments: MatchSegment[] = []
  let pos = 0
  for (;;) {
    const hit = lower.indexOf(q, pos)
    if (hit < 0) break
    if (hit > pos) segments.push({ text: text.slice(pos, hit), match: false })
    segments.push({ text: text.slice(hit, hit + q.length), match: true })
    pos = hit + q.length
  }
  if (segments.length === 0) return [{ text, match: false }]
  if (pos < text.length) segments.push({ text: text.slice(pos), match: false })
  return segments
}
