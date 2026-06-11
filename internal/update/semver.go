// Package update checks GitHub Releases for a newer version of the app.
// Release tags are plain numeric semver (vX.Y.Z, see docs/adr/0011), so a
// tiny three-part comparison is enough and no semver dependency is needed.
package update

import (
	"strconv"
	"strings"
)

// ParseVersion parses "X.Y.Z" with an optional leading "v" and surrounding
// whitespace. ok is false for anything else ("dev", "1.2", "1.2.3-rc1"), which
// callers use to skip update checks for non-release builds.
func ParseVersion(s string) (v [3]int, ok bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p != strconv.Itoa(n) {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}

// IsNewer reports whether latest is a strictly higher version than current.
// Either side failing to parse returns false.
func IsNewer(latest, current string) bool {
	l, ok := ParseVersion(latest)
	if !ok {
		return false
	}
	c, ok := ParseVersion(current)
	if !ok {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
