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
			return client.CreatePage(ctx, call.Profile, call.Input)
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
			return client.GetPage(ctx, call.Profile, call.Input)
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
			return client.UpdatePage(ctx, call.Profile, call.Input)
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
			return client.GetPageMarkdown(ctx, call.Profile, call.Input)
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
			return client.UpdatePageMarkdown(ctx, call.Profile, call.Input)
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
			return client.Search(ctx, call.Profile, call.Input)
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
			return client.ListComments(ctx, call.Profile, call.Input)
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
			return client.CreateComment(ctx, call.Profile, call.Input)
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
			return client.GetDataSource(ctx, call.Profile, call.Input)
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
			return client.QueryDataSource(ctx, call.Profile, call.Input)
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
			return client.GetBlock(ctx, call.Profile, call.Input)
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
			return client.ListBlockChildren(ctx, call.Profile, call.Input)
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
			return client.AppendBlockChildren(ctx, call.Profile, call.Input)
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
			return client.UpdateBlock(ctx, call.Profile, call.Input)
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
			return client.DeleteBlock(ctx, call.Profile, call.Input)
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
			return client.GetUser(ctx, call.Profile, call.Input)
		},
	})
}
