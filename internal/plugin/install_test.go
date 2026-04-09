package plugin

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallListAndRemoveLocalDirectory(t *testing.T) {
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
	if result.Manifest.Name != "demo" {
		t.Fatalf("unexpected install manifest: %+v", result.Manifest)
	}

	items, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "demo" {
		t.Fatalf("unexpected installed plugins: %+v", items)
	}
	if items[0].Install == nil || items[0].Install.Source == "" || items[0].Install.ChecksumSHA == "" {
		t.Fatalf("expected install metadata to be recorded, got: %+v", items[0].Install)
	}

	info, err := InfoInstalled("demo", "0.1.0")
	if err != nil {
		t.Fatalf("InfoInstalled returned error: %v", err)
	}
	if info.Install == nil || info.Install.ChecksumSHA == "" {
		t.Fatalf("expected info install metadata, got: %+v", info.Install)
	}

	removed, err := RemoveInstalled("demo", "0.1.0")
	if err != nil {
		t.Fatalf("RemoveInstalled returned error: %v", err)
	}
	if removed.Name != "demo" {
		t.Fatalf("unexpected remove result: %+v", removed)
	}
}

func TestInstallLocalTarGz(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	result, err := InstallLocal(archivePath)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}
	if result.Manifest.Name != "demo" || result.Manifest.Version != "0.2.0" {
		t.Fatalf("unexpected archive install result: %+v", result)
	}
}

func TestInstallLocalTarGzSkipsMacOSMetadataArtifacts(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchiveWithMacOSMetadata(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive with macOS metadata: %v", err)
	}

	result, err := InstallLocal(archivePath)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	assertInstalledPluginHasNoMacOSMetadataArtifacts(t, result.Path)
}

func TestInstallLocalDirectorySkipsMacOSMetadataArtifacts(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.3.0", ""); err != nil {
		t.Fatalf("failed to write plugin source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "__MACOSX"), 0o755); err != nil {
		t.Fatalf("failed to create __MACOSX dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "._plugin.json"), []byte("metadata"), 0o644); err != nil {
		t.Fatalf("failed to write AppleDouble manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "__MACOSX", "._plugin.json"), []byte("metadata"), 0o644); err != nil {
		t.Fatalf("failed to write __MACOSX AppleDouble manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "._demo-plugin"), []byte("metadata"), 0o644); err != nil {
		t.Fatalf("failed to write AppleDouble binary: %v", err)
	}

	result, err := InstallLocal(sourceDir)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	assertInstalledPluginHasNoMacOSMetadataArtifacts(t, result.Path)
}

func TestInstallHTTPSupport(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read test archive: %v", err)
	}

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/gzip"},
				},
				Body: io.NopCloser(strings.NewReader(string(data))),
			}, nil
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	result, err := Install("https://plugins.example.com/demo-plugin.tar.gz")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Install == nil || result.Install.Source != "https://plugins.example.com/demo-plugin.tar.gz" {
		t.Fatalf("unexpected install metadata: %+v", result.Install)
	}
}

func TestInstallRejectsHTTPByDefault(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	_, err := Install("http://plugins.example.com/demo-plugin.tar.gz")
	if err == nil {
		t.Fatal("expected install to reject http source by default")
	}
	if !strings.Contains(err.Error(), "install trust policy") {
		t.Fatalf("expected trust policy error, got: %v", err)
	}
}

func TestInstallAllowsHTTPWhenTrustPolicyIncludesHTTP(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read test archive: %v", err)
	}

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/gzip"},
				},
				Body: io.NopCloser(strings.NewReader(string(data))),
			}, nil
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	result, err := InstallWithOptions("http://plugins.example.com/demo-plugin.tar.gz", InstallOptions{
		AllowedSources: []string{"http"},
	})
	if err != nil {
		t.Fatalf("InstallWithOptions returned error: %v", err)
	}
	if result.Install == nil || result.Install.Source != "http://plugins.example.com/demo-plugin.tar.gz" {
		t.Fatalf("unexpected install metadata: %+v", result.Install)
	}
}

func TestInstallRejectsRemoteHostOutsideAllowlist(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	_, err := InstallWithOptions("https://plugins.example.com/demo-plugin.tar.gz", InstallOptions{
		AllowedHosts: []string{"downloads.example.com"},
	})
	if err == nil {
		t.Fatal("expected install to reject remote host outside allowlist")
	}
	if !strings.Contains(err.Error(), "source host") {
		t.Fatalf("expected host trust policy error, got: %v", err)
	}
}

