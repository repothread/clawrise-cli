package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
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

const (
	pluginSourceTypeLocal    = "local"
	pluginSourceTypeFile     = "file"
	pluginSourceTypeHTTP     = "http"
	pluginSourceTypeHTTPS    = "https"
	pluginSourceTypeNPM      = "npm"
	pluginSourceTypeRegistry = "registry"
)

var defaultAllowedInstallSources = []string{
	pluginSourceTypeLocal,
	pluginSourceTypeFile,
	pluginSourceTypeHTTPS,
	pluginSourceTypeNPM,
	pluginSourceTypeRegistry,
}

// InstalledPlugin describes one installed plugin package.
type InstalledPlugin struct {
	Name                    string                  `json:"name"`
	Version                 string                  `json:"version"`
	Kind                    string                  `json:"kind"`
	Platforms               []string                `json:"platforms"`
	StorageBackend          *StorageBackendManifest `json:"storage_backend,omitempty"`
	Capabilities            []CapabilityDescriptor  `json:"capabilities,omitempty"`
	CapabilityRoutes        []CapabilityRouteStatus `json:"capability_routes,omitempty"`
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

// InstallOptions describes additional constraints applied to one plugin installation.
type InstallOptions struct {
	CoreVersion      string
	AllowedSources   []string
	AllowedHosts     []string
	AllowedNPMScopes []string
	DiscoveryOptions DiscoveryOptions
	ExpectedName     string
}

// SourceTrustStatus describes how one install source matches the current trust policy.
type SourceTrustStatus struct {
	Source           string   `json:"source"`
	SourceType       string   `json:"source_type"`
	Allowed          bool     `json:"allowed"`
	Host             string   `json:"host,omitempty"`
	PackageName      string   `json:"package_name,omitempty"`
	PackageScope     string   `json:"package_scope,omitempty"`
	RegistrySourceID string   `json:"registry_source_id,omitempty"`
	Reference        string   `json:"reference,omitempty"`
	Issues           []string `json:"issues,omitempty"`
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
	ArtifactURL string `json:"artifact_url,omitempty"`
}

// UpgradeResult describes one plugin upgrade result.
type UpgradeResult struct {
	Name            string             `json:"name"`
	FromVersion     string             `json:"from_version"`
	ToVersion       string             `json:"to_version"`
	Source          string             `json:"source"`
	SourceType      string             `json:"source_type,omitempty"`
	Checked         bool               `json:"checked"`
	UpdateAvailable bool               `json:"update_available"`
	Upgraded        bool               `json:"upgraded"`
	Reinstalled     bool               `json:"reinstalled"`
	RemovedPrevious bool               `json:"removed_previous"`
	Reason          string             `json:"reason,omitempty"`
	Error           string             `json:"error,omitempty"`
	Path            string             `json:"path"`
	Trust           *SourceTrustStatus `json:"trust,omitempty"`
	Install         *InstallMetadata   `json:"install,omitempty"`
	Preflight       *VerifyResult      `json:"preflight,omitempty"`
}

// PluginInfo describes one installed plugin with full manifest and install metadata.
type PluginInfo struct {
	Manifest                Manifest                `json:"manifest"`
	Capabilities            []CapabilityDescriptor  `json:"capabilities,omitempty"`
	RuntimeCapabilities     []CapabilityDescriptor  `json:"runtime_capabilities,omitempty"`
	CapabilityRoutes        []CapabilityRouteStatus `json:"capability_routes,omitempty"`
	Warnings                []string                `json:"warnings,omitempty"`
	Enabled                 bool                    `json:"enabled"`
	EnableRule              string                  `json:"enable_rule,omitempty"`
	Selected                bool                    `json:"selected"`
	SelectionReason         string                  `json:"selection_reason,omitempty"`
	MatchedProviderBindings []string                `json:"matched_provider_bindings,omitempty"`
	Path                    string                  `json:"path"`
	Install                 *InstallMetadata        `json:"install,omitempty"`
}

type installSourceReference struct {
	Raw               string
	SourceType        string
	ResolvedPath      string
	URL               *url.URL
	PackageName       string
	PackageScope      string
	RequestedTag      string
	RegistrySourceID  string
	RegistryReference string
}

type installCandidate struct {
	Manifest       Manifest
	PluginDir      string
	ResolvedSource string
	ArtifactURL    string
	Trust          SourceTrustStatus
}

type remoteDownloadResult struct {
	ArchivePath string
	FinalURL    string
}

// Install installs one plugin from any supported source.
func Install(source string) (InstallResult, error) {
	return InstallWithOptions(source, InstallOptions{})
}

// InstallWithOptions installs one plugin from any supported source with explicit trust and compatibility options.
// 公开 API 签名不变，内部使用 context.Background() 作为 fallback。
func InstallWithOptions(source string, options InstallOptions) (InstallResult, error) {
	ctx := context.Background()
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
	candidate, cleanup, err := resolveInstallCandidate(ctx, source, options)
	if err != nil {
		return InstallResult{}, err
	}
	defer cleanup()

	checksum, err := checksumTree(candidate.PluginDir)
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to compute plugin checksum: %w", err)
	}

	installMetadata := &InstallMetadata{
		Source:      candidate.ResolvedSource,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		ChecksumSHA: checksum,
		ArtifactURL: candidate.ArtifactURL,
	}

	targetDir := filepath.Join(root, candidate.Manifest.Name, candidate.Manifest.Version)
	if err := os.RemoveAll(targetDir); err != nil {
		return InstallResult{}, fmt.Errorf("failed to remove existing plugin target: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("failed to create plugin parent dir: %w", err)
	}
	if err := copyTree(candidate.PluginDir, targetDir); err != nil {
		return InstallResult{}, err
	}
	if err := writeInstallMetadata(targetDir, installMetadata); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Manifest: candidate.Manifest,
		Path:     targetDir,
		Install:  installMetadata,
	}, nil
}

