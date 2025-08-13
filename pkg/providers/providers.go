package providers

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fr12k/feller/pkg/config"
	"github.com/fr12k/feller/pkg/logger"
)

// SecretMap represents a collection of key-value pairs
type SecretMap map[string]string

// MissingVariable represents a missing environment variable
type MissingVariable struct {
	VariableName string // The environment variable name that was missing
	MappedTo     string // The key it should have been mapped to
	Provider     string // The provider name that expected this variable
}

// CollectionResult contains the collected secrets and any missing variables
type CollectionResult struct {
	Secrets        SecretMap
	MissingVars    []MissingVariable
	HasMissingVars bool
}

// CollectSecrets collects all secrets from all providers in the configuration
func CollectSecrets(cfg *config.TellerConfig) (SecretMap, error) {
	result, err := CollectSecretsWithResult(cfg, false)
	if err != nil {
		return nil, err
	}
	return result.Secrets, nil
}

// CollectSecretsWithResult collects all secrets and tracks missing variables
func CollectSecretsWithResult(cfg *config.TellerConfig, silent bool) (*CollectionResult, error) {
	logger.Debug("Collecting secrets from all providers (silent: %v)", silent)
	result := &CollectionResult{
		Secrets:     make(SecretMap),
		MissingVars: []MissingVariable{},
	}

	// Process Google Secret Manager providers (read from environment)
	gsmProviders := cfg.GetProvidersByKind("google_secretmanager")
	logger.Debug("Found %d Google Secret Manager providers", len(gsmProviders))

	for name, provider := range gsmProviders {
		logger.Debug("Processing GSM provider '%s'", name)
		providerSecrets, missingVars := collectGSMSecretsWithMissing(provider, name)
		logger.Debug("GSM provider '%s' returned %d secrets, %d missing", name, len(providerSecrets), len(missingVars))

		// Track missing variables
		result.MissingVars = append(result.MissingVars, missingVars...)

		// Merge secrets, later providers override earlier ones
		for k, v := range providerSecrets {
			if _, exists := result.Secrets[k]; exists {
				logger.Debug("GSM provider '%s' overriding key '%s' (previous value from other provider)", name, k)
			}
			result.Secrets[k] = v
			logger.Debug("Added secret key '%s' (value: %s) from GSM provider '%s'", k, maskSecret(v), name)
		}
	}

	// Process dotenv providers (read from files)
	dotenvProviders := cfg.GetProvidersByKind("dotenv")
	logger.Debug("Found %d dotenv providers", len(dotenvProviders))

	for name, provider := range dotenvProviders {
		logger.Debug("Processing dotenv provider '%s'", name)
		providerSecrets, err := collectDotenvSecrets(provider)
		if err != nil {
			logger.Debug("Failed to collect dotenv secrets from provider '%s': %v", name, err)
			return nil, fmt.Errorf("failed to collect dotenv secrets: %w", err)
		}
		logger.Debug("Dotenv provider '%s' returned %d secrets", name, len(providerSecrets))

		// Merge secrets, later providers override earlier ones
		for k, v := range providerSecrets {
			if _, exists := result.Secrets[k]; exists {
				logger.Debug("Dotenv provider '%s' overriding key '%s' (previous value from other provider)", name, k)
			}
			result.Secrets[k] = v
			logger.Debug("Added secret key '%s' (value: %s) from dotenv provider '%s'", k, maskSecret(v), name)
		}
	}

	result.HasMissingVars = len(result.MissingVars) > 0
	logger.Debug("Total secrets collected: %d, missing variables: %d", len(result.Secrets), len(result.MissingVars))

	return result, nil
}

