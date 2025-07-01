package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-shiori/obelisk"
	"gopkg.in/yaml.v2"

	"sitedog/detectors"
)

//go:embed data/stack-dependency-files.yml
var stackDependencyData []byte

//go:embed data/file-detectors.yml
var fileDetectorsData []byte

//go:embed data/services/*.yml
var servicesFS embed.FS

const (
	defaultConfigPath  = "./sitedog.yml"
	defaultTemplate    = "demo.html.tpl"
	defaultPort        = 8081
	globalTemplatePath = ".sitedog/demo.html.tpl"
	authFilePath       = ".sitedog/auth"
	apiBaseURL         = "https://app.sitedog.io" // Change to your actual API URL
	Version            = "v0.4.0"
	exampleConfig      = `# Describe your project with a free key-value format, think simple.
#
# Random sample:
registrar: gandi # registrar service
dns: Route 53 # dns service
hosting: https://carrd.com # hosting service
mail: zoho # mail service
`
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}
	switch os.Args[1] {
	case "init":
		handleInit()
	case "live":
		handleLive()
	case "push":
		handlePush()
	case "render":
		handleRender()
	case "sniff":
		handleSniff()
	case "logout":
		handleLogout()
	case "version":
		fmt.Println("sitedog version", Version)
	case "help":
		showHelp()
	default:
		fmt.Println("Unknown command:", os.Args[1])
		showHelp()
	}
}

func showHelp() {
	fmt.Println(`Usage: sitedog <command>

Commands:
  init    Create sitedog.yml configuration file
  live    Start live server with preview
  push    Push configuration to cloud
  render  Render template to HTML
  sniff   Detect technology stack and analyze dependencies
  logout  Remove authentication token
  version Print version
  help    Show this help message

Options for init:
  --config PATH    Path to config file (default: ./sitedog.yml)

Options for live:
  --config PATH    Path to config file (default: ./sitedog.yml)
  --port PORT      Port to run server on (default: 8081)

Options for push:
  --config PATH    Path to config file (default: ./sitedog.yml)
  --title TITLE    Configuration title (default: current directory name)
  --remote URL     Custom API base URL (e.g., localhost:3000, api.example.com)
  --namespace NAMESPACE Namespace for the configuration (e.g., my-group)
  SITEDOG_TOKEN    Environment variable for authentication token

Options for render:
  --config PATH    Path to config file (default: ./sitedog.yml)
  --output PATH    Path to output HTML file (default: sitedog.html)

Options for sniff:
  --path PATH      Path to analyze (default: current directory)

Examples:
  sitedog init --config my-config.yml
  sitedog live --port 3030
  sitedog push --title my-project
  sitedog push --remote localhost:3000 --title my-project
  sitedog push --remote api.example.com --title my-project
  sitedog push --remote https://api.example2.com --title my-project
  sitedog push --namespace my-group --title my-project
  SITEDOG_TOKEN=your_token sitedog push --title my-project
  sitedog render --output index.html
  sitedog sniff --path ./my-project
  sitedog logout`)
}

