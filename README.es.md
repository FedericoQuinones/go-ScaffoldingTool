# Go Scaffolding Tool

Una herramienta de línea de comandos potente y fácil de usar para generar proyectos Go de grado de producción con las mejores prácticas integradas.

---

## Características

### 📋 Tipos de Proyecto
- **API REST**: Servidor HTTP con Gin framework  
- **Worker**: Procesador de trabajos en segundo plano

### 💾 Bases de Datos Soportadas
- **PostgreSQL**: Base de datos relacional robusta  
- **MySQL**: Base de datos relacional popular  
- **SQLite**: Base de datos ligera basada en archivos  
- **MongoDB**: Base de datos NoSQL orientada a documentos

### 📨 Colas de Mensajes (para Workers)
- **RabbitMQ**: Broker de mensajes rico en características  
- **Apache Kafka**: Plataforma de streaming de alto rendimiento  
- **AWS SQS**: Servicio de colas administrado

### ⚡ Sistemas de Cache
- **Redis**: Almacén de estructura de datos en memoria  
- **Memcached**: Sistema de caché de alto rendimiento

### Características Adicionales
- 🔐 Autenticación JWT  
- 📄 Logging estructurado con Logrus  
- 📊 Métricas Prometheus  
- 🐳 Configuración Docker completa  
- ☸️ Manifiestos Kubernetes  
- 🧪 Setup de testing con Testify  
- 📚 Documentación Swagger (APIs)  
- ⚡ Soporte gRPC (APIs)  
- 🔍 Soporte GraphQL (APIs)  
- 🔌 Soporte WebSocket (APIs)

---

## Instalación

### Desde el código fuente
```bash
git clone <repository-url>
cd go-scaffolder
go build -o go-scaffolder .
sudo mv go-scaffolder /usr/local/bin/
```

### Usar directamente
```bash
go run . --help
```

---

## Uso

### Modo Interactivo (Recomendado)
```bash
go-scaffolder interactive
```

### Generación Rápida
```bash
# API básica con PostgreSQL
go-scaffolder generate -n my-api -t api -d postgres

# Worker con RabbitMQ y Redis
go-scaffolder generate -n my-worker -t worker -d postgres -q rabbitmq -c redis
```

### Inicializar en directorio actual
```bash
mkdir my-project && cd my-project
go-scaffolder init my-project
```

### Ver templates disponibles
```bash
go-scaffolder list-templates
```

---

## 🔧 Opciones de Configuración

### Flags Principales
```bash
-n, --name string       Nombre del proyecto
-t, --type string       Tipo de proyecto (api/worker)
-d, --database string   Tipo de base de datos
-c, --cache string      Tipo de cache
-q, --queue string      Tipo de cola (para workers)
-p, --port string       Puerto por defecto (APIs)
```

### Flags de Características
```bash
-a, --auth             Incluir autenticación
-l, --logging          Incluir logging estructurado
-m, --metrics          Incluir métricas
--docker               Incluir configuración Docker
-k, --k8s              Incluir manifiestos Kubernetes
--testing               Incluir setup de testing
-s, --swagger          Incluir documentación Swagger
-g, --grpc             Incluir soporte gRPC
--graphql               Incluir soporte GraphQL
-w, --websocket        Incluir soporte WebSocket
```

---

## Estructura de Proyecto Generada

### API REST
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

## Ejemplos de Uso

### API REST Completa
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

### Worker de Procesamiento de Imágenes
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

### Microservicio con gRPC
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

## 🏃‍♂️ Primeros Pasos Después de la Generación

### Para APIs
```bash
cd my-api
go mod tidy
docker-compose up -d
go run main.go

# Probar endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/users
# Swagger: http://localhost:8080/swagger/index.html
```

### Para Workers
```bash
cd my-worker
go mod tidy
docker-compose up -d
go run main.go

# Monitorear logs
docker-compose logs -f my-worker
```

---

## Comandos Make Disponibles
```bash
make help              # Mostrar ayuda
make build             # Construir aplicación
make run               # Ejecutar aplicación
make test              # Ejecutar tests
make test-coverage     # Tests con cobertura
make docker-build      # Construir imagen Docker
make docker-run        # Ejecutar contenedor
make docker-compose-up # Iniciar servicios
make fmt               # Formatear código
make lint              # Ejecutar linter
```

---

## Personalización

### Agregar Nuevos Templates
1. Crear un nuevo generador en el archivo principal  
2. Implementar la interfaz requerida  
3. Agregar la lógica de generación específica  
4. Actualizar las opciones del CLI  

### Modificar Templates Existentes
- Los templates utilizan el sistema de templates de Go con funciones auxiliares personalizadas.

---

## Contribuir
1. Fork el repositorio  
2. Crear una rama para tu característica  
   ```bash
   git checkout -b feature/amazing-feature
   ```  
3. Commit tus cambios  
   ```bash
   git commit -m 'Add some amazing feature'
   ```  
4. Push a la rama  
   ```bash
   git push origin feature/amazing-feature
   ```  
5. Abrir un Pull Request  

**Guidelines de Contribución**
- Seguir las convenciones de código Go  
- Agregar tests para nuevas características  
- Actualizar la documentación  
- Asegurarse que todos los tests pasen

---

## TODO
- Soporte para más bases de datos (CockroachDB, TimescaleDB)  
- Templates para microservicios  
- Integración con CI/CD (GitHub Actions, GitLab CI)  
- Soporte para OpenAPI 3.0  
- Templates para aplicaciones CLI  
- Integración con observabilidad (Jaeger, Zipkin)  
- Soporte para eventos (NATS, EventStore)  
- Templates específicos por industria

---

## 🐛 Reportar Problemas
Si encuentra algún problema o tiene alguna sugerencia:
1. Comprueba si existen problemas similares  
2. Crear un nuevo problema con:
   - Descripción clara del problema  
   - Pasos para reproducir  
   - Versión de Go y sistema operativo  
   - Comando y configuración utilizados

---

## License
Este proyecto está licenciado bajo la Licencia MIT. Consulte el archivo de LICENCIA para más detalles.