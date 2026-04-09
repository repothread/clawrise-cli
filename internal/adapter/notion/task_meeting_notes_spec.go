package notion

import "github.com/clawrise/clawrise-cli/internal/adapter"

func notionTaskMeetingNotesGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Read one meeting notes block or discover meeting notes blocks under one page, then fetch the related summary, notes, and transcript sections.",
		Input: adapter.InputSpec{
			Optional: []string{"block_id", "page_id", "include_summary", "include_notes", "include_transcript"},
			Notes: []string{
				"Provide exactly one of `block_id` or `page_id`.",
				"When `page_id` is used, the command scans descendant blocks and collects every `meeting_notes` or legacy `transcription` block.",
				"`include_summary`, `include_notes`, and `include_transcript` default to true.",
			},
			Sample: map[string]any{
				"page_id":            "page_demo",
				"include_summary":    true,
				"include_notes":      true,
				"include_transcript": true,
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Read one meeting notes block and all related sections",
				Command: `clawrise notion.task.meeting_notes.get --json '{"block_id":"block_demo"}'`,
			},
			{
				Title:   "Discover meeting notes blocks under one page",
				Command: `clawrise notion.task.meeting_notes.get --json '{"page_id":"page_demo"}'`,
			},
		},
	}
}