// maskSecret masks a secret value for debug logging
func maskSecret(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// collectGSMSecretsWithMissing collects secrets and tracks missing environment variables
func collectGSMSecretsWithMissing(provider config.Provider, providerName string) (SecretMap, []MissingVariable) {
	logger.Debug("Collecting GSM secrets from %d path maps", len(provider.Maps))
	secrets := make(SecretMap)
	var missingVars []MissingVariable

	for i, pathMap := range provider.Maps {
		logger.Debug("Processing GSM path map %d (id: %s, path: %s)", i+1, pathMap.ID, pathMap.Path)

		if len(pathMap.Keys) == 0 {
			logger.Debug("Discovery mode not supported for GSM provider, skipping map %d", i+1)
			continue
		}

		logger.Debug("GSM map %d has %d key mappings", i+1, len(pathMap.Keys))

		// Specific key mapping mode
		for fromKey, toKey := range pathMap.Keys {
			logger.Debug("Looking for environment variable '%s' to map to '%s'", fromKey, toKey)
			if value := os.Getenv(fromKey); value != "" {
				secrets[toKey] = value
				logger.Debug("Found env var '%s' with value '%s', mapped to key '%s'", fromKey, maskSecret(value), toKey)
			} else {
				logger.Debug("Environment variable '%s' not found or empty", fromKey)
				missingVars = append(missingVars, MissingVariable{
					VariableName: fromKey,
					MappedTo:     toKey,
					Provider:     providerName,
				})
			}
		}
	}

	logger.Debug("GSM provider collected %d secrets total, %d missing", len(secrets), len(missingVars))
	return secrets, missingVars
}

// collectDotenvSecrets collects secrets from dotenv provider
// This reads from .env files on the filesystem
func collectDotenvSecrets(provider config.Provider) (SecretMap, error) {
	logger.Debug("Collecting dotenv secrets from %d path maps", len(provider.Maps))
	secrets := make(SecretMap)

	for i, pathMap := range provider.Maps {
		logger.Debug("Processing dotenv path map %d (id: %s, path: %s)", i+1, pathMap.ID, pathMap.Path)

		envFile, err := loadEnvFile(pathMap.Path)
		if err != nil {
			logger.Debug("Failed to load env file '%s': %v", pathMap.Path, err)
			return nil, fmt.Errorf("failed to load env file %s: %w", pathMap.Path, err)
		}

		logger.Debug("Loaded %d variables from env file '%s'", len(envFile), pathMap.Path)

		if len(pathMap.Keys) == 0 {
			logger.Debug("Discovery mode: using all %d keys from the file", len(envFile))
			// Discovery mode: use all keys from the file
			for k, v := range envFile {
				secrets[k] = v
				logger.Debug("Added key '%s' (value: %s) from env file", k, maskSecret(v))
			}
		} else {
			logger.Debug("Key mapping mode: processing %d key mappings", len(pathMap.Keys))
			// Specific key mapping mode
			for fromKey, toKey := range pathMap.Keys {
				if value, exists := envFile[fromKey]; exists {
					secrets[toKey] = value
					logger.Debug("Mapped key '%s' to '%s' (value: %s) from env file", fromKey, toKey, maskSecret(value))
				} else {
					logger.Debug("Key '%s' not found in env file '%s'", fromKey, pathMap.Path)
				}
			}
		}
	}

	logger.Debug("Dotenv provider collected %d secrets total", len(secrets))
	return secrets, nil
}

// loadEnvFile loads a .env file and returns key-value pairs
func loadEnvFile(filePath string) (map[string]string, error) {
	logger.Debug("Loading env file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Debug("Failed to open env file '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to open env file %s: %w", filePath, err)
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			logger.Debug("Skipping line %d (empty or comment): %s", lineNum, line)
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					originalValue := value
					value = value[1 : len(value)-1]
					logger.Debug("Removed quotes from value: %s -> %s", originalValue, maskSecret(value))
				}
			}

			env[key] = value
			logger.Debug("Parsed line %d: %s=%s", lineNum, key, maskSecret(value))
		} else {
			logger.Debug("Skipping malformed line %d: %s", lineNum, line)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Debug("Error reading env file '%s': %v", filePath, err)
		return nil, fmt.Errorf("error reading env file %s: %w", filePath, err)
	}

	logger.Debug("Successfully loaded %d variables from env file '%s'", len(env), filePath)
	return env, nil
}
