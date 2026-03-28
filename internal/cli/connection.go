package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
)

// runConnection 管理默认连接和连接列表。
func runConnection(args []string, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printConnectionHelp(stdout)
		return nil
	}

	cfg, err := store.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: clawrise connection use <connection>")
		}

		name := strings.TrimSpace(args[1])
		connection, ok := cfg.Connections[name]
		if !ok {
			return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
		}

		cfg.Defaults.Platform = connection.Platform
		cfg.Defaults.Profile = name
		cfg.Defaults.Connections[connection.Platform] = name
		if err := store.Save(cfg); err != nil {
			return err
		}

		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"connection": map[string]any{
				"name":     name,
				"platform": connection.Platform,
				"subject":  connection.Subject,
				"method":   connection.Method,
			},
		})
	case "current":
		platform := strings.TrimSpace(cfg.Defaults.Platform)
		var current any
		if platform != "" {
			if name := strings.TrimSpace(cfg.Defaults.Connections[platform]); name != "" {
				current = map[string]any{
					"platform": platform,
					"name":     name,
				}
			}
		}
		return output.WriteJSON(stdout, map[string]any{
			"connection": current,
		})
	case "list":
		names := make([]string, 0, len(cfg.Connections))
		for name := range cfg.Connections {
			names = append(names, name)
		}
		sort.Strings(names)

		items := make([]map[string]any, 0, len(names))
		for _, name := range names {
			connection := cfg.Connections[name]
			items = append(items, map[string]any{
				"name":     name,
				"title":    connection.Title,
				"platform": connection.Platform,
				"subject":  connection.Subject,
				"method":   connection.Method,
			})
		}
		return output.WriteJSON(stdout, map[string]any{
			"connections": items,
		})
	default:
		return fmt.Errorf("unknown connection command: %s", args[0])
	}
}

func printConnectionHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise connection [use|current|list]")
}
