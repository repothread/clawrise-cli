package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const installMetadataFileName = "install.json"

var pluginDownloadHTTPClient = &http.Client{Timeout: 60 * time.Second}
var npmRegistryBaseURL = "https://registry.npmjs.org"

// InstalledPlugin describes one installed plugin package.
type InstalledPlugin struct {
	Name                    string                  `json:"name"`
	Version                 string                  `json:"version"`
	Kind                    string                  `json:"kind"`
	Platforms               []string                `json:"platforms"`
	StorageBackend          *StorageBackendManifest `json:"storage_backend,omitempty"`
	Capabilities            []CapabilityDescriptor  `json:"capabilities,omitempty"`
	Enabled                 bool                    `json:"enabled"`
	EnableRule              string                  `json:"enable_rule,omitempty"`
	Selected                bool                    `json:"selected"`
	SelectionReason         string                  `json:"selection_reason,omitempty"`
	MatchedProviderBindings []string                `json:"matched_provider_bindings,omitempty"`
	RootDir                 string                  `json:"root_dir"`
	Install                 *InstallMetadata        `json:"install,omitempty"`
}

// InstallResult describes one plugin installation result.
type InstallResult struct {
	Manifest Manifest         `json:"manifest"`
	Path     string           `json:"path"`
	Install  *InstallMetadata `json:"install,omitempty"`
}

// RemoveResult describes one plugin removal result.
type RemoveResult struct {
	Name    string           `json:"name"`
	Version string           `json:"version"`
	Path    string           `json:"path"`
	Install *InstallMetadata `json:"install,omitempty"`
}

// InstallMetadata describes recorded plugin installation metadata.
type InstallMetadata struct {
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
	ChecksumSHA string `json:"checksum_sha256"`
}

// PluginInfo describes one installed plugin with full manifest and install metadata.
type PluginInfo struct {
	Manifest                Manifest               `json:"manifest"`
	Capabilities            []CapabilityDescriptor `json:"capabilities,omitempty"`
	RuntimeCapabilities     []CapabilityDescriptor `json:"runtime_capabilities,omitempty"`
	Warnings                []string               `json:"warnings,omitempty"`
	Enabled                 bool                   `json:"enabled"`
	EnableRule              string                 `json:"enable_rule,omitempty"`
	Selected                bool                   `json:"selected"`
	SelectionReason         string                 `json:"selection_reason,omitempty"`
	MatchedProviderBindings []string               `json:"matched_provider_bindings,omitempty"`
	Path                    string                 `json:"path"`
	Install                 *InstallMetadata       `json:"install,omitempty"`
}

// Install installs one plugin from any supported source.
func Install(source string) (InstallResult, error) {
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

	pluginDir, resolvedSource, err := materializeSource(source, tempDir)
	if err != nil {
		return InstallResult{}, err
	}

	manifest, err := LoadManifest(filepath.Join(pluginDir, ManifestFileName))
	if err != nil {
		return InstallResult{}, err
	}
	checksum, err := checksumTree(pluginDir)
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to compute plugin checksum: %w", err)
	}

	installMetadata := &InstallMetadata{
		Source:      resolvedSource,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		ChecksumSHA: checksum,
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
	if err := writeInstallMetadata(targetDir, installMetadata); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Manifest: manifest,
		Path:     targetDir,
		Install:  installMetadata,
	}, nil
}

// InstallLocal installs one plugin from a local directory or tar.gz archive.
func InstallLocal(source string) (InstallResult, error) {
	return Install(source)
}

// ListInstalled returns all installed plugins under the default plugins root.
func ListInstalled() ([]InstalledPlugin, error) {
	return ListInstalledWithOptions(DiscoveryOptions{})
}

