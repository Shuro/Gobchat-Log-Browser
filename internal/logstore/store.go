// Package logstore is the central registry of known log files. It scans
// configured directories for lightweight metadata, parses individual logs
// lazily (caching the result), and feeds parsed entries into the search index.
// All file access is read-only.
package logstore

import (
	"os"
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
	metaCache *MetaCache // persistent metadata cache; nil disables it

	// parseMu serializes parse+index of uncached files so two concurrent
	// GetEntries calls for the same path cannot both parse and double-index it.
	parseMu sync.Mutex

	// hashMu guards hashCache, a memoized content-hash store used by List's
	// duplicate dedup (ADR-0015) keyed by file path and validated by mtime+size.
	hashMu    sync.Mutex
	hashCache map[string]hashEntry
}

// New creates a LogStore that indexes parsed entries into idx (idx may be nil to
// disable indexing) and reuses cached metadata from metaCache (nil disables the
// persistent cache).
func New(idx *search.Index, metaCache *MetaCache) *LogStore {
	return &LogStore{
		metas:     map[string]*LogMeta{},
		cache:     map[string]*parser.ParsedLog{},
		index:     idx,
		metaCache: metaCache,
		hashCache: map[string]hashEntry{},
	}
}

// ScanAll rescans the given root directories recursively, replacing the current
// metadata set and recording the directories to watch. It does not parse-cache
// or index entries; that happens lazily per file.
func (s *LogStore) ScanAll(dirs []string) error {
	fresh := map[string]*LogMeta{}
	watch := []string{}
	seenWatch := map[string]struct{}{}
	for i, dir := range dirs {
		metas, wdirs, err := ScanDirectory(dir, s.metaCache)
		if err != nil {
			return err
		}
		for _, m := range metas {
			// Earlier dirs win identical-content dedup ties; effectiveDirs lists
			// GobchatEx before Gobchat so its copies are preferred (ADR-0015).
			m.SourcePriority = i
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

	// Drop memoized content hashes for files that disappeared, mirroring the
	// parse-cache cleanup above so hashCache cannot grow unbounded (ADR-0015).
	s.hashMu.Lock()
	for path := range s.hashCache {
		if _, ok := fresh[path]; !ok {
			delete(s.hashCache, path)
		}
	}
	s.hashMu.Unlock()

	if s.metaCache != nil {
		keep := map[string]struct{}{}
		for path := range fresh {
			keep[path] = struct{}{}
		}
		s.metaCache.Prune(keep)
		// Persistence is an optimization; a failed save must not fail the scan.
		_ = s.metaCache.Save()
	}
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

// List returns all known metadata, newest first, with duplicate log files
// collapsed to a single entry per filename (ADR-0015). Deduplication happens on
// this read boundary — not in ScanAll — so any path updated by the watcher via
// Refresh is re-evaluated on the next List, and because Search builds its pool
// from List the hidden duplicates are never listed, opened, or indexed.
func (s *LogStore) List() []LogMeta {
	s.mu.RLock()
	metas := make([]*LogMeta, 0, len(s.metas))
	for _, m := range s.metas {
		metas = append(metas, m)
	}
	s.mu.RUnlock()
	// Dedup (which may hash files for identical-content comparison) runs outside
	// the lock; the *LogMeta values it reads are never mutated in place after
	// insertion, so copying them here is safe.
	winners := s.dedupe(metas)
	out := make([]LogMeta, 0, len(winners))
	for _, m := range winners {
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

	s.parseMu.Lock()
	defer s.parseMu.Unlock()
	// Re-check: another caller may have parsed the file while we waited.
	s.mu.RLock()
	cached, ok = s.cache[path]
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
	// Stat before parsing so a file growing mid-parse records a stale mtime and
	// gets re-parsed on the next scan rather than served stale from the cache.
	info, statErr := os.Stat(path)
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
		if s.metaCache != nil {
			s.metaCache.Remove(path)
			_ = s.metaCache.Save()
		}
		return err
	}
	if s.metaCache != nil && statErr == nil {
		s.metaCache.Put(meta, info.ModTime())
		_ = s.metaCache.Save()
	}

	s.mu.Lock()
	if old, ok := s.metas[path]; ok {
		// Preserve the scan-root priority assigned in ScanAll; ExtractMeta does
		// not know it, and resetting it to 0 can flip duplicate-log dedup ties
		// away from the preferred source (ADR-0015).
		meta.SourcePriority = old.SourcePriority
	}
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
