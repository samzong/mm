package k8s

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
					timeStr := formatRelativeTime(file.LastModified)
					fmt.Printf("%-8d %-8d %-12s %-8s %s\n", 
						file.AddedLines, 
						file.DeletedLines, 
						timeStr,
						file.LastCommit,
						file.FilePath)
				}
			}
		} else {
			fmt.Printf("All files are up to date\n")
		}

		// Check PR if requested
		if checkPR && len(result.files) > 0 {
			fmt.Printf("\nChecking related PRs...\n")
			err := checkRelatedPRs(result.files)
			if err != nil {
				fmt.Printf("  Error checking PRs: %v\n", err)
			}
		}

		// Save to cache
		if err := saveCache(result); err != nil {
			fmt.Printf("Warning: Failed to save cache: %v\n", err)
		}

		return nil
	},
}

// fileChange represents a single file change with statistics
type fileChange struct {
	AddedLines   int       `json:"added_lines"`
	DeletedLines int       `json:"deleted_lines"`
	FilePath     string    `json:"file_path"`
	LastCommit   string    `json:"last_commit"`    // commit hash
	LastModified time.Time `json:"last_modified"` // last modification time
}

// lsyncResult represents the result of lsync execution
type lsyncResult struct {
	files      []fileChange
	hasChanges bool
	rawOutput  string  // Store raw output for single file diff display
	isSingleFile bool  // Track if this was a single file check
}

// lsyncCache represents cached lsync results
type lsyncCache struct {
	Timestamp time.Time    `json:"timestamp"`
	GitCommit string       `json:"git_commit"`
	Files     []fileChange `json:"files"`
	TTL       time.Duration `json:"ttl"`
}

// isK8sProject checks if current directory is a k8s project
func isK8sProject() bool {
	_, err := os.Stat("./scripts/lsync.sh")
	return err == nil
}

// getCacheFilePath returns the path to the cache file
func getCacheFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".cache", "mm")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "k8s-docs-lsync.json"), nil
}

// getCurrentGitCommit gets the current git HEAD commit hash
func getCurrentGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// isValidCache checks if the cache is still valid
func (c *lsyncCache) isValid() bool {
	if c.Timestamp.IsZero() {
		return false
	}
	
	// Check TTL (30 minutes default)
	ttl := c.TTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	if time.Since(c.Timestamp) > ttl {
		return false
	}
	
	// Check if git HEAD has changed
	currentCommit := getCurrentGitCommit()
	if currentCommit != "" && c.GitCommit != "" && c.GitCommit != currentCommit {
		return false
	}
	
	return true
}

// saveCache saves the lsync result to cache
func saveCache(result *lsyncResult) error {
	if result.isSingleFile || !result.hasChanges {
		// Don't cache single file results or empty results
		return nil
	}
	
	cacheFile, err := getCacheFilePath()
	if err != nil {
		return err
	}
	
	cache := lsyncCache{
		Timestamp: time.Now(),
		GitCommit: getCurrentGitCommit(),
		Files:     result.files,
		TTL:       30 * time.Minute,
	}
	
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(cacheFile, data, 0644)
}

// loadCache loads the cached lsync result
func loadCache() (*lsyncCache, error) {
	cacheFile, err := getCacheFilePath()
	if err != nil {
		return nil, err
	}
	
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	
	var cache lsyncCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	
	return &cache, nil
}

