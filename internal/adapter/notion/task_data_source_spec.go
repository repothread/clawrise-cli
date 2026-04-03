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

func notionTaskDataSourceBulkUpsertSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Bulk create or update multiple rows under one Notion data source by repeatedly applying row upsert semantics and returning per-item results.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id", "items"},
			Optional: []string{"title_property", "create_if_missing", "page_size", "filter_properties", "stop_on_error"},
			Notes: []string{
				"Each `items` entry follows the same shape as `notion.task.data_source.row.upsert`, except `data_source_id` is inherited from the outer request.",
				"Outer optional fields such as `title_property`, `create_if_missing`, `page_size`, and `filter_properties` are used as defaults when one item does not override them.",
				"`stop_on_error` defaults to false so the task can return partial success details for AI batch sync flows.",
			},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
				"title_property": "Name",
				"items": []any{
					map[string]any{
						"match": map[string]any{
							"property": "External ID",
							"rich_text": map[string]any{
								"equals": "crm_123",
							},
						},
						"title": "Acme Corp",
						"properties": map[string]any{
							"Status": map[string]any{
								"select": map[string]any{
									"name": "Active",
								},
							},
						},
					},
				},
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Bulk upsert two CRM rows",
				Command: `clawrise notion.task.data_source.bulk_upsert --json '{"data_source_id":"ds_demo","title_property":"Name","items":[{"match":{"property":"External ID","rich_text":{"equals":"crm_123"}},"title":"Acme Corp"},{"match":{"property":"External ID","rich_text":{"equals":"crm_456"}},"title":"Globex"}]}'`,
			},
		},
	}
}
