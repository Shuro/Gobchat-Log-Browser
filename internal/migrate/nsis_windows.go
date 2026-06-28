// Package migrate handles one-time transitions from older install layouts.
//
// Velopack installs side-by-side with (and cannot auto-migrate from) the old
// per-user NSIS install, so on first run we detect the legacy install via its
// HKCU uninstall key and run its silent uninstaller. User data lives in
// %APPDATA%\GobchatLogBrowser, separate from the install dir, and is untouched
// (docs/adr/0013).
package migrate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gobchat-log-browser/internal/version"

	"golang.org/x/sys/windows/registry"
)

// legacyUninstallKey is the HKCU uninstall key the old NSIS installer wrote
// (UNINST_KEY_NAME = CompanyName+ProductName = "Shuro"+"Gobchat Log Browser").
const legacyUninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ShuroGobchat Log Browser`

// CleanLegacyNSISInstall removes a leftover NSIS install. It is safe to call on
// every startup: the legacy uninstall key's presence is the only state it acts
// on, so it no-ops once that key is gone and self-heals if a legacy copy is
// reinstalled. Errors are returned for logging; callers may ignore them.
//
// There is deliberately no persistent "done" marker: an earlier design latched
// completion in a sentinel file under the version-shared app-data dir, which a
// dev run (or a launch that preceded the legacy install) could write
// prematurely and thereby disable the migration forever (docs/adr/0013).
func CleanLegacyNSISInstall() error {
	// Dev builds run from a temp/build dir, not the install dir, so isCurrentInstall
	// can't protect them — they must never silently uninstall the user's real,
	// separately-installed copy. Only release builds perform the migration.
	if version.Version == "dev" {
		return nil
	}

	installLoc, quietUninstall, found := readLegacyUninstall()
	if !found || quietUninstall == "" {
		return nil // no legacy install present — nothing to do
	}
	if isCurrentInstall(installLoc) {
		return nil // never uninstall ourselves
	}

	if err := runLegacyUninstaller(quietUninstall, installLoc); err != nil {
		return fmt.Errorf("run legacy uninstaller: %w", err) // retry next launch
	}

	// The NSIS uninstaller deletes its own HKCU key as it runs; its absence is
	// the signal that removal completed.
	if _, _, stillPresent := readLegacyUninstall(); stillPresent {
		return fmt.Errorf("legacy uninstall did not complete; will retry next launch")
	}
	return nil
}

// runLegacyUninstaller invokes the NSIS uninstaller synchronously and removes
// the leftovers it cannot delete itself.
//
// The exe is launched directly (no `cmd /C`, whose quote handling mangles the
// already-quoted QuietUninstallString). The `_?=<dir>` flag makes NSIS run
// in-place rather than its default copy-to-temp-and-detach, so cmd.Run() truly
// waits for the uninstall to finish — but in-place mode then cannot delete the
// still-running uninstaller or its (now empty) directory, so we do.
func runLegacyUninstaller(quietUninstall, installLoc string) error {
	exe := uninstallerExe(quietUninstall)
	if exe == "" {
		return fmt.Errorf("no uninstaller path in %q", quietUninstall)
	}
	cmd := exec.Command(exe, "/S", "_?="+installLoc)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uninstaller %q failed: %w", exe, err)
	}
	_ = os.Remove(exe)
	_ = os.Remove(installLoc)
	return nil
}

// uninstallerExe extracts the executable path from a QuietUninstallString such
// as `"C:\…\uninstall.exe" /S`, dropping the surrounding quotes and arguments.
func uninstallerExe(quietUninstall string) string {
	s := strings.TrimSpace(quietUninstall)
	if s == "" {
		return ""
	}
	if s[0] == '"' {
		if end := strings.IndexByte(s[1:], '"'); end >= 0 {
			return s[1 : 1+end]
		}
		return "" // unterminated quote — refuse rather than guess
	}
	if sp := strings.IndexByte(s, ' '); sp >= 0 {
		return s[:sp]
	}
	return s
}

// readLegacyUninstall reads InstallLocation and QuietUninstallString from the
// legacy HKCU uninstall key. found is false when the key is absent.
func readLegacyUninstall() (installLoc, quietUninstall string, found bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, legacyUninstallKey, registry.QUERY_VALUE)
	if err != nil {
		return "", "", false
	}
	defer k.Close()
	installLoc, _, _ = k.GetStringValue("InstallLocation")
	quietUninstall, _, _ = k.GetStringValue("QuietUninstallString")
	return installLoc, quietUninstall, true
}

// isCurrentInstall guards against uninstalling the running app: true when the
// current executable lives under installLoc.
func isCurrentInstall(installLoc string) bool {
	if installLoc == "" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe = strings.ToLower(filepath.Clean(exe))
	loc := strings.ToLower(filepath.Clean(installLoc))
	return strings.HasPrefix(exe, loc)
}
