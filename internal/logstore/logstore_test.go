package logstore

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gobchat-log-browser/internal/search"
)

func TestScanDirectoryRecursive(t *testing.T) {
	metas, watchDirs, err := ScanDirectory("testdata", nil)
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	// One log at the top level + one in the Nevio/ subfolder = 2.
	if len(metas) != 2 {
		t.Fatalf("metas = %d, want 2 (recursive scan)", len(metas))
	}

	byFolder := map[string]*LogMeta{}
	for _, m := range metas {
		byFolder[m.Folder] = m
	}
	top, ok := byFolder[""]
	if !ok {
		t.Fatalf("expected a top-level log (folder \"\"); folders seen: %v", folders(metas))
	}
	if top.MessageCount != 4 {
		t.Fatalf("top-level MessageCount = %d, want 4", top.MessageCount)
	}
	nested, ok := byFolder["Nevio"]
	if !ok {
		t.Fatalf("expected a log in folder \"Nevio\"; folders seen: %v", folders(metas))
	}
	if len(nested.Participants) != 1 || nested.Participants[0] != "Nevio Ateius" {
		t.Fatalf("nested participants = %v", nested.Participants)
	}

	// The scan reports directories to watch, including the subfolder.
	if !contains(watchDirs, filepath.Join("testdata", "Nevio")) {
		t.Fatalf("watchDirs missing the Nevio subfolder: %v", watchDirs)
	}
}

func TestStoreGetEntriesIndexes(t *testing.T) {
	idx := search.NewIndex()
	s := New(idx, nil)
	if err := s.ScanAll([]string{"testdata"}); err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if got := s.List(); len(got) != 2 {
		t.Fatalf("List = %d, want 2", len(got))
	}
	if len(s.WatchDirs()) == 0 {
		t.Fatalf("WatchDirs empty after ScanAll")
	}

	path := filepath.Join("testdata", "chatlog_2026-01-02_20-01.log")
	entries, err := s.GetEntries(path)
	if err != nil {
		t.Fatalf("GetEntries: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(entries))
	}
	if !idx.HasFile(path) {
		t.Fatalf("file not indexed after GetEntries")
	}
	again, _ := s.GetEntries(path)
	if &again[0] != &entries[0] {
		t.Fatalf("expected cached entries on second GetEntries")
	}
	if res := idx.Query("evening", path, 0); len(res) != 1 {
		t.Fatalf("query 'evening' = %d, want 1", len(res))
	}
}

// Concurrent first access to the same uncached file (UI call racing a watcher
// Refresh) must parse and index it exactly once; double-indexing would leave
// duplicate postings that break the index's AND query semantics.
func TestStoreConcurrentGetEntriesIndexesOnce(t *testing.T) {
	idx := search.NewIndex()
	s := New(idx, nil)
	if err := s.ScanAll([]string{"testdata"}); err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	path := filepath.Join("testdata", "chatlog_2026-01-02_20-01.log")

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.GetEntries(path); err != nil {
				t.Errorf("GetEntries: %v", err)
			}
		}()
	}
	wg.Wait()

	// "evening" occurs on exactly one line; an AND query with a word that occurs
	// nowhere must stay empty (a duplicated posting would falsely satisfy it).
	if res := idx.Query("evening", path, 0); len(res) != 1 {
		t.Fatalf("query 'evening' = %d, want 1", len(res))
	}
	if res := idx.Query("evening zzzmissing", path, 0); len(res) != 0 {
		t.Fatalf("unsatisfiable AND returned %d results — file was double-indexed", len(res))
	}
}

// A watcher Refresh of a duplicate must keep the SourcePriority that ScanAll
// stamped from the scan-root order; ExtractMeta does not know it, and resetting
// it to 0 would flip identical-content dedup ties away from the preferred
// source (ADR-0015).
func TestRefreshPreservesSourcePriority(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "chatlog_2026-01-02_20-01.log"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const name = "chatlog_2026-01-02_20-01.log"
	exDir := filepath.Join(t.TempDir(), "GobchatEx")
	gobDir := filepath.Join(t.TempDir(), "Gobchat")
	for _, d := range []string{exDir, gobDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
		if err := os.WriteFile(filepath.Join(d, name), content, 0o644); err != nil {
			t.Fatalf("write %s: %v", d, err)
		}
	}

	s := New(nil, nil)
	// exDir scanned first → priority 0; gobDir → priority 1.
	if err := s.ScanAll([]string{exDir, gobDir}); err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	gobPath := filepath.Join(gobDir, name)
	if m, ok := s.Get(gobPath); !ok || m.SourcePriority != 1 {
		t.Fatalf("after ScanAll gob SourcePriority = %d (ok=%v), want 1", m.SourcePriority, ok)
	}

	if err := s.Refresh(gobPath); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if m, ok := s.Get(gobPath); !ok || m.SourcePriority != 1 {
		t.Fatalf("after Refresh gob SourcePriority = %d (ok=%v), want 1 preserved", m.SourcePriority, ok)
	}
}

func folders(metas []*LogMeta) []string {
	out := make([]string, len(metas))
	for i, m := range metas {
		out[i] = m.Folder
	}
	return out
}

func contains(items []string, want string) bool {
	for _, it := range items {
		if it == want {
			return true
		}
	}
	return false
}
