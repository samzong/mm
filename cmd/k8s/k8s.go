package k8s

import (
	"github.com/spf13/cobra"
)

// K8sCmd represents the k8s command
var K8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Kubernetes project commands",
	Long: `Commands for managing Kubernetes project documentation and workflows.
This command provides unified access to Kubernetes-specific tools and scripts.`,
}

func init() {
	// Add subcommands
	K8sCmd.AddCommand(docsCmd)
}