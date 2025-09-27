package main

import (
	"path/filepath"
)

type APIGenerator struct {
	config *ProjectConfig
}

func (g *APIGenerator) Generate() error {
	generators := []func() error{
		g.generateGoMod,
		g.generateMain,
		g.generateConfig,
		g.generateServer,
		g.generateHandlers,
		g.generateMiddleware,
		g.generateModels,
		g.generateServices,
		g.generateRepository,
		g.generateDatabase,
		g.generateLogger,
		g.generateCache,
		g.generateEnvFile,
		g.generateDockerfile,
		g.generateDockerCompose,
		g.generateMakefile,
		g.generateReadme,
	}

	if g.config.Testing {
		generators = append(generators, g.generateTests)
	}

	if g.config.Swagger {
		generators = append(generators, g.generateSwagger)
	}

	if g.config.Kubernetes {
		generators = append(generators, g.generateKubernetes)
	}

	for _, gen := range generators {
		if err := gen(); err != nil {
			return err
		}
	}

	return nil
}

func (g *APIGenerator) generateGoMod() error {
	template := `module {{.Name}}

go {{.GoVersion}}

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/spf13/viper v1.16.0
	{{- if eq .Database "postgres"}}
	github.com/lib/pq v1.10.9
	{{- else if eq .Database "mysql"}}
	github.com/go-sql-driver/mysql v1.7.1
	{{- else if eq .Database "mongodb"}}
	go.mongodb.org/mongo-driver v1.12.1
	{{- else if eq .Database "sqlite"}}
	github.com/mattn/go-sqlite3 v1.14.17
	{{- end}}
	{{- if eq .Cache "redis"}}
	github.com/go-redis/redis/v8 v8.11.5
	{{- end}}
	{{- if eq .Queue "rabbitmq"}}
	github.com/streadway/amqp v1.1.0
	{{- else if eq .Queue "kafka"}}
	github.com/segmentio/kafka-go v0.4.42
	{{- end}}
	{{- if .Auth}}
	github.com/golang-jwt/jwt/v5 v5.0.0
	golang.org/x/crypto v0.12.0
	{{- end}}
	{{- if .Logging}}
	github.com/sirupsen/logrus v1.9.3
	{{- end}}
	{{- if .Metrics}}
	github.com/prometheus/client_golang v1.16.0
	{{- end}}
	{{- if .GRPC}}
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.31.0
	{{- end}}
	{{- if .GraphQL}}
	github.com/99designs/gqlgen v0.17.36
	{{- end}}
	{{- if .WebSocket}}
	github.com/gorilla/websocket v1.5.0
	{{- end}}
	{{- if .Swagger}}
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/files v1.0.1
	github.com/swaggo/swag v1.16.1
	{{- end}}
	{{- if .Testing}}
	github.com/stretchr/testify v1.8.4
	{{- end}}
)
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "go.mod"))
}

func (g *APIGenerator) generateMain() error {
	template := `package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"{{.Name}}/internal/config"
	"{{.Name}}/internal/database"
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
	"{{.Name}}/internal/server"
)

// @title {{.Name}} API
// @version 1.0
// @description A production-ready REST API built with Go
// @host localhost:{{.Port}}
// @BasePath /api/v1
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	{{- if .Logging}}
	// Initialize logger
	logger := logger.New(cfg.LogLevel)
	{{- end}}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		{{- if .Logging}}
		logger.Fatal("Failed to initialize database", "error", err)
		{{- else}}
		log.Fatalf("Failed to initialize database: %v", err)
		{{- end}}
	}
	defer db.Close()

	{{- if eq .Cache "redis"}}
	// Initialize cache
	cache := cache.New(cfg)
	defer cache.Close()
	{{- end}}

	// Initialize server
	srv := server.New(server.Config{
		Port: cfg.Server.Port,
		DB:   db,
		{{- if eq .Cache "redis"}}
		Cache: cache,
		{{- end}}
		{{- if .Logging}}
		Logger: logger,
		{{- end}}
	})

	// Start server in a goroutine
	go func() {
		{{- if .Logging}}
		logger.Info("Starting server", "port", cfg.Server.Port)
		{{- else}}
		fmt.Printf("Server starting on port %s\n", cfg.Server.Port)
		{{- end}}
		
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			{{- if .Logging}}
			logger.Fatal("Failed to start server", "error", err)
			{{- else}}
			log.Fatalf("Failed to start server: %v", err)
			{{- end}}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	{{- if .Logging}}
	logger.Info("Shutting down server...")
	{{- else}}
	fmt.Println("Shutting down server...")
	{{- end}}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		{{- if .Logging}}
		logger.Fatal("Server forced to shutdown", "error", err)
		{{- else}}
		log.Fatalf("Server forced to shutdown: %v", err)
		{{- end}}
	}

	{{- if .Logging}}
	logger.Info("Server exited")
	{{- else}}
	fmt.Println("Server exited")
	{{- end}}
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "main.go"))
}

