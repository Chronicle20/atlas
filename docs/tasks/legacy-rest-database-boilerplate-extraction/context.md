# REST/Database Boilerplate Extraction — Context

Last Updated: 2026-02-19

## Key Files

### Shared Libraries (modification targets)

| File | Purpose |
|------|---------|
| `libs/atlas-rest/server/handler.go` | Existing: `RetrieveSpan()`, `ParseTenant()` |
| `libs/atlas-rest/server/response.go` | Existing: `Marshal()`, `MarshalResponse()` |
| `libs/atlas-rest/requests/get.go` | Existing: `MakeGetRequest[A]()` |
| `libs/atlas-rest/requests/post.go` | Existing: `MakePostRequest[A]()` |
| `libs/atlas-rest/requests/patch.go` | Existing: `MakePatchRequest[A]()` |
| `libs/atlas-rest/requests/put.go` | Existing: `MakePutRequest[A]()` |
| `libs/atlas-rest/requests/delete.go` | Existing: `MakeDeleteRequest()` |
| `libs/atlas-rest/requests/header.go` | Existing: `SpanHeaderDecorator()`, `TenantHeaderDecorator()` |
| `libs/atlas-database/connection.go` | Existing: `Connect()`, `DSNBuilder` |
| `libs/atlas-database/provider.go` | Existing: `EntityProvider[E]` type |
| `libs/atlas-database/transaction.go` | Existing: `ExecuteTransaction()` |
| `libs/atlas-database/tenant_scope.go` | Existing: GORM callbacks for auto-tenant filtering |

### Representative Service Files (duplication sources)

**Typical handler.go (no DB, with tenant)** — e.g., `services/atlas-buddies/atlas.com/buddies/rest/handler.go`
**Typical handler.go (with DB, with tenant)** — e.g., `services/atlas-account/atlas.com/account/rest/handler.go`
**Typical handler.go (no DB, no tenant)** — e.g., `services/atlas-tenants/atlas.com/tenants/rest/handler.go`
**Typical request.go** — e.g., `services/atlas-guilds/atlas.com/guilds/rest/request.go`
**Typical connection.go** — e.g., `services/atlas-keys/atlas.com/keys/database/connection.go`

## Architecture Decisions

### AD-1: Keep GORM out of atlas-rest

**Decision**: atlas-rest must NOT depend on gorm.io/gorm.

**Rationale**: atlas-rest is used by all 47 services, including 32 that don't use databases. Adding a GORM dependency would bloat those services.

**Approach**: `HandlerDependency` uses an interface or type parameter for the DB field:
```go
// Option A: Interface (preferred for simplicity)
type DBAccessor interface {
    DB() *gorm.DB  // But this still imports gorm...
}

// Option B: Generic HandlerDependency (avoids gorm import)
type HandlerDependency[D any] struct {
    l   logrus.FieldLogger
    dep D
    ctx context.Context
}

// Option C: Separate sub-package atlas-rest/dbserver (keeps gorm isolated)
// atlas-rest/server/ — no gorm
// atlas-rest/dbserver/ — imports gorm, extends HandlerDependency
```

**Recommended**: Option C — a new `atlas-rest/dbserver` sub-package that imports gorm and provides `RegisterDBHandler`. This keeps the main `server` package gorm-free while avoiding generics complexity.

### AD-2: Curried Function Signatures Preserved

**Decision**: Maintain the existing curried function style `RegisterHandler(l)(si)(name, handler)`.

**Rationale**: All 47 services already use this pattern in their router initialization code. Changing it would require modifying every `InitResource()` call, increasing migration scope.

### AD-3: Generic ID Parsers Use Type Constraints

**Decision**: Use Go generics with type constraints for integer-based ID parsers.

**Approach**:
```go
type IntegerId interface {
    ~uint32 | ~int32 | ~int8 | ~uint16
}

func ParseIntId[T IntegerId](l logrus.FieldLogger, varName string, next func(T) http.HandlerFunc) http.HandlerFunc
func ParseUUIDId(l logrus.FieldLogger, varName string, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc
func ParseStringId(l logrus.FieldLogger, varName string, next func(string) http.HandlerFunc) http.HandlerFunc
```

