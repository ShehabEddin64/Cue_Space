// Package runner starts a Cue. It is deliberately simple: walk the enabled
// resources in order and open each one. No AI, no network, no guessing.
package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

// Launcher opens things using the operating system.
type Launcher interface {
	OpenURL(ctx context.Context, url string) error
	OpenApp(ctx context.Context, app cue.AppRef) error
	OpenPath(ctx context.Context, path string, with *cue.AppRef) error
}

type Status string

const (
	StatusOK      Status = "ok"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

// Arranger places windows on desktops. It needs Accessibility permission.
type Arranger interface {
	// Ready reports whether windows can be arranged right now.
	Ready() bool
	// NewDesktop creates a desktop (macOS Space) and switches to it.
	NewDesktop(ctx context.Context) error
	// Place opens rsc and moves its window into slot, or into full screen.
	// The note explains anything that didn't go as planned.
	Place(ctx context.Context, rsc cue.Resource, path string, slot cue.Rect, fullScreen bool) (note string, err error)
}

type StepResult struct {
	ResourceID string `json:"resourceId"`
	Label      string `json:"label"`
	Status     Status `json:"status"`
	Error      string `json:"error,omitempty"`
	// Note adds detail about where the window went, e.g. "Full screen".
	Note string `json:"note,omitempty"`
}

type Result struct {
	CueID string       `json:"cueId"`
	Steps []StepResult `json:"steps"`
	// OK is true when no step failed.
	OK bool `json:"ok"`
	// Notes are run-wide remarks, e.g. a desktop that couldn't be created.
	Notes []string `json:"notes"`
	// NeedsAccessibility is true when a layout was skipped because Cue Space
	// isn't allowed to arrange windows yet.
	NeedsAccessibility bool `json:"needsAccessibility"`
}

type Runner struct {
	Launcher Launcher
	// Arranger places windows for Cues with a layout. Optional.
	Arranger Arranger
	// Home replaces a leading ~ in paths.
	Home string
	// StepTimeout bounds each open call. Zero means 10 seconds; placed
	// windows get longer, since apps may need to launch.
	StepTimeout time.Duration
}

// Run opens every enabled resource. Items the layout doesn't place open first,
// in order, on the current desktop; then each layout desktop is set up in
// turn. A failing step does not stop the rest: the user gets as much of their
// context as possible plus a clear report.
func (r *Runner) Run(ctx context.Context, c cue.Cue) Result {
	res := Result{CueID: c.ID, Steps: make([]StepResult, 0, len(c.Resources)), OK: true, Notes: []string{}}
	timeout := r.StepTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	arrange := c.Layout != nil && len(c.Layout.Placed()) > 0
	if arrange && (r.Arranger == nil || !r.Arranger.Ready()) {
		arrange = false
		res.NeedsAccessibility = true
		res.Notes = append(res.Notes, "Windows weren’t arranged: allow Cue Space under Accessibility to use this Cue’s layout.")
	}
	placed := map[string]bool{}
	if arrange {
		placed = c.Layout.Placed()
	}

	for _, rsc := range c.Resources {
		if placed[rsc.ID] {
			continue
		}
		step := StepResult{ResourceID: rsc.ID, Label: rsc.DisplayName(), Status: StatusOK}
		switch {
		case rsc.Disabled:
			step.Status = StatusSkipped
		case ctx.Err() != nil:
			step.Status, step.Error = StatusSkipped, "cancelled"
		default:
			stepCtx, cancel := context.WithTimeout(ctx, timeout)
			if err := r.open(stepCtx, rsc); err != nil {
				step.Status, step.Error = StatusFailed, err.Error()
				res.OK = false
			}
			cancel()
		}
		res.Steps = append(res.Steps, step)
	}

	if arrange {
		r.arrange(ctx, c, &res, timeout)
	}
	return res
}

// arrange sets up each desktop of the layout.
func (r *Runner) arrange(ctx context.Context, c cue.Cue, res *Result, timeout time.Duration) {
	byID := map[string]cue.Resource{}
	for _, rsc := range c.Resources {
		byID[rsc.ID] = rsc
	}
	for i, d := range c.Layout.Desktops {
		info, ok := cue.ArrangementByID(d.Arrangement)
		if !ok {
			continue
		}
		// Which slots have something to open?
		type item struct {
			rsc  cue.Resource
			slot cue.Rect
		}
		var items []item
		for j, id := range d.Slots {
			rsc, ok := byID[id]
			if !ok || j >= len(info.Slots) {
				continue
			}
			if rsc.Disabled {
				res.Steps = append(res.Steps, StepResult{ResourceID: rsc.ID, Label: rsc.DisplayName(), Status: StatusSkipped})
				continue
			}
			items = append(items, item{rsc, info.Slots[j]})
		}
		if len(items) == 0 || ctx.Err() != nil {
			continue
		}

		where := "this desktop"
		if d.Arrangement != cue.ArrangeFullScreen && d.NewDesktop {
			deskCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := r.Arranger.NewDesktop(deskCtx)
			cancel()
			if err != nil {
				res.Notes = append(res.Notes, fmt.Sprintf("Desktop %d: couldn’t create a new desktop (%s), so it was arranged on the current one.", i+1, err))
			} else {
				where = "a new desktop"
			}
		}

		for k, it := range items {
			step := StepResult{ResourceID: it.rsc.ID, Label: it.rsc.DisplayName(), Status: StatusOK}
			path, err := r.checkTarget(it.rsc)
			if err == nil {
				stepCtx, cancel := context.WithTimeout(ctx, timeout+20*time.Second)
				var note string
				note, err = r.Arranger.Place(stepCtx, it.rsc, path, it.slot, d.Arrangement == cue.ArrangeFullScreen)
				cancel()
				step.Note = describePlacement(info, k, d.Arrangement, where)
				if note != "" {
					step.Note += " · " + note
				}
			}
			if err != nil {
				step.Status, step.Error, step.Note = StatusFailed, err.Error(), ""
				res.OK = false
			}
			res.Steps = append(res.Steps, step)
		}
	}
}

func describePlacement(info cue.ArrangementInfo, k int, a cue.Arrangement, where string) string {
	switch a {
	case cue.ArrangeFullScreen:
		return "Full screen on its own desktop"
	case cue.ArrangeFill:
		return "Fills " + where
	}
	return fmt.Sprintf("%s, slot %d, on %s", info.Title, k+1, where)
}

// checkTarget validates a resource before opening it and returns the expanded
// path for files and folders.
func (r *Runner) checkTarget(rsc cue.Resource) (string, error) {
	switch rsc.Type {
	case cue.TypeApp:
		if rsc.App.IsZero() {
			return "", errors.New("no app set")
		}
	case cue.TypeFile, cue.TypeFolder:
		path := ExpandHome(rsc.Path, r.Home)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s not found: %s", rsc.Type, rsc.Path)
		}
		if err != nil {
			return "", err
		}
		// macOS packages (.app, .key, …) are folders that behave like files, so
		// only a plain folder used as "file" is not flagged; the reverse is.
		if rsc.Type == cue.TypeFolder && !info.IsDir() {
			return "", fmt.Errorf("expected a folder but found a file: %s", rsc.Path)
		}
		return path, nil
	}
	return "", nil
}

func (r *Runner) open(ctx context.Context, rsc cue.Resource) error {
	path, err := r.checkTarget(rsc)
	if err != nil {
		return err
	}
	switch rsc.Type {
	case cue.TypeURL:
		return r.Launcher.OpenURL(ctx, rsc.URL)
	case cue.TypeApp:
		return r.Launcher.OpenApp(ctx, *rsc.App)
	case cue.TypeFile, cue.TypeFolder:
		return r.Launcher.OpenPath(ctx, path, rsc.OpenWith)
	}
	return fmt.Errorf("unknown resource type %q", rsc.Type)
}

// ExpandHome turns "~" and "~/x" into paths under home.
func ExpandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
