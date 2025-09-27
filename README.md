# Go Scaffolding Tool

A powerful and easy-to-use CLI tool for generating production-ready Go projects with best practices built-in.

---

## Features

### 📋 Project Types
- **REST API**: HTTP server using Gin framework  
- **Worker**: Background job processor

### 💾 Supported Databases
- **PostgreSQL**: Robust relational database  
- **MySQL**: Popular relational database  
- **SQLite**: Lightweight file-based database  
- **MongoDB**: Document-oriented NoSQL database

### 📨 Message Queues (for Workers)
- **RabbitMQ**: Feature-rich message broker  
- **Apache Kafka**: High-performance streaming platform  
- **AWS SQS**: Managed queue service

### ⚡ Caching Systems
- **Redis**: In-memory data structure store  
- **Memcached**: High-performance caching system

### Additional Features
- 🔐 JWT Authentication  
- 📄 Structured logging with Logrus  
- 📊 Prometheus metrics  
- 🐳 Full Docker configuration  
- ☸️ Kubernetes manifests  
- 🧪 Testing setup with Testify  
- 📚 Swagger API documentation  
- ⚡ gRPC support (APIs)  
- 🔍 GraphQL support (APIs)  
- 🔌 WebSocket support (APIs)

---

## Installation

### From Source
```bash
git clone <repository-url>
cd go-scaffolder
go build -o go-scaffolder .
sudo mv go-scaffolder /usr/local/bin/
```

### Run Directly
```bash
go run . --help
```

---

## Usage

### Interactive Mode (Recommended)
```bash
go-scaffolder interactive
```

### Quick Generation
```bash
# Basic API with PostgreSQL
go-scaffolder generate -n my-api -t api -d postgres

# Worker with RabbitMQ and Redis
go-scaffolder generate -n my-worker -t worker -d postgres -q rabbitmq -c redis
```

### Initialize in Current Directory
```bash
mkdir my-project && cd my-project
go-scaffolder init my-project
```

### List Available Templates
```bash
go-scaffolder list-templates
```

---

## Configuration Options

### Main Flags
```bash
-n, --name string       Project name
-t, --type string       Project type (api/worker)
-d, --database string   Database type
-c, --cache string      Cache type
-q, --queue string      Queue type (for workers)
-p, --port string       Default port (APIs)
```

### Feature Flags
```bash
-a, --auth             Include authentication
-l, --logging          Include structured logging
-m, --metrics          Include metrics
--docker               Include Docker configuration
-k, --k8s              Include Kubernetes manifests
--testing               Include testing setup
-s, --swagger          Include Swagger documentation
-g, --grpc             Include gRPC support
--graphql               Include GraphQL support
-w, --websocket        Include WebSocket support
```

---

## Generated Project Structure

### REST API
```
my-api/
├── main.go
├── go.mod
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── README.md
└── internal/
    ├── config/
    ├── database/
    ├── handlers/
    ├── middleware/
    ├── models/
    ├── repository/
    ├── server/
    ├── services/
    ├── logger/
    └── cache/
```

### Worker
```
my-worker/
├── main.go
├── go.mod
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── README.md
└── internal/
    ├── config/
    ├── database/
    ├── handlers/
    ├── models/
    ├── queue/
    ├── services/
    ├── worker/
    ├── logger/
    └── cache/
```

---

## Usage Examples

### Full REST API
```bash
go-scaffolder generate \
  --name my-api \
  --type api \
  --database postgres \
  --cache redis \
  --auth \
  --logging \
  --metrics \
  --docker \
  --swagger \
  --testing \
  --port 8080
```

### Image Processing Worker
```bash
go-scaffolder generate \
  --name image-processor \
  --type worker \
  --database mongodb \
  --queue rabbitmq \
  --cache redis \
  --logging \
  --metrics \
  --docker \
  --k8s \
  --testing
```

### gRPC Microservice
```bash
go-scaffolder generate \
  --name user-service \
  --type api \
  --database postgres \
  --cache redis \
  --auth \
  --grpc \
  --logging \
  --metrics \
  --docker \
  --testing
```

---

## First Steps After Generation

### For APIs
```bash
cd my-api
go mod tidy
docker-compose up -d
go run main.go

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/users
# Swagger: http://localhost:8080/swagger/index.html
```

### For Workers
```bash
cd my-worker
go mod tidy
docker-compose up -d
go run main.go

# Monitor logs
docker-compose logs -f my-worker
```

---

## Available Make Commands
```bash
make help              # Show help
make build             # Build application
make run               # Run application
make test              # Run tests
make test-coverage     # Tests with coverage
make docker-build      # Build Docker image
make docker-run        # Run container
make docker-compose-up # Start services
make fmt               # Format code
make lint              # Run linter
```

---

## Customization

### Add New Templates
1. Create a new generator in the main file  
2. Implement the required interface  
3. Add specific generation logic  
4. Update CLI options  

### Modify Existing Templates
- Templates use Go's template system with custom helper functions.

---

## Contributing
1. Fork the repository  
2. Create a branch for your feature  
   ```bash
   git checkout -b feature/amazing-feature
   ```  
3. Commit your changes  
   ```bash
   git commit -m 'Add some amazing feature'
   ```  
4. Push the branch  
   ```bash
   git push origin feature/amazing-feature
   ```  
5. Open a Pull Request  

**Contribution Guidelines**
- Follow Go code conventions  
- Add tests for new features  
- Update documentation  
- Ensure all tests pass

---

## TODO
- Support more databases (CockroachDB, TimescaleDB)  
- Microservice templates  
- CI/CD integration (GitHub Actions, GitLab CI)  
- OpenAPI 3.0 support  
- CLI app templates  
- Observability integration (Jaeger, Zipkin)  
- Event support (NATS, EventStore)  
- Industry-specific templates

---

## Reporting Issues
If you find a problem or have a suggestion:
1. Check for existing similar issues  
2. Create a new issue with:
   - Clear description of the problem  
   - Steps to reproduce  
   - Go version and OS  
   - Command and configuration used

---

## License
This project is licensed under the MIT License. See the LICENSE file for details.
