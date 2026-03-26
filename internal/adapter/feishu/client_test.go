package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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
