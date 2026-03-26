package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/clawrise/clawrise-cli/internal/cli"
)

func main() {
	err := cli.Run(os.Args[1:], cli.Dependencies{
		Version: "0.1.0-dev",
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})

	if err != nil {
		var exitErr cli.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
