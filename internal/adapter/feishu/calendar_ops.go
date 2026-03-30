package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
)

// ListCalendars lists calendars visible to the current identity.
func (c *Client) ListCalendars(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if pageSize, ok := asInt(input["page_size"]); ok && pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	if pageToken, ok := asString(input["page_token"]); ok && strings.TrimSpace(pageToken) != "" {
		query.Set("page_token", strings.TrimSpace(pageToken))
	}
	if syncToken, ok := asString(input["sync_token"]); ok && strings.TrimSpace(syncToken) != "" {
		query.Set("sync_token", strings.TrimSpace(syncToken))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/calendar/v4/calendars",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode calendar list response")
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0)
	for _, item := range extractFeishuRecordList(data, "calendar_list", "items") {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		items = append(items, normalizeCalendar(record))
	}

	result := map[string]any{
		"items":           items,
		"next_page_token": extractFirstNonEmptyString(data, "page_token", "next_page_token"),
		"sync_token":      extractFirstNonEmptyString(data, "sync_token"),
	}
	if hasMore, ok := asBool(data["has_more"]); ok {
		result["has_more"] = hasMore
	}
	return result, nil
}

// ListCalendarEvents lists events in the given calendar and time window.
func (c *Client) ListCalendarEvents(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	calendarID, ok := asString(input["calendar_id"])
	if !ok || strings.TrimSpace(calendarID) == "" {
		return nil, apperr.New("INVALID_INPUT", "calendar_id is required")
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	query := url.Values{}
	if value, ok := asString(input["start_at_from"]); ok && strings.TrimSpace(value) != "" {
		startTime, appErr := buildFeishuTimeInfo(strings.TrimSpace(value), input["timezone"])
		if appErr != nil {
			return nil, appErr
		}
		query.Set("start_time", startTime.Timestamp)
	}
	if value, ok := asString(input["start_at_to"]); ok && strings.TrimSpace(value) != "" {
		endTime, appErr := buildFeishuTimeInfo(strings.TrimSpace(value), input["timezone"])
		if appErr != nil {
			return nil, appErr
		}
		query.Set("end_time", endTime.Timestamp)
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
		"/open-apis/calendar/v4/calendars/"+url.PathEscape(strings.TrimSpace(calendarID))+"/events",
		query,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode calendar event list response")
	if appErr != nil {
		return nil, appErr
	}

	items := make([]map[string]any, 0)
	for _, item := range extractFeishuRecordList(data, "items", "events") {
		record, ok := asMap(item)
		if !ok {
			continue
		}
		items = append(items, normalizeCalendarEvent(record))
	}

	result := map[string]any{
		"calendar_id":     strings.TrimSpace(calendarID),
		"items":           items,
		"next_page_token": extractFirstNonEmptyString(data, "page_token", "next_page_token"),
	}
	if hasMore, ok := asBool(data["has_more"]); ok {
		result["has_more"] = hasMore
	}
	return result, nil
}

// GetCalendarEvent fetches one calendar event by id.
func (c *Client) GetCalendarEvent(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	calendarID, eventID, appErr := requireCalendarEventIdentity(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodGet,
		"/open-apis/calendar/v4/calendars/"+url.PathEscape(calendarID)+"/events/"+url.PathEscape(eventID),
		nil,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode calendar event get response")
	if appErr != nil {
		return nil, appErr
	}
	event, appErr := extractCalendarEventRecord(data)
	if appErr != nil {
		return nil, appErr
	}
	return normalizeCalendarEvent(event), nil
}

// UpdateCalendarEvent updates one calendar event.
func (c *Client) UpdateCalendarEvent(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	calendarID, eventID, appErr := requireCalendarEventIdentity(input)
	if appErr != nil {
		return nil, appErr
	}

	payload, appErr := buildUpdateCalendarEventPayload(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPatch,
		"/open-apis/calendar/v4/calendars/"+url.PathEscape(calendarID)+"/events/"+url.PathEscape(eventID),
		nil,
		payload,
		"Bearer "+accessToken,
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	data, appErr := decodeFeishuEnvelope(responseBody, "failed to decode calendar event update response")
	if appErr != nil {
		return nil, appErr
	}
	event, appErr := extractCalendarEventRecord(data)
	if appErr != nil {
		return nil, appErr
	}
	return normalizeCalendarEvent(event), nil
}

// DeleteCalendarEvent deletes one calendar event.
func (c *Client) DeleteCalendarEvent(ctx context.Context, profile ExecutionProfile, input map[string]any) (map[string]any, *apperr.AppError) {
	calendarID, eventID, appErr := requireCalendarEventIdentity(input)
	if appErr != nil {
		return nil, appErr
	}

	accessToken, appErr := c.requireFeishuAccessToken(ctx, profile)
	if appErr != nil {
		return nil, appErr
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodDelete,
		"/open-apis/calendar/v4/calendars/"+url.PathEscape(calendarID)+"/events/"+url.PathEscape(eventID),
		nil,
		nil,
		"Bearer "+accessToken,
		nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	if _, appErr := decodeFeishuEnvelope(responseBody, "failed to decode calendar event delete response"); appErr != nil {
		return nil, appErr
	}
	return map[string]any{
		"calendar_id": calendarID,
		"event_id":    eventID,
		"deleted":     true,
	}, nil
}

func requireCalendarEventIdentity(input map[string]any) (string, string, *apperr.AppError) {
	calendarID, ok := asString(input["calendar_id"])
	if !ok || strings.TrimSpace(calendarID) == "" {
		return "", "", apperr.New("INVALID_INPUT", "calendar_id is required")
	}
	eventID, ok := asString(input["event_id"])
	if !ok || strings.TrimSpace(eventID) == "" {
		return "", "", apperr.New("INVALID_INPUT", "event_id is required")
	}
	return strings.TrimSpace(calendarID), strings.TrimSpace(eventID), nil
}

func buildUpdateCalendarEventPayload(input map[string]any) (map[string]any, *apperr.AppError) {
	payload := map[string]any{}
	if summary, ok := asString(input["summary"]); ok && strings.TrimSpace(summary) != "" {
		payload["summary"] = strings.TrimSpace(summary)
	}
	if description, ok := asString(input["description"]); ok && strings.TrimSpace(description) != "" {
		payload["description"] = strings.TrimSpace(description)
	}

	if _, exists := input["start_at"]; exists || input["end_at"] != nil {
		startAt, ok := asString(input["start_at"])
		if !ok || strings.TrimSpace(startAt) == "" {
			return nil, apperr.New("INVALID_INPUT", "start_at is required when updating event time")
		}
		endAt, ok := asString(input["end_at"])
		if !ok || strings.TrimSpace(endAt) == "" {
			return nil, apperr.New("INVALID_INPUT", "end_at is required when updating event time")
		}
		startTime, appErr := buildFeishuTimeInfo(strings.TrimSpace(startAt), input["timezone"])
		if appErr != nil {
			return nil, appErr
		}
		endTime, appErr := buildFeishuTimeInfo(strings.TrimSpace(endAt), input["timezone"])
		if appErr != nil {
			return nil, appErr
		}
		payload["start_time"] = startTime
		payload["end_time"] = endTime
	}

	if _, exists := input["location"]; exists {
		location, appErr := buildLocation(input["location"])
		if appErr != nil {
			return nil, appErr
		}
		if location != nil {
			payload["location"] = location
		}
	}
	if _, exists := input["reminders"]; exists {
		reminders, appErr := buildReminders(input["reminders"])
		if appErr != nil {
			return nil, appErr
		}
		payload["reminders"] = reminders
	}
	if len(payload) == 0 {
		return nil, apperr.New("INVALID_INPUT", "at least one updatable field is required")
	}
	return payload, nil
}

func extractFeishuRecordList(data map[string]any, keys ...string) []any {
	for _, key := range keys {
		if list, ok := asArray(data[key]); ok {
			return list
		}
	}
	return []any{}
}

func extractFirstNonEmptyString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := asString(data[key]); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractCalendarEventRecord(data map[string]any) (map[string]any, *apperr.AppError) {
	for _, key := range []string{"event", "calendar_event"} {
		if event, ok := asMap(data[key]); ok {
			return event, nil
		}
	}
	return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "event is empty in Feishu response")
}

func normalizeCalendarEvent(record map[string]any) map[string]any {
	startTime := feishuTimeInfo{}
	if body, ok := asMap(record["start_time"]); ok {
		startTime = feishuTimeInfo{
			Timestamp: extractFirstNonEmptyString(body, "timestamp"),
			Timezone:  extractFirstNonEmptyString(body, "timezone"),
		}
	}
	endTime := feishuTimeInfo{}
	if body, ok := asMap(record["end_time"]); ok {
		endTime = feishuTimeInfo{
			Timestamp: extractFirstNonEmptyString(body, "timestamp"),
			Timezone:  extractFirstNonEmptyString(body, "timezone"),
		}
	}

	result := map[string]any{
		"event_id":    extractFirstNonEmptyString(record, "event_id"),
		"calendar_id": extractFirstNonEmptyString(record, "organizer_calendar_id", "calendar_id"),
		"summary":     extractFirstNonEmptyString(record, "summary"),
		"start_at":    formatFeishuTimeInfo(startTime),
		"end_at":      formatFeishuTimeInfo(endTime),
		"html_url":    extractFirstNonEmptyString(record, "app_link", "html_link"),
		"raw":         cloneFeishuMap(record),
	}
	if description, ok := asString(record["description"]); ok && strings.TrimSpace(description) != "" {
		result["description"] = strings.TrimSpace(description)
	}
	if location, ok := asMap(record["location"]); ok && len(location) > 0 {
		result["location"] = cloneFeishuMap(location)
	}
	return result
}

func normalizeCalendar(record map[string]any) map[string]any {
	result := map[string]any{
		"calendar_id": extractFirstNonEmptyString(record, "calendar_id"),
		"summary":     extractFirstNonEmptyString(record, "summary"),
		"raw":         cloneFeishuMap(record),
	}
	for _, key := range []string{"description", "permissions", "type", "summary_alias", "role"} {
		if value, ok := asString(record[key]); ok && strings.TrimSpace(value) != "" {
			result[key] = strings.TrimSpace(value)
		}
	}
	if color, ok := asInt(record["color"]); ok {
		result["color"] = color
	}
	if isDeleted, ok := asBool(record["is_deleted"]); ok {
		result["is_deleted"] = isDeleted
	}
	if isThirdParty, ok := asBool(record["is_third_party"]); ok {
		result["is_third_party"] = isThirdParty
	}
	return result
}

func decodeCalendarEventResponse(responseBody []byte) (*createCalendarEventResponse, *apperr.AppError) {
	var response createCalendarEventResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode calendar event response: %v", err))
	}
	if response.Code != 0 {
		return nil, normalizeFeishuError(response.Code, response.Msg, 0)
	}
	if strings.TrimSpace(response.Data.Event.EventID) == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "event_id is empty in Feishu response")
	}
	return &response, nil
}
