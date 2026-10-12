//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreGraphics
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

// Private, read-only: which desktop (Space) is active. Used only to notice
// when opening an app jumped to another desktop.
extern int CGSMainConnectionID(void);
extern uint64_t CGSGetActiveSpace(int cid);

static uint64_t cs_active_space(void) {
	return CGSGetActiveSpace(CGSMainConnectionID());
}

static int cs_ax_trusted(int prompt) {
	NSDictionary *opts = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @(prompt != 0)};
	return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts) ? 1 : 0;
}

static int cs_pid_for_bundle(const char *bid) {
	NSArray *apps = [NSRunningApplication runningApplicationsWithBundleIdentifier:[NSString stringWithUTF8String:bid]];
	for (NSRunningApplication *a in apps) {
		if (!a.terminated) return a.processIdentifier;
	}
	return 0;
}

static int cs_pid_for_name(const char *name) {
	NSString *n = [NSString stringWithUTF8String:name];
	for (NSRunningApplication *a in [[NSWorkspace sharedWorkspace] runningApplications]) {
		if ([a.localizedName caseInsensitiveCompare:n] == NSOrderedSame) return a.processIdentifier;
	}
	return 0;
}

static char *cs_bundle_of_app_url(NSURL *app) {
	if (!app) return NULL;
	NSString *bid = [NSBundle bundleWithURL:app].bundleIdentifier;
	return bid ? strdup(bid.UTF8String) : NULL;
}

// The app that opens a URL (the default browser for web links).
static char *cs_app_for_url(const char *url) {
	NSURL *u = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
	if (!u) return NULL;
	return cs_bundle_of_app_url([[NSWorkspace sharedWorkspace] URLForApplicationToOpenURL:u]);
}

// The app that opens a file or folder by default.
static char *cs_app_for_path(const char *path) {
	NSURL *u = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
	return cs_bundle_of_app_url([[NSWorkspace sharedWorkspace] URLForApplicationToOpenURL:u]);
}

// Windows of an app, retained. The caller releases with CFRelease.
static CFArrayRef cs_windows(int pid) {
	if (pid <= 0) return NULL;
	AXUIElementRef app = AXUIElementCreateApplication(pid);
	CFTypeRef windows = NULL;
	AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, &windows);
	CFRelease(app);
	return (CFArrayRef)windows;
}

static int cs_contains(CFArrayRef list, CFTypeRef el) {
	if (!list) return 0;
	for (CFIndex i = 0; i < CFArrayGetCount(list); i++) {
		if (CFEqual(CFArrayGetValueAtIndex(list, i), el)) return 1;
	}
	return 0;
}

static int cs_is_standard(AXUIElementRef w) {
	CFTypeRef sub = NULL;
	int ok = 1;
	if (AXUIElementCopyAttributeValue(w, kAXSubroleAttribute, &sub) == kAXErrorSuccess && sub) {
		ok = CFEqual(sub, kAXStandardWindowSubrole) || CFEqual(sub, CFSTR("AXDialog")) ? 1 : 0;
		CFRelease(sub);
	}
	return ok;
}

// A window of pid that wasn't in `before`. With fallback set, the app's
// focused window if nothing new appeared. Retained; NULL if none.
static AXUIElementRef cs_new_window(int pid, CFArrayRef before, int fallback) {
	CFArrayRef now = cs_windows(pid);
	AXUIElementRef found = NULL;
	if (now) {
		for (CFIndex i = 0; i < CFArrayGetCount(now); i++) {
			AXUIElementRef w = (AXUIElementRef)CFArrayGetValueAtIndex(now, i);
			if (!cs_contains(before, w) && cs_is_standard(w)) {
				found = (AXUIElementRef)CFRetain(w);
				break;
			}
		}
	}
	if (!found && fallback) {
		AXUIElementRef app = AXUIElementCreateApplication(pid);
		CFTypeRef w = NULL;
		if (AXUIElementCopyAttributeValue(app, kAXFocusedWindowAttribute, &w) != kAXErrorSuccess || !w) {
			AXUIElementCopyAttributeValue(app, kAXMainWindowAttribute, &w);
		}
		if (!w && now && CFArrayGetCount(now) > 0) {
			w = CFRetain(CFArrayGetValueAtIndex(now, 0));
		}
		CFRelease(app);
		found = (AXUIElementRef)w;
	}
	if (now) CFRelease(now);
	return found;
}

