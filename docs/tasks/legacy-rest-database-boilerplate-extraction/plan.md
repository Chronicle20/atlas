# REST/Database Boilerplate Extraction Plan

Last Updated: 2026-02-19

## Executive Summary

47 `rest/handler.go` and 47 `rest/request.go` files are copy-pasted across services with minor variations. Additionally, 29 services duplicate `database/connection.go`, `database/provider.go`, `database/transaction.go`, and 30 services duplicate `retry/retry.go`. This totals approximately **10,000+ lines of duplicated code**.

The `libs/atlas-rest` library already provides low-level HTTP client/server primitives, and `libs/atlas-database` already contains the consolidated database connection/transaction code but **zero services import it yet**.

This plan extracts the remaining boilerplate into the two existing shared libraries and migrates all services to use them.

## Current State Analysis

### REST Layer Duplication (47 services)

**handler.go** contains three categories of duplicated code:

| Component | Copies | Identical? | Location Target |
|-----------|--------|------------|-----------------|
| `HandlerContext` struct + accessor | 47 | 100% | atlas-rest/server |
| `GetHandler` type | 47 | 100% | atlas-rest/server |
| `InputHandler[M]` type | 47 | 100% | atlas-rest/server |
| `ParseInput[M]()` | 47 | 100% | atlas-rest/server |
| `HandlerDependency` (no DB) | 32 | 100% within group | atlas-rest/server |
| `HandlerDependency` (with DB) | 15 | 100% within group | atlas-rest/server |
| `RegisterHandler` (no DB, with tenant) | 30 | 100% within group | atlas-rest/server |
| `RegisterHandler` (with DB, with tenant) | 15 | 100% within group | atlas-rest/server |
| `RegisterHandler` (no DB, no tenant) | 2 | 100% within group | atlas-rest/server |
| `RegisterInputHandler` (mirrors above 3) | 47 | 100% within group | atlas-rest/server |
| ID parser functions | ~100+ total | Same pattern | atlas-rest/server |

**request.go** contains thin wrappers that add Span+Tenant headers to outbound HTTP calls:

| Method | Copies | Identical? |
|--------|--------|------------|
| `MakeGetRequest[A]` | 47 | 100% |
| `MakePostRequest[A]` | ~43 | 100% |
| `MakePatchRequest[A]` | ~40 | 100% |
| `MakeDeleteRequest` | ~39 | 100% |
| `MakePutRequest[A]` | 1 | N/A |

### Database Layer Duplication (29 services)

| Component | Copies | Identical? | Already in atlas-database? |
|-----------|--------|------------|---------------------------|
| `connection.go` (~167 lines) | 29 | ~98% (pool defaults vary) | YES |
| `EntityProvider` type | 29 | 100% | YES |
| `transaction.go` | ~24 | 95% (1 simpler variant) | YES |
| `retry/retry.go` (26 lines) | 30 | 100% | YES (inlined) |
| `Query[E]` / `SliceQuery[E]` helpers | ~18 | 100% within group | NO |
| `FoldModelProvider` helper | ~8 | 100% within group | NO |

### Key Insight: atlas-database Is Ready But Unused

`libs/atlas-database` already has the consolidated `Connect()`, `ExecuteTransaction()`, `EntityProvider`, automatic tenant scoping via GORM callbacks, and inlined retry. **Zero services import it.** The database migration is purely a rewiring task.

## Proposed Future State

### atlas-rest (enhanced)

New additions to `libs/atlas-rest/server/`:

```
server/
  handler.go       (existing: RetrieveSpan, ParseTenant)
  response.go      (existing: Marshal, MarshalResponse)
  context.go       (NEW: HandlerContext, GetHandler, InputHandler, ParseInput)
  dependency.go    (NEW: HandlerDependency + variants, RegisterHandler + variants)
  id_parsers.go    (NEW: generic ParseUint32Id, ParseUUIDId, ParseStringId, ParseTypedId)
```

New additions to `libs/atlas-rest/requests/`:

```
requests/
  get.go, post.go, etc.  (existing)
  decorated.go           (NEW: MakeGetRequest, MakePostRequest, etc. with Span+Tenant decorators)
```

### atlas-database (no code changes needed)

Already complete. Services just need to switch imports.

### Per-Service (after migration)

