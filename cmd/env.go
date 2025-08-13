package cmd

import (
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Export secrets in environment variable format",
	Long: `Export secrets in environment variable format suitable for sourcing
or using with tools like docker --env-file.

This is equivalent to 'feller export env'.

Examples:
  feller env
  feller env > .env.secrets
  docker run --env-file <(feller env) myapp`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return exportSecrets(cmd, []string{"env"})
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
