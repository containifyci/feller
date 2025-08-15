package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/containifyci/feller/pkg/config"
	"github.com/containifyci/feller/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	repo             string
	dependabot       bool
	dryRun           bool
	force            bool
	skipExisting     bool
	confirmOverwrite bool

	// Interactive confirmation state
	yesToAll bool
	noToAll  bool
)

// SecretOperationStats tracks statistics for secret operations
type SecretOperationStats struct {
	Created int
	Updated int
	Skipped int
	Failed  int
}

// ExistingSecrets represents existing secrets in GitHub
type ExistingSecrets struct {
	Repository map[string]bool // repository secret names -> exists
	Dependabot map[string]bool // dependabot secret names -> exists
}

// GitHubSecret represents a secret returned by gh secret list
type GitHubSecret struct {
	Name string `json:"name"`
}

// githubSecretAddCmd represents the github-secret add command
var githubSecretAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add secrets from teller configuration to GitHub repository",
	Long: `Add Google Secret Manager secrets from teller configuration to GitHub repository.

This command reads your teller configuration, retrieves only secrets from 
Google Secret Manager providers using the original teller binary, and uploads 
them to GitHub repository secrets using the GitHub CLI.

Only secrets defined in 'google_secretmanager' providers will be uploaded.
Secrets from 'dotenv' providers are ignored as they are meant for local development.

Overwrite Behavior:
By default, existing secrets are overwritten without prompting. You can control
this behavior with the following flags:

  --force             Force overwrite existing secrets (default behavior)
  --skip-existing     Skip existing secrets instead of overwriting them
  --confirm-overwrite Prompt for confirmation before overwriting each existing secret

Only one overwrite strategy can be specified at a time.

The command requires:
- GitHub CLI (gh) to be installed and authenticated
- Original teller binary to be available in PATH
- Repository access permissions for the target repository

Examples:
  # Basic usage (overwrites existing secrets)
  feller github-secret add --repo owner/repo
  
  # Include Dependabot secrets
  feller github-secret add --repo owner/repo --dependabot
  
  # Preview changes without making them
  feller github-secret add --repo owner/repo --dry-run
  
  # Skip existing secrets instead of overwriting
  feller github-secret add --repo owner/repo --skip-existing
  
  # Prompt before overwriting each existing secret
  feller github-secret add --repo owner/repo --confirm-overwrite
  
  # Force overwrite (explicit default behavior)
  feller github-secret add --repo owner/repo --force`,
	RunE: addGitHubSecrets,
}

func init() {
	githubSecretCmd.AddCommand(githubSecretAddCmd)
	githubSecretAddCmd.Flags().StringVarP(&repo, "repo", "r", "", "GitHub repository (owner/repo) (required)")
	githubSecretAddCmd.Flags().BoolVar(&dependabot, "dependabot", false, "Also set secrets for Dependabot app")
	githubSecretAddCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without making changes")
	githubSecretAddCmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing secrets without prompting")
	githubSecretAddCmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "Skip existing secrets instead of overwriting them")
	githubSecretAddCmd.Flags().BoolVar(&confirmOverwrite, "confirm-overwrite", false, "Prompt for confirmation before overwriting existing secrets")
	githubSecretAddCmd.MarkFlagRequired("repo")
}

func addGitHubSecrets(_ *cobra.Command, _ []string) error {
	logger.Debug("Starting github-secret add command")
	logger.Debug("Repository: %s, Dependabot: %v, Dry run: %v", repo, dependabot, dryRun)

	// Validate flag combinations
	if err := validateOverwriteFlags(); err != nil {
		return err
	}

	// Validate required tools
	if err := validateRequiredTools(); err != nil {
		logger.Debug("Tool validation failed: %v", err)
		return err
	}

	// Get secrets using teller
	secrets, err := getSecretsFromTeller()
	if err != nil {
		logger.Debug("Failed to get secrets from teller: %v", err)
		return fmt.Errorf("failed to get secrets from teller: %w", err)
	}

	logger.Debug("Retrieved %d secrets from teller", len(secrets))

	// Get existing secrets for comparison
	existingSecrets, err := getExistingGitHubSecrets()
	if err != nil {
		logger.Debug("Failed to get existing GitHub secrets: %v", err)
		return fmt.Errorf("failed to get existing GitHub secrets: %w", err)
	}

	logger.Debug("Found %d existing repository secrets", len(existingSecrets.Repository))
	if dependabot {
		logger.Debug("Found %d existing Dependabot secrets", len(existingSecrets.Dependabot))
	}

	// Set secrets in GitHub
	stats, err := setGitHubSecrets(secrets, existingSecrets)
	if err != nil {
		logger.Debug("Failed to set GitHub secrets: %v", err)
		return fmt.Errorf("failed to set GitHub secrets: %w", err)
	}

	// Print summary report
	printOperationSummary(stats)

	logger.Verbose("Successfully configured %d GitHub secrets for repository %s", len(secrets), repo)
	return nil
}