Each service's `rest/` package shrinks to:
- **handler.go**: Only service-specific ID parsers (calling generic helpers) and the router initialization
- **request.go**: DELETED entirely (replaced by atlas-rest/requests)
- **resource.go** / domain handlers: Unchanged

Each service's `database/` package shrinks to:
- **connection.go**: DELETED (replaced by atlas-database)
- **provider.go**: DELETED or minimal service-specific helpers only
- **transaction.go**: DELETED (replaced by atlas-database)
- **retry/retry.go**: DELETED (inlined in atlas-database)

## Implementation Phases

### Phase 1: Extract request.go into atlas-rest (Effort: S)

The 47 request.go files are 100% identical wrappers. Add decorated request functions to atlas-rest that bake in Span+Tenant header decorators.

**Why first**: Zero risk, no API changes needed, purely additive to the library.

### Phase 2: Extract handler.go common types into atlas-rest (Effort: M)

Move `HandlerContext`, `GetHandler`, `InputHandler[M]`, `ParseInput[M]` into atlas-rest/server. These are 100% identical across all 47 services.

### Phase 3: Extract RegisterHandler variants into atlas-rest (Effort: M)

The three variants (no-DB/with-tenant, with-DB/with-tenant, no-DB/no-tenant) and their InputHandler mirrors need a clean API design. Options:

**Option A: Three separate functions**
```go
server.RegisterHandler(l)(si)(name, handler)           // no DB, with tenant
server.RegisterDBHandler(l)(db)(si)(name, handler)     // with DB, with tenant
server.RegisterSimpleHandler(l)(si)(name, handler)     // no DB, no tenant
```

**Option B: Functional options**
```go
server.RegisterHandler(l, server.WithDB(db), server.WithTenant())(si)(name, handler)
```

**Recommended: Option A** — matches existing curried patterns, no refactoring needed in services, just import replacement.

### Phase 4: Extract generic ID parsers into atlas-rest (Effort: M)

Create reusable ID parser generators:

```go
// Generic parsers that services compose
server.ParseUint32Id(varName string, next func(uint32) http.HandlerFunc) http.HandlerFunc
server.ParseUUIDId(varName string, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc
server.ParseStringId(varName string, next func(string) http.HandlerFunc) http.HandlerFunc
server.ParseTypedId[T ~uint32|~int32|~int8|~uint16](varName string, next func(T) http.HandlerFunc) http.HandlerFunc
```

Services then define thin wrappers:
```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
    return server.ParseUint32Id(l, "characterId", next)
}
```

### Phase 5: Migrate services to atlas-database (Effort: L)

For each of the 29 database-using services:
1. Replace local `database.Connect()` with `database.Connect()` from atlas-database
2. Replace local `database.EntityProvider` with import from atlas-database
3. Replace local `database.ExecuteTransaction` with import from atlas-database
4. Delete local `database/connection.go`, `database/transaction.go`, `retry/retry.go`
5. Add `Query[E]`/`SliceQuery[E]` to atlas-database if needed, or keep service-local

### Phase 6: Migrate services to new atlas-rest functions (Effort: XL)

For each of the 47 services:
1. Delete `rest/request.go`, import decorated requests from atlas-rest
2. Replace handler types/ParseInput with atlas-rest/server imports
3. Replace RegisterHandler with the appropriate atlas-rest variant
4. Replace ID parser boilerplate with generic helpers
5. Test and build each service

## Detailed Task Breakdown

### Phase 1: Decorated Requests in atlas-rest

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 1.1 | Add `requests.DecoratedGet[A]`, `DecoratedPost[A]`, `DecoratedPatch[A]`, `DecoratedDelete`, `DecoratedPut[A]` to atlas-rest | S | Functions exist, add Span+Tenant headers automatically |
| 1.2 | Add unit tests for decorated request functions | S | Tests pass |
| 1.3 | Migrate 5 pilot services (simple ones: atlas-rates, atlas-fame, atlas-account, atlas-asset-expiration, atlas-families) | S | Services build, tests pass, request.go deleted |
| 1.4 | Migrate remaining 42 services | M | All request.go files deleted |