func handleInit() {
	configPath := flag.NewFlagSet("init", flag.ExitOnError)
	configFile := configPath.String("config", defaultConfigPath, "Path to config file")
	configPath.Parse(os.Args[2:])
	if _, err := os.Stat(*configFile); err == nil {
		fmt.Println("Error:", *configFile, "already exists")
		os.Exit(1)
	}
	if err := ioutil.WriteFile(*configFile, []byte(exampleConfig), 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created", *configFile, "configuration file")
}

func startServer(configFile *string, port int) (*http.Server, string) {
	// Handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		config, err := ioutil.ReadFile(*configFile)
		if err != nil {
			http.Error(w, "Error reading config", http.StatusInternalServerError)
			return
		}

		faviconCache := getFaviconCache(config)
		tmpl, _ := ioutil.ReadFile(findTemplate())
		tmpl = bytes.Replace(tmpl, []byte("{{CONFIG}}"), config, -1)
		tmpl = bytes.Replace(tmpl, []byte("{{FAVICON_CACHE}}"), faviconCache, -1)
		w.Header().Set("Content-Type", "text/html")
		w.Write(tmpl)
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		config, err := ioutil.ReadFile(*configFile)
		if err != nil {
			http.Error(w, "Error reading config", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/yaml")
		w.Write(config)
	})

	// Start the server
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr: addr,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	return server, addr
}

func handleLive() {
	liveFlags := flag.NewFlagSet("live", flag.ExitOnError)
	configFile := liveFlags.String("config", defaultConfigPath, "Path to config file")
	port := liveFlags.Int("port", defaultPort, "Port to run server on")
	liveFlags.Parse(os.Args[2:])

	if _, err := os.Stat(*configFile); err != nil {
		fmt.Println("Error:", *configFile, "not found. Run 'sitedog init' first.")
		os.Exit(1)
	}

	server, addr := startServer(configFile, *port)
	url := "http://localhost" + addr

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(url)
	}()
	fmt.Println("Starting live server at", url)
	fmt.Println("Press Ctrl+C to stop")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func findTemplate() string {
	local := filepath.Join(".", defaultTemplate)
	if _, err := os.Stat(local); err == nil {
		return local
	}
	usr, _ := user.Current()
	global := filepath.Join(usr.HomeDir, globalTemplatePath)
	if _, err := os.Stat(global); err == nil {
		return global
	}
	fmt.Println("Template not found.")
	os.Exit(1)
	return ""
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch {
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case strings.Contains(strings.ToLower(os.Getenv("OSTYPE")), "darwin"):
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}

func handlePush() {
	pushFlags := flag.NewFlagSet("push", flag.ExitOnError)
	configFile := pushFlags.String("config", defaultConfigPath, "Path to config file")
	configName := pushFlags.String("title", "", "Configuration title")
	remoteURL := pushFlags.String("remote", "", "Custom API base URL (e.g., localhost:3000, api.example.com)")
	namespace := pushFlags.String("namespace", "", "Namespace for the configuration (e.g., my-group)")
	pushFlags.Parse(os.Args[2:])

	if _, err := os.Stat(*configFile); err != nil {
		fmt.Println("Error:", *configFile, "not found. Run 'sitedog init' first.")
		os.Exit(1)
	}

	// Determine API base URL first
	apiURL := apiBaseURL
	if *remoteURL != "" {
		// Add protocol if not specified
		if !strings.HasPrefix(*remoteURL, "http://") && !strings.HasPrefix(*remoteURL, "https://") {
			apiURL = "http://" + *remoteURL
		} else {
			apiURL = *remoteURL
		}
	}

	// Get authorization token with correct API URL
	token, err := getAuthToken(apiURL)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Read configuration
	config, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	// Get configuration name from directory name if not specified
	if *configName == "" {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			os.Exit(1)
		}
		*configName = filepath.Base(dir)
	}

	// Send configuration to server
	err = pushConfig(token, *configName, string(config), apiURL, *namespace)
	if err != nil {
		fmt.Println("Error pushing config:", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration '%s' pushed successfully to %s!\n", *configName, apiURL)
}

func getAuthToken(apiURL string) (string, error) {
	// First check for environment variable
	if token := os.Getenv("SITEDOG_TOKEN"); token != "" {
		return strings.TrimSpace(token), nil
	}

	// Fall back to file-based authentication
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("error getting current user: %v", err)
	}

	authFile := filepath.Join(usr.HomeDir, authFilePath)
	if _, err := os.Stat(authFile); err == nil {
		// If file exists, read the token
		token, err := ioutil.ReadFile(authFile)
		if err != nil {
			return "", fmt.Errorf("error reading auth file: %v", err)
		}
		return strings.TrimSpace(string(token)), nil
	}

	// If not authenticated, start device authentication flow
	fmt.Println("Authentication required.")
	fmt.Printf("Please visit: %s/auth_device\n", apiURL)
	fmt.Println("Then enter the PIN code shown on the page.")

	// Loop for PIN code verification
	for {
		fmt.Print("PIN code: ")
		var pincode string
		fmt.Scanln(&pincode)

		// Validate PIN code with server
		token, err := validatePincode(apiURL, strings.TrimSpace(pincode))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("Please check the PIN code and try again.")
			continue
		}

		// Create .sitedog directory if it doesn't exist
		authDir := filepath.Dir(authFile)
		if err := os.MkdirAll(authDir, 0700); err != nil {
			return "", fmt.Errorf("error creating auth directory: %v", err)
		}

		// Save the token
		if err := ioutil.WriteFile(authFile, []byte(token), 0600); err != nil {
			return "", fmt.Errorf("error saving token: %v", err)
		}

		return token, nil
	}
}

// validatePincode sends PIN code to server for validation and returns token if valid
func validatePincode(apiURL, pincode string) (string, error) {
	// Create PIN validation request
	reqBody, err := json.Marshal(map[string]string{
		"pincode": pincode,
	})
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	resp, err := http.Post(apiURL+"/cli/validate_pincode", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body to get error details
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("PIN validation failed: %s (could not read error details)", resp.Status)
		}

		// Try to parse error response
		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error != "" {
			return "", fmt.Errorf("PIN validation failed: %s", errorResponse.Error)
		}

		// Fallback to raw body if JSON parsing fails
		return "", fmt.Errorf("PIN validation failed: %s", strings.TrimSpace(string(body)))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("no token received from server")
	}

	return result.Token, nil
}

