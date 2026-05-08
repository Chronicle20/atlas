# Backend Audit — atlas-wz-extractor (task-062-wz-extractor-parallelism)

- **Service Path:** services/atlas-wz-extractor/atlas.com/wz-extractor
- **Branch:** task-062-wz-extractor-parallelism
- **Review Range:** 427ff8e2462cb3724c559f8251bc9fa26ef5c76a..3d4c6379d951ca263ff68c133d576365583e5f89 (24 commits, 21 Go-code commits)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/`
- **Date:** 2026-05-08
- **Build:** PASS
- **Tests:** PASS — every package under `services/atlas-wz-extractor/atlas.com/wz-extractor/` compiled and tested green (no failed tests, no skipped tests).
- **Vet:** PASS (clean)
- **gofmt:** FAIL — `extraction/job/model.go` has unaligned column drift (lines 98-101). All other flagged files are pre-existing repo drift outside this task's scope.
- **Overall:** NEEDS-WORK

## Build & Test Results

```
go build -C services/atlas-wz-extractor/atlas.com/wz-extractor ./...
# silent — exit 0

go test -C services/atlas-wz-extractor/atlas.com/wz-extractor ./... -count=1
?   	atlas-wz-extractor	[no test files]
ok  	atlas-wz-extractor/characterimage	0.011s
ok  	atlas-wz-extractor/characterrender	0.019s
?   	atlas-wz-extractor/cmd/map-render-spike	[no test files]
ok  	atlas-wz-extractor/extraction	0.058s
ok  	atlas-wz-extractor/extraction/job	0.035s
ok  	atlas-wz-extractor/extraction/lock	0.022s
ok  	atlas-wz-extractor/image	0.005s
?   	atlas-wz-extractor/kafka/consumer	[no test files]
ok  	atlas-wz-extractor/kafka/consumer/extraction	0.030s
?   	atlas-wz-extractor/kafka/message/extraction	[no test files]
?   	atlas-wz-extractor/kafka/producer	[no test files]
?   	atlas-wz-extractor/logger	[no test files]
ok  	atlas-wz-extractor/mapimage	0.003s
?   	atlas-wz-extractor/rest	[no test files]
ok  	atlas-wz-extractor/wz	0.004s
…

