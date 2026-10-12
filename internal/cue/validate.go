package cue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
	"github.com/ShehabEddin64/Cue_Space/schema"
)

// ValidationError lists every problem found in a Cue.
type ValidationError struct {
	Problems []string
	// missing marks problems about a value that hasn't been filled in yet
	// (an empty name, link or path), so the editor can wait until someone
	// tries to save before pointing them out.
	missing []bool
}

func (e *ValidationError) Error() string {
	return "invalid cue: " + strings.Join(e.Problems, "; ")
}

// IsMissing reports whether problem i is about a value not filled in yet.
func (e *ValidationError) IsMissing(i int) bool {
	return i < len(e.missing) && e.missing[i]
}

func (e *ValidationError) add(missing bool, format string, args ...any) {
	e.Problems = append(e.Problems, i18n.T(format, args...))
	e.missing = append(e.missing, missing)
}

// NameMissing is the problem reported for a Cue without a name (in English).
const NameMissing = "Name is missing"

// Validate checks that a Cue is well formed. It does not touch the filesystem:
// whether a path exists is decided when the Cue runs.
func (c *Cue) Validate() error {
	e := &ValidationError{}
	if c.SchemaVersion != SchemaVersion {
		e.add(false, "unsupported schemaVersion %d", c.SchemaVersion)
	}
	if strings.TrimSpace(c.Name) == "" {
		e.add(true, NameMissing)
	}
	ids := map[string]bool{}
	for i, r := range c.Resources {
		where := i18n.T("Item %d", i+1)
		if name := r.DisplayName(); name != "" && name != string(r.Type) {
			where += " (" + name + ")"
		}
		if r.ID == "" {
			e.add(false, "%s: id is required", where)
		} else if ids[r.ID] {
			e.add(false, "%s: duplicate id %s", where, r.ID)
		}
		ids[r.ID] = true

		switch r.Type {
		case TypeURL:
			if r.URL == "" {
				e.add(true, "%s: link is missing", where)
			} else if err := checkURL(r.URL); err != nil {
				e.add(false, "%s: %s", where, err.Error())
			}
		case TypeApp:
			if r.App.IsZero() {
				e.add(true, "%s: app is missing", where)
			}
		case TypeFile, TypeFolder:
			switch {
			case r.Path == "" && r.Type == TypeFolder:
				e.add(true, "%s: folder is missing", where)
			case r.Path == "":
				e.add(true, "%s: file is missing", where)
			case !strings.HasPrefix(r.Path, "/") && r.Path != "~" && !strings.HasPrefix(r.Path, "~/"):
				e.add(false, "%s: path must be absolute or start with ~/", where)
			}
		default:
			e.add(false, "%s: unknown type %q", where, r.Type)
		}
		if r.Type != TypeFile && r.Type != TypeFolder && r.OpenWith != nil {
			e.add(false, "%s: openWith only applies to files and folders", where)
		}
		if r.Newest && r.Type != TypeFolder {
			e.add(false, "%s: newest only applies to folders", where)
		}
	}
	c.validateLayout(e)
	if len(e.Problems) > 0 {
		return e
	}
	return nil
}

func checkURL(raw string) error {
	if raw == "" {
		return i18n.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return i18n.Errorf("%q is not a full URL (include https://)", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "javascript", "data":
		return i18n.Errorf("%s: links are not allowed", u.Scheme)
	case "file":
		return i18n.Errorf("use a file or folder item instead of a file: link")
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
