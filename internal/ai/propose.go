package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
)

// draft is the flat shape the model fills in. It is intentionally simpler than
// the .cue.json schema (no conditionals, every field present) so that strict
// structured output works on every provider. Go code turns it into a Cue.
type draft struct {
	Name        string          `json:"name"`
	Icon        string          `json:"icon"`
	Description string          `json:"description"`
	Resources   []draftResource `json:"resources"`
	Notes       string          `json:"notes"`
}

// Apps are named, not identified by bundle id: the model only sees app
// names (fewer tokens) and Cue Space maps names to bundle ids itself.
type draftResource struct {
	Type         string  `json:"type"`
	Label        string  `json:"label"`
	URL          *string `json:"url"`
	AppName      *string `json:"appName"`
	Path         *string `json:"path"`
	OpenWithName *string `json:"openWithName"`
	NeedsInput   *string `json:"needsInput"`
}

func nullableString(desc string) map[string]any {
	return map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}, "description": desc}
}

var resourceSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"type", "label", "url", "appName", "path", "openWithName", "needsInput"},
	"properties": map[string]any{
		"type":         map[string]any{"type": "string", "enum": []string{"url", "app", "file", "folder"}},
		"label":        map[string]any{"type": "string", "description": "What the user calls it, e.g. \"GitHub\" or \"Textbook\"."},
		"url":          nullableString("For url: the full URL including scheme."),
		"appName":      nullableString("For app: the exact name from the installed list."),
		"path":         nullableString("For file/folder: an absolute path or one starting with ~/. Only if the user gave it."),
		"openWithName": nullableString("For file/folder: the app to open it with, from the installed list."),
		"needsInput":   nullableString("If a required detail is unknown (a path, a private URL), a short question for the user. Otherwise null."),
	},
}

var draftSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"name", "icon", "description", "resources", "notes"},
	"properties": map[string]any{
		"name":        map[string]any{"type": "string", "description": "Short name of the activity, e.g. \"Atlas\" or \"Math\"."},
		"icon":        map[string]any{"type": "string", "description": "One emoji."},
		"description": map[string]any{"type": "string", "description": "One short sentence."},
		"notes":       map[string]any{"type": "string", "description": "Anything the user should double-check, in one or two sentences. Empty if nothing."},
		"resources":   map[string]any{"type": "array", "items": resourceSchema},
	},
}

const systemPrompt = `You help people build Cues in Cue Space, a desktop app.

A Cue is a saved context for one activity, such as working on a project or studying a subject. It is an ordered list of resources that open together when the user starts the Cue. Resource types:
- url: a web page or a deep link (https://, http://localhost:3000, obsidian://, vscode://, ...)
- app: an installed application
- file: a document on disk
- folder: a folder on disk, optionally opened with an app (a code project opened in an IDE is a folder with openWith set to the IDE)

Rules:
- Only use apps from the installed list below, with their exact name. Map generic words to what is installed: "IDE" or "editor" means the code editor they have (Cursor, VS Code, Zed, Xcode, ...), "browser" means their browser, "Claude" means the Claude app if installed, otherwise https://claude.ai.
- Never invent file or folder paths or private URLs such as a specific GitHub repository. If the user did not give one, keep the resource, leave path/url null, and put a short question in needsInput (for example "Where is the Atlas repository on your Mac?").
- Well-known public sites are fine to fill in (https://github.com, https://claude.ai, https://docs.python.org). For localhost without a port, use http://localhost:3000 and mention it in notes.
- Keep the order the user would want things to open in. Do not add resources the user did not ask for, unless they are clearly implied by the activity and you mention them in notes.
- Every field must be present; use null for fields that do not apply to the resource type.`

// Proposal is an AI-suggested Cue awaiting the user's review.
type Proposal struct {
	Cue cue.Cue `json:"cue"`
	// Questions maps a resource id to what the user still needs to fill in.
	Questions map[string]string `json:"questions"`
	Notes     string            `json:"notes"`
}

// Propose asks the provider for a Cue matching description. When existing is
// set, the model edits that Cue instead of starting from scratch.
func Propose(ctx context.Context, p Provider, description string, existing *cue.Cue, apps []platform.App) (Proposal, error) {
	req := Request{
		System:     systemPrompt + "\n\nInstalled apps: " + appList(apps),
		User:       userPrompt(description, existing),
		SchemaName: "cue_draft",
		Schema:     draftSchema,
	}
	text, err := p.Complete(ctx, req)
	if err != nil {
		return Proposal{}, err
	}
	d, err := parseDraft(text)
	if err != nil {
		return Proposal{}, err
	}
	prop := d.toProposal(apps)
	if existing != nil {
		prop.Cue.ID = existing.ID
		prop.Cue.CreatedAt = existing.CreatedAt
		prop.Cue.LastRunAt = existing.LastRunAt
	}
	return prop, nil
}

