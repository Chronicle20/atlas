# Sparse-Mode Tenant Provisioning — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-19
---

## 1. Overview

Task-232 replaced deployment-level isolation for ephemeral PR environments
with two logical boundaries. `isolation-audit.md:7-9` states the design
directly:

> Three substrates currently isolate ephemeral environments by *deployment*;
> sparse mode removes all three, leaving `tenant_id` as the data plane's only
> boundary and `environment` as the control plane's.

Sparse mode therefore shares main's Postgres databases, Kafka topics, and
Redis keyspace, and deploys only the services a PR actually changed. Every
other service is main's, reached through the per-service `NS_*` routing table
in `deploy/k8s/base/atlas-ingress.yaml`. The intended result: a PR env costs
three Deployments instead of sixty-four, gets its own tenant, and never reads
or writes another environment's rows.

The first sparse environment (PR #1411) does not behave that way. It shows
the baseline's tenants in the UI, and its non-overridden services do no work
at all. Three defects, each small and each independently sufficient to break
the boundary, are responsible:

1. **Baseline pods do not know their own environment id**, so they do not
   recognise themselves as the owner of any ephemeral environment and
   silently drop its Kafka traffic.
2. **`atlas-pr-bootstrap` adopts the baseline's tenant** instead of minting
   one, so there is no tenant for `tenant_id` to segregate on and the
   already-written teardown sweep has nothing to reclaim.
3. **Browser-origin requests carry no `ENVIRONMENT` header**, so they are
   served by the documented legacy path, which returns the *unfiltered union*
   of every environment's rows.

The teardown half of this design is already complete and correct —
`cleanup.sh`'s sparse-only `sweep-tenant` phase reads the environment
record's `tenant` attribute and reclaims that tenant's rows from the shared
databases via `sweep-orphans.sh --sweep-tenant`. This task closes the
provisioning half so that mechanism has something to act on.

Data isolation is total: a sparse environment's tenant gets its own full
baseline restore (~48k `atlas-data` documents on the PR #461 measurements)
into the shared databases. That cost is accepted deliberately — an ephemeral
environment must never be tested against live gameplay data.

## 2. Goals

Primary goals:

- A sparse PR environment owns exactly one tenant, distinct from the
  baseline's, whose id is recorded on the control-plane environment record.
- Baseline deployments correctly own, and actually process work for, every
  ephemeral environment whose services they are not overriding.
- Environment-scoped reads return the right rows for browser-origin traffic
  in both `main` and sparse environments, with no frontend change.
- The existing `sweep-tenant` teardown phase reclaims the environment's
  tenant-keyed rows from the shared databases, leaving no residue.
- Isolated mode (`overlays/pr`) behaves exactly as it does today.

Non-goals:

- Revisiting task-232 decision D1 (shared Postgres / Kafka / Redis in sparse
  mode). The substrates stay shared.
- Any change to `services/atlas-ui`. The environment tag is applied at the
  ingress; the SPA remains environment-unaware.
- Tightening the legacy unscoped-caller path (`env == ""` sees every row).
  It stays as the FR-1.8 compatibility path; see §9 Q1.
- Reworking `sweep-orphans.sh`, `tenant-tables.txt`, or
  `tools/gen-tenant-tables.sh`.
- Per-environment WZ data deduplication or any optimisation of the restore
  cost. Full duplication is the accepted trade.
- An environment switcher, badge, or any other environment-aware UI affordance.

## 3. User Stories

- As a developer testing a PR, I want the PR environment's UI to list only
  that environment's tenants, so that I never mistake baseline state for my
  change's behaviour.
- As a developer testing a PR, I want a service I did *not* change to be
  served by main's deployment and still process my environment's events, so
  that a sparse environment is a working game server and not a partial one.
- As a developer testing a PR, I want my environment's gameplay data to be a
  fresh restore under my own tenant, so that I can destroy or corrupt it
  freely without touching live data.
- As an operator, I want teardown to remove every row a sparse environment
  wrote to the shared databases, so that repeated PR environments do not
  accumulate orphaned tenant data in main's Postgres.
- As an operator viewing main's UI, I want to see only main's tenants, so
  that active PR environments do not pollute the baseline's control-plane
  views.

## 4. Functional Requirements

### 4.1 Environment self-identification (prerequisite)

**FR-1.1** Every deployment in `deploy/k8s/overlays/main` MUST read its own
environment id from `ATLAS_ENVIRONMENT` (`env.SelfVar`,
`libs/atlas-env/env.go:29`), set to the baseline environment's name.

**FR-1.2** The value MUST equal the `BASELINE_ENVIRONMENT` that
`.github/workflows/pr-validation.yml:921` derives — the baseline overlay's
`namespace` with the `atlas-` prefix stripped, i.e. `main` for the
`atlas-main` namespace. A literal that disagrees with that derivation is a
defect: `MapRegistry.IsOwner` compares `rec.Baseline == r.self` by string
equality (`libs/atlas-env/registry.go:218`), so any mismatch makes the
baseline own nothing.

**FR-1.3** With FR-1.1 satisfied and an ACTIVE environment record present, a
baseline pod MUST report `IsOwner("pr-<N>", svc) == true` for every service
`svc` absent from that record's `overrides` map, and `false` for every
service present in it.

**FR-1.4** A baseline pod's Kafka consumer MUST process (not
`gateSkipNotOwner`) messages tagged with an ephemeral environment it owns
per FR-1.3, and MUST continue to skip those it does not.

**FR-1.5** Isolated-mode (`overlays/pr`) and local-compose behaviour MUST be
unchanged. `overlays/pr` environments register no control-plane record; with
`r.records` empty, `EnvironmentsOwnedBy` still returns `[""]`
(`registry.go:227`) and the legacy path is preserved.

### 4.2 Per-environment tenant provisioning

**FR-2.1** In sparse mode, `bootstrap.sh`'s `tenant-create` step MUST resolve
"does this environment already have a tenant?" scoped to
`$ATLAS_ENVIRONMENT`, not globally. The existing check
(`bootstrap.sh:210-218`) matches on `(region, majorVersion, minorVersion)`
against an unscoped listing, which against a shared database always matches
the baseline's canonical tenant.

**FR-2.2** The scoped lookup MUST be performed by sending
`ENVIRONMENT: $ATLAS_ENVIRONMENT` on `GET /api/tenants`. `atlas-tenants`
already filters that listing to the caller's environment via `scope.Strict`
(`tenant/provider.go`, `getAll`), so no service change is required. This is
the same pattern PR #1418 applied to the service-config writes.

**FR-2.3** When the scoped lookup returns no tenant, `bootstrap.sh` MUST
`POST /api/tenants` with the `ENVIRONMENT` header set, minting a tenant whose
server-owned `Environment` column is this environment. The canonical
`(region, majorVersion, minorVersion)` payload is unchanged — a sparse tenant
deliberately shares the baseline's version triple and is distinguished only
by its environment and its generated UUID.

**FR-2.4** The step MUST remain idempotent across Job re-runs: a second
invocation MUST find the environment's own tenant via FR-2.2 and reuse its
id, never minting a second.

**FR-2.5** In isolated mode the step MUST behave exactly as today. Gate on
the same `ATLAS_MODE` fact `cleanup.sh`'s phases already gate on, or send no
`ENVIRONMENT` header there — either is acceptable provided isolated
bootstrap's observable behaviour is byte-identical.

**FR-2.6** Every subsequent bootstrap step — tenant-config clone, service
config upsert, baseline restore, readiness probes — MUST use the tenant id
resolved by FR-2.1–2.4.

### 4.3 Recording the tenant on the environment record

**FR-3.1** After the tenant id is resolved, `bootstrap.sh` MUST set it on the
control-plane environment record via
`PATCH /api/configurations/environments/{name}`
(`environments/resource.go:29`).

**FR-3.2** The PATCH MUST target the baseline's `atlas-ingress`, for the same
reason `environment-record.yaml` does: `atlas-configurations` is never in the
override set.

**FR-3.3** `overlays/pr-sparse/environment-record.yaml` continues to POST
`"tenant": ""` at sync-wave 1. It runs before any tenant exists and MUST NOT
be made responsible for minting one.

**FR-3.4** The PATCH MUST be idempotent — re-running bootstrap against a
record whose `tenant` already holds the same id is a no-op, not an error.

**FR-3.5** If the PATCH fails, bootstrap MUST fail loudly rather than
continue. An environment that provisions a tenant but never records it
produces exactly the residue this task exists to prevent, and does so
silently.

### 4.4 Environment tagging for browser-origin traffic

**FR-4.1** `atlas-ingress` MUST apply a default environment to any inbound
request that does not already carry an `ENVIRONMENT` header, rather than
forwarding the empty value. Today `deploy/k8s/base/atlas-ingress.yaml` does
`proxy_set_header ENVIRONMENT $http_environment` — unconditional passthrough.

**FR-4.2** The default MUST be per-overlay: the sparse overlay defaults to
`pr-<N>`, the main overlay to its own baseline id (FR-1.2). The base MUST
retain today's passthrough-with-empty-default so no unrelated overlay changes
behaviour implicitly.

**FR-4.3** An `ENVIRONMENT` header supplied by the caller MUST win over the
default. In-cluster service-to-service calls already set it via
`EnvHeaderDecorator` (`libs/atlas-rest/requests/header.go:46`) and must not be
overwritten.

**FR-4.4** The default MUST name an environment that
`env.CurrentRegistry().IsProvisionable` accepts. `ParseEnvironment`
(`libs/atlas-rest/server/handler.go:68`) returns 400 for an unknown or
inactive id, so an ingress stamping an unregistered environment would
hard-fail every request. For the main overlay this makes an ACTIVE
control-plane record for the baseline a deployment prerequisite; for sparse it
is already guaranteed by `environment-record.yaml` at sync-wave 1.

**FR-4.5** With FR-4.1–4.4 in force, `GET /api/tenants` from a browser MUST
return exactly the calling environment's tenants: one for a sparse
environment, the baseline's set for main, and never the union.

### 4.5 Teardown

**FR-5.1** `cleanup.sh`'s `sweep-tenant` phase MUST find a non-empty `tenant`
attribute on the environment record and reclaim that tenant's rows. Its
current `"has no tenant attribute; cannot reclaim tenant-keyed rows"` warning
path (`cleanup.sh:356`) MUST NOT be reached for an environment that
completed bootstrap.

**FR-5.2** No change to `sweep-orphans.sh`, `tenant-tables.txt`, or the
phase's failure semantics is in scope. Sweep failure remains non-fatal to
teardown (task-232 design §7.5).

**FR-5.3** After teardown, zero rows keyed to the environment's tenant may
remain in the shared databases, and the environment's tenant row itself MUST
be gone (already handled by `do_drop_control_plane`'s `atlas-tenants`
reclaim, `cleanup.sh:306`).

## 5. API Surface

No new endpoints and no changed request or response shapes. This task changes
which *headers* existing callers send and which *rows* existing
environment-scoped queries therefore return.

| Endpoint | Change | Notes |
|---|---|---|
| `GET /api/tenants` | Caller now sends `ENVIRONMENT` | Server-side filtering via `scope.Strict` already exists; behaviour changes only because the header arrives |
| `POST /api/tenants` | Caller now sends `ENVIRONMENT` | `Entity.Environment` is server-owned from context — the header is the only way to stamp it |
| `PATCH /api/configurations/environments/{name}` | New caller (`bootstrap.sh`) | Route exists (`environments/resource.go:29`); sets `attributes.tenant` |

Error cases to honour:

- `ParseEnvironment` returns **400** for an unknown, DEACTIVATING, or DELETED
  environment id. Bootstrap runs while its environment is PROVISIONING, which
  is admitted (see the `ParseEnvironment` doc comment, relaxing FR-3.6 of
  task-232).
- `scope.AuthorizeWrite` returns `ErrCrossEnvironmentWrite` for a write
  targeting another environment's row. Bootstrap must never trip this; if it
  does, that is a defect in the header wiring, not a condition to retry
  around.

## 6. Data Model

No schema changes. Every column this task depends on already exists:

| Table | Column | Source | Current state |
|---|---|---|---|
| `tenants` | `Environment string` (`not null;default:''`) | `services/atlas-tenants/.../tenant/entity.go:16` | Backfilled to the baseline id by `BackfillEnvironment`; server-owned from request context on create |
| `environments` | `Tenant string` (`not null;default:''`) | `services/atlas-configurations/.../environments/entity.go:22` | Always `""` — written by nobody, read by `sweep-tenant` |

`environments` deliberately carries no `Environment` column — it *is* the
environment list, declared to `tools/scopeguard` by its `ScopingDimension()`
marker.

Migration notes: none required. The `Tenant` column already defaults to `''`,
so existing records need no backfill; they are ephemeral by construction and
any environment predating this change is torn down by PR closure.

Data volume: each sparse environment adds a full per-tenant `atlas-data`
restore to the shared databases (~48k documents; the PR #461 cold-start
measured 11,163 documents lost as a 23% deficit against a full restore, so
the full set is on the order of 48k). All `atlas-data` entities key on
`TenantId` (`data/*/entity.go`), so the rows are reclaimable by the existing
sweep. This is accepted, not mitigated.

## 7. Service Impact

| Component | Change |
|---|---|
| `deploy/k8s/overlays/main` | Add `ATLAS_ENVIRONMENT=main` to the `atlas-env` ConfigMap (FR-1.1). Add the ingress environment default (FR-4.2). |
| `deploy/k8s/base/atlas-ingress.yaml` | Replace the unconditional `proxy_set_header ENVIRONMENT $http_environment` with a default-able form (nginx `map`), defaulting to empty in the base so no overlay changes implicitly (FR-4.1, FR-4.2). |
| `deploy/k8s/overlays/pr-sparse` | Ingress default of `pr-<N>` (FR-4.2). **The default MUST NOT go in `patches/ingress-host.yaml`**: that file is in `tools/pr-sparse-mirror-guard.sh`'s `MIRRORS` array (line 38), so it is byte-diffed against `overlays/pr`'s copy and any edit must apply identically to both. Isolated environments register no control-plane record, so stamping them with an environment id would 400 every request under FR-4.4. The sparse default needs a new, non-mirrored patch file. |
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Environment-scoped tenant lookup and create (FR-2). Environment-record PATCH (FR-3). |
| `services/atlas-pr-bootstrap/test/` | New/extended bats coverage for the scoped lookup, the mint path, idempotency, and the PATCH. Existing suites: `cleanup_test.bats`, `service_config_test.bats`, `sweep_test.bats`, `reclaim_test.bats`. |
| `docs/runbooks/ephemeral-pr-deployments.md` | Document that a sparse environment owns its own tenant, what the restore costs, and how to confirm the boundary held. |

Explicitly **not** changed: `services/atlas-ui`, `services/atlas-tenants`,
`services/atlas-configurations`, `sweep-orphans.sh`, `overlays/pr`.

## 8. Non-Functional Requirements

**Multi-tenancy.** This task's entire purpose is the `tenant_id` /
`environment` boundary of task-232's two-plane model. Control-plane rows are
scoped by `environment`; data-plane rows by `tenant_id`. Neither dimension may
be added to the other plane's tables (`isolation-audit.md:27-30`).

**Security / isolation.** The failure mode being closed is cross-environment
read exposure, not merely a cosmetic UI defect: `tenant/provider.go` documents
that a legacy caller with no environment on context "sees every tenant,
unfiltered". Once sparse environments own tenants, main's untagged UI would
list them. FR-4 must land with FR-2, not after it.

**Observability.** The Kafka gate already exports
`atlas_kafka_gate_skipped_not_owner_total{service,environment}` and
`atlas_kafka_gate_dropped_unresolvable_total`. FR-1 is verifiable against
them: before the fix, a baseline pod's `skipped_not_owner` counter for
`environment="pr-<N>"` climbs; after, it stays flat while work is processed.
The ingress access log already emits `env=$http_environment`
(`atlas-ingress.yaml`, `log_format main_env`), which after FR-4 records the
resolved environment rather than a blank.

**Performance.** No hot-path change. The nginx `map` is a compile-time hash
lookup per request. Bootstrap gains one GET and one PATCH.

**Failure modes.** FR-4.4 is the sharp edge: an ingress that stamps an
environment the registry does not know turns every browser request into a
400. The main overlay's default therefore depends on an ACTIVE baseline
environment record existing, which is also what CI's baseline resolution
assumes.

**Backwards compatibility.** Isolated mode, local compose, and every
in-cluster caller that already sets its own header are unaffected. The legacy
empty-environment path remains valid and is not removed.

## 9. Open Questions

**Q1 — Should the legacy unscoped caller be tightened later?** Resolved for
*this* task: no. FR-4 guarantees no real browser request arrives untagged, and
making an untagged read fail closed risks in-cluster callers that depend on
the current behaviour. A follow-up task should audit untagged callers and
decide whether `env == ""` should resolve to `env.Self()` instead of "see
everything". Recorded here so the decision is not lost.

**Q2 — Does the baseline environment record already exist and is it ACTIVE?**
FR-4.4 makes main's ingress default depend on it. CI's baseline resolution
derives the *name* from the overlay's namespace without consulting the
registry, so the record's existence is assumed rather than verified anywhere
in the current pipeline. Design phase must confirm against the live cluster
and, if absent, add its creation as an explicit step.

**Q3 — Ordering between FR-1 and FR-4 at rollout.** Setting
`ATLAS_ENVIRONMENT=main` changes what main's pods own, and stamping
`ENVIRONMENT: main` changes what they are asked for. Whether these can land
in one deploy or need a sequenced rollout is a design-phase question.

**Q4 — Does the per-tenant restore fit the environment's provisioning
budget?** The restore is already tenant-keyed and already runs, but it has
never run into a *shared* database alongside the baseline's copy. Concurrent
sparse environments multiply the row count in main's Postgres. Out of scope to
optimise; worth measuring during acceptance so the ceiling on simultaneous
sparse environments is a known number rather than a surprise.

## 10. Acceptance Criteria

Verification is a live sparse-environment round-trip. The defect class this
task addresses has escaped four times on PR #1411 (#1416, #1418, #1421, and
this one); script-level coverage alone has not been sufficient.

