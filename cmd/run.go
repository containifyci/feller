package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/containifyci/feller/pkg/config"
	"github.com/containifyci/feller/pkg/logger"
	"github.com/containifyci/feller/pkg/providers"

	"github.com/spf13/cobra"
)

var (
	resetEnv bool
	shell    bool
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [flags] -- command [args...]",
	Short: "Run a command with secrets as environment variables",
	Long: `Run a command with secrets loaded as environment variables.

The command will be executed with all secrets from the configured providers
injected into the environment.

Examples:
  feller run -- node app.js
  feller run --reset -- ./deploy.sh
  feller run --shell -- "echo $DATABASE_URL | head -c 10"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCommand,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVarP(&resetEnv, "reset", "r", false, "Reset environment variables before running")
	runCmd.Flags().BoolVarP(&shell, "shell", "s", false, "Run command as shell command")
}

func runCommand(_ *cobra.Command, args []string) error {
	logger.Debug("Starting run command with args: %v", args)
	logger.Debug("Run flags: resetEnv=%v, shell=%v", resetEnv, shell)

	// Check if we're in GitHub Actions
	if !isGitHubActions() {
		logger.Debug("Not in GitHub Actions, preparing fallback to teller")

		// Build the run command with proper flags and separator
		runArgs := []string{"run"}

		// Add run-specific flags
		if resetEnv {
			runArgs = append(runArgs, "--reset")
			logger.Debug("Added --reset flag to teller command")
		}
		if shell {
			runArgs = append(runArgs, "--shell")
			logger.Debug("Added --shell flag to teller command")
		}

		// Add the separator and command args
		runArgs = append(runArgs, "--")
		runArgs = append(runArgs, args...)

		logger.Debug("Teller fallback args: %v", runArgs)
		return fallbackToTeller(runArgs)
	}

	logger.Debug("In GitHub Actions mode, processing secrets")

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
		return handleMissingVariables(result.MissingVars)
	}

	logger.Verbose("Collected %d secrets", len(result.Secrets))
	logger.Debug("Secret keys collected: %v", getSecretKeys(result.Secrets))
	if result.HasMissingVars {
		logger.Debug("Missing %d environment variables (silent mode: %v)", len(result.MissingVars), silent)
	}

	// Prepare environment with pre-allocation
	var env []string
	if !resetEnv {
		// Start with current environment - pre-allocate for current env + secrets
		currentEnv := os.Environ()
		logger.Debug("Starting with current environment (%d vars)", len(currentEnv))
		env = make([]string, 0, len(currentEnv)+len(result.Secrets))
		env = append(env, currentEnv...)
	} else {
		logger.Debug("Resetting environment (starting with empty environment)")
		// Pre-allocate for secrets only
		env = make([]string, 0, len(result.Secrets))
	}

	// Add secrets to environment
	logger.Debug("Adding %d secrets to environment", len(result.Secrets))
	for key, value := range result.Secrets {
		envVar := fmt.Sprintf("%s=%s", key, value)
		env = append(env, envVar)
		logger.Debug("Added env var: %s=%s", key, maskSecret(value))
	}

	logger.Debug("Final environment has %d variables", len(env))

	// Execute the command
	if shell {
		logger.Debug("Executing command in shell mode")
		return executeShellCommand(args, env)
	}
	logger.Debug("Executing command in direct mode")
	return executeDirectCommand(args, env)
}

// getSecretKeys returns a slice of keys from the secret map for logging
func getSecretKeys(secrets providers.SecretMap) []string {
	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}
	return keys
}

// handleMissingVariables generates an error for missing environment variables
func handleMissingVariables(missingVars []providers.MissingVariable) error {
	if len(missingVars) == 0 {
		return nil
	}

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Missing %d required environment variable(s) in GitHub Actions:\n\n", len(missingVars)))

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
	errorMsg.WriteString("- name: Run with secrets\n")
	errorMsg.WriteString("  env:\n")
	for _, mv := range missingVars {
		errorMsg.WriteString(fmt.Sprintf("    %s: ${{ secrets.%s }}\n", mv.VariableName, mv.VariableName))
	}
	errorMsg.WriteString("  run: feller run -- your-command\n")
	errorMsg.WriteString("```\n\n")
	errorMsg.WriteString("Or use --silent flag to suppress this error and continue with available secrets only.")

	return errors.New(errorMsg.String())
}

// maskSecret masks a secret value for debug logging (same as in providers package)
func maskSecret(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func executeDirectCommand(args, env []string) error {
	if len(args) == 0 {
		logger.Debug("No command specified for direct execution")
		return errors.New("no command specified")
	}

	logger.Debug("Setting up direct command execution")
	logger.Debug("Command: %s", args[0])
	logger.Debug("Arguments: %v", args[1:])
	logger.Debug("Environment variables: %d", len(env))

	// #nosec G204 - This is intentional: tool designed to execute user-provided commands with secrets
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	logger.Verbose("Executing: %s", strings.Join(args, " "))
	logger.Debug("Starting command execution...")

	err := cmd.Run()
	if err != nil {
		logger.Debug("Command execution failed: %v", err)
		return fmt.Errorf("direct command execution failed: %w", err)
	}

	logger.Debug("Command execution completed successfully")
	return nil
}

func executeShellCommand(args, env []string) error {
	if len(args) == 0 {
		logger.Debug("No command specified for shell execution")
		return errors.New("no command specified")
	}

	// Determine shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
		logger.Debug("SHELL environment variable not set, using default: %s", shell)
	} else {
		logger.Debug("Using shell from SHELL environment variable: %s", shell)
	}

	// Join all arguments as a single command string
	cmdStr := strings.Join(args, " ")
	logger.Debug("Shell command string: %s", cmdStr)
	logger.Debug("Environment variables: %d", len(env))

	cmd := exec.CommandContext(context.Background(), shell, "-c", cmdStr)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	logger.Verbose("Executing shell: %s -c %s", shell, cmdStr)
	logger.Debug("Starting shell command execution...")

	err := cmd.Run()
	if err != nil {
		logger.Debug("Shell command execution failed: %v", err)
		return fmt.Errorf("shell command execution failed: %w", err)
	}

	logger.Debug("Shell command execution completed successfully")
	return nil
}
