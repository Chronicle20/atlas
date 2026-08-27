# Backend Audit — atlas-kafka-precreate (task-265-compacted-topic-segment-rolling)

- **Service Path:** services/atlas-kafka-precreate
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-26
- **Build:** PASS
- **Tests:** 105 passed, 0 failed
- **Overall:** PASS

## Scope

Merge-base `32d55cb21` .. HEAD `9671bcecf`, restricted to
`services/atlas-kafka-precreate/`. Changed Go files:

- `services/atlas-kafka-precreate/internal/topics/topics.go`
- `services/atlas-kafka-precreate/internal/topics/topics_test.go`
- `services/atlas-kafka-precreate/main.go`

Non-Go changes (`README.md`, task docs) were read for context but carry no
rule of their own beyond confirming no topic env var was added or renamed
(DOM-23 trigger check).

`atlas-kafka-precreate` is a short-lived bootstrap Job: no REST surface, no
GORM entities, no domain models, no processors, no Kafka consumers/producers.
It issues Kafka admin RPCs (`CreateTopics`, `IncrementalAlterConfigs`,
`Metadata`, `ListOffsets`) via `kafkaops.AdminClient` and exits.

## Build & Test Results

```
$ cd services/atlas-kafka-precreate && go build ./...
(clean, exit 0)
$ cd services/atlas-kafka-precreate && go test ./... -count=1
Go test: 105 passed in 5 packages
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | Neither `topics` nor `main` has `model.go`, `entity.go`, `rest.go`, or `provider.go` — `find services/atlas-kafka-precreate -name '*.go'` shows only `topics.go`, `topics_test.go`, `main.go` in scope, none of those names. |
| FILE placement (FILE-01..06) | Fired | Runs on every changed Go package unconditionally. |
| SUB sub-domain (SUB-01..04) | N/A | No `resource.go` in either changed package. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No `resource.go`/`rest.go`/`processor.go`, no HTTP route registration anywhere in the diff. |
| Constants reuse (DOM-21) | Fired | Diff declares `type compactTopicConfig struct` and a `const (...)` block — `internal/topics/topics.go:22-79`. |
| Testing (DOM-10,20,24,33) | Fired | `internal/topics/topics_test.go` changed. |
| Cache (DOM-29) | N/A | No `cache.go`; no processor/struct holds cached state. |
| Messaging (DOM-30) | N/A | No `producer.go`; no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call anywhere in the diff. |
| Multi-tenancy (DOM-31) | N/A | No `rest.go`; no tenant/trace state read or passed. |
| Migration hygiene (DOM-34,35) | N/A | Diff moves no symbols to/from `libs/atlas-*`. |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module; no topic env var added or renamed — README.md diff confirms the same `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` variables, only the values applied to existing compacted topics changed. |
| Runtime safety (DOM-26) | Fired | `topics.go` and `main.go` (non-test) changed. |
| Channel wire values (DOM-25) | N/A | No `atlas-channel`/`atlas-packet` files touched; no client-interpreted byte involved. |
| Resilience (DOM-27,28) | N/A | No DB-backed handlers (service has no DB); no `model.Decorator`/enrichment path changed. |
| External clients (EXT-01..04) | N/A | No `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call — this package talks to Kafka via `kafka-go`, not to another atlas HTTP service. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no channel writer/handler, no `deploy/shared/routes.conf` change. |
| Security (SEC-01..04) | N/A | Service handles no auth, tokens, redirects, or secrets. |
| Foundational: patterns-provider.md | N/A | No provider defined or composed. |
| Foundational: patterns-functional.md | N/A | No curried constructor, decorator, or model combinator defined. |

## Checklist Results

