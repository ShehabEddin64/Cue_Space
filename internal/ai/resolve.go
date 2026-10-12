package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
)

// Resolution is what Resolve returns: new resources plus questions for the
// ones the model couldn't complete.
type Resolution struct {
	Resources []cue.Resource    `json:"resources"`
	Questions map[string]string `json:"questions"`
}

var resolveSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"resources"},
	"properties": map[string]any{
		"resources": map[string]any{"type": "array", "items": resourceSchema},
	},
}

// A deliberately short prompt: this request only turns a few phrases into
// resources, so it needs far fewer tokens than proposing a whole Cue.
const resolvePrompt = `Turn each phrase into one Cue Space resource: url, app, file or folder.
Use apps from the installed list by exact name. Use well-known public URLs only. Never invent file paths or private URLs: leave them null and ask in needsInput.
Every field must be present; use null when it does not apply.`

// Resolve turns phrases the local resolver couldn't handle ("vs code", "the
// React docs") into resources.
func Resolve(ctx context.Context, p Provider, phrases []string, apps []platform.App) (Resolution, error) {
	text, err := p.Complete(ctx, Request{
		System:     resolvePrompt + "\nInstalled apps: " + appList(apps),
		User:       "Phrases:\n- " + strings.Join(phrases, "\n- "),
		SchemaName: "cue_resources",
		Schema:     resolveSchema,
	})
	if err != nil {
		return Resolution{}, err
	}
	var out struct {
		Resources []draftResource `json:"resources"`
	}
	if err := json.Unmarshal([]byte(stripFence(text)), &out); err != nil {
		return Resolution{}, fmt.Errorf("the model's answer was not valid: %w", err)
	}
	prop := draft{Name: "x", Resources: out.Resources}.toProposal(apps)
	return Resolution{Resources: prop.Cue.Resources, Questions: prop.Questions}, nil
}

func stripFence(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
	}
	return text
}
