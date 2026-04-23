# Atlas-Account Remediation Context

**Last Updated:** 2026-01-13

---

## Key Files

### Core Domain Files
| File | Path | Purpose |
|------|------|---------|
| model.go | `services/atlas-account/atlas.com/account/account/model.go` | Immutable domain model with private fields and accessors |
| entity.go | `services/atlas-account/atlas.com/account/account/entity.go` | GORM entity with database tags |
| processor.go | `services/atlas-account/atlas.com/account/account/processor.go` | Business logic with ~15 methods |
| administrator.go | `services/atlas-account/atlas.com/account/account/administrator.go` | CRUD operations and Make() function |
| provider.go | `services/atlas-account/atlas.com/account/account/provider.go` | Data access layer |
| rest.go | `services/atlas-account/atlas.com/account/account/rest.go` | JSON:API transformation |
| resource.go | `services/atlas-account/atlas.com/account/account/resource.go` | REST route handlers |
| registry.go | `services/atlas-account/atlas.com/account/account/registry.go` | Session state singleton |
| producer.go | `services/atlas-account/atlas.com/account/account/producer.go` | Kafka message providers |

### Test Files
| File | Path | Status |
|------|------|--------|
| processor_test.go | `services/atlas-account/atlas.com/account/account/processor_test.go` | 1 test case |
| database_layer_test.go | `services/atlas-account/atlas.com/account/account/database_layer_test.go` | 3 test cases |
| registry_test.go | `services/atlas-account/atlas.com/account/account/registry_test.go` | 6 test cases |

### REST Handler
| File | Path | Purpose |
|------|------|---------|
| handler.go | `services/atlas-account/atlas.com/account/rest/handler.go` | Custom handler abstraction |

### Kafka
| Directory | Path | Purpose |
|-----------|------|---------|
| consumer/ | `services/atlas-account/atlas.com/account/kafka/consumer/` | Message consumption |
| producer/ | `services/atlas-account/atlas.com/account/kafka/producer/` | Message production |
| message/ | `services/atlas-account/atlas.com/account/kafka/message/` | Buffer implementation |

---

## Reference Implementations

### Builder Pattern Reference
**File:** `services/atlas-character/atlas.com/character/character/builder.go`

Key patterns to replicate:
```go
type Builder struct {
    // private fields matching Model
}

func NewBuilder(/* required params */) *Builder {
    // Initialize with defaults
}

func (b *Builder) SetField(value Type) *Builder {
    b.field = value
    return b
}

func (b *Builder) Build() Model {
    // Validate invariants
    // Return constructed Model
}
```

### Testing Reference
**File:** `.claude/skills/backend-dev-guidelines/resources/testing-guide.md`

Key patterns:
- Table-driven tests
- Mock producers
- Test both pure and AndEmit variants
- Test error paths

---

## Decisions Made

### 1. Handler Migration: DEFERRED
**Decision:** Keep custom `rest/handler.go`
**Rationale:**
- Current implementation works correctly
- Follows same patterns as shared library
- Migration would be high effort with low benefit
- No functional issues observed

### 2. Provider Pattern: DOCUMENT AS VARIANT
**Decision:** Document `database.EntityProvider[T]` as acceptable
**Rationale:**
- Functionally equivalent to documented pattern
- Difference is stylistic only
- Existing code works correctly

### 3. Registry Component: NO CHANGES
**Decision:** Keep current implementation
**Rationale:**
- Well-implemented singleton pattern
- Domain-specific requirements
- Already has comprehensive tests (6 cases)

---

## Technical Dependencies

### Internal Dependencies
```
account/processor.go
    → account/provider.go
    → account/administrator.go
    → account/registry.go
    → kafka/producer/producer.go
    → kafka/message/message.go

account/resource.go
    → account/processor.go
    → rest/handler.go

account/administrator.go
    → account/entity.go
    → account/model.go
```

### External Dependencies
```
github.com/Chronicle20/atlas-model/model
github.com/Chronicle20/atlas-tenant
github.com/Chronicle20/atlas-rest/server
github.com/sirupsen/logrus
gorm.io/gorm
golang.org/x/crypto/bcrypt
```

---

## Code Locations Requiring Changes

### Phase 1: Security Fix
```
processor.go:135 - Remove password from log
    Current: p.l.Debugf("Attempting to create account [%s] with password [%s].", name, password)
    Fix:     p.l.Debugf("Attempting to create account [%s].", name)
```

### Phase 2: Builder Pattern
```
NEW FILE: account/builder.go

Modify: administrator.go:87-101
    Current: Make(Entity) directly constructs Model
    Fix:     Use builder pattern internally
```

### Phase 3: REST Pattern
```
rest.go:52-69 - Transform function
    Current: Uses m.id, m.name, m.password (private fields)
    Fix:     Use m.Id(), m.Name(), m.Password() (accessor methods)

rest.go:71-84 - Extract function
    Current: Direct Model construction
    Fix:     Use builder pattern
```

### Phase 4: State Extraction
```
NEW FILE: account/state.go

Modify: model.go:8-14
    Current: State type and constants defined inline
    Fix:     Move to state.go, import in model.go
```

### Phase 5: Test Coverage
```
processor_test.go - Add tests for:
    - GetOrCreate
    - CreateAndEmit
    - Update
    - Login
    - Logout
    - LogoutAndEmit
    - AttemptLogin
    - AttemptLoginAndEmit
    - ProgressState
    - ProgressStateAndEmit
    - GetById
    - GetByName
    - GetByTenant
    - ByIdProvider
    - ByNameProvider
    - ByTenantProvider
    - LoggedInTenantProvider
```

---

## Model Structure

### Current Model Fields
```go
type Model struct {
    tenantId  uuid.UUID
    id        uint32
    name      string
    password  string
    pin       string
    pic       string
    state     State
    gender    byte
    banned    bool
    tos       bool
    updatedAt time.Time
}
```

### Current Accessors
- `Id() uint32`
- `Name() string`
- `Password() string`
- `Banned() bool`
- `State() State`
- `TOS() bool`
- `UpdatedAt() time.Time`
- `TenantId() uuid.UUID`
- `Pin() string`
- `Pic() string`

### Missing Accessor (needed for Transform)
- `Gender() byte` - NOT currently exported

---

## Invariants to Validate in Builder

1. **TenantId** - Must not be nil UUID
2. **Name** - Must not be empty string
3. **Id** - Must be > 0 (after database creation)

Optional validations:
- Password hash format (starts with $2)
- PIN format (if applicable)
- PIC format (if applicable)

---

## Testing Infrastructure

### Existing Test Helpers
```go
// From database_layer_test.go
setupTestDatabase(t *testing.T) *gorm.DB
sampleTenant() tenant.Model
```

### Needed Test Helpers
```go
// Mock producer for testing AndEmit variants
type MockProducer struct {}

// Test fixtures for various account states
sampleAccount() Model
loggedInAccount() Model
bannedAccount() Model
```

---

## Audit Check IDs Reference

| Check ID | Name | Status | Phase |
|----------|------|--------|-------|
| ARCH-003 | Builder Pattern | FAIL | Phase 2 |
| ARCH-005 | Provider Pattern | WARN | Phase 6 |
| ARCH-008 | REST JSON:API | WARN | Phase 3 |
| ARCH-012 | Testing Coverage | WARN | Phase 5 |
| (Security) | Password Logging | ISSUE | Phase 1 |
