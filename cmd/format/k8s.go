package format

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// K8sCmd represents the k8s format command
var K8sCmd = &cobra.Command{
	Use:   "k8s [file/directory]",
	Short: "Format Kubernetes Chinese documentation according to style guide",
	Long: `Format Kubernetes Chinese documentation files according to the official style guide.
This command applies automated formatting rules including:
- Chinese-English spacing
- Punctuation standardization  
- Heading anchor generation
- Link localization

By default, shows preview of changes. Use --apply to actually modify files.

Examples:
  mm format k8s content/zh-cn/docs/concepts/overview.md
  mm format k8s content/zh-cn/docs/concepts/ --recursive
  mm format k8s content/zh-cn/docs/concepts/overview.md --apply
  mm format k8s content/zh-cn/docs/ --rules=spacing,punctuation --apply`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if we're in a k8s project directory
		if !isK8sProject() {
			return fmt.Errorf("not in a Kubernetes project directory. Please run this command in the kubernetes/website project root")
		}

		apply, _ := cmd.Flags().GetBool("apply")
		recursive, _ := cmd.Flags().GetBool("recursive")
		backup, _ := cmd.Flags().GetBool("backup")
		rules, _ := cmd.Flags().GetStringSlice("rules")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Default to current directory if no path provided
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		// Process files
		return processFiles(targetPath, &formatOptions{
			apply:     apply,
			recursive: recursive,
			backup:    backup,
			rules:     rules,
			verbose:   verbose,
		})
	},
}

// formatOptions holds configuration for formatting
type formatOptions struct {
	apply     bool
	recursive bool
	backup    bool
	rules     []string
	verbose   bool
}

// formatResult holds the result of formatting a file
type formatResult struct {
	filePath    string
	changes     []changeInfo
	hasChanges  bool
	errors      []error
}

// changeInfo describes a specific change made to a file
type changeInfo struct {
	line        int
	rule        string
	description string
	before      string
	after       string
}

// isK8sProject checks if current directory is a k8s project
func isK8sProject() bool {
	_, err := os.Stat("./scripts/lsync.sh")
	return err == nil
}

// processFiles processes files or directories according to options
func processFiles(targetPath string, options *formatOptions) error {
	// Check if target exists
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("target path not found: %s", targetPath)
	}

	var files []string

	if info.IsDir() {
		// Collect markdown files from directory
		if options.recursive {
			err = filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.HasSuffix(path, ".md") {
					files = append(files, path)
				}
				return nil
			})
		} else {
			entries, err := os.ReadDir(targetPath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					files = append(files, filepath.Join(targetPath, entry.Name()))
				}
			}
		}
	} else {
		// Single file
		if !strings.HasSuffix(targetPath, ".md") {
			return fmt.Errorf("only markdown files (.md) are supported")
		}
		files = append(files, targetPath)
	}

	if len(files) == 0 {
		fmt.Printf("No markdown files found in: %s\n", targetPath)
		return nil
	}

	// Process each file
	var results []formatResult
	for _, file := range files {
		result, err := processFile(file, options)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
			continue
		}
		results = append(results, result)
	}

	// Display results
	return displayResults(results, options)
}

// processFile processes a single markdown file
func processFile(filePath string, options *formatOptions) (formatResult, error) {
	result := formatResult{
		filePath: filePath,
		changes:  []changeInfo{},
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return result, err
	}

	originalContent := string(content)
	modifiedContent := originalContent

	// Apply formatting rules
	modifiedContent, changes := applyFormattingRules(modifiedContent, options.rules)
	result.changes = changes
	result.hasChanges = len(changes) > 0

	// If applying changes, write back to file
	if options.apply && result.hasChanges {
		// Create backup if requested
		if options.backup {
			backupPath := filePath + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				result.errors = append(result.errors, fmt.Errorf("failed to create backup: %w", err))
				return result, nil
			}
		}

		// Write modified content
		if err := os.WriteFile(filePath, []byte(modifiedContent), 0644); err != nil {
			result.errors = append(result.errors, fmt.Errorf("failed to write file: %w", err))
		}
	}

	return result, nil
}

