package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestCreateCalendarEventSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				if request.Method != http.MethodPost {
					t.Fatalf("unexpected auth method: %s", request.Method)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/calendar/v4/calendars/cal_demo/events":
				if got := request.Header.Get("Authorization"); got != "Bearer tenant-token" {
					t.Fatalf("unexpected authorization header: %s", got)
				}
				if request.URL.Query().Get("idempotency_key") != "idem-demo" {
					t.Fatalf("unexpected idempotency key: %s", request.URL.Query().Get("idempotency_key"))
				}

				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create request: %v", err)
				}
				if payload["summary"] != "Demo Event" {
					t.Fatalf("unexpected summary: %+v", payload["summary"])
				}

				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"event": map[string]any{
							"event_id":              "evt_123",
							"organizer_calendar_id": "cal_demo",
							"summary":               "Demo Event",
							"description":           "Demo Description",
							"start_time": map[string]any{
								"timestamp": "1711764000",
								"timezone":  "Asia/Shanghai",
							},
							"end_time": map[string]any{
								"timestamp": "1711767600",
								"timezone":  "Asia/Shanghai",
							},
							"app_link": "https://calendar.example/event/evt_123",
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
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.CreateCalendarEvent(context.Background(), config.Profile{
		Platform: "feishu",
		Subject:  "bot",
		Grant: config.Grant{
			Type:      "client_credentials",
			AppID:     "env:FEISHU_APP_ID",
			AppSecret: "env:FEISHU_APP_SECRET",
		},
	}, map[string]any{
		"calendar_id": "cal_demo",
		"summary":     "Demo Event",
		"description": "Demo Description",
		"start_at":    "2024-03-30T10:00:00+08:00",
		"end_at":      "2024-03-30T11:00:00+08:00",
		"location":    "Meeting Room A",
	}, "idem-demo")
	if appErr != nil {
		t.Fatalf("CreateCalendarEvent returned error: %+v", appErr)
	}

	if data["event_id"] != "evt_123" {
		t.Fatalf("unexpected event_id: %+v", data["event_id"])
	}
	if data["calendar_id"] != "cal_demo" {
		t.Fatalf("unexpected calendar_id: %+v", data["calendar_id"])
	}
}

