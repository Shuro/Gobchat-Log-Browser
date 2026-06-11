package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The seed is a one-shot message from the installer to the first wizard run:
// it must be applied at most once and never linger to re-apply a stale
// installer choice on later wizard-version bumps.
func TestConsumeInstallerDefaultsReadsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installer-defaults.json")
	if err := os.WriteFile(path, []byte(`{"check_updates_on_start": true}`), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	def, found := ConsumeInstallerDefaults(path)
	if !found || !def.CheckUpdatesOnStart {
		t.Fatalf("first consume = %+v, %v; want value true, found", def, found)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("seed file must be deleted after consumption")
	}
	if _, found := ConsumeInstallerDefaults(path); found {
		t.Fatalf("second consume must report not found")
	}
}

func TestConsumeInstallerDefaultsMissing(t *testing.T) {
	if _, found := ConsumeInstallerDefaults(filepath.Join(t.TempDir(), "nope.json")); found {
		t.Fatalf("missing seed must report not found")
	}
}

// A corrupt seed must not survive: otherwise it would be re-read and fail on
// every launch.
func TestConsumeInstallerDefaultsMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installer-defaults.json")
	if err := os.WriteFile(path, []byte(`{broken`), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	if _, found := ConsumeInstallerDefaults(path); found {
		t.Fatalf("malformed seed must report not found")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("malformed seed file must still be deleted")
	}
}
