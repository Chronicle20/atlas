# Plan Audit — task-120-monster-move-local-state

**Plan Path:** docs/tasks/task-120-monster-move-local-state/plan.md
**Audit Date:** 2026-07-02
**Branch:** task-120-monster-move-local-state
**Base Branch:** main (merge-base 38d4d0ba2 .. HEAD d9954effe)

## Plan-Adherence Review

### Executive Summary

The plan (7 tasks) was implemented **faithfully and completely**. Every file, interface,
and behavioral invariant specified in the plan is present in the diff, the cross-task
interface contracts hold (LiveMirror produced by Task 1 is consumed by Tasks 2 and 3;
the RECOVERY constant by Task 6; the three MP_CHANGED emissions carry authoritative
post-mutation MP end-to-end), both steady-state REST reads are removed from the warm
monster-move path with REST retained only as the miss fallback, and all new state is
tenant-keyed. The Task-7 verification gate genuinely ran (builds/tests/vet/docker-bake/
redis-key-guard all PASS). No `// TODO`, stub, or 501 was introduced. All three
pre-logged Minor items are confirmed non-blocking.

**Verdict: FAITHFUL — READY_TO_MERGE** (subject to the standard backend-guidelines review).

### Per-Task Findings

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | LiveMirror + metrics + main.go wiring | PASS | `monster/live_mirror.go` (full API: Lookup/Put/UpdateMp/UpdateAggro/Remove/EvictTenant/SweepStale/LiveEntryFromModel, singleton + sweeper); `monster/metrics.go` 3 mirror counters, exact names; `builder.go:22,51,86-89,132` controllerHasAggro field+clone+setter+build; `main.go:293` EvictTenant, `main.go:341` `/metrics` mount, promhttp import |
| 2 | Mirror write paths in status-event consumer | PASS | `kafka/consumer/monster/consumer.go`: `monsterGetByIdFn` seam (374); CREATED seeds mirror (140); DESTROYED (191) / KILLED (276) Remove; START_CONTROL (300) / STOP_CONTROL→false (329) / AGGRO_CHANGED (351) UpdateAggro before REST; MP_CHANGED UpdateMp before session gate (595). All pre-existing packet emission unchanged |
| 3 | Movement path consumes mirror | PASS | `movement/processor.go`: `monsterByIdFn` seam (114); `resolveLiveMonster` mirror→REST-fallback→backfill (121-137); `ForMonster` reads `entry` (140), field-consistency rejection preserves `return nil` with comment (149), `ackMp = uint16(entry.Mp)` (154), `useSkills = entry.ControllerHasAggro` (173); goroutines/inbox/snap/command emission untouched |
| 4 | Template-info TTL cache | PASS | `monster/information/cache.go` (env parse with clamp+warn-fallback, lazy expiry, negative caching on ErrNotFound only, `upstreamFn`/`EvictTenant` seams); `metrics.go` 2 counters exact names; `processor.go:24-54` read-through GetById, signature unchanged; `main.go:294` `monsterinfo.EvictTenant` |
| 5 | MP_CHANGED on skill-cast + basic-attack | PASS | Constants in BOTH `atlas-monsters/monster/kafka.go:36-39` and `atlas-channel/.../message/monster/kafka.go:107-110`; `processor.go` UseSkill emits SKILL_CAST with SkillId=`uint32(skillId)`, CharacterId=0 (643-646); UseBasicAttack emits BASIC_ATTACK with SkillId=0 (840-846); `testMobSkillLookup` seam (70) |
| 6 | MP_CHANGED on recovery regen | PASS | `recovery_task.go`: `recoveryMpEmitFn` type (28), `mpEmitFn` field (48), production wiring uses `MpChangeReasonRecovery` CharacterId=0 SkillId=0 (71-75), `Run()` captures `mpApplied` and emits post-HP block (136-146) |
| 7 | Full verification gate | PASS | `.superpowers/sdd/task-7-report.md`: atlas-channel + atlas-monsters `go vet`/`go test -race`/`go build` all exit 0; `docker buildx bake atlas-channel atlas-monsters` both images built; `tools/redis-key-guard.sh` exit 0 |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Cross-Task / Global-Constraint Verification

