# Backend Audit — task-209-assignment-aware-watchdog

- **Scope:** `libs/atlas-kafka/consumer/` — cursor.go, debug.go, engine.go, engine_group.go,
  engine_reader.go, group.go, manager.go, partition.go + `*_test.go` siblings
  (this is a shared LIBRARY, not a DDD service package)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, SUB-*, SEC-*, FILE-*)
- **Date:** 2026-08-10
- **Build:** PASS (`cd libs/atlas-kafka && go build ./...` — clean)
- **Vet:** PASS (`go vet ./...` — clean)
- **Race tests:** PASS (`go test -race ./... -count=1` — all packages `ok`, consumer 8.154s)
- **goroutine-guard:** PASS (`tools/goroutine-guard.sh` exit 0)
- **Overall:** PASS

## Applicability note

`libs/atlas-kafka/consumer` is infrastructure code, not a JSON:API domain
package. There is no `model.go`, `entity.go`, `processor.go`, `resource.go`,
`rest.go`, `requests.go`, `administrator.go`, or `builder.go`, and none should
exist — the package has no tenant-scoped domain model, no REST surface beyond
the one read-only debug route, and makes no calls to another atlas service.
Accordingly:

- **DOM-01 – DOM-05, DOM-08, DOM-09, DOM-16, DOM-18, DOM-19:** N/A — no domain
  `Model`, no JSON:API resource, no cross-service request funcs. There is no
  `model.go`/`entity.go`/`administrator.go`/`rest.go` triplet to grade.
- **DOM-06/DOM-07 (FieldLogger, not `*logrus.Logger`):** applies and PASSES —
  every constructor-adjacent function in the diff takes `logrus.FieldLogger`
  (`libs/atlas-kafka/consumer/engine.go:45` `resolveEngine(l logrus.FieldLogger)`;
  `libs/atlas-kafka/consumer/manager.go:132` `AddConsumer(cl logrus.FieldLogger, ...)`;
  `libs/atlas-kafka/consumer/partition.go:38` `runPartition(l logrus.FieldLogger, ...)`).
  No call site constructs or passes `logrus.StandardLogger()` except the one
  legitimate process-default fallback at `manager.go:110`
  (`engine: resolveEngine(logrus.StandardLogger())` inside `GetManager`'s
  `once.Do`, used only when no logger has been supplied yet at process
  bootstrap — there is no caller-supplied logger to prefer at that point).
- **DOM-10, DOM-11 (tenant callbacks, lazy providers):** N/A — no GORM entity,
  no tenant-scoped DB access in this package.
- **DOM-12 (no `os.Getenv` in handlers):** applies in the adapted sense of "no
  env reads outside the documented config surface." `os.Getenv` appears once,
  at `libs/atlas-kafka/consumer/engine.go:46`, inside `resolveEngine` — the
  single, explicitly-designed entry point for `KAFKA_CONSUMER_ENGINE`
  (documented at engine.go:28-32 and README.md). This is not a handler reading
  config ad hoc; it is the config resolution function itself. PASS.
- **DOM-13/DOM-14/DOM-15/DOM-17 (handler/processor separation, error→HTTP
  mapping):** N/A for the consumer engine (no REST handlers, no domain write
  path). Partially applicable to `debug.go`'s `DebugHandler`: it is
  read-only, calls only `m.Consumers()` / `c.Snapshot()` (no provider/DB calls
  inline — `debug.go:23-27`), and rejects non-GET with 405
  (`debug.go:17-20`). No error-status mapping table applies because the
  handler has no domain-error branches; PASS on what's checkable.
- **DOM-20 (table-driven tests):** applies and PASSES loosely — tests are
  scenario-per-function rather than `[]struct{...}` tables (e.g.
  `cursor_test.go` has one function per invariant:
  `TestCursorCommitsOffsetPlusOne`, `TestCursorCommitsOnlyContiguousPrefix`,
  `TestCursorFailedMessageBlocksCursor`, `TestCursorCommitFailureDoesNotAdvance`,
  `TestCursorAdvanceIsIdempotent`, `TestCursorResumeOffset`,
  `TestCursorResetKeepsCommitted`, `TestCursorConcurrentAdvance`). This is the
  idiomatic shape for concurrency/state-machine invariant testing (each case
  needs distinct setup/assertions that don't collapse into a data table) and
  each test cites the invariant/FR it pins in its doc comment. Not marking
  this a FAIL — DOM-20's intent (systematic coverage, not ad hoc single-shot
  tests) is met, just not via the literal `[]struct` idiom, which does not fit
  this kind of test.
