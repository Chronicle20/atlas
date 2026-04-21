# Atlas-Chairs Service Remediation Plan

**Service Path:** `services/atlas-chairs/atlas.com/chairs`
**Audit Reference:** `docs/audits/atlas-chairs/audit.md`
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This plan addresses the issues identified in the atlas-chairs backend audit. The service is a small, in-memory microservice that manages chair state for characters in the game world. While the service passes most architectural checks (immutability, multi-tenancy, layer separation), it has one blocking issue (missing ingress route) and several non-blocking issues requiring attention.

**Key Objectives:**
1. **P0:** Add missing ingress route for `/api/chairs/{characterId}` endpoint (BLOCKING)
2. **P1:** Add comprehensive unit tests for processor and registry logic
3. **P2:** Migrate local REST handlers to use shared `server.RegisterHandler` pattern
4. **P2:** Consider builder pattern for chair model (optional, given model simplicity)

**Overall Effort:** Medium (M)
**Risk Level:** Low - changes are additive or refactoring with well-defined patterns

---

## 2. Current State Analysis

### What's Working Well
- Immutable domain model with private fields and accessor methods
- Proper multi-tenancy context extraction via `tenant.MustFromContext`
- Kafka producer uses context-aware header decorators
- Correct layer separation (handlers -> processors)
- JSON:API compliance with proper RestModel implementation
- Comprehensive README documentation

### What Needs Work
| Issue ID | Severity | Description |
|----------|----------|-------------|
| INFRA-001 | High | Missing ingress route for `/api/chairs/{characterId}` |
| TEST-001 | Medium | No test files exist in the service |
| STRUCT-003 | Low | Local `rest/handler.go` duplicates shared patterns |
| STRUCT-001 | Low | Missing builder pattern for model construction |

### Service Architecture Overview
```
atlas-chairs/atlas.com/chairs/
├── main.go                 # Application entry point
├── chair/                  # Chair domain package
│   ├── model.go           # Immutable domain model (2 fields: id, chairType)
│   ├── processor.go       # Business logic: Set/Clear/GetById
│   ├── registry.go        # In-memory state with mutex synchronization
│   └── rest.go            # JSON:API transformation
├── character/             # Character tracking domain
│   ├── processor.go       # Enter/Exit/Transition logic
│   └── registry.go        # In-memory map registry with per-key locks
└── rest/
    └── handler.go         # Local handler registration (to be migrated)
```

---

## 3. Proposed Future State

After remediation, the service will:

1. Have both REST endpoints accessible through the ingress:
   - `/api/chairs/{characterId}` - Get chair by character
   - `/api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chairs` - Get chairs in map

2. Have comprehensive test coverage for:
   - Chair processor (Set/Clear/GetById operations)
   - Chair registry (concurrent access patterns)
   - Character processor (Enter/Exit/Transition operations)
   - Character registry (concurrent access patterns)

3. Use shared server patterns from atlas-rest library for handler registration, improving consistency across services.

---

## 4. Implementation Phases

### Phase 1: Infrastructure Fix (P0 - Blocking)
**Objective:** Enable external access to `/api/chairs/{characterId}` endpoint

### Phase 2: Test Coverage (P1)
**Objective:** Add comprehensive unit tests following existing patterns from atlas-account

### Phase 3: Code Quality (P2)
**Objective:** Migrate to shared patterns and consider builder implementation

---

## 5. Detailed Tasks

### Phase 1: Ingress Configuration

#### Task 1.1: Add Missing Ingress Route
**Effort:** S (Small)
**Priority:** P0 (Blocking)
**Dependencies:** None

**Description:**
Add an ingress location block for the `/api/chairs` endpoint in `atlas-ingress.yml`.

**Implementation Details:**
- Location: `atlas-ingress.yml` (around line 134, after existing chairs route)
- Pattern: `location ~ ^/api/chairs(/.*)?$`
- Proxy target: `http://atlas-chairs.atlas.svc.cluster.local:8080`

