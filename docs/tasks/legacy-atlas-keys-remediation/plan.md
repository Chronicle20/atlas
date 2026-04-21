# Atlas-Keys Service Remediation Plan

**Last Updated:** 2026-01-13

---

## Executive Summary

The `atlas-keys` service audit identified 5 issues requiring remediation across testing infrastructure (2 blocking, high-impact) and architectural patterns (3 non-blocking, low-medium impact). This plan addresses all findings through 4 implementation phases, prioritizing the establishment of test infrastructure before architectural improvements.

**Audit Reference:** `dev/audits/atlas-keys/audit.md`

**Key Metrics:**
- Issues to Resolve: 5 (2 FAIL/High, 1 FAIL/Medium, 1 FAIL/Medium, 1 WARN/Low)
- Estimated Total Effort: L (Large)
- Risk Level: Low (non-breaking changes, isolated service)

---

## Current State Analysis

### Service Overview
The `atlas-keys` service manages keyboard bindings for game characters. It's a small, focused microservice with:
- 14 packages
- 0% test coverage
- Well-structured layer separation
- Correct multi-tenancy and Kafka integration

### Issues Summary

| ID | Status | Impact | Description |
|----|--------|--------|-------------|
| TEST-001 | FAIL | High | No test files exist in any package |
| TEST-002 | FAIL | High | No mock implementations for Processor interface |
| ARCH-003 | FAIL | Medium | Missing builder.go for model construction |
| ARCH-006 | FAIL | Medium | Entity transformation functions in wrong file |
| ARCH-002 | WARN | Low | Missing CharacterId() accessor on Model |

### What's Working Well
- Layer separation (ARCH-001: PASS)
- Processor interface definition (ARCH-004: PASS)
- Administrator/Provider split (ARCH-005: PASS)
- JSON:API compliance (REST-001: PASS)
- Handler registration (REST-002: PASS)
- Kafka header parsing (KAFKA-001, KAFKA-002, KAFKA-003: PASS)
- Multi-tenancy context (TENANT-001: PASS)
- Service documentation (DOC-001: PASS)

---

## Proposed Future State

After remediation, the service will have:

1. **Complete test infrastructure** with mock implementations enabling isolated unit tests
2. **Builder pattern** for validated model construction
3. **Proper file organization** with transformation functions in entity.go
4. **Complete model API** with all field accessors

### Target Directory Structure
```
services/atlas-keys/atlas.com/keys/
└── key/
    ├── model.go           # + CharacterId() accessor
    ├── builder.go         # NEW: Fluent builder with validation
    ├── builder_test.go    # NEW: Builder invariant tests
    ├── entity.go          # + Make(), ToEntity() functions
    ├── processor.go       # - makeKey (moved to entity.go)
    ├── processor_test.go  # NEW: Processor logic tests
    ├── rest.go            # + TransformSlice() function
    └── mock/
        └── processor.go   # NEW: Mock implementation
```

---

## Implementation Phases

### Phase 1: Model API Completion (P2 - Quick Win)
**Objective:** Complete the Model's public API by adding the missing accessor.

**Rationale:** Small change that completes the model interface before larger refactoring.

**Tasks:**
1. Add `CharacterId()` accessor to `key/model.go`

**Effort:** S (Small)

---

### Phase 2: Mock Infrastructure (P0 - Blocking)
**Objective:** Create mock implementations to enable isolated testing.

**Rationale:** Mocks are a prerequisite for writing meaningful unit tests. This unblocks Phase 4.

**Tasks:**
1. Create `key/mock/` directory
2. Create `key/mock/processor.go` implementing the Processor interface
3. Each interface method should have a configurable function field

**Reference Implementation:** `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`

**Effort:** M (Medium)

---

### Phase 3: Architectural Alignment (P1-P2)
**Objective:** Align codebase with backend development guidelines.

**Rationale:** Proper file organization and builder pattern improve maintainability and enforce validation.

**Tasks:**

#### 3.1 Entity Transformation Functions (ARCH-006)
1. Move `makeKey` function from `processor.go` to `entity.go`
2. Rename to `Make(entity) (Model, error)` for consistency
3. Add `ToEntity()` method on Model for reverse transformation
4. Update `processor.go` imports and references

