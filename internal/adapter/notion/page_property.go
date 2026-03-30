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

// GetPagePropertyItem 读取页面的单个属性项，并保留 Notion 原生响应结构。
func (c *Client) GetPagePropertyItem(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}
	propertyID, appErr := requireIDField(input, "property_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("start_cursor", strings.TrimSpace(pageToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/pages/"+url.PathEscape(pageID)+"/properties/"+url.PathEscape(propertyID),
		query,
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion page property item response: %v", err))
	}

	objectType, _ := asString(response["object"])
	result := map[string]any{
		"page_id":     pageID,
		"property_id": propertyID,
		"object":      strings.TrimSpace(objectType),
		"raw":         cloneMap(response),
	}

	// 属性项可能直接返回单个对象，也可能返回带分页的 property_item 列表。
	if results, ok := asArray(response["results"]); ok {
		items := make([]map[string]any, 0, len(results))
		for _, item := range results {
			record, ok := asMap(item)
			if !ok {
				continue
			}
			items = append(items, cloneMap(record))
		}
		result["items"] = items
		if nextCursor, ok := asString(response["next_cursor"]); ok && strings.TrimSpace(nextCursor) != "" {
			result["next_page_token"] = strings.TrimSpace(nextCursor)
		} else {
			result["next_page_token"] = ""
		}
		if hasMore, ok := asBool(response["has_more"]); ok {
			result["has_more"] = hasMore
		}
		return result, nil
	}

	result["property_item"] = cloneMap(response)
	return result, nil
}
