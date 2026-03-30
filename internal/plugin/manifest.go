package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ManifestFileName defines the required plugin manifest file name.
const ManifestFileName = "plugin.json"

const (
	// ManifestKindProvider marks a provider plugin.
	ManifestKindProvider = "provider"
	// ManifestKindAuthLauncher marks an auth-launcher plugin.
	ManifestKindAuthLauncher = "auth_launcher"
)

// Manifest describes one installed plugin package.
type Manifest struct {
	SchemaVersion   int           `json:"schema_version"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Kind            string        `json:"kind"`
	ProtocolVersion int           `json:"protocol_version"`
	Platforms       []string      `json:"platforms"`
	Entry           ManifestEntry `json:"entry"`
	CatalogPath     string        `json:"catalog_path,omitempty"`
	MinCoreVersion  string        `json:"min_core_version,omitempty"`
	RootDir         string        `json:"-"`
}

// ManifestEntry describes how to start one plugin executable.
type ManifestEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
}

// LoadManifest reads and validates one plugin manifest from disk.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to read plugin manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("failed to decode plugin manifest: %w", err)
	}
	manifest.RootDir = filepath.Dir(path)
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Validate validates one plugin manifest.
func (m Manifest) Validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported plugin manifest schema_version: %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("plugin manifest name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("plugin manifest version is required")
	}
	kind := strings.TrimSpace(m.Kind)
	switch kind {
	case ManifestKindProvider:
		if len(m.Platforms) == 0 {
			return fmt.Errorf("plugin manifest platforms must not be empty")
		}
	case ManifestKindAuthLauncher:
		// Launcher plugins may remain platform-agnostic and declare only supported action types.
	default:
		return fmt.Errorf("plugin manifest kind must be %s or %s", ManifestKindProvider, ManifestKindAuthLauncher)
	}
	if m.ProtocolVersion <= 0 {
		return fmt.Errorf("plugin manifest protocol_version must be positive")
	}
	if strings.TrimSpace(m.Entry.Type) != "binary" {
		return fmt.Errorf("plugin manifest entry.type must be binary")
	}
	if len(m.Entry.Command) == 0 {
		return fmt.Errorf("plugin manifest entry.command must not be empty")
	}
	return nil
}

// ResolveCommand resolves the manifest command against the plugin root directory.
func (m Manifest) ResolveCommand() []string {
	command := append([]string(nil), m.Entry.Command...)
	if len(command) == 0 {
		return command
	}
	if filepath.IsAbs(command[0]) {
		return command
	}
	command[0] = filepath.Join(m.RootDir, command[0])
	return command
}

// ResolveCatalogPath resolves the optional catalog path against the plugin root directory.
func (m Manifest) ResolveCatalogPath() string {
	if strings.TrimSpace(m.CatalogPath) == "" {
		return ""
	}
	if filepath.IsAbs(m.CatalogPath) {
		return m.CatalogPath
	}
	return filepath.Join(m.RootDir, m.CatalogPath)
}
