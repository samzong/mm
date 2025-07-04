package cmd

import (
	"fmt"

	"github.com/samzong/mm/cmd/k8s"
	"github.com/samzong/mm/cmd/quality"
	"github.com/spf13/cobra"
)

var (
	CLI_NAME = "mm"

	Version   = "dev"
	BuildTime = "unknown"

	verbose bool

	rootCmd = &cobra.Command{
		Use:   CLI_NAME,
		Short: fmt.Sprintf("%s is a multi-language maintenance CLI tool", CLI_NAME),
		Long: fmt.Sprintf(`%s is a command wrapper that unifies different open source project workflows.
It provides a consistent interface for documentation synchronization across projects.`, CLI_NAME),
		Version: fmt.Sprintf("%s (built at %s)", Version, BuildTime),
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Setup command groups
	setupCommandGroups()

	// Add commands to appropriate groups
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(k8s.K8sCmd)
	rootCmd.AddCommand(quality.QualityCmd)
	rootCmd.AddCommand(formatCmd)
}

func setupCommandGroups() {
	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "project",
		Title: "Project Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "tools",
		Title: "Tool Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "basic",
		Title: "Basic Commands:",
	})

	// Set group IDs for commands
	k8s.K8sCmd.GroupID = "project"
	quality.QualityCmd.GroupID = "tools"
	formatCmd.GroupID = "tools"
	versionCmd.GroupID = "basic"
}
