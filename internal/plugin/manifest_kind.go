package plugin

// SplitManifestsByKind 按 manifest kind 拆分 plugin 清单。
func SplitManifestsByKind(manifests []Manifest) ([]Manifest, []Manifest) {
	providers := make([]Manifest, 0)
	launchers := make([]Manifest, 0)
	for _, manifest := range manifests {
		switch manifest.Kind {
		case ManifestKindProvider:
			providers = append(providers, manifest)
		case ManifestKindAuthLauncher:
			launchers = append(launchers, manifest)
		}
	}
	return providers, launchers
}
