package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.Language = "de"
	cfg.MentionNames = []string{"Alpha", "Beta"}
	cfg.LogDirectories = []string{`C:\logs`}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Language != "de" {
		t.Fatalf("Language = %q, want de", got.Language)
	}
	if len(got.MentionNames) != 2 || got.MentionNames[0] != "Alpha" {
		t.Fatalf("MentionNames = %v", got.MentionNames)
	}
	if len(got.Markers.Speech) == 0 {
		t.Fatalf("markers not persisted/seeded")
	}

	cfg.CheckUpdatesOnStart = true
	cfg.SetupWizardVersion = SetupWizardCurrentVersion
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.CheckUpdatesOnStart || got.SetupWizardVersion != SetupWizardCurrentVersion {
		t.Fatalf("update-check fields not persisted: %+v", got)
	}
}

// Configs written before the update-check feature have neither field; they
// must load as "opted out, wizard never completed at the current version" so
// the wizard re-shows exactly once and no network call happens uninvited.
func TestLoadLegacyConfigDefaultsUpdateFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	legacy := `{"log_directories":["C:\\logs"],"auto_detect_appdata":true,"language":"de"}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.CheckUpdatesOnStart {
		t.Fatalf("legacy config must not opt into update checks")
	}
	if got.SetupWizardVersion != 0 {
		t.Fatalf("SetupWizardVersion = %d, want 0 for legacy config", got.SetupWizardVersion)
	}
	if got.SetupWizardVersion >= SetupWizardCurrentVersion {
		t.Fatalf("legacy config must compare below SetupWizardCurrentVersion")
	}
	// Pre-colors configs must come back with a non-nil map so the frontend can
	// index it without null guards everywhere.
	if got.Colors == nil {
		t.Fatalf("legacy config must backfill an empty Colors map")
	}
}

func TestSaveLoadColorOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.Colors = map[string]map[string]string{"blue": {"speech": "#ff0000"}}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Colors["blue"]["speech"] != "#ff0000" {
		t.Fatalf("Colors not persisted: %+v", got.Colors)
	}
}

// Versioning lets future patches run real migrations instead of relying only on
// zero-value backfill (ADR-0014). A config written before versioning has no
// config_version key and must load as the current version (migrated, then
// stamped) so the upgrade path runs exactly once.
func TestConfigVersionMigratesLegacyAndStamps(t *testing.T) {
	if DefaultConfig().ConfigVersion != CurrentConfigVersion {
		t.Fatalf("DefaultConfig.ConfigVersion = %d, want %d", DefaultConfig().ConfigVersion, CurrentConfigVersion)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	legacy := `{"log_directories":["C:\\logs"],"language":"de"}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ConfigVersion != CurrentConfigVersion {
		t.Fatalf("legacy ConfigVersion = %d, want %d after migration", got.ConfigVersion, CurrentConfigVersion)
	}
	// Migration must not disturb existing values.
	if got.Language != "de" || len(got.LogDirectories) != 1 {
		t.Fatalf("migration altered fields: %+v", got)
	}

	// Saving stamps the current version onto disk so the migration is one-shot.
	if err := Save(path, got); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if want := fmt.Sprintf(`"config_version": %d`, CurrentConfigVersion); !strings.Contains(string(data), want) {
		t.Fatalf("saved config missing stamped version %q: %s", want, data)
	}
}

// The "dark" theme was renamed "blue" in schema v2. A config still carrying the
// old value (and color overrides keyed under it) must migrate both so the user's
// theme and customizations survive the rename (ADR-0014).
func TestConfigVersionRenamesDarkTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	old := `{"config_version":1,"theme":"dark","colors":{"dark":{"speech":"#ff0000"}}}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Theme != "blue" {
		t.Fatalf("Theme = %q, want blue after migration", got.Theme)
	}
	if _, stale := got.Colors["dark"]; stale {
		t.Fatalf("colors still keyed under old \"dark\": %+v", got.Colors)
	}
	if got.Colors["blue"]["speech"] != "#ff0000" {
		t.Fatalf("color override not moved to \"blue\": %+v", got.Colors)
	}
	if got.ConfigVersion != CurrentConfigVersion {
		t.Fatalf("ConfigVersion = %d, want %d", got.ConfigVersion, CurrentConfigVersion)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("Load of missing file should not error: %v", err)
	}
	if !got.AutoDetectAppData || got.Language != "en" {
		t.Fatalf("missing-file load did not return defaults: %+v", got)
	}
	if len(got.Markers.Speech) == 0 {
		t.Fatalf("defaults should include a marker set")
	}
}
