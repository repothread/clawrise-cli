package plugin

// SplitManifestsByKind separates manifests by plugin kind.
func SplitManifestsByKind(manifests []Manifest) ([]Manifest, []Manifest, []Manifest) {
	providers := make([]Manifest, 0)
	launchers := make([]Manifest, 0)
	storageBackends := make([]Manifest, 0)
	for _, manifest := range manifests {
		if manifest.SupportsKind(ManifestKindProvider) {
			providers = append(providers, manifest)
		}
		if manifest.SupportsKind(ManifestKindAuthLauncher) {
			launchers = append(launchers, manifest)
		}
		if manifest.SupportsKind(ManifestKindStorageBackend) {
			storageBackends = append(storageBackends, manifest)
		}
	}
	return providers, launchers, storageBackends
}
