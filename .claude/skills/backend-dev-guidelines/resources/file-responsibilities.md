
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

**Write operations** (Create, Update, Delete) that mutate database state. This file handles all state-changing database interactions.

**Key Responsibilities:**
- Define `create(db, ...)` functions that insert new entities and return the created model
- Define `delete(db, ...)` functions that remove entities
- Define `update(db, ...)` and `dynamicUpdate(db)` functions for modifying existing entities
- Define `EntityUpdateFunction` type and `Set*` modifier functions for composable updates
- All functions receive `*gorm.DB` and perform direct database mutations

**Typical Signatures:**
```go
// Create keeps tenantId — needed to set the entity field
func create(db *gorm.DB, tenantId uuid.UUID, ...) (Model, error)

// Update/Delete do NOT take tenantId — automatic tenant filtering via GORM callbacks
func deleteById(db *gorm.DB, id uint32) error
func dynamicUpdate(db *gorm.DB) func(modifiers ...EntityUpdateFunction) func(characterId uint32) model.Operator[Model]

type EntityUpdateFunction func() ([]string, func(e *entity))
func SetLevel(level byte) EntityUpdateFunction
func SetMeso(amount uint32) EntityUpdateFunction
```

**Pattern:** The `EntityUpdateFunction` pattern allows composable, selective field updates. Each `Set*` function returns the column names to update and a mutator function, enabling `dynamicUpdate` to batch multiple field changes into a single database operation.

## `provider.go`

**Read operations** (queries) that fetch data without side effects. This file handles all read-only database access.

**Key Responsibilities:**
- Define query functions returning `database.EntityProvider[T]` or `database.EntityProvider[[]T]`
- Provide `modelFromEntity(e entity) (Model, error)` transformation function
- Use `database.Query[T]` for single-entity lookups
- Use `database.SliceQuery[T]` for multi-entity queries
- Never modify database state

**Typical Signatures:**
```go
// Providers do NOT take tenantId — automatic tenant filtering via GORM callbacks
func getById(id uint32) database.EntityProvider[entity]
func getForAccountInWorld(accountId uint32, worldId world.Id) database.EntityProvider[[]entity]
func getAll() database.EntityProvider[[]entity]
func modelFromEntity(e entity) (Model, error)
```

**Pattern:** Provider functions are curried - they accept query parameters and return a function that takes `*gorm.DB` and returns `model.Provider[T]`. The `*gorm.DB` must have context set via `db.WithContext(ctx)` for automatic tenant filtering. This enables lazy evaluation and composition with `model.Map`, `model.SliceMap`, and `model.ParallelMap`.

**Why This Separation:**
- **Testability** - Read and write operations can be mocked independently
- **Pure composition** - Read path remains side-effect free, enabling functional composition
- **Clear intent** - Code review can quickly identify state-changing operations

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
- **Delegate ALL business logic to processors - NEVER call provider functions directly**
- Use `server.MarshalResponse[T]` for successful responses
- Log errors with context using `d.Logger().WithError(err)`

**Pattern:** Thin handlers that parse input, invoke processors, handle errors, and marshal responses.

**Critical Rules:**
- ✅ `resource.go` → `processor.go` (correct)
- ❌ `resource.go` → `provider.go` (WRONG - bypasses business logic layer)
- ❌ `resource.go` → database/GORM (WRONG - bypasses all layers)


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

## `requests.go`

REST client functions for calling other microservices. Always paired with a `rest.go` in the same package.

**Key Responsibilities:**
- Define `getBaseRequest()` returning `requests.RootUrl("SERVICE_NAME")`
- Implement request functions returning `requests.Request[RestModel]`
- Use `rest.MakeGetRequest[T]` for GET requests
- Use `rest.MakePostRequest[T]` for POST requests
- Build URLs using `fmt.Sprintf` with path parameters

**Example:**
```go
package status

import (
    "myservice/rest"
    "fmt"
    "github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
    return requests.RootUrl("QUEST")  // Uses QUEST_BASE_URL env var
}

func requestByCharacterAndQuest(characterId, questId uint32) requests.Request[RestModel] {
    return rest.MakeGetRequest[RestModel](
        fmt.Sprintf(getBaseRequest()+"/characters/%d/quests/%d", characterId, questId),
    )
}
```

**Calling pattern in processor:**
```go
resp, err := status.requestByCharacterAndQuest(charId, questId)(l, ctx)
```

**Pattern:** Request functions are curried - they return a `requests.Request[T]` which is then called with `(logger, context)` to execute. The `rest.go` in the same package defines the `RestModel` for JSON:API deserialization.

See [cross-service-implementation.md](cross-service-implementation.md) for the complete REST client pattern.
