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
		Operation:       "notion.database.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDatabaseGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetDatabase(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.database.create",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDatabaseCreateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.CreateDatabase(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.database.update",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionDatabaseUpdateSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpdateDatabase(ctx, executionProfileFromCall(call), call.Input)
		},
	})
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
		Operation:       "notion.page.move",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  10 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionPageMoveSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.MovePage(ctx, executionProfileFromCall(call), call.Input)
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
		Operation:       "notion.task.page.patch_section",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskPagePatchSectionSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.PatchPageSection(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.page.read_complete",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskPageReadCompleteSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ReadCompletePage(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.block.attach_file",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  30 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskBlockAttachFileSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.AttachFileBlock(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.meeting_notes.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskMeetingNotesGetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.GetMeetingNotes(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.database.resolve_target",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskDatabaseResolveTargetSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.ResolveDatabaseTarget(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.data_source.row.upsert",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  20 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskDataSourceRowUpsertSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.UpsertDataSourceRow(ctx, executionProfileFromCall(call), call.Input)
		},
	})
	registry.Register(adapter.Definition{
		Operation:       "notion.task.data_source.bulk_upsert",
		Platform:        "notion",
		Mutating:        true,
		DefaultTimeout:  30 * time.Second,
		AllowedSubjects: []string{"integration"},
		Spec:            notionTaskDataSourceBulkUpsertSpec(),
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return client.BulkUpsertDataSourceRows(ctx, executionProfileFromCall(call), call.Input)
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
