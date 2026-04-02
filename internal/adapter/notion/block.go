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

// GetBlock reads the details of a single block.
func (c *Client) GetBlock(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
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
		"/v1/blocks/"+url.PathEscape(blockID),
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	block, appErr := decodeBlockResponse(responseBody, "failed to decode Notion block response")
	if appErr != nil {
		return nil, appErr
	}
	return normalizeBlockData(block), nil
}

// ListBlockChildren reads the direct child blocks under the given block.
func (c *Client) ListBlockChildren(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
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
		"/v1/blocks/"+url.PathEscape(blockID)+"/children",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion block children response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		items = append(items, normalizeBlockData(item))
	}

	nextPageToken := ""
	if response.NextCursor != nil {
		nextPageToken = strings.TrimSpace(*response.NextCursor)
	}

	return map[string]any{
		"block_id":        blockID,
		"items":           items,
		"next_page_token": nextPageToken,
		"has_more":        response.HasMore,
	}, nil
}

// AppendBlockChildren appends child blocks to the end of a page or block.
func (c *Client) AppendBlockChildren(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
	if appErr != nil {
		return nil, appErr
	}

	children, appErr := buildBlockChildren(input["children"])
	if appErr != nil {
		return nil, appErr
	}
	if len(children) == 0 {
		return nil, apperr.New("INVALID_INPUT", "children must contain at least one block")
	}

	payload := map[string]any{
		"children": children,
	}
	if after, ok := asString(input["after"]); ok && strings.TrimSpace(after) != "" {
		payload["after"] = strings.TrimSpace(after)
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPatch,
		"/v1/blocks/"+url.PathEscape(blockID)+"/children",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionBlockChildrenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion block append response: %v", err))
	}

	childIDs := make([]string, 0, len(response.Results))
	for _, item := range response.Results {
		if childID, ok := asString(item["id"]); ok && strings.TrimSpace(childID) != "" {
			childIDs = append(childIDs, strings.TrimSpace(childID))
		}
	}

	return map[string]any{
		"block_id":       blockID,
		"appended_count": len(childIDs),
		"child_ids":      childIDs,
	}, nil
}

// UpdateBlock updates the content of the specified block.
func (c *Client) UpdateBlock(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
	if appErr != nil {
		return nil, appErr
	}

	payload, appErr := buildUpdateBlockPayload(input)
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
		"/v1/blocks/"+url.PathEscape(blockID),
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	block, appErr := decodeBlockResponse(responseBody, "failed to decode Notion block update response")
	if appErr != nil {
		return nil, appErr
	}
	return normalizeBlockData(block), nil
}

// DeleteBlock moves the specified block to the trash.
func (c *Client) DeleteBlock(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	blockID, appErr := requireIDField(input, "block_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodDelete,
		"/v1/blocks/"+url.PathEscape(blockID),
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	block, appErr := decodeBlockResponse(responseBody, "failed to decode Notion block delete response")
	if appErr != nil {
		return nil, appErr
	}

	data := normalizeBlockData(block)
	data["deleted"] = true
	return data, nil
}

// buildBlockChildren maps Clawrise's simplified block structure to Notion blocks.
func buildBlockChildren(raw any) ([]map[string]any, *apperr.AppError) {
	if raw == nil {
		return nil, nil
	}

	list, ok := asArray(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "children must be an array")
	}

	children := make([]map[string]any, 0, len(list))
	for _, item := range list {
		record, ok := asMap(item)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "each child block must be an object")
		}

		child, appErr := buildBlock(record)
		if appErr != nil {
			return nil, appErr
		}
		children = append(children, child)
	}
	return children, nil
}

func buildUpdateBlockPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	blockInput := map[string]any{}
	if rawBlock, exists := input["block"]; exists {
		record, ok := asMap(rawBlock)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "block must be an object")
		}
		blockInput = cloneMap(record)
	} else {
		blockInput = cloneMap(input)
		delete(blockInput, "block_id")
		delete(blockInput, "in_trash")
	}

	payload, appErr := buildBlock(blockInput)
	if appErr != nil {
		return nil, appErr
	}
	delete(payload, "object")

	if inTrash, ok := asBool(input["in_trash"]); ok {
		payload["in_trash"] = inTrash
	}

	return payload, nil
}

