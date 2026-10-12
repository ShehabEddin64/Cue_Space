package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

// touch creates files in order, a little apart, so each is newer than the last.
func touch(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		time.Sleep(15 * time.Millisecond)
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewestFile(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "old.pdf", "notes.txt", "report.PDF", "movie.mp4.crdownload", ".hidden")
	time.Sleep(15 * time.Millisecond)
	os.Mkdir(filepath.Join(dir, "Plain Folder"), 0o755)
	os.Mkdir(filepath.Join(dir, "Slides.key"), 0o755) // a package counts as a file
	cases := []struct {
		exts []string
		want string
	}{
		{nil, "Slides.key"},
		{[]string{"pdf"}, "report.PDF"},
		{[]string{"txt", "md"}, "notes.txt"},
	}
	for _, c := range cases {
		got, err := NewestFile(dir, c.exts)
		if err != nil || filepath.Base(got) != c.want {
			t.Errorf("NewestFile(%v) = %q, %v; want %s", c.exts, got, err, c.want)
		}
	}
	if _, err := NewestFile(dir, []string{"docx"}); err == nil || err.Error() != "no DOCX files in "+filepath.Base(dir) {
		t.Errorf("no match: err = %v", err)
	}
	if _, err := NewestFile(t.TempDir(), nil); err == nil {
		t.Error("empty folder: want an error")
	}
}

func TestRunOpensNewestFile(t *testing.T) {
	home := t.TempDir()
	dl := filepath.Join(home, "Downloads")
	os.Mkdir(dl, 0o755)
	touch(t, dl, "a.pdf", "b.zip", "c.pdf", "d.zip")

	c := cue.Cue{ID: "x", Resources: []cue.Resource{
		{ID: "1", Type: cue.TypeFolder, Path: "~/Downloads", Newest: true, Extensions: []string{"pdf"}, OpenWith: &cue.AppRef{Name: "Preview"}},
	}}
	l := &fakeLauncher{}
	res := (&Runner{Launcher: l, Home: home}).Run(context.Background(), c)
	if want := "path:" + filepath.Join(dl, "c.pdf") + " with Preview"; len(l.calls) != 1 || l.calls[0] != want {
		t.Fatalf("calls = %v, want %s", l.calls, want)
	}
	if s := res.Steps[0]; s.Status != StatusOK || s.Note != "Opened c.pdf" || s.Label != "Newest PDF in Downloads" {
		t.Errorf("step = %+v", s)
	}
}
