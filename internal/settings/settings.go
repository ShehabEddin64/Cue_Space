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
	AI ai.Config `json:"ai"`
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
	var s Settings
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(data, &s)
}

func (f *File) Save(s Settings) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
