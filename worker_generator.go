package main

import (
	"path/filepath"
)

type WorkerGenerator struct {
	config *ProjectConfig
}

func (g *WorkerGenerator) Generate() error {
	generators := []func() error{
		g.generateGoMod,
		g.generateMain,
		g.generateConfig,
		g.generateWorker,
		g.generateJobHandler,
		g.generateQueue,
		g.generateModels,
		g.generateServices,
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

func (g *WorkerGenerator) generateGoMod() error {
	template := `module {{.Name}}

go {{.GoVersion}}

require (
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
	{{- else if eq .Queue "sqs"}}
	github.com/aws/aws-sdk-go v1.44.327
	{{- end}}
	{{- if .Logging}}
	github.com/sirupsen/logrus v1.9.3
	{{- end}}
	{{- if .Metrics}}
	github.com/prometheus/client_golang v1.16.0
	{{- end}}
	{{- if .Testing}}
	github.com/stretchr/testify v1.8.4
	{{- end}}
	golang.org/x/sync v0.3.0
)
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "go.mod"))
}

func (g *WorkerGenerator) generateMain() error {
	template := `package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
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
	"{{.Name}}/internal/queue"
	"{{.Name}}/internal/worker"
)

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

	// Initialize queue
	q, err := queue.New(cfg)
	if err != nil {
		{{- if .Logging}}
		logger.Fatal("Failed to initialize queue", "error", err)
		{{- else}}
		log.Fatalf("Failed to initialize queue: %v", err)
		{{- end}}
	}
	defer q.Close()

	// Initialize worker
	w := worker.New(worker.Config{
		DB:    db,
		Queue: q,
		{{- if eq .Cache "redis"}}
		Cache: cache,
		{{- end}}
		{{- if .Logging}}
		Logger: logger,
		{{- end}}
		WorkerCount: cfg.Worker.Count,
	})

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		{{- if .Logging}}
		logger.Info("Starting worker", "count", cfg.Worker.Count)
		{{- else}}
		log.Printf("Starting worker with %d goroutines\n", cfg.Worker.Count)
		{{- end}}
		
		if err := w.Start(ctx); err != nil {
			{{- if .Logging}}
			logger.Error("Worker error", "error", err)
			{{- else}}
			log.Printf("Worker error: %v\n", err)
			{{- end}}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	{{- if .Logging}}
	logger.Info("Shutting down worker...")
	{{- else}}
	log.Println("Shutting down worker...")
	{{- end}}

	// Cancel context and wait for graceful shutdown
	cancel()
	
	// Wait for shutdown with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		{{- if .Logging}}
		logger.Info("Worker shutdown complete")
		{{- else}}
		log.Println("Worker shutdown complete")
		{{- end}}
	case <-time.After(30 * time.Second):
		{{- if .Logging}}
		logger.Warn("Worker shutdown timeout")
		{{- else}}
		log.Println("Worker shutdown timeout")
		{{- end}}
	}
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "main.go"))
}

func (g *WorkerGenerator) generateConfig() error {
	template := `package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Worker   WorkerConfig   ` + "`yaml:\"worker\"`" + `
	Database DatabaseConfig ` + "`yaml:\"database\"`" + `
	Queue    QueueConfig    ` + "`yaml:\"queue\"`" + `
	{{- if eq .Cache "redis"}}
	Redis    RedisConfig    ` + "`yaml:\"redis\"`" + `
	{{- end}}
	{{- if .Logging}}
	LogLevel string         ` + "`yaml:\"log_level\"`" + `
	{{- end}}
}

type WorkerConfig struct {
	Count int ` + "`yaml:\"count\"`" + `
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

type QueueConfig struct {
	{{- if eq .Queue "rabbitmq"}}
	URL       string ` + "`yaml:\"url\"`" + `
	QueueName string ` + "`yaml:\"queue_name\"`" + `
	{{- else if eq .Queue "kafka"}}
	Brokers []string ` + "`yaml:\"brokers\"`" + `
	Topic   string   ` + "`yaml:\"topic\"`" + `
	GroupID string   ` + "`yaml:\"group_id\"`" + `
	{{- else if eq .Queue "sqs"}}
	Region   string ` + "`yaml:\"region\"`" + `
	QueueURL string ` + "`yaml:\"queue_url\"`" + `
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
	viper.SetDefault("worker.count", 5)
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
	{{- if eq .Queue "rabbitmq"}}
	viper.SetDefault("queue.url", "amqp://localhost:5672")
	viper.SetDefault("queue.queue_name", "{{.Name}}_jobs")
	{{- else if eq .Queue "kafka"}}
	viper.SetDefault("queue.brokers", []string{"localhost:9092"})
	viper.SetDefault("queue.topic", "{{.Name}}_jobs")
	viper.SetDefault("queue.group_id", "{{.Name}}_worker")
	{{- else if eq .Queue "sqs"}}
	viper.SetDefault("queue.region", "us-east-1")
	viper.SetDefault("queue.queue_url", "https://sqs.us-east-1.amazonaws.com/123456789012/{{.Name}}-jobs")
	{{- end}}
	{{- if eq .Cache "redis"}}
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
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

func (g *WorkerGenerator) generateWorker() error {
	template := `package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/handlers"
	"{{.Name}}/internal/queue"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
)

