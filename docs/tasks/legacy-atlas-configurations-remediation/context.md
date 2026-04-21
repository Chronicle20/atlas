# atlas-configurations Remediation Context

**Last Updated:** 2026-01-13 (Implementation Complete)
**Plan Reference:** `plan.md`
**Tasks Reference:** `tasks.md`

---

## Key Files

### Domain Packages

#### Templates Package
| File | Purpose | Notes |
|------|---------|-------|
| `templates/processor.go` | CRUD operations for templates | Lines 65-95 contain Create logic |
| `templates/administrator.go` | Transaction functions for write ops | Contains `create()`, `update()`, `delete()` |
| `templates/provider.go` | Entity query providers | Correctly implemented |
| `templates/entity.go` | GORM entity definition | Uses `json.RawMessage` for data |
| `templates/rest.go` | REST model and JSON:API interface | Public fields (no accessors) |
| `templates/resource.go` | HTTP route registration | Uses `rest.RegisterHandler` |

#### Tenants Package
| File | Purpose | Notes |
|------|---------|-------|
| `tenants/processor.go` | CRUD operations for tenants | Lines 82-122 contain Create with optional ID |
| `tenants/administrator.go` | Transaction functions for write ops | Contains `update()`, `delete()` |
| `tenants/provider.go` | Entity query providers | Correctly implemented |
| `tenants/entity.go` | GORM entity with history tracking | Uses `HistoryEntity` |
| `tenants/rest.go` | REST model and JSON:API interface | Public fields (no accessors) |
| `tenants/resource.go` | HTTP route registration | Uses `rest.RegisterHandler` |

#### Services Package (Read-Only)
| File | Purpose | Notes |
|------|---------|-------|
| `services/processor.go` | Read operations only | No Create/Update/Delete |
| `services/provider.go` | Entity query providers | Correctly implemented |
| `services/entity.go` | GORM entity definition | |
| `services/resource.go` | HTTP route registration | Read endpoints only |
| `services/service/rest.go` | Service REST model | |
| `services/task/rest.go` | Task REST model | |

### Utility Packages
| File | Purpose | Notes |
|------|---------|-------|
| `database/provider.go` | Database EntityProvider type | |
| `database/transaction.go` | Transaction execution helper | |
| `rest/handler.go` | Generic REST handler utilities | |
| `seeder/seeder.go` | Seed data import from JSON | Has tests |
| `seeder/seeder_test.go` | Seeder tests | Good example pattern |

---

## Audit Findings Reference

### Blocking Issues

#### TEST-001: Missing Test Coverage
**Location:** templates/, tenants/, services/ packages
**Current State:** Only seeder package has tests
**Impact:** High - regressions can be introduced without detection
**Resolution:** Add processor_test.go and rest_test.go to each domain package

### Non-Blocking Issues

#### ARCH-001/002: No Immutable Domain Models
**Location:** templates/rest.go:10-20, tenants/rest.go:10-20
**Current State:** RestModel has public fields, used directly as domain model
**Decision:** Acceptable for simple CRUD with JSON blob storage
**Resolution:** Document as intentional; defer model.go introduction

#### ARCH-003: Administrator/Processor Overlap
**Location:** templates/administrator.go, templates/processor.go
**Current State:** Create in processor has duplicate logic; update/delete call administrator
**Resolution:** Clean up unused functions, consolidate ownership

#### ARCH-008: No Multi-Tenancy Context
**Location:** templates/processor.go:20-27, services/processor.go:22-29
**Current State:** No `tenant.MustFromContext(ctx)` calls
**Rationale:** This service manages tenant configurations, not tenant-scoped data
**Resolution:** Document as intentional architectural decision

---

## Code Patterns to Follow

### Existing Test Pattern (seeder_test.go)
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name        string
        input       InputType
        expected    ExpectedType
        expectError bool
    }{
        {
            name:        "valid case",
            input:       validInput,
            expected:    expectedResult,
            expectError: false,
        },
        {
            name:        "error case",
            input:       invalidInput,
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionUnderTest(tt.input)

            if tt.expectError {
                if err == nil {
                    t.Error("Expected error but got none")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
                return
            }

            // Verify result matches expected
        })
    }
}
```

### Processor Pattern
```go
type Processor struct {
    l   logrus.FieldLogger
    ctx context.Context
    db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
    return &Processor{l: l, ctx: ctx, db: db}
}

// Provider methods return model.Provider[T]
func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[RestModel] {
    return model.Map(Make)(byIdEntityProvider(p.ctx)(id)(p.db))
}

// Get methods use providers
func (p *Processor) GetById(id uuid.UUID) (RestModel, error) {
    return p.ByIdProvider(id)()
}
```

### Make Function Pattern
```go
func Make(e Entity) (RestModel, error) {
    var rm RestModel
    err := json.Unmarshal(e.Data, &rm)
    if err != nil {
        return RestModel{}, err
    }
    rm.Id = e.Id.String()
    return rm, nil
}
```

---

## Dependencies

### Testing Dependencies

#### In-Memory SQLite (Recommended)
```go
import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }

    // Auto-migrate test entities
    err = db.AutoMigrate(&Entity{})
    if err != nil {
        t.Fatalf("failed to migrate: %v", err)
    }

    return db
}
```

#### Test Logger
```go
func testLogger() logrus.FieldLogger {
    l := logrus.New()
    l.SetLevel(logrus.ErrorLevel) // Suppress logs during tests
    return l
}
```

---

## Decisions Log

### Decision 1: RestModel as Domain Model
**Date:** 2026-01-13
**Decision:** Keep RestModel as domain model, defer introduction of model.go/builder.go
**Rationale:**
- Service is simple CRUD with JSON blob storage
- No validation or business logic currently exists
- Introducing model layer adds complexity without benefit
**Trigger for revisiting:** Addition of validation rules or business logic

### Decision 2: Administrator.go Cleanup
**Date:** 2026-01-13
**Decision:** Option A - Keep write operations in processor.go, remove/simplify administrator.go
**Rationale:**
- Current code is inconsistent (some ops in processor, some in administrator)
- Service simplicity doesn't benefit from the separation
- Reduces code surface area
**Alternative considered:** Refactor processor to fully delegate to administrator (rejected for simplicity)

### Decision 3: Multi-Tenancy Exception
**Date:** 2026-01-13
**Decision:** Document as intentional architectural deviation
**Rationale:**
- This service manages tenant configurations for other services
- Multi-tenancy context would be semantically incorrect here
- "Tenant" in this service is the configuration being managed, not the request context

---

## Related Audits

- `dev/audits/atlas-configurations/audit.md` - Source audit document
- `dev/audits/atlas-configurations/audit.json` - Machine-readable audit data

---

## External References

### Atlas Backend Guidelines
- Location: `.claude/skills/backend-dev-guidelines/SKILL.md`
- Key sections:
  - Immutable Models pattern
  - Administrator/Processor separation
  - Provider pattern
  - Multi-tenancy context usage

### Similar Services for Reference
- `atlas-buffs` - Example of intentional in-memory deviation
- Other services with comprehensive test coverage for patterns to follow
