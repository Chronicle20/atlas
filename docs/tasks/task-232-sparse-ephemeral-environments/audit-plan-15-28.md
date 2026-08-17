# Plan Audit — task-232-sparse-ephemeral-environments (Tasks 15–28)

**Plan Path:** docs/tasks/task-232-sparse-ephemeral-environments/plan.md
**Audit Date:** 2026-08-16
**Branch:** task-232-sparse-ephemeral-environments
**Base Branch:** main
**Range Under Review:** Tasks 15–28 only (plan.md lines ~1600–4389). Other tasks are covered by sibling shards.

## Executive Summary

All 14 tasks in this range (15–28) are fully implemented and verified against
the diff. Every interface named in the plan's "Produces" sections exists at
the expected file, every plan-specified test scenario is present and passing,
and both new analyzer guards (`scope-guard.sh`, `env-domain-guard.sh`) plus
the producer-seam guard run clean against the fleet. `libs/atlas-service`'s
`ForEachOwnedEnvironment` suite passes under `-race` as the plan requires.
Two minor, non-blocking deviations are noted below (Task 19's `model.go` was
folded into existing files; Task 16's sign-off section landed as "## 6"
rather than "## 4" because the audit document grew between when the plan was
written and when Task 16 executed) — neither weakens the acceptance criteria.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 15 | Scope guards (`control-plane-scope-guard`, `tenant-scope-guard`) | DONE | `tools/scopeguard/{analyzer.go,allowlist.go,cmd/}` implements both Rule 1 (data-plane) and Rule 2 (control-plane) checks (`analyzer.go:175-192`, messages `"control-plane entity without Environment"` / `"data-plane entity without TenantId"`). `tools/scope-guard.sh` exists and wired into `tools/verify.sh:305`. `tools/scopeguard/allowlist.txt` (84 lines) plus a later-added `callsite-allowlist.txt`. `./tools/scope-guard.sh` run directly: exit 0, "scopeguard: 87 module(s), 8 parallel". Commits `3b455232f`, `cdb60aacd`, `91e846592`, `be422f757`. |
| 16 | Phase A gate — full verification + sign-off | DONE | `query-scope-audit.md` carries a `## 6. Sign-off (PRD §12 prerequisites)` section (line 1108) with a per-FR-8 criterion row citing concrete evidence (test names, guard runs, commit hashes). Section number differs from the plan's illustrative "## 4" because earlier fix-round commits inserted sections 2–5 first; content and intent match. Commit `f0a76d8b0`. |
| 17 | `libs/atlas-env` — environment on the context | DONE | `libs/atlas-env/env.go` has `package env`, `Valid`, `WithContext`, `FromContext`, `MustFromContext`, `Self` (env.go:52-80). `go.mod` module path/Go version correct. `.gitignore` present. Registered in `go.work:7`. `go test ./...` → PASS. Commit `c02a1718e`. |
| 18 | `libs/atlas-env` — registry, in-memory impl, staleness | DONE | `registry.go` defines `Registry` interface and `MapRegistry` with `NewMapRegistry`, `Apply`, `ApplyTombstone`, `Observe`, `Stale`, `EnvironmentNamespace`, `ServiceNamespace`, `IsActive`, `IsOwner`, `EnvironmentsOwnedBy`, `BaselineOf`, plus `legacyRegistry`, `SetRegistry`, `CurrentRegistry` (registry.go:16-275). `record.go` present for the `Record` type. Commits `6a5b0c5fe`, `a36f1e2f8`. |
| 19 | Environment records in `atlas-configurations` | DONE (minor deviation) | `environments/` package has `entity.go`, `builder.go`, `provider.go`, `administrator.go`, `processor.go`, `rest.go`, `resource.go`, `heartbeat.go`, `processor_test.go`. `outbox/envelopes.go:51` has `NewEnvironmentEnvelope`. `main.go` wires `environments.Migration` (line 50), `environments.StartHeartbeat` (line 77), `AddRouteInitializer(environments.InitResource...)` (line 110). ConfigMap (`env-configmap.yaml:106`) and precreate job (`atlas-kafka-precreate.yaml:65`) both carry `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` in the compacted list. `go test ./environments/...` → PASS. **Deviation:** no standalone `model.go` file — the package uses `Entity`/`RestModel` directly (mirrors the `services` package's actual shape more than the plan's file list); functionally equivalent, not a gap. Commit `73d42cdb5` + fix-round commits `84a974d6e`, `f1cdfd300`, `7bf5f3fa8`, `1a95052d1`, `725a7396c`. |
| 20 | Project environment topic into registry (`libs/atlas-service`) | DONE | `libs/atlas-service/envregistry.go` has `WithEnvironmentRegistry` (line 34), `startEnvironmentRegistry` (line 51), `envSubscriber` (line 93) with `Start`/`handle`, `envCaughtUp` gate (lines 364-460) ANDed into readiness via `r.gates`/`r.envCaughtUp` (bootstrap.go:44,81). `go build && go test ./...` → PASS. Commit `b278a8c8b` (+ `13bd67a6c` fail-closed fix). |
| 21 | Project tenant → environment; FR-7.7 mismatch error | DONE | `libs/atlas-env/tenants.go`: `ErrEnvironmentMismatch` (line 34), `TenantResolver` interface (line 39), `Reconcile` (line 58) implementing the exact match/mismatch/derive/unknown-tenant/legacy logic from the plan. `envregistry.go:176,194` registers a second subscriber (`handleTenant`, line 237) on the tenant-status topic feeding `registry.ApplyTenant`. `go test ./... -run Reconcile -v` → all 5 plan-specified subtests PASS. Commit `7fe8518a4` (+ `6ff5b4efd`, `41f43fc17`). |
| 22 | `libs/atlas-rest` inbound — parse `ENVIRONMENT` header | DONE | `handler.go:41` `ParseEnvironment`, composed in `register.go` at all 4 registrar sites (lines 16, 31, 46, 59). FR-7.7 reconciliation wired at `handler.go:125` (`env.Reconcile(...)` inside `ParseTenant`'s successor). `atlas-ingress.yaml:53` forwards `proxy_set_header ENVIRONMENT $http_environment;`, with `underscores_in_headers on` already at line 27. `go build && go test ./...` → PASS across all `libs/atlas-rest` subpackages. Commit `650adeaf8`. |
| 23 | `libs/atlas-rest` outbound — target operation's own ingress | DONE | `requests/url.go:34` `RootUrlFor`, `:75` `replaceNamespace`; `requests/header.go:50` `EnvHeaderDecorator`. Wired at every `TenantHeaderDecorator` composition site (`paged.go:66`, `decorated.go:13,22,31,40,49`) — matches the plan's "add EnvHeaderDecorator at every TenantHeaderDecorator site in libs/" instruction exactly. `go test ./requests/ -run RootUrlFor -v` → all 6 subtests PASS (plan specified 5; a 6th, `TestRootUrlForAnEmptyNamespaceErrorsAndNeverFallsBack`, was added in fix-commit `44c36cf6e` closing a real bug where empty namespace silently fell back to baseline — strengthens, does not weaken, G4). Commits `2a3941e69`, `44c36cf6e`. |
| 24 | `libs/atlas-kafka` producer — `ENVIRONMENT` message header | DONE | `producer/header.go:53` `EnvHeaderDecorator`; `provider.go:18` composes it (`ed := EnvHeaderDecorator(ctx)`) into `ProviderImpl`. All 4 direct call sites updated (`atlas-quest` producer.go:97, saga/producer.go:28; `atlas-saga-orchestrator` reactor/processor.go:103, party_quest/processor.go:175 — each has `ed := producer.EnvHeaderDecorator(ctx)`). `tools/producerseamguard/` + `tools/producer-seam-guard.sh` created and wired into `verify.sh:321-324`; run directly: "producerseamguard: 64 module(s), 8 parallel", exit 0. `go build && go test ./...` → PASS for `libs/atlas-kafka` and both dependent services. Commits `1c3539f44`, `dcda19063`. |
| 25 | `libs/atlas-kafka` consumer — parse header, reconcile against tenant | DONE | `consumer/header.go:79` `EnvHeaderParser`, matching the plan's reconcile-and-mark-mismatch logic. `env.WithMismatch`/`env.Mismatched` added to `libs/atlas-env/tenants.go` (lines 20, 26) as specified. Registered after `TenantHeaderParser` at both `libs/` sites (`envregistry.go:176,194`, its own subscriber — the only `libs/` registration points, matching plan's scope note that `services/` sites are Phase C work). `go test ./consumer/ -run EnvHeader -v` → all 4 plan subtests PASS. Commit `4fe533881`. |
| 26 | The Kafka ownership gate | DONE | `consumer/gate.go`: `gateVerdict` enum (lines 14-16), `decide` (line 52) implementing the exact precedence (mismatch → drop; empty env → process; stale+foreign → drop; inactive → drop; not-owner → skip; else process) matching the plan's reference implementation verbatim. Wired into `processMessage` at `manager.go:617-641` with `c.service`, before the tracing span, both drop paths returning `true` (does not block the cursor) as the plan requires. `go test ./consumer/ -run "Gate\|ExactlyOne" -v` → all 9 subtests PASS including `TestExactlyOneDeploymentProcesses` and the counter-increment test. Commit `a9658782a`. |
| 27 | `ForEachOwnedEnvironment` — autonomous-iteration helper | DONE | `libs/atlas-service/foreach.go`: `ForEachOwnedEnvironment` (line 44), `ForEachOwnedEnvironmentConcurrently` (line 57), `eachOwned` (line 78), `safely` (line 98) — matches the plan's implementation shape (serial-by-default via `safely` with no goroutine; concurrent variant uses `routine.Go`). `go test ./... -run ForEachOwnedEnvironment -v` → all 7 plan-specified subtests PASS, and `go test -race ./...` → PASS (the serial-only assertion depends on this). Commit `8b090e1c4`. |
| 28 | `env-domain-guard` and `env-bootstrap-guard` | DONE | `tools/envguard/{analyzer.go,bootstrap-allowlist.txt,cmd/,testdata/}`, `tools/env-domain-guard.sh`, `tools/env-bootstrap-guard.sh` all present. Domain guard wired into `verify.sh:373-376` (bootstrap guard deliberately NOT wired here, per plan — deferred to Task 52, confirmed absent from `verify.sh`). `./tools/env-domain-guard.sh` → exit 0 ("envguard: 64 module(s), 8 parallel"). `./tools/env-bootstrap-guard.sh` now reports "clean (64 service(s) checked)" rather than the plan's expected initial failure — this is because later Phase C commits (batches 1–7, outside this shard's range) completed the `WithEnvironmentRegistry` wiring across all services before this audit ran; it is not evidence Task 28 itself was done incorrectly, since Task 28's own commit only had to produce the guard and record the expected-failure count at the time. Commit `7b6b22743` (+ `b66dc525e` widening the transport-edge exception). |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task in the 15–28 range has direct, verifiable evidence of completion.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `tools/scopeguard` | PASS | PASS | `GOWORK=off go build/test ./...` |
| `tools/envguard` | PASS | PASS | `GOWORK=off go build/test ./...` |
| `tools/producerseamguard` | PASS | PASS | `GOWORK=off go build/test ./...` |
| `libs/atlas-env` | PASS | PASS | includes Reconcile, registry, mismatch tests |
| `libs/atlas-service` | PASS | PASS (incl. `-race`) | ForEachOwnedEnvironment + envregistry subscriber tests |
| `libs/atlas-rest` | PASS | PASS | requests/server/degrade/paginate all green |
| `libs/atlas-kafka` | PASS | PASS | consumer (gate + header parser) and producer (header decorator) |
| `services/atlas-configurations/.../configurations` | PASS | PASS | includes `environments/` package tests |
| `./tools/scope-guard.sh` (fleet run) | — | PASS | exit 0, 87 modules |
| `./tools/env-domain-guard.sh` (fleet run) | — | PASS | exit 0, 64 modules |
| `./tools/producer-seam-guard.sh` (fleet run) | — | PASS | exit 0, 64 modules |

No `--quick`/flagless full-repo `tools/verify.sh` was run by this shard (out of scope for a task-range audit; other in-flight shards may be running it concurrently). All module-local builds/tests and the three new fleet-wide guards introduced in this range were run directly and pass.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch recommendation depends on the other shards' findings)

## Action Items

None required for Tasks 15–28. Two cosmetic notes for the record (not blocking):

1. Task 19: no standalone `model.go` file was created; the `environments` package uses `Entity` and `RestModel` directly. If a future reviewer expects file-for-file parity with the plan's file list, note this is an intentional simplification consistent with the `services` package it mirrors.
2. Task 16: the sign-off section landed as `## 6. Sign-off` rather than the plan's illustrative `## 4. Sign-off` — purely a section-numbering artifact of earlier fix-round commits inserting additional sections into `query-scope-audit.md`; content matches the plan's required table.