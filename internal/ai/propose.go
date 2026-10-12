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

type draftResource struct {
	Type             string  `json:"type"`
	Label            string  `json:"label"`
	URL              *string `json:"url"`
	AppName          *string `json:"appName"`
	AppBundleID      *string `json:"appBundleId"`
	Path             *string `json:"path"`
	OpenWithName     *string `json:"openWithName"`
	OpenWithBundleID *string `json:"openWithBundleId"`
	NeedsInput       *string `json:"needsInput"`
}

func nullableString(desc string) map[string]any {
	return map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}, "description": desc}
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
		"resources": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"type", "label", "url", "appName", "appBundleId", "path", "openWithName", "openWithBundleId", "needsInput"},
				"properties": map[string]any{
					"type":             map[string]any{"type": "string", "enum": []string{"url", "app", "file", "folder"}},
					"label":            map[string]any{"type": "string", "description": "What the user calls it, e.g. \"GitHub\" or \"Textbook\"."},
					"url":              nullableString("For url: the full URL including scheme."),
					"appName":          nullableString("For app: the app's name from the installed list."),
					"appBundleId":      nullableString("For app: the bundle id from the installed list."),
					"path":             nullableString("For file/folder: an absolute path or one starting with ~/. Only if the user gave it."),
					"openWithName":     nullableString("For file/folder: the app to open it with, from the installed list."),
					"openWithBundleId": nullableString("For file/folder: bundle id of the app to open it with."),
					"needsInput":       nullableString("If a required detail is unknown (a path, a private URL), a short question for the user. Otherwise null."),
				},
			},
		},
	},
}

const systemPrompt = `You help people build Cues in Cue Space, a desktop app.

A Cue is a saved context for one activity, such as working on a project or studying a subject. It is an ordered list of resources that open together when the user starts the Cue. Resource types:
- url: a web page or a deep link (https://, http://localhost:3000, obsidian://, vscode://, ...)
- app: an installed application
- file: a document on disk
- folder: a folder on disk, optionally opened with an app (a code project opened in an IDE is a folder with openWith set to the IDE)

Rules:
- Only use apps from the installed list below, with their exact name and bundle id. Map generic words to what is installed: "IDE" or "editor" means the code editor they have (Cursor, VS Code, Zed, Xcode, ...), "browser" means their browser, "Claude" means the Claude app if installed, otherwise https://claude.ai.
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
		System:     systemPrompt + "\n\nInstalled apps (name — bundle id):\n" + appList(apps),
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

func appList(apps []platform.App) string {
	if len(apps) == 0 {
		return "(unknown)"
	}
	var b strings.Builder
	for _, a := range apps {
		fmt.Fprintf(&b, "%s — %s\n", a.Name, a.BundleID)
	}
	return b.String()
}

func userPrompt(description string, existing *cue.Cue) string {
	if existing == nil {
		return "Create a Cue for this:\n\n" + description
	}
	current, _ := json.MarshalIndent(struct {
		Name        string         `json:"name"`
		Icon        string         `json:"icon,omitempty"`
		Description string         `json:"description,omitempty"`
		Resources   []cue.Resource `json:"resources"`
	}{existing.Name, existing.Icon, existing.Description, existing.Resources}, "", "  ")
	return "Here is an existing Cue:\n\n" + string(current) +
		"\n\nReturn the full updated Cue after applying this change, keeping everything else as it is:\n\n" + description
}

func parseDraft(text string) (draft, error) {
	var d draft
	text = strings.TrimSpace(text)
	// Some local models wrap JSON in a code fence despite the schema.
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	}
	if err := json.Unmarshal([]byte(text), &d); err != nil {
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
			app, ok := resolveApp(apps, str(dr.AppBundleID), str(dr.AppName))
			r.App = app
			if !ok && question == "" {
				question = fmt.Sprintf("%q doesn't seem to be installed. Pick an app or remove this item.", app.Name)
			}
		case cue.TypeFile, cue.TypeFolder:
			r.Path = str(dr.Path)
			if name, id := str(dr.OpenWithName), str(dr.OpenWithBundleID); name != "" || id != "" {
				app, ok := resolveApp(apps, id, name)
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
