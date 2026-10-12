package resolve

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
)

type fakeInfo struct{ dir bool }

func (f fakeInfo) Name() string       { return "x" }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return 0 }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.dir }
func (f fakeInfo) Sys() any           { return nil }

func resolver() Resolver {
	return Resolver{
		Home: "/Users/me",
		Apps: []platform.App{
			{Name: "Figma", BundleID: "com.figma.Desktop"},
			{Name: "Visual Studio Code", BundleID: "com.microsoft.VSCode"},
			{Name: "Calculator", BundleID: "com.apple.calculator"},
			{Name: "Calendar", BundleID: "com.apple.iCal"},
		},
		Stat: func(p string) (os.FileInfo, error) {
			switch p {
			case "/Users/me/Notes":
				return fakeInfo{dir: true}, nil
			case "/Users/me/spec.pdf":
				return fakeInfo{}, nil
			}
			return nil, os.ErrNotExist
		},
	}
}

func TestSplit(t *testing.T) {
	got := Split("figma, ~/Notes and github.com/me/atlas; https://a.b\nCalculator")
	want := []string{"figma", "~/Notes", "github.com/me/atlas", "https://a.b", "Calculator"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q", got)
	}
}

func TestResolve(t *testing.T) {
	res := resolver().Resolve([]string{
		"figma", "~/Notes", "~/spec.pdf", "github.com/me/atlas", "localhost:3000",
		"obsidian://open?vault=x", "Visual Studio Code.app", "cal", "vs code", "~/missing", "notes.md",
	})
	type got struct {
		Type  cue.ResourceType
		Value string
		Label string
	}
	var g []got
	for _, r := range res.Resources {
		v := r.URL + r.Path
		if r.App != nil {
			v = r.App.BundleID
		}
		g = append(g, got{r.Type, v, r.Label})
		if r.ID == "" {
			t.Error("missing id")
		}
	}
	want := []got{
		{cue.TypeApp, "com.figma.Desktop", ""},
		{cue.TypeFolder, "~/Notes", ""},
		{cue.TypeFile, "~/spec.pdf", ""},
		{cue.TypeURL, "https://github.com/me/atlas", "GitHub"},
		{cue.TypeURL, "http://localhost:3000", "localhost"},
		{cue.TypeURL, "obsidian://open?vault=x", "Obsidian"},
		{cue.TypeApp, "com.microsoft.VSCode", ""},
	}
	if !reflect.DeepEqual(g, want) {
		t.Errorf("resolved:\n got %+v\nwant %+v", g, want)
	}
	// "cal" is ambiguous (Calculator, Calendar); the rest need the AI or the person.
	wantUn := []string{"cal", "vs code", "~/missing", "notes.md"}
	if !reflect.DeepEqual(res.Unresolved, wantUn) {
		t.Errorf("unresolved = %q", res.Unresolved)
	}
}
