package adapter

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectAdapter interface defines project-specific configurations
type ProjectAdapter interface {
	Name() string
	GetDictionaries() []string
	GetIgnorePatterns() []string
	GetFileExtensions() []string
	GetCustomRules() map[string]bool
}

// K8sAdapter provides configuration for Kubernetes projects
type K8sAdapter struct{}

func (a *K8sAdapter) Name() string {
	return "k8s"
}

func (a *K8sAdapter) GetDictionaries() []string {
	return []string{
		"dictionaries/k8s.txt",
		"dictionaries/cloud-native.txt",
		"dictionaries/docker.txt",
	}
}

func (a *K8sAdapter) GetIgnorePatterns() []string {
	return []string{
		"node_modules/**",
		".git/**",
		"public/**",
		"resources/**",
		"static/images/**",
		"layouts/**/*.html",
		"data/**/*.yaml",
		"data/**/*.yml",
	}
}

func (a *K8sAdapter) GetFileExtensions() []string {
	return []string{".md", ".txt", ".rst"}
}

func (a *K8sAdapter) GetCustomRules() map[string]bool {
	return map[string]bool{
		"ignore_code_blocks":   true,
		"ignore_inline_code":   true,
		"ignore_urls":          true,
		"ignore_yaml_headers":  true,
		"case_sensitive_terms": false,
	}
}

// GoAdapter provides configuration for Go projects
type GoAdapter struct{}

func (a *GoAdapter) Name() string {
	return "go"
}

func (a *GoAdapter) GetDictionaries() []string {
	return []string{
		"dictionaries/go.txt",
		"dictionaries/programming.txt",
	}
}

func (a *GoAdapter) GetIgnorePatterns() []string {
	return []string{
		"vendor/**",
		".git/**",
		"*.go", // Skip Go code files, focus on docs
	}
}

func (a *GoAdapter) GetFileExtensions() []string {
	return []string{".md", ".txt", ".rst"}
}

func (a *GoAdapter) GetCustomRules() map[string]bool {
	return map[string]bool{
		"ignore_code_blocks":   true,
		"ignore_inline_code":   true,
		"ignore_urls":          true,
		"case_sensitive_terms": true, // Go is case-sensitive
	}
}

// DockerAdapter provides configuration for Docker projects
type DockerAdapter struct{}

func (a *DockerAdapter) Name() string {
	return "docker"
}

func (a *DockerAdapter) GetDictionaries() []string {
	return []string{
		"dictionaries/docker.txt",
		"dictionaries/cloud-native.txt",
	}
}

func (a *DockerAdapter) GetIgnorePatterns() []string {
	return []string{
		".git/**",
		"node_modules/**",
		"build/**",
		"dist/**",
	}
}

func (a *DockerAdapter) GetFileExtensions() []string {
	return []string{".md", ".txt", ".rst"}
}

func (a *DockerAdapter) GetCustomRules() map[string]bool {
	return map[string]bool{
		"ignore_code_blocks":   true,
		"ignore_inline_code":   true,
		"ignore_urls":          true,
		"case_sensitive_terms": false,
	}
}

// GenericAdapter provides basic configuration for generic projects
type GenericAdapter struct{}

func (a *GenericAdapter) Name() string {
	return "generic"
}

func (a *GenericAdapter) GetDictionaries() []string {
	return []string{
		"dictionaries/base-en.txt",
	}
}

func (a *GenericAdapter) GetIgnorePatterns() []string {
	return []string{
		".git/**",
		"node_modules/**",
		"build/**",
		"dist/**",
		".next/**",
	}
}

func (a *GenericAdapter) GetFileExtensions() []string {
	return []string{".md", ".txt", ".rst", ".html"}
}

func (a *GenericAdapter) GetCustomRules() map[string]bool {
	return map[string]bool{
		"ignore_code_blocks":   true,
		"ignore_inline_code":   true,
		"ignore_urls":          true,
		"case_sensitive_terms": false,
	}
}

// GetAdapter returns the appropriate adapter for the given project type
func GetAdapter(projectType string) (ProjectAdapter, error) {
	switch strings.ToLower(projectType) {
	case "k8s", "kubernetes":
		return &K8sAdapter{}, nil
	case "go", "golang":
		return &GoAdapter{}, nil
	case "docker":
		return &DockerAdapter{}, nil
	case "generic", "":
		return &GenericAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}
}

// GetAllAdapters returns all available adapters
func GetAllAdapters() []ProjectAdapter {
	return []ProjectAdapter{
		&K8sAdapter{},
		&GoAdapter{},
		&DockerAdapter{},
		&GenericAdapter{},
	}
}

// ShouldIgnoreFile checks if a file should be ignored based on patterns
func ShouldIgnoreFile(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		
		// Handle ** patterns manually (simplified)
		if strings.Contains(pattern, "**") {
			parts := strings.Split(pattern, "**")
			if len(parts) == 2 {
				prefix := parts[0]
				suffix := parts[1]
				
				if strings.HasPrefix(filePath, prefix) && strings.HasSuffix(filePath, suffix) {
					return true
				}
			}
		}
	}
	return false
}