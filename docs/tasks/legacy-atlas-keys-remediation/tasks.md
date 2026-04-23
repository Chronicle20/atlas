# Atlas-Keys Remediation - Task Checklist

**Last Updated:** 2026-01-13

---

## Phase 1: Model API Completion

### 1.1 Add CharacterId Accessor (ARCH-002)
- [x] Add `CharacterId() uint32` method to `key/model.go`
- [x] Verify accessor returns correct value

**Acceptance Criteria:**
- Method signature: `func (m Model) CharacterId() uint32`
- Returns the private `characterId` field
- Follows existing accessor pattern in file

**File:** `services/atlas-keys/atlas.com/keys/key/model.go`

**Status:** COMPLETE

---

## Phase 2: Mock Infrastructure

### 2.1 Create Mock Directory
- [x] Create directory `services/atlas-keys/atlas.com/keys/key/mock/`

### 2.2 Create Processor Mock (TEST-002)
- [x] Create `key/mock/processor.go`
- [x] Add `ProcessorMock` struct with function fields:
  - [x] `ByCharacterIdProviderFunc`
  - [x] `GetByCharacterIdFunc`
  - [x] `ResetFunc`
  - [x] `CreateDefaultFunc`
  - [x] `DeleteFunc`
  - [x] `ChangeKeyFunc`
- [x] Implement each interface method delegating to function field
- [x] Return zero values when function is nil

**Acceptance Criteria:**
- Implements `key.Processor` interface
- Each method has configurable behavior via function field
- Can be used in tests to inject specific behaviors

**Reference:** `services/atlas-expressions/atlas.com/expressions/expression/mock/processor.go`

**Status:** COMPLETE

---

## Phase 3: Architectural Alignment

### 3.1 Entity Transformation Functions (ARCH-006)

#### 3.1.1 Add Make Function to entity.go
- [x] Add `Make(e entity) (Model, error)` function to `key/entity.go`
- [x] Copy logic from `makeKey` in processor.go
- [x] Export function (capital M)