type Worker struct {
	db          database.Database
	queue       queue.Queue
	{{- if eq .Cache "redis"}}
	cache       *cache.Client
	{{- end}}
	{{- if .Logging}}
	logger      *logger.Logger
	{{- end}}
	workerCount int
	handler     *handlers.JobHandler
}

type Config struct {
	DB          database.Database
	Queue       queue.Queue
	{{- if eq .Cache "redis"}}
	Cache       *cache.Client
	{{- end}}
	{{- if .Logging}}
	Logger      *logger.Logger
	{{- end}}
	WorkerCount int
}

func New(cfg Config) *Worker {
	handler := handlers.NewJobHandler(handlers.JobHandlerConfig{
		DB: cfg.DB,
		{{- if eq .Cache "redis"}}
		Cache: cfg.Cache,
		{{- end}}
		{{- if .Logging}}
		Logger: cfg.Logger,
		{{- end}}
	})

	return &Worker{
		db:          cfg.DB,
		queue:       cfg.Queue,
		{{- if eq .Cache "redis"}}
		cache:       cfg.Cache,
		{{- end}}
		{{- if .Logging}}
		logger:      cfg.Logger,
		{{- end}}
		workerCount: cfg.WorkerCount,
		handler:     handler,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	{{- if .Logging}}
	w.logger.Info("Worker starting", "count", w.workerCount)
	{{- end}}

	// Create semaphore to limit concurrent workers
	sem := semaphore.NewWeighted(int64(w.workerCount))
	
	var wg sync.WaitGroup
	
	// Start message consumer
	messages, err := w.queue.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			{{- if .Logging}}
			w.logger.Info("Worker stopping...")
			{{- end}}
			wg.Wait()
			return ctx.Err()
		case msg := <-messages:
			// Acquire semaphore (blocks if all workers are busy)
			if err := sem.Acquire(ctx, 1); err != nil {
				{{- if .Logging}}
				w.logger.Error("Failed to acquire semaphore", "error", err)
				{{- end}}
				continue
			}

			wg.Add(1)
			go func(message queue.Message) {
				defer wg.Done()
				defer sem.Release(1)
				
				w.processMessage(ctx, message)
			}(msg)
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, msg queue.Message) {
	start := time.Now()
	{{- if .Logging}}
	w.logger.Info("Processing message", "id", msg.ID, "type", msg.Type)
	{{- end}}

	err := w.handler.HandleJob(ctx, msg)
	duration := time.Since(start)

	if err != nil {
		{{- if .Logging}}
		w.logger.Error("Job failed", 
			"id", msg.ID, 
			"type", msg.Type, 
			"error", err, 
			"duration", duration,
			"attempts", msg.Attempts)
		{{- end}}

		// Handle retry logic
		if msg.Attempts < msg.MaxRetries {
			msg.Attempts++
			if err := w.queue.Retry(ctx, msg); err != nil {
				{{- if .Logging}}
				w.logger.Error("Failed to retry message", "id", msg.ID, "error", err)
				{{- end}}
			}
		} else {
			{{- if .Logging}}
			w.logger.Error("Job exceeded max retries", "id", msg.ID, "type", msg.Type)
			{{- end}}
			// Send to dead letter queue or handle failure
			if err := w.queue.Fail(ctx, msg, err); err != nil {
				{{- if .Logging}}
				w.logger.Error("Failed to handle job failure", "id", msg.ID, "error", err)
				{{- end}}
			}
		}
		return
	}

	{{- if .Logging}}
	w.logger.Info("Job completed successfully", 
		"id", msg.ID, 
		"type", msg.Type, 
		"duration", duration)
	{{- end}}

	// Acknowledge successful processing
	if err := w.queue.Ack(ctx, msg); err != nil {
		{{- if .Logging}}
		w.logger.Error("Failed to acknowledge message", "id", msg.ID, "error", err)
		{{- end}}
	}
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "worker", "worker.go"))
}

