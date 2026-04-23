# Atlas Drop Information Remediation - Context

**Last Updated:** 2026-01-13

---

## Key Files

### Files to Modify

| File | Modification | Issue |
|------|--------------|-------|
| `services/atlas-drop-information/atlas.com/dis/monster/drop/provider.go` | Rename `makeDrop` to `modelFromEntity` | ARCH-005 |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/provider.go` | Rename `makeDrop` to `modelFromEntity` | ARCH-005 |
| `services/atlas-drop-information/atlas.com/dis/monster/drop/builder.go` | Change `Build()` return to `(Model, error)` | ARCH-002 |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/builder.go` | Change `Build()` return to `(Model, error)` | ARCH-002 |
| `services/atlas-drop-information/atlas.com/dis/monster/drop/processor.go` | Remove legacy wrappers (lines 41-58) | ARCH-013 |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/processor.go` | Remove legacy wrappers (lines 36-45) | ARCH-013 |
| `services/atlas-drop-information/atlas.com/dis/continent/processor.go` | Remove legacy wrappers (lines 58-67) | ARCH-013 |

### Files to Create

| File | Purpose | Issue |
|------|---------|-------|
| `services/atlas-drop-information/atlas.com/dis/monster/drop/mock/processor.go` | Mock implementation for testing | ARCH-012 |
| `services/atlas-drop-information/atlas.com/dis/monster/drop/processor_test.go` | Processor unit tests | ARCH-012 |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/mock/processor.go` | Mock implementation for testing | ARCH-012 |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/processor_test.go` | Processor unit tests | ARCH-012 |

### Reference Files

| File | Purpose |
|------|---------|
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/mock/processor.go` | Mock implementation pattern |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor_test.go` | Table-driven test pattern |
| `services/atlas-drop-information/atlas.com/dis/monster/drop/builder_test.go` | Existing test pattern in service |
| `services/atlas-drop-information/atlas.com/dis/monster/drop/model.go` | Model structure reference |
| `services/atlas-drop-information/atlas.com/dis/continent/drop/model.go` | Model structure reference |

---

## Processor Interfaces

### Monster Drop Processor

```go
// File: monster/drop/processor.go:11-14
type Processor interface {
    GetAll() model.Provider[[]Model]
    GetForMonster(monsterId uint32) model.Provider[[]Model]
}
```

### Continent Drop Processor

```go
// File: continent/drop/processor.go:11-13
type Processor interface {
    GetAll() model.Provider[[]Model]
}
```

### Continent Processor (Aggregate)

```go
// File: continent/processor.go:11-13
type Processor interface {
    GetAll() model.Provider[[]Model]
}
```

---

## Current Builder Implementation

### Monster Drop Builder

```go
// File: monster/drop/builder.go:50-61
func (b *Builder) Build() Model {
    return Model{
        tenantId:        b.tenantId,
        id:              b.id,
        monsterId:       b.monsterId,
        itemId:          b.itemId,
        minimumQuantity: b.minimumQuantity,
        maximumQuantity: b.maximumQuantity,
        questId:         b.questId,
        chance:          b.chance,
    }
}
```

### Continent Drop Builder

```go
// File: continent/drop/builder.go:50-61
func (b *Builder) Build() Model {
    return Model{
        tenantId:        b.tenantId,
        id:              b.id,
        continentId:     b.continentId,
        itemId:          b.itemId,
        minimumQuantity: b.minimumQuantity,
        maximumQuantity: b.maximumQuantity,
        questId:         b.questId,
        chance:          b.chance,
    }
}
```

---

## Legacy Wrappers to Remove

### Monster Drop (monster/drop/processor.go:41-58)

```go
// Legacy function wrappers for backward compatibility during migration
func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
    return func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
        return func(db *gorm.DB) ([]Model, error) {
            return NewProcessor(l, ctx, db).GetAll()()
        }
    }
}

func GetForMonster(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
    return func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
        return func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
            return func(monsterId uint32) ([]Model, error) {
                return NewProcessor(l, ctx, db).GetForMonster(monsterId)()
            }
        }
    }
}
```

### Continent Drop (continent/drop/processor.go:36-45)

```go
// Legacy function wrapper for backward compatibility during migration
func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
    return func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
        return func(db *gorm.DB) func() ([]Model, error) {
            return func() ([]Model, error) {
                return NewProcessor(l, ctx, db).GetAll()()
            }
        }
    }
}
```

### Continent Aggregate (continent/processor.go:58-67)

```go
// Legacy function wrapper for backward compatibility during migration
func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
    return func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
        return func(db *gorm.DB) func() ([]Model, error) {
            return func() ([]Model, error) {
                return NewProcessor(l, ctx, db).GetAll()()
            }
        }
    }
}
```

---

## Dependencies and Imports

### Required Imports for Mocks

```go
import (
    "github.com/Chronicle20/atlas-model/model"
)
```

### Required Imports for Tests

```go
import (
    "context"
    "errors"
    "testing"

    "github.com/Chronicle20/atlas-tenant"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus/hooks/test"
    "github.com/stretchr/testify/assert"
)
```

---

## Decisions

| Decision | Rationale | Date |
|----------|-----------|------|
| Use function fields for mocks | Matches existing mock pattern in atlas-saga-orchestrator | 2026-01-13 |
| Minimal validation in builders | Only validate nil UUID to avoid breaking existing code | 2026-01-13 |
| Remove legacy wrappers without deprecation | No external usage found in codebase search | 2026-01-13 |
| Table-driven tests | Follows Go testing best practices and existing patterns | 2026-01-13 |

---

## External References

- Audit document: `docs/audits/atlas-drop-information/audit.md`
- Audit JSON: `docs/audits/atlas-drop-information/audit.json`
- Backend guidelines: `.claude/skills/backend-dev-guidelines/`
