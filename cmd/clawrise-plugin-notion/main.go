package main

import (
	"fmt"
	"os"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func main() {
	client, err := notionadapter.NewClient(notionadapter.Options{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	registry := adapter.NewRegistry()
	notionadapter.RegisterOperations(registry, client)

	runtime := pluginruntime.NewRegistryRuntime(
		"notion",
		"0.1.0",
		[]string{"notion"},
		registry,
		pluginruntime.FilterCatalogByPrefix(speccatalog.All(), "notion."),
	)
	if err := pluginruntime.ServeRuntime(runtime); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
