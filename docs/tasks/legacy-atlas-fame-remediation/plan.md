# Atlas Fame Service Remediation Plan

**Last Updated:** 2026-01-13
**Source Audit:** `/dev/audits/atlas-fame/audit.md`
**Service Path:** `services/atlas-fame`
**Current Status:** `needs-work` (78% pass rate - 14/18 checks passing)

---

## Executive Summary

The `atlas-fame` service requires remediation to align with Atlas backend architecture guidelines. The audit identified **2 blocking issues** and **4 non-blocking issues** that need to be addressed.

### Blocking Issues (P0)
1. **ARCH-005:** Duplicate `.Find(&result)` bug in `fame/provider.go:15` causing double query execution
2. **ARCH-003:** Missing `fame/builder.go` - domain model creation bypasses validation

### Non-Blocking Issues (P1-P2)
1. **ARCH-002:** Missing model accessors for `tenantId`, `id`, `characterId`, `amount`
2. **ARCH-009:** `character/rest.go` `GetName()` uses pointer receiver instead of value receiver
3. **ARCH-013:** No unit tests in the service
4. **ARCH-018:** Legacy functions retained for backward compatibility

---

## Current State Analysis

### Service Overview
- **Type:** Kafka-only service (no REST endpoints)
- **Purpose:** Manages fame/reputation changes between characters
- **Business Rules:**
  - Characters must be level 15+ to give fame
  - Can only give fame once per day
  - Can only give fame to the same target once per month

### Architecture Compliance

| Component | Status | Notes |
|-----------|--------|-------|
| Layer Separation | Pass | Consumer -> Processor -> Provider/Administrator |
| Model Immutability | Warn | Private fields present, missing accessors |
| Builder Pattern | **Fail** | No builder.go exists |
| Processor Pattern | Pass | Interface + Impl with AndEmit variants |
| Provider Pattern | **Fail** | Duplicate Find() bug |
| Administrator Pattern | Pass | Write operations properly separated |
| Producer Pattern | Pass | Context-aware header decorators |
| Multi-Tenancy | Pass | tenant.MustFromContext used consistently |
| Message Buffer | Pass | Atomic Kafka emissions |
| Testing | Warn | No test files |

---

## Proposed Future State

After remediation, the service will:
1. Have a fully functional provider without query bugs
2. Use validated builder pattern for domain model creation
3. Expose all necessary model accessors for debugging and serialization
4. Follow consistent receiver conventions across all REST models
5. Have comprehensive unit test coverage
6. Use modern Processor interface pattern (legacy functions deprecated/removed)

---

## Implementation Phases

### Phase 1: Critical Bug Fixes (P0)
**Goal:** Fix blocking issues that affect correctness

#### Task 1.1: Fix Duplicate Find() Bug in Provider
- **File:** `services/atlas-fame/atlas.com/fame/fame/provider.go`
- **Line:** 15
- **Issue:** `.Find(&result).Find(&result)` executes query twice
- **Fix:** Remove duplicate `.Find(&result)` call
- **Effort:** S (5 min)
- **Acceptance Criteria:**
  - [ ] Single `.Find(&result)` call in provider
  - [ ] Query executes only once
  - [ ] Existing functionality preserved

#### Task 1.2: Create Builder Pattern for Fame Model
- **File:** `services/atlas-fame/atlas.com/fame/fame/builder.go` (new)
- **Dependencies:** None
- **Fix:** Implement fluent builder with validation
- **Effort:** M (30 min)
- **Acceptance Criteria:**
  - [ ] `Builder` struct with all model fields
  - [ ] `NewBuilder(tenantId, characterId, targetId, amount)` constructor with required fields
  - [ ] Fluent setter methods return `*Builder`
  - [ ] `Build()` method with invariant validation:
    - tenantId must not be nil UUID
    - characterId must be > 0
    - targetId must be > 0
    - amount must be -1 or 1
  - [ ] Returns `(Model, error)` on validation failure

#### Task 1.3: Update Administrator to Use Builder
- **File:** `services/atlas-fame/atlas.com/fame/fame/administrator.go`
- **Dependencies:** Task 1.2
- **Fix:** Replace direct entity construction with builder
- **Effort:** S (15 min)
- **Acceptance Criteria:**
  - [ ] `create()` uses `NewBuilder()` instead of direct Entity construction
  - [ ] Builder validation enforced before entity creation
  - [ ] Error propagation from builder to caller