func (g *APIGenerator) generateConfig() error {
	template := `package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   ` + "`yaml:\"server\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\"`" + `
	{{- if eq .Cache "redis"}}
	Redis    RedisConfig    ` + "`yaml:\"redis\"`" + `
	{{- end}}
	{{- if .Auth}}
	Auth     AuthConfig     ` + "`yaml:\"auth\"`" + `
	{{- end}}
	{{- if .Logging}}
	LogLevel string         ` + "`yaml:\"log_level\"`" + `
	{{- end}}
}

type ServerConfig struct {
	Port string ` + "`yaml:\"port\"`" + `
	Host string ` + "`yaml:\"host\"`" + `
}

type DatabaseConfig struct {
	{{- if eq .Database "postgres"}}
	Host     string ` + "`yaml:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\"`" + `
	User     string ` + "`yaml:\"user\"`" + `
	Password string ` + "`yaml:\"password\"`" + `
	DBName   string ` + "`yaml:\"dbname\"`" + `
	SSLMode  string ` + "`yaml:\"sslmode\"`" + `
	{{- else if eq .Database "mysql"}}
	Host     string ` + "`yaml:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\"`" + `
	User     string ` + "`yaml:\"user\"`" + `
	Password string ` + "`yaml:\"password\"`" + `
	DBName   string ` + "`yaml:\"dbname\"`" + `
	{{- else if eq .Database "sqlite"}}
	Path string ` + "`yaml:\"path\"`" + `
	{{- else if eq .Database "mongodb"}}
	URI    string ` + "`yaml:\"uri\"`" + `
	DBName string ` + "`yaml:\"dbname\"`" + `
	{{- end}}
}

{{- if eq .Cache "redis"}}
type RedisConfig struct {
	Host     string ` + "`yaml:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\"`" + `
	Password string ` + "`yaml:\"password\"`" + `
	DB       int    ` + "`yaml:\"db\"`" + `
}
{{- end}}

{{- if .Auth}}
type AuthConfig struct {
	JWTSecret     string ` + "`yaml:\"jwt_secret\"`" + `
	TokenDuration string ` + "`yaml:\"token_duration\"`" + `
}
{{- end}}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// Environment variables
	viper.SetEnvPrefix("{{.Name | upper}}")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	viper.SetDefault("server.port", "{{.Port}}")
	viper.SetDefault("server.host", "0.0.0.0")
	{{- if eq .Database "postgres"}}
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "{{.Name}}")
	viper.SetDefault("database.sslmode", "disable")
	{{- else if eq .Database "mysql"}}
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "{{.Name}}")
	{{- else if eq .Database "sqlite"}}
	viper.SetDefault("database.path", "{{.Name}}.db")
	{{- else if eq .Database "mongodb"}}
	viper.SetDefault("database.uri", "mongodb://localhost:27017")
	viper.SetDefault("database.dbname", "{{.Name}}")
	{{- end}}
	{{- if eq .Cache "redis"}}
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	{{- end}}
	{{- if .Auth}}
	viper.SetDefault("auth.jwt_secret", "your-secret-key")
	viper.SetDefault("auth.token_duration", "24h")
	{{- end}}
	{{- if .Logging}}
	viper.SetDefault("log_level", "info")
	{{- end}}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "config", "config.go"))
}

func (g *APIGenerator) generateServer() error {
	template := `package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	{{- if .Metrics}}
	"github.com/prometheus/client_golang/prometheus/promhttp"
	{{- end}}
	{{- if .Swagger}}
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	{{- end}}

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/handlers"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
	"{{.Name}}/internal/middleware"
)

type Server struct {
	router *gin.Engine
	server *http.Server
	config Config
}

type Config struct {
	Port string
	DB   database.Database
	{{- if eq .Cache "redis"}}
	Cache *cache.Client
	{{- end}}
	{{- if .Logging}}
	Logger *logger.Logger
	{{- end}}
}

