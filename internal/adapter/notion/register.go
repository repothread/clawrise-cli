package notion

import (
	"context"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// RegisterOperations registers Notion operations into the shared registry.
func RegisterOperations(registry *adapter.Registry, client *Client) {
	registry.Register(adapter.Definition{
		Operation:       "notion.page.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreatePage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.page.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetPage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.page.property_item.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPagePropertyItemGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetPagePropertyItem(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.page.update",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdatePage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.page.markdown.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageMarkdownGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetPageMarkdown(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.page.markdown.update",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageMarkdownUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdatePageMarkdown(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.page.import_markdown",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskPageImportMarkdownSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ImportMarkdownPage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.page.upsert_markdown_child",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskPageUpsertMarkdownChildSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpsertMarkdownChildPage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.search.query",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionSearchQuerySpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.Search(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.comment.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionCommentGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetComment(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.comment.list",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionCommentListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListComments(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.comment.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionCommentCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateComment(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.data_source.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDataSourceGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDataSource(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.data_source.template.list",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDataSourceTemplateListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListDataSourceTemplates(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.data_source.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDataSourceCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateDataSource(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.data_source.update",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDataSourceUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateDataSource(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.data_source.query",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDataSourceQuerySpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.QueryDataSource(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.file_upload.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  30 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionFileUploadCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateFileUpload(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.file_upload.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionFileUploadGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetFileUpload(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.file_upload.list",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionFileUploadListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListFileUploads(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.file_upload.send",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  60 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionFileUploadSendSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.SendFileUpload(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.file_upload.complete",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  30 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionFileUploadCompleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CompleteFileUpload(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetBlock(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.list_children",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockListChildrenSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListBlockChildren(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.get_descendants",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockGetDescendantsSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetBlockDescendants(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.append",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockAppendSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.AppendBlockChildren(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.update",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateBlock(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.block.delete",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionBlockDeleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.DeleteBlock(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.user.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionUserGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetUser(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.user.list",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionUserListSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ListUsers(ctx, executionProfileFromCall(call), call.Input)
		},
	})
}
