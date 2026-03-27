package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestExecutorDryRunSuccess(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Profile:  "feishu_bot_ops",
		},
		Profiles: map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		},
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "calendar.event.create",
		DryRun:         true,
		InputJSON:      `{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Operation != "feishu.calendar.event.create" {
		t.Fatalf("unexpected operation: %s", envelope.Operation)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Platform != "feishu" {
		t.Fatalf("unexpected context platform: %s", envelope.Context.Platform)
	}
	if envelope.Context.Subject != "bot" {
		t.Fatalf("unexpected context subject: %s", envelope.Context.Subject)
	}
	if envelope.Context.Profile != "feishu_bot_ops" {
		t.Fatalf("unexpected context profile: %s", envelope.Context.Profile)
	}
	if envelope.Idempotency == nil || envelope.Idempotency.Status != "dry_run" {
		t.Fatalf("unexpected idempotency state: %+v", envelope.Idempotency)
	}
}

func TestExecutorReadOperationOmitsIdempotency(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.get",
		DryRun:         true,
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Idempotency != nil {
		t.Fatalf("expected no idempotency section for read operation, got: %+v", envelope.Idempotency)
	}
}

func TestExecutorRejectsSubjectMismatch(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "feishu.calendar.event.list",
		ProfileName:    "notion_team_docs",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected execution to fail because of subject mismatch")
	}
	if envelope.Error == nil || envelope.Error.Code != "PROFILE_PLATFORM_MISMATCH" {
		t.Fatalf("unexpected error payload: %+v", envelope.Error)
	}
}

func TestExecutorExecutesNotionPageGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"id":       "page_demo",
					"url":      "https://www.notion.so/page_demo",
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
									"plain_text": "执行器验证",
									"text": map[string]any{
										"content": "执行器验证",
									},
								},
							},
						},
					},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.get",
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["title"] != "执行器验证" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestExecutorExecutesNotionPageMarkdownGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo/markdown" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "page_demo",
					"markdown":          "# 执行器验证",
					"truncated":         false,
					"unknown_block_ids": []string{},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.markdown.get",
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion markdown execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["markdown"] != "# 执行器验证" {
		t.Fatalf("unexpected markdown: %+v", data["markdown"])
	}
}

func TestExecutorExecutesNotionSearch(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/search" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"object": "page",
							"id":     "page_demo",
							"properties": map[string]any{
								"title": map[string]any{
									"title": []map[string]any{
										{
											"type":       "text",
											"plain_text": "搜索命中",
											"text": map[string]any{
												"content": "搜索命中",
											},
										},
									},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "search.query",
		InputJSON:      `{"query":"搜索"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion search execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["title"] != "搜索命中" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesFeishuDocumentBlockGet(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Profile:  "feishu_bot_ops",
		},
		Profiles: map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		},
	})

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/open-apis/auth/v3/tenant_access_token/internal":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code":                0,
						"msg":                 "ok",
						"tenant_access_token": "tenant-token",
						"expire":              7200,
					}), nil
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
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
												"content": "执行器正文",
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
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, nil),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.block.get",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_2"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["plain_text"] != "执行器正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestExecutorExecutesFeishuDocumentBlockUpdate(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Profile:  "feishu_bot_ops",
		},
		Profiles: map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		},
	})

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/open-apis/auth/v3/tenant_access_token/internal":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code":                0,
						"msg":                 "ok",
						"tenant_access_token": "tenant-token",
						"expire":              7200,
					}), nil
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
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
												"content": "执行器更新正文",
											},
										},
									},
								},
							},
							"document_revision_id": 18,
						},
					}), nil
				default:
					t.Fatalf("unexpected request path: %s", request.URL.Path)
					return nil, nil
				}
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, nil),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.block.update",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_2","text":"执行器更新正文"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["plain_text"] != "执行器更新正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestExecutorExecutesNotionBlockListChildren(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/blocks/block_demo/children" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "blk_1",
							"type":         "paragraph",
							"has_children": false,
							"in_trash":     false,
							"paragraph": map[string]any{
								"rich_text": []map[string]any{
									{
										"type":       "text",
										"plain_text": "结构化正文",
										"text": map[string]any{
											"content": "结构化正文",
										},
									},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "block.list_children",
		InputJSON:      `{"block_id":"block_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["plain_text"] != "结构化正文" {
		t.Fatalf("unexpected plain_text: %+v", items[0]["plain_text"])
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonHTTPResponse(t *testing.T, statusCode int, value any) *http.Response {
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

func newTestStore(t *testing.T, cfg *config.Config) *config.Store {
	t.Helper()

	store := config.NewStore(t.TempDir() + "/config.yaml")
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}
	return store
}

func newTestRegistry(t *testing.T, feishuClient *feishuadapter.Client, notionClient *notionadapter.Client) *adapter.Registry {
	t.Helper()

	registry := adapter.NewRegistry()

	if feishuClient == nil {
		client, err := feishuadapter.NewClient(feishuadapter.Options{})
		if err != nil {
			t.Fatalf("failed to construct feishu client: %v", err)
		}
		feishuClient = client
	}
	if notionClient == nil {
		client, err := notionadapter.NewClient(notionadapter.Options{})
		if err != nil {
			t.Fatalf("failed to construct notion client: %v", err)
		}
		notionClient = client
	}

	feishuadapter.RegisterOperations(registry, feishuClient)
	notionadapter.RegisterOperations(registry, notionClient)
	return registry
}