func (g *WorkerGenerator) generateJobHandler() error {
	template := `package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/models"
	"{{.Name}}/internal/queue"
	"{{.Name}}/internal/services"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
	{{- if .Logging}}
	"{{.Name}}/internal/logger"
	{{- end}}
)

type JobHandler struct {
	db      database.Database
	{{- if eq .Cache "redis"}}
	cache   *cache.Client
	{{- end}}
	{{- if .Logging}}
	logger  *logger.Logger
	{{- end}}
	service *services.JobService
}

type JobHandlerConfig struct {
	DB database.Database
	{{- if eq .Cache "redis"}}
	Cache *cache.Client
	{{- end}}
	{{- if .Logging}}
	Logger *logger.Logger
	{{- end}}
}

func NewJobHandler(cfg JobHandlerConfig) *JobHandler {
	service := services.NewJobService(cfg.DB{{- if eq .Cache "redis"}}, cfg.Cache{{- end}})
	
	return &JobHandler{
		db: cfg.DB,
		{{- if eq .Cache "redis"}}
		cache: cfg.Cache,
		{{- end}}
		{{- if .Logging}}
		logger: cfg.Logger,
		{{- end}}
		service: service,
	}
}

func (h *JobHandler) HandleJob(ctx context.Context, msg queue.Message) error {
	switch msg.Type {
	case models.JobTypeEmailSend:
		return h.handleEmailSend(ctx, msg)
	case models.JobTypeDataProcessing:
		return h.handleDataProcessing(ctx, msg)
	case models.JobTypeImageProcessing:
		return h.handleImageProcessing(ctx, msg)
	case models.JobTypeReportGeneration:
		return h.handleReportGeneration(ctx, msg)
	default:
		return fmt.Errorf("unknown job type: %s", msg.Type)
	}
}

func (h *JobHandler) handleEmailSend(ctx context.Context, msg queue.Message) error {
	var payload models.EmailJobPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal email job payload: %w", err)
	}

	{{- if .Logging}}
	h.logger.Info("Processing email job", 
		"to", payload.To, 
		"subject", payload.Subject)
	{{- end}}

	// Implement email sending logic here
	err := h.service.SendEmail(ctx, &payload)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (h *JobHandler) handleDataProcessing(ctx context.Context, msg queue.Message) error {
	var payload models.DataProcessingJobPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal data processing job payload: %w", err)
	}

	{{- if .Logging}}
	h.logger.Info("Processing data job", 
		"source", payload.Source, 
		"records", len(payload.Records))
	{{- end}}

	// Implement data processing logic here
	err := h.service.ProcessData(ctx, &payload)
	if err != nil {
		return fmt.Errorf("failed to process data: %w", err)
	}

	return nil
}

func (h *JobHandler) handleImageProcessing(ctx context.Context, msg queue.Message) error {
	var payload models.ImageProcessingJobPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal image processing job payload: %w", err)
	}

	{{- if .Logging}}
	h.logger.Info("Processing image job", 
		"source", payload.SourceURL, 
		"operations", len(payload.Operations))
	{{- end}}

	// Implement image processing logic here
	err := h.service.ProcessImage(ctx, &payload)
	if err != nil {
		return fmt.Errorf("failed to process image: %w", err)
	}

	return nil
}

func (h *JobHandler) handleReportGeneration(ctx context.Context, msg queue.Message) error {
	var payload models.ReportGenerationJobPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal report generation job payload: %w", err)
	}

	{{- if .Logging}}
	h.logger.Info("Processing report generation job", 
		"type", payload.ReportType, 
		"format", payload.Format)
	{{- end}}

	// Implement report generation logic here
	err := h.service.GenerateReport(ctx, &payload)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "handlers", "job_handler.go"))
}

func (g *WorkerGenerator) generateQueue() error {
	template := `package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"{{.Name}}/internal/config"
	{{- if eq .Queue "rabbitmq"}}
	"github.com/streadway/amqp"
	{{- else if eq .Queue "kafka"}}
	"github.com/segmentio/kafka-go"
	{{- else if eq .Queue "sqs"}}
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	{{- end}}
)

type Queue interface {
	Publish(ctx context.Context, message Message) error
	Consume(ctx context.Context) (<-chan Message, error)
	Ack(ctx context.Context, message Message) error
	Retry(ctx context.Context, message Message) error
	Fail(ctx context.Context, message Message, err error) error
	Close() error
}

