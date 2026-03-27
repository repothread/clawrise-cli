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

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

const (
	defaultBaseURL       = "https://api.notion.com"
	defaultNotionVersion = "2026-03-11"
)

// Client is the Notion API client used by the current MVP runtime.
type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	sessionStore authcache.Store
	now          func() time.Time
}

// Options controls Notion client construction.
type Options struct {
	BaseURL      string
	HTTPClient   *http.Client
	SessionStore authcache.Store
}

// NewClient creates a Notion API client.
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

	sessionStore := options.SessionStore
	if sessionStore == nil {
		sessionStore = newDefaultSessionStore()
	}

	return &Client{
		baseURL:      parsed,
		httpClient:   httpClient,
		sessionStore: sessionStore,
		now:          time.Now,
	}, nil
}

// doJSONRequest handles Notion JSON requests and normalized error mapping.
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

// requireAccessToken resolves a usable access token from the given profile.
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
		cachedSession, ok := c.loadCachedSession(ctx, profile)
		if ok && cachedSession.UsableAt(c.now(), authcache.DefaultRefreshSkew) {
			return strings.TrimSpace(cachedSession.AccessToken), notionVersion, nil
		}

		refreshedSession, appErr := c.refreshAccessToken(ctx, profile, cachedSession)
		if appErr == nil {
			c.saveCachedSession(ctx, profile, *refreshedSession)
			return strings.TrimSpace(refreshedSession.AccessToken), notionVersion, nil
		}

		if token, err := config.ResolveSecret(profile.Grant.AccessToken); err == nil && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token), notionVersion, nil
		}
		return "", "", appErr
	default:
		return "", "", apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Notion operation does not support grant type %s", profile.Grant.Type))
	}
}

// refreshAccessToken exchanges a refresh token for a new access token.
func (c *Client) refreshAccessToken(ctx context.Context, profile config.Profile, currentSession *authcache.Session) (*authcache.Session, *apperr.AppError) {
	clientID, err := config.ResolveSecret(profile.Grant.ClientID)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}
	clientSecret, err := config.ResolveSecret(profile.Grant.ClientSecret)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}
	refreshToken, err := resolveNotionRefreshToken(profile, currentSession)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", err.Error())
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
		return nil, appErr
	}

	var response notionOAuthTokenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Notion token response: %v", err))
	}
	if strings.TrimSpace(response.AccessToken) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "access_token is empty in Notion token response")
	}
	nextRefreshToken := strings.TrimSpace(response.RefreshToken)
	if nextRefreshToken == "" {
		nextRefreshToken = strings.TrimSpace(refreshToken)
	}

	profileName := adapter.ProfileNameFromContext(ctx)
	session := buildOAuthSession(c.now(), profileName, profile, response.AccessToken, nextRefreshToken, response.TokenType, response.ExpiresIn)
	return &session, nil
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
		// Notion uses the same code for both a missing object and an object not shared with the integration.
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

func asInt(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int32:
		return int(number), true
	case int64:
		return int(number), true
	case float64:
		return int(number), true
	case json.Number:
		result, err := number.Int64()
		if err != nil {
			return 0, false
		}
		return int(result), true
	default:
		return 0, false
	}
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
	ExpiresIn    int    `json:"expires_in"`
}

type notionPage struct {
	ID         string         `json:"id"`
	URL        string         `json:"url"`
	Archived   bool           `json:"archived"`
	InTrash    bool           `json:"in_trash"`
	Parent     map[string]any `json:"parent"`
	Properties map[string]any `json:"properties"`
}

type notionPageMarkdown struct {
	Object          string   `json:"object"`
	ID              string   `json:"id"`
	Markdown        string   `json:"markdown"`
	Truncated       bool     `json:"truncated"`
	UnknownBlockIDs []string `json:"unknown_block_ids"`
}

type notionSearchResponse struct {
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
}

type notionQueryDataSourceResponse struct {
	Type       string           `json:"type"`
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
}

type notionBlockChildrenResponse struct {
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
}

type notionCommentListResponse struct {
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
}

type notionUserListResponse struct {
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor *string          `json:"next_cursor"`
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
