// Command cuespace-cli manages Cue Space Cues from the terminal and runs
// Cue Space's MCP server, which lets AI apps such as Claude build Cues.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/ShehabEddin64/Cue_Space/internal/tools"
)

const usage = `cuespace-cli: Cue Space from the terminal and from AI apps.

Usage:
  cuespace-cli list                    List your Cues
  cuespace-cli show <cue>              Show a Cue (name or id)
  cuespace-cli start <cue>             Start a Cue in Cue Space
  cuespace-cli create [file]           Create a Cue from JSON (a file, or stdin)
  cuespace-cli update <cue> [file]     Change a Cue with JSON (a file, or stdin)
  cuespace-cli delete <cue>            Move a Cue to the Trash
  cuespace-cli apps [query]            List installed apps
  cuespace-cli search <name>           Find files and folders by name
  cuespace-cli mcp                     Run the MCP server over stdio (Claude Desktop, Claude Code, ...)
  cuespace-cli mcp --http [addr]       Run it over HTTP for URL-based clients (ChatGPT)

Add --json to list, show, apps and search for machine-readable output.

The JSON for create and update looks like this:
  {
    "name": "Math", "icon": "📐",
    "resources": [
      {"id": "moodle", "type": "url", "url": "https://moodle.example.edu"},
      {"id": "notes", "type": "folder", "path": "~/Documents/Math", "openWith": "Visual Studio Code"},
      {"type": "folder", "path": "~/Downloads", "newest": true, "extensions": ["pdf"]},
      {"type": "app", "app": "Calculator"}
    ],
    "desktops": [{"arrangement": "left-right", "slots": ["moodle", "notes"], "newDesktop": true}]
  }
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "cuespace-cli:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return nil
	}
	cmd, args := args[0], args[1:]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print JSON")
	httpMode := fs.Bool("http", false, "serve MCP over HTTP instead of stdio")
	if err := fs.Parse(reorder(args)); err != nil {
		return err
	}
	args = fs.Args()

	svc, err := tools.New()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	print := func(v any, text func()) error {
		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(v)
		}
		text()
		return nil
	}

	switch cmd {
	case "list", "ls":
		cues, err := svc.List()
		if err != nil {
			return err
		}
		return print(cues, func() {
			if len(cues) == 0 {
				fmt.Fprintln(stdout, "No Cues yet. Create one in Cue Space, or with `cuespace-cli create`.")
				return
			}
			w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
			for _, c := range cues {
				fmt.Fprintf(w, "%s %s\t%d item%s\t%s\n", or(c.Icon, "◌"), c.Name, c.Items, plural(c.Items), c.ID)
			}
			w.Flush()
		})

	case "show":
		v, err := svc.Get(strings.Join(args, " "))
		if err != nil {
			return err
		}
		return print(v, func() { printCue(stdout, v) })

	case "start", "run":
		s, err := svc.Start(ctx, strings.Join(args, " "))
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Started %s %s\n", or(s.Icon, "◌"), s.Name)
		return nil

	case "create", "new":
		var in tools.CueInput
		if err := readJSON(args, 0, stdin, &in); err != nil {
			return err
		}
		saved, err := svc.Create(in)
		if err != nil {
			return err
		}
		return reportSaved(stdout, "Created", saved)

	case "update", "edit":
		if len(args) == 0 {
			return errors.New("say which Cue to update")
		}
		var in tools.CueInput
		if err := readJSON(args, 1, stdin, &in); err != nil {
			return err
		}
		saved, err := svc.Update(args[0], in)
		if err != nil {
			return err
		}
		return reportSaved(stdout, "Updated", saved)

	case "delete", "rm":
		s, err := svc.Delete(strings.Join(args, " "))
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Moved %s %s to the Trash\n", or(s.Icon, "◌"), s.Name)
		return nil

	case "apps":
		apps := svc.Apps(strings.Join(args, " "), 0)
		return print(apps, func() {
			w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
			for _, a := range apps {
				fmt.Fprintf(w, "%s\t%s\n", a.Name, a.BundleID)
			}
			w.Flush()
		})

	case "search", "find":
		found, err := svc.SearchFiles(ctx, strings.Join(args, " "), "any", 30)
		if err != nil {
			return err
		}
		return print(found, func() {
			for _, p := range found {
				fmt.Fprintf(stdout, "%-6s %s\n", p.Kind, p.Path)
			}
		})

	case "mcp":
		if !*httpMode {
			return tools.ServeStdio(ctx, svc)
		}
		addr := "127.0.0.1:8765"
		if len(args) > 0 {
			addr = args[0]
		}
		token, err := tools.Token()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Cue Space MCP server on http://%s/mcp/%s\n", addr, token)
		fmt.Fprintln(os.Stderr, "Keep this address private: anyone with it can change your Cues and start them. Press Ctrl-C to stop.")
		return tools.ServeHTTP(ctx, svc, addr, token)

	case "version":
		fmt.Fprintln(stdout, "cuespace-cli", tools.Version)
		return nil
	}
	return fmt.Errorf("unknown command %q (try cuespace-cli help)", cmd)
}

// reorder moves flags in front of positional arguments, so
// `cuespace-cli show Math --json` works like `show --json Math`.
func reorder(args []string) []string {
	var flags, rest []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			flags = append(flags, a)
		} else {
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

// readJSON decodes args[i] as a file path, or stdin when it's absent or "-".
func readJSON(args []string, i int, stdin io.Reader, v any) error {
	r := stdin
	if len(args) > i && args[i] != "-" {
		f, err := os.Open(args[i])
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("couldn't read the Cue JSON: %w", err)
	}
	return nil
}

func reportSaved(w io.Writer, verb string, s tools.Saved) error {
	fmt.Fprintf(w, "%s %s %s (%s)\n", verb, or(s.Cue.Icon, "◌"), s.Cue.Name, s.Cue.ID)
	for _, warn := range s.Warnings {
		fmt.Fprintln(w, "  note:", warn)
	}
	return nil
}

func printCue(w io.Writer, v tools.CueView) {
	fmt.Fprintf(w, "%s %s\n", or(v.Icon, "◌"), v.Name)
	if v.Description != "" {
		fmt.Fprintln(w, v.Description)
	}
	fmt.Fprintln(w)
	for i, r := range v.Resources {
		target := r.URL + r.App + r.Path
		if r.Newest {
			kind := "newest file in"
			if len(r.Extensions) > 0 {
				kind = "newest " + strings.ToUpper(strings.Join(r.Extensions, "/")) + " in"
			}
			target = kind + " " + target
		}
		if r.OpenWith != "" {
			target += " (with " + r.OpenWith + ")"
		}
		line := fmt.Sprintf("%2d. %-6s %s", i+1, r.Type, target)
		if r.Label != "" {
			line += "  “" + r.Label + "”"
		}
		if r.Disabled {
			line += "  [off]"
		}
		fmt.Fprintln(w, line)
	}
	for i, d := range v.Desktops {
		fmt.Fprintf(w, "\nDesktop %d: %s", i+1, d.Arrangement)
		if d.NewDesktop {
			fmt.Fprint(w, " (new desktop)")
		}
		fmt.Fprintf(w, ": %s\n", strings.Join(d.Slots, ", "))
	}
	fmt.Fprintf(w, "\nid %s\n", v.ID)
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
