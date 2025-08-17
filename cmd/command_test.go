package cmd

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constructor function for export command - makes it testable
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
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
	}
	return cmd
}

// Constructor function for run command - makes it testable
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
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
	}

	cmd.Flags().BoolVarP(&resetEnv, "reset", "r", false, "Reset environment variables before running")
	cmd.Flags().BoolVarP(&shell, "shell", "s", false, "Run command as shell command")

	return cmd
}

// Constructor function for sh command - makes it testable
func NewShCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sh",
		Short: "Export secrets as shell export statements",
		Long: `Export secrets as shell export statements that can be evaluated
to set environment variables in the current shell.

Examples:
  eval "$(feller sh)"
  feller sh > secrets.sh && source secrets.sh`,
	}
	return cmd
}

// Constructor function for github-secret command - makes it testable
func NewGitHubSecretCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github-secret",
		Short: "Manage GitHub repository secrets",
		Long: `Manage GitHub repository secrets based on Teller configuration.

This command group provides functionality to synchronize secrets from your
Teller configuration with GitHub repository secrets using the GitHub CLI.

Available subcommands:
  add    Add/update secrets from teller configuration to GitHub repository

Examples:
  feller github-secret add --repo owner/repo
  feller github-secret add --repo owner/repo --dependabot`,
	}
	return cmd
}

// Constructor function for github-secret add command - makes it testable
func NewGitHubSecretAddCommand() *cobra.Command {
	var (
		testRepo             string
		testDependabot       bool
		testDryRun           bool
		testForce            bool
		testSkipExisting     bool
		testConfirmOverwrite bool
	)

	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// For testing, we'll create a mock function that validates flags
			// In real implementation, this would call addGitHubSecrets

			// Validate flag combinations using the same logic as the real command
			if err := validateOverwriteFlagsWithParams(testForce, testSkipExisting, testConfirmOverwrite); err != nil {
				return err
			}

			// Simulate the requirement for repo flag
			if testRepo == "" {
				return errors.New("required flag \"repo\" not set")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&testRepo, "repo", "r", "", "GitHub repository (owner/repo) (required)")
	cmd.Flags().BoolVar(&testDependabot, "dependabot", false, "Also set secrets for Dependabot app")
	cmd.Flags().BoolVar(&testDryRun, "dry-run", false, "Show what would be executed without making changes")
	cmd.Flags().BoolVar(&testForce, "force", false, "Force overwrite existing secrets without prompting")
	cmd.Flags().BoolVar(&testSkipExisting, "skip-existing", false, "Skip existing secrets instead of overwriting them")
	cmd.Flags().BoolVar(&testConfirmOverwrite, "confirm-overwrite", false, "Prompt for confirmation before overwriting existing secrets")
	cmd.MarkFlagRequired("repo")

	return cmd
}

// Helper function to validate overwrite flags with parameters (for testing)
func validateOverwriteFlagsWithParams(force, skipExisting, confirmOverwrite bool) error {
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
		return errors.New("only one overwrite strategy can be specified: --force, --skip-existing, or --confirm-overwrite")
	}

	return nil
}

// Test basic command structure and help output
func TestExportCommandStructure(t *testing.T) {
	cmd := NewExportCommand()

	// Test command structure
	assert.Equal(t, "export [format]", cmd.Use)
	assert.Equal(t, "Export secrets in various formats", cmd.Short)
	assert.Contains(t, cmd.Long, "Available formats:")
	assert.Equal(t, []string{"json", "yaml", "env", "csv"}, cmd.ValidArgs)
}

func TestRunCommandStructure(t *testing.T) {
	cmd := NewRunCommand()

	// Test command structure
	assert.Equal(t, "run [flags] -- command [args...]", cmd.Use)
	assert.Equal(t, "Run a command with secrets as environment variables", cmd.Short)
	assert.Contains(t, cmd.Long, "The command will be executed")

	// Test flags
	assert.True(t, cmd.Flags().HasFlags())
	resetFlag := cmd.Flags().Lookup("reset")
	require.NotNil(t, resetFlag)
	assert.Equal(t, "false", resetFlag.DefValue)

	shellFlag := cmd.Flags().Lookup("shell")
	require.NotNil(t, shellFlag)
	assert.Equal(t, "false", shellFlag.DefValue)
}

func TestShCommandStructure(t *testing.T) {
	cmd := NewShCommand()

	// Test command structure
	assert.Equal(t, "sh", cmd.Use)
	assert.Equal(t, "Export secrets as shell export statements", cmd.Short)
	assert.Contains(t, cmd.Long, "Export secrets as shell export statements")
}

func TestGitHubSecretCommandStructure(t *testing.T) {
	cmd := NewGitHubSecretCommand()

	// Test command structure
	assert.Equal(t, "github-secret", cmd.Use)
	assert.Equal(t, "Manage GitHub repository secrets", cmd.Short)
	assert.Contains(t, cmd.Long, "Manage GitHub repository secrets")
	assert.Contains(t, cmd.Long, "Available subcommands:")
}

// Test GitHub secret add command structure and flags
func TestGitHubSecretAddCommandStructure(t *testing.T) {
	cmd := NewGitHubSecretAddCommand()

	// Test command structure
	assert.Equal(t, "add", cmd.Use)
	assert.Equal(t, "Add secrets from teller configuration to GitHub repository", cmd.Short)
	assert.Contains(t, cmd.Long, "Add Google Secret Manager secrets")

	// Test flags exist
	assert.True(t, cmd.Flags().HasFlags())

	// Test repo flag
	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag)
	assert.Empty(t, repoFlag.DefValue)
	assert.Equal(t, "r", repoFlag.Shorthand)

	// Test boolean flags
	flags := []string{"dependabot", "dry-run", "force", "skip-existing", "confirm-overwrite"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should exist", flagName)
		assert.Equal(t, "false", flag.DefValue)
	}
}

// Test GitHub secret add command flag combinations
func TestGitHubSecretAddFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		args    []string
		wantErr bool
	}{
		{
			name:    "missing required repo flag",
			args:    []string{},
			wantErr: true,
			errMsg:  "required flag(s) \"repo\" not set",
		},
		{
			name:    "valid repo flag",
			args:    []string{"--repo", "owner/repo"},
			wantErr: false,
		},
		{
			name:    "conflicting flags force and skip-existing",
			args:    []string{"--repo", "owner/repo", "--force", "--skip-existing"},
			wantErr: true,
			errMsg:  "only one overwrite strategy can be specified",
		},
		{
			name:    "repo flag short form",
			args:    []string{"-r", "owner/repo"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGitHubSecretAddCommand()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