// validateOverwriteFlags ensures only one overwrite strategy is selected
func validateOverwriteFlags() error {
	flagCount := 0
	if force {
		flagCount++
	}
	if skipExisting {
		flagCount++
	}
	if confirmOverwrite {
		flagCount++
	}

	if flagCount > 1 {
		return errors.New("only one of --force, --skip-existing, or --confirm-overwrite can be specified")
	}

	return nil
}

// printOperationSummary prints a summary of secret operations
func printOperationSummary(stats *SecretOperationStats) {
	if dryRun {
		fmt.Println("\nDry-run summary:")
	} else {
		fmt.Println("\nOperation summary:")
	}

	if stats.Created > 0 {
		fmt.Printf("  Created: %d secrets\n", stats.Created)
	}
	if stats.Updated > 0 {
		fmt.Printf("  Updated: %d secrets\n", stats.Updated)
	}
	if stats.Skipped > 0 {
		fmt.Printf("  Skipped: %d secrets\n", stats.Skipped)
	}
	if stats.Failed > 0 {
		fmt.Printf("  Failed:  %d secrets\n", stats.Failed)
	}

	total := stats.Created + stats.Updated + stats.Skipped + stats.Failed
	if total == 0 {
		fmt.Println("  No secrets processed")
	}
}

// promptForOverwrite asks user for confirmation to overwrite an existing secret
func promptForOverwrite(secretName, target string) bool {
	// If we already have a global decision, use it
	if yesToAll {
		return true
	}
	if noToAll {
		return false
	}

	// Skip prompts in dry-run mode
	if dryRun {
		logger.Debug("Dry-run: Would prompt for overwrite of %s secret: %s", target, secretName)
		return true // Assume yes for dry-run simulation
	}

	// Prompt user
	fmt.Printf("Secret '%s' already exists in %s. Overwrite? [y/n/ya/na]: ", secretName, target)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			// Handle EOF or error (e.g., Ctrl+C)
			fmt.Println("\nOperation cancelled.")
			return false
		}

		response := strings.ToLower(strings.TrimSpace(scanner.Text()))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		case "ya", "yes-to-all":
			yesToAll = true
			logger.Debug("User selected 'yes to all' - will overwrite all remaining secrets")
			return true
		case "na", "no-to-all":
			noToAll = true
			logger.Debug("User selected 'no to all' - will skip all remaining secrets")
			return false
		default:
			fmt.Printf("Please enter y(es), n(o), ya (yes to all), or na (no to all): ")
		}
	}
}

// validateRequiredTools checks if required tools are available
func validateRequiredTools() error {
	logger.Debug("Validating required tools")

	// Check for GitHub CLI
	if _, err := exec.LookPath("gh"); err != nil {
		logger.Debug("GitHub CLI not found: %v", err)
		return errors.New("GitHub CLI (gh) not found - please install and authenticate with GitHub CLI")
	}

	// Check GitHub CLI authentication (skip in dry-run mode for testing)
	if !dryRun {
		cmd := exec.Command("gh", "auth", "status")
		if err := cmd.Run(); err != nil {
			logger.Debug("GitHub CLI authentication failed: %v", err)
			return errors.New("GitHub CLI not authenticated - run 'gh auth login' first")
		}
	}

	logger.Debug("GitHub CLI is available and authenticated")

	// Check for teller binary
	tellerPath, err := findTellerBinary()
	if err != nil {
		logger.Debug("Teller binary not found: %v", err)
		return fmt.Errorf("teller binary not found: %w", err)
	}

	logger.Debug("Found teller binary at: %s", tellerPath)
	return nil
}

