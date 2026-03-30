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

// CreateDataSource 创建一个 Notion data source。
func (c *Client) CreateDataSource(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildDataSourceWritePayload(input, false)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	// 2025-09-03 之后，页面下新增 data source 需要先走 Create Database API，
	// 再从返回的 database 中提取默认初始 data source。
	if shouldCreateDatabaseForDataSource(payload) {
		return c.createDatabaseBackedDataSource(ctx, profile, payload, accessToken, notionVersion)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/data_sources",
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source create response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || id == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "data source id is empty in Notion response")
	}
	return normalizeDataSourceObject(response), nil
}

// UpdateDataSource 更新一个 Notion data source。
func (c *Client) UpdateDataSource(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceID, appErr := requireIDField(input, "data_source_id")
	if appErr != nil {
		return nil, appErr
	}
	payload, appErr := buildDataSourceWritePayload(input, true)
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
		"/v1/data_sources/"+url.PathEscape(dataSourceID),
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source update response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || id == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "data source id is empty in Notion response")
	}
	return normalizeDataSourceObject(response), nil
}

// buildDataSourceWritePayload 优先支持 provider-native 的 body 透传。
func buildDataSourceWritePayload(input map[string]any, isUpdate bool) (map[string]any, *apperr.AppError) {
	if rawBody, exists := input["body"]; exists {
		body, ok := asMap(rawBody)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "body must be an object")
		}
		if len(body) == 0 {
			return nil, apperr.New("INVALID_INPUT", "body must not be empty")
		}
		return cloneMap(body), nil
	}

	payload := cloneMap(input)
	delete(payload, "body")
	delete(payload, "data_source_id")
	if len(payload) == 0 {
		if isUpdate {
			return nil, apperr.New("INVALID_INPUT", "body is required or at least one updatable field must be provided")
		}
		return nil, apperr.New("INVALID_INPUT", "body is required")
	}
	return payload, nil
}

func shouldCreateDatabaseForDataSource(payload map[string]any) bool {
	parent, ok := asMap(payload["parent"])
	if !ok || len(parent) == 0 {
		return false
	}

	parentType, _ := asString(parent["type"])
	parentType = strings.TrimSpace(parentType)
	switch parentType {
	case "page_id", "workspace":
		return true
	case "database_id", "data_source_id", "block_id":
		return false
	}

	if pageID, ok := asString(parent["page_id"]); ok && strings.TrimSpace(pageID) != "" {
		return true
	}
	if workspace, ok := asBool(parent["workspace"]); ok && workspace {
		return true
	}
	return false
}

func (c *Client) createDatabaseBackedDataSource(ctx context.Context, profile ExecutionProfile, payload map[string]any, accessToken string, notionVersion string) (map[string]any, *apperr.AppError) {
	databasePayload, appErr := buildCreateDatabasePayload(payload)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/databases",
		nil,
		databasePayload,
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

	rawDataSources, ok := asArray(response["data_sources"])
	if !ok || len(rawDataSources) == 0 {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "database create response does not include initial data_sources")
	}
	firstDataSource, ok := asMap(rawDataSources[0])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "initial data source object is invalid in database create response")
	}
	dataSourceID, ok := asString(firstDataSource["id"])
	if !ok || strings.TrimSpace(dataSourceID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "initial data source id is empty in database create response")
	}

	return c.GetDataSource(ctx, profile, map[string]any{
		"data_source_id": strings.TrimSpace(dataSourceID),
	})
}

func buildCreateDatabasePayload(payload map[string]any) (map[string]any, *apperr.AppError) {
	parent, ok := asMap(payload["parent"])
	if !ok || len(parent) == 0 {
		return nil, apperr.New("INVALID_INPUT", "parent is required")
	}

	normalizedParent, appErr := normalizeDatabaseParent(parent)
	if appErr != nil {
		return nil, appErr
	}

	title, exists := payload["title"]
	if !exists {
		return nil, apperr.New("INVALID_INPUT", "title is required when creating a page-backed data source")
	}
	titleItems, ok := asArray(title)
	if !ok || len(titleItems) == 0 {
		return nil, apperr.New("INVALID_INPUT", "title must be a non-empty rich_text array")
	}

	databasePayload := map[string]any{
		"parent": normalizedParent,
		"title":  title,
	}

	initialDataSource := map[string]any{}
	if properties, exists := payload["properties"]; exists {
		record, ok := asMap(properties)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "properties must be an object")
		}
		initialDataSource["properties"] = cloneMap(record)
	}
	if description, exists := payload["description"]; exists {
		initialDataSource["description"] = description
	}
	if len(initialDataSource) > 0 {
		databasePayload["initial_data_source"] = initialDataSource
	}

	return databasePayload, nil
}

func normalizeDatabaseParent(parent map[string]any) (map[string]any, *apperr.AppError) {
	parentType, _ := asString(parent["type"])
	parentType = strings.TrimSpace(parentType)
	switch parentType {
	case "page_id":
		parentID, ok := asString(parent["id"])
		if !ok || strings.TrimSpace(parentID) == "" {
			parentID, _ = asString(parent["page_id"])
		}
		if strings.TrimSpace(parentID) == "" {
			return nil, apperr.New("INVALID_INPUT", "parent.page_id is required")
		}
		return map[string]any{
			"type":    "page_id",
			"page_id": strings.TrimSpace(parentID),
		}, nil
	case "workspace":
		return map[string]any{
			"type":      "workspace",
			"workspace": true,
		}, nil
	case "":
		if pageID, ok := asString(parent["page_id"]); ok && strings.TrimSpace(pageID) != "" {
			return map[string]any{
				"type":    "page_id",
				"page_id": strings.TrimSpace(pageID),
			}, nil
		}
		if workspace, ok := asBool(parent["workspace"]); ok && workspace {
			return map[string]any{
				"type":      "workspace",
				"workspace": true,
			}, nil
		}
	}

	return nil, apperr.New("INVALID_INPUT", "database-backed data source creation currently requires a page or workspace parent")
}
