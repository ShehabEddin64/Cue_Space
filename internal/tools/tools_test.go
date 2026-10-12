package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ShehabEddin64/Cue_Space/internal/platform"
	"github.com/ShehabEddin64/Cue_Space/internal/store"
)

func newTestService(t *testing.T) (*Service, *[]string) {
	t.Helper()
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Documents", "Math"), 0o755)
	os.WriteFile(filepath.Join(home, "Documents", "syllabus.pdf"), nil, 0o644)
	st, err := store.Open(filepath.Join(home, "cues"))
	if err != nil {
		t.Fatal(err)
	}
	st.Trash = os.Remove
	opened := &[]string{}
	return &Service{
		Store: st,
		Home:  home,
		InstalledApps: func() []platform.App {
			return []platform.App{
				{Name: "Visual Studio Code", BundleID: "com.microsoft.VSCode"},
				{Name: "Calculator", BundleID: "com.apple.calculator"},
				{Name: "Safari", BundleID: "com.apple.Safari"},
				{Name: "Notes", BundleID: "com.apple.Notes"},
				{Name: "Notion", BundleID: "notion.id"},
			}
		},
		Open: func(_ context.Context, link string) error {
			*opened = append(*opened, link)
			return nil
		},
		Search: func(_ context.Context, root, query string, limit int) ([]string, error) {
			return []string{filepath.Join(root, "Documents", "Math"), filepath.Join(root, "Documents", "syllabus.pdf")}, nil
		},
	}, opened
}

