
---
title: Provider Pattern
description: Functional data access pattern used for lazy evaluation and error propagation.
---

# Provider Pattern


Encapsulates deferred operations and functional composition for data retrieval.

## Concepts
- Return `model.Provider[T]` for lazy evaluation.
- Compose via `model.Map`, `model.SliceMap`, and `model.ParallelMap`.
- Use `model.ErrorProvider[T]` for error propagation.

## Example
```go
// Provider does NOT take tenantId — automatic filtering via GORM callbacks
func getById(id uint32) database.EntityProvider[Entity] {
    return func(db *gorm.DB) model.Provider[Entity] {
        var result Entity
        err := db.Where("id = ?", id).First(&result).Error
        if err != nil {
            return model.ErrorProvider[Entity](err)
        }
        return model.FixedProvider(result)
    }
}

// Called from processor with contextualized db:
func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
    return model.Map(Make)(getById(id)(p.db.WithContext(p.ctx)))
}
```

## Automatic Tenant Filtering

The `libs/atlas-database` library registers GORM callbacks that inject `WHERE tenant_id = ?` automatically when `db.WithContext(ctx)` is used and the entity has a `tenant_id` column. See [patterns-multitenancy-context.md](patterns-multitenancy-context.md) for full details.

**Key rules:**
- Providers do NOT receive `tenantId` — filtering is automatic
- Always use `p.db.WithContext(p.ctx)` when calling providers from processors
- Use string-based WHERE (not struct-based) to avoid GORM zero-value gotcha

## Benefits
- Declarative data pipelines
- Clear error handling
- Automatic tenant isolation via GORM callbacks
- Testable and composable
