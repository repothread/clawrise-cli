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
	if err := ValidateGrantShape(profile); err != nil {
		return err
	}

	requiredFields, err := requiredGrantFieldSpecs(profile)
	if err != nil {
		return err
	}
	for _, field := range requiredFields {
		if !field.Secret {
			continue
		}
		if _, err := ResolveSecret(field.Value(profile.Grant)); err != nil {
			return fmt.Errorf("missing %s: %w", field.Name, err)
		}
	}
	return nil
}
