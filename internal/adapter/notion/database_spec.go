package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionDatabaseGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion database and expose its child data sources.",
		Input: adapter.InputSpec{
			Required: []string{"database_id"},
			Sample: map[string]any{
				"database_id": "db_demo",
			},
		},
	}
}

func notionDatabaseCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Notion database.",
		Input: adapter.InputSpec{
			Optional: []string{"body", "parent", "title", "description", "is_inline", "icon", "cover", "initial_data_source"},
			Notes: []string{
				"Provide either `body`, or the top-level shorthand fields for common database creation.",
				"When shorthand fields are used, `parent`, `title`, and `initial_data_source` are required.",
				"`parent.type` supports `page_id`, `database_id`, or `workspace`.",
			},
			Sample: map[string]any{
				"parent": map[string]any{
					"type": "page_id",
					"id":   "page_demo",
				},
				"title": "Project Hub",
				"initial_data_source": map[string]any{
					"name": "All Projects",
					"properties": map[string]any{
						"Name": map[string]any{
							"title": map[string]any{},
						},
					},
				},
			},
		},
	}
}

func notionDatabaseUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update a Notion database.",
		Input: adapter.InputSpec{
			Required: []string{"database_id"},
			Optional: []string{"body", "parent", "title", "description", "in_trash", "is_locked", "icon", "cover"},
			Notes: []string{
				"Provide either `body`, or one or more top-level shorthand fields for common database updates.",
				"`parent.type` supports `page_id`, `database_id`, or `workspace`.",
			},
			Sample: map[string]any{
				"database_id": "db_demo",
				"title":       "Project Hub",
				"description": "Managed by Clawrise",
			},
		},
	}
}