func TestInstallRejectsNPMScopeOutsideAllowlist(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	_, err := InstallWithOptions("npm://@forbidden/demo-plugin", InstallOptions{
		AllowedNPMScopes: []string{"@clawrise"},
	})
	if err == nil {
		t.Fatal("expected install to reject npm scope outside allowlist")
	}
	if !strings.Contains(err.Error(), "npm package scope") {
		t.Fatalf("expected npm scope trust policy error, got: %v", err)
	}
}

func TestInstallRejectsPluginThatRequiresNewerCore(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.1.0", ",\n  \"min_core_version\": \"9.0.0\""); err != nil {
		t.Fatalf("failed to write plugin source dir: %v", err)
	}

	_, err := InstallWithOptions(sourceDir, InstallOptions{
		CoreVersion: "0.2.0",
	})
	if err == nil {
		t.Fatal("expected install to reject plugin with newer min_core_version")
	}
	if !strings.Contains(err.Error(), "requires core version") {
		t.Fatalf("expected min_core_version error, got: %v", err)
	}
}

func TestUpgradeInstalledReinstallsFromRecordedSourceAndRemovesPreviousVersion(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.1.0", ""); err != nil {
		t.Fatalf("failed to write initial plugin source dir: %v", err)
	}

	if _, err := InstallLocal(sourceDir); err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.2.0", ""); err != nil {
		t.Fatalf("failed to rewrite plugin source dir: %v", err)
	}

	result, err := UpgradeInstalled("demo", "0.1.0", InstallOptions{
		CoreVersion: "0.2.0",
	})
	if err != nil {
		t.Fatalf("UpgradeInstalled returned error: %v", err)
	}
	if !result.Upgraded || result.ToVersion != "0.2.0" || !result.RemovedPrevious {
		t.Fatalf("unexpected upgrade result: %+v", result)
	}

	items, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled returned error: %v", err)
	}
	if len(items) != 1 || items[0].Version != "0.2.0" {
		t.Fatalf("expected only upgraded version to remain installed, got: %+v", items)
	}
}

func TestUpgradeInstalledRejectsWhenCurrentPluginFailsVerification(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.1.0", ""); err != nil {
		t.Fatalf("failed to write initial plugin source dir: %v", err)
	}

	installed, err := InstallLocal(sourceDir)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(installed.Path, "bin", "demo-plugin"), []byte("#!/bin/sh\necho mutated\n"), 0o755); err != nil {
		t.Fatalf("failed to mutate installed plugin: %v", err)
	}
	if err := writeTestPluginSourceDir(sourceDir, "demo", "0.2.0", ""); err != nil {
		t.Fatalf("failed to rewrite plugin source dir: %v", err)
	}

	result, err := UpgradeInstalled("demo", "0.1.0", InstallOptions{
		CoreVersion: "0.2.0",
	})
	if err == nil {
		t.Fatal("expected upgrade to stop when current plugin fails verification")
	}
	if !strings.Contains(err.Error(), "pre-upgrade verification") {
		t.Fatalf("expected pre-upgrade verification error, got: %v", err)
	}
	if result.Preflight == nil || result.Preflight.Verified {
		t.Fatalf("expected upgrade result to expose failed preflight verification, got: %+v", result)
	}
	if result.Reason != "installed plugin failed pre-upgrade verification" {
		t.Fatalf("unexpected upgrade rejection reason: %+v", result)
	}
}

