package plugin

// SplitManifestsByKind separates manifests by plugin kind.
func SplitManifestsByKind(manifests []Manifest) ([]Manifest, []Manifest, []Manifest) {
	providers := make([]Manifest, 0)
	launchers := make([]Manifest, 0)
	storageBackends := make([]Manifest, 0)
	for _, manifest := range manifests {
		switch manifest.Kind {
		case ManifestKindProvider:
			providers = append(providers, manifest)
		case ManifestKindAuthLauncher:
			launchers = append(launchers, manifest)
		case ManifestKindStorageBackend:
			storageBackends = append(storageBackends, manifest)
		}
	}
	return providers, launchers, storageBackends
}