// applyFormattingRules applies formatting rules to content
func applyFormattingRules(content string, rules []string) (string, []changeInfo) {
	var changes []changeInfo
	modified := content

	// Default rules if none specified
	if len(rules) == 0 {
		rules = []string{"spacing", "punctuation", "linebreaks"}
	}

	// First, identify and protect special regions (code blocks, HTML comments, shortcodes)
	protectedRegions := identifyProtectedRegions(content)

	for _, rule := range rules {
		var ruleChanges []changeInfo
		switch rule {
		case "spacing":
			modified, ruleChanges = applySpacingRuleWithProtection(modified, protectedRegions)
			changes = append(changes, ruleChanges...)
		case "punctuation":
			modified, ruleChanges = applyPunctuationRuleWithProtection(modified, protectedRegions)
			changes = append(changes, ruleChanges...)
		case "linebreaks":
			modified, ruleChanges = applyLineBreakRuleWithProtection(modified, protectedRegions)
			changes = append(changes, ruleChanges...)
		}
	}

	return modified, changes
}

// protectedRegion represents a region that should not be modified
type protectedRegion struct {
	start int
	end   int
	regionType string // "code_block", "html_comment", "inline_code", "hugo_shortcode"
}

// identifyProtectedRegions finds regions that should not be modified
func identifyProtectedRegions(content string) []protectedRegion {
	var regions []protectedRegion
	lines := strings.Split(content, "\n")
	
	var currentPos int
	var inCodeBlock bool
	var inHtmlComment bool
	var inHugoShortcode bool
	var codeBlockStart, commentStart, hugoStart int
	
	for _, line := range lines {
		lineStart := currentPos
		
		// Check for code block boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inCodeBlock {
				// Starting a code block
				inCodeBlock = true
				codeBlockStart = lineStart
			} else {
				// Ending a code block
				inCodeBlock = false
				regions = append(regions, protectedRegion{
					start: codeBlockStart,
					end: currentPos + len(line),
					regionType: "code_block",
				})
			}
		}
		
		// If we're in a code block, skip other processing
		if inCodeBlock {
			currentPos += len(line) + 1
			continue
		}
		
		// Check for Hugo shortcode boundaries (multi-line)
		if (strings.Contains(line, "{{</*") || strings.Contains(line, "{{%/*")) && !inHugoShortcode {
			hugoStart = lineStart
			inHugoShortcode = true
		}
		if (strings.Contains(line, "*/}}") || strings.Contains(line, "*/%}}")) && inHugoShortcode {
			regions = append(regions, protectedRegion{
				start: hugoStart,
				end: currentPos + len(line),
				regionType: "hugo_shortcode",
			})
			inHugoShortcode = false
		}
		
		// Check for HTML comment boundaries
		if strings.Contains(line, "<!--") && !inHtmlComment {
			commentStart = lineStart + strings.Index(line, "<!--")
			inHtmlComment = true
		}
		if strings.Contains(line, "-->") && inHtmlComment {
			commentEnd := lineStart + strings.Index(line, "-->") + 3
			regions = append(regions, protectedRegion{
				start: commentStart,
				end: commentEnd,
				regionType: "html_comment",
			})
			inHtmlComment = false
		}
		
		// If we're in a protected multi-line region, protect the entire line
		if inHtmlComment || inHugoShortcode {
			regions = append(regions, protectedRegion{
				start: lineStart,
				end: currentPos + len(line),
				regionType: func() string {
					if inHtmlComment { return "html_comment" }
					return "hugo_shortcode"
				}(),
			})
		} else {
			// Check for inline code (backticks) - only if not in protected regions
			findInlineCodeRegions(line, lineStart, &regions)
			
			// Check for single-line Hugo shortcodes - only if not in protected regions
			findHugoShortcodeRegions(line, lineStart, &regions)
		}
		
		currentPos += len(line) + 1 // +1 for newline
	}
	
	// Sort regions by start position
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].start < regions[j].start
	})
	
	return regions
}

// findInlineCodeRegions finds inline code spans marked with backticks
func findInlineCodeRegions(line string, lineStart int, regions *[]protectedRegion) {
	var inCode bool
	var codeStart int
	
	for i, char := range line {
		if char == '`' {
			if !inCode {
				inCode = true
				codeStart = lineStart + i
			} else {
				inCode = false
				*regions = append(*regions, protectedRegion{
					start: codeStart,
					end: lineStart + i + 1,
					regionType: "inline_code",
				})
			}
		}
	}
}

