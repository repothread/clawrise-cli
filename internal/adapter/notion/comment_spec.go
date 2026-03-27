package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionCommentListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List open comments under a Notion page or block.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"page_size", "page_token"},
			Notes: []string{
				"`block_id` may point to a page or a block in the Notion comments API.",
			},
			Sample: map[string]any{
				"block_id":  "page_demo",
				"page_size": 20,
			},
		},
	}
}

func notionCommentCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Notion comment on a page, block, or discussion thread.",
		Input: adapter.InputSpec{
			Optional: []string{"page_id", "block_id", "discussion_id", "text", "rich_text", "attachments", "display_name"},
			Notes: []string{
				"Exactly one of page_id, block_id, or discussion_id is required.",
			},
			Sample: map[string]any{
				"page_id": "page_demo",
				"text":    "This is a comment from Clawrise.",
			},
		},
	}
}
