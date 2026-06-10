package logstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gobchat-log-browser/internal/parser"
)

const metaCacheVersion = 1

// metaCacheFile is the on-disk schema of index.json.
type metaCacheFile struct {
	Version int                   `json:"version"`
	Files   map[string]cachedMeta `json:"files"` // keyed by absolute FilePath
}

type cachedMeta struct {
	ModTimeUnixNano int64   `json:"mtime_unix_ns"`
	SizeBytes       int64   `json:"size_bytes"`
	Meta            LogMeta `json:"meta"`
}

// MetaCache is a concurrency-safe, file-backed cache of LogMeta that persists
// between launches in a JSON sidecar (index.json) so unchanged log files are
// not re-parsed on every scan (see docs/adr/0009). Entries are keyed by
// absolute file path and validated by mtime+size. The cache holds only derived
// data: a missing, corrupt, or version-mismatched file silently yields an
// empty cache and the next scan rebuilds it.
type MetaCache struct {
	mu       sync.Mutex
	savePath string
	files    map[string]cachedMeta
}

// LoadMetaCache loads the cache at savePath. It never fails: a missing,
// unreadable, corrupt, or version-mismatched file results in an empty cache,
// which simply means the next scan parses everything (full rescan).
func LoadMetaCache(savePath string) *MetaCache {
	c := &MetaCache{savePath: savePath, files: map[string]cachedMeta{}}
	data, err := os.ReadFile(savePath)
	if err != nil {
		return c
	}
	var f metaCacheFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != metaCacheVersion || f.Files == nil {
		return c
	}
	c.files = f.Files
	return c
}

// Get returns the cached metadata for path if mtime and size still match.
// The returned LogMeta is a copy; mutating it does not affect the cache.
func (c *MetaCache) Get(path string, mtime time.Time, size int64) (*LogMeta, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cm, ok := c.files[path]
	if !ok || cm.ModTimeUnixNano != mtime.UnixNano() || cm.SizeBytes != size {
		return nil, false
	}
	m := cloneMeta(cm.Meta)
	return &m, true
}

// Put stores metadata for meta.FilePath; size is taken from meta.SizeBytes.
func (c *MetaCache) Put(meta *LogMeta, mtime time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[meta.FilePath] = cachedMeta{
		ModTimeUnixNano: mtime.UnixNano(),
		SizeBytes:       meta.SizeBytes,
		Meta:            cloneMeta(*meta),
	}
}

// Remove drops the entry for path, if any.
func (c *MetaCache) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.files, path)
}

// Prune drops every entry whose path is not in keep — used after a full scan
// so deleted files do not linger in the cache.
func (c *MetaCache) Prune(keep map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for path := range c.files {
		if _, ok := keep[path]; !ok {
			delete(c.files, path)
		}
	}
}

// Save writes the cache atomically (.tmp then rename), creating the parent
// directory if needed.
func (c *MetaCache) Save() error {
	c.mu.Lock()
	data, err := json.Marshal(metaCacheFile{Version: metaCacheVersion, Files: c.files})
	c.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.savePath), 0o755); err != nil {
		return err
	}
	tmp := c.savePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.savePath)
}

// cloneMeta deep-copies the slice fields so cached state cannot be mutated
// through returned or stored pointers.
func cloneMeta(m LogMeta) LogMeta {
	cp := m
	cp.Participants = append([]string(nil), m.Participants...)
	cp.Channels = append([]parser.Channel(nil), m.Channels...)
	return cp
}
