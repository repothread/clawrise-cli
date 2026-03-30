package plugin

import (
	"archive/tar"
	"compress/gzip"
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

	previousClient := pluginDownloadHTTPClient
	pluginDownloadHTTPClient = &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.String() {
			case "https://registry.example.com/@clawrise%2Fplugin-demo":
				payload, err := json.Marshal(map[string]any{
					"dist-tags": map[string]any{
						"latest": "0.2.0",
					},
					"versions": map[string]any{
						"0.2.0": map[string]any{
							"dist": map[string]any{
								"tarball": "https://registry.example.com/tarballs/demo-plugin.tar.gz",
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

	result, err := Install("npm://@clawrise/plugin-demo")
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if result.Manifest.Name != "demo" || result.Install == nil || result.Install.Source != "npm://@clawrise/plugin-demo" {
		t.Fatalf("unexpected npm install result: %+v", result)
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func writeTestPluginArchive(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

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
		"demo-plugin/bin/demo-plugin": "#!/bin/sh\n",
	}

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
