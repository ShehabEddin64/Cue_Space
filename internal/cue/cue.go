// Package cue defines the Cue model: a named, ordered list of resources that
// together make up the context for one activity.
package cue

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

// SchemaVersion is the version of the .cue.json format this build writes.
const SchemaVersion = 1

type ResourceType string

const (
	TypeURL    ResourceType = "url"
	TypeApp    ResourceType = "app"
	TypeFile   ResourceType = "file"
	TypeFolder ResourceType = "folder"
)

// AppRef identifies an installed application. BundleID is preferred because it
// survives renames; Name is the fallback (passed to `open -a`).
type AppRef struct {
	BundleID string `json:"bundleId,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (a *AppRef) IsZero() bool {
	return a == nil || (a.BundleID == "" && a.Name == "")
}

// Resource is one thing a Cue opens.
type Resource struct {
	ID       string       `json:"id"`
	Type     ResourceType `json:"type"`
	Label    string       `json:"label,omitempty"`
	Disabled bool         `json:"disabled,omitempty"`

	URL      string  `json:"url,omitempty"`      // type url
	App      *AppRef `json:"app,omitempty"`      // type app
	Path     string  `json:"path,omitempty"`     // type file, folder
	OpenWith *AppRef `json:"openWith,omitempty"` // type file, folder

	// Newest opens the most recently added or changed file directly inside
	// the folder instead of the folder itself. Type folder only.
	Newest bool `json:"newest,omitempty"`
	// Extensions limits Newest to these file types, e.g. ["pdf"]. Empty
	// means any file.
	Extensions []string `json:"extensions,omitempty"`
}

// DisplayName is the label if set, otherwise something derived from the target.
func (r Resource) DisplayName() string {
	if r.Label != "" {
		return r.Label
	}
	switch r.Type {
	case TypeURL:
		// "moodle.concordia.ca" reads better than a long URL.
		if u, err := url.Parse(r.URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			return strings.TrimPrefix(u.Hostname(), "www.")
		}
		return r.URL
	case TypeApp:
		if r.App != nil {
			if r.App.Name != "" {
				return r.App.Name
			}
			return r.App.BundleID
		}
	case TypeFile, TypeFolder:
		p := strings.TrimRight(r.Path, "/")
		if i := strings.LastIndex(p, "/"); i >= 0 {
			p = p[i+1:]
		}
		if r.Type == TypeFolder && r.Newest {
			return r.NewestKind() + " in " + p
		}
		return p
	}
	return string(r.Type)
}

// NewestKind describes what a Newest folder opens: "Newest file",
// "Newest PDF" or "Newest PDF or DOCX".
func (r Resource) NewestKind() string {
	if len(r.Extensions) == 0 {
		return "Newest file"
	}
	exts := make([]string, len(r.Extensions))
	for i, e := range r.Extensions {
		exts[i] = strings.ToUpper(e)
	}
	if len(exts) == 1 {
		return "Newest " + exts[0]
	}
	return "Newest " + strings.Join(exts[:len(exts)-1], ", ") + " or " + exts[len(exts)-1]
}

// normalizeExtensions lowercases extensions and drops dots, blanks and repeats.
func normalizeExtensions(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range in {
		e = strings.ToLower(strings.TrimLeft(strings.TrimSpace(e), "*."))
		if e != "" && !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

// Cue is a saved context for an activity.
type Cue struct {
	SchemaVersion int        `json:"schemaVersion"`
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Icon          string     `json:"icon,omitempty"`
	Description   string     `json:"description,omitempty"`
	Resources     []Resource `json:"resources"`
	// Layout optionally places windows on desktops (macOS Spaces).
	Layout    *Layout   `json:"layout,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitzero"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
	LastRunAt time.Time `json:"lastRunAt,omitzero"`
}

// Normalize trims text fields, fills in missing ids and the schema version.
// It never changes what a Cue does.
func (c *Cue) Normalize() {
	c.SchemaVersion = SchemaVersion
	if c.ID == "" {
		c.ID = NewID()
	}
	c.Name = strings.TrimSpace(c.Name)
	c.Icon = strings.TrimSpace(c.Icon)
	c.Description = strings.TrimSpace(c.Description)
	if c.Resources == nil {
		c.Resources = []Resource{}
	}
	seen := map[string]bool{}
	for i := range c.Resources {
		r := &c.Resources[i]
		if r.ID == "" || seen[r.ID] {
			r.ID = NewID()
		}
		seen[r.ID] = true
		r.Label = strings.TrimSpace(r.Label)
		r.URL = strings.TrimSpace(r.URL)
		r.Path = strings.TrimSpace(r.Path)
		if r.App.IsZero() {
			r.App = nil
		}
		if r.OpenWith.IsZero() {
			r.OpenWith = nil
		}
		r.Extensions = normalizeExtensions(r.Extensions)
		if !r.Newest {
			r.Extensions = nil
		}
	}
	c.normalizeLayout()
}

// NewID returns a short random identifier.
func NewID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
