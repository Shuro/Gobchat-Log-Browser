//go:build !windows

package migrate

// CleanLegacyNSISInstall is a no-op off Windows, where no NSIS install exists.
func CleanLegacyNSISInstall() error { return nil }
