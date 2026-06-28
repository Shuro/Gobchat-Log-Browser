package logstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeLog writes content to <dir>/<name> and returns a *LogMeta describing it
// with the given source priority. ModTime/SizeBytes are taken from the file so
// the content-hash cache validates correctly.
func writeLog(t *testing.T, dir, name, content string, priority int) *LogMeta {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return &LogMeta{
		FilePath:       path,
		FileName:       name,
		SizeBytes:      info.Size(),
		ModTime:        info.ModTime(),
		SourcePriority: priority,
	}
}

func TestDedupe(t *testing.T) {
	const name = "chatlog_2026-01-02_20-01.log"
	exDir := filepath.Join(t.TempDir(), "GobchatEx")
	gobDir := filepath.Join(t.TempDir(), "Gobchat")

	t.Run("identical content prefers lower SourcePriority (GobchatEx)", func(t *testing.T) {
		s := New(nil, nil)
		ex := writeLog(t, exDir, name, "same bytes", 0)   // GobchatEx, scanned first
		gob := writeLog(t, gobDir, name, "same bytes", 1) // Gobchat
		// Give the Gobchat copy a newer mtime to prove identity beats "newer".
		gob.ModTime = ex.ModTime.Add(time.Hour)

		got := s.dedupe([]*LogMeta{gob, ex})
		if len(got) != 1 {
			t.Fatalf("dedupe returned %d entries, want 1", len(got))
		}
		if got[0].FilePath != ex.FilePath {
			t.Fatalf("kept %q, want the GobchatEx copy %q", got[0].FilePath, ex.FilePath)
		}
	})

	t.Run("differing content keeps the newer file", func(t *testing.T) {
		s := New(nil, nil)
		ex := writeLog(t, exDir, name, "old short body", 0)
		gob := writeLog(t, gobDir, name, "newer body with more appended text", 1)
		ex.ModTime = time.Date(2026, 1, 2, 20, 1, 0, 0, time.UTC)
		gob.ModTime = ex.ModTime.Add(time.Hour) // Gobchat copy is newer

		got := s.dedupe([]*LogMeta{ex, gob})
		if len(got) != 1 {
			t.Fatalf("dedupe returned %d entries, want 1", len(got))
		}
		if got[0].FilePath != gob.FilePath {
			t.Fatalf("kept %q, want the newer copy %q", got[0].FilePath, gob.FilePath)
		}
	})

	t.Run("unique filenames pass through", func(t *testing.T) {
		s := New(nil, nil)
		a := writeLog(t, exDir, "chatlog_2026-01-01_10-00.log", "a", 0)
		b := writeLog(t, exDir, "chatlog_2026-01-02_11-00.log", "b", 0)

		got := s.dedupe([]*LogMeta{a, b})
		if len(got) != 2 {
			t.Fatalf("dedupe collapsed distinct filenames: got %d, want 2", len(got))
		}
	})
}
