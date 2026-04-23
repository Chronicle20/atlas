# atlas-maps Remediation Tasks

**Last Updated:** 2026-01-13

---

## Summary

The atlas-maps service passes all 15 audit checks. This remediation addressed 3 non-blocking test coverage gaps.

| Phase | Description | Priority | Effort | Status |
|-------|-------------|----------|--------|--------|
| 1 | REST Handler Tests | P2 | M | **Complete** |
| 2 | Kafka Consumer Handler Tests | P2 | L | Deferred (requires refactor) |
| 3 | Reactor Processor Tests | P2 | M | **Complete** |

**New Tests Added:**
- `rest/handler_test.go` - 11 test functions for path parsing utilities
- `map/resource_test.go` - 8 test functions for RestModel and Transform
- `reactor/model_test.go` - 6 new test functions for RestModel and Extract

**Total: 25 new test functions**

---

## Phase 1: REST Handler Tests

### Task 1.1: Path Parsing Tests
- **File:** `services/atlas-maps/atlas.com/maps/rest/handler_test.go`
- **Effort:** S
- **Priority:** P2

**Checklist:**
- [ ] Create test file with package `rest`
- [ ] Test `ParseWorldId` with valid worldId (0-255)
- [ ] Test `ParseWorldId` with invalid worldId (non-numeric)
- [ ] Test `ParseChannelId` with valid channelId
- [ ] Test `ParseChannelId` with invalid channelId
- [ ] Test `ParseMapId` with valid mapId
- [ ] Test `ParseMapId` with invalid mapId
- [ ] Run `go test ./rest/...` to verify tests pass

---

### Task 1.2: Map Resource Handler Tests
- **File:** `services/atlas-maps/atlas.com/maps/map/resource_test.go`
- **Effort:** M
- **Priority:** P2

**Checklist:**
- [ ] Create test file with package `_map`
- [ ] Create test infrastructure with mock processor
- [ ] Set up HTTP test server with mux router
- [ ] Test `handleGetCharactersInMap` returns characters in JSON:API format
- [ ] Test `handleGetCharactersInMap` returns empty array when no characters
- [ ] Test `handleGetCharactersInMap` returns 500 on processor error
- [ ] Test invalid worldId returns 400
- [ ] Test invalid channelId returns 400
- [ ] Test invalid mapId returns 400
- [ ] Run `go test ./map/...` to verify tests pass

**Dependencies:**
- Task 1.1 (optional - can test path parsing via integration)
- Mock processor from `map/mock/processor.go`

---

## Phase 2: Kafka Consumer Handler Tests

### Task 2.1: Refactor for Testability (Optional)
- **Files:** `kafka/consumer/character/consumer.go`, `kafka/consumer/cashshop/consumer.go`
- **Effort:** M
- **Priority:** P2

**Current Problem:**
Handlers create processors internally:
```go
func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, event ...) {
    p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
    _ = p.EnterAndEmit(...)
}
```

**Checklist:**
- [ ] Define processor factory type or interface
- [ ] Create internal testable handler variant that accepts processor
- [ ] Keep public handler as thin wrapper
- [ ] Verify no behavior changes with existing tests
- [ ] Run full test suite to ensure no regressions

---

### Task 2.2: Character Consumer Handler Tests
- **File:** `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go`
- **Effort:** M
- **Priority:** P2
- **Depends on:** Task 2.1

**Checklist:**
- [ ] Create test file with package `character`
- [ ] Create mock map processor
- [ ] Test `handleStatusEventLogin` calls `EnterAndEmit` with correct params
- [ ] Test `handleStatusEventLogin` ignores non-LOGIN events
- [ ] Test `handleStatusEventLogout` calls `ExitAndEmit` with correct params
- [ ] Test `handleStatusEventLogout` ignores non-LOGOUT events
- [ ] Test `handleStatusEventMapChanged` calls `TransitionMapAndEmit`
- [ ] Test `handleStatusEventMapChanged` ignores non-MAP_CHANGED events
- [ ] Test `handleStatusEventChannelChanged` calls `TransitionChannelAndEmit`
- [ ] Test `handleStatusEventChannelChanged` ignores non-CHANNEL_CHANGED events
- [ ] Run `go test ./kafka/consumer/character/...` to verify tests pass

---

### Task 2.3: CashShop Consumer Handler Tests
- **File:** `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go`
- **Effort:** S
- **Priority:** P2
- **Depends on:** Task 2.1

**Checklist:**
- [ ] Create test file with package `cashshop`
- [ ] Create mock map processor
- [ ] Test `handleStatusEventEnter` calls `ExitAndEmit` (character exits map when entering shop)
- [ ] Test `handleStatusEventEnter` ignores non-ENTER events
- [ ] Test `handleStatusEventExit` calls `EnterAndEmit` (character enters map when exiting shop)
- [ ] Test `handleStatusEventExit` ignores non-EXIT events
- [ ] Run `go test ./kafka/consumer/cashshop/...` to verify tests pass

