# Deprecated Code Migration Plan

Last Updated: 2026-02-19

## Executive Summary

The Atlas codebase contains 9 distinct categories of deprecated code across services and libraries. This plan provides a structured approach to eliminate all deprecated API usage, bringing the codebase to a clean, consistent state. The work is organized into 4 phases by priority and blast radius, from quick wins to cross-service coordination.

**Scope:** 13 discrete migration items across ~20 services, affecting ~80+ call sites total.

**Out of Scope:** High-throughput Redis registry migration (atlas-monsters, atlas-maps, atlas-channel sessions, atlas-login sessions, atlas-party-quests instances) — these are tracked separately in `docs/tasks/legacy-redis-registry-migration/` and `docs/high-throughput-cache-problem.md`.

---

## Current State Analysis

### Deprecated Items Inventory

| # | Item | Location | Replacement | Call Sites | Severity |
|---|------|----------|-------------|------------|----------|
| 1 | `server.Marshal` | libs/atlas-rest | `server.MarshalResponse` | 27 across 11 services | Medium |
| 2 | `xml.Read` | services/atlas-data | `xml.FromPathProvider` | 7 in 1 service | Low |
| 3 | `PetModelDecorator` | services/atlas-channel | Subsume into existing decorators | 8+ in 1 service | Medium-High |
| 4 | `AwardInventory` action | 5 services | `AwardAsset` | 50+ references | High |
| 5 | `ItemId` / `SetItemId` (validation) | atlas-query-aggregator | `ReferenceId` / `SetReferenceId` | Shim in place | Low |
| 6 | `io/ioutil.ReadFile` | atlas-account | `os.ReadFile` | 1 call site | Low |
| 7 | `rand.Seed` | atlas-channel, atlas-monster-death tests | Remove (no-op since Go 1.20) | 2 call sites | Low |
| 8 | `lib/pq` driver | atlas-gachapons | `jackc/pgx/v5` via GORM | 2 files | Low |
| 9 | `InMemoryCache` (saga) | atlas-saga-orchestrator | `PostgresStore` (already default) | Test-only fallback | None |

### Root Causes

- **Incremental evolution**: New APIs added alongside old ones without forced migration (Marshal → MarshalResponse, AwardInventory → AwardAsset)
- **Interface inertia**: PetModelDecorator is on a processor interface — changing it requires coordinated mock/test updates
- **Cross-service constants**: AwardInventory flows through Kafka as a string value, requiring coordinated multi-service rollout
- **Go version drift**: io/ioutil and rand.Seed deprecated in Go 1.16/1.20 but not caught by linters

---

## Proposed Future State

1. All services use `server.MarshalResponse` — supports sparse fieldsets via query params
2. `xml.Read` removed — all callers use `xml.FromPathProvider` functional style
3. `PetModelDecorator` removed from the Processor interface — pet enrichment consolidated
4. All saga producers emit `AwardAsset` — backward compat wrapper removed from orchestrator
5. `ItemId` field and `SetItemId` method removed from query-aggregator validation
6. Zero deprecated Go stdlib usage (`io/ioutil`, `rand.Seed`)
7. atlas-gachapons uses GORM/pgx exclusively (no `lib/pq`)
8. `server.Marshal` function deleted from libs/atlas-rest

---

## Implementation Phases

### Phase 1: Quick Wins (No Cross-Service Impact)

Low-risk, single-file or single-service changes. Can be done independently.

#### 1.1 Replace `io/ioutil.ReadFile` in atlas-account
- **File**: `services/atlas-account/atlas.com/account/configuration/loader.go`
- **Change**: `ioutil.ReadFile("config.yaml")` → `os.ReadFile("config.yaml")`, remove `"io/ioutil"` import
- **Effort**: S
- **Risk**: None — drop-in replacement since Go 1.16

#### 1.2 Remove `rand.Seed` from test files
- **Files**:
  - `services/atlas-channel/atlas.com/channel/tool/uint128_test.go:48`
  - `services/atlas-monster-death/atlas.com/monster/monster/processor_test.go:34`
- **Change**: Delete `rand.Seed(...)` lines, remove `"math/rand"` import if unused
- **Effort**: S
- **Risk**: None — no-op since Go 1.20; removing doesn't change behavior

#### 1.3 Migrate `xml.Read` → `xml.FromPathProvider` in atlas-data
- **Files** (7 call sites in 6 files):
  - `services/atlas-data/atlas.com/data/npc/string_registry.go:43`
  - `services/atlas-data/atlas.com/data/map/string_registry.go:40`
  - `services/atlas-data/atlas.com/data/item/string_registry.go:33,70`
  - `services/atlas-data/atlas.com/data/monster/string_registry.go:35`
  - `services/atlas-data/atlas.com/data/monster/gauge_registry.go:35`
  - `services/atlas-data/atlas.com/data/skill/string_registry.go:35`
