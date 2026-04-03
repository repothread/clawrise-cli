package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionTaskPageImportMarkdownSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a child Notion page from markdown text or one local Markdown file.",
		Input: adapter.InputSpec{
			Required: []string{"parent_page_id"},
			Optional: []string{"title", "markdown", "file_path", "position", "after", "template"},
			Notes: []string{
				"Provide exactly one of `markdown` or `file_path`.",
				"`title` is optional when the markdown body already contains a leading H1 that Notion can promote into the page title.",
				"`position`, `after`, and `template` are forwarded to `notion.page.create`.",
			},
			Sample: map[string]any{
				"parent_page_id": "page_demo",
				"file_path":      "./weekly.md",
				"title":          "Weekly Notes",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Import one local markdown file as a child page",
				Command: `clawrise notion.task.page.import_markdown --json '{"parent_page_id":"page_demo","file_path":"./weekly.md","title":"Weekly Notes"}'`,
			},
		},
	}
}

func notionTaskPageUpsertMarkdownChildSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Find a child page by exact title under one parent page, then replace its markdown body or create it when missing.",
		Input: adapter.InputSpec{
			Required: []string{"parent_page_id", "title"},
			Optional: []string{"markdown", "file_path", "create_if_missing", "search_page_size", "position", "after", "template"},
			Notes: []string{
				"Provide exactly one of `markdown` or `file_path`.",
				"`create_if_missing` defaults to true.",
				"The existing page path always uses `notion.page.markdown.update` with `replace_content` so the final body matches the provided markdown source.",
			},
			Sample: map[string]any{
				"parent_page_id":    "page_demo",
				"title":             "2026-04-03 Daily Notes",
				"file_path":         "./daily.md",
				"create_if_missing": true,
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Create or replace one titled child page from a local Markdown file",
				Command: `clawrise notion.task.page.upsert_markdown_child --json '{"parent_page_id":"page_demo","title":"2026-04-03 Daily Notes","file_path":"./daily.md"}'`,
			},
		},
	}
}
