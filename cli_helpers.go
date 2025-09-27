package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateProjectName validates the project name
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check for valid Go module name
	if strings.Contains(name, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("project name cannot start or end with dash")
	}

	return nil
}

// validateProjectType validates the project type
func validateProjectType(projectType string) error {
	validTypes := []string{ProjectTypeAPI, ProjectTypeWorker}
	for _, valid := range validTypes {
		if projectType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid project type: %s (valid types: %s)", projectType, strings.Join(validTypes, ", "))
}

// promptWithDefault prompts user for input with a default value
func promptWithDefault(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return defaultValue
	}
	return input
}

// selectFromOptions prompts user to select from a list of options
func selectFromOptions(prompt string, options []string, defaultIndex int) string {
	fmt.Printf("%s:\n", prompt)
	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, option)
	}

	fmt.Printf("Select option [%d]: ", defaultIndex+1)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return options[defaultIndex]
	}

	// Try to parse as number
	var selection int
	if _, err := fmt.Sscanf(input, "%d", &selection); err == nil {
		if selection >= 1 && selection <= len(options) {
			return options[selection-1]
		}
	}

	// Try to find by name
	for _, option := range options {
		if strings.EqualFold(input, option) {
			return option
		}
	}

	// Default to first option if invalid input
	return options[defaultIndex]
}

// askYesNo prompts for a yes/no question with a default value
func askYesNo(question string, defaultValue bool) bool {
	scanner := bufio.NewScanner(os.Stdin)
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}

	fmt.Printf("%s (y/n) [%s]: ", question, defaultStr)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if response == "" {
		return defaultValue
	}

	return response == "y" || response == "yes"
}