#### 3.1.2 Add ToEntity Method to Model
- [x] Add `ToEntity(tenantId uuid.UUID) entity` method to `key/entity.go`
- [x] Map all Model fields to entity fields
- [x] Accept tenantId as parameter (Model doesn't store it)

#### 3.1.3 Update processor.go
- [x] Update `entityModelMapper` to use `Make` from entity.go
- [x] Update `entitySliceMapper` to use `Make` from entity.go
- [x] Remove `makeKey` function from processor.go

#### 3.1.4 Update administrator.go
- [x] Update `create` function to use `Make` instead of `makeKey`

**Acceptance Criteria:**
- `makeKey` function no longer exists in processor.go
- `Make` function exists in entity.go with same logic
- `ToEntity` method allows Model -> entity conversion
- All existing functionality continues to work

**Files:**
- `services/atlas-keys/atlas.com/keys/key/entity.go` (modified)
- `services/atlas-keys/atlas.com/keys/key/processor.go` (modified)
- `services/atlas-keys/atlas.com/keys/key/administrator.go` (modified)

**Status:** COMPLETE

---

### 3.2 Builder Pattern (ARCH-003)

#### 3.2.1 Create Builder Structure
- [x] Create `key/builder.go`
- [x] Add `ModelBuilder` struct with fields:
  - [x] `characterId uint32`
  - [x] `key int32`
  - [x] `theType int8`
  - [x] `action int32`

#### 3.2.2 Implement Constructors
- [x] Add `NewModelBuilder() *ModelBuilder`
- [x] Add `CloneModelBuilder(m Model) *ModelBuilder`

#### 3.2.3 Implement Fluent Setters
- [x] Add `SetCharacterId(characterId uint32) *ModelBuilder`
- [x] Add `SetKey(key int32) *ModelBuilder`
- [x] Add `SetType(theType int8) *ModelBuilder`
- [x] Add `SetAction(action int32) *ModelBuilder`

#### 3.2.4 Implement Build Methods
- [x] Add `Build() (Model, error)` with validation:
  - [x] Validate characterId > 0
  - [x] Return error with descriptive message on failure
- [x] Add `MustBuild() Model` that panics on error

#### 3.2.5 Implement Accessor Methods
- [x] Add `CharacterId() uint32` on builder
- [x] Add `Key() int32` on builder
- [x] Add `Type() int8` on builder
- [x] Add `Action() int32` on builder

**Acceptance Criteria:**
- Builder follows fluent pattern
- Build() returns error for invalid characterId
- MustBuild() panics for invalid input
- CloneModelBuilder preserves all fields

**Reference:** `services/atlas-expressions/atlas.com/expressions/expression/builder.go`

**Status:** COMPLETE

---

### 3.3 REST Transform Enhancement (Optional)

- [ ] Add `TransformSlice(models []Model) ([]RestModel, error)` to `key/rest.go`

**Status:** SKIPPED (Optional - not required for audit remediation)

---

## Phase 4: Test Coverage

### 4.1 Builder Tests (TEST-001 partial)

- [x] Create `key/builder_test.go`
- [x] Test: Build succeeds with valid characterId
- [x] Test: Build fails with zero characterId
- [x] Test: CloneModelBuilder copies all fields
- [x] Test: MustBuild panics on invalid input
- [x] Test: Fluent setters return builder for chaining
- [x] Test: Builder accessors return correct values

**Acceptance Criteria:**
- All builder validation rules are tested
- Tests are independent (no shared state)
- Tests use table-driven format where appropriate

**Status:** COMPLETE (6 tests)

---

### 4.2 Entity/Model Tests (TEST-001 partial)

- [x] Create `key/entity_test.go`
- [x] Test: Make transforms entity to model
- [x] Test: ToEntity transforms model to entity
- [x] Test: Round-trip entity->model->entity preserves data
- [x] Test: TableName returns correct value

- [x] Create `key/model_test.go`
- [x] Test: Model accessors return correct values
- [x] Test: Model with zero values works correctly

**Status:** COMPLETE (6 tests)

---

### 4.3 REST Tests (TEST-001 partial)

- [x] Create `key/rest_test.go`
- [x] Test: GetName returns 'keys'
- [x] Test: GetID returns key as string
- [x] Test: SetID parses string to key
- [x] Test: SetID returns error for invalid input
- [x] Test: Transform converts model to rest model

**Status:** COMPLETE (5 tests)

---

### 4.4 Processor Tests (TEST-001 partial)

- [ ] Create `key/processor_test.go`
- [ ] Test: GetByCharacterId returns models
- [ ] Test: CreateDefault creates 40 bindings
- [ ] Test: Reset removes and recreates bindings
- [ ] Test: ChangeKey creates new when not exists
- [ ] Test: ChangeKey updates when exists
- [ ] Test: Delete removes all character bindings

**Status:** DEFERRED (Requires database mocking or integration test setup)

**Note:** Core processor methods require database access. The mock infrastructure is in place to enable these tests when needed. Consider using testcontainers for integration tests.

---

### 4.5 REST Handler Tests (Optional) (TEST-001 partial)

- [ ] Create `character/resource_test.go`
- [ ] Test: GET /characters/{id}/keys returns key map
- [ ] Test: PUT /characters/{id}/keys updates binding
- [ ] Test: Invalid characterId returns error

**Status:** DEFERRED (Optional - can be added when processor tests are implemented)

---

## Verification

### Final Checks
- [x] Run `go test ./...` - all tests pass (17 tests)
- [x] Run `go test -cover ./...` - coverage 40.9% for key package
- [x] Run `go build ./...` - no build errors

### Re-Audit Verification
- [x] ARCH-002: CharacterId accessor exists - PASS
- [x] ARCH-003: builder.go exists with validation - PASS
- [x] ARCH-006: Make/ToEntity in entity.go - PASS
- [x] TEST-001: Test files exist - PASS (17 tests in 4 files)
- [x] TEST-002: Mock implementation exists - PASS

---

## Progress Summary

| Phase | Status | Completed |
|-------|--------|-----------|
| Phase 1: Model API | COMPLETE | 1/1 |
| Phase 2: Mock Infrastructure | COMPLETE | 2/2 |
| Phase 3: Architecture | COMPLETE | 8/9 (optional skipped) |
| Phase 4: Test Coverage | PARTIAL | 3/5 (deferred require DB) |
| **Total** | **COMPLETE** | **14/17** |

---

## Files Created/Modified

### New Files
| File | Purpose |
|------|---------|
| `key/builder.go` | Fluent model builder with validation |
| `key/builder_test.go` | Builder tests (6 tests) |
| `key/entity_test.go` | Entity transformation tests (4 tests) |
| `key/model_test.go` | Model accessor tests (2 tests) |
| `key/rest_test.go` | REST transformation tests (5 tests) |
| `key/mock/processor.go` | Mock Processor implementation |

### Modified Files
| File | Changes |
|------|---------|
| `key/model.go` | Added CharacterId() accessor |
| `key/entity.go` | Added Make() and ToEntity() functions |
| `key/processor.go` | Updated to use Make(), removed makeKey |
| `key/administrator.go` | Updated to use Make() |
