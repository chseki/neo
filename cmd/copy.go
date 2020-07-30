package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy and paste manifests in YAML files",
	Long: `Example:
neo copy secrets -n lion
neo copy configmap --namespace analytics
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Copy what?")
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