---

### Phase 2: Model Improvements (P1)
**Goal:** Complete model interface and fix convention violations

#### Task 2.1: Add Missing Model Accessors
- **File:** `services/atlas-fame/atlas.com/fame/fame/model.go`
- **Issue:** Only `TargetId()` and `CreatedAt()` exposed
- **Fix:** Add accessors for remaining private fields
- **Effort:** S (10 min)
- **Acceptance Criteria:**
  - [ ] `TenantId() uuid.UUID` accessor added
  - [ ] `Id() uuid.UUID` accessor added
  - [ ] `CharacterId() uint32` accessor added
  - [ ] `Amount() int8` accessor added
  - [ ] All use value receivers (not pointer)

#### Task 2.2: Fix GetName() Receiver Type
- **File:** `services/atlas-fame/atlas.com/fame/character/rest.go`
- **Line:** 14
- **Issue:** `func (r *RestModel) GetName()` uses pointer receiver
- **Fix:** Change to value receiver per guidelines
- **Effort:** S (5 min)
- **Acceptance Criteria:**
  - [ ] `func (r RestModel) GetName() string` uses value receiver
  - [ ] Consistent with `GetID()` receiver convention

---

### Phase 3: Test Coverage (P1)
**Goal:** Add comprehensive unit tests for all layers

#### Task 3.1: Create Builder Tests
- **File:** `services/atlas-fame/atlas.com/fame/fame/builder_test.go` (new)
- **Dependencies:** Task 1.2
- **Effort:** M (30 min)
- **Acceptance Criteria:**
  - [ ] Table-driven tests for valid builder construction
  - [ ] Tests for each validation failure case:
    - nil tenantId
    - zero characterId
    - zero targetId
    - invalid amount values
  - [ ] Tests for fluent setter chaining

#### Task 3.2: Create Provider Tests
- **File:** `services/atlas-fame/atlas.com/fame/fame/provider_test.go` (new)
- **Dependencies:** Task 1.1
- **Effort:** M (45 min)
- **Acceptance Criteria:**
  - [ ] Mock database setup with test entities
  - [ ] Test `byCharacterIdLastMonthEntityProvider` returns correct results
  - [ ] Test tenant isolation (only returns matching tenant)
  - [ ] Test date filtering (only returns last month)
  - [ ] Test empty result handling

#### Task 3.3: Create Processor Tests
- **File:** `services/atlas-fame/atlas.com/fame/fame/processor_test.go` (new)
- **Dependencies:** Task 3.1, Task 3.2
- **Effort:** L (1-2 hrs)
- **Acceptance Criteria:**
  - [ ] Mock dependencies (character processor, database)
  - [ ] Test `GetByCharacterIdLastMonth` success path
  - [ ] Test `RequestChange` validation rules:
    - Character not found -> error status
    - Target not found -> invalid name status
    - Level < 15 -> not minimum level status
    - Already famed today -> not today status
    - Already famed target this month -> not this month status
    - Success path -> creates log and forwards request
  - [ ] Test message buffer emissions

#### Task 3.4: Create Model Tests
- **File:** `services/atlas-fame/atlas.com/fame/fame/model_test.go` (new)
- **Dependencies:** Task 2.1
- **Effort:** S (15 min)
- **Acceptance Criteria:**
  - [ ] Test all accessor methods return correct values
  - [ ] Test immutability (model cannot be modified after creation)

---

### Phase 4: Legacy Cleanup (P2)
**Goal:** Remove technical debt once callers are migrated

#### Task 4.1: Migrate Consumer to Processor Interface
- **File:** `services/atlas-fame/atlas.com/fame/kafka/consumer/fame/consumer.go`
- **Line:** 40
- **Issue:** Uses legacy `fame.RequestChange(l)(ctx)(db)(...)` function
- **Fix:** Use `fame.NewProcessor(l, ctx, db).RequestChangeAndEmit(...)`
- **Effort:** S (15 min)
- **Acceptance Criteria:**
  - [ ] Consumer uses `NewProcessor()` pattern
  - [ ] Transaction ID properly generated
  - [ ] All parameters correctly passed

