package logger

import (
	"fmt"
	"os"
	"sync/atomic"
)

var (
	debugEnabled   int32
	verboseEnabled int32
)

// SetDebug enables or disables debug logging
func SetDebug(enabled bool) {
	if enabled {
		atomic.StoreInt32(&debugEnabled, 1)
	} else {
		atomic.StoreInt32(&debugEnabled, 0)
	}
}

// SetVerbose enables or disables verbose logging
func SetVerbose(enabled bool) {
	if enabled {
		atomic.StoreInt32(&verboseEnabled, 1)
	} else {
		atomic.StoreInt32(&verboseEnabled, 0)
	}
}

// Debug prints a debug message if debug logging is enabled
func Debug(format string, args ...interface{}) {
	if atomic.LoadInt32(&debugEnabled) != 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Verbose prints a verbose message if verbose logging is enabled
func Verbose(format string, args ...interface{}) {
	if atomic.LoadInt32(&verboseEnabled) != 0 {
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
	return atomic.LoadInt32(&debugEnabled) != 0
}

// IsVerboseEnabled returns true if verbose logging is enabled
func IsVerboseEnabled() bool {
	return atomic.LoadInt32(&verboseEnabled) != 0
}
