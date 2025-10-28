# ADR-003: Selección del Lenguaje Go para Microservicios

## Estado
Aceptado

## Fecha
2024-10-27

## Contexto

Necesitamos seleccionar un lenguaje de programación principal para nuestros microservicios. Los requisitos clave son:

### Requisitos Técnicos
- Alto rendimiento y baja latencia (< 100ms P99)
- Manejo eficiente de concurrencia (10K+ conexiones simultáneas)
- Bajo consumo de memoria (contenedores < 100MB)
- Tiempo de arranque rápido (< 2 segundos)
- Compilación a binario único sin dependencias

### Requisitos del Equipo
- Curva de aprendizaje razonable
- Ecosistema maduro de bibliotecas
- Buenas herramientas de desarrollo
- Comunidad activa y soporte
- Facilidad para contratar talento

### Contexto Actual
- El equipo tiene experiencia en Java, Python y JavaScript
- Infraestructura basada en contenedores (Kubernetes)
- Necesidad de escalar horizontalmente
- APIs REST como interfaz principal

### Opciones Consideradas

1. **Go (Golang)**
   - Pros: Performance, concurrencia nativa, binarios pequeños
   - Cons: Sin genéricos hasta v1.18, ecosistema más pequeño que Java

2. **Java (Spring Boot)**
   - Pros: Ecosistema maduro, talento disponible, Spring Cloud
   - Cons: JVM overhead, tiempo de arranque lento, contenedores grandes

3. **Node.js (TypeScript)**
   - Pros: Equipo familiar, mismo lenguaje front/back, NPM ecosystem
   - Cons: Single-threaded, callback hell potencial, type safety limitado

4. **Rust**
   - Pros: Máximo performance, memory safety, zero-cost abstractions
   - Cons: Curva de aprendizaje empinada, compilación lenta, menos bibliotecas

5. **Python (FastAPI)**
   - Pros: Desarrollo rápido, sintaxis simple, ML libraries
   - Cons: GIL limitations, performance inferior, type hints opcionales

## Decisión

Seleccionamos **Go (Golang)** como lenguaje principal para nuestros microservicios.

### Factores Determinantes

1. **Performance vs Simplicidad**: Go ofrece el mejor balance entre alto rendimiento y facilidad de desarrollo
2. **Concurrencia**: Goroutines y channels son ideales para I/O intensivo
3. **Despliegue**: Binarios estáticos simplifican containerización
4. **Operaciones**: Excelente observabilidad y profiling built-in
5. **Cloud Native**: Diseñado para sistemas distribuidos modernos

### Comparación de Métricas Clave

| Métrica | Go | Java | Node.js | Rust | Python |
|---------|-----|------|---------|------|--------|
| Startup Time | <1s | 5-10s | 1-2s | <1s | 2-3s |
| Memory (idle) | 10MB | 150MB | 50MB | 5MB | 40MB |
| Requests/sec | 50K | 30K | 20K | 60K | 10K |
| Container Size | 20MB | 200MB | 100MB | 15MB | 150MB |
| Learning Curve | Medium | Low | Low | High | Low |
| Hiring Difficulty | Medium | Easy | Easy | Hard | Easy |

## Consecuencias

### Positivas

✅ **Performance excelente**: 50K+ RPS por instancia típicamente
✅ **Concurrencia simple**: Goroutines más fáciles que threads/callbacks
✅ **Deployment sencillo**: Un binario, sin runtime dependencies
✅ **Tooling integrado**: fmt, test, bench, pprof incluidos
✅ **Compile-time safety**: Errores detectados antes de runtime
✅ **Kubernetes friendly**: La mayoría de k8s está escrito en Go

### Negativas

❌ **Verbosidad**: Manejo de errores explícito puede ser repetitivo
❌ **Sin excepciones**: Error handling diferente a lo acostumbrado
❌ **Ecosistema menor**: Menos librerías que Java/JavaScript
❌ **Sin herencia**: Solo composición, requiere cambio mental
❌ **Talento escaso**: Menos developers Go que Java/JS en el mercado

### Neutrales

➖ Interfaces implícitas (duck typing estructural)
➖ Sin genéricos hasta recientemente (Go 1.18+)
➖ Opinionated formatting (gofmt)
➖ Package management evolution (modules vs GOPATH)

## Implementación

### Stack Tecnológico Go

```yaml
Web Framework: Gorilla Mux / Gin / Echo
ORM: GORM / sqlx
Testing: testify + gomock
Logging: zap / logrus
Metrics: prometheus client
Tracing: opentelemetry
API Docs: swaggo
Linting: golangci-lint
```

### Estructura de Proyecto Estándar

```go
// Siguiendo Clean Architecture (ADR-002)
myservice/
├── cmd/server/main.go      // Entry point
├── internal/               // Private code
│   ├── domain/            // Business entities
│   ├── usecase/           // Business logic
│   └── adapter/           // External interfaces
├── pkg/                   // Public libraries
├── go.mod                 // Dependencies
└── Dockerfile             // Multi-stage build
```

## Mitigaciones

Para abordar las consecuencias negativas:

1. **Error Handling**: Crear helpers y usar error wrapping
2. **Code Generation**: Usar go generate para boilerplate
3. **Training Program**: Curso intensivo de Go para el equipo
4. **Hiring Strategy**: Contratar seniors, entrenar juniors
5. **Library Gap**: Contribuir a open source, crear internamente
6. **Code Reviews**: Seniors revisan todo código inicial

## Métricas de Validación

Mediremos el éxito después de 6 meses:

- **Performance**: P99 latency < 50ms ✓
- **Reliability**: 99.99% uptime ✓
- **Productivity**: 2+ deploys/día/developer ✓
- **Quality**: < 1 bug crítico/mes ✓
- **Team Satisfaction**: NPS > 8 ✓
- **Cost**: 40% reducción vs Java ✓

## Casos de Uso Exitosos

- **Docker**: Escrito completamente en Go
- **Kubernetes**: Orquestador dominante en Go
- **Prometheus**: Monitoring standard en Go
- **Uber**: Migró servicios críticos a Go
- **Dropbox**: Reescribió infraestructura en Go
- **Netflix**: Usa Go para servicios de backend

## Referencias

- [Go at Google: Language Design in the Service of Software Engineering](https://talks.golang.org/2012/splash.article)
- [Why Go?](https://github.com/golang/go/wiki/WhyGo)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Uber's Go Style Guide](https://github.com/uber-go/guide)

## Revisiones

| Versión | Fecha | Autor | Cambios |
|---------|-------|-------|---------|
| 1.0 | 2024-10-27 | Equipo Arquitectura | Versión inicial |

## Aprobación

- **CTO**: ✅ Aprobado
- **Arquitecto Principal**: ✅ Aprobado
- **Team Leads**: ✅ Aprobado (4/5 votos)
- **DevOps**: ✅ Aprobado