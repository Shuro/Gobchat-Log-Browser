// Package logstore is the central registry of known log files. It scans
// configured directories for lightweight metadata, parses individual logs
// lazily (caching the result), and feeds parsed entries into the search index.
// All file access is read-only.
package logstore

import (
	"sort"
	"sync"

	"gobchat-log-browser/internal/parser"
	"gobchat-log-browser/internal/search"
)

// LogStore holds metadata for all known logs and a lazy cache of fully parsed
// logs. It is safe for concurrent use.
type LogStore struct {
	mu        sync.RWMutex
	metas     map[string]*LogMeta          // keyed by file path
	cache     map[string]*parser.ParsedLog // lazily filled on first GetEntries
	watchDirs []string                     // directories (incl. subfolders) to watch
	index     *search.Index
}

// New creates a LogStore that indexes parsed entries into idx (idx may be nil to
// disable indexing).
func New(idx *search.Index) *LogStore {
	return &LogStore{
		metas: map[string]*LogMeta{},
		cache: map[string]*parser.ParsedLog{},
		index: idx,
	}
}

// ScanAll rescans the given root directories recursively, replacing the current
// metadata set and recording the directories to watch. It does not parse-cache
// or index entries; that happens lazily per file.
func (s *LogStore) ScanAll(dirs []string) error {
	fresh := map[string]*LogMeta{}
	watch := []string{}
	seenWatch := map[string]struct{}{}
	for _, dir := range dirs {
		metas, wdirs, err := ScanDirectory(dir)
		if err != nil {
			return err
		}
		for _, m := range metas {
			fresh[m.FilePath] = m
		}
		for _, w := range wdirs {
			if _, ok := seenWatch[w]; !ok {
				seenWatch[w] = struct{}{}
				watch = append(watch, w)
			}
		}
	}
	s.mu.Lock()
	s.metas = fresh
	s.watchDirs = watch
	// Drop cache entries for files that disappeared.
	for path := range s.cache {
		if _, ok := fresh[path]; !ok {
			delete(s.cache, path)
			if s.index != nil {
				s.index.RemoveFile(path)
			}
		}
	}
	s.mu.Unlock()
	return nil
}

// WatchDirs returns the directories (roots plus their non-skipped subfolders)
// that should be watched for log changes. Valid after ScanAll.
func (s *LogStore) WatchDirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.watchDirs))
	copy(out, s.watchDirs)
	return out
}

// List returns all known metadata, newest first.
func (s *LogStore) List() []LogMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LogMeta, 0, len(s.metas))
	for _, m := range s.metas {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LogDate.After(out[j].LogDate) })
	return out
}

// Get returns the metadata for one file path.
func (s *LogStore) Get(path string) (LogMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.metas[path]; ok {
		return *m, true
	}
	return LogMeta{}, false
}

// GetEntries returns the parsed entries for a file, parsing and caching on first
// access and feeding the search index. Subsequent calls return the cached slice.
func (s *LogStore) GetEntries(path string) ([]parser.LogEntry, error) {
	s.mu.RLock()
	cached, ok := s.cache[path]
	s.mu.RUnlock()
	if ok {
		return cached.Entries, nil
	}

	pl, err := parser.Parse(path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[path] = pl
	s.mu.Unlock()
	if s.index != nil {
		s.index.AddEntries(path, pl.Entries)
	}
	return pl.Entries, nil
}

// Refresh re-reads a single file: it updates metadata, invalidates the cache,
// and re-indexes if the file was previously parsed. Used when the watcher
// reports a change. A file that no longer exists is removed.
func (s *LogStore) Refresh(path string) error {
	meta, err := ExtractMeta(path)
	if err != nil {
		// File likely removed/unreadable — drop it.
		s.mu.Lock()
		delete(s.metas, path)
		delete(s.cache, path)
		s.mu.Unlock()
		if s.index != nil {
			s.index.RemoveFile(path)
		}
		return err
	}

	s.mu.Lock()
	s.metas[path] = meta
	_, wasCached := s.cache[path]
	delete(s.cache, path)
	s.mu.Unlock()

	if wasCached {
		// Re-parse and re-index so an open, growing log stays searchable.
		if _, err := s.GetEntries(path); err != nil {
			return err
		}
	}
	return nil
}