#### 3.2 Builder Pattern (ARCH-003)
1. Create `key/builder.go` with `ModelBuilder` struct
2. Implement fluent setters: `SetCharacterId()`, `SetKey()`, `SetType()`, `SetAction()`
3. Implement `Build()` with validation:
   - CharacterId must be > 0
   - Key must be valid (consider valid range if applicable)
   - Type must be valid (values 4-6 based on defaults)
   - Action must be >= 0
4. Implement `MustBuild()` for trusted sources
5. Add `CloneModelBuilder(m Model)` for cloning
6. Add accessor methods on builder for read access

**Reference Implementation:** `services/atlas-expressions/atlas.com/expressions/expression/builder.go`

#### 3.3 REST Transform Enhancement (Optional)
1. Add `TransformSlice([]Model) ([]RestModel, error)` to `rest.go`

**Effort:** M (Medium)

---

### Phase 4: Test Coverage (P0 - Blocking)
**Objective:** Establish comprehensive test coverage for core business logic.

**Rationale:** Tests validate correctness and prevent regressions.

**Tasks:**

#### 4.1 Builder Tests
Create `key/builder_test.go`:
1. Test successful build with valid inputs
2. Test validation failures (zero characterId, invalid type, etc.)
3. Test CloneModelBuilder preserves all fields
4. Test MustBuild panics on invalid input

#### 4.2 Processor Tests
Create `key/processor_test.go`:
1. Test GetByCharacterId returns expected models
2. Test CreateDefault creates 40 default bindings
3. Test Reset removes existing and creates defaults
4. Test ChangeKey creates new binding when not exists
5. Test ChangeKey updates existing binding
6. Test Delete removes all bindings for character

*Note: These may require integration test setup with test database or additional mocking of database layer.*

#### 4.3 REST Handler Tests (Optional)
Create `character/resource_test.go`:
1. Test GET /characters/{id}/keys returns key map
2. Test PUT /characters/{id}/keys creates/updates binding
3. Test error handling for invalid characterId

**Effort:** L (Large)

---

## Risk Assessment and Mitigation

### Risk 1: Breaking Changes to Model Construction
**Likelihood:** Low
**Impact:** Medium
**Mitigation:** The builder is additive. Existing `makeKey` function continues to work until all call sites are migrated. Transformation functions maintain backward compatibility.

### Risk 2: Test Database Setup Complexity
**Likelihood:** Medium
**Impact:** Low
**Mitigation:** Start with mock-based unit tests. Integration tests can be added incrementally. Consider using testcontainers for database tests if needed.

### Risk 3: Validation Rules Too Strict
**Likelihood:** Low
**Impact:** Low
**Mitigation:** Review default key arrays in processor.go to understand valid value ranges before implementing validation. Start with minimal validation and expand based on actual constraints.

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Test Coverage | 0% | >80% for key package |
| FAIL Audit Checks | 4 | 0 |
| WARN Audit Checks | 1 | 0 |
| Mock Implementations | 0 | 1 (Processor) |
| Builder Validation Rules | 0 | 4+ |

---

## Required Resources and Dependencies

### Files to Modify
- `key/model.go` - Add accessor
- `key/entity.go` - Add transformation functions
- `key/processor.go` - Remove makeKey, update references
- `key/rest.go` - Add TransformSlice (optional)

### Files to Create
- `key/builder.go`
- `key/builder_test.go`
- `key/processor_test.go`
- `key/mock/processor.go`
- `character/resource_test.go` (optional)

### External Dependencies
- None - all changes are internal to the service

### Reference Materials
- Builder pattern: `services/atlas-expressions/atlas.com/expressions/expression/builder.go`
- Mock pattern: `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`
- Guidelines: `.claude/skills/backend-dev-guidelines/`

---

## Implementation Order

The recommended implementation order optimizes for unblocking dependencies:

```
Phase 1 (Model API)     ─────────────────┐
                                         │
Phase 2 (Mocks)         ─────────────────┼──> Phase 4 (Tests)
                                         │
Phase 3 (Architecture)  ─────────────────┘
```

**Suggested Sequence:**
1. Phase 1: CharacterId accessor (5 min) - Quick win, no dependencies
2. Phase 2: Mock infrastructure (30 min) - Enables testing
3. Phase 3.1: Entity transformation (15 min) - Prerequisite for builder
4. Phase 3.2: Builder pattern (45 min) - Major architectural improvement
5. Phase 4.1: Builder tests (30 min) - Validate builder implementation
6. Phase 4.2: Processor tests (1-2 hours) - Core business logic coverage
7. Phase 3.3 & 4.3: REST enhancements and tests (optional)
