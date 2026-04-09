package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionTaskBlockAttachFileSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Upload one local file or base64 payload to Notion and append it as an image or file block in one step.",
		Input: adapter.InputSpec{
			Required: []string{"block_id"},
			Optional: []string{"file_path", "content_base64", "filename", "content_type", "block_type", "caption", "caption_rich_text", "position", "after"},
			Notes: []string{
				"Provide exactly one of `file_path` or `content_base64`.",
				"`block_type` may be `image` or `file`; when omitted, the command infers `image` from an image content type and falls back to `file` otherwise.",
				"`filename` is required when `content_base64` is used and optional for `file_path`.",
				"`position` and `after` are forwarded to `notion.block.append`.",
			},
			Sample: map[string]any{
				"block_id":  "page_demo",
				"file_path": "/tmp/screenshot.png",
				"caption":   "最新截图",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Upload one image file and append it to a page",
				Command: `clawrise notion.task.block.attach_file --json '{"block_id":"page_demo","file_path":"/tmp/screenshot.png","caption":"最新截图"}'`,
			},
			{
				Title:   "Upload base64 content as a generic file block",
				Command: `clawrise notion.task.block.attach_file --json '{"block_id":"page_demo","content_base64":"SGVsbG8=","filename":"demo.txt","block_type":"file"}'`,
			},
		},
	}
}
