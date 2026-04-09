package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyInstalledPassesForUntouchedPlugin(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	if _, err := InstallLocal(sourceDir); err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	result, err := VerifyInstalled("demo", "0.1.0", "0.1.0")
	if err != nil {
		t.Fatalf("VerifyInstalled returned error: %v", err)
	}
	if !result.Verified || !result.ChecksumMatch || !result.ProtocolCompatible {
		t.Fatalf("unexpected verify result: %+v", result)
	}
}

func TestVerifyInstalledDetectsChecksumMismatch(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	result, err := InstallLocal(sourceDir)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(result.Path, "bin", "demo-plugin"), []byte("#!/bin/sh\necho mutated\n"), 0o755); err != nil {
		t.Fatalf("failed to mutate installed plugin: %v", err)
	}

	verifyResult, err := VerifyInstalled("demo", "0.1.0", "0.1.0")
	if err != nil {
		t.Fatalf("VerifyInstalled returned error: %v", err)
	}
	if verifyResult.Verified || verifyResult.ChecksumMatch {
		t.Fatalf("expected checksum mismatch: %+v", verifyResult)
	}
}

func TestVerifyInstalledWarnsOnRuntimeCapabilityMismatch(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.capabilities.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"storage_backend","target":"session_store","backend":"plugin.demo_session"}]}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	if _, err := InstallLocal(sourceDir); err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	result, err := VerifyInstalled("demo", "0.1.0", "0.1.0")
	if err != nil {
		t.Fatalf("VerifyInstalled returned error: %v", err)
	}
	if !result.Verified {
		t.Fatalf("expected verification to pass with warnings only, got: %+v", result)
	}
	if len(result.RuntimeCapabilities) != 1 || result.RuntimeCapabilities[0].Type != CapabilityTypeStorageBackend {
		t.Fatalf("unexpected runtime capabilities: %+v", result.RuntimeCapabilities)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected runtime capability warning, got: %+v", result)
	}
}

func TestVerifyInstalledWithOptionsReportsTrustPolicyViolation(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	if _, err := InstallLocal(sourceDir); err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	result, err := VerifyInstalledWithOptions("demo", "0.1.0", InstallOptions{
		CoreVersion:    "0.1.0",
		AllowedSources: []string{"https"},
	})
	if err != nil {
		t.Fatalf("VerifyInstalledWithOptions returned error: %v", err)
	}
	if result.Verified {
		t.Fatalf("expected verification to fail when trust policy rejects the recorded source: %+v", result)
	}
	if result.Trust == nil || result.Trust.Allowed {
		t.Fatalf("expected trust result to report rejection, got: %+v", result.Trust)
	}
	if len(result.Issues) == 0 {
		t.Fatalf("expected trust issues in verify result, got: %+v", result)
	}
}
