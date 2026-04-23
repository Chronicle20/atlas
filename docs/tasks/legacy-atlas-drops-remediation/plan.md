# Atlas-Drops Service Remediation Plan

**Service:** `atlas-drops`
**Path:** `services/atlas-drops/atlas.com/drops`
**Audit Status:** `needs-work` (73% pass rate)
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

The `atlas-drops` service manages in-memory drop state for the game world using a singleton registry with TTL-based expiration. The audit identified the service as architecturally sound but lacking mandatory test coverage.

**Blocking Issues:** 1 (TEST-001: No test coverage)
**Non-Blocking Issues:** 3 (TEST-002, DOC-001, ARCH-004)

**Remediation Goal:** Bring the service to `pass` status by adding comprehensive test coverage and addressing all identified gaps.

---

## 2. Current State Analysis

### What Works Well
- Processor interface pattern with pure and `AndEmit` variants
- Proper Kafka producer initialization with context decorators
- JSON:API compliant REST models
- Handler-to-processor delegation (no direct provider calls)
- Context-based multi-tenancy
- Model immutability with builder pattern
- Thread-safe registry with per-drop and per-map locks

### What Needs Work
| Issue ID | Severity | Description |
|----------|----------|-------------|
| TEST-001 | **High** | Zero test files exist - blocking issue |
| TEST-002 | Medium | No mock infrastructure for Processor interface |
| DOC-001 | Low | No README documentation |
| ARCH-004 | Low | Builder.Build() returns no error for validation |

---

## 3. Proposed Future State

After remediation, the service will have:
- Comprehensive test coverage for all domain logic
- Mock implementations enabling isolated unit testing
- Service-level README documenting REST endpoints and Kafka topics
- Builder validation for required fields (optional enhancement)

**Target Status:** `pass` with >80% test coverage

---

## 4. Implementation Phases

### Phase 1: Test Infrastructure Setup (P0 - Blocking)
**Objective:** Create foundation for testing

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 1.1 | Create `drop/mock/` directory structure | S | Directory exists |
| 1.2 | Implement `ProcessorMock` for Processor interface | M | Mock implements all 16 interface methods |
| 1.3 | Create test helper utilities (tenant creation, model builders) | S | Reusable test fixtures work correctly |

### Phase 2: Registry Tests (P0 - Blocking)
**Objective:** Test the in-memory registry - critical thread-safe code

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 2.1 | Test `CreateDrop` - single drop creation | S | Drop created with correct ID and status |
| 2.2 | Test `CreateDrop` - multiple drops in same map | S | All drops correctly tracked in `dropsInMap` |
| 2.3 | Test `CreateDrop` - multi-tenant isolation | S | Drops isolated by tenant |
| 2.4 | Test `ReserveDrop` - successful reservation | S | Status changes to RESERVED |
| 2.5 | Test `ReserveDrop` - already reserved by same character | S | Returns existing drop without error |
| 2.6 | Test `ReserveDrop` - already reserved by different character | S | Returns error |
| 2.7 | Test `ReserveDrop` - drop not found | S | Returns error |
| 2.8 | Test `CancelDropReservation` - valid cancellation | S | Status returns to AVAILABLE |
| 2.9 | Test `CancelDropReservation` - wrong character | S | No changes made |
| 2.10 | Test `RemoveDrop` - successful removal | S | Drop removed from map and registry |
| 2.11 | Test `RemoveDrop` - drop not found | S | Returns empty model without error |
| 2.12 | Test `GetDrop` - existing drop | S | Returns correct model |
| 2.13 | Test `GetDrop` - non-existent drop | S | Returns error |
| 2.14 | Test `GetDropsForMap` - returns correct drops | S | Only drops for specified map returned |
| 2.15 | Test `GetAllDrops` | S | All drops returned |
| 2.16 | Test unique ID generation wraparound | M | IDs wrap at 2 billion correctly |

### Phase 3: Model and Builder Tests (P0 - Blocking)
**Objective:** Test immutable model behavior

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 3.1 | Test `NewModelBuilder` default values | S | Correct defaults set |
| 3.2 | Test builder fluent setters | S | All setters return builder |
| 3.3 | Test `Build()` creates correct model | S | All fields transferred correctly |
| 3.4 | Test `CloneModelBuilder` copies all fields | S | Clone matches original |
| 3.5 | Test `Model.Reserve()` returns new instance | S | Original unchanged, new has RESERVED status |
| 3.6 | Test `Model.CancelReservation()` returns new instance | S | Original unchanged, new has AVAILABLE status |