func TestInstallRegistrySourceSupport(t *testing.T) {
	homeDir := t.TempDir()
	pluginRoot := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)

	archivePath := filepath.Join(t.TempDir(), "workflow-demo-0.2.0.tgz")
	if err := writeTestPluginArchiveWithManifest(archivePath, "workflow-demo", "0.2.0"); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read plugin archive: %v", err)
	}
	archiveChecksum, err := checksumFile(archivePath)
	if err != nil {
		t.Fatalf("failed to checksum plugin archive: %v", err)
	}

	registryDir := filepath.Join(pluginRoot, "registry-demo", "0.1.0")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("failed to create registry plugin dir: %v", err)
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.registry_source.resolve"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"name":"workflow-demo","version":"0.2.0","artifact_url":"https://downloads.example.com/workflow-demo-0.2.0.tgz","checksum_sha256":"` + archiveChecksum + `"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(registryDir, "registry-demo.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write registry plugin script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "registry-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "registry_source",
      "id": "community",
      "priority": 50
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./registry-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write registry plugin manifest: %v", err)
	}

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.String() != "https://downloads.example.com/workflow-demo-0.2.0.tgz" {
				t.Fatalf("unexpected registry download url: %s", request.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/gzip"},
				},
				Body: io.NopCloser(strings.NewReader(string(archiveData))),
			}, nil
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	result, err := InstallWithOptions("registry://community/workflow-demo", InstallOptions{
		AllowedHosts: []string{"downloads.example.com"},
	})
	if err != nil {
		t.Fatalf("InstallWithOptions returned error: %v", err)
	}
	if result.Manifest.Name != "workflow-demo" || result.Manifest.Version != "0.2.0" {
		t.Fatalf("unexpected registry install result: %+v", result)
	}
	if result.Install == nil || result.Install.Source != "registry://community/workflow-demo" {
		t.Fatalf("unexpected registry install metadata: %+v", result.Install)
	}
	if result.Install.ArtifactURL != "https://downloads.example.com/workflow-demo-0.2.0.tgz" {
		t.Fatalf("expected artifact url to be recorded, got: %+v", result.Install)
	}
}

