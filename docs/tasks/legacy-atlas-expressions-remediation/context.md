# Atlas-Expressions Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Service Files (To Modify)

| File | Purpose | Changes Required |
|------|---------|------------------|
| `services/atlas-expressions/atlas.com/expressions/expression/model.go` | Immutable domain model | Optional: Add builder |
| `services/atlas-expressions/atlas.com/expressions/expression/processor.go` | Business logic | Flatten currying, use message.Emit |
| `services/atlas-expressions/atlas.com/expressions/expression/registry.go` | In-memory singleton | Add ResetForTesting |
| `services/atlas-expressions/atlas.com/expressions/expression/producer.go` | Kafka message provider | No changes |
| `services/atlas-expressions/atlas.com/expressions/expression/task.go` | Background expiration task | No changes |
| `services/atlas-expressions/atlas.com/expressions/kafka/message/message.go` | Message buffer pattern | No changes (use existing Emit) |

### Test Files (To Create)

| File | Purpose |
|------|---------|
| `expression/model_test.go` | Model getter and immutability tests |
| `expression/processor_test.go` | Business logic tests |
| `expression/registry_test.go` | Registry CRUD and concurrency tests |
| `expression/task_test.go` | RevertTask tests |
| `expression/mock/processor.go` | Processor interface mock |

### Reference Files (Patterns to Follow)

| File | Pattern |
|------|---------|
| `services/atlas-buffs/atlas.com/buffs/character/registry.go` | ResetForTesting method |
| `services/atlas-buffs/atlas.com/buffs/character/registry_test.go` | Registry test structure |
| `services/atlas-buffs/atlas.com/buffs/buff/model_test.go` | Model test structure |
| `services/atlas-drops/atlas.com/drops/drop/mock/processor.go` | Processor mock pattern |

---

## Architectural Decisions

### Decision 1: In-Memory Registry Instead of Database

**Context:** Standard services use entity.go + provider.go + administrator.go with GORM database persistence.

**Decision:** atlas-expressions uses an in-memory Registry singleton.

**Rationale:**
- Expressions are ephemeral (5-second TTL)
- Data is recoverable (resets on character map exit anyway)
- Performance critical (O(1) memory vs database round trips)
- No need for persistence across restarts

**Implications:**
- No entity.go needed
- No provider.go/administrator.go split
- Registry handles both reads and writes
- Must document this deviation in README

### Decision 2: Kafka-Only Interface

**Context:** Standard services expose REST JSON:API endpoints.

**Decision:** atlas-expressions communicates exclusively via Kafka.

**Rationale:**
- Internal service only (no external API consumers)
- Event-driven architecture appropriate for ephemeral state
- Lower latency for expression changes

**Implications:**
- No resource.go needed
- No rest.go needed
- No ingress configuration needed
- Consumers must handle expression events

### Decision 3: Flatten Processor Currying

**Context:** Current `Change` method has 7 levels of currying:
```go
Change(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) func(expression uint32) (Model, error)
```

**Decision:** Flatten to single function call matching `ChangeAndEmit` pattern.

**Rationale:**
- Excessive currying reduces readability
- `ChangeAndEmit` already uses flat signature successfully
- Guidelines prefer currying but not to this extreme

**Target Signature:**
```go
Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
```

### Decision 4: Use message.Emit Pattern

**Context:** Current `ChangeAndEmit` manually iterates buffer:
```go
for t := range mb.GetAll() {
    err = producer.ProviderImpl(p.l)(p.ctx)(t)(expressionEventProvider(...))
}
```

**Decision:** Use `message.Emit` or `message.EmitWithResult` for atomic emission.

**Rationale:**
- Consistent with message buffer pattern guidelines
- Atomic emission ensures all-or-nothing semantics
- Pattern already exists in kafka/message/message.go

---

## Dependencies

### Go Modules Required

```go
import (
    "testing"
    "sync"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/Chronicle20/atlas-tenant"
    "github.com/google/uuid"
)
```

### Internal Package Dependencies

- `atlas-expressions/kafka/message` - Buffer and Emit patterns
- `atlas-expressions/kafka/producer` - Producer provider
- `github.com/Chronicle20/atlas-constants/channel` - Channel ID type
- `github.com/Chronicle20/atlas-constants/map` - Map ID type
- `github.com/Chronicle20/atlas-constants/world` - World ID type

