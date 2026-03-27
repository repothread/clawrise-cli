package plugin

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InstalledPlugin describes one installed plugin package.
type InstalledPlugin struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Platforms []string `json:"platforms"`
	RootDir   string   `json:"root_dir"`
}

// InstallResult describes one plugin installation result.
type InstallResult struct {
	Manifest Manifest `json:"manifest"`
	Path     string   `json:"path"`
}

// RemoveResult describes one plugin removal result.
type RemoveResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// InstallLocal installs one plugin from a local directory or tar.gz archive.
func InstallLocal(source string) (InstallResult, error) {
	root, err := pluginsRootDir()
	if err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to create plugins root: %w", err)
	}

	source = strings.TrimSpace(source)
	if source == "" {
		return InstallResult{}, fmt.Errorf("plugin source is required")
	}

	tempDir, err := os.MkdirTemp("", "clawrise-plugin-install-*")
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to create temporary plugin dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	pluginDir, err := materializeLocalSource(source, tempDir)
	if err != nil {
		return InstallResult{}, err
	}

	manifest, err := LoadManifest(filepath.Join(pluginDir, ManifestFileName))
	if err != nil {
		return InstallResult{}, err
	}

	targetDir := filepath.Join(root, manifest.Name, manifest.Version)
	if err := os.RemoveAll(targetDir); err != nil {
		return InstallResult{}, fmt.Errorf("failed to remove existing plugin target: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to create plugin parent dir: %w", err)
	}
	if err := copyTree(pluginDir, targetDir); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Manifest: manifest,
		Path:     targetDir,
	}, nil
}

// ListInstalled returns all installed plugins under the default plugins root.
func ListInstalled() ([]InstalledPlugin, error) {
	root, err := pluginsRootDir()
	if err != nil {
		return nil, err
	}

	manifests, err := discoverManifestsInRoot(root)
	if err != nil {
		return nil, err
	}

	items := make([]InstalledPlugin, 0, len(manifests))
	for _, manifest := range manifests {
		items = append(items, InstalledPlugin{
			Name:      manifest.Name,
			Version:   manifest.Version,
			Platforms: append([]string(nil), manifest.Platforms...),
			RootDir:   manifest.RootDir,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Version < items[j].Version
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// RemoveInstalled removes one installed plugin version.
func RemoveInstalled(name, version string) (RemoveResult, error) {
	root, err := pluginsRootDir()
	if err != nil {
		return RemoveResult{}, err
	}

	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return RemoveResult{}, fmt.Errorf("both plugin name and version are required")
	}

	targetDir := filepath.Join(root, name, version)
	if _, err := os.Stat(targetDir); err != nil {
		if os.IsNotExist(err) {
			return RemoveResult{}, fmt.Errorf("plugin %s@%s is not installed", name, version)
		}
		return RemoveResult{}, fmt.Errorf("failed to stat installed plugin: %w", err)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return RemoveResult{}, fmt.Errorf("failed to remove installed plugin: %w", err)
	}
	return RemoveResult{
		Name:    name,
		Version: version,
		Path:    targetDir,
	}, nil
}

func pluginsRootDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".clawrise", "plugins"), nil
}

func materializeLocalSource(source, tempDir string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("failed to stat plugin source: %w", err)
	}

	if info.IsDir() {
		targetDir := filepath.Join(tempDir, "plugin")
		if err := copyTree(source, targetDir); err != nil {
			return "", err
		}
		return targetDir, nil
	}

	if strings.HasSuffix(source, ".tar.gz") || strings.HasSuffix(source, ".tgz") {
		targetDir := filepath.Join(tempDir, "plugin")
		if err := extractTarGz(source, targetDir); err != nil {
			return "", err
		}
		return locatePluginRoot(targetDir)
	}

	return "", fmt.Errorf("unsupported plugin source format: %s", source)
}

func locatePluginRoot(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, ManifestFileName)); err == nil {
		return root, nil
	}

	var candidate string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() != ManifestFileName {
			return nil
		}
		candidate = filepath.Dir(path)
		return io.EOF
	})
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to inspect extracted plugin dir: %w", err)
	}
	if candidate == "" {
		return "", fmt.Errorf("plugin manifest not found in extracted archive")
	}
	return candidate, nil
}

func copyTree(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative plugin path: %w", err)
		}
		destination := filepath.Join(target, relative)

		if info.IsDir() {
			return os.MkdirAll(destination, info.Mode())
		}
		return copyFile(path, destination, info.Mode())
	})
}

func copyFile(source, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("failed to create plugin file parent dir: %w", err)
	}

	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open plugin source file: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create plugin target file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy plugin file: %w", err)
	}
	return nil
}

func extractTarGz(source, target string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open plugin archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to open gzip stream: %w", err)
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read plugin archive entry: %w", err)
		}

		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("plugin archive contains invalid path: %s", header.Name)
		}
		path := filepath.Join(target, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create extracted plugin dir: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("failed to create extracted plugin parent dir: %w", err)
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create extracted plugin file: %w", err)
			}
			if _, err := io.Copy(file, reader); err != nil {
				file.Close()
				return fmt.Errorf("failed to extract plugin file: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close extracted plugin file: %w", err)
			}
		}
	}
	return nil
}
