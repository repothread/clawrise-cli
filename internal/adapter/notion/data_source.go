package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// GetDataSource reads data source metadata and schema.
func (c *Client) GetDataSource(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceID, appErr := requireIDField(input, "data_source_id")
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
		"/v1/data_sources/"+url.PathEscape(dataSourceID),
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "data source id is empty in Notion response")
	}
	return normalizeDataSourceObject(response), nil
}

// QueryDataSource queries pages or nested data sources under a data source.
func (c *Client) QueryDataSource(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceID, appErr := requireIDField(input, "data_source_id")
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if filterProperties, exists := input["filter_properties"]; exists {
		list, ok := asArray(filterProperties)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "filter_properties must be an array")
		}
		for _, item := range list {
			property, ok := asString(item)
			if !ok || strings.TrimSpace(property) == "" {
				return nil, apperr.New("INVALID_INPUT", "each filter_properties item must be a non-empty string")
			}
			query.Add("filter_properties[]", strings.TrimSpace(property))
		}
	}

	payload, appErr := buildQueryDataSourcePayload(input)
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
		"/v1/data_sources/"+url.PathEscape(dataSourceID)+"/query",
		query,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionQueryDataSourceResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source query response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		items = append(items, normalizeDataSourceQueryResult(item))
	}

	nextPageToken := ""
	if response.NextCursor != nil {
		nextPageToken = strings.TrimSpace(*response.NextCursor)
	}

	result := map[string]any{
		"data_source_id":  dataSourceID,
		"items":           items,
		"next_page_token": nextPageToken,
		"has_more":        response.HasMore,
	}
	if strings.TrimSpace(response.Type) != "" {
		result["type"] = strings.TrimSpace(response.Type)
	}
	return result, nil
}

func normalizeDataSourceObject(item map[string]any) map[string]any {
	result := map[string]any{
		"data_source_id": extractFirstString(item, "id"),
		"raw":            cloneMap(item),
	}
	if titleItems, ok := asArray(item["title"]); ok {
		result["title"] = extractRichTextPlainText(titleItems)
	}
	if properties, ok := asMap(item["properties"]); ok {
		result["properties"] = cloneMap(properties)
	}
	if parent, ok := asMap(item["parent"]); ok && len(parent) > 0 {
		result["parent"] = cloneMap(parent)
	}
	if urlValue := extractFirstString(item, "url"); urlValue != "" {
		result["url"] = urlValue
	}
	return result
}

func extractFirstString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := asString(record[key]); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildQueryDataSourcePayload(input map[string]any) (map[string]any, *apperr.AppError) {
	payload := map[string]any{}

	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		payload["page_size"] = pageSize
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		payload["start_cursor"] = strings.TrimSpace(pageToken)
	}
	if filter, exists := input["filter"]; exists {
		record, ok := asMap(filter)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "filter must be an object")
		}
		payload["filter"] = cloneMap(record)
	}
	if sorts, exists := input["sorts"]; exists {
		list, ok := asArray(sorts)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "sorts must be an array")
		}
		cloned := make([]map[string]any, 0, len(list))
		for _, item := range list {
			record, ok := asMap(item)
			if !ok {
				return nil, apperr.New("INVALID_INPUT", "each sorts item must be an object")
			}
			cloned = append(cloned, cloneMap(record))
		}
		payload["sorts"] = cloned
	}
	return payload, nil
}

func normalizeDataSourceQueryResult(item map[string]any) map[string]any {
	objectType, _ := asString(item["object"])
	result := map[string]any{
		"object": strings.TrimSpace(objectType),
		"raw":    cloneMap(item),
	}
	if id, ok := asString(item["id"]); ok {
		result["id"] = strings.TrimSpace(id)
	}
	if url, ok := asString(item["url"]); ok && strings.TrimSpace(url) != "" {
		result["url"] = strings.TrimSpace(url)
	}
	if parent, ok := asMap(item["parent"]); ok && len(parent) > 0 {
		result["parent"] = cloneMap(parent)
	}
	if archived, ok := asBool(item["archived"]); ok {
		result["archived"] = archived
	}
	if inTrash, ok := asBool(item["in_trash"]); ok {
		result["in_trash"] = inTrash
		if archived, exists := result["archived"]; !exists {
			result["archived"] = inTrash
		} else if archivedBool, ok := archived.(bool); ok {
			result["archived"] = archivedBool || inTrash
		}
	}

	switch objectType {
	case "page":
		if properties, ok := asMap(item["properties"]); ok {
			result["title"] = extractPageTitle(properties)
			result["properties"] = cloneMap(properties)
		}
	case "data_source", "database":
		if titleItems, ok := asArray(item["title"]); ok {
			result["title"] = extractRichTextPlainText(titleItems)
		}
	}

	return result
}