type Message struct {
	ID          string          ` + "`json:\"id\"`" + `
	Type        string          ` + "`json:\"type\"`" + `
	Payload     json.RawMessage ` + "`json:\"payload\"`" + `
	Attempts    int             ` + "`json:\"attempts\"`" + `
	MaxRetries  int             ` + "`json:\"max_retries\"`" + `
	CreatedAt   time.Time       ` + "`json:\"created_at\"`" + `
	ScheduledAt *time.Time      ` + "`json:\"scheduled_at,omitempty\"`" + `
	
	// Internal fields for queue-specific data
	{{- if eq .Queue "rabbitmq"}}
	delivery *amqp.Delivery
	{{- else if eq .Queue "kafka"}}
	kafkaMessage *kafka.Message
	{{- else if eq .Queue "sqs"}}
	sqsMessage *sqs.Message
	receiptHandle *string
	{{- end}}
}

{{- if eq .Queue "rabbitmq"}}
type rabbitmqQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

func New(cfg *config.Config) (Queue, error) {
	conn, err := amqp.Dial(cfg.Queue.URL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare queue
	_, err = ch.QueueDeclare(
		cfg.Queue.QueueName, // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return nil, err
	}

	return &rabbitmqQueue{
		conn:      conn,
		channel:   ch,
		queueName: cfg.Queue.QueueName,
	}, nil
}

func (q *rabbitmqQueue) Publish(ctx context.Context, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return q.channel.Publish(
		"",           // exchange
		q.queueName,  // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (q *rabbitmqQueue) Consume(ctx context.Context) (<-chan Message, error) {
	msgs, err := q.channel.Consume(
		q.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return nil, err
	}

	messageChan := make(chan Message)

	go func() {
		defer close(messageChan)
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-msgs:
				if !ok {
					return
				}

				var message Message
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					delivery.Nack(false, false) // reject message
					continue
				}

				message.delivery = &delivery
				messageChan <- message
			}
		}
	}()

	return messageChan, nil
}

func (q *rabbitmqQueue) Ack(ctx context.Context, message Message) error {
	if message.delivery != nil {
		return message.delivery.Ack(false)
	}
	return nil
}

func (q *rabbitmqQueue) Retry(ctx context.Context, message Message) error {
	// Republish with delay or to retry queue
	return q.Publish(ctx, message)
}

func (q *rabbitmqQueue) Fail(ctx context.Context, message Message, err error) error {
	// Send to dead letter queue or log failure
	if message.delivery != nil {
		return message.delivery.Nack(false, false)
	}
	return nil
}

func (q *rabbitmqQueue) Close() error {
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}
	return nil
}

{{- else if eq .Queue "kafka"}}
type kafkaQueue struct {
	reader *kafka.Reader
	writer *kafka.Writer
	topic  string
}

func New(cfg *config.Config) (Queue, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Queue.Brokers,
		Topic:   cfg.Queue.Topic,
		GroupID: cfg.Queue.GroupID,
	})

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: cfg.Queue.Brokers,
		Topic:   cfg.Queue.Topic,
	})

	return &kafkaQueue{
		reader: reader,
		writer: writer,
		topic:  cfg.Queue.Topic,
	}, nil
}

func (q *kafkaQueue) Publish(ctx context.Context, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return q.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(message.ID),
		Value: body,
	})
}

func (q *kafkaQueue) Consume(ctx context.Context) (<-chan Message, error) {
	messageChan := make(chan Message)

	go func() {
		defer close(messageChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := q.reader.ReadMessage(ctx)
				if err != nil {
					continue
				}

				var message Message
				if err := json.Unmarshal(msg.Value, &message); err != nil {
					continue
				}

				message.kafkaMessage = &msg
				messageChan <- message
			}
		}
	}()

	return messageChan, nil
}

func (q *kafkaQueue) Ack(ctx context.Context, message Message) error {
	// Kafka handles commits automatically with the reader
	return nil
}

func (q *kafkaQueue) Retry(ctx context.Context, message Message) error {
	return q.Publish(ctx, message)
}

func (q *kafkaQueue) Fail(ctx context.Context, message Message, err error) error {
	// Log failure or send to dead letter topic
	return nil
}

func (q *kafkaQueue) Close() error {
	if q.reader != nil {
		q.reader.Close()
	}
	if q.writer != nil {
		q.writer.Close()
	}
	return nil
}

{{- else if eq .Queue "sqs"}}
type sqsQueue struct {
	client   *sqs.SQS
	queueURL string
}

func New(cfg *config.Config) (Queue, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Queue.Region),
	})
	if err != nil {
		return nil, err
	}

	return &sqsQueue{
		client:   sqs.New(sess),
		queueURL: cfg.Queue.QueueURL,
	}, nil
}

func (q *sqsQueue) Publish(ctx context.Context, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = q.client.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(body)),
	})

	return err
}

func (q *sqsQueue) Consume(ctx context.Context) (<-chan Message, error) {
	messageChan := make(chan Message)

	go func() {
		defer close(messageChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				result, err := q.client.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
					QueueUrl:            aws.String(q.queueURL),
					MaxNumberOfMessages: aws.Int64(10),
					WaitTimeSeconds:     aws.Int64(20),
				})
				if err != nil {
					continue
				}

				for _, sqsMsg := range result.Messages {
					var message Message
					if err := json.Unmarshal([]byte(*sqsMsg.Body), &message); err != nil {
						continue
					}

					message.sqsMessage = sqsMsg
					message.receiptHandle = sqsMsg.ReceiptHandle
					messageChan <- message
				}
			}
		}
	}()

	return messageChan, nil
}

func (q *sqsQueue) Ack(ctx context.Context, message Message) error {
	if message.receiptHandle != nil {
		_, err := q.client.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(q.queueURL),
			ReceiptHandle: message.receiptHandle,
		})
		return err
	}
	return nil
}

func (q *sqsQueue) Retry(ctx context.Context, message Message) error {
	return q.Publish(ctx, message)
}

func (q *sqsQueue) Fail(ctx context.Context, message Message, err error) error {
	// Send to dead letter queue or log failure
	return nil
}

func (q *sqsQueue) Close() error {
	return nil
}
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "queue", "queue.go"))
}

func (g *WorkerGenerator) generateModels() error {
	template := `package models

import (
	"time"
)

// Job types
const (
	JobTypeEmailSend        = "email_send"
	JobTypeDataProcessing   = "data_processing"
	JobTypeImageProcessing  = "image_processing"
	JobTypeReportGeneration = "report_generation"
)

// Job status
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusRetrying   = "retrying"
)

