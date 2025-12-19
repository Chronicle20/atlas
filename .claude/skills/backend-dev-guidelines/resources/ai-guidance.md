
---
title: AI Code Generation Guidance
description: Rules for AI agents generating or editing Golang services.
---

# AI Code Generation Guidance

## Core Rules
1. Respect file responsibilities (see file-responsibilities.md).
2. Maintain immutability and functional composition.
3. Use curried functions for dependency injection.
4. Only emit messages via `AndEmit` + buffer.
5. Never include tenant fields in public APIs.
6. Use `server.RegisterHandler` and `server.RegisterInputHandler` for REST endpoints.
7. Implement JSON:API interface methods on all REST models and request types.

## Generation Workflow
1. Create `model.go` - Immutable domain model with accessors
2. Map `entity.go` to DB - GORM entities with migrations
3. Implement `builder.go` - Fluent API for model construction
4. Define processors and providers - Pure business logic
5. Add `rest.go` - JSON:API DTOs with Transform functions
6. Add `resource.go` - Route registration and thin handlers
7. Add Kafka producers (if needed) - Event emission
8. Write table-driven tests

## REST Generation Specifics


### When generating `rest.go`:
- ✅ Implement JSON:API interface on all models (`GetName()`, `GetID()`, `SetID()`)
- ✅ Use flat structure for request models (no nested Data/Type/Attributes)
- ✅ Mark ID field with `json:"-"` tag

- ✅ Use pointer fields for optional attributes with `omitempty`
- ✅ Create `Transform(Model) (RestModel, error)` function

- ✅ Create `TransformSlice([]Model) ([]RestModel, error)` function
- ❌ Never use jsonapi struct tags on fields
- ❌ Never create nested Data/Type/Attributes structures

### When generating `resource.go`:
- ✅ Return `func(db *gorm.DB) server.RouteInitializer` from `InitializeRoutes`
- ✅ Use `server.RegisterHandler(l)(si)("handler-name", handler)` for GET/DELETE
- ✅ Use `server.RegisterInputHandler[T](l)(si)("handler-name", handler)` for POST/PATCH
- ✅ Map domain errors to specific HTTP status codes
- ✅ Use `server.MarshalResponse[T]` for success responses
- ✅ Log errors with `d.Logger().WithError(err)`
- ❌ Never manually parse tenant from headers
- ❌ Never manually decode JSON with nested structures
- ❌ Never create custom error response helpers

## Useful Composition
```go
// Transform entity to model using provider pattern
model.Map(Make)(entityProvider)(model.ParallelMap())


// Transform model to REST representation
res, err := ops.Map(Transform)(ops.FixedProvider(model))()

// Transform slice of models to REST representations
res, err := ops.SliceMap(Transform)(ops.FixedProvider(models))(ops.ParallelMap())()
```


## Common Anti-Patterns to Avoid

### ❌ Manual JSON:API Envelope Handling
```go
// DON'T DO THIS
var req struct {
    Data struct {
        Type       string `json:"type"`
        Attributes struct { ... } `json:"attributes"`
    } `json:"data"`
}
json.NewDecoder(r.Body).Decode(&req)
```

### ✅ Use Flat Request Models
```go
// DO THIS
type CreateRequest struct {
    Id   uuid.UUID `json:"-"`
    Name string    `json:"name"`
}
// Let server.RegisterInputHandler handle JSON:API unmarshaling
```

### ❌ Manual HTTP Handler Registration

```go

// DON'T DO THIS
router.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    // Manual tenant parsing, error handling, etc.
})
```

### ✅ Use server.RegisterHandler
```go
// DO THIS
router.HandleFunc("/users",
    server.RegisterHandler(l)(si)("get-users", listUsersHandler(db))).Methods(http.MethodGet)
```