static int cs_equal(CFTypeRef a, CFTypeRef b) {
	return (a && b && CFEqual(a, b)) ? 1 : 0;
}

static CFTypeRef cs_retain(CFTypeRef ref) {
	return ref ? CFRetain(ref) : NULL;
}

static void cs_release(CFTypeRef ref) {
	if (ref) CFRelease(ref);
}

// The main screen's usable area (without menu bar and Dock), in the
// top-left-origin coordinates Accessibility uses.
static void cs_visible_frame(double *x, double *y, double *w, double *h) {
	NSScreen *primary = NSScreen.screens.firstObject;
	NSScreen *screen = NSScreen.mainScreen ?: primary;
	NSRect v = screen.visibleFrame;
	*x = v.origin.x;
	*y = primary.frame.size.height - (v.origin.y + v.size.height);
	*w = v.size.width;
	*h = v.size.height;
}

static int cs_set_frame(AXUIElementRef win, double x, double y, double w, double h) {
	CGPoint p = CGPointMake(x, y);
	CGSize s = CGSizeMake(w, h);
	AXValueRef pv = AXValueCreate(kAXValueCGPointType, &p);
	AXValueRef sv = AXValueCreate(kAXValueCGSizeType, &s);
	// Size, position, size: some apps clamp the size to the old position.
	AXUIElementSetAttributeValue(win, kAXSizeAttribute, sv);
	AXError e = AXUIElementSetAttributeValue(win, kAXPositionAttribute, pv);
	AXUIElementSetAttributeValue(win, kAXSizeAttribute, sv);
	CFRelease(pv);
	CFRelease(sv);
	AXUIElementPerformAction(win, kAXRaiseAction);
	return e == kAXErrorSuccess ? 1 : 0;
}

static int cs_get_frame(AXUIElementRef win, double *x, double *y, double *w, double *h) {
	CFTypeRef pv = NULL, sv = NULL;
	CGPoint p;
	CGSize s;
	int ok = 0;
	if (AXUIElementCopyAttributeValue(win, kAXPositionAttribute, &pv) == kAXErrorSuccess && pv &&
	    AXUIElementCopyAttributeValue(win, kAXSizeAttribute, &sv) == kAXErrorSuccess && sv &&
	    AXValueGetValue((AXValueRef)pv, kAXValueCGPointType, &p) && AXValueGetValue((AXValueRef)sv, kAXValueCGSizeType, &s)) {
		*x = p.x; *y = p.y; *w = s.width; *h = s.height;
		ok = 1;
	}
	if (pv) CFRelease(pv);
	if (sv) CFRelease(sv);
	return ok;
}

static int cs_set_fullscreen(AXUIElementRef win) {
	AXUIElementPerformAction(win, kAXRaiseAction);
	return AXUIElementSetAttributeValue(win, CFSTR("AXFullScreen"), kCFBooleanTrue) == kAXErrorSuccess ? 1 : 0;
}

// ---- Mission Control -------------------------------------------------------

static CFArrayRef cs_children(AXUIElementRef el) {
	CFTypeRef c = NULL;
	AXUIElementCopyAttributeValue(el, kAXChildrenAttribute, &c);
	return (CFArrayRef)c;
}

static int cs_role_is(AXUIElementRef el, CFStringRef role) {
	CFTypeRef r = NULL;
	int ok = 0;
	if (AXUIElementCopyAttributeValue(el, kAXRoleAttribute, &r) == kAXErrorSuccess && r) {
		ok = CFEqual(r, role) ? 1 : 0;
		CFRelease(r);
	}
	return ok;
}

