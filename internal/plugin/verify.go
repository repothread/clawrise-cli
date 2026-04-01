package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// VerifyResult describes one installed plugin verification result.
type VerifyResult struct {
	Name                  string                 `json:"name"`
	Version               string                 `json:"version"`
	Kind                  string                 `json:"kind,omitempty"`
	Capabilities          []CapabilityDescriptor `json:"capabilities,omitempty"`
	RuntimeCapabilities   []CapabilityDescriptor `json:"runtime_capabilities,omitempty"`
	Path                  string                 `json:"path"`
	Source                string                 `json:"source,omitempty"`
	SourceType            string                 `json:"source_type,omitempty"`
	ExpectedChecksumSHA   string                 `json:"expected_checksum_sha256,omitempty"`
	ActualChecksumSHA     string                 `json:"actual_checksum_sha256,omitempty"`
	ChecksumMatch         bool                   `json:"checksum_match"`
	ProtocolVersion       int                    `json:"protocol_version"`
	ProtocolCompatible    bool                   `json:"protocol_compatible"`
	MinCoreVersion        string                 `json:"min_core_version,omitempty"`
	CoreVersion           string                 `json:"core_version,omitempty"`
	CoreVersionChecked    bool                   `json:"core_version_checked"`
	CoreVersionCompatible bool                   `json:"core_version_compatible"`
	Trust                 *SourceTrustStatus     `json:"trust,omitempty"`
	Verified              bool                   `json:"verified"`
	Warnings              []string               `json:"warnings,omitempty"`
	Issues                []string               `json:"issues,omitempty"`
}

// VerifyInstalled verifies one installed plugin with the default verification options.
func VerifyInstalled(name, version, coreVersion string) (VerifyResult, error) {
	return VerifyInstalledWithOptions(name, version, InstallOptions{CoreVersion: coreVersion})
}

// VerifyInstalledWithOptions verifies one installed plugin with explicit trust and compatibility options.
func VerifyInstalledWithOptions(name, version string, options InstallOptions) (VerifyResult, error) {
	info, err := InfoInstalled(name, version)
	if err != nil {
		return VerifyResult{}, err
	}

	result := VerifyResult{
		Name:               info.Manifest.Name,
		Version:            info.Manifest.Version,
		Kind:               info.Manifest.Kind,
		Capabilities:       cloneCapabilityList(info.Manifest.CapabilityList()),
		Path:               info.Path,
		ProtocolVersion:    info.Manifest.ProtocolVersion,
		ProtocolCompatible: info.Manifest.ProtocolVersion == ProtocolVersion,
		MinCoreVersion:     strings.TrimSpace(info.Manifest.MinCoreVersion),
		CoreVersion:        strings.TrimSpace(options.CoreVersion),
	}
	if info.Install != nil {
		result.Source = info.Install.Source
		result.SourceType = classifyInstallSource(info.Install.Source)
		result.ExpectedChecksumSHA = info.Install.ChecksumSHA

		reference, refErr := parseInstallSourceReference(info.Install.Source)
		if refErr != nil {
			result.Issues = append(result.Issues, refErr.Error())
		} else {
			trust := evaluateInstallSourceTrust(reference, options)
			result.Trust = &trust
			if !trust.Allowed {
				result.Issues = append(result.Issues, trust.Issues...)
			}
		}

		if strings.TrimSpace(info.Install.ArtifactURL) != "" {
			if err := validateRemoteHostPolicy(info.Install.ArtifactURL, options); err != nil {
				result.Issues = append(result.Issues, err.Error())
			}
		}
	}

	actualChecksum, err := checksumTree(info.Path)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("failed to compute plugin checksum: %w", err)
	}
	result.ActualChecksumSHA = actualChecksum

	if result.ExpectedChecksumSHA != "" && result.ExpectedChecksumSHA == actualChecksum {
		result.ChecksumMatch = true
	} else {
		result.Issues = append(result.Issues, "installed plugin checksum does not match recorded metadata")
	}

	if !result.ProtocolCompatible {
		result.Issues = append(result.Issues, "plugin protocol version is incompatible with current core")
	}

	result.CoreVersionCompatible = true
	if compatible, checked := checkCoreVersionCompatibility(result.CoreVersion, result.MinCoreVersion); checked {
		result.CoreVersionChecked = true
		result.CoreVersionCompatible = compatible
		if !compatible {
			result.Issues = append(result.Issues, "plugin min_core_version is newer than current core version")
		}
	}

	capabilityInspection := inspectRuntimeCapabilities(context.Background(), info.Manifest)
	result.RuntimeCapabilities = capabilityInspection.RuntimeCapabilities
	result.Warnings = append(result.Warnings, capabilityInspection.Warnings...)

	result.Verified = len(result.Issues) == 0
	return result, nil
}

// VerifyAllInstalled verifies all installed plugins with the default verification options.
func VerifyAllInstalled(coreVersion string) ([]VerifyResult, error) {
	return VerifyAllInstalledWithOptions(InstallOptions{CoreVersion: coreVersion})
}

// VerifyAllInstalledWithOptions verifies all installed plugins with explicit trust and compatibility options.
func VerifyAllInstalledWithOptions(options InstallOptions) ([]VerifyResult, error) {
	items, err := ListInstalled()
	if err != nil {
		return nil, err
	}

	results := make([]VerifyResult, 0, len(items))
	for _, item := range items {
		result, err := VerifyInstalledWithOptions(item.Name, item.Version, options)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func checkCoreVersionCompatibility(current, min string) (bool, bool) {
	current = strings.TrimSpace(strings.TrimPrefix(current, "v"))
	min = strings.TrimSpace(strings.TrimPrefix(min, "v"))
	if current == "" || min == "" {
		return false, false
	}

	currentParts, ok := parseVersionParts(current)
	if !ok {
		return false, false
	}
	minParts, ok := parseVersionParts(min)
	if !ok {
		return false, false
	}

	maxLen := len(currentParts)
	if len(minParts) > maxLen {
		maxLen = len(minParts)
	}
	for len(currentParts) < maxLen {
		currentParts = append(currentParts, 0)
	}
	for len(minParts) < maxLen {
		minParts = append(minParts, 0)
	}

	for index := 0; index < maxLen; index++ {
		if currentParts[index] > minParts[index] {
			return true, true
		}
		if currentParts[index] < minParts[index] {
			return false, true
		}
	}
	return true, true
}

func parseVersionParts(raw string) ([]int, bool) {
	if raw == "" {
		return nil, false
	}

	parts := strings.Split(raw, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}

		digits := strings.Builder{}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				break
			}
			digits.WriteRune(ch)
		}
		if digits.Len() == 0 {
			return nil, false
		}

		value, err := strconv.Atoi(digits.String())
		if err != nil {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}
