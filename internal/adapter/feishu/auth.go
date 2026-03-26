package feishu

import (
	"context"

	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func (c *Client) requireBotAccessToken(ctx context.Context, profile config.Profile) (string, *apperr.AppError) {
	if profile.Subject != "bot" {
		return "", apperr.New("SUBJECT_NOT_ALLOWED", "this Feishu operation currently requires a bot profile")
	}
	if profile.Grant.Type != "client_credentials" {
		return "", apperr.New("UNSUPPORTED_GRANT", "this Feishu operation currently supports only client_credentials")
	}
	return c.fetchTenantAccessToken(ctx, profile)
}
