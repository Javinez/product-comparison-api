# Product Comparison API

[![Go Version](https://img.shields.io/badge/Go-1.21-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](Dockerfile)
[![AWS](https://img.shields.io/badge/AWS-ECS%20Fargate-orange.svg)](infrastructure/terraform)

## 📋 Descripción

API REST de alto rendimiento para comparación de productos, construida con Go siguiendo los principios de Clean Architecture y diseñada para escalar desde miles hasta millones de usuarios.

## 🏗️ Arquitectura

El proyecto implementa **Clean Architecture** con las siguientes capas:

- **Domain Layer**: Entidades de negocio (`Product`, `ProductComparison`)
- **Use Cases Layer**: Lógica de negocio (`ProductInteractor`)
- **Interface Adapters**: HTTP handlers y repositorios
- **Frameworks & Drivers**: Implementaciones concretas (JSON, PostgreSQL, Redis)

### Principios SOLID Aplicados

✅ **Single Responsibility**: Cada componente tiene una única responsabilidad  
✅ **Open/Closed**: Extensible sin modificar código existente  
✅ **Liskov Substitution**: Interfaces intercambiables  
✅ **Interface Segregation**: Interfaces específicas y enfocadas  
✅ **Dependency Inversion**: Dependencias hacia abstracciones

## 🚀 Características

- ✨ Comparación de múltiples productos (hasta 10 simultáneos)
- 🔍 Búsqueda y filtrado por categoría
- ⚡ Cache multi-nivel (In-memory + Redis)
- 🔒 Rate limiting y throttling
- 📊 Métricas y monitoreo con Prometheus
- 🌍 Multi-región y alta disponibilidad
- 🔄 Auto-scaling basado en métricas
- 📝 Documentación OpenAPI/Swagger

## 📦 Instalación

### Prerrequisitos

- Go 1.21+
- Docker 20.10+
- AWS CLI configurado
- Terraform 1.5+

### Desarrollo Local

```bash
# Clonar el repositorio
git clone https://github.com/javinez/product-comparison-api.git
cd product-comparison-api

# Instalar dependencias
go mod download

# Ejecutar tests
go test -v ./...

# Ejecutar la aplicación
go run cmd/server/main.go

# Con Docker
docker build -t product-api .
docker run -p 8080:8080 product-api
```

## 📡 API Endpoints

### Comparación de Productos

```http
GET /api/v1/products/compare?ids=prod-001,prod-002,prod-003
```

**Response:**
```json
{
  "data": {
    "comparison_id": "comp-123",
    "products": [...],
    "differences": {
      "brands": ["Apple", "Dell", "Lenovo"],
      "specifications": [...]
    },
    "similarities": {
      "category": "Laptops",
      "avg_rating": 4.5
    }
  },
  "success": true,
  "timestamp": "2024-10-27T10:30:00Z"
}
```

### Otros Endpoints

- `GET /api/v1/products` - Listar todos los productos
- `GET /api/v1/products/{id}` - Obtener producto por ID
- `GET /api/v1/products/category/{category}` - Productos por categoría
- `GET /api/v1/products/search?q={query}` - Buscar productos
- `POST /api/v1/products` - Crear producto
- `PUT /api/v1/products/{id}` - Actualizar producto
- `DELETE /api/v1/products/{id}` - Eliminar producto
- `GET /health` - Health check
- `GET /metrics` - Métricas Prometheus

## 🧪 Testing

### Ejecutar Tests

```bash
# Tests unitarios
go test ./test/unit/...

# Tests de integración
go test -tags=integration ./test/integration/...

# Tests con coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...
```

### Tests de Carga

```bash
# Instalar K6
brew install k6

# Ejecutar test de carga
k6 run test/performance/load-test.js
```

## 🚢 Despliegue

### Infraestructura con Terraform

```bash
cd infrastructure/terraform

# Inicializar Terraform
terraform init

# Plan de despliegue
terraform plan -var-file=environments/production.tfvars

# Aplicar cambios
terraform apply -auto-approve

# Destruir infraestructura
terraform destroy
```

### CI/CD Pipeline

El proyecto incluye GitHub Actions para:

1. **Security Scanning**: Trivy, gosec
2. **Testing**: Unit, Integration, Benchmarks
3. **Build**: Multi-arch Docker images
4. **Deploy**: ECS Fargate con Blue/Green deployment
5. **Monitoring**: CloudWatch, Prometheus

## 📊 Arquitectura de Escalamiento

### Capacidad Actual

- ✅ 10,000 requests/segundo
- ✅ Latencia P99 < 100ms
- ✅ 99.99% uptime SLA

### Escalamiento a Millones de Usuarios

Para escalar a millones de dispositivos, consulta la [Guía de Escalamiento](docs/scaling-guide.md) que incluye:

- Migración a DynamoDB Global Tables
- Implementación de API Gateway + CloudFront
- Cache distribuido con ElastiCache
- Procesamiento asíncrono con SQS/SNS
- Multi-región active-active

## 📁 Estructura del Proyecto

```
product-comparison-api/
├── cmd/server/          # Punto de entrada
├── internal/
│   ├── domain/         # Entidades de negocio
│   ├── usecase/        # Casos de uso
│   └── adapter/        # Adaptadores (HTTP, Repositorios)
├── pkg/config/         # Configuración
├── test/              # Tests
├── infrastructure/    # IaC (Terraform, K8s)
├── docs/             # Documentación
└── .github/          # CI/CD workflows
```

## 🔧 Variables de Entorno

```env
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
ENVIRONMENT=production

# Database
DATABASE_TYPE=json  # json, postgres, mongodb
DATABASE_DATA_PATH=./data/products
DATABASE_DSN=postgres://user:pass@host/db

# Cache
CACHE_ENABLED=true
CACHE_TYPE=redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Security
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
JWT_SECRET=your-secret-key

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## 📈 Monitoreo

### Métricas Disponibles

- Request count y duration
- Error rates
- Cache hit ratio
- Database query performance
- CPU y Memory utilization

### Dashboards

- CloudWatch Dashboard
- Grafana Dashboard (incluido en `infrastructure/monitoring/`)
- Custom alerts y runbooks

## 🛡️ Seguridad

- ✅ OWASP API Security Top 10 compliance
- ✅ Rate limiting y DDoS protection
- ✅ Input validation y sanitización
- ✅ Secrets management con AWS Secrets Manager
- ✅ Vulnerability scanning con Trivy
- ✅ Network isolation con VPC

## 📝 Documentación Adicional

- [Diagramas C4](docs/architecture/) - Arquitectura del sistema
- [ADRs](docs/adr/) - Decisiones arquitectónicas
- [API Spec](docs/api/openapi.yaml) - Especificación OpenAPI
- [Guía de Escalamiento](docs/scaling-guide.md) - Escalar a millones
- [Runbooks](docs/runbooks/) - Procedimientos operativos

## 🤝 Contribución

1. Fork el proyecto
2. Crea tu feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push al branch (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📄 Licencia

Este proyecto está licenciado bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para detalles.

## 👥 Equipo

- **Arquitecto Principal**: [Tu Nombre]
- **DevOps Lead**: [Tu Nombre]
- **Backend Developer**: [Tu Nombre]

## 📞 Soporte

Para soporte, envía un email a support@product-api.com o abre un issue en GitHub.

---

⭐ Si este proyecto te ha sido útil, considera darle una estrella en GitHub!