- **Interface consumption:** `monster.GetLiveMirror()` / `LiveEntryFromModel` / `RecordMirrorFallback`
  produced by Task 1 are consumed by the Task 2 consumer handlers and the Task 3 movement
  resolver. `MpChangeReasonRecovery` (Task 5) is consumed by Task 6's `mpEmitFn`. All resolve
  and compile (Task-7 build gate PASS).
- **Two REST reads removed from warm path:** `ForMonster` no longer calls
  `monster.NewProcessor(...).GetById` directly (routed through `resolveLiveMonster`,
  mirror-first) nor `monsterinfo.GetById` uncached (now read-through TTL cache). REST is
  reached only on a mirror miss (`resolveLiveMonster` fallback) or cache miss. The
  `UseSkill`/`UseBasicAttack` command POSTs later in `ForMonster` are pre-existing
  authoritative-action emissions (fire only on skill/attack moves), correctly out of scope.
- **MP accuracy end-to-end:** producer sets `MonsterMpAfter: m.Mp()` (post-deduct/post-regen
  model); both `statusEventMpChangedBody` (monsters) and `StatusEventMpChangedBody` (channel)
  carry `monsterMpAfter`; consumer applies `UpdateMp(e.Body.MonsterMpAfter)` before any gating.
  All three new emissions feed the authoritative post-mutation MP into the mirror.
- **Exact names/strings:** metric names, env var names (`MONSTER_INFO_CACHE_ENABLED/_TTL/_NEGATIVE_TTL`),
  and MP reasons (`SKILL_CAST`/`BASIC_ATTACK`/`RECOVERY`) match the plan verbatim.
- **Tenant-keying:** LiveMirror and infoCache are both `map[uuid.UUID]map[...]`, with
  isolation+eviction tests. No cross-tenant path.
- **No wire drift:** ackMp math (`computeAckMp`), useSkills/aggro logic, snap, and Kafka
  command emission unchanged; regression tests retained.

### Triage of Known Minor Items

1. **go.sum drift (~testcontainers/docker/moby transitives) — ACCEPTABLE, non-blocking.**
   Confirmed: `go.mod` gains only `prometheus/client_golang` + its direct transitives
   (beorn7/perks, munnerz/goautoneg, prometheus/client_model|common|procfs, go.yaml.in/yaml/v2)
   plus minor bumps (golang.org/x/sync 0.20→0.21, text 0.37→0.38). `go.sum` additionally
   gains 28 checksum lines for testcontainers/docker/moby/containerd — checksum-only module-graph
   drift from `go mod tidy` under workspace mode, **not** new direct/indirect deps in go.mod.
   docker-bake, go build, go test, and redis-key-guard all pass with it, so it is non-functional.
   Recommendation: acceptable to merge; optionally regenerate go.sum with `GOWORK=off go mod tidy`
   in a follow-up to trim the checksum noise, but not required for this PR.
2. **Missing test for invalid MONSTER_INFO_CACHE_ENABLED value; no direct Prometheus
   metric-value asserts — ACCEPTABLE.** `TestCache_InvalidEnvFallsBackToDefaults` covers invalid
   TTL/negative-TTL fallback; the enabled-parse path has an explicit default and warn branch in
   `parseBoolEnv`. Metric correctness is covered by call-count surrogates in the resolver/cache
   tests. Coverage gap is cosmetic; behavior is exercised.
3. **Pre-existing gofmt misalignment in atlas-monsters processor_test.go (~line 1708) —
   ACCEPTABLE, out of scope.** Confirmed present on base commit 38d4d0ba2; the misalignment is
   in `applyDoomEffectFromPlayer` (comment column alignment), unrelated to task-120's appended
   tests, which are gofmt-clean. `go vet` passes; CI does not gate on gofmt here.

### New Gaps Found

None. No new stubs, TODOs, 501s, silently-skipped tasks, or behavioral drift beyond the
three logged Minor items.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (run backend-guidelines review as the standard next step)

## Action Items

None required before merge. Optional follow-up: trim the go.sum checksum drift
(`GOWORK=off go mod tidy` in atlas-channel) and, separately from this task, gofmt the
pre-existing `applyDoomEffectFromPlayer` alignment.

