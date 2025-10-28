# ADR-002: Mandato de Clean Architecture para la Implementación del Servicio

## Estado
Aceptado

## Fecha
2024-10-27

## Contexto

Con la decisión de adoptar microservicios (ADR-001), necesitamos establecer una arquitectura interna consistente para cada servicio que garantice:

- Mantenibilidad a largo plazo
- Testabilidad exhaustiva
- Independencia de frameworks y bibliotecas externas
- Facilidad para cambiar decisiones técnicas
- Claridad en la organización del código
- Onboarding rápido de nuevos desarrolladores

El equipo tiene experiencia mixta, algunos vienen de arquitecturas MVC tradicionales y otros de desarrollo sin estructura clara. Necesitamos un enfoque que unifique criterios.

### Problema Específico

En proyectos anteriores hemos experimentado:
- Acoplamiento fuerte entre lógica de negocio y frameworks
- Dificultad para cambiar bases de datos
- Tests que requieren infraestructura completa
- Código duplicado entre servicios
- Lógica de negocio dispersa en controladores

### Opciones Consideradas

1. **MVC Tradicional**: Model-View-Controller clásico
2. **Clean Architecture**: Arquitectura en capas con inversión de dependencias
3. **Hexagonal Architecture**: Puertos y adaptadores
4. **Domain-Driven Design (DDD)**: Diseño dirigido por dominio completo
5. **Vertical Slice Architecture**: Features organizadas verticalmente

## Decisión

Adoptamos **Clean Architecture** como patrón obligatorio para todos los microservicios, específicamente el modelo de 4 capas:

1. **Entities (Domain)**: Lógica de negocio empresarial
2. **Use Cases**: Lógica de negocio de aplicación
3. **Interface Adapters**: Controladores, presentadores, gateways
4. **Frameworks & Drivers**: Detalles de implementación

### Estructura de Carpetas Estándar

```
service-name/
├── cmd/                    # Puntos de entrada
├── internal/
│   ├── domain/            # Entidades y lógica de negocio pura
│   ├── usecase/           # Casos de uso e interfaces
│   └── adapter/           # Adaptadores de interfaz
│       ├── handler/       # HTTP, gRPC, GraphQL handlers
│       └── repository/    # Implementaciones de persistencia
├── pkg/                   # Código reutilizable
└── test/                  # Tests organizados por tipo
```

### Reglas de Dependencia

```
domain → (nada)
usecase → domain
adapter → usecase, domain
main → adapter, usecase, domain
```

## Consecuencias

### Positivas

✅ **Testabilidad superior**: Cada capa se prueba independientemente con mocks
✅ **Flexibilidad tecnológica**: Cambiar BD o framework sin tocar lógica de negocio
✅ **Claridad organizacional**: Ubicación obvia para cada tipo de código
✅ **Mantenibilidad mejorada**: Cambios aislados por capa
✅ **Onboarding acelerado**: Estructura consistente entre servicios
✅ **Reutilización**: Lógica de negocio portable entre adaptadores

### Negativas

❌ **Boilerplate inicial**: Más código para funcionalidad simple
❌ **Curva de aprendizaje**: Requiere entender inversión de dependencias
❌ **Over-engineering potencial**: Puede ser excesivo para servicios triviales
❌ **Mapeo entre capas**: Conversión de DTOs entre boundaries
❌ **Indirección adicional**: Más saltos para seguir el flujo

### Neutrales

➖ Interfaces everywhere en Go (sin herencia clásica)
➖ Más archivos y carpetas que enfoques simples
➖ Requiere disciplina del equipo para mantener boundaries

## Implementación

### Ejemplo de Flujo

```go
// 1. Domain Layer
type Product struct {
    ID    string
    Name  string
    Price float64
}

// 2. Use Case Layer
type ProductRepository interface {
    FindByID(id string) (*Product, error)
}

type ProductInteractor struct {
    repo ProductRepository
}

// 3. Adapter Layer
type HTTPHandler struct {
    interactor *ProductInteractor
}

type PostgresRepository struct {
    db *sql.DB
}

// 4. Main (Dependency Injection)
func main() {
    db := connectDB()
    repo := NewPostgresRepository(db)
    interactor := NewProductInteractor(repo)
    handler := NewHTTPHandler(interactor)
    startServer(handler)
}
```

## Mitigaciones

Para abordar las consecuencias negativas:

1. **Generadores de código**: Templates para scaffolding inicial
2. **Librerías compartidas**: Código común en paquetes internos
3. **Pragmatismo**: Permitir simplificaciones para CRUDs simples
4. **Documentación clara**: Guías y ejemplos de implementación
5. **Code reviews estrictos**: Asegurar adherencia a la arquitectura
6. **Métricas de código**: Detectar violaciones automáticamente

## Criterios de Éxito

- **Coverage de tests > 80%** sin necesidad de infraestructura
- **Tiempo de cambio de BD < 1 día** para cualquier servicio
- **0 imports de frameworks** en domain y usecase layers
- **Nuevos developers productivos en < 1 semana**
- **Tiempo de añadir nuevo adapter < 2 horas**

## Excepciones

Se permiten excepciones para:

- Scripts de migración y utilidades
- Proof of concepts y prototipos
- Servicios de menos de 500 líneas de código
- Herramientas internas de desarrollo

Toda excepción debe ser documentada y aprobada.

## Referencias

- Robert C. Martin - [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- "Clean Architecture" (Robert C. Martin, 2017)
- [go-clean-arch](https://github.com/bxcodec/go-clean-arch) - Ejemplo en Go
- [Wild Workouts](https://github.com/ThreeDotsLabs/wild-workouts-go-ddd-example) - DDD + Clean Architecture en Go

## Revisiones

| Versión | Fecha | Autor | Cambios |
|---------|-------|-------|---------|
| 1.0 | 2024-10-27 | Equipo Arquitectura | Versión inicial |

## Aprobación

- **Arquitecto Principal**: ✅ Aprobado
- **Tech Lead**: ✅ Aprobado
- **Senior Developers**: ✅ Aprobado por unanimidad