// InstallLocal installs one plugin from a local directory or tar.gz archive.
func InstallLocal(source string) (InstallResult, error) {
	return InstallWithOptions(source, InstallOptions{})
}

// UpgradeInstalled upgrades one installed plugin by reinstalling from its recorded source.
// 公开 API 签名不变，内部使用 context.Background() 作为 fallback。
func UpgradeInstalled(name, version string, options InstallOptions) (UpgradeResult, error) {
	ctx := context.Background()
	result, candidate, err := checkUpgradeCandidate(ctx, name, version, options)
	if err != nil {
		return result, err
	}
	if !result.UpdateAvailable {
		return result, nil
	}

	options.ExpectedName = name
	installed, err := InstallWithOptions(result.Source, options)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Checked = true
	result.Upgraded = installed.Manifest.Version != version
	result.Reinstalled = installed.Manifest.Version == version
	result.ToVersion = installed.Manifest.Version
	result.Path = installed.Path
	result.Install = installed.Install

	// Remove the previous version after a successful upgrade so discovery does not pick multiple versions.
	if installed.Manifest.Version != version {
		if _, err := RemoveInstalled(name, version); err != nil {
			result.Error = fmt.Sprintf("plugin %s@%s upgraded to %s but failed to remove previous version: %v", name, version, installed.Manifest.Version, err)
			return result, errors.New(result.Error)
		}
		result.RemovedPrevious = true
	}

	if candidate != nil {
		result.Trust = &candidate.Trust
	}
	return result, nil
}

