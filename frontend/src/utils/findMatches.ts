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

// splitMatchesAny is the multi-term variant: every occurrence of any term is
// a match segment, with overlapping/adjacent hits merged. Used for global
// search snippets, where the query is a set of AND terms rather than one
// contiguous substring.
export function splitMatchesAny(text: string, terms: string[]): MatchSegment[] {
  const clean = terms.map((t) => t.trim().toLowerCase()).filter(Boolean)
  if (clean.length === 0 || !text) return [{ text, match: false }]
  const lower = text.toLowerCase()

  // Collect all hit ranges, then merge overlaps.
  const ranges: Array<[number, number]> = []
  for (const term of clean) {
    let pos = 0
    for (;;) {
      const hit = lower.indexOf(term, pos)
      if (hit < 0) break
      ranges.push([hit, hit + term.length])
      pos = hit + term.length
    }
  }
  if (ranges.length === 0) return [{ text, match: false }]
  ranges.sort((a, b) => a[0] - b[0])
  const merged: Array<[number, number]> = [ranges[0]]
  for (let i = 1; i < ranges.length; i++) {
    const last = merged[merged.length - 1]
    if (ranges[i][0] <= last[1]) last[1] = Math.max(last[1], ranges[i][1])
    else merged.push(ranges[i])
  }

  const segments: MatchSegment[] = []
  let pos = 0
  for (const [start, end] of merged) {
    if (start > pos) segments.push({ text: text.slice(pos, start), match: false })
    segments.push({ text: text.slice(start, end), match: true })
    pos = end
  }
  if (pos < text.length) segments.push({ text: text.slice(pos), match: false })
  return segments
}
