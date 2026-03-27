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