**Acceptance Criteria:**
- [ ] Ingress route added to `atlas-ingress.yml`
- [ ] Route pattern matches `/api/chairs` and `/api/chairs/{characterId}`
- [ ] Route proxies to correct service URL
- [ ] Route placed in alphabetical order with other routes

---

### Phase 2: Test Coverage

#### Task 2.1: Create Chair Processor Tests
**Effort:** M (Medium)
**Priority:** P1
**Dependencies:** None

**Description:**
Create `chair/processor_test.go` with table-driven tests for processor logic.

**Test Cases:**
1. `TestGetById_Success` - Character has a chair entry
2. `TestGetById_NotFound` - Character has no chair entry
3. `TestSet_Success_FixedChair` - Valid fixed chair assignment
4. `TestSet_Success_PortableChair` - Valid portable chair assignment
5. `TestSet_AlreadySitting` - Character already on a chair
6. `TestSet_InvalidFixedChair` - Chair ID exceeds map seats
7. `TestSet_InvalidPortableChair` - Chair item category not 301
8. `TestClear_Success` - Clear existing chair assignment
9. `TestClear_NotSitting` - Clear when not sitting

**Implementation Notes:**
- Mock the Kafka producer to avoid external dependencies
- Mock the map data processor for fixed chair validation
- Use `test.NewNullLogger()` from logrus for silent logging
- Follow patterns from `atlas-account/processor_test.go`

**Acceptance Criteria:**
- [ ] All test cases implemented
- [ ] Tests pass with `go test ./chair/...`
- [ ] Tests are table-driven where applicable
- [ ] No external service dependencies in tests

---

#### Task 2.2: Create Chair Registry Tests
**Effort:** S (Small)
**Priority:** P1
**Dependencies:** None

**Description:**
Create `chair/registry_test.go` with tests for concurrent access patterns.

**Test Cases:**
1. `TestRegistry_GetSet` - Basic get/set operations
2. `TestRegistry_Clear` - Clear existing entry returns true
3. `TestRegistry_Clear_NotExists` - Clear non-existent entry returns false
4. `TestRegistry_Concurrent` - Concurrent access with goroutines

**Acceptance Criteria:**
- [ ] All test cases implemented
- [ ] Concurrent test verifies mutex protection
- [ ] Tests pass with race detector: `go test -race ./chair/...`

---

#### Task 2.3: Create Character Processor Tests
**Effort:** S (Small)
**Priority:** P1
**Dependencies:** None

**Description:**
Create `character/processor_test.go` with tests for character tracking logic.

**Test Cases:**
1. `TestInMapProvider` - Returns characters in specific map
2. `TestEnter` - Character enters map
3. `TestExit` - Character exits map
4. `TestTransitionMap` - Character moves between maps
5. `TestTransitionChannel` - Character moves between channels

**Implementation Notes:**
- Create test tenant with `tenant.WithContext`
- Use `field.NewBuilder` to create test fields
- Verify registry state changes after operations

**Acceptance Criteria:**
- [ ] All test cases implemented
- [ ] Tests verify tenant isolation
- [ ] Tests pass with `go test ./character/...`

---

#### Task 2.4: Create Character Registry Tests
**Effort:** S (Small)
**Priority:** P1
**Dependencies:** None

**Description:**
Create `character/registry_test.go` with tests for concurrent access patterns.

**Test Cases:**
1. `TestRegistry_AddCharacter` - Add character to map
2. `TestRegistry_AddCharacter_Duplicate` - Adding same character twice doesn't duplicate
3. `TestRegistry_RemoveCharacter` - Remove character from map
4. `TestRegistry_RemoveCharacter_NotExists` - Remove non-existent character
5. `TestRegistry_GetInMap` - Get all characters in map
6. `TestRegistry_Concurrent` - Concurrent access to same map key

**Acceptance Criteria:**
- [ ] All test cases implemented
- [ ] Tests verify per-key locking behavior
- [ ] Tests pass with race detector: `go test -race ./character/...`

---

### Phase 3: Code Quality Improvements

#### Task 3.1: Migrate REST Handlers to Shared Pattern
**Effort:** S (Small)
**Priority:** P2
**Dependencies:** None

