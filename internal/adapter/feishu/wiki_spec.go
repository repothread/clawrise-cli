package feishu

import "github.com/clawrise/clawrise-cli/internal/adapter"

func wikiSpaceListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List Feishu wiki spaces visible to the current execution identity.",
		Input: adapter.InputSpec{
			Optional: []string{"page_size", "page_token"},
			Sample: map[string]any{
				"page_size": 20,
			},
		},
	}
}

func wikiNodeListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List child nodes under a Feishu wiki space or parent node for the current execution identity.",
		Input: adapter.InputSpec{
			Required: []string{"space_id"},
			Optional: []string{"parent_node_token", "page_size", "page_token"},
			Sample: map[string]any{
				"space_id":          "space_demo",
				"parent_node_token": "wikcnParent",
			},
		},
	}
}

func wikiNodeCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Feishu wiki node for the current execution identity, defaulting to a docx node.",
		Input: adapter.InputSpec{
			Required: []string{"space_id"},
			Optional: []string{"obj_type", "node_type", "parent_node_token", "origin_node_token", "title"},
			Notes: []string{
				"The current implementation defaults to `obj_type=docx` and `node_type=origin`.",
			},
			Sample: map[string]any{
				"space_id":          "space_demo",
				"parent_node_token": "wikcnParent",
				"title":             "Child doc",
			},
		},
	}
}
