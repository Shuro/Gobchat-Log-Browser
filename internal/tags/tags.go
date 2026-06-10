// Package tags stores user-assigned tags and notes for log files in a single
// JSON sidecar, keyed by filename (not full path) so the metadata survives if
// the user moves their log directory (see docs/adr/0005). Log files themselves
// are never touched.
package tags

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileTags holds the tags and free-text note for one log file.
type FileTags struct {
	FileName string   `json:"file_name"`
	Tags     []string `json:"tags"`
	Note     string   `json:"note"`
}

// TagStore is a concurrency-safe, file-backed store of FileTags.
type TagStore struct {
	mu       sync.RWMutex
	data     map[string]FileTags
	savePath string
}

// NewTagStore loads the sidecar at savePath (a missing file is fine) and returns
// a ready store.
func NewTagStore(savePath string) (*TagStore, error) {
	ts := &TagStore{data: map[string]FileTags{}, savePath: savePath}
	if err := ts.load(); err != nil {
		return nil, err
	}
	return ts, nil
}

// NewEmptyTagStore returns a store with no entries that persists to savePath on
// the next SetTags. It is the last-resort fallback when the sidecar exists but
// cannot be parsed, so the app never has to run without a tag store.
func NewEmptyTagStore(savePath string) *TagStore {
	return &TagStore{data: map[string]FileTags{}, savePath: savePath}
}

func (ts *TagStore) load() error {
	data, err := os.ReadFile(ts.savePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m := map[string]FileTags{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	ts.data = m
	return nil
}

// GetTags returns the tags/note for a filename. If none exist, it returns an
// empty FileTags with the filename set and a non-nil Tags slice.
func (ts *TagStore) GetTags(fileName string) FileTags {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	if ft, ok := ts.data[fileName]; ok {
		return cloneFileTags(ft)
	}
	return FileTags{FileName: fileName, Tags: []string{}}
}

// SetTags replaces the tags/note for a filename and persists. Empty tags are
// de-duplicated; an entry with no tags and no note is removed entirely.
func (ts *TagStore) SetTags(fileName string, tagList []string, note string) error {
	clean := dedupe(tagList)
	ts.mu.Lock()
	if len(clean) == 0 && note == "" {
		delete(ts.data, fileName)
	} else {
		ts.data[fileName] = FileTags{FileName: fileName, Tags: clean, Note: note}
	}
	ts.mu.Unlock()
	return ts.save()
}

// AllTags returns every distinct tag across all files, sorted — used for
// autocomplete.
func (ts *TagStore) AllTags() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	set := map[string]struct{}{}
	for _, ft := range ts.data {
		for _, t := range ft.Tags {
			set[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func (ts *TagStore) save() error {
	ts.mu.RLock()
	data, err := json.MarshalIndent(ts.data, "", "  ")
	ts.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ts.savePath), 0o755); err != nil {
		return err
	}
	tmp := ts.savePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ts.savePath)
}

func cloneFileTags(ft FileTags) FileTags {
	cp := FileTags{FileName: ft.FileName, Note: ft.Note, Tags: make([]string, len(ft.Tags))}
	copy(cp.Tags, ft.Tags)
	return cp
}

// dedupe removes empty and duplicate tags while preserving first-seen order.
func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, t := range in {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
