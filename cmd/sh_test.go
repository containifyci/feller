package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/containifyci/feller/pkg/providers"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with single quote",
			input:    "don't",
			expected: "don'\\''t",
		},
		{
			name:     "string with multiple single quotes",
			input:    "can't won't don't",
			expected: "can'\\''t won'\\''t don'\\''t",
		},
		{
			name:     "string starting with single quote",
			input:    "'hello",
			expected: "'\\''hello",
		},
		{
			name:     "string ending with single quote",
			input:    "hello'",
			expected: "hello'\\''",
		},
		{
			name:     "only single quotes",
			input:    "'''",
			expected: "'\\'''\\'''\\''",
		},
		{
			name:     "special characters without quotes",
			input:    "hello world!@#$%^&*()",
			expected: "hello world!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellReplaceAll(t *testing.T) {
	tests := []struct {
		name        string
		s           string
		old         string
		replacement string
		expected    string
	}{
		{
			name:        "simple replacement",
			s:           "hello world",
			old:         "world",
			replacement: "universe",
			expected:    "hello universe",
		},
		{
			name:        "no matches",
			s:           "hello world",
			old:         "foo",
			replacement: "bar",
			expected:    "hello world",
		},
		{
			name:        "multiple matches",
			s:           "foo bar foo baz foo",
			old:         "foo",
			replacement: "qux",
			expected:    "qux bar qux baz qux",
		},
		{
			name:        "empty string",
			s:           "",
			old:         "foo",
			replacement: "bar",
			expected:    "",
		},
		{
			name:        "empty old string",
			s:           "hello",
			old:         "",
			replacement: "x",
			expected:    "hello",
		},
		{
			name:        "replace with empty string",
			s:           "remove this remove",
			old:         "remove ",
			replacement: "",
			expected:    "this remove",
		},
		{
			name:        "overlapping patterns",
			s:           "aaaa",
			old:         "aa",
			replacement: "b",
			expected:    "bb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellReplaceAll(tt.s, tt.old, tt.replacement)
			if result != tt.expected {
				t.Errorf("shellReplaceAll(%q, %q, %q) = %q, want %q",
					tt.s, tt.old, tt.replacement, result, tt.expected)
			}
		})
	}
}

func TestShellIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "found at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "hello world",
			substr:   "o w",
			expected: 4,
		},
		{
			name:     "found at end",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "hello world",
			substr:   "foo",
			expected: -1,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: 0,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "foo",
			expected: -1,
		},
		{
			name:     "both empty",
			s:        "",
			substr:   "",
			expected: 0,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: -1,
		},
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellIndexOf(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("shellIndexOf(%q, %q) = %d, want %d",
					tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestHandleMissingVariablesShell(t *testing.T) {
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
				"Cannot generate shell exports",
				"Missing 1 required environment variable",
				"MISSING_VAR",
				"mapped_var",
				"test-provider",
				"eval \"$(feller sh)\"",
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
		{
			name: "multiple missing variables from same provider",
			missingVars: []providers.MissingVariable{
				{
					VariableName: "VAR1",
					MappedTo:     "mapped_var1",
					Provider:     "same-provider",
				},
				{
					VariableName: "VAR2",
					MappedTo:     "mapped_var2",
					Provider:     "same-provider",
				},
			},
			wantErr: true,
			errContains: []string{
				"Missing 2 required environment variable",
				"VAR1",
				"VAR2",
				"same-provider",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleMissingVariablesShell(tt.missingVars)

			if tt.wantErr {
				if err == nil {
					t.Errorf("handleMissingVariablesShell() expected error but got none")
					return
				}

				errMsg := err.Error()
				for _, contains := range tt.errContains {
					if !strings.Contains(errMsg, contains) {
						t.Errorf("handleMissingVariablesShell() error should contain %q, got: %v", contains, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("handleMissingVariablesShell() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestExportShell(t *testing.T) {
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
		silent         bool
		wantErr        bool
	}{
		{
			name: "GitHub Actions with valid config and secrets",
			setupEnv: func(t *testing.T) {
				os.Setenv("GITHUB_ACTIONS", "true")
				os.Setenv("TEST_VAR1", "value1")
				os.Setenv("TEST_VAR2", "value with spaces")
				os.Setenv("TEST_VAR3", "value'with'quotes")
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
          TEST_VAR3: MAPPED_VAR3
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
				expectedLines := []string{
					"export MAPPED_VAR1='value1'",
					"export MAPPED_VAR2='value with spaces'",
					"export MAPPED_VAR3='value'\\''with'\\''quotes'",
				}

				for _, expectedLine := range expectedLines {
					if !strings.Contains(output, expectedLine) {
						t.Errorf("exportShell output should contain %q, got: %s", expectedLine, output)
					}
				}
			},
		},
		{
			name: "GitHub Actions with missing config file",
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
			name: "GitHub Actions with missing variables in silent mode",
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
				if !strings.Contains(output, "export MAPPED_EXISTING='existing_value'") {
					t.Errorf("exportShell should export existing variable, got: %s", output)
				}
				if strings.Contains(output, "MAPPED_MISSING") {
					t.Errorf("exportShell should not export missing variable in silent mode, got: %s", output)
				}
			},
		},
		{
			name: "GitHub Actions with missing variables not silent",
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
			errContains: "Cannot generate shell exports",
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

			// Run the function
			err := exportShell(nil, []string{})

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("exportShell() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("exportShell() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("exportShell() unexpected error = %v", err)
				return
			}

			if tt.validateOutput != nil {
				tt.validateOutput(t, output)
			}
		})
	}
}
