package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samzong/mm/internal/quality/checker"
	"github.com/samzong/mm/internal/quality/detector"
	"github.com/spf13/cobra"
)

// spellCmd represents the spell command
var spellCmd = &cobra.Command{
	Use:   "spell [files/directories...]",
	Short: "Check spelling in documentation files",
	Long: `Check spelling in documentation files with automatic project detection.
Supports multiple file formats (markdown, text, etc.) and loads project-specific
dictionaries automatically.

Examples:
  mm quality spell README.md                    # Check single file
  mm quality spell docs/                        # Check directory recursively  
  mm quality spell content/en/docs/concepts/    # Check K8s docs (auto-detects project)
  mm quality spell --project=k8s docs/          # Explicitly use K8s dictionary
  mm quality spell --format=json docs/ > report.json  # Output JSON format`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		projectType, _ := cmd.Flags().GetString("project")
		outputFormat, _ := cmd.Flags().GetString("format")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		// Initialize spell checker
		spellChecker, err := checker.NewSpellChecker()
		if err != nil {
			return fmt.Errorf("failed to initialize spell checker: %w", err)
		}
		
		// Auto-detect project if not specified
		if projectType == "" {
			detectedProject, err := detector.DetectProject(".")
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Could not detect project type: %v\n", err)
				}
				projectType = "generic"
			} else {
				projectType = detectedProject
				if verbose {
					fmt.Printf("Detected project type: %s\n", projectType)
				}
			}
		}
		
		// Set project type for spell checker
		if err := spellChecker.SetProject(projectType); err != nil {
			return fmt.Errorf("failed to set project type: %w", err)
		}
		
		// Collect files to check
		var filesToCheck []string
		for _, arg := range args {
			files, err := collectFiles(arg)
			if err != nil {
				return fmt.Errorf("failed to collect files from %s: %w", arg, err)
			}
			filesToCheck = append(filesToCheck, files...)
		}
		
		if len(filesToCheck) == 0 {
			return fmt.Errorf("no files found to check")
		}
		
		if verbose {
			fmt.Printf("Checking %d files with %s dictionary\n", len(filesToCheck), projectType)
		}
		
		// Run spell check
		result, err := spellChecker.CheckFiles(filesToCheck)
		if err != nil {
			return fmt.Errorf("spell check failed: %w", err)
		}
		
		// Output results
		switch outputFormat {
		case "json":
			return result.OutputJSON(os.Stdout)
		case "console":
			fallthrough
		default:
			return result.OutputConsole(os.Stdout, verbose)
		}
	},
}

// collectFiles recursively collects files to check based on supported extensions
func collectFiles(path string) ([]string, error) {
	var files []string
	
	// Supported file extensions
	supportedExts := map[string]bool{
		".md":   true,
		".txt":  true,
		".rst":  true,
		".html": true,
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	
	if info.IsDir() {
		// Walk directory
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip hidden files and directories
			if strings.HasPrefix(info.Name(), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			
			// Check if file has supported extension
			ext := strings.ToLower(filepath.Ext(filePath))
			if supportedExts[ext] {
				files = append(files, filePath)
			}
			
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Single file
		ext := strings.ToLower(filepath.Ext(path))
		if supportedExts[ext] {
			files = append(files, path)
		} else {
			return nil, fmt.Errorf("unsupported file type: %s", ext)
		}
	}
	
	return files, nil
}

func init() {
	// Add flags for spell command
	spellCmd.Flags().StringP("project", "p", "", "Project type (k8s, go, docker, generic)")
	spellCmd.Flags().StringP("format", "f", "console", "Output format (console, json)")
	spellCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}