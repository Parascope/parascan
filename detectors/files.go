package detectors

import (
	"os"
	"path/filepath"
	"strings"
	"gopkg.in/yaml.v3"
)

// FileDetectors содержит конфигурацию для детекции технологий по файлам
type FileDetectors struct {
	Technologies map[string]TechnologyConfig `yaml:"technologies"`
}

// TechnologyConfig описывает конфигурацию детекции технологии
type TechnologyConfig struct {
	DisplayName  string   `yaml:"display_name"`
	Category     string   `yaml:"category,omitempty"`
	HostingMatch string   `yaml:"hosting_match,omitempty"`
	Files        []string `yaml:"files"`
	URLTemplate  string   `yaml:"url_template,omitempty"`
	FallbackURL  string   `yaml:"fallback_url,omitempty"`
}

// FilesDetector detects technologies based on file presence
type FilesDetector struct {
	data *FileDetectors
}

func NewFilesDetector(data *FileDetectors) *FilesDetector {
	return &FilesDetector{
		data: data,
	}
}

func (f *FilesDetector) Name() string {
	return "files"
}

func (f *FilesDetector) Detect(ctx *DetectionContext) (map[string]string, error) {
	results := make(map[string]string)

	// Детектируем все технологии
	for techKey, techConfig := range f.data.Technologies {
		if f.hasMatchingFiles(ctx.ProjectPath, techConfig.Files) {
			url := f.buildURL(techConfig, techKey, ctx.Results)
			results[techKey] = url
		}
	}

	return results, nil
}

func (f *FilesDetector) buildURL(config TechnologyConfig, technology string, contextResults map[string]string) string {
	// Get repo URL from context
	repoURL, hasRepo := contextResults["repo"]

	// Try to build dynamic URL
	if hasRepo && config.URLTemplate != "" {
		// Check if technology matches repo hosting (data-driven)
		if config.HostingMatch != "" && !strings.Contains(repoURL, config.HostingMatch) {
			// Hosting doesn't match - use fallback
			if config.FallbackURL != "" {
				return config.FallbackURL
			}
			return technology
		}

		// Use template if hosting matches or no hosting requirement
		return strings.ReplaceAll(config.URLTemplate, "{repo}", repoURL)
	}

	// Fallback to documentation URL or technology name
	if config.FallbackURL != "" {
		return config.FallbackURL
	}

	return technology
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

// loadFileDetectors загружает конфигурацию детекторов из YAML файла
func loadFileDetectors(path string) (*FileDetectors, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var detectors FileDetectors
	if err := yaml.Unmarshal(data, &detectors); err != nil {
		return nil, err
	}

	return &detectors, nil
}