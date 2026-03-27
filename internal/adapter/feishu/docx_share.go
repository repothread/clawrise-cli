package feishu

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// ShareDocument 为新版文档增加一名协作者。
func (c *Client) ShareDocument(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	documentID, ok := asString(input["document_id"])
	if !ok || strings.TrimSpace(documentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "document_id is required")
	}
	documentID = strings.TrimSpace(documentID)

	memberType, ok := asString(input["member_type"])
	if !ok || strings.TrimSpace(memberType) == "" {
		return nil, apperr.New("INVALID_INPUT", "member_type is required")
	}
	memberID, ok := asString(input["member_id"])
	if !ok || strings.TrimSpace(memberID) == "" {
		return nil, apperr.New("INVALID_INPUT", "member_id is required")
	}
	perm, ok := asString(input["perm"])
	if !ok || strings.TrimSpace(perm) == "" {
		return nil, apperr.New("INVALID_INPUT", "perm is required")
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("type", "docx")
	if needNotification, ok := asBool(input["need_notification"]); ok {
		query.Set("need_notification", strings.ToLower(strconvFormatBool(needNotification)))
	}

	payload := map[string]any{
		"member_type": strings.TrimSpace(memberType),
		"member_id":   strings.TrimSpace(memberID),
		"perm":        strings.TrimSpace(perm),
	}
	if permType, ok := asString(input["perm_type"]); ok && strings.TrimSpace(permType) != "" {
		payload["perm_type"] = strings.TrimSpace(permType)
	}
	if memberKind, ok := asString(input["type"]); ok && strings.TrimSpace(memberKind) != "" {
		payload["type"] = strings.TrimSpace(memberKind)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/drive/v1/permissions/"+url.PathEscape(documentID)+"/members",
		query,
		payload,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode Feishu document share response")
	if appErr != nil {
		return nil, appErr
	}

	member, ok := asMap(data["member"])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "member is empty in Feishu response")
	}

	result := map[string]any{
		"document_id": documentID,
		"raw":         cloneFeishuMap(member),
	}
	for _, key := range []string{"member_type", "member_id", "perm", "perm_type", "type"} {
		if value, ok := asString(member[key]); ok && strings.TrimSpace(value) != "" {
			result[key] = strings.TrimSpace(value)
		}
	}
	return result, nil
}

func strconvFormatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
