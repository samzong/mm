package cmd

import (
	"github.com/samzong/mm/cmd/format"
	"github.com/spf13/cobra"
)

// formatCmd represents the format command
var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Document formatting commands",
	Long: `Commands for formatting and standardizing documentation according to project guidelines.
This command provides automated formatting for different project types and documentation standards.`,
}

func init() {
	// Add subcommands
	formatCmd.AddCommand(format.K8sCmd)
}