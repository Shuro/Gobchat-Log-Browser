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

	"gobchat-log-browser/internal/config"

	"golang.org/x/sys/windows/registry"
)

// legacyUninstallKey is the HKCU uninstall key the old NSIS installer wrote
// (UNINST_KEY_NAME = CompanyName+ProductName = "Shuro"+"Gobchat Log Browser").
const legacyUninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ShuroGobchat Log Browser`

// migratedMarker is the one-shot sentinel so this runs at most once per machine.
const migratedMarker = "nsis-migrated"

// CleanLegacyNSISInstall removes a leftover NSIS install, once. It is safe to
// call on every startup: it no-ops after success and when no legacy install is
// present. Errors are returned for logging; callers may ignore them.
//
// The success marker is written only after the legacy install is confirmed
// gone, so a transient failure (e.g. the uninstaller is briefly blocked)
// retries on the next launch instead of orphaning the old install forever.
func CleanLegacyNSISInstall() error {
	appData, err := config.AppDataDir()
	if err != nil {
		return fmt.Errorf("locate app data dir: %w", err)
	}
	marker := filepath.Join(appData, migratedMarker)
	if _, err := os.Stat(marker); err == nil {
		return nil // already handled
	}

	installLoc, quietUninstall, found := readLegacyUninstall()
	if !found || quietUninstall == "" {
		return writeMarker(marker) // nothing to remove — done for good
	}
	if isCurrentInstall(installLoc) {
		return writeMarker(marker) // never uninstall ourselves
	}

	if err := runLegacyUninstaller(quietUninstall, installLoc); err != nil {
		return fmt.Errorf("run legacy uninstaller: %w", err) // no marker → retry
	}

	// The NSIS uninstaller deletes its own HKCU key as it runs; its absence is
	// the signal that removal completed. Only then is the migration done.
	if _, _, stillPresent := readLegacyUninstall(); stillPresent {
		return fmt.Errorf("legacy uninstall did not complete; will retry next launch")
	}
	return writeMarker(marker)
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

// writeMarker records that migration completed so it never runs again.
func writeMarker(marker string) error {
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}
	if err := os.WriteFile(marker, []byte("Velopack migration: legacy NSIS install handled.\n"), 0o644); err != nil {
		return fmt.Errorf("write migration marker: %w", err)
	}
	return nil
}