// Finds Mission Control's spaces bar: the group holding the list of desktops
// and the "add desktop" button. Matched by structure, not by (localized)
// labels. Returns 1 and retained refs on success.
static int cs_find_spaces_bar(AXUIElementRef el, int depth, AXUIElementRef *addButton, AXUIElementRef *list) {
	if (depth > 7) return 0;
	CFArrayRef kids = cs_children(el);
	if (!kids) return 0;
	AXUIElementRef button = NULL, l = NULL;
	for (CFIndex i = 0; i < CFArrayGetCount(kids); i++) {
		AXUIElementRef k = (AXUIElementRef)CFArrayGetValueAtIndex(kids, i);
		if (!button && cs_role_is(k, kAXButtonRole)) button = k;
		if (!l && cs_role_is(k, kAXListRole)) l = k;
	}
	if (button && l) {
		*addButton = (AXUIElementRef)CFRetain(button);
		*list = (AXUIElementRef)CFRetain(l);
		CFRelease(kids);
		return 1;
	}
	for (CFIndex i = 0; i < CFArrayGetCount(kids); i++) {
		if (cs_find_spaces_bar((AXUIElementRef)CFArrayGetValueAtIndex(kids, i), depth + 1, addButton, list)) {
			CFRelease(kids);
			return 1;
		}
	}
	CFRelease(kids);
	return 0;
}

static CFIndex cs_count(AXUIElementRef el) {
	CFArrayRef kids = cs_children(el);
	CFIndex n = kids ? CFArrayGetCount(kids) : 0;
	if (kids) CFRelease(kids);
	return n;
}

