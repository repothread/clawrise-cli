package notion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
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

func TestCreatePageSupportsProviderNativeChildren(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create page request: %v", err)
			}
			children := payload["children"].([]any)
			heading := children[0].(map[string]any)["heading_1"].(map[string]any)
			headingText := heading["rich_text"].([]any)[0].(map[string]any)["text"].(map[string]any)["content"]
			if headingText != "Provider 标题" {
				t.Fatalf("unexpected heading payload: %+v", heading)
			}
			paragraph := children[1].(map[string]any)["paragraph"].(map[string]any)
			paragraphText := paragraph["rich_text"].([]any)[0].(map[string]any)["text"].(map[string]any)["content"]
			if paragraphText != "Provider 正文" {
				t.Fatalf("unexpected paragraph payload: %+v", paragraph)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":       "page_provider_123",
				"url":      "https://www.notion.so/page_provider_123",
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
								"plain_text": "Provider 页面",
								"text": map[string]any{
									"content": "Provider 页面",
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
		"title": "Provider 页面",
		"children": []any{
			map[string]any{
				"type": "heading_1",
				"heading_1": map[string]any{
					"rich_text": []map[string]any{
						{
							"type": "text",
							"text": map[string]any{
								"content": "Provider 标题",
							},
						},
					},
				},
			},
			map[string]any{
				"type": "paragraph",
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{
							"type": "text",
							"text": map[string]any{
								"content": "Provider 正文",
							},
						},
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreatePage returned error: %+v", appErr)
	}
	if data["page_id"] != "page_provider_123" {
		t.Fatalf("unexpected page_id: %+v", data["page_id"])
	}
}

func TestCreatePageVerifyAfterWriteSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	ctx := adapter.WithRuntimeOptions(context.Background(), adapter.RuntimeOptions{
		VerifyAfterWrite: true,
	})
	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_verify_123",
					"url":      "https://www.notion.so/page_verify_123",
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
									"plain_text": "验证页面",
									"text": map[string]any{
										"content": "验证页面",
									},
								},
							},
						},
					},
				}), nil
			case "/v1/pages/page_verify_123":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_verify_123",
					"url":      "https://www.notion.so/page_verify_123",
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
									"plain_text": "验证页面",
									"text": map[string]any{
										"content": "验证页面",
									},
								},
							},
						},
					},
				}), nil
			case "/v1/pages/page_verify_123/markdown":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "page_verify_123",
					"markdown":          "# 验证标题\n\n验证正文",
					"truncated":         false,
					"unknown_block_ids": []string{},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.CreatePage(ctx, testStaticProfile(), map[string]any{
		"parent": map[string]any{
			"type": "page_id",
			"id":   "page_demo",
		},
		"title": "验证页面",
		"children": []any{
			map[string]any{
				"type": "heading_1",
				"text": "验证标题",
			},
			map[string]any{
				"type": "paragraph",
				"text": "验证正文",
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreatePage returned error: %+v", appErr)
	}

	verification := data["verification"].(map[string]any)
	if verification["ok"] != true {
		t.Fatalf("unexpected verification result: %+v", verification)
	}
}

func TestUpdatePageVerifyAndDebugSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	ctx := adapter.WithRuntimeOptions(context.Background(), adapter.RuntimeOptions{
		DebugProviderPayload: true,
		VerifyAfterWrite:     true,
	})
	ctx, _ = adapter.WithProviderDebugCapture(ctx)

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_demo":
				switch request.Method {
				case http.MethodPatch:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":       "page_demo",
						"url":      "https://www.notion.so/page_demo",
						"archived": true,
						"parent": map[string]any{
							"type":    "page_id",
							"page_id": "parent_demo",
						},
						"properties": map[string]any{
							"title": map[string]any{
								"title": []map[string]any{
									{
										"type":       "text",
										"plain_text": "已更新页面",
										"text": map[string]any{
											"content": "已更新页面",
										},
									},
								},
							},
						},
					}), nil
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":       "page_demo",
						"url":      "https://www.notion.so/page_demo",
						"archived": true,
						"parent": map[string]any{
							"type":    "page_id",
							"page_id": "parent_demo",
						},
						"properties": map[string]any{
							"title": map[string]any{
								"title": []map[string]any{
									{
										"type":       "text",
										"plain_text": "已更新页面",
										"text": map[string]any{
											"content": "已更新页面",
										},
									},
								},
							},
						},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.UpdatePage(ctx, testStaticProfile(), map[string]any{
		"page_id":  "page_demo",
		"title":    "已更新页面",
		"archived": true,
	})
	if appErr != nil {
		t.Fatalf("UpdatePage returned error: %+v", appErr)
	}

	verification := data["verification"].(map[string]any)
	if verification["ok"] != true {
		t.Fatalf("unexpected verification result: %+v", verification)
	}

	debugData := adapter.ProviderDebugFromContext(ctx)
	requests := debugData["provider_requests"].([]map[string]any)
	if len(requests) != 2 {
		t.Fatalf("unexpected provider debug entries: %+v", debugData)
	}
	if requests[0]["method"] != http.MethodPatch {
		t.Fatalf("unexpected first provider debug entry: %+v", requests[0])
	}
	encodedRequestBody, err := json.Marshal(requests[0]["request_body"])
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}
	if strings.Contains(string(encodedRequestBody), "已更新页面") {
		t.Fatalf("expected request body content to be redacted: %s", string(encodedRequestBody))
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
	data, appErr := client.GetPage(context.Background(), ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Grant: ExecutionGrant{
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

func TestGetPageWithOAuthRefreshableProfileUsesSessionCache(t *testing.T) {
	t.Setenv("NOTION_CLIENT_ID", "client-id")
	t.Setenv("NOTION_CLIENT_SECRET", "client-secret")
	t.Setenv("NOTION_REFRESH_TOKEN", "refresh-token")

	sessionStore := authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	refreshCalls := 0
	pageCalls := 0

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/oauth/token":
				refreshCalls++
				return jsonResponse(t, http.StatusOK, map[string]any{
					"access_token":  "fresh-token",
					"token_type":    "bearer",
					"refresh_token": "refresh-token-2",
					"expires_in":    3600,
				}), nil
			case "/v1/pages/page_123":
				pageCalls++
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
									"plain_text": "Cached Page",
									"text": map[string]any{
										"content": "Cached Page",
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

	client, err := NewClient(Options{
		BaseURL:      "https://api.notion.com",
		HTTPClient:   &http.Client{Transport: transport},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	ctx := adapter.WithAccountName(context.Background(), "notion_public_workspace_a")
	profile := ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Grant: ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     "env:NOTION_CLIENT_ID",
			ClientSecret: "env:NOTION_CLIENT_SECRET",
			RefreshToken: "env:NOTION_REFRESH_TOKEN",
		},
	}

	_, appErr := client.GetPage(ctx, profile, map[string]any{
		"page_id": "page_123",
	})
	if appErr != nil {
		t.Fatalf("GetPage returned error: %+v", appErr)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected one refresh call, got: %d", refreshCalls)
	}

	session, err := sessionStore.Load("notion_public_workspace_a")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if session.AccessToken != "fresh-token" {
		t.Fatalf("unexpected cached access token: %s", session.AccessToken)
	}
	if session.RefreshToken != "refresh-token-2" {
		t.Fatalf("unexpected cached refresh token: %s", session.RefreshToken)
	}

	_, appErr = client.GetPage(ctx, profile, map[string]any{
		"page_id": "page_123",
	})
	if appErr != nil {
		t.Fatalf("GetPage returned error on cached call: %+v", appErr)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected cached call to skip refresh, got refresh count: %d", refreshCalls)
	}
	if pageCalls != 2 {
		t.Fatalf("expected two page calls, got: %d", pageCalls)
	}
}

func TestRequireAccessTokenRequiresInteractiveAuthorization(t *testing.T) {
	t.Setenv("NOTION_CLIENT_ID", "client-id")
	t.Setenv("NOTION_CLIENT_SECRET", "client-secret")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected request without completed authorization: %s", request.URL.Path)
			return nil, nil
		},
	})
	client.sessionStore = authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))

	ctx := adapter.WithAccountName(context.Background(), "notion_public_workspace_a")
	_, _, appErr := client.requireAccessToken(ctx, ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Grant: ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     "env:NOTION_CLIENT_ID",
			ClientSecret: "env:NOTION_CLIENT_SECRET",
		},
	})
	if appErr == nil {
		t.Fatal("expected requireAccessToken to request interactive authorization")
	}
	if appErr.Code != "AUTHORIZATION_REQUIRED" {
		t.Fatalf("unexpected error: %+v", appErr)
	}
}

func TestCreatePageUnderDataSourceSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create page request: %v", err)
			}
			parent := payload["parent"].(map[string]any)
			if parent["data_source_id"] != "ds_demo" {
				t.Fatalf("unexpected parent payload: %+v", parent)
			}
			properties := payload["properties"].(map[string]any)
			if _, ok := properties["Name"]; !ok {
				t.Fatalf("expected Name title property: %+v", properties)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":       "page_ds_123",
				"url":      "https://www.notion.so/page_ds_123",
				"in_trash": false,
				"parent": map[string]any{
					"type":           "data_source_id",
					"data_source_id": "ds_demo",
				},
				"properties": map[string]any{
					"Name": map[string]any{
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "数据源记录",
								"text": map[string]any{
									"content": "数据源记录",
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
			"type": "data_source_id",
			"id":   "ds_demo",
		},
		"title":          "数据源记录",
		"title_property": "Name",
		"properties": map[string]any{
			"状态": map[string]any{
				"select": map[string]any{
					"name": "进行中",
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreatePage returned error: %+v", appErr)
	}
	if data["page_id"] != "page_ds_123" {
		t.Fatalf("unexpected page_id: %+v", data["page_id"])
	}
	if data["title"] != "数据源记录" {
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

func TestGetPageMarkdownSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_123/markdown" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", request.Method)
			}
			if request.URL.Query().Get("include_transcript") != "true" {
				t.Fatalf("unexpected include_transcript: %s", request.URL.Query().Get("include_transcript"))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":            "page_markdown",
				"id":                "page_123",
				"markdown":          "# 项目周报\n\n本周完成了接入验证。",
				"truncated":         false,
				"unknown_block_ids": []string{"blk_unknown"},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.GetPageMarkdown(context.Background(), testStaticProfile(), map[string]any{
		"page_id":            "page_123",
		"include_transcript": true,
	})
	if appErr != nil {
		t.Fatalf("GetPageMarkdown returned error: %+v", appErr)
	}
	if data["page_id"] != "page_123" {
		t.Fatalf("unexpected page_id: %+v", data["page_id"])
	}
	if data["truncated"] != false {
		t.Fatalf("unexpected truncated flag: %+v", data["truncated"])
	}
	unknownBlockIDs := data["unknown_block_ids"].([]string)
	if len(unknownBlockIDs) != 1 || unknownBlockIDs[0] != "blk_unknown" {
		t.Fatalf("unexpected unknown_block_ids: %+v", data["unknown_block_ids"])
	}
}

func TestUpdatePageMarkdownWithUpdateContentSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_123/markdown" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPatch {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update page markdown request: %v", err)
			}
			if payload["type"] != "update_content" {
				t.Fatalf("unexpected type: %+v", payload["type"])
			}
			updateContent := payload["update_content"].(map[string]any)
			if updateContent["allow_deleting_content"] != true {
				t.Fatalf("unexpected allow_deleting_content: %+v", updateContent["allow_deleting_content"])
			}
			updates := updateContent["content_updates"].([]any)
			first := updates[0].(map[string]any)
			if first["old_str"] != "旧文案" || first["new_str"] != "新文案" {
				t.Fatalf("unexpected content update: %+v", first)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":            "page_markdown",
				"id":                "page_123",
				"markdown":          "# 标题\n\n新文案",
				"truncated":         false,
				"unknown_block_ids": []string{},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.UpdatePageMarkdown(context.Background(), testStaticProfile(), map[string]any{
		"page_id": "page_123",
		"type":    "update_content",
		"update_content": map[string]any{
			"allow_deleting_content": true,
			"content_updates": []any{
				map[string]any{
					"old_str":             "旧文案",
					"new_str":             "新文案",
					"replace_all_matches": true,
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("UpdatePageMarkdown returned error: %+v", appErr)
	}
	if data["markdown"] != "# 标题\n\n新文案" {
		t.Fatalf("unexpected markdown: %+v", data["markdown"])
	}
}

func TestUpdatePageMarkdownInfersInsertContent(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update page markdown request: %v", err)
			}
			if payload["type"] != "insert_content" {
				t.Fatalf("unexpected type: %+v", payload["type"])
			}
			insertContent := payload["insert_content"].(map[string]any)
			if insertContent["content"] != "新增段落" || insertContent["after"] != "## 待办" {
				t.Fatalf("unexpected insert_content payload: %+v", insertContent)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object":            "page_markdown",
				"id":                "page_123",
				"markdown":          "# 标题\n\n## 待办\n新增段落",
				"truncated":         false,
				"unknown_block_ids": []string{},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.UpdatePageMarkdown(context.Background(), testStaticProfile(), map[string]any{
		"page_id": "page_123",
		"insert_content": map[string]any{
			"content": "新增段落",
			"after":   "## 待办",
		},
	})
	if appErr != nil {
		t.Fatalf("UpdatePageMarkdown returned error: %+v", appErr)
	}
	if !strings.Contains(data["markdown"].(string), "新增段落") {
		t.Fatalf("unexpected markdown: %+v", data["markdown"])
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

func TestAppendBlockChildrenSupportsProviderNativeBodies(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo/children" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode append blocks request: %v", err)
			}
			children := payload["children"].([]any)
			paragraph := children[0].(map[string]any)["paragraph"].(map[string]any)
			paragraphText := paragraph["rich_text"].([]any)[0].(map[string]any)["text"].(map[string]any)["content"]
			if paragraphText != "Provider 追加正文" {
				t.Fatalf("unexpected paragraph payload: %+v", paragraph)
			}
			toDo := children[1].(map[string]any)["to_do"].(map[string]any)
			toDoText := toDo["rich_text"].([]any)[0].(map[string]any)["text"].(map[string]any)["content"]
			if toDoText != "Provider 待办" || toDo["checked"] != true {
				t.Fatalf("unexpected to_do payload: %+v", toDo)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"id": "blk_provider_1"},
					{"id": "blk_provider_2"},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.AppendBlockChildren(context.Background(), testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"children": []any{
			map[string]any{
				"type": "paragraph",
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{
							"type": "text",
							"text": map[string]any{
								"content": "Provider 追加正文",
							},
						},
					},
				},
			},
			map[string]any{
				"type": "to_do",
				"to_do": map[string]any{
					"checked": true,
					"rich_text": []map[string]any{
						{
							"type": "text",
							"text": map[string]any{
								"content": "Provider 待办",
							},
						},
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

func TestAppendBlockChildrenVerifyAndDebugSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	ctx := adapter.WithRuntimeOptions(context.Background(), adapter.RuntimeOptions{
		DebugProviderPayload: true,
		VerifyAfterWrite:     true,
	})
	ctx, _ = adapter.WithProviderDebugCapture(ctx)

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/blocks/block_demo/children":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{"id": "blk_debug_1"},
						{"id": "blk_debug_2"},
					},
				}), nil
			case "/v1/blocks/blk_debug_1":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "blk_debug_1",
					"type":         "paragraph",
					"has_children": false,
					"in_trash":     false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{
								"type":       "text",
								"plain_text": "调试正文",
								"text": map[string]any{
									"content": "调试正文",
								},
							},
						},
					},
				}), nil
			case "/v1/blocks/blk_debug_2":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":           "blk_debug_2",
					"type":         "to_do",
					"has_children": false,
					"in_trash":     false,
					"to_do": map[string]any{
						"checked": true,
						"rich_text": []map[string]any{
							{
								"type":       "text",
								"plain_text": "调试待办",
								"text": map[string]any{
									"content": "调试待办",
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
	data, appErr := client.AppendBlockChildren(ctx, testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"children": []any{
			map[string]any{
				"type": "paragraph",
				"text": "调试正文",
			},
			map[string]any{
				"type":    "to_do",
				"text":    "调试待办",
				"checked": true,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("AppendBlockChildren returned error: %+v", appErr)
	}

	verification := data["verification"].(map[string]any)
	if verification["ok"] != true {
		t.Fatalf("unexpected verification result: %+v", verification)
	}

	debugData := adapter.ProviderDebugFromContext(ctx)
	requests := debugData["provider_requests"].([]map[string]any)
	if len(requests) != 3 {
		t.Fatalf("unexpected provider debug entries: %+v", debugData)
	}
	if requests[0]["path"] != "/v1/blocks/block_demo/children" {
		t.Fatalf("unexpected first provider debug entry: %+v", requests[0])
	}
	encodedRequestBody, err := json.Marshal(requests[0]["request_body"])
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}
	if strings.Contains(string(encodedRequestBody), "调试正文") || strings.Contains(string(encodedRequestBody), "调试待办") {
		t.Fatalf("expected request body content to be redacted: %s", string(encodedRequestBody))
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

func TestUpdateBlockSupportsProviderNativeBlockWrapper(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/blocks/block_demo" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update block request: %v", err)
			}
			paragraph := payload["paragraph"].(map[string]any)
			richText := paragraph["rich_text"].([]any)
			text := richText[0].(map[string]any)["text"].(map[string]any)["content"]
			if text != "Provider 更新正文" {
				t.Fatalf("unexpected payload text: %+v", paragraph)
			}
			if paragraph["color"] != "green_background" {
				t.Fatalf("unexpected payload color: %+v", paragraph)
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
							"plain_text": "Provider 更新正文",
							"text": map[string]any{
								"content": "Provider 更新正文",
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
			"paragraph": map[string]any{
				"color": "green_background",
				"rich_text": []map[string]any{
					{
						"type": "text",
						"text": map[string]any{
							"content": "Provider 更新正文",
						},
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("UpdateBlock returned error: %+v", appErr)
	}
	if data["plain_text"] != "Provider 更新正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestUpdateBlockVerifyAfterWriteSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	ctx := adapter.WithRuntimeOptions(context.Background(), adapter.RuntimeOptions{
		VerifyAfterWrite: true,
	})
	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/blocks/block_demo":
				switch request.Method {
				case http.MethodPatch:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":           "block_demo",
						"type":         "paragraph",
						"has_children": false,
						"in_trash":     false,
						"paragraph": map[string]any{
							"rich_text": []map[string]any{
								{
									"type":       "text",
									"plain_text": "验证更新正文",
									"text": map[string]any{
										"content": "验证更新正文",
									},
								},
							},
						},
					}), nil
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":           "block_demo",
						"type":         "paragraph",
						"has_children": false,
						"in_trash":     false,
						"paragraph": map[string]any{
							"rich_text": []map[string]any{
								{
									"type":       "text",
									"plain_text": "验证更新正文",
									"text": map[string]any{
										"content": "验证更新正文",
									},
								},
							},
						},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.UpdateBlock(ctx, testStaticProfile(), map[string]any{
		"block_id": "block_demo",
		"type":     "paragraph",
		"text":     "验证更新正文",
	})
	if appErr != nil {
		t.Fatalf("UpdateBlock returned error: %+v", appErr)
	}

	verification := data["verification"].(map[string]any)
	if verification["ok"] != true {
		t.Fatalf("unexpected verification result: %+v", verification)
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

func testStaticProfile() ExecutionProfile {
	return ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Grant: ExecutionGrant{
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
