package plugin

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
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
