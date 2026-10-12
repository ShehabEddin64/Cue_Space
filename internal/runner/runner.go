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

type StepResult struct {
	ResourceID string `json:"resourceId"`
	Label      string `json:"label"`
	Status     Status `json:"status"`
	Error      string `json:"error,omitempty"`
}

type Result struct {
	CueID string       `json:"cueId"`
	Steps []StepResult `json:"steps"`
	// OK is true when no step failed.
	OK bool `json:"ok"`
}

type Runner struct {
	Launcher Launcher
	// Home replaces a leading ~ in paths.
	Home string
	// StepTimeout bounds each open call. Zero means 10 seconds.
	StepTimeout time.Duration
}

// Run opens every enabled resource in order. A failing step does not stop the
// rest: the user gets as much of their context as possible plus a clear report.
func (r *Runner) Run(ctx context.Context, c cue.Cue) Result {
	res := Result{CueID: c.ID, Steps: make([]StepResult, 0, len(c.Resources)), OK: true}
	timeout := r.StepTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	for _, rsc := range c.Resources {
		step := StepResult{ResourceID: rsc.ID, Label: rsc.DisplayName(), Status: StatusOK}
		if rsc.Disabled {
			step.Status = StatusSkipped
			res.Steps = append(res.Steps, step)
			continue
		}
		if err := ctx.Err(); err != nil {
			step.Status, step.Error = StatusSkipped, "cancelled"
			res.Steps = append(res.Steps, step)
			continue
		}
		stepCtx, cancel := context.WithTimeout(ctx, timeout)
		err := r.open(stepCtx, rsc)
		cancel()
		if err != nil {
			step.Status, step.Error = StatusFailed, err.Error()
			res.OK = false
		}
		res.Steps = append(res.Steps, step)
	}
	return res
}

func (r *Runner) open(ctx context.Context, rsc cue.Resource) error {
	switch rsc.Type {
	case cue.TypeURL:
		return r.Launcher.OpenURL(ctx, rsc.URL)
	case cue.TypeApp:
		if rsc.App.IsZero() {
			return errors.New("no app set")
		}
		return r.Launcher.OpenApp(ctx, *rsc.App)
	case cue.TypeFile, cue.TypeFolder:
		path := ExpandHome(rsc.Path, r.Home)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found: %s", rsc.Type, rsc.Path)
		}
		if err != nil {
			return err
		}
		// macOS packages (.app, .key, …) are folders that behave like files, so
		// only a plain folder used as "file" is not flagged; the reverse is.
		if rsc.Type == cue.TypeFolder && !info.IsDir() {
			return fmt.Errorf("expected a folder but found a file: %s", rsc.Path)
		}
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
