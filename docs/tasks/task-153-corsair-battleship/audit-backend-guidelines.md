# Backend Audit — task-153-corsair-battleship (Go changes)

- **Service Path:** `services/atlas-channel/atlas.com/channel` (plus `libs/atlas-constants`, `libs/atlas-redis`, `libs/atlas-packet`)
- **Branch:** `task-153-corsair-battleship`
- **Branch range:** `f5aaced5ca7527f6d5e32d5cca59e1ab0f0020b7..HEAD`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-28
- **Build:** PASS
- **Tests:** all packages `ok` (0 failures) across atlas-channel, atlas-constants, atlas-redis, atlas-packet
- **Overall:** PASS

Scope note: this audit covers only the Go changes enumerated in the task brief. The nine JSON seed-template edits and docs are explicitly out of scope; where a checklist item (DOM-25b) depends on template content, it is marked out-of-scope rather than pass/fail.

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...            # clean, no output
$ cd services/atlas-channel/atlas.com/channel && go test ./... -count=1    # all "ok", including:
    ok  atlas-channel/battleship                 0.027s
    ok  atlas-channel/character/buff              0.123s
    ok  atlas-channel/kafka/consumer/buff          0.024s
    ok  atlas-channel/kafka/consumer/mount         0.023s
    ok  atlas-channel/kafka/message/buff           0.030s
    ok  atlas-channel/mount                        0.029s
    ok  atlas-channel/socket/writer                0.009s
    (0 FAIL lines in full output)

$ cd libs/atlas-constants && go build ./... && go test ./... -count=1      # ok
$ cd libs/atlas-redis && go build ./... && go test ./... -count=1          # ok
$ cd libs/atlas-packet && go build ./... && go test ./... -count=1         # ok

