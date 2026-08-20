# Backend Audit — task-243-sparse-env-scoping-activation (Go diff)

- **Service Path:** services/atlas-channel, services/atlas-configurations, services/atlas-login, services/atlas-pr-bootstrap (shell only), libs/atlas-env, libs/atlas-kafka
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-20
- **Base:** d35509be0
- **Build:** PASS
- **Tests:** all packages `ok` (no `FAIL` lines) across libs/atlas-env, libs/atlas-kafka, atlas-channel, atlas-configurations, atlas-login
- **Overall:** NEEDS-WORK

## Scope

Reviewed the Go diff only (`git diff d35509be0..HEAD -- '*.go'`), 14 files:

- `libs/atlas-env/tenants.go`, `tenants_test.go`
- `libs/atlas-kafka/consumer/gate.go`, `gate_test.go`, `manager.go`
- `services/atlas-channel/.../configuration/projection/state.go`, `projection_test.go`, `main.go`
- `services/atlas-configurations/.../main.go`, `servicesuniq/migration.go`, `servicesuniq/migration_test.go`
- `services/atlas-login/.../configuration/projection/state.go`, `projection_test.go`, `main.go`

Shell/YAML (CI bootstrap, k8s overlays, bats tests) not graded against these Go
rules; checked only for a Go-side seam the shell implies (the `atlasServiceNS`
UUID namespace reciprocal reference — see below).

## Build & Test Results

```
libs/atlas-env:            go build ./... clean; go test ./... -count=1 -> ok (0.003s)
libs/atlas-kafka:          go build ./... clean; go test ./... -count=1 -> ok (all subpackages)
atlas-channel/atlas.com/channel:        go build ./... clean; go test ./... -count=1 -> ok, no FAIL lines
atlas-configurations/atlas.com/configurations: go build ./... clean; go test ./... -count=1 -> ok, no FAIL lines (servicesuniq: ok, 0.020s)
atlas-login/atlas.com/login:            go build ./... clean; go test ./... -count=1 -> ok, no FAIL lines
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (01-05,11,16) | N/A | No changed package in the Go diff has `model.go` |
| FILE placement (01-06) | Fired | Every changed Go package runs this unconditionally |
| SUB (01-04) | N/A | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package has `resource.go`/`rest.go`/`processor.go`, no HTTP routes registered |
| Constants reuse (DOM-21) | Fired (opened, all N/A) | `gateReason` type+const block (gate.go:19-32), `EnvServiceStatusTopic` const (migration.go:26), `DuplicateGroup` struct (migration.go:40) declared — none match a `libs/atlas-constants/` classification (item/inventory/weapon/world/job/skill/monster ids) |
| Testing (DOM-10,20,24,33) | Fired | New/changed `_test.go` files in every touched module |
| Cache (DOM-29) | N/A | No `cache.go`, no processor/struct holding cached state introduced |
| Messaging (DOM-30) | N/A | No `AndEmit`/`message.Emit`/`producer.ProviderImpl` call site in the diff (grep confirmed) |
| Multi-tenancy (DOM-31) | Fired (opened, PASS) | `tenants.go` `Reconcile` passes `tenantId string`, but only as an internal library-function parameter — no `rest.go` in any touched package |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved/extracted between a service and a `libs/atlas-*` module in this diff |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added; `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` reused, not added/renamed |
| Runtime safety (DOM-26) | Fired (opened, N/A) | Non-test Go files changed, but grep for bare `go ` statements found none added |
| Channel wire values (DOM-25) | Fired (opened, N/A) | `services/atlas-channel` touched, but no dispatcher/opcode/writer byte changed — only readiness-gate wiring |
| Resilience (DOM-27,28) | N/A | No DB-backed handler or `model.Decorator` touched |
| External clients (EXT-01..04) | N/A | No `requests.RootUrl`/`requests.*Request[T]` call site added (grep confirmed) |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/`, no new channel Writer/Handler, `routes.conf` untouched |
| Security (SEC-01..04) | Fired (opened, N/A) | `atlas-login` touched, but the only change is `service.WithReadinessGate(state.HasService)` wiring — no token/redirect/secret code in the diff |

## Checklist Results

### libs/atlas-env (support library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `tenants.go` is single-purpose (mismatch-context helpers + `Reconcile`); no Processor/RestModel/requests collapse |
| DOM-31 | Tenant identifiers travel in context only on the public surface | PASS | `Reconcile(r Registry, headerEnv Id, tenantId string)` (tenants.go:58) is an unexported-package internal function, not a REST model/handler — scope carve-out in patterns-multitenancy-context.md §Scope applies |
| DOM-20 | Table-driven tests | FAIL | `tenants_test.go:65-100` adds `TestReconcileTrustsTheHeaderForALegacyTenant`, `TestReconcileStillRejectsTwoNonEmptyDisagreements`, `TestReconcileWithALegacyTenantAndNoHeaderIsTheLegacyValue` as three independent `Test...` funcs testing the same `Reconcile` call under different (registry, headerEnv, tenantId) inputs — the exact shape `tests := []struct{...}` + `t.Run` exists for, not used |

### libs/atlas-kafka/consumer (support library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `gate.go` is a single-purpose gate/decision file (`gateVerdict`, `gateReason`, `decide`, metrics) |
| DOM-20 | Table-driven tests | PASS | `gate_test.go:99-224` `TestGateDropReasons` uses `tests := []struct{...}` + `t.Run` for the new reason-labelling behavior |
| DOM-26 | Goroutines via `routine.Go` | N/A | No `go` statement added in gate.go/manager.go (grep confirmed) |

