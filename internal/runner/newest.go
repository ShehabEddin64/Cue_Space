package runner

import (
	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Files still being written by a browser or another app.
var partialExtensions = map[string]bool{"download": true, "crdownload": true, "part": true, "partial": true, "tmp": true}

// NewestFile returns the most recently added or changed file directly inside
// dir, optionally limited to the given extensions (lowercase, no dots).
// Hidden files, plain subfolders and unfinished downloads are ignored;
// packages such as .key or .app count as files.
func NewestFile(dir string, exts []string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	allowed := map[string]bool{}
	for _, e := range exts {
		allowed[e] = true
	}
	var best string
	var bestTime time.Time
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") || name == "Icon\r" {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		if partialExtensions[ext] || (len(allowed) > 0 && !allowed[ext]) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.IsDir() && ext == "" {
			continue // a plain folder, not a package
		}
		if t := addedOrChanged(info); best == "" || t.After(bestTime) {
			best, bestTime = name, t
		}
	}
	if best == "" {
		if len(exts) > 0 {
			return "", i18n.Errorf("no %s files in %s", strings.ToUpper(strings.Join(exts, "/")), filepath.Base(dir))
		}
		return "", i18n.Errorf("no files in %s", filepath.Base(dir))
	}
	return filepath.Join(dir, best), nil
}

// addedOrChanged is the later of the modification time and the status
// change time, which moves when a file is moved or downloaded into a folder.
func addedOrChanged(info fs.FileInfo) time.Time {
	t := info.ModTime()
	if c := changeTime(info); c.After(t) {
		return c
	}
	return t
}