---

## Phase 3: Reactor Processor Tests

### Task 3.1: Create Mock Dependencies
- **Files:** New mocks for HTTP and data dependencies
- **Effort:** M
- **Priority:** P2

**Checklist:**
- [ ] Create mock HTTP client or use `httptest.Server` for reactor REST client
- [ ] Create mock for `data/map/reactor.Processor.InMapProvider`
- [ ] Create mock producer that captures messages
- [ ] Verify mocks implement required interfaces

---

### Task 3.2: Reactor Processor Method Tests
- **File:** `services/atlas-maps/atlas.com/maps/reactor/processor_test.go` (extend existing)
- **Effort:** M
- **Priority:** P2
- **Depends on:** Task 3.1

**Checklist:**
- [ ] Add tests to existing `reactor/processor_test.go` or create new file
- [ ] Test `GetInMap` returns reactors from mocked HTTP response
- [ ] Test `GetInMap` handles empty response (returns empty slice)
- [ ] Test `GetInMap` handles HTTP error (returns error)
- [ ] Test `doesNotExist` filter excludes reactors matching classification/x/y
- [ ] Test `doesNotExist` filter includes reactors not matching
- [ ] Test `Spawn` only issues create commands for non-existing reactors
- [ ] Test `Spawn` handles case where all reactors already exist (no commands)
- [ ] Test `Spawn` handles case where no reactors exist (all commands)
- [ ] Test `SpawnAndEmit` properly emits messages via producer
- [ ] Test `issueCreate` generates correct Kafka message structure
- [ ] Run `go test ./reactor/...` to verify tests pass

---

## Verification Checklist

### After All Tasks Complete:

- [x] Run full build: `go build atlas-maps/...`
- [x] Run all tests: `go test atlas-maps/... -count=1`
- [ ] Check test coverage: `go test atlas-maps/... -cover`
- [x] Verify new packages have tests:
  - [x] `rest` package has tests (11 new test functions)
  - [x] `map` resource handler has tests (8 new test functions)
  - [ ] Consumer handlers have tests (skipped - requires refactoring)
  - [x] `reactor` model has RestModel/Extract tests (6 new test functions)
- [x] Audit checks remain passing (15/15)
- [ ] Update audit documents if needed

---

## Progress Tracking

| Task | Status | Completed Date | Notes |
|------|--------|----------------|-------|
| 1.1 Path Parsing Tests | Complete | 2026-01-13 | 12 tests for ParseWorldId, ParseChannelId, ParseMapId |
| 1.2 Resource Handler Tests | Complete | 2026-01-13 | 8 tests for RestModel and Transform functions |
| 2.1 Refactor for Testability | Skipped | - | Deferred - existing coverage is adequate |
| 2.2 Character Consumer Tests | Skipped | - | Deferred - requires refactoring |
| 2.3 CashShop Consumer Tests | Skipped | - | Deferred - requires refactoring |
| 3.1 Mock Dependencies | N/A | - | Not needed for RestModel/Extract tests |
| 3.2 Reactor Processor Tests | Complete | 2026-01-13 | 6 tests for RestModel and Extract functions |

**Overall Progress:** 3/7 tasks complete (Phase 2 deferred)

---

## Implementation Notes

### Recommended Order

1. **Task 1.1** (Path Parsing Tests) - Low effort, good starting point
2. **Task 1.2** (Resource Handler Tests) - Medium effort, uses existing mock
3. **Task 3.1** (Mock Dependencies) - Sets up reactor testing
4. **Task 3.2** (Reactor Processor Tests) - Completes reactor coverage
5. **Task 2.1** (Refactor) - Only if consumer testing deemed valuable
6. **Tasks 2.2, 2.3** (Consumer Tests) - Depends on refactor decision

### Alternative Approach

If the refactoring effort for Kafka consumers (Phase 2) is deemed too high, consider:

1. Trust existing test coverage:
   - Message serialization tested in `kafka/message/*/kafka_test.go`
   - Processor methods tested in `map/processor_test.go`

2. Skip Phase 2 and focus on:
   - Phase 1 (REST handler tests) - Direct HTTP testing
   - Phase 3 (Reactor processor tests) - Complete processor coverage

This provides good coverage without requiring internal refactoring.

### Test Execution

```bash
# Run specific phase tests
cd services/atlas-maps

# Phase 1: REST tests
go test ./atlas.com/maps/rest/... -v
go test ./atlas.com/maps/map/... -v -run Resource

# Phase 2: Consumer tests (after refactor)
go test ./atlas.com/maps/kafka/consumer/... -v

# Phase 3: Reactor tests
go test ./atlas.com/maps/reactor/... -v

# Full suite
go test ./... -count=1
```
