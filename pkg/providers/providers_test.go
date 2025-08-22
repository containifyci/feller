package providers

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/containifyci/feller/pkg/config"
)

func TestMaskSecret(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "empty string",
			value:    "",
			expected: "",
		},
		{
			name:     "short string (1 char)",
			value:    "a",
			expected: "*",
		},
		{
			name:     "short string (4 chars)",
			value:    "abcd",
			expected: "****",
		},
		{
			name:     "normal string",
			value:    "secret123",
			expected: "se*****23",
		},
		{
			name:     "long string",
			value:    "very-long-secret-value-here",
			expected: "ve***********************re",
		},
		{
			name:     "five character string",
			value:    "hello",
			expected: "he*lo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := maskSecret(tt.value)
			if result != tt.expected {
				t.Errorf("maskSecret(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestCollectSecrets(t *testing.T) {
	// Set up environment variables for testing
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				t.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set test environment variables
	t.Setenv("TEST_VAR1", "value1")
	t.Setenv("TEST_VAR2", "value2")

	tests := []struct {
		config         *config.TellerConfig
		wantSecrets    SecretMap
		setupEnvFile   func(t *testing.T) string
		cleanupEnvFile func(string)
		name           string
		errContains    string
		wantErr        bool
	}{
		{
			name: "empty config",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{},
			},
			wantSecrets: SecretMap{},
			wantErr:     false,
		},
		{
			name: "GSM provider with environment variables",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-gsm": {
						Kind: "google_secretmanager",
						Maps: []config.PathMap{
							{
								ID:   "test",
								Path: "projects/test/secrets/test",
								Keys: map[string]string{
									"TEST_VAR1": "mapped_var1",
									"TEST_VAR2": "mapped_var2",
								},
							},
						},
					},
				},
			},
			wantSecrets: SecretMap{
				"mapped_var1": "value1",
				"mapped_var2": "value2",
			},
			wantErr: false,
		},
		{
			name: "dotenv provider with valid file",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-dotenv": {
						Kind: "dotenv",
						Maps: []config.PathMap{
							{
								ID:   "env-file",
								Path: "", // Will be set by setupEnvFile
								Keys: map[string]string{
									"FILE_VAR1": "mapped_file_var1",
									"FILE_VAR2": "mapped_file_var2",
								},
							},
						},
					},
				},
			},
			wantSecrets: SecretMap{
				"mapped_file_var1": "file_value1",
				"mapped_file_var2": "file_value2",
			},
			wantErr: false,
			setupEnvFile: func(t *testing.T) string {
				t.Helper()
				tmpFile, err := os.CreateTemp(t.TempDir(), "test-env-*.env")
				if err != nil {
					t.Fatalf("Failed to create temp env file: %v", err)
				}

				content := `FILE_VAR1=file_value1
FILE_VAR2=file_value2
# This is a comment
IGNORED_VAR=ignored_value
`
				if _, err := tmpFile.WriteString(content); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					t.Fatalf("Failed to write temp env file: %v", err)
				}

				tmpFile.Close()
				return tmpFile.Name()
			},
			cleanupEnvFile: func(path string) {
				os.Remove(path)
			},
		},
		{
			name: "dotenv provider with discovery mode",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-dotenv-discovery": {
						Kind: "dotenv",
						Maps: []config.PathMap{
							{
								ID:   "env-file",
								Path: "",  // Will be set by setupEnvFile
								Keys: nil, // Discovery mode
							},
						},
					},
				},
			},
			wantSecrets: SecretMap{
				"FILE_VAR1":   "file_value1",
				"FILE_VAR2":   "file_value2",
				"IGNORED_VAR": "ignored_value",
			},
			wantErr: false,
			setupEnvFile: func(t *testing.T) string {
				t.Helper()
				tmpFile, err := os.CreateTemp(t.TempDir(), "test-env-*.env")
				if err != nil {
					t.Fatalf("Failed to create temp env file: %v", err)
				}

				content := `FILE_VAR1=file_value1
FILE_VAR2=file_value2
# This is a comment
IGNORED_VAR=ignored_value
`
				if _, err := tmpFile.WriteString(content); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					t.Fatalf("Failed to write temp env file: %v", err)
				}

				tmpFile.Close()
				return tmpFile.Name()
			},
			cleanupEnvFile: func(path string) {
				os.Remove(path)
			},
		},
		{
			name: "dotenv provider with missing file",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-dotenv-missing": {
						Kind: "dotenv",
						Maps: []config.PathMap{
							{
								ID:   "missing-file",
								Path: "/nonexistent/file.env",
								Keys: map[string]string{
									"VAR1": "mapped_var1",
								},
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "failed to collect dotenv secrets",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // main function uses t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as main function uses t.Setenv()
			// Setup environment file if needed
			if tt.setupEnvFile != nil {
				envPath := tt.setupEnvFile(t)
				if tt.cleanupEnvFile != nil {
					defer tt.cleanupEnvFile(envPath)
				}
				// Update config with actual file path
				if len(tt.config.Providers) > 0 {
					for name, provider := range tt.config.Providers {
						if provider.Kind == "dotenv" && len(provider.Maps) > 0 {
							provider.Maps[0].Path = envPath
							tt.config.Providers[name] = provider
						}
					}
				}
			}

			secrets, err := CollectSecrets(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CollectSecrets() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CollectSecrets() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("CollectSecrets() unexpected error = %v", err)
				return
			}

			if !reflect.DeepEqual(secrets, tt.wantSecrets) {
				t.Errorf("CollectSecrets() = %v, want %v", secrets, tt.wantSecrets)
			}
		})
	}
}

