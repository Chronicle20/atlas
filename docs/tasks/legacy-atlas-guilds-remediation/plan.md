# Atlas-Guilds Service Remediation Plan

**Service:** `services/atlas-guilds`
**Audit Reference:** `/dev/audits/atlas-guilds/audit.md`
**Last Updated:** 2026-01-13 (Revision 2)

---

## 1. Executive Summary

The `atlas-guilds` service audit identified an **overall status of NEEDS-WORK**. While the service demonstrates good adherence to core architectural patterns (immutability, builder pattern, Kafka producers/consumers, REST handlers), there are gaps that require remediation:

**Critical Issues (P1 - Medium Impact):**
- ARCH-010: Model.Builder() methods return pointers to internal state (immutability concern)

**Non-Critical Issues (P2 - Low Impact):**
- ARCH-003: Missing `provider.go` files in nested packages (WARN)
- REST-002: Missing JSON:API interface methods on embedded REST models (WARN)
- STRUCT-002: Empty `guild/character/administrator.go` file (WARN)

**Resolved Issues:**
- TEST-001: Processor tests now present and comprehensive (PASS)

**Overall Assessment:** No blocking issues. Service is functional with solid core patterns. Primary remediation focus is fixing the Model.Builder() immutability issue.

---

## 2. Current State Analysis

### 2.1 Audit Findings Summary

| Check ID | Status | Impact | Description |
|----------|--------|--------|-------------|
| ARCH-001 | PASS | Low | Model immutability pattern correctly implemented |
| ARCH-002 | PASS | Low | Builder pattern with validation in `Build()` |
| ARCH-003 | WARN | Low | Missing `provider.go` in member, title, reply packages |
| ARCH-004 | PASS | Low | Processor pattern with proper DI |
| ARCH-005 | PASS | Low | Entity and Make functions properly implemented |
| **ARCH-010** | **WARN** | **Medium** | **Model.Builder() returns pointers to internal state** |
| KAFKA-001 | PASS | Low | Producer pattern with header decorators |
| KAFKA-002 | PASS | Low | Consumer pattern with header parsers |
| KAFKA-003 | PASS | Low | Message type definitions with env topic constants |
| REST-001 | PASS | Low | Handler delegation to processors |
| REST-002 | WARN | Low | Nested REST models missing JSON:API methods |
| REST-003 | PASS | Low | RegisterHandler usage correct |
| REST-004 | PASS | Low | Cross-service REST clients follow pattern |
| TEST-001 | PASS | Low | Processor tests present and comprehensive |

### 2.2 Current Test Coverage

Test files present:
- `guild/builder_test.go` - Builder validation tests
- `guild/builder_test.go` - Builder immutability tests
- `guild/processor_test.go` - Processor tests (GetById, GetByName, GetSlice, tenant isolation)
- `guild/member/builder_test.go` - Member builder tests
- `guild/member/processor_test.go` - Member processor tests
- `guild/title/builder_test.go` - Title builder tests
- `guild/title/processor_test.go` - Title processor tests
- `thread/builder_test.go` - Thread builder tests
- `thread/processor_test.go` - Thread processor tests
- `thread/reply/builder_test.go` - Reply builder tests
- `thread/reply/processor_test.go` - Reply processor tests

**Existing Mocks:**
- `character/mock/processor.go` - External character service mock
- `party/mock/processor.go` - External party service mock

### 2.3 Processor Interface Complexity

The guild processor (`guild/processor.go`) has a complex interface with 20+ methods:
- Read operations: `GetById`, `GetByName`, `GetByMemberId`, `GetSlice`, `AllProvider`, `ByIdProvider`, `ByNameProvider`
- Write operations: `RequestCreate`, `Create`, `CreationAgreementResponse`, `ChangeEmblem`, `UpdateMemberOnline`, `ChangeNotice`, `Leave`, `RequestInvite`, `Join`, `ChangeTitles`, `ChangeMemberTitle`, `RequestDisband`, `RequestCapacityIncrease`
- AndEmit variants for all write operations

### 2.4 External Dependencies

The guild processor depends on external services that need mocking:
- `character.Processor` - External character REST client
- `party.Processor` - External party REST client
- `coordinator.Registry` - In-memory guild creation coordinator

---

## 3. Proposed Future State

After remediation, the atlas-guilds service will have:

