# Atlas-Expressions Remediation Tasks

**Last Updated:** 2026-01-13

---

## Phase 1: Test Infrastructure Setup (P0) - COMPLETED

### Task 1.1: Add ResetForTesting to Registry
- [x] Add `ResetForTesting()` method to `expression/registry.go`
- [x] Method resets `expressionReg` map
- [x] Method resets `tenantLock` map
- [x] Method uses proper locking
- [x] Add comment indicating test-only usage
- [x] Added `getOrCreateTenantMaps()` helper for thread-safe access
- [x] Added `get()` method for testing

**File:** `services/atlas-expressions/atlas.com/expressions/expression/registry.go`
**Effort:** S

### Task 1.2: Create Processor Mock
- [x] Create directory `expression/mock/`
- [x] Create `expression/mock/processor.go`
- [x] Implement `ProcessorMock` struct with func fields for all methods
- [x] Implement all Processor interface methods
- [x] Each method checks if func is set, otherwise returns zero value

**File:** `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`
**Effort:** S

### Task 1.3: Create Test Helpers
- [x] Implement `setupTestTenant(t *testing.T)` helper (in model_test.go)
- [x] Implement `setupTestContext(t *testing.T, ten tenant.Model)` helper (in processor_test.go)
- [x] Implement `setupTestLogger(t *testing.T)` helper (in processor_test.go)

**File:** `services/atlas-expressions/atlas.com/expressions/expression/model_test.go`
**Effort:** S

---

## Phase 2: Comprehensive Test Coverage (P0) - COMPLETED

### Task 2.1: Model Tests
- [x] Create `expression/model_test.go`
- [x] Test `Expiration()` returns correct value
- [x] Test `CharacterId()` returns correct value
- [x] Test `MapId()` returns correct value
- [x] Test `Expression()` returns correct value
- [x] Test `Tenant()` returns correct value
- [x] Test `WorldId()` returns correct value
- [x] Test `ChannelId()` returns correct value
- [x] Verify fields are not directly accessible (private)

**File:** `services/atlas-expressions/atlas.com/expressions/expression/model_test.go`
**Effort:** S

### Task 2.2: Registry Tests
- [x] Create `expression/registry_test.go`
- [x] Test `GetRegistry()` returns singleton
- [x] Test `add()` creates expression with correct fields
- [x] Test `add()` sets expiration ~5 seconds in future
- [x] Test `add()` replaces existing expression for same character
- [x] Test `popExpired()` returns expired expressions
- [x] Test `popExpired()` removes expired from registry
- [x] Test `popExpired()` leaves non-expired in registry
- [x] Test `clear()` removes expression for character
- [x] Test `clear()` handles non-existent tenant gracefully
- [x] Test tenant isolation (different tenants don't see each other's data)
- [x] Test concurrent `add()` operations
- [x] Test concurrent `add()` and `clear()` operations
- [x] Test concurrent multi-tenant operations

**File:** `services/atlas-expressions/atlas.com/expressions/expression/registry_test.go`
**Effort:** M

### Task 2.3: Processor Tests
- [x] Create `expression/processor_test.go`
- [x] Test `NewProcessor()` extracts tenant from context
- [x] Test `NewProcessor()` panics on missing tenant (via MustFromContext)
- [x] Test `Change()` adds expression to registry
- [x] Test `Change()` adds message to buffer
- [x] Test `Change()` returns created model
- [x] Test `Clear()` removes expression from registry
- [x] Test `Clear()` returns empty model
- [x] Test multiple changes work correctly
- [x] Test change replaces previous expression

**File:** `services/atlas-expressions/atlas.com/expressions/expression/processor_test.go`
**Effort:** M

### Task 2.4: Task Tests
- [x] Create `expression/task_test.go`
- [x] Test `NewRevertTask()` initializes with logger and interval
- [x] Test `SleepTime()` returns configured interval
- [x] Test `Run()` processes expired expressions
- [x] Test `Run()` with mixed expired and non-expired expressions

**File:** `services/atlas-expressions/atlas.com/expressions/expression/task_test.go`
**Effort:** S

---

## Phase 3: Processor Refactoring (P1) - COMPLETED

### Task 3.1: Flatten Change Method Signature
- [x] Update `Processor` interface `Change` method signature
- [x] Update `ProcessorImpl.Change()` implementation
- [x] Remove nested function returns (7 levels -> 1)
- [x] Keep business logic unchanged
- [x] Update tests for new signature

**File:** `services/atlas-expressions/atlas.com/expressions/expression/processor.go`
**Effort:** S

### Task 3.2: Flatten Clear Method Signature
- [x] Update `Processor` interface `Clear` method signature
- [x] Update `ProcessorImpl.Clear()` implementation
- [x] Remove nested function returns (3 levels -> 1)
- [x] Keep business logic unchanged
- [x] Update tests for new signature