### internal/topics (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor lives in `processor.go` | N/A | No `Processor` type/interface anywhere in this package. |
| FILE-02 | RestModel/Transform in `rest.go` | N/A | No `RestModel` type. |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No cross-atlas-service HTTP calls; this package only calls `kafkaops.AdminClient` (Kafka admin RPCs). |
| FILE-04 | Entity/`Migration`/`TableName` in `entity.go` | N/A | No entity struct. |
| FILE-05 | Builder/Model/administrator/provider/state split | N/A | Package has no domain `Model`, no writes to a DB, no `state.go` enum. |
| FILE-06 | No package-named catch-all carrying ≥2 of FILE-01..05's responsibilities | PASS | `topics.go` carries 0 of the FILE-01..05 responsibilities (no Processor, RestModel, requests, Entity, or Builder+Model+administrator+provider+state) — it is Kafka-admin orchestration code (`Ensure`, `Settle`, `EndOffsets`), which is exactly what a support package is for. `services/atlas-kafka-precreate/internal/topics/topics.go:1-6` (package doc states this scope explicitly). |
| DOM-21 | No redeclaration of an atlas-constants type/const | PASS | `compactTopicConfig` (`topics.go:69-72`) and the const block `topics.go:22-56` (`compactCleanupPolicy`, `compactMaxCompactionLagMs`, `compactSegmentMs`, `compactMinCleanableDirtyRatio`) are Kafka broker config-name/value pairs. `grep -rl "cleanup.policy\|max.compaction.lag\|segment.ms" libs/atlas-constants/` returns nothing — no shared equivalent exists in `libs/atlas-constants/`. |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | No GORM DB opened in `topics_test.go` — service has no DB at all. |
| DOM-20 | Tests are table-driven | PASS | `TestEnsure_CreateErrors` (`topics_test.go:191-287`) uses the `tests := []struct{...}` + `t.Run` pattern for its 6 cases. The two new single-scenario tests (`TestEnsure_CompactConfigsMatchAcrossRequests`, `topics_test.go:391-459`; `TestEnsure_PlainTopicsCarryNoConfig`, `topics_test.go:464-513`) are single-case assertions, matching the guideline's own canonical example (`testing-guide.md:22-26`, `TestBuilderValidation`, a single-case non-table test) — table-driven applies where there are multiple cases to enumerate, not as a mandatory wrapper around one scenario. |
| DOM-24 | Emit-path tests stub the producer | N/A | Neither `Ensure`, `Settle`, nor `EndOffsets` reaches `AndEmit`/`message.Emit`/`producer.Produce` — these are Kafka *admin* RPCs (`CreateTopics`, `IncrementalAlterConfigs`, `Metadata`, `ListOffsets`), not the message-production path. |
| DOM-33 | Interface change updates every mock | N/A | `kafkaops.AdminClient` (`internal/kafkaops/ops.go:17`) is unchanged in this diff — `git diff` over `internal/kafkaops/`, `internal/groups/`, `internal/discover/` is empty. The test-local `stubClient` in `topics_test.go` already implements all seven `AdminClient` methods (`topics_test.go:25-81`), unchanged in shape by this diff. |
| DOM-26 | No bare `go` statement without a guard marker | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])' --include='*.go' internal/topics main.go` returns no hits. `tools/goroutine-guard.sh` exits 0 (verified from repo root). |

### main (support package, `services/atlas-kafka-precreate/main.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | (same responsibilities) | N/A | `main.go` is the Job's entrypoint (`run()` orchestrating phases A-E) — no `Processor`, `RestModel`, cross-service requests, `Entity`, or domain `Model`/administrator/provider/state exist here. |
| FILE-06 | No catch-all collapsing ≥2 responsibilities | PASS | `main.go` carries 0 of the FILE-01..05 responsibilities; it is the command's `main`/`run` wiring, which is what a `main` package is for. |
| DOM-21 | No constants redeclaration | N/A | `main.go`'s diff (`main.go:72-79`) adds no new type or const block — it only extends an existing `logrus.Fields{}` literal and calls `topics.CompactConfigNames()`. |
| DOM-26 | No bare `go` statement | PASS | Same grep as above covers `main.go`; no hits. |

## Not evaluable from the diff

None. The change is self-contained within `internal/topics` and `main.go`; the
one cross-package reference (`kafkaops.AdminClient`) was confirmed unchanged
by diffing `internal/kafkaops/` directly, and the `stubClient` test double
implementing it is in the same changed test file, so nothing needed reading
outside the review surface.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

None.
