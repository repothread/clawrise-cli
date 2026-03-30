package main

import (
	"context"
	"fmt"
	"os"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func main() {
	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.echo",
		Platform:        "demo",
		Mutating:        false,
		DefaultTimeout:  1000,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary:         "Echo one demo page.",
			DryRunSupported: true,
			Input: adapter.InputSpec{
				Required: []string{"message"},
				Sample: map[string]any{
					"message": "hello",
				},
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{
				"message": "echoed",
			}, nil
		},
	})

	runtime := pluginruntime.NewRegistryRuntime(
		"demo",
		"0.1.0",
		[]string{"demo"},
		registry,
		pluginruntime.CatalogFromRegistry(registry),
	)
	if err := pluginruntime.ServeRuntime(runtime); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