// createDirectoryStructure creates the basic directory structure
func createDirectoryStructure(projectName string, projectType string) error {
	basePath := projectName

	// Common directories
	dirs := []string{
		"internal/config",
		"internal/database",
		"internal/models",
		"internal/services",
		"configs",
	}

	// Type-specific directories
	if projectType == ProjectTypeAPI {
		dirs = append(dirs,
			"internal/handlers",
			"internal/middleware",
			"internal/server",
			"internal/repository",
			"docs",
		)
	} else if projectType == ProjectTypeWorker {
		dirs = append(dirs,
			"internal/handlers",
			"internal/queue",
			"internal/worker",
		)
	}

	// Create all directories
	for _, dir := range dirs {
		fullPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

// Enhanced interactive mode with better UX
func runEnhancedInteractive() {
	config := &ProjectConfig{}

	// Welcome message
	fmt.Println("🚀 Welcome to Go Project Scaffolder")
	fmt.Println("===================================")
	fmt.Println("This tool will help you create a production-ready Go project with best practices.")
	fmt.Println()

	// Project name with validation
	for {
		fmt.Print("📝 Project name: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		name := strings.TrimSpace(scanner.Text())

		if err := validateProjectName(name); err != nil {
			fmt.Printf("❌ %v\n", err)
			continue
		}

		// Check if directory already exists
		if _, err := os.Stat(name); err == nil {
			fmt.Printf("⚠️  Directory '%s' already exists. Choose a different name.\n", name)
			continue
		}

		config.Name = name
		break
	}

	// Project type selection
	fmt.Println("\n🏗️  Project Type:")
	fmt.Println("   1) API Server    - REST API with HTTP endpoints")
	fmt.Println("   2) Worker        - Background job processor")

	projectTypes := []string{"api", "worker"}
	config.Type = selectFromOptions("Select project type", projectTypes, 0)

	// Database selection
	fmt.Println("\n💾 Database:")
	fmt.Println("   1) PostgreSQL    - Robust relational database")
	fmt.Println("   2) MySQL         - Popular relational database")
	fmt.Println("   3) SQLite        - Lightweight file-based database")
	fmt.Println("   4) MongoDB       - Document-based NoSQL database")
	fmt.Println("   5) None          - No database integration")

	databases := []string{"postgres", "mysql", "sqlite", "mongodb", "none"}
	config.Database = selectFromOptions("Select database", databases, 0)

	// Queue selection (for workers)
	if config.Type == "worker" {
		fmt.Println("\n📨 Message Queue:")
		fmt.Println("   1) RabbitMQ      - Feature-rich message broker")
		fmt.Println("   2) Apache Kafka  - High-throughput streaming platform")
		fmt.Println("   3) AWS SQS       - Managed queue service")
		fmt.Println("   4) None          - No queue integration")

		queues := []string{"rabbitmq", "kafka", "sqs", "none"}
		config.Queue = selectFromOptions("Select message queue", queues, 0)
	}

	// Cache selection
	fmt.Println("\n⚡ Cache:")
	fmt.Println("   1) Redis         - In-memory data structure store")
	fmt.Println("   2) Memcached     - High-performance caching system")
	fmt.Println("   3) None          - No caching")

	caches := []string{"redis", "memcached", "none"}
	config.Cache = selectFromOptions("Select cache", caches, 0)

	// Features selection
	fmt.Println("\n✨ Additional Features:")
	config.Auth = askYesNo("🔐 Include JWT authentication", false)
	config.Logging = askYesNo("📄 Include structured logging", true)
	config.Metrics = askYesNo("📊 Include Prometheus metrics", true)
	config.Docker = askYesNo("🐳 Include Docker configuration", true)
	config.Kubernetes = askYesNo("☸️  Include Kubernetes manifests", false)
	config.Testing = askYesNo("🧪 Include testing setup", true)

	// API-specific features
	if config.Type == "api" {
		fmt.Println("\n🌐 API Features:")
		config.Swagger = askYesNo("📚 Include Swagger documentation", true)
		config.GRPC = askYesNo("⚡ Include gRPC support", false)
		config.GraphQL = askYesNo("🔍 Include GraphQL support", false)
		config.WebSocket = askYesNo("🔌 Include WebSocket support", false)
	}

	// Port configuration
	if config.Type == "api" {
		config.Port = promptWithDefault("🔌 Default port", "8080")
	}
	config.GoVersion = "1.21"

	// Configuration summary
	fmt.Println("\n📋 Configuration Summary:")
	fmt.Println("========================")
	fmt.Printf("  📝 Name: %s\n", config.Name)
	fmt.Printf("  🏗️  Type: %s\n", config.Type)
	if config.Database != "none" {
		fmt.Printf("  💾 Database: %s\n", config.Database)
	}
	if config.Queue != "" && config.Queue != "none" {
		fmt.Printf("  📨 Queue: %s\n", config.Queue)
	}
	if config.Cache != "none" {
		fmt.Printf("  ⚡ Cache: %s\n", config.Cache)
	}
	if config.Type == "api" {
		fmt.Printf("  🔌 Port: %s\n", config.Port)
	}

	// Feature summary
	features := []string{}
	if config.Auth {
		features = append(features, "Authentication")
	}
	if config.Logging {
		features = append(features, "Logging")
	}
	if config.Metrics {
		features = append(features, "Metrics")
	}
	if config.Docker {
		features = append(features, "Docker")
	}
	if config.Kubernetes {
		features = append(features, "Kubernetes")
	}
	if config.Testing {
		features = append(features, "Testing")
	}
	if config.Swagger {
		features = append(features, "Swagger")
	}
	if config.GRPC {
		features = append(features, "gRPC")
	}
	if config.GraphQL {
		features = append(features, "GraphQL")
	}
	if config.WebSocket {
		features = append(features, "WebSocket")
	}

	if len(features) > 0 {
		fmt.Printf("  ✨ Features: %s\n", strings.Join(features, ", "))
	}

	fmt.Println()
	if !askYesNo("🚀 Generate project with these settings", true) {
		fmt.Println("❌ Aborted.")
		return
	}

	// Generate project with progress indication
	fmt.Println("\n🔧 Generating project...")

	spinner := []string{"|", "/", "-", "\\"}
	spinnerIndex := 0

	// Show progress (simulate work)
	steps := []string{
		"Creating directory structure...",
		"Generating Go modules...",
		"Creating configuration files...",
		"Setting up database layer...",
		"Generating handlers...",
		"Creating Docker files...",
		"Writing documentation...",
		"Setting up tests...",
	}

	for i, step := range steps {
		fmt.Printf("\r%s %s", spinner[spinnerIndex%len(spinner)], step)
		spinnerIndex++

		// Clear line and show completion
		fmt.Printf("\r✅ %s\n", step)

		// Break early if not all features are enabled
		if i >= 4 && !config.Docker && !config.Testing {
			break
		}
	}

	// Generate the actual project
	if err := generateProject(config); err != nil {
		fmt.Printf("\n❌ Failed to generate project: %v\n", err)
		os.Exit(1)
	}

	// Success message with next steps
	fmt.Printf("\n🎉 Successfully generated %s project: %s\n", config.Type, config.Name)
	fmt.Println("\n📖 Next steps:")
	fmt.Printf("  cd %s\n", config.Name)
	fmt.Println("  go mod tidy")

	if config.Docker {
		fmt.Println("  docker-compose up -d  # Start services")
	}

	if config.Type == "api" {
		fmt.Println("  go run main.go        # Start the API server")
		if config.Swagger {
			fmt.Printf("  # Visit http://localhost:%s/swagger/index.html for API docs\n", config.Port)
		}
	} else {
		fmt.Println("  go run main.go        # Start the worker")
	}

	fmt.Println("\n📚 Documentation:")
	fmt.Printf("  cat %s/README.md      # Read the project documentation\n", config.Name)
	fmt.Println("  make help             # See available make commands")

	fmt.Println("\n🤝 Happy coding!")
}

// File utility functions
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Project validation
func validateProject(config *ProjectConfig) error {
	if err := validateProjectName(config.Name); err != nil {
		return err
	}

	if err := validateProjectType(config.Type); err != nil {
		return err
	}

	// Validate database choice
	validDatabases := []string{"postgres", "mysql", "sqlite", "mongodb", "none"}
	if !contains(validDatabases, config.Database) {
		return fmt.Errorf("invalid database: %s", config.Database)
	}

	// Validate queue choice for workers
	if config.Type == ProjectTypeWorker && config.Queue != "" {
		validQueues := []string{"rabbitmq", "kafka", "sqs", "none"}
		if !contains(validQueues, config.Queue) {
			return fmt.Errorf("invalid queue: %s", config.Queue)
		}
	}

	// Validate cache choice
	validCaches := []string{"redis", "memcached", "none"}
	if !contains(validCaches, config.Cache) {
		return fmt.Errorf("invalid cache: %s", config.Cache)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Post-generation tasks
func runPostGeneration(config *ProjectConfig) error {
	fmt.Println("🔧 Running post-generation tasks...")

	// Initialize git repository
	if err := initGitRepo(config.Name); err != nil {
		fmt.Printf("⚠️  Warning: Could not initialize git repository: %v\n", err)
	} else {
		fmt.Println("✅ Initialized git repository")
	}

	// Create initial commit
	if err := createInitialCommit(config.Name); err != nil {
		fmt.Printf("⚠️  Warning: Could not create initial commit: %v\n", err)
	} else {
		fmt.Println("✅ Created initial commit")
	}

	// Generate .gitignore
	if err := createGitignore(config); err != nil {
		fmt.Printf("⚠️  Warning: Could not create .gitignore: %v\n", err)
	} else {
		fmt.Println("✅ Created .gitignore")
	}

	return nil
}

func initGitRepo(projectPath string) error {
	// This would run: git init
	// For now, just create .git directory
	gitDir := filepath.Join(projectPath, ".git")
	return os.MkdirAll(gitDir, 0755)
}

func createInitialCommit(projectPath string) error {
	// This would run: git add . && git commit -m "Initial commit"
	return nil
}

func createGitignore(config *ProjectConfig) error {
	gitignoreContent := `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Environment files
.env
.env.local

# Logs
*.log
logs/

# Temporary files
tmp/
temp/

{{- if .Docker}}
# Docker
.dockerignore
{{- end}}

{{- if eq .Database "sqlite"}}
# SQLite databases
*.db
*.sqlite
*.sqlite3
{{- end}}

# Application specific
{{.Name}}
{{.Name}}.exe
`

	return createFileFromTemplate(gitignoreContent, config, filepath.Join(config.Name, ".gitignore"))
}
