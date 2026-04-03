package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionTaskDataSourceRowUpsertSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Find one row in a Notion data source with a provider-native match filter, then update it or create it when missing.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id", "match"},
			Optional: []string{"title", "title_property", "properties", "markdown", "file_path", "create_if_missing", "page_size", "filter_properties"},
			Notes: []string{
				"`match` is forwarded as the Notion data source query filter and should uniquely identify one row page.",
				"Provide exactly one of `markdown` or `file_path` when you want to replace the row page body.",
				"Provide `title` with `title_property`, or include a provider-native title property inside `properties`, when a create path must populate the data source title field.",
				"`create_if_missing` defaults to true.",
			},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
				"match": map[string]any{
					"property": "External ID",
					"rich_text": map[string]any{
						"equals": "crm_123",
					},
				},
				"title":          "Acme Corp",
				"title_property": "Name",
				"properties": map[string]any{
					"Status": map[string]any{
						"select": map[string]any{
							"name": "Active",
						},
					},
				},
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Create or update one row by exact external id",
				Command: `clawrise notion.task.data_source.row.upsert --json '{"data_source_id":"ds_demo","match":{"property":"External ID","rich_text":{"equals":"crm_123"}},"title":"Acme Corp","title_property":"Name","properties":{"Status":{"select":{"name":"Active"}}}}'`,
			},
			{
				Title:   "Create or update one row and replace its page body from Markdown",
				Command: `clawrise notion.task.data_source.row.upsert --json '{"data_source_id":"ds_demo","match":{"property":"External ID","rich_text":{"equals":"crm_123"}},"title":"Acme Corp","title_property":"Name","file_path":"./customer.md"}'`,
			},
		},
	}
}
