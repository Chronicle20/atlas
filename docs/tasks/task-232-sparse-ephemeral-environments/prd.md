# Sparse Ephemeral Environments — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-15

---

## 1. Overview

Atlas deploys 64 Kubernetes Deployments per environment. The per-PR ephemeral
environment introduced by task-063 duplicates that entire runtime into a
namespace of its own (`atlas-pr-<N>`) even when the pull request changes one
service. The cost is paid in compute, memory, cluster workload count, and — most
visibly — startup latency, since every PR waits for the full stack to become
ready before any test traffic can flow.

This project introduces a **sparse** ephemeral environment: a PR deploys only
the services it changed, and every other service is served by the `main`
baseline deployment. A request entering `pr-123` may execute through
`atlas-character-pr-123`, then `atlas-monsters-main`, then
`atlas-channel-pr-123`, and must remain logically inside `pr-123` for its entire
lifecycle — across REST hops, Kafka messages, saga continuations, scheduled
callbacks, and autonomous background loops. The existing fully-isolated mode
remains available and becomes the documented escalation path.

The central engineering claim of this document is that **execution environment
is a property of the operation, not of the deployment processing it** — exactly
as tenant already is in Atlas. The central engineering *risk* is that Atlas's
current ephemeral isolation is not built that way: it is process-scoped, fixed
at pod start, in three independent substrates. Section 3 records what was
verified in the codebase; Section 4 onward proposes the changes. Findings and
proposals are kept strictly separate, because the gap between them is the
project.

---

## 2. Goals

### Primary goals

- **G1.** A PR deploys only its changed services; unchanged services are served
  by the `main` baseline. Resource consumption scales with the number of changed
  services, not with 64.
- **G2.** Execution environment propagates across every asynchronous and
  synchronous boundary Atlas has: REST, Kafka, saga steps, scheduled work, and
  autonomous loops.
- **G3.** A shared baseline deployment performs autonomous work independently
  for each environment it is the effective implementation for, and for no
  environment it is not.
- **G4.** No operation ever silently transitions from an ephemeral environment
  to `main`. Cross-environment leakage is a correctness bug, not a degradation.
- **G5.** All 64 Deployments are migrated to the environment-aware model in this
  task (see §12 for the phasing that makes this tractable).
- **G6.** The existing fully-isolated mode continues to work unchanged, with
  documented rules for when it is required.

### Non-goals

- **NG1.** A service mesh. Routing is solved with the existing nginx ingress.
- **NG2.** Per-environment Kafka topic duplication *for sparse environments*.
  (Note: this is a **reversal** of current behavior — see F4.)
- **NG3.** A Kafka routing or proxy service.
- **NG4.** Synchronous registry lookups on the REST or Kafka hot path.
- **NG5.** Environment-conditional logic inside domain processors.
- **NG6.** Changing `main`'s observable runtime behavior. Every change must be a
  no-op for a single-environment deployment.
- **NG7.** Replacing the tenant model with a parallel isolation mechanism.

---

## 3. Findings — verified current state

Everything in this section was read from the repository at branch point
`c8d44127c`. File and line references are given so design can re-verify. Nothing
here is inferred.

### 3.1 Scale

| Measure | Value | Source |
|---|---|---|
| Directories under `services/` | 66 | `ls -d services/*/` |
| `atlas-*.yaml` manifests in `deploy/k8s/base` | 68 | — |
| `kind: Deployment` in base | **64** | `grep -h "^kind: Deployment"` |
| Go files constructing a `time.NewTicker` | 18 | `grep -rl NewTicker services` |

The remaining base manifests are Jobs and templates
(`atlas-kafka-precreate`, `atlas-minio-init`, `atlas-minio-reconcile`,
`atlas-data-ingest-job-template`). The PRD input's "~60 services" is close; 64
Deployments is the number that matters for capacity claims.

### 3.2 The three isolation substrates are process-scoped — F1–F5

This is the load-bearing finding. Ephemeral isolation today is achieved by
giving each PR namespace its own *copies* of the underlying resources, selected
by a value fixed for the lifetime of the pod:

| # | Substrate | Mechanism | Source |
|---|---|---|---|
| **F3** | Postgres | `DB_NAME` patched to `<db>-<ATLAS_ENV>` per Deployment | `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh` |
| **F4** | Kafka topics | every `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` suffixed `-<ATLAS_ENV>` | `overlays/pr/scripts/gen-topic-config.sh`; `overlays/pr/kustomization.yaml:154-` |
| **F5** | Redis | key prefix `<ATLAS_ENV>:atlas`, computed from `os.Getenv("ATLAS_ENV")` in a **package-level `var` initializer** | `libs/atlas-redis/keys.go:15-24` |

