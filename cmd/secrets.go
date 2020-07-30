package cmd

import (
	"neo/kubectl"

	"github.com/spf13/cobra"
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Copy and paste all secrets from any namespace that exists in your current cluster",

	Run: func(cmd *cobra.Command, args []string) {
		s := kubectl.Factory(kubectl.Secret)

		flagns, _ := cmd.Flags().GetString("namespace")

		s.Copy(flagns)
	},
}

func init() {
	copyCmd.AddCommand(secretsCmd)

	secretsCmd.Flags().StringP("namespace", "n", "default", "Namespace")
}