func pushConfig(token, name, content, apiURL, namespace string) error {
	reqBody, err := json.Marshal(map[string]string{
		"name":      name,
		"content":   content,
		"namespace": namespace,
	})
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL+"/cli/push", strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body to get error details
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("push failed: %s (could not read error details)", resp.Status)
		}

		// Try to parse error response
		var errorResponse struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error != "" {
			return fmt.Errorf("push failed: %s - %s", resp.Status, errorResponse.Error)
		}

		// Fallback to raw body if JSON parsing fails
		return fmt.Errorf("push failed: %s - %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func spinner(stopSpinner chan bool, message string) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-stopSpinner:
			fmt.Print("\r")
			return
		default:
			fmt.Printf("\r%s %s", spinner[i], message)
			i = (i + 1) % len(spinner)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func handleRender() {
	// Start loading indicator
	stopSpinner := make(chan bool)
	go spinner(stopSpinner, "Rendering...")

	renderFlags := flag.NewFlagSet("render", flag.ExitOnError)
	configFile := renderFlags.String("config", defaultConfigPath, "Path to config file")
	outputFile := renderFlags.String("output", "sitedog.html", "Path to output HTML file")
	renderFlags.Parse(os.Args[2:])

	if _, err := os.Stat(*configFile); err != nil {
		stopSpinner <- true
		fmt.Println("Error:", *configFile, "not found. Run 'sitedog init' first.")
		os.Exit(1)
	}

	port := 34324
	server, addr := startServer(configFile, port)
	url := "http://localhost" + addr

	// Check server availability
	resp, err := http.Get(url)
	if err != nil {
		stopSpinner <- true
		fmt.Println("Error checking server:", err)
		server.Close()
		os.Exit(1)
	}
	resp.Body.Close()

	// Use Obelisk to save the page
	archiver := &obelisk.Archiver{
		EnableLog:             false,
		MaxConcurrentDownload: 10,
	}

	// Validate archiver
	archiver.Validate()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create request
	req := obelisk.Request{
		URL: url,
	}

	// Save the page
	html, _, err := archiver.Archive(ctx, req)
	if err != nil {
		stopSpinner <- true
		fmt.Println("\nError archiving page:", err)
		server.Close()
		os.Exit(1)
	}

	// Save result to file
	if err := ioutil.WriteFile(*outputFile, html, 0644); err != nil {
		stopSpinner <- true
		fmt.Println("Error saving file:", err)
		server.Close()
		os.Exit(1)
	}

	// Stop loading indicator
	stopSpinner <- true

	// Close server
	server.Close()

	fmt.Printf("Rendered cards saved to %s\n", *outputFile)
}

func getFaviconCache(config []byte) []byte {
	// Parse YAML config
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(config, &configMap); err != nil {
		return []byte("{}")
	}

	// Create map for storing favicon cache
	faviconCache := make(map[string]string)

	// Function to extract domain from URL
	extractDomain := func(urlStr string) string {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return ""
		}
		return parsedURL.Hostname()
	}

	// Function for recursive value traversal
	var traverseValue func(value interface{})
	traverseValue = func(value interface{}) {
		switch v := value.(type) {
		case string:
			// Check if string is a URL
			if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
				domain := extractDomain(v)
				if domain != "" {
					// Get favicon
					faviconURL := fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", url.QueryEscape(domain))
					resp, err := http.Get(faviconURL)
					if err != nil {
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						// Read favicon
						faviconData, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							return
						}

						// Convert to base64
						base64Data := base64.StdEncoding.EncodeToString(faviconData)
						// Get content type from response headers
						contentType := resp.Header.Get("Content-Type")
						if contentType == "" {
							contentType = "image/png" // fallback to png if no content type specified
						}
						dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
						faviconCache[v] = dataURL
					}
				}
			}
		case map[interface{}]interface{}:
			for _, val := range v {
				traverseValue(val)
			}
		case []interface{}:
			for _, val := range v {
				traverseValue(val)
			}
		case map[string]interface{}:
			for _, val := range v {
				traverseValue(val)
			}
		}
	}

	// Traverse all values in config
	traverseValue(configMap)

	// Convert map to JSON
	jsonData, err := json.Marshal(faviconCache)
	if err != nil {
		return []byte("{}")
	}

	return jsonData
}

