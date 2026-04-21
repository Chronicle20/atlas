# Atlas Fame Remediation - Context Document

**Last Updated:** 2026-01-13

This document captures key context, decisions, and dependencies for the atlas-fame remediation effort.

---

## Key Files

### Files to Modify

| File | Purpose | Changes Required |
|------|---------|------------------|
| `services/atlas-fame/atlas.com/fame/fame/provider.go` | Database read operations | Fix duplicate `.Find(&result)` bug on line 15 |
| `services/atlas-fame/atlas.com/fame/fame/model.go` | Immutable domain model | Add missing accessors: TenantId(), Id(), CharacterId(), Amount() |
| `services/atlas-fame/atlas.com/fame/fame/administrator.go` | Database write operations | Refactor to use builder pattern instead of direct entity construction |
| `services/atlas-fame/atlas.com/fame/character/rest.go` | Character REST model | Change GetName() from pointer receiver to value receiver |
| `services/atlas-fame/atlas.com/fame/kafka/consumer/fame/consumer.go` | Kafka consumer handler | Migrate from legacy function to Processor interface |
| `services/atlas-fame/atlas.com/fame/fame/processor.go` | Business logic orchestration | Remove legacy functions (lines 127-159) after migration |
| `services/atlas-fame/atlas.com/fame/fame/producer.go` | Kafka message providers | Remove `errorEventStatusProviderLegacy` function |

### Files to Create

| File | Purpose | Template Reference |
|------|---------|-------------------|
| `services/atlas-fame/atlas.com/fame/fame/builder.go` | Fluent builder with validation | `services/atlas-buddies/atlas.com/buddies/buddy/builder.go` |
| `services/atlas-fame/atlas.com/fame/fame/builder_test.go` | Builder unit tests | Standard table-driven tests |
| `services/atlas-fame/atlas.com/fame/fame/provider_test.go` | Provider unit tests | Standard table-driven tests |
| `services/atlas-fame/atlas.com/fame/fame/processor_test.go` | Processor unit tests | Standard table-driven tests |
| `services/atlas-fame/atlas.com/fame/fame/model_test.go` | Model accessor tests | Standard table-driven tests |

### Files to Delete

| File | Reason |
|------|--------|
| `services/atlas-fame/atlas.com/fame/kafka/consumer/fame/kafka.go` | Empty file with backward compatibility comment |

### Reference Files (Read Only)

| File | Purpose |
|------|---------|
| `services/atlas-fame/atlas.com/fame/fame/entity.go` | Entity definition with Make() function |
| `.claude/skills/backend-dev-guidelines/SKILL.md` | Architecture guidelines |
| `services/atlas-buddies/atlas.com/buddies/buddy/builder.go` | Builder pattern reference |
| `/docs/audits/atlas-fame/audit.md` | Full audit report |
| `/docs/audits/atlas-fame/audit.json` | Machine-readable audit data |

---

## Code Patterns

### Builder Pattern Template

Based on `atlas-buddies/buddy/builder.go`:

```go
package fame

import (
    "errors"
    "github.com/google/uuid"
)

type Builder struct {
    tenantId    uuid.UUID
    characterId uint32
    targetId    uint32
    amount      int8
}

func NewBuilder(tenantId uuid.UUID, characterId uint32, targetId uint32, amount int8) *Builder {
    return &Builder{
        tenantId:    tenantId,
        characterId: characterId,
        targetId:    targetId,
        amount:      amount,
    }
}

// Optional setters would go here if needed

func (b *Builder) Build() (Model, error) {
    if b.tenantId == uuid.Nil {
        return Model{}, errors.New("tenantId is required")
    }
    if b.characterId == 0 {
        return Model{}, errors.New("characterId is required")
    }
    if b.targetId == 0 {
        return Model{}, errors.New("targetId is required")
    }
    if b.amount != 1 && b.amount != -1 {
        return Model{}, errors.New("amount must be 1 or -1")
    }

    return Model{
        tenantId:    b.tenantId,
        characterId: b.characterId,
        targetId:    b.targetId,
        amount:      b.amount,
    }, nil
}
```

### Model Accessor Pattern

```go
func (m Model) TenantId() uuid.UUID {
    return m.tenantId
}

func (m Model) Id() uuid.UUID {
    return m.id
}

func (m Model) CharacterId() uint32 {
    return m.characterId
}

func (m Model) Amount() int8 {
    return m.amount
}
```

### Value Receiver Pattern for REST Models

```go
// Correct - value receiver
func (r RestModel) GetName() string {
    return "characters"
}

// Incorrect - pointer receiver (current state)
func (r *RestModel) GetName() string {
    return "characters"
}
```

---

## Business Rules

### Fame Change Validation Rules

These rules are implemented in `processor.go:57-118`:

1. **Character Existence:** Both source and target characters must exist
2. **Minimum Level:** Source character must be level 15 or higher
3. **Daily Limit:** Source character can only give fame once per day
4. **Monthly Target Limit:** Source character can only give fame to same target once per month
5. **Amount Values:** Fame amount is either +1 or -1

