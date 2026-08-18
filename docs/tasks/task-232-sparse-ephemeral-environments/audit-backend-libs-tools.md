# Backend Audit — task-232-sparse-ephemeral-environments (libs/ + tools/ shard)

- **Scope:** `libs/` and `tools/` Go files changed between
  `c8d44127cbb9eb2016c621463f86614b81c618e7` and
  `418b2caf97da2f1c326cafaadca9218456d63daf` (86 files; enumerated via
  `git diff --name-only ... -- libs tools | grep '\.go$'`).
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-16
- **Build:** PASS (per changed module — see below)
- **Tests:** PASS (per changed module — see below)
- **Overall:** PASS (no FAIL findings within this shard's scope; several items
  are structurally out of scope for a libs/tools-only shard and are recorded
  under "Not evaluable from the diff")

## Build & Test Results

This shard does not have a single `atlas.com/<module>` root — the changed
files span nine independent Go modules (six `libs/*`, plus five standalone
`tools/*` modules not listed in the root `go.work`). Each was built/tested
individually:

```
libs/atlas-env        go build ./... -> ok   | go test ./... -count=1 -> ok  (0.004s)
libs/atlas-kafka       go build ./... -> ok   | go test ./... -count=1 -> ok  (consumer 7.13s, producer 0.007s, retry 0.89s, others ok/no-test)
libs/atlas-outbox      go build ./... -> ok   | go test ./... -count=1 -> ok  (0.095s)
libs/atlas-redis       go build ./... -> ok   | go test ./... -count=1 -> ok  (0.169s)
libs/atlas-rest        go build ./... -> ok   | go test ./... -count=1 -> ok  (requests 11.0s, server 0.006s, degrade 0.005s)
libs/atlas-service     go build ./... -> ok   | go test ./... -count=1 -> ok  (1.059s)
tools/envguard         GOWORK=off go build/test -> ok (0.2s)
tools/producerseamguard GOWORK=off go build/test -> ok (0.1s)
tools/rediskeyguard    GOWORK=off go build/test -> ok (0.44s)
tools/scopeguard       GOWORK=off go build/test -> ok (1.3s)
tools/atlasguards      GOWORK=off go build -> ok (no test files)
```

`tools/*` modules are excluded from `go.work`, so each was built with
`GOWORK=off` from its own directory (each carries its own `go.mod`). All nine
modules build and test cleanly.

## Focus Area: Direct-emit / Outbox-bridge Header Parity

The task brief flagged this as the seam where a parity divergence was found
and fixed late on the branch. Traced end-to-end:

- **Direct path** — `libs/atlas-kafka/producer/provider.go:14-23`
  (`ProviderImpl`) composes exactly three decorators: `SpanHeaderDecorator`,
  `TenantHeaderDecorator`, `EnvHeaderDecorator` (added at line 18), folded by
  `Produce`'s `DecorateHeaders` (`libs/atlas-kafka/producer/producer.go:45-75`).
- **Outbox path** — `libs/atlas-outbox/bridge.go:54-71` (`headerMap`) composes
  the identical three decorators, imported directly from the producer
  package, and merges them into one map before `EnqueueBuffer` persists the
  headers on the outbox row (bridge.go:21-48).
- **Regression test proving the fix** —
  `libs/atlas-outbox/bridge_test.go:121-163`
  (`TestEnqueueBuffer_HeaderParityWithDirectPath_IncludesEnvironment`) is
  explicitly commented "task-232: the outbox path dropped ENVIRONMENT while
  the direct path... attached it... Assert parity through the exported
  bridge entry point." It builds a reference header set from the same three
  producer decorators, drains the outbox through a real `Drainer`, and
  asserts `require.Equal(t, want, got)` on the published header set —
  including `env.Key`. A companion test
  (`TestEnqueueBuffer_NoEnvironmentHeaderForLegacyEnvironment`,
  bridge_test.go:170-194) asserts the inverse: a context with no environment
  produces NO `ENVIRONMENT` header through the outbox path either, matching
  the direct path's legacy byte-identical behavior.
- **Guard against regression** —
  `tools/producerseamguard/analyzer.go` bans any new direct
  `producer.Produce` call site under `services/` outside a four-entry
  allowlist (all four predating and already updated to carry
  `EnvHeaderDecorator`, per the doc comment at analyzer.go:8-11), forcing all
  future emit call sites through the composed `ProviderImpl` seam.