// findHugoShortcodeRegions finds Hugo shortcodes like {{< >}} and {{% %}}
func findHugoShortcodeRegions(line string, lineStart int, regions *[]protectedRegion) {
	// Find {{< ... >}} patterns
	re1 := regexp.MustCompile(`\{\{<[^>]*>\}\}`)
	matches := re1.FindAllStringIndex(line, -1)
	for _, match := range matches {
		*regions = append(*regions, protectedRegion{
			start: lineStart + match[0],
			end: lineStart + match[1],
			regionType: "hugo_shortcode",
		})
	}
	
	// Find {{% ... %}} patterns
	re2 := regexp.MustCompile(`\{\{%[^%]*%\}\}`)
	matches = re2.FindAllStringIndex(line, -1)
	for _, match := range matches {
		*regions = append(*regions, protectedRegion{
			start: lineStart + match[0],
			end: lineStart + match[1],
			regionType: "hugo_shortcode",
		})
	}
	
	// Find {{</ ... />}} patterns (closing tags)
	re3 := regexp.MustCompile(`\{\{</[^>]*>\}\}`)
	matches = re3.FindAllStringIndex(line, -1)
	for _, match := range matches {
		*regions = append(*regions, protectedRegion{
			start: lineStart + match[0],
			end: lineStart + match[1],
			regionType: "hugo_shortcode",
		})
	}
	
	// Find {{%/ ... /%}} patterns (closing tags)
	re4 := regexp.MustCompile(`\{\{%/[^%]*%\}\}`)
	matches = re4.FindAllStringIndex(line, -1)
	for _, match := range matches {
		*regions = append(*regions, protectedRegion{
			start: lineStart + match[0],
			end: lineStart + match[1],
			regionType: "hugo_shortcode",
		})
	}
}

// isPositionProtected checks if a position is within a protected region
func isPositionProtected(pos int, regions []protectedRegion) bool {
	for _, region := range regions {
		if pos >= region.start && pos < region.end {
			return true
		}
	}
	return false
}

// applySpacingRuleWithProtection adds spaces between Chinese and English text while respecting protected regions
func applySpacingRuleWithProtection(content string, protectedRegions []protectedRegion) (string, []changeInfo) {
	var changes []changeInfo
	
	// For now, use line-based processing but check if each line is in protected regions
	lines := strings.Split(content, "\n")
	var currentPos int

	for lineNum, line := range lines {
		originalLine := line
		lineStart := currentPos
		lineEnd := currentPos + len(line)
		
		// Check if this entire line is within a protected region
		lineProtected := false
		for _, region := range protectedRegions {
			if lineStart >= region.start && lineEnd <= region.end {
				lineProtected = true
				break
			}
		}
		
		if !lineProtected {
			// Apply spacing patterns only if line is not protected
			patterns := []struct {
				pattern *regexp.Regexp
				replace string
			}{
				{
					pattern: regexp.MustCompile(`([一-龯])([a-zA-Z0-9])`),
					replace: "$1 $2",
				},
				{
					pattern: regexp.MustCompile(`([a-zA-Z0-9])([一-龯])`),
					replace: "$1 $2",
				},
			}

			for _, pattern := range patterns {
				if pattern.pattern.MatchString(line) {
					line = pattern.pattern.ReplaceAllString(line, pattern.replace)
				}
			}

			if line != originalLine {
				lines[lineNum] = line
				changes = append(changes, changeInfo{
					line:        lineNum + 1,
					rule:        "spacing",
					description: "Added space between Chinese and English text",
					before:      originalLine,
					after:       line,
				})
			}
		}
		
		currentPos += len(originalLine) + 1 // +1 for newline
	}

	return strings.Join(lines, "\n"), changes
}

// applySpacingRule adds spaces between Chinese and English text (legacy version)
func applySpacingRule(content string) (string, []changeInfo) {
	return applySpacingRuleWithProtection(content, []protectedRegion{})
}

