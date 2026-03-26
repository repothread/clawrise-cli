package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of the main Clawrise config file.
type Config struct {
	Defaults Defaults           `yaml:"defaults"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Defaults stores the default platform and default execution profile.
type Defaults struct {
	Platform string `yaml:"platform"`
	Subject  string `yaml:"subject,omitempty"`
	Profile  string `yaml:"profile"`
}

// Profile describes one executable identity instance.
type Profile struct {
	Platform string `yaml:"platform"`
	Subject  string `yaml:"subject"`
	Grant    Grant  `yaml:"grant"`
}

// Grant describes the credential acquisition method and its related fields.
type Grant struct {
	Type         string `yaml:"type"`
	AppID        string `yaml:"app_id,omitempty"`
	AppSecret    string `yaml:"app_secret,omitempty"`
	Token        string `yaml:"token,omitempty"`
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	NotionVer    string `yaml:"notion_version,omitempty"`
}

// NamedProfile is a profile value paired with its config key.
type NamedProfile struct {
	Name    string
	Profile Profile
}

// New returns an empty config.
func New() *Config {
	return &Config{
		Profiles: map[string]Profile{},
	}
}

// Ensure initializes nil maps so later writes remain safe.
func (c *Config) Ensure() {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
}

// CandidateProfiles returns candidate profiles for a platform in a stable order.
func (c *Config) CandidateProfiles(platform string) []NamedProfile {
	return c.CandidateProfilesBySubject(platform, "")
}

// CandidateProfilesBySubject returns candidate profiles for a platform and,
// when provided, filters them by subject.
func (c *Config) CandidateProfilesBySubject(platform, subject string) []NamedProfile {
	c.Ensure()

	names := make([]string, 0, len(c.Profiles))
	for name, profile := range c.Profiles {
		if profile.Platform == platform && (subject == "" || profile.Subject == subject) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	result := make([]NamedProfile, 0, len(names))
	for _, name := range names {
		result = append(result, NamedProfile{
			Name:    name,
			Profile: c.Profiles[name],
		})
	}
	return result
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CLAWRISE_CONFIG")); custom != "" {
		return custom, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".clawrise", "config.yaml"), nil
}

// Marshal encodes the config as YAML.
func (c *Config) Marshal() ([]byte, error) {
	c.Ensure()
	return yaml.Marshal(c)
}
