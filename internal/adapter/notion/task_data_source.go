package notion

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// UpsertDataSourceRow 通过 provider-native filter 查找唯一行记录，不存在时创建，存在时更新。
// UpsertDataSourceRow finds one unique row with a provider-native filter, creates it when missing, and updates it when present.
func (c *Client) UpsertDataSourceRow(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceID, appErr := requireIDField(input, "data_source_id")
	if appErr != nil {
		return nil, appErr
	}
	match, appErr := requireTaskDataSourceRowMatch(input["match"])
	if appErr != nil {
		return nil, appErr
	}

	markdown, source, hasMarkdown, appErr := resolveOptionalMarkdownTaskSource(input)
	if appErr != nil {
		return nil, appErr
	}
	if !taskRowHasPageMutation(input) && !hasMarkdown {
		return nil, apperr.New("INVALID_INPUT", "at least one of title, properties, markdown, or file_path is required")
	}

	createIfMissing := true
	if value, ok := asBool(input["create_if_missing"]); ok {
		createIfMissing = value
	}

	pageSize := 2
	if value, ok := asInt(input["page_size"]); ok && value > 0 {
		pageSize = value
	}

	queryInput := map[string]any{
		"data_source_id": dataSourceID,
		"filter":         match,
		"page_size":      pageSize,
	}
	if filterProperties, exists := input["filter_properties"]; exists {
		queryInput["filter_properties"] = cloneDebugValue(filterProperties)
	}

	queryData, appErr := c.QueryDataSource(ctx, profile, queryInput)
	if appErr != nil {
		return nil, appErr
	}

	matches := findUpsertDataSourceRowMatches(queryData["items"])
	switch len(matches) {
	case 0:
		if !createIfMissing {
			return nil, apperr.New("OBJECT_NOT_FOUND", fmt.Sprintf("no row matched the provided filter under data source %s", dataSourceID))
		}

		createInput := map[string]any{
			"parent": map[string]any{
				"type": "data_source_id",
				"id":   dataSourceID,
			},
		}
		copyOptionalTaskFields(input, createInput, "title", "title_property", "properties")
		if hasMarkdown {
			createInput["markdown"] = markdown
		}

		pageData, appErr := c.CreatePage(ctx, profile, createInput)
		if appErr != nil {
			return nil, appErr
		}

		result := map[string]any{
			"action":         "created",
			"data_source_id": dataSourceID,
			"matched_count":  0,
			"page_id":        pageData["page_id"],
			"title":          pageData["title"],
			"page":           cloneMap(pageData),
		}
		if hasMarkdown {
			result["source"] = source
		}
		return result, nil
	case 1:
		pageID, _ := asString(matches[0]["id"])
		pageID = strings.TrimSpace(pageID)
		result := map[string]any{
			"action":         "updated",
			"data_source_id": dataSourceID,
			"matched_count":  1,
			"page_id":        pageID,
		}

		if taskRowHasPageMutation(input) {
			pageUpdateInput := map[string]any{
				"page_id": pageID,
			}
			copyOptionalTaskFields(input, pageUpdateInput, "title", "title_property", "properties")

			pageData, appErr := c.UpdatePage(ctx, profile, pageUpdateInput)
			if appErr != nil {
				return nil, appErr
			}
			result["title"] = pageData["title"]
			result["page"] = cloneMap(pageData)
		} else {
			matchedTitle, _ := asString(matches[0]["title"])
			result["title"] = matchedTitle
			result["page"] = cloneMap(matches[0])
		}

		if hasMarkdown {
			markdownData, appErr := c.UpdatePageMarkdown(ctx, profile, map[string]any{
				"page_id": pageID,
				"type":    "replace_content",
				"replace_content": map[string]any{
					"new_str": markdown,
				},
			})
			if appErr != nil {
				return nil, appErr
			}
			result["source"] = source
			result["markdown_page"] = cloneMap(markdownData)
		}
		return result, nil
	default:
		matchIDs := make([]string, 0, len(matches))
		for _, item := range matches {
			pageID, _ := asString(item["id"])
			if strings.TrimSpace(pageID) != "" {
				matchIDs = append(matchIDs, strings.TrimSpace(pageID))
			}
		}
		return nil, apperr.New("AMBIGUOUS_TARGET", fmt.Sprintf("the provided filter matched %d row pages under data source %s: %s", len(matches), dataSourceID, strings.Join(matchIDs, ", ")))
	}
}

// resolveOptionalMarkdownTaskSource 在 markdown/file_path 是可选时复用相同的输入校验和文件读取逻辑。
// resolveOptionalMarkdownTaskSource reuses the same validation and file loading logic when markdown or file_path is optional.
func resolveOptionalMarkdownTaskSource(input map[string]any) (string, string, bool, *apperr.AppError) {
	_, hasMarkdown := input["markdown"]
	_, hasFilePath := input["file_path"]
	if !hasMarkdown && !hasFilePath {
		return "", "", false, nil
	}

	markdown, source, appErr := resolveMarkdownTaskSource(input)
	if appErr != nil {
		return "", "", false, appErr
	}
	return markdown, source, true, nil
}

// requireTaskDataSourceRowMatch 要求上层明确提供 provider-native 的唯一匹配过滤条件。
// requireTaskDataSourceRowMatch requires one explicit provider-native match filter that should uniquely identify the row.
func requireTaskDataSourceRowMatch(raw any) (map[string]any, *apperr.AppError) {
	match, ok := asMap(raw)
	if !ok || len(match) == 0 {
		return nil, apperr.New("INVALID_INPUT", "match must be a non-empty object")
	}
	return cloneMap(match), nil
}

// taskRowHasPageMutation 判断这次 upsert 是否需要更新 page properties 或 title。
// taskRowHasPageMutation reports whether this upsert needs a page property/title update.
func taskRowHasPageMutation(input map[string]any) bool {
	if value, ok := asString(input["title"]); ok && strings.TrimSpace(value) != "" {
		return true
	}
	if rawProperties, exists := input["properties"]; exists {
		properties, ok := asMap(rawProperties)
		return ok && len(properties) > 0
	}
	return false
}

// findUpsertDataSourceRowMatches 只保留正常 page 结果，忽略归档、回收站和非 page 对象。
// findUpsertDataSourceRowMatches keeps only live page results and ignores archived, trashed, or non-page query results.
func findUpsertDataSourceRowMatches(raw any) []map[string]any {
	items, ok := raw.([]map[string]any)
	if !ok || len(items) == 0 {
		return nil
	}

	seenPageIDs := map[string]struct{}{}
	matches := make([]map[string]any, 0, len(items))
	for _, item := range items {
		objectType, _ := asString(item["object"])
		if strings.TrimSpace(objectType) != "page" {
			continue
		}
		if archived, ok := asBool(item["archived"]); ok && archived {
			continue
		}
		if inTrash, ok := asBool(item["in_trash"]); ok && inTrash {
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