// applyPunctuationRuleWithProtection converts half-width to full-width punctuation while respecting protected regions
func applyPunctuationRuleWithProtection(content string, protectedRegions []protectedRegion) (string, []changeInfo) {
	var changes []changeInfo
	
	// Punctuation conversion map
	punctuationMap := map[string]string{
		",": "，",
		";": "；", 
		":": "：",
		"!": "！",
		"?": "？",
	}

	lines := strings.Split(content, "\n")
	var currentPos int

	for lineNum, line := range lines {
		originalLine := line
		lineStart := currentPos
		lineEnd := currentPos + len(line)
		
		// Check if this entire line is within a protected region
		lineProtected := false
		for _, region := range protectedRegions {
			if lineStart >= region.start && lineEnd <= region.end {
				lineProtected = true
				break
			}
		}
		
		if !lineProtected {
			// Skip YAML frontmatter lines
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "---") || 
			   (strings.Contains(trimmed, ": ") && !regexp.MustCompile(`[一-龯]`).MatchString(trimmed)) {
				currentPos += len(originalLine) + 1
				continue
			}
			
			// Only convert punctuation if line contains Chinese characters
			if regexp.MustCompile(`[一-龯]`).MatchString(line) {
				// For lines with Chinese, convert punctuation more carefully
				for halfWidth, fullWidth := range punctuationMap {
					// Skip colon conversion if it looks like it's part of a URL, time, or YAML
					if halfWidth == ":" {
						if strings.Contains(line, "://") || 
						   regexp.MustCompile(`\d+:\d+`).MatchString(line) ||
						   regexp.MustCompile(`^\s*\w+:\s`).MatchString(line) {
							continue
						}
					}
					
					// Skip exclamation mark conversion if it's part of markdown syntax
					if halfWidth == "!" {
						if strings.Contains(line, "![") || 
						   strings.Contains(line, "<!--") ||
						   strings.Contains(line, "`!") ||
						   strings.Contains(line, "!`") ||
						   strings.Contains(line, "（`！`）") ||
						   strings.Contains(line, "(`!`)") {
							continue
						}
					}
					
					if strings.Contains(line, halfWidth) {
						line = strings.ReplaceAll(line, halfWidth, fullWidth)
					}
				}
			}

			if line != originalLine {
				lines[lineNum] = line
				changes = append(changes, changeInfo{
					line:        lineNum + 1,
					rule:        "punctuation", 
					description: "Converted half-width to full-width punctuation",
					before:      originalLine,
					after:       line,
				})
			}
		}
		
		currentPos += len(originalLine) + 1 // +1 for newline
	}

	return strings.Join(lines, "\n"), changes
}

// applyPunctuationRule converts half-width to full-width punctuation in Chinese contexts (legacy version)
func applyPunctuationRule(content string) (string, []changeInfo) {
	return applyPunctuationRuleWithProtection(content, []protectedRegion{})
}

// applyLineBreakRuleWithProtection enforces 80-120 character line length while respecting protected regions
func applyLineBreakRuleWithProtection(content string, protectedRegions []protectedRegion) (string, []changeInfo) {
	var changes []changeInfo
	const maxLineLength = 120
	const preferredLineLength = 80
	
	lines := strings.Split(content, "\n")
	
	// Process from the end to avoid index shifting issues
	for lineNum := len(lines) - 1; lineNum >= 0; lineNum-- {
		line := lines[lineNum]
		originalLine := line
		
		// Calculate line position in the content
		lineStart := 0
		for i := 0; i < lineNum; i++ {
			lineStart += len(lines[i]) + 1 // +1 for newline
		}
		lineEnd := lineStart + len(line)
		
		// Check if this entire line is within a protected region
		lineProtected := false
		for _, region := range protectedRegions {
			if lineStart >= region.start && lineEnd <= region.end {
				lineProtected = true
				break
			}
		}
		
		if lineProtected {
			continue
		}
		
		// Skip certain line types that shouldn't be broken
		if shouldSkipLineBreaking(line) {
			continue
		}
		
		// Only process lines that exceed preferred length  
		lineLength := len([]rune(line))
		if lineLength <= preferredLineLength {
			continue
		}
		
		// Try to break the line intelligently
		if brokenLines := smartLineBreak(line, maxLineLength, preferredLineLength); len(brokenLines) > 1 {
			// Replace the original line with the first broken line
			lines[lineNum] = brokenLines[0]
			
			// Insert additional lines after the current position
			for i := len(brokenLines) - 1; i >= 1; i-- {
				lines = append(lines[:lineNum+1], append([]string{brokenLines[i]}, lines[lineNum+1:]...)...)
			}
			
			changes = append(changes, changeInfo{
				line:        lineNum + 1,
				rule:        "linebreaks",
				description: fmt.Sprintf("Broke long line (%d chars) into %d lines", len([]rune(originalLine)), len(brokenLines)),
				before:      originalLine,
				after:       strings.Join(brokenLines, "\n"),
			})
		}
	}
	
	return strings.Join(lines, "\n"), changes
}

