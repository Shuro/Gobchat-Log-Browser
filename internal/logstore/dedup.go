package logstore

import (
	"crypto/sha256"
	"io"
	"os"
	"time"
)

// hashEntry is one memoized content hash, validated by mtime+size so a rewritten
// file is re-hashed rather than served stale.
type hashEntry struct {
	modTime time.Time
	size    int64
	hash    [32]byte
}

// dedupe collapses log files that share a filename to a single winner per name
// (ADR-0015). Files in different directories with the same Gobchat filename are
// the same chat session captured by both Gobchat and its fork GobchatEx. The
// resolution per collision is:
//
//   - identical content (equal size and content hash) → keep the lowest
//     SourcePriority (GobchatEx is scanned first, so its copy is preferred);
//   - differing content → keep the newer file (later ModTime);
//   - any remaining tie → keep the lexicographically smaller FilePath so the
//     result is deterministic.
//
// Files with a unique filename pass through untouched and are never hashed.
func (s *LogStore) dedupe(metas []*LogMeta) []*LogMeta {
	groups := map[string][]*LogMeta{}
	for _, m := range metas {
		groups[m.FileName] = append(groups[m.FileName], m)
	}
	winners := make([]*LogMeta, 0, len(groups))
	for _, group := range groups {
		best := group[0]
		for _, m := range group[1:] {
			best = s.preferred(best, m)
		}
		winners = append(winners, best)
	}
	return winners
}

// preferred returns the meta to keep between two same-named candidates.
func (s *LogStore) preferred(a, b *LogMeta) *LogMeta {
	if s.identicalContent(a, b) {
		if a.SourcePriority != b.SourcePriority {
			if a.SourcePriority < b.SourcePriority {
				return a
			}
			return b
		}
		return lowerPath(a, b)
	}
	if a.ModTime.After(b.ModTime) {
		return a
	}
	if b.ModTime.After(a.ModTime) {
		return b
	}
	return lowerPath(a, b)
}

func lowerPath(a, b *LogMeta) *LogMeta {
	if a.FilePath <= b.FilePath {
		return a
	}
	return b
}

// identicalContent reports whether two files have byte-identical content. Sizes
// are compared first (cheap); only equal-size files are hashed. If either hash
// cannot be computed the files are treated as differing, so an unreadable copy
// never masquerades as a duplicate.
func (s *LogStore) identicalContent(a, b *LogMeta) bool {
	if a.SizeBytes != b.SizeBytes {
		return false
	}
	ha, okA := s.contentHash(a)
	hb, okB := s.contentHash(b)
	return okA && okB && ha == hb
}

// contentHash returns the SHA-256 of the file's bytes, memoized and validated by
// mtime+size so repeated List calls do not re-read unchanged files.
func (s *LogStore) contentHash(m *LogMeta) ([32]byte, bool) {
	s.hashMu.Lock()
	if e, ok := s.hashCache[m.FilePath]; ok && e.size == m.SizeBytes && e.modTime.Equal(m.ModTime) {
		s.hashMu.Unlock()
		return e.hash, true
	}
	s.hashMu.Unlock()

	f, err := os.Open(m.FilePath)
	if err != nil {
		return [32]byte{}, false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [32]byte{}, false
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))

	s.hashMu.Lock()
	s.hashCache[m.FilePath] = hashEntry{modTime: m.ModTime, size: m.SizeBytes, hash: sum}
	s.hashMu.Unlock()
	return sum, true
}
