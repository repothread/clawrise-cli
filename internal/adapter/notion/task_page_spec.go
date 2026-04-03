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

func notionTaskPagePatchSectionSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Replace one markdown section under a Notion page by exact heading or heading path, and append it when missing if requested.",
		Input: adapter.InputSpec{
			Required: []string{"page_id"},
			Optional: []string{"heading", "heading_path", "heading_level", "markdown", "file_path", "create_if_missing", "allow_truncated", "allow_unknown_blocks"},
			Notes: []string{
				"Provide exactly one of `heading` or `heading_path`.",
				"Provide exactly one of `markdown` or `file_path`; this content becomes the body of the matched section and does not need to repeat the heading line.",
				"`heading_level` is used to disambiguate one exact heading name, and also controls the level used when appending a missing section.",
				"For safety, the task rejects pages whose markdown is truncated or reports `unknown_block_ids` unless explicitly allowed.",
			},
			Sample: map[string]any{
				"page_id":           "page_demo",
				"heading_path":      []string{"Weekly Review", "Risks"},
				"heading_level":     2,
				"markdown":          "- API 限流仍需观察\n- 依赖升级待验证",
				"create_if_missing": true,
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Replace one Risks section under a page",
				Command: `clawrise notion.task.page.patch_section --json '{"page_id":"page_demo","heading":"Risks","heading_level":2,"markdown":"- API 限流仍需观察"}'`,
			},
		},
	}
}

func notionTaskPageReadCompleteSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Read one Notion page as completely as possible by combining page metadata, full property items, and recursively fetched markdown subtrees.",
		Input: adapter.InputSpec{
			Required: []string{"page_id"},
			Optional: []string{"filter_properties", "include_property_items", "property_item_page_size", "include_markdown", "include_transcript", "expand_unknown_blocks", "unknown_block_limit"},
			Notes: []string{
				"`filter_properties` narrows both the base page response and the property items that will be completed.",
				"`include_property_items` defaults to true and fetches each selected property through `notion.page.property_item.get` until pagination is exhausted.",
				"`include_markdown` defaults to true and reads `notion.page.markdown.get` for the page body.",
				"`expand_unknown_blocks` defaults to true and recursively calls the markdown endpoint again for returned `unknown_block_ids`, up to `unknown_block_limit`.",
			},
			Sample: map[string]any{
				"page_id":                "page_demo",
				"include_property_items": true,
				"include_markdown":       true,
				"expand_unknown_blocks":  true,
				"unknown_block_limit":    10,
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Read one page with full properties and markdown appendices",
				Command: `clawrise notion.task.page.read_complete --json '{"page_id":"page_demo","include_property_items":true,"include_markdown":true,"expand_unknown_blocks":true}'`,
			},
		},
	}
}