func buildBlock(input map[string]any) (map[string]any, *apperr.AppError) {
	blockType, ok := asString(input["type"])
	if !ok || strings.TrimSpace(blockType) == "" {
		return nil, apperr.New("INVALID_INPUT", "block.type is required")
	}
	blockType = strings.TrimSpace(blockType)

	richText, appErr := buildRichText(input["text"], input["rich_text"])
	if appErr != nil {
		return nil, appErr
	}
	children, appErr := buildBlockChildren(input["children"])
	if appErr != nil {
		return nil, appErr
	}

	payload := map[string]any{
		"object": "block",
		"type":   blockType,
	}

	switch blockType {
	case "paragraph", "quote", "bulleted_list_item", "numbered_list_item":
		payload[blockType] = buildTextualBlockBody(richText, children, input["color"])
	case "toggle":
		payload[blockType] = buildTextualBlockBody(richText, children, input["color"])
	case "callout":
		body := buildTextualBlockBody(richText, children, input["color"])
		icon, appErr := buildCalloutIcon(input)
		if appErr != nil {
			return nil, appErr
		}
		if icon != nil {
			body["icon"] = icon
		}
		payload[blockType] = body
	case "heading_1", "heading_2", "heading_3":
		body := buildTextualBlockBody(richText, children, input["color"])
		if toggleable, ok := asBool(input["is_toggleable"]); ok {
			body["is_toggleable"] = toggleable
		} else if len(children) > 0 {
			// Enable toggleable automatically when nested children exist to satisfy Notion validation.
			body["is_toggleable"] = true
		}
		payload[blockType] = body
	case "to_do":
		body := buildTextualBlockBody(richText, children, input["color"])
		if checked, ok := asBool(input["checked"]); ok {
			body["checked"] = checked
		}
		payload[blockType] = body
	case "code":
		if len(children) > 0 {
			return nil, apperr.New("INVALID_INPUT", "code blocks do not support nested children")
		}
		body := map[string]any{
			"rich_text": richText,
		}
		if language, ok := asString(input["language"]); ok && strings.TrimSpace(language) != "" {
			body["language"] = strings.TrimSpace(language)
		} else {
			body["language"] = "plain text"
		}
		payload[blockType] = body
	case "divider":
		if len(children) > 0 {
			return nil, apperr.New("INVALID_INPUT", "divider blocks do not support nested children")
		}
		if len(richText) > 0 {
			return nil, apperr.New("INVALID_INPUT", "divider blocks do not support text")
		}
		payload[blockType] = map[string]any{}
	case "image", "file":
		if len(children) > 0 {
			return nil, apperr.New("INVALID_INPUT", blockType+" blocks do not support nested children")
		}
		body, appErr := buildExternalFileBlockBody(input)
		if appErr != nil {
			return nil, appErr
		}
		payload[blockType] = body
	case "table":
		body, appErr := buildTableBlockBody(input, children)
		if appErr != nil {
			return nil, appErr
		}
		payload[blockType] = body
	case "table_row":
		if len(children) > 0 {
			return nil, apperr.New("INVALID_INPUT", "table_row blocks do not support nested children")
		}
		body, appErr := buildTableRowBlockBody(input)
		if appErr != nil {
			return nil, appErr
		}
		payload[blockType] = body
	default:
		return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("unsupported Notion block type %s", blockType))
	}

	return payload, nil
}

func buildTextualBlockBody(richText []map[string]any, children []map[string]any, colorInput any) map[string]any {
	body := map[string]any{
		"rich_text": richText,
	}
	if len(children) > 0 {
		body["children"] = children
	}
	if color, ok := asString(colorInput); ok && strings.TrimSpace(color) != "" {
		body["color"] = strings.TrimSpace(color)
	}
	return body
}