### Phase 4: Processor Tests (P0 - Blocking)
**Objective:** Test business logic layer

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 4.1 | Test `Spawn` - creates drop and buffers message | M | Drop in registry, message in buffer |
| 4.2 | Test `SpawnForCharacter` - creates character drop | S | Drop created without equipment lookup |
| 4.3 | Test `Reserve` - successful reservation emits message | M | Message buffered with correct topic |
| 4.4 | Test `Reserve` - failed reservation emits failure message | M | Failure message buffered |
| 4.5 | Test `CancelReservation` - emits cancellation message | S | Message buffered correctly |
| 4.6 | Test `Gather` - removes drop and emits message | M | Drop removed, pickup message buffered |
| 4.7 | Test `Expire` - removes drop and emits message | M | Drop removed, expiry message buffered |
| 4.8 | Test `GetById` - returns correct drop | S | Model matches registry |
| 4.9 | Test `GetForMap` - returns filtered drops | S | Only map drops returned |
| 4.10 | Test `ByIdProvider` - functional composition works | S | Provider returns correct model |
| 4.11 | Test `ForMapProvider` - functional composition works | S | Provider returns correct slice |

### Phase 5: REST Model Tests (P1)
**Objective:** Test JSON:API compliance

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 5.1 | Test `RestModel.GetName()` returns "drops" | S | Returns correct resource name |
| 5.2 | Test `RestModel.GetID()` returns string ID | S | Correct string conversion |
| 5.3 | Test `RestModel.SetID()` parses string to uint32 | S | Correct parsing |
| 5.4 | Test `Transform` converts all fields correctly | S | All fields mapped |

### Phase 6: Documentation (P2)
**Objective:** Create service README

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 6.1 | Document REST endpoints | S | All endpoints listed with methods |
| 6.2 | Document Kafka topics (produced/consumed) | S | All topics documented |
| 6.3 | Document service behavior and TTL expiration | S | In-memory design explained |
| 6.4 | Document cross-service dependencies (equipment) | S | External calls documented |

### Phase 7: Builder Validation Enhancement (P2 - Optional)
**Objective:** Add validation to Builder.Build()

| Task | Description | Effort | Acceptance Criteria |
|------|-------------|--------|---------------------|
| 7.1 | Change `Build()` signature to return `(Model, error)` | M | Signature updated |
| 7.2 | Add validation for required fields | S | Zero values rejected |
| 7.3 | Update all call sites | M | All callers handle error |
| 7.4 | Add builder validation tests | S | Tests cover edge cases |

---

## 5. Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Registry tests affect singleton state | High | Medium | Use `t.Cleanup()` to reset registry, or add registry reset method for tests |
| Concurrency tests are flaky | Medium | Medium | Use `-race` flag, add deterministic tests before stress tests |
| Builder validation breaks existing code | Low | High | Make validation changes backward-compatible initially |
| Equipment mocking complexity | Medium | Low | Focus on registry/processor tests first; equipment integration can use stubs |

---

## 6. Success Metrics

| Metric | Target |
|--------|--------|
| Test coverage | >80% |
| All blocking issues resolved | Yes |
| All non-blocking issues resolved | Yes |
| Tests pass with `-race` flag | Yes |
| Service audit status | `pass` |

---

## 7. Dependencies

### Internal Dependencies
- None - this is a standalone remediation effort

### External Dependencies
- `github.com/Chronicle20/atlas-tenant` - for test tenant creation
- `github.com/google/uuid` - for test UUIDs
- Standard Go testing package

### Files to Create
```
services/atlas-drops/atlas.com/drops/
├── drop/
│   ├── mock/
│   │   └── processor.go          # ProcessorMock implementation
│   ├── registry_test.go          # Registry unit tests
│   ├── model_test.go             # Model and builder tests
│   ├── processor_test.go         # Processor logic tests
│   └── rest_test.go              # REST model tests
└── README.md                      # Service documentation
```

---

## 8. Implementation Notes

### Registry Testing Strategy
The registry is a singleton with internal state. Testing requires either:
1. Adding a `Reset()` method to clear state between tests (recommended)
2. Using subtests with careful ordering
3. Creating a registry interface for mocking (over-engineering for this use case)

**Recommended approach:** Add an exported `ResetForTesting()` function in a test file that clears the singleton's maps. This keeps production code clean while enabling isolated tests.

### Processor Testing Strategy
The processor depends on:
- Registry (in-memory, can be real)
- Kafka producer (should be mocked via message buffer pattern)
- Equipment service (cross-service, should be mocked)
- Logger and context (can use real implementations)

Use the message buffer directly for testing - no need to mock Kafka. The buffer pattern is designed for exactly this use case.

### Concurrency Testing
The registry has complex locking with per-drop and per-map mutexes. Add tests that:
1. Create drops concurrently from multiple goroutines
2. Reserve the same drop from multiple goroutines
3. Mix create/reserve/remove operations concurrently

Run with `go test -race` to detect race conditions.
