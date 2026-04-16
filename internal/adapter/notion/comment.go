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

// GetComment reads a single Notion comment object.
func (c *Client) GetComment(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	commentID, appErr := requireIDField(input, "comment_id")
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
		"/v1/comments/"+url.PathEscape(commentID),
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion comment response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "comment id is empty in Notion response")
	}
	return normalizeComment(response), nil
}

// ListComments lists open comments under a page or block.
func (c *Client) ListComments(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, ok := asString(input["block_id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("INVALID_INPUT", "block_id is required")
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("block_id", strings.TrimSpace(blockID))
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("start_cursor", strings.TrimSpace(pageToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/comments",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionCommentListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion comments response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		items = append(items, normalizeComment(item))
	}

	nextPageToken := ""
	if response.NextCursor != nil {
		nextPageToken = strings.TrimSpace(*response.NextCursor)
	}

	return map[string]any{
		"block_id":        strings.TrimSpace(blockID),
		"items":           items,
		"next_page_token": nextPageToken,
		"has_more":        response.HasMore,
	}, nil
}

// CreateComment creates a page, block, or discussion comment.
func (c *Client) CreateComment(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildCreateCommentPayload(input)
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
		"/v1/comments",
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion comment create response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || strings.TrimSpace(id) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "comment id is empty in Notion response")
	}
	return normalizeComment(response), nil
}

func buildCreateCommentPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	if appErr := validateTopLevelInputFields("notion.comment.create", input, notionCommentCreateSpec().Input, nil); appErr != nil {
		return nil, appErr
	}

	richText, appErr := buildRichText(input["text"], input["rich_text"])
	if appErr != nil {
		return nil, appErr
	}
	if len(richText) == 0 {
		return nil, apperr.New("INVALID_INPUT", "text or rich_text is required")
	}

	payload := map[string]any{
		"rich_text": richText,
	}

	parentCount := 0
	if pageID, ok := asString(input["page_id"]); ok && strings.TrimSpace(pageID) != "" {
		payload["parent"] = map[string]any{
			"page_id": strings.TrimSpace(pageID),
		}
		parentCount++
	}
	if blockID, ok := asString(input["block_id"]); ok && strings.TrimSpace(blockID) != "" {
		payload["parent"] = map[string]any{
			"block_id": strings.TrimSpace(blockID),
		}
		parentCount++
	}
	if discussionID, ok := asString(input["discussion_id"]); ok && strings.TrimSpace(discussionID) != "" {
		payload["discussion_id"] = strings.TrimSpace(discussionID)
		parentCount++
	}
	if parentCount != 1 {
		return nil, apperr.New("INVALID_INPUT", "exactly one of page_id, block_id, or discussion_id is required")
	}

	if attachments, exists := input["attachments"]; exists {
		list, ok := asArray(attachments)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "attachments must be an array")
		}
		cloned := make([]map[string]any, 0, len(list))
		for _, item := range list {
			record, ok := asMap(item)
			if !ok {
				return nil, apperr.New("INVALID_INPUT", "each attachment must be an object")
			}
			cloned = append(cloned, cloneMap(record))
		}
		payload["attachments"] = cloned
	}
	if displayName, exists := input["display_name"]; exists {
		record, ok := asMap(displayName)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "display_name must be an object")
		}
		payload["display_name"] = cloneMap(record)
	}

	return payload, nil
}

func normalizeComment(item map[string]any) map[string]any {
	result := map[string]any{
		"comment_id": extractFirstString(item, "id"),
		"raw":        cloneMap(item),
	}
	if parent, ok := asMap(item["parent"]); ok && len(parent) > 0 {
		result["parent"] = cloneMap(parent)
	}
	if discussionID := extractFirstString(item, "discussion_id"); discussionID != "" {
		result["discussion_id"] = discussionID
	}
	if richText, ok := asArray(item["rich_text"]); ok {
		result["plain_text"] = extractRichTextPlainText(richText)
		result["rich_text"] = richText
	}
	if createdTime := extractFirstString(item, "created_time"); createdTime != "" {
		result["created_time"] = createdTime
	}
	if lastEditedTime := extractFirstString(item, "last_edited_time"); lastEditedTime != "" {
		result["last_edited_time"] = lastEditedTime
	}
	if attachments, ok := asArray(item["attachments"]); ok {
		result["attachments"] = attachments
	}
	return result
}