---

## Backend-Guidelines Review

**Reviewer:** backend-guidelines-reviewer (adversarial, FAIL-until-proven)
**Date:** 2026-07-02
**Scope:** task-120 Go changes across atlas-channel (`monster/`, `monster/information/`, `movement/`, `kafka/consumer/monster/`, `kafka/message/monster/`, `main.go`) and atlas-monsters (`monster/processor.go`, `recovery_task.go`, `kafka.go`).

### Objective Gate

| Gate | Module | Result | Evidence |
|---|---|---|---|
| `go build ./...` | atlas-channel | PASS | clean |
| `go build ./...` | atlas-monsters | PASS | clean |
| `go test ./... -count=1` (changed pkgs) | atlas-channel | PASS | `monster`, `monster/information`, `movement`, `kafka/consumer/monster` all `ok` |
| `go test ./... -count=1` | atlas-monsters | PASS | `monster` ok 3.24s, `monster/information` ok |
| `go test -race` (concurrent pkgs) | atlas-channel | PASS | mirror/cache/consumer/movement all `ok` under `-race` |
| `go vet` (changed pkgs) | atlas-channel | PASS | clean |

Note: the fast test wall-time (atlas-monsters `monster` 3.24s, not 42s+) is positive evidence the new MP_CHANGED emit paths never reach the real `producer.ProviderImpl` retry-hang — DOM-24 is satisfied by construction.

### Applicability

These changes add in-process singletons (`LiveMirror`, `infoCache`), a background sweeper, Prometheus wiring, consumer mirror-write paths, and additive Kafka emissions. They are NOT a new REST domain package: no `model.go`/`rest.go`/`resource.go`/`administrator.go` were added, so DOM-01..05, DOM-08..09, DOM-14..19, and SUB-* are N/A (no new HTTP handlers, no new JSON:API models, no new writes-through-handlers). SEC-* is N/A (atlas-channel/monsters are not auth/token services). The applicable checks are listed below.

### Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processors accept `logrus.FieldLogger` | PASS | `information/processor.go:14` (`l logrus.FieldLogger`); `movement/processor.go:28` |
| DOM-07 | No `logrus.StandardLogger()` on request path | PASS (with Minor) | Request paths thread `p.l`. One package-global default `configLogger = logrus.StandardLogger()` at `information/cache.go:43` is used only for one-time config-parse warnings (not per-request); matches task-060 precedent. See Minor-1. |
| DOM-11 | Providers use lazy/upstream seam, not eager | PASS | `information/cache.go:151` `upstreamFn` wraps `requests.Provider[...]`; movement fallback via `monsterByIdFn` seam (`movement/processor.go:114`) |
| DOM-12 | No `os.Getenv` in handlers | PASS | env read only in cache config loader (`cache.go:54,71` via `os.LookupEnv`), not in any handler |
| DOM-20 | Table-driven / builder-pattern tests; no `*_testhelpers.go` | PASS | `live_mirror_test.go`, `recovery_task_test.go` struct-literal + Builder; no testhelpers file added |
| DOM-21 | No duplication of atlas-constants types | PASS | `LiveEntry.Field` reuses `field.Model` (`live_mirror.go:24`); ids/state carried as raw `uint32` runtime values (object id, MP) which atlas-constants does not model; MP-reason strings are Kafka-contract enums, service-local by nature. No shared type redeclared. |
| DOM-23 | Kafka topic naming | PASS (N/A-new) | No new topic; reuses `EVENT_TOPIC_MONSTER_STATUS` (`kafka.go:14`, channel `kafka.go:86`). No configmap/manifest topic churn. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | New emit sites use injectable seams: `p.emit` (`processor.go:632,843`), `mpEmitFn` (`recovery_task.go:71`). Tests inject no-op/recording stubs (`processor_test.go:44`, `recovery_task_test.go:47,190,235`); never the real producer. Test wall-time confirms no retry-hang. |
| — | No raw keyed go-redis outside libs/atlas-redis | PASS | new mirror/cache are pure in-process maps; zero redis imports (`grep` clean on `live_mirror.go`, `cache.go`) |

### Focus-Area Deep Checks

