package settings

import (
	"path/filepath"
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/ai"
)

func TestLoadSave(t *testing.T) {
	f := Open(filepath.Join(t.TempDir(), "nested", "settings.json"))
	s, err := f.Load()
	if err != nil || s.AI.Provider != "" {
		t.Fatalf("defaults = %+v, %v", s, err)
	}
	s.AI = ai.Config{Provider: ai.ProviderOpenAI, Model: "llama3.2", BaseURL: "http://localhost:11434/v1"}
	if err := f.Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := f.Load()
	if err != nil || got != s {
		t.Errorf("round trip = %+v, %v", got, err)
	}
}
