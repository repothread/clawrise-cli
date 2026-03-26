package config

import (
	"fmt"
	"os"
	"strings"
)

// ResolveSecret supports both direct values and env-prefixed references.
func ResolveSecret(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	if !strings.HasPrefix(raw, "env:") {
		return raw, nil
	}

	envName := strings.TrimSpace(strings.TrimPrefix(raw, "env:"))
	if envName == "" {
		return "", fmt.Errorf("invalid environment variable reference")
	}

	value, ok := os.LookupEnv(envName)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("environment variable %s is not set", envName)
	}
	return value, nil
}

// ValidateGrant validates grant completeness without exposing secret values.
func ValidateGrant(profile Profile) error {
	switch profile.Grant.Type {
	case "client_credentials":
		if _, err := ResolveSecret(profile.Grant.AppID); err != nil {
			return fmt.Errorf("missing app_id: %w", err)
		}
		if _, err := ResolveSecret(profile.Grant.AppSecret); err != nil {
			return fmt.Errorf("missing app_secret: %w", err)
		}
	case "static_token":
		if _, err := ResolveSecret(profile.Grant.Token); err != nil {
			return fmt.Errorf("missing token: %w", err)
		}
	case "oauth_user":
		if _, err := ResolveSecret(profile.Grant.ClientID); err != nil {
			return fmt.Errorf("missing client_id: %w", err)
		}
		if _, err := ResolveSecret(profile.Grant.ClientSecret); err != nil {
			return fmt.Errorf("missing client_secret: %w", err)
		}
		if _, err := ResolveSecret(profile.Grant.RefreshToken); err != nil {
			return fmt.Errorf("missing refresh_token: %w", err)
		}
	case "oauth_refreshable":
		if _, err := ResolveSecret(profile.Grant.ClientID); err != nil {
			return fmt.Errorf("missing client_id: %w", err)
		}
		if _, err := ResolveSecret(profile.Grant.ClientSecret); err != nil {
			return fmt.Errorf("missing client_secret: %w", err)
		}
		if _, err := ResolveSecret(profile.Grant.RefreshToken); err != nil {
			return fmt.Errorf("missing refresh_token: %w", err)
		}
	default:
		return fmt.Errorf("unsupported grant type: %s", profile.Grant.Type)
	}
	return nil
}