$ go vet ./...   # clean in atlas-channel, atlas-redis, atlas-packet, atlas-constants
$ tools/goroutine-guard.sh   # exit 0
$ tools/redis-key-guard.sh   # exit 0
```

## Domain Discovery

`services/atlas-channel/atlas.com/channel/battleship/` is a new package. It has `processor.go` + `mirror.go` + `mock/processor.go` — no `model.go`, no `resource.go` — so it is a **support package** (in-process game-state processor, no persistent domain entity, no REST surface). Per the audit instructions, the File Responsibilities Checklist still applies in full; the DOM checklist's model/entity/builder/REST items (DOM-01..05, DOM-18/19) do not apply because there is no persisted domain model or REST resource in this package — noted as N/A, not silently skipped.

`libs/atlas-redis/counter.go` (`TenantCounter`), `libs/atlas-packet/resolve.go` (`ResolveValue`), and `libs/atlas-constants/skill/mount.go` (`IsBattleshipMountSkill`) are single-purpose additions to existing shared libraries, not new packages.

## File Responsibilities Checklist — `battleship` package

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface + Impl in `processor.go` | PASS | `battleship/processor.go:125` (`type Processor interface`), `:137` (`type ProcessorImpl struct`), `:143` (`func NewProcessor`), all `(p *ProcessorImpl)` methods at `:151`,`:162`,`:167`,`:244`,`:263` are in the same file. No processor logic found in `mirror.go`. |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | N/A | Package has no `rest.go` and no REST surface — battleship state is process-internal (mirror) + Redis, never serialized over JSON:API. No violation: nothing REST-shaped exists to misplace. |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | Package makes no `requests.RootUrl`/`GetRequest`/`PostRequest` calls itself; collaborators that do cross-service calls (`character.NewProcessor`, `buff.NewProcessor`, `charskill.NewProcessor`, `dataskill.NewProcessor`) are pre-existing sibling packages, injected via the `cancelBuffFunc`/`characterLevelFunc`/`effectFunc` seams at `battleship/processor.go:83-101` — not duplicated inside `battleship`. |
| FILE-04 | Entity + Migration + TableName in `entity.go` | N/A | No GORM entity — state lives in Redis (`libs/atlas-redis.TenantCounter`) and an in-process mirror, not a database table. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No persisted `Model`, so no `builder.go`/`administrator.go`/`provider.go`/`state.go` are expected. `DrainStatus`/`DrainResult` (an in-memory result enum, not a persisted state machine) live in `processor.go:103-120`, next to the `Drain` method that produces them — consistent with `state.go`'s intent ("state transition helpers") without requiring a separate file for four `iota` values used by one method. |
| FILE-06 | No package-named catch-all file | PASS | Package files: `battleship/mirror.go`, `battleship/processor.go`, `battleship/mirror_test.go`, `battleship/processor_test.go`, `battleship/mock/processor.go`. No `battleship.go` bundling multiple responsibilities. `mirror.go` is single-purpose (the `RideMirror` singleton and its CRUD); `processor.go` is single-purpose (Processor interface/impl + the `ShipHP` formula + break flow). Neither collapses ≥2 of the File Responsibilities table's roles into one file. |

## File Responsibilities Checklist — other new/touched files

| ID | File | Status | Evidence |
|----|------|--------|----------|
| FILE-01 | `socket/writer/options_registry.go` | PASS | Single-purpose registry (`RegisterTenantWriterOptions`/`TenantWriterOptions`/`EvictTenantWriterOptions`), not a Processor — correctly not processor-shaped; no misplaced Processor logic. `options_registry.go:1-55`. |
| FILE-06 | `libs/atlas-redis/counter.go` | PASS | `TenantCounter` type + its four methods + the two Lua scripts, one purpose (a tenant-scoped Redis counter), consistent with the existing `libs/atlas-redis` file layout (`keys.go`, `registry_test.go` companions). |
| FILE-06 | `libs/atlas-packet/resolve.go` | PASS | `ResolveValue` added alongside the pre-existing `ResolveCode`/`ResolveName`/`WithResolvedCode` in the same file — all four are the same "resolve a wire value from tenant writer options" responsibility; not a mixed-responsibility file. |

## Per-Package Mechanical Checks

### `battleship` (support package — Processor + in-memory mirror + Redis-backed counter)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `battleship/processor.go:143` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-07 | Callers pass `d.Logger()` / `l` | PASS | All call sites (`skill/handler/mount.go:272`, `kafka/consumer/buff/consumer.go` seam, `socket/handler/character_damage.go:54`, `session/processor.go` seam) pass the handler/consumer's own `logrus.FieldLogger`, never `logrus.StandardLogger()`. `grep -rn "logrus.StandardLogger()" services/atlas-channel/atlas.com/channel/battleship services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` → no matches. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -n "os.Getenv" services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` → no matches in the touched handler files. |
| DOM-13/14/15 | No cross-domain writes/provider calls/entity creation in handlers | PASS | `character_damage.go`, `character_skill_use.go`, `character_attack_common.go` call only `battleship.NewProcessor(...).Drain/...`, never a `battleship` provider or Redis client directly; no `db.Create`/`db.Save`/`db.Delete` anywhere in the diff (`grep` clean). |
| DOM-21 | No atlas-constants duplication | PASS | `CorsairBattleshipId`/`CorsairBattleshipCannonId`/`CorsairBattleshipTorpedoId` are pre-existing in `libs/atlas-constants/skill/constants.go:3236-3238`, reused (not redeclared) throughout the diff (`processor.go:84,88,92`, `character_attack_common.go:913-919`, `common.go` cooldown gate). `IsBattleshipMountSkill` (new) wraps the existing `skill.Id` type rather than introducing a parallel enum — `libs/atlas-constants/skill/mount.go:35-37`. No new item-classification, inventory-type, or numeric-literal enum was introduced by this diff (verified via `git diff ... | grep -E "/ ?10000|/ ?1000000|itemId ?/"` → no matches). |
| DOM-25 | Client wire values config-resolved, no Go literals | PASS (a,c; b out of scope) | (a) The two client-interpreted values — the battleship vehicle item id and the HP-gauge pseudo-skill id — are resolved via `atlaspacket.ResolveValue(l, opts, "vehicles", "CORSAIR_BATTLESHIP")` (`skill/handler/mount.go:261`) and `atlaspacket.ResolveValue(l, opts, "skills", "BATTLESHIP_HP_GAUGE")` (`socket/handler/character_damage.go:89`), both reading from `writer.TenantWriterOptions(...)` — a per-tenant table, never a Go literal. On any resolve miss the mount/gauge is skipped rather than sent with a guessed value (`mount.go:144-147`, `character_damage.go:90-92`). `libs/atlas-constants/skill/mount.go:30-34` explicitly documents *why* Battleship is excluded from the existing `SkillOnlyMountVehicleId` literal table (that table predates this diff and is out of scope for this branch). (c) No domain service in this diff emits a client byte in a Kafka event; the vehicle id/gauge id are resolved only at the channel's socket-writer boundary. (b) Confirming the `vehicles`/`CORSAIR_BATTLESHIP` and `skills`/`BATTLESHIP_HP_GAUGE` tables exist in all nine `services/atlas-configurations/seed-data/templates/*.json` files is out of scope per this audit's instructions (JSON templates excluded) — **not independently verified here**; flag for the reviewer covering the template diffs. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exits 0 (repo-wide, includes this diff). The only bare `go func()` statements introduced by this diff are inside `_test.go` files (`battleship/processor_test.go:273,317,352`, `libs/atlas-redis/counter_test.go:113,220`), which the guard's own contract excludes. No bare `go` in non-test production code in the diff. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A — no emit paths reached) | Every new test in this diff that touches a battleship-adjacent collaborator does so through an injected function-var seam or a `battleshipmock.ProcessorMock`/mock producer, never the real `buff.NewProcessor(...).Cancel`, `charskill.NewProcessor(...).ApplyCooldown`, or `session.Destroy`'s real Kafka emit path. Confirmed via `git diff ... -- '*_test.go' \| grep -n "producertest\|ResetInstance\|AndEmit(\|message.Emit("` → zero matches, i.e. no new test exercises an emit path at all, so no stub is required. Explicit design comments confirm this was deliberate: `session/battleship_hook_test.go:19-28` and `kafka/consumer/buff/consumer_test.go` both call the extracted hook function directly specifically to avoid `Destroy`'s live-broker Kafka path. |
| DOM-27 | Transient DB errors → 503 | N/A | `battleship` has no database and no REST resource; the touched `socket/handler/*.go` files are socket packet handlers, not REST `resource.go` handlers with `http.StatusInternalServerError` branches — DOM-27 targets DB-backed REST services and does not apply to this packet-handling code path. |
| DOM-28 | No silent degradation in decorators | N/A | No `model.Decorator[...]` implementation is added or modified in this diff (`git diff ... \| grep -n "Decorator"` → no matches), so the trigger condition for DOM-28 is not met. Redis-failure degradation in `battleship.Drain` is handled by an explicit `DrainStatus` enum plus a `Warn`-level log at every degrade point (`processor.go:187,207,213`, `mirror`/`store` nil-guards at `:153,:176`) — logged, not silently swallowed, even though it is not a `Decorator` in the DOM-28 sense. |

