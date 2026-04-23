# Atlas Chalkboards Service Remediation Plan

**Last Updated:** 2026-01-13
**Audit Source:** `/docs/audits/atlas-chalkboards/audit.md`
**Service Path:** `services/atlas-chalkboards/atlas.com/chalkboards`
**Overall Status:** needs-work

---

## 1. Executive Summary

The atlas-chalkboards service audit identified 5 non-blocking issues requiring remediation. The service is a well-structured in-memory microservice for managing chalkboard messages, with correct layer separation and Kafka patterns. However, it lacks test coverage, has a tenant isolation gap, duplicates library functionality, and is missing an ingress route.

**Key Remediation Objectives:**
1. Add comprehensive test coverage (P0 - High Impact)
2. Fix tenant isolation in chalkboard registry (P1 - Security)
3. Migrate custom handlers to atlas-rest library (P1 - Maintenance)
4. Add missing ingress route (P1 - Functionality)
5. Add Builder pattern for Model (P2 - Optional)

**Total Estimated Effort:** M-L (3-5 development sessions)

---

## 2. Current State Analysis

### 2.1 Issues Summary

| Issue ID | Severity | Description | Status |
|----------|----------|-------------|--------|
| NB-001 | Medium | Custom handler infrastructure duplicates atlas-rest library | Open |
| NB-002 | High | No test coverage exists | Open |
| NB-003 | Medium | Chalkboard registry lacks tenant isolation | Open |
| NB-004 | Medium | Missing ingress route for `/api/chalkboards/{characterId}` | Open |
| NB-005 | Low | No Builder pattern for Model | Open |

### 2.2 Architecture Context

The service intentionally uses an in-memory architecture with Registry singletons instead of the standard database-backed provider/administrator pattern. This is appropriate because:
- Chalkboard data is ephemeral and session-bound
- State can be recovered when players re-create chalkboards
- Performance benefits from O(1) memory access

This architectural choice justifies the absence of `entity.go`, `provider.go`, and `administrator.go` files.

### 2.3 Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Cross-tenant data leakage | Medium | High | Priority fix for tenant isolation (NB-003) |
| Character endpoint unreachable | High | Medium | Add ingress route (NB-004) |
| Regression bugs | Medium | Medium | Test coverage (NB-002) |
| Maintenance burden | Low | Low | Migrate to library handlers (NB-001) |

---

## 3. Proposed Future State

After remediation, the service will:
1. Have comprehensive test coverage for all domain logic
2. Properly isolate chalkboard data by tenant
3. Use standard atlas-rest library handlers
4. Have all endpoints accessible via ingress
5. Optionally use Builder pattern for Model construction

### 3.1 Target File Structure

```
atlas-chalkboards/atlas.com/chalkboards/
├── chalkboard/
│   ├── model.go              # Add accessor methods
│   ├── builder.go            # NEW: Builder pattern (optional)
│   ├── registry.go           # MODIFY: Add tenant isolation
│   ├── registry_test.go      # NEW: Registry tests
│   ├── processor.go          # MODIFY: Pass tenant to registry
│   ├── processor_test.go     # NEW: Processor tests
│   ├── rest.go               # No changes
│   ├── rest_test.go          # NEW: Transform tests
│   ├── resource.go           # MODIFY: Use server.RegisterHandler
│   └── producer.go           # No changes
├── character/
│   ├── processor_test.go     # NEW: Processor tests
│   └── registry_test.go      # NEW: Registry tests
└── rest/
    └── handler.go            # DELETE: Remove custom handlers
```

---

## 4. Implementation Phases

### Phase 1: Tenant Isolation Fix (Priority: P1, Effort: S)
**Objective:** Prevent cross-tenant data leakage in chalkboard registry

This is the highest priority fix due to potential security implications.

**Tasks:**
1. Create ChalkboardKey struct with tenant field
2. Modify registry to use tenant-aware keys
3. Update processor to extract and pass tenant context
4. Add tests to verify tenant isolation

### Phase 2: Test Coverage (Priority: P0, Effort: L)
**Objective:** Establish comprehensive test coverage

**Tasks:**
1. Add model tests (if accessor methods added)
2. Add registry tests with concurrency verification
3. Add processor tests with mocked registries
4. Add REST transform tests
5. Add character package tests

### Phase 3: Library Migration (Priority: P1, Effort: M)
**Objective:** Replace custom handlers with atlas-rest library

**Tasks:**
1. Update resource.go to use server.RegisterHandler
2. Remove rest/handler.go custom implementations
3. Verify endpoint behavior unchanged
4. Update imports

### Phase 4: Ingress Configuration (Priority: P1, Effort: S)
**Objective:** Enable access to character-based endpoint

**Tasks:**
1. Add ingress route for `/api/chalkboards/{characterId}`
2. Verify endpoint accessibility

