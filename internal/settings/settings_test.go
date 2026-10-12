package settings

import (
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/ai"
)

func TestLoadSave(t *testing.T) {
	f := Open(filepath.Join(t.TempDir(), "nested", "settings.json"))
	s, err := f.Load()
	if err != nil || s.AI.Provider != "" {
		t.Fatalf("defaults = %+v, %v", s, err)
	}
	if s.Prefs.Appearance != "system" || len(s.Prefs.Home.Sections) != 3 || s.Prefs.Home.RecentCount != 6 {
		t.Errorf("default prefs = %+v", s.Prefs)
	}
	s.AI = ai.Config{Provider: ai.ProviderOpenAI, Model: "llama3.2", BaseURL: "http://localhost:11434/v1"}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := f.Load()
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Errorf("round trip = %+v, %v", got, err)
	}
}

func TestNormalizePrefs(t *testing.T) {
	p := Prefs{
		Appearance: "purple",
		Pinned:     []string{"a", "", "a", "b"},
		Home: HomePrefs{
			Sections:    []HomeSection{{"recent", false}, {"bogus", true}, {"recent", true}},
			RecentCount: 99,
		},
	}
	p.Normalize()
	if p.Appearance != "system" {
		t.Errorf("appearance = %q", p.Appearance)
	}
	if !reflect.DeepEqual(p.Pinned, []string{"a", "b"}) {
		t.Errorf("pinned = %v", p.Pinned)
	}
	want := []HomeSection{{"recent", false}, {"suggestions", true}, {"pinned", true}}
	if !reflect.DeepEqual(p.Home.Sections, want) {
		t.Errorf("sections = %v", p.Home.Sections)
	}
	if p.Home.RecentCount != 6 {
		t.Errorf("recentCount = %d", p.Home.RecentCount)
	}
}

func TestUpdateIsAtomic(t *testing.T) {
	f := Open(filepath.Join(t.TempDir(), "settings.json"))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			f.Update(func(s *Settings) { s.AI.Model = "m" })
		}()
		go func() {
			defer wg.Done()
			f.Update(func(s *Settings) { s.Prefs.Onboarded = true })
		}()
	}
	wg.Wait()
	s, _ := f.Load()
	if s.AI.Model != "m" || !s.Prefs.Onboarded {
		t.Errorf("lost an update: %+v", s)
	}
}
