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
	// ManifestKindStorageBackend marks a storage-backend plugin.
	ManifestKindStorageBackend = "storage_backend"
)

// Manifest describes one installed plugin package.
type Manifest struct {
	SchemaVersion   int                     `json:"schema_version"`
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	Kind            string                  `json:"kind"`
	ProtocolVersion int                     `json:"protocol_version"`
	Platforms       []string                `json:"platforms"`
	Entry           ManifestEntry           `json:"entry"`
	CatalogPath     string                  `json:"catalog_path,omitempty"`
	StorageBackend  *StorageBackendManifest `json:"storage_backend,omitempty"`
	Capabilities    []CapabilityDescriptor  `json:"capabilities,omitempty"`
	MinCoreVersion  string                  `json:"min_core_version,omitempty"`
	RootDir         string                  `json:"-"`
}

// ManifestEntry describes how to start one plugin executable.
type ManifestEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
}

// StorageBackendManifest describes one external storage backend plugin.
type StorageBackendManifest struct {
	Target      string `json:"target"`
	Backend     string `json:"backend"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
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
func (m *Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersionV1 && m.SchemaVersion != ManifestSchemaVersionV2 {
		return fmt.Errorf("unsupported plugin manifest schema_version: %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("plugin manifest name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("plugin manifest version is required")
	}

	m.Platforms = trimmedStrings(m.Platforms)
	m.Capabilities = normalizeCapabilityList(m.Capabilities)
	if len(m.Capabilities) == 0 {
		m.Capabilities = deriveCapabilitiesFromLegacyManifest(*m)
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("plugin manifest must declare kind or capabilities")
	}
	for _, capability := range m.Capabilities {
		if err := capability.Validate(); err != nil {
			return err
		}
	}

	m.normalizeLegacyCompatibilityFields()
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

func (m *Manifest) normalizeLegacyCompatibilityFields() {
	m.Capabilities = normalizeCapabilityList(m.Capabilities)

	providerPlatforms := make([]string, 0)
	storageCapabilities := make([]CapabilityDescriptor, 0)
	for _, capability := range m.Capabilities {
		switch capability.Type {
		case CapabilityTypeProvider:
			providerPlatforms = append(providerPlatforms, capability.Platforms...)
		case CapabilityTypeStorageBackend:
			storageCapabilities = append(storageCapabilities, capability)
		}
	}

	if len(providerPlatforms) > 0 {
		m.Platforms = trimmedStrings(providerPlatforms)
	}

	if len(storageCapabilities) == 1 {
		m.StorageBackend = &StorageBackendManifest{
			Target:      storageCapabilities[0].Target,
			Backend:     storageCapabilities[0].Backend,
			DisplayName: storageCapabilities[0].DisplayName,
			Description: storageCapabilities[0].Description,
		}
	} else if len(storageCapabilities) == 0 {
		m.StorageBackend = nil
	}

	switch len(m.Capabilities) {
	case 0:
		m.Kind = strings.TrimSpace(m.Kind)
	case 1:
		m.Kind = strings.TrimSpace(m.Capabilities[0].Type)
	default:
		m.Kind = ManifestKindMulti
	}
}

// CapabilityList returns a cloned normalized capability list.
func (m Manifest) CapabilityList() []CapabilityDescriptor {
	return append([]CapabilityDescriptor(nil), normalizeCapabilityList(m.Capabilities)...)
}

// CapabilitiesByType returns all capabilities with the requested type.
func (m Manifest) CapabilitiesByType(capabilityType string) []CapabilityDescriptor {
	capabilityType = strings.TrimSpace(capabilityType)
	items := make([]CapabilityDescriptor, 0)
	for _, capability := range m.CapabilityList() {
		if capability.Type == capabilityType {
			items = append(items, capability)
		}
	}
	return items
}

// SupportsKind reports whether the manifest exposes one legacy kind.
func (m Manifest) SupportsKind(kind string) bool {
	kind = strings.TrimSpace(kind)
	switch kind {
	case ManifestKindProvider:
		return len(m.CapabilitiesByType(CapabilityTypeProvider)) > 0
	case ManifestKindAuthLauncher:
		return len(m.CapabilitiesByType(CapabilityTypeAuthLauncher)) > 0
	case ManifestKindStorageBackend:
		return len(m.CapabilitiesByType(CapabilityTypeStorageBackend)) > 0
	case ManifestKindMulti:
		return len(m.CapabilityList()) > 1
	default:
		return strings.TrimSpace(m.Kind) == kind
	}
}

// StorageBackendCapabilities returns all normalized storage backend capabilities.
func (m Manifest) StorageBackendCapabilities() []CapabilityDescriptor {
	return m.CapabilitiesByType(CapabilityTypeStorageBackend)
}

// Validate validates one storage backend descriptor.
func (m StorageBackendManifest) Validate() error {
	target := strings.TrimSpace(m.Target)
	switch target {
	case "secret_store", "session_store", "authflow_store", "governance":
	default:
		return fmt.Errorf("storage backend target must be one of secret_store, session_store, authflow_store, governance")
	}
	if strings.TrimSpace(m.Backend) == "" {
		return fmt.Errorf("storage backend plugin manifest storage_backend.backend is required")
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
