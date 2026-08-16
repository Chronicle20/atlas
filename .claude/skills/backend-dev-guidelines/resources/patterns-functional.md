
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

## Composition Recipes

The combinators most services reach for:

```go
// Entity provider → domain models
model.Map(Make)(entityProvider)(model.ParallelMap())

// One model → its REST representation
res, err := model.Map(Transform)(model.FixedProvider(m))()

// A slice of models → REST representations
res, err := model.SliceMap(Transform)(model.FixedProvider(models))(model.ParallelMap())()
```

**These are primitives, not an exemption from DOM-05.** The last recipe is what
a `TransformSlice([]Model) ([]RestModel, error)` function contains; DOM-05
requires that function to exist in `rest.go` and the list handler to call it,
rather than inlining the composition in `resource.go`. Earlier audits
(task-133, task-198) cited this snippet's former home as sanctioning the inline
form — it does not. The rule of record is DOM-05 in
[audit-checklist.md](audit-checklist.md).
