package feishu

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
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
	profileName := adapter.ProfileNameFromContext(ctx)
	if profileName == "" || c.sessionStore == nil {
		return nil, false
	}

	session, err := c.sessionStore.Load(profileName)
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
	profileName := adapter.ProfileNameFromContext(ctx)
	if profileName == "" || c.sessionStore == nil {
		return
	}

	session.ProfileName = profileName
	session.Platform = profile.Platform
	session.Subject = profile.Subject
	session.GrantType = profile.Grant.Type
	if strings.TrimSpace(session.TokenType) == "" {
		session.TokenType = "Bearer"
	}
	_ = c.sessionStore.Save(session)
}

func buildOAuthSession(now time.Time, profileName string, profile config.Profile, accessToken string, refreshToken string, tokenType string, expiresInSeconds int) authcache.Session {
	session := authcache.Session{
		Version:      authcache.SessionVersion,
		ProfileName:  profileName,
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
