# Remediation Plan - atlas-equipables Service

**Last Updated:** 2026-01-13
**Source Audit:** `docs/audits/atlas-equipables/audit.md`
**Overall Status:** needs-work → compliant

---

## 1. Executive Summary

This remediation plan addresses 8 issues identified in the `atlas-equipables` service audit. The service is a moderately well-structured microservice that manages equipment items with CRUD operations via REST API and Kafka messaging. While the core architecture is sound (immutable models, multi-tenancy, AndEmit pattern), critical gaps exist in testing infrastructure and documentation accuracy.

**Priority Distribution:**
- P0 (Blocking): 2 issues - Documentation accuracy, Zero test coverage
- P1 (High): 2 issues - Typed domain errors, Mock infrastructure
- P2 (Low): 2 issues - Builder separation, Transform accessor usage

**Total Estimated Effort:** M-L (Medium to Large)

---

## 2. Current State Analysis

### Service Health
| Category | Status | Issues |
|----------|--------|--------|
| Architecture | Pass | 8/8 checks passing |
| Kafka | Pass | 3/3 checks passing |
| REST | Mixed | 3/4 passing, 1 warning |
| Documentation | Fail | 2/2 checks failing |
| Testing | Fail | 2/2 checks failing |
| Infrastructure | Pass | 1/1 checks passing |

### Root Causes
1. **Documentation Drift:** API paths were likely updated during development without corresponding README updates
2. **Test Debt:** Service was developed without TDD or test coverage requirements
3. **Style Inconsistencies:** Direct field access in Transform likely predates accessor pattern adoption

---

## 3. Proposed Future State

Upon completion, `atlas-equipables` will have:

1. **Accurate Documentation:** README reflects actual `/api/equipables` paths with complete endpoint coverage
2. **Comprehensive Tests:** Unit tests covering processor, builder, provider, and REST transformation logic
3. **Type-Safe Errors:** Domain-specific error types enabling proper HTTP status code mapping
4. **Testable Architecture:** Mock infrastructure allowing isolated unit testing
5. **Consistent Patterns:** Transform function using accessor methods, matching immutability design

---

## 4. Implementation Phases

### Phase 1: Documentation Fixes (P0)
**Goal:** Eliminate developer confusion from incorrect API documentation

| Task | Check ID | Effort | Dependencies |
|------|----------|--------|--------------|
| 1.1 Update README API paths | DOC-001 | S | None |
| 1.2 Add PATCH endpoint documentation | DOC-002 | S | None |

### Phase 2: Testing Infrastructure (P0/P1)
**Goal:** Establish test coverage and enable future regression protection

| Task | Check ID | Effort | Dependencies |
|------|----------|--------|--------------|
| 2.1 Add processor unit tests | TEST-001 | M | None |
| 2.2 Add builder tests | TEST-001 | S | None |
| 2.3 Add REST transformation tests | TEST-001 | S | None |
| 2.4 Create mock infrastructure | TEST-002 | M | 2.1 |

### Phase 3: Error Handling Improvements (P1)
**Goal:** Enable proper HTTP status code responses for different error conditions

| Task | Check ID | Effort | Dependencies |
|------|----------|--------|--------------|
| 3.1 Define typed domain errors | REST-004 | S | None |
| 3.2 Update handlers with error mapping | REST-004 | S | 3.1 |

### Phase 4: Code Quality (P2)
**Goal:** Align with project style guidelines

| Task | Check ID | Effort | Dependencies |
|------|----------|--------|--------------|
| 4.1 Refactor Transform to use accessors | REST-002 | S | None |
| 4.2 Separate builder to builder.go | ARCH-003 | S | 2.2 (tests first) |

---

## 5. Detailed Tasks

### Task 1.1: Update README API Paths
**Check ID:** DOC-001
**File:** `services/atlas-equipables/README.md`
**Effort:** S

**Current State:**
```
/api/ess/equipment
/api/ess/equipment/{equipmentId}
```

**Target State:**
```
/api/equipables
/api/equipables/{equipmentId}
```

**Acceptance Criteria:**
- [ ] All API path references use `/api/equipables` prefix
- [ ] Path matches `resource.go:20` router configuration
- [ ] Examples use correct path format

---

### Task 1.2: Add PATCH Endpoint Documentation
**Check ID:** DOC-002
**File:** `services/atlas-equipables/README.md`
**Effort:** S

**Target State:**
Add section for PATCH endpoint:
```markdown
#### [PATCH] Update Equipable By Id

`/api/equipables/{equipmentId}`

Updates an existing equipable's attributes.
```

**Acceptance Criteria:**
- [ ] PATCH endpoint documented with path
- [ ] Request body structure described
- [ ] Matches `handleUpdateEquipment` in `resource.go:24`

---

