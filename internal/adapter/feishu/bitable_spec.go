package feishu

import "github.com/clawrise/clawrise-cli/internal/adapter"

func bitableRecordListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List records from a Feishu Bitable table.",
		Input: adapter.InputSpec{
			Required: []string{"app_token", "table_id"},
			Optional: []string{"view_id", "filter", "sort", "page_size", "page_token"},
			Sample: map[string]any{
				"app_token": "app_demo",
				"table_id":  "tbl_demo",
				"page_size": 20,
			},
		},
	}
}

func bitableRecordGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get one Feishu Bitable record.",
		Input: adapter.InputSpec{
			Required: []string{"app_token", "table_id", "record_id"},
			Sample: map[string]any{
				"app_token": "app_demo",
				"table_id":  "tbl_demo",
				"record_id": "rec_demo",
			},
		},
	}
}

func bitableRecordCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create one Feishu Bitable record.",
		Input: adapter.InputSpec{
			Required: []string{"app_token", "table_id", "fields"},
			Sample: map[string]any{
				"app_token": "app_demo",
				"table_id":  "tbl_demo",
				"fields": map[string]any{
					"Title": "Task A",
				},
			},
		},
	}
}

func bitableRecordUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update one Feishu Bitable record.",
		Input: adapter.InputSpec{
			Required: []string{"app_token", "table_id", "record_id", "fields"},
			Sample: map[string]any{
				"app_token": "app_demo",
				"table_id":  "tbl_demo",
				"record_id": "rec_demo",
				"fields": map[string]any{
					"Status": "Done",
				},
			},
		},
	}
}