func TestCollectSecretsWithResult(t *testing.T) {
	// Set up test environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				t.Setenv(parts[0], parts[1])
			}
		}
	}()

	t.Setenv("EXISTING_VAR", "existing_value")

	tests := []struct {
		config               *config.TellerConfig
		name                 string
		wantSecretsCount     int
		wantMissingVarsCount int
		silent               bool
		wantHasMissingVars   bool
		wantErr              bool
	}{
		{
			name: "GSM provider with missing variables",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-gsm": {
						Kind: "google_secretmanager",
						Maps: []config.PathMap{
							{
								ID:   "test",
								Path: "projects/test/secrets/test",
								Keys: map[string]string{
									"EXISTING_VAR":    "mapped_existing",
									"NONEXISTENT_VAR": "mapped_missing",
								},
							},
						},
					},
				},
			},
			silent:               false,
			wantSecretsCount:     1,
			wantMissingVarsCount: 1,
			wantHasMissingVars:   true,
			wantErr:              false,
		},
		{
			name: "GSM provider with all variables present",
			config: &config.TellerConfig{
				Providers: map[string]config.Provider{
					"test-gsm": {
						Kind: "google_secretmanager",
						Maps: []config.PathMap{
							{
								ID:   "test",
								Path: "projects/test/secrets/test",
								Keys: map[string]string{
									"EXISTING_VAR": "mapped_existing",
								},
							},
						},
					},
				},
			},
			silent:               false,
			wantSecretsCount:     1,
			wantMissingVarsCount: 0,
			wantHasMissingVars:   false,
			wantErr:              false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // main function uses t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as main function uses t.Setenv()
			result, err := CollectSecretsWithResult(tt.config, tt.silent)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CollectSecretsWithResult() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CollectSecretsWithResult() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("CollectSecretsWithResult() returned nil result")
				return
			}

			if len(result.Secrets) != tt.wantSecretsCount {
				t.Errorf("CollectSecretsWithResult() secrets count = %d, want %d", len(result.Secrets), tt.wantSecretsCount)
			}

			if len(result.MissingVars) != tt.wantMissingVarsCount {
				t.Errorf("CollectSecretsWithResult() missing vars count = %d, want %d", len(result.MissingVars), tt.wantMissingVarsCount)
			}

			if result.HasMissingVars != tt.wantHasMissingVars {
				t.Errorf("CollectSecretsWithResult() HasMissingVars = %v, want %v", result.HasMissingVars, tt.wantHasMissingVars)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expectedVars map[string]string
		name         string
		fileContent  string
		errContains  string
		wantErr      bool
	}{
		{
			name: "valid env file",
			fileContent: `KEY1=value1
KEY2=value2
KEY3="quoted value"
KEY4='single quoted'
# This is a comment
KEY5=value with spaces`,
			expectedVars: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "quoted value",
				"KEY4": "single quoted",
				"KEY5": "value with spaces",
			},
			wantErr: false,
		},
		{
			name:         "empty file",
			fileContent:  "",
			expectedVars: map[string]string{},
			wantErr:      false,
		},
		{
			name: "file with only comments and empty lines",
			fileContent: `# Comment 1

# Comment 2

`,
			expectedVars: map[string]string{},
			wantErr:      false,
		},
		{
			name: "file with malformed lines",
			fileContent: `VALID_KEY=valid_value
malformed line without equals
ANOTHER_KEY=another_value`,
			expectedVars: map[string]string{
				"VALID_KEY":   "valid_value",
				"ANOTHER_KEY": "another_value",
			},
			wantErr: false,
		},
		{
			name:        "nonexistent file",
			fileContent: "", // Not used for this test
			wantErr:     true,
			errContains: "failed to open env file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var filePath string

			if tt.name == "nonexistent file" {
				filePath = "/nonexistent/file.env"
			} else {
				// Create temp file
				tmpFile, err := os.CreateTemp(t.TempDir(), "test-env-*.env")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()

				filePath = tmpFile.Name()
			}

			result, err := loadEnvFile(filePath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadEnvFile() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("loadEnvFile() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadEnvFile() unexpected error = %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expectedVars) {
				t.Errorf("loadEnvFile() = %v, want %v", result, tt.expectedVars)
			}
		})
	}
}

