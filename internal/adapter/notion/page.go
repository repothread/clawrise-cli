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

// CreatePage 创建页面。
func (c *Client) CreatePage(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildCreatePagePayload(profile, input)
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
		"/v1/pages",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionPage
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion page response: %v", err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "page id is empty in Notion response")
	}

	return mapPageData(response), nil
}

// GetPage 读取页面详情。
func (c *Client) GetPage(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
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
		"/v1/pages/"+url.PathEscape(pageID),
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionPage
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion page response: %v", err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "page id is empty in Notion response")
	}

	return mapPageData(response), nil
}

// buildCreatePagePayload 构造创建页面的请求体。
func buildCreatePagePayload(profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	title, ok := asString(input["title"])
	if !ok || strings.TrimSpace(title) == "" {
		return nil, apperr.New("INVALID_INPUT", "title is required")
	}

	parent, parentType, appErr := buildPageParent(profile, input["parent"])
	if appErr != nil {
		return nil, appErr
	}

	properties, appErr := buildCreatePageProperties(parentType, title, input["title_property"], input["properties"])
	if appErr != nil {
		return nil, appErr
	}

	children, appErr := buildBlockChildren(input["children"])
	if appErr != nil {
		return nil, appErr
	}

	payload := map[string]any{
		"parent":     parent,
		"properties": properties,
	}
	if len(children) > 0 {
		payload["children"] = children
	}
	return payload, nil
}

func buildPageParent(profile config.Profile, raw any) (map[string]any, string, *apperr.AppError) {
	// 公开集成允许在 workspace 级别建顶层私有页，这里在未传 parent 时做兼容。
	if raw == nil && profile.Grant.Type == "oauth_refreshable" {
		return map[string]any{
			"workspace": true,
		}, "workspace", nil
	}
	if raw == nil {
		return nil, "", apperr.New("INVALID_INPUT", "parent is required")
	}

	parent, ok := asMap(raw)
	if !ok {
		return nil, "", apperr.New("INVALID_INPUT", "parent must be an object")
	}

	parentType, ok := asString(parent["type"])
	if !ok || strings.TrimSpace(parentType) == "" {
		return nil, "", apperr.New("INVALID_INPUT", "parent.type is required")
	}
	parentType = strings.TrimSpace(parentType)

	switch parentType {
	case "page_id", "block_id", "database_id", "data_source_id":
		parentID, ok := asString(parent["id"])
		if !ok || strings.TrimSpace(parentID) == "" {
			if directID, exists := asString(parent[parentType]); exists && strings.TrimSpace(directID) != "" {
				parentID = directID
			} else {
				return nil, "", apperr.New("INVALID_INPUT", "parent.id is required")
			}
		}
		requestKey := parentType
		if parentType == "data_source_id" {
			requestKey = "data_source"
		}
		return map[string]any{
			requestKey: strings.TrimSpace(parentID),
		}, parentType, nil
	case "workspace":
		if profile.Grant.Type != "oauth_refreshable" {
			return nil, "", apperr.New("INVALID_INPUT", "workspace-level page creation requires a public Notion integration profile")
		}
		return map[string]any{
			"workspace": true,
		}, "workspace", nil
	default:
		return nil, "", apperr.New("INVALID_INPUT", "parent.type must be one of page_id, block_id, data_source_id, database_id, or workspace")
	}
}

func buildCreatePageProperties(parentType, title string, titlePropertyInput any, propertiesInput any) (map[string]any, *apperr.AppError) {
	properties := map[string]any{}
	if propertiesInput != nil {
		record, ok := asMap(propertiesInput)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "properties must be an object")
		}
		properties = cloneMap(record)
	}

	switch parentType {
	case "page_id", "block_id", "workspace":
		for key := range properties {
			if key != "title" {
				return nil, apperr.New("INVALID_INPUT", "only the title property is supported when creating a child page under another page or workspace")
			}
		}
		properties["title"] = map[string]any{
			"title": buildPlainTextRichText(title),
		}
		return properties, nil
	case "data_source_id", "database_id":
		titlePropertyName := ""
		if value, ok := asString(titlePropertyInput); ok {
			titlePropertyName = strings.TrimSpace(value)
		}
		if titlePropertyName != "" {
			properties[titlePropertyName] = map[string]any{
				"title": buildPlainTextRichText(title),
			}
		}
		if !containsTitleProperty(properties) {
			return nil, apperr.New("INVALID_INPUT", "properties must include a title property when parent.type is data_source_id or database_id; set title_property to map the required title field")
		}
		return properties, nil
	default:
		return nil, apperr.New("INVALID_INPUT", fmt.Sprintf("unsupported parent.type %s", parentType))
	}
}

func containsTitleProperty(properties map[string]any) bool {
	for _, value := range properties {
		record, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := record["title"]; exists {
			return true
		}
	}
	return false
}

func mapPageData(page notionPage) map[string]any {
	return map[string]any{
		"page_id":    page.ID,
		"title":      extractPageTitle(page.Properties),
		"parent":     normalizeParent(page.Parent),
		"url":        page.URL,
		"archived":   page.Archived || page.InTrash,
		"properties": page.Properties,
	}
}

func normalizeParent(parent map[string]any) map[string]any {
	if len(parent) == 0 {
		return nil
	}
	return cloneMap(parent)
}

func extractPageTitle(properties map[string]any) string {
	for _, value := range properties {
		record, ok := value.(map[string]any)
		if !ok {
			continue
		}
		titleItems, ok := asArray(record["title"])
		if !ok {
			continue
		}
		var builder strings.Builder
		for _, item := range titleItems {
			segment, ok := asMap(item)
			if !ok {
				continue
			}
			plainText, ok := asString(segment["plain_text"])
			if ok {
				builder.WriteString(plainText)
				continue
			}
			text, ok := asMap(segment["text"])
			if !ok {
				continue
			}
			content, ok := asString(text["content"])
			if ok {
				builder.WriteString(content)
			}
		}
		if builder.Len() > 0 {
			return builder.String()
		}
	}
	return ""
}

func requireIDField(input map[string]any, field string) (string, *apperr.AppError) {
	value, ok := asString(input[field])
	if !ok || strings.TrimSpace(value) == "" {
		return "", apperr.New("INVALID_INPUT", field+" is required")
	}
	return strings.TrimSpace(value), nil
}
