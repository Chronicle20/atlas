# atlas-maps Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Files with Test Coverage Gaps

| File | Gap | Testing Challenge |
|------|-----|-------------------|
| `kafka/consumer/character/consumer.go` | Handler logic untested | Processor created internally |
| `kafka/consumer/cashshop/consumer.go` | Handler logic untested | Processor created internally |
| `map/resource.go` | HTTP handler untested | Requires HTTP test setup |
| `rest/handler.go` | Path parsers untested | Simple to test |
| `reactor/processor.go` | Spawn methods untested | HTTP client dependency |

### Files to Create

| File | Purpose | Priority |
|------|---------|----------|
| `rest/handler_test.go` | Path parsing utility tests | P2 |
| `map/resource_test.go` | Map resource handler tests | P2 |
| `kafka/consumer/character/consumer_test.go` | Consumer handler tests | P2 (requires refactor) |
| `kafka/consumer/cashshop/consumer_test.go` | Consumer handler tests | P2 (requires refactor) |

### Existing Test Files (Reference)

| File | Lines | Purpose |
|------|-------|---------|
| `map/processor_test.go` | 575 | Example test patterns, mock setup |
| `map/character/processor_test.go` | 326 | Tenant isolation testing patterns |
| `map/monster/processor_test.go` | 1498+ | Comprehensive inline mocks |
| `reactor/model_test.go` | 277 | Builder validation tests |
| `kafka/message/character/kafka_test.go` | 187 | Message serialization tests |
| `kafka/message/map/kafka_test.go` | - | Message serialization tests |

### Existing Mock Directories

| Directory | Methods | Status |
|-----------|---------|--------|
| `map/mock/processor.go` | 9 | Complete |
| `map/character/mock/processor.go` | 4 | Complete |
| `reactor/mock/processor.go` | 4 | Complete |

---

## Interface Definitions

### map.Processor (Complete)

```go
// map/processor.go
type Processor interface {
    Enter(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
    EnterAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
    Exit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
    ExitAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) error
    TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id)
    TransitionMapAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32, oldMapId _map.Id) error
    TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id)
    TransitionChannelAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, oldChannelId channel.Id, characterId uint32, mapId _map.Id) error
    GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
}
```

### character.Processor (Complete)

```go
// map/character/processor.go
type Processor interface {
    GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
    GetMapsWithCharacters() []MapKey
    Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
    Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
}
```

### reactor.Processor (Complete)

```go
// reactor/processor.go
type Processor interface {
    InMapModelProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model]
    GetInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error)
    Spawn(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error
    SpawnAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) error
}
```

---

## Testing Patterns

### HTTP Handler Testing

```go
func TestParseWorldId_Valid(t *testing.T) {
    l, _ := test.NewNullLogger()

    var capturedWorldId byte
    handler := ParseWorldId(l, func(worldId byte) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            capturedWorldId = worldId
            w.WriteHeader(http.StatusOK)
        }
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    req = mux.SetURLVars(req, map[string]string{"worldId": "5"})
    rr := httptest.NewRecorder()

    handler(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rr.Code)
    }
    if capturedWorldId != 5 {
        t.Errorf("Expected worldId 5, got %d", capturedWorldId)
    }
}

func TestParseWorldId_Invalid(t *testing.T) {
    l, _ := test.NewNullLogger()

    handler := ParseWorldId(l, func(worldId byte) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
        }
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    req = mux.SetURLVars(req, map[string]string{"worldId": "invalid"})
    rr := httptest.NewRecorder()

    handler(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", rr.Code)
    }
}
```

### Existing Mock Pattern (from processor_test.go)

```go
type mockCharacterProcessor struct {
    mu                        sync.Mutex
    enterCalls                []enterCall
    exitCalls                 []exitCall
    getCharactersInMapFunc    func(...) ([]uint32, error)
    getMapsWithCharactersFunc func() []character.MapKey
}

func (m *mockCharacterProcessor) GetCharactersInMap(...) ([]uint32, error) {
    if m.getCharactersInMapFunc != nil {
        return m.getCharactersInMapFunc(...)
    }
    return nil, nil
}

func (m *mockCharacterProcessor) Enter(...) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.enterCalls = append(m.enterCalls, enterCall{...})
}
```

### Producer Mock Pattern

```go
type mockProducerProvider struct {
    mu       sync.Mutex
    messages []kafka.Message
}

func (m *mockProducerProvider) Provider(token string) kafkaProducer.MessageProducer {
    return func(messages ...kafka.Message) error {
        m.mu.Lock()
        defer m.mu.Unlock()
        m.messages = append(m.messages, messages...)
        return nil
    }
}
```

### Tenant Context Setup

