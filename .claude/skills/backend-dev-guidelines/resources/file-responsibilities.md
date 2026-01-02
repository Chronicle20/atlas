
---
title: File Responsibilities
description: Responsibilities of each standard file in a domain package.
---

# File Responsibilities

## `model.go`
Defines immutable domain objects with private fields and accessor methods.

## `entity.go`

Database entity definitions and migration helpers using GORM. Provides `Make(Entity) (Model, error)` and `Model.ToEntity()`.

## `builder.go`

Fluent API for constructing validated domain models. `Build()` enforces invariants.


## `processor.go`
Pure curried business functions plus `AndEmit` variants for Kafka emission. Dependency order: `NewProcessor(log, ctx, db)`.

## `administrator.go`
High-level curried functions coordinating transactional DB operations; return `model.Provider[Entity]`.

## `provider.go`
Lazy data access layer returning `model.Provider[T]`. Compose with `model.Map`, `model.SliceMap`, `model.ParallelMap`.

## `producer.go`
Kafka message creation using context-aware header decorators via `producer.ProviderImpl(log)(ctx)`.

## `resource.go`
Route registration and handler definitions for REST endpoints.

**Key Responsibilities:**
- Define `InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer` for route registration

- Use `server.RegisterHandler(l)(si)` for GET/DELETE handlers (no request body)
- Use `server.RegisterInputHandler[T](l)(si)` for POST/PATCH handlers (with typed request model)
- Implement handler functions matching `server.GetHandler` or `server.InputHandler[T]` signatures
- Map domain errors to HTTP status codes (400, 404, 409, 500)
- Delegate business logic to processors
- Use `server.MarshalResponse[T]` for successful responses

- Log errors with context using `d.Logger().WithError(err)`

**Pattern:** Thin handlers that parse input, invoke processors, handle errors, and marshal responses.


## `rest.go`
Serialization and transformation between domain models and JSON:API.

**Key Responsibilities:**
- Define `RestModel` struct implementing JSON:API interface (`GetName()`, `GetID()`, `SetID()`)
- Define request models (`CreateRequest`, `UpdateRequest`) implementing JSON:API interface
- Implement `Transform(Model) (RestModel, error)` to convert domain models to REST representations
- Implement `TransformSlice([]Model) ([]RestModel, error)` for bulk transformations
- Use flat structure for request models (no nested Data/Type/Attributes)
- Mark ID field with `json:"-"` tag (set via SetID)
- Use pointer fields for optional attributes with `omitempty`


**Pattern:** JSON:API-compliant DTOs with automatic marshaling/unmarshaling via api2go library. No tenant data in payloads.

## `state.go`

Domain-specific enums and constants; state transition helpers.

## `cache.go`

Singleton cache implementation using `sync.Once` for application-scoped data caching.

**Key Responsibilities:**
- Define `CacheInterface` for cache operations (`Get`, `Put`)
- Implement singleton cache struct with `sync.RWMutex` for thread safety
- Provide `GetCache()` function using `sync.Once` for singleton initialization
- Include TTL-based expiration for cached entries
- Provide test helper functions (`SetCacheForTesting`, `ResetCache`)
- Support multi-tenant caching when needed (partition by tenant ID)

**Pattern:** Application-scoped singleton shared across all requests, never per-instance or per-request. See [patterns-cache.md](patterns-cache.md) for complete implementation guide.