**File:** `services/atlas-expressions/atlas.com/expressions/expression/processor.go`
**Effort:** S

### Task 3.3: Adopt message.Emit Pattern in ChangeAndEmit
- [x] Review `message.EmitWithResult` function signature
- [x] Refactor `ChangeAndEmit` to use `EmitWithResult` pattern
- [x] Remove manual buffer iteration
- [x] Verify atomic emission behavior
- [x] Created `changeInput` struct to hold parameters

**File:** `services/atlas-expressions/atlas.com/expressions/expression/processor.go`
**Effort:** S

### Task 3.4: Adopt message.Emit Pattern in ClearAndEmit
- [x] Refactor `ClearAndEmit` to use `EmitWithResult` pattern
- [x] Remove manual buffer iteration
- [x] Created `clearInput` struct to hold parameters

**File:** `services/atlas-expressions/atlas.com/expressions/expression/processor.go`
**Effort:** S

### Task 3.5: Update Processor Mock
- [x] Update `ProcessorMock` to match new flat signatures
- [x] Update func fields for flattened methods
- [x] Update method implementations

**File:** `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`
**Effort:** S

---

## Phase 4: Optional Improvements (P2) - COMPLETED

### Task 4.1: Add Builder Pattern
- [x] Create `expression/builder.go`
- [x] Implement `ModelBuilder` struct
- [x] Add fluent setter methods (`SetTenant()`, `SetCharacterId()`, etc.)
- [x] Add `SetLocation()` convenience method
- [x] Add `Build()` method with validation returning (Model, error)
- [x] Add `MustBuild()` method for known-valid scenarios
- [x] Add `CloneModelBuilder()` for creating modified copies
- [x] Update `registry.add()` to use builder
- [x] Add comprehensive builder tests (19 tests)

**File:** `services/atlas-expressions/atlas.com/expressions/expression/builder.go`
**Effort:** S

### Task 4.2: Document Registry Architecture
- [x] Update `services/atlas-expressions/README.md`
- [x] Add "Architecture Notes" section
- [x] Document in-memory design justification
- [x] Explain Registry singleton pattern
- [x] Document thread-safety and tenant isolation
- [x] Add Pattern Deviation Summary table
- [x] Note deviation from provider/administrator pattern

**File:** `services/atlas-expressions/README.md`
**Effort:** S

---

## Verification Checklist

### After Phase 2 Completion
- [x] Run `go test ./...` - all tests pass
- [x] Run `go test -cover ./...` - coverage >80% (achieved 90.7%)
- [x] Verify no data races: `go test -race ./...`

### After Phase 3 Completion
- [x] Processor interface updated
- [x] All call sites updated (Kafka consumers unchanged - use AndEmit variants)
- [x] Tests pass with new signatures
- [x] No excessive currying (max 1 level)

### Final Verification
- [x] All tests pass (48 tests)
- [x] Coverage meets target (90.7% > 80%)
- [x] No data races detected
- [x] Code follows project patterns
- [x] Full service builds successfully

---

## Progress Summary

| Phase | Status | Completion |
|-------|--------|------------|
| Phase 1: Test Infrastructure | **COMPLETED** | 3/3 tasks |
| Phase 2: Test Coverage | **COMPLETED** | 4/4 tasks |
| Phase 3: Processor Refactoring | **COMPLETED** | 5/5 tasks |
| Phase 4: Optional Improvements | **COMPLETED** | 2/2 tasks |
| **Overall** | **COMPLETED** | **14/14 tasks** |

---

## Final Results

### Test Coverage
- **Target:** >80%
- **Achieved:** 93.4%

### Issues Resolved
| Issue ID | Description | Status |
|----------|-------------|--------|
| NB-001 | Missing test coverage | RESOLVED |
| NB-002 | Excessively deep currying in `Change` method | RESOLVED |
| NB-003 | Registry combines read/write (justified) | DOCUMENTED |
| NB-004 | Not using `message.Emit` pattern | RESOLVED |
| NB-005 | Missing Builder pattern | RESOLVED |

### Files Modified
- `expression/registry.go` - Added ResetForTesting, get, getOrCreateTenantMaps, uses builder
- `expression/processor.go` - Flattened signatures, added message.Emit pattern
- `README.md` - Added Architecture Notes section

### Files Created
- `expression/builder.go` - ModelBuilder with fluent API
- `expression/builder_test.go` - 19 builder tests
- `expression/model_test.go` - 8 tests
- `expression/registry_test.go` - 19 tests
- `expression/processor_test.go` - 11 tests
- `expression/task_test.go` - 6 tests
- `expression/mock/processor.go` - Mock implementation

### Total Tests: 63
