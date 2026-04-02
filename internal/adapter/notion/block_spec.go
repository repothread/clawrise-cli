package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionBlockGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a single Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Sample: map[string]any{
				"block_id": "blk_demo",
			},
		},
	}
}

func notionBlockListChildrenSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List child blocks under a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"page_size", "page_token"},
			Sample: map[string]any{
				"block_id":  "blk_demo",
				"page_size": 50,
			},
		},
	}
}

func notionBlockGetDescendantsSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Recursively collect all descendant blocks under a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"page_size"},
			Notes: []string{
				"The adapter handles pagination and recursion internally and returns a flat depth-first list.",
			},
			Sample: map[string]any{
				"block_id":  "blk_demo",
				"page_size": 100,
			},
		},
	}
}

func notionBlockAppendSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Append child blocks to a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id", "children"},
			Optional: []string{"after"},
			Notes: []string{
				"`children` supports both shorthand top-level fields such as `text`, `rich_text`, `children`, and `checked`, and provider-native nested block bodies such as `paragraph.rich_text` and `to_do.checked`.",
				"When both shorthand and provider-native fields are present on the same block, the top-level fields take precedence.",
			},
			Sample: map[string]any{
				"block_id": "blk_demo",
				"children": []any{
					map[string]any{
						"type": "paragraph",
						"text": "Hello Clawrise",
					},
				},
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Append blocks with shorthand fields",
				Command: `clawrise notion.block.append --dry-run --json '{"block_id":"blk_demo","children":[{"type":"paragraph","text":"Hello Clawrise"}]}'`,
			},
			{
				Title:   "Append blocks with provider-native nested bodies",
				Command: `clawrise notion.block.append --dry-run --json '{"block_id":"blk_demo","children":[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello Clawrise"}}]}}]}'`,
			},
		},
	}
}

func notionBlockUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update the content of a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"type", "text", "rich_text", "children", "checked", "color", "language", "is_toggleable", "block", "in_trash"},
			Notes: []string{
				"The block payload may be provided directly at the top level or under `block`.",
				"Textual and structured fields support both shorthand top-level fields and provider-native nested block bodies.",
				"When both shorthand and provider-native fields are present on the same block, the top-level fields take precedence.",
			},
			Sample: map[string]any{
				"block_id": "blk_demo",
				"type":     "paragraph",
				"text":     "Updated paragraph",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Update a block with shorthand fields",
				Command: `clawrise notion.block.update --dry-run --json '{"block_id":"blk_demo","type":"paragraph","text":"Updated paragraph"}'`,
			},
			{
				Title:   "Update a block with a provider-native body wrapper",
				Command: `clawrise notion.block.update --dry-run --json '{"block_id":"blk_demo","block":{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Updated paragraph"}}]}}}'`,
			},
		},
	}
}

func notionBlockDeleteSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Archive a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Sample: map[string]any{
				"block_id": "blk_demo",
			},
		},
	}
}
