package cmd

import (
	"os"
	"strings"
	"testing"
)

//nolint:paralleltest // Cannot run in parallel due to global rootCmd manipulation
func TestExecuteError(t *testing.T) {
	// Save original command and restore after test
	originalCmd := rootCmd
	t.Cleanup(func() {
		rootCmd = originalCmd
	})

	tests := []struct {
		setupCmd    func()
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "execute success",
			setupCmd: func() {
				// Use the original command which should work
				rootCmd = originalCmd
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // Cannot run in parallel due to global rootCmd manipulation
			if tt.setupCmd != nil {
				tt.setupCmd()
			}

			err := Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Execute() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestExecTeller(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tellerPath  string
		errContains string
		args        []string
		wantErr     bool
	}{
		{
			name:        "invalid teller path",
			tellerPath:  "/nonexistent/teller",
			args:        []string{"export", "json"},
			wantErr:     true,
			errContains: "teller execution failed",
		},
		{
			name:        "empty teller path",
			tellerPath:  "",
			args:        []string{"export", "json"},
			wantErr:     true,
			errContains: "teller execution failed",
		},
		{
			name:       "valid command with echo",
			tellerPath: "/bin/echo",
			args:       []string{"test"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := execTeller(tt.tellerPath, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("execTeller() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("execTeller() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("execTeller() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestFallbackToTellerEdgeCases(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	originalVerbose := verbose
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() {
		cfgFile = originalCfgFile
		verbose = originalVerbose
		t.Setenv("PATH", originalPath)
	})

	tests := []struct {
		setupPath   func()
		name        string
		cfgFile     string
		errContains string
		args        []string
		verbose     bool
		wantErr     bool
	}{
		{
			name:    "empty args",
			args:    []string{},
			cfgFile: "",
			verbose: false,
			setupPath: func() {
				t.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "failed to find teller binary",
		},
		{
			name:    "complex args with flags",
			args:    []string{"run", "--reset", "--shell", "--", "/bin/echo", "test"},
			cfgFile: "/complex/path/config.yml",
			verbose: true,
			setupPath: func() {
				t.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "failed to find teller binary",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // uses t.Setenv() and modifies global variables
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as sub-tests use t.Setenv() and modify global variables
			cfgFile = tt.cfgFile
			verbose = tt.verbose
			if tt.setupPath != nil {
				tt.setupPath()
			}

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

func TestFindTellerBinaryEdgeCases(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() {
		t.Setenv("PATH", originalPath)
	})

	tests := []struct {
		setupPath   func()
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "empty PATH",
			setupPath: func() {
				t.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "teller binary not found in PATH",
		},
		{
			name: "PATH with nonexistent directories",
			setupPath: func() {
				t.Setenv("PATH", "/nonexistent/dir1:/nonexistent/dir2")
			},
			wantErr:     true,
			errContains: "teller binary not found in PATH",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // uses t.Setenv()
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here as sub-tests use t.Setenv()
			if tt.setupPath != nil {
				tt.setupPath()
			}

			path, err := findTellerBinary()

			if tt.wantErr {
				if err == nil {
					t.Errorf("findTellerBinary() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
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
