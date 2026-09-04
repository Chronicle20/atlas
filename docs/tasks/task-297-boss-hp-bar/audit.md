# Backend Audit — atlas-channel (task-297-boss-hp-bar)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Scope:** branch diff `31a791e3a..5b6ed61a1` (11 commits)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-04
- **Build:** PASS
- **Tests:** 178 passed, 0 failed (154 packages with no test files)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(no output — success)

$ go test ./... -count=1
178 "ok" lines, 0 "FAIL" lines
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Yes | Every changed package audited: `data/monster`, `monster/bosshp`, `monster` (live_mirror.go), `kafka/consumer/monster`, `kafka/consumer/map`, `data/monster/mock`. |
| DOM structure (DOM-01..05,11,16) | Yes | `data/monster` has `model.go` and `rest.go` (both changed). |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `data/monster` has `rest.go` + `processor.go`. |
| Constants reuse (DOM-21) | Yes (new type `bosshp.Gauge`) | No shared equivalent found in `libs/atlas-constants/monster/` — N/A per rule text. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `_test.go` files in every touched package. |
| Cache (DOM-29) | Yes | `data/monster/cache.go` added. |
| Messaging (DOM-30) | No | No `AndEmit` / `message.Emit` / `producer.Produce` / `producer.ProviderImpl` call sites in any diffed file (grep across all diffed non-test files returned zero matches). |
| Multi-tenancy (DOM-31) | Yes | `data/monster` has `rest.go`; `cache.go`/`consumer.go` read `tenant.MustFromContext(ctx)`. |
| Migration hygiene (DOM-34/35) | No | Diff moves no symbols between a service and `libs/atlas-*`. |
| Deploy & topics (DOM-22/23) | No | No `libs/atlas-*` module added, no Kafka topic env var added/renamed. |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; `tools/goroutine-guard.sh` run, exit 0. |
| Channel wire values (DOM-25) | Yes | Diff is entirely within `services/atlas-channel`. |
| Resilience (DOM-27/28) | Yes | DOM-27 N/A (no `resource.go`/HTTP handler changed, no DB). DOM-28 fires: `bosshp.Resolver.Resolve` fetches remote atlas-data and both call sites branch on its error as a fallback. |
| External clients (EXT-01..04) | Yes | `data/monster/cache.go`/`processor.go` reach `requests.Provider[...]` → `requests.GetRequest[RestModel]` for atlas-data. |
| Scaffolding (SCAFFOLD-01..09) | No | No new service directory, no new channel `Writer`/`Handler` registered (`FieldEffectWriter` pre-exists), `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | No | atlas-channel's monster/map domain does not handle auth tokens, redirects, or secrets in this diff. |

## Checklist Results

### data/monster (domain package — has `model.go`, `rest.go`, `processor.go`, `cache.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | N/A | Package has `model.go` but is a pure REST-client projection (constructed only via `Extract`, `data/monster/rest.go:34-41`); no new construction pathway is added by this diff. Same shape/no-builder as the ported precedent `data/skill` (no `builder.go` there either). Per established audit practice (e.g. `docs/tasks/task-054-effect-duration-units/audit.md:113`), DOM-01 governs new domain-construction code, of which none is added here. |
| DOM-02/DOM-03 | `ToEntity`/`Make` in `entity.go` | N/A | No `entity.go` — package has no persistence. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `data/monster/rest.go:44-51`. |
| DOM-05 | `TransformSlice` used by list handlers | N/A | No list handler exists in this REST-client package (`grep TransformSlice` → 0 matches; package has no `resource.go`). |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `data/monster/processor.go:19` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-11 | Lazy providers | N/A | No `provider.go`. |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no writes. |
| DOM-18 | JSON:API interface (`GetName/GetID/SetID`) | PASS | `data/monster/rest.go:17-31`. |
| DOM-19 | Flat request models | PASS | `RestModel` is flat: `Id, Boss, FixedDamage, TagColor, TagBackgroundColor` (`rest.go:8-14`). |
| DOM-21 | No redeclared shared constant/type | PASS (N/A) | `bosshp.Gauge` (new type) has no equivalent in `libs/atlas-constants/monster/` (`constants.go`, `skill.go`, `status.go`, `temporary_stat.go` inspected — no gauge/tag-color/boss-hp declarations). |
| DOM-29 | Cache is an application-scoped singleton via accessor | PASS | `data/monster/cache.go:100-109` `getMonsterCache()` uses `sync.Once` + package-level pointer; `ProcessorImpl` (`processor.go:12-16`) holds no cache field, only `l`/`ctx`. Mirrors the accepted `data/skill/cache.go` pattern. |
| DOM-31 | Tenant/trace travel in context only | PASS | `data/monster/cache.go:174` `t := tenant.MustFromContext(ctx)`; `RestModel` (`rest.go:8-14`) carries no tenant field. |
| FILE-01 | Processor in `processor.go` | PASS | `data/monster/processor.go:9-29` — interface, constructor, and `ProcessorImpl.GetById` all present. |
| FILE-02 | RestModel/Transform/Extract/JSON:API in `rest.go` | PASS | `data/monster/rest.go` (whole file). |
| FILE-03 | Cross-service requests in `requests.go` | PASS | `data/monster/requests.go:17-24` (unchanged by this diff but correctly placed). |
| FILE-04 | Entity in `entity.go` | N/A | No entity/persistence. |
| FILE-05 | Model in `model.go` | PASS | `data/monster/model.go` (whole file). |
| FILE-06 | No package-named catch-all file | PASS | `cache.go`, `model.go`, `processor.go`, `requests.go`, `rest.go` are each single-purpose; none combines ≥2 of the responsibilities above. |
| EXT-01 | Target RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** | `data/monster/rest.go:8-31` — `RestModel` has `GetName/GetID/SetID` but no `SetToOneReferenceID`/`SetToManyReferenceIDs` (no-op or otherwise). `grep -n "SetToOneReferenceID\|SetToManyReferenceIDs" data/monster/rest.go` → 0 matches. |
| EXT-02 | `httptest`-backed integration test with populated-struct assertion | **FAIL** | `grep -rn httptest data/monster/*.go` → 0 matches. `rest_test.go`'s `TestUnmarshalDataPayload` only unmarshals a literal byte payload directly into `RestModel`; it never exercises `requestById`/`getBaseRequest`/`upstreamFn` through a real HTTP round trip. |
| EXT-03 | Only genuine 404s map to "not found" | PASS | `data/monster/cache.go:186-196` `getByIdCached` only negative-caches `errors.Is(err, requests.ErrNotFound)`; all other errors pass through unchanged, confirmed by `TestMonsterDataCache/TransientErrorNotCached` (`cache_test.go:141-161`, asserts a non-`ErrNotFound` error is never cached and is refetched every call). |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor`, not hardcoded DNS | PASS | `data/monster/requests.go:16` `requests.RootUrlFor(ctx, "DATA")` — accepted variant per established precedent (e.g. `docs/tasks/task-254-party-experience-sharing/audit.md:90`). |
| Testing DOM-20 | Table-driven or single-scenario per guideline example | PASS | `cache_test.go:51-249` `TestMonsterDataCache` is table-driven (`tests := []struct{...}` + `t.Run`). `rest_test.go`'s `TestExtract`/`TestTransformRoundTrip`/`TestUnmarshalDataPayload` are single-scenario tests matching the guideline's own example (`testing-guide.md:21-27`). |
| Testing DOM-24 | Emit-path stubbing | N/A | No `AndEmit`/`message.Emit`/`producer.Produce` reachable from this package's tests. |
| Testing DOM-33 | Mock kept in sync with interface | PASS | `Processor` interface's `GetById` signature is unchanged; `data/monster/mock/processor.go:1-18` is a newly-added mock matching the existing signature, `var _ monster.Processor = (*ProcessorMock)(nil)` (`mock/processor.go:11`). |

### monster/bosshp (support package — single file, no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Processor/RestModel/requests/entity/Builder placement | N/A | None of the named symbols (`type Processor interface`, `type RestModel`, `requests.RootUrl`, `type entity struct`, `type Builder`) appear anywhere in `bosshp.go` — mechanical grep for each returns 0 matches. The package's `Gauge`/`Resolver`/`AnnounceOperator` are a business-rule qualification + announce helper, not a domain package in the FILE-01..05 sense. |
| FILE-06 | No package-named catch-all file | PASS | `bosshp.go` is a genuine single-purpose utility (FR-1 qualification + gauge announce, both explicitly one workflow per the package doc comment `bosshp.go:1-4`) — falls under the documented "genuine single-purpose utility" carve-out (`file-responsibilities.md:216`). |
| DOM-26 | Goroutines via `routine.Go` | PASS | `bosshp.go` spawns no goroutines directly; `tools/goroutine-guard.sh` exit 0 (see Runtime safety row below). |
| DOM-28 | Fallible enrichment degrades loudly | **FAIL** | See consumer-package rows below — the two call sites of `bosshp.Resolver.Resolve` (the remote-data-fetching function this package exposes) swallow the error with a bare log, not `model.ErrDecorator` + `degrade.Observe(...)`. |
| Testing DOM-20 | Table-driven | PASS | `bosshp_test.go:16-129` `TestResolve` and `bosshp_test.go:131-164` `TestBossHpBodyBytes` both use `tests := []struct{...}` + `t.Run`. |

### kafka/consumer/monster (support package — no `model.go`, has changed non-test/test files)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No package-named catch-all file | PASS | `consumer.go` is the conventional Kafka-handler file for this package; the new `bossHpBroadcaster` var is a handler helper, not a second domain concern. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `consumer.go:535` `routine.Go(l, ctx, func(_ context.Context) {...})` wraps `bossHpBroadcaster`'s body; `tools/goroutine-guard.sh` exit 0 (ran against the whole module). |
| DOM-28 | Fallible enrichment degrades loudly | **FAIL** | `kafka/consumer/monster/consumer.go:534-539` — `bossHpBroadcaster`'s `Resolve(...)` failure branch is `l.WithError(err).Errorf(...); return` with no `model.ErrDecorator`/`degrade.Observe(...)`. `degrade.Observe` is already imported and used elsewhere in this same file for an analogous remote-fetch fallback (`consumer.go:374` `degrade.Observe(l, "channel.monster.control_grant_fetch", e.UniqueId, err)`, pre-existing/unchanged), so the established in-file pattern was available and not applied to the new boss-HP fetch. |
| Testing DOM-20 | Table-driven | PASS | `consumer_test.go:743-810` `TestHandleStatusEventDamaged_BossHpGauge` is table-driven; `consumer_test.go:465-537` `TestHandleStatusEventDeath_BossHpGaugeEmpties` uses `t.Run` subtests per scenario. |
| Testing DOM-33 | Mock sync | N/A | No `Processor`/`Provider`/`Administrator` interface changed in this package. |
| Multi-tenancy DOM-31 | Tenant travels in context only | PASS | `consumer.go:210` / `:281` / `:329` `t := tenant.MustFromContext(ctx)`; the `StatusEvent` bodies carry no tenant field. |

### kafka/consumer/map (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0; `bossHpSenderFn` (`consumer.go:836-844`) spawns no goroutine of its own — it runs inline inside the per-session announce loop, consistent with the existing `doorAnnounce`/spawn path shape. |
| DOM-28 | Fallible enrichment degrades loudly | **FAIL** | `kafka/consumer/map/consumer.go:836-841` — `bossHpSenderFn`'s `Resolve(...)` failure branch is `l.WithError(err).Errorf(...); return nil` with no `model.ErrDecorator`/`degrade.Observe(...)`. Same file already imports and uses `degrade.Observe` for an analogous remote-fetch fallback at `consumer.go:425` (`degrade.Observe(l.WithField("instance", f.Instance()), "channel.map.environment_replay", uint32(f.MapId()), eerr)`, pre-existing/unchanged). |
| Testing DOM-20 | Table-driven | PASS | `consumer_test.go:433-563` `TestSpawnMonsterForSession_SendsBossHpAfterSpawn` is table-driven (`tests := []struct{...}` + `t.Run`). |
| FR-12 ordering (design, not a rule ID) | Spawn before gauge | PASS | `consumer.go:775-787` — `bossHpSenderFn` is invoked after `doorAnnounce(...MonsterSpawnWriter...)` and the control branch, matching the documented Spawn-then-Control(-then-gauge) ordering invariant. Verified by test `wantSeq` assertions (`consumer_test.go:459-539`). |

### monster (package `monster`, `live_mirror.go` — support/domain mirror, no persistence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | `LiveEntry`/model shape stays in its existing file | PASS | `MaxHp` field added to the pre-existing `LiveEntry` struct in `live_mirror.go`, not scattered elsewhere. |
| Testing DOM-20 | Single-scenario test matches guideline example | PASS | `live_mirror_test.go:213-226` `TestLiveEntryFromModel_SeedsMaxHp` — single-scenario, matches `testing-guide.md:21-27`'s own example shape. |

## Security Review

Not applicable — SEC-01..04's trigger ("service handles authentication, authorization, tokens, redirects, or secrets") did not fire on this diff; no touched file reads/writes tokens, performs redirects, or handles secrets.

## Not evaluable from the diff

- Live smoke acceptance criteria (PRD §10: gauge appears on first hit, tracks damage, empties on kill, visible to mid-fight entrant, ordinary mob unaffected) — would require a live client/tenant session; out of reach of a static code review.
- Whether `template_gms_12_1.json`/`template_gms_92_1.json`'s missing `FieldEffect` writer is acceptable long-term — this is explicitly deferred by design (§9 OQ1) and recorded in `docs/tasks/task-297-boss-hp-bar/follow-up-field-effect-writer-gms12-gms92.md`, so it is disposed by that document rather than left open here.

## Summary

### Blocking (must fix)
- EXT-01: `services/atlas-channel/atlas.com/channel/data/monster/rest.go:8-31` — `RestModel` (used to decode atlas-data's `GET /monsters/{monsterId}`) has no `SetToOneReferenceID`/`SetToManyReferenceIDs` no-ops; an api2go decode error is one upstream `relationships` block away.
- EXT-02: `services/atlas-channel/atlas.com/channel/data/monster/` — no `httptest`-backed integration test exercises the actual client (`requestById` → `requests.GetRequest[RestModel]` → `Extract`); only a hand-built byte payload is unmarshalled directly.
- DOM-28: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:534-539` — `bossHpBroadcaster`'s atlas-data lookup failure is a bare `l.WithError(err).Errorf(...); return`, not `model.ErrDecorator` + `degrade.Observe(...)`, despite the same file already using that pattern for an analogous fetch (`consumer.go:374`).
- DOM-28: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:836-841` — `bossHpSenderFn` has the same gap; the same file already uses `degrade.Observe` elsewhere (`consumer.go:425`).

### Non-Blocking (should fix)
- None identified beyond the blocking items above.

### Not evaluable
- 2 items (see "Not evaluable from the diff" section).
