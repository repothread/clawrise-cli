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

// AppendBlockChildren 向页面或块末尾追加子块。
func (c *Client) AppendBlockChildren(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
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
		if strings.TrimSpace(item.ID) != "" {
			childIDs = append(childIDs, item.ID)
		}
	}

	return map[string]any{
		"block_id":       blockID,
		"appended_count": len(childIDs),
		"child_ids":      childIDs,
	}, nil
}

// buildBlockChildren 将 Clawrise 的简化块结构映射为 Notion 块。
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
	case "heading_1", "heading_2", "heading_3":
		body := buildTextualBlockBody(richText, children, input["color"])
		if toggleable, ok := asBool(input["is_toggleable"]); ok {
			body["is_toggleable"] = toggleable
		} else if len(children) > 0 {
			// 有子块时自动开启 toggleable，避免 Notion 拒绝嵌套标题。
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
