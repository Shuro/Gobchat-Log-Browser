// Package reassemble reconstructs roleplay posts that were split across several
// log entries — sometimes interrupted by other players — into single logical
// threads. It is a pure, read-only, in-memory transformation over already-parsed
// entries: it copies, references original line numbers, and NEVER mutates the
// source (see docs/adr/0007). Grouping uses the best-effort heuristics from
// docs/adr/0006, so it is intentionally approximate; the Raw view remains the
// faithful default in the UI.
package reassemble

import (
	"regexp"
	"strings"
	"time"

	"gobchat-log-browser/internal/parser"
)

// Thread is a group of entries believed to form one logical post. Lines are the
// original 1-based line numbers in file order; Combined is the parts joined for
// display with continuation/part markers trimmed.
type Thread struct {
	Sender    string         `json:"sender"`
	Channel   parser.Channel `json:"channel"`
	Lines     []int          `json:"lines"`
	Combined  string         `json:"combined"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
}

// maxGap caps how far apart (in time) two parts may be and still be joined, so
// a continuation hours later is not glued onto an abandoned earlier post.
const maxGap = 15 * time.Minute

type openState struct {
	thread        *Thread
	parts         []string
	lastTime      time.Time
	lastPartIndex int
	expectMore    bool
}

func expectsMore(e parser.LogEntry) bool {
	return e.HasContinuation || (e.PartTotal > 0 && e.PartIndex > 0 && e.PartIndex < e.PartTotal)
}

// Reassemble groups entries (in file order) into threads. Entries that cannot
// be confidently attached to an open thread start a new (often single-line)
// thread. Grouping is keyed by sender only — continuations frequently cross
// channels (e.g. an Emote post continuing in a Say line).
func Reassemble(entries []parser.LogEntry) []Thread {
	built := make([]*Thread, 0, len(entries))
	open := map[string]*openState{}

	for _, e := range entries {
		key := e.Sender
		var st *openState
		// System lines without a sender (e.g. the game's Error channel) must
		// never be glued together just because they share the empty key.
		if key != "" {
			st = open[key]
		}

		appendable := false
		if st != nil {
			withinWindow := e.Timestamp.IsZero() || st.lastTime.IsZero() ||
				e.Timestamp.Sub(st.lastTime) <= maxGap
			isNextPart := e.PartTotal > 0 && e.PartIndex > 0 && e.PartIndex == st.lastPartIndex+1
			if withinWindow && (st.expectMore || e.IsContinuation || isNextPart) {
				appendable = true
			}
		}

		if appendable {
			st.thread.Lines = append(st.thread.Lines, e.LineNumber)
			st.parts = append(st.parts, trimMarkers(e.Message))
			st.thread.Combined = strings.Join(st.parts, " ")
			st.thread.EndTime = e.Timestamp
			st.lastTime = e.Timestamp
			if e.PartIndex > 0 {
				st.lastPartIndex = e.PartIndex
			}
			st.expectMore = expectsMore(e)
			if !st.expectMore {
				delete(open, key)
			}
			continue
		}

		t := &Thread{
			Sender:    e.Sender,
			Channel:   e.Channel,
			Lines:     []int{e.LineNumber},
			Combined:  trimMarkers(e.Message),
			StartTime: e.Timestamp,
			EndTime:   e.Timestamp,
		}
		built = append(built, t)

		ns := &openState{
			thread:        t,
			parts:         []string{t.Combined},
			lastTime:      e.Timestamp,
			lastPartIndex: e.PartIndex,
			expectMore:    expectsMore(e),
		}
		if ns.expectMore && key != "" {
			open[key] = ns // supersedes any prior open thread for this sender
		} else {
			delete(open, key)
		}
	}

	threads := make([]Thread, len(built))
	for i, t := range built {
		threads[i] = *t
	}
	return threads
}

var partTrimRe = regexp.MustCompile(`\s*\(?\d{1,3}\s*/\s*\d{1,3}\)?\s*$`)

// continuationMarkers mirrors the detection set in internal/parser. Ordered
// longest-first so trimming only ever removes a whole marker ("text ->" must
// not become "text -", "text >>" not "text >").
var continuationMarkers = []string{"->", ">>", ">", "+"}

// trimMarkers strips leading/trailing continuation markers and a trailing
// multi-part marker from a message for display in a combined thread. It leaves
// the inner roleplay punctuation untouched.
func trimMarkers(msg string) string {
	s := strings.TrimSpace(msg)
	// A leading marker may sit behind an opening speech quote; the quote is
	// dropped together with the marker. A quote without a marker stays.
	lead := strings.TrimPrefix(s, `"`)
	for _, m := range continuationMarkers {
		if strings.HasPrefix(lead, m) {
			s = strings.TrimLeft(lead[len(m):], " ")
			break
		}
	}
	s = strings.TrimRight(s, " ")
	for _, m := range continuationMarkers {
		if strings.HasSuffix(s, m) {
			s = strings.TrimRight(s[:len(s)-len(m)], " ")
			break
		}
	}
	s = partTrimRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
