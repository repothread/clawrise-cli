package main

import (
	"fmt"
	"os"

	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func main() {
	if err := pluginruntime.ServeAuthLauncherRuntime(pluginruntime.NewSystemAuthLauncherRuntime()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