go vet -C services/atlas-wz-extractor/atlas.com/wz-extractor ./...
# silent — exit 0
```

## Domain Discovery

This service is "verb-centric": there is no `extraction/model.go`, and the new task introduces five surfaces:

| Surface | Type | Notes |
|---------|------|-------|
| `extraction/` | Sub-domain (handler + dispatcher + processor for the Extract verb) | No `model.go`/`builder.go`/`entity.go` — the closest thing to a domain model is the new `extraction/job` sub-package. Most domain checklist items map onto the sub-domain checklist. |
| `extraction/job/` | Domain (immutable model + builder + Redis store) | Has `model.go`, `store.go`, `keys.go`. No GORM entity (Redis-only), so DOM-02/DOM-03 are N/A. |
| `extraction/lock/` | Support (Redis primitive helper) | No model. Skip checklist; treat as a wrapper. |
| `kafka/consumer/extraction/`, `kafka/producer/`, `kafka/consumer/` | Kafka transport | Mirror atlas-data convention (verified). |
| `kafka/message/extraction/` | Message contract | Producer-side message provider only. |

## DOM-* Checklist — `extraction/job/` (Redis-backed domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | Builder pattern | PASS | `extraction/job/model.go:60-79` (`JobBuilder`), `extraction/job/model.go:96-107` (`UnitBuilder`). Fluent setters return `*Builder`; `Build()` returns the immutable struct value. No setters mutate after Build (Build returns a copy). |
| DOM-02 | `ToEntity()` | N/A | No GORM entity (Redis-only persistence). The shape of `unitJSON` at `store.go:40-45` is the on-wire form. |
| DOM-03 | `Make(Entity)` | N/A | Same as DOM-02. The Redis hydrator `unitFromJSON` at `store.go:63-83` is the equivalent. |
| DOM-04 | `Transform` (rest) | N/A — domain is internal; not exposed as JSON:API. Closest exposure is `extraction/job_handler.go` which builds an envelope inline (see DOM-18). |
| DOM-05 | `TransformSlice` | N/A — see DOM-04. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `extraction/processor.go:25,31` — both `Extract` and `ExtractUnit` take `logrus.FieldLogger`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `dispatcher.go:35,129` and `job_handler.go:60` use `d.Logger()`. No `logrus.StandardLogger()` anywhere. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | FAIL | `resource.go:36` registers POST `/wz/extractions` via `register("create_extraction", handleExtract(...))` where `register := rest.RegisterHandler(l)(si)` (`resource.go:33`). Plain `RegisterHandler` is used for a body-bearing POST; the dispatcher does not parse a JSON body so the practical risk is low, but per `patterns-rest-jsonapi.md` POST should go through `RegisterInputHandler[T]`. The PATCH `/wz/input` upload at `resource.go:41` is also `RegisterHandler`, but multipart upload is the documented exception. **Severity: Minor.** |
| DOM-09 | Transform errors handled | PASS | No `Transform()` calls in `resource.go`/`dispatcher.go`. The JSON encode at `dispatcher.go:133` uses `_ = json.NewEncoder(...).Encode(...)`; the `_=` discard on a success-path JSON encode is consistent with project convention. |
| DOM-10 | Test DB has tenant callbacks | N/A | No GORM/database in this service. |
| DOM-11 | Providers use lazy evaluation | PASS | `kafka/message/extraction/kafka.go:29-41` uses `producer.SingleMessageProvider` (lazy `model.Provider[[]kafka.Message]`). |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `resource.go`, `dispatcher.go`, `job_handler.go` contain zero `os.Getenv` calls. The only env reads in this PR are in `processor.go:122` (`WZ_EXTRACT_PARALLELISM`, configuration plane) and `kafka/consumer/consumer.go:26` (`BOOTSTRAP_SERVERS`, transport config). Acceptable. |
| DOM-13 | No cross-domain logic in handlers | PASS | `dispatcher.go` orchestrates only the extraction-job lifecycle (job store + lock + producer); no other domain. |
| DOM-14 | Handlers don't call providers directly | PASS — but with one nuance | `dispatcher.go:42` uses `filepath.Glob(...)` to enumerate WZ files directly in the handler instead of delegating to the processor. This is semantically a "provider call" against the filesystem and arguably should live on the processor (the processor's `Extract` does the same glob at `processor.go:53`). **Severity: Minor — cohesion smell.** |
| DOM-15 | No direct entity creation in handlers | PASS — but see DOM-14 | `dispatcher.go:64-126` writes Redis state via `store.Create / MarkJobRunning / MarkUnitsSkippedByStatus / MarkJobTerminal`, which is the correct administrator-equivalent layer. The handler does NOT touch Redis directly. |
| DOM-16 | `administrator.go` exists for writes | PASS (equivalent) | `extraction/job/store.go` plays the administrator role for Redis writes. Not named `administrator.go`, but Redis-backed services in this monorepo do not follow the GORM administrator naming convention. Acceptable. |
| DOM-17 | Domain-error → HTTP-status mapping | PASS | `dispatcher.go` maps no-files → 400 (line 49), redis unreachable → 503 (line 59), lock held → 409 (line 63), publish failure → 500 (line 124), other internal → 500. `job_handler.go:55-56` maps `job.ErrNotFound` → 404, generic → 500. Matches PRD §7.3 contract. |
| DOM-18 | JSON:API interface on REST models | FAIL | `extraction/job_handler.go:14-47` declares anonymous structs (`jobResource`, `jobEnvelope`) and serializes via `json.NewEncoder` rather than implementing `MarshalIdentifier`/`UnmarshalIdentifier`/`GetName`/etc. and routing through `api2go/jsonapi`. The handler does set `Content-Type: application/vnd.api+json` (line 117) but never goes through the project-standard transform layer. Other Atlas services (e.g., atlas-monsters) use `jsonapi` even for read-only resources. **Severity: Important — drift from project REST convention. Behavior is observable to clients (no `relationships`/`links` blocks).** |
| DOM-19 | Request models use flat structure | N/A | POST has no body. |
| DOM-20 | Table-driven tests | PARTIAL | `processor_test.go:138-197` (`TestExtract_TenantPathFormat`) uses table-driven form. Most other new tests (`store_test.go`, `dispatcher_test.go`, `tenant_lock_test.go`, `consumer_test.go`) are individually-named `TestX_Y` cases rather than table-driven. Per guidelines this is "should", not "must"; the cases are heterogeneous enough that the per-case naming is defensible. **No fail.** |
| DOM-21 | No duplication of atlas-constants types | PASS | The new types in this PR — `JobStatus` (`extraction/job/model.go:6-14`), `UnitStatus` (`model.go:17-25`), `Counters` (`model.go:111-117`) — are workflow lifecycle types that have no equivalent under `libs/atlas-constants/` (which covers item/inventory/world/channel/character/job/skill/monster IDs). The dispatcher consumes `tenant.Region`/`MajorVersion`/`MinorVersion` directly off the tenant model rather than redeclaring them. |

## SUB-* Checklist — `extraction/` (parent verb-domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Has processor or uses parent processor | PASS | `extraction/processor.go:20-46` defines `Processor` interface with `Extract`/`ExtractUnit` and an impl. |
| SUB-02 | Has administrator for writes | PASS (equivalent) | Filesystem writes flow through `processor.go:81-117` (`ExtractUnit`); Redis writes flow through `extraction/job/store.go`. |
| SUB-03 | Uses `RegisterInputHandler[T]` for POST | FAIL — duplicate of DOM-08 | See DOM-08. |
| SUB-04 | No manual JSON parsing | PASS | `dispatcher.go` reads no body. `upload.go` uses `r.MultipartReader()` (the supported transport for multipart). No `json.NewDecoder(r.Body)` / `io.ReadAll(r.Body)` anywhere. |

## Kafka Transport Conformance

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| K-01 | Curried `InitConsumers(l)(cmf)(groupId)` | PASS | `kafka/consumer/extraction/consumer.go:25-35`. Matches atlas-data verbatim. |
| K-02 | Curried `InitHandlers(l)(deps...)(rf)` | PASS | `kafka/consumer/extraction/consumer.go:37-47`. Style consistent with project. |
| K-03 | `kafka/consumer/consumer.go` mirrors atlas-data | PASS | Confirmed identical to `services/atlas-data/atlas.com/data/kafka/consumer/consumer.go` (compared byte-for-byte at `consumer.go:1-28`). |
| K-04 | `kafka/producer/producer.go` mirrors atlas-data | PASS | Confirmed identical to `services/atlas-data/atlas.com/data/kafka/producer/producer.go` (`producer.go:1-23`). |
| K-05 | `EnvCommandTopic` constant duplicated across producer + consumer | WARN | `kafka/consumer/extraction/kafka.go:4` and `kafka/message/extraction/kafka.go:10` both declare `EnvCommandTopic = "COMMAND_TOPIC_WZ_EXTRACTION"`. Drift risk if one is renamed. **Severity: Minor — cosmetic.** |
| K-06 | Header parsers (Span + Tenant) on consumer | PASS | `kafka/consumer/extraction/consumer.go:30` sets `consumer.SpanHeaderParser, consumer.TenantHeaderParser`. |
| K-07 | Header decorators on producer | PASS | `kafka/producer/producer.go:17-18` sets `SpanHeaderDecorator + TenantHeaderDecorator`. |
| K-08 | Consumer is idempotent on redelivery | PASS | `kafka/consumer/extraction/consumer.go:56-64` calls `MarkUnitRunning` and bails when `claimed=false`. Redelivery test exists at `consumer_test.go:82-107`. |
| K-09 | Producer keying for partition affinity | PASS | `kafka/message/extraction/kafka.go:30,46-52` keys messages with `djb2(jobId)` so per-job ordering is preserved when partition count permits, while still distributing across partitions. |
| K-10 | Producer context for header decorators | PASS — but observe lifecycle | `main.go:74` builds `prodProvider` from `context.Background()`. Acceptable for a service-lifetime producer (matches atlas-data's pattern). The per-request span/tenant headers come from the consumer's `tdm.Context()` indirectly via the WithContext middleware. |

## Idempotency / Atomicity — Redis transactions in `extraction/job/store.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| TX-01 | `MarkUnitRunning` uses WATCH/MULTI/EXEC + bounded retries | PASS | `store.go:201-245`. 5-attempt retry on `goredis.TxFailedErr`; non-`TxFailedErr` errors return immediately; exhaustion returns explicit error. No infinite loop. |
| TX-02 | `FinalizeUnit` uses WATCH/MULTI/EXEC + bounded retries | PASS | `store.go:254-345`. Same 5-attempt bound. |
| TX-03 | Redelivery double-increment guard | PASS | `store.go:268-281` — when current unit is already terminal, returns existing counters without HIncrBy. `store_test.go:188-209` exercises this explicitly. |
| TX-04 | `MarkJobTerminal` uses WATCH + status guard | PASS | `store.go:347-383`. Only transitions `Running → terminal`; second call returns `claimed=false` (test at `store_test.go:211-243`). |
| TX-05 | `MarkUnitsSkippedByStatus` is atomic | FAIL | `store.go:390-419`: `HGetAll` is OUTSIDE the `TxPipeline`, so a concurrent consumer can flip a unit `Pending → Running` between the read and the pipelined writes. The pipeline then overwrites that consumer's `Running` with `Skipped`, double-counting could ensue (consumer's later `FinalizeUnit` would see `Skipped` in the redelivery branch and read stale counters). In practice the dispatcher only invokes this on the partial-publish failure path, where the consumer for that unit either (a) hasn't received the message (race window unlikely) or (b) already won `MarkUnitRunning` (so the unit is `Running` and would be wrongly clobbered). **Severity: Important — correctness bug under race; missed in tests.** |
| TX-06 | `Create` uses TxPipeline | PASS | `store.go:104-125`. HSet + Expire batched. |
| TX-07 | Counter re-read after pipeline EXEC | PASS | `store.go:301-329` — pipeline returns `*StringCmd`s populated with post-EXEC values; the inline `_ = newCounter` comment notes the increment was already applied. Verified semantically. |

