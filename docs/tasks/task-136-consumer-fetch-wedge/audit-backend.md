# Backend Audit — atlas-kafka/consumer (task-136-consumer-fetch-wedge)

- **Scope:** changed Go package `libs/atlas-kafka/consumer` (`manager.go`, `debug.go`, `config.go`, `config_test.go`, `timing_test.go`, `idle_stuck_test.go`, `dwell_integration_test.go`, plus additive `manager_test.go` warn tests)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-08
- **Build:** PASS (`go build ./...` clean)
- **Vet:** PASS (`go vet ./...` clean)
- **Tests:** PASS (`go test -race ./...` — consumer package 6.057s, all packages ok)
- **Overall:** PASS

## Package Classification (Phase 2)

`libs/atlas-kafka/consumer` has no `model.go`, `resource.go`, `entity.go`,
`processor.go`, or `provider.go`. It is a **low-level Kafka consumer library
(Support package)**, not a DDD domain or sub-domain service. Per the Phase 2
rules the DOM-* / SUB-* / SEC-* domain checklists do not apply mechanically;
they are marked N/A with reason below. The audit focuses on the dimensions
that DO apply to library code: concurrency/mutex correctness, error handling,
logging conventions, interface/impl patterns, immutability/functional
composition, goroutine/context lifecycle, and test quality.

## Applicable Findings

| Area | Status | Evidence |
|------|--------|----------|
| Mutex correctness — all new mutable state guarded | PASS | New fields `idleTicks/noProgressTicks/lastIdleTickAt/lastNoProgressAt/readerCreatedAt/awaitingFirstFetch/timeToFirstFetch/last+maxFetchDuration/last+maxHandlerDuration/totalBackoff` declared under the "protected by mu" block (manager.go:257-273); every writer (`recordIdleTick` 364, `recordNoProgressTick` 375, `recordFetchDuration` 396, `recordHandlerDuration` 405, `recordBackoff` 414, `recordFetch` 349, `onReaderCreated` 335) acquires `c.mu`; the sole reader `Snapshot()` (manager.go:321) copies all fields under `c.mu`. `-race` clean. |
| Stats() ownership / no concurrent delta readers | PASS | `readerMadeProgress` (manager.go:63) is the only `Stats()` caller and runs solely on the fetch-loop goroutine via `handleFetchDeadline` (manager.go:424, called at 553 serial / 647 parallel). Reader is never exposed externally; `DebugHandler` reads `Snapshot()` not `Stats()` (debug.go:26). Ownership documented at manager.go:52-56. |
| Error handling / classification | PASS | `context.Canceled`/parent-ctx handled before deadline branch (manager.go serial 549-551, parallel); `context.DeadlineExceeded` routed to `handleFetchDeadline`; sentinel `errFetchWedged` (manager.go:24) returned only at threshold; commit errors logged and tolerated (at-least-once) at manager.go:568/610. |
| Logging conventions | PASS | Uses `logrus.FieldLogger` interface throughout, not `*logrus.Logger`. Level choice correct: idle tick → Debug (manager.go:428), no-progress stall suspect → Warn (441), wedge → Warn (433), misconfig guard → one-time Warn (manager.go:153). |
| Interface/impl pattern | PASS | New `StatsProvider` interface (manager.go:55) with graceful type-assertion fallback for non-implementers (test mocks) documented as conservative legacy behavior (manager.go:59-62). |
| Immutability / functional composition | PASS | `Config` decorators (`SetMaxWait`/`SetFetchTimeout`/etc.) are pure value-receiver functions returning a copy (config.go:68-127); `model.Decorator[Config]` pattern preserved. |
| No leaked goroutines | PASS | `start` goroutine exits on ctx cancel and is tracked by `wg` (manager.go:330-331); parallel handlers bounded by `sem` (cap `maxInFlight`) and `maxQueue` back-pressure; per-message handler goroutines awaited via `handlerWg.Wait()` in `processMessage`. |
| Context handling | PASS | Per-call `fetchCtx` with `context.WithTimeout` and explicit `cancelFetch()` each iteration (not `defer` in a loop, so no accumulation) — serial manager.go:544-546, parallel 638-640. Recreate backoff selects on `ctx.Done()` (manager.go:507). |
| Additive Snapshot/debug fields wired consistently | PASS | `Snapshot` struct (manager.go:291-301) ↔ `Snapshot()` copy (321-331) ↔ `debugAttributes` (debug.go:68-79) ↔ `snapshotToAttributes` — one-to-one, no dropped field; `Ns`-suffixed JSON keys for durations. |
| Test quality — table/behavioral coverage, error+happy paths, log assertions | PASS | Idle-never-wedges (idle_stuck_test.go:67), no-progress-escalates-with-warn (idle_stuck_test.go:130), interleave-resets (idle_stuck_test.go:194), phase-timing population (timing_test.go:36), misconfig warn fires (manager_test.go:1369) AND stays silent on healthy defaults (manager_test.go:1411); default-value assertions updated (config_test.go). `-race` clean. |
| DOM-06 (FieldLogger, not *Logger) — spirit applies to library | PASS | All entry points take `logrus.FieldLogger` (manager.go:101, 329, 424, 534). |