### `libs/atlas-redis` (`TenantCounter`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Multi-tenancy | Keys scoped per tenant | PASS | `TenantCounter.entityKey` delegates to the pre-existing `tenantEntityKey(namespace, t, key)` helper (`libs/atlas-redis/counter.go:57-59`, helper at `libs/atlas-redis/keys.go:47`) — no tenant-unscoped raw key construction. |
| Concurrency | Exactly-once zero-crossing under concurrent decrements | PASS | Both `decrByIfExistsScript` and `initIfMissingAndDecrByScript` are Lua scripts executed atomically server-side by Redis (`counter.go:18-25`, `:36-42`), and this is exercised under real concurrency in `TestTenantCounter_ConcurrentDecrExactlyOneCrossing` (`counter_test.go:94-147`, using 10 goroutines against a real miniredis instance) and `TestTenantCounter_InitIfMissingAndDecrBy_ConcurrentNoLostDecrement` (`counter_test.go:204-243`). |
| Redis-key-guard | No raw keyed go-redis calls outside `libs/atlas-redis` | PASS | `tools/redis-key-guard.sh` exits 0; `TenantCounter` itself lives inside `libs/atlas-redis`, and the `battleship` package only calls it through the `counterStore` interface seam, never `*goredis.Client` directly. |

### `libs/atlas-packet` (`ResolveValue`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Functional/fail-loud pattern | Consistent with sibling `ResolveCode`/`ResolveName` | PASS | `ResolveValue` follows the exact same nested-map lookup shape as `ResolveCode` (`resolve.go:97-136` vs `:28-61`), differing correctly on failure behavior: `ResolveCode` returns a sentinel `99` (documented as intentionally crash-prone but non-fatal for a 1-byte code), `ResolveValue` returns `(0, false)` since a 4-byte wire value has no safe sentinel — documented at `resolve.go:100-102`. Every failure branch logs `Errorf` before returning, consistent with the existing functions in the same file. |
| Test coverage | Valid + all miss branches | PASS | `resolve_test.go:160-197` (`TestResolveValueValid`, `TestResolveValueMisses`) covers the hex-string, missing-property, non-map, missing-key, unparseable-string, and unsupported-type paths (5 sub-cases). |

