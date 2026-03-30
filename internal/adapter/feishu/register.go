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
		Operation:       "feishu.calendar.calendar.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarCalendarListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListCalendars(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarEventCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateCalendarEvent(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarEventListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListCalendarEvents(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarEventGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetCalendarEvent(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarEventUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateCalendarEvent(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.calendar.event.delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            calendarEventDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.DeleteCalendarEvent(ctx, executionProfileFromCall(call), call.Input)
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
			return client.CreateDocument(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocument(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.list_blocks",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentListBlocksSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListDocumentBlocks(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsBlockGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentBlock(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.list_children",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsBlockListChildrenSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentBlockChildren(ctx, executionProfileFromCall(call), call.Input)
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
			return client.GetDocumentBlockDescendants(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsBlockUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateDocumentBlock(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.block.batch_delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsBlockBatchDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BatchDeleteDocumentBlockChildren(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.space.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            wikiSpaceListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListWikiSpaces(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.node.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            wikiNodeListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListWikiNodes(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.wiki.node.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            wikiNodeCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateWikiNode(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.append_blocks",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentAppendBlocksSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.AppendDocumentBlocks(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
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
			return client.EditDocument(ctx, executionProfileFromCall(call), call.Input, call.IdempotencyKey)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.get_raw_content",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentGetRawContentSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDocumentRawContent(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.docs.document.share",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            docsDocumentShareSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ShareDocument(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.contact.user.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            contactUserGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetUser(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.contact.user.search",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            contactUserSearchSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.SearchUsers(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.contact.department.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            contactDepartmentListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListDepartments(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.department.user.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            departmentUserListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListDepartmentUsers(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.table.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableTableListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListBitableTables(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.field.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableFieldListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListBitableFields(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.list",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListBitableRecords(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.get",
		Platform:        "feishu",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetBitableRecord(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateBitableRecord(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.batch_create",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordBatchCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BatchCreateBitableRecords(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateBitableRecord(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.batch_update",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordBatchUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BatchUpdateBitableRecords(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.DeleteBitableRecord(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "feishu.bitable.record.batch_delete",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"bot", "user"},
		Spec:            bitableRecordBatchDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BatchDeleteBitableRecords(ctx, executionProfileFromCall(call), call.Input)
		},
	})

}
