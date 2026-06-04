package logstore

import (
	"path/filepath"
	"testing"

	"gobchat-log-browser/internal/search"
)

func TestScanDirectoryRecursive(t *testing.T) {
	metas, watchDirs, err := ScanDirectory("testdata")
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
	s := New(idx)
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