// ListInstalledWithOptions returns all installed plugins with the current discovery selection state.
func ListInstalledWithOptions(options DiscoveryOptions) ([]InstalledPlugin, error) {
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
		metadata, err := loadInstallMetadata(manifest.RootDir)
		if err != nil {
			return nil, err
		}
		items = append(items, buildInstalledPlugin(manifest, metadata, options))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Version < items[j].Version
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// InfoInstalled returns one installed plugin with manifest and install metadata.
func InfoInstalled(name, version string) (PluginInfo, error) {
	return InfoInstalledWithOptions(name, version, DiscoveryOptions{})
}

// InfoInstalledWithOptions returns one installed plugin with install metadata and current selection state.
func InfoInstalledWithOptions(name, version string, options DiscoveryOptions) (PluginInfo, error) {
	root, err := pluginsRootDir()
	if err != nil {
		return PluginInfo{}, err
	}

	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return PluginInfo{}, fmt.Errorf("both plugin name and version are required")
	}

	targetDir := filepath.Join(root, name, version)
	manifest, err := LoadManifest(filepath.Join(targetDir, ManifestFileName))
	if err != nil {
		return PluginInfo{}, err
	}
	metadata, err := loadInstallMetadata(targetDir)
	if err != nil {
		return PluginInfo{}, err
	}

	selectionState := resolveManifestSelectionState(manifest, options)
	info := PluginInfo{
		Manifest:                manifest,
		Capabilities:            cloneCapabilityList(manifest.CapabilityList()),
		Enabled:                 selectionState.Enabled,
		EnableRule:              selectionState.EnableRule,
		Selected:                selectionState.Selected,
		SelectionReason:         selectionState.SelectionReason,
		MatchedProviderBindings: matchedProviderBindingPlatforms(manifest, options.ProviderBindings),
		Path:                    targetDir,
		Install:                 metadata,
	}

	capabilityInspection := inspectRuntimeCapabilities(context.Background(), manifest)
	info.RuntimeCapabilities = capabilityInspection.RuntimeCapabilities
	info.Warnings = append(info.Warnings, capabilityInspection.Warnings...)

	return info, nil
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
	metadata, err := loadInstallMetadata(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return RemoveResult{}, err
	}
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
		Install: metadata,
	}, nil
}

func pluginsRootDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".clawrise", "plugins"), nil
}

func materializeSource(source, tempDir string) (string, string, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "file://") {
		trimmed := strings.TrimPrefix(source, "file://")
		pluginDir, _, err := materializeFileSource(trimmed, tempDir)
		return pluginDir, source, err
	}
	switch {
	case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
		archivePath, resolvedSource, err := downloadRemoteSource(source, tempDir)
		if err != nil {
			return "", "", err
		}
		pluginDir, _, err := materializeFileSource(archivePath, tempDir)
		return pluginDir, resolvedSource, err
	case strings.HasPrefix(source, "npm://"):
		archivePath, resolvedSource, err := resolveNPMSource(source, tempDir)
		if err != nil {
			return "", "", err
		}
		pluginDir, _, err := materializeFileSource(archivePath, tempDir)
		return pluginDir, resolvedSource, err
	default:
		return materializeFileSource(source, tempDir)
	}
}

func cloneStorageBackendManifest(manifest *StorageBackendManifest) *StorageBackendManifest {
	if manifest == nil {
		return nil
	}
	cloned := *manifest
	return &cloned
}

func buildInstalledPlugin(manifest Manifest, metadata *InstallMetadata, options DiscoveryOptions) InstalledPlugin {
	selectionState := resolveManifestSelectionState(manifest, options)
	return InstalledPlugin{
		Name:                    manifest.Name,
		Version:                 manifest.Version,
		Kind:                    manifest.Kind,
		Platforms:               append([]string(nil), manifest.Platforms...),
		StorageBackend:          cloneStorageBackendManifest(manifest.StorageBackend),
		Capabilities:            cloneCapabilityList(manifest.CapabilityList()),
		Enabled:                 selectionState.Enabled,
		EnableRule:              selectionState.EnableRule,
		Selected:                selectionState.Selected,
		SelectionReason:         selectionState.SelectionReason,
		MatchedProviderBindings: matchedProviderBindingPlatforms(manifest, options.ProviderBindings),
		RootDir:                 manifest.RootDir,
		Install:                 metadata,
	}
}