**Description:**
Investigate whether the local `rest/handler.go` can be replaced with shared patterns from the atlas-rest library.

**Analysis Required:**
- Review `server.RegisterHandler` from atlas-rest library
- Compare with local `RegisterHandler` implementation
- Identify any custom functionality in local implementation

**Current Local Implementation:**
- `RegisterHandler` - Wraps handlers with span and tenant parsing
- `RegisterInputHandler` - Additionally parses JSON:API input
- `ParseInput` - Generic JSON:API body deserialization

**Implementation Notes:**
- The local implementation includes `ParseWorldId`, `ParseChannelId`, `ParseMapId`, `ParseCharacterId` helpers
- These path parameter parsers may need to remain local
- Only migrate handler registration if atlas-rest provides equivalent functionality

**Acceptance Criteria:**
- [ ] Analysis completed documenting migration feasibility
- [ ] If feasible: handlers migrated to shared pattern
- [ ] If not feasible: document reasons and close as "won't fix"
- [ ] Local path parsers preserved if needed

---

#### Task 3.2: Consider Builder Pattern for Chair Model (Optional)
**Effort:** S (Small)
**Priority:** P2
**Dependencies:** None

**Description:**
Evaluate whether adding a builder pattern for the Chair model provides value.

**Current State:**
- Chair model has only 2 fields: `id uint32` and `chairType string`
- Model is constructed directly in processor: `Model{id: chairId, chairType: chairType}`
- No validation beyond type checking

**Decision Criteria:**
- Builder pattern recommended if validation requirements expand
- Current model simplicity suggests this is low priority
- May close as "won't fix" given model simplicity

**Acceptance Criteria:**
- [ ] Decision documented with rationale
- [ ] If builder added: follows pattern from other services
- [ ] If not added: document decision to skip

---

## 6. Risk Assessment and Mitigation

### Risk 1: Ingress Configuration Errors
**Likelihood:** Low
**Impact:** High (could affect other services)
**Mitigation:**
- Copy existing route pattern structure exactly
- Test in staging environment before production
- Verify existing routes still work after change

### Risk 2: Test Flakiness with Concurrency Tests
**Likelihood:** Medium
**Impact:** Low (tests only)
**Mitigation:**
- Use proper synchronization primitives
- Run tests multiple times with race detector
- Follow established patterns from atlas-account tests

### Risk 3: Breaking Changes in Handler Migration
**Likelihood:** Low
**Impact:** Medium (could break REST endpoints)
**Mitigation:**
- Thoroughly analyze current behavior before migration
- Maintain backwards compatibility
- Test both endpoints after migration

---

## 7. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Ingress Route | Working | HTTP 200 response from `/api/chairs/1` |
| Test Coverage | >80% | `go test -cover ./...` |
| Race Conditions | 0 | `go test -race ./...` passes |
| Build Status | Passing | All tests pass in CI |

---

## 8. Required Resources and Dependencies

### Technical Dependencies
- Access to `atlas-ingress.yml` for ingress changes
- Knowledge of atlas-rest library patterns
- Access to `atlas-account` tests as reference implementation

### Testing Infrastructure
- Go test runner with race detector support
- Mock frameworks may be needed for Kafka producer
- Test tenant/context setup utilities

### External Service Dependencies (for integration testing)
- atlas-chairs service endpoint
- Kubernetes ingress controller

---

## 9. Notes and Considerations

1. **In-Memory Design:** The registry pattern is intentional for this service. Chair state is transient and doesn't need persistence. No changes planned to this architecture.

2. **Character Registry Duplication:** The character registry duplicates state from other services but is necessary for the "chairs in map" query pattern. This is acceptable.

3. **TODO Comment:** `chair/processor.go:73` contains `// TODO ensure character has item` for portable chair validation. This represents incomplete validation logic but is out of scope for this remediation (pre-existing technical debt).

4. **Cross-Service Calls:** The `data/map/processor.go` makes REST calls to the DATA service. This follows correct patterns and needs no changes.