#### Task 4.2: Remove Legacy Functions
- **Files:**
  - `services/atlas-fame/atlas.com/fame/fame/processor.go` (lines 127-159)
  - `services/atlas-fame/atlas.com/fame/fame/producer.go` (lines 28-31)
  - `services/atlas-fame/atlas.com/fame/kafka/consumer/fame/kafka.go`
- **Dependencies:** Task 4.1
- **Effort:** S (10 min)
- **Acceptance Criteria:**
  - [ ] Remove `byCharacterIdLastMonthProvider` (legacy)
  - [ ] Remove `GetByCharacterIdLastMonth` (legacy)
  - [ ] Remove `RequestChange` (legacy curried function)
  - [ ] Remove `errorEventStatusProviderLegacy`
  - [ ] Remove empty `kafka.go` file
  - [ ] No compilation errors after removal

#### Task 4.3: Evaluate REST Infrastructure Cleanup
- **Files:**
  - `services/atlas-fame/atlas.com/fame/rest/handler.go`
  - `services/atlas-fame/atlas.com/fame/rest/request.go`
- **Issue:** REST handler infrastructure exists but service has no REST endpoints
- **Decision Required:** Keep for cross-service client consistency or remove?
- **Effort:** S (if removing) or N/A (if keeping)
- **Acceptance Criteria:**
  - [ ] Document decision in code comments
  - [ ] If removing: ensure no cross-service client functionality is broken

---

## Risk Assessment and Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Duplicate Find() fix breaks query logic | High | Low | Unit test confirms single execution |
| Builder validation too strict | Medium | Low | Review business rules before implementation |
| Legacy function removal breaks callers | High | Medium | Verify no external callers before removal |
| Test mocking complexity | Low | Medium | Follow existing test patterns in codebase |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Audit Pass Rate | 78% (14/18) | 100% (18/18) |
| Blocking Issues | 2 | 0 |
| Test Coverage | 0% | >80% |
| Legacy Functions | 4 | 0 |

---

## Required Resources and Dependencies

### Internal Dependencies
- Atlas backend guidelines (`.claude/skills/backend-dev-guidelines/SKILL.md`)
- Existing builder patterns (reference: `atlas-buddies/buddy/builder.go`)
- Testing utilities from other services

### External Dependencies
- `github.com/Chronicle20/atlas-model/model`
- `github.com/Chronicle20/atlas-tenant`
- `github.com/google/uuid`
- `gorm.io/gorm`

### Testing Dependencies
- Test database setup utilities
- Mock framework compatible with existing tests

---

## Task Dependency Graph

```
Phase 1 (P0):
  Task 1.1 (Fix Provider Bug) ─────────────────────┐
  Task 1.2 (Create Builder) ───> Task 1.3 (Update Admin)
                                                    │
Phase 2 (P1):                                       │
  Task 2.1 (Model Accessors) ──────────────────────┤
  Task 2.2 (Fix GetName Receiver) ─────────────────┤
                                                    │
Phase 3 (P1):                                       │
  Task 3.1 (Builder Tests) ─────────────────────────┤
  Task 3.2 (Provider Tests) ────────────────────────┼──> Task 3.3 (Processor Tests)
  Task 3.4 (Model Tests) ──────────────────────────┤
                                                    │
Phase 4 (P2):                                       │
  Task 4.1 (Migrate Consumer) ───> Task 4.2 (Remove Legacy)
  Task 4.3 (REST Infrastructure Decision) ─────────┘
```

---

## Appendix: Relevant File Paths

| Purpose | Path |
|---------|------|
| Provider (bug) | `services/atlas-fame/atlas.com/fame/fame/provider.go` |
| Model | `services/atlas-fame/atlas.com/fame/fame/model.go` |
| Administrator | `services/atlas-fame/atlas.com/fame/fame/administrator.go` |
| Processor | `services/atlas-fame/atlas.com/fame/fame/processor.go` |
| Entity | `services/atlas-fame/atlas.com/fame/fame/entity.go` |
| Character REST | `services/atlas-fame/atlas.com/fame/character/rest.go` |
| Consumer | `services/atlas-fame/atlas.com/fame/kafka/consumer/fame/consumer.go` |
| Producer | `services/atlas-fame/atlas.com/fame/fame/producer.go` |
| Builder Reference | `services/atlas-buddies/atlas.com/buddies/buddy/builder.go` |
