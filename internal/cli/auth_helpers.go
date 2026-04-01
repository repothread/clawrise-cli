package cli

import (
	"strings"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/authflow"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

func openCLISecretStore(cfg *config.Config, store *config.Store) (secretstore.Store, error) {
	binding := config.ResolveStorageBinding(cfg, "secret_store")
	backend := strings.TrimSpace(binding.Backend)
	if backend == "" {
		backend = "encrypted_file"
	}
	return secretstore.Open(secretstore.Options{
		ConfigPath:      store.Path(),
		Backend:         backend,
		FallbackBackend: binding.FallbackBackend,
		Plugin:          binding.Plugin,
		EnabledPlugins:  config.ResolveEnabledPlugins(cfg),
	})
}

func openCLISessionStore(cfg *config.Config, store *config.Store) (authcache.Store, error) {
	binding := config.ResolveStorageBinding(cfg, "session_store")
	backend := strings.TrimSpace(binding.Backend)
	if backend == "" {
		backend = "file"
	}
	return authcache.OpenStoreWithOptions(authcache.StoreOptions{
		ConfigPath:     store.Path(),
		Backend:        backend,
		Plugin:         binding.Plugin,
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
}

func openCLIAuthFlowStore(cfg *config.Config, store *config.Store) (authflow.Store, error) {
	binding := config.ResolveStorageBinding(cfg, "authflow_store")
	backend := strings.TrimSpace(binding.Backend)
	if backend == "" {
		backend = "file"
	}
	return authflow.OpenStoreWithOptions(authflow.StoreOptions{
		ConfigPath:     store.Path(),
		Backend:        backend,
		Plugin:         binding.Plugin,
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
}
