//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework AppKit
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#include <stdlib.h>

// Keeps App Nap from throttling user-requested work (an AI request, starting
// a Cue) while the app is in the background.
static void *cs_begin_activity(const char *reason) {
	id a = [[NSProcessInfo processInfo] beginActivityWithOptions:NSActivityUserInitiatedAllowingIdleSystemSleep
	                                                      reason:[NSString stringWithUTF8String:reason]];
	return (__bridge_retained void *)a;
}

static void cs_end_activity(void *a) {
	if (a) [[NSProcessInfo processInfo] endActivity:(__bridge_transfer id)a];
}

// 0 follows the system, 1 is light, 2 is dark. Setting the app's appearance
// also switches the web view's prefers-color-scheme, menus and panels.
static void cs_set_appearance(int mode) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSAppearance *a = nil;
		if (mode == 1) a = [NSAppearance appearanceNamed:NSAppearanceNameAqua];
		if (mode == 2) a = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
		[NSApp setAppearance:a];
	});
}

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
	"time"
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

// SearchFiles finds files and folders under root whose name contains query,
// using Spotlight. Hidden items and anything inside a Library folder are
// skipped. At most limit paths are returned.
func SearchFiles(ctx context.Context, root, query string, limit int) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "mdfind", "-onlyin", root, "-name", query).Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("Spotlight search failed: %w", err)
	}
	var paths []string
	for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if p == "" || strings.Contains(p, "/.") || strings.Contains(p, "/Library/") {
			continue
		}
		paths = append(paths, p)
		if len(paths) == limit {
			break
		}
	}
	return paths, nil
}

// OpenCueSpaceURL hands a cuespace:// link to the app, launching it if needed.
func OpenCueSpaceURL(ctx context.Context, link string) error {
	return run(ctx, "open", "-g", link)
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

// BeginActivity tells macOS the app is doing work the person asked for, so
// App Nap doesn't slow it down in the background. Call the returned func
// when done.
func BeginActivity(reason string) func() {
	r := C.CString(reason)
	defer C.free(unsafe.Pointer(r))
	a := C.cs_begin_activity(r)
	return func() { C.cs_end_activity(a) }
}

// SetAppearance switches the whole app to "light", "dark", or back to
// following the system ("system" or anything else).
func SetAppearance(mode string) {
	switch mode {
	case "light":
		C.cs_set_appearance(1)
	case "dark":
		C.cs_set_appearance(2)
	default:
		C.cs_set_appearance(0)
	}
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
