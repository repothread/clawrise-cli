package feishu

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// ListDepartments 列出指定部门下的子部门，未传 department_id 时默认从根部门开始。
func (c *Client) ListDepartments(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	departmentID := "0"
	if value, ok := asString(input["department_id"]); ok && strings.TrimSpace(value) != "" {
		departmentID = strings.TrimSpace(value)
	}

	query := url.Values{}
	if userIDType, ok := asString(input["user_id_type"]); ok && strings.TrimSpace(userIDType) != "" {
		query.Set("user_id_type", strings.TrimSpace(userIDType))
	}
	if departmentIDType, ok := asString(input["department_id_type"]); ok && strings.TrimSpace(departmentIDType) != "" {
		query.Set("department_id_type", strings.TrimSpace(departmentIDType))
	}
	if fetchChild, ok := asBool(input["fetch_child"]); ok {
		query.Set("fetch_child", strconv.FormatBool(fetchChild))
	}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	query.Set("parent_department_id", departmentID)

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/contact/v3/departments",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode Feishu department list response")
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0)
	for _, item := range extractFeishuRecordList(data, "items", "departments") {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		items = append(items, normalizeDepartment(record))
	}

	result := map[string]any{
		"department_id":    departmentID,
		"items":            items,
		"next_page_token":  extractFirstNonEmptyString(data, "page_token", "next_page_token"),
		"has_more":         false,
		"department_scope": "children",
	}
	if hasMore, ok := asBool(data["has_more"]); ok {
		result["has_more"] = hasMore
	}
	return result, nil
}

// ListDepartmentUsers 列出一个部门的直属用户。
func (c *Client) ListDepartmentUsers(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	departmentID, ok := asString(input["department_id"])
	if !ok || strings.TrimSpace(departmentID) == "" {
		return nil, apperr.New("INVALID_INPUT", "department_id is required")
	}
	departmentID = strings.TrimSpace(departmentID)

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	query.Set("department_id", departmentID)
	if userIDType, ok := asString(input["user_id_type"]); ok && strings.TrimSpace(userIDType) != "" {
		query.Set("user_id_type", strings.TrimSpace(userIDType))
	}
	if departmentIDType, ok := asString(input["department_id_type"]); ok && strings.TrimSpace(departmentIDType) != "" {
		query.Set("department_id_type", strings.TrimSpace(departmentIDType))
	}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
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
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode Feishu department user list response")
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0)
	for _, item := range extractFeishuRecordList(data, "items", "users") {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		items = append(items, normalizeContactUser(record))
	}

	result := map[string]any{
		"department_id":   departmentID,
		"items":           items,
		"next_page_token": extractFirstNonEmptyString(data, "page_token", "next_page_token"),
		"has_more":        false,
	}
	if hasMore, ok := asBool(data["has_more"]); ok {
		result["has_more"] = hasMore
	}
	return result, nil
}

func normalizeDepartment(record map[string]any) map[string]any {
	result := map[string]any{
		"department_id":      extractFirstNonEmptyString(record, "open_department_id", "department_id"),
		"open_department_id": extractFirstNonEmptyString(record, "open_department_id"),
		"raw":                cloneFeishuMap(record),
	}
	for _, key := range []string{"name", "department_id", "parent_department_id", "leader_user_id", "chat_id"} {
		if value, ok := asString(record[key]); ok && strings.TrimSpace(value) != "" {
			result[key] = strings.TrimSpace(value)
		}
	}
	if count, ok := asInt(record["member_count"]); ok {
		result["member_count"] = count
	}
	if count, ok := asInt(record["primary_member_count"]); ok {
		result["primary_member_count"] = count
	}
	if status, ok := asMap(record["status"]); ok && len(status) > 0 {
		result["status"] = cloneFeishuMap(status)
	}
	return result
}