// Job represents a job record in the database
type Job struct {
	ID          int       ` + "`json:\"id\" db:\"id\"`" + `
	Type        string    ` + "`json:\"type\" db:\"type\"`" + `
	Status      string    ` + "`json:\"status\" db:\"status\"`" + `
	Payload     string    ` + "`json:\"payload\" db:\"payload\"`" + `
	Attempts    int       ` + "`json:\"attempts\" db:\"attempts\"`" + `
	MaxRetries  int       ` + "`json:\"max_retries\" db:\"max_retries\"`" + `
	Error       *string   ` + "`json:\"error,omitempty\" db:\"error\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt   time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
	StartedAt   *time.Time ` + "`json:\"started_at,omitempty\" db:\"started_at\"`" + `
	CompletedAt *time.Time ` + "`json:\"completed_at,omitempty\" db:\"completed_at\"`" + `
	ScheduledAt *time.Time ` + "`json:\"scheduled_at,omitempty\" db:\"scheduled_at\"`" + `
}

// Email job payload
type EmailJobPayload struct {
	To      string            ` + "`json:\"to\"`" + `
	From    string            ` + "`json:\"from\"`" + `
	Subject string            ` + "`json:\"subject\"`" + `
	Body    string            ` + "`json:\"body\"`" + `
	HTML    bool              ` + "`json:\"html\"`" + `
	Headers map[string]string ` + "`json:\"headers,omitempty\"`" + `
}

// Data processing job payload
type DataProcessingJobPayload struct {
	Source      string                 ` + "`json:\"source\"`" + `
	Destination string                 ` + "`json:\"destination\"`" + `
	Records     []map[string]interface{} ` + "`json:\"records\"`" + `
	Options     map[string]interface{} ` + "`json:\"options,omitempty\"`" + `
}

// Image processing job payload
type ImageProcessingJobPayload struct {
	SourceURL   string                 ` + "`json:\"source_url\"`" + `
	TargetURL   string                 ` + "`json:\"target_url\"`" + `
	Operations  []ImageOperation       ` + "`json:\"operations\"`" + `
	Options     map[string]interface{} ` + "`json:\"options,omitempty\"`" + `
}

type ImageOperation struct {
	Type   string                 ` + "`json:\"type\"`" + `
	Params map[string]interface{} ` + "`json:\"params\"`" + `
}

// Report generation job payload
type ReportGenerationJobPayload struct {
	ReportType string                 ` + "`json:\"report_type\"`" + `
	Format     string                 ` + "`json:\"format\"`" + `
	Parameters map[string]interface{} ` + "`json:\"parameters\"`" + `
	Recipients []string               ` + "`json:\"recipients,omitempty\"`" + `
	Template   string                 ` + "`json:\"template,omitempty\"`" + `
}

// Job statistics
type JobStats struct {
	Total      int64 ` + "`json:\"total\"`" + `
	Pending    int64 ` + "`json:\"pending\"`" + `
	Processing int64 ` + "`json:\"processing\"`" + `
	Completed  int64 ` + "`json:\"completed\"`" + `
	Failed     int64 ` + "`json:\"failed\"`" + `
	Retrying   int64 ` + "`json:\"retrying\"`" + `
}

// Worker statistics
type WorkerStats struct {
	ActiveWorkers int           ` + "`json:\"active_workers\"`" + `
	TotalJobs     int64         ` + "`json:\"total_jobs\"`" + `
	Uptime        time.Duration ` + "`json:\"uptime\"`" + `
	JobStats      JobStats      ` + "`json:\"job_stats\"`" + `
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "models", "models.go"))
}

func (g *WorkerGenerator) generateServices() error {
	template := `package services

import (
	"context"
	"fmt"

	"{{.Name}}/internal/database"
	"{{.Name}}/internal/models"
	{{- if eq .Cache "redis"}}
	"{{.Name}}/internal/cache"
	{{- end}}
)

type JobService struct {
	db database.Database
	{{- if eq .Cache "redis"}}
	cache *cache.Client
	{{- end}}
}

func NewJobService(db database.Database{{- if eq .Cache "redis"}}, cache *cache.Client{{- end}}) *JobService {
	return &JobService{
		db: db,
		{{- if eq .Cache "redis"}}
		cache: cache,
		{{- end}}
	}
}

func (s *JobService) SendEmail(ctx context.Context, payload *models.EmailJobPayload) error {
	// Implement email sending logic
	// This could use services like SendGrid, AWS SES, SMTP, etc.
	
	fmt.Printf("Sending email to %s with subject: %s\n", payload.To, payload.Subject)
	
	return nil
}

func (s *JobService) ProcessData(ctx context.Context, payload *models.DataProcessingJobPayload) error {
	// Implement data processing logic
	// This could involve ETL operations, data transformation, etc.
	
	fmt.Printf("Processing %d records from %s to %s\n", 
		len(payload.Records), payload.Source, payload.Destination)
	
	for _, record := range payload.Records {
		// Process each record
		_ = record
	}
	
	return nil
}

func (s *JobService) ProcessImage(ctx context.Context, payload *models.ImageProcessingJobPayload) error {
	// Implement image processing logic
	// This could use libraries like imaging, gocv, etc.
	
	fmt.Printf("Processing image from %s with %d operations\n", 
		payload.SourceURL, len(payload.Operations))
	
	for _, operation := range payload.Operations {
		fmt.Printf("Applying operation: %s\n", operation.Type)
		// Apply operation based on type
		switch operation.Type {
		case "resize":
			// Resize image
		case "crop":
			// Crop image
		case "filter":
			// Apply filter
		default:
			return fmt.Errorf("unknown operation: %s", operation.Type)
		}
	}
	
	return nil
}

func (s *JobService) GenerateReport(ctx context.Context, payload *models.ReportGenerationJobPayload) error {
	// Implement report generation logic
	// This could use libraries for PDF, Excel, CSV generation
	
	fmt.Printf("Generating %s report in %s format\n", 
		payload.ReportType, payload.Format)
	
	switch payload.Format {
	case "pdf":
		return s.generatePDFReport(ctx, payload)
	case "excel":
		return s.generateExcelReport(ctx, payload)
	case "csv":
		return s.generateCSVReport(ctx, payload)
	default:
		return fmt.Errorf("unsupported format: %s", payload.Format)
	}
}

func (s *JobService) generatePDFReport(ctx context.Context, payload *models.ReportGenerationJobPayload) error {
	// Implement PDF report generation
	fmt.Printf("Generating PDF report for %s\n", payload.ReportType)
	return nil
}

func (s *JobService) generateExcelReport(ctx context.Context, payload *models.ReportGenerationJobPayload) error {
	// Implement Excel report generation
	fmt.Printf("Generating Excel report for %s\n", payload.ReportType)
	return nil
}

func (s *JobService) generateCSVReport(ctx context.Context, payload *models.ReportGenerationJobPayload) error {
	// Implement CSV report generation
	fmt.Printf("Generating CSV report for %s\n", payload.ReportType)
	return nil
}

// Job management methods
func (s *JobService) CreateJob(ctx context.Context, job *models.Job) error {
	// Save job to database
	// Implementation depends on database type
	fmt.Printf("Creating job of type %s\n", job.Type)
	return nil
}

func (s *JobService) UpdateJobStatus(ctx context.Context, jobID int, status string) error {
	// Update job status in database
	fmt.Printf("Updating job %d status to %s\n", jobID, status)
	return nil
}

func (s *JobService) GetJobStats(ctx context.Context) (*models.JobStats, error) {
	// Get job statistics from database
	return &models.JobStats{
		Total:      100,
		Pending:    10,
		Processing: 5,
		Completed:  80,
		Failed:     3,
		Retrying:   2,
	}, nil
}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "services", "job_service.go"))
}

