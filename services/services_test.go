package services

import (
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

func TestShortcutOrder(t *testing.T) {
	cues := []cue.Cue{{ID: "a", Name: "Atlas"}, {ID: "m", Name: "Math"}, {ID: "r", Name: "Research"}}
	got := ShortcutOrder(cues, []string{"r", "gone", "m", "r"})
	var ids string
	for _, c := range got {
		ids += c.ID
	}
	if ids != "rma" {
		t.Errorf("order = %q, want rma", ids)
	}
}
