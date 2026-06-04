package logstore

import (
	"path/filepath"
	"testing"

	"gobchat-log-browser/internal/search"
)

func TestScanDirectory(t *testing.T) {
	metas, err := ScanDirectory("testdata")
	if err != nil {
		t.Fatalf("ScanDirectory: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas = %d, want 1", len(metas))
	}
	m := metas[0]
	if m.MessageCount != 4 {
		t.Fatalf("MessageCount = %d, want 4", m.MessageCount)
	}
	// GobchatInfo's "Gobchat" sender is excluded; only RP participants remain.
	if len(m.Participants) != 2 || m.Participants[0] != "Alpha Tester" || m.Participants[1] != "Beta User" {
		t.Fatalf("Participants = %v, want [Alpha Tester Beta User]", m.Participants)
	}
	if y := m.LogDate.Year(); y != 2026 {
		t.Fatalf("LogDate year = %d, want 2026 (from filename)", y)
	}
	if m.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", m.Duration)
	}
}

func TestStoreGetEntriesIndexes(t *testing.T) {
	idx := search.NewIndex()
	s := New(idx)
	if err := s.ScanAll([]string{"testdata"}); err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if got := s.List(); len(got) != 1 {
		t.Fatalf("List = %d, want 1", len(got))
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

	// A second call returns the cached slice (same backing array).
	again, _ := s.GetEntries(path)
	if &again[0] != &entries[0] {
		t.Fatalf("expected cached entries on second GetEntries")
	}

	// Sanity: the index can find a word from the fixture.
	if res := idx.Query("evening", path, 0); len(res) != 1 {
		t.Fatalf("query 'evening' = %d, want 1", len(res))
	}
}
