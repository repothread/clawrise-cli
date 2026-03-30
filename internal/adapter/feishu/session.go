package feishu

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func newDefaultSessionStore() authcache.Store {
	configPath, err := config.DefaultPath()
	if err != nil {
		return nil
	}
	return authcache.NewFileStore(configPath)
}

func (c *Client) loadCachedSession(ctx context.Context, profile config.Profile) (*authcache.Session, bool) {
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

func (c *Client) saveCachedSession(ctx context.Context, profile config.Profile, session authcache.Session) {
	accountName := adapter.AccountNameFromContext(ctx)
	if accountName == "" || c.sessionStore == nil {
		return
	}

	session.AccountName = accountName
	session.Platform = profile.Platform
	session.Subject = profile.Subject
	session.GrantType = profile.Grant.Type
	if strings.TrimSpace(session.TokenType) == "" {
		session.TokenType = "Bearer"
	}
	_ = c.sessionStore.Save(session)
}

func buildOAuthSession(now time.Time, accountName string, profile config.Profile, accessToken string, refreshToken string, tokenType string, expiresInSeconds int) authcache.Session {
	session := authcache.Session{
		Version:      authcache.SessionVersion,
		AccountName:  accountName,
		Platform:     profile.Platform,
		Subject:      profile.Subject,
		GrantType:    profile.Grant.Type,
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

func sessionMatchesProfile(session *authcache.Session, profile config.Profile) bool {
	if session == nil {
		return false
	}
	return strings.TrimSpace(session.Platform) == strings.TrimSpace(profile.Platform) &&
		strings.TrimSpace(session.Subject) == strings.TrimSpace(profile.Subject) &&
		strings.TrimSpace(session.GrantType) == strings.TrimSpace(profile.Grant.Type)
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
func (c *Client) RefreshSession(ctx context.Context, accountName string, profile config.Profile) (*authcache.Session, *apperr.AppError) {
	if profile.Subject != "user" {
		return nil, apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu session refresh currently requires a user profile")
	}
	if profile.Grant.Type != "oauth_user" {
		return nil, apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Feishu session refresh currently supports only oauth_user, got %s", profile.Grant.Type))
	}

	ctx = adapter.WithAccountName(ctx, accountName)
	currentSession, _ := c.loadCachedSession(ctx, profile)

	session, appErr := c.refreshUserAccessToken(ctx, profile, currentSession)
	if appErr != nil {
		return nil, appErr
	}
	c.saveCachedSession(ctx, profile, *session)
	return session, nil
}

// ExchangeAuthorizationCode 使用授权码换取新的 OAuth session，并写回本地 cache。
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, accountName string, profile config.Profile, code string, redirectURI string) (*authcache.Session, *apperr.AppError) {
	if profile.Subject != "user" {
		return nil, apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu auth code exchange currently requires a user profile")
	}
	if profile.Grant.Type != "oauth_user" {
		return nil, apperr.New("UNSUPPORTED_GRANT", fmt.Sprintf("this Feishu auth code exchange currently supports only oauth_user, got %s", profile.Grant.Type))
	}

	ctx = adapter.WithAccountName(ctx, accountName)
	session, appErr := c.exchangeAuthorizationCode(ctx, profile, code, redirectURI)
	if appErr != nil {
		return nil, appErr
	}
	c.saveCachedSession(ctx, profile, *session)
	return session, nil
}
