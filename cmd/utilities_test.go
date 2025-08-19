package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/containifyci/feller/pkg/providers"
)

var PATH = ensureStandardPathsInPATH(os.Getenv("PATH"))

// ensureStandardPathsInPATH adds standard paths to PATH if they're missing
func ensureStandardPathsInPATH(currentPath string) string {
	standardPaths := []string{"/bin", "/usr/bin", "/usr/local/bin"}
	pathEntries := strings.Split(currentPath, ":")

	// Create map of existing paths for quick lookup
	existingPaths := make(map[string]bool)
	for _, path := range pathEntries {
		existingPaths[path] = true
	}

	// Add missing standard paths
	for _, stdPath := range standardPaths {
		if !existingPaths[stdPath] {
			currentPath = stdPath + ":" + currentPath
		}
	}

	return currentPath
}

func TestIsGitHubActions(t *testing.T) {
	// Save original environment
	originalVal := os.Getenv("GITHUB_ACTIONS")
	t.Cleanup(func() {
		if originalVal == "" {
			os.Unsetenv("GITHUB_ACTIONS")
		} else {
			t.Setenv("GITHUB_ACTIONS", originalVal)
		}
	})

	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "GitHub Actions environment",
			envValue: "true",
			expected: true,
		},
		{
			name:     "not GitHub Actions environment",
			envValue: "false",
			expected: false,
		},
		{
			name:     "empty environment variable",
			envValue: "",
			expected: false,
		},
		{
			name:     "different value",
			envValue: "yes",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as sub-test uses t.Setenv()
			if tt.envValue == "" {
				os.Unsetenv("GITHUB_ACTIONS")
			} else {
				t.Setenv("GITHUB_ACTIONS", tt.envValue)
			}

			result := isGitHubActions()
			if result != tt.expected {
				t.Errorf("isGitHubActions() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetSecretKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		secrets  providers.SecretMap
		expected []string
	}{
		{
			name:     "empty secrets map",
			secrets:  providers.SecretMap{},
			expected: []string{},
		},
		{
			name: "single secret",
			secrets: providers.SecretMap{
				"key1": "value1",
			},
			expected: []string{"key1"},
		},
		{
			name: "multiple secrets",
			secrets: providers.SecretMap{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expected: []string{"key1", "key2", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getSecretKeys(tt.secrets)

			if len(result) != len(tt.expected) {
				t.Errorf("getSecretKeys() returned %d keys, want %d", len(result), len(tt.expected))
			}

			// Check that all expected keys are present (order may vary due to map iteration)
			for _, expectedKey := range tt.expected {
				found := false
				for _, gotKey := range result {
					if gotKey == expectedKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getSecretKeys() missing expected key: %s", expectedKey)
				}
			}

			// Check that no unexpected keys are present
			for _, gotKey := range result {
				found := false
				for _, expectedKey := range tt.expected {
					if gotKey == expectedKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getSecretKeys() unexpected key: %s", gotKey)
				}
			}
		})
	}
}

func TestMaskSecretFromCmd(t *testing.T) {
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

func TestHandleMissingVariables(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		missingVars []providers.MissingVariable
		errContains []string
		wantErr     bool
	}{
		{
			name:        "no missing variables",
			missingVars: []providers.MissingVariable{},
			wantErr:     false,
		},
		{
			name: "single missing variable",
			missingVars: []providers.MissingVariable{
				{
					VariableName: "MISSING_VAR",
					MappedTo:     "mapped_var",
					Provider:     "test-provider",
				},
			},
			wantErr: true,
			errContains: []string{
				"Missing 1 required environment variable",
				"MISSING_VAR",
				"mapped_var",
				"test-provider",
				"feller run -- your-command",
			},
		},
		{
			name: "multiple missing variables from same provider",
			missingVars: []providers.MissingVariable{
				{
					VariableName: "VAR1",
					MappedTo:     "mapped_var1",
					Provider:     "test-provider",
				},
				{
					VariableName: "VAR2",
					MappedTo:     "mapped_var2",
					Provider:     "test-provider",
				},
			},
			wantErr: true,
			errContains: []string{
				"Missing 2 required environment variable",
				"VAR1",
				"VAR2",
				"test-provider",
				"--silent flag",
			},
		},
		{
			name: "multiple missing variables from different providers",
			missingVars: []providers.MissingVariable{
				{
					VariableName: "VAR1",
					MappedTo:     "mapped_var1",
					Provider:     "provider1",
				},
				{
					VariableName: "VAR2",
					MappedTo:     "mapped_var2",
					Provider:     "provider2",
				},
			},
			wantErr: true,
			errContains: []string{
				"Missing 2 required environment variable",
				"VAR1",
				"VAR2",
				"provider1",
				"provider2",
				"secrets.VAR1",
				"secrets.VAR2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := handleMissingVariables(tt.missingVars)

			if tt.wantErr {
				if err == nil {
					t.Errorf("handleMissingVariables() expected error but got none")
					return
				}

				errMsg := err.Error()
				for _, contains := range tt.errContains {
					if !strings.Contains(errMsg, contains) {
						t.Errorf("handleMissingVariables() error should contain %q, got: %v", contains, err)
					}
				}
			} else if err != nil {
				t.Errorf("handleMissingVariables() unexpected error = %v", err)
			}
		})
	}
}

func TestFindTellerBinary(t *testing.T) { //nolint:paralleltest // sub-tests use t.Setenv()
	tests := []struct {
		setupFunc   func(t *testing.T)
		cleanupFunc func(t *testing.T)
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "no teller binary found",
			setupFunc: func(t *testing.T) {
				t.Helper()
				// Save original PATH
				originalPath := os.Getenv("PATH")
				t.Cleanup(func() {
					t.Setenv("PATH", originalPath)
				})
				// Set empty PATH to ensure no binaries are found
				t.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "teller binary not found in PATH",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // setupFunc may use t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as setupFunc may use t.Setenv()
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc(t)
			}

			path, err := findTellerBinary()

			if tt.wantErr {
				if err == nil {
					t.Errorf("findTellerBinary() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("findTellerBinary() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("findTellerBinary() unexpected error = %v", err)
				return
			}

			if path == "" {
				t.Errorf("findTellerBinary() returned empty path")
			}
		})
	}
}

func TestExecuteDirectCommand(t *testing.T) {
	t.Logf("Testing direct command execution with PATH:`%s`", PATH)
	t.Parallel()
	tests := []struct {
		name        string
		errContains string
		args        []string
		env         []string
		wantErr     bool
	}{
		{
			name:        "no command specified",
			args:        []string{},
			env:         []string{},
			wantErr:     true,
			errContains: "no command specified",
		},
		{
			name:    "valid command with echo",
			args:    []string{"/bin/echo", "test"},
			env:     []string{"PATH=" + PATH},
			wantErr: false,
		},
		{
			name:        "invalid command",
			args:        []string{"nonexistent-command-12345"},
			env:         []string{"PATH=" + PATH},
			wantErr:     true,
			errContains: "direct command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := executeDirectCommand(tt.args, tt.env)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeDirectCommand() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("executeDirectCommand() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("executeDirectCommand() unexpected error = %v", err)
			}
		})
	}
}