func TestCreateCalendarEventRejectsAttendees(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	_, appErr := client.CreateCalendarEvent(context.Background(), config.Profile{
		Platform: "feishu",
		Subject:  "bot",
		Grant: config.Grant{
			Type:      "client_credentials",
			AppID:     "env:FEISHU_APP_ID",
			AppSecret: "env:FEISHU_APP_SECRET",
		},
	}, map[string]any{
		"calendar_id": "cal_demo",
		"summary":     "Demo Event",
		"start_at":    "2024-03-30T10:00:00+08:00",
		"end_at":      "2024-03-30T11:00:00+08:00",
		"attendees": []any{
			map[string]any{
				"type":  "user_id",
				"value": "ou_xxx",
			},
		},
	}, "idem-demo")
	if appErr == nil {
		t.Fatal("expected CreateCalendarEvent to reject attendees")
	}
	if appErr.Code != "UNSUPPORTED_FIELD" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestRequireUserAccessTokenUsesSessionCacheAndRotatedRefreshToken(t *testing.T) {
	t.Setenv("FEISHU_CLIENT_ID", "client-id")
	t.Setenv("FEISHU_CLIENT_SECRET", "client-secret")
	t.Setenv("FEISHU_REFRESH_TOKEN", "refresh-token-1")

	sessionStore := authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	refreshCalls := 0

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/open-apis/authen/v2/oauth/token" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			refreshCalls++
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode oauth token request: %v", err)
			}

			switch refreshCalls {
			case 1:
				if payload["refresh_token"] != "refresh-token-1" {
					t.Fatalf("unexpected first refresh token: %+v", payload["refresh_token"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "ok",
					"data": map[string]any{
						"access_token":  "user-token-1",
						"refresh_token": "refresh-token-2",
						"token_type":    "bearer",
						"expires_in":    7200,
					},
				}), nil
			case 2:
				if payload["refresh_token"] != "refresh-token-2" {
					t.Fatalf("unexpected rotated refresh token: %+v", payload["refresh_token"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "ok",
					"data": map[string]any{
						"access_token":  "user-token-2",
						"refresh_token": "refresh-token-3",
						"token_type":    "bearer",
						"expires_in":    7200,
					},
				}), nil
			default:
				t.Fatalf("unexpected refresh call count: %d", refreshCalls)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL:      "https://open.feishu.cn",
		HTTPClient:   &http.Client{Transport: transport},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	client.now = func() time.Time {
		return now
	}

	ctx := adapter.WithProfileName(context.Background(), "feishu_user_alice")
	profile := config.Profile{
		Platform: "feishu",
		Subject:  "user",
		Grant: config.Grant{
			Type:         "oauth_user",
			ClientID:     "env:FEISHU_CLIENT_ID",
			ClientSecret: "env:FEISHU_CLIENT_SECRET",
			RefreshToken: "env:FEISHU_REFRESH_TOKEN",
		},
	}

	token, appErr := client.requireUserAccessToken(ctx, profile)
	if appErr != nil {
		t.Fatalf("requireUserAccessToken returned error: %+v", appErr)
	}
	if token != "user-token-1" {
		t.Fatalf("unexpected access token: %s", token)
	}

	session, err := sessionStore.Load("feishu_user_alice")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if session.RefreshToken != "refresh-token-2" {
		t.Fatalf("unexpected cached refresh token: %s", session.RefreshToken)
	}

	token, appErr = client.requireUserAccessToken(ctx, profile)
	if appErr != nil {
		t.Fatalf("requireUserAccessToken returned error on cached call: %+v", appErr)
	}
	if token != "user-token-1" {
		t.Fatalf("unexpected cached access token: %s", token)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected cached call to skip refresh, got: %d", refreshCalls)
	}

	now = now.Add(3 * time.Hour)
	token, appErr = client.requireUserAccessToken(ctx, profile)
	if appErr != nil {
		t.Fatalf("requireUserAccessToken returned error after expiry: %+v", appErr)
	}
	if token != "user-token-2" {
		t.Fatalf("unexpected refreshed access token: %s", token)
	}
	if refreshCalls != 2 {
		t.Fatalf("expected second refresh after expiry, got: %d", refreshCalls)
	}

	session, err = sessionStore.Load("feishu_user_alice")
	if err != nil {
		t.Fatalf("Load returned error after refresh: %v", err)
	}
	if session.RefreshToken != "refresh-token-3" {
		t.Fatalf("unexpected rotated cached refresh token: %s", session.RefreshToken)
	}
}

func TestRequireUserAccessTokenRequiresInteractiveAuthorization(t *testing.T) {
	t.Setenv("FEISHU_CLIENT_ID", "client-id")
	t.Setenv("FEISHU_CLIENT_SECRET", "client-secret")

	client, err := NewClient(Options{
		BaseURL:      "https://open.feishu.cn",
		SessionStore: authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml")),
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	ctx := adapter.WithProfileName(context.Background(), "feishu_user_alice")
	_, appErr := client.requireUserAccessToken(ctx, config.Profile{
		Platform: "feishu",
		Subject:  "user",
		Grant: config.Grant{
			Type:         "oauth_user",
			ClientID:     "env:FEISHU_CLIENT_ID",
			ClientSecret: "env:FEISHU_CLIENT_SECRET",
		},
	})
	if appErr == nil {
		t.Fatal("expected requireUserAccessToken to request interactive authorization")
	}
	if appErr.Code != "AUTHORIZATION_REQUIRED" {
		t.Fatalf("unexpected error: %+v", appErr)
	}
}

func TestListWikiSpacesSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/wiki/v2/spaces":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"items": []map[string]any{
							{
								"space_id":    "space_123",
								"name":        "Knowledge Base",
								"description": "Demo Space",
								"space_type":  "team",
								"visibility":  "private",
							},
						},
						"page_token": "",
						"has_more":   false,
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.ListWikiSpaces(context.Background(), testBotProfile(), map[string]any{
		"page_size": 10,
	})
	if appErr != nil {
		t.Fatalf("ListWikiSpaces returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["space_id"] != "space_123" {
		t.Fatalf("unexpected space_id: %+v", items[0]["space_id"])
	}
}

func TestCreateWikiNodeSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/wiki/v2/spaces/space_123/nodes":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create wiki node request: %v", err)
				}
				if payload["obj_type"] != "docx" {
					t.Fatalf("unexpected obj_type: %+v", payload["obj_type"])
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"node": map[string]any{
							"space_id":          "space_123",
							"node_token":        "wik_123",
							"obj_token":         "dox_123",
							"obj_type":          "docx",
							"parent_node_token": "parent_123",
							"node_type":         "origin",
							"title":             "Child Doc",
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
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.CreateWikiNode(context.Background(), testBotProfile(), map[string]any{
		"space_id":          "space_123",
		"parent_node_token": "parent_123",
		"title":             "Child Doc",
	})
	if appErr != nil {
		t.Fatalf("CreateWikiNode returned error: %+v", appErr)
	}

	if data["document_id"] != "dox_123" {
		t.Fatalf("unexpected document_id: %+v", data["document_id"])
	}
}

func TestAppendDocumentBlocksSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/dox_123/children":
				if got := request.URL.Query().Get("client_token"); got != "client-demo" {
					t.Fatalf("unexpected client_token: %s", got)
				}
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode append blocks request: %v", err)
				}
				children := payload["children"].([]any)
				if len(children) != 2 {
					t.Fatalf("unexpected children count: %d", len(children))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"client_token": "client-demo",
						"children": []map[string]any{
							{
								"block_id":   "blk_1",
								"block_type": 3,
							},
							{
								"block_id":   "blk_2",
								"block_type": 2,
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
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.AppendDocumentBlocks(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"blocks": []any{
			map[string]any{
				"type": "heading_1",
				"text": "Child Document",
			},
			map[string]any{
				"type": "paragraph",
				"text": "Generated by Clawrise.",
			},
		},
	}, "client-demo")
	if appErr != nil {
		t.Fatalf("AppendDocumentBlocks returned error: %+v", appErr)
	}

	if data["appended_count"] != 2 {
		t.Fatalf("unexpected appended_count: %+v", data["appended_count"])
	}
}

func TestGetDocumentRawContentSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/raw_content":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"content": "Hello from knowledge base document",
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetDocumentRawContent(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
	})
	if appErr != nil {
		t.Fatalf("GetDocumentRawContent returned error: %+v", appErr)
	}

	if data["content"] != "Hello from knowledge base document" {
		t.Fatalf("unexpected content: %+v", data["content"])
	}
}

func TestGetDocumentSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123":
				if got := request.Header.Get("Authorization"); got != "Bearer tenant-token" {
					t.Fatalf("unexpected authorization header: %s", got)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"document": map[string]any{
							"document_id": "dox_123",
							"revision_id": 12,
							"title":       "项目周报",
							"display_setting": map[string]any{
								"show_authors": true,
							},
							"cover": map[string]any{
								"type": "colorful",
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
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetDocument(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
	})
	if appErr != nil {
		t.Fatalf("GetDocument returned error: %+v", appErr)
	}
	if data["document_id"] != "dox_123" {
		t.Fatalf("unexpected document_id: %+v", data["document_id"])
	}
	if data["title"] != "项目周报" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestListDocumentBlocksSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks":
				if request.URL.Query().Get("page_size") != "100" {
					t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
				}
				if request.URL.Query().Get("page_token") != "token_demo" {
					t.Fatalf("unexpected page_token: %s", request.URL.Query().Get("page_token"))
				}
				if request.URL.Query().Get("document_revision_id") != "-1" {
					t.Fatalf("unexpected document_revision_id: %s", request.URL.Query().Get("document_revision_id"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"items": []map[string]any{
							{
								"block_id":   "blk_1",
								"parent_id":  "",
								"children":   []string{"blk_2"},
								"block_type": 3,
								"heading1": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "项目概览",
											},
										},
									},
								},
							},
							{
								"block_id":   "blk_2",
								"parent_id":  "blk_1",
								"children":   []string{},
								"block_type": 2,
								"text": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "本周完成了接入验证。",
											},
										},
									},
								},
							},
						},
						"page_token": "token_next",
						"has_more":   true,
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.ListDocumentBlocks(context.Background(), testBotProfile(), map[string]any{
		"document_id":          "dox_123",
		"page_size":            100,
		"page_token":           "token_demo",
		"document_revision_id": -1,
	})
	if appErr != nil {
		t.Fatalf("ListDocumentBlocks returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["plain_text"] != "项目概览" {
		t.Fatalf("unexpected first item plain_text: %+v", items[0]["plain_text"])
	}
	if items[1]["block_type_name"] != "paragraph" {
		t.Fatalf("unexpected second item block_type_name: %+v", items[1]["block_type_name"])
	}
	if data["next_page_token"] != "token_next" {
		t.Fatalf("unexpected next_page_token: %+v", data["next_page_token"])
	}
}

func TestGetDocumentBlockSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
				if request.URL.Query().Get("document_revision_id") != "-1" {
					t.Fatalf("unexpected document_revision_id: %s", request.URL.Query().Get("document_revision_id"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"block": map[string]any{
							"block_id":   "blk_2",
							"parent_id":  "blk_1",
							"children":   []string{},
							"block_type": 2,
							"text": map[string]any{
								"elements": []map[string]any{
									{
										"text_run": map[string]any{
											"content": "正文段落",
										},
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
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetDocumentBlock(context.Background(), testBotProfile(), map[string]any{
		"document_id":          "dox_123",
		"block_id":             "blk_2",
		"document_revision_id": -1,
	})
	if appErr != nil {
		t.Fatalf("GetDocumentBlock returned error: %+v", appErr)
	}
	if data["plain_text"] != "正文段落" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
	if data["block_type_name"] != "paragraph" {
		t.Fatalf("unexpected block_type_name: %+v", data["block_type_name"])
	}
}

func TestGetDocumentBlockChildrenSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/blk_1/children":
				if request.URL.Query().Get("page_size") != "50" {
					t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
				}
				if request.URL.Query().Get("page_token") != "token_demo" {
					t.Fatalf("unexpected page_token: %s", request.URL.Query().Get("page_token"))
				}
				if request.URL.Query().Get("document_revision_id") != "-1" {
					t.Fatalf("unexpected document_revision_id: %s", request.URL.Query().Get("document_revision_id"))
				}
				if request.URL.Query().Get("with_descendants") != "true" {
					t.Fatalf("unexpected with_descendants: %s", request.URL.Query().Get("with_descendants"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"items": []map[string]any{
							{
								"block_id":   "blk_2",
								"parent_id":  "blk_1",
								"children":   []string{"blk_3"},
								"block_type": 12,
								"bullet": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "任务一",
											},
										},
									},
								},
							},
							{
								"block_id":   "blk_3",
								"parent_id":  "blk_2",
								"children":   []string{},
								"block_type": 2,
								"text": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "任务详情",
											},
										},
									},
								},
							},
						},
						"page_token": "token_next",
						"has_more":   true,
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetDocumentBlockChildren(context.Background(), testBotProfile(), map[string]any{
		"document_id":          "dox_123",
		"block_id":             "blk_1",
		"page_size":            50,
		"page_token":           "token_demo",
		"document_revision_id": -1,
		"with_descendants":     true,
	})
	if appErr != nil {
		t.Fatalf("GetDocumentBlockChildren returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["block_type_name"] != "bulleted_list_item" {
		t.Fatalf("unexpected first item block_type_name: %+v", items[0]["block_type_name"])
	}
	if items[1]["plain_text"] != "任务详情" {
		t.Fatalf("unexpected second item plain_text: %+v", items[1]["plain_text"])
	}
}

func TestGetDocumentBlockDescendantsForcesWithDescendants(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/blk_root/children":
				if request.URL.Query().Get("with_descendants") != "true" {
					t.Fatalf("unexpected with_descendants: %s", request.URL.Query().Get("with_descendants"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"items": []map[string]any{
							{
								"block_id":   "blk_child",
								"parent_id":  "blk_root",
								"children":   []string{"blk_grandchild"},
								"block_type": 12,
								"bullet": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "任务一",
											},
										},
									},
								},
							},
							{
								"block_id":   "blk_grandchild",
								"parent_id":  "blk_child",
								"children":   []string{},
								"block_type": 2,
								"text": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "子任务详情",
											},
										},
									},
								},
							},
						},
						"page_token": "",
						"has_more":   false,
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetDocumentBlockDescendants(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"block_id":    "blk_root",
	})
	if appErr != nil {
		t.Fatalf("GetDocumentBlockDescendants returned error: %+v", appErr)
	}
	items := data["items"].([]map[string]any)
	if len(items) != 2 || items[1]["plain_text"] != "子任务详情" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestUpdateDocumentBlockSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
				if request.Method != http.MethodPatch {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				if request.URL.Query().Get("document_revision_id") != "-1" {
					t.Fatalf("unexpected document_revision_id: %s", request.URL.Query().Get("document_revision_id"))
				}
				if request.URL.Query().Get("client_token") != "client-demo" {
					t.Fatalf("unexpected client_token: %s", request.URL.Query().Get("client_token"))
				}

				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode update block request: %v", err)
				}
				if payload["block_id"] != "blk_2" {
					t.Fatalf("unexpected block_id: %+v", payload["block_id"])
				}
				updateText := payload["update_text"].(map[string]any)
				elements := updateText["elements"].([]any)
				text := elements[0].(map[string]any)["text_run"].(map[string]any)["content"]
				if text != "更新后的正文" {
					t.Fatalf("unexpected update text: %+v", text)
				}

				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"block": map[string]any{
							"block_id":   "blk_2",
							"parent_id":  "blk_1",
							"children":   []string{},
							"block_type": 2,
							"text": map[string]any{
								"elements": []map[string]any{
									{
										"text_run": map[string]any{
											"content": "更新后的正文",
										},
									},
								},
							},
						},
						"document_revision_id": 22,
						"client_token":         "client-demo",
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.UpdateDocumentBlock(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"block_id":    "blk_2",
		"text":        "更新后的正文",
	}, "client-demo")
	if appErr != nil {
		t.Fatalf("UpdateDocumentBlock returned error: %+v", appErr)
	}
	if data["plain_text"] != "更新后的正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
	if data["document_revision_id"] != 22 {
		t.Fatalf("unexpected document_revision_id: %+v", data["document_revision_id"])
	}
}

func TestBatchDeleteDocumentBlockChildrenSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"code":                0,
					"msg":                 "ok",
					"tenant_access_token": "tenant-token",
					"expire":              7200,
				}), nil
			case "/open-apis/docx/v1/documents/dox_123/blocks/blk_1/children/batch_delete":
				if request.Method != http.MethodDelete {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				if request.URL.Query().Get("document_revision_id") != "-1" {
					t.Fatalf("unexpected document_revision_id: %s", request.URL.Query().Get("document_revision_id"))
				}
				if request.URL.Query().Get("client_token") != "delete-demo" {
					t.Fatalf("unexpected client_token: %s", request.URL.Query().Get("client_token"))
				}

				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode batch delete request: %v", err)
				}
				if payload["start_index"] != float64(0) || payload["end_index"] != float64(2) {
					t.Fatalf("unexpected delete payload: %+v", payload)
				}

				return jsonResponse(t, http.StatusOK, map[string]any{
					"code": 0,
					"msg":  "success",
					"data": map[string]any{
						"document_revision_id": 23,
						"client_token":         "delete-demo",
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	}

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: transport,
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.BatchDeleteDocumentBlockChildren(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"block_id":    "blk_1",
		"start_index": 0,
		"end_index":   2,
	}, "delete-demo")
	if appErr != nil {
		t.Fatalf("BatchDeleteDocumentBlockChildren returned error: %+v", appErr)
	}
	if data["document_revision_id"] != 23 {
		t.Fatalf("unexpected document_revision_id: %+v", data["document_revision_id"])
	}
	if data["end_index"] != 2 {
		t.Fatalf("unexpected end_index: %+v", data["end_index"])
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

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal response body: %v", err)
	}

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

func TestNormalizeFeishuErrorRateLimited(t *testing.T) {
	appErr := normalizeFeishuError(190004, "method rate limited", 0)
	if appErr.Code != "RATE_LIMITED" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
	if !appErr.Retryable {
		t.Fatal("expected rate limited error to be retryable")
	}
	if !strings.Contains(appErr.Message, "method rate limited") {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}

func testBotProfile() config.Profile {
	return config.Profile{
		Platform: "feishu",
		Subject:  "bot",
		Grant: config.Grant{
			Type:      "client_credentials",
			AppID:     "env:FEISHU_APP_ID",
			AppSecret: "env:FEISHU_APP_SECRET",
		},
	}
}
