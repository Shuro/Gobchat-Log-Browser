package migrate

import "testing"

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
