package cue

import "fmt"

// Arrangement is how windows share one desktop, mirroring macOS's own
// Move & Resize and Fill & Arrange choices.
type Arrangement string

const (
	ArrangeFullScreen   Arrangement = "fullscreen"    // one item, native full screen on its own desktop
	ArrangeFill         Arrangement = "fill"          // one item fills the desktop
	ArrangeLeftRight    Arrangement = "left-right"    // two halves side by side
	ArrangeTopBottom    Arrangement = "top-bottom"    // two halves stacked
	ArrangeLeftQuarters Arrangement = "left-quarters" // left half, right half split in two
	ArrangeQuarters     Arrangement = "quarters"      // four quarters
	ArrangeThirds       Arrangement = "thirds"        // three columns
)

// Rect is a slot on a desktop, as fractions of the screen's usable area,
// with the origin at the top left.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// ArrangementInfo describes an arrangement for the editor and the runner.
type ArrangementInfo struct {
	ID    Arrangement `json:"id"`
	Title string      `json:"title"`
	Slots []Rect      `json:"slots"`
}

// Arrangements lists every arrangement, in the order the editor shows them.
var Arrangements = []ArrangementInfo{
	{ArrangeFullScreen, "Full Screen", []Rect{{0, 0, 1, 1}}},
	{ArrangeFill, "Fill", []Rect{{0, 0, 1, 1}}},
	{ArrangeLeftRight, "Left & Right", []Rect{{0, 0, .5, 1}, {.5, 0, .5, 1}}},
	{ArrangeTopBottom, "Top & Bottom", []Rect{{0, 0, 1, .5}, {0, .5, 1, .5}}},
	{ArrangeLeftQuarters, "Left & Quarters", []Rect{{0, 0, .5, 1}, {.5, 0, .5, .5}, {.5, .5, .5, .5}}},
	{ArrangeQuarters, "Quarters", []Rect{{0, 0, .5, .5}, {.5, 0, .5, .5}, {0, .5, .5, .5}, {.5, .5, .5, .5}}},
	{ArrangeThirds, "Thirds", []Rect{{0, 0, 1.0 / 3, 1}, {1.0 / 3, 0, 1.0 / 3, 1}, {2.0 / 3, 0, 1.0 / 3, 1}}},
}

func ArrangementByID(id Arrangement) (ArrangementInfo, bool) {
	for _, a := range Arrangements {
		if a.ID == id {
			return a, true
		}
	}
	return ArrangementInfo{}, false
}

// Desktop is one macOS desktop (Space) in a Cue's layout.
type Desktop struct {
	ID          string      `json:"id"`
	Arrangement Arrangement `json:"arrangement"`
	// Slots holds a resource id per position; "" leaves a slot empty.
	Slots []string `json:"slots"`
	// NewDesktop puts a tiled arrangement on a new desktop instead of the
	// current one. Full screen always gets its own desktop.
	NewDesktop bool `json:"newDesktop,omitempty"`
}

// Layout says where a Cue's windows go. Items not placed in any slot open
// normally on the current desktop, before the layout's desktops.
type Layout struct {
	Desktops []Desktop `json:"desktops"`
}

// Placed reports which resource ids the layout places.
func (l *Layout) Placed() map[string]bool {
	placed := map[string]bool{}
	if l == nil {
		return placed
	}
	for _, d := range l.Desktops {
		for _, id := range d.Slots {
			if id != "" {
				placed[id] = true
			}
		}
	}
	return placed
}

// normalizeLayout fits slots to their arrangement and drops references to
// resources that no longer exist or are already placed elsewhere.
func (c *Cue) normalizeLayout() {
	if c.Layout == nil {
		return
	}
	exists := map[string]bool{}
	for _, r := range c.Resources {
		exists[r.ID] = true
	}
	used := map[string]bool{}
	ids := map[string]bool{}
	for i := range c.Layout.Desktops {
		d := &c.Layout.Desktops[i]
		if d.ID == "" || ids[d.ID] {
			d.ID = NewID()
		}
		ids[d.ID] = true
		info, ok := ArrangementByID(d.Arrangement)
		if !ok {
			continue // Validate reports it
		}
		slots := make([]string, len(info.Slots))
		for j := range slots {
			if j < len(d.Slots) {
				id := d.Slots[j]
				if exists[id] && !used[id] {
					slots[j] = id
					used[id] = true
				}
			}
		}
		d.Slots = slots
		if d.Arrangement == ArrangeFullScreen {
			d.NewDesktop = false // implied
		}
	}
	if len(c.Layout.Desktops) == 0 {
		c.Layout = nil
	}
}

func (c *Cue) validateLayout() []string {
	if c.Layout == nil {
		return nil
	}
	var p []string
	for i, d := range c.Layout.Desktops {
		if _, ok := ArrangementByID(d.Arrangement); !ok {
			p = append(p, fmt.Sprintf("desktop %d: unknown arrangement %q", i+1, d.Arrangement))
		}
	}
	return p
}