func materializeFileSource(source, tempDir string) (string, string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", "", fmt.Errorf("failed to stat plugin source: %w", err)
	}

	if info.IsDir() {
		targetDir := filepath.Join(tempDir, "plugin")
		if err := copyTree(source, targetDir); err != nil {
			return "", "", err
		}
		return targetDir, filepath.Clean(source), nil
	}

	if strings.HasSuffix(source, ".tar.gz") || strings.HasSuffix(source, ".tgz") {
		targetDir := filepath.Join(tempDir, "plugin")
		if err := extractTarGz(source, targetDir); err != nil {
			return "", "", err
		}
		pluginDir, err := locatePluginRoot(targetDir)
		return pluginDir, filepath.Clean(source), err
	}

	return "", "", fmt.Errorf("unsupported plugin source format: %s", source)
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

func downloadRemoteSource(source, tempDir string) (string, string, error) {
	request, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to build plugin download request: %w", err)
	}
	response, err := pluginDownloadHTTPClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("failed to download plugin source: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return "", "", fmt.Errorf("plugin source download failed with status %d", response.StatusCode)
	}

	path := filepath.Join(tempDir, "download.tgz")
	file, err := os.Create(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to create downloaded plugin archive: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		return "", "", fmt.Errorf("failed to write downloaded plugin archive: %w", err)
	}
	return path, source, nil
}

func resolveNPMSource(source, tempDir string) (string, string, error) {
	spec := strings.TrimSpace(strings.TrimPrefix(source, "npm://"))
	if spec == "" {
		return "", "", fmt.Errorf("npm package spec is required")
	}

	packageName, requestedVersion := parseNPMPackageSpec(spec)
	if packageName == "" {
		return "", "", fmt.Errorf("invalid npm package spec: %s", spec)
	}

	packageURL := strings.TrimRight(npmRegistryBaseURL, "/") + "/" + url.PathEscape(packageName)
	request, err := http.NewRequest(http.MethodGet, packageURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to build npm metadata request: %w", err)
	}
	response, err := pluginDownloadHTTPClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch npm package metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return "", "", fmt.Errorf("npm metadata request failed with status %d", response.StatusCode)
	}

	var metadata struct {
		DistTags map[string]string `json:"dist-tags"`
		Versions map[string]struct {
			Dist struct {
				Tarball string `json:"tarball"`
			} `json:"dist"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return "", "", fmt.Errorf("failed to decode npm package metadata: %w", err)
	}

	version := requestedVersion
	if version == "" {
		version = "latest"
	}
	if resolvedVersion, ok := metadata.DistTags[version]; ok {
		version = resolvedVersion
	}

	versionMetadata, ok := metadata.Versions[version]
	if !ok || strings.TrimSpace(versionMetadata.Dist.Tarball) == "" {
		return "", "", fmt.Errorf("npm package version %s not found for %s", version, packageName)
	}

	archivePath, _, err := downloadRemoteSource(versionMetadata.Dist.Tarball, tempDir)
	if err != nil {
		return "", "", err
	}
	return archivePath, source, nil
}

func parseNPMPackageSpec(spec string) (string, string) {
	if spec == "" {
		return "", ""
	}
	if strings.HasPrefix(spec, "@") {
		index := strings.LastIndex(spec, "@")
		if index <= 0 {
			return spec, ""
		}
		if strings.Count(spec[:index], "/") == 0 {
			return spec, ""
		}
		return spec[:index], spec[index+1:]
	}
	index := strings.LastIndex(spec, "@")
	if index < 0 {
		return spec, ""
	}
	return spec[:index], spec[index+1:]
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

func checksumTree(root string) (string, error) {
	hash := sha256.New()
	paths := make([]string, 0)

	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == installMetadataFileName {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(relative)); err != nil {
			return "", err
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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

func writeInstallMetadata(root string, metadata *InstallMetadata) error {
	if metadata == nil {
		return nil
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode plugin install metadata: %w", err)
	}
	path := filepath.Join(root, installMetadataFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write plugin install metadata: %w", err)
	}
	return nil
}

func loadInstallMetadata(root string) (*InstallMetadata, error) {
	path := filepath.Join(root, installMetadataFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read plugin install metadata: %w", err)
	}
	var metadata InstallMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode plugin install metadata: %w", err)
	}
	return &metadata, nil
}
