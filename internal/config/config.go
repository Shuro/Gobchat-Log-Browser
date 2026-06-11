// Package config holds the application's persistent settings and the
// platform-aware locations they live in. Settings are stored as JSON and
// written atomically.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gobchat-log-browser/internal/highlight"
)

// Config is the user-facing application configuration.
type Config struct {
	LogDirectories      []string            `json:"log_directories"`
	AutoDetectAppData   bool                `json:"auto_detect_appdata"`
	Language            string              `json:"language"` // "en" | "de"
	MentionNames        []string            `json:"mention_names"`
	RoleplayCharacters  []string            `json:"roleplay_characters"` // pinned in the player filter
	Markers             highlight.MarkerSet `json:"markers"`             // configurable RP delimiters
	Theme               string              `json:"theme"`               // "light" | "dark"
	ChannelFilters      map[string]bool     `json:"channel_filters"`
	CheckUpdatesOnStart bool                `json:"check_updates_on_start"` // opt-in update check (docs/adr/0012)
	SetupWizardVersion  int                 `json:"setup_wizard_version"`   // last completed wizard version; 0 = never/pre-versioning
}

// SetupWizardCurrentVersion is bumped whenever the setup wizard gains content
// existing users should see; a config with an older (or missing, i.e. 0) value
// makes the wizard show once more, pre-filled. History:
//
//	1 = original wizard (language, theme, log dir, RP character); configs from
//	    that era have no setup_wizard_version field and load as 0
//	2 = update-check opt-in added
const SetupWizardCurrentVersion = 2

// DefaultConfig returns the baseline configuration, seeding the RP marker set
// with the Gobchat defaults.
func DefaultConfig() Config {
	return Config{
		LogDirectories:      []string{},
		AutoDetectAppData:   true,
		Language:            "en",
		MentionNames:        []string{},
		RoleplayCharacters:  []string{},
		Markers:             highlight.DefaultMarkerSet(),
		Theme:               "dark",
		ChannelFilters:      map[string]bool{},
		CheckUpdatesOnStart: false, // never phone home without consent
		SetupWizardVersion:  0,
	}
}

// Load reads the config from path. A missing file is not an error: defaults are
// returned. Empty marker sets in an older file are backfilled with defaults so
// highlighting keeps working.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	if len(cfg.Markers.Speech) == 0 && len(cfg.Markers.Emote) == 0 && len(cfg.Markers.OOC) == 0 {
		cfg.Markers = highlight.DefaultMarkerSet()
	}
	if cfg.ChannelFilters == nil {
		cfg.ChannelFilters = map[string]bool{}
	}
	return cfg, nil
}

// Save writes the config to path using an atomic temp-file-and-rename so a
// crash mid-write cannot corrupt the existing file.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
