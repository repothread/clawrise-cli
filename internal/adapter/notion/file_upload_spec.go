package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionFileUploadCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Notion file upload.",
		Input: adapter.InputSpec{
			Optional: []string{"mode", "filename", "content_type", "number_of_parts", "external_url"},
			Notes: []string{
				"`mode` defaults to `single_part` and can also be `multi_part` or `external_url`.",
				"`filename` is required for `multi_part` uploads.",
				"`external_url` is required when `mode` is `external_url`.",
			},
			Sample: map[string]any{
				"mode":         "single_part",
				"filename":     "demo.txt",
				"content_type": "text/plain",
			},
		},
	}
}

func notionFileUploadGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get a Notion file upload.",
		Input: adapter.InputSpec{
			Required: []string{"file_upload_id"},
			Sample: map[string]any{
				"file_upload_id": "fu_demo",
			},
		},
	}
}

func notionFileUploadListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List Notion file uploads visible to the current integration.",
		Input: adapter.InputSpec{
			Optional: []string{"status", "page_size", "page_token"},
			Sample: map[string]any{
				"status":    "uploaded",
				"page_size": 20,
			},
		},
	}
}

func notionFileUploadSendSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Send file content to a Notion file upload.",
		Input: adapter.InputSpec{
			Required: []string{"file_upload_id"},
			Optional: []string{"file_path", "content_base64", "filename", "content_type", "part_number"},
			Notes: []string{
				"Provide exactly one of `file_path` or `content_base64`.",
				"`filename` is optional for `file_path`, but required when sending `content_base64`.",
				"`part_number` is used for one part of a multi-part upload.",
			},
			Sample: map[string]any{
				"file_upload_id": "fu_demo",
				"file_path":      "/tmp/demo.txt",
				"content_type":   "text/plain",
			},
		},
	}
}

func notionFileUploadCompleteSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Complete a Notion multi-part file upload.",
		Input: adapter.InputSpec{
			Required: []string{"file_upload_id"},
			Sample: map[string]any{
				"file_upload_id": "fu_demo",
			},
		},
	}
}
