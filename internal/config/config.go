package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/lukas/focus/internal/domain"
)

const fileName = "config.json"

type Config struct {
	Enabled bool     `json:"enabled"`
	Domains []string `json:"domains"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func DefaultPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find configuration directory: %w", err)
	}
	return filepath.Join(directory, "focus", fileName), nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Config, error) {
	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	var configuration Config
	if err := json.Unmarshal(contents, &configuration); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	domains, err := normalizedDomains(configuration.Domains)
	if err != nil {
		return Config{}, err
	}
	configuration.Domains = domains
	return configuration, nil
}

func (s *Store) Save(configuration Config) error {
	domains, err := normalizedDomains(configuration.Domains)
	if err != nil {
		return err
	}
	configuration.Domains = domains
	contents, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	contents = append(contents, '\n')

	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary configuration: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}

func normalizedDomains(domains []string) ([]string, error) {
	seen := make(map[string]struct{}, len(domains))
	for _, name := range domains {
		normalized, err := domain.Normalize(name)
		if err != nil {
			return nil, fmt.Errorf("invalid configured website %q: %w", name, err)
		}
		seen[normalized] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
