package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Store is responsible for loading and saving the main config file.
type Store struct {
	path string
}

// NewStore creates a config store instance.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// ResolveStore creates a config store using the default path.
func ResolveStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

// Path returns the config file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads the config. If the file does not exist, it returns an empty config.
func (s *Store) Load() (*Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := New()
	if len(data) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Ensure()
	return cfg, nil
}

// Save persists the config to disk.
func (s *Store) Save(cfg *Config) error {
	cfg.Ensure()

	data, err := cfg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