### Phase 5: Optional Improvements (Priority: P2, Effort: S)
**Objective:** Align with guidelines where beneficial

**Tasks:**
1. Add Builder pattern for chalkboard.Model
2. Add accessor methods to Model
3. Document architectural decisions in README

---

## 5. Detailed Task Breakdown

### Phase 1: Tenant Isolation Fix

#### Task 1.1: Create ChalkboardKey struct
**File:** `chalkboard/registry.go`
**Effort:** S
**Dependencies:** None

**Current State:**
```go
type Registry struct {
    mutex             sync.RWMutex
    characterRegister map[uint32]string  // Uses characterId only
}
```

**Target State:**
```go
type ChalkboardKey struct {
    Tenant      tenant.Model
    CharacterId uint32
}

type Registry struct {
    mutex             sync.RWMutex
    characterRegister map[ChalkboardKey]string
}
```

**Acceptance Criteria:**
- [ ] ChalkboardKey struct defined with Tenant and CharacterId fields
- [ ] Registry uses ChalkboardKey as map key
- [ ] Compiles successfully

#### Task 1.2: Update registry methods
**File:** `chalkboard/registry.go`
**Effort:** S
**Dependencies:** Task 1.1

**Changes Required:**
- Get(tenant, characterId) - add tenant parameter
- Set(tenant, characterId, value) - add tenant parameter
- Clear(tenant, characterId) - add tenant parameter

**Acceptance Criteria:**
- [ ] All registry methods accept tenant parameter
- [ ] Registry correctly isolates data by tenant
- [ ] Compiles successfully

#### Task 1.3: Update processor to extract tenant
**File:** `chalkboard/processor.go`
**Effort:** S
**Dependencies:** Task 1.2

**Changes Required:**
- Add tenant.Model field to ProcessorImpl
- Extract tenant in NewProcessor using tenant.MustFromContext
- Pass tenant to all registry calls

**Acceptance Criteria:**
- [ ] ProcessorImpl includes tenant field
- [ ] NewProcessor extracts tenant from context
- [ ] All processor methods pass tenant to registry
- [ ] Compiles successfully

#### Task 1.4: Add tenant isolation tests
**File:** `chalkboard/registry_test.go` (new)
**Effort:** S
**Dependencies:** Task 1.3

**Acceptance Criteria:**
- [ ] Test verifies different tenants cannot see each other's data
- [ ] Test verifies same tenant can access their own data
- [ ] All tests pass

---

### Phase 2: Test Coverage

#### Task 2.1: Add registry tests
**File:** `chalkboard/registry_test.go`
**Effort:** M
**Dependencies:** Phase 1 complete

**Test Cases:**
- Get returns empty for non-existent key
- Set stores value correctly
- Get retrieves stored value
- Clear removes value
- Clear returns false for non-existent key
- Concurrent access (multiple goroutines)
- Tenant isolation (separate tenant data)

**Acceptance Criteria:**
- [ ] Table-driven tests for all registry methods
- [ ] Concurrency test with multiple goroutines
- [ ] Tenant isolation verified
- [ ] All tests pass

#### Task 2.2: Add processor tests
**File:** `chalkboard/processor_test.go` (new)
**Effort:** M
**Dependencies:** Task 2.1

**Test Cases:**
- GetById returns error for non-existent chalkboard
- GetById returns model for existing chalkboard
- Set stores chalkboard message
- Clear removes chalkboard
- Clear does nothing for non-existent chalkboard

**Acceptance Criteria:**
- [ ] Table-driven tests for all processor methods
- [ ] Tests use proper tenant context
- [ ] All tests pass

#### Task 2.3: Add REST transform tests
**File:** `chalkboard/rest_test.go` (new)
**Effort:** S
**Dependencies:** None

**Test Cases:**
- Transform converts Model to RestModel correctly
- GetName returns "chalkboards"
- GetID returns string id
- SetID parses string to uint32
- SetID returns error for invalid input

**Acceptance Criteria:**
- [ ] All JSON:API interface methods tested
- [ ] Transform function tested
- [ ] All tests pass

#### Task 2.4: Add character package tests
**Files:** `character/processor_test.go`, `character/registry_test.go` (new)
**Effort:** M
**Dependencies:** None

**Test Cases (Registry):**
- GetInMap returns empty for non-existent key
- AddCharacter adds character to map
- RemoveCharacter removes character from map
- Tenant isolation verified

**Test Cases (Processor):**
- InMapProvider returns correct characters
- Enter adds character to map
- Exit removes character from map
- TransitionMap moves character between maps
- TransitionChannel moves character between channels

**Acceptance Criteria:**
- [ ] All registry methods tested
- [ ] All processor methods tested
- [ ] Tenant isolation verified
- [ ] All tests pass

---

### Phase 3: Library Migration

#### Task 3.1: Update resource.go to use server.RegisterHandler
**File:** `chalkboard/resource.go`
**Effort:** M
**Dependencies:** None

