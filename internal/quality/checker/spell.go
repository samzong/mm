package checker

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samzong/mm/internal/quality/adapter"
	"github.com/samzong/mm/internal/quality/dictionary"
)

// SpellChecker implements the Checker interface for spell checking
type SpellChecker struct {
	projectType string
	adapter     adapter.ProjectAdapter
	dictManager *dictionary.Manager
}

// NewSpellChecker creates a new spell checker instance
func NewSpellChecker() (*SpellChecker, error) {
	dictManager, err := dictionary.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dictionary manager: %w", err)
	}
	
	return &SpellChecker{
		projectType: "generic",
		dictManager: dictManager,
	}, nil
}

// Name returns the name of this checker
func (s *SpellChecker) Name() string {
	return "Spell Checker"
}

// Type returns the type of this checker
func (s *SpellChecker) Type() CheckerType {
	return SpellCheckerType
}

// SetProject sets the project type and loads appropriate configuration
func (s *SpellChecker) SetProject(projectType string) error {
	projectAdapter, err := adapter.GetAdapter(projectType)
	if err != nil {
		return fmt.Errorf("failed to get adapter for project type %s: %w", projectType, err)
	}
	
	s.projectType = projectType
	s.adapter = projectAdapter
	
	// Load dictionaries for this project type
	return s.dictManager.LoadDictionaries(projectAdapter.GetDictionaries())
}

// CheckFile checks a single file for spelling errors
func (s *SpellChecker) CheckFile(filePath string) ([]Issue, error) {
	// Check if file should be ignored
	if s.adapter != nil && adapter.ShouldIgnoreFile(filePath, s.adapter.GetIgnorePatterns()) {
		return nil, nil
	}
	
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	// Extract text content based on file type
	textContent, err := s.extractTextContent(string(content), filepath.Ext(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from %s: %w", filePath, err)
	}
	
	// Run spell check using aspell
	issues, err := s.runAspellCheck(filePath, textContent)
	if err != nil {
		return nil, fmt.Errorf("aspell check failed for %s: %w", filePath, err)
	}
	
	return issues, nil
}

// CheckFiles checks multiple files for spelling errors
func (s *SpellChecker) CheckFiles(filePaths []string) (*CheckResult, error) {
	result := &CheckResult{
		TotalFiles:  len(filePaths),
		CheckedFiles: 0,
		TotalIssues: 0,
		Issues:      []Issue{},
		ProjectType: s.projectType,
		CheckerType: SpellCheckerType,
	}
	
	for _, filePath := range filePaths {
		issues, err := s.CheckFile(filePath)
		if err != nil {
			// Log error but continue with other files
			fmt.Fprintf(os.Stderr, "Warning: Failed to check %s: %v\n", filePath, err)
			continue
		}
		
		result.CheckedFiles++
		for _, issue := range issues {
			result.AddIssue(issue)
		}
	}
	
	return result, nil
}

// extractTextContent extracts readable text from different file formats
func (s *SpellChecker) extractTextContent(content, fileExt string) (string, error) {
	switch strings.ToLower(fileExt) {
	case ".md", ".markdown":
		return s.extractFromMarkdown(content), nil
	case ".txt":
		return content, nil
	case ".rst":
		return s.extractFromRST(content), nil
	case ".html":
		return s.extractFromHTML(content), nil
	default:
		return content, nil
	}
}

// extractFromMarkdown extracts text content from markdown, ignoring code blocks and links
func (s *SpellChecker) extractFromMarkdown(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	codeBlockPattern := regexp.MustCompile("^```")
	
	for lineNum, line := range lines {
		// Skip YAML front matter
		if lineNum < 10 && strings.TrimSpace(line) == "---" {
			continue
		}
		
		// Handle code blocks
		if codeBlockPattern.MatchString(line) {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}
		
		// Remove inline code
		inlineCodePattern := regexp.MustCompile("`[^`]+`")
		line = inlineCodePattern.ReplaceAllString(line, "")
		
		// Remove links but keep link text
		linkPattern := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
		line = linkPattern.ReplaceAllString(line, "$1")
		
		// Remove image references
		imagePattern := regexp.MustCompile(`!\[[^\]]*\]\([^\)]+\)`)
		line = imagePattern.ReplaceAllString(line, "")
		
		// Remove HTML tags
		htmlPattern := regexp.MustCompile(`<[^>]+>`)
		line = htmlPattern.ReplaceAllString(line, "")
		
		// Remove markdown formatting
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "_", "")
		line = strings.ReplaceAll(line, "##", "")
		line = strings.ReplaceAll(line, "#", "")
		
		result.WriteString(line + "\n")
	}
	
	return result.String()
}

// extractFromRST extracts text content from reStructuredText
func (s *SpellChecker) extractFromRST(content string) string {
	// Basic RST text extraction (simplified)
	lines := strings.Split(content, "\n")
	var result strings.Builder
	
	for _, line := range lines {
		// Skip directive lines
		if strings.HasPrefix(strings.TrimSpace(line), ".. ") {
			continue
		}
		
		// Remove inline markup
		line = regexp.MustCompile(`\*\*[^*]+\*\*`).ReplaceAllString(line, "")
		line = regexp.MustCompile(`\*[^*]+\*`).ReplaceAllString(line, "")
		line = regexp.MustCompile("``[^`]+``").ReplaceAllString(line, "")
		
		result.WriteString(line + "\n")
	}
	
	return result.String()
}

