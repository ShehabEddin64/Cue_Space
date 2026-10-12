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

func TestWidgetSnapshot(t *testing.T) {
	cues := []cue.Cue{
		{ID: "a", Name: "Atlas", Icon: "🛰️", Resources: []cue.Resource{{ID: "1"}, {ID: "2", Disabled: true}}},
		{ID: "m", Name: "Math"},
	}
	snap := BuildWidgetSnapshot(cues, []string{"m"})
	if len(snap.Cues) != 2 || snap.Cues[0].ID != "m" || snap.Cues[0].Shortcut != "⌘1" {
		t.Fatalf("snapshot = %+v", snap.Cues)
	}
	if snap.Cues[1].Items != 1 || snap.Cues[0].Icon != "◌" {
		t.Errorf("items/icon = %+v", snap.Cues)
	}
}
