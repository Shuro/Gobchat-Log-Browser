package parser

import (
	"path/filepath"
	"testing"
)

func TestParseSample(t *testing.T) {
	pl, err := Parse(filepath.Join("testdata", "sample.log"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if pl.Version != FormatCCLv1 {
		t.Fatalf("version = %q, want %q", pl.Version, FormatCCLv1)
	}

	// 6 well-formed content lines + 1 malformed line = 7 entries; the two
	// header lines are consumed and excluded. No line may be dropped.
	if len(pl.Entries) != 7 {
		t.Fatalf("entries = %d, want 7", len(pl.Entries))
	}
	if len(pl.ParseErrors) != 1 {
		t.Fatalf("parse errors = %d, want 1", len(pl.ParseErrors))
	}

	info := pl.Entries[0]
	if info.Channel != ChannelGobchatInfo {
		t.Fatalf("entry[0].Channel = %q, want %q", info.Channel, ChannelGobchatInfo)
	}

	alpha := pl.Entries[1]
	if alpha.Channel != ChannelSay {
		t.Fatalf("alpha.Channel = %q, want Say", alpha.Channel)
	}
	if alpha.StatusSymbol != "★" {
		t.Fatalf("alpha.StatusSymbol = %q, want ★", alpha.StatusSymbol)
	}
	if alpha.DisplayName != "Alpha Tester" {
		t.Fatalf("alpha.DisplayName = %q, want %q", alpha.DisplayName, "Alpha Tester")
	}
	if alpha.Realm != "[Shiva]" {
		t.Fatalf("alpha.Realm = %q, want [Shiva]", alpha.Realm)
	}
	if alpha.PartIndex != 1 || alpha.PartTotal != 2 {
		t.Fatalf("alpha part = %d/%d, want 1/2", alpha.PartIndex, alpha.PartTotal)
	}
	if y := alpha.Timestamp.Year(); y != 2026 {
		t.Fatalf("alpha.Timestamp year = %d, want 2026", y)
	}

	beta1 := pl.Entries[3] // "ponders ... >"
	if !beta1.HasContinuation {
		t.Fatalf("beta1.HasContinuation = false, want true")
	}
	beta2 := pl.Entries[4] // `"> and finally answers ..."`
	if !beta2.IsContinuation {
		t.Fatalf("beta2.IsContinuation = false, want true")
	}

	gamma := pl.Entries[5]
	if gamma.StatusSymbol != "♥" || gamma.DisplayName != "Gamma Person" || gamma.Realm != "[Moogle]" {
		t.Fatalf("gamma sender parse wrong: symbol=%q name=%q realm=%q", gamma.StatusSymbol, gamma.DisplayName, gamma.Realm)
	}

	unknown := pl.Entries[6]
	if unknown.Channel != ChannelUnknown {
		t.Fatalf("malformed line Channel = %q, want Unknown", unknown.Channel)
	}
	if unknown.Message == "" || unknown.Message[:3] != "???" {
		t.Fatalf("malformed line did not preserve raw text: %q", unknown.Message)
	}
}

func TestBuildRegexDefaultFormat(t *testing.T) {
	cf, err := BuildRegex(DefaultFormat)
	if err != nil {
		t.Fatalf("BuildRegex: %v", err)
	}
	line := `Say [2026-05-16 20:09:30+02:00] ★Name Surname [Shiva]: Some message: with a colon.`
	g := cf.match(line)
	if g == nil {
		t.Fatalf("default format did not match a representative line")
	}
	if g["channel"] != "Say" {
		t.Fatalf("channel = %q, want Say", g["channel"])
	}
	// Sender is non-greedy and stops at the first ": " separator; the message
	// keeps its own internal colon.
	if g["sender"] != "★Name Surname [Shiva]" {
		t.Fatalf("sender = %q", g["sender"])
	}
	if g["message"] != "Some message: with a colon." {
		t.Fatalf("message = %q", g["message"])
	}
}

func TestParseSenderVariants(t *testing.T) {
	cases := []struct {
		raw, symbol, name, realm string
	}{
		{"Nevio Ateius", "", "Nevio Ateius", ""},
		{"★M'iqo Tester [Shiva]", "★", "M'iqo Tester", "[Shiva]"},
		{"♥Darya Khah [Shiva]", "♥", "Darya Khah", "[Shiva]"},
		{"Plain Name [Moogle]", "", "Plain Name", "[Moogle]"},
	}
	for _, c := range cases {
		sym, name, realm := parseSender(c.raw)
		if sym != c.symbol || name != c.name || realm != c.realm {
			t.Fatalf("parseSender(%q) = (%q,%q,%q), want (%q,%q,%q)",
				c.raw, sym, name, realm, c.symbol, c.name, c.realm)
		}
	}
}