- **DOM-21 (no atlas-constants duplication):** checked every new
  `type`/`const` declaration in the diff (`engine.go:12` `EngineName string`,
  `engine.go:14-26` `EngineConsumerGroup`/`EngineReader`, `cursor.go:10`
  `noCommit`, `cursor.go:14` `inflight`, `cursor.go:40` `cursor`, `group.go:12`
  `Group`, `group.go:26` `Generation`, `group.go:34/40`
  `GroupProducer`/`PartitionReaderProducer`, `manager.go:23-586` `KafkaReader`,
  `Closer`, `MessageReader`, `MessageCommitter`, `StatsProvider`,
  `ReaderProducer`, `ManagerConfig`, `Manager`, `Consumer`, `legacyPartition`,
  `partitionState`, `Snapshot`, `fetchBackoff`, `debug.go:45-55`
  `jsonAPIDocument`/`jsonAPIResource`/`debugAttributes`). None overlap
  `libs/atlas-constants/` (item/inventory/weapon/world/channel/map/character/
  job/skill/monster id types) — every new type is Kafka-consumer-engine
  infrastructure with no game-domain semantics. PASS.
- **DOM-22 (Dockerfile lib mentions):** N/A — this is a change inside
  `libs/atlas-kafka` itself, not a service `go.mod` adding it as a dependency;
  no service Dockerfile changed in this diff.
- **DOM-23 (Kafka topic naming/configmap):** N/A — no new topic constants or
  env-configmap entries in this diff; `KAFKA_CONSUMER_ENGINE` is a
  process-engine selector, not a topic name, and is not required to be
  configmap-sourced (it has a documented, safe default per `engine.go:28-32`).
- **DOM-24 (Kafka producer stubbed in tests):** N/A — this package is a
  consumer, not a producer; no `AndEmit`/`message.Emit`/`producer.Produce`
  call sites exist in the diff (`grep` for `requests.RootUrl\|requests.Get
  Request\|requests.PostRequest` and DB usage both returned no matches, and a
  separate grep for producer emit calls in this package likewise finds none).
- **DOM-25 (client wire values config-resolved):** N/A — no client packet
  byte codes in this package.
