package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fr12k/feller/pkg/logger"
	"gopkg.in/yaml.v3"
)

// TellerConfig represents the structure of a .teller.yml configuration file
type TellerConfig struct {
	Providers map[string]Provider `yaml:"providers"`
}

// Provider represents a single provider configuration
type Provider struct {
	Kind    string    `yaml:"kind"`
	Maps    []PathMap `yaml:"maps"`
	Options yaml.Node `yaml:"options,omitempty"`
}

// PathMap represents a path mapping within a provider
type PathMap struct {
	Keys map[string]string `yaml:"keys,omitempty"`
	ID   string            `yaml:"id"`
	Path string            `yaml:"path"`
}

// LoadConfig loads and parses a Teller configuration file
func LoadConfig(configPath string) (*TellerConfig, error) {
	logger.Debug("Loading configuration...")

	if configPath == "" {
		logger.Debug("No config path provided, searching upwards from current directory")
		// Find config file upwards from current directory
		var err error
		configPath, err = findConfigFile()
		if err != nil {
			logger.Debug("Config file search failed: %v", err)
			return nil, err
		}
	}

	logger.Debug("Using config file: %s", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Debug("Failed to read config file: %v", err)
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	logger.Debug("Config file size: %d bytes", len(data))

	var config TellerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Debug("Failed to parse YAML: %v", err)
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	logger.Debug("Parsed %d providers from config", len(config.Providers))
	for name, provider := range config.Providers {
		logger.Debug("  Provider '%s': kind=%s, maps=%d", name, provider.Kind, len(provider.Maps))
	}

	return &config, nil
}

// findConfigFile searches for .teller.yml upward from the current directory
func findConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		logger.Debug("Failed to get current working directory: %v", err)
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	logger.Debug("Searching for config file starting from: %s", dir)

	for {
		configPath := filepath.Join(dir, ".teller.yml")
		logger.Debug("Checking for config file at: %s", configPath)

		if _, err := os.Stat(configPath); err == nil {
			logger.Debug("Found config file at: %s", configPath)
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			logger.Debug("Reached root directory without finding config file")
			break
		}
		dir = parent
	}

	return "", errors.New("no .teller.yml file found in current directory or any parent directory")
}

// GetProvidersByKind returns all providers of a specific kind
func (c *TellerConfig) GetProvidersByKind(kind string) map[string]Provider {
	providers := make(map[string]Provider)
	for name, provider := range c.Providers {
		if provider.Kind == kind {
			providers[name] = provider
		}
	}
	return providers
}
