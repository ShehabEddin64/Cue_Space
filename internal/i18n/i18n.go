// Package i18n translates the messages Cue Space shows people. English
// strings are the keys: T("%s is busy", v) looks the format up in the
// current language's table and falls back to English when it isn't there,
// so untranslated text is never lost.
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

var current atomic.Value // "en" or "fr"

// Resolve turns a preference ("system", "en", "fr") and the system's
// preferred language tag into a supported language.
func Resolve(pref, system string) string {
	switch pref {
	case "en", "fr":
		return pref
	}
	if strings.HasPrefix(strings.ToLower(system), "fr") {
		return "fr"
	}
	return "en"
}

// SetLanguage switches the language for messages created from now on.
func SetLanguage(lang string) {
	if lang != "fr" {
		lang = "en"
	}
	current.Store(lang)
}

// Language is the current language, "en" or "fr".
func Language() string {
	if l, ok := current.Load().(string); ok {
		return l
	}
	return "en"
}

// Translate returns format in the current language.
func Translate(format string) string {
	if Language() == "fr" {
		if t, ok := fr[format]; ok {
			return t
		}
	}
	return format
}

// T translates format and fills in args, like fmt.Sprintf.
func T(format string, args ...any) string {
	if len(args) == 0 {
		return Translate(format)
	}
	return fmt.Sprintf(Translate(format), args...)
}

// Errorf translates format and returns an error, like fmt.Errorf (%w works).
func Errorf(format string, args ...any) error {
	return fmt.Errorf(Translate(format), args...)
}
