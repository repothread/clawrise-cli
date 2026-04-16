package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// GetDatabase reads database metadata and exposes child data source summaries.
// 获取一个 database，并把它下面的 data source 摘要一并标准化返回。
func (c *Client) GetDatabase(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	databaseID, appErr := requireIDField(input, "database_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/databases/"+url.PathEscape(databaseID),
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion database response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "database id is empty in Notion response")
	}
	return normalizeDatabaseObject(response), nil
}

// CreateDatabase creates one database with either a raw provider-native body or a common shorthand payload.
// 创建 database 时，既支持直接透传 body，也支持常见字段的简写输入。
func (c *Client) CreateDatabase(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildDatabaseCreateRequestPayload(profile, input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/databases",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion database create response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "database id is empty in Notion create response")
	}
	return normalizeDatabaseObject(response), nil
}

// UpdateDatabase updates database metadata or placement.
// 更新 database 的元数据，或调整它的 parent。
func (c *Client) UpdateDatabase(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	databaseID, appErr := requireIDField(input, "database_id")
	if appErr != nil {
		return nil, appErr
	}

	payload, appErr := buildUpdateDatabasePayload(profile, input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPatch,
		"/v1/databases/"+url.PathEscape(databaseID),
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion database update response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "database id is empty in Notion update response")
	}
	return normalizeDatabaseObject(response), nil
}

// buildDatabaseCreateRequestPayload builds the request body for database creation.
// 为 database.create 构造请求体；常见场景优先走简写字段，复杂场景仍可直接透传 body。
func buildDatabaseCreateRequestPayload(profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	if appErr := validateTopLevelInputFields("notion.database.create", input, notionDatabaseCreateSpec().Input, nil); appErr != nil {
		return nil, appErr
	}

	if body, exists := input["body"]; exists {
		record, ok := asMap(body)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "body must be an object")
		}
		return cloneMap(record), nil
	}

	parent, appErr := normalizeDatabaseRequestParent(profile, input["parent"])
	if appErr != nil {
		return nil, appErr
	}

	title, provided, appErr := buildOptionalRichTextInput(input["title"], "title")
	if appErr != nil {
		return nil, appErr
	}
	if !provided || len(title) == 0 {
		return nil, apperr.New("INVALID_INPUT", "title is required when body is not provided")
	}

	initialDataSource, ok := asMap(input["initial_data_source"])
	if !ok || len(initialDataSource) == 0 {
		return nil, apperr.New("INVALID_INPUT", "initial_data_source is required when body is not provided")
	}

	payload := map[string]any{
		"parent":              parent,
		"title":               title,
		"initial_data_source": cloneMap(initialDataSource),
	}

	if description, provided, appErr := buildOptionalRichTextInput(input["description"], "description"); appErr != nil {
		return nil, appErr
	} else if provided {
		payload["description"] = description
	}
	if isInline, ok := asBool(input["is_inline"]); ok {
		payload["is_inline"] = isInline
	}
	if icon, exists := input["icon"]; exists {
		normalized, appErr := normalizeNotionFileObject(icon, true)
		if appErr != nil {
			return nil, appErr
		}
		payload["icon"] = normalized
	}
	if cover, exists := input["cover"]; exists {
		normalized, appErr := normalizeNotionFileObject(cover, false)
		if appErr != nil {
			return nil, appErr
		}
		payload["cover"] = normalized
	}
	return payload, nil
}

// buildUpdateDatabasePayload builds the request body for database updates.
// database.update 的简写字段会映射到新版 Notion API 的原生字段。
func buildUpdateDatabasePayload(profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	if appErr := validateTopLevelInputFields("notion.database.update", input, notionDatabaseUpdateSpec().Input, nil); appErr != nil {
		return nil, appErr
	}

	if body, exists := input["body"]; exists {
		record, ok := asMap(body)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "body must be an object")
		}
		return cloneMap(record), nil
	}

	payload := map[string]any{}
	if parent, exists := input["parent"]; exists {
		normalizedParent, appErr := normalizeDatabaseRequestParent(profile, parent)
		if appErr != nil {
			return nil, appErr
		}
		payload["parent"] = normalizedParent
	}
	if title, provided, appErr := buildOptionalRichTextInput(input["title"], "title"); appErr != nil {
		return nil, appErr
	} else if provided {
		payload["title"] = title
	}
	if description, provided, appErr := buildOptionalRichTextInput(input["description"], "description"); appErr != nil {
		return nil, appErr
	} else if provided {
		payload["description"] = description
	}
	if inTrash, ok := asBool(input["in_trash"]); ok {
		payload["in_trash"] = inTrash
	}
	if isLocked, ok := asBool(input["is_locked"]); ok {
		payload["is_locked"] = isLocked
	}
	if icon, exists := input["icon"]; exists {
		normalized, appErr := normalizeNotionFileObject(icon, true)
		if appErr != nil {
			return nil, appErr
		}
		payload["icon"] = normalized
	}
	if cover, exists := input["cover"]; exists {
		normalized, appErr := normalizeNotionFileObject(cover, false)
		if appErr != nil {
			return nil, appErr
		}
		payload["cover"] = normalized
	}
	if len(payload) == 0 {
		return nil, apperr.New("INVALID_INPUT", "at least one updatable field is required")
	}
	return payload, nil
}