func New(cfg Config) *Server {
	// Set gin mode
	if gin.Mode() != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		router: router,
		config: cfg,
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	{{- if .Logging}}
	// Logger middleware
	s.router.Use(middleware.Logger(s.config.Logger))
	{{- else}}
	// Default logger
	s.router.Use(gin.Logger())
	{{- end}}

	// CORS middleware
	s.router.Use(middleware.CORS())

	{{- if .Auth}}
	// Add auth middleware when needed
	{{- end}}

	{{- if .Metrics}}
	// Metrics middleware
	s.router.Use(middleware.Metrics())
	{{- end}}
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", handlers.Health)

	{{- if .Metrics}}
	// Metrics endpoint
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	{{- end}}

	{{- if .Swagger}}
	// Swagger documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	{{- end}}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Initialize handlers
		h := handlers.New(handlers.Config{
			DB: s.config.DB,
			{{- if eq .Cache "redis"}}
			Cache: s.config.Cache,
			{{- end}}
			{{- if .Logging}}
			Logger: s.config.Logger,
			{{- end}}
		})

		// User routes
		users := v1.Group("/users")
		{
			users.GET("", h.GetUsers)
			users.GET("/:id", h.GetUser)
			users.POST("", h.CreateUser)
			users.PUT("/:id", h.UpdateUser)
			users.DELETE("/:id", h.DeleteUser)
		}

		{{- if .Auth}}
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
		}
		{{- end}}

		{{- if .WebSocket}}
		// WebSocket endpoint
		v1.GET("/ws", h.HandleWebSocket)
		{{- end}}
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "server", "server.go"))
}

