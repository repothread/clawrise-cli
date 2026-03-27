package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverManifests discovers manifests under one or more plugin roots.
func DiscoverManifests(roots []string) ([]Manifest, error) {
	discovered := make([]Manifest, 0)

	for _, root := range roots {
		entries, err := discoverManifestsInRoot(root)
		if err != nil {
			return nil, err
		}
		discovered = append(discovered, entries...)
	}

	sort.Slice(discovered, func(i, j int) bool {
		if discovered[i].Name == discovered[j].Name {
			return discovered[i].Version < discovered[j].Version
		}
		return discovered[i].Name < discovered[j].Name
	})
	return discovered, nil
}

// DefaultDiscoveryRoots returns the default plugin discovery roots.
func DefaultDiscoveryRoots() ([]string, error) {
	roots := make([]string, 0, 3)

	if raw := strings.TrimSpace(os.Getenv("CLAWRISE_PLUGIN_PATHS")); raw != "" {
		for _, item := range strings.Split(raw, string(os.PathListSeparator)) {
			item = strings.TrimSpace(item)
			if item != "" {
				roots = append(roots, item)
			}
		}
	}

	roots = append(roots, filepath.Join(".clawrise", "plugins"))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user home directory for plugin discovery: %w", err)
	}
	roots = append(roots, filepath.Join(homeDir, ".clawrise", "plugins"))
	return roots, nil
}

func discoverManifestsInRoot(root string) ([]Manifest, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat plugin root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	manifests := make([]Manifest, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) != ManifestFileName {
			return nil
		}

		manifest, err := LoadManifest(path)
		if err != nil {
			return err
		}
		manifests = append(manifests, manifest)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover manifests in %s: %w", root, err)
	}
	return manifests, nil
}
