package quality

import (
	"github.com/spf13/cobra"
)

// QualityCmd represents the quality command
var QualityCmd = &cobra.Command{
	Use:   "quality",
	Short: "Quality checking tools for documentation and code",
	Long: `Quality checking tools that help improve documentation and code quality.
Supports spell checking, grammar checking, and various file format validations.
Automatically adapts to different project types and loads appropriate dictionaries.`,
}

func init() {
	// Add subcommands
	QualityCmd.AddCommand(spellCmd)
}