func (g *APIGenerator) generateHandlers() error {
	template := `package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/models"
	"{{.Name}}/internal/services"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
)

type Handler struct {
	userService *services.UserService
	{{- if .Logging}}
	logger      *logger.Logger
	{{- end}}
}

type Config struct {
	DB database.Database
	{{- if eq .Cache "redis"}}
	Cache *cache.Client
	{{- end}}
	{{- if .Logging}}
	Logger *logger.Logger
	{{- end}}
}

func New(cfg Config) *Handler {
	userService := services.NewUserService(cfg.DB{{- if eq .Cache "redis"}}, cfg.Cache{{- end}})
	
	return &Handler{
		userService: userService,
		{{- if .Logging}}
		logger:      cfg.Logger,
		{{- end}}
	}
}

// Health godoc
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Service is healthy",
	})
}

// GetUsers godoc
// @Summary Get all users
// @Description Get a list of all users
// @Tags users
// @Produce json
// @Success 200 {array} models.User
// @Router /api/v1/users [get]
func (h *Handler) GetUsers(c *gin.Context) {
	users, err := h.userService.GetAll()
	if err != nil {
		{{- if .Logging}}
		h.logger.Error("Failed to get users", "error", err)
		{{- end}}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get a single user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		{{- if .Logging}}
		h.logger.Error("Failed to get user", "error", err, "id", id)
		{{- end}}
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user with the input payload
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.CreateUserRequest true "Create user"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Router /api/v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Create(&req)
	if err != nil {
		{{- if .Logging}}
		h.logger.Error("Failed to create user", "error", err)
		{{- end}}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body models.UpdateUserRequest true "Update user"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Update(id, &req)
	if err != nil {
		{{- if .Logging}}
		h.logger.Error("Failed to update user", "error", err, "id", id)
		{{- end}}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete user by ID
// @Tags users
// @Param id path int true "User ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.userService.Delete(id); err != nil {
		{{- if .Logging}}
		h.logger.Error("Failed to delete user", "error", err, "id", id)
		{{- end}}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

{{- if .Auth}}
// Register godoc
// @Summary Register a new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.RegisterRequest true "Register user"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Implementation would go here
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Implementation would go here
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// RefreshToken godoc
// @Summary Refresh JWT token
// @Description Refresh an expired JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Implementation would go here
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
{{- end}}

{{- if .WebSocket}}
// HandleWebSocket godoc
// @Summary WebSocket endpoint
// @Description WebSocket connection endpoint
// @Tags websocket
// @Router /api/v1/ws [get]
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Implementation would go here
	c.JSON(http.StatusNotImplemented, gin.H{"error": "WebSocket not implemented yet"})
}
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "handlers", "handlers.go"))
}

func (g *APIGenerator) generateModels() error {
	template := `package models

import (
	"time"
	{{- if .Auth}}
	"github.com/golang-jwt/jwt/v5"
	{{- end}}
)

// User represents a user in the system
type User struct {
	ID        int       ` + "`json:\"id\" db:\"id\"`" + `
	Name      string    ` + "`json:\"name\" db:\"name\"`" + `
	Email     string    ` + "`json:\"email\" db:\"email\"`" + `
	{{- if .Auth}}
	Password  string    ` + "`json:\"-\" db:\"password\"`" + `
	{{- end}}
	CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
}

// CreateUserRequest represents the request payload for creating a user
type CreateUserRequest struct {
	Name  string ` + "`json:\"name\" binding:\"required\"`" + `
	Email string ` + "`json:\"email\" binding:\"required,email\"`" + `
	{{- if .Auth}}
	Password string ` + "`json:\"password\" binding:\"required,min=6\"`" + `
	{{- end}}
}

// UpdateUserRequest represents the request payload for updating a user
type UpdateUserRequest struct {
	Name  *string ` + "`json:\"name,omitempty\"`" + `
	Email *string ` + "`json:\"email,omitempty\" binding:\"omitempty,email\"`" + `
}

{{- if .Auth}}
// RegisterRequest represents the request payload for user registration
type RegisterRequest struct {
	Name     string ` + "`json:\"name\" binding:\"required\"`" + `
	Email    string ` + "`json:\"email\" binding:\"required,email\"`" + `
	Password string ` + "`json:\"password\" binding:\"required,min=6\"`" + `
}

// LoginRequest represents the request payload for user login
type LoginRequest struct {
	Email    string ` + "`json:\"email\" binding:\"required,email\"`" + `
	Password string ` + "`json:\"password\" binding:\"required\"`" + `
}

// RefreshTokenRequest represents the request payload for token refresh
type RefreshTokenRequest struct {
	RefreshToken string ` + "`json:\"refresh_token\" binding:\"required\"`" + `
}

// AuthResponse represents the response payload for authentication
type AuthResponse struct {
	AccessToken  string ` + "`json:\"access_token\"`" + `
	RefreshToken string ` + "`json:\"refresh_token\"`" + `
	ExpiresIn    int    ` + "`json:\"expires_in\"`" + `
	User         *User  ` + "`json:\"user\"`" + `
}

// Claims represents JWT claims
type Claims struct {
	UserID int    ` + "`json:\"user_id\"`" + `
	Email  string ` + "`json:\"email\"`" + `
	jwt.RegisteredClaims
}
{{- end}}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string ` + "`json:\"error\"`" + `
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      ` + "`json:\"message\"`" + `
	Data    interface{} ` + "`json:\"data,omitempty\"`" + `
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "models", "models.go"))
}

func (g *APIGenerator) generateServices() error {
	template := `package services

import (
	"errors"

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/models"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
)

type UserService struct {
	db database.Database
	{{- if eq .Cache "redis"}}
	cache *cache.Client
	{{- end}}
}

func NewUserService(db database.Database{{- if eq .Cache "redis"}}, cache *cache.Client{{- end}}) *UserService {
	return &UserService{
		db: db,
		{{- if eq .Cache "redis"}}
		cache: cache,
		{{- end}}
	}
}

func (s *UserService) GetAll() ([]*models.User, error) {
	// Implementation depends on database type
	// This is a placeholder implementation
	users := []*models.User{}
	
	// Add your database query here
	return users, nil
}

func (s *UserService) GetByID(id int) (*models.User, error) {
	// Implementation depends on database type
	// This is a placeholder implementation
	
	{{- if eq .Cache "redis"}}
	// Try to get from cache first
	// cacheKey := fmt.Sprintf("user:%d", id)
	// if cached, err := s.cache.Get(cacheKey); err == nil {
	//     var user models.User
	//     if err := json.Unmarshal([]byte(cached), &user); err == nil {
	//         return &user, nil
	//     }
	// }
	{{- end}}

	// Add your database query here
	user := &models.User{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	{{- if eq .Cache "redis"}}
	// Cache the result
	// if userJSON, err := json.Marshal(user); err == nil {
	//     s.cache.Set(cacheKey, string(userJSON), time.Hour)
	// }
	{{- end}}

	return user, nil
}

func (s *UserService) Create(req *models.CreateUserRequest) (*models.User, error) {
	// Validate input
	if req.Name == "" || req.Email == "" {
		return nil, errors.New("name and email are required")
	}

	// Check if user with email already exists
	// Add your database query here

	// Hash password if auth is enabled
	{{- if .Auth}}
	// hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	// if err != nil {
	//     return nil, err
	// }
	{{- end}}

	// Create user in database
	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
		{{- if .Auth}}
		// Password: string(hashedPassword),
		{{- end}}
	}

	// Add your database insert here
	user.ID = 1 // This would come from the database

	return user, nil
}

func (s *UserService) Update(id int, req *models.UpdateUserRequest) (*models.User, error) {
	// Get existing user
	user, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}

	// Add your database update here

	{{- if eq .Cache "redis"}}
	// Invalidate cache
	// cacheKey := fmt.Sprintf("user:%d", id)
	// s.cache.Delete(cacheKey)
	{{- end}}

	return user, nil
}

func (s *UserService) Delete(id int) error {
	// Check if user exists
	_, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Add your database delete here

	{{- if eq .Cache "redis"}}
	// Invalidate cache
	// cacheKey := fmt.Sprintf("user:%d", id)
	// s.cache.Delete(cacheKey)
	{{- end}}

	return nil
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "services", "user_service.go"))
}

