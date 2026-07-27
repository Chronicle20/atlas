# Plan Audit — task-118-shared-service-bootstrap

**Plan Path:** docs/tasks/task-118-shared-service-bootstrap/plan.md
**Audit Date:** 2026-07-13
**Branch:** task-118-shared-service-bootstrap
**Base Branch:** main (merge-base dd14be700)

## Executive Summary

All 15 plan tasks (0–14) are VERIFIED implemented. The five library tasks landed first (correct commit ordering), followed by exactly one commit per migrated service (59 services), a gofmt style commit, docs, and acceptance evidence — 71 commits over merge-base, matching the plan's commit discipline. Every PRD §10 acceptance grep re-run during this audit returns the expected value (0 wrappers, 0 logger dirs, 59 `service.Bootstrap` mains, etc.). Spot-verified builds (`libs/atlas-service`, `atlas-fame`) pass; the lib API surface (`Provider`/`ProviderImpl`, `Bootstrap`/`Runtime`, `WithConfigProjection`/`AwaitProjectionCatchUp`, `ProjectionFuncs`, `CreateLogger`, snake_case hook) exists exactly as the plan specifies. All ten documented deviations match context.md and are correct, not gaps.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Rebase gate + re-measure | DONE | Commit 060c0af5d; rebased onto origin/main with both gate commits (d2e13ba3d task-114, e15b343b1 task-116) confirmed in context.md; drifts #1–#4 re-derived into plan.md (+49/−13) and context.md |
| 1 | `producer.Provider` + `ProviderImpl` | DONE | Commit 96f04396e; libs/atlas-kafka/producer/provider.go:10 `type Provider`, :14 `func ProviderImpl`; provider_test.go present |
| 2 | snake_case normalizer hook (CP-9) | DONE | Commit bcdef4376; fieldnorm.go:20 `fieldKeyNormalizerHook`, :51 `normalizeFieldKey`; fieldnorm_test.go present |
| 3 | `CreateLogger` (DUP-2) | DONE | Commit aa2438605; logger.go:15 `func CreateLogger`; logger_test.go present |
| 4 | `Bootstrap`/`Runtime`/readiness | DONE | Commit e6abe1cbd; bootstrap.go:48 `func Bootstrap`, :24 `WithoutTracer`, :30 `WithReadinessGate`, :84–109 Runtime accessors + `Ready()`/`Wait()` |
| 5 | config-projection option | DONE | Commit 51fb04699; projection.go:59 `WithConfigProjection`, :89 `AwaitProjectionCatchUp`, :34 `ProjectionFuncs`, :103 `parseProjectionCatchupTimeout`, :18 `Projection` iface |
| 6 | Pilot: fame | DONE | Commit 5283eaa70; main.go:25 `service.Bootstrap`, :26 `rt.Logger()`, :60 `MountReadiness("/readyz", rt.Ready)`, :63 `rt.Wait()`; outbox drainer block retained (:32–38 via routine.Go); `go build ./...` clean |
| 7 | Cohort A (45 svcs) | DONE | 45 per-service commits (account…transports + mts); logger dirs = 0, wrappers = 0 |
| 8 | Cohort B (5, no wrapper) | DONE | Commits for configurations, drop-information, gachapons, query-aggregator, rates |
| 9 | Cohort C (quest/merchant/saga) | DONE | Commits 7388e8945, c744ae9ee, 048267eba; merchant private teardown.go deleted; merchant go.mod gained atlas-service require+replace |
| 10 | world + character-factory | DONE | Commits df99b8c1b, 3c868118b; both use WithConfigProjection + AwaitProjectionCatchUp; character-factory gate kept BEFORE Run() per drift #3 |
| 11 | login + channel | DONE | Commits 77d0c4269, 641a5d199; channel main.go:157 WithConfigProjection + :165 ProjectionFuncs + :231 AwaitProjectionCatchUp |
| 12 | renders | DONE | Commit e787ae45a; only main without MountReadiness (root-mounts /readyz by hand); listener via routine.Go (main.go:64) per drift #4 |
| 13 | docs | DONE | Commit e31265afd; docs/observability.md:141–149 snake_case field-key convention section (collision rule + dotted-key passthrough documented); architectural-improvements.md finding-ID note correctly recorded in context.md |
| 14 | verification | DONE | Commit 4b3b26521 acceptance evidence; audit re-ran all greps (match) + spot-built libs/atlas-service (test ok) and atlas-fame (build OK) |

**Completion Rate:** 15/15 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No SKIPPED, PARTIAL, or DEFERRED tasks.

## Documented Deviations (verified, not gaps)

Each matches context.md "Post-rebase measurements" / "Notable execution deviations" and was independently confirmed:

