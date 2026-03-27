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

// ListBitableRecords lists records from one Feishu Bitable table.
func (c *Client) ListBitableRecords(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	appToken, tableID, appErr := requireBitableTableIdentity(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if viewID, ok := asString(input["view_id"]); ok && strings.TrimSpace(viewID) != "" {
		query.Set("view_id", strings.TrimSpace(viewID))
	}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if filter, ok := asString(input["filter"]); ok && strings.TrimSpace(filter) != "" {
		query.Set("filter", strings.TrimSpace(filter))
	}
	if sort, ok := asString(input["sort"]); ok && strings.TrimSpace(sort) != "" {
		query.Set("sort", strings.TrimSpace(sort))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/bitable/v1/apps/"+url.PathEscape(appToken)+"/tables/"+url.PathEscape(tableID)+"/records",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode bitable record list response")
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0)
	for _, item := range extractFeishuRecordList(data, "items", "records") {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		items = append(items, normalizeBitableRecord(record))
	}

	result := map[string]any{
		"app_token":       appToken,
		"table_id":        tableID,
		"items":           items,
		"next_page_token": extractFirstNonEmptyString(data, "page_token", "next_page_token"),
	}
	if total, ok := asInt(data["total"]); ok {
		result["total"] = total
	}
	if hasMore, ok := asBool(data["has_more"]); ok {
		result["has_more"] = hasMore
	}
	return result, nil
}

// GetBitableRecord fetches one Bitable record by id.
func (c *Client) GetBitableRecord(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	appToken, tableID, recordID, appErr := requireBitableRecordIdentity(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/bitable/v1/apps/"+url.PathEscape(appToken)+"/tables/"+url.PathEscape(tableID)+"/records/"+url.PathEscape(recordID),
		nil,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode bitable record get response")
	if appErr != nil {
		return nil, appErr
	}

	record, ok := asMap(data["record"])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "record is empty in Feishu response")
	}
	normalized := normalizeBitableRecord(record)
	normalized["app_token"] = appToken
	normalized["table_id"] = tableID
	return normalized, nil
}

// CreateBitableRecord creates one Bitable record.
func (c *Client) CreateBitableRecord(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	appToken, tableID, fields, appErr := buildBitableRecordWriteRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/bitable/v1/apps/"+url.PathEscape(appToken)+"/tables/"+url.PathEscape(tableID)+"/records",
		nil,
		map[string]any{
			"fields": fields,
		},
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode bitable record create response")
	if appErr != nil {
		return nil, appErr
	}

	record, ok := asMap(data["record"])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "record is empty in Feishu response")
	}
	normalized := normalizeBitableRecord(record)
	normalized["app_token"] = appToken
	normalized["table_id"] = tableID
	return normalized, nil
}

// UpdateBitableRecord updates one Bitable record.
func (c *Client) UpdateBitableRecord(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	appToken, tableID, recordID, appErr := requireBitableRecordIdentity(input)
	if appErr != nil {
		return nil, appErr
	}
	fields, ok := asMap(input["fields"])
	if !ok || len(fields) == 0 {
		return nil, apperr.New("INVALID_INPUT", "fields is required")
	}

	accessToken, appErr := c.requireBotAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPut,
		"/open-apis/bitable/v1/apps/"+url.PathEscape(appToken)+"/tables/"+url.PathEscape(tableID)+"/records/"+url.PathEscape(recordID),
		nil,
		map[string]any{
			"fields": cloneFeishuMap(fields),
		},
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode bitable record update response")
	if appErr != nil {
		return nil, appErr
	}

	record, ok := asMap(data["record"])
	if !ok {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "record is empty in Feishu response")
	}
	normalized := normalizeBitableRecord(record)
	normalized["app_token"] = appToken
	normalized["table_id"] = tableID
	return normalized, nil
}

func requireBitableTableIdentity(input map[string]any) (string, string, *apperr.AppError) {
	appToken, ok := asString(input["app_token"])
	if !ok || strings.TrimSpace(appToken) == "" {
		return "", "", apperr.New("INVALID_INPUT", "app_token is required")
	}
	tableID, ok := asString(input["table_id"])
	if !ok || strings.TrimSpace(tableID) == "" {
		return "", "", apperr.New("INVALID_INPUT", "table_id is required")
	}
	return strings.TrimSpace(appToken), strings.TrimSpace(tableID), nil
}

func requireBitableRecordIdentity(input map[string]any) (string, string, string, *apperr.AppError) {
	appToken, tableID, appErr := requireBitableTableIdentity(input)
	if appErr != nil {
		return "", "", "", appErr
	}
	recordID, ok := asString(input["record_id"])
	if !ok || strings.TrimSpace(recordID) == "" {
		return "", "", "", apperr.New("INVALID_INPUT", "record_id is required")
	}
	return appToken, tableID, strings.TrimSpace(recordID), nil
}

func buildBitableRecordWriteRequest(input map[string]any) (string, string, map[string]any, *apperr.AppError) {
	appToken, tableID, appErr := requireBitableTableIdentity(input)
	if appErr != nil {
		return "", "", nil, appErr
	}
	fields, ok := asMap(input["fields"])
	if !ok || len(fields) == 0 {
		return "", "", nil, apperr.New("INVALID_INPUT", "fields is required")
	}
	return appToken, tableID, cloneFeishuMap(fields), nil
}

func normalizeBitableRecord(record map[string]any) map[string]any {
	result := map[string]any{
		"record_id": extractFirstNonEmptyString(record, "record_id"),
		"raw":       cloneFeishuMap(record),
	}
	if fields, ok := asMap(record["fields"]); ok {
		result["fields"] = cloneFeishuMap(fields)
	}
	if createdTime, ok := asInt(record["created_time"]); ok {
		result["created_time"] = createdTime
	}
	if lastModifiedTime, ok := asInt(record["last_modified_time"]); ok {
		result["last_modified_time"] = lastModifiedTime
	}
	return result
}
