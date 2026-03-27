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
