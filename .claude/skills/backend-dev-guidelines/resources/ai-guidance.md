
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
8. **Always verify referenced types exist before using them** - Never assume a type/constant/operation exists.
9. **Always run builds AND tests after code changes** - Verify ALL affected services compile and pass tests.
10. **Always ask before implementing new features** - Get user approval before adding new functionality.
11. **Follow cross-service implementation guidelines** - See [cross-service-implementation.md](cross-service-implementation.md) when working across multiple services.

## Validation Rules

### Before Using Any Type/Constant/Operation:
1. **Always verify it exists** - Read the relevant file to confirm the type/constant is implemented
2. **Check all dependent services** - If using a condition type, verify it's in validation/model.go; if using an operation, verify it's in operation_executor.go
3. **Never assume** - Just because it makes logical sense doesn't mean it's implemented

### Examples:
❌ **BAD** - Using condition without verification:
```go
// Assuming "petCount" condition exists without checking
conditions := []validation.Condition{
    {Type: "petCount", Operator: ">=", Value: "1"},
}
```

✅ **GOOD** - Verify first, ask if missing:
```
1. Read validation/model.go
2. Search for "petCount" in ConditionType constants
3. If not found: "The condition type 'petCount' doesn't exist in validation/model.go.
   Should I implement it first, or use a different approach?"
```

❌ **BAD** - Using operation without verification:
```go
// Assuming "gain_closeness" operation exists
operation := Operation{
    Type: "gain_closeness",
    Params: map[string]string{"petIndex": "0", "amount": "2"},
}
```

✅ **GOOD** - Verify first, ask if missing:
```
1. Read operation_executor.go
2. Search for case "gain_closeness": in the switch statement
3. If not found: "The operation 'gain_closeness' doesn't exist in operation_executor.go.
   To implement it, I would need to:
   - Add case in operation_executor.go
   - Add action/payload to saga model
   - Create Kafka producer for pets service

   Should I implement this feature first?"
```

## Testing Rules

### After ANY Code Change:
1. **Always run builds for ALL affected services** - Not just the one you modified
2. **Always run tests** for all modified and dependent services
3. **Report failures immediately** - Never commit/continue with failing builds or tests
4. **Update all mocks** - When interfaces change, update ALL mock implementations
5. **No partial implementations** - A feature isn't done until all services build and test successfully

### Build & Test Workflow:
```bash
# CRITICAL: Always build from workspace root to catch cross-service issues
cd /path/to/workspace/root

# Build ALL affected services (not just one!)
go build ./services/atlas-character/atlas.com/character/...
go build ./services/atlas-npc-conversations/atlas.com/npc/...
go build ./services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/...

# If ANY build fails:
# 1. Report the failure to the user with error details
# 2. Fix ALL compilation errors (missing methods, type mismatches, etc.)
# 3. Re-run builds for ALL services
# 4. Only proceed when ALL services build successfully

# After successful builds, run tests
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test ./...

# If tests fail:
# 1. Report the failure to the user
# 2. Fix the tests or code (usually missing mock methods)
# 3. Re-run tests
# 4. Only proceed when all tests pass
```

### Common Build Failures & Fixes:
| Error | Cause | Solution |
|-------|-------|----------|
| `missing method ChangeFace` | Interface updated but mock not updated | Add method to mock struct and implement it |
| `redeclared in this block` | Duplicate function declarations | Remove old/duplicate version |
| `cannot use X as Y value` | Function signature changed incompletely | Update ALL call sites (use `grep -r`) |
| `undefined: saga.ChangeHair` | Type used before defined | Add type to saga model FIRST |

### When to Run Tests:
- ✅ After adding new files (model.go, processor.go, etc.)
- ✅ After modifying existing files
- ✅ After adding new dependencies
- ✅ Before creating a pull request
- ✅ After implementing new features
- ❌ Never skip tests "to save time"

## Implementation Rules

### Before Implementing New Features:
1. **ALWAYS ask the user first** if you identify a missing feature during any task
2. **Explain what's missing** and what would need to be implemented
3. **Provide options** for how to proceed
4. **Wait for explicit approval** before implementing

### Example Dialog:
❌ **WRONG** - Implementing without asking:
```
"I noticed the 'gain_closeness' operation doesn't exist. Let me implement it for you..."
[Proceeds to implement without approval]
```

✅ **CORRECT** - Ask first:
```
"I need to use the 'gain_closeness' operation for this NPC conversion, but it doesn't
exist in operation_executor.go yet.

To implement it, I would need to:
1. Add 'gain_closeness' case to operation_executor.go (npc-conversations service)
2. Add GainCloseness action and payload to saga/model.go (saga-orchestrator service)
3. Add handleGainCloseness to saga/handler.go (saga-orchestrator service)
4. Create pet/producer.go for Kafka messages (saga-orchestrator service)
5. Possibly add petCount condition to validation (query-aggregator service)

This would touch 3 services and require coordination with the pets service.

How would you like to proceed?
1. Implement the missing features first
2. Skip this NPC for now
3. Use a different approach
"
```

## Generation Workflow
0. **Validate dependencies** - Verify all types/operations you plan to use exist
1. Create `model.go` - Immutable domain model with accessors
2. Map `entity.go` to DB - GORM entities with migrations
3. Implement `builder.go` - Fluent API for model construction
4. Define processors and providers - Pure business logic
5. Add `rest.go` - JSON:API DTOs with Transform functions
6. Add `resource.go` - Route registration and thin handlers
7. Add Kafka producers (if needed) - Event emission
8. Write table-driven tests
9. **Build ALL affected services** - From workspace root, build every service touched
10. **Run tests for ALL affected services** - Verify nothing broke
11. **Report build/test results** - Show pass/fail status for EACH service to user
12. **Fix ALL issues before proceeding** - No partial implementations allowed

### For Cross-Service Features:
See [cross-service-implementation.md](cross-service-implementation.md) for detailed checklist including:
- Implementation order (types → implementations → mocks)
- Interface change verification
- Mock synchronization
- Build verification for all services
- Test execution for all services

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