// Data structures for working with dependency analysis

type StackDependencyFiles struct {
	Languages map[string]Language `yaml:"languages"`
}

type Language struct {
	API             API                           `yaml:"api"`
	PackageManagers map[string]PackageManager `yaml:"package_managers"`
}

type API struct {
	CheckURL     string  `yaml:"check_url"`
	DelaySeconds float64 `yaml:"delay_seconds"`
}

type PackageManager struct {
	Files []string `yaml:"files"`
}

type ServiceData struct {
	Name   string                    `yaml:"name"`
	URL    string                    `yaml:"url"`
	Stacks map[string][]string `yaml:"stacks"`
}

type DetectionResult struct {
	Language     string
	Files        []string
	Services     []ServiceDetection
}

type ServiceDetection struct {
	Name     string
	Language string
	Packages []PackageInfo
}

type PackageInfo struct {
	Name string
	File string
}

func handleSniff() {
	sniffFlags := flag.NewFlagSet("sniff", flag.ExitOnError)
	projectPath := sniffFlags.String("path", ".", "Path to analyze")
	sniffFlags.Parse(os.Args[2:])

	displayPath := *projectPath
	if *projectPath == "." {
		if cwd, err := os.Getwd(); err == nil {
			displayPath = "current directory (" + filepath.Base(cwd) + ")"
		} else {
			displayPath = "current directory"
		}
	}
	fmt.Printf("🔍 Analyzing project in %s...\n\n", displayPath)

	// Load stack dependency files data
	stackData, err := loadStackDependencyFiles()
	if err != nil {
		fmt.Printf("❌ Error loading stack data: %v\n", err)
		return
	}

	// Load services data
	servicesData, err := loadServicesData()
	if err != nil {
		fmt.Printf("❌ Error loading services data: %v\n", err)
		return
	}

	// Load file detectors data
	fileDetectorsData, err := loadFileDetectorsData()
	if err != nil {
		fmt.Printf("❌ Error loading file detectors data: %v\n", err)
		return
	}

		// Create detectors in two phases:
	// Phase 1: Simple detectors (don't need context)
	var phase1Detectors []detectors.Detector

	// Create adapter for services dependencies
	adapter := &ServicesDependenciesAdapter{
		stackData:    stackData,
		servicesData: servicesData,
	}

	// Add Services detector (simple)
	servicesDetector := detectors.NewServicesDetector(adapter)
	phase1Detectors = append(phase1Detectors, detectors.NewSimpleDetectorAdapter(servicesDetector))

	// Add Git detector (simple)
	gitDetector := &detectors.GitRepositoryDetector{}
	phase1Detectors = append(phase1Detectors, detectors.NewSimpleDetectorAdapter(gitDetector))

	// Phase 2: Context-aware detectors
	var phase2Detectors []detectors.Detector

	// Add Files detector (needs context for URL building)
	filesDetector := detectors.NewFilesDetector(fileDetectorsData)
	phase2Detectors = append(phase2Detectors, filesDetector)

	// Run phase 1 detectors
	allResults := make(map[string]string)
	ctx := &detectors.DetectionContext{
		ProjectPath: *projectPath,
		Results:     make(map[string]string),
	}

	for _, detector := range phase1Detectors {
		results, err := detector.Detect(ctx)
		if err != nil {
			fmt.Printf("❌ Error running %s detector: %v\n", detector.Name(), err)
			continue
		}

		// Merge results
		for key, value := range results {
			allResults[key] = value
			ctx.Results[key] = value // Update context for next phase
		}
	}

	// Run phase 2 detectors with context
	for _, detector := range phase2Detectors {
		results, err := detector.Detect(ctx)
		if err != nil {
			fmt.Printf("❌ Error running %s detector: %v\n", detector.Name(), err)
			continue
		}

		// Merge results
		for key, value := range results {
			allResults[key] = value
		}
	}

	// Show language detection for user feedback (keep existing behavior)
	detectedLanguages := detectProjectLanguages(*projectPath, stackData)
	if len(detectedLanguages) > 0 {
		if len(detectedLanguages) == 1 {
			fmt.Printf("👃 Smells like %s in here!\n", strings.Title(detectedLanguages[0]))
		} else {
			var titleLanguages []string
			for _, lang := range detectedLanguages {
				titleLanguages = append(titleLanguages, strings.Title(lang))
			}
			fmt.Printf("👃 Smells like a mix of %s!\n", strings.Join(titleLanguages, ", "))
		}
		fmt.Println()
	}

	// Display results
	displayDetectorResults(allResults)

	// Create configuration
	configPath := filepath.Join(*projectPath, "sitedog.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		createConfigFromDetectorResults(configPath, allResults)
	}
}

