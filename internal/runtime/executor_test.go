package runtime

import (
	"context"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
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

	executor := NewExecutor(store, adapter.NewRegistry())
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

	executor := NewExecutor(store, adapter.NewRegistry())
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

	executor := NewExecutor(store, adapter.NewRegistry())
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

func newTestStore(t *testing.T, cfg *config.Config) *config.Store {
	t.Helper()

	store := config.NewStore(t.TempDir() + "/config.yaml")
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}
	return store
}