**Verdict: the fix holds.** Both paths compose the same three-decorator set
from the same shared functions (no duplicated/copy-pasted decorator logic),
and the regression test that encodes the original bug asserts full header-set
equality by name, not just presence of the tenant/span keys.

## Domain Checklist Results

None of the changed packages are DOM (`model.go`-bearing) or SUB
(`resource.go`-bearing, no `model.go`) packages — every changed file in this
shard is either shared-library infrastructure (`libs/atlas-env`,
`libs/atlas-kafka/{producer,consumer}`, `libs/atlas-outbox`,
`libs/atlas-redis`, `libs/atlas-rest/{requests,server}`,
`libs/atlas-service`) or standalone static-analysis tooling (`tools/*`). The
DOM-01..DOM-28 / SUB-01..SUB-04 / FILE-01..FILE-06 checklists are written
against the domain-package file-responsibilities table (`processor.go`,
`rest.go`, `entity.go`, `builder.go`, `administrator.go`, `provider.go`,
`resource.go`) and do not map onto this shard's package shapes. Rather than
force-fitting every ID onto non-domain code, the mechanically-checkable IDs
that DO generalize to any Go package were run directly:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-12/DOM-14/DOM-15 (generalized: no handler-layer violations) | N/A — no `resource.go` in this shard | N/A | No file matching `resource.go` in the changed-file list |
| DOM-21 | No duplication of atlas-constants types | PASS | New types in this shard (`env.Id`, `env.Record`, redis generic registries keyed by tenant/env) have no equivalent in `libs/atlas-constants` (item/inventory/weapon/world/job/skill/monster ids) — orthogonal domain (environment routing, not game data) |
| DOM-23 | Kafka topic naming convention (env-configmap.yaml entries) | Not evaluable from this shard | `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` / `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` are read via `os.Getenv` in `libs/atlas-service/envregistry.go:52,59`; `deploy/k8s/env-configmap.yaml` and `deploy/k8s/*.yaml` are outside `libs/` and `tools/` and out of this shard's scope per the SHARD instruction |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `libs/atlas-outbox/bridge_test.go` uses a `fakePublisher`/`PublisherFunc`, never the real Kafka writer. `libs/atlas-kafka/producer/provider_test.go` (`TestProviderImplComposesSpanAndTenantHeaders`, `TestProviderImplComposesTheEnvironmentDecorator`) uses `MockWriter` via `ConfigWriterFactory`, the package's own low-level test double — these are the producer package's own unit tests, not a downstream consumer needing `producertest.InstallNoop()`. No `t.Cleanup(producer.ResetInstance)` pattern that defeats a `TestMain`-installed stub was found (`ResetInstance` cleanup here is paired with an explicit per-test `ConfigWriterFactory` install, not a `TestMain`-level stub) |
| DOM-26 | Goroutines spawned via `routine.Go` | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])'` over all 40 changed non-test, non-testdata `.go` files in this shard returns zero matches. `libs/atlas-service/foreach.go:65-74` (`ForEachOwnedEnvironmentConcurrently`) spawns via `routine.Go(el, c, func(gc context.Context) {...})`, not a bare `go` statement |
| DOM-27 | Transient DB errors -> 503 | N/A | No `WriteHeader(http.StatusInternalServerError)` in any changed file in this shard (`grep` zero matches); this shard has no DB-backed `resource.go` |
| DOM-28 | No silent degradation in decorators/enrichment | PASS (by inspection) | `env.FromContext` (`libs/atlas-env/env.go:66-69`) never errors by design (documented: "FromContext never errors... it returns a Provider anyway") — this is the intentional FR-1.8 legacy-default contract, not a silently-dropped failure. `EnvHeaderDecorator`/`EnvHeaderParser` on both the producer and REST-client side discard the (always-nil) error from `env.FromContext` — consistent with that documented contract, not an anti-pattern instance |

## Consumer-side Environment Reconciliation (task-232 core mechanism)

Traced `libs/atlas-kafka/consumer/header.go` (`EnvHeaderParser`,
lines 79-98) and `libs/atlas-kafka/consumer/gate.go` (`decide`, lines 52-71):

- `EnvHeaderParser` is documented (header.go:70-72) as requiring registration
  **after** `TenantHeaderParser` so the tenant is already on context when it
  reconciles. This ordering is enforced only by doc comment and the test at
  `libs/atlas-kafka/consumer/header_env_test.go:94`
  (`TestSetHeaderParsersReconcilesInOrder`), which drives
  `SetHeaderParsers(TenantHeaderParser, EnvHeaderParser)` in the correct
  order — there is no compile-time or runtime guard against a caller
  registering them in the wrong order. Every service's own `main.go` decides
  parser order via `consumer.SetHeaderParsers(...)`, which is out of this
  shard's scope (see "Not evaluable from the diff" below).