### Task 2.1: Add Processor Unit Tests
**Check ID:** TEST-001
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/processor_test.go` (new)
**Effort:** M

**Test Cases Required:**
1. `TestCreateSunny` - Create equipable with provided stats
2. `TestCreateWithTemplateStats` - Create equipable with zero stats (fetches from template)
3. `TestCreateRandomSunny` - Create equipable with randomized stats
4. `TestGetByIdSunny` - Retrieve existing equipable
5. `TestGetByIdNotFound` - Retrieve non-existent equipable returns error
6. `TestUpdateSunny` - Update equipable attributes
7. `TestUpdatePartialFields` - Update preserves unchanged fields
8. `TestDeleteByIdSunny` - Delete existing equipable
9. `TestDeleteByIdNotFound` - Delete non-existent equipable returns error

**Pattern Reference:** `atlas-character/character/processor_test.go`

**Test Infrastructure:**
```go
func testDatabase(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    // Run migrations
    return db
}

func testTenant() tenant.Model {
    t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
    return t
}

func testLogger() logrus.FieldLogger {
    l, _ := test.NewNullLogger()
    return l
}
```

**Acceptance Criteria:**
- [ ] All processor methods have at least sunny-day and error-case tests
- [ ] Tests use in-memory SQLite database
- [ ] Tests properly initialize tenant context
- [ ] `go test ./...` passes in service directory

---

### Task 2.2: Add Builder Tests
**Check ID:** TEST-001
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/builder_test.go` (new)
**Effort:** S

**Test Cases Required:**
1. `TestNewBuilderSetsId` - NewBuilder initializes with ID
2. `TestBuilderFluentMethods` - Setter methods return builder (fluent)
3. `TestBuilderBuild` - Build creates immutable model
4. `TestCloneCreatesBuilderFromModel` - Clone preserves all fields
5. `TestAddStatClampsAtZero` - AddStrength(-100) on 50 results in 0
6. `TestAddStatClampsAtMax` - AddStrength(70000) clamps to uint16 max

**Acceptance Criteria:**
- [ ] Builder fluent interface tested
- [ ] Clone function tested for field preservation
- [ ] Arithmetic operations tested for boundary conditions

---

### Task 2.3: Add REST Transformation Tests
**Check ID:** TEST-001
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/rest_test.go` (new)
**Effort:** S

**Test Cases Required:**
1. `TestTransformSunny` - Transform maps all fields correctly
2. `TestExtractSunny` - Extract creates model from REST input
3. `TestTransformExtractRoundTrip` - Transform then Extract preserves data
4. `TestGetNameReturnsEquipables` - JSON:API type name correct
5. `TestGetIDFormatsAsString` - ID conversion to string
6. `TestSetIDParsesString` - String parsing to ID

**Pattern Reference:** `atlas-character/character/rest_test.go`

**Acceptance Criteria:**
- [ ] Transform function tested for all 28 fields
- [ ] Extract function tested for all 28 fields
- [ ] JSON:API interface methods tested

---

### Task 2.4: Create Mock Infrastructure
**Check ID:** TEST-002
**Files:**
- `services/atlas-equipables/atlas.com/equipables/equipable/mock/processor.go` (new)
- `services/atlas-equipables/atlas.com/equipables/data/equipable/mock/processor.go` (new)
**Effort:** M

**Components Required:**

1. **Internal Processor Mock:**
```go
package mock

type ProcessorMock struct {
    GetByIdFunc             func(id uint32) (equipable.Model, error)
    CreateAndEmitFunc       func(i equipable.Model) (equipable.Model, error)
    CreateRandomAndEmitFunc func(id uint32) (equipable.Model, error)
    UpdateAndEmitFunc       func(i equipable.Model) (equipable.Model, error)
    DeleteByIdAndEmitFunc   func(id uint32) error
}
```

2. **External Data Processor Mock:**
```go
package mock

type DataProcessorMock struct {
    GetByIdFunc func(itemId uint32) (equipable.Model, error)
}
```

**Acceptance Criteria:**
- [ ] Mock implementations allow injection in tests
- [ ] Mocks can simulate both success and failure cases
- [ ] Pattern matches existing mocks in codebase

---

### Task 3.1: Define Typed Domain Errors
**Check ID:** REST-004
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/errors.go` (new)
**Effort:** S

**Target Implementation:**
```go
package equipable

import "errors"

var (
    ErrEquipableNotFound = errors.New("equipable not found")
    ErrInvalidItemId     = errors.New("invalid item ID")
    ErrTemplateNotFound  = errors.New("equipable template not found")
    ErrCreateFailed      = errors.New("failed to create equipable")
    ErrUpdateFailed      = errors.New("failed to update equipable")
    ErrDeleteFailed      = errors.New("failed to delete equipable")
)
```

**Acceptance Criteria:**
- [ ] Sentinel errors defined for distinct failure modes
- [ ] Errors support `errors.Is()` comparison
- [ ] Error messages are descriptive but not leaking internals

---

