
---
title: Anti-Patterns
description: Common pitfalls to avoid when implementing Golang microservices.
---

# Anti-Patterns


| Anti-Pattern | Why It's Wrong |
|---------------|----------------|
| Business logic in handlers | Breaks separation of concerns |
| **Handlers calling provider functions directly** | **Breaks layer separation - handlers must call processors, not providers** |
| Mutable public fields | Violates immutability |
| Database logic in processors | Violates functional purity |
| **Cache in processor constructor** | **Cache is per-request instead of singleton; defeats caching purpose** |
| **Cache as processor instance field** | **Each request gets fresh empty cache; see [patterns-cache.md](patterns-cache.md)** |

| Hardcoded topics | Breaks environment portability |
| Missing validation | Allows invalid domain states |
| Passing TenantId to providers/update/delete | Automatic via GORM callbacks — only pass to create functions |
| Manual `Where("tenant_id = ?", ...)` in queries | Use `db.WithContext(ctx)` — GORM callback injects tenant filter |
| Adding `RegisterTenantCallbacks` to main.go | `database.Connect()` already registers them — only use in test files |
| Using struct-based WHERE after removing TenantId | GORM skips zero-value fields — use string-based `.Where("col = ?", val)` |
| Skipping header decorators | Breaks tracing and tenancy propagation |
| Global context usage | Breaks request isolation |
| Manual JSON:API envelope handling | Breaks JSON:API integration, adds boilerplate |
| Nested Data/Type/Attributes in requests | Use flat structures, let api2go handle envelope |
| Manual tenant parsing in handlers | Use `server.RegisterHandler` for automatic parsing |
| Custom error response helpers | Just write status codes directly |
| jsonapi struct tags on REST models | Use interface methods (`GetName`, `GetID`, `SetID`) |
| Plain http.HandlerFunc for routes | Use `server.RegisterHandler` for automatic tenant/tracing |

**Always** prefer pure, context-aware, curried, and testable functions.

**For REST:** Use `server.RegisterHandler` and `server.RegisterInputHandler` with flat JSON:API-compliant models.

---

## Critical Layer Violations

### ❌ Handlers Calling Providers Directly

**WRONG - Handler bypassing processor:**
```go
// resource.go - ANTI-PATTERN
func handleGetStorageRequest(db *gorm.DB) func(...) http.HandlerFunc {
    return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // ❌ WRONG - calling provider function directly from handler
            s, err := GetByWorldAndAccountId(d.Logger(), db, tenantId)(worldId, accountId)
            // ...
        }
    }
}
```

**Correct layer flow:**
```
resource.go (handler) → processor.go (business logic) → provider.go (data access) → database
```

**✅ CORRECT - Handler calling processor:**
```go
// resource.go - CORRECT PATTERN
func handleGetStorageRequest(db *gorm.DB) func(...) http.HandlerFunc {
    return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // ✅ CORRECT - calling processor method
            s, err := NewProcessor(d.Logger(), d.Context(), db).GetOrCreateStorage(worldId, accountId)
            // ...
        }
    }
}
```

**Why this matters:**
1. **Separation of concerns** - Handlers parse requests and marshal responses, processors contain business logic
2. **Testability** - Business logic in processors can be tested without HTTP infrastructure
3. **Reusability** - Processor methods can be called from handlers, Kafka consumers, or other processors
4. **Maintainability** - Changes to data access don't affect handlers
5. **Single responsibility** - Each layer has a clear, focused purpose

**Valid dependencies:**
- ✅ `resource.go` → `processor.go`
- ✅ `processor.go` → `provider.go`
- ✅ `provider.go` → `entity.go` + GORM

**Invalid dependencies:**
- ❌ `resource.go` → `provider.go` (bypasses processor layer)
- ❌ `resource.go` → `entity.go` (bypasses both processor and provider)
- ❌ `processor.go` → `entity.go` directly for database queries (should use provider)

### Exception: Cross-Domain Read-Only Views with Circular Dependencies

In rare cases where circular package dependencies prevent proper layering (e.g., `storage` imports `asset`, `asset` needs `storage`), read-only view handlers MAY use providers directly or raw DB queries for cross-domain orchestration.

**When this exception applies:**
- Handler aggregates data from multiple domains (e.g., assets + storage + stackable)
- Circular package dependency prevents calling processors
- Operation is read-only (no state changes)
- Alternative would require significant architectural refactoring

**Example:**
```go
// asset/resource.go - Read-only view handler
func handleGetAssetsRequest(db *gorm.DB) func(...) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ⚠️ EXCEPTION: Raw DB query to avoid circular dependency with storage package
        // Documented reason: storage package imports asset, can't import storage here
        var storageId uuid.UUID
        db.Table("storages").Select("id").
            Where("tenant_id = ? AND world_id = ? AND account_id = ?", ...).
            Scan(&storageId)

        // Then use asset provider
        assets, _ := asset.GetByStorageId(...)(storageId)
        // ...
    }
}
```

**Requirements for using this exception:**
1. Add a comment explaining WHY the circular dependency exists
2. Keep the raw query minimal (single table, simple where clause)
3. Consider architectural refactoring if this pattern appears frequently
4. Never use this exception for write operations - those MUST go through processors

---
