package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionUserGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion user object.",
		Input: adapter.InputSpec{
			Required: []string{"user_id"},
			Notes: []string{
				"Use `user_id=me` to inspect the current integration user.",
			},
			Sample: map[string]any{
				"user_id": "me",
			},
		},
	}
}
