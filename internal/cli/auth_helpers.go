package cli

import (
	"strings"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/authflow"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

func openCLISecretStore(cfg *config.Config, store *config.Store) (secretstore.Store, error) {
	backend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
	if backend == "" {
		backend = "encrypted_file"
	}
	return secretstore.Open(secretstore.Options{
		ConfigPath:      store.Path(),
		Backend:         backend,
		FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
	})
}

func openCLISessionStore(cfg *config.Config, store *config.Store) (authcache.Store, error) {
	backend := strings.TrimSpace(cfg.Auth.SessionStore.Backend)
	if backend == "" {
		backend = "file"
	}
	return authcache.OpenStore(store.Path(), backend)
}

func openCLIAuthFlowStore(cfg *config.Config, store *config.Store) (authflow.Store, error) {
	backend := strings.TrimSpace(cfg.Auth.AuthFlowStore.Backend)
	if backend == "" {
		backend = "file"
	}
	return authflow.OpenStore(store.Path(), backend)
}