func buildCalloutIcon(input map[string]any) (map[string]any, *apperr.AppError) {
	if rawIcon, exists := input["icon"]; exists {
		return normalizeNotionFileObject(rawIcon, true)
	}
	if emoji, ok := asString(input["emoji"]); ok && strings.TrimSpace(emoji) != "" {
		return map[string]any{
			"type":  "emoji",
			"emoji": strings.TrimSpace(emoji),
		}, nil
	}
	return nil, nil
}

func buildExternalFileBlockBody(input map[string]any) (map[string]any, *apperr.AppError) {
	body := map[string]any{}
	if urlValue, ok := asString(input["url"]); ok && strings.TrimSpace(urlValue) != "" {
		body["type"] = "external"
		body["external"] = map[string]any{
			"url": strings.TrimSpace(urlValue),
		}
	} else if external, ok := asMap(input["external"]); ok && len(external) > 0 {
		body["type"] = "external"
		body["external"] = cloneMap(external)
	} else {
		return nil, apperr.New("INVALID_INPUT", "url is required for external file blocks")
	}

	caption, appErr := buildRichText(input["caption"], input["caption_rich_text"])
	if appErr != nil {
		return nil, appErr
	}
	if len(caption) > 0 {
		body["caption"] = caption
	}
	return body, nil
}

func buildTableBlockBody(input map[string]any, children []map[string]any) (map[string]any, *apperr.AppError) {
	if rows, exists := input["rows"]; exists {
		var appErr *apperr.AppError
		children, appErr = buildBlockChildren(rows)
		if appErr != nil {
			return nil, appErr
		}
	}
	for _, child := range children {
		if blockType, ok := asString(child["type"]); !ok || strings.TrimSpace(blockType) != "table_row" {
			return nil, apperr.New("INVALID_INPUT", "table children must be table_row blocks")
		}
	}

	tableWidth := 0
	if value, ok := asInt(input["table_width"]); ok && value > 0 {
		tableWidth = value
	} else {
		tableWidth = inferTableWidth(children)
	}
	if tableWidth <= 0 {
		return nil, apperr.New("INVALID_INPUT", "table_width is required")
	}

	body := map[string]any{
		"table_width": tableWidth,
	}
	if hasColumnHeader, ok := asBool(input["has_column_header"]); ok {
		body["has_column_header"] = hasColumnHeader
	}
	if hasRowHeader, ok := asBool(input["has_row_header"]); ok {
		body["has_row_header"] = hasRowHeader
	}
	if len(children) > 0 {
		body["children"] = children
	}
	return body, nil
}

func buildTableRowBlockBody(input map[string]any) (map[string]any, *apperr.AppError) {
	rawCells, ok := asArray(input["cells"])
	if !ok || len(rawCells) == 0 {
		return nil, apperr.New("INVALID_INPUT", "cells is required for table_row blocks")
	}

	cells := make([][]map[string]any, 0, len(rawCells))
	for _, rawCell := range rawCells {
		switch value := rawCell.(type) {
		case string:
			cells = append(cells, buildPlainTextRichText(value))
		case []any:
			richText := make([]map[string]any, 0, len(value))
			for _, rawRichText := range value {
				record, ok := asMap(rawRichText)
				if !ok {
					return nil, apperr.New("INVALID_INPUT", "table_row cell rich text items must be objects")
				}
				richText = append(richText, cloneMap(record))
			}
			cells = append(cells, richText)
		default:
			return nil, apperr.New("INVALID_INPUT", "each table_row cell must be a string or a rich_text array")
		}
	}
	return map[string]any{
		"cells": cells,
	}, nil
}

func inferTableWidth(children []map[string]any) int {
	if len(children) == 0 {
		return 0
	}
	body, ok := asMap(children[0]["table_row"])
	if !ok {
		return 0
	}

	// 这里同时兼容两种中间形态：
	// 1. 运行时经过 JSON 反序列化后的 []any
	// 2. 单测 / 本地构造阶段仍保持强类型的二维切片
	switch cells := body["cells"].(type) {
	case []any:
		return len(cells)
	case [][]map[string]any:
		return len(cells)
	case [][]any:
		return len(cells)
	default:
		return 0
	}
}