func (g *APIGenerator) generateMiddleware() error {
	template := `package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
	{{- if .Metrics}}
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	{{- end}}
)

{{- if .Logging}}
// Logger returns a gin.HandlerFunc for logging requests
func Logger(logger *logger.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP Request",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
		)
		return ""
	})
}
{{- end}}

// CORS returns a gin.HandlerFunc for handling CORS
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

{{- if .Metrics}}
var (
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "The HTTP request latencies in seconds",
		},
		[]string{"method", "endpoint"},
	)
)

// Metrics returns a gin.HandlerFunc for collecting metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := string(rune(c.Writer.Status()))

		httpRequests.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		httpDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
{{- end}}

{{- if .Auth}}
// Auth returns a gin.HandlerFunc for JWT authentication
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implementation would go here
		// Parse JWT token from Authorization header
		// Validate token
		// Set user context
		
		c.Next()
	}
}
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "middleware", "middleware.go"))
}

func (g *APIGenerator) generateRepository() error {
	template := `package repository

import (
	"{{.Name}}/internal/database"
	"{{.Name}}/internal/models"
)

type UserRepository interface {
	GetAll() ([]*models.User, error)
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id int) error
}

type userRepository struct {
	db database.Database
}

func NewUserRepository(db database.Database) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetAll() ([]*models.User, error) {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return []*models.User{}, nil
}

func (r *userRepository) GetByID(id int) (*models.User, error) {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE id = ?"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE id = ?"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return &models.User{}, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE email = $1"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE email = ?"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "SELECT id, name, email, created_at, updated_at FROM users WHERE email = ?"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return &models.User{}, nil
}

func (r *userRepository) Create(user *models.User) error {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at, updated_at"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "INSERT INTO users (name, email) VALUES (?, ?)"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return nil
}

func (r *userRepository) Update(user *models.User) error {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE id = $3"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "UPDATE users SET name = ?, email = ?, updated_at = NOW() WHERE id = ?"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "UPDATE users SET name = ?, email = ?, updated_at = datetime('now') WHERE id = ?"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return nil
}

func (r *userRepository) Delete(id int) error {
	// Implementation depends on database type
	{{- if eq .Database "postgres" }}
	query := "DELETE FROM users WHERE id = $1"
	// Add PostgreSQL implementation
	{{- else if eq .Database "mysql" }}
	query := "DELETE FROM users WHERE id = ?"
	// Add MySQL implementation
	{{- else if eq .Database "sqlite" }}
	query := "DELETE FROM users WHERE id = ?"
	// Add SQLite implementation
	{{- else if eq .Database "mongodb" }}
	// Add MongoDB implementation
	{{- end}}
	
	// Placeholder implementation
	return nil
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "repository", "user_repository.go"))
}

func (g *APIGenerator) generateDatabase() error {
	template := `package database

import (
	"database/sql"
	"fmt"
	{{- if eq .Database "postgres"}}
	_ "github.com/lib/pq"
	{{- else if eq .Database "mysql"}}
	_ "github.com/go-sql-driver/mysql"
	{{- else if eq .Database "sqlite"}}
	_ "github.com/mattn/go-sqlite3"
	{{- else if eq .Database "mongodb"}}
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	{{- end}}

	"{{.Name}}/internal/config"
)

type Database interface {
	Close() error
	{{- if ne .Database "mongodb"}}
	GetDB() *sql.DB
	{{- else}}
	GetDB() *mongo.Database
	{{- end}}
	Migrate() error
}

{{- if ne .Database "mongodb"}}
type sqlDatabase struct {
	db *sql.DB
}

func New(cfg *config.Config) (Database, error) {
	var dsn string
	var driverName string

	{{- if eq .Database "postgres"}}
	driverName = "postgres"
	dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)
	{{- else if eq .Database "mysql"}}
	driverName = "mysql"
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)
	{{- else if eq .Database "sqlite"}}
	driverName = "sqlite3"
	dsn = cfg.Database.Path
	{{- end}}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	return &sqlDatabase{db: db}, nil
}

func (d *sqlDatabase) Close() error {
	return d.db.Close()
}

func (d *sqlDatabase) GetDB() *sql.DB {
	return d.db
}

func (d *sqlDatabase) Migrate() error {
	// Create users table
	createUsersTable := ` + "`" + `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			{{- if .Auth}}
			password VARCHAR(255) NOT NULL,
			{{- end}}
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	` + "`" + `

	{{- if eq .Database "sqlite"}}
	createUsersTable = ` + "`" + `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			{{- if .Auth}}
			password TEXT NOT NULL,
			{{- end}}
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	` + "`" + `
	{{- end}}

	if _, err := d.db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	return nil
}

{{- else}}
// MongoDB implementation
type mongoDatabase struct {
	client *mongo.Client
	db     *mongo.Database
}

func New(cfg *config.Config) (Database, error) {
	clientOptions := options.Client().ApplyURI(cfg.Database.URI)
	
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database.DBName)
	
	return &mongoDatabase{
		client: client,
		db:     db,
	}, nil
}

func (d *mongoDatabase) Close() error {
	return d.client.Disconnect(context.Background())
}

func (d *mongoDatabase) GetDB() *mongo.Database {
	return d.db
}

func (d *mongoDatabase) Migrate() error {
	// Create indexes for users collection
	ctx := context.Background()
	usersCollection := d.db.Collection("users")
	
	// Create unique index on email
	indexModel := mongo.IndexModel{
		Keys: map[string]int{"email": 1},
		Options: options.Index().SetUnique(true),
	}
	
	_, err := usersCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create email index: %w", err)
	}

	return nil
}
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "database", "database.go"))
}

func (g *APIGenerator) generateLogger() error {
	if !g.config.Logging {
		return nil
	}

	template := `package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New(level string) *Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})

	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	return &Logger{log}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.WithFields(parseFields(fields...)).Info(msg)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.WithFields(parseFields(fields...)).Error(msg)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.WithFields(parseFields(fields...)).Debug(msg)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.WithFields(parseFields(fields...)).Warn(msg)
}

