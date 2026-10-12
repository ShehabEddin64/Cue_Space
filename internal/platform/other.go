//go:build !darwin

package platform

import (
	"context"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

// Launcher is a placeholder until Windows (`start`) and Linux (`xdg-open`)
// support lands.
type Launcher struct{}

func NewLauncher() *Launcher { return &Launcher{} }

func (Launcher) OpenURL(context.Context, string) error               { return ErrUnsupported }
func (Launcher) OpenApp(context.Context, cue.AppRef) error           { return ErrUnsupported }
func (Launcher) OpenPath(context.Context, string, *cue.AppRef) error { return ErrUnsupported }
func Reveal(string) error                                            { return ErrUnsupported }
func MoveToTrash(string) error                                       { return ErrUnsupported }
func AppDirs() []string                                              { return nil }
func InstalledApps([]string) []App                                   { return nil }
