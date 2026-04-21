# Atlas-Maps Service Remediation Plan

**Last Updated:** 2026-01-13
**Service:** atlas-maps
**Audit Reference:** `/dev/audits/atlas-maps/audit.md`
**Overall Audit Status:** `pass` (15/15 checks passing)

---

## Executive Summary

The atlas-maps service has passed its backend audit with all 15 checks passing. Previous issues have been resolved through prior remediation work. Three non-blocking test coverage gaps were identified that should be addressed to achieve comprehensive test coverage.

**Audit Status:** `pass` (15/15 checks)
**Confidence Level:** `high`

**Previously Resolved Issues:**
- REST-001 (SetID method) - Resolved
- ARCH-001 (layer bypass) - Resolved
- MODEL-002 (builder validation) - Resolved
- TEST-001 (low coverage) - Significantly improved with 6 test packages
- TEST-002 (mock directories) - Resolved with 3 mock directories

**Remaining Non-Blocking Gaps:**
1. No tests for Kafka consumers (handler logic)
2. No tests for REST handlers
3. No tests for reactor processor (Spawn/SpawnAndEmit methods)

---

## Current State Analysis

### Test Coverage Summary

| Package | Test File | Lines | Coverage |
|---------|-----------|-------|----------|
| `kafka/message/character` | `kafka_test.go` | ~187 | Message serialization |
| `kafka/message/map` | `kafka_test.go` | - | Message serialization |
| `map` | `processor_test.go` | 575 | Comprehensive |
| `map/character` | `processor_test.go` | 326 | Includes tenant isolation |
| `map/monster` | `processor_test.go` | 1498+ | Comprehensive cooldown tests |
| `reactor` | `model_test.go` | 277 | Builder validation |

### Mock Infrastructure

| Directory | Status |
|-----------|--------|
| `map/mock/processor.go` | 9 methods implemented |
| `map/character/mock/processor.go` | 4 methods implemented |
| `reactor/mock/processor.go` | 4 methods implemented |

### Remaining Gaps

1. **Kafka Consumer Handlers** (`kafka/consumer/character/` and `kafka/consumer/cashshop/`)
   - Handlers create processors internally, making unit testing difficult
   - No handler-level tests for `handleStatusEventLogin`, `handleStatusEventLogout`, `handleStatusEventMapChanged`, `handleStatusEventChannelChanged`
   - No handler-level tests for cashshop `handleStatusEventEnter`, `handleStatusEventExit`

2. **REST Handler** (`map/resource.go`)
   - No HTTP-level tests for `handleGetCharactersInMap` endpoint
   - Path parsing utilities in `rest/handler.go` untested

3. **Reactor Processor** (`reactor/processor.go`)
   - `Spawn` and `SpawnAndEmit` methods tightly coupled to HTTP client
   - External service dependency makes unit testing challenging

---

## Proposed Future State

After remediation, the atlas-maps service will have:

1. Unit tests for Kafka consumer handlers (where feasible with refactoring)
2. HTTP-level tests for REST handlers using httptest
3. Unit tests for reactor processor with mocked HTTP dependencies
4. Test coverage for path parsing utilities

---

## Implementation Phases

### Phase 1: REST Handler Tests (Priority: P2, Effort: M)

Add unit tests for REST API handlers and path parsing utilities.

#### Section 1.1: Path Parsing Tests

**File:** `rest/handler_test.go`

| Task # | Task | Effort | Acceptance Criteria |
|--------|------|--------|---------------------|
| 1.1.1 | Test `ParseWorldId` with valid worldId | S | Correctly parses and passes worldId to handler |
| 1.1.2 | Test `ParseWorldId` with invalid worldId | S | Returns 400 Bad Request |
| 1.1.3 | Test `ParseChannelId` with valid channelId | S | Correctly parses and passes channelId to handler |
| 1.1.4 | Test `ParseChannelId` with invalid channelId | S | Returns 400 Bad Request |
| 1.1.5 | Test `ParseMapId` with valid mapId | S | Correctly parses and passes mapId to handler |
| 1.1.6 | Test `ParseMapId` with invalid mapId | S | Returns 400 Bad Request |