func buildRichText(textInput any, richTextInput any) ([]map[string]any, *apperr.AppError) {
	if richTextInput != nil {
		list, ok := asArray(richTextInput)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "rich_text must be an array")
		}
		richText := make([]map[string]any, 0, len(list))
		for _, item := range list {
			record, ok := asMap(item)
			if !ok {
				return nil, apperr.New("INVALID_INPUT", "each rich_text item must be an object")
			}
			richText = append(richText, cloneMap(record))
		}
		return richText, nil
	}

	if text, ok := asString(textInput); ok {
		return buildPlainTextRichText(text), nil
	}
	if textInput == nil {
		return []map[string]any{}, nil
	}
	return nil, apperr.New("INVALID_INPUT", "text must be a string")
}

func decodeBlockResponse(responseBody []byte, decodeErrorMessage string) (map[string]any, *apperr.AppError) {
	var block map[string]any
	if err := json.Unmarshal(responseBody, &block); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("%s: %v", decodeErrorMessage, err))
	}

	blockID, ok := asString(block["id"])
	if !ok || strings.TrimSpace(blockID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "block id is empty in Notion response")
	}
	return block, nil
}

func normalizeBlockData(block map[string]any) map[string]any {
	blockID, _ := asString(block["id"])
	blockType, _ := asString(block["type"])
	hasChildren, _ := asBool(block["has_children"])
	archived, _ := asBool(block["archived"])
	inTrash, _ := asBool(block["in_trash"])

	result := map[string]any{
		"block_id":     strings.TrimSpace(blockID),
		"type":         strings.TrimSpace(blockType),
		"has_children": hasChildren,
		"archived":     archived || inTrash,
		"in_trash":     inTrash,
		"plain_text":   extractBlockPlainText(block),
		"raw":          cloneMap(block),
	}

	if parent, ok := asMap(block["parent"]); ok && len(parent) > 0 {
		result["parent"] = cloneMap(parent)
	}
	if checked, ok := extractTodoChecked(block); ok {
		result["checked"] = checked
	}
	if language := extractCodeLanguage(block); language != "" {
		result["language"] = language
	}

	return result
}

func extractBlockPlainText(block map[string]any) string {
	blockType, ok := asString(block["type"])
	if !ok || strings.TrimSpace(blockType) == "" {
		return ""
	}

	if blockType == "table_row" {
		return extractTableRowPlainText(block)
	}

	body, ok := asMap(block[blockType])
	if !ok {
		return ""
	}

	if richText, ok := asArray(body["rich_text"]); ok {
		return extractRichTextPlainText(richText)
	}
	if title, ok := asString(body["title"]); ok {
		return strings.TrimSpace(title)
	}
	if caption, ok := asArray(body["caption"]); ok {
		return extractRichTextPlainText(caption)
	}
	return ""
}

func extractTableRowPlainText(block map[string]any) string {
	body, ok := asMap(block["table_row"])
	if !ok {
		return ""
	}
	cells, ok := asArray(body["cells"])
	if !ok {
		return ""
	}

	parts := make([]string, 0, len(cells))
	for _, rawCell := range cells {
		items, ok := asArray(rawCell)
		if !ok {
			continue
		}
		parts = append(parts, extractRichTextPlainText(items))
	}
	return strings.Join(parts, " | ")
}

func extractRichTextPlainText(items []any) string {
	var builder strings.Builder
	for _, item := range items {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		if plainText, ok := asString(record["plain_text"]); ok {
			builder.WriteString(plainText)
			continue
		}
		text, ok := asMap(record["text"])
		if !ok {
			continue
		}
		content, ok := asString(text["content"])
		if ok {
			builder.WriteString(content)
		}
	}
	return builder.String()
}

func extractTodoChecked(block map[string]any) (bool, bool) {
	body, ok := asMap(block["to_do"])
	if !ok {
		return false, false
	}
	checked, ok := asBool(body["checked"])
	return checked, ok
}

func extractCodeLanguage(block map[string]any) string {
	body, ok := asMap(block["code"])
	if !ok {
		return ""
	}
	language, ok := asString(body["language"])
	if !ok {
		return ""
	}
	return strings.TrimSpace(language)
}
