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

// CreatePage creates a page.
func (c *Client) CreatePage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
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

// GetPage reads page details.
func (c *Client) GetPage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
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

// UpdatePage updates page properties or archive state.
func (c *Client) UpdatePage(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	payload, appErr := buildUpdatePagePayload(input)
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
		"/v1/pages/"+url.PathEscape(pageID),
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
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion page update response: %v", err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "page id is empty in Notion response")
	}

	return mapPageData(response), nil
}

// GetPageMarkdown reads page content or unknown subtrees in enhanced markdown form.
func (c *Client) GetPageMarkdown(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if includeTranscript, ok := asBool(input["include_transcript"]); ok {
		query.Set("include_transcript", fmt.Sprintf("%t", includeTranscript))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/v1/pages/"+url.PathEscape(pageID)+"/markdown",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	response, appErr := decodePageMarkdownResponse(responseBody, "failed to decode Notion page markdown response")
	if appErr != nil {
		return nil, appErr
	}
	return mapPageMarkdownData(response), nil
}

// UpdatePageMarkdown applies incremental or full-page updates with enhanced markdown commands.
func (c *Client) UpdatePageMarkdown(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	pageID, appErr := requireIDField(input, "page_id")
	if appErr != nil {
		return nil, appErr
	}

	payload, appErr := buildUpdatePageMarkdownPayload(input)
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
		"/v1/pages/"+url.PathEscape(pageID)+"/markdown",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	response, appErr := decodePageMarkdownResponse(responseBody, "failed to decode Notion page markdown update response")
	if appErr != nil {
		return nil, appErr
	}
	return mapPageMarkdownData(response), nil
}

// buildCreatePagePayload builds the request payload used to create a page.
func buildCreatePagePayload(profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
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

func buildUpdatePagePayload(input map[string]any) (map[string]any, *apperr.AppError) {
	payload := map[string]any{}
	properties := map[string]any{}

	if rawProperties, exists := input["properties"]; exists {
		record, ok := asMap(rawProperties)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "properties must be an object")
		}
		properties = cloneMap(record)
	}
	if title, ok := asString(input["title"]); ok && strings.TrimSpace(title) != "" {
		titleProperty := "title"
		if value, ok := asString(input["title_property"]); ok && strings.TrimSpace(value) != "" {
			titleProperty = strings.TrimSpace(value)
		}
		properties[titleProperty] = map[string]any{
			"title": buildPlainTextRichText(strings.TrimSpace(title)),
		}
	}
	if len(properties) > 0 {
		payload["properties"] = properties
	}
	if archived, ok := asBool(input["archived"]); ok {
		payload["archived"] = archived
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

func normalizeNotionFileObject(raw any, allowEmoji bool) (map[string]any, *apperr.AppError) {
	switch value := raw.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, apperr.New("INVALID_INPUT", "file object value cannot be empty")
		}
		if allowEmoji && !strings.Contains(value, "://") {
			return map[string]any{
				"type":  "emoji",
				"emoji": strings.TrimSpace(value),
			}, nil
		}
		return map[string]any{
			"type": "external",
			"external": map[string]any{
				"url": strings.TrimSpace(value),
			},
		}, nil
	case map[string]any:
		return cloneMap(value), nil
	default:
		return nil, apperr.New("INVALID_INPUT", "file object must be a string or an object")
	}
}

