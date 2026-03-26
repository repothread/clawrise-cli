package output

import (
	"encoding/json"
	"io"
)

// WriteJSON prints indented JSON in a shared output format.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
