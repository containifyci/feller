package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/containifyci/feller/pkg/config"
	"github.com/containifyci/feller/pkg/providers"
	"github.com/spf13/cobra"
)

// shCmd represents the sh command
var shCmd = &cobra.Command{
	Use:   "sh",
	Short: "Export secrets as shell export statements",
	Long: `Export secrets as shell export statements that can be evaluated
to set environment variables in the current shell.

Examples:
  eval "$(feller sh)"
  feller sh > secrets.sh && source secrets.sh`,
	RunE: exportShell,
}

func init() {
	rootCmd.AddCommand(shCmd)
}

func exportShell(_ *cobra.Command, args []string) error {
	// Check if we're in GitHub Actions
	if !isGitHubActions() {
		return fallbackToTeller(append([]string{"sh"}, args...))
	}

	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Collect all secrets and check for missing variables
	result, err := providers.CollectSecretsWithResult(cfg, silent)
	if err != nil {
		return fmt.Errorf("failed to collect secrets: %w", err)
	}

	// Handle missing environment variables
	if result.HasMissingVars && !silent {
		return handleMissingVariablesShell(result.MissingVars)
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(result.Secrets))
	for k := range result.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := result.Secrets[key]
		// Shell-escape the value
		// For safety, we'll single-quote the value and escape any single quotes within it
		escapedValue := "'" + shellEscape(value) + "'"
		fmt.Printf("export %s=%s\n", key, escapedValue)
	}

	return nil
}

// shellEscape escapes single quotes in a string for use within single quotes
func shellEscape(s string) string {
	// Replace any single quote with '\''
	// This ends the current single-quoted string, adds an escaped single quote, then starts a new single-quoted string
	return shellReplaceAll(s, "'", "'\\''")
}

// shellReplaceAll is a simple string replacement function
func shellReplaceAll(s, old, replacement string) string {
	// Handle edge case: empty old string should return original string
	if old == "" {
		return s
	}
	var result strings.Builder
	for {
		i := shellIndexOf(s, old)
		if i == -1 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:i] + replacement)
		s = s[i+len(old):]
	}
	return result.String()
}

// shellIndexOf finds the index of substr in s, or -1 if not found
func shellIndexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// handleMissingVariablesShell generates an error for missing environment variables during shell export
func handleMissingVariablesShell(missingVars []providers.MissingVariable) error {
	if len(missingVars) == 0 {
		return nil
	}

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot generate shell exports: Missing %d required environment variable(s) in GitHub Actions:\n\n", len(missingVars)))

	// Group by provider for better organization
	providerGroups := make(map[string][]providers.MissingVariable)
	for _, mv := range missingVars {
		providerGroups[mv.Provider] = append(providerGroups[mv.Provider], mv)
	}

	for provider, vars := range providerGroups {
		errorMsg.WriteString(fmt.Sprintf("Provider '%s':\n", provider))
		for _, mv := range vars {
			errorMsg.WriteString(fmt.Sprintf("  • %s (maps to: %s)\n", mv.VariableName, mv.MappedTo))
		}
		errorMsg.WriteString("\n")
	}

	errorMsg.WriteString("To fix this, add the missing environment variables to your GitHub Actions workflow:\n\n")
	errorMsg.WriteString("```yaml\n")
	errorMsg.WriteString("- name: Set shell variables\n")
	errorMsg.WriteString("  env:\n")
	for _, mv := range missingVars {
		errorMsg.WriteString(fmt.Sprintf("    %s: ${{ secrets.%s }}\n", mv.VariableName, mv.VariableName))
	}
	errorMsg.WriteString("  run: eval \"$(feller sh)\"\n")
	errorMsg.WriteString("```\n\n")
	errorMsg.WriteString("Or use --silent flag to export only available secrets.")

	return errors.New(errorMsg.String())
}
