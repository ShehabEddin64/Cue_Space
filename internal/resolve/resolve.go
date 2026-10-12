// Package resolve turns short phrases such as "figma", "github.com/me/atlas"
// or "~/Notes" into Cue resources without asking an AI. Whatever it can't
// resolve is returned so the caller can decide what to do (ask the AI, or
// tell the person).
package resolve

import (
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
)

// Result of resolving a list of phrases.
type Result struct {
	Resources []cue.Resource `json:"resources"`
	// Unresolved phrases, in the order given.
	Unresolved []string `json:"unresolved"`
}

// Resolver holds what resolution needs from the system; tests swap it out.
type Resolver struct {
	Apps []platform.App
	Home string
	Stat func(string) (os.FileInfo, error)
}

// Split breaks "figma, ~/Notes and github.com/me/atlas" into phrases.
func Split(text string) []string {
	var out []string
	for _, part := range splitter.Split(text, -1) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

var splitter = regexp.MustCompile(`\s*(?:,|;|\n|\s+and\s+|\s+&\s+)\s*`)

func (r Resolver) Resolve(phrases []string) Result {
	res := Result{Resources: []cue.Resource{}, Unresolved: []string{}}
	for _, p := range phrases {
		if rsc, ok := r.one(p); ok {
			rsc.ID = cue.NewID()
			res.Resources = append(res.Resources, rsc)
		} else {
			res.Unresolved = append(res.Unresolved, p)
		}
	}
	return res
}

var (
	schemeURL = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://\S+$`)
	bareHost  = regexp.MustCompile(`^(localhost|[\w-]+(\.[\w-]+)+)(:\d+)?(/\S*)?$`)
)

func (r Resolver) one(phrase string) (cue.Resource, bool) {
	p := strings.Trim(strings.TrimSpace(phrase), `"'`)

	// Links: full URLs, or a bare host like github.com/me/atlas.
	if schemeURL.MatchString(p) {
		return cue.Resource{Type: cue.TypeURL, URL: p, Label: labelFromURL(p)}, true
	}
	if bareHost.MatchString(p) && !strings.Contains(p, " ") && !looksLikeFile(p) {
		scheme := "https://"
		if strings.HasPrefix(p, "localhost") {
			scheme = "http://"
		}
		u := scheme + p
		return cue.Resource{Type: cue.TypeURL, URL: u, Label: labelFromURL(u)}, true
	}

	// Paths that exist on this Mac.
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		full := p
		if p == "~" || strings.HasPrefix(p, "~/") {
			full = r.Home + strings.TrimPrefix(p, "~")
		}
		if info, err := r.stat(full); err == nil {
			t := cue.TypeFile
			if info.IsDir() && !strings.HasSuffix(full, ".app") {
				t = cue.TypeFolder
			}
			return cue.Resource{Type: t, Path: p}, true
		}
		return cue.Resource{}, false
	}

	// Installed apps, by exact name, then by a unique prefix ("vs code" is
	// not a prefix of "Visual Studio Code", so the AI handles that one).
	if app, ok := r.app(p); ok {
		return cue.Resource{Type: cue.TypeApp, App: &cue.AppRef{BundleID: app.BundleID, Name: app.Name}}, true
	}
	return cue.Resource{}, false
}

func (r Resolver) stat(path string) (os.FileInfo, error) {
	if r.Stat != nil {
		return r.Stat(path)
	}
	return os.Stat(path)
}

func (r Resolver) app(name string) (platform.App, bool) {
	n := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), ".app"))
	if n == "" {
		return platform.App{}, false
	}
	for _, a := range r.Apps {
		if strings.ToLower(a.Name) == n {
			return a, true
		}
	}
	var match platform.App
	count := 0
	for _, a := range r.Apps {
		if strings.HasPrefix(strings.ToLower(a.Name), n) {
			match = a
			count++
		}
	}
	return match, count == 1
}

func looksLikeFile(p string) bool {
	// "notes.md" or "report.pdf" are file names, not hosts.
	for _, ext := range []string{".md", ".pdf", ".txt", ".doc", ".docx", ".key", ".pages", ".numbers", ".xlsx", ".png", ".jpg"} {
		if strings.HasSuffix(strings.ToLower(p), ext) {
			return true
		}
	}
	return false
}

// labelFromURL gives "GitHub" for github.com, "localhost" for localhost.
func labelFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "" {
		// Deep links: obsidian://… → "Obsidian".
		return strings.ToUpper(u.Scheme[:1]) + u.Scheme[1:]
	}
	if u.Hostname() == "" {
		return ""
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	if host == "localhost" || host == "127.0.0.1" {
		return "localhost"
	}
	if known, ok := knownSites[host]; ok {
		return known
	}
	name := strings.Split(host, ".")[0]
	return strings.ToUpper(name[:1]) + name[1:]
}

var knownSites = map[string]string{
	"github.com":        "GitHub",
	"gitlab.com":        "GitLab",
	"claude.ai":         "Claude",
	"chatgpt.com":       "ChatGPT",
	"figma.com":         "Figma",
	"notion.so":         "Notion",
	"youtube.com":       "YouTube",
	"linkedin.com":      "LinkedIn",
	"stackoverflow.com": "Stack Overflow",
}