func (g *WorkerGenerator) generateDatabase() error {
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
	// Create jobs table
	createJobsTable := ` + "`" + `
		CREATE TABLE IF NOT EXISTS jobs (
			id SERIAL PRIMARY KEY,
			type VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			payload TEXT NOT NULL,
			attempts INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			scheduled_at TIMESTAMP
		);
	` + "`" + `

	{{- if eq .Database "sqlite"}}
	createJobsTable = ` + "`" + `
		CREATE TABLE IF NOT EXISTS jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			payload TEXT NOT NULL,
			attempts INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			scheduled_at DATETIME
		);
	` + "`" + `
	{{- end}}

	if _, err := d.db.Exec(createJobsTable); err != nil {
		return fmt.Errorf("failed to create jobs table: %w", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);",
		"CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);",
		"CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_at ON jobs(scheduled_at);",
	}

	for _, index := range indexes {
		if _, err := d.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
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
	// Create indexes for jobs collection
	ctx := context.Background()
	jobsCollection := d.db.Collection("jobs")
	
	// Create indexes for better query performance
	indexModels := []mongo.IndexModel{
		{Keys: map[string]int{"status": 1}},
		{Keys: map[string]int{"type": 1}},
		{Keys: map[string]int{"created_at": 1}},
		{Keys: map[string]int{"scheduled_at": 1}},
	}
	
	_, err := jobsCollection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "internal", "database", "database.go"))
}

