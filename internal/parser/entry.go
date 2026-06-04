// Package parser reads Gobchat plain-text chat logs into structured entries.
// Logs are opened read-only and never modified (see docs/adr/0007). Everything
// inside a message is player-authored, so split/continuation markers are treated
// as best-effort heuristics, never guaranteed structure (see docs/adr/0006).
package parser

import "time"

// Channel identifies a chat channel. The constants cover Gobchat's common
// channels but the list is NOT exhaustive: any unrecognised channel token is
// stored verbatim as Channel(token) rather than discarded.
type Channel string

const (
	ChannelSay           Channel = "Say"
	ChannelEmote         Channel = "Emote"
	ChannelYell          Channel = "Yell"
	ChannelShout         Channel = "Shout"
	ChannelTellSend      Channel = "TellSend"
	ChannelTellReceive   Channel = "TellRecieve" // matches Gobchat's spelling
	ChannelParty         Channel = "Party"
	ChannelGuild         Channel = "Guild"
	ChannelAlliance      Channel = "Alliance"
	ChannelNPCDialog     Channel = "NPC_Dialog"
	ChannelAnimatedEmote Channel = "AnimatedEmote"
	ChannelGobchatInfo   Channel = "GobchatInfo"
	ChannelUnknown       Channel = "Unknown" // line did not match the format; raw text preserved
)

// LogEntry is one parsed log line. Message is always populated, even when the
// line failed to match the format (in which case Channel is ChannelUnknown and
// Message holds the raw line) — no line is ever dropped.
type LogEntry struct {
	LineNumber   int       `json:"line_number"`
	Channel      Channel   `json:"channel"`
	Timestamp    time.Time `json:"timestamp"`
	Sender       string    `json:"sender"`        // raw, e.g. "★M'iqo Tester [Shiva]"
	DisplayName  string    `json:"display_name"`  // stripped, e.g. "M'iqo Tester"
	Realm        string    `json:"realm"`         // any "[…]" suffix, or ""
	StatusSymbol string    `json:"status_symbol"` // leading non-letter symbol(s), or ""
	Message      string    `json:"message"`       // raw message text

	// Heuristic, low-confidence fields (player-typed conventions). They never
	// cause an entry to be skipped; on failure they simply stay zero/false.
	PartIndex       int  `json:"part_index"`       // 0 = none; best-effort "(N/M)" or "N/M"
	PartTotal       int  `json:"part_total"`       // 0 = none
	IsContinuation  bool `json:"is_continuation"`  // opens with ">" / `">` (marker often absent)
	HasContinuation bool `json:"has_continuation"` // ends with ">"
}
