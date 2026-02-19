
---
title: Multi-Tenancy and Context
description: Context-based tenant extraction, automatic database filtering, and header propagation.
---


# Multi-Tenancy and Context

Tenant and trace identifiers flow through context, never request payloads.

## Extraction
```go
t := tenant.MustFromContext(ctx)
```

## Automatic Database Tenant Filtering

The shared `libs/atlas-database` library registers GORM callbacks that automatically inject `WHERE tenant_id = ?` from context on every Query, Update, and Delete operation. This eliminates the need for manual tenant filtering in providers and administrators.

### How It Works

1. `database.Connect()` registers GORM callbacks internally — no setup needed in `main.go`
2. Processors call `p.db.WithContext(p.ctx)` to propagate the tenant context to GORM
3. The callback reads `tenant.FromContext(ctx)` and adds `WHERE tenant_id = ?` automatically
4. Entities with no `tenant_id` column are unaffected (callback is a no-op)

### Provider Pattern (after automatic filtering)
```go
// Providers do NOT receive tenantId — filtering is automatic via context
func getById(id uint32) database.EntityProvider[Entity] {
    return func(db *gorm.DB) model.Provider[Entity] {
        var result Entity
        err := db.Where("id = ?", id).First(&result).Error
        return model.FixedProvider(result)
    }
}
```

### Administrator Pattern (after automatic filtering)
```go
// Create functions KEEP tenantId — needed to set the entity field
func create(db *gorm.DB, tenantId uuid.UUID, name string) (Model, error) {
    e := &Entity{TenantId: tenantId, Name: name}
    err := db.Create(e).Error
    return Make(*e)
}

// Update/Delete functions do NOT receive tenantId — filtering is automatic
func deleteById(db *gorm.DB, id uint32) error {
    return db.Where("id = ?", id).Delete(&Entity{}).Error
}
```

### Processor Pattern
```go
// Always use p.db.WithContext(p.ctx) to enable automatic filtering
func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
    return model.Map(Make)(getById(id)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) Create(name string) (Model, error) {
    return create(p.db.WithContext(p.ctx), p.t.Id(), name)
}
```

### Cross-Tenant Queries

For queries that must span all tenants (e.g., startup recovery, admin operations):
```go
import database "github.com/Chronicle20/atlas-database"

// Bypass automatic tenant filtering
ctx := database.WithoutTenantFilter(ctx)
db.WithContext(ctx).Find(&allResults)
```

### Test Setup

Tests that create SQLite databases directly (not via `database.Connect()`) must register callbacks manually:
```go
import database "github.com/Chronicle20/atlas-database"

func setupTestDB(t *testing.T) *gorm.DB {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    l, _ := test.NewNullLogger()
    database.RegisterTenantCallbacks(l, db)
    db.AutoMigrate(&Entity{})
    return db
}
```

### Important Gotchas

1. **GORM zero-value fields**: When removing `TenantId` from struct-based WHERE, use string-based `.Where("col = ?", val)` instead of `db.Where(&Entity{Field: val})`. GORM silently skips zero-value fields in struct queries.

2. **Batch deletes**: GORM requires a WHERE clause for batch deletes. When the only filter was `tenant_id`, use `Where("1 = 1")` — the callback still injects the tenant filter.

3. **Do NOT add `RegisterTenantCallbacks` to main.go**: `database.Connect()` already registers them. Only add to test files using SQLite directly.

## Required Headers
| Header | Example |
|--------|---------|
| TENANT_ID | 083839c6-c47c-42a6-9585-76492795d123 |

## Decorators
- `TenantHeaderDecorator(ctx)`
- `SpanHeaderDecorator(ctx)`

Always initialize producers using: `producer.ProviderImpl(log)(ctx)`.