func TestUpgradeInstalledFromRegistrySource(t *testing.T) {
	homeDir := t.TempDir()
	pluginRoot := t.TempDir()
	controlPath := filepath.Join(t.TempDir(), "registry-version.txt")
	t.Setenv("HOME", homeDir)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)

	if err := os.WriteFile(controlPath, []byte("0.1.0"), 0o644); err != nil {
		t.Fatalf("failed to seed control file: %v", err)
	}

	archiveV1Path := filepath.Join(t.TempDir(), "workflow-demo-0.1.0.tgz")
	archiveV2Path := filepath.Join(t.TempDir(), "workflow-demo-0.2.0.tgz")
	if err := writeTestPluginArchiveWithManifest(archiveV1Path, "workflow-demo", "0.1.0"); err != nil {
		t.Fatalf("failed to write plugin v1 archive: %v", err)
	}
	if err := writeTestPluginArchiveWithManifest(archiveV2Path, "workflow-demo", "0.2.0"); err != nil {
		t.Fatalf("failed to write plugin v2 archive: %v", err)
	}
	archiveV1Data, err := os.ReadFile(archiveV1Path)
	if err != nil {
		t.Fatalf("failed to read plugin v1 archive: %v", err)
	}
	archiveV2Data, err := os.ReadFile(archiveV2Path)
	if err != nil {
		t.Fatalf("failed to read plugin v2 archive: %v", err)
	}
	archiveV1Checksum, err := checksumFile(archiveV1Path)
	if err != nil {
		t.Fatalf("failed to checksum plugin v1 archive: %v", err)
	}
	archiveV2Checksum, err := checksumFile(archiveV2Path)
	if err != nil {
		t.Fatalf("failed to checksum plugin v2 archive: %v", err)
	}

	registryDir := filepath.Join(pluginRoot, "registry-demo", "0.1.0")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("failed to create registry plugin dir: %v", err)
	}
	script := `#!/bin/sh
state_file="` + controlPath + `"
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.registry_source.resolve"'*)
      version=$(cat "$state_file")
      case "$version" in
        "0.1.0")
          url="https://downloads.example.com/workflow-demo-0.1.0.tgz"
          checksum="` + archiveV1Checksum + `"
          ;;
        *)
          url="https://downloads.example.com/workflow-demo-0.2.0.tgz"
          checksum="` + archiveV2Checksum + `"
          version="0.2.0"
          ;;
      esac
      printf '{"jsonrpc":"2.0","id":"1","result":{"name":"workflow-demo","version":"%s","artifact_url":"%s","checksum_sha256":"%s"}}'"\n" "$version" "$url" "$checksum"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(registryDir, "registry-demo.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write registry plugin script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "registry-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "registry_source",
      "id": "community",
      "priority": 50
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./registry-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write registry plugin manifest: %v", err)
	}

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case "https://downloads.example.com/workflow-demo-0.1.0.tgz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/gzip"},
					},
					Body: io.NopCloser(strings.NewReader(string(archiveV1Data))),
				}, nil
			case "https://downloads.example.com/workflow-demo-0.2.0.tgz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/gzip"},
					},
					Body: io.NopCloser(strings.NewReader(string(archiveV2Data))),
				}, nil
			default:
				t.Fatalf("unexpected registry download url: %s", request.URL.String())
				return nil, nil
			}
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	if _, err := InstallWithOptions("registry://community/workflow-demo", InstallOptions{
		AllowedHosts: []string{"downloads.example.com"},
	}); err != nil {
		t.Fatalf("failed to install registry-backed plugin: %v", err)
	}

	if err := os.WriteFile(controlPath, []byte("0.2.0"), 0o644); err != nil {
		t.Fatalf("failed to switch registry control file: %v", err)
	}

	result, err := UpgradeInstalled("workflow-demo", "0.1.0", InstallOptions{
		CoreVersion:  "0.2.0",
		AllowedHosts: []string{"downloads.example.com"},
	})
	if err != nil {
		t.Fatalf("UpgradeInstalled returned error: %v", err)
	}
	if !result.Upgraded || result.ToVersion != "0.2.0" || !result.RemovedPrevious {
		t.Fatalf("unexpected registry upgrade result: %+v", result)
	}

	info, err := InfoInstalled("workflow-demo", "0.2.0")
	if err != nil {
		t.Fatalf("failed to load upgraded plugin info: %v", err)
	}
	if info.Install == nil || info.Install.Source != "registry://community/workflow-demo" {
		t.Fatalf("expected registry source to remain floating and source-pinned, got: %+v", info.Install)
	}
}

func TestUpgradeAllInstalledReturnsPerPluginResults(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDirA := filepath.Join(t.TempDir(), "plugin-src-a")
	sourceDirB := filepath.Join(t.TempDir(), "plugin-src-b")
	if err := writeTestPluginSourceDir(sourceDirA, "demo-a", "0.1.0", ""); err != nil {
		t.Fatalf("failed to write initial plugin A source dir: %v", err)
	}
	if err := writeTestPluginSourceDir(sourceDirB, "demo-b", "0.1.0", ""); err != nil {
		t.Fatalf("failed to write initial plugin B source dir: %v", err)
	}

	if _, err := InstallLocal(sourceDirA); err != nil {
		t.Fatalf("InstallLocal returned error for plugin A: %v", err)
	}
	if _, err := InstallLocal(sourceDirB); err != nil {
		t.Fatalf("InstallLocal returned error for plugin B: %v", err)
	}

	if err := writeTestPluginSourceDir(sourceDirA, "demo-a", "0.2.0", ""); err != nil {
		t.Fatalf("failed to rewrite plugin A source dir: %v", err)
	}

	results, err := UpgradeAllInstalled(InstallOptions{
		CoreVersion: "0.2.0",
	})
	if err != nil {
		t.Fatalf("UpgradeAllInstalled returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two upgrade results, got: %+v", results)
	}

	var upgraded UpgradeResult
	var unchanged UpgradeResult
	for _, item := range results {
		switch item.Name {
		case "demo-a":
			upgraded = item
		case "demo-b":
			unchanged = item
		}
	}
	if !upgraded.Upgraded || upgraded.ToVersion != "0.2.0" {
		t.Fatalf("expected plugin A to upgrade, got: %+v", upgraded)
	}
	if unchanged.Upgraded || !unchanged.Checked || unchanged.Reason == "" {
		t.Fatalf("expected plugin B to be checked but left unchanged, got: %+v", unchanged)
	}
}

func TestInstallStorageBackendPlugin(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "storage-plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "secret-demo",
  "version": "0.1.0",
  "kind": "storage_backend",
  "protocol_version": 1,
  "storage_backend": {
    "target": "secret_store",
    "backend": "plugin.demo_secret",
    "display_name": "Demo Secret Store"
  },
  "entry": {
    "type": "binary",
    "command": ["./bin/secret-demo"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "secret-demo"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	result, err := InstallLocal(sourceDir)
	if err != nil {
		t.Fatalf("InstallLocal returned error: %v", err)
	}
	if result.Manifest.Kind != ManifestKindStorageBackend {
		t.Fatalf("unexpected storage backend install result: %+v", result.Manifest)
	}
	if result.Manifest.StorageBackend == nil || result.Manifest.StorageBackend.Backend != "plugin.demo_secret" {
		t.Fatalf("unexpected storage backend descriptor: %+v", result.Manifest.StorageBackend)
	}

	items, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled returned error: %v", err)
	}
	if len(items) != 1 || items[0].Kind != ManifestKindStorageBackend {
		t.Fatalf("unexpected installed storage backend plugins: %+v", items)
	}
	if items[0].StorageBackend == nil || items[0].StorageBackend.Target != "secret_store" {
		t.Fatalf("unexpected installed storage backend metadata: %+v", items[0])
	}
}

func TestListAndInfoInstalledWithOptionsExposeSelectionAndRuntimeCapabilities(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo-provider",
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
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"provider","platforms":["demo"]}]}}'"\n"
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

	options := DiscoveryOptions{
		ProviderBindings: map[string]string{
			"demo": "demo-provider",
		},
	}

	items, err := ListInstalledWithOptions(options)
	if err != nil {
		t.Fatalf("ListInstalledWithOptions returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected installed plugins: %+v", items)
	}
	if !items[0].Selected || len(items[0].MatchedProviderBindings) != 1 || items[0].MatchedProviderBindings[0] != "demo" {
		t.Fatalf("expected provider binding hit, got: %+v", items[0])
	}

	info, err := InfoInstalledWithOptions("demo-provider", "0.1.0", options)
	if err != nil {
		t.Fatalf("InfoInstalledWithOptions returned error: %v", err)
	}
	if len(info.RuntimeCapabilities) != 1 || info.RuntimeCapabilities[0].Type != CapabilityTypeProvider {
		t.Fatalf("unexpected runtime capabilities: %+v", info)
	}
	if len(info.MatchedProviderBindings) != 1 || info.MatchedProviderBindings[0] != "demo" {
		t.Fatalf("unexpected provider binding hits: %+v", info.MatchedProviderBindings)
	}
	if len(info.Warnings) != 0 {
		t.Fatalf("expected no runtime capability warnings, got: %+v", info.Warnings)
	}
}

func TestInstallNPMSupport(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read npm test archive: %v", err)
	}
	integrity := buildTestSRI(archiveData)
	shasum := buildTestSHA1Hex(archiveData)

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case "https://registry.example.com/@clawrise%2Fclawrise-plugin-demo":
				payload, err := json.Marshal(map[string]any{
					"dist-tags": map[string]any{
						"latest": "0.2.0",
					},
					"versions": map[string]any{
						"0.2.0": map[string]any{
							"dist": map[string]any{
								"tarball":   "https://registry.example.com/tarballs/demo-plugin.tar.gz",
								"integrity": integrity,
								"shasum":    shasum,
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("failed to encode npm metadata payload: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: io.NopCloser(strings.NewReader(string(payload))),
				}, nil
			case "https://registry.example.com/tarballs/demo-plugin.tar.gz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/gzip"},
					},
					Body: io.NopCloser(strings.NewReader(string(archiveData))),
				}, nil
			default:
				t.Fatalf("unexpected npm test url: %s", request.URL.String())
				return nil, nil
			}
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	previousRegistryURL := npmRegistryBaseURL
	npmRegistryBaseURL = "https://registry.example.com"
	defer func() {
		npmRegistryBaseURL = previousRegistryURL
	}()

	result, err := Install("npm://@clawrise/clawrise-plugin-demo")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Manifest.Name != "demo" || result.Install == nil || result.Install.Source != "npm://@clawrise/clawrise-plugin-demo" {
		t.Fatalf("unexpected npm install result: %+v", result)
	}
	if result.Install.ArtifactURL != "https://registry.example.com/tarballs/demo-plugin.tar.gz" {
		t.Fatalf("expected npm install to record artifact url, got: %+v", result.Install)
	}
}

func TestInstallAcceptsBareNPMPackageSpec(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read npm test archive: %v", err)
	}
	integrity := buildTestSRI(archiveData)
	shasum := buildTestSHA1Hex(archiveData)

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case "https://registry.example.com/@clawrise%2Fclawrise-plugin-demo":
				payload, err := json.Marshal(map[string]any{
					"dist-tags": map[string]any{
						"latest": "0.2.0",
					},
					"versions": map[string]any{
						"0.2.0": map[string]any{
							"dist": map[string]any{
								"tarball":   "https://registry.example.com/tarballs/demo-plugin.tar.gz",
								"integrity": integrity,
								"shasum":    shasum,
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("failed to encode npm metadata payload: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: io.NopCloser(strings.NewReader(string(payload))),
				}, nil
			case "https://registry.example.com/tarballs/demo-plugin.tar.gz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/gzip"},
					},
					Body: io.NopCloser(strings.NewReader(string(archiveData))),
				}, nil
			default:
				t.Fatalf("unexpected npm test url: %s", request.URL.String())
				return nil, nil
			}
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	previousRegistryURL := npmRegistryBaseURL
	npmRegistryBaseURL = "https://registry.example.com"
	defer func() {
		npmRegistryBaseURL = previousRegistryURL
	}()

	result, err := Install("@clawrise/clawrise-plugin-demo")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Manifest.Name != "demo" || result.Install == nil || result.Install.Source != "@clawrise/clawrise-plugin-demo" {
		t.Fatalf("unexpected bare npm install result: %+v", result)
	}
}

func TestInstallRejectsNPMArtifactIntegrityMismatch(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	archivePath := filepath.Join(t.TempDir(), "demo-plugin.tar.gz")
	if err := writeTestPluginArchive(archivePath); err != nil {
		t.Fatalf("failed to write plugin archive: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read npm test archive: %v", err)
	}

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case "https://registry.example.com/@clawrise%2Fclawrise-plugin-demo":
				payload, err := json.Marshal(map[string]any{
					"dist-tags": map[string]any{
						"latest": "0.2.0",
					},
					"versions": map[string]any{
						"0.2.0": map[string]any{
							"dist": map[string]any{
								"tarball":   "https://registry.example.com/tarballs/demo-plugin.tar.gz",
								"integrity": "sha512-invalid",
								"shasum":    buildTestSHA1Hex(archiveData),
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("failed to encode npm metadata payload: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: io.NopCloser(strings.NewReader(string(payload))),
				}, nil
			case "https://registry.example.com/tarballs/demo-plugin.tar.gz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/gzip"},
					},
					Body: io.NopCloser(strings.NewReader(string(archiveData))),
				}, nil
			default:
				t.Fatalf("unexpected npm test url: %s", request.URL.String())
				return nil, nil
			}
		}),
	}
	defer func() {
		pluginDownloadHTTPClient = previousClient
	}()

	previousRegistryURL := npmRegistryBaseURL
	npmRegistryBaseURL = "https://registry.example.com"
	defer func() {
		npmRegistryBaseURL = previousRegistryURL
	}()

	_, err = Install("npm://@clawrise/clawrise-plugin-demo")
	if err == nil {
		t.Fatal("expected npm install to reject mismatched artifact integrity")
	}
	if !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity error, got: %v", err)
	}
}

func TestValidateFinalRemoteDownloadTargetRejectsRedirectedHostOutsideAllowlist(t *testing.T) {
	err := validateFinalRemoteDownloadTarget(
		"https://plugins.example.com/demo-plugin.tar.gz",
		"https://cdn.example.com/demo-plugin.tar.gz",
		InstallOptions{AllowedHosts: []string{"plugins.example.com"}},
	)
	if err == nil {
		t.Fatal("expected redirected download target to be rejected")
	}
	if !strings.Contains(err.Error(), "source host") {
		t.Fatalf("expected redirected host trust error, got: %v", err)
	}
}

func TestValidateFinalRemoteDownloadTargetRejectsHTTPSDowngrade(t *testing.T) {
	err := validateFinalRemoteDownloadTarget(
		"https://plugins.example.com/demo-plugin.tar.gz",
		"http://plugins.example.com/demo-plugin.tar.gz",
		InstallOptions{},
	)
	if err == nil {
		t.Fatal("expected https redirect downgrade to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure scheme") {
		t.Fatalf("expected scheme downgrade error, got: %v", err)
	}
}

func TestListAndInfoInstalledExposeCapabilityRoutes(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "demo-governance",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review"
    },
    {
      "type": "audit_sink",
      "id": "capture"
    }
  ],
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
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"policy","id":"review"},{"type":"audit_sink","id":"capture"}]}}'"\n"
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

	options := DiscoveryOptions{
		PolicyMode: "manual",
		PolicySelectors: []PolicyCapabilitySelector{
			{Plugin: "demo-governance", PolicyID: "review"},
		},
		AuditMode: "manual",
	}

	items, err := ListInstalledWithOptions(options)
	if err != nil {
		t.Fatalf("ListInstalledWithOptions returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected installed plugins: %+v", items)
	}
	if len(items[0].CapabilityRoutes) != 2 {
		t.Fatalf("expected two capability routes, got: %+v", items[0].CapabilityRoutes)
	}
	policyRoute := findCapabilityRoute(t, items[0].CapabilityRoutes, CapabilityTypePolicy, "review")
	if !policyRoute.Active || policyRoute.Source != "configured" {
		t.Fatalf("unexpected policy route: %+v", policyRoute)
	}
	auditRoute := findCapabilityRoute(t, items[0].CapabilityRoutes, CapabilityTypeAuditSink, "capture")
	if auditRoute.Active || auditRoute.Reason != "manual_mode_without_sink_selector" {
		t.Fatalf("unexpected audit route: %+v", auditRoute)
	}

	info, err := InfoInstalledWithOptions("demo-governance", "0.1.0", options)
	if err != nil {
		t.Fatalf("InfoInstalledWithOptions returned error: %v", err)
	}
	if len(info.CapabilityRoutes) != 2 {
		t.Fatalf("expected two capability routes in info, got: %+v", info.CapabilityRoutes)
	}
}

func findCapabilityRoute(t *testing.T, routes []CapabilityRouteStatus, capabilityType string, id string) CapabilityRouteStatus {
	t.Helper()

	for _, route := range routes {
		if route.Type == capabilityType && route.ID == id {
			return route
		}
	}
	t.Fatalf("failed to find capability route %s/%s in %+v", capabilityType, id, routes)
	return CapabilityRouteStatus{}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func writeTestPluginArchive(path string) error {
	return writeTestPluginArchiveWithManifest(path, "demo", "0.2.0")
}

func writeTestPluginArchiveWithManifest(path string, name string, version string) error {
	files := map[string]string{
		"demo-plugin/plugin.json": `{
  "schema_version": 1,
  "name": "` + name + `",
  "version": "` + version + `",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`,
		"demo-plugin/bin/demo-plugin": "#!/bin/sh\n",
	}
	return writeTestPluginArchiveEntries(path, files)
}

func writeTestPluginArchiveWithMacOSMetadata(path string) error {
	files := map[string]string{
		"demo-plugin/plugin.json": `{
  "schema_version": 1,
  "name": "demo",
  "version": "0.2.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`,
		"demo-plugin/bin/demo-plugin":         "#!/bin/sh\n",
		"demo-plugin/._plugin.json":           "metadata",
		"demo-plugin/bin/._demo-plugin":       "metadata",
		"__MACOSX/demo-plugin/._plugin.json":  "metadata",
		"__MACOSX/demo-plugin/._demo-plugin":  "metadata",
		"__MACOSX/demo-plugin/._README.md":    "metadata",
		"demo-plugin/__MACOSX/._ignored-file": "metadata",
	}
	return writeTestPluginArchiveEntries(path, files)
}

func writeTestPluginArchiveEntries(path string, files map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func writeTestPluginSourceDir(path string, name string, version string, extraFields string) error {
	if err := os.MkdirAll(filepath.Join(path, "bin"), 0o755); err != nil {
		return err
	}

	manifest := `{
  "schema_version": 1,
  "name": "` + name + `",
  "version": "` + version + `",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }` + extraFields + `
}`
	if err := os.WriteFile(filepath.Join(path, "plugin.json"), []byte(manifest), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755)
}

func assertInstalledPluginHasNoMacOSMetadataArtifacts(t *testing.T, root string) {
	t.Helper()

	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if shouldSkipPackagedArtifactPath(relative) {
			t.Fatalf("unexpected macOS metadata artifact in installed plugin: %s", relative)
		}
		return nil
	}); err != nil {
		t.Fatalf("failed to inspect installed plugin tree: %v", err)
	}
}

func buildTestSRI(data []byte) string {
	sum := sha512.Sum512(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
}

func buildTestSHA1Hex(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:])
}
