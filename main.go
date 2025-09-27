package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const (
	ProjectTypeAPI    = "api"
	ProjectTypeWorker = "worker"
)

type ProjectConfig struct {
	Name       string
	Type       string
	Database   string
	Cache      string
	Queue      string
	Auth       bool
	Logging    bool
	Metrics    bool
	Docker     bool
	Kubernetes bool
	Testing    bool
	Swagger    bool
	GRPC       bool
	GraphQL    bool
	WebSocket  bool
	Middleware []string
	Port       string
	GoVersion  string
}

var rootCmd = &cobra.Command{
	Use:   "go-scaffolder",
	Short: "A powerful Go project scaffolding tool",
	Long:  `Generate production-ready Go APIs and Workers with best practices built-in.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new Go project",
	Run:   runGenerate,
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Interactive project generation",
	Run:   runInteractive,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Go Scaffolder v1.0.0")
	},
}

var listTemplatesCmd = &cobra.Command{
	Use:   "list-templates",
	Short: "List available project templates",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available project templates:")
		fmt.Println("  api    - REST API with Gin framework")
		fmt.Println("  worker - Background worker with message queues")
		fmt.Println()
		fmt.Println("Database options:")
		fmt.Println("  postgres  - PostgreSQL database")
		fmt.Println("  mysql     - MySQL database")
		fmt.Println("  sqlite    - SQLite database")
		fmt.Println("  mongodb   - MongoDB database")
		fmt.Println()
		fmt.Println("Queue options (for workers):")
		fmt.Println("  rabbitmq  - RabbitMQ message broker")
		fmt.Println("  kafka     - Apache Kafka")
		fmt.Println("  sqs       - AWS SQS")
		fmt.Println()
		fmt.Println("Cache options:")
		fmt.Println("  redis     - Redis cache")
		fmt.Println("  memcached - Memcached")
		fmt.Println("  none      - No caching")
	},
}

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new project in current directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		if err := validateProjectName(projectName); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Check if current directory is empty
		entries, err := os.ReadDir(".")
		if err != nil {
			fmt.Printf("Error reading current directory: %v\n", err)
			os.Exit(1)
		}

		if len(entries) > 0 {
			fmt.Println("Current directory is not empty. Continue? (y/N)")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if !strings.EqualFold(scanner.Text(), "y") {
				fmt.Println("Aborted.")
				return
			}
		}

		// Interactive configuration
		config := &ProjectConfig{Name: projectName}

		// Project type
		projectTypes := []string{"api", "worker"}
		config.Type = selectFromOptions("Select project type", projectTypes, 0)

		// Database
		databases := []string{"postgres", "mysql", "sqlite", "mongodb", "none"}
		config.Database = selectFromOptions("Select database", databases, 0)

		// Queue
		if config.Type == "worker" {
			queues := []string{"rabbitmq", "kafka", "sqs", "none"}
			config.Queue = selectFromOptions("Select message queue", queues, 0)
		}

		// Cache
		caches := []string{"redis", "memcached", "none"}
		config.Cache = selectFromOptions("Select cache", caches, 0)

		// Additional features
		config.Auth = askYesNo("Include authentication?", false)
		config.Logging = askYesNo("Include structured logging?", true)
		config.Metrics = askYesNo("Include metrics collection?", true)
		config.Docker = askYesNo("Include Docker configuration?", true)
		config.Kubernetes = askYesNo("Include Kubernetes manifests?", false)
		config.Testing = askYesNo("Include testing setup?", true)

		if config.Type == "api" {
			config.Swagger = askYesNo("Include Swagger documentation?", false)
			config.GRPC = askYesNo("Include gRPC support?", false)
			config.GraphQL = askYesNo("Include GraphQL support?", false)
			config.WebSocket = askYesNo("Include WebSocket support?", false)
		}

		// Port
		config.Port = promptWithDefault("Default port", "8080")
		config.GoVersion = "1.21"

		// Generate in current directory
		originalName := config.Name
		config.Name = "."

		if err := generateProject(config); err != nil {
			fmt.Printf("Failed to generate project: %v\n", err)
			os.Exit(1)
		}

		// Restore original name for messages
		config.Name = originalName

		fmt.Printf("\n🎉 Successfully initialized %s project in current directory\n", config.Type)
		fmt.Println("\n📖 Next steps:")
		fmt.Println("  go mod tidy")
		fmt.Println("  go run main.go")
	},
}

func init() {
	generateCmd.Flags().StringP("name", "n", "", "Project name")
	generateCmd.Flags().StringP("type", "t", "api", "Project type (api/worker)")
	generateCmd.Flags().StringP("database", "d", "postgres", "Database type (postgres/mysql/mongodb/sqlite)")
	generateCmd.Flags().StringP("cache", "c", "redis", "Cache type (redis/memcached/none)")
	generateCmd.Flags().StringP("queue", "q", "rabbitmq", "Queue type (rabbitmq/kafka/sqs/none)")
	generateCmd.Flags().BoolP("auth", "a", false, "Include authentication")
	generateCmd.Flags().BoolP("logging", "l", true, "Include structured logging")
	generateCmd.Flags().BoolP("metrics", "m", true, "Include metrics collection")
	generateCmd.Flags().BoolP("docker", "", true, "Include Docker configuration")
	generateCmd.Flags().BoolP("k8s", "k", false, "Include Kubernetes manifests")
	generateCmd.Flags().BoolP("testing", "", true, "Include testing setup")
	generateCmd.Flags().BoolP("swagger", "s", false, "Include Swagger documentation")
	generateCmd.Flags().BoolP("grpc", "g", false, "Include gRPC support")
	generateCmd.Flags().BoolP("graphql", "", false, "Include GraphQL support")
	generateCmd.Flags().BoolP("websocket", "w", false, "Include WebSocket support")
	generateCmd.Flags().StringP("port", "p", "8080", "Default port")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listTemplatesCmd)
	rootCmd.AddCommand(initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runGenerate(cmd *cobra.Command, args []string) {
	config := &ProjectConfig{}

	// Parse flags
	config.Name, _ = cmd.Flags().GetString("name")
	config.Type, _ = cmd.Flags().GetString("type")
	config.Database, _ = cmd.Flags().GetString("database")
	config.Cache, _ = cmd.Flags().GetString("cache")
	config.Queue, _ = cmd.Flags().GetString("queue")
	config.Auth, _ = cmd.Flags().GetBool("auth")
	config.Logging, _ = cmd.Flags().GetBool("logging")
	config.Metrics, _ = cmd.Flags().GetBool("metrics")
	config.Docker, _ = cmd.Flags().GetBool("docker")
	config.Kubernetes, _ = cmd.Flags().GetBool("k8s")
	config.Testing, _ = cmd.Flags().GetBool("testing")
	config.Swagger, _ = cmd.Flags().GetBool("swagger")
	config.GRPC, _ = cmd.Flags().GetBool("grpc")
	config.GraphQL, _ = cmd.Flags().GetBool("graphql")
	config.WebSocket, _ = cmd.Flags().GetBool("websocket")
	config.Port, _ = cmd.Flags().GetString("port")
	config.GoVersion = "1.21"

	if config.Name == "" {
		fmt.Println("Project name is required")
		os.Exit(1)
	}

	if err := generateProject(config); err != nil {
		log.Fatalf("Failed to generate project: %v", err)
	}

	fmt.Printf("🎉 Successfully generated %s project: %s\n", config.Type, config.Name)
}

func runInteractive(cmd *cobra.Command, args []string) {
	config := &ProjectConfig{}
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🚀 Go Project Scaffolder - Interactive Mode")
	fmt.Println("==========================================")

	// Project name
	fmt.Print("Project name: ")
	scanner.Scan()
	config.Name = strings.TrimSpace(scanner.Text())

	// Project type
	fmt.Print("Project type (api/worker) [api]: ")
	scanner.Scan()
	projectType := strings.TrimSpace(scanner.Text())
	if projectType == "" {
		projectType = "api"
	}
	config.Type = projectType

	// Database
	fmt.Print("Database (postgres/mysql/mongodb/sqlite/none) [postgres]: ")
	scanner.Scan()
	database := strings.TrimSpace(scanner.Text())
	if database == "" {
		database = "postgres"
	}
	config.Database = database

	// Cache
	fmt.Print("Cache (redis/memcached/none) [redis]: ")
	scanner.Scan()
	cache := strings.TrimSpace(scanner.Text())
	if cache == "" {
		cache = "redis"
	}
	config.Cache = cache

	// Queue
	if config.Type == "worker" || askYesNo("Include message queue?", false) {
		fmt.Print("Queue (rabbitmq/kafka/sqs/none) [rabbitmq]: ")
		scanner.Scan()
		queue := strings.TrimSpace(scanner.Text())
		if queue == "" {
			queue = "rabbitmq"
		}
		config.Queue = queue
	}

	// Additional features
	config.Auth = askYesNo("Include authentication?", false)
	config.Logging = askYesNo("Include structured logging?", true)
	config.Metrics = askYesNo("Include metrics collection?", true)
	config.Docker = askYesNo("Include Docker configuration?", true)
	config.Kubernetes = askYesNo("Include Kubernetes manifests?", false)
	config.Testing = askYesNo("Include testing setup?", true)

	if config.Type == "api" {
		config.Swagger = askYesNo("Include Swagger documentation?", false)
		config.GRPC = askYesNo("Include gRPC support?", false)
		config.GraphQL = askYesNo("Include GraphQL support?", false)
		config.WebSocket = askYesNo("Include WebSocket support?", false)
	}

	// Port
	fmt.Print("Default port [8080]: ")
	scanner.Scan()
	port := strings.TrimSpace(scanner.Text())
	if port == "" {
		port = "8080"
	}
	config.Port = port

	config.GoVersion = "1.21"

	fmt.Println("\n📋 Configuration Summary:")
	fmt.Printf("  Name: %s\n", config.Name)
	fmt.Printf("  Type: %s\n", config.Type)
	fmt.Printf("  Database: %s\n", config.Database)
	fmt.Printf("  Cache: %s\n", config.Cache)
	if config.Queue != "" {
		fmt.Printf("  Queue: %s\n", config.Queue)
	}
	fmt.Printf("  Port: %s\n", config.Port)

	if !askYesNo("Generate project with these settings?", true) {
		fmt.Println("Aborted.")
		return
	}

	if err := generateProject(config); err != nil {
		log.Fatalf("Failed to generate project: %v", err)
	}

	fmt.Printf("\n🎉 Successfully generated %s project: %s\n", config.Type, config.Name)
	fmt.Println("\n📖 Next steps:")
	fmt.Printf("  cd %s\n", config.Name)
	fmt.Println("  go mod tidy")
	fmt.Println("  go run main.go")
}

func generateProject(config *ProjectConfig) error {
	// Create project directory
	if err := os.MkdirAll(config.Name, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate different project types
	switch config.Type {
	case ProjectTypeAPI:
		return generateAPIProject(config)
	case ProjectTypeWorker:
		return generateWorkerProject(config)
	default:
		return fmt.Errorf("unknown project type: %s", config.Type)
	}
}

func generateAPIProject(config *ProjectConfig) error {
	generator := &APIGenerator{config: config}
	return generator.Generate()
}

func generateWorkerProject(config *ProjectConfig) error {
	generator := &WorkerGenerator{config: config}
	return generator.Generate()
}

// Template helper functions
var templateFuncs = template.FuncMap{
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"title": strings.Title,
}

func createFileFromTemplate(templateContent string, data interface{}, outputPath string) error {
	tmpl, err := template.New("template").Funcs(templateFuncs).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create directories if they don't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
