package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestNormalizesLegacyProviderToCapability(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	capabilities := manifest.CapabilityList()
	if len(capabilities) != 1 {
		t.Fatalf("expected one capability, got: %+v", capabilities)
	}
	if capabilities[0].Type != CapabilityTypeProvider {
		t.Fatalf("unexpected capability: %+v", capabilities[0])
	}
	if len(capabilities[0].Platforms) != 1 || capabilities[0].Platforms[0] != "demo" {
		t.Fatalf("unexpected provider platforms: %+v", capabilities[0].Platforms)
	}
}

func TestLoadManifestSupportsCapabilitySchemaV2(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "demo-suite",
  "version": "0.2.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "provider",
      "platforms": ["demo"]
    },
    {
      "type": "auth_launcher",
      "id": "demo_launcher",
      "action_types": ["open_url"]
    },
    {
      "type": "storage_backend",
      "target": "session_store",
      "backend": "plugin.demo_session"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	if manifest.Kind != ManifestKindMulti {
		t.Fatalf("expected multi kind, got: %s", manifest.Kind)
	}
	if !manifest.SupportsKind(ManifestKindProvider) {
		t.Fatal("expected provider capability to be discoverable")
	}
	if !manifest.SupportsKind(ManifestKindAuthLauncher) {
		t.Fatal("expected auth launcher capability to be discoverable")
	}
	if !manifest.SupportsKind(ManifestKindStorageBackend) {
		t.Fatal("expected storage backend capability to be discoverable")
	}
	if manifest.StorageBackend == nil || manifest.StorageBackend.Target != "session_store" {
		t.Fatalf("unexpected legacy storage compatibility field: %+v", manifest.StorageBackend)
	}
}

func TestSplitManifestsByKindIncludesMultiCapabilityPlugin(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:   2,
		Name:            "demo-suite",
		Version:         "0.2.0",
		ProtocolVersion: 1,
		Entry: ManifestEntry{
			Type:    "binary",
			Command: []string{"./demo-plugin"},
		},
		Capabilities: []CapabilityDescriptor{
			{Type: CapabilityTypeProvider, Platforms: []string{"demo"}},
			{Type: CapabilityTypeAuthLauncher, ID: "demo_launcher"},
			{Type: CapabilityTypeStorageBackend, Target: "secret_store", Backend: "plugin.demo_secret"},
		},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	providers, launchers, storageBackends := SplitManifestsByKind([]Manifest{manifest})
	if len(providers) != 1 {
		t.Fatalf("expected provider manifest, got: %+v", providers)
	}
	if len(launchers) != 1 {
		t.Fatalf("expected launcher manifest, got: %+v", launchers)
	}
	if len(storageBackends) != 1 {
		t.Fatalf("expected storage manifest, got: %+v", storageBackends)
	}
}

func TestLoadManifestSupportsWorkflowAndRegistrySourceCapabilities(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "demo-suite",
  "version": "0.3.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "workflow",
      "id": "planner"
    },
    {
      "type": "registry_source",
      "id": "community"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	capabilities := manifest.CapabilityList()
	if len(capabilities) != 2 {
		t.Fatalf("expected two capabilities, got: %+v", capabilities)
	}
	if capabilities[0].Type != CapabilityTypeRegistrySource || capabilities[1].Type != CapabilityTypeWorkflow {
		t.Fatalf("unexpected capability normalization result: %+v", capabilities)
	}
}
