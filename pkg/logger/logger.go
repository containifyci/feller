package logger

import (
	"fmt"
	"os"
)

var (
	debugEnabled   bool
	verboseEnabled bool
)

// SetDebug enables or disables debug logging
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// SetVerbose enables or disables verbose logging
func SetVerbose(enabled bool) {
	verboseEnabled = enabled
}

// Debug prints a debug message if debug logging is enabled
func Debug(format string, args ...interface{}) {
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Verbose prints a verbose message if verbose logging is enabled
func Verbose(format string, args ...interface{}) {
	if verboseEnabled {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

// Error prints an error message
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return debugEnabled
}

// IsVerboseEnabled returns true if verbose logging is enabled
func IsVerboseEnabled() bool {
	return verboseEnabled
}
