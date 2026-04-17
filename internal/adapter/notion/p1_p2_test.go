package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetPagePropertyItemSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_123/properties/prop_456" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.URL.Query().Get("page_size") != "10" {
				t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
			}
			if request.URL.Query().Get("start_cursor") != "cursor_demo" {
				t.Fatalf("unexpected start_cursor: %s", request.URL.Query().Get("start_cursor"))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":      "list",
				"results":     []map[string]any{{"object": "property_item", "id": "pi_1", "type": "people"}},
				"next_cursor": "cursor_next",
				"has_more":    true,
			}), nil
		},
	})

	data, appErr := client.GetPagePropertyItem(context.Background(), testStaticProfile(), map[string]any{
		"page_id":     "page_123",
		"property_id": "prop_456",
		"page_size":   10,
		"page_token":  "cursor_demo",
	})
	if appErr != nil {
		t.Fatalf("GetPagePropertyItem returned error: %+v", appErr)
	}
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["id"] != "pi_1" {
		t.Fatalf("unexpected property items: %+v", data["items"])
	}
	if data["next_page_token"] != "cursor_next" {
		t.Fatalf("unexpected next_page_token: %+v", data["next_page_token"])
	}
}

func TestListUsersSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/users" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.URL.Query().Get("page_size") != "5" {
				t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{
						"id":         "user_1",
						"type":       "person",
						"name":       "Alice",
						"avatar_url": "https://example.com/alice.png",
						"person": map[string]any{
							"email": "alice@example.com",
						},
					},
				},
				"has_more": false,
			}), nil
		},
	})

	data, appErr := client.ListUsers(context.Background(), testStaticProfile(), map[string]any{
		"page_size": 5,
	})
	if appErr != nil {
		t.Fatalf("ListUsers returned error: %+v", appErr)
	}
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["email"] != "alice@example.com" {
		t.Fatalf("unexpected users: %+v", data["items"])
	}
}

func TestGetBlockDescendantsSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/blocks/root_block/children":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "child_1",
							"type":         "paragraph",
							"has_children": true,
							"paragraph": map[string]any{
								"rich_text": []map[string]any{
									{"plain_text": "first child"},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			case "/v1/blocks/child_1/children":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "grandchild_1",
							"type":         "to_do",
							"has_children": false,
							"to_do": map[string]any{
								"checked": true,
								"rich_text": []map[string]any{
									{"plain_text": "nested todo"},
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

	data, appErr := client.GetBlockDescendants(context.Background(), testStaticProfile(), map[string]any{
		"block_id":  "root_block",
		"page_size": 50,
	})
	if appErr != nil {
		t.Fatalf("GetBlockDescendants returned error: %+v", appErr)
	}
	items := data["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("unexpected descendant count: %d", len(items))
	}
	if items[1]["block_id"] != "grandchild_1" {
		t.Fatalf("unexpected descendant order: %+v", items)
	}
}

func TestDataSourceWriteOperationsSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/databases":
				if request.Method != http.MethodPost {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create database payload: %v", err)
				}
				if _, ok := payload["initial_data_source"].(map[string]any); !ok {
					t.Fatalf("expected initial_data_source in create payload: %+v", payload)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id": "db_123",
					"data_sources": []map[string]any{
						{
							"id": "ds_123",
						},
					},
				}), nil
			case "/v1/data_sources/ds_123":
				if request.Method == http.MethodGet {
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":  "ds_123",
						"url": "https://www.notion.so/ds_123",
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "Project Tasks",
								"text": map[string]any{
									"content": "Project Tasks",
								},
							},
						},
						"properties": map[string]any{
							"Name": map[string]any{
								"type": "title",
							},
						},
					}), nil
				}
				if request.Method != http.MethodPatch {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode update data source payload: %v", err)
				}
				if _, ok := payload["title"].([]any); !ok {
					t.Fatalf("unexpected update payload: %+v", payload)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":  "ds_123",
					"url": "https://www.notion.so/ds_123",
					"title": []map[string]any{
						{
							"type":       "text",
							"plain_text": "Project Tasks",
							"text": map[string]any{
								"content": "Project Tasks",
							},
						},
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"type": "title",
						},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	created, appErr := client.CreateDataSource(context.Background(), testStaticProfile(), map[string]any{
		"body": map[string]any{
			"parent": map[string]any{
				"page_id": "page_demo",
			},
			"title": []any{
				map[string]any{
					"type": "text",
					"text": map[string]any{
						"content": "Project Tasks",
					},
				},
			},
			"properties": map[string]any{
				"Name": map[string]any{
					"title": map[string]any{},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreateDataSource returned error: %+v", appErr)
	}
	if created["data_source_id"] != "ds_123" {
		t.Fatalf("unexpected create result: %+v", created)
	}

	updated, appErr := client.UpdateDataSource(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_123",
		"body": map[string]any{
			"title": []any{
				map[string]any{
					"type": "text",
					"text": map[string]any{
						"content": "Project Tasks Updated",
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("UpdateDataSource returned error: %+v", appErr)
	}
	if updated["data_source_id"] != "ds_123" {
		t.Fatalf("unexpected update result: %+v", updated)
	}
}

func TestGetCommentSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/comments/cmt_123" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":            "cmt_123",
				"discussion_id": "discussion_1",
				"parent": map[string]any{
					"type":    "page_id",
					"page_id": "page_demo",
				},
				"rich_text": []map[string]any{
					{
						"type":       "text",
						"plain_text": "Loaded comment",
						"text": map[string]any{
							"content": "Loaded comment",
						},
					},
				},
			}), nil
		},
	})

	data, appErr := client.GetComment(context.Background(), testStaticProfile(), map[string]any{
		"comment_id": "cmt_123",
	})
	if appErr != nil {
		t.Fatalf("GetComment returned error: %+v", appErr)
	}
	if data["plain_text"] != "Loaded comment" {
		t.Fatalf("unexpected comment plain_text: %+v", data["plain_text"])
	}
}
