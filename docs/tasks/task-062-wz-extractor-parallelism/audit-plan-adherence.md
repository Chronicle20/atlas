# Plan Adherence Audit — task-062-wz-extractor-parallelism

- **Plan path:** `docs/tasks/task-062-wz-extractor-parallelism/plan.md`
- **Audit date:** 2026-05-08
- **Branch:** `task-062-wz-extractor-parallelism`
- **Base:** `main` (last common ancestor: `427ff8e2`)
- **Range audited:** `427ff8e2..3d4c6379d` (24 commits — 3 phase artifacts + 21 implementation/docs commits)

## Executive summary

All 25 plan tasks are implemented, with 21 visible in dedicated commits and the remainder bundled per the plan's own merge instructions (4+5, 18+19) or completed without a commit per plan rules (24, 25). The four known controller-flagged deviations are individually verifiable, narrowly-scoped, and either fully resolved (Deviation 2) or correctly documented as transitional follow-ups (Deviation 1). All build, vet, and test checks pass cleanly. Every PRD §10 acceptance criterion maps to a corresponding commit and observable code path. **Verdict: plan-faithful, ready to merge after backend-guidelines audit completes in parallel.**

## Per-task verdicts

| # | Task | Commit | Status | Evidence |
|---|---|---|---|---|
| 0.1 | Verify cwd/branch | (prep, no commit) | PASS | branch confirmed `task-062-wz-extractor-parallelism`; cwd is the worktree |
| 0.2 | Verify clean tree | (prep, no commit) | PASS | working tree clean before each implementation commit (git log linear) |
| 1 | Add deps (redis, kafka, miniredis) | `0628666a6` | PASS | `services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod:14-19` shows `miniredis/v2 v2.37.0`, `redis/go-redis/v9 v9.19.0`, `segmentio/kafka-go v0.4.51` plus the `atlas-redis`/`atlas-kafka` lib refs |
| 2 | `extraction/job/keys.go` | `1d916d8a6` | PASS | `extraction/job/keys.go:1-21` — exports `LockKey`, package-private `jobKey`/`unitsKey`; matches plan verbatim |
| 3 | `extraction/job/model.go` | `fafa043c3` | PASS | `extraction/job/model.go:1-118` — `Job`/`Unit` immutable structs with builder pattern + `Counters` for last-one-home; `model_test.go:8,38` covers Job+Unit builders |
| 4 | Store interface skeleton | `e9fcbb81c` | PASS | Bundled with Task 5 per plan note "Tasks 4 + 5 ship in one commit"; interface at `extraction/job/store.go:16-27` |
| 5 | Store Create/Get/Delete | `e9fcbb81c` | PASS | `store.go:85` (`Create`), `store.go` (`Get`/`Delete`); `store_test.go:19` `TestStore_CreateGetDelete` |
| 6 | Store.MarkJobRunning | `b4580aacb` | PASS | `store_test.go:63` `TestStore_MarkJobRunning` |
| 7 | Store.MarkUnitRunning | `2f006e644` | PASS | WATCH-guarded; `store_test.go:88,103` cover first-time + already-terminal paths |
| 8 | Store.FinalizeUnit | `afe627618` | PASS | `store_test.go:142,170,188` cover succeeded/failed/redelivery-noop |
| 9 | Store.MarkJobTerminal CAS | `b45dbe24c` | PASS | `store_test.go:211` `TestStore_MarkJobTerminal_Once` |
| 10 | Store.MarkUnitsSkippedByStatus | `eae3c4920` | PASS | `store_test.go:245` |
| 11 | TenantLock | `b8d15a97f` | PASS | `extraction/lock/tenant_lock.go:1-64` — Acquire (SETNX), Refresh (Lua compare-and-extend), Release (Lua compare-and-delete); `tenant_lock_test.go:19,53,76` |
| 12 | pool.go | `543f57cd3` | PASS | `extraction/pool.go:1-46` generic `runPool[T]`; `pool_test.go:13,46` cover bounded concurrency + continue-on-error |
| 13 | Processor split | `b59389aa6` | PASS | `processor.go:48` (`Extract` now uses `runPool`), `processor.go:81` (`ExtractUnit`); env helper at `processor.go:121` (`parallelismFromEnv`) with `runtime.NumCPU()` fallback |
| 14 | Delete in-process mutex | `5ed720e2e` | DEVIATION-ACCEPTED | `mutex.go`/`mutex_test.go` deleted; commit message documents extension to `upload.go`. See "Deviation 1" below. |
| 15 | producer.go | `d4b9ba65c` | PASS | `kafka/producer/producer.go:1-23` — verbatim atlas-data shape with `SpanHeaderDecorator` + `TenantHeaderDecorator` |
| 16 | Consumer config helper | `f102e6641` | PASS | `kafka/consumer/consumer.go:1-27` — curried `NewConfig(l)(name)(token)(groupId)` |
| 17 | Extraction consumer | `e26cb5378` | PASS | `kafka/consumer/extraction/{kafka.go,consumer.go}` + `kafka/message/extraction/kafka.go`; `consumer_test.go:37,82,109` cover happy path / redelivery / failed unit |
| 18+19 | Dispatcher + resource refactor + job_handler | `833f97e36` | PASS | `extraction/dispatcher.go:1-176`, `extraction/job_handler.go:1-122`, `extraction/resource.go:27-45`; `dispatcher_test.go:93,128,157` cover happy path / 400 empty / 409 conflict; `job_handler_test.go:25,47` cover 404 + 200 |
| 20 | main.go wiring | `5184d6927` | PASS | `main.go:1-92` — uses `atlasredis.Connect(l)`, `lockTTL = 60 * time.Minute`, full consumer registration via `extconsumer.InitConsumers`/`InitHandlers`. See "Deviation 2". |
| 21 | docs/kafka.md | `e3b27a48e` | PASS | `docs/kafka.md:1-48` — partition recommendation ≥16, group id, header parsers, idempotency note |
| 22 | docs/storage.md | `82d8b5683` | PASS | `docs/storage.md:1-73` — full Redis schema (`wz-extractor:job:{jobId}`, `:units`, `:tenant-lock:`), TTL policy, idempotency invariants |
| 23 | Deploy manifest | `3d4c6379d` | PASS | `deploy/k8s/atlas-wz-extractor.yaml` adds `COMMAND_TOPIC_WZ_EXTRACTION=command.wz.extraction`, `WZ_EXTRACT_PARALLELISM=16`, `resources.requests/limits` matching PRD §8 (cpu 1/4, memory 2Gi/8Gi), inline note that `BOOTSTRAP_SERVERS`/`REDIS_URL` flow from env ConfigMap |
| 24 | Caller audit | (no commit per plan) | PASS | Only in-repo caller is `services/atlas-ui/src/services/api/seed.service.ts:154-160`; `runWzExtraction` only checks `response.ok` — does NOT parse the body, so no transition fix needed. See "Deviation 3". |
| 25 | Final verification | (verification only) | PASS | `go build ./... ` passes; `go vet ./...` clean; `go test ./...` all packages PASS; no `extraction.Acquire`/`tenantMutexRegistry` matches; `runExtraction` only appears in a code comment at `processor_test.go:183` |

