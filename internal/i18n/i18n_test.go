package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	call = regexp.MustCompile(`(?:i18n\.(?:T|Errorf)|e\.add\((?:true|false),)\s*\(?("(?:[^"\\]|\\.)*")`)
	verb = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)
)

// Every message marked for translation in the Go code has a French version
// with the same format verbs, in the same order.
func TestFrenchCoversEveryMessage(t *testing.T) {
	keys := map[string]string{} // key → where it's used
	root := filepath.Join("..", "..")
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (info.Name() == "frontend" || info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, _ := os.ReadFile(path)
		for _, m := range call.FindAllStringSubmatch(string(data), -1) {
			if k, err := strconv.Unquote(m[1]); err == nil {
				keys[k] = path
			}
		}
		return nil
	})
	if len(keys) < 50 {
		t.Fatalf("found only %d messages; is the pattern right?", len(keys))
	}
	keys["Name is missing"] = "internal/cue/validate.go"
	// Standard menu items relabeled in main.go.
	for _, k := range []string{"About Cue Space", "Hide Cue Space", "Hide Others", "Show All", "Quit Cue Space", "Close Window", "Undo", "Redo", "Cut", "Copy", "Paste", "Paste and Match Style", "Delete", "Select All", "Start Speaking", "Stop Speaking", "Enter Full Screen", "Minimize", "Zoom", "Bring All to Front", "Edit", "Window", "Speech", "Services"} {
		keys[k] = "main.go"
	}
	for _, title := range []string{"Full Screen", "Fill", "Left & Right", "Top & Bottom", "Left & Quarters", "Quarters", "Thirds"} {
		keys[title] = "internal/cue/layout.go"
	}
	for k, where := range keys {
		tr, ok := fr[k]
		if !ok {
			t.Errorf("no French for %q (%s)", k, where)
			continue
		}
		if a, b := verb.FindAllString(k, -1), verb.FindAllString(tr, -1); strings.Join(a, " ") != strings.Join(b, " ") {
			t.Errorf("format verbs differ for %q: %v vs %v", k, a, b)
		}
	}
}

func TestTranslate(t *testing.T) {
	defer SetLanguage("en")
	SetLanguage("fr")
	if got := T("Opened %s", "a.pdf"); got != "a.pdf ouvert" {
		t.Errorf("fr: %q", got)
	}
	if got := T("not in the table"); got != "not in the table" {
		t.Errorf("fallback: %q", got)
	}
	SetLanguage("en")
	if got := T("Opened %s", "a.pdf"); got != "Opened a.pdf" {
		t.Errorf("en: %q", got)
	}
	for _, c := range []struct{ pref, sys, want string }{
		{"system", "fr-CA", "fr"}, {"system", "en-US", "en"}, {"en", "fr-FR", "en"}, {"fr", "en", "fr"}, {"", "fr", "fr"},
	} {
		if got := Resolve(c.pref, c.sys); got != c.want {
			t.Errorf("Resolve(%q, %q) = %s", c.pref, c.sys, got)
		}
	}
}
