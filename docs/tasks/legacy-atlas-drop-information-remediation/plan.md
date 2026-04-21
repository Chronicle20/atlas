# Atlas Drop Information Service Remediation Plan

**Service:** `atlas-drop-information`
**Path:** `services/atlas-drop-information`
**Audit Source:** `docs/audits/atlas-drop-information/audit.md`
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

This remediation plan addresses 4 non-blocking issues identified in the backend audit of the `atlas-drop-information` service. The service is functionally sound with good architectural compliance, but requires improvements in test coverage, naming conventions, and code cleanup.

### Issues to Address

| Issue ID | Description | Priority | Effort |
|----------|-------------|----------|--------|
| ARCH-012 | Missing processor and provider tests with mocks | P1 | M |
| ARCH-005 | Provider transformation function naming inconsistency | P2 | S |
| ARCH-002 | Builder `Build()` methods don't return errors | P2 | S |
| ARCH-013 | Legacy function wrappers need evaluation/removal | P2 | S |

### Scope

- **In Scope:** Test infrastructure, naming conventions, builder validation, legacy code removal
- **Out of Scope:** Functional changes, API modifications, new features

---

## 2. Current State Analysis

### Service Architecture

The service follows the Atlas backend guidelines with proper layer separation:

```
atlas-drop-information/
├── atlas.com/dis/
│   ├── monster/drop/       # Monster drop domain
│   │   ├── model.go        # Immutable domain model
│   │   ├── entity.go       # GORM entity
│   │   ├── builder.go      # Fluent builder (no validation)
│   │   ├── processor.go    # Business logic + legacy wrappers
│   │   ├── provider.go     # Data access (uses makeDrop)
│   │   └── ...
│   ├── continent/drop/     # Continent drop domain
│   │   ├── model.go
│   │   ├── entity.go
│   │   ├── builder.go      # Fluent builder (no validation)
│   │   ├── processor.go    # Business logic + legacy wrappers
│   │   ├── provider.go     # Data access (uses makeDrop)
│   │   └── ...
│   └── continent/          # Aggregate view
│       └── processor.go    # Legacy wrappers present
```

### Existing Test Coverage

| File | Tests Present | Coverage |
|------|---------------|----------|
| `monster/drop/builder_test.go` | Yes | Builder fluent API |
| `monster/drop/seed_test.go` | Yes | Seed file loading |
| `continent/drop/builder_test.go` | Yes | Builder fluent API |
| `continent/drop/seed_test.go` | Yes | Seed file loading |
| `seed/processor_test.go` | Yes | JSON serialization only |
| **Processor tests** | **No** | **Missing** |
| **Provider tests** | **No** | **Missing** |

### Issues Detail

1. **ARCH-012 (Test Coverage):** No mock implementations exist for Processor interfaces. Processor and provider logic is untested.

2. **ARCH-005 (Naming):** The `makeDrop` function in `provider.go` files should be named `modelFromEntity` per guidelines.

3. **ARCH-002 (Builder Validation):** `Build()` methods return `Model` directly without `(Model, error)` signature for validation.

4. **ARCH-013 (Legacy Wrappers):** Three processor files contain legacy function wrappers that appear unused.

---

## 3. Proposed Future State

### Target Architecture

```
atlas-drop-information/
├── atlas.com/dis/
│   ├── monster/drop/
│   │   ├── mock/
│   │   │   └── processor.go       # NEW: Mock implementation
│   │   ├── processor_test.go      # NEW: Processor tests
│   │   ├── builder.go             # MODIFIED: Build() returns (Model, error)
│   │   ├── provider.go            # MODIFIED: Rename makeDrop -> modelFromEntity
│   │   └── processor.go           # MODIFIED: Remove legacy wrappers
│   ├── continent/drop/
│   │   ├── mock/
│   │   │   └── processor.go       # NEW: Mock implementation
│   │   ├── processor_test.go      # NEW: Processor tests
│   │   ├── builder.go             # MODIFIED: Build() returns (Model, error)
│   │   ├── provider.go            # MODIFIED: Rename makeDrop -> modelFromEntity
│   │   └── processor.go           # MODIFIED: Remove legacy wrappers
│   └── continent/
│       └── processor.go           # MODIFIED: Remove legacy wrappers
```

### Success Criteria

- [ ] All processors have corresponding mock implementations
- [ ] Processor tests cover GetAll and GetForMonster methods with table-driven tests
- [ ] `modelFromEntity` naming convention followed in all provider files
- [ ] Builder `Build()` methods return `(Model, error)` with basic validation
- [ ] Legacy function wrappers removed from all processor files
- [ ] All existing tests continue to pass
- [ ] New tests achieve >80% coverage for processor methods

---

## 4. Implementation Phases

### Phase 1: Test Infrastructure (Priority: P1)

**Objective:** Add mock implementations and processor tests for both drop domains.

#### Section 1.1: Monster Drop Mocks and Tests

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 1.1.1 | Create `monster/drop/mock/` directory | Directory exists | S |
| 1.1.2 | Implement `ProcessorMock` for monster/drop | Mock struct with configurable function fields for `GetAll` and `GetForMonster` methods | S |
| 1.1.3 | Create `monster/drop/processor_test.go` | Table-driven tests for `GetAll` method covering success and error cases | M |
| 1.1.4 | Add `GetForMonster` tests | Table-driven tests covering valid monsterId, invalid monsterId, and error cases | M |

#### Section 1.2: Continent Drop Mocks and Tests

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 1.2.1 | Create `continent/drop/mock/` directory | Directory exists | S |
| 1.2.2 | Implement `ProcessorMock` for continent/drop | Mock struct with configurable function field for `GetAll` method | S |
| 1.2.3 | Create `continent/drop/processor_test.go` | Table-driven tests for `GetAll` method covering success and error cases | M |