- **F1.** PR environments are namespace-per-PR (`atlas-pr-<N>`), built by
  applying the *entire* `deploy/k8s/base` kustomization plus a PR overlay.
  `overlays/pr/kustomization.yaml` lists `../../base` as its first resource.
- **F2.** Placeholders (`PLACEHOLDER_ATLAS_ENV`, `PLACEHOLDER_PR_NUMBER`,
  `PLACEHOLDER_FULL_SHA`) are `sed`-substituted at CI time by
  `.github/workflows/pr-validation.yml` and force-pushed to a derived branch
  `bot/pr-<N>-resolved`, which the Argo CD ApplicationSet targets.

**Consequence.** A shared `atlas-monsters-main` pod holds exactly one DSN, one
topic set, and one Redis key prefix. It is structurally incapable of serving
`pr-123` today. This is precisely the class of resource the input's §14 asks us
to identify: *isolation currently based on deployment identity rather than
tenant identity.* All three are in that class.

**Consequence for NG2.** The input lists per-environment topic duplication as
something the design should avoid introducing. Atlas already does it, for every
topic. Sparse mode does not avoid introducing it — it **removes** it. That is a
larger and riskier change than "don't add this," and the PRD treats it as such.

### 3.3 Context propagation has a clean seam — F6–F9

