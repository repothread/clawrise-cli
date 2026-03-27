package feishu

import "github.com/clawrise/clawrise-cli/internal/adapter"

func calendarEventCreateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Create a Feishu calendar event.",
		Input: adapter.InputSpec{
			Required: []string{"calendar_id", "summary", "start_at", "end_at"},
			Optional: []string{"description", "location", "reminders", "timezone"},
			Notes: []string{
				"Time fields use RFC3339.",
				"`attendees` is not supported in the current implementation.",
			},
			Sample: map[string]any{
				"calendar_id": "cal_demo",
				"summary":     "Weekly sync",
				"start_at":    "2026-03-30T10:00:00+08:00",
				"end_at":      "2026-03-30T11:00:00+08:00",
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "Create an event",
				Command: `clawrise feishu.calendar.event.create --json '{"calendar_id":"cal_demo","summary":"Weekly sync","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}'`,
			},
		},
	}
}

func calendarEventListSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "List calendar events in a time range.",
		Input: adapter.InputSpec{
			Required: []string{"calendar_id"},
			Optional: []string{"start_at_from", "start_at_to", "page_size", "page_token"},
			Sample: map[string]any{
				"calendar_id": "cal_demo",
				"page_size":   20,
			},
		},
		Examples: []adapter.ExampleSpec{
			{
				Title:   "List events",
				Command: `clawrise feishu.calendar.event.list --json '{"calendar_id":"cal_demo","page_size":20}'`,
			},
		},
	}
}

func calendarEventGetSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Get one Feishu calendar event.",
		Input: adapter.InputSpec{
			Required: []string{"calendar_id", "event_id"},
			Sample: map[string]any{
				"calendar_id": "cal_demo",
				"event_id":    "evt_demo",
			},
		},
	}
}

func calendarEventUpdateSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Update a Feishu calendar event.",
		Input: adapter.InputSpec{
			Required: []string{"calendar_id", "event_id"},
			Optional: []string{"summary", "description", "start_at", "end_at", "location", "reminders", "timezone"},
			Sample: map[string]any{
				"calendar_id": "cal_demo",
				"event_id":    "evt_demo",
				"summary":     "Updated weekly sync",
			},
		},
	}
}

func calendarEventDeleteSpec() adapter.OperationSpec {
	return adapter.OperationSpec{
		Summary: "Delete a Feishu calendar event.",
		Input: adapter.InputSpec{
			Required: []string{"calendar_id", "event_id"},
			Sample: map[string]any{
				"calendar_id": "cal_demo",
				"event_id":    "evt_demo",
			},
		},
	}
}