**Automated**

- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] Bats coverage proves the scoped tenant lookup finds no baseline tenant
      when the environment has none, mints exactly one, and returns the same
      id on a second run.
- [ ] Bats coverage proves the environment-record PATCH is issued with the
      minted id and is idempotent.
- [ ] Bats coverage proves isolated-mode bootstrap's tenant step is
      unchanged.
- [ ] A rendered-manifest assertion proves `overlays/main` sets
      `ATLAS_ENVIRONMENT=main` and that `overlays/pr` does not gain it.
- [ ] `tools/pr-sparse-mirror-guard.sh` passes.

**Live round-trip on a sparse PR environment**

- [ ] The environment's control-plane record carries a non-empty `tenant`
      whose value is *not* the baseline's tenant id.
- [ ] `GET /api/tenants` against `<N>.atlas.home` returns exactly one tenant —
      the environment's own.
- [ ] `GET /api/tenants` against `dev.atlas.home` returns only main's tenants
      and does not list the PR's.
- [ ] `atlas-data` document count under the PR tenant reaches the full
      baseline-restore total, with the baseline's own count unchanged.
- [ ] A non-overridden service deployed only in main (e.g. `atlas-ban`)
      demonstrably processes an operation issued in the PR environment;
      `atlas_kafka_gate_skipped_not_owner_total{environment="pr-<N>"}` stays
      flat for it.
- [ ] A game client connects through the PR environment's login LB IP and
      reaches character select under the PR tenant.
- [ ] After PR closure, `sweep-tenant` reports a successful reclaim (no
      "cannot reclaim tenant-keyed rows" warning), and a direct query
      confirms zero rows remain for that tenant across the tables in
      `tenant-tables.txt`.
- [ ] Main's gameplay is unaffected throughout: its tenant, its data counts,
      and its own environment's traffic are untouched before, during, and
      after the PR environment's lifetime.