// extractFromHTML extracts text content from HTML
func (s *SpellChecker) extractFromHTML(content string) string {
	// Remove HTML tags
	htmlPattern := regexp.MustCompile(`<[^>]*>`)
	return htmlPattern.ReplaceAllString(content, " ")
}

// runAspellCheck runs aspell on the given text content
func (s *SpellChecker) runAspellCheck(filePath, content string) ([]Issue, error) {
	// Check if aspell is available
	if _, err := exec.LookPath("aspell"); err != nil {
		return nil, fmt.Errorf("aspell not found in PATH. Please install aspell")
	}
	
	// Build aspell command
	args := []string{
		"--mode=none",
		"--encoding=utf-8",
		"--list",
	}
	
	// Add custom dictionaries if available
	personalDict := s.dictManager.GetPersonalDictPath()
	if personalDict != "" {
		args = append(args, "--personal="+personalDict)
	}
	
	// Run aspell with stdin input
	cmd := exec.Command("aspell", args...)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aspell command failed: %w", err)
	}
	
	// Parse aspell output
	return s.parseAspellOutput(filePath, content, string(output))
}

// parseAspellOutput parses aspell output and creates Issue objects
func (s *SpellChecker) parseAspellOutput(filePath, content, aspellOutput string) ([]Issue, error) {
	var issues []Issue
	
	misspelledWords := strings.Fields(strings.TrimSpace(aspellOutput))
	if len(misspelledWords) == 0 {
		return issues, nil
	}
	
	// Create a map to track already reported words (avoid duplicates)
	reportedWords := make(map[string]bool)
	
	lines := strings.Split(content, "\n")
	
	for _, word := range misspelledWords {
		// Use lowercase for deduplication since aspell returns lowercase
		lowerWord := strings.ToLower(word)
		if reportedWords[lowerWord] {
			continue
		}
		reportedWords[lowerWord] = true
		
		// Find word positions in content (this will handle case-insensitive matching)
		positions := s.findWordPositions(lines, word)
		
		for _, pos := range positions {
			// Get the actual word from the content for the error message
			actualWord := s.getActualWordAtPosition(lines, pos, word)
			
			// Get spelling suggestions
			suggestions := s.getSpellingSuggestions(word)
			
			issue := Issue{
				Type:        SpellCheckerType,
				Severity:    ErrorSeverity,
				File:        filePath,
				Line:        pos.Line,
				Column:      pos.Column,
				Word:        actualWord, // Use the actual word from content, not the lowercase version
				Message:     fmt.Sprintf("Misspelled word: '%s'", actualWord),
				Suggestions: suggestions,
				RuleID:      "spell-check",
			}
			
			issues = append(issues, issue)
		}
	}
	
	return issues, nil
}

// WordPosition represents the position of a word in text
type WordPosition struct {
	Line   int
	Column int
}

// findWordPositions finds all positions of a word in the content
func (s *SpellChecker) findWordPositions(lines []string, word string) []WordPosition {
	var positions []WordPosition
	
	// Create case-insensitive word boundary regex
	wordPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
	
	for lineNum, line := range lines {
		matches := wordPattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			// Get the actual word from the line to preserve original case
			actualWord := line[match[0]:match[1]]
			
			// Check if this specific case variant is known in dictionary
			if !s.dictManager.IsWordKnown(actualWord) && !s.dictManager.IsWordKnown(strings.ToLower(actualWord)) {
				positions = append(positions, WordPosition{
					Line:   lineNum + 1, // 1-based line numbering
					Column: match[0] + 1, // 1-based column numbering
				})
			}
		}
	}
	
	return positions
}

// getActualWordAtPosition extracts the actual word from the content at the given position
func (s *SpellChecker) getActualWordAtPosition(lines []string, pos WordPosition, expectedWord string) string {
	if pos.Line-1 >= len(lines) {
		return expectedWord
	}
	
	line := lines[pos.Line-1]
	if pos.Column-1 >= len(line) {
		return expectedWord
	}
	
	// Find word boundaries around the position
	start := pos.Column - 1
	end := start
	
	// Find start of word
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	
	// Find end of word
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	
	if start < end {
		return line[start:end]
	}
	
	return expectedWord
}

// isWordChar checks if a character is part of a word
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '\'' || c == '-'
}

// getSpellingSuggestions gets spelling suggestions for a misspelled word
func (s *SpellChecker) getSpellingSuggestions(word string) []string {
	// Use aspell to get suggestions
	cmd := exec.Command("aspell", "--mode=none", "--encoding=utf-8", "pipe")
	
	var stdin bytes.Buffer
	stdin.WriteString("!" + word + "\n")
	cmd.Stdin = &stdin
	
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	
	// Parse aspell pipe output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "&") {
			// Format: & word count offset: suggestion1, suggestion2, ...
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				suggestions := strings.Split(strings.TrimSpace(parts[1]), ",")
				var cleanSuggestions []string
				for _, s := range suggestions {
					cleanSuggestions = append(cleanSuggestions, strings.TrimSpace(s))
				}
				return cleanSuggestions[:min(5, len(cleanSuggestions))] // Return max 5 suggestions
			}
		}
	}
	
	return nil
}

// Helper function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}