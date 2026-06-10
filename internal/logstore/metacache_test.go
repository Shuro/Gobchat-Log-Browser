package logstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// copyTestLog copies the top-level testdata log into dir and returns its path
// and stat info. Tests that touch mtimes work on a copy so testdata stays
// pristine.
func copyTestLog(t *testing.T, dir string) (string, os.FileInfo) {
	t.Helper()
	src := filepath.Join("testdata", "chatlog_2026-01-02_20-01.log")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	dst := filepath.Join(dir, "chatlog_2026-01-02_20-01.log")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write copy: %v", err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat copy: %v", err)
	}
	return dst, info
}

// A cache hit must return the metadata that was stored — including the
// Participants payload the player filter is built on — across a save/load
// cycle, because losing it silently would make the overview wrong without any
// error surfacing.
func TestMetaCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	savePath := filepath.Join(dir, "index.json")
	mtime := time.Now()

	c := LoadMetaCache(savePath)
	c.Put(&LogMeta{
		FilePath:     "/logs/a.log",
		FileName:     "a.log",
		MessageCount: 7,
		Participants: []string{"Nevio Ateius", "Zara Voss"},
		SizeBytes:    123,
	}, mtime)
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := LoadMetaCache(savePath)
	m, ok := loaded.Get("/logs/a.log", mtime, 123)
	if !ok {
		t.Fatalf("expected cache hit after save/load")
	}
	if m.MessageCount != 7 || len(m.Participants) != 2 || m.Participants[0] != "Nevio Ateius" {
		t.Fatalf("loaded meta = %+v, want original metadata back", m)
	}
}

// A changed mtime or size must invalidate the entry: serving stale metadata
// would hide new participants/messages from the overview with no way for the
// user to notice.
func TestMetaCacheInvalidation(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "index.json")
	mtime := time.Now()
	c := LoadMetaCache(savePath)
	c.Put(&LogMeta{FilePath: "/logs/a.log", SizeBytes: 123}, mtime)

	if _, ok := c.Get("/logs/a.log", mtime.Add(time.Second), 123); ok {
		t.Fatalf("expected miss on changed mtime")
	}
	if _, ok := c.Get("/logs/a.log", mtime, 999); ok {
		t.Fatalf("expected miss on changed size")
	}
	if _, ok := c.Get("/logs/other.log", mtime, 123); ok {
		t.Fatalf("expected miss on unknown path")
	}
	if _, ok := c.Get("/logs/a.log", mtime, 123); !ok {
		t.Fatalf("expected hit on unchanged mtime+size")
	}
}

// The cache is derived data, so a corrupt or future-versioned index.json must
// degrade to an empty cache (full rescan) instead of failing startup, and the
// next Save must recover the file.
func TestMetaCacheTolerantLoad(t *testing.T) {
	dir := t.TempDir()

	for name, content := range map[string]string{
		"garbage.json": "{not json",
		"version.json": `{"version": 99, "files": {}}`,
	} {
		savePath := filepath.Join(dir, name)
		if err := os.WriteFile(savePath, []byte(content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
		c := LoadMetaCache(savePath)
		mtime := time.Now()
		c.Put(&LogMeta{FilePath: "/logs/a.log", SizeBytes: 1}, mtime)
		if err := c.Save(); err != nil {
			t.Fatalf("Save over %s: %v", name, err)
		}
		if _, ok := LoadMetaCache(savePath).Get("/logs/a.log", mtime, 1); !ok {
			t.Fatalf("cache did not recover after overwriting %s", name)
		}
	}
}

// The whole point of the cache is skipping the parse for unchanged files.
// Observable proof: a seeded sentinel meta survives a scan untouched, and only
// after the file's mtime changes does the scan re-parse and return real data.
func TestScanDirectoryUsesCache(t *testing.T) {
	dir := t.TempDir()
	logPath, info := copyTestLog(t, dir)

	cache := LoadMetaCache(filepath.Join(dir, "index.json"))
	cache.Put(&LogMeta{
		FilePath:     logPath,
		FileName:     filepath.Base(logPath),
		MessageCount: 999, // sentinel: only survives if the parse is skipped
		SizeBytes:    info.Size(),
	}, info.ModTime())

	metas, _, err := ScanDirectory(dir, cache)
	if err != nil || len(metas) != 1 {
		t.Fatalf("ScanDirectory: metas=%d err=%v", len(metas), err)
	}
	if metas[0].MessageCount != 999 {
		t.Fatalf("MessageCount = %d, want sentinel 999 (file was re-parsed despite cache hit)", metas[0].MessageCount)
	}

	// Touch the file: the sentinel must be discarded and the real parse return.
	if err := os.Chtimes(logPath, time.Now(), info.ModTime().Add(2*time.Second)); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	metas, _, err = ScanDirectory(dir, cache)
	if err != nil || len(metas) != 1 {
		t.Fatalf("ScanDirectory after touch: metas=%d err=%v", len(metas), err)
	}
	if metas[0].MessageCount != 4 {
		t.Fatalf("MessageCount = %d, want 4 (stale cache served after mtime change)", metas[0].MessageCount)
	}
}

// ScanAll must prune cache entries for files that no longer exist and persist
// the result, so deleted logs do not accumulate in index.json forever.
func TestScanAllPrunesAndSavesCache(t *testing.T) {
	dir := t.TempDir()
	logPath, _ := copyTestLog(t, dir)
	savePath := filepath.Join(t.TempDir(), "index.json")

	cache := LoadMetaCache(savePath)
	staleMtime := time.Now()
	cache.Put(&LogMeta{FilePath: filepath.Join(dir, "deleted.log"), SizeBytes: 5}, staleMtime)

	s := New(nil, cache)
	if err := s.ScanAll([]string{dir}); err != nil {
		t.Fatalf("ScanAll: %v", err)
	}

	loaded := LoadMetaCache(savePath)
	if _, ok := loaded.Get(filepath.Join(dir, "deleted.log"), staleMtime, 5); ok {
		t.Fatalf("deleted file survived pruning in the saved cache")
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if _, ok := loaded.Get(logPath, info.ModTime(), info.Size()); !ok {
		t.Fatalf("scanned file missing from the saved cache")
	}
}
