package main

import (
	"bytes"
	"context"
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
)

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