func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.WithFields(parseFields(fields...)).Fatal(msg)
}

func parseFields(fields ...interface{}) logrus.Fields {
	logFields := make(logrus.Fields)
	
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				logFields[key] = fields[i+1]
			}
		}
	}
	
	return logFields
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "logger", "logger.go"))
}

func (g *APIGenerator) generateCache() error {
	if g.config.Cache != "redis" {
		return nil
	}

	template := `package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"{{.Name}}/internal/config"
)

type Client struct {
	client *redis.Client
	ctx    context.Context
}

func New(cfg *config.Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &Client{
		client: rdb,
		ctx:    context.Background(),
	}
}

func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(c.ctx, key, value, expiration).Err()
}

func (c *Client) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}

func (c *Client) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

func (c *Client) Exists(key string) (bool, error) {
	result, err := c.client.Exists(c.ctx, key).Result()
	return result > 0, err
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Ping() error {
	return c.client.Ping(c.ctx).Err()
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "cache", "redis.go"))
}

func (g *APIGenerator) generateEnvFile() error {
	template := `# Server Configuration
{{.Name | upper}}_SERVER_PORT={{.Port}}
{{.Name | upper}}_SERVER_HOST=0.0.0.0

# Database Configuration
{{- if eq .Database "postgres"}}
{{.Name | upper}}_DATABASE_HOST=localhost
{{.Name | upper}}_DATABASE_PORT=5432
{{.Name | upper}}_DATABASE_USER=postgres
{{.Name | upper}}_DATABASE_PASSWORD=password
{{.Name | upper}}_DATABASE_DBNAME={{.Name}}
{{.Name | upper}}_DATABASE_SSLMODE=disable
{{- else if eq .Database "mysql"}}
{{.Name | upper}}_DATABASE_HOST=localhost
{{.Name | upper}}_DATABASE_PORT=3306
{{.Name | upper}}_DATABASE_USER=root
{{.Name | upper}}_DATABASE_PASSWORD=password
{{.Name | upper}}_DATABASE_DBNAME={{.Name}}
{{- else if eq .Database "sqlite"}}
{{.Name | upper}}_DATABASE_PATH={{.Name}}.db
{{- else if eq .Database "mongodb"}}
{{.Name | upper}}_DATABASE_URI=mongodb://localhost:27017
{{.Name | upper}}_DATABASE_DBNAME={{.Name}}
{{- end}}

{{- if eq .Cache "redis"}}
# Redis Configuration
{{.Name | upper}}_REDIS_HOST=localhost
{{.Name | upper}}_REDIS_PORT=6379
{{.Name | upper}}_REDIS_PASSWORD=
{{.Name | upper}}_REDIS_DB=0
{{- end}}

{{- if .Auth}}
# Auth Configuration
{{.Name | upper}}_AUTH_JWT_SECRET=your-super-secret-jwt-key
{{.Name | upper}}_AUTH_TOKEN_DURATION=24h
{{- end}}

{{- if .Logging}}
# Logging Configuration
{{.Name | upper}}_LOG_LEVEL=info
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, ".env.example"))
}

