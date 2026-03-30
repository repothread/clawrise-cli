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

// GetUser reads a user object and supports both a concrete user_id and user_id=me.
func (c *Client) GetUser(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	userID, appErr := requireIDField(input, "user_id")
	if appErr != nil {
		return nil, appErr
	}

	accessToken, notionVersion, appErr := c.requireAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	requestPath := "/v1/users/" + url.PathEscape(userID)
	if userID == "me" {
		requestPath = "/v1/users/me"
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		requestPath,
		nil,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionUser
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion user response: %v", err))
	}
	if strings.TrimSpace(response.ID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "user id is empty in Notion response")
	}

	return normalizeNotionUser(response), nil
}

// ListUsers 列出当前集成可见的 Notion 用户。
func (c *Client) ListUsers(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
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
		"/v1/users",
		query,
		nil,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response notionUserListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion user list response: %v", err))
	}

	items := make([]map[string]any, 0, len(response.Results))
	for _, item := range response.Results {
		userBytes, err := json.Marshal(item)
		if err != nil {
			continue
		}

		user := notionUser{}
		if err := json.Unmarshal(userBytes, &user); err != nil {
			continue
		}
		if strings.TrimSpace(user.ID) == "" {
			continue
		}
		items = append(items, normalizeNotionUser(user))
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

func normalizeNotionUser(user notionUser) map[string]any {
	data := map[string]any{
		"user_id":    user.ID,
		"type":       user.Type,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
	}
	if user.Person != nil && strings.TrimSpace(user.Person.Email) != "" {
		data["email"] = user.Person.Email
	}
	return data
}
