//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func writeApp(t *testing.T, dir, name, plistBody string) {
	t.Helper()
	contents := filepath.Join(dir, name+".app", "Contents")
	if err := os.MkdirAll(contents, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>` + plistBody + `</dict></plist>`
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledApps(t *testing.T) {
	root := t.TempDir()
	writeApp(t, root, "Zed", `<key>CFBundleIdentifier</key><string>dev.zed.Zed</string>`)
	writeApp(t, root, "Cursor", `<key>CFBundleIdentifier</key><string>com.cursor</string><key>CFBundleName</key><string>Electron</string>`)
	writeApp(t, root, "Helper", `<key>CFBundleIdentifier</key><string>x.helper</string><key>LSBackgroundOnly</key><true/>`)
	writeApp(t, root, "NoID", ``)
	utils := filepath.Join(root, "Utilities")
	writeApp(t, utils, "Terminal", `<key>CFBundleIdentifier</key><string>com.apple.Terminal</string>`)
	writeApp(t, utils, "Duplicate Zed", `<key>CFBundleIdentifier</key><string>dev.zed.Zed</string>`)
	// Too deep: only one level of subfolders is scanned.
	writeApp(t, filepath.Join(utils, "Deep"), "Hidden", `<key>CFBundleIdentifier</key><string>x.deep</string>`)

	apps := InstalledApps([]string{root, filepath.Join(root, "does-not-exist")})
	var names []string
	for _, a := range apps {
		names = append(names, a.Name+"="+a.BundleID)
	}
	want := []string{"Cursor=com.cursor", "Terminal=com.apple.Terminal", "Zed=dev.zed.Zed"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("got %v, want %v", names, want)
			break
		}
	}
}

func TestMoveToTrashMissingFile(t *testing.T) {
	if err := MoveToTrash(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestCleanOpenError(t *testing.T) {
	msg := "Unable to find application named 'Nope'\n\n2026-10-11 LSOpenURLsWithRole() failed"
	if got := cleanOpenError(msg); got != "app is not installed" {
		t.Errorf("got %q", got)
	}
}

func TestAppleScriptString(t *testing.T) {
	got := appleScriptString(`https://a.b/?q="x"&p=\y`)
	want := `https://a.b/?q=\"x\"&p=\\y`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
