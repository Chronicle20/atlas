# Task 26b review — release the in-flight craft guard on the saga's terminal event

Commit reviewed: `974cf0257` (12 files, all under `services/atlas-maker/`).
Brief: `.superpowers/sdd/plan/task-26b-brief.md`. Report: `.superpowers/sdd/plan/task-26b-report.md`.

## Scope confirmed

The diff matches the brief's file list exactly: `craft/inflight.go` (+`inflight_test.go`),
`craft/processor.go`, `craft/eligibility.go`, `craft/emitter.go`, the new
`kafka/message/saga`, `kafka/consumer/consumer.go`, `kafka/consumer/saga/{consumer,consumer_test}.go`,
`main.go`, `README.md`, `docs/kafka.md`. No file outside `services/atlas-maker/` touched. No
scope mismatch.

## Findings

### 1. BLOCKING — `Track` is called after `Emit` succeeds, even though the transaction id is
available before `Emit` is ever called; the implementer's stated reason for not tracking earlier
is factually wrong, and the chosen ordering reopens the exact class of race the brief asked to be
ruled out.

- `craft/processor.go:104-111` (`Create`): `TryAcquire` → `p.create(...)` (which builds the saga
  and calls `p.em.Emit(s)`) → **only on success** → `craftGuard.Track(t.Id(), characterId, txId)`.
- `craft/processor.go:446-456` (`emit`): `s := b.Build()` (line 447) assigns `s.TransactionId`,
  and **only after that** does `p.em.Emit(s)` run (line 448) — the actual Kafka produce.
- `libs/atlas-saga/builder.go:18-23` (`NewBuilder`): `transactionId: uuid.New()` is assigned when
  the builder is constructed, long before `Build()` or `Emit()`. The transaction id exists from
  the very start of every `create*` path (`createOrUpgrade`, `crystal`, `disassemble`), not "after
  `p.create()` builds and emits the saga" as the report states (task-26b-report.md:13-15).
- `libs/atlas-kafka/producer/producer.go`'s `Produce`/`tryMessage` (the seam
  `producer.ProviderImpl` resolves to) is a synchronous `w.WriteMessages(ctx, m)` — it blocks
  until the broker acknowledges the write. So by the time `Emit` returns and `Create` reaches
  `craftGuard.Track`, the COMMAND_TOPIC_SAGA message is already durably visible to
  atlas-saga-orchestrator's consumer group, in a **different process**, before this process's own
  goroutine has recorded the transaction-id mapping locally.
- Consequence exactly matches the brief's worry: if the orchestrator (and, on the way back, this
  service's own EVENT_TOPIC_SAGA_STATUS consumer) processes the terminal event before the
  originating goroutine executes the next line (`Track`), `ReleaseByTransactionId` finds nothing
  in `byTransaction` (`craft/inflight.go:100-110`) and is a no-op; `Track` then runs afterward and
  installs a mapping nothing will ever release again — the guard leaks for the life of the pod,
  reproducing the bug this task exists to close, on a narrower but non-zero window.
- The safe, no-additional-cost fix was available and not taken: `s.TransactionId` is known at
  `emit()`'s line 447, strictly before the produce at line 448. Calling `craftGuard.Track` there
  (passing tenant/character through to `emit`, or tracking at each `emit(...)` call site before
  the call) removes the window by construction instead of relying on cross-process Kafka
  round-trip latency to outrun a single in-process statement.
- Assessed exploitability: in a real deployment the window requires two full cross-process
  round-trips (produce → orchestrator consumer poll+process → produce back → this service's own
  consumer poll+process) to complete before the *next Go statement* in the same goroutine that
  just returned from a blocking, acknowledged `WriteMessages` call. That is not realistically
  reachable in production, but it is not ruled out by construction either — it is a timing
  assumption about a distributed system, not a guarantee, and the correct guarantee was one line
  away. No test in this commit exercises or pins this ordering (`TestGuardReleasesByTransactionId`
  and the consumer tests all seed `Track` before invoking the release path — see
  `craft/inflight_test.go:96-108`, `kafka/consumer/saga/consumer_test.go:51-58` — none of them
  construct the "release arrives before Track" ordering the brief asked to be interrogated).
- Per the brief's own framing ("if it is not [safe], this is BLOCKING"), and because a strictly
  safer construction was available at no cost and skipped based on an incorrect premise, this is
  a blocking finding, not a note.

### 2. PASS — both terminal types release, tenant-scoped.

- `kafka/consumer/saga/consumer.go:52-59` (`handleStatusEventCompleted`) and `:65-75`
  (`handleStatusEventFailed`) both call `craft.ReleaseInFlightByTransaction(t.Id(),
  e.TransactionId)`, gated on the correct `Type` constant each.
- `TestCompletedEventReleasesGuard` and `TestFailedEventReleasesGuard`
  (`kafka/consumer/saga/consumer_test.go:50-90`) genuinely exercise both, each seeding
  `AcquireForTest`+`TrackForTest` then asserting `AcquireForTest` succeeds again post-handler —
  a real assertion that would fail without the release, not a tautology.
- `TestNonTerminalEventDoesNotRelease` (`consumer_test.go:93-109`) confirms an unmatched `Type`
  leaves the guard held.

### 3. PASS — tenant scoping reaches the handler and is used in the key.

