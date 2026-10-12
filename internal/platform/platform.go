// Package platform holds everything that talks to the operating system:
// opening things, finding installed apps, and the Trash. Each OS gets its own
// build-tagged file; the rest of Cue Space stays portable.
package platform

import "errors"

// App is an installed application.
type App struct {
	Name     string `json:"name"`
	BundleID string `json:"bundleId"`
	Path     string `json:"path"`
}

var ErrUnsupported = errors.New("not supported on this platform yet")