- **F6.** REST tenant context travels as four flat headers set by
  `requests.TenantHeaderDecorator` (`libs/atlas-rest/requests/header.go:29-43`):
  `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
- **F7.** The nginx ingress forwards them explicitly and enables
  `underscores_in_headers on` (`deploy/k8s/base/atlas-ingress.yaml`).
- **F8.** Kafka mirrors the same four values as message headers, parsed by
  `consumer.TenantHeaderParser` (`libs/atlas-kafka/consumer/header.go:28-66`),
  which reconstructs the tenant and returns `tenant.WithContext(ctx, t)`.
- **F9.** Consumer group IDs are already **runtime-resolved**, not baked:
  `consumergroup.Resolve` reads `KAFKA_CONSUMER_GROUP` and `fmt.Sprintf`s
  variadic args (`libs/atlas-kafka/consumergroup/resolver.go:38-50`). This is
  the one substrate that already has the shape sparse mode needs, and it is the
  model the other three should follow.

**Consequence.** Adding an environment dimension needs no new transport
mechanism. It is one more header on each side of an existing, centralized seam.

### 3.4 REST routing is a generated path→service map — F10

`deploy/k8s/base/routes.conf.template.generated` is a flat list of nginx
`location` blocks:

```nginx
location ~ ^/api/parties(/.*)?$ {
  set $u "atlas-parties.${POD_NAMESPACE}.svc.cluster.local:8080";
  proxy_pass http://$u$request_uri;
}
```

The namespace is **already a variable**. Environment-aware routing is therefore
a change of what that variable resolves to, not a restructuring of the file.

### 3.5 Tenant scoping in Postgres is near-total — F11

Of 89 `entity.go` files, **85 carry a `TenantId` field**. The four exceptions:

| Entity | Status | Assessment |
|---|---|---|
| `atlas-tenants/.../tenant/entity.go` | Expected | It *is* the tenant table. |
| `atlas-quest/.../quest/medal/entity.go` | Transitive | Scoped via FK `QuestStatusId` → a tenant-scoped parent. |
| `atlas-configurations/.../templates/entity.go` | **Gap** | Keyed by `(Region, MajorVersion, MinorVersion)` — no tenant column. |
| `atlas-configurations/.../services/entity.go` | **Gap** | Keyed by `Type` only — fully global. |

**Consequence.** `atlas-configurations` is the one service where the
shared-datastore model is unsafe as written. A PR that changes a socket template
or a service configuration would mutate the row `main` reads. This is a blocking
prerequisite, not a footnote — see FR-8.3.

### 3.6 Redis tenant scoping is convention, not type-enforced — F12

`libs/atlas-redis` exposes two families:

- **Tenant-scoped by construction:** `TenantRegistry`, `TenantCoalescedRegistry`,
  `TenantSet`, `TenantKeyedSet`, `TenantKeyedHash`, `TenantKeyedSortedSet`,
  `TenantCounter`. These build keys through
  `tenantEntityKey(namespace, t, entityKey)` (`keys.go:46`).
- **Not tenant-scoped by construction:** `Registry`, `Set`, `Hash`, `KeyedSet`,
  `KeyedHash`, `CoalescedRegistry`, `IDGenerator`, `GlobalIDGenerator`, `Index`,
  `Uint32Index`, `Lock`, `TTLRegistry`. These build keys through
  `namespacedKey(namespace, parts...)` — tenant appears only if the caller's
  `keyFn` puts it there.

Six services use the second family: `atlas-data`, `atlas-doors`,
`atlas-monsters`, `atlas-npc-shops`, `atlas-storage`, `atlas-summons`.

Spot-checking the most significant one: `atlas-monsters` **does** embed tenant —
`monsterKey(t tenant.Model, uniqueId uint32)`
(`services/atlas-monsters/.../monster/registry.go:303`). So live monster state is
in practice tenant-isolated despite the generic type.

**Consequence.** The risk is real but *per-callsite*, not systemic. The audit
must enumerate every `keyFn` in those six services rather than assume either
outcome. Today the `ATLAS_ENV` prefix hides any callsite that forgot tenant;
removing it makes such a callsite an immediate cross-environment leak. See
`isolation-audit.md`.

### 3.7 Autonomous work already loops over tenants — F13

The established pattern, e.g. `services/atlas-transports/.../main.go:93-151`:

```go
tenants, err := tenant2.NewProcessor(l, rt.Context()).GetAll()
for _, t := range tenants {
    ctx := tenant.WithContext(rt.Context(), t)
    // ... reconcile / tick for this tenant
}
```

**Consequence.** Per-environment autonomous execution is a natural
generalization of an idiom Atlas already uses — the loop gains a dimension
rather than a new architecture. **But** the tenant list comes from
`atlas-tenants`, whose data lives in the per-environment database (F3). A
baseline `atlas-tenants` therefore cannot see ephemeral tenants today.

### 3.8 Environment lifecycle machinery already exists — F14

Teardown is a PostDelete hook (`atlas-pr-cleanup-<N>` in the `argocd`
namespace) running a phased `cleanup.sh` that reclaims databases, topics,
consumer groups, Redis keys, GHCR tags, and the bot branch; it runs all phases
regardless of individual failure and reports `failed_phases`
(`docs/runbooks/ephemeral-pr-deployments.md` §9.4). Bootstrap restores a
canonical data baseline from MinIO rather than re-ingesting WZ (§9.1).

**Consequence.** Sparse mode inherits a working lifecycle and must extend it
(deactivate-before-destroy), not invent one.

---

## 4. Decisions taken

These were settled during the specification interview and are binding on design.

| # | Decision | Rationale |
|---|---|---|
| **D1** | **Shared datastores, tenant-scoped rows.** Sparse environments use `main`'s databases, Kafka topics, and Redis keyspace. Isolation is by `tenant_id` on the data plus `environment` on the operation. | Avoids making every service's persistence layer environment-aware. Matches the input's §11 preference for deriving environment from tenant. Cost: the §3.5/§3.6 gaps become blocking prerequisites. |
| **D2** | **Environment CRD reconciled by Argo CD**, watched by services through a shared library and cached in memory; nginx configuration generated from the same source. | Declarative, colocated with the deployment that creates the overrides, and gives readiness/lifecycle a natural home. |
| **D3** | **POC plus full migration of all 64 Deployments** in this task. | Partial migration leaves an ambiguous middle state where some services leak and some don't, which is worse than either endpoint. |
| **D4** | **Fail closed.** An operation whose environment cannot be resolved is dropped and alerted, never executed by the baseline. | Per input §15: duplicate or misdirected processing is strictly worse than non-processing. |

D1 has a direct consequence worth stating plainly: **PR data lands in `main`'s
databases**, separated only by `tenant_id`. This is a deliberate trade. It makes
the feature tractable and it is consistent with how Atlas already treats tenancy
as the isolation primitive — but it means a single missing `WHERE tenant_id`
becomes a cross-environment data leak where previously it was contained by the
database boundary. FR-8 exists to close that gap before anything else ships.

---

## 5. User stories

- As an **Atlas developer**, I want my one-service PR to spin up in under a
  minute so that I can iterate without waiting for 64 Deployments to become
  ready.
- As an **Atlas developer**, I want a request I send into my PR environment to
  execute my changed code and the baseline's code for everything else, so that I
  am testing my change in a realistic system.
- As an **Atlas developer**, I want to know that a monster spawned by
  `atlas-monsters-main` on behalf of my PR is *my* monster, so that background
  behavior is testable in a sparse environment.
- As a **platform operator**, I want to distinguish `atlas-monsters-main`
  processing `main` from the same pod processing `pr-123` in logs and metrics,
  so that I can debug a shared deployment.
- As a **platform operator**, I want a PR's teardown to guarantee no delayed work
  later executes against `main`, so that ephemeral environments cannot corrupt
  the baseline.
- As a **release engineer**, I want changes that sparse mode cannot safely
  validate to escalate automatically to a fully-isolated environment, so that
  the cheap path is never silently the wrong path.

---

## 6. Functional requirements

### FR-1 — Environment model and registry

- **FR-1.1** An `Environment` custom resource represents one execution
  environment, carrying: identity, baseline reference, the set of service
  overrides, associated tenant(s), and lifecycle phase.
- **FR-1.2** The registry answers `Resolve(environment, service) → deployment`.
  For an environment with no override for that service, it returns the
  baseline's deployment.
- **FR-1.3** The registry answers `EnvironmentsOwnedBy(service, deployment) →
  [environment]`: every active environment for which this deployment is the
  effective implementation of this service.
- **FR-1.4** The registry answers `IsOwner(deployment, environment, service) →
  bool` and `IsEnvironmentActive(environment) → bool`.
- **FR-1.5** The baseline must not be hard-coded to `main`. It is a field on the
  Environment resource. Supporting a second baseline must require no code change.
- **FR-1.6** All four queries resolve from an in-process cache. No REST, Kafka,
  database, or Kubernetes API call occurs on the resolution path.
- **FR-1.7** Cache staleness has a defined bound. Design must specify the bound
  and the behavior on exceeding it; per D4 the behavior is fail-closed.
- **FR-1.8** For a deployment running in a single-environment cluster with no
  Environment resources present, every query returns the legacy answer: the
  local deployment owns everything. This makes the feature inert for `main`.

### FR-2 — Execution context

- **FR-2.1** Environment is carried in `context.Context` alongside tenant, using
  the same `WithContext` / `FromContext` idiom.
- **FR-2.2** Environment is established at exactly four origination points:
  an inbound external request; an inbound REST call carrying the header; an
  inbound Kafka message carrying the header; and an autonomous loop iterating
  owned environments (FR-6).
- **FR-2.3** Any operation that reaches domain code without a resolvable
  environment is rejected before the domain call. It does not default to
  baseline.
- **FR-2.4** Environment context is immutable for the duration of an operation.
  No code path may rewrite it.

### FR-3 — REST

- **FR-3.1** REST calls carry environment as a header alongside the existing
  four tenant headers, set centrally by the request decorator in
  `libs/atlas-rest`. No service sets it by hand.
- **FR-3.2** A service handling a request for `pr-123` emits `pr-123` on every
  downstream REST call it makes, regardless of which deployment it is. A
  baseline deployment must never rewrite the environment to its own.
- **FR-3.3** The ingress resolves the upstream from `(environment, service)`:
  the override's Service when one exists, the baseline's otherwise.
- **FR-3.4** The ingress obtains routing data from generated configuration
  derived from the Environment resources. It performs no per-request lookup
  against the registry.
- **FR-3.5** If an environment is `ACTIVE` and declares an override for the
  target service, and that override is unavailable, the ingress returns an
  error. It must not fall back to the baseline.
- **FR-3.6** A request naming an unknown or inactive environment is rejected.

### FR-4 — Kafka

- **FR-4.1** Messages carry environment as a message header, written centrally by
  the producer wrapper and parsed centrally by the consumer header parser,
  mirroring the existing tenant treatment (F8).
- **FR-4.2** A consumer processing a message for `pr-123` emits `pr-123` on every
  message it produces, regardless of the deployment's own identity.
- **FR-4.3** Override deployments use consumer groups distinct from the baseline,
  via the existing `KAFKA_CONSUMER_GROUP` mechanism (F9). No new mechanism is
  introduced.
- **FR-4.4** An **ownership gate** runs in the consumer infrastructure, before
  the domain handler is invoked. If this deployment is not the effective
  implementation for `(message.environment, service)`, the message is
  acknowledged without domain processing.
- **FR-4.5** The gate lives in `libs/atlas-kafka`. No domain processor contains
  environment-conditional logic. This must be enforceable by a repo guard.
- **FR-4.6** Exactly one deployment processes any environment-scoped message.
  Baseline and override must never both process it.
- **FR-4.7** A message whose environment is unknown to the cached registry is
  **not processed by any deployment**, is acknowledged, and emits a distinct
  alertable metric and log line (D4).
- **FR-4.8** Sparse environments consume the **unsuffixed** baseline topics.
  Topic suffixing (F4) applies to isolated mode only. Design must specify how a
  single `env-configmap` serves both modes.
- **FR-4.9** New ephemeral consumer groups must not skip messages by starting at
  `latest`. Design must define the offset initialization and prove that a
  message produced between group creation and first poll is not lost.
- **FR-4.10** An environment does not become `ACTIVE` until its override
  consumer groups exist and their offsets are initialized (FR-5.3).

### FR-5 — Lifecycle

- **FR-5.1** Phases: `PROVISIONING → ACTIVE → DEACTIVATING → DELETED`.
- **FR-5.2** During `PROVISIONING`, baseline deployments continue to own the
  environment's services; overrides do not yet receive work; the ingress does not
  route to the environment.
- **FR-5.3** Transition to `ACTIVE` requires every override Deployment ready
  **and** every override consumer group initialized (FR-4.9).
- **FR-5.4** Activation is atomic from the perspective of ownership: there is no
  observable window in which both baseline and override own a service, or in
  which neither does.
- **FR-5.5** `DEACTIVATING` stops new work being routed or gated to the
  environment *before* any override workload is destroyed.
- **FR-5.6** Design must define drain semantics for in-flight REST requests,
  unacknowledged Kafka messages, and scheduled work.
- **FR-5.7** After `DELETED`, any surviving delayed or persisted work for that
  environment is discarded. It must never execute against the baseline.

### FR-6 — Autonomous processing

- **FR-6.1** Every background loop — tickers, schedulers, watchdogs, respawn
  processors, cleanup tasks — iterates the environments it owns and executes
  independently within each. The 18 files identified in F13/§3.1 are the starting
  inventory; design must confirm it is complete.
- **FR-6.2** Work originated autonomously carries the environment of the
  iteration that produced it, on both REST and Kafka.
- **FR-6.3** A baseline deployment must not originate work for an environment
  that overrides its service. The registry is authoritative.
- **FR-6.4** Ownership changes between iterations take effect on the next
  iteration. A loop must not cache an ownership set across ticks.
- **FR-6.5** Per-environment execution is isolated for faults and latency: a
  failure or a slow iteration in one environment must not prevent or delay
  another.
- **FR-6.6** Iteration cost scales with the number of *owned* environments. A
  deployment owning only `main` performs the same work it does today.

### FR-7 — Tenant integration

- **FR-7.1** The ephemeral tenant ↔ environment relationship is **one-to-one**
  for this task. One environment has exactly one ephemeral tenant; that tenant
  belongs to exactly one environment.
- **FR-7.2** `main` is exempt: the baseline environment may contain many
  tenants. The one-to-one constraint applies to ephemeral environments only.
- **FR-7.3** Environment must be derivable from tenant identity, so that
  autonomous work starting from persisted tenant-owned state resolves its
  environment without an environment column.
- **FR-7.4** Environment identity is **not** persisted on domain objects.
  `tenant_id` remains the only scoping column.
- **FR-7.5** Ephemeral tenants must be registered in the **baseline** tenant
  store, since baseline services enumerate tenants from it (F13). Design must
  specify how, and how registration is reversed on teardown.
- **FR-7.6** Tenant lifecycle is bound to environment lifecycle: the tenant
  exists no earlier than `PROVISIONING` and is removed as part of teardown.
- **FR-7.7** Because environment derives from tenant, a mismatch between the
  environment header and the environment implied by the tenant is a hard error,
  not a reconciliation. Design must specify which wins for detection purposes;
  neither may silently override the other.

### FR-8 — Data isolation prerequisites (blocking)

D1 makes these load-bearing. **None of the routing work is safe to enable until
FR-8.1–8.3 are complete.**

- **FR-8.1** Every Postgres query path in every service must be tenant-scoped.
  The audit must be a sweep, not a sample, and its result recorded per service.
- **FR-8.2** Every Redis key constructed through the non-tenant-scoped family
  (F12) in `atlas-data`, `atlas-doors`, `atlas-monsters`, `atlas-npc-shops`,
  `atlas-storage`, `atlas-summons` must be shown to embed tenant in its `keyFn`,
  or be migrated to the tenant-scoped family. Enumerate every callsite.
- **FR-8.3** `atlas-configurations` must become tenant-scoped. Its `templates`
  table is keyed `(Region, MajorVersion, MinorVersion)` and its `services` table
  is global (F11); under shared datastores a PR editing either would mutate the
  rows `main` reads. This requires a schema change and a data migration.
- **FR-8.4** A repo guard must fail CI on a new non-tenant-scoped entity or a
  new non-tenant Redis keyFn, so the audit cannot silently regress.
- **FR-8.5** Any remaining resource whose isolation depends on deployment
  identity must be enumerated with a disposition: migrate to tenant scoping, or
  force isolated mode (FR-9).

### FR-9 — Mode selection and escalation

- **FR-9.1** Two modes exist: **sparse** (default for PR validation) and
  **isolated** (existing behavior).
- **FR-9.2** Affected-service determination must account for direct service
  changes, shared-library (`libs/`) changes, deployment/config changes, Kafka
  contract changes, and schema changes. It must be conservative: when in doubt,
  treat a service as overridden.
- **FR-9.3** The following **force isolated mode**, automatically:
  - changes to `deploy/k8s/base` or the ingress configuration;
  - changes to Kafka infrastructure or message contracts;
  - database migrations;
  - changes to cross-cutting libraries where the blast radius cannot be bounded
    (`libs/atlas-kafka`, `libs/atlas-rest`, `libs/atlas-tenant`,
    `libs/atlas-redis`, and the environment library itself);
  - any case where affected-service determination is unreliable.
- **FR-9.4** Mode selection is overridable per PR by an explicit label, in both
  directions.
- **FR-9.5** A scheduled full-stack isolated deployment continues to run, so that
  Atlas's ability to bootstrap from nothing is still proven independently of
  sparse environments.
- **FR-9.6** The chosen mode and the reason for it must be reported on the PR.

### FR-10 — Observability

- **FR-10.1** Environment is a field in structured logs, emitted automatically by
  the logging setup rather than by callers.
- **FR-10.2** Environment is present in Kafka processing logs, REST access logs,
  and error reporting.
- **FR-10.3** Environment is a label on metrics where cardinality permits; design
  must state the cardinality budget and which metrics are excluded.
- **FR-10.4** The ownership gate emits counters for messages processed, skipped
  as not-owner, and dropped as unresolvable (FR-4.7). The third is alertable.
- **FR-10.5** Operators can filter logs for one shared deployment serving one
  environment. Note: Loki has no `app` label in this cluster — selectors use
  `service_name`.

---

## 7. API surface

No new externally-facing REST API is required. The changes are to transport
metadata and to one Kubernetes resource.

### 7.1 REST header

A single new header alongside the four tenant headers (F6), following their
flat-uppercase-underscore convention so that the ingress's
`underscores_in_headers on` and explicit `proxy_set_header` handling apply
unchanged. Exact spelling is a design decision; it must be set only by the
central decorator in `libs/atlas-rest/requests` and read only by the central
parser.

### 7.2 Kafka header

One message header mirroring the REST header, written by the producer wrapper
and read by a parser in `libs/atlas-kafka/consumer` alongside
`TenantHeaderParser`. Message *bodies* are unchanged — no domain schema is
touched, which is what keeps this migration mechanical across 64 services.

### 7.3 Environment resource

```yaml
apiVersion: <group>/<version>
kind: Environment
metadata:
  name: pr-123
