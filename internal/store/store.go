// Package store keeps Cues as one .cue.json file each in a folder the user
// owns. Files can be edited by hand; the store picks up changes on the next List.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

const ext = ".cue.json"

var ErrNotFound = errors.New("cue not found")

// LoadError reports a file in the folder that could not be read as a Cue.
type LoadError struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

type Store struct {
	dir string
	// Trash moves a deleted Cue's file somewhere recoverable. Defaults to
	// os.Remove only when nil, which is meant for tests.
	Trash func(path string) error
	now   func() time.Time

	mu    sync.Mutex
	paths map[string]string // cue id -> file path
}

// Open uses dir as the Cue folder, creating it if needed.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir, now: time.Now, paths: map[string]string{}}, nil
}

func (s *Store) Dir() string { return s.dir }

// List reads every Cue in the folder, sorted by name. Files that fail to parse
// are returned as LoadErrors instead of failing the whole list.
func (s *Store) List() ([]cue.Cue, []LoadError, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scan()
}

func (s *Store) scan() ([]cue.Cue, []LoadError, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, nil, err
	}
	cues := []cue.Cue{}
	var bad []LoadError
	paths := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			bad = append(bad, LoadError{File: e.Name(), Error: err.Error()})
			continue
		}
		c, err := cue.Parse(data)
		if err != nil {
			bad = append(bad, LoadError{File: e.Name(), Error: err.Error()})
			continue
		}
		if other, dup := paths[c.ID]; dup {
			bad = append(bad, LoadError{File: e.Name(), Error: "same id as " + filepath.Base(other)})
			continue
		}
		if !hasID(data) {
			// A hand-written file without an id: persist the generated one so
			// the Cue keeps a stable identity.
			if err := writeAtomic(path, c); err != nil {
				bad = append(bad, LoadError{File: e.Name(), Error: err.Error()})
				continue
			}
		}
		paths[c.ID] = path
		cues = append(cues, c)
	}
	s.paths = paths
	sort.SliceStable(cues, func(i, j int) bool {
		return strings.ToLower(cues[i].Name) < strings.ToLower(cues[j].Name)
	})
	return cues, bad, nil
}

func hasID(data []byte) bool {
	var probe struct {
		ID string `json:"id"`
	}
	return json.Unmarshal(data, &probe) == nil && probe.ID != ""
}

func (s *Store) Get(id string) (cue.Cue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.pathFor(id)
	if err != nil {
		return cue.Cue{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cue.Cue{}, err
	}
	return cue.Parse(data)
}

// pathFor finds the file for id, rescanning once in case it was added by hand.
func (s *Store) pathFor(id string) (string, error) {
	if p, ok := s.paths[id]; ok {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if _, _, err := s.scan(); err != nil {
		return "", err
	}
	if p, ok := s.paths[id]; ok {
		return p, nil
	}
	return "", ErrNotFound
}

// Save validates and writes c, creating a new file named after the Cue if it
// is new. It returns the Cue as stored, with ids and timestamps filled in.
func (s *Store) Save(c cue.Cue) (cue.Cue, error) {
	c.Normalize()
	if err := c.Validate(); err != nil {
		return c, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC().Truncate(time.Second)
	path, err := s.pathFor(c.ID)
	switch {
	case errors.Is(err, ErrNotFound):
		path = s.newPath(c.Name)
	case err != nil:
		return c, err
	default:
		// Keep the original creation time; the file is the source of truth.
		if data, err := os.ReadFile(path); err == nil {
			if old, err := cue.Parse(data); err == nil && !old.CreatedAt.IsZero() {
				c.CreatedAt = old.CreatedAt
			}
		}
	}
	if c.CreatedAt.IsZero() {
		// New, or a hand-written file saved from the app for the first time.
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if err := writeAtomic(path, c); err != nil {
		return c, err
	}
	s.paths[c.ID] = path
	return c, nil
}

// MarkRun records that a Cue was started.
func (s *Store) MarkRun(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.pathFor(id)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	c, err := cue.Parse(data)
	if err != nil {
		return err
	}
	c.LastRunAt = at.UTC().Truncate(time.Second)
	return writeAtomic(path, c)
}

// Delete moves the Cue's file to the Trash.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.pathFor(id)
	if err != nil {
		return err
	}
	trash := s.Trash
	if trash == nil {
		trash = os.Remove
	}
	if err := trash(path); err != nil {
		return fmt.Errorf("move to trash: %w", err)
	}
	delete(s.paths, id)
	return nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 48 {
		s = strings.TrimRight(s[:48], "-")
	}
	if s == "" {
		s = "cue"
	}
	return s
}

func (s *Store) newPath(name string) string {
	base := slug(name)
	for i := 1; ; i++ {
		file := base + ext
		if i > 1 {
			file = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		p := filepath.Join(s.dir, file)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}

// writeAtomic writes via a temp file and rename so a crash never leaves a
// half-written Cue behind.
func writeAtomic(path string, c cue.Cue) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
