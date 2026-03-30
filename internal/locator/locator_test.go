package locator

import (
	"path/filepath"
	"testing"
)

func TestResolveStateDirUsesConfigSiblingForExplicitConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	stateDir, err := ResolveStateDir(configPath)
	if err != nil {
		t.Fatalf("ResolveStateDir returned error: %v", err)
	}
	expected := filepath.Join(filepath.Dir(configPath), "state")
	if stateDir != expected {
		t.Fatalf("unexpected state dir: got=%s want=%s", stateDir, expected)
	}
}

func TestResolveRuntimeDirUsesConfigSiblingForExplicitConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	runtimeDir, err := ResolveRuntimeDir(configPath)
	if err != nil {
		t.Fatalf("ResolveRuntimeDir returned error: %v", err)
	}
	expected := filepath.Join(filepath.Dir(configPath), "runtime")
	if runtimeDir != expected {
		t.Fatalf("unexpected runtime dir: got=%s want=%s", runtimeDir, expected)
	}
}

func TestResolveStateDirResolutionPrefersEnvironmentOverride(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_STATE_DIR", filepath.Join(t.TempDir(), "env-state"))

	resolution, err := ResolveStateDirResolution(configPath)
	if err != nil {
		t.Fatalf("ResolveStateDirResolution returned error: %v", err)
	}
	if resolution.Source != "env.CLAWRISE_STATE_DIR" {
		t.Fatalf("unexpected state dir source: %+v", resolution)
	}
}

func TestResolveConfigPathResolutionUsesConfigEnv(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom-config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)

	resolution, err := ResolveConfigPathResolution()
	if err != nil {
		t.Fatalf("ResolveConfigPathResolution returned error: %v", err)
	}
	if resolution.Path != configPath {
		t.Fatalf("unexpected config path: got=%s want=%s", resolution.Path, configPath)
	}
	if resolution.Source != "env.CLAWRISE_CONFIG" {
		t.Fatalf("unexpected config path source: %+v", resolution)
	}
}

func TestResolveRuntimeDirResolutionUsesDefaultStateDirForDefaultConfigPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config-home"))
	t.Setenv("CLAWRISE_STATE_HOME", filepath.Join(homeDir, ".state-home"))

	configPath, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}
	resolution, err := ResolveRuntimeDirResolution(configPath)
	if err != nil {
		t.Fatalf("ResolveRuntimeDirResolution returned error: %v", err)
	}

	expected := filepath.Join(homeDir, ".state-home", "runtime")
	if resolution.Path != expected {
		t.Fatalf("unexpected runtime dir: got=%s want=%s", resolution.Path, expected)
	}
	if resolution.Source != "env.CLAWRISE_STATE_HOME" {
		t.Fatalf("unexpected runtime dir source: %+v", resolution)
	}
}
