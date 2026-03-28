package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

const (
	defaultBaseURL = "https://open.feishu.cn"
)

// Client is a minimal Feishu API client used by the current MVP runtime.
type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	sessionStore authcache.Store
	now          func() time.Time
}

// Options controls Feishu client construction.
type Options struct {
	BaseURL      string
	HTTPClient   *http.Client
	SessionStore authcache.Store
}

// NewClient creates a Feishu API client.
func NewClient(options Options) (*Client, error) {
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Feishu base URL: %w", err)
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

// CreateCalendarEvent creates a Feishu calendar event using the selected Feishu identity.
func (c *Client) CreateCalendarEvent(ctx context.Context, profile config.Profile, input map[string]any, idempotencyKey string) (map[string]any, *apperr.AppError) {
	request, appErr := buildCreateCalendarEventRequest(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	response, appErr := c.createCalendarEvent(ctx, accessToken, request.CalendarID, request.Body, idempotencyKey)
	if appErr != nil {
		return nil, appErr
	}

	return mapCreateCalendarEventData(response), nil
}

func (c *Client) fetchTenantAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	appID, err := config.ResolveSecret(profile.Grant.AppID)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}
	appSecret, err := config.ResolveSecret(profile.Grant.AppSecret)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", err.Error())
	}

	payload := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}

	responseBody, appErr := c.doJSONRequest(ctx, http.MethodPost, "/open-apis/auth/v3/tenant_access_token/internal", nil, payload, "", nil)
	if appErr != nil {
		return "", appErr
	}

	var response tenantAccessTokenResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode tenant access token response: %v", err))
	}

	if response.Code != 0 {
		return "", normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if strings.TrimSpace(response.TenantAccessToken) == "" {
		return "", apperr.New("UPSTREAM_INVALID_RESPONSE", "tenant_access_token is empty")
	}
	return response.TenantAccessToken, nil
}

func (c *Client) createCalendarEvent(ctx context.Context, accessToken, calendarID string, payload createCalendarEventPayload, idempotencyKey string) (*createCalendarEventResponse, *apperr.AppError) {
	query := url.Values{}
	if strings.TrimSpace(idempotencyKey) != "" {
		query.Set("idempotency_key", idempotencyKey)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/calendar/v4/calendars/"+url.PathEscape(calendarID)+"/events",
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

	var response createCalendarEventResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode create event response: %v", err))
	}

	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if response.Data.Event.EventID == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "event_id is empty in Feishu response")
	}

	return &response, nil
}

func (c *Client) doJSONRequest(ctx context.Context, method, rawPath string, query url.Values, body any, authorization string, extraHeaders map[string]string) ([]byte, *apperr.AppError) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, rawPath)
	endpoint.RawQuery = query.Encode()

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, apperr.New("REQUEST_ENCODE_FAILED", fmt.Sprintf("failed to encode request body: %v", err))
		}
		bodyReader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return nil, apperr.New("REQUEST_BUILD_FAILED", fmt.Sprintf("failed to build HTTP request: %v", err))
	}

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	for key, value := range extraHeaders {
		request.Header.Set(key, value)
	}
	if request.Header.Get("Content-Type") == "" && body != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, apperr.New("HTTP_REQUEST_FAILED", fmt.Sprintf("failed to call Feishu API: %v", err)).WithRetryable(true)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, apperr.New("HTTP_READ_FAILED", fmt.Sprintf("failed to read Feishu response: %v", err))
	}

	if response.StatusCode >= 400 {
		return nil, normalizeFeishuError(0, strings.TrimSpace(string(responseBody)), response.StatusCode)
	}

	return responseBody, nil
}

type tenantAccessTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

type createCalendarEventRequest struct {
	CalendarID string
	Body       createCalendarEventPayload
}

type createCalendarEventPayload struct {
	Summary          string               `json:"summary,omitempty"`
	Description      string               `json:"description,omitempty"`
	NeedNotification bool                 `json:"need_notification"`
	StartTime        feishuTimeInfo       `json:"start_time"`
	EndTime          feishuTimeInfo       `json:"end_time"`
	Location         *feishuEventLocation `json:"location,omitempty"`
	Reminders        []feishuReminder     `json:"reminders,omitempty"`
}

