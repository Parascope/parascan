package detectors

// DetectionContext provides context for detectors
type DetectionContext struct {
	ProjectPath string
	Results     map[string]string // results from previous detectors
}

// Detector interface for all detection plugins
type Detector interface {
	Name() string
	Detect(ctx *DetectionContext) (map[string]string, error) // key -> value for sitedog.yml
}

// SimpleDetector is for detectors that don't need context
type SimpleDetector interface {
	Name() string
	Detect(projectPath string) (map[string]string, error)
}