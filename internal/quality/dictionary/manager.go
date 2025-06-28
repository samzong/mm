package dictionary

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles dictionary loading and management
type Manager struct {
	personalDictPath string
	loadedWords      map[string]bool
}

// NewManager creates a new dictionary manager
func NewManager() (*Manager, error) {
	// Create a personal dictionary file in user's cache dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	cacheDir := filepath.Join(homeDir, ".cache", "mm")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	// Create dictionaries directory for user customization
	dictDir := filepath.Join(cacheDir, "dictionaries")
	if err := os.MkdirAll(dictDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create dictionaries directory: %w", err)
	}
	
	// Create example file if not exists
	if err := createExampleDictionary(dictDir); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to create example dictionary: %v\n", err)
	}
	
	personalDictPath := filepath.Join(cacheDir, "personal.dict")
	
	return &Manager{
		personalDictPath: personalDictPath,
		loadedWords:      make(map[string]bool),
	}, nil
}

// createExampleDictionary creates an example custom dictionary
func createExampleDictionary(dictDir string) error {
	examplePath := filepath.Join(dictDir, "custom-example.txt")
	
	// Don't overwrite if already exists
	if _, err := os.Stat(examplePath); err == nil {
		return nil
	}
	
	exampleContent := `# Custom Dictionary Example
# Add your project-specific terms here
# One word per line, comments start with #

# Example technical terms
api
frontend
backend
microservice
devops

# Example company/project names
# yourcompany
# yourproject

# Example abbreviations
# ui
# ux
# qa
`
	
	return os.WriteFile(examplePath, []byte(exampleContent), 0644)
}

// LoadDictionaries loads dictionaries from the specified paths
func (m *Manager) LoadDictionaries(dictPaths []string) error {
	// Clear previously loaded words
	m.loadedWords = make(map[string]bool)
	
	// Load each dictionary
	for _, dictPath := range dictPaths {
		if err := m.loadDictionary(dictPath); err != nil {
			// Log warning but continue with other dictionaries
			fmt.Fprintf(os.Stderr, "Warning: Failed to load dictionary %s: %v\n", dictPath, err)
		}
	}
	
	// Auto-load user custom dictionaries
	if err := m.loadUserCustomDictionaries(); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to load user custom dictionaries: %v\n", err)
	}
	
	// Create/update personal dictionary file
	return m.updatePersonalDictionary()
}

// loadUserCustomDictionaries automatically loads all .txt files from ~/.cache/mm/dictionaries/
func (m *Manager) loadUserCustomDictionaries() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	dictDir := filepath.Join(homeDir, ".cache", "mm", "dictionaries")
	
	// Check if directory exists
	if _, err := os.Stat(dictDir); os.IsNotExist(err) {
		return nil // No custom dictionaries, that's fine
	}
	
	// Read all .txt files in the directory
	files, err := filepath.Glob(filepath.Join(dictDir, "*.txt"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		// Skip if it's already loaded via project dictionaries
		relPath := "dictionaries/" + filepath.Base(file)
		alreadyLoaded := false
		for word := range m.loadedWords {
			if word == relPath {
				alreadyLoaded = true
				break
			}
		}
		
		if !alreadyLoaded {
			if err := m.loadSingleCustomDictionary(file); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load custom dictionary %s: %v\n", file, err)
			} else if os.Getenv("MM_VERBOSE") == "1" {
				fmt.Fprintf(os.Stderr, "Loaded custom dictionary %s\n", filepath.Base(file))
			}
		}
	}
	
	return nil
}

// loadSingleCustomDictionary loads a single custom dictionary file
func (m *Manager) loadSingleCustomDictionary(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	// Parse dictionary content (same logic as loadDictionary)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Skip words with hyphens (aspell doesn't support them in personal dictionaries)
		if strings.Contains(line, "-") {
			continue
		}
		
		// Skip words with underscores (aspell doesn't support them either)
		if strings.Contains(line, "_") {
			continue
		}
		
		// Skip words with numbers (aspell doesn't support them in personal dictionaries)
		hasNumber := false
		for _, char := range line {
			if char >= '0' && char <= '9' {
				hasNumber = true
				break
			}
		}
		if hasNumber {
			continue
		}
		
		// Add word to loaded words (case-insensitive)
		word := strings.ToLower(line)
		m.loadedWords[word] = true
	}
	
	return scanner.Err()
}