// clearCache removes the cached lsync result
func clearCache() error {
	cacheFile, err := getCacheFilePath()
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil // Cache doesn't exist
	}
	
	return os.Remove(cacheFile)
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
			// Check if this is numstat format: "added_lines deleted_lines filename" (tab separated)
			parts := strings.Split(line, "\t")
			if len(parts) == 3 && strings.HasPrefix(parts[2], "content/") {
				// Parse the numbers
				added, err1 := strconv.Atoi(parts[0])
				deleted, err2 := strconv.Atoi(parts[1])
				
				if err1 == nil && err2 == nil {
					// Get last modification time for the file
					lastCommit, lastModified := getLastModificationTime(parts[2])
					
					fileChange := fileChange{
						AddedLines:   added,
						DeletedLines: deleted,
						FilePath:     parts[2],
						LastCommit:   lastCommit,
						LastModified: lastModified,
					}
					result.files = append(result.files, fileChange)
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
				AddedLines:   addedCount,
				DeletedLines: deletedCount,
				FilePath:     filePath,
				LastCommit:   lastCommit,
				LastModified: lastModified,
			})
		}
	}

	// Sort by last modification time (descending - newest first)
	sort.Slice(result.files, func(i, j int) bool {
		return result.files[i].LastModified.After(result.files[j].LastModified)
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

// prInfo represents pull request information
type prInfo struct {
	number int
	title  string
	url    string
	files  []string
}

// checkRelatedPRs checks if there are existing PRs for the files
func checkRelatedPRs(files []fileChange) error {
	const pageSize = 5
	
	// Process files in batches of 5
	for offset := 0; offset < len(files); offset += pageSize {
		end := offset + pageSize
		if end > len(files) {
			end = len(files)
		}
		
		batch := files[offset:end]
		fmt.Printf("\nChecking batch %d-%d of %d files:\n", offset+1, end, len(files))
		
		availableFiles, err := checkBatchPRs(batch)
		if err != nil {
			return err
		}
		
		// If we found files to work on, show them and stop
		if len(availableFiles) > 0 {
			fmt.Printf("\nFound %d files available for contribution in this batch\n", len(availableFiles))
			break
		} else {
			fmt.Printf("\nAll files in this batch already have PRs, checking next batch...\n")
		}
	}
	
	return nil
}

// checkBatchPRs checks a batch of files for existing PRs
func checkBatchPRs(batch []fileChange) ([]fileChange, error) {
	var availableFiles []fileChange
	
	// Print table header
	fmt.Printf("%-80s %-15s %s\n", "File", "Status", "PR Link")
	fmt.Printf("%-80s %-15s %s\n", strings.Repeat("-", 80), strings.Repeat("-", 15), strings.Repeat("-", 50))
	
	for _, file := range batch {
		// Convert English path to Chinese path for PR search
		zhPath := file.FilePath
		if strings.HasPrefix(file.FilePath, "content/en/") {
			zhPath = strings.Replace(file.FilePath, "content/en/", "content/zh-cn/", 1)
		}
		
		// Search for PRs containing this Chinese file
		prs, err := searchPRsForFile(zhPath)
		if err != nil {
			fmt.Printf("%-80s %-15s %s\n", zhPath, "Error", fmt.Sprintf("Error: %v", err))
			continue
		}
		
		if len(prs) == 0 {
			// No PRs found, this file is available
			availableFiles = append(availableFiles, file)
			fmt.Printf("%-80s %-15s %s\n", zhPath, "Available", "-")
		} else {
			// Found existing PRs - show the first one
			pr := prs[0]
			fmt.Printf("%-80s %-15s %s\n", zhPath, "In Progress", pr.url)
		}
	}
	
	return availableFiles, nil
}

// searchPRsForFile searches for PRs that contain the specified Chinese file
func searchPRsForFile(zhPath string) ([]prInfo, error) {
	// Search for open PRs that contain this Chinese file
	query := fmt.Sprintf("repo:kubernetes/website type:pr state:open %s in:files", zhPath)
	
	return searchPRs(query)
}

// searchPRs executes a GitHub search query for PRs
func searchPRs(query string) ([]prInfo, error) {
	cmd := exec.Command("gh", "api", "-X", "GET", "search/issues", "-f", fmt.Sprintf("q=%s", query), "--jq", ".items[] | {number: .number, title: .title, html_url: .html_url}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api failed: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var prs []prInfo
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Parse JSON line
		var prData struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			HTMLURL string `json:"html_url"`
		}
		
		if err := json.Unmarshal([]byte(line), &prData); err != nil {
			continue // Skip invalid lines
		}
		
		prs = append(prs, prInfo{
			number: prData.Number,
			title:  prData.Title,
			url:    prData.HTMLURL,
		})
	}
	
	return prs, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// workflowCmd represents the workflow command
var workflowCmd = &cobra.Command{
	Use:   "workflow [file-path]",
	Short: "Generate git workflow commands for documentation translation",
	Long: `Generate standardized git commands for Kubernetes documentation translation workflow.

This command can work in two modes:
1. Interactive mode (no arguments): Select from cached lsync results
2. Direct mode (with file path): Generate commands for specific file

Examples:
  mm k8s docs workflow                                       # Interactive selection from cache
  mm k8s docs workflow docs/concepts/overview/what-is-kubernetes.md  # Direct file specification
  mm k8s docs workflow --available-only                     # Show only files without existing PRs

Branch format: docs/sync/zh/{filename}
Commit format: [zh-cn] sync {filepath}
PR format: Same as commit message with full content path`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fresh, _ := cmd.Flags().GetBool("fresh")
		availableOnly, _ := cmd.Flags().GetBool("available-only")
		
		if len(args) > 0 {
			// Direct mode: generate commands for specific file
			return generateWorkflowCommands(args[0])
		}
		
		// Interactive mode: use cached results
		cache, err := loadCache()
		if err != nil || !cache.isValid() || fresh {
			if fresh {
				fmt.Printf("Refreshing cache...\n")
			} else if err != nil {
				fmt.Printf("No cache found.\n")
			} else {
				fmt.Printf("Cache expired (last updated: %s)\n", cache.Timestamp.Format("15:04"))
			}
			fmt.Printf("Please run: mm k8s docs lsync\n")
			return nil
		}
		
		// Filter files if --available-only is specified
		if availableOnly {
			return showAvailableFiles(cache)
		}
		
		// Show cached results and let user select
		return showInteractiveSelection(cache)
	},
}

// generateWorkflowCommands generates git workflow commands for a specific file
func generateWorkflowCommands(filePath string) error {
	// Remove leading/trailing spaces and normalize path
	filePath = strings.TrimSpace(filePath)
	
	// Extract filename without extension for branch name
	filename := filepath.Base(filePath)
	if strings.HasSuffix(filename, ".md") {
		filename = filename[:len(filename)-3]
	}
	
	// Generate components
	branchName := fmt.Sprintf("docs/sync/zh/%s", filename)
	commitMessage := fmt.Sprintf("[zh-cn] sync %s", filePath)
	fullPath := fmt.Sprintf("content/zh-cn/%s", filePath)
	
	// Display the commands
	fmt.Printf("Git workflow commands for: %s\n\n", filePath)
	fmt.Printf("# 1. Create and switch to new branch\n")
	fmt.Printf("git switch -c %s\n\n", branchName)
	fmt.Printf("# 2. After translation, add file to staging area\n")
	fmt.Printf("git add %s\n\n", fullPath)
	fmt.Printf("# 3. Commit changes with signed-off-by\n")
	fmt.Printf("git commit -s -m \"%s\"\n\n", commitMessage)
	fmt.Printf("# 4. Push branch to remote\n")
	fmt.Printf("git push origin %s\n\n", branchName)
	fmt.Printf("# 5. Create pull request\n")
	fmt.Printf("gh pr create --title \"%s\" --body \"%s\"\n", commitMessage, fullPath)
	
	return nil
}

// showInteractiveSelection shows cached files and lets user select one
func showInteractiveSelection(cache *lsyncCache) error {
	if len(cache.Files) == 0 {
		fmt.Printf("No files need translation (cache from %s)\n", cache.Timestamp.Format("15:04"))
		return nil
	}
	
	fmt.Printf("Found %d files needing translation (cached at %s):\n\n", 
		len(cache.Files), cache.Timestamp.Format("15:04"))
	
	// Display files with numbers
	for i, file := range cache.Files {
		// Convert English path to display path
		displayPath := file.FilePath
		if strings.HasPrefix(file.FilePath, "content/en/") {
			displayPath = strings.TrimPrefix(file.FilePath, "content/en/")
		}
		
		timeStr := formatRelativeTime(file.LastModified)
		fmt.Printf("[%2d] %-60s (modified %s)\n", i+1, displayPath, timeStr)
	}
	
	fmt.Printf("\nSelect a file number (1-%d), or press Enter to exit: ", len(cache.Files))
	
	var input string
	fmt.Scanln(&input)
	
	if input == "" {
		return nil
	}
	
	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(cache.Files) {
		return fmt.Errorf("invalid selection: %s", input)
	}
	
	// Get selected file
	selectedFile := cache.Files[selection-1]
	
	// Convert English path to docs path for command generation
	filePath := selectedFile.FilePath
	if strings.HasPrefix(filePath, "content/en/") {
		filePath = strings.TrimPrefix(filePath, "content/en/")
	}
	
	fmt.Printf("\n")
	return generateWorkflowCommands(filePath)
}

// showAvailableFiles shows only files that don't have existing PRs
func showAvailableFiles(cache *lsyncCache) error {
	if len(cache.Files) == 0 {
		fmt.Printf("No files need translation (cache from %s)\n", cache.Timestamp.Format("15:04"))
		return nil
	}
	
	fmt.Printf("Checking for existing PRs... (this may take a moment)\n\n")
	
	var availableFiles []fileChange
	
	// Check each file for existing PRs
	for _, file := range cache.Files {
		// Convert English path to Chinese path for PR search
		zhPath := file.FilePath
		if strings.HasPrefix(file.FilePath, "content/en/") {
			zhPath = strings.Replace(file.FilePath, "content/en/", "content/zh-cn/", 1)
		}
		
		// Search for PRs containing this Chinese file
		prs, err := searchPRsForFile(zhPath)
		if err != nil {
			fmt.Printf("Error checking PRs for %s: %v\n", zhPath, err)
			continue
		}
		
		if len(prs) == 0 {
			// No PRs found, this file is available
			availableFiles = append(availableFiles, file)
		}
	}
	
	if len(availableFiles) == 0 {
		fmt.Printf("All files already have existing PRs. No files available for translation.\n")
		return nil
	}
	
	fmt.Printf("Found %d files available for translation (cached at %s):\n\n", 
		len(availableFiles), cache.Timestamp.Format("15:04"))
	
	// Display available files with numbers
	for i, file := range availableFiles {
		// Convert English path to display path
		displayPath := file.FilePath
		if strings.HasPrefix(file.FilePath, "content/en/") {
			displayPath = strings.TrimPrefix(file.FilePath, "content/en/")
		}
		
		timeStr := formatRelativeTime(file.LastModified)
		fmt.Printf("[%2d] %-60s (modified %s)\n", i+1, displayPath, timeStr)
	}
	
	fmt.Printf("\nSelect a file number (1-%d), or press Enter to exit: ", len(availableFiles))
	
	var input string
	fmt.Scanln(&input)
	
	if input == "" {
		return nil
	}
	
	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(availableFiles) {
		return fmt.Errorf("invalid selection: %s", input)
	}
	
	// Get selected file
	selectedFile := availableFiles[selection-1]
	
	// Convert English path to docs path for command generation
	filePath := selectedFile.FilePath
	if strings.HasPrefix(filePath, "content/en/") {
		filePath = strings.TrimPrefix(filePath, "content/en/")
	}
	
	fmt.Printf("\n")
	return generateWorkflowCommands(filePath)
}

// clearCacheCmd represents the clear-cache command
var clearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Clear the cached lsync results",
	Long:  `Remove the cached lsync results to force fresh scanning on next workflow command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := clearCache(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		fmt.Printf("Cache cleared successfully\n")
		return nil
	},
}

func init() {
	// Add lsync command to docs
	docsCmd.AddCommand(lsyncCmd)
	docsCmd.AddCommand(workflowCmd)
	docsCmd.AddCommand(clearCacheCmd)
	
	// Add flags for lsync
	lsyncCmd.Flags().Bool("check-pr", false, "Check for related pull requests")
	
	// Add flags for workflow
	workflowCmd.Flags().Bool("fresh", false, "Force refresh cache before showing selection")
	workflowCmd.Flags().Bool("available-only", false, "Show only files without existing PRs")
}