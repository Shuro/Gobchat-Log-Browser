package config

import (
	"encoding/json"
	"os"
)

// InstallerDefaults are one-shot defaults the Windows installer seeds for the
// first-run wizard (docs/adr/0012). On other platforms the file never exists.
type InstallerDefaults struct {
	CheckUpdatesOnStart bool `json:"check_updates_on_start"`
}

// ConsumeInstallerDefaults reads the seed file at path and then deletes it
// (read-once). found is false when the file is absent or unreadable; the file
// is removed in either case so a corrupt seed cannot re-apply forever.
func ConsumeInstallerDefaults(path string) (def InstallerDefaults, found bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return InstallerDefaults{}, false
	}
	defer os.Remove(path)
	if err := json.Unmarshal(data, &def); err != nil {
		return InstallerDefaults{}, false
	}
	return def, true
}
