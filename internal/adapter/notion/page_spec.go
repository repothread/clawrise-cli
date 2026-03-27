package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionPageCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Notion page.",
		Input: adapter.InputSpec{
			Required: []string{"title", "parent"},
			Optional: []string{"title_property", "properties", "children"},
			Notes: []string{
				"When using a public integration, `parent` may be omitted for workspace-level page creation.",
			},
			Sample: map[string]any{
				"title": "Project notes",
				"parent": map[string]any{
					"type": "page_id",
					"id":   "page_demo",
				},
			},
		},
	}
}

func notionPageGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion page.",
		Input: adapter.InputSpec{
			Required: []string{"page_id"},
			Sample: map[string]any{
				"page_id": "page_demo",
			},
		},
	}
}

func notionPagePropertyItemGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion page property item and preserve provider-native structure.",
		Input: adapter.InputSpec{
			Required: []string{"page_id", "property_id"},
			Optional: []string{"page_size", "page_token"},
			Sample: map[string]any{
				"page_id":     "page_demo",
				"property_id": "prop_demo",
				"page_size":   20,
			},
		},
	}
}

func notionPageUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update a Notion page's properties or archive state.",
		Input: adapter.InputSpec{
			Required: []string{"page_id"},
			Optional: []string{"title", "title_property", "properties", "archived", "icon", "cover"},
			Sample: map[string]any{
				"page_id": "page_demo",
				"title":   "Updated project notes",
			},
		},
	}
}

func notionPageMarkdownGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get Notion page content as markdown.",
		Input: adapter.InputSpec{
			Required: []string{"page_id"},
			Optional: []string{"include_transcript"},
			Sample: map[string]any{
				"page_id":            "page_demo",
				"include_transcript": true,
			},
		},
	}
}

func notionPageMarkdownUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update Notion page content with markdown commands.",
		Input: adapter.InputSpec{
			Required: []string{"page_id", "type"},
			Optional: []string{"update_content", "replace_content", "insert_content", "replace_content_range"},
			Notes: []string{
				"`type` selects which command payload is required.",
			},
			Sample: map[string]any{
				"page_id": "page_demo",
				"type":    "replace_content",
				"replace_content": map[string]any{
					"new_str": "# Updated title",
				},
			},
		},
	}
}