func TestExecuteShellCommand(t *testing.T) {
	// Save original SHELL environment variable
	originalShell := os.Getenv("SHELL")
	defer func() {
		if originalShell == "" {
			os.Unsetenv("SHELL")
		} else {
			t.Setenv("SHELL", originalShell)
		}
	}()

	tests := []struct {
		name        string
		shell       string
		errContains string
		args        []string
		env         []string
		wantErr     bool
	}{
		{
			name:        "no command specified",
			args:        []string{},
			env:         []string{},
			wantErr:     true,
			errContains: "no command specified",
		},
		{
			name:    "valid shell command with default shell",
			args:    []string{"/bin/echo", "test"},
			env:     []string{"PATH=" + PATH},
			shell:   "", // Will use default /bin/sh
			wantErr: false,
		},
		{
			name:    "valid shell command with custom shell",
			args:    []string{"/bin/echo", "test"},
			env:     []string{"PATH=" + PATH},
			shell:   "/bin/sh",
			wantErr: false,
		},
		{
			name:        "invalid shell command",
			args:        []string{"nonexistent-command-12345"},
			env:         []string{"PATH=" + PATH},
			shell:       "/bin/sh",
			wantErr:     true,
			errContains: "shell command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set SHELL environment variable
			if tt.shell == "" {
				os.Unsetenv("SHELL")
			} else {
				t.Setenv("SHELL", tt.shell)
			}

			err := executeShellCommand(tt.args, tt.env)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executeShellCommand() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("executeShellCommand() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("executeShellCommand() unexpected error = %v", err)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "execute root command",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Since Execute() calls rootCmd.Execute(), and we don't want to actually run
			// the CLI during tests, we can test the error handling path instead
			err := Execute()

			// Execute should return nil in the test environment since no subcommand is called
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Execute() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestFallbackToTeller(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	originalVerbose := verbose
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() {
		cfgFile = originalCfgFile
		verbose = originalVerbose
		t.Setenv("PATH", originalPath)
	})

	// Clear PATH to ensure teller binary won't be found
	t.Setenv("PATH", "")

	tests := []struct {
		name        string
		cfgFile     string
		errContains string
		args        []string
		verbose     bool
		wantErr     bool
	}{
		{
			name:        "teller binary not found",
			args:        []string{"run", "/bin/echo", "test"},
			cfgFile:     "",
			verbose:     false,
			wantErr:     true,
			errContains: "failed to find teller binary",
		},
		{
			name:        "with config file flag",
			args:        []string{"export", "json"},
			cfgFile:     "/path/to/config.yml",
			verbose:     false,
			wantErr:     true,
			errContains: "failed to find teller binary", // Will still fail because no teller binary
		},
		{
			name:        "with verbose flag",
			args:        []string{"export", "json"},
			cfgFile:     "",
			verbose:     true,
			wantErr:     true,
			errContains: "failed to find teller binary", // Will still fail because no teller binary
		},
		{
			name:        "with both flags",
			args:        []string{"run", "--", "/bin/echo", "test"},
			cfgFile:     "/path/to/config.yml",
			verbose:     true,
			wantErr:     true,
			errContains: "failed to find teller binary", // Will still fail because no teller binary
		},
	}

	for _, tt := range tests { //nolint:paralleltest // sub-tests modify global variables
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as sub-tests modify global variables
			cfgFile = tt.cfgFile
			verbose = tt.verbose

			err := fallbackToTeller(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("fallbackToTeller() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("fallbackToTeller() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("fallbackToTeller() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				t.Setenv(parts[0], parts[1])
			}
		}
	})

	// Save original values
	originalCfgFile := cfgFile
	originalSilent := silent
	originalResetEnv := resetEnv
	originalShell := shell
	t.Cleanup(func() {
		cfgFile = originalCfgFile
		silent = originalSilent
		resetEnv = originalResetEnv
		shell = originalShell
	})

	tests := []struct {
		setupEnv      func(t *testing.T)
		setupConfig   func(t *testing.T) string
		cleanupConfig func(string)
		name          string
		errContains   string
		args          []string
		silent        bool
		resetEnv      bool
		shell         bool
		wantErr       bool
	}{
		{
			name: "GitHub Actions with valid command",
			args: []string{"/bin/echo", "test"},
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("GITHUB_ACTIONS", "true")
				t.Setenv("TEST_VAR", "test_value")
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpFile, err := os.CreateTemp(t.TempDir(), "teller-*.yml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}

				content := `providers:
  test-gsm:
    kind: google_secretmanager
    maps:
      - id: test
        path: projects/test/secrets/test
        keys:
          TEST_VAR: MAPPED_VAR
`
				if _, err := tmpFile.WriteString(content); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()
				return tmpFile.Name()
			},
			cleanupConfig: func(path string) {
				os.Remove(path)
			},
			silent:   false,
			resetEnv: false,
			shell:    false,
			wantErr:  false,
		},
		{
			name: "GitHub Actions with missing config file",
			args: []string{"/bin/echo", "test"},
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("GITHUB_ACTIONS", "true")
			},
			setupConfig: func(_ *testing.T) string {
				return "/nonexistent/config.yml"
			},
			cleanupConfig: func(_ string) {},
			silent:        false,
			resetEnv:      false,
			shell:         false,
			wantErr:       true,
			errContains:   "failed to load config",
		},
		{
			name: "GitHub Actions with missing variables not silent",
			args: []string{"/bin/echo", "test"},
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("GITHUB_ACTIONS", "true")
				t.Setenv("EXISTING_VAR", "existing_value")
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpFile, err := os.CreateTemp(t.TempDir(), "teller-*.yml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}

				content := `providers:
  test-gsm:
    kind: google_secretmanager
    maps:
      - id: test
        path: projects/test/secrets/test
        keys:
          EXISTING_VAR: MAPPED_EXISTING
          MISSING_VAR: MAPPED_MISSING
`
				if _, err := tmpFile.WriteString(content); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()
				return tmpFile.Name()
			},
			cleanupConfig: func(path string) {
				os.Remove(path)
			},
			silent:      false,
			resetEnv:    false,
			shell:       false,
			wantErr:     true,
			errContains: "Missing 1 required environment variable",
		},
		{
			name: "GitHub Actions with missing variables in silent mode",
			args: []string{"/bin/echo", "test"},
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("GITHUB_ACTIONS", "true")
				t.Setenv("EXISTING_VAR", "existing_value")
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpFile, err := os.CreateTemp(t.TempDir(), "teller-*.yml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}

				content := `providers:
  test-gsm:
    kind: google_secretmanager
    maps:
      - id: test
        path: projects/test/secrets/test
        keys:
          EXISTING_VAR: MAPPED_EXISTING
          MISSING_VAR: MAPPED_MISSING
`
				if _, err := tmpFile.WriteString(content); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					t.Fatalf("Failed to write temp file: %v", err)
				}
				tmpFile.Close()
				return tmpFile.Name()
			},
			cleanupConfig: func(path string) {
				os.Remove(path)
			},
			silent:   true,
			resetEnv: false,
			shell:    false,
			wantErr:  false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // setupEnv uses t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as setupEnv uses t.Setenv()
			// Setup environment
			if tt.setupEnv != nil {
				tt.setupEnv(t)
			}

			// Setup config
			if tt.setupConfig != nil {
				configPath := tt.setupConfig(t)
				cfgFile = configPath
				if tt.cleanupConfig != nil {
					defer tt.cleanupConfig(configPath)
				}
			}

			// Set flags
			silent = tt.silent
			resetEnv = tt.resetEnv
			shell = tt.shell

			// Run the function
			err := runCommand(nil, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("runCommand() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("runCommand() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("runCommand() unexpected error = %v", err)
				}
			}
		})
	}
}
