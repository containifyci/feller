package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetDebug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enable debug",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disable debug",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			SetDebug(tt.enabled)
			if got := IsDebugEnabled(); got != tt.expected {
				t.Errorf("SetDebug(%v) = %v, want %v", tt.enabled, got, tt.expected)
			}
		})
	}
}

func TestSetVerbose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enable verbose",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disable verbose",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			SetVerbose(tt.enabled)
			if got := IsVerboseEnabled(); got != tt.expected {
				t.Errorf("SetVerbose(%v) = %v, want %v", tt.enabled, got, tt.expected)
			}
		})
	}
}

func TestIsDebugEnabled(t *testing.T) {
	t.Parallel()
	// Reset state
	SetDebug(false)
	if IsDebugEnabled() {
		t.Errorf("IsDebugEnabled() = true, want false")
	}

	SetDebug(true)
	if !IsDebugEnabled() {
		t.Errorf("IsDebugEnabled() = false, want true")
	}
}

func TestIsVerboseEnabled(t *testing.T) {
	t.Parallel()
	// Reset state
	SetVerbose(false)
	if IsVerboseEnabled() {
		t.Errorf("IsVerboseEnabled() = true, want false")
	}

	SetVerbose(true)
	if !IsVerboseEnabled() {
		t.Errorf("IsVerboseEnabled() = false, want true")
	}
}

//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
func TestDebugLogging(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		expectedText string
		args         []interface{}
		debugEnabled bool
		expectOutput bool
	}{
		{
			name:         "debug enabled with simple message",
			debugEnabled: true,
			format:       "test message",
			args:         nil,
			expectOutput: true,
			expectedText: "[DEBUG] test message",
		},
		{
			name:         "debug enabled with formatted message",
			debugEnabled: true,
			format:       "test %s with %d value",
			args:         []interface{}{"message", 42},
			expectOutput: true,
			expectedText: "[DEBUG] test message with 42 value",
		},
		{
			name:         "debug disabled",
			debugEnabled: false,
			format:       "test message",
			args:         nil,
			expectOutput: false,
			expectedText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Set debug state
			SetDebug(tt.debugEnabled)

			// Call function
			if tt.args != nil {
				Debug(tt.format, tt.args...)
			} else {
				Debug(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expectOutput {
				if !strings.Contains(output, tt.expectedText) {
					t.Errorf("Debug() output = %q, want to contain %q", output, tt.expectedText)
				}
			} else {
				if output != "" {
					t.Errorf("Debug() output = %q, want empty", output)
				}
			}
		})
	}
}

//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
func TestVerboseLogging(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedText   string
		args           []interface{}
		verboseEnabled bool
		expectOutput   bool
	}{
		{
			name:           "verbose enabled with simple message",
			verboseEnabled: true,
			format:         "test message",
			args:           nil,
			expectOutput:   true,
			expectedText:   "[VERBOSE] test message",
		},
		{
			name:           "verbose enabled with formatted message",
			verboseEnabled: true,
			format:         "test %s with %d value",
			args:           []interface{}{"message", 42},
			expectOutput:   true,
			expectedText:   "[VERBOSE] test message with 42 value",
		},
		{
			name:           "verbose disabled",
			verboseEnabled: false,
			format:         "test message",
			args:           nil,
			expectOutput:   false,
			expectedText:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Set verbose state
			SetVerbose(tt.verboseEnabled)

			// Call function
			if tt.args != nil {
				Verbose(tt.format, tt.args...)
			} else {
				Verbose(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expectOutput {
				if !strings.Contains(output, tt.expectedText) {
					t.Errorf("Verbose() output = %q, want to contain %q", output, tt.expectedText)
				}
			} else {
				if output != "" {
					t.Errorf("Verbose() output = %q, want empty", output)
				}
			}
		})
	}
}

//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
func TestInfoLogging(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		expectedText string
		args         []interface{}
	}{
		{
			name:         "info with simple message",
			format:       "test message",
			args:         nil,
			expectedText: "[INFO] test message",
		},
		{
			name:         "info with formatted message",
			format:       "test %s with %d value",
			args:         []interface{}{"message", 42},
			expectedText: "[INFO] test message with 42 value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Call function
			if tt.args != nil {
				Info(tt.format, tt.args...)
			} else {
				Info(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !strings.Contains(output, tt.expectedText) {
				t.Errorf("Info() output = %q, want to contain %q", output, tt.expectedText)
			}
		})
	}
}

//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
func TestErrorLogging(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		expectedText string
		args         []interface{}
	}{
		{
			name:         "error with simple message",
			format:       "test error",
			args:         nil,
			expectedText: "[ERROR] test error",
		},
		{
			name:         "error with formatted message",
			format:       "test %s with code %d",
			args:         []interface{}{"error", 500},
			expectedText: "[ERROR] test error with code 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:paralleltest // Cannot run in parallel due to os.Stderr manipulation
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Call function
			if tt.args != nil {
				Error(tt.format, tt.args...)
			} else {
				Error(tt.format)
			}

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !strings.Contains(output, tt.expectedText) {
				t.Errorf("Error() output = %q, want to contain %q", output, tt.expectedText)
			}
		})
	}
}

func TestLoggerStateIsolation(t *testing.T) {
	t.Parallel()
	// Reset initial state
	SetDebug(false)
	SetVerbose(false)

	// Test that debug and verbose are independent
	SetDebug(true)
	if !IsDebugEnabled() {
		t.Errorf("Debug should be enabled")
	}
	if IsVerboseEnabled() {
		t.Errorf("Verbose should still be disabled")
	}

	SetVerbose(true)
	if !IsDebugEnabled() {
		t.Errorf("Debug should still be enabled")
	}
	if !IsVerboseEnabled() {
		t.Errorf("Verbose should now be enabled")
	}

	SetDebug(false)
	if IsDebugEnabled() {
		t.Errorf("Debug should now be disabled")
	}
	if !IsVerboseEnabled() {
		t.Errorf("Verbose should still be enabled")
	}
}
