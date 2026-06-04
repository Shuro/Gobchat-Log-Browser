package api

import "gobchat-log-browser/internal/highlight"

// The DTO types below are the wire contract exposed to the frontend. Wails
// serialises them to JSON and generates matching TypeScript. Times are ISO 8601
// strings and durations are pre-formatted for display.

// LogSummary is the overview-row representation of one log file.
type LogSummary struct {
	FilePath     string   `json:"file_path"`
	FileName     string   `json:"file_name"`
	Folder       string   `json:"folder"`
	LogDate      string   `json:"log_date"`
	MessageCount int      `json:"message_count"`
	Participants []string `json:"participants"`
	Channels     []string `json:"channels"`
	Duration     string   `json:"duration"`
	Tags         []string `json:"tags"`
	Note         string   `json:"note"`
}

// EntryDTO is one log line with pre-computed highlight spans.
type EntryDTO struct {
	LineNumber      int              `json:"line_number"`
	Channel         string           `json:"channel"`
	Timestamp       string           `json:"timestamp"`
	Sender          string           `json:"sender"`
	DisplayName     string           `json:"display_name"`
	Realm           string           `json:"realm"`
	StatusSymbol    string           `json:"status_symbol"`
	Message         string           `json:"message"`
	Spans           []highlight.Span `json:"spans"`
	PartIndex       int              `json:"part_index"`
	PartTotal       int              `json:"part_total"`
	IsContinuation  bool             `json:"is_continuation"`
	HasContinuation bool             `json:"has_continuation"`
}

// ThreadDTO is a reassembled (in-memory) thread for the optional combined view.
type ThreadDTO struct {
	Sender    string           `json:"sender"`
	Channel   string           `json:"channel"`
	Lines     []int            `json:"lines"`
	Combined  string           `json:"combined"`
	Spans     []highlight.Span `json:"spans"`
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
}

// SearchResultDTO is one search hit, enriched with entry context.
type SearchResultDTO struct {
	FilePath   string `json:"file_path"`
	FileName   string `json:"file_name"`
	LineNumber int    `json:"line_number"`
	Channel    string `json:"channel"`
	Sender     string `json:"sender"`
	Snippet    string `json:"snippet"`
	Score      int    `json:"score"`
}