## Checklist Items Marked N/A (with reason)

- **DOM-01..05, DOM-16, DOM-18, DOM-19 (builder/ToEntity/Make/Transform/TransformSlice/administrator/JSON:API model/flat request):** N/A — no domain model, entity, or REST resource in this package.
- **DOM-07, DOM-08, DOM-12, DOM-13, DOM-14, DOM-15, DOM-17 (handlers/RegisterInputHandler/os.Getenv/cross-domain/provider calls/db writes/error→HTTP):** N/A — no `resource.go` HTTP handlers. The one HTTP surface (`DebugHandler`, debug.go:15) is a read-only GET, method-guarded, tenant-agnostic, no DB. Grep confirms zero `os.Getenv` in package.
- **DOM-10, DOM-11 (tenant callbacks / lazy providers):** N/A — no GORM, no DB, no multi-tenancy layer in this library.
- **DOM-20 (table-driven tests):** Partially applicable — tests are behavioral/scenario-driven rather than strict `tests := []struct{}` tables, which is appropriate for concurrency lifecycle testing; not a violation for library code.
- **DOM-21 (atlas-constants duplication):** N/A — no new domain/id/enum types; only Kafka-timing durations and counters.
- **DOM-22, DOM-23 (Dockerfile lib mentions / Kafka topic configmap):** N/A — shared library, no service Dockerfile or k8s manifest in scope.
- **DOM-24 (Kafka producer stubbed in emit tests):** N/A — this is the consumer side; grep confirms zero `AndEmit` / `message.Emit` / `producer.Produce` in the package's test files. No emit path to stub.
- **SUB-01..04:** N/A — not a sub-domain (action-event) package.
- **SEC-01..04:** N/A — not an auth/token service.
- **EXT-*, SCAFFOLD-*:** N/A — no external atlas-service HTTP client, no new service scaffold.

## Non-Blocking Observations (not guideline violations)

1. **Test cleanup ordering (minor robustness).** `idle_stuck_test.go` and
   `timing_test.go` register `defer wg.Wait()` and call `cancel()` only as the
   last non-deferred statement (idle_stuck_test.go:78/123, :140/188, :203/231;
   timing_test.go:47/97). On an assertion failure (`t.Fatalf`) `cancel()` is
   skipped, so the deferred `wg.Wait()` blocks on a still-running consumer
   goroutine until the `go test` timeout instead of failing fast. The newer
   tests use the robust `defer func(){ cancel(); wg.Wait() }()` form
   (dwell_integration_test.go:217; manager_test.go:1380). Cosmetic; all tests
   currently pass.

2. **`Dials > 0` counts as progress (design decision, documented).**
   `readerMadeProgress` (manager.go:70) treats dial activity as
   "made progress → idle/healthy", so a reader stuck in a broker-unreachable
   dial-retry loop is never force-recreated by this path. This is intentional
   (recreate does not help an unreachable broker; kafka-go owns reconnection)
   and matches the task design; flagged only for reviewer awareness, not a
   defect.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should consider)
- Test cleanup ordering in `idle_stuck_test.go` / `timing_test.go` (observation 1).

**Overall: PASS.** Build, vet, and `-race` tests are clean. The change is
additive and correctly synchronized: every new counter/timing field is
mutex-protected on both write and snapshot paths, `Stats()` is called by a
single owning goroutine with documented ownership, the idle-vs-no-progress
reclassification preserves the existing wedge-recreate contract (threshold
still `maxConsecutiveTimeouts`), and the `maxWait >= fetchTimeout` guard is a
non-mutating Warn as designed. No Critical or Important findings.