1. **Complete Test Coverage (P1):**
   - Processor tests for all domain packages (guild, thread, member, title, reply)
   - Test patterns matching `atlas-fame` service reference implementation
   - Mock directories for test isolation of external dependencies
   - Use of SQLite in-memory database for unit test isolation

2. **Consistent Package Structure (P2):**
   - Optional: Provider.go files in nested packages (if direct queries needed)
   - Optional: JSON:API interface compliance on all REST models
   - Clean removal of empty placeholder files

3. **Maintainability:**
   - Clear test patterns for future development
   - Consistent file organization across packages

---

## 4. Implementation Phases

### Phase 1: Fix Model.Builder() Immutability (P1 - Medium Impact)
**Objective:** Fix Model.Builder() methods to use value copies instead of pointers to internal state

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 1.1 | Fix `guild/builder.go` Model.Builder() method | S | None |
| 1.2 | Fix `guild/member/builder.go` Model.Builder() method | S | None |
| 1.3 | Fix `guild/title/builder.go` Model.Builder() method | S | None |
| 1.4 | Fix `thread/builder.go` Model.Builder() method | S | None |
| 1.5 | Fix `thread/reply/builder.go` Model.Builder() method | S | None |
| 1.6 | Add builder immutability tests to verify fix | S | 1.1-1.5 |

### Phase 2: Add Provider Files (P2 - Low Priority)
**Objective:** Add provider.go files for consistency across packages

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 2.1 | Create `guild/member/provider.go` with getByGuildId, getById | S | None |
| 2.2 | Create `guild/title/provider.go` with getByGuildId | S | None |
| 2.3 | Create `thread/reply/provider.go` with getByThreadId | S | None |

### Phase 3: Structural Cleanup (P2 - Low Priority)
**Objective:** Address remaining audit warnings

| Task | Description | Effort | Dependencies |
|------|-------------|--------|--------------|
| 3.1 | Handle `guild/character/administrator.go` (empty file) | S | None |
| 3.2 | Add JSON:API methods to `guild/member/rest.go` (optional) | S | None |
| 3.3 | Add JSON:API methods to `guild/title/rest.go` (optional) | S | None |
| 3.4 | Add JSON:API methods to `thread/reply/rest.go` (optional) | S | None |

---

## 5. Detailed Task Specifications

### 5.1 Phase 1: Fix Model.Builder() Immutability

#### Task 1.1-1.5: Fix Model.Builder() Methods

**Current Issue (all builder files):**
```go
func (m Model) Builder() *Builder {
    return &Builder{
        tenantId: &m.tenantId,  // Direct pointer to model field - BAD
        guildId:  &m.guildId,   // Allows indirect mutation of original model
        // ...
    }
}
```

**Required Fix:**
```go
func (m Model) Builder() *Builder {
    // Create value copies first
    tenantId := m.tenantId
    guildId := m.guildId
    name := m.name
    // ... all fields

    return &Builder{
        tenantId: &tenantId,    // Pointer to copy - GOOD
        guildId:  &guildId,     // Original model remains unmodified
        name:     &name,
        // ...
    }
}
```

**Files to Update:**
- `guild/builder.go` (Task 1.1)
- `guild/member/builder.go` (Task 1.2)
- `guild/title/builder.go` (Task 1.3)
- `thread/builder.go` (Task 1.4)
- `thread/reply/builder.go` (Task 1.5)

**Acceptance Criteria:**
- Each Model.Builder() creates local copies before pointer assignment
- Existing builder tests continue to pass
- `go build` succeeds
- `go vet` shows no warnings

#### Task 1.6: Add Builder Immutability Tests
**Location:** Existing builder test files

**Test Pattern:**
```go
func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
    tenantId := uuid.New()
    original, err := NewBuilder(tenantId, 1, "TestGuild", 100).Build()
    require.NoError(t, err)

    originalNotice := original.Notice()

    // Get builder from model and modify it
    builder := original.Builder()
    builder.SetNotice("Modified Notice")

    // Build new model
    modified, err := builder.Build()
    require.NoError(t, err)

    // Verify original is unchanged
    assert.Equal(t, originalNotice, original.Notice(), "original should not be mutated")
    assert.NotEqual(t, original.Notice(), modified.Notice(), "modified should be different")
}
```

