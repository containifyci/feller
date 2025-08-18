package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestValidateOverwriteFlags(t *testing.T) {
	// Save original flag values
	originalForce := force
	originalSkipExisting := skipExisting
	originalConfirmOverwrite := confirmOverwrite
	defer func() {
		force = originalForce
		skipExisting = originalSkipExisting
		confirmOverwrite = originalConfirmOverwrite
	}()

	tests := []struct {
		name             string
		force            bool
		skipExisting     bool
		confirmOverwrite bool
		wantErr          bool
		errContains      string
	}{
		{
			name:             "no flags set",
			force:            false,
			skipExisting:     false,
			confirmOverwrite: false,
			wantErr:          false,
		},
		{
			name:             "only force set",
			force:            true,
			skipExisting:     false,
			confirmOverwrite: false,
			wantErr:          false,
		},
		{
			name:             "only skip existing set",
			force:            false,
			skipExisting:     true,
			confirmOverwrite: false,
			wantErr:          false,
		},
		{
			name:             "only confirm overwrite set",
			force:            false,
			skipExisting:     false,
			confirmOverwrite: true,
			wantErr:          false,
		},
		{
			name:             "force and skip existing both set",
			force:            true,
			skipExisting:     true,
			confirmOverwrite: false,
			wantErr:          true,
			errContains:      "only one of --force, --skip-existing, or --confirm-overwrite can be specified",
		},
		{
			name:             "force and confirm overwrite both set",
			force:            true,
			skipExisting:     false,
			confirmOverwrite: true,
			wantErr:          true,
			errContains:      "only one of --force, --skip-existing, or --confirm-overwrite can be specified",
		},
		{
			name:             "skip existing and confirm overwrite both set",
			force:            false,
			skipExisting:     true,
			confirmOverwrite: true,
			wantErr:          true,
			errContains:      "only one of --force, --skip-existing, or --confirm-overwrite can be specified",
		},
		{
			name:             "all three flags set",
			force:            true,
			skipExisting:     true,
			confirmOverwrite: true,
			wantErr:          true,
			errContains:      "only one of --force, --skip-existing, or --confirm-overwrite can be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			force = tt.force
			skipExisting = tt.skipExisting
			confirmOverwrite = tt.confirmOverwrite

			err := validateOverwriteFlags()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateOverwriteFlags() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateOverwriteFlags() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateOverwriteFlags() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestPrintOperationSummary(t *testing.T) {
	// Save original dryRun value
	originalDryRun := dryRun
	defer func() {
		dryRun = originalDryRun
	}()

	tests := []struct {
		name          string
		stats         *SecretOperationStats
		dryRun        bool
		expectedLines []string
	}{
		{
			name: "normal mode with all stats",
			stats: &SecretOperationStats{
				Created: 3,
				Updated: 2,
				Skipped: 1,
				Failed:  0,
			},
			dryRun: false,
			expectedLines: []string{
				"Operation summary:",
				"Created: 3 secrets",
				"Updated: 2 secrets",
				"Skipped: 1 secrets",
			},
		},
		{
			name: "dry run mode with all stats",
			stats: &SecretOperationStats{
				Created: 5,
				Updated: 1,
				Skipped: 2,
				Failed:  1,
			},
			dryRun: true,
			expectedLines: []string{
				"Dry-run summary:",
				"Created: 5 secrets",
				"Updated: 1 secrets",
				"Skipped: 2 secrets",
				"Failed:  1 secrets",
			},
		},
		{
			name: "no operations",
			stats: &SecretOperationStats{
				Created: 0,
				Updated: 0,
				Skipped: 0,
				Failed:  0,
			},
			dryRun: false,
			expectedLines: []string{
				"Operation summary:",
			},
		},
		{
			name: "only created secrets",
			stats: &SecretOperationStats{
				Created: 10,
				Updated: 0,
				Skipped: 0,
				Failed:  0,
			},
			dryRun: false,
			expectedLines: []string{
				"Operation summary:",
				"Created: 10 secrets",
			},
		},
		{
			name: "only failed operations",
			stats: &SecretOperationStats{
				Created: 0,
				Updated: 0,
				Skipped: 0,
				Failed:  3,
			},
			dryRun: true,
			expectedLines: []string{
				"Dry-run summary:",
				"Failed:  3 secrets",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dryRun = tt.dryRun

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printOperationSummary(tt.stats)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check that all expected lines are present
			for _, expectedLine := range tt.expectedLines {
				if !strings.Contains(output, expectedLine) {
					t.Errorf("printOperationSummary() output should contain %q, got: %s", expectedLine, output)
				}
			}
		})
	}
}

func TestUpdateStats(t *testing.T) {
	tests := []struct {
		name          string
		initialStats  *SecretOperationStats
		operation     string
		expectedStats *SecretOperationStats
	}{
		{
			name: "increment created",
			initialStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
			operation: "created",
			expectedStats: &SecretOperationStats{
				Created: 2,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
		},
		{
			name: "increment updated",
			initialStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
			operation: "updated",
			expectedStats: &SecretOperationStats{
				Created: 1,
				Updated: 3,
				Skipped: 3,
				Failed:  4,
			},
		},
		{
			name: "increment skipped",
			initialStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
			operation: "skipped",
			expectedStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 4,
				Failed:  4,
			},
		},
		{
			name: "unknown operation (no change)",
			initialStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
			operation: "unknown",
			expectedStats: &SecretOperationStats{
				Created: 1,
				Updated: 2,
				Skipped: 3,
				Failed:  4,
			},
		},
		{
			name: "increment from zero",
			initialStats: &SecretOperationStats{
				Created: 0,
				Updated: 0,
				Skipped: 0,
				Failed:  0,
			},
			operation: "created",
			expectedStats: &SecretOperationStats{
				Created: 1,
				Updated: 0,
				Skipped: 0,
				Failed:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of initial stats to avoid modifying the test data
			stats := &SecretOperationStats{
				Created: tt.initialStats.Created,
				Updated: tt.initialStats.Updated,
				Skipped: tt.initialStats.Skipped,
				Failed:  tt.initialStats.Failed,
			}

			updateStats(stats, tt.operation)

			if stats.Created != tt.expectedStats.Created {
				t.Errorf("updateStats() Created = %d, want %d", stats.Created, tt.expectedStats.Created)
			}
			if stats.Updated != tt.expectedStats.Updated {
				t.Errorf("updateStats() Updated = %d, want %d", stats.Updated, tt.expectedStats.Updated)
			}
			if stats.Skipped != tt.expectedStats.Skipped {
				t.Errorf("updateStats() Skipped = %d, want %d", stats.Skipped, tt.expectedStats.Skipped)
			}
			if stats.Failed != tt.expectedStats.Failed {
				t.Errorf("updateStats() Failed = %d, want %d", stats.Failed, tt.expectedStats.Failed)
			}
		})
	}
}

func TestValidateRequiredTools(t *testing.T) {
	// Save original values
	originalDryRun := dryRun
	originalPath := os.Getenv("PATH")
	defer func() {
		dryRun = originalDryRun
		os.Setenv("PATH", originalPath)
	}()

	tests := []struct {
		name        string
		dryRun      bool
		setupPath   func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "no gh binary found",
			dryRun: false,
			setupPath: func() {
				// Clear PATH to ensure gh binary won't be found
				os.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "GitHub CLI (gh) not found",
		},
		{
			name:   "dry run mode with no gh binary",
			dryRun: true,
			setupPath: func() {
				// Clear PATH to ensure gh binary won't be found
				os.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "GitHub CLI (gh) not found",
		},
		{
			name:   "empty PATH",
			dryRun: false,
			setupPath: func() {
				os.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "GitHub CLI (gh) not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dryRun = tt.dryRun
			if tt.setupPath != nil {
				tt.setupPath()
			}

			err := validateRequiredTools()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRequiredTools() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateRequiredTools() error = %v, expected to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateRequiredTools() unexpected error = %v", err)
				}
			}
		})
	}
}
