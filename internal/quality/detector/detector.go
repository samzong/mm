package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectType represents different project types
type ProjectType string

const (
	K8sWebsiteProject ProjectType = "k8s"
	GoProject         ProjectType = "go"
	DockerProject     ProjectType = "docker"
	GenericProject    ProjectType = "generic"
)

// ProjectDetector interface for detecting project types
type ProjectDetector interface {
	Name() string
	Detect(rootPath string) bool
	Priority() int // Higher number = higher priority
}

// K8sWebsiteDetector detects Kubernetes website projects
type K8sWebsiteDetector struct{}

func (d *K8sWebsiteDetector) Name() string {
	return string(K8sWebsiteProject)
}

func (d *K8sWebsiteDetector) Detect(rootPath string) bool {
	// Check for kubernetes/website specific files
	lsyncPath := filepath.Join(rootPath, "scripts", "lsync.sh")
	contentPath := filepath.Join(rootPath, "content", "en")
	
	return fileExists(lsyncPath) && dirExists(contentPath)
}

func (d *K8sWebsiteDetector) Priority() int {
	return 100 // High priority for specific detection
}

// GoProjectDetector detects Go projects
type GoProjectDetector struct{}

func (d *GoProjectDetector) Name() string {
	return string(GoProject)
}

func (d *GoProjectDetector) Detect(rootPath string) bool {
	// Check for go.mod or go.sum files
	goModPath := filepath.Join(rootPath, "go.mod")
	goSumPath := filepath.Join(rootPath, "go.sum")
	
	return fileExists(goModPath) || fileExists(goSumPath)
}

func (d *GoProjectDetector) Priority() int {
	return 50 // Medium priority
}

// DockerProjectDetector detects Docker projects
type DockerProjectDetector struct{}

func (d *DockerProjectDetector) Name() string {
	return string(DockerProject)
}

func (d *DockerProjectDetector) Detect(rootPath string) bool {
	// Check for Dockerfile or docker-compose.yml
	dockerfilePath := filepath.Join(rootPath, "Dockerfile")
	dockerComposePath := filepath.Join(rootPath, "docker-compose.yml")
	dockerComposeYamlPath := filepath.Join(rootPath, "docker-compose.yaml")
	
	return fileExists(dockerfilePath) || fileExists(dockerComposePath) || fileExists(dockerComposeYamlPath)
}

func (d *DockerProjectDetector) Priority() int {
	return 30 // Lower priority
}

// GenericProjectDetector fallback detector
type GenericProjectDetector struct{}

func (d *GenericProjectDetector) Name() string {
	return string(GenericProject)
}

func (d *GenericProjectDetector) Detect(rootPath string) bool {
	return true // Always matches as fallback
}

func (d *GenericProjectDetector) Priority() int {
	return 1 // Lowest priority
}

// DetectProject detects the project type in the given root path
func DetectProject(rootPath string) (string, error) {
	// List of all detectors, ordered by priority
	detectors := []ProjectDetector{
		&K8sWebsiteDetector{},
		&GoProjectDetector{},
		&DockerProjectDetector{},
		&GenericProjectDetector{},
	}
	
	// Find the highest priority detector that matches
	var bestDetector ProjectDetector
	bestPriority := -1
	
	for _, detector := range detectors {
		if detector.Detect(rootPath) && detector.Priority() > bestPriority {
			bestDetector = detector
			bestPriority = detector.Priority()
		}
	}
	
	if bestDetector == nil {
		return string(GenericProject), fmt.Errorf("no project detector matched")
	}
	
	return bestDetector.Name(), nil
}

// GetSupportedProjects returns a list of all supported project types
func GetSupportedProjects() []string {
	return []string{
		string(K8sWebsiteProject),
		string(GoProject),
		string(DockerProject),
		string(GenericProject),
	}
}

// Helper functions

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}