type feishuTimeInfo struct {
	Timestamp string `json:"timestamp,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
}

type feishuEventLocation struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type feishuReminder struct {
	Minutes int `json:"minutes"`
}

type createCalendarEventResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Event feishuCalendarEvent `json:"event"`
	} `json:"data"`
}

type feishuCalendarEvent struct {
	EventID             string               `json:"event_id"`
	OrganizerCalendarID string               `json:"organizer_calendar_id"`
	Summary             string               `json:"summary"`
	Description         string               `json:"description"`
	StartTime           feishuTimeInfo       `json:"start_time"`
	EndTime             feishuTimeInfo       `json:"end_time"`
	AppLink             string               `json:"app_link"`
	Location            *feishuEventLocation `json:"location"`
}

func buildCreateCalendarEventRequest(input map[string]any) (*createCalendarEventRequest, *apperr.AppError) {
	calendarID, ok := asString(input["calendar_id"])
	if !ok || strings.TrimSpace(calendarID) == "" {
		return nil, apperr.New("INVALID_INPUT", "calendar_id is required")
	}
	summary, ok := asString(input["summary"])
	if !ok || strings.TrimSpace(summary) == "" {
		return nil, apperr.New("INVALID_INPUT", "summary is required")
	}
	startAt, ok := asString(input["start_at"])
	if !ok || strings.TrimSpace(startAt) == "" {
		return nil, apperr.New("INVALID_INPUT", "start_at is required")
	}
	endAt, ok := asString(input["end_at"])
	if !ok || strings.TrimSpace(endAt) == "" {
		return nil, apperr.New("INVALID_INPUT", "end_at is required")
	}
	if attendees, exists := input["attendees"]; exists {
		if list, ok := attendees.([]any); ok && len(list) > 0 {
			return nil, apperr.New("UNSUPPORTED_FIELD", "attendees are not supported in calendar.event.create; use a separate attendee API")
		}
	}

	startTime, appErr := buildFeishuTimeInfo(startAt, input["timezone"])
	if appErr != nil {
		return nil, appErr
	}
	endTime, appErr := buildFeishuTimeInfo(endAt, input["timezone"])
	if appErr != nil {
		return nil, appErr
	}

	startParsed, _ := time.Parse(time.RFC3339, startAt)
	endParsed, _ := time.Parse(time.RFC3339, endAt)
	if !endParsed.After(startParsed) {
		return nil, apperr.New("INVALID_INPUT", "end_at must be later than start_at")
	}

	payload := createCalendarEventPayload{
		Summary:          summary,
		NeedNotification: false,
		StartTime:        startTime,
		EndTime:          endTime,
	}

	if description, ok := asString(input["description"]); ok && strings.TrimSpace(description) != "" {
		payload.Description = description
	}

	location, appErr := buildLocation(input["location"])
	if appErr != nil {
		return nil, appErr
	}
	payload.Location = location

	reminders, appErr := buildReminders(input["reminders"])
	if appErr != nil {
		return nil, appErr
	}
	payload.Reminders = reminders

	return &createCalendarEventRequest{
		CalendarID: calendarID,
		Body:       payload,
	}, nil
}

func buildFeishuTimeInfo(value string, timezoneInput any) (feishuTimeInfo, *apperr.AppError) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return feishuTimeInfo{}, apperr.New("INVALID_INPUT", fmt.Sprintf("failed to parse RFC3339 time %q: %v", value, err))
	}

	timezone := ""
	if rawTimezone, ok := asString(timezoneInput); ok && strings.TrimSpace(rawTimezone) != "" {
		timezone = strings.TrimSpace(rawTimezone)
	} else {
		timezone = bestEffortTimezone(parsed)
	}

	return feishuTimeInfo{
		Timestamp: strconv.FormatInt(parsed.Unix(), 10),
		Timezone:  timezone,
	}, nil
}

func bestEffortTimezone(value time.Time) string {
	name, offset := value.Zone()
	if strings.Contains(name, "/") || name == "UTC" {
		return name
	}

	switch offset {
	case 0:
		return "UTC"
	case 8 * 3600:
		return "Asia/Shanghai"
	default:
		return ""
	}
}

func buildLocation(raw any) (*feishuEventLocation, *apperr.AppError) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, nil
		}
		return &feishuEventLocation{Name: strings.TrimSpace(value)}, nil
	case map[string]any:
		location := &feishuEventLocation{}
		if name, ok := asString(value["name"]); ok {
			location.Name = strings.TrimSpace(name)
		}
		if address, ok := asString(value["address"]); ok {
			location.Address = strings.TrimSpace(address)
		}
		if location.Name == "" && location.Address == "" {
			return nil, nil
		}
		return location, nil
	default:
		return nil, apperr.New("INVALID_INPUT", "location must be either a string or an object")
	}
}

func buildReminders(raw any) ([]feishuReminder, *apperr.AppError) {
	if raw == nil {
		return nil, nil
	}

	list, ok := raw.([]any)
	if !ok {
		return nil, apperr.New("INVALID_INPUT", "reminders must be an array")
	}

	reminders := make([]feishuReminder, 0, len(list))
	for _, item := range list {
		record, ok := item.(map[string]any)
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "each reminder must be an object")
		}
		minutes, ok := asInt(record["minutes_before"])
		if !ok {
			return nil, apperr.New("INVALID_INPUT", "minutes_before is required for each reminder")
		}
		reminders = append(reminders, feishuReminder{Minutes: minutes})
	}
	return reminders, nil
}

func mapCreateCalendarEventData(response *createCalendarEventResponse) map[string]any {
	return map[string]any{
		"event_id":    response.Data.Event.EventID,
		"calendar_id": response.Data.Event.OrganizerCalendarID,
		"summary":     response.Data.Event.Summary,
		"description": response.Data.Event.Description,
		"start_at":    formatFeishuTimeInfo(response.Data.Event.StartTime),
		"end_at":      formatFeishuTimeInfo(response.Data.Event.EndTime),
		"html_url":    response.Data.Event.AppLink,
		"raw": map[string]any{
			"provider_event_id": response.Data.Event.EventID,
		},
	}
}

func formatFeishuTimeInfo(value feishuTimeInfo) string {
	if strings.TrimSpace(value.Timestamp) == "" {
		return ""
	}

	seconds, err := strconv.ParseInt(value.Timestamp, 10, 64)
	if err != nil {
		return value.Timestamp
	}

	location := time.UTC
	if strings.TrimSpace(value.Timezone) != "" {
		if loaded, err := time.LoadLocation(value.Timezone); err == nil {
			location = loaded
		}
	}
	return time.Unix(seconds, 0).In(location).Format(time.RFC3339)
}

func normalizeFeishuError(code int, message string, httpStatus int) *apperr.AppError {
	err := apperr.New("FEISHU_API_ERROR", strings.TrimSpace(message))
	if err.Message == "" {
		err.Message = "Feishu API request failed"
	}
	err.HTTPStatus = httpStatus
	if code != 0 {
		err.UpstreamCode = strconv.Itoa(code)
	}

	switch code {
	case 190004, 190005, 190010:
		err.Code = "RATE_LIMITED"
		err.Retryable = true
		if err.HTTPStatus == 0 {
			err.HTTPStatus = http.StatusTooManyRequests
		}
	case 191000, 191001:
		err.Code = "CALENDAR_NOT_FOUND"
		if err.HTTPStatus == 0 {
			err.HTTPStatus = http.StatusNotFound
		}
	case 191002, 193002:
		err.Code = "PERMISSION_DENIED"
		if err.HTTPStatus == 0 {
			err.HTTPStatus = http.StatusForbidden
		}
	case 190007:
		err.Code = "BOT_NOT_ENABLED"
		if err.HTTPStatus == 0 {
			err.HTTPStatus = http.StatusForbidden
		}
	}

	if httpStatus == http.StatusUnauthorized || httpStatus == http.StatusForbidden {
		err.Code = "AUTH_FAILED"
	}

	return err
}

func asString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok
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
