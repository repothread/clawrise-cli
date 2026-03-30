package catalog

import (
	"fmt"
	"strings"
)

// Entry describes one structured operation declaration.
type Entry struct {
	Operation string `json:"operation"`
}

// Index converts catalog entries into a lookup map and validates duplicates.
func Index(entries []Entry) (map[string]Entry, error) {
	index := make(map[string]Entry, len(entries))

	for _, entry := range entries {
		operation := strings.TrimSpace(entry.Operation)
		if operation == "" {
			return nil, fmt.Errorf("catalog entry operation is empty")
		}
		if _, exists := index[operation]; exists {
			return nil, fmt.Errorf("duplicate catalog operation: %s", operation)
		}

		entry.Operation = operation
		index[operation] = entry
	}
	return index, nil
}
