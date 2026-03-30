package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

var errFeishuAuthorizationRequired = errors.New("feishu interactive authorization is required")

func (c *Client) requireFeishuAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	if profile.Grant.Type == "resolved_access_token" && strings.TrimSpace(profile.Grant.AccessToken) != "" {
		return strings.TrimSpace(profile.Grant.AccessToken), nil
	}
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

	cachedSession, ok := c.loadCachedSession(ctx, profile)
	if ok && cachedSession.UsableAt(c.now(), authcache.DefaultRefreshSkew) {
		return strings.TrimSpace(cachedSession.AccessToken), nil
	}

	refreshedSession, appErr := c.refreshUserAccessToken(ctx, profile, cachedSession)
	if appErr == nil {
		c.saveCachedSession(ctx, profile, *refreshedSession)
		return strings.TrimSpace(refreshedSession.AccessToken), nil
	}

	if token, err := config.ResolveSecret(profile.Grant.AccessToken); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), nil
	}
	return "", appErr
}

func (c *Client) refreshUserAccessToken(ctx context.Context, profile config.Profile, currentSession *authcache.Session) (*authcache.Session, *apperr.AppError) {
	clientID, err := config.ResolveSecret(profile.Grant.ClientID)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_id: %v", err))
	}
	clientSecret, err := config.ResolveSecret(profile.Grant.ClientSecret)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_secret: %v", err))
	}
	refreshToken, err := resolveFeishuRefreshToken(profile, currentSession)
	if err != nil {
		if errors.Is(err, errFeishuAuthorizationRequired) {
			return nil, buildFeishuAuthorizationRequiredError(ctx)
		}
		return nil, apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing refresh_token: %v", err))
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
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Feishu user token response: %v", err))
	}

	if code, ok := asInt(response["code"]); ok && code != 0 {
		message, _ := asString(response["msg"])
		return nil, normalizeFeishuError(code, message, 0)
	}

	payload := response
	if data, ok := asMap(response["data"]); ok {
		payload = data
	}

	accessToken := extractFirstNonEmptyString(payload, "access_token")
	if accessToken == "" {
		accessToken = extractFirstNonEmptyString(response, "access_token")
	}
	if accessToken == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "access_token is empty in Feishu user token response")
	}

	nextRefreshToken := extractFirstNonEmptyString(payload, "refresh_token")
	if nextRefreshToken == "" {
		nextRefreshToken = extractFirstNonEmptyString(response, "refresh_token")
	}
	if nextRefreshToken == "" {
		nextRefreshToken = strings.TrimSpace(refreshToken)
	}

	tokenType := extractFirstNonEmptyString(payload, "token_type")
	if tokenType == "" {
		tokenType = extractFirstNonEmptyString(response, "token_type")
	}

	profileName := adapter.ProfileNameFromContext(ctx)
	session := buildOAuthSession(c.now(), profileName, profile, accessToken, nextRefreshToken, tokenType, extractFeishuExpiresInSeconds(response, payload))
	return &session, nil
}

func (c *Client) exchangeAuthorizationCode(ctx context.Context, profile config.Profile, code string, redirectURI string) (*authcache.Session, *apperr.AppError) {
	clientID, err := config.ResolveSecret(profile.Grant.ClientID)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_id: %v", err))
	}
	clientSecret, err := config.ResolveSecret(profile.Grant.ClientSecret)
	if err != nil {
		return nil, apperr.New("INVALID_AUTH_CONFIG", fmt.Sprintf("missing client_secret: %v", err))
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, apperr.New("INVALID_INPUT", "authorization code is empty")
	}

	body := map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	}
	if strings.TrimSpace(redirectURI) != "" {
		body["redirect_uri"] = strings.TrimSpace(redirectURI)
	}

	responseBody, appErr := c.doJSONRequest(
		ctx,
		http.MethodPost,
		"/open-apis/authen/v2/oauth/token",
		nil,
		body,
		"",
		map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
	)
	if appErr != nil {
		return nil, appErr
	}

	var response map[string]any
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", fmt.Sprintf("failed to decode Feishu auth code response: %v", err))
	}

	if codeValue, ok := asInt(response["code"]); ok && codeValue != 0 {
		message, _ := asString(response["msg"])
		return nil, normalizeFeishuError(codeValue, message, 0)
	}

	payload := response
	if data, ok := asMap(response["data"]); ok {
		payload = data
	}

	accessToken := extractFirstNonEmptyString(payload, "access_token")
	if accessToken == "" {
		accessToken = extractFirstNonEmptyString(response, "access_token")
	}
	if accessToken == "" {
		return nil, apperr.New("UPSTREAM_INVALID_RESPONSE", "access_token is empty in Feishu auth code response")
	}

	refreshToken := extractFirstNonEmptyString(payload, "refresh_token")
	if refreshToken == "" {
		refreshToken = extractFirstNonEmptyString(response, "refresh_token")
	}

	tokenType := extractFirstNonEmptyString(payload, "token_type")
	if tokenType == "" {
		tokenType = extractFirstNonEmptyString(response, "token_type")
	}

	profileName := adapter.ProfileNameFromContext(ctx)
	session := buildOAuthSession(c.now(), profileName, profile, accessToken, refreshToken, tokenType, extractFeishuExpiresInSeconds(response, payload))
	return &session, nil
}

func resolveFeishuRefreshToken(profile config.Profile, currentSession *authcache.Session) (string, error) {
	if currentSession != nil && strings.TrimSpace(currentSession.RefreshToken) != "" {
		return strings.TrimSpace(currentSession.RefreshToken), nil
	}
	raw := strings.TrimSpace(profile.Grant.RefreshToken)
	if raw == "" {
		return "", errFeishuAuthorizationRequired
	}

	refreshToken, err := config.ResolveSecret(raw)
	if err != nil {
		if shouldTreatOAuthSecretAsPending(raw, err) {
			return "", errFeishuAuthorizationRequired
		}
		return "", err
	}
	if strings.TrimSpace(refreshToken) == "" {
		return "", errFeishuAuthorizationRequired
	}
	return strings.TrimSpace(refreshToken), nil
}

func extractFeishuExpiresInSeconds(response map[string]any, payload map[string]any) int {
	if value, ok := asInt(payload["expires_in"]); ok && value > 0 {
		return value
	}
	if value, ok := asInt(response["expires_in"]); ok && value > 0 {
		return value
	}
	if value, ok := asInt(payload["expire"]); ok && value > 0 {
		return value
	}
	if value, ok := asInt(response["expire"]); ok && value > 0 {
		return value
	}
	return 0
}

func buildFeishuAuthorizationRequiredError(ctx context.Context) *apperr.AppError {
	profileName := strings.TrimSpace(adapter.ProfileNameFromContext(ctx))
	if profileName == "" {
		return apperr.New("AUTHORIZATION_REQUIRED", "interactive authorization has not been completed; run `clawrise auth login <account>` first")
	}
	return apperr.New("AUTHORIZATION_REQUIRED", fmt.Sprintf("interactive authorization has not been completed for account %s; run `clawrise auth login %s` first", profileName, profileName))
}

func shouldTreatOAuthSecretAsPending(raw string, err error) bool {
	if err == nil {
		return false
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if strings.HasPrefix(raw, "secret:") || strings.HasPrefix(raw, "env:") {
		message := err.Error()
		return strings.Contains(message, "is not set") || strings.Contains(message, "is empty")
	}
	return false
}
