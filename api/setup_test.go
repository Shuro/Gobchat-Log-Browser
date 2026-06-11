package api

import (
	"testing"

	"gobchat-log-browser/internal/config"
)

// The wizard must show on first run, when no usable log directory exists, and
// exactly once more for users whose config predates the current wizard version
// — but never for users who completed the current wizard.
func TestNeedsSetup(t *testing.T) {
	current := config.SetupWizardCurrentVersion
	tests := []struct {
		name                                          string
		configExists, defaultDirExists, anyConfigured bool
		savedWizardVersion                            int
		want                                          bool
	}{
		{"first run, no config", false, true, false, 0, true},
		{"config but no usable log dir", true, false, false, current, true},
		{"detected default dir is enough", true, true, false, current, false},
		{"configured dir is enough", true, false, true, current, false},
		{"pre-versioning config re-shows once", true, true, true, 0, true},
		{"older wizard version re-shows", true, true, true, current - 1, true},
		{"current wizard version stays hidden", true, true, true, current, false},
	}
	for _, tt := range tests {
		if got := needsSetup(tt.configExists, tt.defaultDirExists, tt.anyConfigured, tt.savedWizardVersion); got != tt.want {
			t.Errorf("%s: needsSetup(%v, %v, %v, %d) = %v; want %v",
				tt.name, tt.configExists, tt.defaultDirExists, tt.anyConfigured, tt.savedWizardVersion, got, tt.want)
		}
	}
}