func buildPageParent(profile ExecutionProfile, raw any) (map[string]any, string, *apperr.AppError) {
	profile = normalizeExecutionProfile(profile)
	// Public integrations may create top-level private workspace pages, so missing parent is allowed here.
	if raw == nil && profile.Method == "notion.oauth_public" {
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
		return map[string]any{
			requestKey: strings.TrimSpace(parentID),
		}, parentType, nil
	case "workspace":
		if profile.Method != "notion.oauth_public" {
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

func mapPageMarkdownData(page notionPageMarkdown) map[string]any {
	unknownBlockIDs := make([]string, 0, len(page.UnknownBlockIDs))
	for _, blockID := range page.UnknownBlockIDs {
		if strings.TrimSpace(blockID) != "" {
			unknownBlockIDs = append(unknownBlockIDs, strings.TrimSpace(blockID))
		}
	}

	return map[string]any{
		"page_id":           page.ID,
		"object":            page.Object,
		"markdown":          page.Markdown,
		"truncated":         page.Truncated,
		"unknown_block_ids": unknownBlockIDs,
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

func decodePageMarkdownResponse(responseBody []byte, decodeErrorMessage string) (notionPageMarkdown, *apperr.AppError) {
	var response notionPageMarkdown
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return notionPageMarkdown{}, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("%s: %v", decodeErrorMessage, err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return notionPageMarkdown{}, apperr.New("UPSTREAM_INVALID_RESPONSE", "page markdown id is empty in Notion response")
	}
	return response, nil
}

func buildUpdatePageMarkdownPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	commandType, ok := asString(input["type"])
	if !ok || strings.TrimSpace(commandType) == "" {
		commandType = inferMarkdownUpdateType(input)
	}
	commandType = strings.TrimSpace(commandType)
	if commandType == "" {
		return nil, apperr.New("INVALID_INPUT", "type is required for page.markdown.update")
	}

	switch commandType {
	case "update_content":
		body, appErr := buildUpdateContentCommand(input["update_content"])
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"type":           "update_content",
			"update_content": body,
		}, nil
	case "replace_content":
		body, appErr := buildReplaceContentCommand(input["replace_content"])
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"type":            "replace_content",
			"replace_content": body,
		}, nil
	case "insert_content":
		body, appErr := buildInsertContentCommand(input["insert_content"])
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"type":           "insert_content",
			"insert_content": body,
		}, nil
	case "replace_content_range":
		body, appErr := buildReplaceContentRangeCommand(input["replace_content_range"])
		if appErr != nil {
			return nil, appErr
		}
		return map[string]any{
			"type":                  "replace_content_range",
			"replace_content_range": body,
		}, nil
	default:
		return nil, apperr.New("INVALID_INPUT", "type must be one of update_content, replace_content, insert_content, or replace_content_range")
	}
}

func inferMarkdownUpdateType(input map[string]any) string {
	for _, commandType := range []string{"update_content", "replace_content", "insert_content", "replace_content_range"} {
		if _, exists := input[commandType]; exists {
			return commandType
		}
	}
	return ""
}

func buildUpdateContentCommand(raw any) (map[string]any, *apperr.AppError) {
	command, ok := asMap(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "update_content must be an object")
	}

	rawUpdates, ok := asArray(command["content_updates"])
	if !ok || len(rawUpdates) == 0 {
		return nil, apperr.New("INVALID_INPUT", "update_content.content_updates must be a non-empty array")
	}

	updates := make([]map[string]any, 0, len(rawUpdates))
	for _, rawUpdate := range rawUpdates {
		record, ok := asMap(rawUpdate)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "each content update must be an object")
		}
		oldStr, ok := asString(record["old_str"])
		if !ok || strings.TrimSpace(oldStr) == "" {
			return nil, apperr.New("INVALID_INPUT", "old_str is required for each content update")
		}
		newStr, ok := asString(record["new_str"])
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "new_str is required for each content update")
		}

		update := map[string]any{
			"old_str": strings.TrimSpace(oldStr),
			"new_str": newStr,
		}
		if replaceAllMatches, ok := asBool(record["replace_all_matches"]); ok {
			update["replace_all_matches"] = replaceAllMatches
		}
		updates = append(updates, update)
	}

	result := map[string]any{
		"content_updates": updates,
	}
	if allowDeletingContent, ok := asBool(command["allow_deleting_content"]); ok {
		result["allow_deleting_content"] = allowDeletingContent
	}
	return result, nil
}

func buildReplaceContentCommand(raw any) (map[string]any, *apperr.AppError) {
	command, ok := asMap(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "replace_content must be an object")
	}
	newStr, ok := asString(command["new_str"])
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "replace_content.new_str is required")
	}

	result := map[string]any{
		"new_str": newStr,
	}
	if allowDeletingContent, ok := asBool(command["allow_deleting_content"]); ok {
		result["allow_deleting_content"] = allowDeletingContent
	}
	return result, nil
}

func buildInsertContentCommand(raw any) (map[string]any, *apperr.AppError) {
	command, ok := asMap(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "insert_content must be an object")
	}
	content, ok := asString(command["content"])
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "insert_content.content is required")
	}

	result := map[string]any{
		"content": content,
	}
	if after, ok := asString(command["after"]); ok && strings.TrimSpace(after) != "" {
		result["after"] = strings.TrimSpace(after)
	}
	return result, nil
}

func buildReplaceContentRangeCommand(raw any) (map[string]any, *apperr.AppError) {
	command, ok := asMap(raw)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "replace_content_range must be an object")
	}
	content, ok := asString(command["content"])
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "replace_content_range.content is required")
	}
	contentRange, ok := asString(command["content_range"])
	if !ok || strings.TrimSpace(contentRange) == "" {
		return nil, apperr.New("INVALID_INPUT", "replace_content_range.content_range is required")
	}

	result := map[string]any{
		"content":       content,
		"content_range": strings.TrimSpace(contentRange),
	}
	if allowDeletingContent, ok := asBool(command["allow_deleting_content"]); ok {
		result["allow_deleting_content"] = allowDeletingContent
	}
	return result, nil
}
