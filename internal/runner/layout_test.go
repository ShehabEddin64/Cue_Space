package runner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

type fakeArranger struct {
	ready     bool
	desktopOK bool
	calls     []string
}

func (f *fakeArranger) Ready() bool { return f.ready }

func (f *fakeArranger) NewDesktop(context.Context) error {
	f.calls = append(f.calls, "new-desktop")
	if !f.desktopOK {
		return errors.New("no Mission Control")
	}
	return nil
}

func (f *fakeArranger) Place(_ context.Context, r cue.Resource, _ string, slot cue.Rect, full bool) (string, error) {
	if full {
		f.calls = append(f.calls, "full:"+r.ID)
	} else {
		f.calls = append(f.calls, fmt.Sprintf("place:%s@%.2f,%.2f", r.ID, slot.X, slot.Y))
	}
	return "", nil
}

func layoutCue() cue.Cue {
	app := func(id string) cue.Resource {
		return cue.Resource{ID: id, Type: cue.TypeApp, App: &cue.AppRef{BundleID: "x." + id}}
	}
	c := cue.Cue{ID: "c", Name: "Atlas", Resources: []cue.Resource{app("a"), app("b"), app("c"), app("d"), app("off")},
		Layout: &cue.Layout{Desktops: []cue.Desktop{
			{ID: "1", Arrangement: cue.ArrangeFullScreen, Slots: []string{"b"}},
			{ID: "2", Arrangement: cue.ArrangeLeftRight, Slots: []string{"c", "off"}, NewDesktop: true},
			{ID: "3", Arrangement: cue.ArrangeQuarters, Slots: []string{"", "", "", ""}}, // empty: skipped
		}}}
	c.Resources[4].Disabled = true
	c.Normalize()
	return c
}

func TestRunWithLayout(t *testing.T) {
	l := &fakeLauncher{}
	a := &fakeArranger{ready: true, desktopOK: true}
	res := (&Runner{Launcher: l, Arranger: a}).Run(context.Background(), layoutCue())

	// Unplaced items open first, normally.
	if !reflect.DeepEqual(l.calls, []string{"app:x.a", "app:x.d"}) {
		t.Errorf("launcher calls = %q", l.calls)
	}
	want := []string{"full:b", "new-desktop", "place:c@0.00,0.00"}
	if !reflect.DeepEqual(a.calls, want) {
		t.Errorf("arranger calls = %q, want %q", a.calls, want)
	}
	if !res.OK || len(res.Notes) != 0 {
		t.Errorf("result = %+v", res)
	}
	notes := map[string]string{}
	for _, s := range res.Steps {
		notes[s.ResourceID] = s.Note
	}
	if notes["b"] != "Full screen on its own desktop" || !strings.Contains(notes["c"], "Left & Right, slot 1, on a new desktop") {
		t.Errorf("notes = %v", notes)
	}
}

func TestLayoutFallsBackWhenDesktopFails(t *testing.T) {
	a := &fakeArranger{ready: true, desktopOK: false}
	res := (&Runner{Launcher: &fakeLauncher{}, Arranger: a}).Run(context.Background(), layoutCue())
	if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "arranged on the current one") {
		t.Errorf("notes = %q", res.Notes)
	}
	// The window is still placed, just on this desktop.
	if a.calls[len(a.calls)-1] != "place:c@0.00,0.00" {
		t.Errorf("calls = %q", a.calls)
	}
	if !res.OK {
		t.Error("a missing desktop is a note, not a failure")
	}
}

func TestLayoutWithoutAccessibilityOpensEverything(t *testing.T) {
	l := &fakeLauncher{}
	a := &fakeArranger{ready: false}
	res := (&Runner{Launcher: l, Arranger: a}).Run(context.Background(), layoutCue())
	if len(a.calls) != 0 {
		t.Errorf("arranger used without permission: %q", a.calls)
	}
	if !reflect.DeepEqual(l.calls, []string{"app:x.a", "app:x.b", "app:x.c", "app:x.d"}) {
		t.Errorf("everything should still open: %q", l.calls)
	}
	if !res.NeedsAccessibility {
		t.Error("NeedsAccessibility not set")
	}
}
