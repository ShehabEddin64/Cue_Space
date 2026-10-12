package cue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/ShehabEddin64/Cue_Space/schema"
)

// ValidationError lists every problem found in a Cue.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid cue: " + strings.Join(e.Problems, "; ")
}

// Problems about a value that hasn't been filled in yet end with this, so
// the editor can wait until someone tries to save before pointing them out.
const missing = " is missing"

// NameMissing is the problem reported for a Cue without a name.
const NameMissing = "Name" + missing

// Validate checks that a Cue is well formed. It does not touch the filesystem:
// whether a path exists is decided when the Cue runs.
func (c *Cue) Validate() error {
	var p []string
	if c.SchemaVersion != SchemaVersion {
		p = append(p, fmt.Sprintf("unsupported schemaVersion %d", c.SchemaVersion))
	}
	if strings.TrimSpace(c.Name) == "" {
		p = append(p, NameMissing)
	}
	ids := map[string]bool{}
	for i, r := range c.Resources {
		where := fmt.Sprintf("Item %d", i+1)
		if name := r.DisplayName(); name != "" && name != string(r.Type) {
			where += " (" + name + ")"
		}
		if r.ID == "" {
			p = append(p, where+": id is required")
		} else if ids[r.ID] {
			p = append(p, where+": duplicate id "+r.ID)
		}
		ids[r.ID] = true

		switch r.Type {
		case TypeURL:
			if r.URL == "" {
				p = append(p, where+": link"+missing)
			} else if err := checkURL(r.URL); err != nil {
				p = append(p, where+": "+err.Error())
			}
		case TypeApp:
			if r.App.IsZero() {
				p = append(p, where+": app"+missing)
			}
		case TypeFile, TypeFolder:
			if r.Path == "" {
				p = append(p, where+": "+string(r.Type)+missing)
			} else if !strings.HasPrefix(r.Path, "/") && r.Path != "~" && !strings.HasPrefix(r.Path, "~/") {
				p = append(p, where+": path must be absolute or start with ~/")
			}
		default:
			p = append(p, fmt.Sprintf("%s: unknown type %q", where, r.Type))
		}
		if r.Type != TypeFile && r.Type != TypeFolder && r.OpenWith != nil {
			p = append(p, where+": openWith only applies to files and folders")
		}
		if r.Newest && r.Type != TypeFolder {
			p = append(p, where+": newest only applies to folders")
		}
	}
	p = append(p, c.validateLayout()...)
	if len(p) > 0 {
		return &ValidationError{Problems: p}
	}
	return nil
}

func checkURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return fmt.Errorf("%q is not a full URL (include https://)", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "javascript", "data":
		return fmt.Errorf("%s: URLs are not allowed", u.Scheme)
	case "file":
		return fmt.Errorf("use a file or folder resource instead of a file: URL")
	}
	return nil
}

var (
	compiled    *jsonschema.Schema
	compileOnce sync.Once
	compileErr  error
)

func jsonSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema.CueV1))
		if err != nil {
			compileErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("cue.v1.json", doc); err != nil {
			compileErr = err
			return
		}
		compiled, compileErr = c.Compile("cue.v1.json")
	})
	return compiled, compileErr
}

// Parse decodes a .cue.json document, checking it against the JSON Schema and
// then the semantic rules in Validate. Missing ids are filled in.
func Parse(data []byte) (Cue, error) {
	var c Cue
	sch, err := jsonSchema()
	if err != nil {
		return c, fmt.Errorf("compile schema: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return c, fmt.Errorf("not valid JSON: %w", err)
	}
	if err := sch.Validate(inst); err != nil {
		return c, &ValidationError{Problems: []string{err.Error()}}
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	c.Normalize()
	return c, c.Validate()
}
