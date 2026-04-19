package notion

import (
	"strings"
	"testing"
)

func TestNotionPageMarkdownUpdateSpecDocumentsDeletionGuardrail(t *testing.T) {
	spec := notionPageMarkdownUpdateSpec()

	if len(spec.Input.Notes) < 5 {
		t.Fatalf("expected markdown update notes to document deletion guardrails: %+v", spec.Input.Notes)
	}
	if !strings.Contains(spec.Input.Notes[2], "allow_deleting_content") {
		t.Fatalf("expected note to mention allow_deleting_content: %+v", spec.Input.Notes)
	}
	if !strings.Contains(spec.Input.Notes[3], "notion.block.append") || !strings.Contains(spec.Input.Notes[3], "notion.block.delete") {
		t.Fatalf("expected note to recommend block-level edits: %+v", spec.Input.Notes)
	}
	if !strings.Contains(spec.Input.Notes[4], "notion.page.move") {
		t.Fatalf("expected note to mention notion.page.move: %+v", spec.Input.Notes)
	}
}

func TestNotionPageMarkdownUpdateSpecIncludesExplicitDeleteOverrideExample(t *testing.T) {
	spec := notionPageMarkdownUpdateSpec()
	if len(spec.Examples) < 2 {
		t.Fatalf("expected markdown update examples to include a destructive override example: %+v", spec.Examples)
	}

	commands := make([]string, 0, len(spec.Examples))
	for _, example := range spec.Examples {
		commands = append(commands, example.Command)
	}

	assertContainsExampleCommand(t, commands, "clawrise notion.page.markdown.update --dry-run --json")
	assertContainsExampleCommand(t, commands, "\"allow_deleting_content\":true")
}
