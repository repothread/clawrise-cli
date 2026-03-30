package locator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStateDirUsesPathsConfigOverride(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := `paths:
  state_dir: .state-data
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	stateDir, err := ResolveStateDir(configPath)
	if err != nil {
		t.Fatalf("ResolveStateDir returned error: %v", err)
	}
	expected := filepath.Join(filepath.Dir(configPath), ".state-data")
	if stateDir != expected {
		t.Fatalf("unexpected state dir: got=%s want=%s", stateDir, expected)
	}
}

func TestResolveRuntimeDirUsesResolvedStateDir(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := `paths:
  state_dir: runtime-state
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	runtimeDir, err := ResolveRuntimeDir(configPath)
	if err != nil {
		t.Fatalf("ResolveRuntimeDir returned error: %v", err)
	}
	expected := filepath.Join(filepath.Dir(configPath), "runtime-state", "runtime")
	if runtimeDir != expected {
		t.Fatalf("unexpected runtime dir: got=%s want=%s", runtimeDir, expected)
	}
}
