package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
)

// WidgetSnapshot is what the desktop widget shows. It holds only names,
// icons and counts: never paths or URLs, so the widget can't see what a Cue
// opens.
type WidgetSnapshot struct {
	UpdatedAt time.Time    `json:"updatedAt"`
	Cues      []WidgetItem `json:"cues"`
	Labels    WidgetLabels `json:"labels"`
}

// WidgetLabels are the widget's words in the app's language.
type WidgetLabels struct {
	NoCues      string `json:"noCues"`
	DescribeOne string `json:"describeOne"`
	StartCue    string `json:"startCue"`
	Cues        string `json:"cues"`
}

type WidgetItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Items    int    `json:"items"`
	Shortcut string `json:"shortcut,omitempty"`
	// Detail is the item count as text, e.g. "3 items".
	Detail string `json:"detail"`
}

// WidgetWriter keeps widget.json in the data folder up to date and asks
// WidgetKit to refresh. Calls are debounced.
type WidgetWriter struct {
	Dir string
	// Reload is the helper that refreshes widget timelines; optional.
	Reload string

	mu    sync.Mutex
	timer *time.Timer
	last  []byte
}

// BuildWidgetSnapshot orders Cues like the ⌘1–⌘9 shortcuts.
func BuildWidgetSnapshot(cues []cue.Cue, pinned []string) WidgetSnapshot {
	ordered := ShortcutOrder(cues, pinned)
	snap := WidgetSnapshot{
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Cues:      []WidgetItem{},
		Labels: WidgetLabels{
			NoCues:      i18n.T("No Cues yet"),
			DescribeOne: i18n.T("Describe one in Cue Space."),
			StartCue:    i18n.T("Start Cue"),
			Cues:        i18n.T("Cues"),
		},
	}
	for i, c := range ordered {
		if i == 12 {
			break
		}
		item := WidgetItem{ID: c.ID, Name: c.Name, Icon: c.Icon}
		for _, r := range c.Resources {
			if !r.Disabled {
				item.Items++
			}
		}
		if item.Items == 1 {
			item.Detail = i18n.T("1 item")
		} else {
			item.Detail = i18n.T("%d items", item.Items)
		}
		if item.Icon == "" {
			item.Icon = "◌"
		}
		if i < 9 {
			item.Shortcut = fmt.Sprintf("⌘%d", i+1)
		}
		snap.Cues = append(snap.Cues, item)
	}
	return snap
}

// Update schedules a snapshot write.
func (w *WidgetWriter) Update(cues []cue.Cue, pinned []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(300*time.Millisecond, func() { w.write(cues, pinned) })
}

func (w *WidgetWriter) write(cues []cue.Cue, pinned []string) {
	snap := BuildWidgetSnapshot(cues, pinned)
	compare := snap
	compare.UpdatedAt = time.Time{}
	key, _ := json.Marshal(compare)

	w.mu.Lock()
	unchanged := string(key) == string(w.last)
	w.last = key
	w.mu.Unlock()
	if unchanged {
		return
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(w.Dir, "widget.json")
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) != nil || os.Rename(tmp, path) != nil {
		return
	}
	if w.Reload != "" {
		if _, err := os.Stat(w.Reload); err == nil {
			_ = exec.Command(w.Reload).Run()
		}
	}
}
