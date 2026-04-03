package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpdatePageSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_123" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPatch {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update page payload: %v", err)
			}
			properties := payload["properties"].(map[string]any)
			titleProperty := properties["title"].(map[string]any)
			titleItems := titleProperty["title"].([]any)
			titleText := titleItems[0].(map[string]any)["text"].(map[string]any)["content"]
			if titleText != "Updated title" {
				t.Fatalf("unexpected title text: %+v", titleText)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":       "page_123",
				"url":      "https://www.notion.so/page_123",
				"in_trash": false,
				"parent": map[string]any{
					"type":    "page_id",
					"page_id": "parent_demo",
				},
				"properties": map[string]any{
					"title": map[string]any{
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "Updated title",
								"text": map[string]any{
									"content": "Updated title",
								},
							},
						},
					},
				},
			}), nil
		},
	})

	data, appErr := client.UpdatePage(context.Background(), testStaticProfile(), map[string]any{
		"page_id": "page_123",
		"title":   "Updated title",
	})
	if appErr != nil {
		t.Fatalf("UpdatePage returned error: %+v", appErr)
	}
	if data["title"] != "Updated title" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestGetDataSourceSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/data_sources/ds_123" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object": "data_source",
				"id":     "ds_123",
				"url":    "https://www.notion.so/ds_123",
				"title": []map[string]any{
					{
						"type":       "text",
						"plain_text": "Project DB",
						"text": map[string]any{
							"content": "Project DB",
						},
					},
				},
				"properties": map[string]any{
					"Name": map[string]any{
						"type": "title",
					},
				},
			}), nil
		},
	})

	data, appErr := client.GetDataSource(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_123",
	})
	if appErr != nil {
		t.Fatalf("GetDataSource returned error: %+v", appErr)
	}
	if data["title"] != "Project DB" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestAppendBlockChildrenSupportsExtendedTypes(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo/children" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode append payload: %v", err)
			}
			children := payload["children"].([]any)
			if len(children) != 5 {
				t.Fatalf("unexpected children count: %d", len(children))
			}
			table := children[4].(map[string]any)["table"].(map[string]any)
			if table["table_width"] != float64(2) {
				t.Fatalf("unexpected table_width: %+v", table["table_width"])
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"id": "blk_1"},
					{"id": "blk_2"},
					{"id": "blk_3"},
					{"id": "blk_4"},
					{"id": "blk_5"},
				},
			}), nil
		},
	})

	data, appErr := client.AppendBlockChildren(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"children": []any{
			map[string]any{
				"type": "toggle",
				"text": "Toggle item",
			},
			map[string]any{
				"type":  "callout",
				"text":  "Callout body",
				"emoji": "💡",
			},
			map[string]any{
				"type": "image",
				"url":  "https://example.com/demo.png",
			},
			map[string]any{
				"type": "file",
				"url":  "https://example.com/demo.txt",
			},
			map[string]any{
				"type":              "table",
				"has_column_header": true,
				"table_width":       2,
				"rows": []any{
					map[string]any{
						"type":  "table_row",
						"cells": []any{"H1", "H2"},
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("AppendBlockChildren returned error: %+v", appErr)
	}
	if data["appended_count"] != 5 {
		t.Fatalf("unexpected appended_count: %+v", data["appended_count"])
	}
}

func TestAppendBlockChildrenSupportsFileUploadBackedMediaBlocks(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo/children" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode append payload: %v", err)
			}
			children := payload["children"].([]any)
			imageBody := children[0].(map[string]any)["image"].(map[string]any)
			if imageBody["type"] != "file_upload" || imageBody["file_upload"].(map[string]any)["id"] != "fu_image_demo" {
				t.Fatalf("unexpected image file_upload payload: %+v", imageBody)
			}
			fileBody := children[1].(map[string]any)["file"].(map[string]any)
			if fileBody["type"] != "file_upload" || fileBody["file_upload"].(map[string]any)["id"] != "fu_file_demo" {
				t.Fatalf("unexpected file file_upload payload: %+v", fileBody)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"id": "blk_media_1"},
					{"id": "blk_media_2"},
				},
			}), nil
		},
	})

	data, appErr := client.AppendBlockChildren(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"children": []any{
			map[string]any{
				"type":           "image",
				"file_upload_id": "fu_image_demo",
			},
			map[string]any{
				"type": "file",
				"file": map[string]any{
					"type": "file_upload",
					"file_upload": map[string]any{
						"id": "fu_file_demo",
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("AppendBlockChildren returned error: %+v", appErr)
	}
	if data["appended_count"] != 2 {
		t.Fatalf("unexpected appended_count: %+v", data["appended_count"])
	}
}

func TestCommentOperationsSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/comments":
				switch request.Method {
				case http.MethodGet:
					if request.URL.Query().Get("block_id") != "page_demo" {
						t.Fatalf("unexpected block_id: %s", request.URL.Query().Get("block_id"))
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"results": []map[string]any{
							{
								"id":            "cmt_1",
								"discussion_id": "discussion_1",
								"rich_text": []map[string]any{
									{
										"type":       "text",
										"plain_text": "First comment",
										"text": map[string]any{
											"content": "First comment",
										},
									},
								},
							},
						},
						"has_more": false,
					}), nil
				case http.MethodPost:
					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode comment create payload: %v", err)
					}
					parent := payload["parent"].(map[string]any)
					if parent["page_id"] != "page_demo" {
						t.Fatalf("unexpected parent payload: %+v", parent)
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":            "cmt_2",
						"discussion_id": "discussion_2",
						"parent": map[string]any{
							"type":    "page_id",
							"page_id": "page_demo",
						},
						"rich_text": []map[string]any{
							{
								"type":       "text",
								"plain_text": "Created comment",
								"text": map[string]any{
									"content": "Created comment",
								},
							},
						},
					}), nil
				default:
					t.Fatalf("unexpected comments method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	listed, appErr := client.ListComments(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "page_demo",
	})
	if appErr != nil {
		t.Fatalf("ListComments returned error: %+v", appErr)
	}
	items := listed["items"].([]map[string]any)
	if len(items) != 1 || items[0]["comment_id"] != "cmt_1" {
		t.Fatalf("unexpected comment list: %+v", listed["items"])
	}

	created, appErr := client.CreateComment(context.Background(), testStaticProfile(), map[string]any{
		"page_id": "page_demo",
		"text":    "Created comment",
	})
	if appErr != nil {
		t.Fatalf("CreateComment returned error: %+v", appErr)
	}
	if created["comment_id"] != "cmt_2" {
		t.Fatalf("unexpected comment_id: %+v", created["comment_id"])
	}
}
