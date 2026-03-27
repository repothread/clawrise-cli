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

func notionBlockAppendSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Append child blocks to a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id", "children"},
			Optional: []string{"after"},
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
	}
}

func notionBlockUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update the content of a Notion block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"type", "text", "rich_text", "children", "checked", "color", "language", "is_toggleable", "block", "in_trash"},
			Sample: map[string]any{
				"block_id": "blk_demo",
				"type":     "paragraph",
				"text":     "Updated paragraph",
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