## Lock semantics — `extraction/lock/tenant_lock.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| LOCK-01 | Acquire is atomic SETNX-with-owner | PASS | `tenant_lock.go:27-33`. Uses `SetNX(key, owner, ttl)`. |
| LOCK-02 | Refresh is owner-checked Lua | PASS | `tenant_lock.go:36-50`. Lua compares GET to ARGV[1] before EXPIRE. |
| LOCK-03 | Release is owner-checked Lua | PASS | `tenant_lock.go:53-63`. Lua compares GET to ARGV[1] before DEL. Exercised by `tenant_lock_test.go:53-74` (wrong-owner release no-op). |
| LOCK-04 | Lock key is tenant-scoped | PASS | `extraction/job/keys.go:18-20` — `LockKey(tenantId, region, major, minor)` includes all four discriminators. |
| LOCK-05 | Lua `KEYS`/`ARGV` indexing correct | PASS | refreshLua uses `KEYS[1]` + `ARGV[1]` (owner) + `ARGV[2]` (seconds) — both args are passed in that order at line 49. releaseLua uses `KEYS[1]` + `ARGV[1]` — passed at line 62. |

## Goroutine / Lifecycle — `extraction/dispatcher.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GR-01 | `startLockRefresh` exits cleanly on terminal job | PASS — for the happy path | `dispatcher.go:160-162`: returns when `j.Status() != JobRunning && j.Status() != JobPending`. Consumer-side `MarkJobTerminal` triggers this. |
| GR-02 | `startLockRefresh` exits on Refresh failure | PASS | `dispatcher.go:163-166`: returns on Refresh error (covers Redis going away or another owner stealing the lock). |
| GR-03 | `startLockRefresh` honours service shutdown | FAIL | `dispatcher.go:151` uses `context.Background()`. The goroutine has no link to `tdm.Context()` and is not registered in `tdm.WaitGroup()`. On service shutdown, this goroutine leaks until either the job is finalized (driven by Kafka traffic that may not be coming since consumers also shut down) or the 24h job-hash TTL expires causing `Get` to return `ErrNotFound`. Worst case: a pod that took a request just before SIGTERM keeps a `time.Ticker` and Redis client alive in the background indefinitely (in practice bounded by lockTTL=60min once Refresh errors against a closed Redis pool, but the Redis client is closed in `main.go:58 (defer)` so the next Refresh after shutdown will fail and exit). **Severity: Minor — not a true leak in production, but the `_ = wg` at `resource.go:30` shows the `wg` parameter is dead code; either wire the goroutine into it or remove it.** |
| GR-04 | `runPool` propagates ctx cancellation | PASS | `pool.go:29-31` checks `ctx.Err()` per job; `pool.go:39-41` skips remaining work on cancellation. `wg.Wait()` ensures clean exit. |
| GR-05 | Consumer goroutine respects redelivery | PASS | `consumer/extraction/consumer.go:56-99` returns early on redelivery and on store errors (so atlas-kafka can redeliver on the next epoch). |

