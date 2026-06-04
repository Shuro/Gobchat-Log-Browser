// Package search provides a lazy, in-memory inverted index over log entries
// (see docs/adr/0004). It maps lowercased word tokens to the (file, line)
// locations they occur in. Channel/sender/snippet enrichment of results is left
// to the caller, which has the parsed entries on hand — keeping this package
// dependency-light.
package search

import (
	"sort"
	"strings"
	"sync"
	"unicode"

	"gobchat-log-browser/internal/parser"
)

// Posting is a single occurrence location: one entry in one file.
type Posting struct {
	FilePath   string
	LineNumber int
}

// SearchResult is one matching entry with a relevance score (number of distinct
// query terms it contains).
type SearchResult struct {
	FilePath   string
	LineNumber int
	Score      int
}

// Index is a concurrency-safe inverted index.
type Index struct {
	mu      sync.RWMutex
	posting map[string][]Posting
	files   map[string]struct{}
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	return &Index{posting: map[string][]Posting{}, files: map[string]struct{}{}}
}

// tokenize splits text into lowercased word tokens on any non-letter/non-digit
// boundary. No stemming or stopwords — roleplay text is full of proper and
// invented nouns that standard stopword lists would wrongly drop.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// HasFile reports whether a file has already been indexed.
func (idx *Index) HasFile(path string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, ok := idx.files[path]
	return ok
}

// AddEntries indexes all entries of a file. It is idempotent per file: a file is
// removed and re-added if indexed again, so re-indexing a changed file is safe.
func (idx *Index) AddEntries(filePath string, entries []parser.LogEntry) {
	idx.RemoveFile(filePath)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, e := range entries {
		seen := map[string]struct{}{}
		for _, tok := range tokenize(e.Message) {
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
			idx.posting[tok] = append(idx.posting[tok], Posting{filePath, e.LineNumber})
		}
	}
	idx.files[filePath] = struct{}{}
}

// RemoveFile drops all postings for a file.
func (idx *Index) RemoveFile(filePath string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if _, ok := idx.files[filePath]; !ok {
		return
	}
	for tok, list := range idx.posting {
		kept := list[:0]
		for _, p := range list {
			if p.FilePath != filePath {
				kept = append(kept, p)
			}
		}
		if len(kept) == 0 {
			delete(idx.posting, tok)
		} else {
			idx.posting[tok] = kept
		}
	}
	delete(idx.files, filePath)
}

// Query returns entries containing ALL distinct query terms (AND semantics). If
// filePath is non-empty the search is restricted to that file. Results are
// sorted deterministically by file then line and capped at maxResults (<=0
// means a default of 200).
func (idx *Index) Query(text, filePath string, maxResults int) []SearchResult {
	terms := uniqueStrings(tokenize(text))
	if len(terms) == 0 {
		return nil
	}
	if maxResults <= 0 {
		maxResults = 200
	}

	idx.mu.RLock()
	counts := map[Posting]int{}
	for _, term := range terms {
		for _, p := range idx.posting[term] {
			if filePath != "" && p.FilePath != filePath {
				continue
			}
			counts[p]++
		}
	}
	idx.mu.RUnlock()

	results := make([]SearchResult, 0, len(counts))
	for p, c := range counts {
		if c >= len(terms) { // all terms present
			results = append(results, SearchResult{p.FilePath, p.LineNumber, c})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].FilePath != results[j].FilePath {
			return results[i].FilePath < results[j].FilePath
		}
		return results[i].LineNumber < results[j].LineNumber
	})
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := in[:0:0]
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
