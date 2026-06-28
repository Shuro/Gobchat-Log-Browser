package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

// GobchatEx keeps Gobchat's "log" subfolder name under its own app-data folder
// (ADR-0015); the dedup logic relies on matching filenames across the two dirs.
func TestGobchatExDefaultLogDir(t *testing.T) {
	got, err := GobchatExDefaultLogDir()
	if err != nil {
		t.Fatalf("GobchatExDefaultLogDir: %v", err)
	}
	wantTail := filepath.Join("GobchatEx", "log")
	if filepath.Base(filepath.Dir(got)) != "GobchatEx" || filepath.Base(got) != "log" {
		t.Fatalf("GobchatExDefaultLogDir = %q, want it to end in %q", got, wantTail)
	}
	if runtime.GOOS == "windows" && !filepath.IsAbs(got) {
		t.Fatalf("expected an absolute path on Windows, got %q", got)
	}
}
