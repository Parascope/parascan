package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestIsPackageInFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		fileName    string
		packageName string
		language    string
		expected    bool
	}{
		{
			name:        "Ruby gem in Gemfile",
			content:     "gem 'stripe', '~> 5.0'",
			fileName:    "Gemfile",
			packageName: "stripe",
			language:    "ruby",
			expected:    true,
		},
		{
			name:        "Ruby gem with exact match",
			content:     "gem 'twilio-ruby', '~> 5.0'",
			fileName:    "Gemfile",
			packageName: "twilio-ruby",
			language:    "ruby",
			expected:    true,
		},
		{
			name:        "Package not found",
			content:     "gem 'rails', '~> 7.0'",
			fileName:    "Gemfile",
			packageName: "stripe",
			language:    "ruby",
			expected:    false,
		},
		{
			name:        "JavaScript package in package.json",
			content:     `{"dependencies": {"stripe": "^8.0.0"}}`,
			fileName:    "package.json",
			packageName: "stripe",
			language:    "nodejs",
			expected:    true,
		},
		{
			name:        "Python package in requirements.txt",
			content:     "stripe==2.60.0\ndjango==3.2.0",
			fileName:    "requirements.txt",
			packageName: "stripe",
			language:    "python",
			expected:    true,
		},
	}

		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPackageInFile(tt.content, tt.fileName, tt.packageName, tt.language)
			if result != tt.expected {
				t.Errorf("isPackageInFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	// Create test services data
	servicesData := map[string]*ServiceData{
		"stripe": {
			Name: "Stripe",
			URL:  "https://dashboard.stripe.com",
			Stacks: map[string][]string{
				"ruby":   {"stripe", "stripe_event"},
				"nodejs": {"stripe", "@stripe/stripe-js"},
			},
		},
		"twilio": {
			Name: "Twilio",
			URL:  "https://console.twilio.com",
			Stacks: map[string][]string{
				"ruby":   {"twilio-ruby", "twilio-rails"},
				"nodejs": {"twilio", "@twilio/voice-sdk"},
			},
		},
	}

	tests := []struct {
		name            string
		fileContent     string
		language        string
		expectedCount   int
		expectedService string
	}{
		{
			name:            "Ruby Gemfile with stripe",
			fileContent:     "gem 'stripe', '~> 5.0'\ngem 'rails', '~> 7.0'",
			language:        "ruby",
			expectedCount:   1,
			expectedService: "stripe",
		},
		{
			name:            "Ruby Gemfile with twilio",
			fileContent:     "gem 'twilio-ruby', '~> 5.0'\ngem 'rails', '~> 7.0'",
			language:        "ruby",
			expectedCount:   1,
			expectedService: "twilio",
		},
		{
			name:            "Ruby Gemfile with both services",
			fileContent:     "gem 'stripe', '~> 5.0'\ngem 'twilio-ruby', '~> 5.0'\ngem 'rails', '~> 7.0'",
			language:        "ruby",
			expectedCount:   2,
			expectedService: "", // multiple services
		},
		{
			name:          "No matching packages",
			fileContent:   "gem 'rails', '~> 7.0'\ngem 'pg', '~> 1.0'",
			language:      "ruby",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir, err := ioutil.TempDir("", "sitedog-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "Gemfile")
			err = ioutil.WriteFile(testFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Test analyzeFile
			detections := analyzeFile(testFile, tt.language, servicesData)

			if len(detections) != tt.expectedCount {
				t.Errorf("analyzeFile() returned %d detections, want %d", len(detections), tt.expectedCount)
			}

			if tt.expectedCount == 1 && tt.expectedService != "" {
				if detections[0].Name != tt.expectedService {
					t.Errorf("analyzeFile() detected service %s, want %s", detections[0].Name, tt.expectedService)
				}
			}
		})
	}
}

func TestAnalyzeProjectDependencies(t *testing.T) {
	// Create test services data
	servicesData := map[string]*ServiceData{
		"stripe": {
			Name: "Stripe",
			URL:  "https://dashboard.stripe.com",
			Stacks: map[string][]string{
				"ruby": {"stripe", "stripe_event"},
			},
		},
		"twilio": {
			Name: "Twilio",
			URL:  "https://console.twilio.com",
			Stacks: map[string][]string{
				"ruby": {"twilio-ruby", "twilio-rails"},
			},
		},
	}

	// Create test stack data
	stackData := &StackDependencyFiles{
		Languages: map[string]Language{
			"ruby": {
				PackageManagers: map[string]PackageManager{
					"bundler": {
						Files: []string{"Gemfile"},
					},
				},
			},
		},
	}

	// Create temporary project directory
	tmpDir, err := ioutil.TempDir("", "sitedog-test-project")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test Gemfile
	gemfileContent := `source 'https://rubygems.org'
gem 'rails', '~> 7.0'
gem 'stripe', '~> 5.0'
gem 'twilio-ruby', '~> 5.0'
gem 'pg', '~> 1.0'
`
	gemfilePath := filepath.Join(tmpDir, "Gemfile")
	err = ioutil.WriteFile(gemfilePath, []byte(gemfileContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test analyzeProjectDependencies
	results := analyzeProjectDependencies(tmpDir, []string{"ruby"}, stackData, servicesData)

	if len(results) != 1 {
		t.Fatalf("Expected 1 language result, got %d", len(results))
	}

	result := results[0]
	if result.Language != "ruby" {
		t.Errorf("Expected language 'ruby', got '%s'", result.Language)
	}

	if len(result.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(result.Services))
		for i, service := range result.Services {
			t.Logf("Service %d: %s", i, service.Name)
		}
	}

	// Check that both stripe and twilio are detected
	foundServices := make(map[string]bool)
	for _, service := range result.Services {
		foundServices[service.Name] = true
	}

	if !foundServices["stripe"] {
		t.Error("Expected to find stripe service")
	}
	if !foundServices["twilio"] {
		t.Error("Expected to find twilio service")
	}
}

func TestDetectProjectLanguages(t *testing.T) {
	stackData := &StackDependencyFiles{
		Languages: map[string]Language{
			"ruby": {
				PackageManagers: map[string]PackageManager{
					"bundler": {
						Files: []string{"Gemfile"},
					},
				},
			},
			"nodejs": {
				PackageManagers: map[string]PackageManager{
					"npm": {
						Files: []string{"package.json"},
					},
				},
			},
		},
	}

	tests := []struct {
		name             string
		files            map[string]string
		expectedLanguages []string
	}{
		{
			name: "Ruby project",
			files: map[string]string{
				"Gemfile": "gem 'rails'",
			},
			expectedLanguages: []string{"ruby"},
		},
		{
			name: "Node.js project",
			files: map[string]string{
				"package.json": `{"name": "test"}`,
			},
			expectedLanguages: []string{"nodejs"},
		},
		{
			name: "Multi-language project",
			files: map[string]string{
				"Gemfile":      "gem 'rails'",
				"package.json": `{"name": "test"}`,
			},
			expectedLanguages: []string{"nodejs", "ruby"},
		},
		{
			name:              "No recognized files",
			files:             map[string]string{"README.md": "# Test"},
			expectedLanguages: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := ioutil.TempDir("", "sitedog-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				err = ioutil.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Test detectProjectLanguages
			languages := detectProjectLanguages(tmpDir, stackData)

			if len(languages) != len(tt.expectedLanguages) {
				t.Errorf("Expected %d languages, got %d: %v", len(tt.expectedLanguages), len(languages), languages)
			}

			// Check that all expected languages are found
			foundLanguages := make(map[string]bool)
			for _, lang := range languages {
				foundLanguages[lang] = true
			}

			for _, expectedLang := range tt.expectedLanguages {
				if !foundLanguages[expectedLang] {
					t.Errorf("Expected to find language '%s', but it was not detected", expectedLang)
				}
			}
		})
	}
}