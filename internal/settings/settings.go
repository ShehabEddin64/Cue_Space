// Package settings stores app preferences as a small JSON file. Secrets such
// as API keys never go here; see package secrets.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/ShehabEddin64/Cue_Space/internal/ai"
)

type Settings struct {
	AI    ai.Config `json:"ai"`
	Prefs Prefs     `json:"prefs"`
}

// Prefs are interface choices: appearance, the Home layout, pinned Cues.
type Prefs struct {
	// Appearance is "system" (follow macOS), "light" or "dark".
	Appearance string `json:"appearance"`
	// Language is "system" (follow macOS), "en" or "fr".
	Language string `json:"language"`
	// Onboarded is true once the welcome tour has been finished or skipped.
	Onboarded bool `json:"onboarded"`
	// Pinned lists Cue ids shown first, in this order. The first nine Cues
	// (pinned, then the rest by name) get ⌘1–⌘9.
	Pinned []string  `json:"pinned"`
	Home   HomePrefs `json:"home"`
}

type HomePrefs struct {
	// Greeting replaces "What do you want to do?" when set.
	Greeting string `json:"greeting"`
	// Sections are Home's blocks below the composer, in display order.
	Sections []HomeSection `json:"sections"`
	// RecentCount is how many Cues "Recently Used" shows.
	RecentCount int `json:"recentCount"`
}

type HomeSection struct {
	ID      string `json:"id"` // "pinned" | "recent"
	Visible bool   `json:"visible"`
	// Size works like a widget size: "small", "medium" or "large".
	Size string `json:"size"`
}

var defaultSections = []HomeSection{{"pinned", true, "medium"}, {"recent", true, "medium"}}

// Normalize fills in defaults and drops unknown or repeated values, so older
// or hand-edited settings files keep working.
func (p *Prefs) Normalize() {
	switch p.Appearance {
	case "light", "dark":
	default:
		p.Appearance = "system"
	}
	switch p.Language {
	case "en", "fr":
	default:
		p.Language = "system"
	}
	if p.Pinned == nil {
		p.Pinned = []string{}
	}
	seen := map[string]bool{}
	pinned := p.Pinned[:0]
	for _, id := range p.Pinned {
		if id != "" && !seen[id] {
			seen[id] = true
			pinned = append(pinned, id)
		}
	}
	p.Pinned = pinned

	known := map[string]bool{}
	var sections []HomeSection
	for _, s := range p.Home.Sections {
		for _, d := range defaultSections {
			if s.ID == d.ID && !known[s.ID] {
				known[s.ID] = true
				switch s.Size {
				case "small", "medium", "large":
				default:
					s.Size = d.Size
				}
				sections = append(sections, s)
			}
		}
	}
	for _, d := range defaultSections {
		if !known[d.ID] {
			sections = append(sections, d)
		}
	}
	p.Home.Sections = sections
	if p.Home.RecentCount <= 0 || p.Home.RecentCount > 12 {
		p.Home.RecentCount = 6
	}
}

type File struct {
	path string
	mu   sync.Mutex
}

func Open(path string) *File { return &File{path: path} }

// Load returns the saved settings, or defaults if none are saved yet.
func (f *File) Load() (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.load()
}

func (f *File) load() (Settings, error) {
	var s Settings
	data, err := os.ReadFile(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return s, err
	}
	if err == nil {
		if err := json.Unmarshal(data, &s); err != nil {
			return s, err
		}
	}
	s.Prefs.Normalize()
	return s, nil
}

func (f *File) Save(s Settings) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.save(s)
}

// Update applies change to the stored settings in one locked
// read-modify-write, so concurrent saves of different parts don't clobber
// each other.
func (f *File) Update(change func(*Settings)) (Settings, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.load()
	if err != nil {
		return s, err
	}
	change(&s)
	s.Prefs.Normalize()
	return s, f.save(s)
}

func (f *File) save(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