// loadDictionary loads a single dictionary file using priority order
func (m *Manager) loadDictionary(dictPath string) error {
	var content []byte
	var err error
	var source string
	
	// Priority 1: User cache directory (~/.cache/mm/dictionaries/)
	if strings.HasPrefix(dictPath, "dictionaries/") {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr == nil {
			userDictPath := filepath.Join(homeDir, ".cache", "mm", dictPath)
			if content, err = os.ReadFile(userDictPath); err == nil {
				source = "user cache"
				goto parseContent
			}
		}
	}
	
	// Priority 2: Embedded dictionaries (built-in) - Skip for now, implement later
	// TODO: Implement embedded dictionaries with proper go:embed
	
	// Priority 3: Project-level dictionaries (./dictionaries/ for backward compatibility)
	if strings.HasPrefix(dictPath, "dictionaries/") {
		// Try relative to executable
		if execPath, execErr := os.Executable(); execErr == nil {
			execDir := filepath.Dir(execPath)
			actualPath := filepath.Join(execDir, dictPath)
			if content, err = os.ReadFile(actualPath); err == nil {
				source = "executable dir"
				goto parseContent
			}
		}
		
		// Try relative to working directory
		if content, err = os.ReadFile(dictPath); err == nil {
			source = "working dir"
			goto parseContent
		}
	} else {
		// Direct path
		if content, err = os.ReadFile(dictPath); err == nil {
			source = "direct path"
			goto parseContent
		}
	}
	
	return fmt.Errorf("dictionary file not found: %s (tried user cache, embedded, project dir)", dictPath)

parseContent:
	if os.Getenv("MM_VERBOSE") == "1" {
		fmt.Fprintf(os.Stderr, "Loaded dictionary %s from %s\n", dictPath, source)
	}
	
	// Parse dictionary content
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Skip words with hyphens (aspell doesn't support them in personal dictionaries)
		if strings.Contains(line, "-") {
			continue
		}
		
		// Skip words with underscores (aspell doesn't support them either)
		if strings.Contains(line, "_") {
			continue
		}
		
		// Skip words with numbers (aspell doesn't support them in personal dictionaries)
		hasNumber := false
		for _, char := range line {
			if char >= '0' && char <= '9' {
				hasNumber = true
				break
			}
		}
		if hasNumber {
			continue
		}
		
		// Add word to loaded words (case-insensitive)
		word := strings.ToLower(line)
		m.loadedWords[word] = true
	}
	
	return scanner.Err()
}

// updatePersonalDictionary creates/updates the personal dictionary file for aspell
func (m *Manager) updatePersonalDictionary() error {
	file, err := os.Create(m.personalDictPath)
	if err != nil {
		return fmt.Errorf("failed to create personal dictionary: %w", err)
	}
	defer file.Close()
	
	// Write personal dictionary header
	fmt.Fprintln(file, "personal_ws-1.1 en 0")
	
	// Write all loaded words
	for word := range m.loadedWords {
		fmt.Fprintln(file, word)
	}
	
	return nil
}

// GetPersonalDictPath returns the path to the personal dictionary file
func (m *Manager) GetPersonalDictPath() string {
	return m.personalDictPath
}

// AddWord adds a word to the personal dictionary
func (m *Manager) AddWord(word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return fmt.Errorf("empty word")
	}
	
	m.loadedWords[word] = true
	return m.updatePersonalDictionary()
}

// IsWordKnown checks if a word is in the loaded dictionaries
func (m *Manager) IsWordKnown(word string) bool {
	return m.loadedWords[strings.ToLower(word)]
}

// GetLoadedWordsCount returns the number of loaded words
func (m *Manager) GetLoadedWordsCount() int {
	return len(m.loadedWords)
}