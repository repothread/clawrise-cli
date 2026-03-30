package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestExecutorDryRunSuccess(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
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
	if envelope.Context.Account != "feishu_bot_ops" {
		t.Fatalf("unexpected context account: %s", envelope.Context.Account)
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
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
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
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "feishu.calendar.event.list",
		AccountName:    "notion_team_docs",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected execution to fail because of subject mismatch")
	}
	if envelope.Error == nil || envelope.Error.Code != "ACCOUNT_PLATFORM_MISMATCH" {
		t.Fatalf("unexpected error payload: %+v", envelope.Error)
	}
}

func TestExecutorUsesDefaultSubjectToSelectMatchingConnection(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Subject:  "user",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				Grant: config.Grant{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" {
		t.Fatalf("unexpected selected account: %+v", envelope.Context)
	}
	if envelope.Context.Subject != "user" {
		t.Fatalf("unexpected selected subject: %+v", envelope.Context)
	}
}

func TestExecutorExplicitSubjectSkipsMismatchedDefaultConnection(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				Grant: config.Grant{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		SubjectName:    "user",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" {
		t.Fatalf("unexpected selected account: %+v", envelope.Context)
	}
	if envelope.Context.Subject != "user" {
		t.Fatalf("unexpected selected subject: %+v", envelope.Context)
	}
}

func TestExecutorExplicitAccountOverridesDefaultSubject(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Subject:  "bot",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				Grant: config.Grant{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		AccountName:    "feishu_user_alice",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected explicit account to override default subject, got: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" || envelope.Context.Subject != "user" {
		t.Fatalf("unexpected context: %+v", envelope.Context)
	}
}

func TestExecutorExecutesNotionPageGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
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
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
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
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
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
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
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
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
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

