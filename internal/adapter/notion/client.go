package notion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

const (
	defaultBaseURL       = "https://api.notion.com"
	defaultNotionVersion = "2026-03-11"
)

// Client 是当前 MVP 阶段使用的 Notion API 客户端。
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// Options 控制 Notion 客户端的构造参数。
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient 创建 Notion API 客户端。
func NewClient(options Options) (*Client, error) {
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Notion base URL: %w", err)
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		baseURL:    parsed,
		httpClient: httpClient,
	}, nil
}

// doJSONRequest 统一处理 Notion 的 JSON 请求和错误归一化。
func (c *Client) doJSONRequest(ctx context.Context, method, rawPath string, query url.Values, body any, authorization string, notionVersion string, extraHeaders map[string]string) ([]byte, *apperr.AppError) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, rawPath)
	endpoint.RawQuery = query.Encode()

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to encode Notion request body: %v", err))
		}
		bodyReader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return nil, apperr.New("REQUEST_BUILD_FAILED", fmt.Sprintf("failed to build Notion request: %v", err))
	}

	request.Header.Set("Accept", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	if strings.TrimSpace(notionVersion) != "" {
		request.Header.Set("Notion-Version", notionVersion)
	}
	for key, value := range extraHeaders {
		request.Header.Set(key, value)
	}
	if body != nil && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, apperr.New("HTTP_REQUEST_FAILED", fmt.Sprintf("failed to call Notion API: %v", err)).WithRetryable(true)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, apperr.New("HTTP_READ_FAILED", fmt.Sprintf("failed to read Notion response: %v", err))
	}

	if response.StatusCode >= 400 {
		return nil, normalizeNotionHTTPError(response.StatusCode, response.Header, responseBody)
	}

	return responseBody, nil
}

// requireAccessToken 根据 profile 获取可用的访问令牌。
func (c *Client) requireAccessToken(ctx context.Context, profile config.Profile) (string, string, *apperr.AppError) {
	if profile.Subject != "integration" {
		return "", "", apperr.New("SUBJECT_NOT_ALLOWED", "this Notion operation currently requires an integration profile")
	}

	notionVersion := resolveNotionVersion(profile)
	switch profile.Grant.Type {
	case "static_token":
		token, err := config.ResolveSecret(profile.Grant.Token)
		if err != nil {
			return "", "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
		}
		if strings.TrimSpace(token) == "" {
			return "", "", apperr.New("INVALID_AUTH_CONFIG", "Notion token is empty")
		}
		return token, notionVersion, nil
	case "oauth_refreshable":
		token, appErr := c.refreshAccessToken(ctx, profile)
		if appErr != nil {
			return "", "", appErr
		}
		return token, notionVersion, nil
	default:
		return "", "", apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Notion operation does not support grant type %s", profile.Grant.Type))
	}
}

// refreshAccessToken 使用 refresh token 即时换取新的 access token。
func (c *Client) refreshAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	clientID, err := config.ResolveSecret(profile.Grant.ClientID)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}
	clientSecret, err := config.ResolveSecret(profile.Grant.ClientSecret)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}
	refreshToken, err := config.ResolveSecret(profile.Grant.RefreshToken)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/v1/oauth/token",
		nil,
		map[string]any{
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
		},
		"Basic "+credentials,
		"",
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		if appErr.Code == "AUTH_FAILED" {
			appErr.Code = "AUTH_REFRESH_FAILED"
		}
		return "", appErr
	}

	var response notionOAuthTokenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion token response: %v", err))
	}
	if strings.TrimSpace(response.AccessToken) == "" {
		return "", apperr.New("UPSTREAM_INVALID_RESPONSE", "access_token is empty in Notion token response")
	}
	return response.AccessToken, nil
}

func resolveNotionVersion(profile config.Profile) string {
	if strings.TrimSpace(profile.Grant.NotionVer) != "" {
		return strings.TrimSpace(profile.Grant.NotionVer)
	}
	return defaultNotionVersion
}

func normalizeNotionHTTPError(httpStatus int, headers http.Header, responseBody []byte) *apperr.AppError {
	var response notionErrorResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = "Notion API request failed"
		}
		appErr := apperr.New("NOTION_API_ERROR", message).WithHTTPStatus(httpStatus)
		if httpStatus >= 500 {
			appErr.Code = "UPSTREAM_TEMPORARY_FAILURE"
			appErr.Retryable = true
		}
		return appErr
	}

	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = "Notion API request failed"
	}
	appErr := apperr.New("NOTION_API_ERROR", message).WithHTTPStatus(httpStatus)
	appErr.UpstreamCode = strings.TrimSpace(response.Code)

	switch response.Code {
	case "unauthorized":
		appErr.Code = "AUTH_FAILED"
	case "invalid_grant":
		appErr.Code = "AUTH_REFRESH_FAILED"
	case "restricted_resource":
		appErr.Code = "PERMISSION_DENIED"
	case "object_not_found":
		appErr.Code = "RESOURCE_NOT_FOUND"
		// Notion 会把“对象不存在”和“对象未共享给 integration”都归到这个错误码。
		if !strings.Contains(appErr.Message, "integration") {
			appErr.Message += "; the resource may not exist or may not be shared with the integration"
		}
	case "validation_error", "invalid_json", "invalid_request", "missing_version":
		appErr.Code = "INVALID_INPUT"
	case "rate_limited":
		appErr.Code = "RATE_LIMITED"
		appErr.Retryable = true
	case "conflict_error", "internal_server_error", "service_unavailable", "database_connection_unavailable", "gateway_timeout":
		appErr.Code = "UPSTREAM_TEMPORARY_FAILURE"
		appErr.Retryable = true
	}

	if response.Code == "" {
		switch httpStatus {
		case http.StatusUnauthorized:
			appErr.Code = "AUTH_FAILED"
		case http.StatusForbidden:
			appErr.Code = "PERMISSION_DENIED"
		case http.StatusNotFound:
			appErr.Code = "RESOURCE_NOT_FOUND"
		case http.StatusTooManyRequests:
			appErr.Code = "RATE_LIMITED"
			appErr.Retryable = true
		default:
			if httpStatus >= 500 {
				appErr.Code = "UPSTREAM_TEMPORARY_FAILURE"
				appErr.Retryable = true
			}
		}
	}

	if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" && !strings.Contains(appErr.Message, "Retry-After") {
		appErr.Message += fmt.Sprintf(" (Retry-After: %s)", retryAfter)
	}

	return appErr
}

func cloneMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func asString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok
}

func asBool(value any) (bool, bool) {
	boolean, ok := value.(bool)
	return boolean, ok
}

func asMap(value any) (map[string]any, bool) {
	record, ok := value.(map[string]any)
	return record, ok
}

func asArray(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}

func buildPlainTextRichText(content string) []map[string]any {
	if strings.TrimSpace(content) == "" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"type": "text",
			"text": map[string]any{
				"content": content,
			},
		},
	}
}

type notionErrorResponse struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type notionOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

type notionPage struct {
	ID         string         `json:"id"`
	URL        string         `json:"url"`
	Archived   bool           `json:"archived"`
	InTrash    bool           `json:"in_trash"`
	Parent     map[string]any `json:"parent"`
	Properties map[string]any `json:"properties"`
}

type notionBlock struct {
	ID string `json:"id"`
}

type notionBlockChildrenResponse struct {
	Results []notionBlock `json:"results"`
}

type notionUser struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	AvatarURL string            `json:"avatar_url"`
	Person    *notionPersonInfo `json:"person"`
}

type notionPersonInfo struct {
	Email string `json:"email"`
}
