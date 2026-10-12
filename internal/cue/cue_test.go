package cue

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const atlas = `{
  "schemaVersion": 1,
  "name": "Atlas",
  "icon": "🛰️",
  "resources": [
    {"type": "url", "label": "GitHub", "url": "https://github.com/me/atlas"},
    {"type": "app", "app": {"bundleId": "com.anthropic.claudefordesktop", "name": "Claude"}},
    {"type": "folder", "path": "~/Developer/atlas", "openWith": {"name": "Cursor"}},
    {"type": "url", "url": "http://localhost:3000", "disabled": true},
    {"type": "file", "path": "/Users/me/spec.pdf"}
  ]
}`

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(atlas))
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == "" {
		t.Error("expected an id to be assigned")
	}
	if len(c.Resources) != 5 {
		t.Fatalf("got %d resources", len(c.Resources))
	}
	for _, r := range c.Resources {
		if r.ID == "" {
			t.Errorf("resource %s has no id", r.DisplayName())
		}
	}
	if !c.Resources[3].Disabled {
		t.Error("disabled flag lost")
	}
	if got := c.Resources[2].DisplayName(); got != "atlas" {
		t.Errorf("DisplayName = %q", got)
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"missing name":    `{"schemaVersion":1,"resources":[]}`,
		"empty name":      `{"schemaVersion":1,"name":"","resources":[]}`,
		"unknown field":   `{"schemaVersion":1,"name":"x","resources":[],"run":"rm -rf /"}`,
		"unknown type":    `{"schemaVersion":1,"name":"x","resources":[{"type":"shell","url":"x"}]}`,
		"url without url": `{"schemaVersion":1,"name":"x","resources":[{"type":"url"}]}`,
		"relative url":    `{"schemaVersion":1,"name":"x","resources":[{"type":"url","url":"github.com"}]}`,
		"javascript url":  `{"schemaVersion":1,"name":"x","resources":[{"type":"url","url":"javascript:alert(1)"}]}`,
		"relative path":   `{"schemaVersion":1,"name":"x","resources":[{"type":"file","path":"notes.md"}]}`,
		"empty app":       `{"schemaVersion":1,"name":"x","resources":[{"type":"app","app":{}}]}`,
		"openWith on url": `{"schemaVersion":1,"name":"x","resources":[{"type":"url","url":"https://a.b","openWith":{"name":"Safari"}}]}`,
		"future version":  `{"schemaVersion":2,"name":"x","resources":[]}`,
		"not json":        `{`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(doc)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestValidateCollectsAllProblems(t *testing.T) {
	c := Cue{SchemaVersion: 1, Resources: []Resource{
		{ID: "a", Type: TypeURL},
		{ID: "a", Type: TypeFolder, Path: "relative"},
	}}
	err := c.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("got %v", err)
	}
	if len(ve.Problems) != 4 { // name, url, duplicate id, relative path
		t.Errorf("got %d problems: %s", len(ve.Problems), strings.Join(ve.Problems, " | "))
	}
}

func TestNormalizeFixesDuplicateIDs(t *testing.T) {
	c := Cue{Name: "  x  ", Resources: []Resource{
		{ID: "same", Type: TypeURL, URL: "https://a.b"},
		{ID: "same", Type: TypeURL, URL: "https://c.d", App: &AppRef{}},
	}}
	c.Normalize()
	if c.Name != "x" {
		t.Errorf("name not trimmed: %q", c.Name)
	}
	if c.Resources[0].ID == c.Resources[1].ID {
		t.Error("duplicate ids kept")
	}
	if c.Resources[1].App != nil {
		t.Error("empty AppRef not cleared")
	}
	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}

func TestRoundTripOmitsEmptyFields(t *testing.T) {
	c, err := Parse([]byte(atlas))
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "lastRunAt") {
		t.Error("zero lastRunAt should be omitted")
	}
	if _, err := Parse(out); err != nil {
		t.Errorf("re-parse failed: %v", err)
	}
}

func TestURLDisplayName(t *testing.T) {
	cases := map[string]string{
		"https://www.moodle.concordia.ca/course/view.php?id=1": "moodle.concordia.ca",
		"http://localhost:3000":                                "localhost",
		"obsidian://open?vault=x":                              "obsidian://open?vault=x",
	}
	for u, want := range cases {
		if got := (Resource{Type: TypeURL, URL: u}).DisplayName(); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", u, got, want)
		}
	}
}