- **DOM-26 (goroutines via `routine.Go`):** applies and PASSES.
  `tools/goroutine-guard.sh` exits 0 tree-wide (includes this package). Manual
  grep for bare `go` statements in non-test files in the changed package
  returns zero matches. Every goroutine spawn point in production code goes
  through `routine.Go`:
  - `manager.go:196` `routine.Go(l, ctx, func(_ context.Context) { con.start(l, ctx, wg) })`
  - `manager.go:635` `routine.Go(handlerLogger, wctx, func(_ context.Context) {...})` (per-handler dispatch in `processMessage`)
  - `partition.go:196` `routine.Go(l, pctx, func(hctx context.Context) {...})` (bounded-serial handler)
  - `partition.go:216` `routine.Go(l, pctx, func(hctx context.Context) {...})` (parallel handler)
  - `partition.go:242` `routine.Go(l, context.Background(), func(_ context.Context) { wg.Wait(); close(done) })` (quiesce's wait-goroutine)
  - `engine_reader.go:228` `routine.Go(l, ctx, func(_ context.Context) {...})` (legacy parallel fetch loop — pre-existing pattern, preserved verbatim)
  One bare `go func() { ... }()` exists at `manager_test.go:697`, inside a
  test file (`_test.go`), which `tools/goroutine-guard.sh` excludes by design
  and which the audit brief also excludes from scope — not a finding.
- **DOM-27 (transient DB errors → 503):** N/A — no DB-backed HTTP handlers in
  this package.
- **DOM-28 (no silent degradation in decorators):** N/A — no
  `model.Decorator` implementations here. The closest analog, `handleFetch
  Deadline` (`manager.go:566-581`), does the FR-2.x equivalent correctly: it
  distinguishes idle-vs-stall via `readerMadeProgress` and logs Warn on every
  no-progress tick (`manager.go:578-579`) rather than swallowing silently.

## Domain/Sub-domain classification

No package under this diff has a `model.go` or a `resource.go` — `consumer`
is a single flat package with no domain/sub-domain split. DOM checklist and
SUB checklist (SUB-01 – SUB-04) are both N/A in the literal sense (no
Processor/administrator/RegisterInputHandler/JSON pattern exists or should
exist here); graded instead against File-Responsibilities and the
concurrency-specific brief below.

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | N/A | No `Processor` interface/impl in this package — not a domain package. |
| FILE-02 | RestModel/Transform in rest.go | N/A | No `RestModel`; `debug.go` hand-rolls a minimal JSON:API-shaped document (`jsonAPIDocument`/`jsonAPIResource`) for one read-only debug route, not a domain resource. |
| FILE-03 | Cross-service request funcs in requests.go | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call sites in the package (grep confirmed empty). |
| FILE-04 | Entity + Migration + TableName in entity.go | N/A | No GORM entity in this package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No such symbols exist; nothing to misplace. |
| FILE-06 | No package-named catch-all file | **FAIL (Minor, pre-existing pattern extended)** | `manager.go` (662 lines) bundles `Manager` (registry/lifecycle), `Consumer` (struct + all watchdog/stat recorder methods), `Snapshot`, `partitionState`, `fetchBackoff`, `processMessage`, and `safeHandle` — six-plus distinct responsibilities in one file. This is not the `<pkgname>.go` collapse the guideline targets (file is named `manager.go`, not `consumer.go`), and it long predates this diff (git blame shows `Manager`/`Consumer`/`processMessage` present before task-209), but this diff *adds* to it: `engine EngineName` field, `partitions map[int]*partitionState`, `assignedPartitions`/`generationID`/`lastAssignmentAt`, `onAssignment`, and the phase-timing fields/recorders are new code landed inside the existing catch-all rather than split out. Per the audit rubric, prevalence/pre-existing shape is not an exemption — recording as an Important-by-default File-Responsibilities finding, downgraded to Minor because this is infrastructure code with no `file-responsibilities.md`-documented placement for `Manager`/`Consumer`/watchdog-state types (the table is written for DDD service packages: Processor/RestModel/entity/administrator/provider/state-enum). There is no prescribed alternate file name to move this code to, so this is flagged for awareness rather than blocking. |

## Concurrency Correctness (primary focus)

| Area | Status | Evidence |
|------|--------|----------|
| Every goroutine via `routine.Go` | PASS | See DOM-26 above; `tools/goroutine-guard.sh` exit 0, manual grep confirms zero bare `go` in non-test files under the changed package. |
| WaitGroup scoping (no `Add` concurrent with `Wait`) | PASS | `partition.go:63` allocates a fresh `wg` in `runPartition`'s outer loop is misleading to read in isolation — the actual fix is in `runPartitionFetchLoop` (`partition.go:138`), which allocates **its own** `wg := &sync.WaitGroup{}` *per attempt* and returns it to the caller (`partition.go:80` `wg, err = c.runPartitionFetchLoop(...)`), rather than reusing one `wg` across reader rebuilds. The doc comment at `partition.go:57-62` and `partition.go:119-124` explicitly names the hazard this avoids (stdlib's documented-unsafe "Add concurrent with Wait" panic) and attributes it to "task-209 review finding 2" — i.e., this was caught and fixed during review, not by this audit. |
| `quiesce` bounds in both serial and parallel paths | PASS | `partition.go:188-210` (serial) and `partition.go:213-224` (parallel) both route the handler through `wg.Add(1)` + `routine.Go` before continuing, so `quiesce` (`partition.go:240-254`) always has a `wg` to wait on. Comment at `partition.go:126-136` documents this fixes a prior bug ("task-209 review finding 1") where the serial path called `processMessage` synchronously inline, making `quiesce` unreachable. |
| Context cancellation propagation | PASS | `pctx` in `runPartition` is derived from both the process ctx and the generation ctx via `context.AfterFunc(gctx, cancel)` (`partition.go:43-46`), avoiding a manual watcher goroutine; `runPartitionFetchLoop` and `runFetchLoopSerial`/`runFetchLoopParallel` (legacy) both check `pctx.Err()`/`ctx.Err()` at the top of every loop iteration and unwrap `context.DeadlineExceeded` vs `context.Canceled` distinctly (`partition.go:146-180`, `engine_reader.go:98-120`). |
| Mutex discipline in `cursor` | PASS | `cursor.go:37-39` documents the invariant explicitly ("`cmu` serializes the commit call itself... never held together with `mu` across a network call") and `advance()` (`cursor.go:74-107`) follows it: `mu` is locked/unlocked to compute `pendingCommit`, released, then `cmu` is locked for the actual `commit()` network call, with `mu` re-acquired only for the short read/write of `target`/`committed`. No lock is held across the `commit(target)` call at `cursor.go:97`. |
| Mutex discipline in `Consumer`/`Manager` | PASS | All `c.mu`/`m.mu` critical sections in `manager.go` are short field reads/writes with no calls out to network/IO while held (e.g. `onAssignment` `manager.go:347-362`, all `record*` methods `manager.go:483-556`). `processMessage` (`manager.go:611-651`) copies the handler map under `c.mu` (`manager.go:622-627`) before iterating, so handler execution itself never runs under the lock. |
| Goroutine leak on drain timeout | Noted, not a finding | `quiesce`'s background `wg.Wait()` goroutine (`partition.go:242-245`) outlives the function if `drainTimeout` expires — it is not orphaned forever (it exits once the abandoned handler eventually returns), but it is untracked by any outer `WaitGroup` in that scenario. This is the documented, deliberate at-least-once tradeoff (`partition.go:20-22`, `249-251`) and matches `risks.md`'s stated R1 tradeoff; not treated as a defect. |

