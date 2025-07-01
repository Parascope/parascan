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
	Files               []string `yaml:"files"`
	URLTemplate         string   `yaml:"url_template"`
	GitHubURLTemplate   string   `yaml:"github_url_template"`
	FallbackURL         string   `yaml:"fallback_url"`
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

func (f *FilesDetector) Detect(ctx *DetectionContext) (map[string]string, error) {
	results := make(map[string]string)

	// Iterate through all categories and technologies
	for category, technologies := range f.data.Categories {
		var foundTechs []string
		var foundPatterns []FilePattern

		// Collect all matching technologies for this category
		for technology, pattern := range technologies {
			if f.hasMatchingFiles(ctx.ProjectPath, pattern.Files) {
				foundTechs = append(foundTechs, technology)
				foundPatterns = append(foundPatterns, pattern)
			}
		}

		// If multiple technologies found, prioritize by repo hosting match
		if len(foundTechs) > 1 {
			repoURL := ctx.Results["repo"]
			for i, tech := range foundTechs {
				if f.isMatchingHosting(tech, repoURL) {
					url := f.buildURL(foundPatterns[i], tech, ctx.Results)
					results[category] = url
					break
				}
			}
		} else if len(foundTechs) == 1 {
			// Single technology found
			url := f.buildURL(foundPatterns[0], foundTechs[0], ctx.Results)
			results[category] = url
		}
	}

	return results, nil
}

func (f *FilesDetector) buildURL(pattern FilePattern, technology string, contextResults map[string]string) string {
	// Get repo URL from context
	repoURL, hasRepo := contextResults["repo"]

	// Try to build dynamic URL
	if hasRepo && pattern.URLTemplate != "" {
		// Special handling for GitHub when github_url_template is available
		if pattern.GitHubURLTemplate != "" && f.isGitHubRepo(repoURL) {
			repoName := f.extractRepoName(repoURL)
			url := strings.ReplaceAll(pattern.GitHubURLTemplate, "{repo}", repoURL)
			url = strings.ReplaceAll(url, "{repo_name}", repoName)
			return url
		}

		// Check if CI technology matches repo hosting
		if f.isCITechnology(technology) {
			if f.isGitHubRepo(repoURL) && technology != "github-actions" {
				// GitLab CI on GitHub repo - use fallback
				if pattern.FallbackURL != "" {
					return pattern.FallbackURL
				}
				return technology
			}
			if f.isGitLabRepo(repoURL) && technology != "gitlab" {
				// GitHub Actions on GitLab repo - use fallback
				if pattern.FallbackURL != "" {
					return pattern.FallbackURL
				}
				return technology
			}
		}

		// Use default template if hosting matches
		return strings.ReplaceAll(pattern.URLTemplate, "{repo}", repoURL)
	}

	// Fallback to documentation URL or technology name
	if pattern.FallbackURL != "" {
		return pattern.FallbackURL
	}

	return technology
}

func (f *FilesDetector) isCITechnology(technology string) bool {
	ciTechs := map[string]bool{
		"gitlab":         true,
		"github-actions": true,
		"bitbucket":      true,
		"azure-devops":   true,
		"jenkins":        true,
		"circleci":       true,
	}
	return ciTechs[technology]
}

func (f *FilesDetector) isGitLabRepo(repoURL string) bool {
	return strings.Contains(repoURL, "gitlab.com")
}

func (f *FilesDetector) isMatchingHosting(technology, repoURL string) bool {
	if repoURL == "" {
		return false
	}

	switch technology {
	case "github-actions":
		return f.isGitHubRepo(repoURL)
	case "gitlab":
		return f.isGitLabRepo(repoURL)
	case "bitbucket":
		return strings.Contains(repoURL, "bitbucket.org")
	default:
		return false
	}
}

func (f *FilesDetector) isGitHubRepo(repoURL string) bool {
	return strings.Contains(repoURL, "github.com")
}

func (f *FilesDetector) extractRepoName(repoURL string) string {
	// Extract repo name from URL like https://github.com/user/repo
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return ""
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