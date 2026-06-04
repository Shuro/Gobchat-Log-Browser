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
