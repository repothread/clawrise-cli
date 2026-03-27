package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func (c *Client) requireFeishuAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	switch profile.Subject {
	case "bot":
		return c.requireBotAccessToken(ctx, profile)
	case "user":
		return c.requireUserAccessToken(ctx, profile)
	default:
		return "", apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu operation currently supports only bot or user profiles")
	}
}

func (c *Client) requireBotAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	if profile.Subject != "bot" {
		return "", apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu operation currently requires a bot profile")
	}
	if profile.Grant.Type != "client_credentials" {
		return "", apperr.New("UNSUPPORTED_GRANT", "this Feishu operation currently supports only client_credentials")
	}
	return c.fetchTenantAccessToken(ctx, profile)
}

func (c *Client) requireUserAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	if profile.Subject != "user" {
		return "", apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu operation currently requires a user profile")
	}
	if profile.Grant.Type != "oauth_user" {
		return "", apperr.New("UNSUPPORTED_GRANT", "this Feishu operation currently supports only oauth_user for user subject")
	}

	if token, err := config.ResolveSecret(profile.Grant.AccessToken); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), nil
	}
	return c.refreshUserAccessToken(ctx, profile)
}

func (c *Client) refreshUserAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	clientID, err := config.ResolveSecret(profile.Grant.ClientID)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_id: %v", err))
	}
	clientSecret, err := config.ResolveSecret(profile.Grant.ClientSecret)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_secret: %v", err))
	}
	refreshToken, err := config.ResolveSecret(profile.Grant.RefreshToken)
	if err != nil {
		return "", apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing refresh_token: %v", err))
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/authen/v2/oauth/token",
		nil,
		map[string]any{
			"grant_type":    "refresh_token",
			"client_id":     clientID,
			"client_secret": clientSecret,
			"refresh_token": refreshToken,
		},
		"",
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return "", appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Feishu user token response: %v", err))
	}

	if code, ok := asInt(response["code"]); ok && code != 0 {
		message, _ := asString(response["msg"])
		return "", normalizeFeishuError(code, message, 0)
	}

	if token, ok := asString(response["access_token"]); ok && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), nil
	}
	if data, ok := asMap(response["data"]); ok {
		if token, ok := asString(data["access_token"]); ok && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token), nil
		}
	}
	return "", apperr.New("UPSTREAM_INVALID_RESPONSE", "access_token is empty in Feishu user token response")
}