// normalizeDatabaseRequestParent normalizes the supported shorthand parent shapes for database requests.
// 这里显式收敛可接受的 parent 形状，避免 task 层到 provider 层之间出现语义漂移。
func normalizeDatabaseRequestParent(profile ExecutionProfile, raw any) (map[string]any, *apperr.AppError) {
	parent, ok := asMap(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "parent must be an object")
	}

	parentType, ok := asString(parent["type"])
	if !ok || strings.TrimSpace(parentType) == "" {
		return nil, apperr.New("INVALID_INPUT", "parent.type is required")
	}
	parentType = strings.TrimSpace(parentType)

	switch parentType {
	case "page_id", "database_id":
		parentID, ok := asString(parent["id"])
		if !ok || strings.TrimSpace(parentID) == "" {
			if directID, ok := asString(parent[parentType]); ok && strings.TrimSpace(directID) != "" {
				parentID = directID
			} else {
				return nil, apperr.New("INVALID_INPUT", "parent.id is required")
			}
		}
		return map[string]any{
			parentType: strings.TrimSpace(parentID),
		}, nil
	case "workspace":
		profile = normalizeExecutionProfile(profile)
		if profile.Method != "notion.oauth_public" {
			return nil, apperr.New("INVALID_INPUT", "workspace-level database creation requires a public Notion integration profile")
		}
		return map[string]any{
			"workspace": true,
		}, nil
	default:
		return nil, apperr.New("INVALID_INPUT", "parent.type must be one of page_id, database_id, or workspace")
	}
}

// buildOptionalRichTextInput supports either a plain string or a provider-native rich text array.
// 这样 AI 既可以传简单字符串，也可以在需要时传原生 rich_text 结构。
func buildOptionalRichTextInput(raw any, field string) ([]map[string]any, bool, *apperr.AppError) {
	if raw == nil {
		return nil, false, nil
	}

	if text, ok := asString(raw); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return []map[string]any{}, true, nil
		}
		return buildPlainTextRichText(text), true, nil
	}

	items, ok := asArray(raw)
	if !ok {
		return nil, false, apperr.New("INVALID_INPUT", field+" must be a string or an array of rich text objects")
	}

	normalized := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := asMap(item)
		if !ok {
			return nil, false, apperr.New("INVALID_INPUT", "each "+field+" item must be an object")
		}
		normalized = append(normalized, cloneMap(record))
	}
	return normalized, true, nil
}

func normalizeDatabaseObject(item map[string]any) map[string]any {
	result := map[string]any{
		"database_id": extractFirstString(item, "id"),
		"object":      "database",
		"raw":         cloneMap(item),
	}
	if titleItems, ok := asArray(item["title"]); ok {
		result["title"] = extractRichTextPlainText(titleItems)
		result["title_rich_text"] = titleItems
	}
	if descriptionItems, ok := asArray(item["description"]); ok {
		result["description"] = extractRichTextPlainText(descriptionItems)
		result["description_rich_text"] = descriptionItems
	}
	if parent, ok := asMap(item["parent"]); ok && len(parent) > 0 {
		result["parent"] = cloneMap(parent)
	}
	if dataSources, ok := asArray(item["data_sources"]); ok {
		normalized := normalizeDatabaseDataSourceSummaries(dataSources)
		result["data_sources"] = normalized
		result["data_source_count"] = len(normalized)
	}
	if urlValue := extractFirstString(item, "url"); urlValue != "" {
		result["url"] = urlValue
	}
	if publicURL := extractFirstString(item, "public_url"); publicURL != "" {
		result["public_url"] = publicURL
	}
	if createdTime := extractFirstString(item, "created_time"); createdTime != "" {
		result["created_time"] = createdTime
	}
	if lastEditedTime := extractFirstString(item, "last_edited_time"); lastEditedTime != "" {
		result["last_edited_time"] = lastEditedTime
	}
	if inTrash, ok := asBool(item["in_trash"]); ok {
		result["in_trash"] = inTrash
		result["archived"] = inTrash
	}
	if isInline, ok := asBool(item["is_inline"]); ok {
		result["is_inline"] = isInline
	}
	if isLocked, ok := asBool(item["is_locked"]); ok {
		result["is_locked"] = isLocked
	}
	if icon, ok := asMap(item["icon"]); ok && len(icon) > 0 {
		result["icon"] = cloneMap(icon)
	}
	if cover, ok := asMap(item["cover"]); ok && len(cover) > 0 {
		result["cover"] = cloneMap(cover)
	}
	return result
}

func normalizeDatabaseDataSourceSummaries(items []any) []map[string]any {
	normalized := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := asMap(item)
		if !ok || len(record) == 0 {
			continue
		}

		summary := map[string]any{
			"data_source_id": extractFirstString(record, "id"),
			"raw":            cloneMap(record),
		}
		if name := extractFirstString(record, "name"); name != "" {
			summary["name"] = name
		}
		if urlValue := extractFirstString(record, "url"); urlValue != "" {
			summary["url"] = urlValue
		}
		normalized = append(normalized, summary)
	}
	return normalized
}
