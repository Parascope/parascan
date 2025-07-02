package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v2"
	"sitedog/detectors"
)

type ExpectedResults struct {
	ExpectedServices  []string `yaml:"expected_services"`
	ExpectedLanguages []string `yaml:"expected_languages"`
	Description       string   `yaml:"description"`
}

func TestServiceDetectionWithFixtures(t *testing.T) {
	// Load test data and services data
	stackData, err := loadStackDependencyFiles()
	if err != nil {
		t.Fatalf("Failed to load stack data: %v", err)
	}

	servicesData, err := loadServicesData()
	if err != nil {
		t.Fatalf("Failed to load services data: %v", err)
	}

	testCases := []string{
		"ruby-project",
		"nodejs-project",
		"python-project",
		"multi-language",
		"empty-project",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			projectPath := filepath.Join("testdata", testCase)
			expectedPath := filepath.Join(projectPath, "expected.yml")

			// Load expected results
			expectedData, err := ioutil.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected.yml for %s: %v", testCase, err)
			}

			var expected ExpectedResults
			err = yaml.Unmarshal(expectedData, &expected)
			if err != nil {
				t.Fatalf("Failed to parse expected.yml for %s: %v", testCase, err)
			}

			// Test language detection
			detectedLanguages := detectProjectLanguages(projectPath, stackData)
			sort.Strings(detectedLanguages)
			sort.Strings(expected.ExpectedLanguages)

			if !equalStringSlices(detectedLanguages, expected.ExpectedLanguages) {
				t.Errorf("%s: Expected languages %v, got %v",
					testCase, expected.ExpectedLanguages, detectedLanguages)
			}

			// Test service detection
			if len(detectedLanguages) > 0 {
				results := analyzeProjectDependencies(projectPath, detectedLanguages, stackData, servicesData)

				// Collect all detected services
				var detectedServices []string
				for _, result := range results {
					for _, service := range result.Services {
						detectedServices = append(detectedServices, service.Name)
					}
				}

				// Remove duplicates and sort
				detectedServices = removeDuplicates(detectedServices)
				sort.Strings(detectedServices)
				sort.Strings(expected.ExpectedServices)

				if !equalStringSlices(detectedServices, expected.ExpectedServices) {
					t.Errorf("%s: Expected services %v, got %v",
						testCase, expected.ExpectedServices, detectedServices)
				}

				// Print success message with details
				t.Logf("✅ %s: Successfully detected %d languages and %d services",
					testCase, len(detectedLanguages), len(detectedServices))
				if expected.Description != "" {
					t.Logf("   Description: %s", expected.Description)
				}
			}
		})
	}
}

func TestEndToEndServiceDetection(t *testing.T) {
	// Test the complete sniff workflow using fixtures
	testCases := []struct {
		name     string
		project  string
		minServices int
	}{
		{"Ruby project", "ruby-project", 8},
		{"Node.js project", "nodejs-project", 8},
		{"Python project", "python-project", 10},
		{"Multi-language project", "multi-language", 6},
		{"Empty project", "empty-project", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectPath := filepath.Join("testdata", tc.project)

			// Load dependencies
			stackData, err := loadStackDependencyFiles()
			if err != nil {
				t.Fatalf("Failed to load stack data: %v", err)
			}

			servicesData, err := loadServicesData()
			if err != nil {
				t.Fatalf("Failed to load services data: %v", err)
			}

			// Create adapter (same as in handleSniff)
			adapter := &ServicesDependenciesAdapter{
				stackData:    stackData,
				servicesData: servicesData,
			}

			// Test services detector
			servicesDetector := detectors.NewServicesDetector(adapter)
			results, err := servicesDetector.Detect(projectPath)
			if err != nil {
				t.Fatalf("Services detector failed: %v", err)
			}

			serviceCount := len(results)
			if serviceCount < tc.minServices {
				t.Errorf("Expected at least %d services, got %d", tc.minServices, serviceCount)
			}

			// Log detected services
			if serviceCount > 0 {
				t.Logf("Detected %d services:", serviceCount)
				for service, url := range results {
					t.Logf("  - %s: %s", service, url)
				}
			} else {
				t.Logf("No services detected (expected for %s)", tc.project)
			}
		})
	}
}

// Helper function to check if two string slices are equal
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function to remove duplicates from string slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

func TestSpecificServiceDetection(t *testing.T) {
	// Test specific services that were problematic
	tests := []struct {
		project     string
		service     string
		shouldFind  bool
	}{
		{"ruby-project", "stripe", true},
		{"ruby-project", "twilio", true},
		{"ruby-project", "aws", true},
		{"nodejs-project", "openai", true},
		{"nodejs-project", "anthropic", true},
		{"python-project", "stripe", true},
		{"empty-project", "stripe", false},
	}

	// Load dependencies
	stackData, err := loadStackDependencyFiles()
	if err != nil {
		t.Fatalf("Failed to load stack data: %v", err)
	}

	servicesData, err := loadServicesData()
	if err != nil {
		t.Fatalf("Failed to load services data: %v", err)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.project, tt.service), func(t *testing.T) {
			projectPath := filepath.Join("testdata", tt.project)

			detectedLanguages := detectProjectLanguages(projectPath, stackData)
			if len(detectedLanguages) == 0 && tt.shouldFind {
				t.Fatalf("No languages detected for %s", tt.project)
			}

			results := analyzeProjectDependencies(projectPath, detectedLanguages, stackData, servicesData)

			found := false
			for _, result := range results {
				for _, service := range result.Services {
					if service.Name == tt.service {
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if found != tt.shouldFind {
				if tt.shouldFind {
					t.Errorf("Expected to find service %s in %s, but it was not detected", tt.service, tt.project)
				} else {
					t.Errorf("Did not expect to find service %s in %s, but it was detected", tt.service, tt.project)
				}
			}
		})
	}
}