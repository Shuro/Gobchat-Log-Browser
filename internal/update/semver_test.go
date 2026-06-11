package update

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want [3]int
		ok   bool
	}{
		{"1.2.3", [3]int{1, 2, 3}, true},
		{"v1.2.3", [3]int{1, 2, 3}, true},
		{" v0.1.0 ", [3]int{0, 1, 0}, true},
		{"0.0.0", [3]int{0, 0, 0}, true},
		{"10.20.30", [3]int{10, 20, 30}, true},
		{"dev", [3]int{}, false},
		{"", [3]int{}, false},
		{"1.2", [3]int{}, false},
		{"1.2.3.4", [3]int{}, false},
		{"1.2.3-rc1", [3]int{}, false},
		{"1.2.x", [3]int{}, false},
		{"1.02.3", [3]int{}, false}, // leading zero is not a release tag form
		{"-1.2.3", [3]int{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseVersion(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "0.1.0", true},
		{"0.2.0", "v0.1.0", true},
		{"0.10.0", "0.9.9", true}, // numeric, not lexical
		{"1.0.0", "0.99.99", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"v1.2.3", "dev", false}, // unparsable current: never report an update
		{"dev", "0.1.0", false},
		{"", "", false},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v; want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}
