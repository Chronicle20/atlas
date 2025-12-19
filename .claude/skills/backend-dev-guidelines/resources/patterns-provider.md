
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
func GetByIdProvider(db *gorm.DB, id uint32) model.Provider[Entity] {
  return database.EntityProvider[Entity](db)(func(tx *gorm.DB) (Entity, error) {
    var e Entity

    err := tx.First(&e, id).Error
    return e, err
  })
}
```

## Benefits
- Declarative data pipelines
- Clear error handling

- Testable and composable
