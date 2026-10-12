package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "cues"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sample(name string) cue.Cue {
	return cue.Cue{Name: name, Resources: []cue.Resource{
		{Type: cue.TypeURL, URL: "https://example.com"},
	}}
}

func TestSaveListGetRoundTrip(t *testing.T) {
	s := newStore(t)
	saved, err := s.Save(sample("Atlas Project"))
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Fatalf("missing generated fields: %+v", saved)
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), "atlas-project.cue.json")); err != nil {
		t.Errorf("expected slug file name: %v", err)
	}

	list, bad, err := s.List()
	if err != nil || len(bad) != 0 || len(list) != 1 {
		t.Fatalf("List = %v %v %v", list, bad, err)
	}
	got, err := s.Get(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Atlas Project" || got.Resources[0].ID != saved.Resources[0].ID {
		t.Errorf("Get = %+v", got)
	}
}

func TestSaveExistingKeepsFileAndCreatedAt(t *testing.T) {
	s := newStore(t)
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	first, err := s.Save(sample("Math"))
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(time.Hour)
	first.Name = "Math (renamed)"
	first.CreatedAt = time.Time{} // the UI may not send it back
	second, err := s.Save(first)
	if err != nil {
		t.Fatal(err)
	}
	if !second.CreatedAt.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("CreatedAt changed: %v", second.CreatedAt)
	}
	if !second.UpdatedAt.Equal(clock) {
		t.Errorf("UpdatedAt = %v", second.UpdatedAt)
	}
	entries, _ := os.ReadDir(s.Dir())
	if len(entries) != 1 || entries[0].Name() != "math.cue.json" {
		t.Errorf("rename should keep the original file, got %v", entries)
	}
}

func TestSaveRejectsInvalid(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save(cue.Cue{Name: ""}); err == nil {
		t.Error("expected validation error")
	}
	entries, _ := os.ReadDir(s.Dir())
	if len(entries) != 0 {
		t.Errorf("nothing should be written, got %v", entries)
	}
}

func TestSlugCollisions(t *testing.T) {
	s := newStore(t)
	for range 3 {
		if _, err := s.Save(sample("Study!")); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"study.cue.json", "study-2.cue.json", "study-3.cue.json"} {
		if _, err := os.Stat(filepath.Join(s.Dir(), f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
}

func TestHandWrittenFiles(t *testing.T) {
	s := newStore(t)
	good := `{"schemaVersion":1,"name":"Hand made","resources":[{"type":"url","url":"https://a.b"}]}`
	os.WriteFile(filepath.Join(s.Dir(), "hand.cue.json"), []byte(good), 0o644)
	os.WriteFile(filepath.Join(s.Dir(), "broken.cue.json"), []byte(`{nope`), 0o644)
	os.WriteFile(filepath.Join(s.Dir(), "notes.txt"), []byte(`ignored`), 0o644)

	list, bad, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || len(bad) != 1 || bad[0].File != "broken.cue.json" {
		t.Fatalf("list=%v bad=%v", list, bad)
	}
	// The generated id is persisted so it stays stable across scans.
	again, _, _ := s.List()
	if again[0].ID != list[0].ID {
		t.Error("id of hand-written cue is not stable")
	}
	// Saving it from the app fills in the missing creation time.
	saved, err := s.Save(list[0])
	if err != nil || saved.CreatedAt.IsZero() {
		t.Errorf("first save of hand-written cue: %+v, %v", saved.CreatedAt, err)
	}
}

func TestDeleteUsesTrash(t *testing.T) {
	s := newStore(t)
	trashed := ""
	s.Trash = func(p string) error { trashed = p; return os.Remove(p) }
	c, _ := s.Save(sample("Gone"))
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(trashed) != "gone.cue.json" {
		t.Errorf("trashed %q", trashed)
	}
	if _, err := s.Get(c.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete = %v", err)
	}
}

func TestMarkRun(t *testing.T) {
	s := newStore(t)
	c, _ := s.Save(sample("Run me"))
	at := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	if err := s.MarkRun(c.ID, at); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(c.ID)
	if !got.LastRunAt.Equal(at) {
		t.Errorf("LastRunAt = %v", got.LastRunAt)
	}
}

func TestWatchSeesExternalEdits(t *testing.T) {
	s := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changed := make(chan struct{}, 1)
	if err := s.Watch(ctx, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(s.Dir(), "x.cue.json"), []byte(`{}`), 0o644)
	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("no change event")
	}
}
