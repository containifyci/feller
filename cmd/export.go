package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/containifyci/feller/pkg/config"
	"github.com/containifyci/feller/pkg/logger"
	"github.com/containifyci/feller/pkg/providers"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [format]",
	Short: "Export secrets in various formats",
	Long: `Export secrets in various formats.

Available formats:
  json - Export as JSON object
  yaml - Export as YAML document  
  env  - Export as environment variable format
  csv  - Export as CSV (key,value pairs)

Examples:
  feller export json
  feller export yaml
  feller export env`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"json", "yaml", "env", "csv"},
	RunE:      exportSecrets,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func exportSecrets(_ *cobra.Command, args []string) error {
	format := args[0]
	logger.Debug("Starting export command with format: %s", format)

	// Check if we're in GitHub Actions
	if !isGitHubActions() {
		logger.Debug("Not in GitHub Actions, falling back to teller")
		return fallbackToTeller(append([]string{"export"}, args...))
	}

	logger.Debug("In GitHub Actions mode, processing secrets for export")

	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		logger.Debug("Failed to load config: %v", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Collect all secrets and check for missing variables
	result, err := providers.CollectSecretsWithResult(cfg, silent)
	if err != nil {
		logger.Debug("Failed to collect secrets: %v", err)
		return fmt.Errorf("failed to collect secrets: %w", err)
	}

	// Handle missing environment variables
	if result.HasMissingVars && !silent {
		return handleMissingVariablesExport(result.MissingVars)
	}

	logger.Debug("Collected %d secrets for export in format: %s", len(result.Secrets), format)
	if result.HasMissingVars {
		logger.Debug("Missing %d environment variables (silent mode: %v)", len(result.MissingVars), silent)
	}

	switch format {
	case "json":
		logger.Debug("Exporting in JSON format")
		return exportJSON(result.Secrets)
	case "yaml":
		logger.Debug("Exporting in YAML format")
		return exportYAML(result.Secrets)
	case "env":
		logger.Debug("Exporting in ENV format")
		return exportEnv(result.Secrets)
	case "csv":
		logger.Debug("Exporting in CSV format")
		return exportCSV(result.Secrets)
	default:
		logger.Debug("Unsupported format requested: %s", format)
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportJSON(secrets providers.SecretMap) error {
	output, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

func exportYAML(secrets providers.SecretMap) error {
	output, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(output))
	return nil
}

func exportEnv(secrets providers.SecretMap) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := secrets[key]
		// Escape quotes and newlines for env format
		value = strings.ReplaceAll(value, `\`, `\\`)
		value = strings.ReplaceAll(value, `"`, `\"`)
		value = strings.ReplaceAll(value, "\n", `\n`)

		fmt.Printf("%s=\"%s\"\n", key, value)
	}
	return nil
}

func exportCSV(secrets providers.SecretMap) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// CSV header
	fmt.Println("key,value")

	for _, key := range keys {
		value := secrets[key]
		// Escape quotes for CSV format
		value = strings.ReplaceAll(value, `"`, `""`)

		fmt.Printf("\"%s\",\"%s\"\n", key, value)
	}
	return nil
}

// handleMissingVariablesExport generates an error for missing environment variables during export
func handleMissingVariablesExport(missingVars []providers.MissingVariable) error {
	if len(missingVars) == 0 {
		return nil
	}

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot export: Missing %d required environment variable(s) in GitHub Actions:\n\n", len(missingVars)))

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
	errorMsg.WriteString("- name: Export with secrets\n")
	errorMsg.WriteString("  env:\n")
	for _, mv := range missingVars {
		errorMsg.WriteString(fmt.Sprintf("    %s: ${{ secrets.%s }}\n", mv.VariableName, mv.VariableName))
	}
	errorMsg.WriteString("  run: feller export json\n")
	errorMsg.WriteString("```\n\n")
	errorMsg.WriteString("Or use --silent flag to export only available secrets.")

	return errors.New(errorMsg.String())
}