---

## Test Strategy

### Unit Tests

1. **Model Tests:** Verify getters return expected values
2. **Registry Tests:** CRUD operations, tenant isolation, concurrency
3. **Processor Tests:** Business logic with mocked registry (or using ResetForTesting)
4. **Task Tests:** Expiration processing

### Integration Considerations

- No database integration tests needed (in-memory only)
- Kafka integration tested via consumer tests if needed
- Focus on unit tests for domain logic

### Concurrency Testing

Registry must be tested for concurrent access:
- Multiple goroutines adding expressions
- Mixed add/clear operations
- Multi-tenant concurrent access

Reference: `atlas-buffs/character/registry_test.go:207-306`

---

## Code Snippets

### ResetForTesting Pattern

```go
// ResetForTesting clears all registry state. Only for use in tests.
func (r *Registry) ResetForTesting() {
    r.lock.Lock()
    defer r.lock.Unlock()
    r.expressionReg = make(map[tenant.Model]map[uint32]Model)
    r.tenantLock = make(map[tenant.Model]*sync.RWMutex)
}
```

### Test Tenant Helper

```go
func setupTestTenant(t *testing.T) tenant.Model {
    t.Helper()
    ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
    if err != nil {
        t.Fatalf("Failed to create tenant: %v", err)
    }
    return ten
}
```

### Processor Mock Pattern

```go
type ProcessorMock struct {
    ChangeFunc        func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
    ChangeAndEmitFunc func(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error)
    ClearFunc         func(mb *message.Buffer, transactionId uuid.UUID, characterId uint32) (Model, error)
    ClearAndEmitFunc  func(transactionId uuid.UUID, characterId uint32) (Model, error)
}

func (m *ProcessorMock) Change(mb *message.Buffer, transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error) {
    if m.ChangeFunc != nil {
        return m.ChangeFunc(mb, transactionId, characterId, worldId, channelId, mapId, expression)
    }
    return Model{}, nil
}
```

### message.EmitWithResult Usage

```go
type ChangeInput struct {
    TransactionId uuid.UUID
    CharacterId   uint32
    WorldId       world.Id
    ChannelId     channel.Id
    MapId         _map.Id
    Expression    uint32
}

func (p *ProcessorImpl) ChangeAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, expression uint32) (Model, error) {
    input := ChangeInput{
        TransactionId: transactionId,
        CharacterId:   characterId,
        WorldId:       worldId,
        ChannelId:     channelId,
        MapId:         mapId,
        Expression:    expression,
    }

    return message.EmitWithResult[Model, ChangeInput](
        producer.ProviderImpl(p.l)(p.ctx),
    )(func(mb *message.Buffer) func(input ChangeInput) (Model, error) {
        return func(i ChangeInput) (Model, error) {
            return p.Change(mb, i.TransactionId, i.CharacterId, i.WorldId, i.ChannelId, i.MapId, i.Expression)
        }
    })(input)
}
```

---

## Related Audit Findings

| Check ID | Status | Issue |
|----------|--------|-------|
| ARCH-001 | pass | Immutability pattern correct |
| ARCH-002 | warn | Missing builder (optional) |
| ARCH-003 | warn | Excessive currying |
| ARCH-004 | warn | Registry combines read/write (justified) |
| ARCH-005 | pass | Layer separation correct |
| ARCH-006 | pass | Kafka producer pattern correct |
| ARCH-007 | pass | Kafka consumer pattern correct |
| ARCH-008 | pass | No REST (by design) |
| ARCH-009 | pass | Multi-tenancy correct |
| ARCH-010 | warn | Not using message.Emit |
| ARCH-011 | fail | No test coverage |
| ARCH-012 | pass | No ingress (by design) |
| ARCH-013 | pass | Documentation adequate |

---

## Notes from Audit

### Error Handling in Consumers

Both Kafka consumers discard errors:
```go
_, _ = processor.ChangeAndEmit(...)
_, _ = processor.ClearAndEmit(...)
```

Consider adding error logging in a future iteration.

### OpenTracing vs OpenTelemetry

The service uses both tracing libraries:
- `tracing/tracing.go` - OpenTracing (Jaeger)
- `expression/task.go` - OpenTelemetry

Works due to OpenTelemetry's bridge support but is a minor inconsistency.
