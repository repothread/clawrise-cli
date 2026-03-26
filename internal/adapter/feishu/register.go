package feishu

import (
	"context"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// RegisterOperations 将飞书 operation 注册到统一 registry。
func RegisterOperations(registry *adapter.Registry, client *Client) {
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateCalendarEvent(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.space.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListWikiSpaces(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.node.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListWikiNodes(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.node.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateWikiNode(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.append_blocks",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.AppendDocumentBlocks(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.get_raw_content",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentRawContent(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.contact.user.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
	})
}