func (g *WorkerGenerator) generateLogger() error {
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

func (g *WorkerGenerator) generateCache() error {
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

func (g *WorkerGenerator) generateEnvFile() error {
	template := `# Worker Configuration
{{.Name | upper}}_WORKER_COUNT=5

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

# Queue Configuration
{{- if eq .Queue "rabbitmq"}}
{{.Name | upper}}_QUEUE_URL=amqp://localhost:5672
{{.Name | upper}}_QUEUE_QUEUE_NAME={{.Name}}_jobs
{{- else if eq .Queue "kafka"}}
{{.Name | upper}}_QUEUE_BROKERS=localhost:9092
{{.Name | upper}}_QUEUE_TOPIC={{.Name}}_jobs
{{.Name | upper}}_QUEUE_GROUP_ID={{.Name}}_worker
{{- else if eq .Queue "sqs"}}
{{.Name | upper}}_QUEUE_REGION=us-east-1
{{.Name | upper}}_QUEUE_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/123456789012/{{.Name}}-jobs
{{- end}}

{{- if eq .Cache "redis"}}
# Redis Configuration
{{.Name | upper}}_REDIS_HOST=localhost
{{.Name | upper}}_REDIS_PORT=6379
{{.Name | upper}}_REDIS_PASSWORD=
{{.Name | upper}}_REDIS_DB=0
{{- end}}

{{- if .Logging}}
# Logging Configuration
{{.Name | upper}}_LOG_LEVEL=info
{{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, ".env.example"))
}

func (g *WorkerGenerator) generateDockerfile() error {
	if !g.config.Docker {
		return nil
	}

	template := `# Build stage
FROM golang:{{.GoVersion}}-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
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

# Run the binary
CMD ["./main"]
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "Dockerfile"))
}

func (g *WorkerGenerator) generateDockerCompose() error {
	if !g.config.Docker {
		return nil
	}

	template := `version: '3.8'

services:
  {{.Name}}:
    build: .
    environment:
      - {{.Name | upper}}_WORKER_COUNT=5
      {{- if eq .Database "postgres"}}
      - {{.Name | upper}}_DATABASE_HOST=postgres
      - {{.Name | upper}}_DATABASE_PASSWORD=password
      {{- else if eq .Database "mysql"}}
      - {{.Name | upper}}_DATABASE_HOST=mysql
      - {{.Name | upper}}_DATABASE_PASSWORD=password
      {{- else if eq .Database "mongodb"}}
      - {{.Name | upper}}_DATABASE_URI=mongodb://mongodb:27017
      {{- end}}
      {{- if eq .Queue "rabbitmq"}}
      - {{.Name | upper}}_QUEUE_URL=amqp://rabbitmq:5672
      {{- else if eq .Queue "kafka"}}
      - {{.Name | upper}}_QUEUE_BROKERS=kafka:9092
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
      {{- if eq .Queue "rabbitmq"}}
      - rabbitmq
      {{- else if eq .Queue "kafka"}}
      - kafka
      - zookeeper
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

  {{- if eq .Queue "rabbitmq"}}
  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    restart: unless-stopped
  {{- else if eq .Queue "kafka"}}
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    restart: unless-stopped

  kafka:
    image: confluentinc/cp-kafka:latest
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    volumes:
      - kafka_data:/var/lib/kafka/data
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
  {{- if eq .Queue "rabbitmq"}}
  rabbitmq_data:
  {{- else if eq .Queue "kafka"}}
  kafka_data:
  {{- end}}
  {{- if eq .Cache "redis"}}
  redis_data:
  {{- end}}
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "docker-compose.yml"))
}

func (g *WorkerGenerator) generateMakefile() error {
	template := `# {{.Name}} Worker Makefile

.PHONY: help build run test clean docker-build docker-run docker-compose-up docker-compose-down

# Variables
APP_NAME={{.Name}}
DOCKER_IMAGE={{.Name}}:latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $1, $2}' $(MAKEFILE_LIST)

build: ## Build the worker application
	go build -o bin/$(APP_NAME) .

run: ## Run the worker application
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
	docker run $(DOCKER_IMAGE)

docker-compose-up: ## Start services with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	docker-compose down

docker-compose-logs: ## View logs from docker-compose
	docker-compose logs -f
{{- end}}

migrate-up: ## Run database migrations up
	# Add your migration command here

migrate-down: ## Run database migrations down
	# Add your migration command here

dev: ## Run in development mode with hot reload
	air

{{- if eq .Queue "rabbitmq"}}
queue-status: ## Check RabbitMQ queue status
	docker-compose exec rabbitmq rabbitmqctl list_queues
{{- else if eq .Queue "kafka"}}
queue-status: ## Check Kafka topic status
	docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list
{{- end}}

worker-stats: ## Show worker statistics
	# Add worker statistics command here
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "Makefile"))
}

