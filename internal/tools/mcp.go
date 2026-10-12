package tools

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is reported to MCP clients.
const Version = "0.1.0"

const instructions = `Cue Space is a Mac app that opens everything someone needs for an activity in one click. A Cue is a saved context such as "Study math" or "Work on Atlas": an ordered list of links, apps, files and folders, optionally arranged across macOS desktops.

How to build a good Cue:
1. Call list_cues first, so you update an existing Cue instead of making a duplicate.
2. Use real things only. Find apps with list_apps and use their exact names. Find files and folders with search_files; never invent paths or private URLs. If something can't be found, ask the person.
3. Create the Cue with create_cue. Give items short ids (e.g. "notes") when you want to place them on desktops.
4. Desktops are optional. Use them when the person describes where windows go: e.g. a fullscreen desktop for the editor, then left-right on a new desktop for the browser and the PDF.
5. Tell the person what you made. Start a Cue only when they ask.

Starting a Cue never uses AI: Cue Space opens exactly what is saved.`

// NewServer returns an MCP server exposing svc's operations as tools.
func NewServer(svc *Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:       "cuespace",
		Title:      "Cue Space",
		Version:    Version,
		WebsiteURL: "https://github.com/ShehabEddin64/Cue_Space",
	}, &mcp.ServerOptions{Instructions: instructions})

	no, yes := false, true
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &no}

	type empty struct{}
	type ref struct {
		Cue string `json:"cue" jsonschema:"The Cue's name or id."`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_cues",
		Title:       "List Cues",
		Description: "Lists the person's Cues with their ids, names and how many items each opens.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, struct {
		Cues []Summary `json:"cues"`
	}, error) {
		cues, err := svc.List()
		return nil, struct {
			Cues []Summary `json:"cues"`
		}{cues}, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_cue",
		Title:       "Get Cue",
		Description: "Returns one Cue in full: its items and desktop layout, in the same shape update_cue accepts.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ref) (*mcp.CallToolResult, CueView, error) {
		v, err := svc.Get(in.Cue)
		return nil, v, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:  "create_cue",
		Title: "Create Cue",
		Description: "Creates and saves a new Cue. It appears in Cue Space right away. " +
			"Apps must be installed (see list_apps) and paths should be real (see search_files); paths that don't exist yet are saved with a warning.",
		Annotations: &mcp.ToolAnnotations{OpenWorldHint: &no, DestructiveHint: &no},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in CueInput) (*mcp.CallToolResult, Saved, error) {
		saved, err := svc.Create(in)
		return nil, saved, err
	})

	type updateIn struct {
		Cue string `json:"cue" jsonschema:"The Cue's name or id."`
		CueInput
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:  "update_cue",
		Title: "Update Cue",
		Description: "Changes an existing Cue. Fields you leave out stay as they are. " +
			"resources, when given, replaces the whole list, so get_cue first and send every item back, keeping the ids of items you keep. " +
			"desktops, when given, replaces the layout; send [] to remove it.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true, OpenWorldHint: &no, DestructiveHint: &yes},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateIn) (*mcp.CallToolResult, Saved, error) {
		saved, err := svc.Update(in.Cue, in.CueInput)
		return nil, saved, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_cue",
		Title:       "Delete Cue",
		Description: "Moves a Cue's file to the Trash (it can be put back from there). Only do this when the person asks.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true, OpenWorldHint: &no, DestructiveHint: &yes},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ref) (*mcp.CallToolResult, Summary, error) {
		s, err := svc.Delete(in.Cue)
		return nil, s, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:  "start_cue",
		Title: "Start Cue",
		Description: "Starts a Cue in Cue Space, exactly as if the person clicked Start: its links, apps, files and folders open and windows are arranged. " +
			"Only do this when the person asks. The result of the run is shown in Cue Space.",
		Annotations: &mcp.ToolAnnotations{OpenWorldHint: &yes, DestructiveHint: &no},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ref) (*mcp.CallToolResult, Summary, error) {
		s, err := svc.Start(ctx, in.Cue)
		return nil, s, err
	})

	type appsIn struct {
		Query string `json:"query,omitempty" jsonschema:"Part of an app's name, e.g. \"code\" or \"mail\". Leave empty to list every app."`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_apps",
		Title:       "List Installed Apps",
		Description: "Lists apps installed on this Mac, so Cues only use apps that exist. Map generic words to what is installed (\"IDE\" means their code editor, \"browser\" their browser).",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in appsIn) (*mcp.CallToolResult, struct {
		Apps []AppInfo `json:"apps"`
	}, error) {
		return nil, struct {
			Apps []AppInfo `json:"apps"`
		}{svc.Apps(in.Query, 200)}, nil
	})

	type searchIn struct {
		Query string `json:"query" jsonschema:"Part of the file or folder name, e.g. \"linear algebra\" or \"atlas\"."`
		Kind  string `json:"kind,omitempty" jsonschema:"file, folder or any (the default)."`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_files",
		Title:       "Search Files",
		Description: "Finds files and folders in the person's home folder by name, using Spotlight. Use it to get real paths for file and folder items.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, struct {
		Results []PathInfo `json:"results"`
	}, error) {
		found, err := svc.SearchFiles(ctx, in.Query, in.Kind, 20)
		return nil, struct {
			Results []PathInfo `json:"results"`
		}{found}, err
	})

	type pathsIn struct {
		Paths []string `json:"paths" jsonschema:"Absolute paths or paths starting with ~/."`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_paths",
		Title:       "Check Paths",
		Description: "Says whether each path exists and whether it is a file or a folder. For folders it also names the newest file inside.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in pathsIn) (*mcp.CallToolResult, struct {
		Paths []PathInfo `json:"paths"`
	}, error) {
		return nil, struct {
			Paths []PathInfo `json:"paths"`
		}{svc.CheckPaths(in.Paths)}, nil
	})

	return server
}

// ServeStdio runs the MCP server over stdin and stdout, the way Claude
// Desktop, Claude Code and most MCP clients start local servers.
func ServeStdio(ctx context.Context, svc *Service) error {
	return NewServer(svc).Run(ctx, &mcp.StdioTransport{})
}

// Token returns the secret that guards the HTTP server, creating it on first use.
func Token() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "mcp-token")
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) >= 32 {
		return strings.TrimSpace(string(data)), nil
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return token, os.WriteFile(path, []byte(token+"\n"), 0o600)
}

// HTTPHandler serves MCP over Streamable HTTP for clients that connect to a
// URL, such as ChatGPT. Every request needs the token, either in the path
// (/mcp/<token>, for clients that can't send headers) or as a bearer token
// on /mcp.
func HTTPHandler(svc *Service, token string) http.Handler {
	server := NewServer(svc)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		// Requests may arrive through a tunnel with another Host; the token
		// is what keeps them out.
		DisableLocalhostProtection: true,
	})
	ok := func(got string) bool { return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1 }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed := false
		if rest, found := strings.CutPrefix(r.URL.Path, "/mcp/"); found {
			allowed = ok(strings.SplitN(rest, "/", 2)[0])
		} else if r.URL.Path == "/mcp" {
			allowed = ok(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		}
		if !allowed {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// ServeHTTP runs the HTTP server until ctx is cancelled.
func ServeHTTP(ctx context.Context, svc *Service, addr, token string) error {
	if token == "" {
		return errors.New("a token is required")
	}
	srv := &http.Server{Addr: addr, Handler: HTTPHandler(svc, token), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("couldn't serve on %s: %w", addr, err)
	}
	return nil
}