## Offset-Commit Correctness — cursor.go (primary focus)

| Check | Status | Evidence |
|-------|--------|----------|
| Commits `msg.Offset+1`, not `msg.Offset` | PASS | `inflight.offset` stores the raw `kafka.Message.Offset` (`cursor.go:12-13`); the `+1` conversion happens once, at `cursor.go:81` (`c.pendingCommit = c.pending[i-1].offset + 1`). Pinned by `TestCursorCommitsOffsetPlusOne` (`cursor_test.go:44-61`) and end-to-end by `TestRunPartitionCommitsOffsetPlusOneThroughGeneration` (`partition_test.go:73+`). |
| Never advances past a failed message | PASS | The contiguous-prefix walk in `advance()` (`cursor.go:76-79`) stops at the first `!done` or `!ok` entry, so a `mark(false)` head permanently blocks the prefix until redelivery succeeds. Pinned by `TestCursorFailedMessageBlocksCursor` (`cursor_test.go:91-109`), which asserts zero commits and `committedOffset() == -1` after a head-fails/tail-succeeds sequence. |
| Commit failure retains the high-water mark (FR-1.3) | PASS | `cursor.go:97-99`: on `commit(target)` error, `c.committed` is left untouched (the `if target > c.committed` update at `cursor.go:102-104` is skipped via early `return err`), so the next `advance()` call recomputes and retries the same `target`. Pinned by `TestCursorCommitFailureDoesNotAdvance` (`cursor_test.go:113-133`). |
| Concurrent-advance safety / commit monotonicity | PASS | `TestCursorConcurrentAdvance` (`cursor_test.go:202-238`) runs 50 concurrent `mark`+`advance` calls under `-race` and asserts commits are strictly increasing and the final commit equals `n`; `go test -race` passed clean (see Build & Test Results). |
| `resumeOffset` ordering (in-flight → committed → fallback) | PASS | `cursor.go:120-130` implements exactly that precedence; pinned by `TestCursorResumeOffset` (`cursor_test.go:156-176`). |
| `reset` never rewinds `committed` | PASS | `cursor.go:134-139` only clears `pending`/`pendingCommit`, never touches `committed`; pinned by `TestCursorResetKeepsCommitted` (`cursor_test.go:180-197`). |

