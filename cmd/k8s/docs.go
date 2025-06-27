package k8s

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

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
				// For multiple files, show summary table with modification time
				fmt.Printf("%-8s %-8s %-12s %-8s %s\n", "Added", "Deleted", "LastModified", "Commit", "File")
				fmt.Printf("%-8s %-8s %-12s %-8s %s\n", "-----", "-------", "------------", "------", "----")
				for _, file := range result.files {
					// Format time as relative (e.g., "2 days ago") 
					timeStr := formatRelativeTime(file.lastModified)
					fmt.Printf("%-8d %-8d %-12s %-8s %s\n", 
						file.addedLines, 
						file.deletedLines, 
						timeStr,
						file.lastCommit,
						file.filePath)
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
	lastCommit   string    // commit hash
	lastModified time.Time // last modification time
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
					// Get last modification time for the file
					lastCommit, lastModified := getLastModificationTime(parts[2])
					
					result.files = append(result.files, fileChange{
						addedLines:   added,
						deletedLines: deleted,
						filePath:     parts[2],
						lastCommit:   lastCommit,
						lastModified: lastModified,
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
			// Get last modification time for the file
			lastCommit, lastModified := getLastModificationTime(filePath)
			
			result.files = append(result.files, fileChange{
				addedLines:   addedCount,
				deletedLines: deletedCount,
				filePath:     filePath,
				lastCommit:   lastCommit,
				lastModified: lastModified,
			})
		}
	}

	// Sort by last modification time (descending - newest first)
	sort.Slice(result.files, func(i, j int) bool {
		return result.files[i].lastModified.After(result.files[j].lastModified)
	})

	result.hasChanges = len(result.files) > 0
	
	// Return success even if lsync.sh exits with non-zero (it's normal behavior)
	return result, nil
}

// getLastModificationTime gets the last commit time for a file
func getLastModificationTime(filePath string) (string, time.Time) {
	// Get commit hash and timestamp
	cmd := exec.Command("git", "log", "-n", "1", "--pretty=format:%h %ct", "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", time.Time{}
	}
	
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) < 2 {
		return "", time.Time{}
	}
	
	commitHash := parts[0]
	timestampStr := parts[1]
	
	// Parse Unix timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return commitHash, time.Time{}
	}
	
	return commitHash, time.Unix(timestamp, 0)
}

// formatRelativeTime formats time as relative string (e.g., "2 days ago")
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	
	now := time.Now()
	diff := now.Sub(t)
	
	days := int(diff.Hours() / 24)
	hours := int(diff.Hours())
	minutes := int(diff.Minutes())
	
	if days > 365 {
		years := days / 365
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	} else if days > 30 {
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else if days > 0 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if hours > 0 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", minutes)
	} else {
		return "just now"
	}
}

func init() {
	// Add lsync command to docs
	docsCmd.AddCommand(lsyncCmd)
	
	// Add flags for lsync
	lsyncCmd.Flags().Bool("check-pr", false, "Check for related pull requests")
}