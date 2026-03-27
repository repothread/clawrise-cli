package feishu

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

const (
	defaultContactUserSearchPageSize = 20
	maxContactUserSearchPageSize     = 100
	contactUserSearchScanPageSize    = 50
)

type contactUserSearchCursor struct {
	UpstreamPageToken string `json:"upstream_page_token,omitempty"`
	MatchOffset       int    `json:"match_offset,omitempty"`
}

// GetUser reads one Feishu user profile.
func (c *Client) GetUser(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	userID, ok := asString(input["user_id"])
	if !ok || strings.TrimSpace(userID) == "" {
		return nil, apperr.New("INVALID_INPUT", "user_id is required")
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if userIDType, ok := asString(input["user_id_type"]); ok && strings.TrimSpace(userIDType) != "" {
		query.Set("user_id_type", strings.TrimSpace(userIDType))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/contact/v3/users/"+url.PathEscape(strings.TrimSpace(userID)),
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode Feishu user response")
	if appErr != nil {
		return nil, appErr
	}

	user, ok := asMap(data["user"])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "user is empty in Feishu response")
	}

	return normalizeContactUser(user), nil
}

// SearchUsers searches visible Feishu users by partial identity input.
func (c *Client) SearchUsers(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	query, ok := asString(input["query"])
	if !ok || strings.TrimSpace(query) == "" {
		return nil, apperr.New("INVALID_INPUT", "query is required")
	}
	query = strings.TrimSpace(query)

	pageSize := defaultContactUserSearchPageSize
	if value, ok := asInt(input["page_size"]); ok && value > 0 {
		pageSize = value
	}
	if pageSize > maxContactUserSearchPageSize {
		pageSize = maxContactUserSearchPageSize
	}

	cursor, appErr := decodeContactUserSearchCursor(input["page_token"])
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0, pageSize)
	currentPageToken := cursor.UpstreamPageToken
	currentMatchOffset := cursor.MatchOffset

	for len(items) < pageSize {
		pageItems, nextUpstreamPageToken, hasMore, appErr := c.listContactUsersPage(ctx, accessToken, input, currentPageToken)
		if appErr != nil {
			return nil, appErr
		}

		matchedItems := make([]map[string]any, 0)
		for _, item := range pageItems {
			user, ok := asMap(item)
			if !ok {
				continue
			}
			matchedFields := matchContactUser(query, user)
			if len(matchedFields) == 0 {
				continue
			}

			normalized := normalizeContactUser(user)
			normalized["matched_fields"] = matchedFields
			matchedItems = append(matchedItems, normalized)
		}

		if currentMatchOffset > len(matchedItems) {
			currentMatchOffset = len(matchedItems)
		}
		remainingMatches := matchedItems[currentMatchOffset:]
		if len(remainingMatches) == 0 {
			if !hasMore || strings.TrimSpace(nextUpstreamPageToken) == "" {
				break
			}
			currentPageToken = nextUpstreamPageToken
			currentMatchOffset = 0
			continue
		}

		remainingSlots := pageSize - len(items)
		if len(remainingMatches) > remainingSlots {
			items = append(items, remainingMatches[:remainingSlots]...)
			return map[string]any{
				"query":           query,
				"items":           items,
				"next_page_token": encodeContactUserSearchCursor(currentPageToken, currentMatchOffset+remainingSlots),
				"has_more":        true,
			}, nil
		}

		items = append(items, remainingMatches...)
		if len(items) == pageSize {
			nextPageToken := ""
			hasMoreResult := false
			if hasMore && strings.TrimSpace(nextUpstreamPageToken) != "" {
				nextPageToken = encodeContactUserSearchCursor(nextUpstreamPageToken, 0)
				hasMoreResult = true
			}
			return map[string]any{
				"query":           query,
				"items":           items,
				"next_page_token": nextPageToken,
				"has_more":        hasMoreResult,
			}, nil
		}

		if !hasMore || strings.TrimSpace(nextUpstreamPageToken) == "" {
			break
		}
		currentPageToken = nextUpstreamPageToken
		currentMatchOffset = 0
	}

	return map[string]any{
		"query":           query,
		"items":           items,
		"next_page_token": "",
		"has_more":        false,
	}, nil
}