func (g *WorkerGenerator) generateReadme() error {
	template := `# {{.Name}} Worker

A production-ready background worker built with Go, featuring:

{{- if eq .Database "postgres"}}
- PostgreSQL database
{{- else if eq .Database "mysql"}}
- MySQL database
{{- else if eq .Database "sqlite"}}
- SQLite database
{{- else if eq .Database "mongodb"}}
- MongoDB database
{{- end}}
{{- if eq .Queue "rabbitmq"}}
- RabbitMQ message queue
{{- else if eq .Queue "kafka"}}
- Apache Kafka message queue
{{- else if eq .Queue "sqs"}}
- AWS SQS message queue
{{- end}}
{{- if eq .Cache "redis"}}
- Redis caching
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
{{- if eq .Queue "rabbitmq"}}
- RabbitMQ
{{- else if eq .Queue "kafka"}}
- Apache Kafka with Zookeeper
{{- else if eq .Queue "sqs"}}
- AWS Account with SQS access
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

4. Run database migrations:
` + "```bash" + `
make migrate-up
` + "```" + `

5. Start the worker:
` + "```bash" + `
go run main.go
` + "```" + `

### Using Docker

1. Start all services with Docker Compose:
` + "```bash" + `
docker-compose up -d
` + "```" + `

2. View logs:
` + "```bash" + `
docker-compose logs -f {{.Name}}
` + "```" + `

3. Stop services:
` + "```bash" + `
docker-compose down
` + "```" + `

## Job Types

The worker supports the following job types:

### Email Jobs (` + "`email_send`" + `)
Sends emails using configured email service.

### Data Processing Jobs (` + "`data_processing`" + `)
Processes data records with transformations.

### Image Processing Jobs (` + "`image_processing`" + `)
Processes images with various operations.

### Report Generation Jobs (` + "`report_generation`" + `)
Generates reports in various formats.

## Project Structure

` + "```" + `
{{.Name}}/
├── main.go                 # Application entry point
├── go.mod                  # Go module file
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
    ├── handlers/          # Job handlers
    ├── models/            # Data models
    ├── queue/             # Message queue implementation
    ├── services/          # Business logic
    ├── worker/            # Worker implementation
    {{- if .Logging}}
    ├── logger/            # Logging utilities
    {{- end}}
    {{- if eq .Cache "redis"}}
    └── cache/             # Cache utilities
    {{- end}}
` + "```" + `

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

This project is licensed under the MIT License.
`

	return createFileFromTemplate(template, g.config, filepath.Join(g.config.Name, "README.md"))
}

func (g *WorkerGenerator) generateTests() error {
	if !g.config.Testing {
		return nil
	}

	handlerTestTemplate := `package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"{{.Name}}/internal/models"
	"{{.Name}}/internal/queue"
)

type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) SendEmail(ctx context.Context, payload *models.EmailJobPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockJobService) ProcessData(ctx context.Context, payload *models.DataProcessingJobPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockJobService) ProcessImage(ctx context.Context, payload *models.ImageProcessingJobPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func (m *MockJobService) GenerateReport(ctx context.Context, payload *models.ReportGenerationJobPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func TestHandleEmailSendJob(t *testing.T) {
	// Create mock service
	mockService := &MockJobService{}
	
	// Create handler with mock
	handler := &JobHandler{service: mockService}
	
	// Prepare test data
	payload := models.EmailJobPayload{
		To:      "test@example.com",
		From:    "sender@example.com",
		Subject: "Test Email",
		Body:    "This is a test",
		HTML:    false,
	}
	
	payloadBytes, _ := json.Marshal(payload)
	message := queue.Message{
		ID:      "test-123",
		Type:    models.JobTypeEmailSend,
		Payload: payloadBytes,
	}
	
	// Set expectations
	mockService.On("SendEmail", mock.Anything, &payload).Return(nil)
	
	// Execute test
	err := handler.HandleJob(context.Background(), message)
	
	// Assertions
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestHandleUnknownJobType(t *testing.T) {
	handler := &JobHandler{}
	
	message := queue.Message{
		ID:   "test-789",
		Type: "unknown_job_type",
	}
	
	err := handler.HandleJob(context.Background(), message)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown job type")
}
`

	return createFileFromTemplate(handlerTestTemplate, g.config, filepath.Join(g.config.Name, "internal", "handlers", "job_handler_test.go"))
}

func (g *WorkerGenerator) generateKubernetes() error {
	if !g.config.Kubernetes {
		return nil
	}

	deploymentTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}-worker
  labels:
    app: {{.Name}}-worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{.Name}}-worker
  template:
    metadata:
      labels:
        app: {{.Name}}-worker
    spec:
      containers:
      - name: {{.Name}}-worker
        image: {{.Name}}:latest
        env:
        - name: {{.Name | upper}}_WORKER_COUNT
          value: "5"
        {{- if eq .Database "postgres"}}
        - name: {{.Name | upper}}_DATABASE_HOST
          value: "postgres"
        - name: {{.Name | upper}}_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{.Name}}-secrets
              key: database-password
        {{- end}}
        {{- if eq .Queue "rabbitmq"}}
        - name: {{.Name | upper}}_QUEUE_URL
          value: "amqp://rabbitmq:5672"
        {{- end}}
        {{- if eq .Cache "redis"}}
        - name: {{.Name | upper}}_REDIS_HOST
          value: "redis"
        {{- end}}
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}-worker-config
data:
  config.yaml: |
    worker:
      count: 5
    {{- if .Logging}}
    log_level: "info"
    {{- end}}
`

	return createFileFromTemplate(deploymentTemplate, g.config, filepath.Join(g.config.Name, "k8s", "deployment.yaml"))
}
