// Package highlight segments a raw roleplay message into typed, non-overlapping
// spans (speech, emote, OOC, mention, plain) for display. The delimiter set is
// configurable (see MarkerSet) because roleplay punctuation conventions vary
// between players and communities — see docs/adr/0006.
package highlight

import (
	"sort"
	"strings"
)

// SpanType classifies a segment of a message.
type SpanType string

const (
	SpanTypePlain   SpanType = "plain"
	SpanTypeSpeech  SpanType = "speech"
	SpanTypeEmote   SpanType = "emote"
	SpanTypeOOC     SpanType = "ooc"
	SpanTypeMention SpanType = "mention"
)

// Span is one contiguous, typed segment of a message. Start/End are byte
// offsets into the original message; concatenating all spans' Text in order
// reproduces the message exactly.
type Span struct {
	Type  SpanType `json:"type"`
	Text  string   `json:"text"`
	Start int      `json:"start"`
	End   int      `json:"end"`
}

// MarkerPair is one open/close delimiter for a span type. Open and Close may
// differ (e.g. »…«) and may be multi-byte.
type MarkerPair struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// MarkerSet holds the configurable delimiters per span type. It mirrors
// Gobchat's own configurable markers; DefaultMarkerSet matches Gobchat defaults.
type MarkerSet struct {
	Speech []MarkerPair `json:"speech"`
	Emote  []MarkerPair `json:"emote"`
	OOC    []MarkerPair `json:"ooc"`
}

// DefaultMarkerSet returns the Gobchat default delimiters. Users may extend or
// replace these in their config.
func DefaultMarkerSet() MarkerSet {
	return MarkerSet{
		Speech: []MarkerPair{
			{`"`, `"`},
			{`„`, `”`},
			{`“`, `”`},
			{`»`, `«`},
			{`«`, `»`},
		},
		Emote: []MarkerPair{
			{`*`, `*`},
			{`<`, `>`},
		},
		OOC: []MarkerPair{
			{`((`, `))`},
		},
	}
}

type typedMarker struct {
	pair MarkerPair
	typ  SpanType
}

// ordered returns all markers with the longest open delimiter first, so that
// e.g. "((" is preferred over "(" when both are configured.
func (m MarkerSet) ordered() []typedMarker {
	var ms []typedMarker
	for _, p := range m.OOC {
		ms = append(ms, typedMarker{p, SpanTypeOOC})
	}
	for _, p := range m.Speech {
		ms = append(ms, typedMarker{p, SpanTypeSpeech})
	}
	for _, p := range m.Emote {
		ms = append(ms, typedMarker{p, SpanTypeEmote})
	}
	sort.SliceStable(ms, func(i, j int) bool {
		return len(ms[i].pair.Open) > len(ms[j].pair.Open)
	})
	return ms
}

// Tokenize segments message into a flat list of typed spans. It is a
// left-to-right stateful scanner: it finds top-level delimited regions
// (speech/emote/OOC) and never nests them, then splits every resulting span on
// any configured mention name so the output stays flat (e.g. a mention inside
// speech yields [speech, mention, speech]). markers and mentionNames come from
// config; nothing is hardcoded.
func Tokenize(message string, markers MarkerSet, mentionNames []string) []Span {
	coarse := scanDelimited(message, markers)
	out := make([]Span, 0, len(coarse))
	for _, s := range coarse {
		out = append(out, splitMentions(s, mentionNames)...)
	}
	return out
}

// scanDelimited produces coarse spans of plain/speech/emote/ooc, without
// nesting and without mention handling.
func scanDelimited(message string, markers MarkerSet) []Span {
	ms := markers.ordered()
	spans := make([]Span, 0, 4)
	n := len(message)
	i := 0
	plainStart := 0
	for i < n {
		matched := false
		for _, tm := range ms {
			op := tm.pair.Open
			// An empty close would match at offset 0 and create degenerate
			// spans, so half-configured pairs are skipped entirely.
			if op == "" || tm.pair.Close == "" || !strings.HasPrefix(message[i:], op) {
				continue
			}
			rel := strings.Index(message[i+len(op):], tm.pair.Close)
			if rel < 0 {
				continue // no closing delimiter; not a span here
			}
			end := i + len(op) + rel + len(tm.pair.Close)
			if i > plainStart {
				spans = append(spans, Span{SpanTypePlain, message[plainStart:i], plainStart, i})
			}
			spans = append(spans, Span{tm.typ, message[i:end], i, end})
			i = end
			plainStart = i
			matched = true
			break
		}
		if !matched {
			i++
		}
	}
	if plainStart < n {
		spans = append(spans, Span{SpanTypePlain, message[plainStart:n], plainStart, n})
	}
	return spans
}

// splitMentions splits one span wherever a mention name occurs (case-insensitive),
// emitting mention spans and preserving the original type for the rest.
func splitMentions(s Span, names []string) []Span {
	if len(names) == 0 {
		return []Span{s}
	}
	text := s.Text
	lower := strings.ToLower(text)

	type hit struct{ start, end int }
	var hits []hit
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		ln := strings.ToLower(name)
		from := 0
		for {
			idx := strings.Index(lower[from:], ln)
			if idx < 0 {
				break
			}
			abs := from + idx
			hits = append(hits, hit{abs, abs + len(ln)})
			from = abs + len(ln)
		}
	}
	if len(hits) == 0 {
		return []Span{s}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].start < hits[j].start })

	merged := hits[:0:0]
	for _, h := range hits {
		if len(merged) > 0 && h.start < merged[len(merged)-1].end {
			if h.end > merged[len(merged)-1].end {
				merged[len(merged)-1].end = h.end
			}
			continue
		}
		merged = append(merged, h)
	}

	out := make([]Span, 0, len(merged)*2+1)
	cur := 0
	for _, h := range merged {
		if h.start > cur {
			out = append(out, Span{s.Type, text[cur:h.start], s.Start + cur, s.Start + h.start})
		}
		out = append(out, Span{SpanTypeMention, text[h.start:h.end], s.Start + h.start, s.Start + h.end})
		cur = h.end
	}
	if cur < len(text) {
		out = append(out, Span{s.Type, text[cur:], s.Start + cur, s.Start + len(text)})
	}
	return out
}