### Task 3.2: Update Handlers with Error Mapping
**Check ID:** REST-004
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/resource.go`
**Effort:** S

**Current Pattern (line 82-85):**
```go
if err != nil {
    d.Logger().WithError(err).Errorf("Unable to retrieve equipable %d.", equipmentId)
    w.WriteHeader(http.StatusNotFound)
    return
}
```

**Target Pattern:**
```go
if err != nil {
    d.Logger().WithError(err).Errorf("Unable to retrieve equipable %d.", equipmentId)
    if errors.Is(err, ErrEquipableNotFound) {
        w.WriteHeader(http.StatusNotFound)
    } else {
        w.WriteHeader(http.StatusInternalServerError)
    }
    return
}
```

**Handlers to Update:**
- `handleGetEquipment` - Map not-found vs internal errors
- `handleUpdateEquipment` - Map not-found vs validation vs internal errors
- `handleDeleteEquipment` - Map not-found vs internal errors

**Acceptance Criteria:**
- [ ] 404 returned only for "not found" errors
- [ ] 500 returned for unexpected/internal errors
- [ ] Error logging preserved for debugging

---

### Task 4.1: Refactor Transform to Use Accessors
**Check ID:** REST-002
**File:** `services/atlas-equipables/atlas.com/equipables/equipable/rest.go`
**Effort:** S

**Current Implementation (lines 57-89):**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:             m.id,      // Direct field access
        ItemId:         m.itemId,  // Direct field access
        // ... 26 more direct accesses
    }, nil
}
```

**Target Implementation:**
```go
func Transform(m Model) (RestModel, error) {
    return RestModel{
        Id:             m.Id(),      // Accessor method
        ItemId:         m.ItemId(),  // Accessor method
        // ... 26 more accessor calls
    }, nil
}
```

**Acceptance Criteria:**
- [ ] All 28 fields use accessor methods
- [ ] Tests pass after refactoring (Task 2.3 must complete first)
- [ ] No direct private field access in Transform

---

### Task 4.2: Separate Builder to builder.go
**Check ID:** ARCH-003
**Files:**
- `services/atlas-equipables/atlas.com/equipables/equipable/model.go` (modify)
- `services/atlas-equipables/atlas.com/equipables/equipable/builder.go` (new)
**Effort:** S

**Components to Move:**
- `Clone` function (lines 153-185)
- `ModelBuilder` struct (lines 187-217)
- `NewBuilder` function (line 219)
- All setter methods (lines 223-361)
- `Build` method (lines 458-490)
- Helper functions: `addUint16`, `addUint32`, `addByte` (lines 492-523)

**Acceptance Criteria:**
- [ ] `builder.go` contains all builder-related code
- [ ] `model.go` contains only Model struct and accessors
- [ ] Builder tests pass after separation (Task 2.2 must complete first)
- [ ] No import cycle introduced

---

## 6. Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Test infrastructure differs from codebase patterns | Low | Medium | Reference existing tests in atlas-character, atlas-marriages |
| Mock patterns incompatible with processor design | Medium | Low | Verify processor uses interfaces; adapt mock approach if needed |
| Error type changes break Kafka consumers | Low | Medium | Ensure error wrapping preserves original error for logging |
| Builder separation introduces import cycles | Low | Low | Keep helpers in builder.go; no cross-package imports |

---

## 7. Success Metrics

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Test file count | 0 | 4+ | `ls *_test.go \| wc -l` |
| Test coverage | 0% | >70% | `go test -cover` |
| Documentation accuracy | 60% | 100% | Manual review vs resource.go |
| Audit check pass rate | 14/20 | 20/20 | Re-run audit script |

---

## 8. Required Resources and Dependencies

### Internal Dependencies
- `atlas-character/character/*_test.go` - Testing patterns reference
- `atlas-marriages/marriages/*_test.go` - Additional testing patterns
- `github.com/Chronicle20/atlas-tenant` - Tenant context for tests

### External Dependencies
- `gorm.io/driver/sqlite` - In-memory database for tests
- `github.com/sirupsen/logrus/hooks/test` - Null logger for tests
- `github.com/google/uuid` - UUID generation for test data

### Build Verification
```bash
cd services/atlas-equipables
go test ./...
go build ./...
```

---

## 9. Implementation Order

```
Phase 1 (Documentation)
    ├── 1.1 Update README paths
    └── 1.2 Add PATCH documentation

Phase 2 (Testing) - Can start in parallel with Phase 1
    ├── 2.1 Processor tests
    ├── 2.2 Builder tests
    ├── 2.3 REST tests
    └── 2.4 Mock infrastructure (depends on 2.1)

Phase 3 (Error Handling) - After Phase 2
    ├── 3.1 Define errors
    └── 3.2 Update handlers (depends on 3.1)

Phase 4 (Code Quality) - After relevant tests exist
    ├── 4.1 Transform refactor (after 2.3)
    └── 4.2 Builder separation (after 2.2)
```

---

## 10. Post-Implementation Verification

1. **Re-run Audit:** Execute audit script against modified service
2. **Test Coverage:** Run `go test -coverprofile=coverage.out ./...`
3. **Build Verification:** Ensure `go build ./...` succeeds
4. **Integration Test:** Deploy to dev environment and test API endpoints
5. **Documentation Review:** Verify README matches live API behavior
