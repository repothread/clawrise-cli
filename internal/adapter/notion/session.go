package notion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

var errNotionAuthorizationRequired = errors.New("notion interactive authorization is required")

func newDefaultSessionStore() authcache.Store {
	configPath, err := config.DefaultPath()
	if err != nil {
		return nil
	}
	return authcache.NewFileStore(configPath)
}

func (c *Client) loadCachedSession(ctx context.Context, profile ExecutionProfile) (*authcache.Session, bool) {
	profile = normalizeExecutionProfile(profile)
	accountName := adapter.AccountNameFromContext(ctx)
	if accountName == "" || c.sessionStore == nil {
		return nil, false
	}

	session, err := c.sessionStore.Load(accountName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false
		}
		return nil, false
	}
	if !sessionMatchesProfile(session, profile) {
		return nil, false
	}
	return session, true
}

func (c *Client) saveCachedSession(ctx context.Context, profile ExecutionProfile, session authcache.Session) {
	profile = normalizeExecutionProfile(profile)
	accountName := adapter.AccountNameFromContext(ctx)
	if accountName == "" || c.sessionStore == nil {
		return
	}

	session.AccountName = accountName
	session.Platform = profile.Platform
	session.Subject = profile.Subject
	session.GrantType = profile.Method
	if strings.TrimSpace(session.TokenType) == "" {
		session.TokenType = "Bearer"
	}
	_ = c.sessionStore.Save(session)
}

func buildOAuthSession(now time.Time, accountName string, profile ExecutionProfile, accessToken string, refreshToken string, tokenType string, expiresInSeconds int) authcache.Session {
	profile = normalizeExecutionProfile(profile)
	session := authcache.Session{
		Version:      authcache.SessionVersion,
		AccountName:  accountName,
		Platform:     profile.Platform,
		Subject:      profile.Subject,
		GrantType:    profile.Method,
		AccessToken:  strings.TrimSpace(accessToken),
		RefreshToken: strings.TrimSpace(refreshToken),
		TokenType:    normalizeTokenType(tokenType),
	}
	if expiresInSeconds > 0 {
		expiresAt := now.UTC().Add(time.Duration(expiresInSeconds) * time.Second)
		session.ExpiresAt = &expiresAt
	}
	return session
}

func resolveNotionRefreshToken(profile ExecutionProfile, currentSession *authcache.Session) (string, error) {
	profile = normalizeExecutionProfile(profile)
	if currentSession != nil && strings.TrimSpace(currentSession.RefreshToken) != "" {
		return strings.TrimSpace(currentSession.RefreshToken), nil
	}
	raw := strings.TrimSpace(profile.Grant.RefreshToken)
	if raw == "" {
		return "", errNotionAuthorizationRequired
	}

	refreshToken, err := config.ResolveSecret(raw)
	if err != nil {
		if shouldTreatOAuthSecretAsPending(raw, err) {
			return "", errNotionAuthorizationRequired
		}
		return "", err
	}
	if strings.TrimSpace(refreshToken) == "" {
		return "", errNotionAuthorizationRequired
	}
	return strings.TrimSpace(refreshToken), nil
}

func sessionMatchesProfile(session *authcache.Session, profile ExecutionProfile) bool {
	profile = normalizeExecutionProfile(profile)
	if session == nil {
		return false
	}
	legacyGrantType := strings.TrimSpace(profile.Grant.Type)
	if strings.TrimSpace(session.GrantType) == legacyGrantType && legacyGrantType != "" {
		return strings.TrimSpace(session.Platform) == strings.TrimSpace(profile.Platform) &&
			strings.TrimSpace(session.Subject) == strings.TrimSpace(profile.Subject)
	}
	return strings.TrimSpace(session.Platform) == strings.TrimSpace(profile.Platform) &&
		strings.TrimSpace(session.Subject) == strings.TrimSpace(profile.Subject) &&
		strings.TrimSpace(session.GrantType) == strings.TrimSpace(profile.Method)
}

func normalizeTokenType(tokenType string) string {
	tokenType = strings.TrimSpace(tokenType)
	if tokenType == "" {
		return "Bearer"
	}
	if strings.EqualFold(tokenType, "bearer") {
		return "Bearer"
	}
	return tokenType
}

// RefreshSession 强制刷新指定 profile 的 OAuth session，并写回本地 cache。
func (c *Client) RefreshSession(ctx context.Context, accountName string, profile ExecutionProfile) (*authcache.Session, *apperr.AppError) {
	profile = normalizeExecutionProfile(profile)
	if profile.Subject != "integration" {
		return nil, apperr.New("SUBJECT_NOT_ALLOWED", "this Notion session refresh currently requires an integration profile")
	}
	if profile.Method != "notion.oauth_public" {
		return nil, apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Notion session refresh currently supports only notion.oauth_public, got %s", profile.Method))
	}

	ctx = adapter.WithAccountName(ctx, accountName)
	currentSession, _ := c.loadCachedSession(ctx, profile)

	session, appErr := c.refreshAccessToken(ctx, profile, currentSession)
	if appErr != nil {
		return nil, appErr
	}
	c.saveCachedSession(ctx, profile, *session)
	return session, nil
}

// ExchangeAuthorizationCode 使用授权码换取新的 OAuth session，并写回本地 cache。
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, accountName string, profile ExecutionProfile, code string, redirectURI string) (*authcache.Session, *apperr.AppError) {
	profile = normalizeExecutionProfile(profile)
	if profile.Subject != "integration" {
		return nil, apperr.New("SUBJECT_NOT_ALLOWED", "this Notion auth code exchange currently requires an integration profile")
	}
	if profile.Method != "notion.oauth_public" {
		return nil, apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Notion auth code exchange currently supports only notion.oauth_public, got %s", profile.Method))
	}

	ctx = adapter.WithAccountName(ctx, accountName)
	session, appErr := c.exchangeAuthorizationCode(ctx, profile, code, redirectURI)
	if appErr != nil {
		return nil, appErr
	}
	c.saveCachedSession(ctx, profile, *session)
	return session, nil
}