### `libs/atlas-constants/skill/mount.go` (`IsBattleshipMountSkill`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Wraps shared type, no duplication | PASS | `func IsBattleshipMountSkill(id Id) bool { return id == CorsairBattleshipId }` (`mount.go:35-37`) uses the pre-existing `skill.Id` type and the pre-existing `CorsairBattleshipId` constant — a thin predicate, not a new enum or classification scheme. |
| Test coverage | PASS | `mount_test.go:59-78` (`TestIsBattleshipMountSkill`) covers true/false across Battleship, Cannon, Torpedo, a skill-only mount, and a tamed mount. |

### `socket/writer/options_registry.go` (new per-tenant writer-options registry)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Multi-tenancy | Isolated per tenant UUID | PASS | `tenantWriterOptions map[uuid.UUID]map[string]map[string]interface{}` keyed by `tenantId` (`options_registry.go:19`); `RegisterTenantWriterOptions`/`TenantWriterOptions`/`EvictTenantWriterOptions` all take/read `tenantId` explicitly — no global unscoped map. |
| Concurrency | Process-global mutable state guarded | PASS | Single `sync.RWMutex` (`tenantWriterOptionsMu`) guards all reads/writes (`options_registry.go:18,31-33,40-41,52-53`); `RWMutex` (not plain `Mutex`) matches the read-heavy access pattern (every packet-encode/skill-cast call site reads; only tenant listener build/teardown writes). |
| Lifecycle | Registered on listener build, evicted on tenant teardown | PASS | `main.go` diff: `writer.RegisterTenantWriterOptions(t.Id(), tenantCfg.Socket.Writers)` in `buildListener` and `writer.EvictTenantWriterOptions(tid)` in the tenant-unregister callback — both confirmed in the `main.go` diff hunk. |
| Test coverage | Lifecycle + isolation + miss cases | PASS | `options_registry_test.go:11-44` (`TestTenantWriterOptionsLifecycle`) covers register/lookup, a writer with nil options (`ok=false`), an unknown writer, an unregistered tenant, and post-eviction lookup. |

### `battleship.RideMirror` (per-channel-process in-memory ride state)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Multi-tenancy | Tenant isolation | PASS | `perTenant map[uuid.UUID]map[uint32]RideState` keyed first by `t.Id()` (`mirror.go:29,48-57,60-69`); `mirror_test.go:37-39` explicitly asserts `t2` cannot observe `t1`'s rider ("tenant isolation violated"). |
| Singleton pattern | `sync.Once` singleton, matches `patterns-cache.md`/repo precedent | PASS | `rideMirrorOnce sync.Once` + `GetRideMirror()` (`mirror.go:32-43`), same shape as the file's own comment cites (`monster.StatusMirror` precedent, `mirror.go:26`). |
| Concurrency | `RWMutex`-guarded CRUD | PASS | `sync.RWMutex` guards `Put`/`Get`/`Remove`/`EvictTenant` (`mirror.go:28,49,61,73,82`); exactly-once break correctness under concurrent `Drain` calls is verified with real goroutines in `battleship/processor_test.go:261-333` (`TestDrainBreakExactlyOnceUnderConcurrency`, `TestDrainLazyReinitBreakExactlyOnceUnderConcurrency`, `TestDrainLazyReinitNoLostDecrementUnderConcurrency` — 10 and 8 concurrent goroutines respectively, each asserting exactly one `DrainBroke` and no lost decrement). |

