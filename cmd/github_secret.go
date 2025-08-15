package cmd

import (
	"github.com/spf13/cobra"
)

// githubSecretCmd represents the github-secret command group
var githubSecretCmd = &cobra.Command{
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

func init() {
	rootCmd.AddCommand(githubSecretCmd)
}
