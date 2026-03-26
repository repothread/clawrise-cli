package notion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestCreatePageSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}
			if got := request.Header.Get("Authorization"); got != "Bearer notion-token" {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			if got := request.Header.Get("Notion-Version"); got != defaultNotionVersion {
				t.Fatalf("unexpected Notion-Version: %s", got)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create page request: %v", err)
			}
			parent := payload["parent"].(map[string]any)
			if parent["page_id"] != "page_demo" {
				t.Fatalf("unexpected parent payload: %+v", parent)
			}
			properties := payload["properties"].(map[string]any)
			titleProperty := properties["title"].(map[string]any)
			titleItems := titleProperty["title"].([]any)
			titleText := titleItems[0].(map[string]any)["text"].(map[string]any)["content"]
			if titleText != "项目周报" {
				t.Fatalf("unexpected title text: %+v", titleText)
			}
			children := payload["children"].([]any)
			if len(children) != 2 {
				t.Fatalf("unexpected children count: %d", len(children))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":       "page_123",
				"url":      "https://www.notion.so/page_123",
				"in_trash": false,
				"parent": map[string]any{
					"type":    "page_id",
					"page_id": "page_demo",
				},
				"properties": map[string]any{
					"title": map[string]any{
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "项目周报",
								"text": map[string]any{
									"content": "项目周报",
								},
							},
						},
					},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.CreatePage(context.Background(), testStaticProfile(), map[string]any{
		"parent": map[string]any{
			"type": "page_id",
			"id":   "page_demo",
		},
		"title": "项目周报",
		"children": []any{
			map[string]any{
				"type": "heading_1",
				"text": "本周完成",
			},
			map[string]any{
				"type": "paragraph",
				"text": "完成 Notion 接入原型。",
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreatePage returned error: %+v", appErr)
	}

	if data["page_id"] != "page_123" {
		t.Fatalf("unexpected page_id: %+v", data["page_id"])
	}
	if data["title"] != "项目周报" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestCreatePageWithOAuthRefreshableProfile(t *testing.T) {
	t.Setenv("NOTION_CLIENT_ID", "client-id")
	t.Setenv("NOTION_CLIENT_SECRET", "client-secret")
	t.Setenv("NOTION_REFRESH_TOKEN", "refresh-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/oauth/token":
				if request.Method != http.MethodPost {
					t.Fatalf("unexpected token method: %s", request.Method)
				}
				expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
				if got := request.Header.Get("Authorization"); got != expected {
					t.Fatalf("unexpected authorization header: %s", got)
				}
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode oauth token request: %v", err)
				}
				if payload["grant_type"] != "refresh_token" {
					t.Fatalf("unexpected grant_type: %+v", payload["grant_type"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"access_token":  "fresh-token",
					"token_type":    "bearer",
					"refresh_token": "refresh-token-2",
				}), nil
			case "/v1/pages/page_123":
				if got := request.Header.Get("Authorization"); got != "Bearer fresh-token" {
					t.Fatalf("unexpected authorization header: %s", got)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_123",
					"url":      "https://www.notion.so/page_123",
					"in_trash": false,
					"parent": map[string]any{
						"type":      "workspace",
						"workspace": true,
					},
					"properties": map[string]any{
						"title": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "Public Root Page",
									"text": map[string]any{
										"content": "Public Root Page",
									},
								},
							},
						},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.GetPage(context.Background(), config.Profile{
		Platform: "notion",
		Subject:  "integration",
		Grant: config.Grant{
			Type:         "oauth_refreshable",
			ClientID:     "env:NOTION_CLIENT_ID",
			ClientSecret: "env:NOTION_CLIENT_SECRET",
			RefreshToken: "env:NOTION_REFRESH_TOKEN",
		},
	}, map[string]any{
		"page_id": "page_123",
	})
	if appErr != nil {
		t.Fatalf("GetPage returned error: %+v", appErr)
	}
	if data["title"] != "Public Root Page" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestCreatePageDataSourceRequiresTitleProperty(t *testing.T) {
	client := newTestClient(t, nil)

	_, appErr := client.CreatePage(context.Background(), testStaticProfile(), map[string]any{
		"parent": map[string]any{
			"type": "data_source_id",
			"id":   "ds_demo",
		},
		"title": "需求卡片",
		"properties": map[string]any{
			"状态": map[string]any{
				"select": map[string]any{
					"name": "进行中",
				},
			},
		},
	})
	if appErr == nil {
		t.Fatal("expected CreatePage to reject missing title property mapping")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestAppendBlockChildrenSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo/children" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPatch {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode append blocks request: %v", err)
			}
			if payload["after"] != "before_demo" {
				t.Fatalf("unexpected after marker: %+v", payload["after"])
			}
			children := payload["children"].([]any)
			if len(children) != 2 {
				t.Fatalf("unexpected children count: %d", len(children))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"id": "blk_1"},
					{"id": "blk_2"},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.AppendBlockChildren(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"after":    "before_demo",
		"children": []any{
			map[string]any{
				"type": "paragraph",
				"text": "第一段",
			},
			map[string]any{
				"type": "to_do",
				"text": "补测试",
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

func TestGetBlockSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":           "block_demo",
				"type":         "paragraph",
				"has_children": false,
				"in_trash":     false,
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{
							"type":       "text",
							"plain_text": "第一段正文",
							"text": map[string]any{
								"content": "第一段正文",
							},
						},
					},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.GetBlock(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
	})
	if appErr != nil {
		t.Fatalf("GetBlock returned error: %+v", appErr)
	}
	if data["plain_text"] != "第一段正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestListBlockChildrenSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo/children" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", request.Method)
			}
			if request.URL.Query().Get("page_size") != "20" {
				t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
			}
			if request.URL.Query().Get("start_cursor") != "cursor_demo" {
				t.Fatalf("unexpected start_cursor: %s", request.URL.Query().Get("start_cursor"))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{
						"id":           "blk_1",
						"type":         "heading_2",
						"has_children": true,
						"in_trash":     false,
						"heading_2": map[string]any{
							"rich_text": []map[string]any{
								{
									"type":       "text",
									"plain_text": "待办事项",
									"text": map[string]any{
										"content": "待办事项",
									},
								},
							},
						},
					},
					{
						"id":           "blk_2",
						"type":         "to_do",
						"has_children": false,
						"in_trash":     false,
						"to_do": map[string]any{
							"checked": true,
							"rich_text": []map[string]any{
								{
									"type":       "text",
									"plain_text": "补测试",
									"text": map[string]any{
										"content": "补测试",
									},
								},
							},
						},
					},
				},
				"next_cursor": "cursor_next",
				"has_more":    true,
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.ListBlockChildren(context.Background(), testStaticProfile(), map[string]any{
		"block_id":   "block_demo",
		"page_size":  20,
		"page_token": "cursor_demo",
	})
	if appErr != nil {
		t.Fatalf("ListBlockChildren returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["plain_text"] != "待办事项" {
		t.Fatalf("unexpected first item plain_text: %+v", items[0]["plain_text"])
	}
	if items[1]["checked"] != true {
		t.Fatalf("unexpected second item checked flag: %+v", items[1]["checked"])
	}
	if data["next_page_token"] != "cursor_next" {
		t.Fatalf("unexpected next_page_token: %+v", data["next_page_token"])
	}
}

func TestUpdateBlockSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPatch {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update block request: %v", err)
			}
			if _, exists := payload["object"]; exists {
				t.Fatal("update block payload should not contain object field")
			}
			if payload["type"] != "paragraph" {
				t.Fatalf("unexpected payload type: %+v", payload["type"])
			}
			paragraph := payload["paragraph"].(map[string]any)
			richText := paragraph["rich_text"].([]any)
			text := richText[0].(map[string]any)["text"].(map[string]any)["content"]
			if text != "更新后的正文" {
				t.Fatalf("unexpected payload text: %+v", text)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":           "block_demo",
				"type":         "paragraph",
				"has_children": false,
				"in_trash":     false,
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{
							"type":       "text",
							"plain_text": "更新后的正文",
							"text": map[string]any{
								"content": "更新后的正文",
							},
						},
					},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.UpdateBlock(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"block": map[string]any{
			"type": "paragraph",
			"text": "更新后的正文",
		},
	})
	if appErr != nil {
		t.Fatalf("UpdateBlock returned error: %+v", appErr)
	}
	if data["plain_text"] != "更新后的正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestDeleteBlockSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodDelete {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":           "block_demo",
				"type":         "paragraph",
				"has_children": false,
				"in_trash":     true,
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{
							"type":       "text",
							"plain_text": "待删除正文",
							"text": map[string]any{
								"content": "待删除正文",
							},
						},
					},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.DeleteBlock(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
	})
	if appErr != nil {
		t.Fatalf("DeleteBlock returned error: %+v", appErr)
	}
	if data["deleted"] != true {
		t.Fatalf("unexpected deleted flag: %+v", data["deleted"])
	}
	if data["archived"] != true {
		t.Fatalf("unexpected archived flag: %+v", data["archived"])
	}
}

func TestGetUserSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/users/user_demo" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":         "user_demo",
				"type":       "person",
				"name":       "Alice",
				"avatar_url": "https://example.com/avatar.png",
				"person": map[string]any{
					"email": "alice@example.com",
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.GetUser(context.Background(), testStaticProfile(), map[string]any{
		"user_id": "user_demo",
	})
	if appErr != nil {
		t.Fatalf("GetUser returned error: %+v", appErr)
	}
	if data["email"] != "alice@example.com" {
		t.Fatalf("unexpected email: %+v", data["email"])
	}
}

func TestNormalizeNotionErrorObjectNotFound(t *testing.T) {
	appErr := normalizeNotionHTTPError(http.StatusNotFound, http.Header{}, mustJSON(t, map[string]any{
		"object":  "error",
		"status":  404,
		"code":    "object_not_found",
		"message": "Could not find page",
	}))
	if appErr.Code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
	if !strings.Contains(appErr.Message, "shared with the integration") {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}

func newTestClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()

	httpClient := &http.Client{}
	if transport != nil {
		httpClient.Transport = transport
	}

	client, err := NewClient(Options{
		BaseURL:    "https://api.notion.com",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func testStaticProfile() config.Profile {
	return config.Profile{
		Platform: "notion",
		Subject:  "integration",
		Grant: config.Grant{
			Type:  "static_token",
			Token: "env:NOTION_ACCESS_TOKEN",
		},
	}
}

type roundTripFunc struct {
	handler func(request *http.Request) (*http.Response, error)
}

func (f *roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f.handler(request)
}

func jsonResponse(t *testing.T, statusCode int, value any) *http.Response {
	t.Helper()

	data := mustJSON(t, value)
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request: &http.Request{
			Header: http.Header{},
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}
