//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// Returns NULL on success, or an error message the caller must free.
static char *cs_trash(const char *path) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSError *err = nil;
		if ([[NSFileManager defaultManager] trashItemAtURL:url resultingItemURL:nil error:&err]) {
			return NULL;
		}
		return strdup(err.localizedDescription.UTF8String);
	}
}
*/
import "C"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unsafe"

	"howett.net/plist"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

// Launcher opens things with /usr/bin/open.
type Launcher struct{}

func NewLauncher() *Launcher { return &Launcher{} }

func (Launcher) OpenURL(ctx context.Context, url string) error {
	return run(ctx, "open", url)
}

func (Launcher) OpenApp(ctx context.Context, app cue.AppRef) error {
	if app.BundleID != "" {
		err := run(ctx, "open", "-b", app.BundleID)
		if err == nil || app.Name == "" {
			return err
		}
	}
	return run(ctx, "open", "-a", app.Name)
}

func (Launcher) OpenPath(ctx context.Context, path string, with *cue.AppRef) error {
	if with.IsZero() {
		return run(ctx, "open", path)
	}
	if with.BundleID != "" {
		err := run(ctx, "open", "-b", with.BundleID, path)
		if err == nil || with.Name == "" {
			return err
		}
	}
	return run(ctx, "open", "-a", with.Name, path)
}

// Reveal shows path selected in Finder.
func Reveal(path string) error {
	return run(context.Background(), "open", "-R", path)
}

func run(ctx context.Context, name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return errors.New(cleanOpenError(msg))
		}
		return err
	}
	return nil
}

// cleanOpenError turns `open`'s stderr into a short sentence.
func cleanOpenError(msg string) string {
	switch {
	case strings.Contains(msg, "Unable to find application named"),
		strings.Contains(msg, "LSCopyApplicationURLsForBundleIdentifier"),
		strings.Contains(msg, "-10814"):
		return "app is not installed"
	case strings.Contains(msg, "does not exist"):
		return "file does not exist"
	}
	if i := strings.IndexByte(msg, '\n'); i > 0 {
		msg = msg[:i]
	}
	return msg
}

// MoveToTrash moves path to the user's Trash, where it can be restored.
func MoveToTrash(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	if msg := C.cs_trash(cpath); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New(C.GoString(msg))
	}
	return nil
}

// AppDirs are the folders scanned for installed applications.
func AppDirs() []string {
	dirs := []string{"/Applications", "/System/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}
	return dirs
}

// InstalledApps lists .app bundles in dirs (and one level of subfolders, e.g.
// /Applications/Utilities), sorted by name and de-duplicated by bundle id.
func InstalledApps(dirs []string) []App {
	seen := map[string]bool{}
	var apps []App
	var visit func(dir string, depth int)
	visit = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		// Apps at this level win over same-id copies in subfolders.
		var subdirs []string
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if strings.HasSuffix(e.Name(), ".app") {
				a, err := readApp(p)
				if err != nil || a.BundleID == "" || seen[a.BundleID] {
					continue
				}
				seen[a.BundleID] = true
				apps = append(apps, a)
				continue
			}
			if depth < 1 && e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				subdirs = append(subdirs, p)
			}
		}
		for _, p := range subdirs {
			visit(p, depth+1)
		}
	}
	for _, d := range dirs {
		visit(d, 0)
	}
	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})
	return apps
}

func readApp(path string) (App, error) {
	data, err := os.ReadFile(filepath.Join(path, "Contents", "Info.plist"))
	if err != nil {
		return App{}, err
	}
	var info struct {
		ID          string `plist:"CFBundleIdentifier"`
		Name        string `plist:"CFBundleName"`
		DisplayName string `plist:"CFBundleDisplayName"`
		Background  any    `plist:"LSBackgroundOnly"`
	}
	if _, err := plist.Unmarshal(data, &info); err != nil {
		return App{}, fmt.Errorf("%s: %w", path, err)
	}
	if isTrue(info.Background) {
		return App{}, errors.New("background-only app")
	}
	// The bundle's file name is what people see in Finder and the Dock, so it
	// wins over CFBundleName (which is sometimes an internal name).
	name := strings.TrimSuffix(filepath.Base(path), ".app")
	if name == "" {
		name = info.DisplayName
	}
	return App{Name: name, BundleID: info.ID, Path: path}, nil
}

func isTrue(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "1" || strings.EqualFold(b, "true") || strings.EqualFold(b, "yes")
	case uint64:
		return b != 0
	case int64:
		return b != 0
	}
	return false
}