**Completion rate:** 25/25 (100%); 1 task carries a documented scope-extension deviation (Task 14); 0 skipped without approval.

## Deviation analysis

### Deviation 1 — `upload.go` mutex removal scope extension (Task 14)

The plan wording targeted only `resource.go`'s `handleExtract` `Acquire/Release` pair. `upload.go` also held a `TryAcquire/Release` 409-busy guard. The implementer correctly identified that deleting `mutex.go` would break the upload caller, and resolved it by:

- removing the `TryAcquire`/`Release` calls from `handleUpload` (`5ed720e2e` diff `upload.go:141-150`)
- removing the now-stale `TestUpload_MutexBusy409` test (`5ed720e2e` diff `upload_test.go:204-230`)
- declaring the 409-on-busy-upload behavior to be a **transitional regression** in the commit body: *"Upload 409 conflict guard will be restored via Redis distributed lock in a later task."*

Independent assessment: **DEVIATION-ACCEPTED**. The scope extension was structurally required (you cannot delete `mutex.go` and leave callers compiling against it), narrowly-scoped (no other functionality changed), and explicitly flagged. The dropped 409 guard is genuinely a transient regression — two parallel uploads of the same tenant zip will now race on the filesystem `RemoveAll`/extract pair — but this is a separate behavior from extraction concurrency and is not in PRD §10. Recommend a follow-up task tracked in the PR body to restore upload-busy guarding via the Redis lock once dispatcher behavior is confirmed stable in production. Not a merge blocker.

### Deviation 2 — Partial main.go in dispatcher commit, fully replaced in Task 20