## REST surface

| Concern | Status | Evidence |
|---------|--------|----------|
| 202 on accept | PASS | `dispatcher.go:132` |
| 400 on empty input | PASS | `dispatcher.go:49` |
| 409 on lock conflict | PASS | `dispatcher.go:63`, exercised at `dispatcher_test.go:157-182` |
| 503 on redis unavailable | PASS | `dispatcher.go:59` |
| 500 on publish failure with cleanup | PASS | `dispatcher.go:120-126` (skip pending units, mark job failed, release lock) |
| 404 on unknown job | PASS | `job_handler.go:55-57`, exercised at `job_handler_test.go:25-45` |
| Error responses don't leak internals | PARTIAL | Strings are short ("internal error", "kafka publish failed", "redis unavailable") — acceptable. The empty-input message at `dispatcher.go:49` includes the `/api/wz/input` path; that's a deliberate hint for operators, not an information leak. |

## SEC-* Checklist (relevant items)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC-01 | Lua input handling — injection-safe | PASS | Both Lua scripts use `KEYS`/`ARGV` parameter passing (`tenant_lock.go:36-42, 53-59`). No string concatenation; the lock key flows through `Eval(... []string{key} ...)` so it cannot escape into the Lua source. |
| SEC-02 | Lock-stealing semantics | PASS | Owner is generated via `uuid.NewString()` per dispatch (`dispatcher.go:53`); the consumer never spoofs an owner because Release is keyed by `c.Body.JobId` (`consumer.go:95`) which is the same uuid the dispatcher used as owner (`dispatcher.go:56`). Verified end-to-end. |
| SEC-03 | Multipart upload validation (existing) | PASS | `upload.go:76-101` rejects path separators, `..`, absolute paths, non-`.wz` extensions, dirs, non-regular files. Pre-existing; not changed by this PR but transitively exercised. |
| SEC-04 | Tenant header trust | PASS | `tenant.MustFromContext(ctx)` is the project-standard mechanism (`dispatcher.go:32`, `processor.go:49,82`); the `TenantHeaderParser` enforces presence/format upstream. |
| SEC-05 | No hardcoded secrets | PASS | grep clean. Connection details flow through `REDIS_URL`/`REDIS_PASSWORD`/`BOOTSTRAP_SERVERS`. |
| SEC-06 | TENANT_ID/region not confused | PASS | `Region()` returns string and is used as a path component, never as a `world.Id`. |

