package checker

import (
	"encoding/json"
	"fmt"
	"io"
)

// CheckerType represents the type of quality checker
type CheckerType string

const (
	SpellCheckerType    CheckerType = "spell"
	GrammarCheckerType  CheckerType = "grammar"
	MarkdownCheckerType CheckerType = "markdown"
	ChineseCheckerType  CheckerType = "chinese"
)

// Severity represents the severity level of an issue
type Severity string

const (
	ErrorSeverity   Severity = "error"
	WarningSeverity Severity = "warning"
	InfoSeverity    Severity = "info"
)

// Issue represents a single quality issue found in a file
type Issue struct {
	Type        CheckerType `json:"type"`
	Severity    Severity    `json:"severity"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	Column      int         `json:"column"`
	Word        string      `json:"word,omitempty"`
	Message     string      `json:"message"`
	Suggestions []string    `json:"suggestions,omitempty"`
	RuleID      string      `json:"rule_id,omitempty"`
}

// CheckResult represents the result of a quality check operation
type CheckResult struct {
	TotalFiles   int     `json:"total_files"`
	CheckedFiles int     `json:"checked_files"`
	TotalIssues  int     `json:"total_issues"`
	Issues       []Issue `json:"issues"`
	ProjectType  string  `json:"project_type"`
	CheckerType  CheckerType `json:"checker_type"`
}

// OutputConsole outputs the check result to console format
func (r *CheckResult) OutputConsole(w io.Writer, verbose bool) error {
	if r.TotalIssues == 0 {
		fmt.Fprintf(w, "✅ No issues found in %d files\n", r.CheckedFiles)
		return nil
	}
	
	fmt.Fprintf(w, "Found %d issues in %d files:\n\n", r.TotalIssues, r.CheckedFiles)
	
	// Group issues by file
	fileIssues := make(map[string][]Issue)
	for _, issue := range r.Issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}
	
	for file, issues := range fileIssues {
		fmt.Fprintf(w, "📁 %s (%d issues):\n", file, len(issues))
		
		for _, issue := range issues {
			severityIcon := getSeverityIcon(issue.Severity)
			
			if issue.Line > 0 {
				fmt.Fprintf(w, "  %s Line %d:%d - %s", severityIcon, issue.Line, issue.Column, issue.Message)
			} else {
				fmt.Fprintf(w, "  %s %s", severityIcon, issue.Message)
			}
			
			if issue.Word != "" {
				fmt.Fprintf(w, " ('%s')", issue.Word)
			}
			
			if len(issue.Suggestions) > 0 {
				fmt.Fprintf(w, " → Suggestions: %s", joinStrings(issue.Suggestions, ", "))
			}
			
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}
	
	return nil
}

// OutputJSON outputs the check result in JSON format
func (r *CheckResult) OutputJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// AddIssue adds a new issue to the check result
func (r *CheckResult) AddIssue(issue Issue) {
	r.Issues = append(r.Issues, issue)
	r.TotalIssues++
}

// Checker interface defines the contract for all quality checkers
type Checker interface {
	Name() string
	Type() CheckerType
	CheckFile(filePath string) ([]Issue, error)
	CheckFiles(filePaths []string) (*CheckResult, error)
	SetProject(projectType string) error
}

// getSeverityIcon returns an icon for the given severity level
func getSeverityIcon(severity Severity) string {
	switch severity {
	case ErrorSeverity:
		return "❌"
	case WarningSeverity:
		return "⚠️"
	case InfoSeverity:
		return "ℹ️"
	default:
		return "•"
	}
}

// joinStrings joins string slice with separator (helper function)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}