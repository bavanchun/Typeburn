package output

import (
	"encoding/json"
	"io"
)

// RenderJSON writes deterministic indented JSON followed by a newline.
func RenderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