**Changes Required:**
- Replace rest.RegisterHandler with server.RegisterHandler
- Update handler signatures to match library expectations
- Update handler implementations

**Acceptance Criteria:**
- [ ] Uses server.RegisterHandler from atlas-rest
- [ ] All endpoints functional
- [ ] Tests pass (if any)

#### Task 3.2: Remove custom handler infrastructure
**File:** `rest/handler.go`
**Effort:** S
**Dependencies:** Task 3.1

**Changes Required:**
- Delete RegisterHandler function (lines 63-74)
- Delete RegisterInputHandler function (lines 76-87)
- Keep parser functions (ParseChannelId, ParseWorldId, etc.) if still needed
- Or delete entire file if parsers available in library

**Acceptance Criteria:**
- [ ] No duplicate handler registration code
- [ ] Service compiles and runs
- [ ] All endpoints functional

#### Task 3.3: Verify endpoint behavior
**Effort:** S
**Dependencies:** Task 3.2

**Acceptance Criteria:**
- [ ] GET /chalkboards/{characterId} returns correct response
- [ ] GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards returns correct response
- [ ] Headers (tenant, span) correctly parsed
- [ ] Error responses match previous behavior

---

### Phase 4: Ingress Configuration

#### Task 4.1: Add ingress route for character endpoint
**File:** `atlas-ingress.yml`
**Effort:** S
**Dependencies:** None

**Changes Required:**
Add route pattern:
```nginx
location ~ ^/api/chalkboards/[^/]+$ {
    proxy_pass http://atlas-chalkboards.atlas.svc.cluster.local:8080;
}
```

**Acceptance Criteria:**
- [ ] Route pattern matches `/api/chalkboards/{characterId}`
- [ ] Requests proxied to atlas-chalkboards service
- [ ] Does not interfere with existing routes

---

### Phase 5: Optional Improvements

#### Task 5.1: Add Builder pattern for Model (Optional)
**File:** `chalkboard/builder.go` (new)
**Effort:** S
**Dependencies:** None

**Implementation:**
```go
type Builder struct {
    id      uint32
    message string
}

func NewBuilder(id uint32) *Builder {
    return &Builder{id: id}
}

func (b *Builder) SetMessage(message string) *Builder {
    b.message = message
    return b
}

func (b *Builder) Build() Model {
    return Model{id: b.id, message: b.message}
}
```

**Acceptance Criteria:**
- [ ] Builder follows project conventions
- [ ] Processor uses Builder instead of inline construction
- [ ] Tests added for Builder

#### Task 5.2: Add accessor methods to Model (Optional)
**File:** `chalkboard/model.go`
**Effort:** S
**Dependencies:** None

**Implementation:**
```go
func (m Model) Id() uint32 {
    return m.id
}

func (m Model) Message() string {
    return m.message
}
```

**Acceptance Criteria:**
- [ ] Accessor methods defined
- [ ] Transform function uses accessors

#### Task 5.3: Document architectural decisions
**File:** `services/atlas-chalkboards/README.md`
**Effort:** S
**Dependencies:** None

**Changes Required:**
Add section documenting:
- Why Registry singletons instead of provider/administrator
- In-memory architecture rationale
- Ephemeral data considerations

**Acceptance Criteria:**
- [ ] Architectural decisions documented
- [ ] Justification for guideline deviations explained

---

## 6. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test Coverage | >80% | `go test -cover` |
| Tenant Isolation | Verified | Test cases pass |
| Library Compliance | Full | No custom handler code |
| Endpoint Accessibility | 100% | All endpoints reachable via ingress |
| Build Status | Green | CI/CD passes |

---

## 7. Dependencies and Resources

### External Dependencies
- atlas-rest library (for server.RegisterHandler)
- atlas-tenant library (for tenant context)
- Testing frameworks (standard Go testing)

### Knowledge Requirements
- Go testing patterns used in this codebase
- JSON:API conventions
- Kubernetes ingress configuration

### Reference Files
- `services/atlas-account/atlas.com/account/account/rest_test.go` - Test patterns
- `services/atlas-account/atlas.com/account/account/registry_test.go` - Registry test patterns
- `character/processor.go` - Correct tenant extraction example

---

## 8. Notes and Considerations

### 8.1 Error Handling in Kafka Consumers
The audit noted that some Kafka consumer handlers ignore errors:
```go
_ = chalkboard.NewProcessor(...).Clear(...)
```
This is likely intentional (fire-and-forget), but should be verified during implementation.

### 8.2 RestModel ID Type
The RestModel uses `uint32` for Id but converts via `strconv.Atoi/Itoa`. Consider using `strconv.ParseUint` for proper uint32 handling if this causes issues.

### 8.3 In-Memory State Recovery
The service design accepts that state is lost on restart. Ensure this is acceptable for the use case and documented.
