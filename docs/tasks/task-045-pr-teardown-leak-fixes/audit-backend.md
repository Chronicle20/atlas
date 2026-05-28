# Backend Guidelines Audit — task-045-pr-teardown-leak-fixes

- **Scope:** changed Go packages on `task-045-pr-teardown-leak-fixes` vs base `6815a919e58855eae4b8fa0db51675913740287b`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-27
- **Build:** PASS (libs/atlas-redis build+vet clean)
- **Tests:** PASS (atlas-redis, atlas-monsters/monster, atlas-reactors/reactor verified green; full suite per task verification)
- **Redis key guard:** `./tools/redis-key-guard.sh` exit 0
- **Overall:** PASS (no blocking violations) — 3 Minor observations

This change is a cross-cutting Redis-key-namespacing refactor of registries/caches
(not domain models/handlers), so the relevant guideline surface is the cache pattern
(`patterns-cache.md`), DOM-21 (constant reuse), immutable-model preservation, and test
quality. Standard DOM-01..DOM-20 (builder/ToEntity/Transform/REST/handler) checks do
not apply to these files — none of them are domain `model.go`/`resource.go` packages.

## What I verified

### Lib abstraction correctness (PASS)
- Every new lib type composes keys via `namespacedKey`/`tenantEntityKey`, which prepend
  `keyPrefix` (env-aware via `ATLAS_ENV`): `libs/atlas-redis/keys.go:38-46`,
  `set.go:30,68-70`, `hash.go:20,63`, `keyed_set.go:23,102-104`, `keyed_hash.go:23-25`,
  `keyed_sorted_set.go:23-25`. No new type emits a bare/un-prefixed key.
- `Lock.ReleaseToken` is an atomic compare-and-delete via Lua (`lock.go:15-20,104-111`);
  `AcquireWithToken`/`ForceAcquire` correctly store the token as the value so a holder
  releases only its own lock (`lock.go:84-99`).
- `Registry.Update` optimistic-lock retry is sound: WATCH+GET+fn+TxPipelined(SET),
  retries ONLY on `goredis.TxFailedErr`, bounded at `updateMaxRetries=1000`, returns the
  original error on any non-contention failure, and surfaces a distinct "optimistic lock
  failed after N retries" error when exhausted (`registry.go:74-122`). No API break — it
  is an additive method.

### Lua-script port (atlas-monsters) (PASS)
- `applyDamageScript`/`applyRecoveryScript`/`decayDamageEntriesScript` correctly ported
  to `Registry.Update` closures (`monster/registry.go:427-483,497-542,669-714`). Closures
  are pure in observable effects (captured flags derive only from `cur`), satisfying the
  retry-safety requirement the `Update` doc states.
- Key shape preserved byte-identically vs the pre-migration raw form
  (`monster/registry.go:264-272,291-302`).
- `damageEntryList`/`statusEffectList` custom `UnmarshalJSON` tolerate the Lua-cjson
  empty-object `"{}"` form (`monster/registry.go:53-77`) — JSON round-trip safety handled
  explicitly. This is the correct fix, not the default decoder.

### Bare-key-leak fixes (PASS — all 3 now namespaced)
- atlas-data: old keys `atlas-data:ingest:...` (bare) → `namespacedKey("data-ingest", …)`
  on both writer and reader. Writer suffix `ingestJobKeySuffix` (`rest/jobs.go:46-48`) and
  reader-from-env `ingestJobSuffixFromEnv` (`ingest/heartbeat.go:84-90`) produce matching
  `scope:region:major.minor`; label-based reader `ingestJobKeySuffixFromLabels`
  (`rest/jobs.go:52-58`) produces `scope:region:version` where `version` == `major.minor`.
  Cross-pod key consistency preserved.
- atlas-inventory `lock_registry.go`: old `invlock:%d:%d` (bare) → `atlas.NewLockWithTTL`
  namespace `"inventory"` (`lock_registry.go:54-58`). Transient TTL lock; key-shape change
  acceptable.
- atlas-inventory `reservation_registry.go`: old `reservation:<uuid>:...` (raw client) →
  `TenantRegistry[reservationKey, …]` namespace `"reservation"`
  (`reservation_registry.go:54-63`). `reservationKey` reuses `inventory.Type` from
  atlas-constants — no reinvented type.