#### Section 1.2: Map Resource Handler Tests

**File:** `map/resource_test.go`

| Task # | Task | Effort | Acceptance Criteria |
|--------|------|--------|---------------------|
| 1.2.1 | Create test infrastructure with mock processor and HTTP test server | M | Can make HTTP requests with mocked dependencies |
| 1.2.2 | Test `handleGetCharactersInMap` returns characters in JSON:API format | M | Correct JSON:API structure |
| 1.2.3 | Test `handleGetCharactersInMap` returns empty array when no characters | S | Empty data array |
| 1.2.4 | Test `handleGetCharactersInMap` returns 500 on processor error | S | Returns 500 status |
| 1.2.5 | Test `handleGetCharactersInMap` with invalid path parameters | S | Returns 400 status |

---

### Phase 2: Kafka Consumer Handler Tests (Priority: P2, Effort: L)

Testing Kafka consumer handlers requires refactoring to support dependency injection, as handlers currently create processors internally.

#### Section 2.1: Refactor for Testability (Optional)

**Files:** `kafka/consumer/character/consumer.go`, `kafka/consumer/cashshop/consumer.go`

| Task # | Task | Effort | Acceptance Criteria |
|--------|------|--------|---------------------|
| 2.1.1 | Extract processor creation to configurable factory | M | Handlers accept processor factory |
| 2.1.2 | Create testable variants of handlers | M | Handlers can receive mock processors |

#### Section 2.2: Handler Tests (Depends on 2.1)

**File:** `kafka/consumer/character/consumer_test.go`

| Task # | Task | Effort | Dependencies | Acceptance Criteria |
|--------|------|--------|--------------|---------------------|
| 2.2.1 | Test `handleStatusEventLogin` calls `EnterAndEmit` | S | 2.1 | Correct parameters passed |
| 2.2.2 | Test `handleStatusEventLogin` ignores non-LOGIN events | S | 2.1 | No processor calls |
| 2.2.3 | Test `handleStatusEventLogout` calls `ExitAndEmit` | S | 2.1 | Correct parameters passed |
| 2.2.4 | Test `handleStatusEventLogout` ignores non-LOGOUT events | S | 2.1 | No processor calls |
| 2.2.5 | Test `handleStatusEventMapChanged` calls `TransitionMapAndEmit` | S | 2.1 | Correct parameters |
| 2.2.6 | Test `handleStatusEventChannelChanged` calls `TransitionChannelAndEmit` | S | 2.1 | Correct parameters |

**File:** `kafka/consumer/cashshop/consumer_test.go`

| Task # | Task | Effort | Dependencies | Acceptance Criteria |
|--------|------|--------|--------------|---------------------|
| 2.2.7 | Test `handleStatusEventEnter` calls `ExitAndEmit` | S | 2.1 | Character exits map |
| 2.2.8 | Test `handleStatusEventExit` calls `EnterAndEmit` | S | 2.1 | Character enters map |

---

### Phase 3: Reactor Processor Tests (Priority: P2, Effort: M)

The reactor processor is tightly coupled to external HTTP calls. Testing requires either HTTP mocking or interface extraction.

#### Section 3.1: Mock HTTP Dependencies

| Task # | Task | Effort | Acceptance Criteria |
|--------|------|--------|---------------------|
| 3.1.1 | Create HTTP test server or mock for reactor REST client | M | Can stub reactor API responses |
| 3.1.2 | Create mock for `data/map/reactor.Processor` | M | Can stub `InMapProvider` responses |
| 3.1.3 | Create mock producer for `SpawnAndEmit` | S | Captures Kafka messages |

#### Section 3.2: Processor Method Tests

**File:** `reactor/processor_test.go` (extend existing)

