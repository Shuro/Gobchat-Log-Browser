package config

import (
	"os"
	"path/filepath"
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
