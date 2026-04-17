package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionDataSourceQuerySpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Query a Notion data source.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id"},
			Optional: []string{"filter_properties", "filter", "sorts", "page_size", "page_token"},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
				"page_size":      20,
			},
		},
	}
}

func notionDataSourceGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion data source schema.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id"},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
			},
		},
	}
}

func notionDataSourceTemplateListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List templates available under a Notion data source.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id"},
			Optional: []string{"name", "page_size", "page_token"},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
				"name":           "Weekly",
				"page_size":      20,
			},
		},
	}
}

func notionDataSourceCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Notion data source with provider-native payload fields.",
		Input: adapter.InputSpec{
			Required: []string{"body"},
			Notes: []string{
				"`body` is passed through to the Notion create data source API as-is.",
			},
			Sample: map[string]any{
				"body": map[string]any{
					"parent": map[string]any{
						"page_id": "page_demo",
					},
					"title": []any{
						map[string]any{
							"type": "text",
							"text": map[string]any{
								"content": "Project Tasks",
							},
						},
					},
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

func notionDataSourceUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update a Notion data source with provider-native payload fields.",
		Input: adapter.InputSpec{
			Required: []string{"data_source_id", "body"},
			Notes: []string{
				"`body` is passed through to the Notion update data source API as-is.",
				"`body.description` is not supported by Notion for data source updates; use `notion.database.update` for database descriptions.",
			},
			Sample: map[string]any{
				"data_source_id": "ds_demo",
				"body": map[string]any{
					"title": []any{
						map[string]any{
							"type": "text",
							"text": map[string]any{
								"content": "Project Tasks",
							},
						},
					},
				},
			},
		},
	}
}