- **Pattern**: Replace `exml, err := xml.Read(path)` with `exml, err := xml.FromPathProvider(path)()`
- **Then**: Delete `xml.Read` function from `xml/reader.go`
- **Effort**: S
- **Risk**: Low — `xml.Read` is just a wrapper around `FromPathProvider`

#### 1.4 Remove `ItemId` deprecation shim from query-aggregator
- **File**: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go`
- **Change**:
  1. Remove `ItemId` field from `ConditionInput` struct (line 77)
  2. Remove `SetItemId` method from `ConditionBuilder` (line 196)
  3. Remove the `else if input.ItemId != 0` fallback branch (lines 229-231)
  4. Update any tests referencing `ItemId`
- **Effort**: S
- **Risk**: Low — any external callers sending `itemId` in JSON will silently ignore it. The field has been deprecated with a replacement (`ReferenceId`) already in use.

---

### Phase 2: Single-Service Refactors

Changes contained within a single service but requiring more careful coordination.

#### 2.1 Migrate `lib/pq` → GORM/pgx in atlas-gachapons
- **Files**:
  - `services/atlas-gachapons/atlas.com/gachapons/gachapon/administrator.go` — imports `lib/pq`
  - `services/atlas-gachapons/atlas.com/gachapons/gachapon/entity.go` — imports `lib/pq`
  - `services/atlas-gachapons/atlas.com/gachapons/go.mod` — depends on `lib/pq`
- **Change**: Replace direct SQL with GORM operations or use `pq.Array` → `pgx` equivalent. Audit how `lib/pq` is being used (likely `pq.Int64Array` or `pq.StringArray` for PostgreSQL array columns).
- **Effort**: M
- **Risk**: Medium — need to verify array column handling works identically with pgx
- **Acceptance**: `go test ./... -count=1` passes, `go build` succeeds, no `lib/pq` imports remain

#### 2.2 Remove `PetModelDecorator` from atlas-channel
- **Files**:
  - `services/atlas-channel/atlas.com/channel/character/processor.go` — interface definition + implementation
  - `services/atlas-channel/atlas.com/channel/character/mock/processor.go` — mock
  - `services/atlas-channel/atlas.com/channel/character/processor_test.go` — tests
  - 6 consumer/handler files calling `cp.PetModelDecorator`
- **Analysis needed**: Examine whether `PetAssetEnrichmentDecorator` (already used alongside PetModelDecorator in many call sites) fully subsumes its functionality, or whether pet model enrichment needs to be merged into it.
- **Change**:
  1. Determine if `PetAssetEnrichmentDecorator` already loads pets (making PetModelDecorator redundant)
  2. If yes: remove `PetModelDecorator` from all decorator chains and from the interface
  3. If no: merge pet-loading logic into `PetAssetEnrichmentDecorator`
  4. Update mock and tests
- **Effort**: M
- **Risk**: Medium — pet data must still be enriched; need to verify no functionality is lost
- **Acceptance**: All tests pass; pet data correctly enriched in all game scenarios

---

### Phase 3: Cross-Service Migration (`server.Marshal` → `MarshalResponse`)

Mechanical but wide-reaching change across 11 services.

#### 3.1 Migrate all `server.Marshal` call sites to `server.MarshalResponse`

**Signature change:**
```go
// Old (deprecated):
server.Marshal[T](logger)(w)(serverInfo)(data)