// appList is a compact, comma-separated list of app names. Names are enough
// for the model; bundle ids are filled in locally.
func appList(apps []platform.App) string {
	if len(apps) == 0 {
		return "(unknown)"
	}
	names := make([]string, len(apps))
	for i, a := range apps {
		names[i] = a.Name
	}
	return strings.Join(names, ", ")
}

func userPrompt(description string, existing *cue.Cue) string {
	if existing == nil {
		return "Create a Cue for this:\n\n" + description
	}
	return "Current Cue:\n" + compactCue(*existing) +
		"\nReturn the full updated Cue after this change, keeping everything else as it is:\n" + description
}

// compactCue describes a Cue in a few short lines rather than JSON.
func compactCue(c cue.Cue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", c.Icon, c.Name)
	if c.Description != "" {
		fmt.Fprintf(&b, ": %s", c.Description)
	}
	b.WriteByte('\n')
	for i, r := range c.Resources {
		fmt.Fprintf(&b, "%d. %s %q", i+1, r.Type, r.DisplayName())
		switch r.Type {
		case cue.TypeURL:
			fmt.Fprintf(&b, " %s", r.URL)
		case cue.TypeApp:
			if r.App != nil {
				fmt.Fprintf(&b, " app=%s", r.App.Name)
			}
		case cue.TypeFile, cue.TypeFolder:
			fmt.Fprintf(&b, " %s", r.Path)
			if r.OpenWith != nil {
				fmt.Fprintf(&b, " with=%s", r.OpenWith.Name)
			}
		}
		if r.Disabled {
			b.WriteString(" (off)")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func parseDraft(text string) (draft, error) {
	var d draft
	// Some local models wrap JSON in a code fence despite the schema.
	if err := json.Unmarshal([]byte(stripFence(text)), &d); err != nil {
		return d, fmt.Errorf("the model's answer was not a valid Cue: %w", err)
	}
	if strings.TrimSpace(d.Name) == "" {
		return d, fmt.Errorf("the model's answer had no name")
	}
	return d, nil
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func (d draft) toProposal(apps []platform.App) Proposal {
	c := cue.Cue{Name: d.Name, Icon: d.Icon, Description: d.Description}
	questions := map[string]string{}
	for _, dr := range d.Resources {
		r := cue.Resource{ID: cue.NewID(), Type: cue.ResourceType(dr.Type), Label: strings.TrimSpace(dr.Label)}
		question := str(dr.NeedsInput)
		switch r.Type {
		case cue.TypeURL:
			r.URL = str(dr.URL)
		case cue.TypeApp:
			app, ok := resolveApp(apps, "", str(dr.AppName))
			r.App = app
			if !ok && question == "" {
				question = fmt.Sprintf("%q doesn't seem to be installed. Pick an app or remove this item.", app.Name)
			}
		case cue.TypeFile, cue.TypeFolder:
			r.Path = str(dr.Path)
			if name := str(dr.OpenWithName); name != "" {
				app, ok := resolveApp(apps, "", name)
				r.OpenWith = app
				if !ok && question == "" {
					question = fmt.Sprintf("%q doesn't seem to be installed. Pick another app to open this with.", app.Name)
				}
			}
		default:
			continue // unknown types are dropped rather than guessed at
		}
		if question != "" {
			questions[r.ID] = question
		}
		c.Resources = append(c.Resources, r)
	}
	c.Normalize()
	return Proposal{Cue: c, Questions: questions, Notes: strings.TrimSpace(d.Notes)}
}

// resolveApp matches the model's app reference against what is actually
// installed, by bundle id first and then by name. ok is false when nothing
// installed matches (or the catalog is unavailable).
func resolveApp(apps []platform.App, bundleID, name string) (*cue.AppRef, bool) {
	for _, a := range apps {
		if bundleID != "" && strings.EqualFold(a.BundleID, bundleID) {
			return &cue.AppRef{BundleID: a.BundleID, Name: a.Name}, true
		}
	}
	for _, a := range apps {
		if name != "" && strings.EqualFold(a.Name, name) {
			return &cue.AppRef{BundleID: a.BundleID, Name: a.Name}, true
		}
	}
	if name == "" {
		name = bundleID
	}
	return &cue.AppRef{BundleID: bundleID, Name: name}, len(apps) == 0
}
