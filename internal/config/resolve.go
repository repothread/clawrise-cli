package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

// ResolveSecret supports both direct values and env-prefixed references.
func ResolveSecret(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	switch {
	case strings.HasPrefix(raw, "env:"):
		envName := strings.TrimSpace(strings.TrimPrefix(raw, "env:"))
		if envName == "" {
			return "", fmt.Errorf("invalid environment variable reference")
		}

		value, ok := os.LookupEnv(envName)
		if !ok || strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("environment variable %s is not set", envName)
		}
		return value, nil
	case strings.HasPrefix(raw, "secret:"):
		connectionName, fieldName, err := parseSecretReference(raw)
		if err != nil {
			return "", err
		}

		configPath, err := DefaultPath()
		if err != nil {
			return "", err
		}

		backend := "auto"
		fallbackBackend := ""
		cfgStore := NewStore(configPath)
		if cfg, loadErr := cfgStore.Load(); loadErr == nil {
			backend = strings.TrimSpace(cfg.Auth.SecretStore.Backend)
			fallbackBackend = strings.TrimSpace(cfg.Auth.SecretStore.FallbackBackend)
		}

		store, err := secretstore.Open(secretstore.Options{
			ConfigPath:      configPath,
			Backend:         backend,
			FallbackBackend: fallbackBackend,
		})
		if err != nil {
			return "", err
		}
		value, err := store.Get(connectionName, fieldName)
		if err != nil {
			if err == secretstore.ErrSecretNotFound {
				return "", fmt.Errorf("secret %s/%s is not set", connectionName, fieldName)
			}
			return "", err
		}
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("secret %s/%s is empty", connectionName, fieldName)
		}
		return value, nil
	default:
		return raw, nil
	}
}

// ValidateGrant 校验执行所需的静态授权配置是否齐全。
// 对交互式 OAuth 连接，这里只校验 client_id/client_secret 这类静态材料，
// 不把首次授权后才会产生的 refresh_token 当成前置阻塞项。
func ValidateGrant(profile Profile) error {
	if err := ValidateResolvedAuthShape(profile); err != nil {
		return err
	}

	requiredFields, err := requiredAuthFieldSpecs(profile)
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

// ValidateAccountShape validates account-level required structure.
func ValidateAccountShape(accountName string, account Account) error {
	_ = accountName
	if strings.TrimSpace(account.Platform) == "" {
		return fmt.Errorf("missing platform")
	}
	if strings.TrimSpace(account.Subject) == "" {
		return fmt.Errorf("missing subject")
	}
	if strings.TrimSpace(account.Auth.Method) == "" {
		return fmt.Errorf("missing auth method")
	}
	for field, ref := range account.Auth.SecretRefs {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("secret_refs field must not be empty")
		}
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("secret_refs.%s must not be empty", field)
		}
	}
	return nil
}

// ValidateAccount validates static auth materials required before execution.
func ValidateAccount(accountName string, account Account) error {
	return ValidateAccountShape(accountName, account)
}

func parseSecretReference(raw string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(raw, "secret:")), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid secret reference: %s", raw)
	}
	connectionName := strings.TrimSpace(parts[0])
	fieldName := strings.TrimSpace(parts[1])
	if connectionName == "" || fieldName == "" {
		return "", "", fmt.Errorf("invalid secret reference: %s", raw)
	}
	return connectionName, fieldName, nil
}
