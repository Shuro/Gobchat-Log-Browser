package api

import "gobchat-log-browser/internal/highlight"

// The DTO types below are the wire contract exposed to the frontend. Wails
// serialises them to JSON and generates matching TypeScript. Times are ISO 8601
// strings and durations are pre-formatted for display.

// LogSummary is the overview-row representation of one log file.
type LogSummary struct {
	FilePath     string   `json:"file_path"`
	FileName     string   `json:"file_name"`
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

// SetupState tells the frontend whether to show the first-run setup wizard and
// provides the detected Gobchat log directory to prefill it.
type SetupState struct {
	NeedsSetup          bool   `json:"needs_setup"`
	ConfigExists        bool   `json:"config_exists"`
	DefaultLogDir       string `json:"default_log_dir"`
	DefaultLogDirExists bool   `json:"default_log_dir_exists"`
	// WizardVersion is the version the wizard stamps into config on save, so
	// Go stays the single source of the current wizard version.
	WizardVersion int `json:"wizard_version"`
	// Installer seed (docs/adr/0012): the Windows installer may leave a one-shot
	// default for the update-check opt-in; InstallerCheckUpdates is only
	// meaningful when InstallerSeedFound is true.
	InstallerSeedFound    bool `json:"installer_seed_found"`
	InstallerCheckUpdates bool `json:"installer_check_updates"`
}

// UpdateCheckResult is the outcome of a successful update check; network and
// API failures are returned as an error (Promise rejection) instead.
type UpdateCheckResult struct {
	Status         string `json:"status"` // "dev" | "up_to_date" | "update_available"
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseURL     string `json:"release_url"`
}

// SearchResponse wraps the hits of one search; Truncated tells the frontend
// that the list was cut off and more matches exist.
type SearchResponse struct {
	Results   []SearchResultDTO `json:"results"`
	Truncated bool              `json:"truncated"`
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