// New:
server.MarshalResponse[T](logger)(w)(serverInfo)(r.URL.Query())(data)
```

The key difference: `MarshalResponse` accepts `queryParams map[string][]string` for sparse fieldset filtering. For migration, pass `r.URL.Query()` from the HTTP request.

**Services to update (27 call sites):**

| Service | File | Call Sites |
|---------|------|------------|
| atlas-buddies | `list/resource.go` | 2 |
| atlas-parties | `party/resource.go` | 4 |
| atlas-maps | `map/resource.go`, `map/weather/resource.go` | 2 |
| atlas-monsters | `world/resource.go`, `monster/resource.go` | 3 |
| atlas-messengers | `messenger/resource.go` | 4 |
| atlas-quest | `quest/resource.go` | 7 |
| atlas-reactors | `reactor/resource.go` | 3 |
| atlas-character | `character/rest_test.go` | 1 |
| atlas-keys | `character/resource.go` | 1 |

**Approach**: Migrate one service at a time. Each service gets its own build+test cycle.

**After all migrations**:
- Delete `server.Marshal` function from `libs/atlas-rest/server/response.go`
- Run full workspace build to confirm no remaining callers

- **Effort**: L (mechanical but many files)
- **Risk**: Low — `MarshalResponse` is a superset; passing `r.URL.Query()` is equivalent to current behavior
- **Acceptance**: All services build and pass tests. `server.Marshal` function deleted. `grep -r 'server\.Marshal[^R]' services/` returns no results.

---

### Phase 4: Cross-Service Coordination (`AwardInventory` → `AwardAsset`)

The highest-risk migration. `AwardInventory` is a runtime string value (`"award_inventory"`) that flows through Kafka messages and is persisted in the saga PostgreSQL store.

#### 4.1 Phase 4a — Migrate all producers to emit `AwardAsset`

**Producer services (create sagas with AwardInventory):**
- `services/atlas-quest/atlas.com/quest/saga/producer.go:57`
- `services/atlas-messages/atlas.com/messages/character/inventory/commands.go:93`
- `services/atlas-npc-conversations/atlas.com/npc/operation_executor.go:853`
- `services/atlas-npc-conversations/atlas.com/npc/processor.go:720`

**Change**: Replace `AwardInventory` → `AwardAsset` in all producer call sites.

**The orchestrator already handles both** — `handleAwardInventory` wraps `handleAwardAsset` (handler.go:886), so the orchestrator is already forward-compatible.

- **Effort**: M
- **Risk**: Low (orchestrator already handles AwardAsset)

#### 4.2 Phase 4b — Remove backward compatibility from orchestrator

**Only after all producers are migrated AND no in-flight sagas use AwardInventory.**

Wait period: Deploy Phase 4a, then wait for the stale saga reaper timeout (configurable) to ensure all existing `AwardInventory` sagas have completed or been compensated.

**Changes in atlas-saga-orchestrator:**
- Remove `AwardInventory` constant from `saga/model.go`
- Remove `handleAwardInventory` wrapper from `handler.go`
- Remove `AwardInventory` case from model matching and REST unmarshal
- Update tests to use `AwardAsset` exclusively

**Changes in other services:**
- Remove `AwardInventory` constant from `atlas-character-factory/saga/model.go`
- Remove `AwardInventory` constant from `atlas-messages/saga/model.go`
- Remove `AwardInventory` constant from `libs/atlas-script-core/saga/model.go`

- **Effort**: M
- **Risk**: Medium — must ensure zero in-flight `AwardInventory` sagas exist before removing
- **Acceptance**: `grep -r 'AwardInventory\|award_inventory' services/ libs/` returns no results

---

## Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| PetModelDecorator removal breaks pet enrichment | High | Medium | Analyze PetAssetEnrichmentDecorator coverage first; add integration test |
| AwardInventory in-flight sagas fail after removal | High | Low | Wait period between 4a and 4b; verify via DB query |
| lib/pq → pgx breaks array column handling | Medium | Medium | Write focused tests for gachapon array operations before migrating |
| server.Marshal removal misses a call site | Low | Low | Workspace-wide `go build` after deletion catches any remaining callers |
| ItemId removal breaks external callers | Low | Low | Field is `omitempty`; removal just silently ignores it in JSON unmarshal |

---

## Success Metrics

1. **Zero deprecated function calls** — `grep -r 'server\.Marshal[^R]\|xml\.Read\|PetModelDecorator\|AwardInventory\|ioutil\.\|rand\.Seed' services/ libs/` returns no results
2. **All services build** — `go build` succeeds for all 56 workspace modules
3. **All tests pass** — `go test ./... -count=1` succeeds for all affected services
4. **Dead code removed** — `server.Marshal`, `xml.Read`, `PetModelDecorator`, `AwardInventory` constants, `SetItemId` method all deleted
5. **No lib/pq dependency** — `grep -r 'lib/pq' services/*/go.mod` returns no results

---

## Required Resources and Dependencies

- **Go workspace**: All changes are within the existing `go.work` workspace
- **No new dependencies** — all replacements use existing APIs
- **Database access**: Phase 4b requires querying saga table for in-flight `AwardInventory` sagas before removal
- **Test infrastructure**: Existing miniredis + SQLite test setup is sufficient

---

## Timeline Estimates

| Phase | Estimated Effort | Dependencies |
|-------|-----------------|--------------|
| Phase 1: Quick Wins | S (4 items, ~30 min each) | None |
| Phase 2: Single-Service | M (2 items, ~1-2 hours each) | None |
| Phase 3: server.Marshal | L (11 services, ~15-30 min each) | None |
| Phase 4a: AwardInventory producers | M (~1 hour) | None |
| Phase 4b: AwardInventory cleanup | M (~1 hour) | Phase 4a deployed + wait period |

Phases 1-3 and 4a can be executed in parallel. Phase 4b requires Phase 4a to be deployed and operational.
