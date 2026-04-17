package notion

import (
	"strings"
	"testing"
)

func TestNotionDatabaseUpdateSpecDocumentsDescriptionEntryPoint(t *testing.T) {
	spec := notionDatabaseUpdateSpec()

	if !strings.Contains(spec.Description, "notion.data_source.update") {
		t.Fatalf("expected database update description to reference notion.data_source.update: %s", spec.Description)
	}
	if len(spec.Input.Notes) < 3 {
		t.Fatalf("expected database update notes to document description usage: %+v", spec.Input.Notes)
	}
	if !strings.Contains(spec.Input.Notes[1], "top-level database description") {
		t.Fatalf("expected note to mention top-level database descriptions: %+v", spec.Input.Notes)
	}
	if !strings.Contains(spec.Input.Notes[2], "rich_text") {
		t.Fatalf("expected note to explain raw rich_text support: %+v", spec.Input.Notes)
	}
	if got := spec.Input.Sample["description"]; got != "Managed by Clawrise" {
		t.Fatalf("unexpected database update description sample: %+v", spec.Input.Sample)
	}
}

func TestNotionDatabaseUpdateSpecIncludesDiscoveryAndUpdateExamples(t *testing.T) {
	spec := notionDatabaseUpdateSpec()

	if len(spec.Examples) < 5 {
		t.Fatalf("expected database update examples to cover discovery and usage: %+v", spec.Examples)
	}

	commands := make([]string, 0, len(spec.Examples))
	for _, example := range spec.Examples {
		commands = append(commands, example.Command)
	}

	assertContainsExampleCommand(t, commands, "clawrise spec list notion.database")
	assertContainsExampleCommand(t, commands, "clawrise spec get notion.database.update")
	assertContainsExampleCommand(t, commands, "clawrise notion.database.update --dry-run --json")
	assertContainsExampleCommand(t, commands, "\"description\":\"Managed by Clawrise\"")
	assertContainsExampleCommand(t, commands, "\"body\":{\"description\":[{\"type\":\"text\"")
	assertContainsExampleCommand(t, commands, "clawrise notion.database.get --json")
}

func assertContainsExampleCommand(t *testing.T, commands []string, fragment string) {
	t.Helper()
	for _, command := range commands {
		if strings.Contains(command, fragment) {
			return
		}
	}
	t.Fatalf("expected one example command to contain %q in %+v", fragment, commands)
}
