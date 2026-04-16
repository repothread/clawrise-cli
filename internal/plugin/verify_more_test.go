package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersionPartsAndCoreCompatibility(t *testing.T) {
	if parts, ok := parseVersionParts("1.2.3-beta"); !ok || len(parts) != 3 || parts[0] != 1 || parts[1] != 2 || parts[2] != 3 {
		t.Fatalf("unexpected parsed version parts: parts=%v ok=%v", parts, ok)
	}
	if _, ok := parseVersionParts("1..2"); ok {
		t.Fatal("expected invalid version with empty segment to fail")
	}
	if _, ok := parseVersionParts("beta"); ok {
		t.Fatal("expected non-numeric version to fail")
	}

	for _, tc := range []struct {
		current string
		min     string
		compat  bool
		checked bool
	}{
		{current: "v1.2.3", min: "1.2.0", compat: true, checked: true},
		{current: "1.2", min: "1.2.1", compat: false, checked: true},
		{current: "", min: "1.0.0", compat: false, checked: false},
		{current: "broken", min: "1.0.0", compat: false, checked: false},
	} {
		compat, checked := checkCoreVersionCompatibility(tc.current, tc.min)
		if compat != tc.compat || checked != tc.checked {
			t.Fatalf("checkCoreVersionCompatibility(%q, %q) = (%v, %v), want (%v, %v)", tc.current, tc.min, compat, checked, tc.compat, tc.checked)
		}
	}
}

func TestVerifyAllInstalledWithOptions(t *testing.T) {
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

	results, err := VerifyAllInstalledWithOptions(InstallOptions{CoreVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("VerifyAllInstalledWithOptions returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one verify result, got %+v", results)
	}
	if !results[0].Verified || !results[0].ChecksumMatch || !results[0].ProtocolCompatible {
		t.Fatalf("unexpected verify-all result: %+v", results[0])
	}
	if results[0].Name != "demo" || results[0].Version != "0.1.0" {
		t.Fatalf("unexpected verify-all identity: %+v", results[0])
	}
}
