package cmd

// CLI command and export-format name constants. These short strings are repeated
// across the command definitions and format-dispatch logic, so they are declared
// as constants to satisfy the goconst linter.
const (
	// Export formats.
	formatJSON = "json"
	formatYAML = "yaml"
	formatENV  = "env"
	formatCSV  = "csv"

	// Command names passed to the teller fallback.
	cmdExport = "export"
	cmdRun    = "run"
)
