package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
)

type fakeLauncher struct {
	calls   []string
	failURL string
}

func (f *fakeLauncher) OpenURL(_ context.Context, u string) error {
	f.calls = append(f.calls, "url:"+u)
	if u == f.failURL {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeLauncher) OpenApp(_ context.Context, a cue.AppRef) error {
	f.calls = append(f.calls, "app:"+a.BundleID)
	return nil
}

func (f *fakeLauncher) OpenPath(_ context.Context, p string, with *cue.AppRef) error {
	call := "path:" + p
	if with != nil {
		call += " with " + with.Name
	}
	f.calls = append(f.calls, call)
	return nil
}

func TestRunOrderSkipAndFailures(t *testing.T) {
	home := t.TempDir()
	os.Mkdir(filepath.Join(home, "repo"), 0o755)
	os.WriteFile(filepath.Join(home, "notes.md"), nil, 0o644)

	c := cue.Cue{ID: "atlas", Resources: []cue.Resource{
		{ID: "1", Type: cue.TypeURL, URL: "https://github.com"},
		{ID: "2", Type: cue.TypeApp, App: &cue.AppRef{BundleID: "com.example.ide"}},
		{ID: "3", Type: cue.TypeFolder, Path: "~/repo", OpenWith: &cue.AppRef{Name: "Cursor"}},
		{ID: "4", Type: cue.TypeURL, URL: "http://localhost:3000", Disabled: true},
		{ID: "5", Type: cue.TypeURL, URL: "https://broken.example"},
		{ID: "6", Type: cue.TypeFile, Path: "~/missing.pdf"},
		{ID: "7", Type: cue.TypeFolder, Path: "~/notes.md"},
		{ID: "8", Type: cue.TypeFile, Path: "~/notes.md"},
	}}
	l := &fakeLauncher{failURL: "https://broken.example"}
	res := (&Runner{Launcher: l, Home: home}).Run(context.Background(), c)

	wantCalls := []string{
		"url:https://github.com",
		"app:com.example.ide",
		"path:" + filepath.Join(home, "repo") + " with Cursor",
		"url:https://broken.example",
		"path:" + filepath.Join(home, "notes.md"),
	}
	if !reflect.DeepEqual(l.calls, wantCalls) {
		t.Errorf("calls:\n got %q\nwant %q", l.calls, wantCalls)
	}
	wantStatus := []Status{StatusOK, StatusOK, StatusOK, StatusSkipped, StatusFailed, StatusFailed, StatusFailed, StatusOK}
	for i, s := range res.Steps {
		if s.Status != wantStatus[i] {
			t.Errorf("step %d: status %s (%s), want %s", i+1, s.Status, s.Error, wantStatus[i])
		}
	}
	if res.OK {
		t.Error("result should not be OK")
	}
}

func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l := &fakeLauncher{}
	res := (&Runner{Launcher: l}).Run(ctx, cue.Cue{Resources: []cue.Resource{
		{ID: "1", Type: cue.TypeURL, URL: "https://a.b"},
	}})
	if len(l.calls) != 0 || res.Steps[0].Status != StatusSkipped {
		t.Errorf("cancelled run opened things: %v %+v", l.calls, res.Steps)
	}
}

func TestExpandHome(t *testing.T) {
	cases := map[string]string{"~": "/h", "~/a/b": "/h/a/b", "/abs": "/abs", "~x": "~x"}
	for in, want := range cases {
		if got := ExpandHome(in, "/h"); got != want {
			t.Errorf("ExpandHome(%q) = %q, want %q", in, got, want)
		}
	}
}
