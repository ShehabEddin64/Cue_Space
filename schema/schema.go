// Package schema embeds the Cue JSON Schema, the public contract for .cue.json files.
package schema

import _ "embed"

//go:embed cue.v1.json
var CueV1 []byte
