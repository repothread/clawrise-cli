package feishu

import "github.com/clawrise/clawrise-cli/internal/adapter"

func contactUserGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Feishu user profile.",
		Input: adapter.InputSpec{
			Required: []string{"user_id"},
			Sample: map[string]any{
				"user_id": "ou_demo",
			},
		},
	}
}

func contactUserSearchSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Search visible Feishu users by partial identity input.",
		Input: adapter.InputSpec{
			Required: []string{"query"},
			Optional: []string{"department_id", "department_id_type", "user_id_type", "page_size", "page_token"},
			Notes: []string{
				"`page_token` is an opaque cursor returned by this operation.",
				"Set `department_id` when you want to narrow search scope to a specific department.",
			},
			Sample: map[string]any{
				"query":     "demo@example.com",
				"page_size": 10,
			},
		},
	}
}

func contactDepartmentListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List child departments under a Feishu department.",
		Input: adapter.InputSpec{
			Optional: []string{"department_id", "department_id_type", "user_id_type", "fetch_child", "page_size", "page_token"},
			Notes: []string{
				"When `department_id` is omitted, the operation starts from the root department `0`.",
			},
			Sample: map[string]any{
				"department_id": "0",
				"page_size":     10,
			},
		},
	}
}

func departmentUserListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List direct users under a Feishu department.",
		Input: adapter.InputSpec{
			Required: []string{"department_id"},
			Optional: []string{"department_id_type", "user_id_type", "page_size", "page_token"},
			Sample: map[string]any{
				"department_id": "od-demo",
				"page_size":     10,
			},
		},
	}
}
