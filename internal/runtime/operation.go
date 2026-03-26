package runtime

import (
	"fmt"
	"strings"
)

// Operation is the normalized operation shape after parsing.
type Operation struct {
	Raw          string
	Normalized   string
	Platform     string
	ResourcePath string
	Action       string
}

// ParseOperation converts user input into the normalized operation shape.
func ParseOperation(raw, defaultPlatform string) (Operation, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Operation{}, fmt.Errorf("operation must not be empty")
	}

	parts := strings.Split(raw, ".")
	switch {
	case len(parts) >= 3 && isKnownPlatform(parts[0]):
		platform := parts[0]
		action := parts[len(parts)-1]
		resourcePath := strings.Join(parts[1:len(parts)-1], ".")
		return Operation{
			Raw:          raw,
			Normalized:   raw,
			Platform:     platform,
			ResourcePath: resourcePath,
			Action:       action,
		}, nil
	case len(parts) >= 2 && strings.TrimSpace(defaultPlatform) != "":
		action := parts[len(parts)-1]
		resourcePath := strings.Join(parts[:len(parts)-1], ".")
		normalized := defaultPlatform + "." + raw
		return Operation{
			Raw:          raw,
			Normalized:   normalized,
			Platform:     defaultPlatform,
			ResourcePath: resourcePath,
			Action:       action,
		}, nil
	case len(parts) >= 3:
		platform := parts[0]
		resourcePath := strings.Join(parts[1:len(parts)-1], ".")
		return Operation{
			Raw:          raw,
			Normalized:   raw,
			Platform:     platform,
			ResourcePath: resourcePath,
			Action:       parts[len(parts)-1],
		}, nil
	default:
		return Operation{}, fmt.Errorf("invalid operation format; expected <platform>.<resource-path>.<action> or <resource-path>.<action> when a default platform is set")
	}
}

func isKnownPlatform(value string) bool {
	switch value {
	case "feishu", "notion", "google":
		return true
	default:
		return false
	}
}