// getSecretsFromTeller retrieves only GSM secrets using the teller binary
func getSecretsFromTeller() (map[string]string, error) {
	logger.Debug("Retrieving GSM secrets from teller")

	// Load configuration to identify GSM secrets
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		logger.Debug("Failed to load config: %v", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get GSM providers to determine which secrets we want
	gsmProviders := cfg.GetProvidersByKind("google_secretmanager")
	if len(gsmProviders) == 0 {
		logger.Debug("No GSM providers found in configuration")
		return map[string]string{}, nil
	}

	logger.Debug("Found %d GSM providers", len(gsmProviders))

	// Build expected GSM secret keys and reverse mapping from configuration
	expectedGSMKeys := make(map[string]bool)
	outputKeyToGSMKey := make(map[string]string)
	for providerName, provider := range gsmProviders {
		logger.Debug("Processing GSM provider: %s", providerName)
		for _, pathMap := range provider.Maps {
			for gsmKey, outputKey := range pathMap.Keys {
				expectedGSMKeys[outputKey] = true
				outputKeyToGSMKey[outputKey] = gsmKey
				logger.Debug("Expected GSM secret: %s -> %s (GSM key -> output key)", gsmKey, outputKey)
			}
		}
	}

	// Find teller binary
	tellerPath, err := findTellerBinary()
	if err != nil {
		return nil, err
	}

	// Build teller command arguments
	args := []string{"export", "json"}
	if cfgFile != "" {
		args = append([]string{"--config", cfgFile}, args...)
		logger.Debug("Using config file: %s", cfgFile)
	}
	if verbose {
		args = append([]string{"--verbose"}, args...)
	}

	logger.Debug("Executing: %s %s", tellerPath, strings.Join(args, " "))

	// Execute teller export json
	cmd := exec.CommandContext(context.Background(), tellerPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			logger.Debug("Teller stderr: %s", string(exitError.Stderr))
		}
		return nil, fmt.Errorf("failed to execute teller export: %w", err)
	}

	logger.Debug("Teller output length: %d bytes", len(output))

	// Parse JSON output
	var allSecrets map[string]string
	if err := json.Unmarshal(output, &allSecrets); err != nil {
		logger.Debug("Failed to parse teller JSON output: %v", err)
		logger.Debug("Raw output: %s", string(output))
		return nil, fmt.Errorf("failed to parse teller JSON output: %w", err)
	}

	logger.Debug("Parsed %d total secrets from teller output", len(allSecrets))

	// Filter to only include GSM secrets and map back to GSM key names
	gsmSecrets := make(map[string]string)
	for outputKey, value := range allSecrets {
		if expectedGSMKeys[outputKey] {
			gsmKey := outputKeyToGSMKey[outputKey]
			gsmSecrets[gsmKey] = value
			logger.Debug("Including GSM secret: %s (output key: %s)", gsmKey, outputKey)
		} else {
			logger.Debug("Skipping non-GSM secret key: %s", outputKey)
		}
	}

	logger.Debug("Filtered to %d GSM secrets for GitHub upload", len(gsmSecrets))
	return gsmSecrets, nil
}

// getExistingGitHubSecrets retrieves existing secrets from GitHub repository
func getExistingGitHubSecrets() (*ExistingSecrets, error) {
	logger.Debug("Retrieving existing GitHub secrets")

	existing := &ExistingSecrets{
		Repository: make(map[string]bool),
		Dependabot: make(map[string]bool),
	}

	// Get repository secrets
	if secrets, err := listGitHubSecrets(false); err != nil {
		return nil, fmt.Errorf("failed to list repository secrets: %w", err)
	} else {
		for _, secret := range secrets {
			existing.Repository[secret] = true
			logger.Debug("Found existing repository secret: %s", secret)
		}
	}

	// Get Dependabot secrets if needed
	if dependabot {
		if secrets, err := listGitHubSecrets(true); err != nil {
			return nil, fmt.Errorf("failed to list Dependabot secrets: %w", err)
		} else {
			for _, secret := range secrets {
				existing.Dependabot[secret] = true
				logger.Debug("Found existing Dependabot secret: %s", secret)
			}
		}
	}

	return existing, nil
}

// listGitHubSecrets lists secrets for repository or Dependabot
func listGitHubSecrets(isDependabot bool) ([]string, error) {
	target := "repository"
	if isDependabot {
		target = "Dependabot"
	}

	logger.Debug("Listing %s secrets", target)

	// Skip actual listing in dry-run mode to avoid API calls
	// if dryRun {
	// 	logger.Debug("Skipping secret listing in dry-run mode")
	// 	return []string{}, nil
	// }

	// Build gh command
	args := []string{"secret", "list", "--repo", repo, "--json", "name"}
	if isDependabot {
		args = append(args[:2], append([]string{"--app", "dependabot"}, args[2:]...)...)
	}

	logger.Debug("Executing: gh %s", strings.Join(args, " "))

	// Execute gh secret list
	cmd := exec.CommandContext(context.Background(), "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			logger.Debug("gh stderr: %s", string(exitError.Stderr))
		}
		return nil, fmt.Errorf("failed to list %s secrets: %w", target, err)
	}

	// Parse JSON output
	var secrets []GitHubSecret
	if err := json.Unmarshal(output, &secrets); err != nil {
		logger.Debug("Failed to parse secret list JSON: %v", err)
		return nil, fmt.Errorf("failed to parse secret list JSON: %w", err)
	}

	// Extract secret names
	var names []string
	for _, secret := range secrets {
		names = append(names, secret.Name)
	}

	logger.Debug("Found %d existing %s secrets", len(names), target)
	return names, nil
}

