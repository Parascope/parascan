package detectors

import (
	"os"
	"path/filepath"
	"strings"
)

// FileDetectorData represents the structure of file-detectors.yml
type FileDetectorData struct {
	Categories map[string]map[string]FilePattern `yaml:",inline"`
}

type FilePattern struct {
	Files []string `yaml:"files"`
}

// FilesDetector detects technologies based on file presence
type FilesDetector struct {
	data *FileDetectorData
}

func NewFilesDetector(data *FileDetectorData) *FilesDetector {
	return &FilesDetector{
		data: data,
	}
}

func (f *FilesDetector) Name() string {
	return "files"
}

func (f *FilesDetector) Detect(projectPath string) (map[string]string, error) {
	results := make(map[string]string)

	// Iterate through all categories and technologies
	for category, technologies := range f.data.Categories {
		for technology, pattern := range technologies {
			if f.hasMatchingFiles(projectPath, pattern.Files) {
				results[category] = technology
				break // Only one technology per category
			}
		}
	}

	return results, nil
}

func (f *FilesDetector) hasMatchingFiles(projectPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if f.hasMatchingFile(projectPath, pattern) {
			return true
		}
	}
	return false
}

func (f *FilesDetector) hasMatchingFile(dir, pattern string) bool {
	// If pattern ends with /, it's a directory check
	if strings.HasSuffix(pattern, "/") {
		dirPath := filepath.Join(dir, strings.TrimSuffix(pattern, "/"))
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			return true
		}
		return false
	}

	// If pattern contains subdirectories (e.g. "k8s/*.yml")
	if strings.Contains(pattern, "/") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		return err == nil && len(matches) > 0
	}

	// If pattern contains wildcards (e.g. "*.tf")
	if strings.Contains(pattern, "*") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		return err == nil && len(matches) > 0
	}

	// Regular file
	_, err := os.Stat(filepath.Join(dir, pattern))
	return err == nil
}