`833f97e36` (Tasks 18+19) introduced a working but interim `main.go` using `goredis.NewClient` directly and a `30 * time.Minute` lock TTL, purely to keep the build green. `5184d6927` (Task 20) then replaced that with the spec'd implementation: `atlasredis.Connect(l)`, `lockTTL = 60 * time.Minute`, full `extconsumer.InitConsumers`/`InitHandlers` registration, and `kproducer.ProviderImpl` wiring.

Independent assessment: **DEVIATION-ACCEPTED**. The final state of `main.go` (verified at `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go:1-92`) contains:
- `atlasredis.Connect(l)` at line 57 (no direct `goredis.NewClient`)
- `lockTTL = 60 * time.Minute` at line 37 (no `30 * time.Minute`)
- consumer registration at lines 66-70

`grep -nE "30\s*\*\s*time\.Minute|goredis\.|go-redis/v9" services/.../main.go` returns zero matches. The interim shape is fully overwritten. The bundling pattern (build-green at every commit, then converge on the spec'd shape) is good engineering hygiene. Not a merge blocker.

### Deviation 3 — No Task 24 commit; only atlas-ui calls the endpoint and does not parse body

Task 24 instructed grepping the repo for callers of `POST /api/wz/extractions` and updating any that parse `{"status":"started"}`. The implementer's audit found one in-repo caller — `services/atlas-ui/src/services/api/seed.service.ts:154-160`'s `runWzExtraction` — which only checks `response.ok` and discards the body. No `.go`/`.sh`/`.py`/`.yaml` callers exist. Plan §24.4 explicitly says "skip the commit" if no changes are required.

Independent assessment: **DEVIATION-ACCEPTED**. Verified directly: `grep -rn "wz/extractions" --include="*.ts" services/atlas-ui` returns only the `runWzExtraction` and `getExtractionStatus` lines, and `runWzExtraction`'s body never reads `await response.json()`. No external scripts found in `deploy/`. The frontend will need a forward-looking enhancement to capture `jobId` and poll `GET /jobs/{jobId}` to display per-unit progress, but that is a UI feature, not a back-compat fix. Not a merge blocker.

### Deviation 4 — `mockProcessor` retained for status_test.go and upload_test.go

Plan Task 19.4 said to delete `mockProcessor` along with the obsolete dispatcher tests in `resource_test.go`. The implementer kept a minimal `mockProcessor` (`resource_test.go:20-32`) because `status_test.go` and `upload_test.go` still need a `Processor` to construct the router via `setupRouterWithDirs`.

Independent assessment: **DEVIATION-ACCEPTED**. Verified at `resource_test.go:20-32`: the kept mock has zero behavior — both `Extract` and `ExtractUnit` are `return nil` no-ops. It does not exercise any of the deleted POST-handler logic. `status_test.go:38,69,113,156` and `upload_test.go:79,111,143,162,182,209` legitimately need an `extraction.Processor` to wire the resource. The kept stub is the smallest surface that keeps adjacent tests compiling. Plan-pragmatic, not a merge blocker.

## PRD §10 acceptance criteria mapping

| PRD §10 requirement | Status | Evidence |
|---|---|---|
| `extraction/processor.go` no longer iterates `wzFiles` serially; `ExtractUnit` exists | PASS | `processor.go:75-78` uses `runPool`; `processor.go:81` defines `ExtractUnit` |
| `WZ_EXTRACT_PARALLELISM` honored with `runtime.NumCPU()` fallback + warning on invalid | PASS | `processor.go:121-132` (`parallelismFromEnv`) |
| `POST /api/wz/extractions` publishes one `START_EXTRACTION_UNIT` per WZ file with fresh jobId, returns 202 `{jobId, unitsTotal, status}`, no synchronous unit work | PASS | `dispatcher.go:107-118` (publish loop); `dispatcher.go:131-137` (response shape); no synchronous extract call |
| Kafka consumer in group `wz-extractor-extraction` handles `START_EXTRACTION_UNIT` via `ExtractUnit` and updates Redis | PASS | `kafka/consumer/extraction/consumer.go:49-101` (`handleStartExtractionUnit`); `main.go:25` (group id constant) |
| Two concurrent POSTs for same tenant → exactly one 202 + one 409 | PASS | `dispatcher.go:56-65` enforces 409; `dispatcher_test.go:157` `TestDispatcher_LockConflict409` |
| Terminal job status ∈ {completed, completed_with_errors, failed} per §4.6 + lock release | PASS | `consumer.go:82-99` selects terminal status and calls `tl.Release` on CAS-winner; `consumer_test.go:37,109` |
| `GET /api/wz/extractions/jobs/{jobId}` returns wzExtractionJob JSON:API resource correctly from any pod | PASS | `job_handler.go:49-121` reads only Redis; `job_handler_test.go:47` |
| `GET /api/wz/extractions/jobs/{unknown}` → 404 | PASS | `job_handler.go:55-58`; `job_handler_test.go:25` |
| Unit-level failure does not abort siblings; `unitsFailed > 0` → `completed_with_errors` | PASS | `consumer.go:67-88` (terminal selection); `consumer_test.go:109` `TestHandler_FailedUnit_MarksJobFailed` |
| Pod crash mid-unit → redelivery → no double-count, no stuck running | PASS | `store.go` WATCH/MULTI/EXEC guards on MarkUnitRunning + FinalizeUnit; CAS in MarkJobTerminal; `store_test.go:188,211` |
| Deployment manifest declares `resources` + supports `replicas > 1` (kept at 1) | PASS | `deploy/k8s/atlas-wz-extractor.yaml` adds resources block; replicas remains the existing default |
| Topic creation / partition count documented | PASS | `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/kafka.md:7` (partitions ≥ 16) |
| Redis schema documented | PASS | `services/atlas-wz-extractor/atlas.com/wz-extractor/docs/storage.md` |
| Per-unit logs include `jobId` and `wzFile` structured fields | PASS | `consumer.go:54` `l.WithFields(logrus.Fields{"jobId": ..., "wzFile": ...})` |
| All affected Go packages build and tests pass | PASS | see Build/Test section below |
| Manual smoke (replicas=1 / replicas=3 wall-clock) | OUT-OF-SCOPE | plan §"Acceptance criteria mapping" defers this to rollout (design §10) |

## Build / vet / test results

Run from `<worktree-root>`:

```
$ go build -C services/atlas-wz-extractor/atlas.com/wz-extractor ./...
(no output — success)

$ go vet -C services/atlas-wz-extractor/atlas.com/wz-extractor ./...
(no output — success)

$ go test -C services/atlas-wz-extractor/atlas.com/wz-extractor ./... -count=1
?       atlas-wz-extractor                          [no test files]
ok      atlas-wz-extractor/characterimage           0.011s
ok      atlas-wz-extractor/characterrender          0.021s
?       atlas-wz-extractor/cmd/map-render-spike     [no test files]
ok      atlas-wz-extractor/extraction               0.063s
ok      atlas-wz-extractor/extraction/job           0.041s
ok      atlas-wz-extractor/extraction/lock          0.026s
ok      atlas-wz-extractor/image                    0.005s
?       atlas-wz-extractor/kafka/consumer           [no test files]
ok      atlas-wz-extractor/kafka/consumer/extraction 0.034s
?       atlas-wz-extractor/kafka/message/extraction [no test files]
?       atlas-wz-extractor/kafka/producer           [no test files]
?       atlas-wz-extractor/logger                   [no test files]
ok      atlas-wz-extractor/mapimage                 0.004s
?       atlas-wz-extractor/rest                     [no test files]
ok      atlas-wz-extractor/wz                       0.005s
ok      atlas-wz-extractor/wz/canvas                0.004s
ok      atlas-wz-extractor/wz/crypto                0.004s
ok      atlas-wz-extractor/wz/property              0.004s
ok      atlas-wz-extractor/xml                      0.004s
```

```
$ grep -rn "extraction\.Acquire\|extraction\.TryAcquire\|extraction\.Release\|tenantMutexRegistry\|runExtraction" --include="*.go" services/atlas-wz-extractor/
services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go:183:    // These are the paths that would have been passed to runExtraction.
```

Single match is a **code comment in a test**, describing what the file paths represent in a now-renamed code path. No live references. Plan §25.4-25.5 expectation met.

## Action items / blockers

**Blockers for merge:** none.

**Recommended PR-body callouts (informational, not blocking):**

1. Document the upload-busy 409 transient regression with a follow-up task ID in the PR description so the reviewer is not surprised by the missing guard.
2. atlas-ui's `runWzExtraction` would benefit from capturing the new `jobId` to drive per-job progress polling against `GET /jobs/{jobId}`. File a Phase-2 UI task; not in this PR's scope.
3. The Kafka topic `command.wz.extraction` must be provisioned out-of-band with `partitions=16`, `replication=3`, `retention=24h`, per `docs/kafka.md`. The deploy manifest comment line correctly notes this; ensure ops is aware.

## Overall assessment

- **Plan adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending parallel `backend-guidelines-reviewer` audit and a brief PR-body note about the upload 409 transient regression)
