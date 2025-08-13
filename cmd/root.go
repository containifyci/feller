package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fr12k/feller/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	debug   bool
	silent  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "feller",
	Short: "A GitHub Actions optimized secret management tool",
	Long: `Feller is a lightweight secret management tool optimized for GitHub Actions.
It can parse Teller configuration files and handle secrets in GitHub Actions
environments, with fallback to the original Teller binary when not in GitHub Actions.`,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		// Initialize logging based on flags
		logger.SetDebug(debug)
		logger.SetVerbose(verbose)

		logger.Debug("Debug logging enabled")
		logger.Debug("GitHub Actions environment: %v", isGitHubActions())
		logger.Debug("Config file: %s", cfgFile)
		logger.Debug("Silent mode: %v", silent)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to your teller.yml config")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "Suppress missing environment variable errors (not recommended)")
}

// isGitHubActions checks if we're running in a GitHub Actions environment
func isGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// fallbackToTeller executes the original teller binary with the same arguments
func fallbackToTeller(args []string) error {
	logger.Verbose("Not in GitHub Actions environment, falling back to teller")
	logger.Debug("Building teller command arguments")

	// Build the full argument list
	tellerArgs := []string{}

	// Add global flags if they were set
	if cfgFile != "" {
		tellerArgs = append(tellerArgs, "--config", cfgFile)
		logger.Debug("Added --config flag: %s", cfgFile)
	}
	if verbose {
		tellerArgs = append(tellerArgs, "--verbose")
		logger.Debug("Added --verbose flag")
	}

	// Add the command and its arguments
	tellerArgs = append(tellerArgs, args...)
	logger.Debug("Final teller arguments: %v", tellerArgs)

	// Look for teller binary
	logger.Debug("Searching for teller binary")
	tellerPath, err := findTellerBinary()
	if err != nil {
		logger.Debug("Failed to find teller binary: %v", err)
		return fmt.Errorf("failed to find teller binary: %w", err)
	}

	logger.Debug("Found teller binary at: %s", tellerPath)

	// Execute teller with syscall.Exec for complete replacement
	return execTeller(tellerPath, tellerArgs)
}

// findTellerBinary locates the teller binary in the system PATH
func findTellerBinary() (string, error) {
	// Look for common teller binary names
	candidates := []string{"teller", "teller-original"}
	logger.Debug("Searching for teller binary candidates: %v", candidates)

	for _, candidate := range candidates {
		logger.Debug("Checking for binary: %s", candidate)
		path, err := exec.LookPath(candidate)
		if err == nil {
			logger.Debug("Found binary '%s' at path: %s", candidate, path)
			return path, nil
		}
		logger.Debug("Binary '%s' not found: %v", candidate, err)
	}

	logger.Debug("No teller binary found in PATH")
	return "", errors.New("teller binary not found in PATH")
}

// execTeller executes the teller binary, replacing the current process
func execTeller(tellerPath string, args []string) error {
	logger.Debug("Setting up teller execution")
	logger.Debug("Binary path: %s", tellerPath)
	logger.Debug("Arguments: %v", args)

	// Use exec.CommandContext for compatibility and proper error handling
	cmd := exec.CommandContext(context.Background(), tellerPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	logger.Verbose("Executing: %s %s", tellerPath, strings.Join(args, " "))
	logger.Debug("Starting teller execution...")

	err := cmd.Run()
	if err != nil {
		logger.Debug("Teller execution failed: %v", err)
		exitError := &exec.ExitError{}
		if errors.As(err, &exitError) {
			logger.Debug("Teller exited with code: %d", exitError.ExitCode())
			os.Exit(exitError.ExitCode())
		}
		return fmt.Errorf("teller execution failed: %w", err)
	}

	logger.Debug("Teller execution completed successfully")
	return nil
}
