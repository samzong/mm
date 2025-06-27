package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  "Print detailed version information about this build",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version: %s\n", CLI_NAME, Version)
		fmt.Printf("Build time: %s\n", BuildTime)
	},
}
