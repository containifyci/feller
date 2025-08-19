package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		setupFunc   func(t *testing.T) string
		cleanupFunc func(string)
		name        string
		configPath  string
		configData  string
		errContains string
		wantErr     bool
	}{
		{
			name:        "valid config file",
			configData:  validConfigYAML(),
			wantErr:     false,
			setupFunc:   createTempConfigFile,
			cleanupFunc: cleanupTempFile,
		},
		{
			name:        "file not found",
			configPath:  "/nonexistent/path/teller.yml",
			wantErr:     true,
			errContains: "failed to read config file",
		},
		{
			name:        "invalid YAML",
			configData:  "invalid: yaml: content: [",
			wantErr:     true,
			errContains: "failed to parse config file",
			setupFunc:   createTempConfigFile,
			cleanupFunc: cleanupTempFile,
		},
		{
			name:        "empty config file",
			configData:  "",
			wantErr:     false,
			setupFunc:   createTempConfigFile,
			cleanupFunc: cleanupTempFile,
		},
		{
			name:        "empty config path triggers search",
			configPath:  "",
			configData:  validConfigYAML(),
			wantErr:     false,
			setupFunc:   createTellerYmlInCurrentDir,
			cleanupFunc: cleanupTempFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var configPath string
			if tt.setupFunc != nil {
				configPath = tt.setupFunc(t)
				if tt.cleanupFunc != nil {
					defer tt.cleanupFunc(configPath)
				}
			} else {
				configPath = tt.configPath
			}

			// Override configPath if test specifies empty path (for search test case)
			if tt.name == "empty config path triggers search" {
				configPath = ""
			} else if tt.configPath != "" {
				configPath = tt.configPath
			}

			config, err := LoadConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadConfig() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadConfig() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadConfig() unexpected error = %v", err)
				return
			}

			if config == nil {
				t.Errorf("LoadConfig() returned nil config")
				return
			}

			// Validate structure for valid configs
			if tt.configData != "" && tt.configData != "invalid: yaml: content: [" {
				if len(config.Providers) == 0 && tt.configData != "" {
					// Only expect providers if we have valid YAML content
					if strings.Contains(tt.configData, "providers:") {
						t.Errorf("LoadConfig() expected providers but got none")
					}
				}
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) { //nolint:paralleltest // uses t.Chdir() in main and sub-tests
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		t.Chdir(originalWd)
	}()

	tests := []struct {
		setupFunc   func(t *testing.T) (tempDir string, cleanup func())
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "config file in current directory",
			setupFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, ".teller.yml")
				if err := os.WriteFile(configPath, []byte("providers: {}"), 0o644); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				return tempDir, func() {}
			},
			wantErr: false,
		},
		{
			name: "config file in parent directory",
			setupFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				tempDir := t.TempDir()
				subDir := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(subDir, 0o755); err != nil {
					t.Fatalf("Failed to create subdirectory: %v", err)
				}
				configPath := filepath.Join(tempDir, ".teller.yml")
				if err := os.WriteFile(configPath, []byte("providers: {}"), 0o644); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				return subDir, func() {}
			},
			wantErr: false,
		},
		{
			name: "no config file found",
			setupFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				tempDir := t.TempDir()
				return tempDir, func() {}
			},
			wantErr:     true,
			errContains: "no .teller.yml file found",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // sub-tests use t.Chdir()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as sub-tests use t.Chdir()
			tempDir, cleanup := tt.setupFunc(t)
			defer cleanup()

			// Change to test directory
			t.Chdir(tempDir)

			configPath, err := findConfigFile()

			if tt.wantErr {
				if err == nil {
					t.Errorf("findConfigFile() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("findConfigFile() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("findConfigFile() unexpected error = %v", err)
				return
			}

			if configPath == "" {
				t.Errorf("findConfigFile() returned empty path")
				return
			}

			// Verify the file exists
			if _, err := os.Stat(configPath); err != nil {
				t.Errorf("findConfigFile() returned path that doesn't exist: %s", configPath)
			}
		})
	}
}

func TestGetProvidersByKind(t *testing.T) {
	t.Parallel()
	config := &TellerConfig{
		Providers: map[string]Provider{
			"gsm1": {
				Kind: "google_secretmanager",
				Maps: []PathMap{{ID: "test", Path: "test"}},
			},
			"gsm2": {
				Kind: "google_secretmanager",
				Maps: []PathMap{{ID: "test2", Path: "test2"}},
			},
			"dotenv1": {
				Kind: "dotenv",
				Maps: []PathMap{{ID: "test3", Path: "test3"}},
			},
			"vault1": {
				Kind: "hashicorp_vault",
				Maps: []PathMap{{ID: "test4", Path: "test4"}},
			},
		},
	}

	tests := []struct {
		name         string
		kind         string
		expectedKeys []string
	}{
		{
			name:         "google secretmanager providers",
			kind:         "google_secretmanager",
			expectedKeys: []string{"gsm1", "gsm2"},
		},
		{
			name:         "dotenv providers",
			kind:         "dotenv",
			expectedKeys: []string{"dotenv1"},
		},
		{
			name:         "vault providers",
			kind:         "hashicorp_vault",
			expectedKeys: []string{"vault1"},
		},
		{
			name:         "nonexistent kind",
			kind:         "nonexistent",
			expectedKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			providers := config.GetProvidersByKind(tt.kind)

			if len(providers) != len(tt.expectedKeys) {
				t.Errorf("GetProvidersByKind() returned %d providers, expected %d", len(providers), len(tt.expectedKeys))
			}

			for _, expectedKey := range tt.expectedKeys {
				if _, exists := providers[expectedKey]; !exists {
					t.Errorf("GetProvidersByKind() missing expected provider: %s", expectedKey)
				}
			}

			// Verify all returned providers have the correct kind
			for name, provider := range providers {
				if provider.Kind != tt.kind {
					t.Errorf("GetProvidersByKind() provider %s has kind %s, expected %s", name, provider.Kind, tt.kind)
				}
			}
		})
	}
}

// Helper functions

func validConfigYAML() string {
	return `providers:
  gsm_provider:
    kind: google_secretmanager
    maps:
      - id: test-secret
        path: projects/test/secrets/test-secret/versions/latest
        keys:
          DATABASE_URL: db_url
  dotenv_provider:
    kind: dotenv
    maps:
      - id: local-env
        path: .env
`
}

func createTempConfigFile(t *testing.T) string {
	t.Helper()
	content := validConfigYAML()

	// Check specific test cases
	testName := t.Name()
	if strings.Contains(testName, "invalid_YAML") {
		content = "invalid: yaml: content: ["
	} else if strings.Contains(testName, "empty_config_file") {
		content = ""
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "teller-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func createTellerYmlInCurrentDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	configPath := filepath.Join(wd, ".teller.yml")
	content := validConfigYAML()

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create .teller.yml: %v", err)
	}

	return configPath
}

func cleanupTempFile(path string) {
	os.Remove(path)
}