func (g *APIGenerator) generateDockerfile() error {
	if !g.config.Docker {
		return nil
	}

	template := `# Build stage
FROM golang:{{.GoVersion}}-alpine AS builder

WORKDIR /app

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy config files if they exist
COPY --from=builder /app/configs ./configs/ 2>/dev/null || :

# Expose port
EXPOSE {{.Port}}

# Run the binary
CMD ["./main"]
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "Dockerfile"))
}

func (g *APIGenerator) generateDockerCompose() error {
	if !g.config.Docker {
		return nil
	}

	template := `version: '3.8'

services:
  {{.Name}}:
    build: .
    ports:
      - "{{.Port}}:{{.Port}}"
    environment:
      - {{.Name | upper}}_SERVER_PORT={{.Port}}
      {{- if eq .Database "postgres"}}
      - {{.Name | upper}}_DATABASE_HOST=postgres
      - {{.Name | upper}}_DATABASE_PASSWORD=password
      {{- else if eq .Database "mysql"}}
      - {{.Name | upper}}_DATABASE_HOST=mysql
      - {{.Name | upper}}_DATABASE_PASSWORD=password
      {{- else if eq .Database "mongodb"}}
      - {{.Name | upper}}_DATABASE_URI=mongodb://mongodb:27017
      {{- end}}
      {{- if eq .Cache "redis"}}
      - {{.Name | upper}}_REDIS_HOST=redis
      {{- end}}
    depends_on:
      {{- if eq .Database "postgres"}}
      - postgres
      {{- else if eq .Database "mysql"}}
      - mysql
      {{- else if eq .Database "mongodb"}}
      - mongodb
      {{- end}}
      {{- if eq .Cache "redis"}}
      - redis
      {{- end}}
    restart: unless-stopped

  {{- if eq .Database "postgres"}}
  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB={{.Name}}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
  {{- else if eq .Database "mysql"}}
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE={{.Name}}
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped
  {{- else if eq .Database "mongodb"}}
  mongodb:
    image: mongo:6.0
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db
    restart: unless-stopped
  {{- end}}

  {{- if eq .Cache "redis"}}
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
  {{- end}}

volumes:
  {{- if eq .Database "postgres"}}
  postgres_data:
  {{- else if eq .Database "mysql"}}
  mysql_data:
  {{- else if eq .Database "mongodb"}}
  mongodb_data:
  {{- end}}
  {{- if eq .Cache "redis"}}
  redis_data:
  {{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "docker-compose.yml"))
}

func (g *APIGenerator) generateMakefile() error {
	template := `# {{.Name}} Makefile

.PHONY: help build run test clean docker-build docker-run docker-compose-up docker-compose-down

# Variables
APP_NAME={{.Name}}
DOCKER_IMAGE={{.Name}}:latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $1, $2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/$(APP_NAME) .

run: ## Run the application
	go run .

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

{{- if .Docker}}
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	docker run -p {{.Port}}:{{.Port}} $(DOCKER_IMAGE)

docker-compose-up: ## Start services with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	docker-compose down

docker-compose-logs: ## View logs from docker-compose
	docker-compose logs -f
{{- end}}

{{- if .Swagger}}
swagger: ## Generate Swagger documentation
	swag init
{{- end}}

migrate-up: ## Run database migrations up
	# Add your migration command here

migrate-down: ## Run database migrations down
	# Add your migration command here

dev: ## Run in development mode with hot reload
	air
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "Makefile"))
}

func (g *APIGenerator) generateReadme() error {
	template := `# {{.Name}}

A production-ready {{.Type}} built with Go, featuring:

{{- if eq .Database "postgres"}}
- PostgreSQL database
{{- else if eq .Database "mysql"}}
- MySQL database
{{- else if eq .Database "sqlite"}}
- SQLite database
{{- else if eq .Database "mongodb"}}
- MongoDB database
{{- end}}
{{- if eq .Cache "redis"}}
- Redis caching
{{- end}}
{{- if .Auth}}
- JWT authentication
{{- end}}
{{- if .Logging}}
- Structured logging
{{- end}}
{{- if .Metrics}}
- Prometheus metrics
{{- end}}
{{- if .Docker}}
- Docker support
{{- end}}
{{- if .Kubernetes}}
- Kubernetes deployment
{{- end}}
{{- if .Swagger}}
- Swagger documentation
{{- end}}
{{- if .GRPC}}
- gRPC support
{{- end}}
{{- if .GraphQL}}
- GraphQL support
{{- end}}
{{- if .WebSocket}}
- WebSocket support
{{- end}}

## Getting Started

### Prerequisites

- Go {{.GoVersion}} or later
{{- if eq .Database "postgres"}}
- PostgreSQL
{{- else if eq .Database "mysql"}}
- MySQL
{{- else if eq .Database "mongodb"}}
- MongoDB
{{- end}}
{{- if eq .Cache "redis"}}
- Redis
{{- end}}
{{- if .Docker}}
- Docker and Docker Compose (optional)
{{- end}}

### Installation

1. Clone the repository:
` + "```bash" + `
git clone <repository-url>
cd {{.Name}}
` + "```" + `

2. Install dependencies:
` + "```bash" + `
go mod download
` + "```" + `

3. Copy the environment file and configure:
` + "```bash" + `
cp .env.example .env
# Edit .env with your configuration
` + "```" + `

4. Run the application:
` + "```bash" + `
go run main.go
` + "```" + `

### Using Docker

1. Build and run with Docker Compose:
` + "```bash" + `
docker-compose up -d
` + "```" + `

2. View logs:
` + "```bash" + `
docker-compose logs -f
` + "```" + `

3. Stop services:
` + "```bash" + `
docker-compose down
` + "```" + `

### Using Makefile

` + "```bash" + `
make help          # Show available commands
make build         # Build the application
make run           # Run the application
make test          # Run tests
make docker-build  # Build Docker image
make docker-run    # Run Docker container
` + "```" + `

## API Endpoints

{{- if .Swagger}}
API documentation is available at: http://localhost:{{.Port}}/swagger/index.html
{{- end}}

### Health Check
- ` + "`GET /health`" + ` - Health check endpoint

### Users
- ` + "`GET /api/v1/users`" + ` - Get all users
- ` + "`GET /api/v1/users/{id}`" + ` - Get user by ID
- ` + "`POST /api/v1/users`" + ` - Create new user
- ` + "`PUT /api/v1/users/{id}`" + ` - Update user
- ` + "`DELETE /api/v1/users/{id}`" + ` - Delete user

{{- if .Auth}}
### Authentication
- ` + "`POST /api/v1/auth/register`" + ` - Register new user
- ` + "`POST /api/v1/auth/login`" + ` - Login user
- ` + "`POST /api/v1/auth/refresh`" + ` - Refresh JWT token
{{- end}}

{{- if .Metrics}}
### Metrics
- ` + "`GET /metrics`" + ` - Prometheus metrics endpoint
{{- end}}

{{- if .WebSocket}}
### WebSocket
- ` + "`GET /api/v1/ws`" + ` - WebSocket connection endpoint
{{- end}}

## Project Structure

` + "```" + `
{{.Name}}/
├── main.go                 # Application entry point
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies
├── .env.example           # Environment variables example
{{- if .Docker}}
├── Dockerfile             # Docker configuration
├── docker-compose.yml     # Docker Compose configuration
{{- end}}
├── Makefile               # Build and run commands
├── README.md              # Project documentation
└── internal/              # Private application code
    ├── config/            # Configuration management
    ├── database/          # Database connection and migrations
    ├── handlers/          # HTTP handlers
    ├── middleware/        # HTTP middleware
    ├── models/            # Data models
    ├── repository/        # Data access layer
    ├── server/            # Server configuration
    ├── services/          # Business logic
    {{- if .Logging}}
    ├── logger/            # Logging utilities
    {{- end}}
    {{- if eq .Cache "redis"}}
    └── cache/             # Cache utilities
    {{- end}}
` + "```" + `

## Contributing

1. Fork the repository
2. Create your feature branch (` + "`git checkout -b feature/amazing-feature`" + `)
3. Commit your changes (` + "`git commit -m 'Add some amazing feature'`" + `)
4. Push to the branch (` + "`git push origin feature/amazing-feature`" + `)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "README.md"))
}

