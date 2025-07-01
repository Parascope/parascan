package detectors

// Dependencies interface for services detector
type ServicesDependencies interface {
	DetectProjectLanguages(projectPath string) []string
	AnalyzeProjectDependencies(projectPath string, languages []string) []ProjectResult
	GetServicesData() map[string]*ServiceInfo
}

// ServiceInfo represents service metadata
type ServiceInfo struct {
	Name string
	URL  string
}

// ProjectResult represents analysis result for a project
type ProjectResult struct {
	Language string
	Services []ServiceResult
}

// ServiceResult represents a detected service
type ServiceResult struct {
	Name string
}

// ServicesDetector wraps existing services detection logic
type ServicesDetector struct {
	deps ServicesDependencies
}

func NewServicesDetector(deps ServicesDependencies) *ServicesDetector {
	return &ServicesDetector{
		deps: deps,
	}
}

func (s *ServicesDetector) Name() string {
	return "services"
}

func (s *ServicesDetector) Detect(projectPath string) (map[string]string, error) {
	results := make(map[string]string)

	// Use existing logic through interface
	detectedLanguages := s.deps.DetectProjectLanguages(projectPath)
	if len(detectedLanguages) == 0 {
		return results, nil
	}

	projectResults := s.deps.AnalyzeProjectDependencies(projectPath, detectedLanguages)
	servicesData := s.deps.GetServicesData()

	// Convert to simple key-value pairs
	for _, result := range projectResults {
		for _, service := range result.Services {
			if serviceData, exists := servicesData[service.Name]; exists {
				results[serviceData.Name] = serviceData.URL
			}
		}
	}

	return results, nil
}