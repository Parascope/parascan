package detectors

import (
	"os/exec"
	"regexp"
	"strings"
)

// GitRepositoryDetector detects git repository information
type GitRepositoryDetector struct{}

func (g *GitRepositoryDetector) Name() string {
	return "git"
}

func (g *GitRepositoryDetector) Detect(projectPath string) (map[string]string, error) {
	results := make(map[string]string)

	if !isGitRepository() {
		return results, nil
	}

	originURL, err := getGitOriginURL()
	if err != nil {
		return results, err
	}

	if originURL != "" {
		repoURL := convertToHTTPSURL(originURL)
		results["repo"] = repoURL
	}

	return results, nil
}

// Helper functions for git operations
func isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

func getGitOriginURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func convertToHTTPSURL(gitURL string) string {
	// Pattern for SSH URLs like git@github.com:user/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+)\.git$`)
	if matches := sshPattern.FindStringSubmatch(gitURL); len(matches) == 3 {
		return "https://" + matches[1] + "/" + matches[2]
	}

	// Pattern for SSH URLs like git@github.com:user/repo (without .git)
	sshPatternNoGit := regexp.MustCompile(`^git@([^:]+):(.+)$`)
	if matches := sshPatternNoGit.FindStringSubmatch(gitURL); len(matches) == 3 {
		return "https://" + matches[1] + "/" + matches[2]
	}

	// If it's already HTTPS or HTTP, remove .git suffix if present
	if strings.HasPrefix(gitURL, "http://") || strings.HasPrefix(gitURL, "https://") {
		return strings.TrimSuffix(gitURL, ".git")
	}

	return gitURL
}