**Concurrency (locking correctness).** PASS.
- `LiveMirror` uses one `sync.RWMutex` consistently: reads under `RLock` (`Lookup` 79-92), all mutators under `Lock` (`Put` 97, `UpdateMp` 114, `UpdateAggro` 132, `Remove` 149, `EvictTenant` 161, `SweepStale` 170). No read under write-lock or vice versa; no map access outside a lock. `SweepStale` deletes during range (safe in Go) and prunes empty tenant maps.
- No lock is held across a blocking call: the only work under lock is map access and `prometheus` counter `Inc` (non-blocking). The REST fallback (`monsterByIdFn`) in `resolveLiveMonster` runs entirely outside any mirror lock (`movement/processor.go:127`); the `Put` backfill takes the lock separately (135).
- Lazy singletons: `GetLiveMirror` (`live_mirror.go:49`) and `getInfoCache` (`cache.go:103`) use `sync.Once`; the sweeper goroutine is started exactly once inside the `Once.Do` (52). Tests bypass the singleton via `newTestLiveMirror()`, so no goroutine leak in the suite. `-race` clean including the dedicated `TestLiveMirror_ConcurrentAccess`.

**Multi-tenancy.** PASS. All state nested `map[tenant-uuid]map[id]entry` (`live_mirror.go:39`, `cache.go:95`); every op keys on `t.Id()` / `tid`. Consumers derive tenant via `tenant.MustFromContext(ctx)` at each handler; movement path via `p.t` (`MustFromContext` in `NewProcessor`, `movement/processor.go:40`). `EvictTenant` wired into the listener evictor for both caches (`main.go:293-294`), alongside the existing StatusMirror/inbox evictors. Cross-tenant isolation is unit-tested (`TestLiveMirror_TenantIsolationAndEviction`).

**Error handling.** PASS. Negative-cache classification is strictly `errors.Is(err, requests.ErrNotFound)` gated (`information/processor.go:50`); transient/5xx/parse errors are returned uncached. Negative hits re-synthesize the sentinel via `notFoundError` (`cache.go:158`) so callers see a consistent shape. Emit failures are logged-not-fatal at all three new sites (`processor.go:634,844`, `recovery_task.go:145`). The `return nil` on field-consistency rejection (`movement/processor.go:149`) is the documented anti-drift preservation of pre-mirror behavior, not a swallowed error.

**Metrics cardinality.** PASS. Label sets are bounded: `{tenant}` (mirror hits/misses, cache misses), `{tenant, outcome∈success|failure}` (fallback), `{tenant, kind∈positive|negative}` (cache hits) — `metrics.go:15,23,31`, `information/metrics.go:15,23`. No unbounded label (no monsterId/characterId/objectId). Tenant count is the only growth axis and is small/bounded.

### Findings

**Minor-1 (style, non-blocking) — `information/cache.go:43`.** `configLogger` defaults to `logrus.StandardLogger()` for config-parse warnings rather than a request-scoped `FieldLogger`. It fires at most a few times at process start (invalid env only) and is overridable in tests. This mirrors the task-060 Redis-cache precedent. Not on any request path, so it does not violate DOM-07 in substance. Fix (optional): pass the bootstrap logger into `getInfoCache()`/`loadConfig()` at first init, or accept the current one-time-warning tradeoff.

**Observation-1 (pre-existing, out of task-120 scope) — `movement/processor.go` `ForMonster`.** Three `go func()` blocks concurrently assign the shared local `err` (lines 182, 189, 196/215), a benign-in-practice but real data race on the `err` word. Verified identical at base commit `38d4d0ba2` (the pre-mirror `mo, err := ... GetById` version has the same shared-`err` writes); task-120 did not introduce or widen it. Flagging for awareness only — not attributable to this task and not a merge blocker for it. If addressed later, give each goroutine its own `err`.

### Verdict

**PASS (no blocking findings).** Build/test/vet/-race gates green in both modules. Every applicable DOM-* check passes with file:line evidence. Concurrency, multi-tenancy, error-classification, DOM-24 producer-stubbing, and metric-cardinality invariants all hold. The single Minor item (config logger) is cosmetic and precedent-consistent; the one data-race observation is pre-existing and outside task-120's scope.
