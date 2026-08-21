# Sparse Environment Service Binding and Activation — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-20
---

## 1. Overview

Task-232 introduced sparse ephemeral environments: a PR deploys only the
services it changed (the *override set*), shares the baseline's Postgres,
Kafka and Redis, and routes everything else to the baseline through the
per-service `NS_*` table. PR #1411 is the first such environment. After four
successive fixes (#1416, #1418, #1421, #1427, #1432) its pods finally run
clean — and a client still cannot complete a login.

The proximate cause is not in the login flow at all. An override Deployment
has no durable way to point at its own service-configuration row. The row's id
does not exist until a PostSync hook mints it, so `deploy/k8s/base/atlas-login.yaml:45`
bakes the *baseline's* pinned id and `bootstrap.sh` compensates with an
out-of-band `kubectl set env SERVICE_ID=…`. The Argo CD Application has
`selfHeal: true`, so that patch is reverted on the next reconcile. The
override therefore loads the baseline's service row, binds the baseline's
tenants to its own sockets, and every message it emits carries its own
`ENVIRONMENT` header alongside a tenant the baseline owns — which the FR-7.7
reconciliation correctly rejects.

Two further defects sit behind that one and would surface the moment it is
fixed: a legacy tenant projects as environment `""` and is treated as a hard
mismatch rather than as legacy, and nothing in the repository ever promotes a
sparse environment out of `PROVISIONING`, which the ownership gate requires
before it will process any of the environment's traffic. This task closes all
three, in that order, so that a sparse environment can complete a login
end-to-end without manual intervention.

Full diagnostic evidence, including the live-cluster observations that
produced this scope, is in
`diagnosis.md` (this folder).
That document also records two corrections to earlier hypotheses — the
service-status projection *is* scoped (by service id), and the control-plane
delete path *does* emit tombstones — which are why this PRD is narrower than
the initial triage suggested.

## 2. Goals

Primary goals:

- An override Deployment resolves and keeps a reference to the service-config
  row belonging to its own environment, surviving Argo CD reconciliation
  indefinitely.
- Repeated bootstrap runs against the same environment converge on one
  service-config row per service type, not one per run.
- A message naming a legacy (environment-less) tenant is admitted rather than
  dropped, consistent with FR-1.8's treatment of the empty environment
  everywhere else in the codebase.
- A sparse environment transitions `PROVISIONING → ACTIVE` automatically, on a
  condition that is observed rather than assumed, so its traffic is processed
  by the correct owner.
- A client can create an account and log in against a sparse ephemeral
  environment with no manual database, Kubernetes, or Argo intervention.

Non-goals:

- Adding an `ENVIRONMENT` header to `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`.
  The projection already filters by service id (`subscriber.go:112`), which is
  a strictly stronger key; the header would be defence-in-depth only and is
  explicitly deferred.
- Changing the `NS_*` REST routing table, the ingress header defaulting, or
  tenant provisioning — task-242 owns those.
- Making sparse the default PR mode (task-232 Task 54).
- Fixing that a ConfigMap change does not roll the Deployments consuming it via
  `envFrom`. This is a real and dangerous defect — #1427 added
  `ATLAS_ENVIRONMENT`, which the entire ownership gate depends on, and it
  reached zero running pods — but it is a repo-wide deploy-mechanics change
  unrelated to this task's seam. It must be filed separately.
- Retroactively repairing PR #1411's live state. That environment has been
  hand-patched during diagnosis and should be torn down and recreated to
  validate this work.

## 3. User Stories

- As a developer opening a PR, I want the ephemeral environment to serve *my*
  tenant on its own sockets, so that what I test is my change and not the
  baseline's data.
- As a developer, I want to log into my PR environment and reach character
  select, so that a sparse environment is actually usable for the manual
  testing it exists to support.
- As an operator, I want an environment to advertise `ACTIVE` only once it can
  genuinely serve traffic, so that "the environment is up" is a claim I can
  trust.
- As an operator, I want repeated syncs of the same PR to leave the control
  plane in the same shape, so that a long-lived PR does not accumulate
  unbounded rows in the shared baseline database.
- As a maintainer of a legacy single-environment deployment, I want none of
  this to change my behaviour, so that the environment feature stays opt-in.

## 4. Functional Requirements

### 4.1 Durable service-config binding

- **FR-1.1** An override Deployment MUST resolve the service-config row for
  its own environment. It MUST NOT load the baseline's row.
- **FR-1.2** The binding MUST survive Argo CD reconciliation with
  `syncPolicy.automated.selfHeal: true`. Any mechanism that writes to a
  managed Deployment's spec out of band does not satisfy this requirement.
  (Verified: `atlas-pr-1411`'s Application has
  `{"automated":{"prune":true,"selfHeal":true},"syncOptions":["ServerSideApply=true",…]}`.)
- **FR-1.3** The binding MUST be establishable without a prior API call, so
  that it can be rendered into the manifest at CI time rather than patched
  after the fact.
- **FR-1.4** In isolated mode and in the baseline overlay, the resolved id MUST
  be byte-identical to today's pinned canonical id. No behaviour change outside
  sparse mode.
- **FR-1.5** When an override's service-config row is absent at pod start, the
  service MUST fail loudly (non-Ready, logged) rather than silently falling
  back to the baseline's row or to an empty config. Silent fallback is what
  made this defect invisible for five deploys.

### 4.2 Idempotent service-config provisioning

- **FR-2.1** Running bootstrap N times against one environment MUST leave
  exactly one service-config row per service type for that environment.
  (Observed today: five `login-service` rows and five `channel-service` rows
  for `pr-1411`, one per bootstrap run.)
- **FR-2.2** The row's id MUST be stable across runs for a given
  (service type, environment) pair.
- **FR-2.3** Re-running bootstrap MUST NOT mutate any row belonging to another
  environment, and MUST NOT read or merge into the baseline's canonical row
  (task-232 G7/NG6).
- **FR-2.4** Rows orphaned by earlier non-idempotent runs MUST be reclaimable —
  either by `cleanup.sh`'s environment-scoped sweep or by an explicit one-time
  reconciliation. Removal MUST go through the API path that emits a tombstone
  (`services.ProcessorImpl.DeleteById`), never raw SQL, or consumers will keep
  replaying the row from the compacted topic.

### 4.3 Legacy tenant reconciliation

- **FR-3.1** When a tenant's projected environment is empty, `Reconcile` MUST
  treat it as legacy and trust the message's `ENVIRONMENT` header, rather than
  reporting a mismatch. Today `MapRegistry.ApplyTenant` (`registry.go:67`)
  stores unconditionally, so a pre-#1427 tenant is *known* with environment
  `""`, and `Reconcile` (`tenants.go:70`) then treats `"pr-1411" != ""` as a
  hard mismatch and the gate drops the message.
- **FR-3.2** A genuine disagreement — both environments non-empty and
  different — MUST still be reported as a mismatch and dropped (FR-7.7 is
  preserved, not weakened).
- **FR-3.3** The drop counter and log line MUST distinguish the three
  `gateDropUnresolvable` arms (mismatched / stale / not-active). They are
  currently indistinguishable, which cost several hours of this
  investigation: the same message text is emitted for all three.

### 4.4 Environment activation

- **FR-4.1** A sparse environment MUST transition `PROVISIONING → ACTIVE`
  automatically once it can serve traffic. Today nothing in the repository
  performs this transition — the only producer of `ACTIVE` outside tests and
  docs is `deploy/k8s/overlays/main/environment-record.yaml:82`, the baseline
  creating itself.
- **FR-4.2** The transition MUST be gated on observed conditions, per PRD
  FR-5.3: every override Deployment ready, every override consumer group
  initialized, and the mandatory socket services bound.
- **FR-4.3** "Consumer group initialized" MUST be an observed signal, not an
  inference. This requires task-232 Task 45 (offset seeding at sync-wave 0),
  which is **unimplemented**: every step is unchecked and
  `deploy/k8s/base/atlas-kafka-precreate.yaml` contains zero occurrences of
  `seed_group`, although the test file `atlas-kafka-precreate_test.sh` exists.
- **FR-4.4** Activating without FR-4.3 MUST NOT be shipped: an override group
  created on a long-lived shared topic would otherwise take the library's
  default start offset (`libs/atlas-kafka/consumer/config.go:37`), replaying
  the whole retention window or losing everything produced before first poll.
- **FR-4.5** Teardown ordering MUST be preserved: `DEACTIVATING` before any
  override workload is destroyed (FR-5.5), which `cleanup.sh` already
  implements.
- **FR-4.6** A deployment with no environment records projected MUST continue
  to operate exactly as today. This is a hard constraint on any change to the
  gate: with records absent, `IsActive(msgEnv)` returns false for a non-empty
  `msgEnv`, so a naive tightening would make a legacy deployment stop
  processing its own traffic.

## 5. API Surface

No new endpoints are required. Existing surfaces that participate:

| Endpoint | Method | Role in this task |
|---|---|---|
| `/api/configurations/services` | POST | Creates an environment's service row. Must become idempotent on a stable id (FR-2.1, FR-2.2). |
| `/api/configurations/services/{id}` | GET, PATCH, DELETE | Upsert and tombstone-emitting delete. `DeleteById` already enqueues a nil-value tombstone (`processor.go:228`). |
| `/api/configurations/environments/{name}` | PATCH | The activation transition (FR-4.1). `UpdateByName` already validates the `PROVISIONING → ACTIVE → DEACTIVATING → DELETED` chain and rejects skips and reverts. |
| `/api/configurations/environments` | GET | Readiness/condition evaluation for the activation gate. |

The `ENVIRONMENT` request header remains server-owned for the environment
column on writes: `administrator.go`'s INSERT takes `env.MustFromContext(ctx)`
and the request body's `attributes.environment` is deliberately ignored. Any
new provisioning path must send the header, exactly as `create_service_config`
does today.

## 6. Data Model

No new entities. Relevant existing shape:

```
services
  id           uuid    PK, default uuid_generate_v4()
  type         varchar not null
  data         json    not null
  environment  text    not null default ''
```

Notes and constraints:

- `services` has **no uniqueness constraint** on `(type, environment)`. That
  absence is what permits FR-2.1's violation. Adding one is the obvious
  enforcement point, but it interacts with the baseline's existing rows —
  `main` currently holds 19 rows across several types, and a co-resident
  multi-tenant `login-service` row is legitimate. The design phase must decide
  whether the constraint is `(type, environment)` or whether idempotency is
  enforced purely by deterministic id.
- A **deterministic id** derived from (canonical id, environment) — a UUIDv5,
  for instance — satisfies FR-1.3, FR-2.2 and FR-1.2 simultaneously: it is
  computable at CI time with no API call, so the sparse overlay can render
  `SERVICE_ID` statically and Argo has nothing to revert; and bootstrap's POST
  becomes a plain upsert. This is the leading candidate, not a decision —
  see Open Questions.
- Today's id is `new_uuid()`, freshly random per run
  (`service-config.sh:117-127`), which is precisely why it cannot be rendered
  into a manifest.
- Migration: existing sparse rows carry random ids. Any move to deterministic
  ids needs a reconciliation pass for in-flight environments, or acceptance
  that existing sparse environments must be recreated. Ephemeral environments
  are cheap to recreate, which argues for the latter.

## 7. Service Impact

| Service / component | Change |
|---|---|
| `services/atlas-pr-bootstrap` | `service-config.sh` id generation becomes deterministic; `bootstrap.sh`'s `create_service_config` becomes an upsert and drops the `kubectl set env SERVICE_ID` patch; the activation step is added after readiness. |
| `deploy/k8s/overlays/pr-sparse` | Renders `SERVICE_ID` per override service at CI time; gains whatever job or hook performs activation. |
| `deploy/k8s/base/atlas-kafka-precreate.yaml` | Gains `seed_group` and the readiness observation (task-232 Task 45, FR-4.3). |
| `libs/atlas-env` | `Reconcile` / `ApplyTenant` legacy-empty handling (FR-3.1, FR-3.2). |
| `libs/atlas-kafka/consumer` | Gate drop-reason distinguishability (FR-3.3). No change to the decision logic itself. |
| `services/atlas-login`, `atlas-channel`, `atlas-character-factory`, `atlas-world` | Possibly FR-1.5's fail-loud-on-missing-row behaviour. The projections themselves are correct and should not otherwise change. |
| `.github/workflows/pr-validation.yml` | If `SERVICE_ID` is rendered at CI time, the `update-pr-overlay` job computes it. |

`atlas-configurations` needs no change unless the design chooses a uniqueness
constraint or a server-side deterministic id.

## 8. Non-Functional Requirements

- **Backward compatibility.** A deployment with no environment records must be
  byte-identical in behaviour (FR-4.6, FR-1.4). This is the single largest
  risk in the task: the ownership gate fails closed, so any tightening can
  silently disable a legacy deployment.
- **Observability.** The three `gateDropUnresolvable` arms must be
  distinguishable in logs and metrics (FR-3.3). Separately,
  `gateSkipNotOwner` is currently *silent* — no log line at all, only
  `atlas_kafka_gate_skipped_not_owner_total` — which means a message nobody
  processes leaves no trace. At minimum this should be debug-logged.
- **Idempotency.** Every provisioning step runs on every Argo sync
  (`Force=true,Replace=true` on the hook Jobs), so each must be safely
  repeatable.
- **Failure visibility.** Provisioning failures must fail the hook Job rather
  than leave a half-configured environment that looks healthy. The current
  `kubectl set env … || log warn` swallows exactly the failure that matters.
- **Multi-tenancy.** Unchanged: `tenant_id` remains the data plane's boundary
  and `environment` the control plane's. Nothing here may let one
  environment's bootstrap write another's rows (FR-2.3).
- **Security.** No change to authn/authz. Bootstrap's Role already carries
  `get/patch/update` on deployments; if the `kubectl set env` patch is removed,
  that grant should be re-examined and narrowed if it becomes unnecessary.

## 9. Open Questions

1. **Deterministic id derivation.** UUIDv5 over (canonical id, environment
   name) is the leading candidate. Where is it computed — the CI overlay job,
   bootstrap, or `atlas-configurations` server-side? All three must agree, so
   the derivation belongs in exactly one place with the others deriving from
   it. Which?
2. **Uniqueness enforcement.** Is `(type, environment)` unique in the
   `services` table, or is idempotency carried solely by the deterministic id?
   A constraint is stronger but must not reject the baseline's legitimate
   existing rows.
3. **Who performs activation.** A PostSync Job in `overlays/pr-sparse` is the
   obvious home, but it must observe consumer-group readiness (FR-4.3) that
   sync-wave-0 produces. Does the activation Job poll the environment's own
   readiness, or does the precreate Job publish a signal it consumes?
4. **FR-5.4 activation atomicity.** Registry caches update independently per
   pod off the environment-status topic, so there is an observable window
   around the flip where ownership is ambiguous. Task-232 PRD open question 3
   asked whether a two-phase handover is needed and never answered it. Is a
   window acceptable for ephemeral environments?
5. **Scope of Task 45.** FR-4.3 pulls in an entire unimplemented task-232 task.
   Is it absorbed here, or executed as a prerequisite with its own review?
6. **Existing sparse environments.** Recreate, or reconcile in place? Recreation
   is far simpler and ephemeral environments are cheap, but it means any
   open PR environment is disrupted when this lands.
7. **FR-1.5 fail-loud.** Should a missing service-config row block pod
   readiness, or should the service start and report unhealthy? The former is
   safer; the latter is easier to diagnose.

## 10. Acceptance Criteria

Functional:

- [ ] A freshly created sparse PR environment binds each override service to
      the service-config row whose `environment` equals its own, verified by
      reading `SERVICE_ID` from the running Deployment and matching it against
      the `services` table.
- [ ] After an Argo CD hard refresh and re-sync, `SERVICE_ID` is unchanged.
      (Today `selfHeal` reverts it.)
- [ ] The override's login socket binds only its own environment's tenant.
      Concretely: no port bound for a tenant whose `environment` differs from
      the pod's.
- [ ] Running bootstrap three times against one environment leaves exactly one
      service-config row per service type for that environment.
- [ ] A message carrying `ENVIRONMENT: <env>` and a legacy (environment-less)
      tenant is processed, not dropped.
- [ ] A message carrying `ENVIRONMENT: <env>` and a tenant owned by a
      *different, non-empty* environment is still dropped as a mismatch.
- [ ] A sparse environment reaches `phase: ACTIVE` with no manual PATCH.
- [ ] The environment does not reach `ACTIVE` while any override Deployment is
      unready or any override consumer group lacks committed offsets.
- [ ] **End-to-end: a client connects to a freshly created sparse PR
      environment, creates an account, logs in, and reaches character select —
      with no manual database, `kubectl`, or Argo intervention at any point.**

Regression / compatibility:

- [ ] A deployment with no environment records projected behaves identically
      to before this change, demonstrated by test, not by inspection.
- [ ] Isolated-mode PR environments are unaffected: rendered `SERVICE_ID`
      values are byte-identical to today's.
- [ ] The baseline overlay's rendered manifests are unchanged except where
      explicitly intended.
- [ ] Teardown still deactivates before destroying, and reclaims every
      service-config row the environment created.

Observability:

- [ ] The three `gateDropUnresolvable` arms are distinguishable in logs and in
      metric labels.
- [ ] A provisioning failure fails the hook Job rather than logging a warning
      and continuing.

Gate:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