func loadStackDependencyFiles() (*StackDependencyFiles, error) {
	var stackData StackDependencyFiles
	err := yaml.Unmarshal(stackDependencyData, &stackData)
	if err != nil {
		return nil, err
	}

	return &stackData, nil
}

func loadServicesData() (map[string]*ServiceData, error) {
	servicesData := make(map[string]*ServiceData)

	entries, err := servicesFS.ReadDir("data/services")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yml") {
			data, err := servicesFS.ReadFile("data/services/" + entry.Name())
			if err != nil {
				continue
			}

			var service ServiceData
			err = yaml.Unmarshal(data, &service)
			if err != nil {
				continue
			}

			serviceName := entry.Name()[:len(entry.Name())-4] // remove .yml extension
			servicesData[serviceName] = &service
		}
	}

	return servicesData, nil
}

func loadFileDetectorsData() (*detectors.FileDetectors, error) {
	var fileData detectors.FileDetectors
	err := yaml.Unmarshal(fileDetectorsData, &fileData)
	if err != nil {
		return nil, err
	}

	return &fileData, nil
}

func detectProjectLanguages(projectPath string, stackData *StackDependencyFiles) []string {
	var technologies []string

	for tech, lang := range stackData.Languages {
		found := false
		for _, pm := range lang.PackageManagers {
			for _, filePattern := range pm.Files {
				if hasMatchingFiles(projectPath, filePattern) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			technologies = append(technologies, tech)
		}
	}

	return technologies
}

func hasMatchingFiles(dir, pattern string) bool {
	// If pattern contains subdirectories (e.g. "requirements/*.txt")
	if strings.Contains(pattern, "/") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		return err == nil && len(matches) > 0
	}

	// If pattern contains wildcards (e.g. "*.txt")
	if strings.Contains(pattern, "*") {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		return err == nil && len(matches) > 0
	}

	// Regular file
	_, err := os.Stat(filepath.Join(dir, pattern))
	return err == nil
}

func analyzeProjectDependencies(projectPath string, languages []string, stackData *StackDependencyFiles, servicesData map[string]*ServiceData) []DetectionResult {
	var results []DetectionResult

	for _, language := range languages {
		langData := stackData.Languages[language]
		foundFilesMap := make(map[string]bool)
		servicesMap := make(map[string]*ServiceDetection)

		// Collect all dependency files for this language (without duplicates)
		for _, packageManager := range langData.PackageManagers {
			for _, filePattern := range packageManager.Files {
				matches, err := filepath.Glob(filepath.Join(projectPath, filePattern))
				if err != nil {
					continue
				}
				for _, match := range matches {
					foundFilesMap[match] = true
				}
			}
		}

		// Convert map to slice
		var foundFiles []string
		for file := range foundFilesMap {
			foundFiles = append(foundFiles, file)
		}

		// Analyze found files only once each
		analyzedFiles := make(map[string]bool)
		for _, file := range foundFiles {
			if !analyzedFiles[file] {
				analyzedFiles[file] = true
				fileServices := analyzeFile(file, language, servicesData)
				for _, service := range fileServices {
					if existing, exists := servicesMap[service.Name]; exists {
						// Merge packages, avoiding duplicates
						packageMap := make(map[string]PackageInfo)
						for _, pkg := range existing.Packages {
							packageMap[pkg.Name] = pkg
						}
						for _, pkg := range service.Packages {
							packageMap[pkg.Name] = pkg
						}

						var mergedPackages []PackageInfo
						for _, pkg := range packageMap {
							mergedPackages = append(mergedPackages, pkg)
						}
						existing.Packages = mergedPackages
					} else {
						servicesMap[service.Name] = &service
					}
				}
			}
		}

		// Convert map to slice
		var services []ServiceDetection
		for _, service := range servicesMap {
			services = append(services, *service)
		}

		if len(foundFiles) > 0 || len(services) > 0 {
			result := DetectionResult{
				Language: language,
				Files:    foundFiles,
				Services: services,
			}
			results = append(results, result)
		}
	}

	return results
}

