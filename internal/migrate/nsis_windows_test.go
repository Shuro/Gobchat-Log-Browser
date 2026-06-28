package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

// restoreShortcut exists because the NSIS uninstaller deletes the Start Menu
// shortcut whose name Velopack reuses; we snapshot Velopack's copy and put it
// back. The intent that matters: restore it only when it is actually gone, and
// never overwrite a shortcut that is already there (which could be a newer one).
func TestRestoreShortcut(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Gobchat Log Browser.lnk")
	const saved = "velopack-shortcut-bytes"

	// Gone after the uninstall → the saved copy is restored.
	restoreShortcut(path, []byte(saved))
	if got, _ := os.ReadFile(path); string(got) != saved {
		t.Fatalf("missing shortcut not restored: got %q, want %q", got, saved)
	}

	// Already present (e.g. a newer shortcut) → left untouched.
	if err := os.WriteFile(path, []byte("newer-shortcut"), 0o644); err != nil {
		t.Fatal(err)
	}
	restoreShortcut(path, []byte(saved))
	if got, _ := os.ReadFile(path); string(got) != "newer-shortcut" {
		t.Fatalf("present shortcut was clobbered: got %q", got)
	}

	// Nothing was saved (no Velopack shortcut existed) → nothing is created.
	absent := filepath.Join(dir, "absent.lnk")
	restoreShortcut(absent, nil)
	if _, err := os.Stat(absent); !os.IsNotExist(err) {
		t.Fatalf("restoreShortcut created a file from nil data")
	}
}

// uninstallerExe must pull a clean path out of the registry's
// QuietUninstallString. This matters because the previous implementation handed
// the whole string to `cmd /C`, whose quote handling could mangle the quoted
// path so the uninstaller never launched — the bug that left a legacy install
// behind. Parsing the path ourselves and launching it directly avoids cmd.
func TestUninstallerExe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "quoted path with silent flag (the real NSIS form)",
			input: `"C:\Users\Shuro\AppData\Local\GobchatLogBrowser\uninstall.exe" /S`,
			want:  `C:\Users\Shuro\AppData\Local\GobchatLogBrowser\uninstall.exe`,
		},
		{
			name:  "quoted path, spaces inside the path are preserved",
			input: `"C:\Program Files\App\uninstall.exe" /S`,
			want:  `C:\Program Files\App\uninstall.exe`,
		},
		{
			name:  "unquoted path stops at the first argument",
			input: `C:\App\uninstall.exe /S`,
			want:  `C:\App\uninstall.exe`,
		},
		{
			name:  "bare path, no arguments",
			input: `C:\App\uninstall.exe`,
			want:  `C:\App\uninstall.exe`,
		},
		{
			name:  "surrounding whitespace is trimmed",
			input: `  "C:\App\uninstall.exe" /S  `,
			want:  `C:\App\uninstall.exe`,
		},
		{
			name:  "empty string yields no path",
			input: ``,
			want:  ``,
		},
		{
			name:  "unterminated quote is refused rather than guessed",
			input: `"C:\App\uninstall.exe /S`,
			want:  ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uninstallerExe(tt.input); got != tt.want {
				t.Errorf("uninstallerExe(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