### services/atlas-channel — configuration/projection (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all bundling | PASS | `state.go` holds only the `State` struct + its methods (pre-existing shape; `HasService` added at state.go:80-84 follows the same pattern as `ApplyTenantTombstone`) |
| DOM-20 | Table-driven tests | FAIL | `projection_test.go` (new block, +50 lines) adds `TestHasServiceIsFalseBeforeAnyServiceIsApplied`, `TestHasServiceIsTrueAfterTheMatchingServiceIsApplied`, `TestHasServiceIsFalseAgainAfterATombstone`, `TestHasServiceIsFalseAfterOnlyATenantIsApplied` — four sequential-state assertions on the same `HasService()` behavior, written as four separate `Test...` funcs instead of one table with `t.Run` |
| DOM-25 | Client-interpreted bytes resolved from tenant writer-options table | N/A | No dispatcher/opcode/writer byte touched — diff is limited to readiness-gate wiring (main.go:196-198) |

### services/atlas-login — configuration/projection (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all bundling | PASS | `state.go` mirrors the atlas-channel shape; `HasService` at state.go:80-84 |
| DOM-20 | Table-driven tests | FAIL | Same four non-table-driven `HasService` tests duplicated in `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go` (+50 lines, identical shape to the atlas-channel instance above) |

### services/atlas-configurations/atlas.com/configurations (main.go wiring)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all bundling | PASS | main.go:49-51 only adds one import + one line registering `servicesuniq.Migration` in the existing migration list; no new responsibility collapse |

### services/atlas-configurations/.../servicesuniq (support / data-migration package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-04 | Entity struct / `func Migration(` / `TableName()` live in `entity.go` | FAIL (Important) | `migration.go:70` defines `func Migration(db *gorm.DB) error`; the package has no `entity.go` at all — the function operates on the `services`/`service_history` tables owned by the sibling `services` package via raw SQL, never in an entity.go co-located with an entity struct. The rule's Pass criteria ("All in entity.go") is unmet. (Mirrors an existing gap in the pre-existing, out-of-scope `environmentcol/migration.go`, which does not exempt this new instance — see Mindset §prevalence.) |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `migration.go` is single-purpose (dedupe + index-creation logic only); no Processor/RestModel/requests/entity-struct collapse — the FILE-06 carve-out for "a genuine single-purpose utility" covers the file's *shape*, but does not cure the FILE-04 finding above, which is about the specific required *location* of a `Migration(` function |
| DOM-21 | No redeclaration of a `libs/atlas-constants` symbol | N/A | `EnvServiceStatusTopic` (migration.go:26) duplicates `services.EnvServiceStatusTopic` (services/processor.go:24), but DOM-21 is scoped to symbols in `libs/atlas-constants/` specifically (anti-patterns.md §DOM-21) — an intra-service duplication documented inline as import-cycle avoidance is outside this rule's text |
| DOM-30 | DB write + Kafka event emitted atomically via `AndEmit`/`message.Buffer` | N/A | No `AndEmit`/`message.Emit`/`producer.ProviderImpl` call in migration.go; the DB delete and the outbox enqueue are instead both inside the same `database.ExecuteTransaction` (migration.go:92-109) using the transactional-outbox library (`outboxlib.Enqueue`) — DOM-30's own trigger (the three named call patterns) never fires |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | `migration_test.go:49-76` opens a GORM DB directly, but the `services`/`service_history` tables carry no `TenantId` column (entity.go, confirmed by servicesuniq's own shadow entities at migration_test.go:26-44) — there is no tenant filtering for the callback to enable |
| DOM-24 | Test package reaching an emit path installs `producertest` | N/A | No `AndEmit`/`message.Emit`/`producer.Produce` call in migration_test.go — writes go straight to the `outbox_entries` table via GORM, not through the Kafka producer |
| DOM-20 | Table-driven tests | WARN | `migration_test.go` (9 new `Test...` funcs) are each a single independent scenario (dedupe-keeps-derived, dedupe-keeps-newest, dedupe-falls-back-to-lowest, idempotency, etc.) rather than one table with `t.Run`; each scenario differs enough in setup/assertion shape that forcing a single table would be more contorted than the atlas-env/projection cases above — downgraded to non-blocking |

## Not evaluable from the diff

- SCAFFOLD-09 (`tools/service-registration-guard.sh` exit status) — not run; the diff adds no new service directory, so the family's own trigger did not fire and the script was not invoked. Recorded here only because the shell/CI changes (bootstrap.sh, service-config.sh, derive-service-id.sh) are outside this Go-scoped review's surface and were not separately swept for a script-verified guard.
- Whether `derive-service-id.sh`'s `ATLAS_SERVICE_NS` value and `servicesuniq.atlasServiceNS` (migration.go:36) are exercised end-to-end against a live sparse-environment rollout — confirmed only that the two literals match textually (`c8f90111-a0cf-513e-95e6-c54609e5dec0`); an actual cross-process UUIDv5 derivation match was not executed.

## Summary

### Blocking (must fix)
- FILE-04: `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go:70` — `func Migration(` is not in an `entity.go`; the package has no `entity.go`.

### Non-Blocking (should fix)
- DOM-20: `libs/atlas-env/tenants_test.go:65-100` — three new `Reconcile` scenario tests not table-driven.
- DOM-20: `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` (new `HasService` block) — four sequential-state tests not table-driven.
- DOM-20: `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go` (new `HasService` block) — same four tests duplicated, not table-driven.
- DOM-20: `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration_test.go` — nine new scenario tests not table-driven.