| Task # | Task | Effort | Dependencies | Acceptance Criteria |
|--------|------|--------|--------------|---------------------|
| 3.2.1 | Test `GetInMap` returns reactors | M | 3.1.1 | Correctly deserializes models |
| 3.2.2 | Test `GetInMap` handles empty response | S | 3.1.1 | Returns empty slice |
| 3.2.3 | Test `GetInMap` handles error | S | 3.1.1 | Returns error |
| 3.2.4 | Test `doesNotExist` filter excludes existing reactors | S | None | Matching reactors filtered |
| 3.2.5 | Test `doesNotExist` filter includes new reactors | S | None | Non-matching pass through |
| 3.2.6 | Test `Spawn` issues create commands for new reactors only | M | 3.1.* | Correct commands buffered |
| 3.2.7 | Test `SpawnAndEmit` emits messages | M | 3.1.3 | Messages emitted via producer |

---

## Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Refactoring Kafka handlers may introduce bugs | Medium | Low | Thorough integration testing |
| HTTP mocking complexity | Medium | Medium | Use `httptest.Server` from standard library |
| Test isolation with singleton registries | Low | Low | Use unique tenant IDs per test |
| External API contract changes | Medium | Low | Document expected responses in tests |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Audit checks passing | 15/15 | 15/15 (maintain) |
| Packages with tests | 6 | 8+ |
| Kafka consumer handler tests | 0 | 8+ (if refactored) |
| REST handler tests | 0 | 5+ |
| Reactor processor method tests | 0 | 7+ |

---

## Required Resources and Dependencies

### Dependencies
- Existing mock infrastructure in `map/mock/`, `map/character/mock/`, `reactor/mock/`
- Standard library `net/http/httptest` for HTTP handler testing
- `github.com/sirupsen/logrus/hooks/test` for logger testing
- Existing test patterns from `map/processor_test.go`

### Files to Create
| File | Description |
|------|-------------|
| `rest/handler_test.go` | Path parsing utility tests |
| `map/resource_test.go` | Map resource handler tests |
| `kafka/consumer/character/consumer_test.go` | Character consumer handler tests (requires refactor) |
| `kafka/consumer/cashshop/consumer_test.go` | CashShop consumer handler tests (requires refactor) |

### Files to Extend
| File | Description |
|------|-------------|
| `reactor/processor_test.go` | Add Spawn/SpawnAndEmit tests |

---

## Implementation Notes

### Why Full Consumer Testing Requires Refactoring

The current Kafka consumer handlers create processors internally:

```go
// Current pattern in consumer.go
func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event ...) {
    p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))  // Created internally
    _ = p.EnterAndEmit(...)
}
```

To unit test the handler logic, we would need to inject the processor:

```go
// Testable pattern
func handleStatusEventLoginWithProcessor(l logrus.FieldLogger, ctx context.Context, event ..., p Processor) {
    _ = p.EnterAndEmit(...)
}
```

**Recommendation:** Consider whether the refactoring effort is worth the test coverage gain, given that the processor methods are already well-tested independently.

### Alternative: Integration-Style Testing

Instead of unit testing handlers, consider:
1. Testing message serialization (already done in `kafka/message/*/kafka_test.go`)
2. Testing processor methods (already done in `map/processor_test.go`)
3. Running end-to-end integration tests in a test environment

This approach trusts that if messages deserialize correctly and processors work correctly, the handlers will work correctly.

---

## Appendix: Test Pattern Examples

### HTTP Handler Test Pattern
```go
func TestHandleGetCharactersInMap_ReturnsCharacters(t *testing.T) {
    l, _ := test.NewNullLogger()

    // Create mock processor
    mp := &mock.Processor{
        GetCharactersInMapFunc: func(transactionId uuid.UUID, worldId world.Id,
            channelId channel.Id, mapId _map.Id) ([]uint32, error) {
            return []uint32{123, 456, 789}, nil
        },
    }

    // Setup test server
    router := mux.NewRouter()
    // ... register handler with mocked processor

    req := httptest.NewRequest(http.MethodGet,
        "/worlds/1/channels/2/maps/100000000/characters", nil)
    rr := httptest.NewRecorder()

    router.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rr.Code)
    }
    // Assert JSON:API response structure
}
```

### Path Parsing Test Pattern
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

    if capturedWorldId != 5 {
        t.Errorf("Expected worldId 5, got %d", capturedWorldId)
    }
}
```
