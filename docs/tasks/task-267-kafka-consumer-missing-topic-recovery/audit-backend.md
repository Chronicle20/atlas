# Backend Audit — task-267-kafka-consumer-missing-topic-recovery (changed-package review)

- **Service Path:** `libs/atlas-kafka/consumer` (bulk of the change) + `services/atlas-character-factory/atlas.com/character-factory/factory` (one log line)
- **Range:** `e880af7e4..daf46558c`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-26
- **Build:** PASS
- **Tests:** all passed (no explicit pass/fail counts emitted by `go test`; every package reported `ok` or `[no test files]`, none `FAIL`)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd libs/atlas-kafka && go build ./... && go test ./... -count=1
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumer        9.085s
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup   0.003s
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/handler         [no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/message         0.006s
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/producer        0.007s
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest [no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/retry           0.961s
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/topic           [no test files]

cd services/atlas-character-factory/atlas.com/character-factory && go build ./... && go test ./... -count=1
ok  	atlas-character-factory                0.021s
ok  	atlas-character-factory/configuration  0.105s
ok  	atlas-character-factory/configuration/projection 0.011s
ok  	atlas-character-factory/factory        0.029s
(all other subpackages: [no test files])
```

`tools/verify.sh` was not re-run per instructions (already green on this exact tree). `tools/goroutine-guard.sh` was run directly (see DOM-26 below) and exited 0.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE placement (FILE-01..06) | Fired | Every changed Go package (`libs/atlas-kafka/consumer`, `.../factory`) is in scope unconditionally. |
| DOM structure (DOM-01..05,11,16) | Fired (factory only) | `services/.../factory` has `model.go`-equivalent domain files (`rest.go`, `preset_rest.go`); `libs/atlas-kafka/consumer` has neither `model.go` nor `entity.go` nor `rest.go` nor `provider.go` — DOM structure N/A for that package. |
| SUB sub-domain (SUB-01..04) | N/A | Neither changed package has `resource.go` with no `model.go`; `consumer` has no `resource.go` at all, `factory` has `model.go`-class files. |
| REST (DOM-06..09,12..15,17..19,32) | Fired (factory only) | `factory/resource.go` changed and the package registers routes; `libs/atlas-kafka/consumer` has no `resource.go`/`rest.go`/`processor.go` and registers no HTTP routes — N/A there. |
| Constants reuse (DOM-21) | Fired | Diff declares `type emptyAssignmentClass`, a const block, `var ErrTopicNotFound`, `topicMetadataTimeout`, `topicMissingWarnInterval`. |
| Testing (DOM-10,20,24,33) | Fired | Diff touches many `_test.go` files. |
| Cache (DOM-29) | N/A | No `cache.go`, no processor/struct holding cached state added. |
| Messaging (DOM-30) | N/A | No `producer.go`; `grep -n "producer.ProviderImpl\|AndEmit\|message.Emit\|producer.Produce"` across all changed non-test files returns nothing. |
| Multi-tenancy (DOM-31) | Fired (factory only), narrow | `factory` package has `rest.go`; the one changed line in `resource.go` is a log statement and touches no tenant/trace state. |
| Migration hygiene (DOM-34,35) | N/A | No symbols moved between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added (existing `libs/atlas-kafka` module edited in place); no Kafka topic env var added or renamed. |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed in both packages. |
| Channel wire values (DOM-25) | N/A | Diff touches neither `services/atlas-channel` nor `libs/atlas-packet`; no client-interpreted byte is emitted. |
| Resilience (DOM-27,28) | N/A | `factory` package calls no `database.Connect` anywhere in the service (`grep -rn "database.Connect"` empty); no `model.Decorator` changed. |
| External clients (EXT-01..04) | N/A | `grep -n "requests\.\(Root\|Get\|Post\)"` across all changed non-test files returns nothing. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new service directory, no new channel Writer/Handler, `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | N/A | Neither package handles authentication, tokens, or secrets. |
| Foundational: patterns-provider.md | N/A | No provider defined or composed in the diff (`PartitionCountProducer` is a producer-shaped seam analogous to `GroupProducer`/`PartitionReaderProducer`, not a `database.Query`-style provider). |
| Foundational: patterns-functional.md | N/A | No curried constructor, decorator, or model combinator added. |

## Checklist Results

### `libs/atlas-kafka/consumer` (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/impl live in `processor.go` | N/A | `grep -n "type Processor interface\|func NewProcessor("` across all changed files in this package returns nothing — package has no Processor concept. |
| FILE-02 | RestModel/Transform/Extract/JSON:API methods live in `rest.go` | N/A | Same grep for `type RestModel\|GetName()\|GetID()\|SetID()` returns nothing. |
| FILE-03 | Cross-service request functions live in `requests.go` | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` call sites. |
| FILE-04 | Entity struct/Migration/TableName live in `entity.go` | N/A | No `type entity struct`, `func Migration(`, or `TableName()` in the diff. |
| FILE-05 | Builder/Model/administrator writes/provider readers/state.go enums placed per file table | N/A | No `type Builder`, `func NewBuilder(`, domain `Model`, or DB writes/readers added. |
| FILE-06 | No package-named catch-all file bundling ≥2 responsibilities | PASS | None of FILE-01..05's responsibilities are present anywhere in the changed files, so there is nothing to bundle; `manager.go`, `group.go`, `engine_group.go`, `debug.go` each carry a single, library-internal concern (consumer state/config, group wiring, generation loop, debug snapshot). |
| DOM-21 | No redeclaration of a shared `libs/atlas-constants` type/const | PASS | `type emptyAssignmentClass` (`libs/atlas-kafka/consumer/engine_group.go:132-142`), `var ErrTopicNotFound` (`libs/atlas-kafka/consumer/group.go:60`), `const topicMetadataTimeout` (`group.go:65`), `const topicMissingWarnInterval` (`manager.go` new const) are all engine-internal observability/control-flow concepts — none match `libs/atlas-constants`'s documented categories (item-id classification, inventory types, weapon types, world/channel/character/map id widths, job/skill/monster id types); `find libs/atlas-constants -name '*.go'` shows no topic/timeout/backoff equivalents. |
| DOM-26 | Every goroutine via `routine.Go`, bare `go` needs a marker | PASS | `tools/goroutine-guard.sh` exit 0 (ran from repo root, `goroutineguard: 91 module(s), 8 parallel`, no failures reported). Manual grep of changed non-test files for `^\s*go (func|[A-Za-z_])` in `engine_group.go`, `group.go`, `manager.go`, `debug.go` returns no hits. |
| DOM-20 | Tests are table-driven (`tests := []struct{...}` + `t.Run`) | WARN | `TestPartitionCountFromMetadata` (`libs/atlas-kafka/consumer/group_test.go`, new) is correctly table-driven. But the six new scenario tests in `engine_group_test.go` (`TestGateDoesNotJoinUntilTopicExists`, `TestGroupEngineRecoversWhenTopicAppears`, `TestGateExitsOnContextCancel`, `TestNilPartitionCountProducerSkipsGate`, `TestGateJoinsImmediatelyOnIndeterminateLookup`, and its sibling truncated from this diff) plus `TestRecordTopicMissingCountsAndStamps`/`TestSnapshotTopicMissingSupersededByAssignment` in `state_test.go` are each a single hardcoded scenario, not a `tests := []struct{...}` table. Each does test a genuinely distinct behavior (cannot be meaningfully tabulated without losing the goroutine/timing narrative), and this mirrors the pre-existing scenario-test style already in this file (e.g. `TestGroupCloseWaitsForPartitionGoroutine`, unchanged by this diff) — but per the Mindset rule that prevalence is not compliance, that precedent does not exempt these new tests from DOM-20's plain text. Non-blocking because the guideline's own prose calls this a "prefer," not an absolute, and no rule text carves out an exception for integration-style goroutine tests the way it does for `DOM-20`'s packet-fixture playbook. |
| DOM-24 | Emit-path tests stub the producer | N/A | `grep -n "AndEmit(\|message.Emit(\|producer.Produce("` across all changed test and non-test files in this package returns nothing — no emit path is reached, directly or transitively. |
| DOM-33 | Interface method changes update every mock | N/A | `Group`/`Generation` interfaces (`libs/atlas-kafka/consumer/group.go:14`, `:28`) are unchanged by this diff (only new declarations were added after them); `ManagerConfig` signature is unchanged. No `Processor`/`Provider`/`Administrator` interface exists in this package. |
| Deploy DOM-22 | New `libs/atlas-*` module wired into Dockerfile/go.work | N/A | Diff modifies the existing `libs/atlas-kafka` module in place; no new `libs/` directory added. |
| Deploy DOM-23 | Kafka topic env vars follow convention, live in base configmap, re-listed in both overlays | N/A | Diff adds no topic env var; `ErrTopicNotFound`/`topicMetadataTimeout` etc. are Go-internal, not env-var-backed topic names. |

### `services/atlas-character-factory/atlas.com/character-factory/factory` (domain package — has `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor`, never `logrus.StandardLogger()` | PASS | `resource.go:59` — `processor := NewProcessor(d.Logger())`, unchanged by this diff and still correct. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `resource.go` (full file read) contains no `os.Getenv` call. |
| DOM-13 | Handlers contain no cross-domain orchestration | PASS | `handleCreateFromPreset` (`resource.go:57-72`) only calls `processor.CreateFromPreset` and maps the error; no orchestration logic added or present. |
| DOM-14 | Handlers call processor methods, never provider functions directly | PASS | Same call site; no provider function called from `resource.go`. |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | No such calls anywhere in `resource.go`. |
| DOM-17 | Domain errors map to specific HTTP status | PASS | `categorizePresetError` (`resource.go:35-53`, unchanged by this diff) maps `ErrInvalidPresetId`→400, `ErrPresetNotFound`→404, `ErrNameDuplicate`→409, etc.; the new line at `resource.go:62` only adds logging before this existing mapping is invoked at line 63. |
| DOM-27 | 503 on transient DB error, not bare 500 | N/A | `grep -rn "database.Connect" services/atlas-character-factory/atlas.com/character-factory` returns nothing — this service is not DB-backed. |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | The one changed line (`resource.go:62`, a log statement) reads no tenant/trace field; `rest.go` and `preset_rest.go` (read for JSON:API-contract verification) carry no `tenantId`/trace field on any REST model. |
| FILE-01 | Processor lives in `processor.go` | PASS | `factory/processor.go` exists and is unchanged by this diff; `resource.go` (changed) contains no `ProcessorImpl` methods. |
| FILE-02 | RestModel/Transform/JSON:API methods live in `rest.go` | **FAIL (Important)** | `PresetCreateRestModel` and its `GetName()`/`GetID()`/`SetID()` methods live in `services/atlas-character-factory/atlas.com/character-factory/factory/preset_rest.go:1-12`, not in `factory/rest.go`. `rest.go` (`factory/rest.go:9-58`) already holds `RestModel` and `CreateCharacterResponse` correctly; `preset_rest.go` is a second, package-named-adjacent file carrying the same FILE-02 responsibility. This is a pre-existing violation (added in commit `0e3c15927`, predating this diff's range `e880af7e4..daf46558c`), surfaced here because `resource.go`'s changed handler (`handleCreateFromPreset`, `resource.go:57`) calls `PresetCreateRestModel`'s contract directly. Per the reviewer's Mindset rules, prevalence/pre-existence does not exempt a structural FILE violation from the changed package it sits in. |
| DOM-18/19 | RestModel implements JSON:API interface; request models are flat | PASS | `preset_rest.go:9-12` — `GetName()`/`GetID()`/`SetID()` present; struct (`preset_rest.go:3-8`) is flat, no nested `Data`/`Type`/`Attributes`. (Placement is still wrong per FILE-02 above — this is a different, orthogonal check.) |
| DOM-20 | New test is table-driven | WARN | `TestHandleCreateFromPreset_LogsErrorWithPresetMessage` (`factory/resource_test.go`, new) is a single hardcoded scenario, not a `tests := []struct{...}` table. Same non-blocking disposition as the kafka-consumer tests above — it tests one specific regression (the swallowed error), which does not naturally tabulate. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `resource.go` contains no `go` statement, bare or otherwise; `tools/goroutine-guard.sh` exit 0. |

## Not evaluable from the diff

- none

## Summary

### Blocking (must fix)
- FILE-02: `services/atlas-character-factory/atlas.com/character-factory/factory/preset_rest.go:1-12` — `PresetCreateRestModel` and its JSON:API methods (`GetName`/`GetID`/`SetID`) live outside `rest.go`, in violation of FILE-02's file-placement rule. Pre-existing (predates this diff), but the changed package (`factory/resource.go`) is in scope for FILE-* per the checklist's unconditional trigger, and the changed handler's correctness depends on this exact type.

### Non-Blocking (should fix)
- DOM-20: Six new scenario tests in `libs/atlas-kafka/consumer/engine_group_test.go` and two in `state_test.go` are not table-driven (`TestGateDoesNotJoinUntilTopicExists`, `TestGroupEngineRecoversWhenTopicAppears`, `TestGateExitsOnContextCancel`, `TestNilPartitionCountProducerSkipsGate`, `TestGateJoinsImmediatelyOnIndeterminateLookup`, `TestRecordTopicMissingCountsAndStamps`, `TestSnapshotTopicMissingSupersededByAssignment`).
- DOM-20: `TestHandleCreateFromPreset_LogsErrorWithPresetMessage` in `services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go` is a single-scenario test, not table-driven.
