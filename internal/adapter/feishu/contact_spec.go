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