func TestExecutorExecutesFeishuDocumentBlockDescendants(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
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
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_root/children":
					if request.URL.Query().Get("with_descendants") != "true" {
						t.Fatalf("unexpected with_descendants: %s", request.URL.Query().Get("with_descendants"))
					}
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code": 0,
						"msg":  "success",
						"data": map[string]any{
							"items": []map[string]any{
								{
									"block_id":   "blk_child",
									"parent_id":  "blk_root",
									"children":   []string{},
									"block_type": 2,
									"text": map[string]any{
										"elements": []map[string]any{
											{
												"text_run": map[string]any{
													"content": "后代正文",
												},
											},
										},
									},
								},
							},
							"has_more": false,
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
		OperationInput: "docs.block.get_descendants",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_root"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["plain_text"] != "后代正文" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesNotionDataSourceQuery(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/data_sources/ds_demo/query" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"type":     "page_or_data_source",
					"has_more": false,
					"results": []map[string]any{
						{
							"object": "page",
							"id":     "page_demo",
							"properties": map[string]any{
								"title": map[string]any{
									"title": []map[string]any{
										{
											"type":       "text",
											"plain_text": "数据源命中",
											"text": map[string]any{
												"content": "数据源命中",
											},
										},
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

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "data_source.query",
		InputJSON:      `{"data_source_id":"ds_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["title"] != "数据源命中" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesNotionBlockListChildren(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
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

func TestExecutorPersistsIdempotencyAndReplaysWrite(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Runtime: config.RuntimeConfig{
			Retry: config.RetryConfig{
				MaxAttempts: 0,
			},
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "feishu.demo.write",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"bot"},
		Spec: adapter.OperationSpec{
			Summary: "测试写操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			return map[string]any{
				"message": call.Input["message"],
				"id":      "demo_1",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	firstEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}
	if !firstEnvelope.OK {
		t.Fatalf("expected first execution to succeed, got: %+v", firstEnvelope.Error)
	}
	if firstEnvelope.Idempotency == nil || firstEnvelope.Idempotency.Status != "executed" || !firstEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected first idempotency state: %+v", firstEnvelope.Idempotency)
	}

	secondEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if !secondEnvelope.OK {
		t.Fatalf("expected second execution to succeed, got: %+v", secondEnvelope.Error)
	}
	if secondEnvelope.Idempotency == nil || secondEnvelope.Idempotency.Status != "replayed" || !secondEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected replay idempotency state: %+v", secondEnvelope.Idempotency)
	}
	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}

	idempotencyDir := filepath.Join(filepath.Dir(store.Path()), "runtime", "idempotency")
	entries, err := os.ReadDir(idempotencyDir)
	if err != nil {
		t.Fatalf("failed to read idempotency dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one idempotency record, got %d", len(entries))
	}
}

func TestExecutorRejectsIdempotencyConflict(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "feishu.demo.write",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"bot"},
		Spec: adapter.OperationSpec{
			Summary: "测试写操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			return map[string]any{
				"message": call.Input["message"],
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	if _, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		IdempotencyKey: "idem-fixed",
		InputJSON:      `{"message":"hello"}`,
	}); err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		IdempotencyKey: "idem-fixed",
		InputJSON:      `{"message":"world"}`,
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected idempotency conflict to fail")
	}
	if envelope.Error == nil || envelope.Error.Code != "IDEMPOTENCY_KEY_CONFLICT" {
		t.Fatalf("unexpected conflict error: %+v", envelope.Error)
	}
	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}
}

func TestExecutorRetriesRetryableReadByConfig(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Runtime: config.RuntimeConfig{
			Retry: config.RetryConfig{
				MaxAttempts: 1,
				BaseDelayMS: 1,
				MaxDelayMS:  1,
			},
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "notion.demo.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "测试读操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			if callCount == 1 {
				return nil, apperr.New("TEMPORARY", "temporary upstream error").WithRetryable(true)
			}
			return map[string]any{
				"id": "ok",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.get",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected retry to succeed, got: %+v", envelope.Error)
	}
	if envelope.Meta.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %+v", envelope.Meta)
	}
	if callCount != 2 {
		t.Fatalf("expected handler to run twice, got %d", callCount)
	}
}

func TestExecutorWritesRedactedAuditLog(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromProfiles(map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "notion.demo.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "测试审计。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{
				"access_token":  "secret-token",
				"authorization": "Bearer top-secret",
				"message":       "safe",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.get",
		InputJSON:      `{"token":"very-secret","note":"visible"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected audit test execution to succeed, got: %+v", envelope.Error)
	}

	auditFile := filepath.Join(filepath.Dir(store.Path()), "runtime", "audit", time.Now().UTC().Format("2006-01-02")+".jsonl")
	data, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "very-secret") || strings.Contains(content, "secret-token") || strings.Contains(content, "top-secret") {
		t.Fatalf("expected audit log to redact secrets, got: %s", content)
	}
	if !strings.Contains(content, `"note":"visible"`) {
		t.Fatalf("expected audit log to keep non-sensitive fields, got: %s", content)
	}
	if !strings.Contains(content, `"authorization":"***"`) {
		t.Fatalf("expected audit log to redact authorization field, got: %s", content)
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

func accountsFromProfiles(profiles map[string]config.Profile) map[string]config.Account {
	accounts := make(map[string]config.Account, len(profiles))
	for name, profile := range profiles {
		method := strings.TrimSpace(profile.Method)
		if method == "" {
			switch strings.TrimSpace(profile.Grant.Type) {
			case "client_credentials":
				method = "feishu.app_credentials"
			case "oauth_user":
				method = "feishu.oauth_user"
			case "static_token":
				method = "notion.internal_token"
			case "oauth_refreshable":
				method = "notion.oauth_public"
			}
		}

		account := config.Account{
			Title:    profile.Title,
			Platform: profile.Platform,
			Subject:  profile.Subject,
			Auth: config.AccountAuth{
				Method:     method,
				Public:     map[string]any{},
				SecretRefs: map[string]string{},
			},
		}

		switch method {
		case "feishu.app_credentials":
			if strings.TrimSpace(profile.Params.AppID) != "" {
				account.Auth.Public["app_id"] = strings.TrimSpace(profile.Params.AppID)
			} else if strings.TrimSpace(profile.Grant.AppID) != "" {
				account.Auth.Public["app_id"] = strings.TrimSpace(profile.Grant.AppID)
			}
			if strings.TrimSpace(profile.Grant.AppSecret) != "" {
				account.Auth.SecretRefs["app_secret"] = strings.TrimSpace(profile.Grant.AppSecret)
			}
		case "feishu.oauth_user":
			if strings.TrimSpace(profile.Params.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Params.ClientID)
			} else if strings.TrimSpace(profile.Grant.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Grant.ClientID)
			}
			if strings.TrimSpace(profile.Params.RedirectMode) != "" {
				account.Auth.Public["redirect_mode"] = strings.TrimSpace(profile.Params.RedirectMode)
			}
			if len(profile.Params.Scopes) > 0 {
				account.Auth.Public["scopes"] = append([]string(nil), profile.Params.Scopes...)
			}
			if strings.TrimSpace(profile.Grant.ClientSecret) != "" {
				account.Auth.SecretRefs["client_secret"] = strings.TrimSpace(profile.Grant.ClientSecret)
			}
			if strings.TrimSpace(profile.Grant.AccessToken) != "" {
				account.Auth.SecretRefs["access_token"] = strings.TrimSpace(profile.Grant.AccessToken)
			}
			if strings.TrimSpace(profile.Grant.RefreshToken) != "" {
				account.Auth.SecretRefs["refresh_token"] = strings.TrimSpace(profile.Grant.RefreshToken)
			}
		case "notion.internal_token":
			if strings.TrimSpace(profile.Params.NotionVersion) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Params.NotionVersion)
			} else if strings.TrimSpace(profile.Grant.NotionVer) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Grant.NotionVer)
			}
			if strings.TrimSpace(profile.Grant.Token) != "" {
				account.Auth.SecretRefs["token"] = strings.TrimSpace(profile.Grant.Token)
			}
		case "notion.oauth_public":
			if strings.TrimSpace(profile.Params.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Params.ClientID)
			} else if strings.TrimSpace(profile.Grant.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Grant.ClientID)
			}
			if strings.TrimSpace(profile.Params.NotionVersion) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Params.NotionVersion)
			} else if strings.TrimSpace(profile.Grant.NotionVer) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Grant.NotionVer)
			}
			if strings.TrimSpace(profile.Params.RedirectMode) != "" {
				account.Auth.Public["redirect_mode"] = strings.TrimSpace(profile.Params.RedirectMode)
			}
			if len(profile.Params.Scopes) > 0 {
				account.Auth.Public["scopes"] = append([]string(nil), profile.Params.Scopes...)
			}
			if strings.TrimSpace(profile.Grant.ClientSecret) != "" {
				account.Auth.SecretRefs["client_secret"] = strings.TrimSpace(profile.Grant.ClientSecret)
			}
			if strings.TrimSpace(profile.Grant.AccessToken) != "" {
				account.Auth.SecretRefs["access_token"] = strings.TrimSpace(profile.Grant.AccessToken)
			}
			if strings.TrimSpace(profile.Grant.RefreshToken) != "" {
				account.Auth.SecretRefs["refresh_token"] = strings.TrimSpace(profile.Grant.RefreshToken)
			}
		}

		if len(account.Auth.Public) == 0 {
			account.Auth.Public = nil
		}
		if len(account.Auth.SecretRefs) == 0 {
			account.Auth.SecretRefs = nil
		}
		accounts[name] = account
	}
	return accounts
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
