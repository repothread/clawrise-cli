package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// Entry describes one structured operation declaration.
type Entry struct {
	Operation string `json:"operation"`
}

// All returns the full built-in catalog declaration set.
func All() []Entry {
	entries := append([]Entry{}, feishuEntries()...)
	entries = append(entries, notionEntries()...)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Operation < entries[j].Operation
	})
	return entries
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

func entriesFromOperations(operations []string) []Entry {
	entries := make([]Entry, 0, len(operations))
	for _, operation := range operations {
		entries = append(entries, Entry{Operation: operation})
	}
	return entries
}
