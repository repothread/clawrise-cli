package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// ListWikiSpaces lists Feishu wiki spaces visible to the current bot.
func (c *Client) ListWikiSpaces(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/wiki/v2/spaces",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response wikiSpaceListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode wiki spaces response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		items = append(items, map[string]any{
			"space_id":     item.SpaceID,
			"name":         item.Name,
			"description":  item.Description,
			"space_type":   item.SpaceType,
			"visibility":   item.Visibility,
			"open_sharing": item.OpenSharing,
		})
	}

	return map[string]any{
		"items":           items,
		"next_page_token": response.Data.PageToken,
		"has_more":        response.Data.HasMore,
	}, nil
}

// ListWikiNodes lists child wiki nodes under a given parent.
func (c *Client) ListWikiNodes(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	spaceID, ok := asString(input["space_id"])
	if !ok || strings.TrimSpace(spaceID) == "" {
		return nil, apperr.New("INVALID_INPUT", "space_id is required")
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if parentNodeToken, ok := asString(input["parent_node_token"]); ok && strings.TrimSpace(parentNodeToken) != "" {
		query.Set("parent_node_token", strings.TrimSpace(parentNodeToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/wiki/v2/spaces/"+url.PathEscape(spaceID)+"/nodes",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response wikiNodeListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode wiki nodes response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	items := make([]map[string]any, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		items = append(items, map[string]any{
			"space_id":          item.SpaceID,
			"node_token":        item.NodeToken,
			"obj_token":         item.ObjToken,
			"obj_type":          item.ObjType,
			"parent_node_token": item.ParentNodeToken,
			"node_type":         item.NodeType,
			"title":             item.Title,
			"has_child":         item.HasChild,
		})
	}

	return map[string]any{
		"items":           items,
		"next_page_token": response.Data.PageToken,
		"has_more":        response.Data.HasMore,
	}, nil
}

// CreateWikiNode creates a docx node under the given wiki space/parent.
func (c *Client) CreateWikiNode(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	request, appErr := buildCreateWikiNodeRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/wiki/v2/spaces/"+url.PathEscape(request.SpaceID)+"/nodes",
		nil,
		request.Body,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	var response wikiNodeCreateResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode wiki node create response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}

	return map[string]any{
		"space_id":          response.Data.Node.SpaceID,
		"node_token":        response.Data.Node.NodeToken,
		"obj_token":         response.Data.Node.ObjToken,
		"obj_type":          response.Data.Node.ObjType,
		"parent_node_token": response.Data.Node.ParentNodeToken,
		"node_type":         response.Data.Node.NodeType,
		"title":             response.Data.Node.Title,
		"document_id":       response.Data.Node.ObjToken,
	}, nil
}

type wikiSpaceListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items     []wikiSpace `json:"items"`
		PageToken string      `json:"page_token"`
		HasMore   bool        `json:"has_more"`
	} `json:"data"`
}

type wikiSpace struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SpaceID     string `json:"space_id"`
	SpaceType   string `json:"space_type"`
	Visibility  string `json:"visibility"`
	OpenSharing string `json:"open_sharing"`
}

type wikiNodeListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Items     []wikiNode `json:"items"`
		PageToken string     `json:"page_token"`
		HasMore   bool       `json:"has_more"`
	} `json:"data"`
}

type wikiNodeCreateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Node wikiNode `json:"node"`
	} `json:"data"`
}

type wikiNode struct {
	SpaceID          string `json:"space_id"`
	NodeToken        string `json:"node_token"`
	ObjToken         string `json:"obj_token"`
	ObjType          string `json:"obj_type"`
	ParentNodeToken  string `json:"parent_node_token"`
	NodeType         string `json:"node_type"`
	OriginNodeToken  string `json:"origin_node_token"`
	OriginSpaceID    string `json:"origin_space_id"`
	HasChild         bool   `json:"has_child"`
	Title            string `json:"title"`
	ObjectCreateTime string `json:"obj_create_time"`
	ObjectEditTime   string `json:"obj_edit_time"`
}

type createWikiNodeRequest struct {
	SpaceID string
	Body    createWikiNodePayload
}

type createWikiNodePayload struct {
	ObjType         string `json:"obj_type"`
	NodeType        string `json:"node_type"`
	ParentNodeToken string `json:"parent_node_token,omitempty"`
	OriginNodeToken string `json:"origin_node_token,omitempty"`
	Title           string `json:"title,omitempty"`
}

func buildCreateWikiNodeRequest(input map[string]any) (*createWikiNodeRequest, *apperr.AppError) {
	spaceID, ok := asString(input["space_id"])
	if !ok || strings.TrimSpace(spaceID) == "" {
		return nil, apperr.New("INVALID_INPUT", "space_id is required")
	}

	objType := "docx"
	if value, ok := asString(input["obj_type"]); ok && strings.TrimSpace(value) != "" {
		objType = strings.TrimSpace(value)
	}
	nodeType := "origin"
	if value, ok := asString(input["node_type"]); ok && strings.TrimSpace(value) != "" {
		nodeType = strings.TrimSpace(value)
	}

	body := createWikiNodePayload{
		ObjType:  objType,
		NodeType: nodeType,
	}
	if value, ok := asString(input["parent_node_token"]); ok && strings.TrimSpace(value) != "" {
		body.ParentNodeToken = strings.TrimSpace(value)
	}
	if value, ok := asString(input["origin_node_token"]); ok && strings.TrimSpace(value) != "" {
		body.OriginNodeToken = strings.TrimSpace(value)
	}
	if value, ok := asString(input["title"]); ok && strings.TrimSpace(value) != "" {
		body.Title = value
	}

	return &createWikiNodeRequest{
		SpaceID: strings.TrimSpace(spaceID),
		Body:    body,
	}, nil
}
