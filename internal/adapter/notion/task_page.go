package notion

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// ImportMarkdownPage 读取 Markdown 文本或本地文件，并在指定父页面下创建一个子页面。
// ImportMarkdownPage creates one child page under the target parent page from inline Markdown or a local file.
func (c *Client) ImportMarkdownPage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	parentPageID, appErr := requireIDField(input, "parent_page_id")
	if appErr != nil {
		return nil, appErr
	}

	markdown, source, appErr := resolveMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}

	createInput := map[string]any{
		"parent": map[string]any{
			"type": "page_id",
			"id":   parentPageID,
		},
		"markdown": markdown,
	}
	if title, ok := asString(input["title"]); ok && strings.TrimSpace(title) != "" {
		createInput["title"] = strings.TrimSpace(title)
	}
	copyOptionalTaskFields(input, createInput, "position", "after", "template")

	data, appErr := c.CreatePage(ctx, profile, createInput)
	if appErr != nil {
		return nil, appErr
	}

	return map[string]any{
		"action":         "created",
		"parent_page_id": parentPageID,
		"page_id":        data["page_id"],
		"title":          data["title"],
		"source":         source,
		"page":           cloneMap(data),
	}, nil
}

// UpsertMarkdownChildPage 按标题查找父页面下的直接子页面；找到则覆盖正文，找不到则创建。
// UpsertMarkdownChildPage looks up one direct child page by title under the parent page; when found it replaces the markdown body, otherwise it creates a new page.
func (c *Client) UpsertMarkdownChildPage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	parentPageID, appErr := requireIDField(input, "parent_page_id")
	if appErr != nil {
		return nil, appErr
	}
	title, appErr := requireIDField(input, "title")
	if appErr != nil {
		return nil, appErr
	}

	markdown, source, appErr := resolveMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}
	createIfMissing := true
	if value, ok := asBool(input["create_if_missing"]); ok {
		createIfMissing = value
	}

	searchPageSize := 100
	if value, ok := asInt(input["search_page_size"]); ok && value > 0 {
		searchPageSize = value
	}

	searchData, appErr := c.Search(ctx, profile, map[string]any{
		"query": title,
		"filter": map[string]any{
			"property": "object",
			"value":    "page",
		},
		"page_size": searchPageSize,
	})
	if appErr != nil {
		return nil, appErr
	}

	matches := findExactChildPageMatches(searchData["items"], parentPageID, title)
	switch len(matches) {
	case 0:
		if !createIfMissing {
			return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no child page named %q was found under parent page %s", title, parentPageID))
		}

		createInput := map[string]any{
			"parent": map[string]any{
				"type": "page_id",
				"id":   parentPageID,
			},
			"title":    title,
			"markdown": markdown,
		}
		copyOptionalTaskFields(input, createInput, "position", "after", "template")

		data, appErr := c.CreatePage(ctx, profile, createInput)
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"action":         "created",
			"parent_page_id": parentPageID,
			"page_id":        data["page_id"],
			"title":          data["title"],
			"source":         source,
			"matched_count":  0,
			"page":           cloneMap(data),
		}, nil
	case 1:
		pageID, _ := asString(matches[0]["id"])
		updateData, appErr := c.UpdatePageMarkdown(ctx, profile, map[string]any{
			"page_id": pageID,
			"type":    "replace_content",
			"replace_content": map[string]any{
				"new_str": markdown,
			},
		})
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"action":         "updated",
			"parent_page_id": parentPageID,
			"page_id":        pageID,
			"title":          title,
			"source":         source,
			"matched_count":  1,
			"page":           cloneMap(matches[0]),
			"markdown_page":  cloneMap(updateData),
		}, nil
	default:
		matchIDs := make([]string, 0, len(matches))
		for _, item := range matches {
			pageID, _ := asString(item["id"])
			if strings.TrimSpace(pageID) != "" {
				matchIDs = append(matchIDs, strings.TrimSpace(pageID))
			}
		}
		return nil, apperr.New("AMBIGUOUS_TARGET", fmt.Sprintf("found %d child pages named %q under parent page %s: %s", len(matches), title, parentPageID, strings.Join(matchIDs, ", ")))
	}
}

// resolveMarkdownTaskSource 统一处理 Markdown 任务命令的内联文本和本地文件输入。
// resolveMarkdownTaskSource resolves either inline Markdown or one local file for the task-level Markdown commands.
func resolveMarkdownTaskSource(input map[string]any) (string, string, *apperr.AppError) {
	rawMarkdown, hasMarkdown := input["markdown"]
	rawFilePath, hasFilePath := input["file_path"]
	if hasMarkdown == hasFilePath {
		return "", "", apperr.New("INVALID_INPUT", "provide exactly one of markdown or file_path")
	}

	if hasMarkdown {
		markdown, ok := asString(rawMarkdown)
		if !ok {
			return "", "", apperr.New("INVALID_INPUT", "markdown must be a string")
		}
		return markdown, "inline_markdown", nil
	}

	filePath, ok := asString(rawFilePath)
	if !ok || strings.TrimSpace(filePath) == "" {
		return "", "", apperr.New("INVALID_INPUT", "file_path must be a non-empty string")
	}
	data, err := os.ReadFile(strings.TrimSpace(filePath))
	if err != nil {
		return "", "", apperr.New("INVALID_INPUT", fmt.Sprintf("failed to read markdown file: %v", err))
	}
	return string(data), "file_path", nil
}

// findExactChildPageMatches 只保留标题完全匹配且父页面一致的直接子页面。
// findExactChildPageMatches keeps only direct child pages whose titles match exactly and whose parent page id matches the requested parent.
func findExactChildPageMatches(raw any, parentPageID string, title string) []map[string]any {
	items, ok := raw.([]map[string]any)
	if !ok || len(items) == 0 {
		return nil
	}

	expectedParentID := strings.TrimSpace(parentPageID)
	expectedTitle := strings.TrimSpace(title)
	seenPageIDs := map[string]struct{}{}
	matches := make([]map[string]any, 0, len(items))
	for _, item := range items {
		objectType, _ := asString(item["object"])
		if strings.TrimSpace(objectType) != "page" {
			continue
		}
		itemTitle, _ := asString(item["title"])
		if strings.TrimSpace(itemTitle) != expectedTitle {
			continue
		}
		if archived, ok := asBool(item["archived"]); ok && archived {
			continue
		}
		if inTrash, ok := asBool(item["in_trash"]); ok && inTrash {
			continue
		}

		parent, ok := asMap(item["parent"])
		if !ok {
			continue
		}
		parentID, _ := asString(parent["page_id"])
		if strings.TrimSpace(parentID) != expectedParentID {
			continue
		}

		pageID, _ := asString(item["id"])
		pageID = strings.TrimSpace(pageID)
		if pageID == "" {
			continue
		}
		if _, exists := seenPageIDs[pageID]; exists {
			continue
		}
		seenPageIDs[pageID] = struct{}{}
		matches = append(matches, cloneMap(item))
	}
	return matches
}

// copyOptionalTaskFields 把组合命令允许透传的创建参数复制到实际 provider 调用输入里。
// copyOptionalTaskFields copies the optional create fields that the task-level commands are allowed to forward into the provider-native call input.
func copyOptionalTaskFields(source map[string]any, target map[string]any, fields ...string) {
	for _, field := range fields {
		value, exists := source[field]
		if !exists {
			continue
		}
		target[field] = cloneDebugValue(value)
	}
}
