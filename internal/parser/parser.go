package parser

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ParseError records a line that did not match the format. The same line is
// also kept in Entries as a ChannelUnknown entry, so nothing is lost.
type ParseError struct {
	LineNumber int
	Raw        string
	Reason     string
}

// ParsedLog is the full result of parsing one log file.
type ParsedLog struct {
	FilePath    string
	Version     FormatVersion
	FormatStr   string
	Entries     []LogEntry
	ParseErrors []ParseError
}

const (
	idPrefix     = "Chatlogger Id:"
	formatPrefix = "Chatlogger format:"

	// maxPartTotal bounds the multi-part heuristic to keep fractions like dates
	// or quantities from being misread as "(N/M)" split markers.
	maxPartTotal = 30
)

// Parse reads and parses a log file. The file is opened read-only. Header lines
// are consumed; every other non-empty line produces exactly one LogEntry.
func Parse(path string) (*ParsedLog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pl := &ParsedLog{FilePath: path}
	sc := bufio.NewScanner(f)
	// RP lines can be long; allow up to 4 MiB per line.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	lineNo := 0
	var cf *CompiledFormat
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if lineNo == 1 {
			line = strings.TrimPrefix(line, string(rune(0xFEFF)))
		}

		switch {
		case strings.HasPrefix(line, idPrefix):
			pl.Version = FormatVersion(strings.TrimSpace(strings.TrimPrefix(line, idPrefix)))
			continue
		case strings.HasPrefix(line, formatPrefix):
			pl.FormatStr = strings.TrimPrefix(line, formatPrefix)
			continue
		case strings.TrimSpace(line) == "":
			continue
		}

		if cf == nil {
			if pl.FormatStr == "" {
				pl.FormatStr = DefaultFormat
			}
			cf, err = BuildRegex(pl.FormatStr)
			if err != nil {
				return nil, err
			}
		}

		entry, perr := parseLine(line, lineNo, cf)
		if perr != nil {
			pl.ParseErrors = append(pl.ParseErrors, *perr)
		}
		pl.Entries = append(pl.Entries, entry)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return pl, nil
}

func parseLine(line string, lineNo int, cf *CompiledFormat) (LogEntry, *ParseError) {
	groups := cf.match(line)
	if groups == nil {
		return LogEntry{LineNumber: lineNo, Channel: ChannelUnknown, Message: line},
			&ParseError{LineNumber: lineNo, Raw: line, Reason: "line did not match format"}
	}
	e := LogEntry{
		LineNumber: lineNo,
		Channel:    Channel(groups["channel"]),
		Message:    groups["message"],
		Sender:     groups["sender"],
		Timestamp:  parseTimestamp(groups["date"], groups["time"]),
	}
	e.StatusSymbol, e.DisplayName, e.Realm = parseSender(e.Sender)
	detectHeuristics(&e)
	return e, nil
}

func parseTimestamp(date, t string) time.Time {
	combined := strings.TrimSpace(date + " " + t)
	if combined == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if ts, err := time.Parse(l, combined); err == nil {
			return ts
		}
	}
	return time.Time{}
}

// parseSender splits a raw sender field into its leading status symbol(s),
// display name, and trailing "[Realm]" suffix. All parts are best-effort and
// the raw value is always preserved by the caller.
func parseSender(s string) (symbol, display, realm string) {
	display = strings.TrimSpace(s)

	// Trailing "[…]" realm tag.
	if i := strings.LastIndex(display, "["); i >= 0 && strings.HasSuffix(display, "]") {
		realm = display[i:]
		display = strings.TrimSpace(display[:i])
	}

	// Leading run of non-letter/non-digit/non-space runes (status badges).
	runes := []rune(display)
	j := 0
	for j < len(runes) {
		r := runes[j]
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			break
		}
		j++
	}
	if j > 0 {
		symbol = StripPrivateUse(string(runes[:j]))
		display = strings.TrimSpace(string(runes[j:]))
	}
	return symbol, display, realm
}

// StripPrivateUse drops Unicode private-use-area runes (U+E000–U+F8FF) from a
// sender/status string for display. FFXIV prefixes party senders with PUA
// glyphs (party slot icons such as U+E0E1/U+E091…) that only the game font can
// render — outside the game they show as tofu boxes. Real symbols (★, ♥, …)
// are kept; the raw sender field is never modified.
func StripPrivateUse(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0xE000 && r <= 0xF8FF {
			return -1
		}
		return r
	}, s)
}

var partRe = regexp.MustCompile(`\(?(\d{1,3})\s*/\s*(\d{1,3})\)?\s*$`)

// continuationMarkers are the hardcoded player conventions for "post continues"
// (trailing) / "continues a post" (leading). Best-effort heuristics, see
// docs/adr/0006; ordered longest-first so trimming never strips a partial
// marker (e.g. "->" before ">").
var continuationMarkers = []string{"->", ">>", ">", "+"}

// detectHeuristics fills the low-confidence continuation/multi-part fields from
// the message text. These are player conventions and may be wrong or absent.
func detectHeuristics(e *LogEntry) {
	rTrim := strings.TrimRight(e.Message, " \t")
	lTrim := strings.TrimLeft(e.Message, " \t\"")
	for _, m := range continuationMarkers {
		if strings.HasSuffix(rTrim, m) {
			e.HasContinuation = true
		}
		if strings.HasPrefix(lTrim, m) {
			e.IsContinuation = true
		}
	}
	if m := partRe.FindStringSubmatch(rTrim); m != nil {
		idx, _ := strconv.Atoi(m[1])
		tot, _ := strconv.Atoi(m[2])
		if idx > 0 && tot > 0 && idx <= tot && tot <= maxPartTotal {
			e.PartIndex = idx
			e.PartTotal = tot
		}
	}
}
