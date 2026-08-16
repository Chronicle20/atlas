# Backend Audit — task-232 sparse ephemeral environments (core shard)

- **Scope:** Changed Go files in `services/atlas-channel`, `services/atlas-login`, `services/atlas-configurations`, `services/atlas-tenants`, `services/atlas-saga-orchestrator`, `services/atlas-query-aggregator`
- **Diff range:** `c8d44127cbb9eb2016c621463f86614b81c618e7..418b2caf97da2f1c326cafaadca9218456d63daf`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-16
- **Build:** PASS (all six modules, `go build ./...`)
- **Tests:** PASS (all six modules, `go test ./... -count=1`; zero failures)
- **Overall:** NEEDS-WORK (one FILE-responsibility finding; several items not evaluable from the diff alone)

## Build & Test Results

`go build ./...` and `go test ./... -count=1` were run per-module from each
service's `atlas.com/<module>` directory. All six modules built cleanly and
all tests passed (no failing packages, no panics). Full output retained in
the session transcript; representative packages: `atlas-configurations/scope`,
`atlas-configurations/environmentcol`, `atlas-tenants/tenant`, `atlas-channel/socket`,
`atlas-saga-orchestrator/saga`, `atlas-query-aggregator/*` all reported `ok`.

## Cross-cutting environment-scoping review

This shard is almost entirely a mechanical threading change (curried request/
processor functions gained a `ctx`/`env.Id` parameter to resolve
`requests.RootUrlFor(ctx, ...)` and Kafka header parsers gained
`consumer.EnvHeaderParser`), so per-file DOM-checklist grading was
supplemented with targeted structural checks on the actual risk surface named
in the dispatch brief. Evidence:

