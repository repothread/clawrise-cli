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