// applyLineBreakRule enforces 80-120 character line length with smart breaking (legacy version)
func applyLineBreakRule(content string) (string, []changeInfo) {
	return applyLineBreakRuleWithProtection(content, []protectedRegion{})
}

// shouldSkipLineBreaking determines if a line should be skipped for line breaking
func shouldSkipLineBreaking(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Skip empty lines
	if trimmed == "" {
		return true
	}
	
	// Skip code blocks
	if strings.HasPrefix(trimmed, "```") {
		return true
	}
	
	// Skip inline code lines (lines that are mostly code)
	if strings.Count(line, "`") >= 2 {
		return true
	}
	
	// Skip lines with URLs (to preserve link integrity)
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
		return true
	}
	
	// Skip lines with markdown links that would be broken
	if strings.Contains(line, "](") && (strings.Count(line, "[") == strings.Count(line, "]")) {
		return true
	}
	
	// Skip frontmatter and yaml-like content
	if strings.HasPrefix(trimmed, "---") || strings.Contains(trimmed, ": ") && !strings.Contains(trimmed, "。") && !strings.Contains(trimmed, "，") {
		return true
	}
	
	// Skip table rows
	if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
		return true
	}
	
	// Skip headings
	if strings.HasPrefix(trimmed, "#") {
		return true
	}
	
	return false
}

// smartLineBreak breaks a line intelligently while preserving readability  
func smartLineBreak(line string, maxLength, preferredLength int) []string {
	runes := []rune(line)
	
	// If line is not too long, don't break it
	if len(runes) <= maxLength {
		return []string{line}
	}
	
	var result []string
	remaining := line
	remainingRunes := runes
	
	for len(remainingRunes) > preferredLength {
		breakPoint := findBestBreakPoint(remaining, preferredLength, maxLength)
		if breakPoint == -1 {
			// Can't find a good break point, keep the line as is
			result = append(result, remaining)
			break
		}
		
		// Convert rune position back to byte position for string slicing
		runesSegment := remainingRunes[:breakPoint]
		segment := string(runesSegment)
		segment = strings.TrimSpace(segment)
		
		// Update remaining content
		remainingRunes = remainingRunes[breakPoint:]
		remaining = string(remainingRunes)
		remaining = strings.TrimSpace(remaining)
		
		// Handle indentation for continuation lines
		if len(result) > 0 {
			// Check if original line has list indentation
			indent := getIndentation(line)
			if strings.Contains(line, "- ") || strings.Contains(line, "* ") || regexp.MustCompile(`^\s*\d+\.\s`).MatchString(line) {
				// For list items, add 2 extra spaces for continuation
				segment = indent + "  " + strings.TrimSpace(segment)
			} else if indent != "" {
				// Preserve original indentation
				segment = indent + strings.TrimSpace(segment)
			}
		}
		
		result = append(result, segment)
	}
	
	// Add the remaining part
	if remaining != "" {
		if len(result) > 0 {
			indent := getIndentation(line)
			if strings.Contains(line, "- ") || strings.Contains(line, "* ") || regexp.MustCompile(`^\s*\d+\.\s`).MatchString(line) {
				remaining = indent + "  " + strings.TrimSpace(remaining)
			} else if indent != "" {
				remaining = indent + strings.TrimSpace(remaining)
			}
		}
		result = append(result, remaining)
	}
	
	return result
}

