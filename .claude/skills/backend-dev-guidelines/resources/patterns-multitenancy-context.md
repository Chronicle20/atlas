
---
title: Multi-Tenancy and Context
description: Context-based tenant extraction and header propagation.
---


# Multi-Tenancy and Context

Tenant and trace identifiers flow through context, never request payloads.

## Extraction
```go
t := tenant.MustFromContext(ctx)
member, err := CreateMember(db, log)(userId, t.Id())()
```

## Required Headers
| Header | Example |
|--------|---------|
| TENANT_ID | 083839c6-c47c-42a6-9585-76492795d123 |

## Decorators
- `TenantHeaderDecorator(ctx)`
- `SpanHeaderDecorator(ctx)`

Always initialize producers using: `producer.ProviderImpl(log)(ctx)`.
