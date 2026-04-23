# atlas-buffs Remediation Context

**Last Updated:** 2026-01-13
**Service Path:** `services/atlas-buffs/atlas.com/buffs/`

---

## Key Files

### Core Domain Files

| File | Purpose | Remediation Needed |
|------|---------|-------------------|
| `character/model.go` | Character aggregate holding buff map | Fix `Buffs()` to return copy |
| `character/processor.go` | Business logic orchestration | Implement message buffer pattern |
| `character/registry.go` | In-memory singleton state store | Add test coverage |
| `character/producer.go` | Kafka message creation | No changes needed |
| `character/resource.go` | REST handlers | No changes needed |

### Buff Domain Files

| File | Purpose | Remediation Needed |
|------|---------|-------------------|
| `buff/model.go` | Immutable buff model | Add input validation to NewBuff |
| `buff/rest.go` | JSON:API transform | No changes needed |
| `buff/stat/model.go` | Stat change model | No changes needed |
| `buff/stat/rest.go` | Stat REST model | Add JSON:API interface methods |

### Infrastructure Files

| File | Purpose | Remediation Needed |
|------|---------|-------------------|
| `tasks/expiration.go` | Periodic expiration task | Rename Respawn to Expiration |
| `kafka/producer/producer.go` | Producer with decorators | No changes needed |
| `README.md` | Service documentation | Add architecture decision section |

### Files to Create

| File | Purpose |
|------|---------|
| `character/registry_test.go` | Registry concurrency tests |
| `character/processor_test.go` | Processor business logic tests |
| `buff/model_test.go` | Buff model tests |
| `buff/rest_test.go` | REST transform tests |
| `buff/stat/rest_test.go` | Stat REST transform tests |
| `kafka/message/buffer.go` | Message buffer utility (if needed) |

---

## Key Decisions

### D1: Test Strategy for Singleton Registry

**Decision:** Use registry directly in tests, reset state between tests

**Rationale:** The registry is a package-level singleton initialized with `sync.Once`. For testing:
- Cannot easily mock the registry
- Must reset state between tests by clearing the internal maps
- Consider adding a `Reset()` method for testing purposes

**Alternative Considered:** Dependency injection of registry interface - rejected as too invasive for this service.

### D2: Message Buffer Implementation

**Decision:** Create service-local message buffer if not available in shared library

**Rationale:** The message buffer pattern needs to:
- Accumulate kafka.Message instances
- Emit all at once to a topic
- Return aggregate error if any emit fails

Check if `atlas-kafka` or similar shared library provides this. If not, create locally.

### D3: NewBuff Signature Change

**Decision:** Change `NewBuff` to return `(Model, error)`

**Rationale:** Adding validation requires error returns. This is a breaking change affecting:
- `character/registry.go:59` - `b := buff.NewBuff(sourceId, duration, changes)`
- `kafka/consumer/character/consumer.go` - consumer handlers

Callers must be updated to handle the error.

---

## Dependencies

### Package Dependencies

```
character/
├── depends on: buff/, kafka/producer/, kafka/message/character/
└── used by: kafka/consumer/character/, tasks/, rest/

buff/
├── depends on: buff/stat/
└── used by: character/

tasks/
├── depends on: character/
└── used by: main.go
```

### External Dependencies

| Package | Usage |
|---------|-------|
| `github.com/Chronicle20/atlas-tenant` | Multi-tenancy context |
| `github.com/sirupsen/logrus` | Logging |
| `github.com/google/uuid` | Buff ID generation |
| `go.opentelemetry.io/otel` | Tracing |

---

## Code Patterns

### Registry Thread Safety Pattern

The registry uses a two-level locking strategy:

```go
type Registry struct {
    lock         sync.Mutex              // Global lock for tenant map modifications
    characterReg map[tenant.Model]map[uint32]Model
    tenantLock   map[tenant.Model]*sync.RWMutex  // Per-tenant locks
}
```

Operations follow this pattern:
1. Acquire global lock briefly to get/create per-tenant lock
2. Release global lock
3. Acquire per-tenant lock for actual operation
4. Release per-tenant lock

This allows concurrent operations on different tenants.

### Producer Pattern

```go
_ = producer.ProviderImpl(p.l)(p.ctx)(topic)(messageProvider(...))
```

The curried function pattern:
- `ProviderImpl(logger)` - sets up logger
- `(context)` - extracts tenant/span info
- `(topic)` - specifies Kafka topic
- `(provider)` - message creation function

### REST Transform Pattern

```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        // field mappings
    }, nil
}
```

Models implement:
- `GetName() string` - JSON:API type name
- `GetID() string` - Resource identifier
- `SetID(string) error` - For deserialization

---

## Test Utilities Needed

### Registry Reset Function

For testing, add to `registry.go`:

```go
// ResetForTesting clears all registry state. Only for use in tests.
func (r *Registry) ResetForTesting() {
    r.lock.Lock()
    defer r.lock.Unlock()
    r.characterReg = make(map[tenant.Model]map[uint32]Model)
    r.tenantLock = make(map[tenant.Model]*sync.RWMutex)
}
```

### Time Mocking for Expiration Tests

Buff expiration depends on `time.Now()`. Options:
1. Accept time as parameter to `NewBuff`
2. Use a package-level time function that tests can override
3. Create buffs with known expiration and sleep in tests

Recommend option 1 for testability - add optional `createdAt` parameter.

### Producer Mock

For processor tests, need to verify Kafka messages without actually sending. Options:
1. Interface-based injection (invasive change)
2. Capture calls via test-only producer implementation
3. Integration test approach - verify side effects

Recommend approach depends on existing test patterns in codebase.

---

## Audit Check References

| Check ID | Status | Location in audit.md |
|----------|--------|---------------------|
| ARCH-001 | pass | Layer Separation |
| ARCH-002 | warn | Model Immutability |
| ARCH-003 | fail | Builder Pattern |
| ARCH-004 | pass | Processor Pattern |
| ARCH-005 | warn | Provider Pattern |
| ARCH-006 | pass | Producer Pattern |
| ARCH-007 | pass | Multi-Tenancy Context |
| ARCH-008 | warn | REST JSON:API Pattern |
| ARCH-009 | pass | Ingress Configuration |
| ARCH-010 | pass | Documentation |
| ARCH-011 | pass | Kafka Consumer Pattern |
| ARCH-012 | fail | Testing Coverage |
| ARCH-013 | pass | Singleton Cache Pattern |
| ARCH-014 | warn | Entity Pattern |
| ARCH-015 | warn | Message Buffer Pattern |
| ARCH-016 | warn | Administrator/Provider Separation |

---

## Related Files Outside Service

| File | Relevance |
|------|-----------|
| `atlas-ingress.yml:72-74` | Ingress routing for buffs endpoint |
| `.claude/skills/backend-dev-guidelines/SKILL.md` | Guidelines used for audit |

---

## Notes

### Singleton Testing Consideration

The registry singleton is initialized once. Tests running in parallel could interfere with each other. Consider:
- Running registry tests with `-p 1` (no parallelism)
- Using subtests with proper cleanup
- Implementing a test registry separate from production singleton

### Goroutine Leaks in ExpireBuffs

The current `ExpireBuffs` spawns goroutines without waiting:

```go
for _, t := range ts {
    go func() {
        tctx := tenant.WithContext(ctx, t)
        _ = NewProcessor(l, tctx).ExpireBuffs()
    }()
}
```

This could cause issues if the service shuts down while expirations are processing. Consider using a WaitGroup if this becomes problematic, but this is outside the scope of current remediation.
