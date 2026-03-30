package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// Search looks up pages and data sources visible to the current integration.
func (c *Client) Search(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildSearchPayload(input)
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
		"/v1/search",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionSearchResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion search response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		items = append(items, normalizeSearchResult(item))
	}

	nextPageToken := ""
	if response.NextCursor != nil {
		nextPageToken = strings.TrimSpace(*response.NextCursor)
	}

	return map[string]any{
		"items":           items,
		"next_page_token": nextPageToken,
		"has_more":        response.HasMore,
	}, nil
}

func buildSearchPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	payload := map[string]any{}

	if query, ok := asString(input["query"]); ok && strings.TrimSpace(query) != "" {
		payload["query"] = strings.TrimSpace(query)
	}
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
	if sort, exists := input["sort"]; exists {
		record, ok := asMap(sort)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "sort must be an object")
		}
		payload["sort"] = cloneMap(record)
	}

	return payload, nil
}

func normalizeSearchResult(item map[string]any) map[string]any {
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
		}
	case "data_source", "database":
		if titleItems, ok := asArray(item["title"]); ok {
			result["title"] = extractRichTextPlainText(titleItems)
		}
	}

	return result
}
