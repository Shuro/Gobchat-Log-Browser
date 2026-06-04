package i18n

import "testing"

func TestNewAndTranslate(t *testing.T) {
	de, err := New("de")
	if err != nil {
		t.Fatalf("New(de): %v", err)
	}
	if got := de.T("error.parseFailed"); got != "Die Logdatei konnte nicht gelesen werden." {
		t.Fatalf("german translation wrong: %q", got)
	}
	// Formatting with args.
	if got := de.TF("notify.newLog", "chatlog_x.log"); got != "Ein neues Log wurde erkannt: chatlog_x.log" {
		t.Fatalf("TF wrong: %q", got)
	}
}

func TestFallbackToEnglish(t *testing.T) {
	// Unknown language falls back to English for both messages and fallback.
	l, err := New("fr")
	if err != nil {
		t.Fatalf("New(fr): %v", err)
	}
	if got := l.T("error.scanFailed"); got != "Could not scan the log directory." {
		t.Fatalf("expected english fallback, got %q", got)
	}
}

func TestUnknownKeyReturnsKey(t *testing.T) {
	l, _ := New("en")
	if got := l.T("no.such.key"); got != "no.such.key" {
		t.Fatalf("unknown key should echo itself, got %q", got)
	}
}
