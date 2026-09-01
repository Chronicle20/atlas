# Backend Audit — task-276-kafka-topic-manifest (delta: b5552f669~1..HEAD)

- **Service Path:** repo-wide delta (libs/atlas-kafka, services/atlas-channel, tools/)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-01
- **Range audited:** `b5552f669~1..HEAD` (HEAD = `b0a434473`)
- **Build:** PASS (module-local, scoped)
- **Tests:** all passed in touched modules
- **Overall:** PASS (with 1 non-blocking finding)

This is a delta audit layered on top of the existing full audit at
`docs/tasks/task-276-kafka-topic-manifest/backend-audit.md`, which covered the
branch up to just before `b5552f669` and raised 2 blocking findings — both
independently re-verified as resolved in the current tree (consumables
pickup now passes `mbmsg.EnvCommandTopic`; `libs/atlas-kafka/gen` is in
`go.work:9`) and NOT re-litigated here.

## Scope — exact file list for this range

```
git diff --name-only b5552f669~1..HEAD
```
```
.gitignore
deploy/k8s/overlays/main/kustomization.yaml          (image-tag bumps only, from origin/main merge)
deploy/seed/**/CATALOG_REVISION                      (data, non-Go)
docs/tasks/task-276-kafka-topic-manifest/*.md         (docs, non-Go)
go.work
go.work.sum
libs/atlas-kafka/consumer/engine.go
libs/atlas-kafka/consumer/engine_group.go
libs/atlas-kafka/consumer/engine_group_test.go
libs/atlas-kafka/consumer/engine_reader.go
libs/atlas-kafka/consumer/manager.go
libs/atlas-kafka/consumer/manager_test.go
libs/atlas-kafka/gen/manifest.go
libs/atlas-kafka/gen/topics.yaml
services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/kafka.go
tools/lib/analyzer-guard.sh
tools/lib/go-work.sh
tools/lint.sh
tools/verify.sh
tools/verify_test.sh
```

