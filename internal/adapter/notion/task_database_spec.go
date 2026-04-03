package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionTaskDatabaseResolveTargetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Resolve one Notion database target from an id, URL, page, or data source, and expose the related database/data source context.",
		Input: adapter.InputSpec{
			Optional: []string{"target", "url", "database_id", "data_source_id", "page_id", "data_source_name"},
			Notes: []string{
				"Provide exactly one of `target`, `url`, `database_id`, `data_source_id`, or `page_id`.",
				"`target` accepts either a raw Notion id or a Notion URL.",
				"When the resolved database exposes exactly one child data source, the task also returns it as the default resolved data source.",
				"When a database has multiple child data sources, `data_source_name` can pick one by exact name.",
			},
			Sample: map[string]any{
				"target":           "https://www.notion.so/workspace/Project-Hub-0123456789abcdef0123456789abcdef",
				"data_source_name": "All Projects",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Resolve a database from a Notion URL",
				Command: `clawrise notion.task.database.resolve_target --json '{"target":"https://www.notion.so/workspace/Project-Hub-0123456789abcdef0123456789abcdef"}'`,
			},
			{
				Title:   "Resolve a row page back to its parent data source and database",
				Command: `clawrise notion.task.database.resolve_target --json '{"page_id":"page_demo"}'`,
			},
		},
	}
}
