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
		Summary:     "Update a Notion database, including its top-level description.",
		Description: "Use this operation when you need to change database-level metadata such as the database title or top-level description. `notion.data_source.update` cannot update `body.description` for a database's underlying data source.",
		Input: adapter.InputSpec{
			Required: []string{"database_id"},
			Optional: []string{"body", "parent", "title", "description", "in_trash", "is_locked", "icon", "cover"},
			Notes: []string{
				"Provide either `body`, or one or more top-level shorthand fields for common database updates.",
				"Use this operation for top-level database description updates; `notion.data_source.update` only supports property-level description changes.",
				"`description` accepts either a plain string or a provider-native rich_text array. Use `body.description` only when you want to send the raw Notion patch payload yourself.",
				"`parent.type` supports `page_id`, `database_id`, or `workspace`.",
			},
			Sample: map[string]any{
				"database_id": "db_demo",
				"title":       "Project Hub",
				"description": "Managed by Clawrise",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "List database operations from the Notion capability tree",
				Command: `clawrise spec list notion.database`,
			},
			{
				Title:   "Inspect the database update input contract",
				Command: `clawrise spec get notion.database.update`,
			},
			{
				Title:   "Update a database description with shorthand text",
				Command: `clawrise notion.database.update --dry-run --json '{"database_id":"db_demo","description":"Managed by Clawrise"}'`,
			},
			{
				Title:   "Update a database description with provider-native rich text",
				Command: `clawrise notion.database.update --dry-run --json '{"database_id":"db_demo","body":{"description":[{"type":"text","text":{"content":"Managed by Clawrise"}}]}}'`,
			},
			{
				Title:   "Read the database back after the update",
				Command: `clawrise notion.database.get --json '{"database_id":"db_demo"}'`,
			},
		},
	}
}