### Error Status Types

Defined in `kafka/message/fame/kafka.go`:
- `StatusEventErrorTypeUnexpected` - General errors
- `StatusEventErrorInvalidName` - Target character not found
- `StatusEventErrorTypeNotMinimumLevel` - Character below level 15
- `StatusEventErrorTypeNotToday` - Already gave fame today
- `StatusEventErrorTypeNotThisMonth` - Already gave fame to target this month

---

## Architectural Decisions

### Decision 1: Builder Validates Amount Values

**Context:** The fame system only allows +1 or -1 amounts.

**Decision:** Validate amount in builder rather than in administrator.

**Rationale:**
- Fail fast at model creation time
- Consistent with other builder implementations
- Administrator remains focused on persistence

**Alternatives Considered:**
- Validate in administrator (rejected: late validation)
- Validate in processor (rejected: business logic already there, but this is invariant)

### Decision 2: Keep REST Infrastructure

**Context:** The service has REST handler infrastructure but no REST endpoints.

**Decision:** Keep the infrastructure for cross-service client consistency.

**Rationale:**
- `rest/request.go` provides cross-service request helpers
- Removal could break pattern consistency across services
- Low maintenance burden

**Alternatives Considered:**
- Remove unused code (rejected: may break patterns)

### Decision 3: Test Coverage Priority

**Context:** No tests exist. Must prioritize which tests to add first.

**Decision:** Prioritize in order: Builder > Processor > Provider > Model

**Rationale:**
- Builder is new code with validation logic
- Processor has most complex business logic
- Provider is simple but had a bug
- Model accessors are trivial

---

## Dependencies

### Internal Package Dependencies

```
fame/processor.go
  ├── fame/provider.go (byCharacterIdLastMonthEntityProvider)
  ├── fame/administrator.go (create)
  ├── fame/producer.go (errorEventStatusProvider)
  ├── character/processor.go (character lookups, fame requests)
  ├── database/transaction.go (ExecuteTransaction)
  └── kafka/message/message.go (Buffer, Emit)

fame/administrator.go
  ├── fame/entity.go (Entity, Make)
  └── fame/builder.go (NEW: will depend on this)

kafka/consumer/fame/consumer.go
  └── fame/processor.go (legacy: RequestChange, future: NewProcessor)
```

### External Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/Chronicle20/atlas-model/model` | latest | Model providers, SliceMap |
| `github.com/Chronicle20/atlas-tenant` | latest | Multi-tenancy context |
| `github.com/Chronicle20/atlas-constants/world` | latest | World ID type |
| `github.com/Chronicle20/atlas-constants/channel` | latest | Channel ID type |
| `github.com/google/uuid` | latest | UUID generation |
| `github.com/sirupsen/logrus` | latest | Structured logging |
| `gorm.io/gorm` | latest | ORM for database |
| `github.com/jtumidanski/api2go/jsonapi` | latest | JSON:API interfaces |

---

## Test Strategy

### Mock Requirements

| Component | Mock Type | Purpose |
|-----------|-----------|---------|
| `*gorm.DB` | Actual test DB | Provider/Administrator tests |
| `character.Processor` | Interface mock | Processor tests |
| `message.Buffer` | Actual or mock | Verify emissions |
| `tenant.Model` | Actual | Context extraction |

### Test Data Requirements

```go
// Standard test tenant
testTenantId := uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Test fame logs
testLogs := []Entity{
    {TenantId: testTenantId, CharacterId: 100, TargetId: 200, Amount: 1, CreatedAt: time.Now()},
    {TenantId: testTenantId, CharacterId: 100, TargetId: 201, Amount: 1, CreatedAt: time.Now().AddDate(0, 0, -15)},
}

// Boundary dates
today := time.Now()
yesterday := today.AddDate(0, 0, -1)
lastMonth := today.AddDate(0, -1, 0)
twoMonthsAgo := today.AddDate(0, -2, 0)
```

---

## Rollback Plan

If issues are discovered after implementation:

1. **Phase 1 Rollback:**
   - Revert provider.go to original (duplicate Find kept)
   - Delete builder.go
   - Revert administrator.go to direct entity construction

2. **Phase 2 Rollback:**
   - Remove added accessors from model.go
   - Revert GetName() receiver change

3. **Phase 3 Rollback:**
   - Delete all test files (no production impact)

4. **Phase 4 Rollback:**
   - Revert consumer to legacy function usage
   - Restore legacy functions in processor.go and producer.go

---

## Open Questions

1. **Q:** Should builder set `createdAt` or should administrator set it?
   - **Proposed Answer:** Administrator sets it (persistence concern), builder validates other fields only.

2. **Q:** Should we add `SetCreatedAt()` to builder for testing purposes?
   - **Proposed Answer:** No, use entity.Make() for test model creation with specific timestamps.

3. **Q:** Are there any external callers of legacy functions beyond this service?
   - **Status:** Need to verify via codebase search before Phase 4.
