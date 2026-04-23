# Deprecated Code Migration — Context

Last Updated: 2026-02-19

## Key Files

### Deprecated Function Definitions

| Deprecated Item | Definition File | Line |
|----------------|----------------|------|
| `server.Marshal` | `libs/atlas-rest/server/response.go` | 11-21 |
| `server.MarshalResponse` (replacement) | `libs/atlas-rest/server/response.go` | 24+ |
| `xml.Read` | `services/atlas-data/atlas.com/data/xml/reader.go` | 14-21 |
| `xml.FromPathProvider` (replacement) | `services/atlas-data/atlas.com/data/xml/reader.go` | 23+ |
| `PetModelDecorator` | `services/atlas-channel/atlas.com/channel/character/processor.go` | 26 (interface), 67-80 (impl) |
| `PetAssetEnrichmentDecorator` (potential replacement) | `services/atlas-channel/atlas.com/channel/character/processor.go` | (same file) |
| `AwardInventory` constant | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` | 402 |
| `AwardInventory` compat wrapper | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` | 886 |
| `ItemId` deprecated field | `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` | 77 |
| `SetItemId` deprecated method | `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` | 196 |
| `io/ioutil.ReadFile` usage | `services/atlas-account/atlas.com/account/configuration/loader.go` | (top-level) |
| `rand.Seed` usage | `services/atlas-channel/atlas.com/channel/tool/uint128_test.go` | 48 |
| `rand.Seed` usage | `services/atlas-monster-death/atlas.com/monster/monster/processor_test.go` | 34 |
| `lib/pq` usage | `services/atlas-gachapons/atlas.com/gachapons/gachapon/administrator.go` | 7 |
| `lib/pq` usage | `services/atlas-gachapons/atlas.com/gachapons/gachapon/entity.go` | 5 |

### AwardInventory Constant Copies (all need removal in Phase 4b)

| File | Line |
|------|------|
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` | 402 |
| `services/atlas-character-factory/atlas.com/character-factory/saga/model.go` | 114 |
| `services/atlas-messages/atlas.com/messages/saga/model.go` | 97 |
| `libs/atlas-script-core/saga/model.go` | 99 |

### AwardInventory Producer Call Sites (Phase 4a targets)

| Service | File | Line |
|---------|------|------|
| atlas-quest | `saga/producer.go` | 57 |
| atlas-messages | `character/inventory/commands.go` | 93 |
| atlas-npc-conversations | `operation_executor.go` | 853 |
| atlas-npc-conversations | `processor.go` | 720 |

### server.Marshal Call Sites (Phase 3 targets)

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

---

## Key Decisions

### 1. InMemoryCache in saga-orchestrator is NOT deprecated

The `InMemoryCache` in `saga/cache.go` is retained intentionally as a test-only fallback behind the `Cache` interface. Production uses `PostgresStore` set at startup via `SetCache()`. No action needed.

### 2. Configuration registries are NOT migration targets

In-memory configuration registries in atlas-login, atlas-channel, atlas-world, atlas-character-factory, atlas-cashshop are read-once-at-startup caches. They don't need Redis migration — they are small, immutable, and local.

### 3. atlas-data document registry is NOT a migration target

The generic document registry (`document/registry.go`) caches WZ/XML game data files parsed at startup. This data is read-only and massive — Redis would add latency with no benefit.

### 4. AwardInventory requires phased rollout

Cannot remove `AwardInventory` from the orchestrator until all producers are migrated AND in-flight sagas have drained. The orchestrator's backward-compatibility wrapper (`handleAwardInventory` → `handleAwardAsset`) must remain during the transition.

### 5. server.Marshal migration requires `r *http.Request` in scope

The replacement `MarshalResponse` needs `r.URL.Query()` for sparse fieldsets. All call sites in resource handler functions already have `r` available since they're HTTP handler closures.

### 6. PetModelDecorator needs investigation before removal

The relationship between `PetModelDecorator` (loads pet models) and `PetAssetEnrichmentDecorator` (enriches pet assets) needs clarification. They appear alongside each other in most call sites, suggesting they serve different purposes. Investigation required before assuming one subsumes the other.

---

## Dependencies

### Internal Dependencies

- Phase 4b depends on Phase 4a being deployed
- Phase 3 final step (delete `server.Marshal`) depends on all 11 services being migrated
- Phase 1.3 final step (delete `xml.Read`) depends on all 7 call sites being migrated

### External Dependencies

- None — all changes are internal to the Atlas workspace

### Related Dev Plans

- `docs/tasks/legacy-redis-registry-migration/` — high-throughput registry migration (separate track)
- `docs/tasks/legacy-atlas-fame-remediation/` — references "legacy functions deprecated/removed"
- `docs/tasks/legacy-atlas-invites-remediation/` — references deprecated custom REST handlers
- `docs/architectural-improvements.md` — tracks overall architecture health

---

## Testing Strategy

| Phase | Test Approach |
|-------|--------------|
| Phase 1 (Quick Wins) | `go test ./... -count=1` in each affected service, then `go build` |
| Phase 2.1 (lib/pq) | Focus on gachapon array column operations; `go test ./... -count=1` |
| Phase 2.2 (PetModelDecorator) | Existing processor tests + verify pet enrichment in decorator chain |
| Phase 3 (server.Marshal) | Per-service `go test ./... -count=1` + `go build`; final workspace-wide build |
| Phase 4a (AwardInventory producers) | Existing saga integration tests; verify AwardAsset sagas execute correctly |
| Phase 4b (AwardInventory cleanup) | DB query: `SELECT count(*) FROM sagas WHERE saga_data::text LIKE '%award_inventory%' AND status != 'completed'` must return 0 before proceeding |

---

## Verification Commands

```bash
# Check for remaining deprecated usage (run after all phases complete)
grep -r 'server\.Marshal[^R]' services/ libs/     # Phase 3
grep -r 'xml\.Read\b' services/                    # Phase 1.3
grep -r 'PetModelDecorator' services/              # Phase 2.2
grep -r 'AwardInventory\|award_inventory' services/ libs/  # Phase 4
grep -r 'ioutil\.' services/                       # Phase 1.1
grep -r 'rand\.Seed' services/                     # Phase 1.2
grep -r 'lib/pq' services/*/go.mod                 # Phase 2.1
grep -r 'SetItemId\|"itemId"' services/atlas-query-aggregator/  # Phase 1.4

# Workspace-wide build
cd /home/tumidanski/source/pers/atlas && go build ./...
```