- atlas-merchant `teardown.go`: genuine concurrency-bug fix — `waitGroup.Add(1)` hoisted
  out of the goroutine (was racing `Wait()`) (`service/teardown.go:40-41`). Correct.

### DOM-21 (constant/type reuse) (PASS)
- Only new declarations in service code are `ingestJobNamespace` (a namespace string, not
  a domain type) and `reservationKey` (a local composite-key struct over
  `inventory.Type`). No item-id/inventory/world/channel/job/skill type was redeclared.

### Test quality (PASS)
- Every changed `*_test.go` exercises real Redis behavior via `alicebob/miniredis` (25
  files), not mocks. miniredis as a direct test dep in atlas-guilds/atlas-merchant is the
  acceptable, intended addition.

### Guard tool (PASS)
- `tools/rediskeyguard/analyzer.go:36-82` correctly gates on go-redis
  `Client/ClusterClient/Conn/Pipeliner/Tx` receivers, allowlists the lib package itself
  (`:37-39`), and bans exactly the keyed methods used by the migrated code (`:20-27`). It
  does NOT flag `Run`/`Eval`/`Watch`/`Pipeline` or client-as-argument, matching the design
  contract. `tools/rediskeyguard` is correctly NOT in `go.work` and carries its own
  `go.mod`.

## Findings

### Critical
None.

### Important
None.

### Minor

1. **Duplicated namespace constant + constructor across two atlas-data packages.**
   `ingestJobNamespace = "data-ingest"` and `newIngestJobRegistry(...)` are copy-pasted in
   `services/atlas-data/atlas.com/data/runtime/rest/jobs.go:37,42` and
   `services/atlas-data/atlas.com/data/runtime/ingest/heartbeat.go:26,31`. The writer
   (ingest pod) and reader (REST pod) live in separate packages, so a single shared symbol
   would require a third package; each copy carries a "must match" comment. A silent
   divergence of this literal would break the heartbeat-watchdog read path with no compile
   error. Low risk given the comments, but it is two sources of truth for one wire
   contract. Consider extracting to a tiny shared `runtime/ingestkey` package.

2. **`GetAll`/`GetAllEntries`/`Clear`-family silently swallow pipeline + unmarshal errors.**
   `libs/atlas-redis/registry.go:154` (`_, _ = pipe.Exec(ctx)`) and the per-entry
   `continue`-on-error loops (`registry.go:158-172`, `tenant_registry.go` `GetAllEntries`)
   mean a transient Redis error mid-pipeline drops entries rather than surfacing. This
   mirrors the pre-existing `TenantRegistry.GetAllValues` convention in the same file, so
   it is consistent rather than a new regression — but the aggregate readers
   (`GetMonsters`, `GetAll` cooldowns) can under-report under partial Redis failure. Note
   only.

3. **`ApplyDamage` collapses all `Update` failures to `errMonsterNotFound`.**
   `services/atlas-monsters/atlas.com/monsters/monster/registry.go:466-469` maps every
   non-nil error — including "optimistic lock failed after 1000 retries" and genuine Redis
   transport errors — to the not-found sentinel. The comment states this preserves the old
   Lua behavior ("collapsed every failure to 'monster not found'") and no caller
   string-matches it, so it is intentional and behavior-preserving. Note only.

## Accepted-by-design (not findings)

- **Key/data-shape churn on main is intentional** (context.md "Gotchas"): per-entity keys
  moving from raw `tenant.Id().String()` to `TenantKey(t)`, and the atlas-rates re-model
  from per-`(char,template)` STRING keys to one HASH per `(tenant,character)`
  (`rates/character/item_tracker.go:161-165`), orphan old keys on main; the FR-1.6 reclaim
  script cleans them once. atlas-maps is excluded per P3.
- **atlas-reactors cooldown semantics change** (context.md P2): per-key native-TTL keys →
  `TenantKeyedHash` with expiry-unix-ms in the field and lazy pruning in `IsOnCooldown`
  (`reactors/reactor/registry.go`). Explicitly accepted ("reactors are a repopulating
  runtime cache").