// findBestBreakPoint finds the best position to break a line
func findBestBreakPoint(line string, preferredLength, maxLength int) int {
	runes := []rune(line)
	lineLength := len(runes)
	
	// Ensure we don't go out of bounds
	searchEnd := preferredLength
	if searchEnd >= lineLength {
		searchEnd = lineLength - 1
	}
	
	searchStart := preferredLength / 2
	if searchStart >= lineLength {
		searchStart = lineLength - 1
	}
	
	// Prefer breaking at sentence boundaries (。！？)
	for i := searchEnd; i >= searchStart && i < lineLength; i-- {
		char := string(runes[i])
		if char == "。" || char == "！" || char == "？" {
			return i + 1
		}
	}
	
	// Break at Chinese punctuation (，；：)
	for i := searchEnd; i >= searchStart && i < lineLength; i-- {
		char := string(runes[i])
		if char == "，" || char == "；" || char == "：" {
			return i + 1
		}
	}
	
	// Break at spaces (English words) - use byte index for ASCII characters
	lineBytes := []byte(line)
	searchEndBytes := preferredLength
	if searchEndBytes >= len(lineBytes) {
		searchEndBytes = len(lineBytes) - 1
	}
	searchStartBytes := preferredLength / 2
	if searchStartBytes >= len(lineBytes) {
		searchStartBytes = len(lineBytes) - 1
	}
	
	for i := searchEndBytes; i >= searchStartBytes && i < len(lineBytes); i-- {
		if lineBytes[i] == ' ' {
			return i + 1
		}
	}
	
	// Break between Chinese and non-Chinese characters
	for i := searchEnd; i >= searchStart && i < lineLength-1; i-- {
		currentChar := runes[i]
		nextChar := runes[i+1]
		
		// Break between Chinese and English/numbers
		if isChinese(currentChar) && !isChinese(nextChar) {
			return i + 1
		}
		if !isChinese(currentChar) && isChinese(nextChar) {
			return i + 1
		}
	}
	
	// If no good break point found and line exceeds max length, force break
	if lineLength > maxLength {
		if preferredLength < lineLength {
			return preferredLength
		}
		return lineLength / 2
	}
	
	return -1 // No break needed
}

// getIndentation extracts the leading whitespace from a line
func getIndentation(line string) string {
	for i, char := range line {
		if char != ' ' && char != '\t' {
			return line[:i]
		}
	}
	return line // Entire line is whitespace
}

// isChinese checks if a character is Chinese
func isChinese(char rune) bool {
	return char >= 0x4e00 && char <= 0x9fff
}

// displayResults shows the formatting results
func displayResults(results []formatResult, options *formatOptions) error {
	totalChanges := 0
	totalErrors := 0

	for _, result := range results {
		totalChanges += len(result.changes)
		totalErrors += len(result.errors)

		if len(result.errors) > 0 {
			fmt.Printf("ERROR %s: %d errors\n", result.filePath, len(result.errors))
			for _, err := range result.errors {
				fmt.Printf("  Error: %v\n", err)
			}
		} else if result.hasChanges {
			if options.apply {
				fmt.Printf("APPLIED %s: %d changes applied\n", result.filePath, len(result.changes))
			} else {
				fmt.Printf("PREVIEW %s: %d changes available\n", result.filePath, len(result.changes))
			}
			
			if options.verbose {
				for _, change := range result.changes {
					fmt.Printf("  Line %d (%s): %s\n", change.line, change.rule, change.description)
					if len(change.before) < 100 && len(change.after) < 100 {
						fmt.Printf("    - %s\n", change.before)
						fmt.Printf("    + %s\n", change.after)
					}
				}
			}
		} else {
			fmt.Printf("CLEAN %s: no changes needed\n", result.filePath)
		}
	}

	// Summary
	fmt.Printf("\nSummary: %d files processed, %d changes", len(results), totalChanges)
	if options.apply {
		fmt.Printf(" applied")
	} else {
		fmt.Printf(" available")
	}
	if totalErrors > 0 {
		fmt.Printf(", %d errors", totalErrors)
	}
	fmt.Printf("\n")

	if !options.apply && totalChanges > 0 {
		fmt.Printf("\nTo apply changes, add --apply flag\n")
	}

	return nil
}

func init() {
	// Add flags
	K8sCmd.Flags().Bool("apply", false, "Apply changes to files (default is preview only)")
	K8sCmd.Flags().BoolP("recursive", "r", false, "Process directories recursively")
	K8sCmd.Flags().Bool("backup", false, "Create backup files before modifying")
	K8sCmd.Flags().StringSlice("rules", []string{}, "Comma-separated list of rules to apply (spacing,punctuation,linebreaks,anchors,links,emphasis)")
	K8sCmd.Flags().BoolP("verbose", "v", false, "Show detailed change information")
}