### Phase 2: Naming Convention Compliance (Priority: P2)

**Objective:** Rename transformation functions to follow guidelines.

#### Section 2.1: Provider Function Renaming

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 2.1.1 | Rename `makeDrop` in `monster/drop/provider.go` | Function renamed to `modelFromEntity`, all references updated | S |
| 2.1.2 | Rename `makeDrop` in `continent/drop/provider.go` | Function renamed to `modelFromEntity`, all references updated | S |
| 2.1.3 | Verify no external references | Grep confirms no external usage of old function names | S |

### Phase 3: Builder Validation (Priority: P2)

**Objective:** Add validation to builder `Build()` methods.

#### Section 3.1: Monster Drop Builder

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 3.1.1 | Update `Build()` signature in `monster/drop/builder.go` | Returns `(Model, error)` instead of `Model` | S |
| 3.1.2 | Add basic validation | Validate tenantId is not nil UUID, return error if invalid | S |
| 3.1.3 | Update all callers | Provider `modelFromEntity` handles error return | S |
| 3.1.4 | Update builder tests | Verify validation behavior and error returns | S |

#### Section 3.2: Continent Drop Builder

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 3.2.1 | Update `Build()` signature in `continent/drop/builder.go` | Returns `(Model, error)` instead of `Model` | S |
| 3.2.2 | Add basic validation | Validate tenantId is not nil UUID, return error if invalid | S |
| 3.2.3 | Update all callers | Provider `modelFromEntity` handles error return | S |
| 3.2.4 | Update builder tests | Verify validation behavior and error returns | S |

### Phase 4: Legacy Code Removal (Priority: P2)

**Objective:** Remove unused legacy function wrappers.

#### Section 4.1: Verify No Usage

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 4.1.1 | Search for legacy `GetAll` usage | Confirm `drop.GetAll(l)` pattern is unused across codebase | S |
| 4.1.2 | Search for legacy `GetForMonster` usage | Confirm `drop.GetForMonster(l)` pattern is unused | S |
| 4.1.3 | Search for legacy `continent.GetAll` usage | Confirm `continent.GetAll(l)` pattern is unused | S |

#### Section 4.2: Remove Legacy Wrappers

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 4.2.1 | Remove legacy wrappers from `monster/drop/processor.go` | Lines 41-58 removed (GetAll, GetForMonster wrappers) | S |
| 4.2.2 | Remove legacy wrappers from `continent/drop/processor.go` | Lines 36-45 removed (GetAll wrapper) | S |
| 4.2.3 | Remove legacy wrappers from `continent/processor.go` | Lines 58-67 removed (GetAll wrapper) | S |

### Phase 5: Validation and Cleanup

**Objective:** Ensure all changes are working and tests pass.

| Task | Description | Acceptance Criteria | Effort |
|------|-------------|---------------------|--------|
| 5.1 | Run all existing tests | `go test ./...` passes in service directory | S |
| 5.2 | Run new processor tests | All new tests pass | S |
| 5.3 | Build service | `go build ./...` succeeds with no errors | S |
| 5.4 | Update audit status | Mark audit as resolved in `audit.json` | S |

---

## 5. Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Legacy wrappers are used elsewhere | Low | Medium | Codebase search confirmed no external usage |
| Builder validation breaks existing code | Low | Medium | Validation is minimal (nil UUID check only) |
| Mock pattern doesn't match existing conventions | Low | Low | Follow established mock pattern from other services |
| Test infrastructure incomplete | Medium | Low | Follow table-driven test patterns from other services |

---

## 6. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test coverage for processors | >80% | `go test -cover` |
| New test files created | 4 | File count |
| Mock implementations | 2 | File count |
| Naming convention compliance | 100% | Grep for `makeDrop` returns 0 |
| Legacy wrappers remaining | 0 | Grep for "Legacy function wrapper" |
| Build success | Pass | `go build ./...` |
| Existing tests pass | Pass | `go test ./...` |

---

## 7. Required Resources and Dependencies

### Dependencies

- No external dependencies required
- Mock pattern follows existing conventions in:
  - `atlas-saga-orchestrator/character/mock/processor.go`
  - `atlas-query-aggregator/validation/mock/processor.go`

### Reference Files

| Purpose | File Path |
|---------|-----------|
| Mock pattern example | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go` |
| Test pattern example | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor_test.go` |
| Builder test example | `services/atlas-drop-information/atlas.com/dis/monster/drop/builder_test.go` |

---

## 8. Implementation Notes

### Mock Implementation Pattern

```go
package mock

import "github.com/Chronicle20/atlas-model/model"

type ProcessorMock struct {
    GetAllFunc       func() model.Provider[[]Model]
    GetForMonsterFunc func(monsterId uint32) model.Provider[[]Model]
}

func (m *ProcessorMock) GetAll() model.Provider[[]Model] {
    if m.GetAllFunc != nil {
        return m.GetAllFunc()
    }
    return model.FixedProvider[[]Model](nil)
}
```

### Builder Validation Pattern

```go
func (b *Builder) Build() (Model, error) {
    if b.tenantId == uuid.Nil {
        return Model{}, errors.New("tenantId cannot be nil")
    }
    return Model{
        tenantId: b.tenantId,
        // ... other fields
    }, nil
}
```

### Model From Entity Pattern

```go
// Rename from makeDrop to modelFromEntity
func modelFromEntity(m entity) (Model, error) {
    return NewMonsterDropBuilder(m.TenantId, m.ID).
        SetMonsterId(m.MonsterId).
        // ... other fields
        Build()
}
```
