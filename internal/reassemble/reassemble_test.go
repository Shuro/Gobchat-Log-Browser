package reassemble

import (
	"testing"
	"time"

	"gobchat-log-browser/internal/parser"
)

func at(base time.Time, secs int) time.Time { return base.Add(time.Duration(secs) * time.Second) }

func TestReassembleNumberedParts(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	entries := []parser.LogEntry{
		{LineNumber: 1, Sender: "Alpha", Channel: parser.ChannelSay, Message: "first part (1/2)", Timestamp: at(base, 0), PartIndex: 1, PartTotal: 2},
		{LineNumber: 2, Sender: "Alpha", Channel: parser.ChannelSay, Message: "second part (2/2)", Timestamp: at(base, 5), PartIndex: 2, PartTotal: 2},
	}
	threads := Reassemble(entries)
	if len(threads) != 1 {
		t.Fatalf("threads = %d, want 1", len(threads))
	}
	if got := threads[0].Lines; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("lines = %v, want [1 2]", got)
	}
	if threads[0].Combined != "first part second part" {
		t.Fatalf("combined = %q", threads[0].Combined)
	}
}

func TestReassembleInterruptedByOtherSpeaker(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	entries := []parser.LogEntry{
		{LineNumber: 1, Sender: "Alpha", Channel: parser.ChannelSay, Message: "intro (1/3)", Timestamp: at(base, 0), PartIndex: 1, PartTotal: 3},
		{LineNumber: 2, Sender: "Beta", Channel: parser.ChannelSay, Message: "an interjection", Timestamp: at(base, 3)},
		{LineNumber: 3, Sender: "Alpha", Channel: parser.ChannelSay, Message: "middle (2/3)", Timestamp: at(base, 6), PartIndex: 2, PartTotal: 3},
		{LineNumber: 4, Sender: "Alpha", Channel: parser.ChannelSay, Message: "end (3/3)", Timestamp: at(base, 9), PartIndex: 3, PartTotal: 3},
	}
	threads := Reassemble(entries)
	if len(threads) != 2 {
		t.Fatalf("threads = %d, want 2 (Alpha + Beta)", len(threads))
	}
	// First built thread is Alpha's (started at line 1) and must contain all 3 parts.
	alpha := threads[0]
	if alpha.Sender != "Alpha" {
		t.Fatalf("threads[0].Sender = %q, want Alpha", alpha.Sender)
	}
	if got := alpha.Lines; len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 4 {
		t.Fatalf("alpha lines = %v, want [1 3 4]", got)
	}
}

func TestReassembleGtContinuationCrossChannel(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	// An Emote post ends with ">", continued in a Say line starting with `"> `.
	entries := []parser.LogEntry{
		{LineNumber: 1, Sender: "Nevio", Channel: parser.ChannelEmote, Message: "speaks at length >", Timestamp: at(base, 0), HasContinuation: true},
		{LineNumber: 2, Sender: "Nevio", Channel: parser.ChannelSay, Message: `"> and concludes."`, Timestamp: at(base, 30), IsContinuation: true},
	}
	threads := Reassemble(entries)
	if len(threads) != 1 {
		t.Fatalf("threads = %d, want 1", len(threads))
	}
	th := threads[0]
	if len(th.Lines) != 2 {
		t.Fatalf("lines = %v, want 2 entries", th.Lines)
	}
	if th.Channel != parser.ChannelEmote {
		t.Fatalf("thread channel = %q, want first part's channel Emote", th.Channel)
	}
	if th.Combined != `speaks at length and concludes."` {
		t.Fatalf("combined = %q", th.Combined)
	}
}

// All hardcoded continuation markers (->, >>, +, and mixing them) must join
// into one thread with the markers fully trimmed — a partial trim like
// "text -" or "text >" leaking into Combined would corrupt the RP text.
func TestReassembleNewMarkerVariants(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	cases := []struct {
		name, first, second, combined string
	}{
		{"arrow", "speaks at length ->", "-> and concludes.", "speaks at length and concludes."},
		{"double-gt", "speaks at length >>", ">> and concludes.", "speaks at length and concludes."},
		{"plus", "speaks at length +", "+ and concludes.", "speaks at length and concludes."},
		{"mixed", "speaks at length >", "-> and concludes.", "speaks at length and concludes."},
	}
	for _, c := range cases {
		entries := []parser.LogEntry{
			{LineNumber: 1, Sender: "Nevio", Channel: parser.ChannelEmote, Message: c.first, Timestamp: at(base, 0), HasContinuation: true},
			{LineNumber: 2, Sender: "Nevio", Channel: parser.ChannelSay, Message: c.second, Timestamp: at(base, 30), IsContinuation: true},
		}
		threads := Reassemble(entries)
		if len(threads) != 1 {
			t.Fatalf("%s: threads = %d, want 1", c.name, len(threads))
		}
		if threads[0].Combined != c.combined {
			t.Fatalf("%s: combined = %q, want %q", c.name, threads[0].Combined, c.combined)
		}
	}
}

// trimMarkers must remove whole edge markers only: inner punctuation is
// player-authored RP text and has to survive untouched (docs/adr/0006).
func TestTrimMarkers(t *testing.T) {
	cases := []struct{ in, out string }{
		{"text >", "text"},
		{"text ->", "text"},
		{"text >>", "text"},
		{"text +", "text"},
		{"> lead", "lead"},
		{"-> lead", "lead"},
		{">> lead", "lead"},
		{"+ lead", "lead"},
		{`"> quoted.`, "quoted."},
		{`"-> quoted.`, "quoted."},
		{`">> quoted.`, "quoted."},
		{"2 + 2 = 4", "2 + 2 = 4"},                      // no edge marker, nothing trimmed
		{"five+", "five"},                               // exact trailing marker strip
		{"keep -inner-> text ->", "keep -inner-> text"}, // inner markers stay
		{`"a quote without marker`, `"a quote without marker`}, // lone quote stays
		{"part one (1/2) >", "part one"},                       // part marker after continuation
		{"ends with part (2/3)", "ends with part"},
	}
	for _, c := range cases {
		if got := trimMarkers(c.in); got != c.out {
			t.Fatalf("trimMarkers(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

// System lines without a sender (e.g. the game's Error channel) share the empty
// sender key; they must never be glued together by the continuation heuristics.
func TestReassembleEmptySenderNeverJoins(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	entries := []parser.LogEntry{
		{LineNumber: 1, Sender: "", Channel: parser.Channel("Error"), Message: "Message could not be sent +", Timestamp: at(base, 0), HasContinuation: true},
		{LineNumber: 2, Sender: "", Channel: parser.Channel("Error"), Message: "+ Another unrelated error.", Timestamp: at(base, 5), IsContinuation: true},
	}
	threads := Reassemble(entries)
	if len(threads) != 2 {
		t.Fatalf("threads = %d, want 2 (empty senders must not merge)", len(threads))
	}
}

func TestReassembleSeparatePostsStaySeparate(t *testing.T) {
	base := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC)
	entries := []parser.LogEntry{
		{LineNumber: 1, Sender: "Alpha", Channel: parser.ChannelSay, Message: "a complete thought.", Timestamp: at(base, 0)},
		{LineNumber: 2, Sender: "Alpha", Channel: parser.ChannelSay, Message: "an unrelated later thought.", Timestamp: at(base, 600)},
	}
	threads := Reassemble(entries)
	if len(threads) != 2 {
		t.Fatalf("threads = %d, want 2 (no false merge)", len(threads))
	}
}
