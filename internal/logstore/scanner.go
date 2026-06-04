package logstore

import (
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

// ScanDirectory parses every *.log file in dir (non-recursive) and returns their
// metadata, sorted by log date descending (newest first). Files that cannot be
// read are skipped; a parse never fails outright (malformed lines are tolerated).
func ScanDirectory(dir string) ([]*LogMeta, error) {
	pattern := filepath.Join(dir, "*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	metas := make([]*LogMeta, 0, len(matches))
	for _, path := range matches {
		meta, err := ExtractMeta(path)
		if err != nil {
			continue // unreadable file: skip rather than fail the whole scan
		}
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].LogDate.After(metas[j].LogDate)
	})
	return metas, nil
}

// ExtractMeta parses a single log file and derives its overview metadata.
func ExtractMeta(path string) (*LogMeta, error) {
	pl, err := parser.Parse(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	var size int64
	if err == nil {
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

// IsLogFile reports whether a path looks like a log file this app handles.
func IsLogFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".log")
}
