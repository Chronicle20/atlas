
---
title: Testing Conventions
description: Testing patterns and practices for Atlas Golang microservices.
---

# Testing Conventions


## Focus Areas

1. **Builders** — Validate invariants.
2. **Processors** — Test pure and `AndEmit` forms separately.
3. **Providers** — Validate retrieval and error paths.
4. **REST** — Verify status mapping and JSON:API output.

## Guidelines
- Prefer table-driven tests.
- Mock Kafka producers and DB providers.
- Verify tenant + span propagation.

## Example
```go
func TestBuilderValidation(t *testing.T) {
  _, err := NewBuilder().SetId(0).Build()
  require.Error(t, err)
}
```
