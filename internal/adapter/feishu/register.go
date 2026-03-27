package feishu

import (
	"context"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// RegisterOperations registers Feishu operations into the shared registry.
func RegisterOperations(registry *adapter.Registry, client *Client) {
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            calendarEventCreateSpec(),
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
		Spec:            calendarEventListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListCalendarEvents(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            calendarEventGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetCalendarEvent(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            calendarEventUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateCalendarEvent(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            calendarEventDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.DeleteCalendarEvent(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateDocument(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsDocumentGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocument(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.list_blocks",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsDocumentListBlocksSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListDocumentBlocks(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsBlockGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentBlock(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.list_children",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsBlockListChildrenSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentBlockChildren(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.get_descendants",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsBlockGetDescendantsSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentBlockDescendants(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsBlockUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateDocumentBlock(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.batch_delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsBlockBatchDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BatchDeleteDocumentBlockChildren(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.space.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            wikiSpaceListSpec(),
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
		Spec:            wikiNodeListSpec(),
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
		Spec:            wikiNodeCreateSpec(),
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
		Spec:            docsDocumentAppendBlocksSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.AppendDocumentBlocks(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.edit",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentEditSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.EditDocument(ctx, call.Profile, call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.get_raw_content",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            docsDocumentGetRawContentSpec(),
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
		Spec:            contactUserGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetUser(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            bitableRecordListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListBitableRecords(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            bitableRecordGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetBitableRecord(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            bitableRecordCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateBitableRecord(ctx, call.Profile, call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot"},
		Spec:            bitableRecordUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateBitableRecord(ctx, call.Profile, call.Input)
		},
	})
}
