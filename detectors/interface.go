package detectors

// Detector interface for all detection plugins
type Detector interface {
	Name() string
	Detect(projectPath string) (map[string]string, error) // key -> value for sitedog.yml
}