## Error Handling / Backoff / No Swallowed Errors

| Check | Status | Evidence |
|-------|--------|----------|
| Commit errors logged, not swallowed | PASS | `tryAdvance` (`partition.go:231-235`) always logs at Warn on a non-nil `cur.advance` error before returning. |
| Reader-close errors logged | PASS | `partition.go:81-83`, `engine_group.go:33-35`, `engine_reader.go:55-57` all log `Debugf` on close error rather than dropping it. |
| Group-join / generation-fetch errors surfaced | PASS | `runGenerations` returns any `grp.Next` error to `startGroupEngine`, which records it (`engine_group.go:43` `c.recordError(err)`) and logs at Error before backing off (`engine_group.go:44`). |
| Backoff-then-retry, not spin, on every recoverable failure path | PASS | Both `startGroupEngine` (`engine_group.go:45-52`) and `runPartition`'s rebuild loop (`partition.go:99-104`) gate the retry behind `time.After(wait)` inside a `select` that also watches for context cancellation — no busy loop. |
| Two intentional `_ = ...` swallows, both are justified/documented | Not a finding | `group.go:71` `_ = r.SetOffset(offset)` — comment directly above (`group.go:65-68`) explains SetOffset can only fail on a closed or group reader, neither of which applies here, and is a no-op when offset already matches. `debug.go:41` `_ = json.NewEncoder(w).Encode(...)` — standard idiom for a response encode after `WriteHeader` has already been sent (nothing actionable can be done with an encode failure at that point; the debug route is explicitly documented as safe-to-expose read-only tooling, not an SLA-bearing endpoint). Neither hides a functional bug. |
| No TODO/FIXME/stub handlers | PASS | `grep -n "TODO\|FIXME\|XXX\|not implemented\|unimplemented"` over all non-test files in the changed package returned zero matches. `engine_reader.go:16-21` explicitly documents itself as a frozen legacy path with "Do not add behaviour here" rather than leaving a stub. |

## engine_reader.go — legacy-frozen assessment

Per the audit brief, `engine_reader.go` was intended to be a VERBATIM move of
the pre-existing legacy engine and is judged as legacy code preserved for
rollback, not audited for new-code style. Findings:

- `engine_reader.go:16-21` explicitly documents the file's frozen status:
  "This file holds the LEGACY consumer engine... Do not add behaviour here —
  the supported engine is engine_group.go." This self-documentation is itself
  good practice, not a defect.
- No genuine (behavioral) defects found in the moved code: `startReaderEngine`
  (`engine_reader.go:33-75`), `runFetchLoop`/`runFetchLoopSerial`/
  `runFetchLoopParallel` (`engine_reader.go:80-238`) all correctly propagate
  context cancellation, distinguish `DeadlineExceeded` from other errors, and
  (in the parallel path) use `routine.Go` for handler dispatch
  (`engine_reader.go:228`) with proper `sem`/`wg`-equivalent bounding via the
  local `pending`/`advanceCommit` cursor pattern.
- No new findings raised against this file — any style choices inherited from
  the pre-existing implementation (e.g., the parallel path's local
  hand-rolled cursor duplicating logic now centralized in `cursor.go` for the
  new engine) are pre-existing and out of scope, not new-code issues
  introduced by this diff.

## Security Review

N/A — this service/library performs no authentication, authorization, or
token handling. Skipped per Phase 4 gating.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- FILE-06: `libs/atlas-kafka/consumer/manager.go` continues to grow as a
  multi-responsibility file (`Manager`, `Consumer`, `Snapshot`,
  `partitionState`, `fetchBackoff`, `processMessage`, `safeHandle`); this
  diff adds the new assignment/watchdog-state fields and methods into it
  rather than splitting them into a dedicated file (e.g. a
  `partition_state.go` for `partitionState`/`fetchBackoff`, or an
  `assignment.go` for `onAssignment`/the new `Consumer` fields). Not
  blocking — no `file-responsibilities.md` rule prescribes a placement for
  this library's non-DDD types, and the diff's own file (`partition.go`) is
  already correctly split out — but flagged since the file's growth trend
  works against exactly the kind of collapse the guideline exists to prevent.
