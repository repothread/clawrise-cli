package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionSearchQuerySpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Search pages and data sources visible to the current Notion integration.",
		Input: adapter.InputSpec{
			Optional: []string{"query", "filter", "sort", "page_size", "page_token"},
			Sample: map[string]any{
				"query":     "project",
				"page_size": 20,
			},
		},
	}
}
