package tags

import (
	"path/filepath"
	"testing"
)

func TestTagStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	ts, err := NewTagStore(path)
	if err != nil {
		t.Fatalf("NewTagStore: %v", err)
	}

	if err := ts.SetTags("chatlog_a.log", []string{"arc-1", "tavern", "arc-1"}, "first meeting"); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	if err := ts.SetTags("chatlog_b.log", []string{"arc-2"}, ""); err != nil {
		t.Fatalf("SetTags: %v", err)
	}

	got := ts.GetTags("chatlog_a.log")
	if len(got.Tags) != 2 { // duplicate "arc-1" removed
		t.Fatalf("tags = %v, want 2 deduped", got.Tags)
	}
	if got.Note != "first meeting" {
		t.Fatalf("note = %q", got.Note)
	}

	// Reload from disk to confirm persistence.
	ts2, err := NewTagStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if all := ts2.AllTags(); len(all) != 3 || all[0] != "arc-1" {
		t.Fatalf("AllTags = %v, want sorted [arc-1 arc-2 tavern]", all)
	}
}

func TestSetEmptyRemovesEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	ts, _ := NewTagStore(path)
	_ = ts.SetTags("x.log", []string{"a"}, "n")
	_ = ts.SetTags("x.log", nil, "")
	if got := ts.GetTags("x.log"); len(got.Tags) != 0 || got.Note != "" {
		t.Fatalf("expected entry removed, got %+v", got)
	}
}

func TestGetUnknownReturnsEmpty(t *testing.T) {
	ts, _ := NewTagStore(filepath.Join(t.TempDir(), "tags.json"))
	got := ts.GetTags("nope.log")
	if got.FileName != "nope.log" || got.Tags == nil {
		t.Fatalf("unknown lookup should return empty non-nil: %+v", got)
	}
}