- atlas-mts added (new service, drift #1) — Cohort A now 45, fleet 59. `find services -name main.go` = 59.
- 17 DB services retain the task-114 outbox drainer block — confirmed in atlas-fame main.go:32–38 using `routine.Go` (goroutine-guard compliant).
- character-factory gates catch-up BEFORE Run() (drift #3); world AFTER — both use AwaitProjectionCatchUp.
- renders listener uses `routine.Go` not bare `go` (drift #4) — main.go:64.
- marriages/reactors producer_test.go removed — confirmed absent (`find` returns nothing).
- character/storage alias lib import as `lifecycle` — confirmed (character main.go:20, storage main.go:13).
- merchant deleted private service/teardown.go + gained lib dep — confirmed (file absent; go.mod:13/113).
- maps kept local `kafka/producer` domain package (character.go) via `mapsproducer` — confirmed present.
- atlas-cashshop rest_test.go rewritten to `service.CreateLogger` — confirmed (rest_test.go:21).
- Task 13 architectural-improvements.md finding IDs (DUP-*/CP-9/OPS-*) do not exist there — correctly documented; snake_case landed in observability.md instead.

## Build & Test Results

Per plan's Task 14, the executor reported 61/61 modules clean on build/vet/test-race + full `docker buildx bake all-go-services` exit 0. This audit spot-verified (not the full sweep):

| Module | Build | Tests | Notes |
|--------|-------|-------|-------|
| libs/atlas-service | PASS | PASS | `go build/vet/test -race ./...` → `ok` (cached) |
| services/atlas-fame | PASS | n/a | `go build ./...` clean |

Guards (`redis-key-guard.sh`, `goroutine-guard.sh`) and full bake reported PASS/exit-0 in context.md Step 3; not re-run in this audit (executor evidence accepted per audit scope).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. All 15 tasks are implemented with evidence; all deviations are documented and correct. If a final gate is desired before PR, re-run the full `go test -race ./...` sweep + `docker buildx bake all-go-services` + both guard scripts from the worktree root (executor already reports these clean).

---

# Backend guidelines audit — task-118 shared library code (DOM/SUB/SEC)

**Auditor:** backend-guidelines-reviewer (adversarial, FAIL-until-proven)
**Audit Date:** 2026-07-13
**Scope:** NOVEL shared abstractions only — `libs/atlas-kafka/producer/provider.go`, `libs/atlas-service/{fieldnorm,logger,bootstrap,projection}.go` + tests. The 59 mechanical `service.Bootstrap` main.go swaps were NOT re-audited (behavior-preservation established per-batch). Base merge-base: dd14be700.
**Build/Test:** reported clean fleet-wide; focused `go test -race` on atlas-service (normalizer parallel-emit, readiness gates, projection) passed locally (1.054s).

## Verdict

**PASS.** The new library code conforms to the applicable backend guidelines. No Critical, Important, or Minor guideline violations found. The load-bearing concurrency and correctness claims were independently verified against source, not accepted on the strength of comments.

Note on checklist applicability: these are shared-infrastructure packages (logger/bootstrap/kafka-provider), not DDD domain or sub-domain packages. The DOM file-responsibility layout checks (builder.go/rest.go/entity.go/processor.go/administrator.go), JSON:API/REST checks (DOM-04/05/08/09/17/18/19), DB-write/handler checks (DOM-13/14/15/16/27), atlas-constants (DOM-21), Docker/topic/deploy (DOM-22/23), and packet/wire checks (DOM-25) have no surface here (no model.go, no resource.go, no REST handler, no entity, no domain type, no packet). They are N/A by construction, not passed-by-omission. The checks that DO have surface are enumerated below with file:line evidence.

## Applicable checks with evidence

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Logger params typed `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `ProviderImpl(l logrus.FieldLogger)` producer/provider.go:14; `Projection.Start(... l logrus.FieldLogger ...)` projection.go:19; `ProjectionFuncs` fields projection.go:35. `CreateLogger` returning `*logrus.Logger` (logger.go:15) and `Runtime.Logger() *logrus.Logger` (bootstrap.go:84) are the root-logger constructor/handle, which is the correct concrete type — DOM-06 governs consumers, not the factory. |
| DOM-11 | Lazy provider evaluation | PASS | `ProviderImpl` returns a curried `Provider` (func(token) MessageProducer) that only resolves the writer + decorators when invoked per-token, over lazy `Produce`/`ManagerWriterProvider` (provider.go:14-22). No eager execution. |
| DOM-12 | No `os.Getenv` in request handlers | PASS (N/A surface) | `os.Getenv`/`LookupEnv` appear only in startup/config code — logger.go:20 (LOG_LEVEL), projection.go:67-68 (topics), projection.go:105 (catch-up timeout). None are HTTP handlers; this is legitimate bootstrap config reading. |
| DOM-20 | Table-driven tests | PASS | `TestNormalizeFieldKey` fieldnorm_test.go:9-32 and `TestParseProjectionCatchupTimeout` projection_test.go:112-129 use the `tests := []struct{...}` + loop idiom. Integration-style lifecycle tests (bootstrap/projection) are legitimately scenario-shaped. |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | `provider_test.go:19` installs a `MockWriter` via `GetManager(ConfigWriterFactory(...))` — the emit at :31 hits the mock, not a real writer, so no 10-retry/42s hang. This is the producer package's OWN unit test verifying header composition (needs to inspect written messages); `producertest.InstallNoop()` (the shared stub for SERVICE test packages) discards messages and could not verify headers, so it is correctly not used here. `ResetInstance` is in `t.Cleanup` (:16) — acceptable because this test package is the producer singleton's home, not a downstream consumer relying on a TestMain stub. |
| DOM-26 | Goroutines via `routine.Go` | PASS | No bare `go` statement in any non-test lib file (grep clean). The only production goroutine is `routine.Go(...)` in the pre-existing teardown.go:44 (unchanged by this task). Bare `go func()` in logger_test.go:52 and zz_lifecycle_test.go:23 are `_test.go`, explicitly excluded by DOM-26. |
| SEC-04 | No hardcoded secrets | PASS | No keys/passwords/tokens in any new file. Config is env-sourced. (Service is not auth/token-related; SEC-01/02/03 N/A.) |
| FILE-* | File responsibility layout | PASS (N/A domain surface) | These are shared-lib packages, not domain/sub-domain packages, so the domain file table does not apply. The new files are nonetheless cleanly separated by concern (bootstrap.go / projection.go / logger.go / fieldnorm.go / producer/provider.go); no catch-all `<pkg>.go` bundling multiple responsibilities. |

## Concurrency / correctness verifications (independently confirmed, not trusted from comments)

1. **Readiness controller atomic access — SAFE.** `Runtime.shuttingDown` is `atomic.Bool` (bootstrap.go:39). Written once by the teardown closure via `.Store(true)` (bootstrap.go:73), read by HTTP-facing `Ready()` via `.Load()` (bootstrap.go:96). `Runtime` is used exclusively through `*Runtime` (constructed `&Runtime{...}` bootstrap.go:58; all methods pointer-receiver), so the `atomic.Bool` value field is never copied — no `go vet copylocks` hazard, no torn access. `gates` slice is set once at Bootstrap and only read thereafter. Verified by `TestBootstrapReadinessGatesAnd` under `-race`.

2. **fieldnorm in-place `entry.Data` mutation — SAFE.** The central safety claim (fieldnorm.go:14-17) was verified against the actual module source `github.com/sirupsen/logrus@v1.9.4/entry.go`: `Entry.log()` (:224) calls `newEntry := entry.Dup()` (:227) then `newEntry.fireHooks()` (:245); `Dup()` (:83-89) allocates a FRESH `Fields` map and copies entries. Therefore both the normalizer (fieldnorm.go:24-46) and `serviceNameHook.Fire` (logger.go:41-44) mutate the per-emission copy's map, never a shared derived `*Entry`. Confirmed empirically by `TestCreateLoggerSharedEntryParallelEmitNoRace` (8 goroutines × 100 emits on one shared entry) passing under `-race`. Hook ordering is correct: serviceNameHook registered first (logger.go:18) adds the dotted `service.name` which the normalizer (registered last, logger.go:25) passes through unchanged (dot short-circuit, fieldnorm.go:52). Collision rule (snake_case wins) is deterministic via the pre-sort (fieldnorm.go:35).

3. **Projection group-id / catch-up semantics — CORRECT.** Per-process group id `"<base> - projection - <uuid>"` (projection.go:77) guarantees FirstOffset replay of the compacted log on every container start (verified by regex assertion, projection_test.go:52-55). Catch-up timeout parsing defaults to 5m and silently keeps default on empty/invalid/non-positive (projection.go:103-114; table test projection_test.go:112-129). Timeout → `Fatal` (projection.go:96); missing-option → `panic` programmer-guard (projection.go:90-91) — both intentional per task context and covered by tests.

## Non-blocking observations (NOT guideline violations — recorded for completeness only)

- `startProjection` (projection.go:70-72) warns when `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` is empty but is silent when `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` is empty. This asymmetry is deliberate (tenant-status drives live config propagation) and violates no guideline; flagging only so a future reader knows it is intentional, not an oversight.
- The teardown singleton's unbuffered signal channel (`make(chan os.Signal)`, teardown.go:37) is the classic "signal.Notify wants a buffered channel" shape, but teardown.go is PRE-EXISTING and untouched by task-118 — out of scope, not a new defect.

**Overall guideline verdict for the new library code: PASS (READY).**