func TestCreateWithLayout(t *testing.T) {
	svc, _ := newTestService(t)
	saved, err := svc.Create(CueInput{
		Name: "Math", Icon: "📐",
		Resources: []Resource{
			{ID: "moodle", Type: "url", URL: "https://moodle.example.edu"},
			{ID: "notes", Type: "folder", Path: filepath.Join(svc.Home, "Documents", "Math"), OpenWith: "visual studio code"},
			{Type: "folder", Path: "~/Downloads", Newest: true, Extensions: []string{".PDF"}},
			{Type: "app", App: "calc"},
			{Type: "app", App: "code"},
		},
		Desktops: []Desktop{{Arrangement: "left-right", Slots: []string{"moodle", "notes"}, NewDesktop: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	v := saved.Cue
	if v.Resources[1].Path != "~/Documents/Math" || v.Resources[1].OpenWith != "Visual Studio Code" {
		t.Errorf("folder = %+v", v.Resources[1])
	}
	if v.Resources[3].App != "Calculator" {
		t.Errorf("app prefix not resolved: %+v", v.Resources[3])
	}
	if v.Resources[4].App != "Visual Studio Code" {
		t.Errorf("app by part of its name not resolved: %+v", v.Resources[4])
	}
	if got := v.Resources[2].Extensions; len(got) != 1 || got[0] != "pdf" {
		t.Errorf("extensions = %v", got)
	}
	if len(v.Desktops) != 1 || v.Desktops[0].Slots[0] != "moodle" || !v.Desktops[0].NewDesktop {
		t.Errorf("desktops = %+v", v.Desktops)
	}
	// ~/Downloads doesn't exist in the test home: saved, with a warning.
	if len(saved.Warnings) != 1 || !strings.Contains(saved.Warnings[0], "~/Downloads") {
		t.Errorf("warnings = %v", saved.Warnings)
	}
	if !strings.HasPrefix(v.File, "~/cues/") {
		t.Errorf("file = %q", v.File)
	}
}

func TestCreateErrors(t *testing.T) {
	svc, _ := newTestService(t)
	cases := map[string]CueInput{
		"needs a name":                 {Resources: []Resource{{Type: "url", URL: "https://a.b"}}},
		`no app named "Figma"`:         {Name: "x", Resources: []Resource{{Type: "app", App: "Figma"}}},
		`"no" could be Notes, Notion`:  {Name: "x", Resources: []Resource{{Type: "app", App: "no"}}},
		`unknown type "shell"`:         {Name: "x", Resources: []Resource{{Type: "shell"}}},
		`unknown arrangement "grid"`:   {Name: "x", Resources: []Resource{{ID: "a", Type: "url", URL: "https://a.b"}}, Desktops: []Desktop{{Arrangement: "grid", Slots: []string{"a"}}}},
		`no item has id "b"`:           {Name: "x", Resources: []Resource{{ID: "a", Type: "url", URL: "https://a.b"}}, Desktops: []Desktop{{Arrangement: "fill", Slots: []string{"b"}}}},
		"fill has 1 slot(s), not 2":    {Name: "x", Resources: []Resource{{ID: "a", Type: "url", URL: "https://a.b"}}, Desktops: []Desktop{{Arrangement: "fill", Slots: []string{"a", ""}}}},
		"link is missing":              {Name: "x", Resources: []Resource{{Type: "url"}}},
		"is a file; use type \"file\"": {Name: "x", Resources: []Resource{{Type: "folder", Path: "~/Documents/syllabus.pdf"}}},
	}
	for want, in := range cases {
		if _, err := svc.Create(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want error containing %q, got %v", want, err)
		}
	}
	if cues, _ := svc.List(); len(cues) != 0 {
		t.Errorf("nothing should be saved, got %d Cues", len(cues))
	}
}

func TestUpdateKeepsWhatIsLeftOut(t *testing.T) {
	svc, _ := newTestService(t)
	saved, err := svc.Create(CueInput{
		Name: "Math", Description: "Study",
		Resources: []Resource{{ID: "a", Type: "url", URL: "https://a.b"}, {ID: "b", Type: "app", App: "Safari"}},
		Desktops:  []Desktop{{Arrangement: "left-right", Slots: []string{"a", "b"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// A new name only: items and layout stay.
	up, err := svc.Update("math", CueInput{Name: "Calculus"})
	if err != nil {
		t.Fatal(err)
	}
	if up.Cue.ID != saved.Cue.ID || up.Cue.Name != "Calculus" || up.Cue.Description != "Study" || len(up.Cue.Resources) != 2 || len(up.Cue.Desktops) != 1 {
		t.Errorf("after rename: %+v", up.Cue)
	}
	// New items without b: the layout keeps a, warns about b.
	up, err = svc.Update(saved.Cue.ID, CueInput{Resources: []Resource{{ID: "a", Type: "url", URL: "https://a.b"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(up.Warnings) != 1 || up.Cue.Desktops[0].Slots[0] != "a" || up.Cue.Desktops[0].Slots[1] != "" {
		t.Errorf("after removing b: %+v, warnings %v", up.Cue.Desktops, up.Warnings)
	}
	// An empty desktops list removes the layout.
	up, err = svc.Update("Calculus", CueInput{Desktops: []Desktop{}})
	if err != nil || len(up.Cue.Desktops) != 0 {
		t.Errorf("layout not removed: %+v, %v", up.Cue.Desktops, err)
	}
}

func TestFind(t *testing.T) {
	svc, _ := newTestService(t)
	for _, n := range []string{"Math", "Math Exam", "Music"} {
		if _, err := svc.Create(CueInput{Name: n, Resources: []Resource{{Type: "url", URL: "https://a.b"}}}); err != nil {
			t.Fatal(err)
		}
	}
	if c, err := svc.Find("math"); err != nil || c.Name != "Math" {
		t.Errorf("exact: %v %v", c.Name, err)
	}
	if c, err := svc.Find("exam"); err != nil || c.Name != "Math Exam" {
		t.Errorf("contains: %v %v", c.Name, err)
	}
	if _, err := svc.Find("m"); err == nil || !strings.Contains(err.Error(), "several") {
		t.Errorf("ambiguous: %v", err)
	}
	if _, err := svc.Find("art"); err == nil {
		t.Error("want no match")
	}
}

func TestStartAndDelete(t *testing.T) {
	svc, opened := newTestService(t)
	saved, _ := svc.Create(CueInput{Name: "Math", Resources: []Resource{{Type: "url", URL: "https://a.b"}}})
	if _, err := svc.Start(context.Background(), "Math"); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || (*opened)[0] != "cuespace://start/"+saved.Cue.ID {
		t.Errorf("opened = %v", *opened)
	}
	if _, err := svc.Delete("Math"); err != nil {
		t.Fatal(err)
	}
	if cues, _ := svc.List(); len(cues) != 0 {
		t.Error("still listed after delete")
	}
}

// An MCP client creates and reads a Cue through the server, as Claude would.
func TestMCPServer(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	server := NewServer(svc)
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	if !strings.Contains(cs.InitializeResult().Instructions, "list_apps") {
		t.Error("instructions missing")
	}
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range list.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"list_cues", "get_cue", "create_cue", "update_cue", "delete_cue", "start_cue", "list_apps", "search_files", "check_paths"} {
		if !names[want] {
			t.Errorf("tool %s missing", want)
		}
	}

	call := func(name string, args any) map[string]any {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.IsError {
			t.Fatalf("%s: tool error: %v", name, res.Content[0].(*mcp.TextContent).Text)
		}
		data, _ := json.Marshal(res.StructuredContent)
		var out map[string]any
		json.Unmarshal(data, &out)
		return out
	}

	call("create_cue", map[string]any{
		"name":      "Atlas",
		"resources": []any{map[string]any{"id": "code", "type": "folder", "path": "~/Documents/Math", "openWith": "Visual Studio Code"}},
		"desktops":  []any{map[string]any{"arrangement": "fullscreen", "slots": []string{"code"}}},
	})
	got := call("get_cue", map[string]any{"cue": "atlas"})
	if got["name"] != "Atlas" || len(got["desktops"].([]any)) != 1 {
		t.Errorf("get_cue = %v", got)
	}
	if cues := call("list_cues", map[string]any{})["cues"].([]any); len(cues) != 1 {
		t.Errorf("list_cues = %v", cues)
	}
	if found := call("search_files", map[string]any{"query": "math", "kind": "folder"})["results"].([]any); len(found) != 1 {
		t.Errorf("search_files = %v", found)
	}

	// A bad request comes back as a tool error the model can read and fix.
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "create_cue", Arguments: map[string]any{"name": "x", "resources": []any{map[string]any{"type": "app", "app": "Figma"}}}})
	if err != nil || !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "list_apps") {
		t.Errorf("want a readable tool error, got %+v, %v", res, err)
	}
}

func TestHTTPHandlerNeedsToken(t *testing.T) {
	svc, _ := newTestService(t)
	srv := httptest.NewServer(HTTPHandler(svc, "secret-token"))
	defer srv.Close()

	for _, path := range []string{"/mcp", "/mcp/wrong", "/", "/mcp/secret-token-but-longer"} {
		resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, resp.StatusCode)
		}
	}

	// With the token in the path, a real client connects.
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp/secret-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if _, err := cs.ListTools(ctx, nil); err != nil {
		t.Fatal(err)
	}
}