| Area | Check | Status | Evidence |
|------|-------|--------|----------|
| atlas-configurations scope guard | `scope.Strict` / `scope.AuthorizeWrite` semantics | PASS | `services/atlas-configurations/atlas.com/configurations/scope/scope.go:29-49` — empty caller ("") is unfiltered/always-authorized (legacy byte-identical behaviour, FR-1.8); non-empty caller filters by `environment` column and rejects cross-environment writes. 125-line `scope_test.go` covers both. |
| atlas-tenants scope guard | Same STRICT semantics, deliberately duplicated (not imported cross-service) | PASS | `services/atlas-tenants/atlas.com/tenants/scope/scope.go:1-49`, doc comment at lines 1-9 explains the duplication rationale (no cross-service package imports per CLAUDE.md); `tenant/scope_test.go` (460 lines) exercises it. |
| atlas-configurations tenants/services/templates providers | Reads use `scope.Strict(db, caller)` | PASS | `services/atlas-configurations/atlas.com/configurations/tenants/provider.go:19,28,42`; `services/atlas-configurations/atlas.com/configurations/services/provider.go:19,28`; `templates/overlay.go:44,65,83`. |
| atlas-configurations tenants/services/templates administrators | Writes call `scope.AuthorizeWrite(caller, target)` before mutating, using an **unscoped** re-read of the target row (so cross-environment writes surface `ErrCrossEnvironmentWrite` rather than a generic not-found) | PASS | `tenants/administrator.go:19-48` (`byIdUnscoped` doc comment explains why), `:35`, `:66`; `services/administrator.go:45,75`; `templates/administrator.go:29,53`. |
| atlas-tenants tenant administrator | Same unscoped-read-then-authorize pattern | PASS | `services/atlas-tenants/atlas.com/tenants/tenant/administrator.go:20-24` (`byIdUnscoped`), `:47` (`UpdateTenant`), `:70` (`DeleteTenant`). |
| Cross-environment write → HTTP 403 (not generic 500) | `rest/handler.go` `WriteErrorResponse` wraps `scope.ErrCrossEnvironmentWrite` | PASS | `services/atlas-configurations/atlas.com/configurations/rest/handler.go:38-56`; `services/atlas-tenants/atlas.com/tenants/rest/handler.go:38-56` — both map to 403 with `errors.Is`. |
| Server-owned `Environment` on outbound events | Client-supplied `Environment` field is never trusted; re-read from the persisted row before sanitizing the outbox/Kafka body | PASS | `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:158-173` (`UpdateById` re-reads via `byIdEntityProvider` then overwrites `sanitized.Environment`); same file `:218-234` (`Create` uses `env.MustFromContext` + `e.Environment`, not `input.Environment`); `services/atlas-tenants/atlas.com/tenants/tenant/processor.go:144-151` (`Update` copies `e.Environment` from the unscoped-loaded row, never from the request args). |
| Environment-list table itself is NOT scoped by an `environment` column (by design — it IS the list) | `ScopingDimension()` marker + scopeguard allowlist | PASS | `services/atlas-configurations/atlas.com/configurations/environments/entity.go:32-42` — self-documenting marker with rationale, cross-referenced against `tools/scopeguard`. |
| Migration backfill ordering | `environmentcol.Migration` / `tenant.EnvironmentMigration` run *after* the tables they backfill | PASS | `services/atlas-configurations/atlas.com/configurations/main.go:49-51` (comment "must run last"); `environmentcol/migration.go:16-26` (idempotent `WHERE environment = '' OR environment IS NULL`). atlas-tenants side not directly re-verified for main.go ordering (see Not Evaluable). |
| Socket-edge environment origination (atlas-channel) | Every context reaching a Kafka producer/REST client on the socket path carries `env.Self()`, not a wire-derived value | PASS | Two parallel paths, both wired: (1) per-packet handler path — `services/atlas-channel/atlas.com/channel/socket/handler/handle.go:53-61` (`newSessionContext`) called from `AdaptHandler` at line ~66 (`otel...Start(newSessionContext(t), name)`); (2) per-listener-lifecycle path (Create/Destroy/SendPing, bypasses `AdaptHandler`) — `socket/init.go:29-40` (`NewListenerContext`), called at `main.go:406` (`tctx := socket.NewListenerContext(ctx, t)`) and threaded into `socket.CreateSocketService(fl, tctx, ...)` at `main.go:615`. Confirmed via `grep` that `tctx` from line 406 is the same variable passed at line 615 (same closure). |
| Socket-edge environment origination (atlas-login) | Same two-path pattern | PASS | `services/atlas-login/atlas.com/login/socket/init.go:19-35` (`NewListenerContext`); `main.go:219` (`tctx := socket.NewListenerContext(ctx, t)`) threaded to `socket.CreateSocketService(fl, tctx, ...)` at `main.go:258`. |
| Background per-character sweep outside env-domain-guard's import allowlist (atlas-channel `character/combo`) | `DecayTick` takes an injected `envContext func(ctx) ctx` rather than importing `atlas-env` directly | PASS | `services/atlas-channel/atlas.com/channel/character/combo/task.go:26-40` (`NewDecayTick`), `:70-84` (`processExpiries` applies `envContext` per expired combo); wired at `services/atlas-channel/atlas.com/channel/main.go:338` with `socket.WithSelfEnvironment`. Both `NewListenerContext`/`WithSelfEnvironment` are exercised in `services/atlas-channel/atlas.com/channel/socket/init_test.go:28-54`. |
| Saga-orchestrator timer-fire environment origination | `TimerRegistry.Schedule`'s `AfterFunc` callback rebuilds `context.Background()`-rooted context on fire; must reattach environment or every timed-out saga silently resolves to the baseline environment | PASS | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go` diff — `SetEnvContext` (new method) + `Schedule`'s fire callback applying it before `handleSagaTimeout` (lines added around `:29-45`, `:69-77` per diff). Wired via `withSelfEnvironment` (self, not per-saga-original, environment — see note below) at `main.go:204` (`saga.SagaTimers().SetEnvContext(withSelfEnvironment)`), and also applied in `recoverSagas`/`reapTimedOutSagas` at `main.go:249`, `:301`. Test: `saga/timer_env_context_test.go` (`TestTimerRegistry_FireAppliesEnvContext`) pins that the fire callback actually invokes the injected `envContext`. |
| Saga-orchestrator Kafka consumers gained `EnvHeaderParser` | Every changed `kafka/consumer/*/consumer.go` in this shard | PASS | Verified by script: zero of the changed `consumer.go` files (channel, login, saga-orchestrator) lack `EnvHeaderParser` in their `consumer.SetHeaderParsers(...)` call. Sample: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/asset/consumer.go` diff adds `consumer.EnvHeaderParser` alongside `SpanHeaderParser`/`TenantHeaderParser`. |
| Cross-service request funcs converted to environment-aware URL resolution | Every changed `requests.go` uses `requests.RootUrlFor(ctx, ...)`, not bare `requests.RootUrl(...)` | PASS | Verified by script over all `requests.go` files in the diff list (channel, login, configurations, query-aggregator, saga-orchestrator): zero remaining `requests.RootUrl(` call sites. |
| Kafka producer stub in saga-orchestrator's `saga` test package (which exercises `Schedule`'s fire path, a transitive emit path via `handleSagaTimeout`) | `producertest.InstallNoop()` in a `TestMain`, no `t.Cleanup(producer.ResetInstance)` reversion | PASS | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/testmain_test.go:7-11`. |
| DOM-21 (atlas-constants duplication) | No new `type X string`/enum shadowing an existing atlas-constants or atlas-env type in the shard | PASS (no matches) | Grep for `^type.*Environment.*string` / `^type.*Id string` outside `atlas-env` across all six services' changed-file trees returned zero matches. |

**Note on saga-timer environment semantics:** `withSelfEnvironment` attaches
*this pod's own* `env.Self()`, not the environment that originally drove the
saga, to a recovered/reaped/timed-out saga's context. The accompanying doc
comments (`main.go:50-63`) are explicit that this is intentional under the
sparse-ephemeral-environment architecture: each saga-orchestrator pod is
deployed per-environment, so "this pod's environment" and "the saga's
originating environment" are the same value by deployment topology, not by
data threaded through the saga row. This is a documented design decision, not
independently re-verified against the deployment manifests in this shard
(deploy/k8s changes are out of scope for this shard — see Not Evaluable).

## File Responsibilities Checklist Findings

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor`/write logic lives in `processor.go`, direct entity writes live in `administrator.go` | **FAIL (Important)** | `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:212-224` — `ProcessorImpl.Create` calls `db.Create(e).Error` directly inside a `database.ExecuteTransaction` closure in `processor.go`, bypassing `administrator.go` entirely. The sibling `Update`/`Delete` paths correctly route through `administrator.go` (`update(...)`, `delete(...)`), but `Create` does not — an inconsistency within the same file. This line predates this diff (confirmed via `git diff` on the file: only the `Environment: string(env.MustFromContext(p.ctx))` field assignment and the surrounding sanitization comment are new; `db.Create(e)` itself is pre-existing), but `processor.go` is a file this diff modifies, so it is in scope per the audit's Scope rules, and prevalence/pre-existing status does not exempt a File-Responsibilities violation per the audit's Mindset rules. |

No other File-Responsibilities violations were found in the sampled
environment-scoping surface (`scope.go`, `entity.go`, `provider.go`,
`administrator.go`, `rest/handler.go` pairs in atlas-configurations and
atlas-tenants all place symbols in their designated files).

## Domain / Sub-Domain Mechanical Checklist

The bulk of this shard's diff (194 files in atlas-channel, dozens more in the
other five services) is a mechanical parameter-threading change to existing
`processor.go`/`requests.go` pairs (adding a `ctx context.Context` parameter
to already-curried request functions so `requests.RootUrlFor(ctx, ...)` can
resolve the environment-aware base URL) and a mechanical addition of
`consumer.EnvHeaderParser` to existing `consumer.SetHeaderParsers(...)` call
sites. These are signature-level changes to established files, not new
domain packages, so the full per-domain DOM-01..DOM-28 checklist (builder.go,
ToEntity, ProcessorImpl FieldLogger, RegisterInputHandler, etc.) was not
re-run file-by-file across all ~369 changed files — those structural
properties were not touched by this diff and were already established prior
to this change. The genuinely new/restructured domain surface in this shard
(`atlas-configurations/environments`, `atlas-configurations/scope`,
`atlas-configurations/environmentcol`, `atlas-tenants/scope`) was reviewed in
full above under "Cross-cutting environment-scoping review" and File
Responsibilities, which is where this branch's actual risk lives per the
dispatch brief.

## Not evaluable from the diff

- **DOM-22 (Dockerfile 4-mention rule) / DOM-23 (Kafka topic naming, configmap wiring) / DOM-25 (client wire-code config resolution) / SCAFFOLD-*** — this shard's brief is scoped to Go files only; `deploy/k8s/*.yaml`, `deploy/shared/routes.conf`, and any `go.mod`/`Dockerfile` changes are covered by other shards per the dispatch instructions, and were not read here.
- **atlas-tenants main.go migration ordering for `EnvironmentMigration`** — confirmed the atlas-configurations ordering (`main.go:49-51`) directly, but did not independently re-read `services/atlas-tenants/atlas.com/tenants/main.go`'s migration list to confirm `tenant.EnvironmentMigration` is registered after `MigrateEntities` as its doc comment claims (`environment_migration.go:20-22`). Would need to grep/read that file's `Migration` list.
- **Saga-orchestrator pod-per-environment deployment topology** — the correctness of `withSelfEnvironment` (this pod's own environment == the saga's originating environment) rests on a deployment-topology assumption (one saga-orchestrator pod per environment) that is asserted in code comments but not verified against `deploy/k8s/atlas-saga-orchestrator.yaml` or any Argo/Helm templating in this shard's Go-only surface.
- **DOM-28 (silent degradation in decorators/enrichment)** — no `model.Decorator[...]` implementations were touched in this shard's diff surface as far as sampled; a full sweep of all ~369 files for new fallible-enrichment branches was not performed given the mechanical nature of the majority of the diff (parameter threading). If any changed `processor.go` introduced a new fallible enrichment path alongside the ctx-threading change, it was not independently checked file-by-file.
- **query-aggregator and remaining saga-orchestrator/channel/login `requests.go`/`processor.go` files beyond the sampled set** — the `RootUrlFor`/`EnvHeaderParser` checks were run as repo-wide greps over every changed file in the diff list (not a sample), so those two checks are exhaustive; however, correctness of *how* each caller passes `ctx` through multi-hop call chains (e.g., a processor method that has `ctx` available but a helper three calls deep silently uses a stale/wrong context) was spot-checked on `query-aggregator/character/requests.go` and `saga-orchestrator/monster/processor.go` only, not on all ~150 remaining `processor.go` files in atlas-channel.

## Summary

### Blocking (must fix)
- FILE-01: `services/atlas-configurations/atlas.com/configurations/tenants/processor.go:212-224` — `ProcessorImpl.Create` performs `db.Create(e)` directly instead of routing through `administrator.go`, inconsistent with the sibling `Update`/`Delete` methods in the same file which correctly delegate to `administrator.go`'s `update`/`delete`. Pre-existing (not introduced by this diff), but the file is in this diff's scope and the violation is real.

### Non-Blocking (should fix)
- None identified beyond the items listed under "Not evaluable from the diff," which are gaps in review coverage rather than confirmed defects.