// listContactUsersPage 拉取一个上游分页，搜索逻辑仍然由适配器层完成。
func (c *Client) listContactUsersPage(ctx context.Context, accessToken string, input map[string]any, pageToken string) ([]any, string, bool, *apperr.AppError) {
	query := url.Values{}
	query.Set("page_size", strconv.Itoa(contactUserSearchScanPageSize))
	if strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if departmentID, ok := asString(input["department_id"]); ok && strings.TrimSpace(departmentID) != "" {
		query.Set("department_id", strings.TrimSpace(departmentID))
	}
	if departmentIDType, ok := asString(input["department_id_type"]); ok && strings.TrimSpace(departmentIDType) != "" {
		query.Set("department_id_type", strings.TrimSpace(departmentIDType))
	}
	if userIDType, ok := asString(input["user_id_type"]); ok && strings.TrimSpace(userIDType) != "" {
		query.Set("user_id_type", strings.TrimSpace(userIDType))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/contact/v3/users",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, "", false, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode Feishu user search response")
	if appErr != nil {
		return nil, "", false, appErr
	}

	items := extractFeishuRecordList(data, "items", "user_list")
	nextPageToken := extractFirstNonEmptyString(data, "page_token", "next_page_token")
	hasMore, _ := asBool(data["has_more"])
	return items, nextPageToken, hasMore, nil
}

func normalizeContactUser(user map[string]any) map[string]any {
	result := map[string]any{
		"user_id": extractFirstNonEmptyString(user, "user_id", "open_id"),
		"raw":     cloneFeishuMap(user),
	}
	for _, key := range []string{"name", "en_name", "email", "mobile", "employee_no", "open_id", "union_id"} {
		if value, ok := asString(user[key]); ok && strings.TrimSpace(value) != "" {
			result[key] = strings.TrimSpace(value)
		}
	}
	if status, ok := asMap(user["status"]); ok && len(status) > 0 {
		result["status"] = cloneFeishuMap(status)
	}
	if departmentIDs, ok := asArray(user["department_ids"]); ok && len(departmentIDs) > 0 {
		cloned := make([]string, 0, len(departmentIDs))
		for _, item := range departmentIDs {
			value, ok := asString(item)
			if !ok || strings.TrimSpace(value) == "" {
				continue
			}
			cloned = append(cloned, strings.TrimSpace(value))
		}
		if len(cloned) > 0 {
			result["department_ids"] = cloned
		}
	} else if departmentIDs, ok := user["department_ids"].([]string); ok && len(departmentIDs) > 0 {
		cloned := make([]string, 0, len(departmentIDs))
		for _, item := range departmentIDs {
			if strings.TrimSpace(item) == "" {
				continue
			}
			cloned = append(cloned, strings.TrimSpace(item))
		}
		if len(cloned) > 0 {
			result["department_ids"] = cloned
		}
	}
	if avatar, ok := asMap(user["avatar"]); ok && len(avatar) > 0 {
		result["avatar"] = cloneFeishuMap(avatar)
	}
	return result
}

func matchContactUser(query string, user map[string]any) []string {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return nil
	}

	matchedFields := make([]string, 0)
	for _, key := range []string{"name", "en_name", "email", "mobile", "employee_no", "user_id", "open_id", "union_id"} {
		value, ok := asString(user[key])
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), normalizedQuery) {
			matchedFields = append(matchedFields, key)
		}
	}
	return matchedFields
}

func decodeContactUserSearchCursor(raw any) (contactUserSearchCursor, *apperr.AppError) {
	pageToken, ok := asString(raw)
	if !ok || strings.TrimSpace(pageToken) == "" {
		return contactUserSearchCursor{}, nil
	}
	pageToken = strings.TrimSpace(pageToken)

	decoded, err := base64.RawURLEncoding.DecodeString(pageToken)
	if err != nil {
		return contactUserSearchCursor{
			UpstreamPageToken: pageToken,
		}, nil
	}

	cursor := contactUserSearchCursor{}
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return contactUserSearchCursor{}, apperr.New("INVALID_INPUT", "page_token is invalid")
	}
	if cursor.MatchOffset < 0 {
		cursor.MatchOffset = 0
	}
	return cursor, nil
}

func encodeContactUserSearchCursor(pageToken string, matchOffset int) string {
	cursor := contactUserSearchCursor{
		UpstreamPageToken: strings.TrimSpace(pageToken),
		MatchOffset:       matchOffset,
	}
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(encoded)
}