## Behavior Regression — 409-busy guard removal

Commit `5ed720e2e refactor(atlas-wz-extractor): remove in-process tenant mutex (replaced by Redis lock)` removed the in-process tenant mutex that previously gated `upload.go`. Re-reading `upload.go` after the change:

- `upload.go:141-175` (`uploadDeps.handleUpload`) no longer holds any concurrency primitive. Two parallel uploads against the same tenant will race on `extractFlat`, which `os.RemoveAll(dst)` then `MkdirAll`. The second upload's `RemoveAll` will succeed even mid-write of the first, leaving a partially-populated tenant input directory.
- The replacement Redis lock (`TenantLock`) is acquired only by the **extraction dispatcher** (`dispatcher.go:56`), not by the upload path. So the mutex was NOT replaced for uploads; it was simply removed.

**Verdict: Important — pre-existing 409-busy contract for `PATCH /api/wz/input` is silently dropped.** PRD §7 lists 202/400/500 for upload and not 409, so the contract on paper is unchanged, but two clients PATCHing simultaneously now silently corrupt the destination. **Severity: Important** — should be either (a) restored via the same Redis lock keyed differently, or (b) explicitly accepted and documented in `docs/storage.md` (today's `storage.md` doesn't mention concurrent-upload semantics).

## Test Discipline

