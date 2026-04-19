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
			Optional: []string{"position", "after"},
			Notes: []string{
				"`children` supports both shorthand top-level fields such as `text`, `rich_text`, `children`, and `checked`, and provider-native nested block bodies such as `paragraph.rich_text` and `to_do.checked`.",
				"`position` supports `start`, `end`, or `after_block`; `after` remains accepted as a backward-compatible alias for `position.type=after_block`.",
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
				Title:   "Insert blocks at the start of one block's children",
				Command: `clawrise notion.block.append --dry-run --json '{"block_id":"blk_demo","position":{"type":"start"},"children":[{"type":"paragraph","text":"Hello Clawrise"}]}'`,
			},
			{
				Title:   "Append blocks with provider-native nested bodies",
				Command: `clawrise notion.block.append --dry-run --json '{"block_id":"blk_demo","children":[{"type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello Clawrise"}}]}}]}'`,
			},
			{
				Title:   "Append blocks with provider payload debug and verification",
				Command: `clawrise notion.block.append --debug-provider-payload --verify --json '{"block_id":"blk_demo","children":[{"type":"paragraph","text":"Hello Clawrise"}]}'`,
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
			{
				Title:   "Update a block with provider payload debug and verification",
				Command: `clawrise notion.block.update --debug-provider-payload --verify --json '{"block_id":"blk_demo","type":"paragraph","text":"Updated paragraph"}'`,
			},
		},
	}
}

func notionBlockDeleteSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Archive a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"allow_child_page_delete"},
			Notes: []string{
				"Deleting a `child_page` block can archive/trash the underlying Notion page, not just remove one visual entry.",
				"Set `allow_child_page_delete=true` only when you intentionally want that destructive behavior.",
				"If the integration lacks Notion read content capability, the adapter cannot inspect the block type safely and will also require `allow_child_page_delete=true` before deleting.",
			},
			Sample: map[string]any{
				"block_id": "blk_demo",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Delete a regular block",
				Command: `clawrise notion.block.delete --json '{"block_id":"blk_demo"}'`,
			},
			{
				Title:   "Explicitly allow deleting a child_page block",
				Command: `clawrise notion.block.delete --json '{"block_id":"blk_child_page","allow_child_page_delete":true}'`,
			},
		},
	}
}