- `kafka/consumer/saga/consumer.go:26-30` registers `consumer.TenantHeaderParser` alongside
  `SpanHeaderParser`/`EnvHeaderParser`; both handlers call `tenant.MustFromContext(ctx)` before
  releasing (`consumer.go:57`, `:71`).
- `TestReleaseIsTenantScoped` (`consumer_test.go:112-131`) seeds tenant B's entry, delivers a
  terminal event for tenant A carrying the **same** transaction id, and asserts tenant B's entry
  is still held afterward — a real cross-tenant isolation assertion, not just a smoke test.
- `craft/inflight.go`'s `transactionKey{tenantId, transactionId}` composite key (`:18-22`) is what
  makes this possible; `ReleaseByTransactionId` (`:100-110`) looks up by the full composite key.

### 4. PASS — synchronous-rejection path unchanged and safe against double-release/cross-character
release.

- `craft/processor.go:104-108`: on a `p.create` error, `craftGuard.Release(t.Id(), characterId)`
  is still called directly, character-keyed, exactly as before this commit — Task 23's path is
  untouched in shape.
- `Release` (`craft/inflight.go:79-92`) deletes from `inflight` and sweeps `byTransaction` for any
  mapping pointing at the deleted key; both `delete()` calls on already-absent keys are Go no-ops,
  not panics — releasing twice, or releasing a key that was never tracked, cannot panic or affect
  another character's entry (the key is `(tenantId, characterId)`, never anything narrower).
  `TestGuardTrackAndReleaseConcurrencySafe` (`inflight_test.go:66-84`) exercises concurrent
  `Track`/`Release` under `-race` without incident (confirmed independently — see below).

### 5. PASS — first-consumer wiring matches the map-actions precedent.

- `main.go` diff: `consumerGroupId = consumergroup.Resolve("Maker Service")`,
  `consumer.GetManager().AddConsumer(...)`, `saga.InitConsumers(l)(cmf)(consumerGroupId)`,
  `saga.InitHandlers(l)(consumer.GetManager().RegisterHandler)` with a `Fatal` on error, and
  `AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler()))`
  — all present and shaped identically to
  `services/atlas-map-actions/atlas.com/map-actions/main.go:23,62-69,82`.
- Deploy reachability independently verified (not merely trusted from the report):
  `deploy/k8s/base/env-configmap.yaml:180` defines `EVENT_TOPIC_SAGA_STATUS`;
  `deploy/k8s/base/atlas-maker.yaml:24-26` has `envFrom: configMapRef: name: atlas-env`. No deploy
  change was needed and none was made.

### 6. PASS — no duplicate topic constant left behind.

- `grep -rn "EnvCommandTopic"` across the module returns exactly two hits: the definition in
  `kafka/message/saga/kafka.go:13` and the one call site in `craft/emitter.go:31`
  (`msgsaga.EnvCommandTopic`). The old local `const EnvCommandTopic = "COMMAND_TOPIC_SAGA"` in
  `emitter.go` is gone.

### 7. PASS (independently reproduced) — concurrency.

Ran, from `services/atlas-maker/atlas.com/maker`:

```
go test ./craft/... ./kafka/... -race -count=1
```

```
ok  	atlas-maker/craft	1.141s
?   	atlas-maker/kafka/consumer	[no test files]
ok  	atlas-maker/kafka/consumer/saga	1.044s
?   	atlas-maker/kafka/message/saga	[no test files]
```

Also independently ran `go build ./...` (clean), `go vet ./...` (clean), `gofmt -l .` (no output),
and `go test ./... -count=1` from the same module root — all packages pass, matching the report.

### 8. PASS (independently verified) — `go mod tidy` failure is genuinely pre-existing.

Checked out the parent commit (`52c89dc5d`) into the working tree and ran `go mod tidy` there:

```
go: downloading github.com/Chronicle20/atlas/libs/atlas-env v0.0.0
...
reading github.com/Chronicle20/atlas/libs/atlas-env/go.mod at revision libs/atlas-env/v0.0.0:
unknown revision libs/atlas-env/v0.0.0
```

Same failure, same missing `replace`, present before this commit. Restored the worktree to
`974cf0257` afterward (`git checkout 974cf0257 -- services/atlas-maker`); `git status --short`
confirms a clean tree post-restore. Not introduced by this commit.

## Not evaluable

- Whether the theoretical race in Finding 1 can be triggered under the project's actual Kafka
  broker/consumer configuration (fetch/poll intervals, whether atlas-saga-orchestrator's own
  processing could ever complete inside the single-digit-microsecond window between a
  `WriteMessages` return and the next Go statement) was reasoned about from source and library
  semantics only; no integration/load test against a live broker was run as part of this review
  to attempt to reproduce it. This does not change the blocking disposition — the correct fix
  removes the question entirely rather than relying on an empirical answer.

## Recommendation

Move the `Track` call to before the produce: track `s.TransactionId` against
`(tenantId, characterId)` at `emit()`'s `craft/processor.go:447` (right after `b.Build()`, before
`p.em.Emit(s)`), rather than after `p.create()` returns in `Create`. This requires threading
`tenantId`/`characterId` into `emit()` (or its three call sites) but removes the race by
construction instead of by distributed-system timing, which is exactly what Task 26b exists to
guarantee.
