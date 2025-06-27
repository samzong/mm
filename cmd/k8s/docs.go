package k8s

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Documentation management commands",
	Long:  `Commands for managing Kubernetes documentation synchronization.`,
}

// lsyncCmd represents the lsync command
var lsyncCmd = &cobra.Command{
	Use:   "lsync [path]",
	Short: "Language synchronization for documentation",
	Long: `Check documentation synchronization between different languages.
This command calls the Kubernetes lsync.sh script to identify outdated translations.

Examples:
  mm k8s docs lsync                                      # Check all documents
  mm k8s docs lsync content/zh-cn/docs/concepts/        # Check specific directory
  mm k8s docs lsync content/zh-cn/docs/concepts/cri.md  # Check specific file`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		checkPR, _ := cmd.Flags().GetBool("check-pr")
		
		// Check if we're in a k8s project directory
		if !isK8sProject() {
			return fmt.Errorf("scripts/lsync.sh not found. Please run this command in the kubernetes/website project root")
		}

		// Determine the path to check
		var targetPath string
		if len(args) > 0 {
			inputPath := args[0]
			// If user provides English path, convert to corresponding localized path
			if strings.HasPrefix(inputPath, "content/en/") {
				// Try to find corresponding zh-cn file
				zhPath := strings.Replace(inputPath, "content/en/", "content/zh-cn/", 1)
				if _, err := os.Stat(zhPath); err == nil {
					targetPath = zhPath
				} else {
					return fmt.Errorf("corresponding Chinese file not found: %s", zhPath)
				}
			} else {
				targetPath = inputPath
			}
		} else {
			targetPath = "content/zh-cn/"
		}

		// Execute lsync.sh
		result, err := executeLsync(targetPath)
		if err != nil {
			return fmt.Errorf("failed to execute lsync: %w", err)
		}

		// Display results
		if result.hasChanges {
			if result.isSingleFile {
				// For single file, show detailed diff directly
				fmt.Print(result.rawOutput)
			} else {
				// For multiple files, show summary table
				fmt.Printf("%-8s %-8s %s\n", "Added", "Deleted", "File")
				fmt.Printf("%-8s %-8s %s\n", "-----", "-------", "----")
				for _, file := range result.files {
					fmt.Printf("%-8d %-8d %s\n", file.addedLines, file.deletedLines, file.filePath)
				}
			}
		} else {
			fmt.Printf("✅ All files are up to date\n")
		}

		// Check PR if requested
		if checkPR && len(result.files) > 0 {
			fmt.Printf("\n🔍 Checking related PRs...\n")
			// TODO: Implement PR checking logic
			fmt.Printf("  └── PR checking will be implemented in next iteration\n")
		}

		return nil
	},
}

// fileChange represents a single file change with statistics
type fileChange struct {
	addedLines   int
	deletedLines int
	filePath     string
}

// lsyncResult represents the result of lsync execution
type lsyncResult struct {
	files      []fileChange
	hasChanges bool
	rawOutput  string  // Store raw output for single file diff display
	isSingleFile bool  // Track if this was a single file check
}

// isK8sProject checks if current directory is a k8s project
func isK8sProject() bool {
	_, err := os.Stat("./scripts/lsync.sh")
	return err == nil
}

// executeLsync runs the lsync.sh script and parses the output
func executeLsync(path string) (*lsyncResult, error) {
	cmd := exec.Command("./scripts/lsync.sh", path)
	output, _ := cmd.CombinedOutput()
	
	// Check if this is a single file (ends with .md and is a file)
	isSingleFile := strings.HasSuffix(path, ".md")
	if isSingleFile {
		if stat, err := os.Stat(path); err != nil || stat.IsDir() {
			isSingleFile = false
		}
	}
	
	// Parse output to extract file list
	result := &lsyncResult{
		files:        []fileChange{},
		rawOutput:    string(output),
		isSingleFile: isSingleFile,
	}

	lines := strings.Split(string(output), "\n")
	var hasNumstat bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Check if this is numstat format: "added_lines deleted_lines filename"
			parts := strings.Fields(line)
			if len(parts) >= 3 && strings.HasPrefix(parts[2], "content/") {
				// Parse the numbers
				added, err1 := strconv.Atoi(parts[0])
				deleted, err2 := strconv.Atoi(parts[1])
				
				if err1 == nil && err2 == nil {
					result.files = append(result.files, fileChange{
						addedLines:   added,
						deletedLines: deleted,
						filePath:     parts[2],
					})
					hasNumstat = true
				}
			}
		}
	}
	
	// If no numstat found, check if output contains diff content (single file mode)
	if !hasNumstat && len(strings.TrimSpace(string(output))) > 0 {
		// Look for "diff --git" lines to extract file path and count changes
		diffFound := false
		var filePath string
		addedCount := 0
		deletedCount := 0
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			
			// Extract file path from diff header
			if strings.HasPrefix(line, "diff --git") && strings.Contains(line, "content/") {
				// Extract the "b/path" part
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasPrefix(part, "b/content/") {
						filePath = strings.TrimPrefix(part, "b/")
						diffFound = true
						break
					}
				}
			}
			
			// Count added/deleted lines
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				addedCount++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				deletedCount++
			}
		}
		
		if diffFound && filePath != "" && (addedCount > 0 || deletedCount > 0) {
			result.files = append(result.files, fileChange{
				addedLines:   addedCount,
				deletedLines: deletedCount,
				filePath:     filePath,
			})
		}
	}

	// Sort by added lines (descending), then by deleted lines (descending)
	sort.Slice(result.files, func(i, j int) bool {
		if result.files[i].addedLines == result.files[j].addedLines {
			return result.files[i].deletedLines > result.files[j].deletedLines
		}
		return result.files[i].addedLines > result.files[j].addedLines
	})

	result.hasChanges = len(result.files) > 0
	
	// Return success even if lsync.sh exits with non-zero (it's normal behavior)
	return result, nil
}

func init() {
	// Add lsync command to docs
	docsCmd.AddCommand(lsyncCmd)
	
	// Add flags for lsync
	lsyncCmd.Flags().Bool("check-pr", false, "Check for related pull requests")
}