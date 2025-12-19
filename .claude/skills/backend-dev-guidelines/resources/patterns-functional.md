
---
title: Functional and Builder Patterns
description: Immutability, builders, and curried function patterns for services.
---

# Functional and Builder Patterns


## Immutability
- Domain models have private fields.
- Public getters expose read-only state.
- All mutations occur via builders returning new instances.


## Builder Pattern
```go
m, err := NewBuilder().
  SetId(1).
  SetName("Example").
  Build()
```
- Validation occurs in `Build()`.
- Builders are fluent and chainable.
- `Model.Builder()` supports modification flows.


## Curried Function Pattern
```go
func Create(db *gorm.DB, log logrus.FieldLogger) func(input CreateParams) model.Provider[Entity]
```
- Encourages composition and DI without interfaces.
- Consistent function-first design over interface abstractions.

## Functional Composition
```go
result, err := model.
  Map(Transform)(entityProvider).

  (model.ParallelMap())()
```
