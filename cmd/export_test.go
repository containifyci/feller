package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/containifyci/feller/pkg/providers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestExportJSON(t *testing.T) {
	tests := []struct {
		secrets  providers.SecretMap
		validate func(t *testing.T, output string)
		name     string
		wantErr  bool
	}{
		{
			name: "valid secrets map",
			secrets: providers.SecretMap{
				"key1": "value1",
				"key2": "value2",
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				if result["key1"] != "value1" || result["key2"] != "value2" {
					t.Errorf("JSON output has incorrect values: %v", result)
				}
			},
		},
		{
			name:    "empty secrets map",
			secrets: providers.SecretMap{},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				if len(result) != 0 {
					t.Errorf("Expected empty JSON object, got: %v", result)
				}
			},
		},
		{
			name: "secrets with special characters",
			secrets: providers.SecretMap{
				"key/with/slashes": "value with spaces",
				"key-with-dashes":  "value\nwith\nnewlines",
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				expected := map[string]string{
					"key/with/slashes": "value with spaces",
					"key-with-dashes":  "value\nwith\nnewlines",
				}
				for k, v := range expected {
					if result[k] != v {
						t.Errorf("JSON output mismatch for key %s: got %q, want %q", k, result[k], v)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := exportJSON(tt.secrets)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportJSON() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("exportJSON() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				// Remove trailing newline for validation
				output = strings.TrimRight(output, "\n")
				tt.validate(t, output)
			}
		})
	}
}

func TestExportYAML(t *testing.T) {
	tests := []struct {
		secrets  providers.SecretMap
		validate func(t *testing.T, output string)
		name     string
		wantErr  bool
	}{
		{
			name: "valid secrets map",
			secrets: providers.SecretMap{
				"key1": "value1",
				"key2": "value2",
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid YAML output: %v", err)
				}
				if result["key1"] != "value1" || result["key2"] != "value2" {
					t.Errorf("YAML output has incorrect values: %v", result)
				}
			},
		},
		{
			name:    "empty secrets map",
			secrets: providers.SecretMap{},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid YAML output: %v", err)
				}
				if len(result) != 0 {
					t.Errorf("Expected empty YAML object, got: %v", result)
				}
			},
		},
		{
			name: "secrets with special characters",
			secrets: providers.SecretMap{
				"key/with/slashes": "value with spaces",
				"key-with-dashes":  "value\nwith\nnewlines",
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				var result map[string]string
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid YAML output: %v", err)
				}
				expected := map[string]string{
					"key/with/slashes": "value with spaces",
					"key-with-dashes":  "value\nwith\nnewlines",
				}
				for k, v := range expected {
					if result[k] != v {
						t.Errorf("YAML output mismatch for key %s: got %q, want %q", k, result[k], v)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := exportYAML(tt.secrets)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportYAML() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("exportYAML() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestExportEnv(t *testing.T) {
	tests := []struct {
		secrets   providers.SecretMap
		name      string
		wantLines []string
		wantErr   bool
	}{
		{
			name: "valid secrets map",
			secrets: providers.SecretMap{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
			wantLines: []string{
				`KEY1="value1"`,
				`KEY2="value2"`,
			},
		},
		{
			name:      "empty secrets map",
			secrets:   providers.SecretMap{},
			wantErr:   false,
			wantLines: []string{},
		},
		{
			name: "secrets with special characters",
			secrets: providers.SecretMap{
				"KEY_WITH_QUOTES":    `value with "quotes"`,
				"KEY_WITH_NEWLINES":  "value\nwith\nnewlines",
				"KEY_WITH_BACKSLASH": `value\with\backslash`,
			},
			wantErr: false,
			wantLines: []string{
				`KEY_WITH_BACKSLASH="value\\with\\backslash"`,
				`KEY_WITH_NEWLINES="value\nwith\nnewlines"`,
				`KEY_WITH_QUOTES="value with \"quotes\""`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := exportEnv(tt.secrets)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportEnv() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("exportEnv() unexpected error = %v", err)
				return
			}

			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
			if len(tt.wantLines) == 0 {
				if output != "" {
					t.Errorf("exportEnv() expected no output, got: %q", output)
				}
				return
			}

			if len(lines) != len(tt.wantLines) {
				t.Errorf("exportEnv() got %d lines, want %d lines", len(lines), len(tt.wantLines))
			}

			// Check that all expected lines are present (order may vary due to map iteration)
			for _, wantLine := range tt.wantLines {
				found := false
				for _, gotLine := range lines {
					if gotLine == wantLine {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("exportEnv() missing expected line: %s", wantLine)
				}
			}
		})
	}
}

func TestExportCSV(t *testing.T) {
	tests := []struct {
		secrets   providers.SecretMap
		name      string
		wantLines []string
		wantErr   bool
	}{
		{
			name: "valid secrets map",
			secrets: providers.SecretMap{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
			wantLines: []string{
				"key,value",
				`"KEY1","value1"`,
				`"KEY2","value2"`,
			},
		},
		{
			name:    "empty secrets map",
			secrets: providers.SecretMap{},
			wantErr: false,
			wantLines: []string{
				"key,value",
			},
		},
		{
			name: "secrets with quotes",
			secrets: providers.SecretMap{
				"KEY_WITH_QUOTES": `value with "quotes"`,
			},
			wantErr: false,
			wantLines: []string{
				"key,value",
				`"KEY_WITH_QUOTES","value with ""quotes"""`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := exportCSV(tt.secrets)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportCSV() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("exportCSV() unexpected error = %v", err)
				return
			}

			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

			if len(lines) != len(tt.wantLines) {
				t.Errorf("exportCSV() got %d lines, want %d lines", len(lines), len(tt.wantLines))
			}

			// First line should always be the header
			if len(lines) > 0 && lines[0] != "key,value" {
				t.Errorf("exportCSV() first line should be header, got: %s", lines[0])
			}

			// Check remaining lines (data rows may be in different order due to map iteration)
			if len(tt.wantLines) > 1 {
				expectedDataLines := tt.wantLines[1:] // Skip header
				actualDataLines := lines[1:]          // Skip header

				for _, wantLine := range expectedDataLines {
					found := false
					for _, gotLine := range actualDataLines {
						if gotLine == wantLine {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("exportCSV() missing expected line: %s", wantLine)
					}
				}
			}
		})
	}
}

func TestHandleMissingVariablesExport(t *testing.T) {
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
				"feller export json",
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
				"--silent flag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleMissingVariablesExport(tt.missingVars)

			if tt.wantErr {
				if err == nil {
					t.Errorf("handleMissingVariablesExport() expected error but got none")
					return
				}

				errMsg := err.Error()
				for _, contains := range tt.errContains {
					if !strings.Contains(errMsg, contains) {
						t.Errorf("handleMissingVariablesExport() error should contain %q, got: %v", contains, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("handleMissingVariablesExport() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestExportSecrets(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Save original values
	originalCfgFile := cfgFile
	originalSilent := silent
	defer func() {
		cfgFile = originalCfgFile
		silent = originalSilent
	}()

	tests := []struct {
		setupEnv       func(t *testing.T)
		setupConfig    func(t *testing.T) string
		cleanupConfig  func(string)
		validateOutput func(t *testing.T, output string)
		name           string
		errContains    string
		args           []string
		silent         bool
		wantErr        bool
	}{
		{
			name: "GitHub Actions with valid JSON export",
			args: []string{"json"},
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
				os.Setenv("TEST_VAR1", "value1")
				os.Setenv("TEST_VAR2", "value2")
			},
			setupConfig: func(t *testing.T) string {
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
          TEST_VAR1: MAPPED_VAR1
          TEST_VAR2: MAPPED_VAR2
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
			silent:  false,
			wantErr: false,
			validateOutput: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("exportSecrets should produce valid JSON: %v", err)
				}
				if result["MAPPED_VAR1"] != "value1" || result["MAPPED_VAR2"] != "value2" {
					t.Errorf("exportSecrets JSON output has incorrect values: %v", result)
				}
			},
		},
		{
			name: "GitHub Actions with invalid format",
			args: []string{"invalid"},
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
			},
			setupConfig: func(t *testing.T) string {
				tmpFile, err := os.CreateTemp(t.TempDir(), "teller-*.yml")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}

				content := `providers: {}`
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
			wantErr:     true,
			errContains: "unsupported format: invalid",
		},
		{
			name: "GitHub Actions with missing config file",
			args: []string{"json"},
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
			},
			setupConfig: func(t *testing.T) string {
				return "/nonexistent/config.yml"
			},
			cleanupConfig: func(path string) {},
			silent:        false,
			wantErr:       true,
			errContains:   "failed to load config",
		},
		{
			name: "GitHub Actions with missing variables not silent",
			args: []string{"json"},
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
				os.Setenv("EXISTING_VAR", "existing_value")
			},
			setupConfig: func(t *testing.T) string {
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
			wantErr:     true,
			errContains: "Cannot export: Missing 1 required environment variable",
		},
		{
			name: "GitHub Actions with missing variables in silent mode",
			args: []string{"json"},
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
				os.Setenv("EXISTING_VAR", "existing_value")
			},
			setupConfig: func(t *testing.T) string {
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
			silent:  true,
			wantErr: false,
			validateOutput: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("exportSecrets should produce valid JSON: %v", err)
				}
				if result["MAPPED_EXISTING"] != "existing_value" {
					t.Errorf("exportSecrets should export existing variable")
				}
				if _, exists := result["MAPPED_MISSING"]; exists {
					t.Errorf("exportSecrets should not export missing variable in silent mode")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Set silent mode
			silent = tt.silent

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a cobra command for testing
			cmd := &cobra.Command{}

			// Run the function
			err := exportSecrets(cmd, tt.args)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportSecrets() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("exportSecrets() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("exportSecrets() unexpected error = %v", err)
				return
			}

			if tt.validateOutput != nil {
				// Remove trailing newlines for validation
				output = strings.TrimRight(output, "\n")
				tt.validateOutput(t, output)
			}
		})
	}
}
