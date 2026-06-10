package search

import (
	"sync"
	"testing"

	"gobchat-log-browser/internal/parser"
)

func entries() []parser.LogEntry {
	return []parser.LogEntry{
		{LineNumber: 1, Message: "The copper fox stole the artifact."},
		{LineNumber: 2, Message: "We found a document by Kupferfuchs."},
		{LineNumber: 3, Message: "The artifact is dangerous."},
	}
}

func TestIndexSingleTerm(t *testing.T) {
	idx := NewIndex()
	idx.AddEntries("a.log", entries())
	if !idx.HasFile("a.log") {
		t.Fatalf("HasFile false after AddEntries")
	}
	res := idx.Query("artifact", "", 0)
	if len(res) != 2 {
		t.Fatalf("artifact matches = %d, want 2", len(res))
	}
	if res[0].LineNumber != 1 || res[1].LineNumber != 3 {
		t.Fatalf("results not sorted by line: %+v", res)
	}
}

func TestIndexAndSemantics(t *testing.T) {
	idx := NewIndex()
	idx.AddEntries("a.log", entries())
	// Both "the" and "artifact" only co-occur on lines 1 and 3.
	res := idx.Query("the artifact", "", 0)
	if len(res) != 2 {
		t.Fatalf("AND matches = %d, want 2 (%+v)", len(res), res)
	}
	// A term present in no entry yields no results (AND fails).
	if r := idx.Query("artifact missingword", "", 0); len(r) != 0 {
		t.Fatalf("expected 0 for unsatisfiable AND, got %+v", r)
	}
}

func TestIndexFileRestrictionAndRemove(t *testing.T) {
	idx := NewIndex()
	idx.AddEntries("a.log", entries())
	idx.AddEntries("b.log", []parser.LogEntry{{LineNumber: 1, Message: "artifact elsewhere"}})

	if all := idx.Query("artifact", "", 0); len(all) != 3 {
		t.Fatalf("global artifact = %d, want 3", len(all))
	}
	if only := idx.Query("artifact", "b.log", 0); len(only) != 1 || only[0].FilePath != "b.log" {
		t.Fatalf("file-restricted query wrong: %+v", only)
	}

	idx.RemoveFile("a.log")
	if idx.HasFile("a.log") {
		t.Fatalf("HasFile true after RemoveFile")
	}
	if all := idx.Query("artifact", "", 0); len(all) != 1 {
		t.Fatalf("after remove, artifact = %d, want 1", len(all))
	}
}

// Concurrent AddEntries for the same file must not leave duplicate postings.
// A duplicated posting would make a single term count twice, falsely satisfying
// a two-term AND query — so the unsatisfiable query below must stay empty.
func TestIndexConcurrentAddEntriesKeepsANDSemantics(t *testing.T) {
	idx := NewIndex()
	es := []parser.LogEntry{{LineNumber: 1, Message: "alpha only here"}}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idx.AddEntries("a.log", es)
		}()
	}
	wg.Wait()

	if r := idx.Query("alpha missingword", "", 0); len(r) != 0 {
		t.Fatalf("unsatisfiable AND returned %+v — duplicate postings after concurrent AddEntries", r)
	}
	if r := idx.Query("alpha", "", 0); len(r) != 1 {
		t.Fatalf("alpha = %d, want exactly 1", len(r))
	}
}
