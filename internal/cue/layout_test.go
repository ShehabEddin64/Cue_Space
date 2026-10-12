package cue

import (
	"reflect"
	"testing"
)

func TestLayoutParseAndNormalize(t *testing.T) {
	doc := `{"schemaVersion":1,"name":"Atlas","resources":[
	  {"id":"a","type":"app","app":{"name":"Calculator"}},
	  {"id":"b","type":"url","url":"https://a.b"},
	  {"id":"c","type":"app","app":{"name":"TextEdit"}}],
	 "layout":{"desktops":[
	  {"arrangement":"fullscreen","slots":["a","b"],"newDesktop":true},
	  {"arrangement":"left-right","slots":["a","gone"],"newDesktop":true},
	  {"arrangement":"quarters","slots":["c"]}]}}`
	c, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	d := c.Layout.Desktops
	if !reflect.DeepEqual(d[0].Slots, []string{"a"}) || d[0].NewDesktop {
		t.Errorf("full screen desktop = %+v", d[0])
	}
	// "a" is already placed and "gone" doesn't exist.
	if !reflect.DeepEqual(d[1].Slots, []string{"", ""}) {
		t.Errorf("halves = %q", d[1].Slots)
	}
	if !reflect.DeepEqual(d[2].Slots, []string{"c", "", "", ""}) {
		t.Errorf("quarters = %q", d[2].Slots)
	}
	if d[0].ID == "" || d[0].ID == d[1].ID {
		t.Error("desktop ids not assigned")
	}
	placed := c.Layout.Placed()
	if !placed["a"] || !placed["c"] || placed["b"] {
		t.Errorf("placed = %v", placed)
	}
}

func TestLayoutRejectsUnknownArrangement(t *testing.T) {
	doc := `{"schemaVersion":1,"name":"x","resources":[],"layout":{"desktops":[{"arrangement":"spiral","slots":[]}]}}`
	if _, err := Parse([]byte(doc)); err == nil {
		t.Error("expected an error")
	}
}

func TestRemovingAResourceClearsItsSlot(t *testing.T) {
	c := Cue{Name: "x", Resources: []Resource{{ID: "a", Type: TypeURL, URL: "https://a.b"}},
		Layout: &Layout{Desktops: []Desktop{{Arrangement: ArrangeFill, Slots: []string{"a"}}}}}
	c.Normalize()
	c.Resources = nil
	c.Normalize()
	if c.Layout.Desktops[0].Slots[0] != "" {
		t.Error("slot still points at a removed resource")
	}
}

func TestArrangementSlotsCoverTheScreen(t *testing.T) {
	for _, a := range Arrangements {
		area := 0.0
		for _, r := range a.Slots {
			area += r.W * r.H
			if r.X < 0 || r.Y < 0 || r.X+r.W > 1.0001 || r.Y+r.H > 1.0001 {
				t.Errorf("%s: slot %+v leaves the screen", a.ID, r)
			}
		}
		if area < 0.999 || area > 1.001 {
			t.Errorf("%s: slots cover %.3f of the screen", a.ID, area)
		}
	}
}
