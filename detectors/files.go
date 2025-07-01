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
	DisplayName string   `yaml:"display_name"`
	Files       []string `yaml:"files"`
	URLTemplate string   `yaml:"url_template,omitempty"`
	FallbackURL string   `yaml:"fallback_url,omitempty"`
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
	ciTechs := make(map[string]string)
	otherTechs := make(map[string]string)

	for techKey, techConfig := range f.data.Technologies {
		if f.hasMatchingFiles(ctx.ProjectPath, techConfig.Files) {
			url := f.buildURL(techConfig, techKey, ctx.Results)

			// Разделяем CI системы от остальных для приоритизации
			if f.isCITechnology(techKey) {
				ciTechs[techKey] = url
			} else {
				otherTechs[techKey] = url
			}
		}
	}

	// Применяем приоритизацию для CI в зависимости от хостинга
	if len(ciTechs) > 1 {
		repoURL := ctx.Results["repo"]
		for tech, url := range ciTechs {
			if f.isMatchingHosting(tech, repoURL) {
				results[tech] = url
				goto addOthers // Добавляем только подходящий CI
			}
		}
		// Если ни один не подходит, добавляем все
		for tech, url := range ciTechs {
			results[tech] = url
		}
	} else {
		// Один или ноль CI - добавляем как есть
		for tech, url := range ciTechs {
			results[tech] = url
		}
	}

addOthers:
	// Добавляем все остальные технологии
	for tech, url := range otherTechs {
		results[tech] = url
	}

	return results, nil
}

func (f *FilesDetector) buildURL(config TechnologyConfig, technology string, contextResults map[string]string) string {
	// Get repo URL from context
	repoURL, hasRepo := contextResults["repo"]

	// Try to build dynamic URL
	if hasRepo && config.URLTemplate != "" {
		// Check if CI technology matches repo hosting
		if f.isCITechnology(technology) {
			if f.isGitHubRepo(repoURL) && technology != "github-actions" {
				// Не GitHub Actions на GitHub repo - используем fallback
				if config.FallbackURL != "" {
					return config.FallbackURL
				}
				return technology
			}
			if f.isGitLabRepo(repoURL) && technology != "gitlab-ci" {
				// Не GitLab CI на GitLab repo - используем fallback
				if config.FallbackURL != "" {
					return config.FallbackURL
				}
				return technology
			}
		}

		// Use template if hosting matches
		return strings.ReplaceAll(config.URLTemplate, "{repo}", repoURL)
	}

	// Fallback to documentation URL or technology name
	if config.FallbackURL != "" {
		return config.FallbackURL
	}

	return technology
}

func (f *FilesDetector) isCITechnology(technology string) bool {
	ciTechs := map[string]bool{
		"gitlab-ci":           true,
		"github-actions":      true,
		"bitbucket-pipelines": true,
		"azure-devops":        true,
		"jenkins":             true,
		"circleci":            true,
		"travis-ci":           true,
	}
	return ciTechs[technology]
}

func (f *FilesDetector) isGitLabRepo(repoURL string) bool {
	return strings.Contains(repoURL, "gitlab.com")
}

func (f *FilesDetector) isGitHubRepo(repoURL string) bool {
	return strings.Contains(repoURL, "github.com")
}

func (f *FilesDetector) isMatchingHosting(technology, repoURL string) bool {
	if repoURL == "" {
		return false
	}

	switch technology {
	case "github-actions":
		return f.isGitHubRepo(repoURL)
	case "gitlab-ci":
		return f.isGitLabRepo(repoURL)
	case "bitbucket-pipelines":
		return strings.Contains(repoURL, "bitbucket.org")
	default:
		return false
	}
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