This is the complete set of files touched anywhere in the 8-commit range,
confirmed by `git diff --stat`/`--name-only` against the same range — there
is no service-level producer/consumer code beyond `playernpc/kafka.go`
touched by the merge (`598cc6c27`). The merge itself (`git show --stat
598cc6c27`) brought in only deploy manifests, `CATALOG_REVISION` bumps, and
the `libs/atlas-kafka/consumer` WaitGroup-ordering fix (issue #1586) from
`origin/main` — no new service-level Kafka producer/consumer code.

## Build & Test Results

Ran module-local build/test (not `tools/verify.sh`, per instructions):

```
cd libs/atlas-kafka/gen && go build ./... && go run . --check
  → OK: generated files are up to date (161 topics)

cd libs/atlas-kafka && go build ./... && go test ./... -count=1
  → ok  consumer            9.120s
  → ok  consumergroup       0.003s
  → (no test files) handler
  → ok  message             0.006s
  → ok  producer            0.007s
  → (no test files) producer/producertest
  → ok  retry               0.279s
  → ok  topic               0.003s

cd services/atlas-channel/atlas.com/channel && go build ./... && \
  go vet ./kafka/consumer/playernpc/... && \
  go test ./kafka/consumer/playernpc/... -count=1
  → ok  atlas-channel/kafka/consumer/playernpc   0.023s
```

All green. `tools/verify.sh`, `tools/lint.sh`, and the docker bake were
intentionally not run per the task's constraints (already green over HEAD per
`verify-final7.log`, cited by the requester).

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | No changed package has `model.go`, `entity.go`, `rest.go`, or `provider.go`. `kafka.go` in `playernpc` is not `rest.go`. |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs FILE-*, no exemption. See per-package table below. |
| SUB (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package has `resource.go`, `rest.go`, `processor.go`, or registers HTTP routes. |
| Constants reuse (DOM-21) | N/A | Diff retypes an existing const (`string`→`topic.Token`); declares no new type, const block, or numeric-literal classification. `services/atlas-channel/.../playernpc/kafka.go:21`. |
| Testing (DOM-10,20,24,33) | Fired | `engine_group_test.go` and `manager_test.go` changed; no changed package reaches a GORM DB directly or an emit path; one wholly-new test added. |
| Cache (DOM-29) | N/A | No changed package has `cache.go` or holds cached processor state. |
| Messaging (DOM-30) | N/A | Grepped `AndEmit\|message.Buffer\|producer.ProviderImpl\|topic.Provider` across every changed Go file in the range — zero matches. No changed package has `producer.go`. |
| Multi-tenancy (DOM-31) | N/A | No changed file is `rest.go`; grepped `tenant\.|MustFromContext|db.WithContext` across every changed non-test Go file — zero matches. |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and a `libs/atlas-*` module in this range. |
| Deploy & topics (DOM-22,23) | N/A (see below) | `libs/atlas-kafka/gen` was already added to `go.work`/Dockerfile per the prior audit (not re-litigated). No topic env var was added or renamed — `EnvEventTopicStatus`'s value (`EVENT_TOPIC_PLAYER_NPC_STATUS`) is unchanged; only its Go type changed `string`→`topic.Token`. |
| Runtime safety (DOM-26) | Fired, PASS | Every non-test Go file changed. `manager.go:206` uses `routine.Go(l, ctx, func(_ context.Context) { con.start(l, ctx, wg) })`; no other bare `go` statement in any changed non-test file. |
| Channel wire values (DOM-25) | Fired, N/A | Diff touches `services/atlas-channel` (`kafka.go`). The only change there is a const type retype + one import; no client-interpreted byte/opcode is touched. |
| Resilience (DOM-27,28) | N/A | No DB-backed handler or `model.Decorator`/enrichment path in the changed files. |
| External clients (EXT-01..04) | N/A | No changed package calls `requests.RootUrl`/`requests.*Request[T]`. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/`, channel `Writer`/`Handler`, or `routes.conf` change. |
| Security (SEC-01..04) | N/A | No token/auth/redirect/secret handling touched. |
| patterns-provider.md (foundational) | N/A | No provider defined or composed in the changed files. |
| patterns-functional.md (foundational) | N/A | No curried constructor/decorator/combinator added. |

## Checklist Results

### libs/atlas-kafka/consumer (support package: `manager.go`, `engine.go`, `engine_group.go`, `engine_reader.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement / no catch-all | PASS | Package predates this diff with the same focused-file shape (`manager.go` owns registration, `engine*.go` own the two dispatch engines); this diff only moves a `wg.Add(1)` call and adds doc comments — no file gains a new responsibility. `libs/atlas-kafka/consumer/manager.go:199-207`. |
| DOM-26 | goroutine spawned via `routine.Go` | PASS | `libs/atlas-kafka/consumer/manager.go:206` — `routine.Go(l, ctx, func(_ context.Context) { con.start(l, ctx, wg) })`; `wg.Add(1)` moved to line 205, strictly before the launch, closing the issue #1586 race (`wg.Wait()` in a canceller could previously observe a zero counter while `con.start`'s goroutine had not yet run its own `wg.Add(1)`). |
| DOM-33 | Mocks updated for interface changes | N/A | `Consumer.start`/`Manager.AddConsumer` signatures are unchanged; this is an internal call-ordering fix, not an interface re-sign. |
| DOM-20 | New/changed tests are table-driven | FAIL (non-blocking) | `libs/atlas-kafka/consumer/manager_test.go:1472-1522` — `TestAddConsumerRegistersWaitGroupBeforeLaunch` is a wholly new test function, not `tests := []struct{...}` + `t.Run`. No documented DOM-20 exception (only the packet-fixture playbook exception exists, and it does not apply here) covers a single-scenario happens-before/concurrency test. |
| DOM-20 | Existing tests changed to add `wg.Add(1)` | N/A (informational) | `libs/atlas-kafka/consumer/engine_group_test.go` (9 call sites, e.g. line 45) only inserts one line (`wg.Add(1)`) into pre-existing, already-non-tabular test functions; the diff does not restructure or newly author these tests, so DOM-20's "changes tests" trigger is not read as re-opening their pre-existing shape. |
| DOM-24 | producertest stub for emit-reaching tests | N/A | Neither `manager_test.go` nor `engine_group_test.go` reaches `AndEmit`/`message.Emit`/`producer.Produce`, directly or transitively — grepped, zero matches. |

### libs/atlas-kafka/gen (support/tool package: `manifest.go`, `topics.yaml` data file)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement / no catch-all | PASS | `manifest.go:48` — the only change is `yamlEntry{Token: e.Token, Cleanup: e.Cleanup, Packages: e.Packages}` → `yamlEntry(e)`, a struct-conversion lint fix; no new responsibility added to the file. |
| (manifest drift) | `topics.yaml` matches the workspace scan | PASS | `go run . --check` in `libs/atlas-kafka/gen` → `OK: generated files are up to date (161 topics)`. |
| DOM-23 | Topic env var naming/configmap/overlay hygiene | N/A | No topic env var added or renamed — `topics.yaml` only gained 3 package-reference lines (`atlas-messages`, `atlas-saga-orchestrator`, `atlas-channel`) under two already-existing tokens (`COMMAND_TOPIC_PLAYER_NPC`, `EVENT_TOPIC_PLAYER_NPC_STATUS`), `libs/atlas-kafka/gen/topics.yaml:332-337,879-884`. |

### services/atlas-channel/.../kafka/consumer/playernpc (support package: `kafka.go`, `consumer.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement / no catch-all | PASS | `kafka.go` carries only topic-const + wire-envelope-model declarations, unchanged shape; `consumer.go` (unmodified this range) carries the consumer wiring. No file crosses ≥2 of FILE-01..05's responsibilities. |
| — | Topic const retyped `string`→`topic.Token` | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/playernpc/kafka.go:21` — `const EnvEventTopicStatus topic.Token = "EVENT_TOPIC_PLAYER_NPC_STATUS"`. |
| — | Every call site of `EnvEventTopicStatus` in this package accepts a `topic.Token` | PASS | `consumer.go:53` — `consumer2.NewConfig(l)("player_npc_status_event")(EnvEventTopicStatus)(consumerGroupId)`, and `consumer2.NewConfig`'s signature is `func(name string) func(token topic.Token) func(groupId string) consumer.Config` (`services/atlas-channel/atlas.com/channel/kafka/consumer/consumer.go:12`) — token-typed, matches. `consumer.go:66` — `topic.EnvProvider(l)(EnvEventTopicStatus)()` — also token-typed. Neither call site recasts a resolved name back to a bare string or reads the env var directly. |
| — | Sibling declarations of the same token, repo-wide | PASS | `EVENT_TOPIC_PLAYER_NPC_STATUS` is independently declared as `topic.Token` in `services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go:20`, `services/atlas-messages/atlas.com/messages/kafka/message/playernpc/kafka.go:25`, and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/playernpc/kafka.go:24` — atlas-channel was the only remaining string-typed holdout this commit closes; no sibling call site was missed. |
| DOM-25 | No client-interpreted byte written as a Go literal | N/A | The only change in `kafka.go` is the const retype and one import; no wire-op-code/dispatcher-byte is touched. |

### tools/ shell scripts (`go-work.sh`, `verify.sh`, `lint.sh`, `analyzer-guard.sh`, `verify_test.sh`)

No DOM/FILE/SUB/EXT/SCAFFOLD/SEC rule applies — these are shell, not Go, and the checklist's rule families are Go-package-scoped. Recorded here for completeness only, not scored against a numbered rule:

- `tools/lib/go-work.sh` adds `check_workspace_drift()`, called from `verify.sh`, `lint.sh`, and `analyzer-guard.sh`, each treating a non-empty drift set as fatal except `verify.sh --facts`, which reports it via `WORKSPACE_DRIFT_FILE` instead of aborting (`tools/verify.sh:91-101,262-273`). This directly implements the DOM-22 intent (a `libs/*` module unlisted in `go.work` must not go silently unbuilt/unvetted/unlinted) as tooling, not a Go rule — no numbered rule to score it against, but it is consistent with the documented DOM-22 concern in `file-responsibilities.md`.
- `tools/verify_test.sh` gained matching coverage: `assert_eq "--facts exits 0 with a go.work drift present"` and `assert_true "no probe, no drift fact"` (`tools/verify_test.sh:388-407`).

## Security Review

Not applicable — SEC-* trigger did not fire (no token/auth/redirect/secret handling in this range).

## Not evaluable from the diff

- None. Every file in the range's `git diff --name-only` list was read in full or grepped directly; every rule family's trigger was resolved from the diff itself plus the two targeted symbol lookups (`consumer2.NewConfig`'s signature, and the repo-wide `EVENT_TOPIC_PLAYER_NPC_STATUS` declaration grep) permitted for a scoped review.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- DOM-20: `libs/atlas-kafka/consumer/manager_test.go:1472` — `TestAddConsumerRegistersWaitGroupBeforeLaunch` is a new, non-table-driven test function with no applicable DOM-20 exception on record.