func TestCollectGSMSecretsWithMissing(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				t.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Set test environment
	t.Setenv("PRESENT_VAR", "present_value")

	tests := []struct {
		expectedSecrets      SecretMap
		name                 string
		providerName         string
		provider             config.Provider
		expectedMissingCount int
	}{
		{
			name: "provider with mixed present and missing vars",
			provider: config.Provider{
				Kind: "google_secretmanager",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "projects/test/secrets/test",
						Keys: map[string]string{
							"PRESENT_VAR": "mapped_present",
							"MISSING_VAR": "mapped_missing",
						},
					},
				},
			},
			providerName: "test-provider",
			expectedSecrets: SecretMap{
				"mapped_present": "present_value",
			},
			expectedMissingCount: 1,
		},
		{
			name: "provider with no key mappings (discovery mode)",
			provider: config.Provider{
				Kind: "google_secretmanager",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "projects/test/secrets/test",
						Keys: nil,
					},
				},
			},
			providerName:         "test-provider",
			expectedSecrets:      SecretMap{},
			expectedMissingCount: 0,
		},
		{
			name: "provider with all variables present",
			provider: config.Provider{
				Kind: "google_secretmanager",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "projects/test/secrets/test",
						Keys: map[string]string{
							"PRESENT_VAR": "mapped_present",
						},
					},
				},
			},
			providerName: "test-provider",
			expectedSecrets: SecretMap{
				"mapped_present": "present_value",
			},
			expectedMissingCount: 0,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // main function uses t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as main function uses t.Setenv()
			secrets, missingVars := collectGSMSecretsWithMissing(tt.provider, tt.providerName)

			if !reflect.DeepEqual(secrets, tt.expectedSecrets) {
				t.Errorf("collectGSMSecretsWithMissing() secrets = %v, want %v", secrets, tt.expectedSecrets)
			}

			if len(missingVars) != tt.expectedMissingCount {
				t.Errorf("collectGSMSecretsWithMissing() missing vars count = %d, want %d", len(missingVars), tt.expectedMissingCount)
			}

			// Verify missing variable details
			for _, mv := range missingVars {
				if mv.Provider != tt.providerName {
					t.Errorf("Missing variable has wrong provider: got %s, want %s", mv.Provider, tt.providerName)
				}
			}
		})
	}
}

func TestCollectDotenvSecrets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expectedSecrets SecretMap
		name            string
		fileContent     string
		errContains     string
		provider        config.Provider
		wantErr         bool
	}{
		{
			name: "dotenv with key mapping",
			provider: config.Provider{
				Kind: "dotenv",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "", // Will be set to temp file
						Keys: map[string]string{
							"FILE_VAR1": "mapped_var1",
							"FILE_VAR2": "mapped_var2",
						},
					},
				},
			},
			fileContent: `FILE_VAR1=value1
FILE_VAR2=value2
UNMAPPED_VAR=unmapped_value`,
			expectedSecrets: SecretMap{
				"mapped_var1": "value1",
				"mapped_var2": "value2",
			},
			wantErr: false,
		},
		{
			name: "dotenv with discovery mode",
			provider: config.Provider{
				Kind: "dotenv",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "",  // Will be set to temp file
						Keys: nil, // Discovery mode
					},
				},
			},
			fileContent: `VAR1=value1
VAR2=value2
VAR3=value3`,
			expectedSecrets: SecretMap{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
			wantErr: false,
		},
		{
			name: "dotenv with missing file",
			provider: config.Provider{
				Kind: "dotenv",
				Maps: []config.PathMap{
					{
						ID:   "test",
						Path: "/nonexistent/file.env",
						Keys: map[string]string{
							"VAR1": "mapped_var1",
						},
					},
				},
			},
			fileContent: "", // Not used
			wantErr:     true,
			errContains: "failed to load env file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.name != "dotenv with missing file" {
				// Create temp file
				tmpFile, err := os.CreateTemp(t.TempDir(), "test-env-*.env")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()

				// Update provider with actual file path
				tt.provider.Maps[0].Path = tmpFile.Name()
			}

			secrets, err := collectDotenvSecrets(tt.provider)

			if tt.wantErr {
				if err == nil {
					t.Errorf("collectDotenvSecrets() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("collectDotenvSecrets() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("collectDotenvSecrets() unexpected error = %v", err)
				return
			}

			if !reflect.DeepEqual(secrets, tt.expectedSecrets) {
				t.Errorf("collectDotenvSecrets() = %v, want %v", secrets, tt.expectedSecrets)
			}
		})
	}
}