```go
ctx := context.Background()
ctx = tenant.WithContext(ctx, tenant.Model{
    Id:           uuid.New(),
    Region:       "GMS",
    MajorVersion: 83,
    MinorVersion: 1,
})
```

---

## Testing Challenges

### Challenge 1: Kafka Consumer Handler Internal Processor Creation

**Problem:** Handlers create processors internally, preventing mock injection.

```go
// Current pattern - not testable
func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event ...) {
    p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
    _ = p.EnterAndEmit(...)
}
```

**Solutions:**
1. **Refactor to accept processor factory** (preferred for testability)
2. **Integration test with real components** (tests full flow)
3. **Trust existing coverage** (processor tests cover logic)

### Challenge 2: Reactor Processor HTTP Dependency

**Problem:** `GetInMap` calls external reactor service via HTTP.

```go
func (p *ProcessorImpl) GetInMap(...) ([]Model, error) {
    return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(
        requestInMap(byte(worldId), byte(channelId), uint32(mapId)),
        Extract, model.Filters[Model]())()
}
```

**Solutions:**
1. **Use httptest.Server** to mock external service
2. **Extract HTTP client to interface** for mocking
3. **Test filter logic separately** (`doesNotExist` is testable)

### Challenge 3: Singleton Registry in Character Processor

**Problem:** Registry is a singleton, tests may interfere.

```go
var registry *Registry
var once sync.Once

func getRegistry() *Registry {
    once.Do(func() {
        registry = &Registry{}
        registry.characterRegister = make(map[MapKey][]uint32)
    })
    return registry
}
```

**Solution:** Use unique tenant IDs per test to ensure isolation.

---

## Dependencies

### External Libraries
- `github.com/gorilla/mux` - Router with URL vars
- `github.com/sirupsen/logrus/hooks/test` - Test logger
- `github.com/google/uuid` - UUID generation
- `net/http/httptest` - HTTP testing utilities

### Internal Dependencies
- `atlas-maps/kafka/producer` - Producer provider interface
- `atlas-maps/kafka/message` - Message buffer
- `atlas-maps/map/character` - Character processor
- `atlas-maps/reactor` - Reactor processor

---

## Environment & Build

### Build Commands
```bash
cd services/atlas-maps && go build -o atlas-maps ./atlas.com/maps
```

### Test Commands
```bash
# Run all tests
cd services/atlas-maps && go test ./...

# Run tests with coverage
cd services/atlas-maps && go test -cover ./...

# Run specific package tests
cd services/atlas-maps && go test ./atlas.com/maps/rest/...
```

### Package Structure
```
services/atlas-maps/atlas.com/maps/
├── main.go
├── map/
│   ├── processor.go, processor_test.go
│   ├── resource.go                      # Needs tests
│   ├── rest.go
│   ├── mock/processor.go
│   ├── character/
│   │   ├── processor.go, processor_test.go
│   │   └── mock/processor.go
│   └── monster/
│       └── processor.go, processor_test.go
├── reactor/
│   ├── processor.go                     # Needs tests (Spawn methods)
│   ├── model.go, model_test.go
│   └── mock/processor.go
├── rest/
│   ├── handler.go                       # Needs tests
│   └── request.go
└── kafka/
    ├── consumer/
    │   ├── character/consumer.go        # Needs tests (requires refactor)
    │   └── cashshop/consumer.go         # Needs tests (requires refactor)
    ├── message/
    │   ├── character/kafka.go, kafka_test.go
    │   └── map/kafka.go, kafka_test.go
    └── producer/producer.go
```

---

## Audit Reference

| Check ID | Name | Status | Notes |
|----------|------|--------|-------|
| ARCH-001 | Layer Separation | pass | Fixed in prior remediation |
| ARCH-002 | Processor Constructor | pass | - |
| MODEL-001 | Immutable Models | pass | - |
| MODEL-002 | Builder Validation | pass | Fixed in prior remediation |
| REST-001 | JSON:API Interface | pass | Fixed in prior remediation |
| REST-002 | Handler Registration | pass | - |
| KAFKA-001 | Producer Pattern | pass | - |
| KAFKA-002 | Consumer Parsers | pass | - |
| KAFKA-003 | AndEmit Pattern | pass | - |
| MULTI-001 | Context Tenancy | pass | - |
| CACHE-001 | Singleton Pattern | pass | - |
| TEST-001 | Test Coverage | pass | 6 packages with tests |
| TEST-002 | Mock Infrastructure | pass | 3 mock directories |
| INGRESS-001 | Route Config | pass | - |
| DOC-001 | README | pass | - |

**Non-Blocking Gaps:**
- No tests for Kafka consumers (handler logic)
- No tests for REST handlers
- No tests for reactor processor (Spawn methods)