// UpgradeAllInstalled upgrades every installed plugin version that still records an install source.
func UpgradeAllInstalled(options InstallOptions) ([]UpgradeResult, error) {
	items, err := ListInstalled()
	if err != nil {
		return nil, err
	}

	results := make([]UpgradeResult, 0, len(items))
	for _, item := range items {
		result, upgradeErr := UpgradeInstalled(item.Name, item.Version, options)
		if upgradeErr != nil {
			if result.Name == "" {
				result.Name = item.Name
				result.FromVersion = item.Version
			}
			result.Error = upgradeErr.Error()
		}
		results = append(results, result)
	}
	return results, nil
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
// 公开 API 签名不变，内部使用 context.Background() 作为 fallback。
func InfoInstalledWithOptions(name, version string, options DiscoveryOptions) (PluginInfo, error) {
	ctx := context.Background()
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
		CapabilityRoutes:        inspectCapabilityRoutes(manifest, options),
		Enabled:                 selectionState.Enabled,
		EnableRule:              selectionState.EnableRule,
		Selected:                selectionState.Selected,
		SelectionReason:         selectionState.SelectionReason,
		MatchedProviderBindings: matchedProviderBindingPlatforms(manifest, options.ProviderBindings),
		Path:                    targetDir,
		Install:                 metadata,
	}

	// 使用内部 ctx 替代硬编码的 context.Background()，保持 context 传播链路
	capabilityInspection := inspectRuntimeCapabilities(ctx, manifest)
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

// resolveInstallCandidate 解析插件安装源并准备候选安装包，使用调用方 context 控制远程请求生命周期。
func resolveInstallCandidate(ctx context.Context, source string, options InstallOptions) (installCandidate, func(), error) {
	source = strings.TrimSpace(source)
	reference, err := parseInstallSourceReference(source)
	if err != nil {
		return installCandidate{}, func() {}, err
	}

	trust := evaluateInstallSourceTrust(reference, options)
	if !trust.Allowed {
		return installCandidate{}, func() {}, errors.New(strings.Join(trust.Issues, "; "))
	}

	tempDir, err := os.MkdirTemp("", "clawrise-plugin-install-*")
	if err != nil {
		return installCandidate{}, func() {}, fmt.Errorf("failed to create temporary plugin dir: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	pluginDir, resolvedSource, artifactURL, err := materializeSource(ctx, reference, tempDir, options)
	if err != nil {
		cleanup()
		return installCandidate{}, func() {}, err
	}

	manifest, err := LoadManifest(filepath.Join(pluginDir, ManifestFileName))
	if err != nil {
		cleanup()
		return installCandidate{}, func() {}, err
	}
	if err := validateInstallManifest(manifest, options); err != nil {
		cleanup()
		return installCandidate{}, func() {}, err
	}

	candidate := installCandidate{
		Manifest:       manifest,
		PluginDir:      pluginDir,
		ResolvedSource: resolvedSource,
		ArtifactURL:    artifactURL,
		Trust:          trust,
	}
	return candidate, cleanup, nil
}

// checkUpgradeCandidate 检查已安装插件是否有可升级版本，使用调用方 context 控制远程请求生命周期。
func checkUpgradeCandidate(ctx context.Context, name, version string, options InstallOptions) (UpgradeResult, *installCandidate, error) {
	info, err := InfoInstalled(name, version)
	if err != nil {
		return UpgradeResult{}, nil, err
	}
	if info.Install == nil || strings.TrimSpace(info.Install.Source) == "" {
		return UpgradeResult{
			Name:        info.Manifest.Name,
			FromVersion: info.Manifest.Version,
			ToVersion:   info.Manifest.Version,
			Reason:      "installed plugin does not record an install source",
		}, nil, fmt.Errorf("plugin %s@%s does not record an install source, so it cannot be upgraded automatically", info.Manifest.Name, info.Manifest.Version)
	}

	options.ExpectedName = info.Manifest.Name
	preflight, err := VerifyInstalledWithOptions(info.Manifest.Name, info.Manifest.Version, options)
	if err != nil {
		return UpgradeResult{}, nil, err
	}

	result := UpgradeResult{
		Name:        info.Manifest.Name,
		FromVersion: info.Manifest.Version,
		ToVersion:   info.Manifest.Version,
		Source:      info.Install.Source,
		SourceType:  classifyInstallSource(info.Install.Source),
		Checked:     true,
		Trust:       preflight.Trust,
		Install:     info.Install,
		Preflight:   &preflight,
	}
	if !preflight.Verified {
		result.Path = info.Path
		result.Reason = "installed plugin failed pre-upgrade verification"
		result.Error = buildPreUpgradeVerificationError(preflight)
		return result, nil, fmt.Errorf("plugin %s@%s failed pre-upgrade verification: %s", info.Manifest.Name, info.Manifest.Version, result.Error)
	}

	candidate, cleanup, err := resolveInstallCandidate(ctx, info.Install.Source, options)
	if err != nil {
		return result, nil, err
	}
	defer cleanup()

	result.ToVersion = candidate.Manifest.Version
	result.SourceType = candidate.Trust.SourceType
	result.Trust = &candidate.Trust

	comparison, comparable := compareVersionStrings(info.Manifest.Version, candidate.Manifest.Version)
	switch {
	case comparable && comparison < 0:
		result.UpdateAvailable = true
	case comparable && comparison == 0:
		result.Reason = upgradeNoChangeReason(info.Install.Source, true)
	case comparable && comparison > 0:
		result.Reason = "recorded source currently resolves to an older plugin version"
	case !comparable && strings.TrimSpace(candidate.Manifest.Version) != strings.TrimSpace(info.Manifest.Version):
		result.UpdateAvailable = true
	default:
		result.Reason = upgradeNoChangeReason(info.Install.Source, false)
	}

	return result, &candidate, nil
}

func evaluateInstallSourceTrust(reference installSourceReference, options InstallOptions) SourceTrustStatus {
	status := SourceTrustStatus{
		Source:           reference.Raw,
		SourceType:       reference.SourceType,
		Host:             normalizeHostName(hostnameFromURL(reference.URL)),
		PackageName:      reference.PackageName,
		PackageScope:     reference.PackageScope,
		RegistrySourceID: reference.RegistrySourceID,
		Reference:        reference.RegistryReference,
		Allowed:          true,
	}

	allowedSources := normalizeAllowedInstallSources(options.AllowedSources)
	if _, ok := allowedSources[reference.SourceType]; !ok {
		status.Allowed = false
		status.Issues = append(status.Issues, fmt.Sprintf("plugin source type %s is not allowed by the current install trust policy", reference.SourceType))
	}

	allowedHosts := normalizeAllowedHosts(options.AllowedHosts)
	if isRemoteSourceType(reference.SourceType) && len(allowedHosts) > 0 {
		host := normalizeHostName(hostnameFromURL(reference.URL))
		if host == "" || !matchesAnyAllowedHost(host, allowedHosts) {
			status.Allowed = false
			status.Issues = append(status.Issues, fmt.Sprintf("plugin source host %s is not allowed by the current install trust policy", firstNonEmptyString(host, "<unknown>")))
		}
	}

	allowedNPMScopes := normalizeAllowedNPMScopes(options.AllowedNPMScopes)
	if reference.SourceType == pluginSourceTypeNPM && len(allowedNPMScopes) > 0 {
		if !matchesAllowedNPMScope(reference.PackageScope, allowedNPMScopes) {
			scope := reference.PackageScope
			if scope == "" {
				scope = "unscoped"
			}
			status.Allowed = false
			status.Issues = append(status.Issues, fmt.Sprintf("npm package scope %s is not allowed by the current install trust policy", scope))
		}
	}

	return status
}

func normalizeAllowedInstallSources(values []string) map[string]struct{} {
	if len(values) == 0 {
		values = defaultAllowedInstallSources
	}

	items := make(map[string]struct{}, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case pluginSourceTypeLocal, pluginSourceTypeFile, pluginSourceTypeHTTP, pluginSourceTypeHTTPS, pluginSourceTypeNPM, pluginSourceTypeRegistry:
			items[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
		}
	}
	if len(items) == 0 {
		for _, value := range defaultAllowedInstallSources {
			items[value] = struct{}{}
		}
	}
	return items
}

func normalizeAllowedHosts(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeHostName(value)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func normalizeAllowedNPMScopes(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		switch {
		case value == "":
			continue
		case value == "*" || value == "unscoped":
		case !strings.HasPrefix(value, "@"):
			value = "@" + value
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func parseInstallSourceReference(source string) (installSourceReference, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return installSourceReference{}, fmt.Errorf("plugin source is required")
	}

	reference := installSourceReference{
		Raw:        source,
		SourceType: classifyInstallSource(source),
	}
	switch reference.SourceType {
	case pluginSourceTypeFile:
		reference.ResolvedPath = strings.TrimPrefix(source, "file://")
	case pluginSourceTypeHTTP, pluginSourceTypeHTTPS:
		parsed, err := url.Parse(source)
		if err != nil {
			return installSourceReference{}, fmt.Errorf("failed to parse plugin source url: %w", err)
		}
		reference.URL = parsed
	case pluginSourceTypeNPM:
		spec := strings.TrimSpace(strings.TrimPrefix(source, "npm://"))
		if spec == "" {
			spec = source
		}
		packageName, requestedVersion := parseNPMPackageSpec(spec)
		if packageName == "" {
			return installSourceReference{}, fmt.Errorf("invalid npm package spec: %s", spec)
		}
		reference.PackageName = packageName
		reference.PackageScope = npmPackageScope(packageName)
		reference.RequestedTag = requestedVersion
		if parsed, err := url.Parse(strings.TrimRight(npmRegistryBaseURL, "/")); err == nil {
			reference.URL = parsed
		}
	case pluginSourceTypeRegistry:
		registrySourceID, registryReference, requestedVersion, err := parseRegistryInstallSpec(source)
		if err != nil {
			return installSourceReference{}, err
		}
		reference.RegistrySourceID = registrySourceID
		reference.RegistryReference = registryReference
		reference.RequestedTag = requestedVersion
	default:
		reference.ResolvedPath = source
	}
	return reference, nil
}

func classifyInstallSource(source string) string {
	source = strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(source, "file://"):
		return pluginSourceTypeFile
	case strings.HasPrefix(source, "registry://"):
		return pluginSourceTypeRegistry
	case strings.HasPrefix(source, "https://"):
		return pluginSourceTypeHTTPS
	case strings.HasPrefix(source, "http://"):
		return pluginSourceTypeHTTP
	case strings.HasPrefix(source, "npm://"):
		return pluginSourceTypeNPM
	case shouldResolveBareNPMPackageSpec(source):
		return pluginSourceTypeNPM
	default:
		return pluginSourceTypeLocal
	}
}

func parseRegistryInstallSpec(source string) (string, string, string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(source, "registry://"))
	if trimmed == "" {
		return "", "", "", fmt.Errorf("registry source reference is required")
	}

	var sourceID string
	var reference string
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		sourceID = strings.TrimSpace(parts[0])
		reference = strings.TrimSpace(parts[1])
	} else {
		reference = trimmed
	}

	reference = strings.TrimPrefix(reference, "/")
	if reference == "" {
		return "", "", "", fmt.Errorf("registry install reference is required")
	}

	version := ""
	if index := strings.LastIndex(reference, "@"); index > 0 {
		version = strings.TrimSpace(reference[index+1:])
		reference = strings.TrimSpace(reference[:index])
	}
	if reference == "" {
		return "", "", "", fmt.Errorf("registry install reference is required")
	}
	return sourceID, reference, version, nil
}

func validateInstallManifest(manifest Manifest, options InstallOptions) error {
	expectedName := strings.TrimSpace(options.ExpectedName)
	if expectedName != "" && manifest.Name != expectedName {
		return fmt.Errorf("expected plugin source to resolve to %s, got %s", expectedName, manifest.Name)
	}
	if manifest.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("plugin %s@%s uses protocol_version %d, but current core expects %d", manifest.Name, manifest.Version, manifest.ProtocolVersion, ProtocolVersion)
	}

	coreVersion := strings.TrimSpace(options.CoreVersion)
	minCoreVersion := strings.TrimSpace(manifest.MinCoreVersion)
	if compatible, checked := checkCoreVersionCompatibility(coreVersion, minCoreVersion); checked && !compatible {
		return fmt.Errorf("plugin %s@%s requires core version %s or newer, current core is %s", manifest.Name, manifest.Version, minCoreVersion, coreVersion)
	}
	return nil
}

// materializeSource 根据源类型将插件安装包下载或复制到临时目录，使用调用方 context 控制远程请求生命周期。
func materializeSource(ctx context.Context, reference installSourceReference, tempDir string, options InstallOptions) (string, string, string, error) {
	source := strings.TrimSpace(reference.Raw)
	switch reference.SourceType {
	case pluginSourceTypeFile:
		pluginDir, _, err := materializeFileSource(reference.ResolvedPath, tempDir)
		return pluginDir, source, "", err
	case pluginSourceTypeHTTP, pluginSourceTypeHTTPS:
		// HTTP/HTTPS 远程下载，传入 ctx 使下载可被取消
		downloaded, err := downloadRemoteSource(ctx, source, tempDir, options)
		if err != nil {
			return "", "", "", err
		}
		pluginDir, _, err := materializeFileSource(downloaded.ArchivePath, tempDir)
		return pluginDir, source, downloaded.FinalURL, err
	case pluginSourceTypeNPM:
		// NPM registry 解析，传入 ctx 使 npm 元数据查询和下载可被取消
		archivePath, resolvedSource, artifactURL, err := resolveNPMSource(ctx, reference, tempDir, options)
		if err != nil {
			return "", "", "", err
		}
		pluginDir, _, err := materializeFileSource(archivePath, tempDir)
		return pluginDir, resolvedSource, artifactURL, err
	case pluginSourceTypeRegistry:
		// Registry source 插件解析，传入 ctx 使插件 RPC 调用和下载可被取消
		archivePath, resolvedSource, artifactURL, err := resolveRegistrySource(ctx, reference, tempDir, options)
		if err != nil {
			return "", "", "", err
		}
		pluginDir, _, err := materializeFileSource(archivePath, tempDir)
		if err != nil {
			return "", "", "", err
		}
		return pluginDir, resolvedSource, artifactURL, err
	default:
		pluginDir, resolvedSource, err := materializeFileSource(reference.ResolvedPath, tempDir)
		return pluginDir, resolvedSource, "", err
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
		CapabilityRoutes:        inspectCapabilityRoutes(manifest, options),
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

// downloadRemoteSource 从远程 URL 下载插件安装包，使用调用方 context 控制请求生命周期。
// 当 context 被取消时，进行中的 HTTP 请求会被立即终止。
func downloadRemoteSource(ctx context.Context, source, tempDir string, options InstallOptions) (remoteDownloadResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return remoteDownloadResult{}, fmt.Errorf("failed to build plugin download request: %w", err)
	}
	response, err := pluginDownloadHTTPClient.Do(request)
	if err != nil {
		return remoteDownloadResult{}, fmt.Errorf("failed to download plugin source: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return remoteDownloadResult{}, fmt.Errorf("plugin source download failed with status %d", response.StatusCode)
	}

	finalURL := source
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}
	// 即使初始地址通过了 allowlist，最终重定向目标仍要再次校验。
	if err := validateFinalRemoteDownloadTarget(source, finalURL, options); err != nil {
		return remoteDownloadResult{}, err
	}

	path := filepath.Join(tempDir, "download.tgz")
	file, err := os.Create(path)
	if err != nil {
		return remoteDownloadResult{}, fmt.Errorf("failed to create downloaded plugin archive: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		return remoteDownloadResult{}, fmt.Errorf("failed to write downloaded plugin archive: %w", err)
	}
	return remoteDownloadResult{
		ArchivePath: path,
		FinalURL:    finalURL,
	}, nil
}

// resolveNPMSource 从 npm registry 解析插件包，使用调用方 context 控制请求生命周期。
func resolveNPMSource(ctx context.Context, reference installSourceReference, tempDir string, options InstallOptions) (string, string, string, error) {
	if reference.PackageName == "" {
		return "", "", "", fmt.Errorf("npm package spec is required")
	}

	packageURL := strings.TrimRight(npmRegistryBaseURL, "/") + "/" + url.PathEscape(reference.PackageName)
	if err := validateRemoteHostPolicy(packageURL, options); err != nil {
		return "", "", "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, packageURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to build npm metadata request: %w", err)
	}
	response, err := pluginDownloadHTTPClient.Do(request)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch npm package metadata: %w", err)
	}
	defer response.Body.Close()
	if response.Request != nil && response.Request.URL != nil {
		// npm metadata 请求本身也可能跳到别的 host，需要复验最终位置。
		if err := validateFinalRemoteDownloadTarget(packageURL, response.Request.URL.String(), options); err != nil {
			return "", "", "", err
		}
	}
	if response.StatusCode >= 400 {
		return "", "", "", fmt.Errorf("npm metadata request failed with status %d", response.StatusCode)
	}

	var metadata struct {
		DistTags map[string]string `json:"dist-tags"`
		Versions map[string]struct {
			Dist struct {
				Tarball   string `json:"tarball"`
				Integrity string `json:"integrity"`
				Shasum    string `json:"shasum"`
			} `json:"dist"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return "", "", "", fmt.Errorf("failed to decode npm package metadata: %w", err)
	}

	version := reference.RequestedTag
	if version == "" {
		version = "latest"
	}
	if resolvedVersion, ok := metadata.DistTags[version]; ok {
		version = resolvedVersion
	}

	versionMetadata, ok := metadata.Versions[version]
	if !ok || strings.TrimSpace(versionMetadata.Dist.Tarball) == "" {
		return "", "", "", fmt.Errorf("npm package version %s not found for %s", version, reference.PackageName)
	}
	if err := validateRemoteHostPolicy(versionMetadata.Dist.Tarball, options); err != nil {
		return "", "", "", err
	}

	downloaded, err := downloadRemoteSource(ctx, versionMetadata.Dist.Tarball, tempDir, options)
	if err != nil {
		return "", "", "", err
	}
	if err := verifyDownloadedNPMArtifact(downloaded.ArchivePath, versionMetadata.Dist.Integrity, versionMetadata.Dist.Shasum); err != nil {
		return "", "", "", err
	}
	return downloaded.ArchivePath, reference.Raw, downloaded.FinalURL, nil
}

// resolveRegistrySource 通过 registry source 插件解析插件包，使用调用方 context 控制请求生命周期。
func resolveRegistrySource(ctx context.Context, reference installSourceReference, tempDir string, options InstallOptions) (string, string, string, error) {
	runtimes, err := DiscoverRegistrySourceRuntimes(options.DiscoveryOptions)
	if err != nil {
		return "", "", "", err
	}
	if len(runtimes) == 0 {
		return "", "", "", fmt.Errorf("no registry source plugins are available for registry installs")
	}

	candidates := make([]RegistrySourceRuntime, 0, len(runtimes))
	for _, runtime := range runtimes {
		if reference.RegistrySourceID != "" && runtime.ID() != reference.RegistrySourceID {
			continue
		}
		candidates = append(candidates, runtime)
	}
	if len(candidates) == 0 {
		return "", "", "", fmt.Errorf("registry source %s is not available", reference.RegistrySourceID)
	}

	var lastErr error
	for _, runtime := range candidates {
		if strings.TrimSpace(runtime.ID()) == "" {
			lastErr = fmt.Errorf("registry source plugin %s does not declare a capability id", runtime.Name())
			continue
		}
		resolveResult, resolveErr := runtime.Resolve(ctx, RegistrySourceResolveParams{
			SourceID:  runtime.ID(),
			Reference: reference.RegistryReference,
			Version:   reference.RequestedTag,
		})
		_ = runtime.Close()
		if resolveErr != nil {
			lastErr = resolveErr
			if reference.RegistrySourceID != "" {
				return "", "", "", resolveErr
			}
			continue
		}
		if strings.TrimSpace(resolveResult.ArtifactURL) == "" {
			lastErr = fmt.Errorf("registry source %s returned an empty artifact_url", runtime.ID())
			if reference.RegistrySourceID != "" {
				return "", "", "", lastErr
			}
			continue
		}
		if err := validateRemoteHostPolicy(resolveResult.ArtifactURL, options); err != nil {
			lastErr = err
			if reference.RegistrySourceID != "" {
				return "", "", "", err
			}
			continue
		}

		downloaded, err := downloadRemoteSource(ctx, resolveResult.ArtifactURL, tempDir, options)
		if err != nil {
			lastErr = err
			if reference.RegistrySourceID != "" {
				return "", "", "", err
			}
			continue
		}
		if err := verifyDownloadedArtifactChecksum(downloaded.ArchivePath, resolveResult.ChecksumSHA256); err != nil {
			lastErr = err
			if reference.RegistrySourceID != "" {
				return "", "", "", err
			}
			continue
		}

		canonicalSource := canonicalRegistrySource(runtime.ID(), firstNonEmptyString(resolveResult.Name, reference.RegistryReference), reference.RequestedTag)
		return downloaded.ArchivePath, canonicalSource, downloaded.FinalURL, nil
	}

	if lastErr != nil {
		return "", "", "", lastErr
	}
	return "", "", "", fmt.Errorf("no registry source could resolve %s", reference.RegistryReference)
}

func shouldResolveBareNPMPackageSpec(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if strings.HasSuffix(source, ".tar.gz") || strings.HasSuffix(source, ".tgz") {
		return false
	}
	if filepath.IsAbs(source) || strings.HasPrefix(source, ".") || strings.HasPrefix(source, "~") {
		return false
	}
	if strings.Contains(source, string(os.PathSeparator)) && !strings.HasPrefix(source, "@") {
		return false
	}
	if strings.Contains(source, `\`) {
		return false
	}
	if _, err := os.Stat(source); err == nil {
		return false
	}
	return looksLikeNPMPackageSpec(source)
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

func npmPackageScope(packageName string) string {
	if !strings.HasPrefix(packageName, "@") {
		return ""
	}
	parts := strings.Split(packageName, "/")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[0])
}

func validateRemoteHostPolicy(rawURL string, options InstallOptions) error {
	allowedHosts := normalizeAllowedHosts(options.AllowedHosts)
	if len(allowedHosts) == 0 {
		return nil
	}

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("failed to parse remote source url: %w", err)
	}
	host := normalizeHostName(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("remote source host is required")
	}
	if matchesAnyAllowedHost(host, allowedHosts) {
		return nil
	}
	return fmt.Errorf("plugin source host %s is not allowed by the current install trust policy", host)
}

func validateFinalRemoteDownloadTarget(initialURL string, finalURL string, options InstallOptions) error {
	initialURL = strings.TrimSpace(initialURL)
	finalURL = strings.TrimSpace(finalURL)
	if finalURL == "" {
		return fmt.Errorf("plugin download did not report a final source url")
	}
	if err := validateRemoteHostPolicy(finalURL, options); err != nil {
		return err
	}

	initialParsed, initialErr := url.Parse(initialURL)
	finalParsed, finalErr := url.Parse(finalURL)
	if initialErr != nil || finalErr != nil {
		return nil
	}
	if strings.EqualFold(initialParsed.Scheme, "https") && !strings.EqualFold(finalParsed.Scheme, "https") {
		return fmt.Errorf("plugin download redirected from https to insecure scheme %s", strings.TrimSpace(finalParsed.Scheme))
	}
	return nil
}

func isRemoteSourceType(sourceType string) bool {
	switch strings.TrimSpace(sourceType) {
	case pluginSourceTypeHTTP, pluginSourceTypeHTTPS, pluginSourceTypeNPM:
		return true
	default:
		return false
	}
}

func hostnameFromURL(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	return parsed.Hostname()
}

func normalizeHostName(host string) string {
	return strings.TrimSpace(strings.ToLower(host))
}

func matchesAnyAllowedHost(host string, allowedHosts []string) bool {
	host = normalizeHostName(host)
	for _, allowedHost := range allowedHosts {
		allowedHost = normalizeHostName(allowedHost)
		if allowedHost == "" {
			continue
		}
		switch {
		case allowedHost == "*":
			return true
		case strings.HasPrefix(allowedHost, "*."):
			suffix := strings.TrimPrefix(allowedHost, "*")
			if strings.HasSuffix(host, suffix) && len(host) > len(strings.TrimPrefix(suffix, ".")) {
				return true
			}
		case strings.HasPrefix(allowedHost, "."):
			if strings.HasSuffix(host, allowedHost) && len(host) > len(strings.TrimPrefix(allowedHost, ".")) {
				return true
			}
		case host == allowedHost:
			return true
		}
	}
	return false
}

func matchesAllowedNPMScope(scope string, allowedScopes []string) bool {
	scope = strings.TrimSpace(strings.ToLower(scope))
	for _, allowedScope := range allowedScopes {
		allowedScope = strings.TrimSpace(strings.ToLower(allowedScope))
		switch allowedScope {
		case "*":
			return true
		case "unscoped":
			if scope == "" {
				return true
			}
		default:
			if scope == allowedScope {
				return true
			}
		}
	}
	return false
}

func compareVersionStrings(current string, candidate string) (int, bool) {
	currentParts, currentOK := parseVersionParts(strings.TrimSpace(strings.TrimPrefix(current, "v")))
	candidateParts, candidateOK := parseVersionParts(strings.TrimSpace(strings.TrimPrefix(candidate, "v")))
	if !currentOK || !candidateOK {
		return 0, false
	}

	maxLen := len(currentParts)
	if len(candidateParts) > maxLen {
		maxLen = len(candidateParts)
	}
	for len(currentParts) < maxLen {
		currentParts = append(currentParts, 0)
	}
	for len(candidateParts) < maxLen {
		candidateParts = append(candidateParts, 0)
	}

	for index := 0; index < maxLen; index++ {
		switch {
		case currentParts[index] < candidateParts[index]:
			return -1, true
		case currentParts[index] > candidateParts[index]:
			return 1, true
		}
	}
	return 0, true
}

func upgradeNoChangeReason(source string, comparable bool) string {
	if isPinnedSourceReference(source) {
		return "recorded source is pinned to the installed plugin version"
	}
	if comparable {
		return "recorded source currently resolves to the installed plugin version"
	}
	return "recorded source resolves to the same plugin version"
}

func isPinnedSourceReference(source string) bool {
	switch classifyInstallSource(source) {
	case pluginSourceTypeNPM:
		spec := strings.TrimSpace(strings.TrimPrefix(source, "npm://"))
		if spec == "" {
			spec = strings.TrimSpace(source)
		}
		_, version := parseNPMPackageSpec(spec)
		return strings.TrimSpace(version) != ""
	case pluginSourceTypeRegistry:
		_, _, version, err := parseRegistryInstallSpec(source)
		return err == nil && strings.TrimSpace(version) != ""
	default:
		return false
	}
}

func looksLikeNPMPackageSpec(spec string) bool {
	packageName, _ := parseNPMPackageSpec(spec)
	if packageName == "" {
		return false
	}

	if strings.HasPrefix(packageName, "@") {
		parts := strings.Split(packageName, "/")
		if len(parts) != 2 {
			return false
		}
		scope := strings.TrimPrefix(parts[0], "@")
		return isValidNPMPackageNamePart(scope) && isValidNPMPackageNamePart(parts[1])
	}

	if strings.Contains(packageName, "/") {
		return false
	}
	return isValidNPMPackageNamePart(packageName)
}

func isValidNPMPackageNamePart(part string) bool {
	if part == "" {
		return false
	}

	for _, ch := range part {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.', ch == '~':
		default:
			return false
		}
	}
	return true
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
		if relative != "." && shouldSkipPackagedArtifactPath(relative) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
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

func checksumFile(path string) (string, error) {
	sum, err := digestFile(path, sha256.New)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum), nil
}

func verifyDownloadedArtifactChecksum(path string, expected string) error {
	expected = strings.TrimSpace(strings.ToLower(expected))
	if expected == "" {
		return nil
	}

	actual, err := checksumFile(path)
	if err != nil {
		return fmt.Errorf("failed to checksum downloaded plugin artifact: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("downloaded plugin artifact checksum does not match registry metadata")
	}
	return nil
}

func verifyDownloadedNPMArtifact(path string, integrity string, shasum string) error {
	integrity = strings.TrimSpace(integrity)
	if integrity != "" {
		if err := verifySubresourceIntegrity(path, integrity); err != nil {
			return err
		}
	}

	shasum = strings.TrimSpace(strings.ToLower(shasum))
	if shasum == "" {
		return nil
	}

	actual, err := digestFile(path, sha1.New)
	if err != nil {
		return fmt.Errorf("failed to checksum downloaded npm plugin artifact: %w", err)
	}
	if hex.EncodeToString(actual) != shasum {
		return fmt.Errorf("downloaded npm plugin artifact shasum does not match registry metadata")
	}
	return nil
}

func verifySubresourceIntegrity(path string, expected string) error {
	tokens := strings.Fields(expected)
	if len(tokens) == 0 {
		return nil
	}

	supportedChecks := 0
	for _, token := range tokens {
		parts := strings.SplitN(strings.TrimSpace(token), "-", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			continue
		}

		actual, supported, err := checksumFileBase64(path, parts[0])
		if err != nil {
			return err
		}
		if !supported {
			continue
		}

		supportedChecks++
		if actual == strings.TrimSpace(parts[1]) {
			return nil
		}
	}

	if supportedChecks == 0 {
		return fmt.Errorf("downloaded npm plugin artifact uses unsupported integrity metadata")
	}
	return fmt.Errorf("downloaded npm plugin artifact integrity does not match registry metadata")
}

func checksumFileBase64(path string, algorithm string) (string, bool, error) {
	var newHash func() hash.Hash
	switch strings.ToLower(strings.TrimSpace(algorithm)) {
	case "sha256":
		newHash = sha256.New
	case "sha384":
		newHash = sha512.New384
	case "sha512":
		newHash = sha512.New
	default:
		return "", false, nil
	}

	sum, err := digestFile(path, newHash)
	if err != nil {
		return "", true, fmt.Errorf("failed to checksum downloaded npm plugin artifact: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sum), true, nil
}

func digestFile(path string, newHash func() hash.Hash) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	digest := newHash()
	if _, err := io.Copy(digest, file); err != nil {
		return nil, err
	}
	return digest.Sum(nil), nil
}

func buildPreUpgradeVerificationError(result VerifyResult) string {
	if len(result.Issues) > 0 {
		return strings.Join(result.Issues, "; ")
	}
	return "installed plugin failed pre-upgrade verification"
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
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("plugin archive contains invalid path: %s", header.Name)
		}
		if shouldSkipPackagedArtifactPath(cleanName) {
			continue
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

func shouldSkipPackagedArtifactPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return false
	}

	// 忽略 macOS 归档里常见的 AppleDouble 和 Finder 元数据目录，
	// 避免第三方发布物把宿主环境无关的噪音文件带进安装目录。
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for _, part := range parts {
		switch {
		case part == "__MACOSX":
			return true
		case strings.HasPrefix(part, "._"):
			return true
		}
	}
	return false
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

func canonicalRegistrySource(sourceID string, reference string, version string) string {
	sourceID = strings.TrimSpace(sourceID)
	reference = strings.Trim(strings.TrimSpace(reference), "/")
	version = strings.TrimSpace(version)

	var builder strings.Builder
	builder.WriteString("registry://")
	if sourceID != "" {
		builder.WriteString(sourceID)
		builder.WriteString("/")
	}
	builder.WriteString(reference)
	if version != "" {
		builder.WriteString("@")
		builder.WriteString(version)
	}
	return builder.String()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
