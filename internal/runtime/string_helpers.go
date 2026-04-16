package runtime

import "strings"

func supportsWriteEnhancement(operation string) bool {
	switch strings.TrimSpace(operation) {
	case "notion.page.create", "notion.page.update", "notion.block.append", "notion.block.update":
		return true
	default:
		return false
	}
}

func formatRuntimeCapabilityLabel(name string, id string, collapseDuplicate bool) string {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name != "" && id != "" && (!collapseDuplicate || name != id):
		return name + "/" + id
	case id != "":
		return id
	default:
		return name
	}
}