- The ownership gate (`manager.go:620-636`, `decide` in gate.go) runs before
  the tracing span and before any domain handler, and both drop paths
  (`gateDropUnresolvable`, `gateSkipNotOwner`) return `true` (ack) rather than
  `false`, correctly avoiding a wedged partition — matches the documented
  intent (FR-4.4/FR-4.7/D4) and is covered by `gate_test.go`.

## Sub-Domain Checklist Results

N/A — no `resource.go`-bearing packages without `model.go` in this shard's
changed-file set.

## File Responsibilities Checklist Results

N/A in the domain-package sense (see Domain Checklist Results above for why).
Spot-checked for the anti-pattern this checklist exists to catch — a single
file bundling multiple unrelated responsibilities (Processor + RestModel +
requests) — and found none: every changed file in this shard has a single,
named responsibility matching its filename (`header.go` = header
decorators/parsers only, `gate.go` = ownership-gate decision only,
`registry.go` = the in-memory projection only, `keys.go` = key-prefix
composition only, `bridge.go` = outbox enqueue only). No `<pkg>.go`
catch-all file was introduced.

## External HTTP Client Checklist

N/A — no new package in this shard calls another Atlas service via
`requests.GetRequest[T]`/`requests.PostRequest[T]`. The changed
`libs/atlas-rest/requests/*.go` files modify the shared REST-client
*library* itself (header decorator composition, `RootUrlFor`), not a
service-specific client package, so EXT-01..EXT-04 (which target a
client package's target `RestModel`, httptest coverage, 404-vs-other error
handling) do not apply to library-internal changes.

## Service Scaffolding Checklist

N/A — no new `services/atlas-<service>/` directory and no atlas-channel
Writer/Handler registration in this shard's changed-file set.

## Security Review

N/A — this shard touches no auth/session/token code.

## Not evaluable from the diff

- **DOM-23** (Kafka topic naming convention, `env-configmap.yaml` entries for
  `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` /
  `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`) — would require reading
  `deploy/k8s/env-configmap.yaml` and `deploy/k8s/*.yaml`, which are outside
  `libs/` and `tools/` and explicitly assigned to another shard.
- **Consumer header-parser registration order** (`SpanHeaderParser`,
  `TenantHeaderParser`, `EnvHeaderParser` must be registered in that order
  per `header.go:70-72`'s documented precondition) — the lib only supplies
  the pieces and a unit test proving correct-order behavior; whether every
  service's `main.go` actually calls `consumer.SetHeaderParsers(...)` in the
  required order is a per-service fact outside `libs/` and `tools/`.
- **REST-handler equivalent ordering** (`ParseEnvironment` then `ParseTenant`
  in `libs/atlas-rest/server/register.go`) — this one IS in-shard and IS
  fixed at the library level (`RegisterHandler`/`RegisterInputHandler`/
  `RegisterSimpleHandler`/`RegisterSimpleInputHandler` all hard-code
  `ParseEnvironment` wrapping `ParseTenant`), so no equivalent "Not
  evaluable" gap exists for the REST inbound path — noted here only to
  contrast with the Kafka consumer path above, where the order is
  caller-configurable.
- **DOM-22** (Dockerfile 4-mention check per direct `go.mod` require) — not
  applicable to `libs/`/`tools/` modules, which have no Dockerfile; would
  apply to the services that vendor these libs, out of this shard's scope.
- **SCAFFOLD-07** (tenant opcode template seeding) — no atlas-channel
  Writer/Handler in this shard; N/A rather than unevaluated, but noted since
  the branch as a whole touches atlas-channel per other shards.

## Summary

### Blocking (must fix)
- None found in this shard's scope.

### Non-Blocking (should fix)
- None found in this shard's scope. The one structural soft spot noted above
  (Kafka consumer header-parser ordering enforced only by doc comment + test,
  not by the type system) is a design tradeoff consistent with the rest of
  `libs/atlas-kafka/consumer`'s existing `HeaderParser` composition API, not a
  regression introduced by this branch — flagged for awareness, not as a
  finding, since fixing it would be a library-wide API redesign outside this
  task's scope.
