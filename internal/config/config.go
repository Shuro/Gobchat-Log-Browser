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
	ConfigVersion       int                 `json:"config_version"` // schema version; 0 = pre-versioning (legacy)
	LogDirectories      []string            `json:"log_directories"`
	AutoDetectAppData   bool                `json:"auto_detect_appdata"`
	Language            string              `json:"language"` // "en" | "de"
	MentionNames        []string            `json:"mention_names"`
	RoleplayCharacters  []string            `json:"roleplay_characters"` // pinned in the player filter
	Markers             highlight.MarkerSet `json:"markers"`             // configurable RP delimiters
	Theme               string              `json:"theme"`               // "light" | "blue" | "dark-gobchat-ex"
	ChannelFilters      map[string]bool     `json:"channel_filters"`
	CheckUpdatesOnStart bool                `json:"check_updates_on_start"` // opt-in update check (docs/adr/0012)
	SetupWizardVersion  int                 `json:"setup_wizard_version"`   // last completed wizard version; 0 = never/pre-versioning
	// Colors holds per-theme highlight color overrides: theme ("blue"|"light")
	// → category ("speech"|"emote"|"ooc"|"mention-fg"|"mention-bg") → hex
	// color. A missing entry means the theme's default color.
	Colors map[string]map[string]string `json:"colors"`
}

// CurrentConfigVersion is the config schema version this build writes. A loaded
// config with a lower (or missing, i.e. 0) value is run through the migration
// steps below and re-stamped on the next Save. History:
//
//	1 = baseline (the schema in effect when versioning was introduced; ADR-0014)
//	2 = the "dark" theme was renamed "blue" (theme value + colors override key)
const CurrentConfigVersion = 2

// configMigration transforms an in-memory Config from the version it indexes
// (its slice position) up to the next. migrations[0] migrates a version-0
// (legacy/unversioned) config to version 1, and so on. They run in order for
// every step the stored config is behind, then ConfigVersion is stamped to
// CurrentConfigVersion. v0→v1 is a no-op: the baseline schema is unchanged, so
// zero-value backfill (handled in Load) already produces a valid v1 config.
//
// A future migration that renames or removes a JSON field cannot see the old
// key here (json.Unmarshal has already dropped it). Such a step must instead
// decode the raw bytes into a map; add that seam only when first needed.
var configMigrations = []configMigration{
	func(*Config) {}, // v0 → v1: no-op
	func(cfg *Config) { // v1 → v2: the "dark" theme was renamed "blue"
		if cfg.Theme == "dark" {
			cfg.Theme = "blue"
		}
		// Move any per-theme color overrides stored under the old key. The guard
		// on the new key avoids clobbering "blue" overrides on the rare config
		// that somehow has both.
		if c, ok := cfg.Colors["dark"]; ok {
			if _, exists := cfg.Colors["blue"]; !exists {
				cfg.Colors["blue"] = c
			}
			delete(cfg.Colors, "dark")
		}
	},
}

type configMigration func(*Config)

// runConfigMigrations applies every migration step the config is behind and
// stamps it to the current version. A config already at or above the current
// version is only re-stamped (never downgraded).
func runConfigMigrations(cfg *Config) {
	for v := cfg.ConfigVersion; v < CurrentConfigVersion && v < len(configMigrations); v++ {
		configMigrations[v](cfg)
	}
	cfg.ConfigVersion = CurrentConfigVersion
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
		ConfigVersion:       CurrentConfigVersion,
		LogDirectories:      []string{},
		AutoDetectAppData:   true,
		Language:            "en",
		MentionNames:        []string{},
		RoleplayCharacters:  []string{},
		Markers:             highlight.DefaultMarkerSet(),
		Theme:               "blue",
		ChannelFilters:      map[string]bool{},
		CheckUpdatesOnStart: false, // never phone home without consent
		SetupWizardVersion:  0,
		Colors:              map[string]map[string]string{},
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
	// A file written before versioning has no config_version key; decode it as 0
	// (rather than inheriting the default's current version) so legacy configs
	// are correctly migrated below.
	cfg.ConfigVersion = 0
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	if len(cfg.Markers.Speech) == 0 && len(cfg.Markers.Emote) == 0 && len(cfg.Markers.OOC) == 0 {
		cfg.Markers = highlight.DefaultMarkerSet()
	}
	if cfg.ChannelFilters == nil {
		cfg.ChannelFilters = map[string]bool{}
	}
	if cfg.Colors == nil {
		cfg.Colors = map[string]map[string]string{}
	}
	runConfigMigrations(&cfg)
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