// setGitHubSecrets uploads secrets to GitHub repository and returns operation statistics
func setGitHubSecrets(secrets map[string]string, existing *ExistingSecrets) (*SecretOperationStats, error) {
	logger.Debug("Setting GitHub secrets for repository: %s", repo)

	stats := &SecretOperationStats{}

	for key, value := range secrets {
		// Check and set repository secret
		if result, err := setGitHubSecretIfNeeded(key, value, false, existing); err != nil {
			stats.Failed++
			return stats, fmt.Errorf("failed to set secret %s: %w", key, err)
		} else {
			updateStats(stats, result)
		}

		// Also set for Dependabot if requested
		if dependabot {
			if result, err := setGitHubSecretIfNeeded(key, value, true, existing); err != nil {
				stats.Failed++
				return stats, fmt.Errorf("failed to set Dependabot secret %s: %w", key, err)
			} else {
				updateStats(stats, result)
			}
		}
	}

	return stats, nil
}

// updateStats updates the statistics based on the operation result
func updateStats(stats *SecretOperationStats, result string) {
	switch result {
	case "created":
		stats.Created++
	case "updated":
		stats.Updated++
	case "skipped":
		stats.Skipped++
	}
}

// setGitHubSecretIfNeeded sets a secret based on the selected overwrite strategy and returns the operation type
func setGitHubSecretIfNeeded(key, value string, isDependabot bool, existing *ExistingSecrets) (string, error) {
	target := "repository"
	existingSecrets := existing.Repository
	if isDependabot {
		target = "Dependabot"
		existingSecrets = existing.Dependabot
	}

	// Check if secret already exists
	if existingSecrets[key] {
		// Handle existing secret based on flags
		switch {
		case skipExisting:
			logger.Debug("Skipping existing %s secret: %s", target, key)
			logger.Verbose("Skipped existing %s secret: %s", target, key)
			return "skipped", nil
		case confirmOverwrite:
			if !promptForOverwrite(key, target) {
				logger.Debug("User chose not to overwrite %s secret: %s", target, key)
				logger.Verbose("Skipped %s secret: %s (user declined)", target, key)
				return "skipped", nil
			}
			logger.Debug("User confirmed overwrite of %s secret: %s", target, key)
			logger.Verbose("Updating existing %s secret: %s (user confirmed)", target, key)
		case force:
			logger.Debug("Force overwriting existing %s secret: %s", target, key)
			logger.Verbose("Updating existing %s secret: %s (forced)", target, key)
		default:
			// Default behavior - overwrite without prompting (backward compatibility)
			logger.Debug("%s secret '%s' already exists, updating it", target, key)
			logger.Verbose("Updating existing %s secret: %s", target, key)
		}

		if err := setGitHubSecret(key, value, isDependabot); err != nil {
			return "", err
		}
		return "updated", nil
	} else {
		logger.Debug("%s secret '%s' does not exist, creating it", target, key)
		logger.Verbose("Creating new %s secret: %s", target, key)

		if err := setGitHubSecret(key, value, isDependabot); err != nil {
			return "", err
		}
		return "created", nil
	}
}

// setGitHubSecret sets a single secret in GitHub
func setGitHubSecret(key, value string, isDependabot bool) error {
	target := "repository"
	if isDependabot {
		target = "Dependabot"
	}

	logger.Debug("Setting %s secret: %s", target, key)

	if dryRun {
		if isDependabot {
			fmt.Printf("Would execute: gh secret set %s --app dependabot --repo %s --body \"<redacted>\"\n", key, repo)
		} else {
			fmt.Printf("Would execute: gh secret set %s --repo %s --body \"<redacted>\"\n", key, repo)
		}
		return nil
	}

	// Build gh command
	args := []string{"secret", "set", key, "--repo", repo, "--body", value}
	if isDependabot {
		args = append(args[:3], append([]string{"--app", "dependabot"}, args[3:]...)...)
	}

	logger.Debug("Executing: gh %s", strings.Join(args[:len(args)-1], " ")+" --body <redacted>")

	// Execute gh secret set
	cmd := exec.CommandContext(context.Background(), "gh", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			logger.Debug("gh stderr: %s", string(output))
		}
		return fmt.Errorf("failed to set %s secret %s: %w", target, key, err)
	}

	logger.Verbose("Set %s secret: %s", target, key)
	return nil
}
