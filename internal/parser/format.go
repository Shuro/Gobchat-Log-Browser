package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// FormatVersion is the chat logger identifier from the file header.
type FormatVersion string

const (
	FormatFCLv1 FormatVersion = "FCLv1" // fixed default format
	FormatCCLv1 FormatVersion = "CCLv1" // custom format, declared in the header
)

// DefaultFormat is Gobchat's default log line layout, used when no explicit
// format string is present (e.g. FCLv1).
const DefaultFormat = "{channel} [{date} {time-full}] {sender}: {message}"

// CompiledFormat is a format string compiled into a line-matching regexp with
// named capture groups (channel, date, time, sender, message).
type CompiledFormat struct {
	FormatStr string
	Pattern   *regexp.Regexp
}

// tokenGroup maps Gobchat format tokens to regex capture fragments. All time
// variants share the "time" group. Tokens are deliberately permissive (\S+)
// because the surrounding literals disambiguate them; sender is non-greedy so
// it stops at the first separator, and message is greedy to take the remainder.
var tokenGroup = map[string]string{
	"channel":    `(?P<channel>\S+)`,
	"date":       `(?P<date>\S+)`,
	"time":       `(?P<time>\S+)`,
	"time-short": `(?P<time>\S+)`,
	"time-full":  `(?P<time>\S+)`,
	"sender":     `(?P<sender>.+?)`,
	"sender-cha": `(?P<sender>.+?)`,
	"message":    `(?P<message>.*)`,
}

// BuildRegex converts a Gobchat format string into an anchored regexp. Literal
// text between tokens is regex-escaped; unknown tokens become a generic
// non-greedy capture so an unusual custom format still parses best-effort.
func BuildRegex(formatStr string) (*CompiledFormat, error) {
	var b strings.Builder
	b.WriteString("^")
	i := 0
	for i < len(formatStr) {
		if formatStr[i] == '{' {
			end := strings.IndexByte(formatStr[i:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unterminated token in format %q", formatStr)
			}
			token := formatStr[i+1 : i+end]
			frag, ok := tokenGroup[token]
			if !ok {
				frag = `(?:.+?)`
			}
			b.WriteString(frag)
			i += end + 1
			continue
		}
		next := strings.IndexByte(formatStr[i:], '{')
		if next < 0 {
			b.WriteString(regexp.QuoteMeta(formatStr[i:]))
			break
		}
		b.WriteString(regexp.QuoteMeta(formatStr[i : i+next]))
		i += next
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil, fmt.Errorf("compile format regex from %q: %w", formatStr, err)
	}
	return &CompiledFormat{FormatStr: formatStr, Pattern: re}, nil
}

// match returns the named capture groups for a line, or nil if it does not match.
func (cf *CompiledFormat) match(line string) map[string]string {
	m := cf.Pattern.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	groups := make(map[string]string, len(m))
	for i, name := range cf.Pattern.SubexpNames() {
		if name != "" {
			groups[name] = m[i]
		}
	}
	return groups
}
