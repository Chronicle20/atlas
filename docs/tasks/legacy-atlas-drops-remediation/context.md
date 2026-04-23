# Atlas-Drops Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Service Core
| File | Purpose | Lines | Key Elements |
|------|---------|-------|--------------|
| `drop/model.go` | Domain model with builder | 325 | `Model`, `ModelBuilder`, status constants |
| `drop/processor.go` | Business logic interface | 258 | `Processor` (16 methods), `ProcessorImpl` |
| `drop/registry.go` | In-memory storage | 263 | `dropRegistry` singleton, locking mechanisms |
| `drop/rest.go` | JSON:API model | 72 | `RestModel`, `Transform()` |
| `drop/producer.go` | Kafka message providers | - | Event status providers |
| `drop/resource.go` | REST handlers | - | HTTP route handlers |

### Kafka Infrastructure
| File | Purpose |
|------|---------|
| `kafka/message/message.go` | Message buffer pattern for atomic emission |
| `kafka/producer/producer.go` | Curried producer with header decorators |
| `kafka/consumer/drop/consumer.go` | Command handlers with header parsers |

### Cross-Service
| File | Purpose |
|------|---------|
| `equipment/processor.go` | Equipment creation/deletion via REST |
| `equipment/requests.go` | REST client helpers |

---

## Architecture Decisions

### In-Memory Design (Intentional Deviation)
The service does not use `entity.go` or `provider.go` because drops are transient game state with TTL expiration. The `registry.go` singleton serves as the data layer.

**Justification:** Drops are short-lived objects (typically <5 minutes) that don't require database persistence. In-memory storage provides lower latency for high-frequency operations.

### Processor Interface Pattern
```go
type Processor interface {
    // Pure methods - accept buffer, don't emit
    Spawn(mb *message.Buffer) func(mb *ModelBuilder) (Model, error)
    Reserve(mb *message.Buffer) func(...) (Model, error)

    // AndEmit variants - emit internally
    SpawnAndEmit(mb *ModelBuilder) (Model, error)
    ReserveAndEmit(...) (Model, error)
}
```

This pattern enables:
- Unit testing with pure methods (inspect buffer contents)
- Production use with AndEmit variants (automatic emission)

### Registry Locking Strategy
```go
type dropRegistry struct {
    lock      sync.RWMutex           // Global lock for maps
    dropLocks map[uint32]*sync.Mutex // Per-drop locks
    mapLocks  map[mapKey]*sync.Mutex // Per-map locks
}
```

Operations acquire locks in order: global → drop-specific → map-specific

---

## Dependencies

### External Packages
```go
import (
    "github.com/Chronicle20/atlas-constants/channel"
    "github.com/Chronicle20/atlas-constants/inventory"
    "github.com/Chronicle20/atlas-constants/item"
    "github.com/Chronicle20/atlas-constants/map"
    "github.com/Chronicle20/atlas-constants/world"
    "github.com/Chronicle20/atlas-model/model"
    "github.com/Chronicle20/atlas-tenant"
    "github.com/google/uuid"
    "github.com/segmentio/kafka-go"
    "github.com/sirupsen/logrus"
)
```

### Cross-Service Calls
- **EQUIPABLES service:** Equipment creation for equip-type drops
  - Called from: `drop/processor.go:84` (Spawn)
  - Called from: `drop/processor.go:210` (Expire - deletion)

---

## Testing Reference

### Similar Services with Tests
- `atlas-parties/party/registry_test.go` - Registry pattern tests
- `atlas-monsters/monster/registry_test.go` - In-memory registry tests
- `atlas-notes/note/processor_test.go` - Processor tests
- `atlas-notes/note/mock/processor.go` - Mock implementation pattern

### Test Patterns to Follow
```go
// Tenant creation for tests
ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

// Registry access
r := GetRegistry()

// Table-driven tests
tests := []struct {
    name     string
    input    func() *ModelBuilder
    expected func(Model) bool
}{...}
```

### Message Buffer Testing
```go
// Create buffer
buf := message.NewBuffer()

// Call pure method
model, err := processor.Spawn(buf)(builder)

// Inspect buffered messages
messages := buf.GetAll()
assert(messages["TOPIC_DROP_STATUS"] != nil)
```

---

## Audit Evidence

### Passing Checks
| ID | Check | Evidence Location |
|----|-------|-------------------|
| ARCH-001 | Layer Separation | `drop/resource.go:24` |
| ARCH-002 | Processor Interface | `drop/processor.go:21-61` |
| ARCH-003 | Model Immutability | `drop/model.go:17-40` |
| ARCH-005 | Provider Pattern | `drop/processor.go:241-252` |
| ARCH-006 | Kafka Producer | `kafka/producer/producer.go:12-20` |
| ARCH-007 | Kafka Consumer | `kafka/consumer/drop/consumer.go:20` |
| ARCH-008 | REST JSON:API | `drop/rest.go:33-48` |
| ARCH-009 | Multi-Tenancy | `drop/processor.go:75` |
| ARCH-010 | Message Buffer | `kafka/message/message.go:9-47` |

### Failing Checks
| ID | Check | Required Action |
|----|-------|-----------------|
| TEST-001 | Test Coverage | Add `*_test.go` files |
| TEST-002 | Mock Infrastructure | Create `drop/mock/processor.go` |

### Warning Checks
| ID | Check | Required Action |
|----|-------|-----------------|
| ARCH-004 | Builder Validation | Consider adding validation to `Build()` |
| DOC-001 | README | Create service README |

---

## Known Quirks

### Unique ID Wraparound
```go
// registry.go:52-59
func getNextUniqueId() uint32 {
    id := atomic.AddUint32(&uniqueId, 1)
    if id > 2000000000 {
        atomic.StoreUint32(&uniqueId, 1000000001)
        return 1000000001
    }
    return id
}
```
IDs start at 1000000001 and wrap at 2 billion. Potential collision if old drops still exist when wrap occurs (extremely unlikely in practice).

### Model Mutation Methods
`Model.Reserve()` and `Model.CancelReservation()` appear to mutate but actually return new instances via builder clone pattern. This is correct immutability.

### Equipment Dependency
When spawning equip-type drops, the processor makes a synchronous REST call to create equipment. This is a potential failure point that should be handled gracefully.
