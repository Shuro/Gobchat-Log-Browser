// Package i18n provides translations for backend-generated strings (errors and
// watcher notifications). UI strings live in the frontend (vue-i18n); the
// frontend can fetch this map and merge it. Locale files are embedded so the
// binary stays self-contained.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/*.json
var localeFS embed.FS

const defaultLang = "en"

// Localizer resolves message keys for one language, falling back to English.
type Localizer struct {
	lang     string
	messages map[string]string
	fallback map[string]string
}

// New loads the localizer for lang. English is always loaded as the fallback;
// if lang is missing or fails to load, English is used for both.
func New(lang string) (*Localizer, error) {
	fallback, err := loadLocale(defaultLang)
	if err != nil {
		return nil, err
	}
	messages := fallback
	if lang != "" && lang != defaultLang {
		if m, err := loadLocale(lang); err == nil {
			messages = m
		}
	}
	return &Localizer{lang: lang, messages: messages, fallback: fallback}, nil
}

func loadLocale(lang string) (map[string]string, error) {
	data, err := localeFS.ReadFile("locales/" + lang + ".json")
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse locale %q: %w", lang, err)
	}
	return m, nil
}

// Lang returns the active language code.
func (l *Localizer) Lang() string { return l.lang }

// T translates key, falling back to English and finally to the key itself.
func (l *Localizer) T(key string) string {
	if v, ok := l.messages[key]; ok {
		return v
	}
	if v, ok := l.fallback[key]; ok {
		return v
	}
	return key
}

// TF translates key and formats it with fmt.Sprintf-style args.
func (l *Localizer) TF(key string, args ...any) string {
	return fmt.Sprintf(l.T(key), args...)
}

// Messages returns the active language's full message map (for the frontend to
// merge into its own translations).
func (l *Localizer) Messages() map[string]string {
	out := make(map[string]string, len(l.messages))
	for k, v := range l.messages {
		out[k] = v
	}
	return out
}
