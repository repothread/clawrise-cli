package main

import (
	"fmt"
	"os"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func main() {
	client, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	registry := adapter.NewRegistry()
	feishuadapter.RegisterOperations(registry, client)

	runtime := pluginruntime.NewRegistryRuntimeWithOptions(
		"feishu",
		"0.1.0",
		[]string{"feishu"},
		registry,
		pluginruntime.CatalogFromRegistry(registry),
		pluginruntime.RegistryRuntimeOptions{
			AuthProvider: feishuadapter.NewAuthProvider(client),
		},
	)
	if err := pluginruntime.ServeRuntime(runtime); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
