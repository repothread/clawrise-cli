package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// CreateDocument creates an empty Feishu document shell.
func (c *Client) CreateDocument(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	title, ok := asString(input["title"])
	if !ok || strings.TrimSpace(title) == "" {
		return nil, apperr.New("INVALID_INPUT", "title is required")
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	payload := map[string]any{
		"title": strings.TrimSpace(title),
	}
	if folderToken, ok := asString(input["folder_token"]); ok && strings.TrimSpace(folderToken) != "" {
		payload["folder_token"] = strings.TrimSpace(folderToken)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/docx/v1/documents",
		nil,
		payload,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode document create response")
	if appErr != nil {
		return nil, appErr
	}

	document, ok := asMap(data["document"])
	if !ok {
		document = data
	}
	documentID := extractFirstNonEmptyString(document, "document_id", "obj_token")
	if documentID == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "document_id is empty in Feishu response")
	}

	result := map[string]any{
		"document_id": documentID,
		"title":       extractFirstNonEmptyString(document, "title"),
		"raw":         cloneFeishuMap(document),
	}
	if revisionID, ok := asInt(document["revision_id"]); ok {
		result["revision_id"] = revisionID
	}
	if urlValue := extractFirstNonEmptyString(document, "url", "document_url"); urlValue != "" {
		result["url"] = urlValue
	}
	return result, nil
}

// EditDocument applies task-oriented edit modes on a Feishu document.
func (c *Client) EditDocument(ctx context.Context, profile config.Profile, input map[string]any, clientToken string) (map[string]any, *apperr.AppError) {
	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	documentID = strings.TrimSpace(documentID)

	blocks, appErr := buildDocumentEditBlocks(input)
	if appErr != nil {
		return nil, appErr
	}

	mode := "append"
	if rawMode, ok := asString(input["mode"]); ok && strings.TrimSpace(rawMode) != "" {
		mode = strings.TrimSpace(rawMode)
	}

	switch mode {
	case "append":
		appendInput := cloneFeishuInputMap(input)
		appendInput["document_id"] = documentID
		appendInput["blocks"] = blocks
		return c.AppendDocumentBlocks(ctx, profile, appendInput, clientToken)
	case "replace_all":
		deletedCount, appErr := c.deleteAllDocumentChildren(ctx, profile, documentID, clientToken)
		if appErr != nil {
			return nil, appErr
		}

		appendInput := cloneFeishuInputMap(input)
		appendInput["document_id"] = documentID
		appendInput["blocks"] = blocks
		appended, appErr := c.AppendDocumentBlocks(ctx, profile, appendInput, clientToken)
		if appErr != nil {
			return nil, appErr
		}

		return map[string]any{
			"document_id":    documentID,
			"mode":           "replace_all",
			"deleted_count":  deletedCount,
			"appended_count": appended["appended_count"],
			"children":       appended["children"],
		}, nil
	default:
		return nil, apperr.New("INVALID_INPUT", "mode must be append or replace_all")
	}
}

func buildDocumentEditBlocks(input map[string]any) ([]any, *apperr.AppError) {
	if blocks, ok := asArray(input["blocks"]); ok && len(blocks) > 0 {
		return blocks, nil
	}
	if text, ok := asString(input["text"]); ok && strings.TrimSpace(text) != "" {
		return []any{
			map[string]any{
				"type": "paragraph",
				"text": strings.TrimSpace(text),
			},
		}, nil
	}
	return nil, apperr.New("INVALID_INPUT", "blocks or text is required")
}

func (c *Client) deleteAllDocumentChildren(ctx context.Context, profile config.Profile, documentID string, clientToken string) (int, *apperr.AppError) {
	items, appErr := c.listDirectDocumentChildren(ctx, profile, documentID)
	if appErr != nil {
		return 0, appErr
	}
	if len(items) == 0 {
		return 0, nil
	}

	_, appErr = c.BatchDeleteDocumentBlockChildren(ctx, profile, map[string]any{
		"document_id": documentID,
		"block_id":    documentID,
		"start_index": 0,
		"end_index":   len(items),
	}, clientToken)
	if appErr != nil {
		return 0, appErr
	}
	return len(items), nil
}

func (c *Client) listDirectDocumentChildren(ctx context.Context, profile config.Profile, documentID string) ([]map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("page_size", "500")

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/docx/v1/documents/"+url.PathEscape(documentID)+"/blocks/"+url.PathEscape(documentID)+"/children",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response documentBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "failed to decode document root children response")
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		items = append(items, normalizeDocxBlock(item))
	}
	return items, nil
}
