package config

import (
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