func analyzeFile(filePath, language string, servicesData map[string]*ServiceData) []ServiceDetection {
	var detections []ServiceDetection

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return detections
	}

	fileName := filepath.Base(filePath)

	for serviceName, serviceData := range servicesData {
		if packages, exists := serviceData.Stacks[language]; exists {
			var foundPackages []PackageInfo

			for _, pkg := range packages {
				if isPackageInFile(string(content), fileName, pkg, language) {
					foundPackages = append(foundPackages, PackageInfo{
						Name: pkg,
						File: filePath,
					})
				}
			}

			if len(foundPackages) > 0 {
				detection := ServiceDetection{
					Name:     serviceName,
					Language: language,
					Packages: foundPackages,
				}
				detections = append(detections, detection)
			}
		}
	}

	return detections
}

// Simple and efficient package search
func isPackageInFile(content, fileName, packageName, language string) bool {
	// Simply search for the package as is - names are unique
	return strings.Contains(content, packageName)
}

// Create sitedog.yml configuration based on detected technologies and services
func createConfigFromDetection(configPath string, languages []string, results []DetectionResult, servicesData map[string]*ServiceData) {
	var config strings.Builder

	// Get project name from directory
	projectDir := filepath.Dir(configPath)
	projectName := filepath.Base(projectDir)
	if projectDir == "." {
		if cwd, err := os.Getwd(); err == nil {
			projectName = filepath.Base(cwd)
		}
	}

	// Add project name as root key
	config.WriteString(fmt.Sprintf("%s:\n", projectName))

	// Add detected services with their URLs
	allServices := make(map[string]bool)
	for _, result := range results {
		for _, service := range result.Services {
			allServices[service.Name] = true
		}
	}

	if len(allServices) > 0 {
		for serviceKey := range allServices {
			// Get name and URL from servicesData
			displayName := serviceKey // fallback to file key
			serviceURL := serviceKey  // fallback
			if serviceData, exists := servicesData[serviceKey]; exists {
				displayName = serviceData.Name // use name from YAML
				if serviceData.URL != "" {
					serviceURL = serviceData.URL
				}
			}
			config.WriteString(fmt.Sprintf("  %s: %s\n", displayName, serviceURL))
		}
	}

	if err := os.WriteFile(configPath, []byte(config.String()), 0644); err != nil {
		fmt.Printf("⚠️  Could not create %s: %v\n", configPath, err)
		return
	}

	fmt.Printf("\n✨ Created %s with detected services\n", configPath)
}

func displayResults(results []DetectionResult, servicesData map[string]*ServiceData) {
	// Collect all unique services across all languages
	allServices := make(map[string]bool)

	for _, result := range results {
		for _, service := range result.Services {
			allServices[service.Name] = true
		}
	}

	if len(allServices) == 0 {
		fmt.Println("🔍 No services detected")
		return
	}

	fmt.Printf("🔍 Detected %d service(s):\n", len(allServices))
	for serviceName := range allServices {
		if serviceData, exists := servicesData[serviceName]; exists {
			if serviceData.URL != "" {
				fmt.Printf("  🔗 %s → %s\n", serviceData.Name, serviceData.URL)
			} else {
				fmt.Printf("  🔗 %s\n", serviceData.Name)
			}
		} else {
			fmt.Printf("  🔗 %s\n", strings.Title(serviceName))
		}
	}
}