### AD-4: Tenant-Scoped GORM Callbacks

**Decision**: When services migrate to atlas-database, they gain automatic `WHERE tenant_id = ?` injection.

**Impact**: Services that already manually filter by `tenant_id` in their provider queries will effectively have the filter applied twice. This is safe (idempotent) but redundant. Services can optionally clean up explicit tenant filters after migration, but it's not required for correctness.

### AD-5: Database Migration Does Not Change Entity Code

**Decision**: Entity structs, `Migration()`, `Make()`, and entity-level provider functions stay in each service.

**Rationale**: These are domain-specific. Only the infrastructure code (connection, transaction, retry) is extracted.

## Service Classification

### By Handler Variant

**No DB, with tenant (30 services)**:
atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs, atlas-chalkboards, atlas-channel,
atlas-character, atlas-character-factory, atlas-consumables, atlas-drops, atlas-expressions,
atlas-guilds, atlas-inventory, atlas-invites, atlas-keys, atlas-login, atlas-maps,
atlas-marriages, atlas-messages, atlas-messengers, atlas-monsters, atlas-npc-shops,
atlas-parties, atlas-pets, atlas-portals, atlas-query-aggregator, atlas-reactors,
atlas-skills, atlas-storage, atlas-world

*Note: Some of these have a `*gorm.DB` in the REST layer but pass it through to providers.*

**With DB in HandlerDependency (15 services)**:
atlas-account, atlas-ban, atlas-data, atlas-drop-information, atlas-fame,
atlas-gachapons, atlas-map-actions, atlas-monster-death, atlas-notes,
atlas-npc-conversations, atlas-npc-shops, atlas-party-quests, atlas-pets,
atlas-portal-actions, atlas-quest, atlas-reactor-actions

**No tenant parsing (2 services)**:
atlas-tenants, atlas-configurations

### By request.go Subset

**GET only (4 services)**: atlas-account, atlas-fame, atlas-rates, atlas-asset-expiration
**GET+POST only (5 services)**: atlas-portal-actions, atlas-reactor-actions, atlas-map-actions, atlas-effective-stats, atlas-monster-death
**GET+POST+DELETE (2 services)**: atlas-gachapons, atlas-party-quests
**GET+POST+PATCH+DELETE (35 services)**: majority
**GET+POST+PUT+PATCH+DELETE (1 service)**: atlas-saga-orchestrator

### By Database Usage

**With database (29 services)**:
atlas-account, atlas-ban, atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs,
atlas-chalkboards, atlas-character, atlas-character-factory, atlas-configurations,
atlas-consumables, atlas-data, atlas-drop-information, atlas-drops, atlas-expressions,
atlas-fame, atlas-families, atlas-gachapons, atlas-guilds, atlas-inventory, atlas-invites,
atlas-keys, atlas-maps, atlas-marriages, atlas-messages, atlas-notes, atlas-npc-conversations,
atlas-parties, atlas-quest

**Without database (non-exhaustive)**: atlas-rates, atlas-asset-expiration, atlas-effective-stats,
atlas-login, atlas-channel, atlas-world, atlas-saga-orchestrator, atlas-tenants, etc.

## Dependencies Between Phases

```
Phase 1 (requests) ──────────────────────────────────────→ Phase 6 (full migration)
Phase 2 (handler types) ─→ Phase 3 (RegisterHandler) ──→ Phase 6
                                                          Phase 4 (ID parsers) ──→ Phase 6
Phase 5 (atlas-database) ────────────────────────────────→ Phase 6
```

Phases 1, 2, 4, and 5 can run in parallel. Phase 3 depends on Phase 2. Phase 6 depends on all others.

## go.work References

All libraries are already in the workspace:
```
libs/atlas-constants
libs/atlas-database
libs/atlas-kafka
libs/atlas-model
libs/atlas-redis
libs/atlas-rest
libs/atlas-script-core
libs/atlas-socket
libs/atlas-tenant
```

No new workspace entries needed.
