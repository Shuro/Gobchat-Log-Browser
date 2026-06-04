package logstore

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gobchat-log-browser/internal/parser"
)

// LogMeta is the lightweight overview information for one log file. Tags/notes
// are intentionally NOT here — the API layer merges those from the tag store so
// this package stays independent of tags.
type LogMeta struct {
	FilePath     string           `json:"file_path"`
	FileName     string           `json:"file_name"`
	Folder       string           `json:"folder"` // subfolder relative to the scan root ("" = top level)
	LogDate      time.Time        `json:"log_date"`
	MessageCount int              `json:"message_count"`
	Participants []string         `json:"participants"`
	Channels     []parser.Channel `json:"channels"`
	FirstEntry   time.Time        `json:"first_entry"`
	LastEntry    time.Time        `json:"last_entry"`
	Duration     time.Duration    `json:"duration"`
	SizeBytes    int64            `json:"size_bytes"`
}

var filenameDateRe = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})[_ ](\d{2})-(\d{2})`)

// skipDirs are directory names created by the Electron/Chromium runtime that
// Gobchat ships with. They never contain chat logs and can be large, so the
// recursive scan skips them entirely.
var skipDirs = map[string]struct{}{
	"DawnCache":       {},
	"GPUCache":        {},
	"Cache":           {},
	"Code Cache":      {},
	"blob_storage":    {},
	"Local Storage":   {},
	"Session Storage": {},
	"IndexedDB":       {},
	"Network":         {},
}

// ScanDirectory walks root recursively and returns metadata for every *.log file
// found, plus the list of directories that should be watched for changes. Logs
// may be flat or split across subfolders: Gobchat's log path is configurable per
// profile, so users often point different profiles at different subfolders.
// Electron cache and hidden directories are skipped. A missing root is not an error.
func ScanDirectory(root string) (metas []*LogMeta, watchDirs []string, err error) {
	if fi, statErr := os.Stat(root); statErr != nil || !fi.IsDir() {
		return nil, nil, nil
	}

	seenDir := map[string]struct{}{}
	addDir := func(d string) {
		if _, ok := seenDir[d]; !ok {
			seenDir[d] = struct{}{}
			watchDirs = append(watchDirs, d)
		}
	}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // tolerate unreadable entries; keep scanning
		}
		if d.IsDir() {
			if path != root {
				name := d.Name()
				if _, skip := skipDirs[name]; skip || strings.HasPrefix(name, ".") {
					return fs.SkipDir
				}
			}
			addDir(path)
			return nil
		}
		if !IsLogFile(d.Name()) {
			return nil
		}
		meta, e := ExtractMeta(path)
		if e != nil {
			return nil // unreadable file: skip rather than fail the scan
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		if rel == "." {
			rel = ""
		}
		meta.Folder = filepath.ToSlash(rel)
		metas = append(metas, meta)
		return nil
	})
	if walkErr != nil {
		return nil, nil, walkErr
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].LogDate.After(metas[j].LogDate) })
	return metas, watchDirs, nil
}

// ExtractMeta parses a single log file and derives its overview metadata.
func ExtractMeta(path string) (*LogMeta, error) {
	pl, err := parser.Parse(path)
	if err != nil {
		return nil, err
	}
	var size int64
	if info, statErr := os.Stat(path); statErr == nil {
		size = info.Size()
	}
	return metaFromParsed(path, pl, size), nil
}

func metaFromParsed(path string, pl *parser.ParsedLog, size int64) *LogMeta {
	name := filepath.Base(path)
	m := &LogMeta{
		FilePath:     path,
		FileName:     name,
		MessageCount: len(pl.Entries),
		SizeBytes:    size,
		LogDate:      filenameDate(name),
	}

	seenParticipant := map[string]struct{}{}
	seenChannel := map[parser.Channel]struct{}{}
	for _, e := range pl.Entries {
		// System/info lines are not roleplay participants.
		if e.DisplayName != "" && e.Channel != parser.ChannelGobchatInfo {
			if _, ok := seenParticipant[e.DisplayName]; !ok {
				seenParticipant[e.DisplayName] = struct{}{}
				m.Participants = append(m.Participants, e.DisplayName)
			}
		}
		if e.Channel != "" {
			if _, ok := seenChannel[e.Channel]; !ok {
				seenChannel[e.Channel] = struct{}{}
				m.Channels = append(m.Channels, e.Channel)
			}
		}
		if !e.Timestamp.IsZero() {
			if m.FirstEntry.IsZero() || e.Timestamp.Before(m.FirstEntry) {
				m.FirstEntry = e.Timestamp
			}
			if e.Timestamp.After(m.LastEntry) {
				m.LastEntry = e.Timestamp
			}
		}
	}
	sort.Strings(m.Participants)
	if !m.FirstEntry.IsZero() && !m.LastEntry.IsZero() {
		m.Duration = m.LastEntry.Sub(m.FirstEntry)
	}
	if m.LogDate.IsZero() {
		m.LogDate = m.FirstEntry // fall back to first timestamp if filename lacks a date
	}
	return m
}

// filenameDate extracts the date/time encoded in a Gobchat filename like
// "chatlog_2026-01-02_20-01.log". Returns the zero time if absent.
func filenameDate(name string) time.Time {
	m := filenameDateRe.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}
	}
	ts, err := time.ParseInLocation("2006-01-02 15-04", m[1]+" "+m[2]+"-"+m[3], time.Local)
	if err != nil {
		return time.Time{}
	}
	return ts
}

// IsLogFile reports whether a filename is a log file this app handles.
func IsLogFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".log")
}
