package notion

import (
	"context"
	"net/http"
	"testing"
)

func TestGetMeetingNotesByBlockID(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/blocks/mn_block_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "mn_block_1",
					"type":         "meeting_notes",
					"has_children": false,
					"meeting_notes": map[string]any{
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "周会",
								"text": map[string]any{
									"content": "周会",
								},
							},
						},
						"status": "processed",
						"children": map[string]any{
							"summary_block_id":    "mn_summary_1",
							"notes_block_id":      "mn_notes_1",
							"transcript_block_id": "mn_transcript_1",
						},
					},
				}), nil
			case "/v1/blocks/mn_summary_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "mn_summary_1",
					"type":         "paragraph",
					"has_children": false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "会议摘要"},
						},
					},
				}), nil
			case "/v1/blocks/mn_notes_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "mn_notes_1",
					"type":         "bulleted_list_item",
					"has_children": false,
					"bulleted_list_item": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "行动项 1"},
						},
					},
				}), nil
			case "/v1/blocks/mn_transcript_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "mn_transcript_1",
					"type":         "paragraph",
					"has_children": true,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "发言开头"},
						},
					},
				}), nil
			case "/v1/blocks/mn_transcript_1/children":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "mn_transcript_child_1",
							"type":         "paragraph",
							"has_children": false,
							"paragraph": map[string]any{
								"rich_text": []map[string]any{
									{"plain_text": "发言续段"},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.GetMeetingNotes(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "mn_block_1",
	})
	if appErr != nil {
		t.Fatalf("GetMeetingNotes returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("unexpected meeting notes item count: %+v", items)
	}
	item := items[0]
	if item["type"] != "meeting_notes" || item["title"] != "周会" {
		t.Fatalf("unexpected meeting notes payload: %+v", item)
	}
	transcript := item["transcript"].(map[string]any)
	if transcript["plain_text"] != "发言开头\n发言续段" {
		t.Fatalf("unexpected transcript plain_text: %+v", transcript)
	}
}

func TestGetMeetingNotesByPageIDDiscoversLegacyTranscriptionBlocks(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/blocks/page_demo/children":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "legacy_transcription_1",
							"type":         "transcription",
							"has_children": false,
							"transcription": map[string]any{
								"title": []map[string]any{
									{
										"type":       "text",
										"plain_text": "历史会议记录",
										"text": map[string]any{
											"content": "历史会议记录",
										},
									},
								},
								"status": "processed",
								"children": map[string]any{
									"summary_block_id": "legacy_summary_1",
								},
							},
						},
					},
					"has_more": false,
				}), nil
			case "/v1/blocks/legacy_summary_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "legacy_summary_1",
					"type":         "paragraph",
					"has_children": false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"plain_text": "历史摘要"},
						},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.GetMeetingNotes(context.Background(), testStaticProfile(), map[string]any{
		"page_id":            "page_demo",
		"include_notes":      false,
		"include_transcript": false,
	})
	if appErr != nil {
		t.Fatalf("GetMeetingNotes returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("unexpected discovered meeting notes count: %+v", items)
	}
	item := items[0]
	if item["type"] != "meeting_notes" || item["title"] != "历史会议记录" {
		t.Fatalf("unexpected discovered meeting notes payload: %+v", item)
	}
	if _, exists := item["notes"]; exists {
		t.Fatalf("did not expect notes section when include_notes=false: %+v", item)
	}
}

func TestGetMeetingNotesRejectsMissingSelectors(t *testing.T) {
	client := newTestClient(t, nil)

	_, appErr := client.GetMeetingNotes(context.Background(), testStaticProfile(), map[string]any{})
	if appErr == nil {
		t.Fatal("expected GetMeetingNotes to reject missing selectors")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}