## Processor / Functional Convention Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Immutable model | `RideState`/`DrainResult` are plain value structs, no shared mutable state leaked | PASS | `RideState` (`mirror.go:17-20`) and `DrainResult` (`processor.go:117-120`) are returned by value from `Get`/`Drain`; callers cannot mutate the mirror's internal map through them. |
| `Interface` + `Impl` | PASS | `type Processor interface` / `type ProcessorImpl struct` (`processor.go:125,137`), matching the repo-wide convention. |
| `NewProcessor(l, ctx)` | PASS | `processor.go:143`; `tenant.MustFromContext(ctx)` captured once at construction (`processor.go:144`), not re-derived per call. |
| `tenant.MustFromContext(ctx)` | PASS | `processor.go:144`, and in every new call site that needs the tenant directly (`mount.go:255`, `character_damage.go:83`, `character_attack_common.go:670,727,913`). |
| Pure `Method(mb)` vs `MethodAndEmit()` | N/A (documented deviation) | `battleship.Processor` has no `*AndEmit` methods because it emits nothing itself — all side effects it triggers (buff cancel, cooldown apply) are delegated to the pre-existing `buff`/`charskill` processors' own `AndEmit`-shaped APIs via the `cancelBuffFunc`/`applyCooldownFunc` seams (`processor.go:83-89`). This is a legitimate shape for a package that owns no Kafka topic of its own, not a violation of the pattern. |
| Cache singleton pattern (adapted for the mirror) | PASS | `GetRideMirror()` is not instantiated inside `NewProcessor` — it is fetched via the package-level singleton getter on each call (`processor.go:163,171,264`), matching the "singleton, not per-instance" spirit of `patterns-cache.md` even though this is a mirror rather than a `CacheInterface`. |

## Testing Guide Compliance

| Check | Status | Evidence |
|-------|--------|----------|
| Table-driven tests | PASS | `TestShipHPFormula` (`processor_test.go:144-184`), `TestResolveValueMisses` (`resolve_test.go:178-197`), `TestBattleshipAttackPermitted` (`character_attack_battleship_gate_test.go:22-53`), `TestGaugeCooldownValue`/`TestShouldAnnounceGauge` (`character_damage_test.go:25-70`), `TestShouldApplyCastCooldown` (`common_cooldown_test.go:9-28`), `TestBattleshipCastBlocked` (`character_skill_use_test.go:8-30`), `TestIsBattleshipMountSkill`/`TestSkillOnlyMountVehicleId` (`mount_test.go` constants pkg) all use `tests := []struct{...}` + `t.Run`. |
| Mocks updated for interface change | PASS | `battleship/mock/processor.go` implements all four `Processor` methods with the nil-check/default-behavior pattern from `testing-guide.md`, and `var _ battleship.Processor = (*ProcessorMock)(nil)` (`mock/processor.go:20`) statically pins the mock to the interface. |
| Race-safety of concurrent tests | PASS | `go test ./... -count=1` ran clean; the concurrency tests (`TestDrainBreakExactlyOnceUnderConcurrency` et al., `TestTenantCounter_ConcurrentDecrExactlyOneCrossing`) use `sync.WaitGroup`/`sync.Mutex` correctly around shared counters, and rely on Redis/Lua-script atomicity (and the mirror's own `RWMutex`) for the property under test rather than the test's own locking. |

## Security Review

Not applicable — this is not an auth/token-handling service. No SEC-* checks triggered.

## Summary

### Blocking (must fix)
None found.

### Non-Blocking (should fix / follow-up)
- DOM-25(b): this audit did not verify that the `vehicles`/`CORSAIR_BATTLESHIP` and `skills`/`BATTLESHIP_HP_GAUGE` writer-option tables are present in all nine `services/atlas-configurations/seed-data/templates/*.json` files, since JSON templates are explicitly out of scope for this Go-focused audit per the task brief. Whoever reviews the template diffs for this branch should confirm those two keys exist under every supported version's `writers[]` entry for `CharacterBuffGive`/`CharacterSkillCooldown` before merge — a missing entry means `ResolveValue` returns `(0, false)` at runtime and the mount/gauge is silently skipped for that tenant version (logged, not crashing, but a live feature gap).