func handleLogout() {
	// Get current user to find auth file path
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		os.Exit(1)
	}

	authFile := filepath.Join(usr.HomeDir, authFilePath)

	// Check if auth file exists
	if _, err := os.Stat(authFile); err != nil {
		fmt.Println("No authentication token found. You are already logged out.")
		return
	}

	// Remove the auth file
	if err := os.Remove(authFile); err != nil {
		fmt.Println("Error removing authentication token:", err)
		os.Exit(1)
	}

	fmt.Println("Successfully logged out. Authentication token removed.")
}

func displayDetectorResults(results map[string]string) {
	if len(results) == 0 {
		fmt.Println("🔍 No services or repositories detected")
		return
	}

	serviceCount := len(results)
	// Don't count 'repo' as a service
	if _, hasRepo := results["repo"]; hasRepo {
		serviceCount--
	}

	if serviceCount > 0 {
		fmt.Printf("🔍 Detected %d service(s):\n", serviceCount)
		for key, value := range results {
			if key != "repo" { // Skip repo for services display
				displayName := getTechnologyDisplayName(key, value)
				fmt.Printf("  🔗 %s → %s\n", displayName, value)
			}
		}
	}

	if repo, hasRepo := results["repo"]; hasRepo {
		fmt.Printf("📁 Repository: %s\n", repo)
	}
}

func getTechnologyDisplayName(techKey, url string) string {
	// Try to load file detectors config to get display names
	fileDetectors, err := loadFileDetectorsData()
	if err == nil {
		if techConfig, exists := fileDetectors.Technologies[techKey]; exists {
			if techConfig.DisplayName != "" {
				return techConfig.DisplayName
			}
		}
	}

	// Special case for repository
	if techKey == "repo" {
		return "Repository"
	}

	// Fallback: convert key to title case
	return strings.Title(techKey)
}

func createConfigFromDetectorResults(configPath string, results map[string]string) {
	var config strings.Builder

	// Get project name from directory
	projectDir := filepath.Dir(configPath)
	projectName := filepath.Base(projectDir)
	if projectDir == "." {
		if cwd, err := os.Getwd(); err == nil {
			projectName = filepath.Base(cwd)
		}
	}

	// Add project name as root key
	config.WriteString(fmt.Sprintf("%s:\n", projectName))

	// Add all detected key-value pairs with display names
	for key, value := range results {
		if key == "repo" {
			config.WriteString(fmt.Sprintf("  Repository: %s\n", value))
		} else {
			displayName := getTechnologyDisplayName(key, value)
			config.WriteString(fmt.Sprintf("  %s: %s\n", displayName, value))
		}
	}

	if err := os.WriteFile(configPath, []byte(config.String()), 0644); err != nil {
		fmt.Printf("⚠️  Could not create %s: %v\n", configPath, err)
		return
	}

	fmt.Printf("\n✨ Created %s with detected services\n", configPath)
}

// ServicesDependenciesAdapter adapts existing functions to detectors interface
type ServicesDependenciesAdapter struct {
	stackData    *StackDependencyFiles
	servicesData map[string]*ServiceData
}

func (a *ServicesDependenciesAdapter) DetectProjectLanguages(projectPath string) []string {
	return detectProjectLanguages(projectPath, a.stackData)
}

func (a *ServicesDependenciesAdapter) AnalyzeProjectDependencies(projectPath string, languages []string) []detectors.ProjectResult {
	results := analyzeProjectDependencies(projectPath, languages, a.stackData, a.servicesData)

	// Convert to detectors format
	var detectorResults []detectors.ProjectResult
	for _, result := range results {
		var services []detectors.ServiceResult
		for _, service := range result.Services {
			services = append(services, detectors.ServiceResult{
				Name: service.Name,
			})
		}
		detectorResults = append(detectorResults, detectors.ProjectResult{
			Language: result.Language,
			Services: services,
		})
	}

	return detectorResults
}

func (a *ServicesDependenciesAdapter) GetServicesData() map[string]*detectors.ServiceInfo {
	result := make(map[string]*detectors.ServiceInfo)
	for key, service := range a.servicesData {
		result[key] = &detectors.ServiceInfo{
			Name: service.Name,
			URL:  service.URL,
		}
	}
	return result
}
