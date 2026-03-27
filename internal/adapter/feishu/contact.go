package feishu

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

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
	return result, nil
}
