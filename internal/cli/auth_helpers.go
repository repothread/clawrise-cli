package cli

import (
	"fmt"
	"os/exec"
	goRuntime "runtime"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

var openAuthURL = func(rawURL string) error {
	return openURLInBrowser(rawURL)
}

func openCLISecretStore(cfg *config.Config, store *config.Store) (secretstore.Store, error) {
	backend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
	if backend == "" {
		backend = "auto"
	}
	return secretstore.Open(secretstore.Options{
		ConfigPath:      store.Path(),
		Backend:         backend,
		FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
	})
}

func openURLInBrowser(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("authorization url is empty")
	}

	var command *exec.Cmd
	switch goRuntime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("failed to open authorization url: %s", message)
	}
	return nil
}
