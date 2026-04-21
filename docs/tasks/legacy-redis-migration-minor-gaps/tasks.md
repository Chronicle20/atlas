# Redis Migration Minor Gaps — Tasks

**Last Updated: 2026-02-19**

## Phase 1: atlas-messengers — Fix Deadlock + Distributed Lock ✅ COMPLETE

- [x] **1.1** Fix deadlock: extract `createCore`/`createCoreBuffered` from `Create`/`CreateAndEmit` in `processor.go`. Public functions acquire lock + call internal. `RequestInvite`/`RequestInviteAndEmit` call internal variant (already hold lock).
- [x] **1.2** Replace `createAndJoinLock` (`sync.RWMutex`) with Redis distributed lock using `atlas-redis` `Lock` type. Lock key: `messenger-create:{tenantKey}:{characterId}`. Add spin-wait retry (50ms interval, 3s timeout). Update all 4 call sites.
- [x] **1.3** Remove `sync` import from `processor.go`.
- [x] **1.4** Update test helpers: `setupTestRegistry` now calls `InitLock(rc)` for miniredis.
- [x] **1.5** Run `go test ./... -count=1` — all pass.
- [x] **1.6** Run `go build` — succeeds.

## Phase 2: atlas-npc-shops — Redis-Backed Consumable Cache ✅ COMPLETE

- [x] **2.1** Add `MarshalJSON`/`UnmarshalJSON` to `consumable.Model` in new file `data/consumable/model_json.go`. Private `jsonModel`, `jsonSummon`, `jsonReward` structs handle all unexported fields.
- [x] **2.2** Add round-trip serialization tests in `data/consumable/model_json_test.go`: full model, empty model, slice of models.
- [x] **2.3** Replace `ConsumableCache` in `shops/cache.go`: removed `sync.Once` + `sync.RWMutex` + `map`. Uses direct `goredis.Client` GET/SET with JSON serialization. Lazy-load-on-miss: check Redis → if miss, REST call → populate Redis.
- [x] **2.4** Add `InitConsumableCache(client *goredis.Client)` function. Updated `main.go` to call it after `shops.InitRegistry(rc)`.
- [x] **2.5** Kept `SetConsumableCacheForTesting` and `GetConsumableCache` for backward compatibility with existing tests using `mockConsumableCache`.
- [x] **2.6** Add cache tests in `shops/cache_test.go`: set+get round-trip, tenant isolation, empty on miss. Uses miniredis.
- [x] **2.7** Verified existing tests pass — all 6 test packages pass.
- [x] **2.8** Run `go test ./... -count=1` — all pass.
- [x] **2.9** Run `go build` — succeeds.