spec:
  baseline: main
  tenant: <ephemeral tenant id>
  overrides:
    atlas-character: { deployment: atlas-character-pr-123 }
    atlas-channel:   { deployment: atlas-channel-pr-123 }
status:
  phase: ACTIVE
  overridesReady: true
  consumerGroupsReady: true
```

Group, version, and the readiness condition vocabulary are design decisions.

### 7.4 Error semantics

| Condition | Behavior |
|---|---|
| Unknown / inactive environment on a REST request | Reject (FR-3.6) |
| Override declared but unavailable | Error; never fall back (FR-3.5) |
| Unknown environment on a Kafka message | Ack, do not process, alert (FR-4.7) |
| Registry cache stale beyond bound | Fail closed (FR-1.7) |

---

## 8. Data model

### 8.1 No new domain columns

Per FR-7.4, environment is not persisted on domain objects. It is derived from
tenant (FR-7.3). This is what keeps the change out of 85 entity definitions.

### 8.2 Required schema changes

Confined to `atlas-configurations` (FR-8.3), which is the only service with
non-tenant-scoped domain tables (F11):

| Table | Current key | Required |
|---|---|---|
| `templates` | `(Region, MajorVersion, MinorVersion)` | add tenant scoping |
| `services` | `Type` only | add tenant scoping |

Both need a backfill migration assigning existing rows to the appropriate
tenant(s), and both are read on hot configuration paths — design must confirm
the change does not alter `main`'s resolution behavior (NG6).

### 8.3 Tenant registration

Ephemeral tenants must exist in the baseline tenant store (FR-7.5), which today
they do not, because `atlas-tenants` runs against a per-environment database
(F3). Under D1 that database is shared, so registration becomes a row insert
during `PROVISIONING` and a delete during teardown — but the ordering against
FR-5 is a design obligation, since a tenant visible before activation would be
picked up by baseline autonomous loops (F13) prematurely.

---

## 9. Service impact

### 9.1 Shared libraries — where the work concentrates

| Library | Change |
|---|---|
| `libs/atlas-tenant` (or a new environment lib) | environment context type, `WithContext` / `FromContext` |
| `libs/atlas-rest` | header decorator + inbound parser (FR-3.1, FR-3.2) |
| `libs/atlas-kafka` | header write/parse, **ownership gate** (FR-4.4), group naming already sufficient (F9) |
| new environment registry lib | CRD watch, in-memory cache, the four queries (FR-1) |
| `libs/atlas-redis` | tenant-scoping migration for the six services in F12; `ATLAS_ENV` prefix must become inert in sparse mode |
| `libs/atlas-routine` | per-environment iteration helper for background loops (FR-6) |

The design goal is that a typical service changes **only its `main.go` wiring
and its background loops**. If a domain package needs editing, the abstraction
is in the wrong place (NG5).

### 9.2 Services needing individual attention

- **`atlas-configurations`** — schema change and migration (FR-8.3). Blocking.
- **`atlas-tenants`** — ephemeral tenant registration against the baseline store
  (FR-7.5), and `GetAll()` semantics now spanning environments.
- **`atlas-monsters`, `atlas-summons`, `atlas-doors`, `atlas-storage`,
  `atlas-npc-shops`, `atlas-data`** — Redis keyFn audit (FR-8.2).
- **`atlas-saga-orchestrator`** — saga continuation must restore environment
  across steps (input §7.3); it has a ticker and is a persisted-work path.
- **`atlas-channel` / `atlas-login`** — templated consumer groups; they already
  pass variadic args to `consumergroup.Resolve` (F9), so the interaction with
  per-environment group naming needs explicit design.
- **The 18 ticker-bearing files** (§3.1) — per-environment iteration (FR-6.1).

### 9.3 Deployment and CI

- `deploy/k8s/overlays/` gains a sparse overlay producing only overridden
  services plus the Environment resource.
- `.github/workflows/pr-validation.yml` gains affected-service determination and
  mode selection (FR-9).
- `cleanup.sh` gains deactivate-before-destroy ordering (FR-5.5) and tenant
  deregistration.
- Under D1, the per-env DB creation and topic pre-creation phases become
  no-ops for sparse environments — but must remain for isolated mode.

---

## 10. Non-functional requirements

- **NFR-1 (Performance).** Environment resolution on the REST and Kafka hot path
  is an in-memory map lookup. No I/O (FR-1.6, NG4).
- **NFR-2 (Overhead).** For a deployment owning only `main`, added per-operation
  cost is negligible and no additional background work is performed (FR-6.6).
- **NFR-3 (Capacity).** A sparse environment's incremental footprint is
  proportional to its override count. Several concurrent sparse environments cost
  a small multiple of one, not a multiple of 64.
- **NFR-4 (Startup).** A sparse environment reaches `ACTIVE` substantially faster
  than the isolated equivalent. Design should state a target; the bootstrap's
  ~60s baseline restore (§3.8) is no longer on the critical path when the
  environment shares `main`'s data.
- **NFR-5 (Correctness).** Cross-environment leakage and duplicate processing are
  P0 defects. Test coverage must assert the *new* contract explicitly — per
  CLAUDE.md, existing tests may currently pin the old silent-drop behavior.
- **NFR-6 (Multi-tenancy).** Environment composes with tenancy; it does not
  replace or bypass it (NG7).
- **NFR-7 (Backward compatibility).** With no Environment resources present,
  behavior is byte-for-byte today's behavior (FR-1.8, NG6).
- **NFR-8 (Security).** Under D1, ephemeral tenants share `main`'s datastores.
  Design must confirm that no ephemeral tenant can read or write another
  tenant's data through any endpoint — the tenant boundary now carries weight the
  database boundary used to.

---

## 11. Open questions

1. **Header spelling and precedence.** Which name, and what happens when the
   environment header and the tenant-derived environment disagree (FR-7.7)?
2. **Consumer offset initialization.** What concrete mechanism satisfies FR-4.9,
   and how is "initialized" observed for the activation gate (FR-5.3)?
3. **Activation atomicity.** How is FR-5.4 achieved given that registry caches
   across many pods update independently? Is a two-phase ownership handover
   needed?
4. **Staleness bound.** What is the numeric bound in FR-1.7, and what is the
   observable behavior at the boundary?
5. **nginx configuration distribution.** Generated ConfigMap plus reload, or
   `njs`-based map lookup? The ~60s ConfigMap propagation delay interacts badly
   with FR-5.4.
6. **Drain semantics.** Concretely, what does FR-5.6 mean for an unacknowledged
   Kafka message belonging to a `DEACTIVATING` environment?
7. **`atlas-configurations` migration shape.** Does it get a tenant column, or a
   tenant-scoped overlay table that falls back to a global default? The latter
   may preserve `main` behavior more cleanly.
8. **Metrics cardinality budget** (FR-10.3).
9. **Sequencing of the 64-service migration.** D3 commits to all of them; what
   is the safe order, and is there an intermediate state where some services are
   environment-aware and others are not that is *not* dangerous?
10. **Does `atlas-channel` hold per-environment socket state** that the
    one-to-one tenant/environment mapping does not cover? It is the input's own
    example override and it carries session state.

---

## 12. Acceptance criteria

### Prerequisites (must be complete before routing is enabled)

- [ ] Every service's Postgres access is verified tenant-scoped, recorded per
      service (FR-8.1).
- [ ] Every non-tenant-scoped Redis keyFn in the six identified services is
      enumerated and shown tenant-scoped or migrated (FR-8.2).
- [ ] `atlas-configurations` `templates` and `services` are tenant-scoped and
      migrated, with `main`'s resolution behavior unchanged (FR-8.3).
- [ ] A repo guard fails CI on new non-tenant-scoped entities or Redis keyFns
      (FR-8.4).
- [ ] Every remaining deployment-scoped resource is enumerated with a
      disposition (FR-8.5).

### Core capability

- [ ] The Environment CRD, its controller, and the registry library exist, and
      all four queries (FR-1.2–1.4) resolve from an in-memory cache with no I/O.
- [ ] Environment propagates over REST and Kafka via central plumbing; no domain
      processor references environment (FR-4.5, enforced by guard).
- [ ] The ingress routes on `(environment, service)` with no per-request registry
      call, and fails rather than falling back when an override is down.
- [ ] The Kafka ownership gate guarantees exactly-one-processor, and drops +
      alerts on unresolvable environments.
- [ ] All 64 Deployments are environment-aware (D3).
- [ ] All 18 ticker-bearing background loops iterate owned environments (FR-6.1).

### The §17 proof — the case that proves environment ≠ deployment

- [ ] Given `pr-123` overriding `atlas-character` and `atlas-channel`, **only
      those two additional workloads are deployed.** The other 62 are not.
- [ ] A request entering `pr-123` executes through a mix of override and
      baseline deployments and remains in `environment=pr-123`,
      `tenant=ephemeral-T123` for its entire lifecycle.
- [ ] The equivalent `main` request is unaffected and behaviorally unchanged.
- [ ] **A baseline deployment originates autonomous work separately for `main`
      and for `pr-123`**, and the `pr-123` work is delivered to the `pr-123`
      override.
- [ ] A baseline deployment originates **no** autonomous work for an environment
      that overrides its service (FR-6.3).

### Lifecycle and teardown

- [ ] An environment becomes `ACTIVE` only when overrides *and* consumer groups
      are ready, with no window of double or zero ownership.
- [ ] Teardown deactivates routing before destroying workloads.
- [ ] After deletion, no delayed, scheduled, or in-flight work executes against
      `main`. Verified by test, not by inspection.

### Mode selection

- [ ] Affected-service determination is conservative and covers direct, library,
      deployment, contract, and schema changes.
- [ ] Every FR-9.3 condition escalates to isolated mode automatically.
- [ ] Isolated mode still works, unchanged.
- [ ] The scheduled full-stack deployment still runs (FR-9.5).

### Verification

- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] Tests assert the new contract, including the negative cases — not-owner
      skip, unresolvable drop, no-fallback-on-override-down (NFR-5).
- [ ] Multiple concurrent sparse environments coexist with resource use
      proportional to overrides (NFR-3).
