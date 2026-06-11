package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	windowsAppDir = "GobchatLogBrowser"
	unixAppDir    = "gobchat-log-browser"
)

// AppDataDir returns the directory where this app stores its own config and
// tags. On Windows: %APPDATA%\GobchatLogBrowser. Elsewhere: the user config dir.
func AppDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			var err error
			if base, err = os.UserConfigDir(); err != nil {
				return "", err
			}
		}
		return filepath.Join(base, windowsAppDir), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, unixAppDir), nil
}

// ConfigFilePath is the absolute path to config.json.
func ConfigFilePath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// TagsFilePath is the absolute path to tags.json.
func TagsFilePath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tags.json"), nil
}

// IndexFilePath is the absolute path to index.json (the persistent log
// metadata cache, see docs/adr/0009).
func IndexFilePath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.json"), nil
}

// InstallerDefaultsFilePath is the absolute path to installer-defaults.json,
// the one-shot seed the Windows installer may leave for the first-run wizard.
func InstallerDefaultsFilePath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "installer-defaults.json"), nil
}

// GobchatDefaultLogDir returns Gobchat's default log output directory for the
// current platform. It is detected at runtime and not stored in config.
func GobchatDefaultLogDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			var err error
			if base, err = os.UserConfigDir(); err != nil {
				return "", err
			}
		}
		return filepath.Join(base, "Gobchat", "log"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "Gobchat", "log"), nil
}