### Phase 2: Handler Types in atlas-rest

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 2.1 | Add `server.HandlerContext`, `server.GetHandler`, `server.InputHandler[M]`, `server.ParseInput[M]` to atlas-rest | S | Types exported from atlas-rest/server |
| 2.2 | Ensure GORM is NOT a dependency of atlas-rest (HandlerDependency with DB must use interface or separate package) | S | atlas-rest go.mod has no gorm dependency |
| 2.3 | Add unit tests | S | Tests pass |

### Phase 3: RegisterHandler in atlas-rest

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 3.1 | Design HandlerDependency to support optional DB without importing gorm | M | Interface-based or separate sub-package |
| 3.2 | Implement `RegisterHandler`, `RegisterDBHandler`, `RegisterSimpleHandler` + InputHandler variants | M | All 3 variants functional |
| 3.3 | Add unit tests | S | Tests pass |

### Phase 4: Generic ID Parsers

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 4.1 | Implement `ParseUint32Id`, `ParseUUIDId`, `ParseStringId`, `ParseTypedId` in atlas-rest/server | S | Generic parsers work with mux vars |
| 4.2 | Add unit tests with mux router | S | Tests pass for all ID types |

### Phase 5: atlas-database Migration

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 5.1 | Add `Query[E]`, `SliceQuery[E]` to atlas-database if used by >5 services | S | Functions available in atlas-database |
| 5.2 | Migrate 3 pilot services (atlas-account, atlas-ban, atlas-keys) | M | Services build, tests pass, local database/ packages trimmed |
| 5.3 | Migrate remaining 26 database services | L | All services using atlas-database for connection/transaction |
| 5.4 | Delete all `retry/retry.go` copies | S | Zero local retry packages remain |

### Phase 6: Full atlas-rest Migration

| # | Task | Effort | Acceptance Criteria |
|---|------|--------|-------------------|
| 6.1 | Migrate 5 pilot services to new atlas-rest handler types | M | Services build, handler.go significantly smaller |
| 6.2 | Migrate remaining 42 services | XL | All services using atlas-rest for handler registration |
| 6.3 | Clean up any remaining dead code | S | No unused imports or functions |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **GORM dependency in atlas-rest** | High | Medium | Use interface for DB access in HandlerDependency, keep gorm out of atlas-rest |
| **Breaking changes during migration** | Medium | High | Pilot with 3-5 simple services first, maintain backward compatibility |
| **Circular dependency** | Low | High | atlas-rest depends on atlas-tenant and atlas-model; atlas-database depends on atlas-tenant and atlas-model; no circular risk |
| **go.work resolution issues** | Low | Medium | All libs already in go.work; test with `go build` from workspace root |
| **Tenant-scoped callbacks break existing queries** | Medium | High | atlas-database auto-tenant callbacks may change query behavior; test thoroughly in pilot phase |
| **ID parser type constraints** | Low | Low | Go generics support `~uint32|~int32|~uint16|~int8` constraints; atlas-constants types work |

## Success Metrics

| Metric | Before | After |
|--------|--------|-------|
| Duplicated LoC (REST) | ~5,000 | ~0 (replaced by ~200 lines in atlas-rest) |
| Duplicated LoC (Database) | ~5,000 | ~0 (replaced by existing atlas-database) |
| `rest/request.go` files | 47 | 0 |
| `database/connection.go` files | 29 | 0 |
| `retry/retry.go` files | 30 | 0 |
| Avg service `rest/handler.go` size | ~150 lines | ~30-50 lines (only ID parsers + router init) |

## Dependencies

- `libs/atlas-rest` — already in go.work, already imported by all services
- `libs/atlas-database` — already in go.work, NOT yet imported by any service
- `libs/atlas-model` — already in go.work, already imported by all services
- `libs/atlas-tenant` — already in go.work, already imported by most services
- No new libraries needed

## Recommended Execution Order

1. **Phase 1** (requests) — lowest risk, highest immediate impact
2. **Phase 5** (atlas-database) — library already built, just rewiring
3. **Phase 2** (handler types) — additive to atlas-rest
4. **Phase 3** (RegisterHandler) — requires design decision on DB dependency
5. **Phase 4** (ID parsers) — additive, optional (services can keep local parsers)
6. **Phase 6** (full migration) — depends on phases 1-4

Total estimated effort: **~2-3 weeks of focused migration work**, heavily parallelizable across services.