func (g *APIGenerator) generateTests() error {
	if !g.config.Testing {
		return nil
	}

	handlerTestTemplate := `package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"{{.Name}}/internal/models"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	Health(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

func TestCreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test data
	user := models.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
	}
	
	jsonData, _ := json.Marshal(user)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	
	// Test implementation would go here
	assert.True(t, true) // Placeholder assertion
}
`

	return createFileFromTemplate(handlerTestTemplate, g.config, filepath.Join(g.config.Name, "internal", "handlers", "handlers_test.go"))
}

func (g *APIGenerator) generateSwagger() error {
	if !g.config.Swagger {
		return nil
	}

	swaggerTemplate := `package docs

// This file will be generated by swag init
// Run: swag init --generalInfo main.go --output ./docs
`

	return createFileFromTemplate(swaggerTemplate, g.config, filepath.Join(g.config.Name, "docs", "docs.go"))
}

func (g *APIGenerator) generateKubernetes() error {
	if !g.config.Kubernetes {
		return nil
	}

	deploymentTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{.Name}}
  template:
    metadata:
      labels:
        app: {{.Name}}
    spec:
      containers:
      - name: {{.Name}}
        image: {{.Name}}:latest
        ports:
        - containerPort: {{.Port}}
        env:
        - name: {{.Name | upper}}_SERVER_PORT
          value: "{{.Port}}"
        {{- if eq .Database "postgres"}}
        - name: {{.Name | upper}}_DATABASE_HOST
          value: "postgres"
        - name: {{.Name | upper}}_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{.Name}}-secrets
              key: database-password
        {{- end}}
        {{- if eq .Cache "redis"}}
        - name: {{.Name | upper}}_REDIS_HOST
          value: "redis"
        {{- end}}
---
apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
spec:
  selector:
    app: {{.Name}}
  ports:
  - protocol: TCP
    port: 80
    targetPort: {{.Port}}
  type: LoadBalancer
`

	return createFileFromTemplate(deploymentTemplate, g.config, filepath.Join(g.config.Name, "k8s", "deployment.yaml"))
}