| Concern | Status | Evidence |
|---------|--------|----------|
| No `*_testhelpers.go` files | PASS | `find -name '*_testhelpers.go'` empty in changed paths. |
| Builder used for test setup | PASS | `store_test.go:25-38`, `dispatcher_test.go`, `consumer_test.go`. |
| miniredis used appropriately | PASS | Each test creates its own `miniredis.RunT(t)` instance — no cross-test state. `RunT` auto-cleans-up via `t.Cleanup`. |
| Fake processor / fake emitter pattern | PASS | `consumer_test.go:17-29` (`fakeProcessor`), `dispatcher_test.go:33-64` (`fakeEmitter`). Both are local to `_test.go` files. |
| Tenant header coverage in handler tests | PASS | `dispatcher_test.go:99-103,164-167` exercises real `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers. |
| End-to-end coverage of redelivery semantics | PASS | `consumer_test.go:82-107` and `store_test.go:188-209`. |
| Coverage of the partial-publish failure path | FAIL | `dispatcher.go:107-126` (the partial-publish failure branch using `MarkUnitsSkippedByStatus` + `MarkJobTerminal(JobFailed)` + `Release`) has no integration test. The only failure path tested is the lock-conflict 409. **Severity: Minor — not a correctness defect, but a coverage gap for the most error-prone branch in the dispatcher.** |
| Pool concurrency bound test | PASS | `pool_test.go:13-44` empirically verifies `maxInflight <= workers`. |

## gofmt drift in this PR

- `extraction/job/model.go:98-101` — `UnitBuilder` setters' aligned column whitespace doesn't agree with the multi-line `SetCompletedAt` / `SetErrorMessage` setters that follow. `gofmt -d extraction/job/model.go` produces a 4-line diff. **Severity: Minor** — hygiene issue introduced by this PR.
- All other `gofmt`-flagged files (`characterimage/filter.go`, `image/extract_test.go`, `mapimage/decoder.go`, `wz/property/property.go`, …) are pre-existing repo drift and out of scope.

## Summary

### Blocking-Critical (must fix before merge)
None — build/tests/vet are green and there are no security-critical findings.

### Blocking-Important (must address before PR ships)
1. **TX-05 — `MarkUnitsSkippedByStatus` is not atomic.** `store.go:390-419` reads with `HGetAll` outside the `TxPipeline`, then writes; a concurrent `MarkUnitRunning` can lose. Either (a) move the per-unit decision inside a `Watch`/CAS retry, or (b) document the constraint that this method is only ever called from the dispatcher AFTER the `MarkJobTerminal` has been claimed (which is currently NOT the order — the dispatcher calls `MarkUnitsSkippedByStatus` BEFORE `MarkJobTerminal` at `dispatcher.go:121-122`).
2. **DOM-18 — `job_handler.go` does not use `api2go/jsonapi`.** Sets `Content-Type: application/vnd.api+json` but emits a hand-rolled envelope. This drifts from the rest of the monorepo's REST convention. Either implement `MarshalIdentifier`/`GetName` on a real REST model and route through `RegisterHandler`'s normal `Transform` path, or change the `Content-Type` to `application/json` and stop pretending. Behavior is observable to clients.
3. **Upload 409-busy regression.** Commit `5ed720e2e` removed the in-process mutex and did not replace it on the upload path. Concurrent `PATCH /api/wz/input` for the same tenant will race on `os.RemoveAll(dst)` mid-extract, leaving a corrupted tenant input directory. Either reintroduce a Redis lock for uploads (recommended; it's two lines using the existing `TenantLock`) or explicitly document the new behavior + test for "last-writer-wins is OK". Plan section ‑‑ doesn't acknowledge this trade-off; design.md §10 mentions the lock but only for extractions.

### Blocking-Minor (should fix in this PR; non-blocking if landed as a follow-up)
4. **DOM-08 / SUB-03 — POST `/wz/extractions` uses `RegisterHandler` instead of `RegisterInputHandler[T]`.** Practical risk is low (no body), but the convention exists for a reason; future bodies on this endpoint will hit the wrong code path.
5. **GR-03 — `startLockRefresh` goroutine is fire-and-forget with `context.Background()`.** Wire it into `tdm.Context()` and `tdm.WaitGroup()` (the `wg` parameter is already plumbed through `InitResource` but discarded at `resource.go:30`), or document the leak window in design.md.
6. **gofmt — `extraction/job/model.go`.** Run `gofmt -w extraction/job/model.go` before the PR. CI should be told to enforce this.
7. **K-05 — `EnvCommandTopic` constant duplicated across `kafka/consumer/extraction/kafka.go` and `kafka/message/extraction/kafka.go`.** Promote to a shared file under `kafka/message/extraction/` and import from the consumer package.
8. **DOM-14 — Glob in dispatcher.** `dispatcher.go:42` does the file enumeration the processor already knows how to do; either add a `Processor.ListUnits(ctx) ([]string, error)` or accept the duplication and add a comment pointing at `processor.go:53`.
9. **Test gap — partial-publish failure branch in dispatcher** (`dispatcher.go:107-126`) is uncovered. A miniredis-backed test using `fakeEmitter{failOn: 1}` (which is already plumbed through the test scaffold but not exercised) would close this in ~30 lines.

### Non-Blocking (nice-to-have)
- Consider renaming `wg *sync.WaitGroup` away from `InitResource`'s signature once `startLockRefresh` is wired into shutdown — the parameter is dead code after the mutex removal.
- The `_ = p` at `dispatcher.go:139` and the `var _ = ...` block at `dispatcher.go:173-175` are leftover scaffolding from the refactor and should be removed.
