package paths

import (
	"path/filepath"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/locator"
)

func TestPathWrappersMatchLocatorImplementations(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configPath, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}
	locatorConfigPath, err := locator.ResolveConfigPath()
	if err != nil {
		t.Fatalf("locator.ResolveConfigPath returned error: %v", err)
	}
	if configPath != locatorConfigPath {
		t.Fatalf("config path wrapper mismatch: got=%s want=%s", configPath, locatorConfigPath)
	}

	configDir, err := DefaultConfigDir()
	if err != nil {
		t.Fatalf("DefaultConfigDir returned error: %v", err)
	}
	locatorConfigDir, err := locator.DefaultConfigDir()
	if err != nil {
		t.Fatalf("locator.DefaultConfigDir returned error: %v", err)
	}
	if configDir != locatorConfigDir {
		t.Fatalf("config dir wrapper mismatch: got=%s want=%s", configDir, locatorConfigDir)
	}

	explicitConfigPath := filepath.Join(t.TempDir(), "config.yaml")
	stateDir, err := ResolveStateDir(explicitConfigPath)
	if err != nil {
		t.Fatalf("ResolveStateDir returned error: %v", err)
	}
	locatorStateDir, err := locator.ResolveStateDir(explicitConfigPath)
	if err != nil {
		t.Fatalf("locator.ResolveStateDir returned error: %v", err)
	}
	if stateDir != locatorStateDir {
		t.Fatalf("state dir wrapper mismatch: got=%s want=%s", stateDir, locatorStateDir)
	}
}
