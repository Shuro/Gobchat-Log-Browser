package highlight

import (
	"strings"
	"testing"
)

// assertReconstructs verifies the core invariant: concatenating span texts in
// order reproduces the original message, and offsets are contiguous. This
// matters because the viewer renders spans verbatim — a gap or overlap would
// silently drop or duplicate message text.
func assertReconstructs(t *testing.T, message string, spans []Span) {
	t.Helper()
	var b strings.Builder
	prevEnd := 0
	for _, s := range spans {
		if s.Start != prevEnd {
			t.Fatalf("non-contiguous spans: span starts at %d, expected %d (message=%q)", s.Start, prevEnd, message)
		}
		if s.Text != message[s.Start:s.End] {
			t.Fatalf("span text %q != message slice %q", s.Text, message[s.Start:s.End])
		}
		b.WriteString(s.Text)
		prevEnd = s.End
	}
	if b.String() != message {
		t.Fatalf("reconstructed %q != original %q", b.String(), message)
	}
}

func types(spans []Span) []SpanType {
	out := make([]SpanType, len(spans))
	for i, s := range spans {
		out[i] = s.Type
	}
	return out
}

func TestTokenizeDefaultMarkers(t *testing.T) {
	m := DefaultMarkerSet()
	cases := []struct {
		name string
		in   string
		want []SpanType
	}{
		{"plain only", "just narration here", []SpanType{SpanTypePlain}},
		{"straight quotes", `"Hello there."`, []SpanType{SpanTypeSpeech}},
		{"speech then plain", `"Hi." she waved`, []SpanType{SpanTypeSpeech, SpanTypePlain}},
		{"emote stars", `*waves*`, []SpanType{SpanTypeEmote}},
		{"ooc parens", `((out of character))`, []SpanType{SpanTypeOOC}},
		{"german guillemets", `»Guten Tag«`, []SpanType{SpanTypeSpeech}},
		{"french guillemets", `«Bonjour»`, []SpanType{SpanTypeSpeech}},
		{"plain speech plain", `she said "yes" softly`, []SpanType{SpanTypePlain, SpanTypeSpeech, SpanTypePlain}},
		{"trailing gt is not emote", `ponders the point >`, []SpanType{SpanTypePlain}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spans := Tokenize(c.in, m, nil)
			if got := types(spans); !equalTypes(got, c.want) {
				t.Fatalf("types = %v, want %v (spans=%+v)", got, c.want, spans)
			}
			assertReconstructs(t, c.in, spans)
		})
	}
}

func TestTokenizeMentionInsideSpeech(t *testing.T) {
	m := DefaultMarkerSet()
	in := `"Hello Alpha, glad you came."`
	spans := Tokenize(in, m, []string{"Alpha"})
	assertReconstructs(t, in, spans)

	var sawMention bool
	for _, s := range spans {
		if s.Type == SpanTypeMention {
			sawMention = true
			if s.Text != "Alpha" {
				t.Fatalf("mention text = %q, want %q", s.Text, "Alpha")
			}
		}
	}
	if !sawMention {
		t.Fatalf("expected a mention span, got %+v", spans)
	}
	// The surrounding fragments must keep the speech type.
	if spans[0].Type != SpanTypeSpeech {
		t.Fatalf("first span type = %v, want speech", spans[0].Type)
	}
}

func TestTokenizeCustomMarkerSet(t *testing.T) {
	// A community that marks speech with a custom delimiter and disables others.
	m := MarkerSet{Speech: []MarkerPair{{"[[", "]]"}}}
	in := `narr [[spoken words]] narr`
	spans := Tokenize(in, m, nil)
	assertReconstructs(t, in, spans)
	want := []SpanType{SpanTypePlain, SpanTypeSpeech, SpanTypePlain}
	if got := types(spans); !equalTypes(got, want) {
		t.Fatalf("types = %v, want %v", got, want)
	}
}

func TestTokenizeUnterminatedDelimiter(t *testing.T) {
	m := DefaultMarkerSet()
	// An opening quote with no close must not create a speech span; it stays plain.
	in := `she said "hello with no close`
	spans := Tokenize(in, m, nil)
	assertReconstructs(t, in, spans)
	for _, s := range spans {
		if s.Type == SpanTypeSpeech {
			t.Fatalf("did not expect a speech span for unterminated quote: %+v", spans)
		}
	}
}

func equalTypes(a, b []SpanType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
