package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// CreateDataSource 创建一个 Notion data source。
func (c *Client) CreateDataSource(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	payload, appErr := buildDataSourceWritePayload(input, false)
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
		"/v1/data_sources",
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source create response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || id == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "data source id is empty in Notion response")
	}
	return normalizeDataSourceObject(response), nil
}

// UpdateDataSource 更新一个 Notion data source。
func (c *Client) UpdateDataSource(ctx context.Context, profile config.Profile, input map[string]any) (map[string]any, *apperr.AppError) {
	dataSourceID, appErr := requireIDField(input, "data_source_id")
	if appErr != nil {
		return nil, appErr
	}
	payload, appErr := buildDataSourceWritePayload(input, true)
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
		"/v1/data_sources/"+url.PathEscape(dataSourceID),
		nil,
		payload,
		"Bearer "+accessToken,
		notionVersion,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion data source update response: %v", err))
	}
	if id, ok := asString(response["id"]); !ok || id == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "data source id is empty in Notion response")
	}
	return normalizeDataSourceObject(response), nil
}

// buildDataSourceWritePayload 优先支持 provider-native 的 body 透传。
func buildDataSourceWritePayload(input map[string]any, isUpdate bool) (map[string]any, *apperr.AppError) {
	if rawBody, exists := input["body"]; exists {
		body, ok := asMap(rawBody)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "body must be an object")
		}
		if len(body) == 0 {
			return nil, apperr.New("INVALID_INPUT", "body must not be empty")
		}
		return cloneMap(body), nil
	}

	payload := cloneMap(input)
	delete(payload, "body")
	delete(payload, "data_source_id")
	if len(payload) == 0 {
		if isUpdate {
			return nil, apperr.New("INVALID_INPUT", "body is required or at least one updatable field must be provided")
		}
		return nil, apperr.New("INVALID_INPUT", "body is required")
	}
	return payload, nil
}