// Step 1: with Mission Control open, find the spaces bar. 0 if not (yet) visible.
static int cs_mc_find(AXUIElementRef *addButton, AXUIElementRef *list) {
	NSArray *docks = [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.apple.dock"];
	if (docks.count == 0) return 0;
	AXUIElementRef dock = AXUIElementCreateApplication(((NSRunningApplication *)docks.firstObject).processIdentifier);
	int ok = cs_find_spaces_bar(dock, 0, addButton, list);
	CFRelease(dock);
	return ok;
}

static int cs_press(AXUIElementRef el) {
	return AXUIElementPerformAction(el, kAXPressAction) == kAXErrorSuccess ? 1 : 0;
}

// Presses the last desktop in the list, which switches to it and closes
// Mission Control.
static int cs_mc_press_last(AXUIElementRef list) {
	CFArrayRef kids = cs_children(list);
	if (!kids || CFArrayGetCount(kids) == 0) {
		if (kids) CFRelease(kids);
		return 0;
	}
	int ok = cs_press((AXUIElementRef)CFArrayGetValueAtIndex(kids, CFArrayGetCount(kids) - 1));
	CFRelease(kids);
	return ok;
}

// A text dump of Dock's accessibility tree, for diagnosing Mission Control
// changes in new macOS versions.
static void cs_dump(AXUIElementRef el, int depth, NSMutableString *out) {
	if (depth > 6) return;
	CFTypeRef role = NULL, desc = NULL;
	AXUIElementCopyAttributeValue(el, kAXRoleAttribute, &role);
	AXUIElementCopyAttributeValue(el, kAXDescriptionAttribute, &desc);
	NSString *r = role ? (__bridge NSString *)role : @"?";
	NSString *d = desc && CFGetTypeID(desc) == CFStringGetTypeID() ? (__bridge NSString *)desc : @"";
	[out appendFormat:@"%*s%@ %@\n", depth * 2, "", r, d];
	if (role) CFRelease(role);
	if (desc) CFRelease(desc);
	CFArrayRef kids = cs_children(el);
	if (!kids) return;
	for (CFIndex i = 0; i < CFArrayGetCount(kids) && i < 40; i++) {
		cs_dump((AXUIElementRef)CFArrayGetValueAtIndex(kids, i), depth + 1, out);
	}
	CFRelease(kids);
}

static char *cs_dump_dock(void) {
	NSArray *docks = [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.apple.dock"];
	if (docks.count == 0) return strdup("no Dock");
	AXUIElementRef dock = AXUIElementCreateApplication(((NSRunningApplication *)docks.firstObject).processIdentifier);
	NSMutableString *out = [NSMutableString string];
	cs_dump(dock, 0, out);
	CFRelease(dock);
	return strdup(out.UTF8String);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

// AccessibilityTrusted reports whether Cue Space may move other apps'
// windows. With prompt, macOS shows its permission dialog if it may not.
func AccessibilityTrusted(prompt bool) bool {
	p := 0
	if prompt {
		p = 1
	}
	return C.cs_ax_trusted(C.int(p)) == 1
}

// OpenAccessibilitySettings shows the Accessibility pane in System Settings.
func OpenAccessibilitySettings() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}

// Arranger opens resources and places their windows on desktops.
type Arranger struct {
	Launcher *Launcher
	// LogDir, when set, receives a Dock accessibility dump if Mission Control
	// can't be driven (useful after macOS updates).
	LogDir string

	// Windows already placed in the current run, so one window is never
	// moved into two slots.
	placed []C.CFTypeRef
}

// BeginRun forgets the windows placed by the previous run.
func (a *Arranger) BeginRun() {
	for _, w := range a.placed {
		C.cs_release(w)
	}
	a.placed = nil
}

func (a *Arranger) alreadyPlaced(win C.AXUIElementRef) bool {
	for _, w := range a.placed {
		if C.cs_equal(w, C.CFTypeRef(win)) == 1 {
			return true
		}
	}
	return false
}

func (a *Arranger) Ready() bool { return AccessibilityTrusted(false) }

// NewDesktop adds a desktop in Mission Control and switches to it.
func (a *Arranger) NewDesktop(ctx context.Context) error {
	if err := exec.CommandContext(ctx, "open", "-b", "com.apple.exposelauncher").Run(); err != nil {
		return fmt.Errorf("couldn't open Mission Control: %w", err)
	}
	var add, list C.AXUIElementRef
	if !poll(ctx, 3*time.Second, func() bool { return C.cs_mc_find(&add, &list) == 1 }) {
		a.dumpDock("mission-control-not-found")
		a.closeMissionControl()
		return errors.New("Mission Control's desktop bar didn't appear")
	}
	defer C.cs_release(C.CFTypeRef(add))
	defer C.cs_release(C.CFTypeRef(list))

	before := C.cs_count(list)
	if C.cs_press(add) != 1 {
		a.closeMissionControl()
		return errors.New("couldn't press Mission Control's add desktop button")
	}
	if !poll(ctx, 2*time.Second, func() bool { return C.cs_count(list) > before }) {
		a.dumpDock("desktop-not-added")
		a.closeMissionControl()
		return errors.New("Mission Control didn't add a desktop")
	}
	sleep(ctx, 350*time.Millisecond) // let the new thumbnail settle
	space := C.cs_active_space()
	if C.cs_mc_press_last(list) != 1 {
		a.closeMissionControl()
		return errors.New("couldn't switch to the new desktop")
	}
	if !poll(ctx, 2500*time.Millisecond, func() bool { return C.cs_active_space() != space }) {
		return errors.New("macOS didn't switch to the new desktop")
	}
	sleep(ctx, 600*time.Millisecond) // the switch animation
	return nil
}

func (a *Arranger) closeMissionControl() {
	_ = exec.Command("open", "-b", "com.apple.exposelauncher").Run() // toggles it closed
	time.Sleep(500 * time.Millisecond)
}

func (a *Arranger) dumpDock(name string) {
	if a.LogDir == "" {
		return
	}
	dump := C.cs_dump_dock()
	defer C.free(unsafe.Pointer(dump))
	_ = os.MkdirAll(a.LogDir, 0o755)
	_ = os.WriteFile(filepath.Join(a.LogDir, name+".txt"), []byte(C.GoString(dump)), 0o644)
}

// Place opens rsc (path is its expanded path, for files and folders) and moves
// its window into slot, or into full screen on its own desktop. The returned
// note explains anything that didn't go as planned.
func (a *Arranger) Place(ctx context.Context, rsc cue.Resource, path string, slot cue.Rect, fullScreen bool) (string, error) {
	bundle, name := targetApp(rsc, path)
	pid := 0
	if bundle != "" {
		pid = pidForBundle(bundle)
	} else if name != "" {
		pid = pidForName(name)
	}
	wasRunning := pid > 0
	var before C.CFArrayRef
	if pid > 0 {
		before = C.cs_windows(C.int(pid))
		defer C.cs_release(C.CFTypeRef(before))
	}
	space := C.cs_active_space()

	openNote, err := a.open(ctx, rsc, path, bundle, wasRunning)
	if err != nil {
		return "", err
	}

	if pid == 0 && !poll(ctx, 12*time.Second, func() bool {
		if bundle != "" {
			pid = pidForBundle(bundle)
		} else {
			pid = pidForName(name)
		}
		return pid > 0
	}) {
		return "opened, but couldn't find its window to arrange", nil
	}

	var win C.AXUIElementRef
	// Prefer a brand-new window; fall back to the app's focused one.
	poll(ctx, 6*time.Second, func() bool {
		win = C.cs_new_window(C.int(pid), before, 0)
		return win != 0
	})
	if win == 0 {
		win = C.cs_new_window(C.int(pid), before, 1)
	}
	if win == 0 {
		return "opened, but it has no window to arrange", nil
	}
	defer C.cs_release(C.CFTypeRef(win))

	if a.alreadyPlaced(win) {
		return joinNote(openNote, "opened in a window that already holds an earlier item, so it wasn't moved"), nil
	}
	a.placed = append(a.placed, C.cs_retain(C.CFTypeRef(win)))

	note := openNote
	// Only an app that was already running can have pulled us to another
	// desktop; a fresh launch can switch away from a full-screen space too.
	if wasRunning && C.cs_active_space() != space {
		note = "was already open on another desktop, so it was arranged there"
	}
	// Let the window finish its opening animation before moving it.
	sleep(ctx, 250*time.Millisecond)

	if fullScreen {
		if C.cs_set_fullscreen(win) != 1 {
			return joinNote(note, "this app doesn't support full screen"), nil
		}
		sleep(ctx, 1200*time.Millisecond) // the full-screen animation
		return note, nil
	}

	var x, y, w, h C.double
	C.cs_visible_frame(&x, &y, &w, &h)
	const gap = 4.0
	fx := float64(x) + slot.X*float64(w) + gap
	fy := float64(y) + slot.Y*float64(h) + gap
	fw := slot.W*float64(w) - 2*gap
	fh := slot.H*float64(h) - 2*gap
	// Apps may still be animating or restoring their window, which can undo
	// a move; set the frame until it sticks.
	placed := poll(ctx, 2500*time.Millisecond, func() bool {
		C.cs_set_frame(win, C.double(fx), C.double(fy), C.double(fw), C.double(fh))
		time.Sleep(120 * time.Millisecond)
		var gx, gy, gw, gh C.double
		if C.cs_get_frame(win, &gx, &gy, &gw, &gh) != 1 {
			return false
		}
		return near(float64(gx), fx) && near(float64(gy), fy) && near(float64(gw), fw) && near(float64(gh), fh)
	})
	if !placed {
		var gx, gy, gw, gh C.double
		C.cs_get_frame(win, &gx, &gy, &gw, &gh)
		if !near(float64(gx), fx) || !near(float64(gy), fy) {
			return joinNote(note, "this window couldn't be moved into place"), nil
		}
		// Position held but size didn't: the app has a minimum or fixed size.
		return joinNote(note, "this app keeps its own window size"), nil
	}
	return note, nil
}

func near(a, b float64) bool {
	d := a - b
	return d > -3 && d < 3
}

// open opens rsc so that it gets a window of its own. Links normally open
// as tabs in the browser's front window, so ask the browser for a new window.
func (a *Arranger) open(ctx context.Context, rsc cue.Resource, path, browser string, browserRunning bool) (string, error) {
	switch rsc.Type {
	case cue.TypeURL:
		if !browserRunning {
			// A browser launched with a link opens it in its first window;
			// asking it for a new window too would leave an extra one.
			return "", a.Launcher.OpenURL(ctx, rsc.URL)
		}
		return a.openURLInNewWindow(ctx, rsc.URL, browser)
	case cue.TypeApp:
		return "", a.Launcher.OpenApp(ctx, *rsc.App)
	default:
		return "", a.Launcher.OpenPath(ctx, path, rsc.OpenWith)
	}
}

var chromiumBrowsers = map[string]bool{
	"com.google.Chrome": true, "com.google.Chrome.beta": true, "com.microsoft.edgemac": true,
	"com.brave.Browser": true, "org.chromium.Chromium": true, "com.vivaldi.Vivaldi": true,
	"com.operasoftware.Opera": true,
}

func (a *Arranger) openURLInNewWindow(ctx context.Context, link, browser string) (string, error) {
	switch {
	case browser == "com.apple.Safari":
		// Needs Automation permission (asked once). Falls back to a tab.
		script := fmt.Sprintf(`tell application id "com.apple.Safari"
	make new document with properties {URL:"%s"}
	activate
end tell`, appleScriptString(link))
		if err := exec.CommandContext(ctx, "osascript", "-e", script).Run(); err == nil {
			return "", nil
		}
		return "opened as a tab: allow Cue Space to control Safari (System Settings › Privacy & Security › Automation) to give each link its own window",
			a.Launcher.OpenURL(ctx, link)
	case chromiumBrowsers[browser]:
		return "", exec.CommandContext(ctx, "open", "-na", "-b", browser, "--args", "--new-window", link).Run()
	case browser == "org.mozilla.firefox":
		return "", exec.CommandContext(ctx, "open", "-na", "-b", browser, "--args", "-new-window", link).Run()
	}
	return "", a.Launcher.OpenURL(ctx, link)
}

func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// targetApp is the app whose window will hold rsc.
func targetApp(rsc cue.Resource, path string) (bundle, name string) {
	cstr := func(p *C.char) string {
		if p == nil {
			return ""
		}
		defer C.free(unsafe.Pointer(p))
		return C.GoString(p)
	}
	switch rsc.Type {
	case cue.TypeApp:
		if rsc.App != nil {
			return rsc.App.BundleID, rsc.App.Name
		}
	case cue.TypeURL:
		u := C.CString(rsc.URL)
		defer C.free(unsafe.Pointer(u))
		return cstr(C.cs_app_for_url(u)), ""
	case cue.TypeFile, cue.TypeFolder:
		if !rsc.OpenWith.IsZero() {
			return rsc.OpenWith.BundleID, rsc.OpenWith.Name
		}
		p := C.CString(path)
		defer C.free(unsafe.Pointer(p))
		return cstr(C.cs_app_for_path(p)), ""
	}
	return "", ""
}

func pidForBundle(bundle string) int {
	b := C.CString(bundle)
	defer C.free(unsafe.Pointer(b))
	return int(C.cs_pid_for_bundle(b))
}

func pidForName(name string) int {
	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))
	return int(C.cs_pid_for_name(n))
}

func joinNote(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

func poll(ctx context.Context, limit time.Duration, ok func() bool) bool {
	deadline := time.Now().Add(limit)
	for {
		if ok() {
			return true
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return false
		}
		time.Sleep(120 * time.Millisecond)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