**Acceptance Criteria:**
- Test added to each builder_test.go file
- Tests verify original model is not mutated when builder modifies values
- All tests pass

### 5.2 Phase 2: Add Provider Files

#### Task 2.1: Create guild/member/provider.go
**File:** `guild/member/provider.go`

**Functions to Create:**
```go
func getByGuildId(tenantId uuid.UUID, guildId uint32) func(db *gorm.DB) func() ([]Entity, error)
func getById(tenantId uuid.UUID, guildId uint32, characterId uint32) func(db *gorm.DB) func() (Entity, error)
```

**Acceptance Criteria:**
- Functions follow curried provider pattern
- Tenant filtering included
- `go build` succeeds

#### Task 2.2: Create guild/title/provider.go
**File:** `guild/title/provider.go`

**Functions to Create:**
```go
func getByGuildId(tenantId uuid.UUID, guildId uint32) func(db *gorm.DB) func() ([]Entity, error)
```

#### Task 2.3: Create thread/reply/provider.go
**File:** `thread/reply/provider.go`

**Functions to Create:**
```go
func getByThreadId(tenantId uuid.UUID, threadId uint32) func(db *gorm.DB) func() ([]Entity, error)
```

### 5.3 Phase 3: Structural Cleanup

#### Task 3.1: Handle Empty Administrator File
**File:** `guild/character/administrator.go`

**Decision:** Delete the empty file
- The package only contains read operations (provider.go)
- Write operations (`SetGuild`) are in processor.go
- Empty placeholder provides no value

**Acceptance Criteria:**
- File deleted
- `go build` succeeds
- No broken imports

#### Tasks 3.2-3.4: Add JSON:API Methods (Optional)
**Files:**
- `guild/member/rest.go`
- `guild/title/rest.go`
- `thread/reply/rest.go`

**Methods to Add (if needed):**
```go
func (r RestModel) GetName() string { return "members" }  // or "titles", "replies"
func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(strId string) error { /* parse and set */ }
```

**Note:** These are optional. The REST models are currently embedded-only in parent responses. Add interface methods only if standalone resource use is required.

---

## 6. Risk Assessment and Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Builder fix breaks existing code | Low | Low | Existing tests validate behavior; adding immutability-specific tests |
| Provider files create duplication | Low | Low | Provider files complement preloading, don't replace it |
| REST interface methods conflict | Very Low | Low | Methods follow established patterns from guild/rest.go |

---

## 7. Success Metrics

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| ARCH-010 Status | WARN | PASS | Model.Builder() uses value copies |
| Builder immutability tests | Partial | Complete | Tests verify no mutation |
| Provider file consistency | 60% | 100% | All nested packages have provider.go |
| STRUCT-002 (empty file) | 1 | 0 | File removed |

---

## 8. Required Resources and Dependencies

### Go Dependencies (Already Present)
- `github.com/stretchr/testify` - Assertions
- `gorm.io/driver/sqlite` - In-memory test database
- `github.com/sirupsen/logrus/hooks/test` - Null logger for tests

### Reference Implementation
- `guild/provider.go` - Provider pattern reference
- `guild/builder.go` - Builder pattern reference

### Key Files to Modify
- `guild/builder.go` - Model.Builder() method (~15 fields)
- `guild/member/builder.go` - Model.Builder() method (~9 fields)
- `guild/title/builder.go` - Model.Builder() method (~4 fields)
- `thread/builder.go` - Model.Builder() method (~10 fields)
- `thread/reply/builder.go` - Model.Builder() method (~5 fields)

---

## 9. Notes and Decisions

### Architectural Notes

1. **GORM Preloading Pattern:** The member, title, and reply packages use GORM preloading through their parent providers rather than separate provider files. This is a valid optimization. Adding provider.go files provides an alternative access pattern but doesn't replace preloading.

2. **Embedded REST Models:** The `guild/member/rest.go`, `guild/title/rest.go`, and `thread/reply/rest.go` models are used as embedded fields in parent responses. The JSON:API interface methods are optional since these are embedded-only.

3. **Immutability Fix:** The Model.Builder() fix is straightforward - create local copies of all model fields before assigning to builder pointers. This prevents indirect mutation of the original model through the builder.

### Out of Scope

- Integration tests with actual Kafka
- End-to-end tests with external services
- Performance testing
