# Sparse Ephemeral Environments — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-08-15
Updated: 2026-08-15 — control-plane scoping, mandatory socket overrides, Redis API policy

---

## 1. Overview

Atlas deploys 64 Kubernetes Deployments per environment. The per-PR ephemeral
environment introduced by task-063 duplicates that entire runtime into a
namespace of its own (`atlas-pr-<N>`) even when the pull request changes one
service. The cost is paid in compute, memory, cluster workload count, and — most
visibly — startup latency, since every PR waits for the full stack to become
ready before any test traffic can flow.

This project introduces a **sparse** ephemeral environment: a PR deploys only
the services it changed, plus a small mandatory floor, and every other service is
served by the `main` baseline. A request entering `pr-123` may execute through
`atlas-character-pr-123`, then `atlas-monsters-main`, then `atlas-channel-pr-123`,
and must remain logically inside `pr-123` for its entire lifecycle — across REST
hops, Kafka messages, saga continuations, scheduled callbacks, and autonomous
background loops. The existing fully-isolated mode remains available and becomes
the documented escalation path.

The central engineering claim of this document is that **execution environment is
a property of the operation, not of the deployment processing it** — exactly as
tenant already is in Atlas. The central engineering *risk* is that Atlas's
current ephemeral isolation is not built that way: it is process-scoped, fixed at
pod start, in three independent substrates.

A second distinction runs through the whole design and is worth stating up front:
Atlas has a **data plane** (gameplay state, isolated by tenant) and a **control
plane** (the tenant registry, socket templates, and login/channel service
registrations — the things that *provision and serve* tenants). These scope
differently. The data plane is tenant-scoped; the control plane is
environment-scoped. Conflating them was the principal error in v1 of this
document.

---

## 2. Goals

### Primary goals

- **G1.** A PR deploys only its changed services plus the mandatory socket floor
  (§4, D6); unchanged services are served by the `main` baseline. Resource
  consumption scales with changed services, not with 64.
- **G2.** Execution environment propagates across every asynchronous and
  synchronous boundary Atlas has: REST, Kafka, saga steps, scheduled work, and
  autonomous loops.
- **G3.** A shared baseline deployment performs autonomous work independently for
  each environment it is the effective implementation for, and for no environment
  it is not.
- **G4.** No operation ever silently transitions from an ephemeral environment to
  `main`. Cross-environment leakage is a correctness bug, not a degradation.
- **G5.** All 64 Deployments are migrated to the environment-aware model in this
  task (see §12 for phasing).
- **G6.** The existing fully-isolated mode continues to work unchanged, with
  documented rules for when it is required.
- **G7.** `main` is never mutated or redeployed in order to serve an ephemeral
  environment.

### Non-goals

- **NG1.** A service mesh. Routing is solved with the existing nginx ingress.
- **NG2.** Per-environment Kafka topic duplication *for sparse environments*.
  (Note: this is a **reversal** of current behavior — see F4.)
- **NG3.** A Kafka routing or proxy service.
- **NG4.** Synchronous registry lookups on the REST or Kafka hot path.
- **NG5.** Environment-conditional logic inside domain processors.
- **NG6.** Changing `main`'s observable runtime behavior, configuration, or
  deployment in order to support sparse environments.
- **NG7.** Replacing the tenant model with a parallel isolation mechanism.

---

## 3. Findings — verified current state

Everything in this section was read from the repository at branch point
`c8d44127c`. File and line references are given so design can re-verify.

### 3.1 Scale

| Measure | Value | Source |
|---|---|---|
| Directories under `services/` | 66 | `ls -d services/*/` |
| `atlas-*.yaml` manifests in `deploy/k8s/base` | 68 | — |
| `kind: Deployment` in base | **64** | `grep -h "^kind: Deployment"` |
| Go files constructing a `time.NewTicker` | 18 | `grep -rl NewTicker services` |

### 3.2 The three isolation substrates are process-scoped — F1–F5

Ephemeral isolation today is achieved by giving each PR namespace its own
*copies* of the underlying resources, selected by a value fixed for the lifetime
of the pod:

| # | Substrate | Mechanism | Source |
|---|---|---|---|
| **F3** | Postgres | `DB_NAME` patched to `<db>-<ATLAS_ENV>` per Deployment | `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh` |
| **F4** | Kafka topics | every `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` suffixed `-<ATLAS_ENV>` | `overlays/pr/scripts/gen-topic-config.sh`; `overlays/pr/kustomization.yaml:154-` |
| **F5** | Redis | key prefix `<ATLAS_ENV>:atlas`, computed from `os.Getenv("ATLAS_ENV")` in a **package-level `var` initializer** | `libs/atlas-redis/keys.go:15-24` |

- **F1.** PR environments are namespace-per-PR (`atlas-pr-<N>`), built by applying
  the *entire* `deploy/k8s/base` kustomization plus a PR overlay.
- **F2.** Placeholders are `sed`-substituted at CI time by
  `.github/workflows/pr-validation.yml` and force-pushed to `bot/pr-<N>-resolved`,
  which the Argo CD ApplicationSet targets.

**Consequence.** A shared `atlas-monsters-main` pod holds exactly one DSN, one
topic set, and one Redis key prefix. It is structurally incapable of serving
`pr-123` today. All three are instances of the input's §14 concern: isolation
based on deployment identity rather than tenant identity.

**Consequence for NG2.** Atlas already duplicates topics per environment. Sparse
mode does not avoid introducing that — it **removes** it.

### 3.3 Context propagation has a clean seam — F6–F9

- **F6.** REST tenant context travels as four flat headers set by
  `requests.TenantHeaderDecorator` (`libs/atlas-rest/requests/header.go:29-43`):
  `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`.
- **F7.** The nginx ingress forwards them explicitly and enables
  `underscores_in_headers on` (`deploy/k8s/base/atlas-ingress.yaml`).
- **F8.** Kafka mirrors the same four values as message headers, parsed by
  `consumer.TenantHeaderParser` (`libs/atlas-kafka/consumer/header.go:28-66`).
- **F9.** Consumer group IDs are already **runtime-resolved**:
  `consumergroup.Resolve` reads `KAFKA_CONSUMER_GROUP` and `fmt.Sprintf`s variadic
  args (`libs/atlas-kafka/consumergroup/resolver.go:38-50`). This is the one
  substrate already shaped the way sparse mode needs, and the model for the rest.

### 3.4 REST routing is a generated path→service map — F10

`deploy/k8s/base/routes.conf.template.generated` is a flat list of nginx
`location` blocks whose upstream is
`atlas-<svc>.${POD_NAMESPACE}.svc.cluster.local:8080`. The namespace is **already
a variable**, so environment-aware routing changes what it resolves to, not the
structure of the file.

### 3.5 Postgres scoping splits along a control-plane / data-plane line — F11

Of 89 `entity.go` files, **84 have a tenant-scoped primary entity.** Five do not:

| Entity | Finding | Plane |
|---|---|---|
| `atlas-tenants/.../tenant/entity.go` | It *is* the tenant table | Control |
| `atlas-configurations/.../tenants/entity.go` | Tenant registry keyed `(Id, Region, Major, Minor)`. **No tenant column on the primary entity** — only its `HistoryEntity` has `TenantId` (line 29) | Control |
| `atlas-configurations/.../templates/entity.go` | Socket templates keyed `(Region, MajorVersion, MinorVersion)` | Control |
| `atlas-configurations/.../services/entity.go` | Service registrations keyed by `Type` | Control |
| `atlas-quest/.../quest/medal/entity.go` | Keyed `(QuestStatusId, MapId)`; scoped transitively through a tenant-scoped parent | Data |

**This is not a set of bugs.** Four of the five are control-plane tables whose
entire purpose is to enumerate and serve *many* tenants. Adding a `tenant_id`
column to the table that lists tenants is incoherent; a socket template is shared
by every tenant on a version; a login-service registration deliberately carries a
list of tenants. They are correctly not tenant-scoped. What they lack is an
**environment** dimension — see D5.

### 3.6 Redis tenant scoping is convention, not type-enforced — F12

`libs/atlas-redis` exposes two families:

- **Tenant-scoped by construction:** `NewTenantRegistry`,
  `NewTenantCoalescedRegistry`, `NewTenantSet`, `NewTenantKeyedSet`,
  `NewTenantKeyedHash`, `NewTenantKeyedSortedSet`, `NewTenantCounter` — keys built
  via `tenantEntityKey(namespace, t, entityKey)` (`keys.go:46`).
- **Not tenant-scoped by construction:** `NewRegistry`, `NewSet`, `NewHash`,
  `NewKeyedSet`, `NewKeyedHash`, `NewCoalescedRegistry`, `NewIDGenerator`,
  `NewGlobalIDGenerator`, `NewIndex`, `NewUint32Index`, `NewLock`,
  `NewLockWithTTL`, `NewTTLRegistry` — keys built via
  `namespacedKey(namespace, parts...)`; tenant appears only if the caller's `keyFn`
  supplies it.

Six services use the second family. Four callsites were checked during
specification:

| Callsite | Key | Verdict |
|---|---|---|
| `atlas-monsters` `monsterKey(t tenant.Model, uniqueId uint32)` (`monster/registry.go:303`) | embeds tenant | Scoped |
| `atlas-storage` `NpcContextCache` (`storage/cache.go:31-36`) | `strconv(characterId)` only | **UNSCOPED** |
| `atlas-npc-shops` consumable cache (`shops/cache.go:32`) | shop `uuid` only | Suspect |
| `atlas-doors` registry (`door/registry.go:69-76`) | identity `keyFn`; key built by callers | Unknown |
| `atlas-data` `ingestrun` (`ingestrun.go:114-120`) | identity `keyFn`, namespace `ingestrun` | Legitimately cross-tenant (control plane) |

**`atlas-storage`'s `npc-context` cache is keyed by character id with no tenant.**
Character IDs are per-tenant sequences, so this is a latent cross-tenant
collision *today* within multi-tenant `main`; the `ATLAS_ENV` prefix currently
masks it across environments. Four callsites checked, one already wrong — the
generic API is a live footgun, not a theoretical one. See D7.

### 3.7 Autonomous work already loops over tenants — F13

The established pattern, e.g. `services/atlas-transports/.../main.go:93-151`:

```go
tenants, err := tenant2.NewProcessor(l, rt.Context()).GetAll()
for _, t := range tenants {
    ctx := tenant.WithContext(rt.Context(), t)
    // ... reconcile / tick for this tenant
}
```

Per-environment autonomous execution is a natural generalization: the loop gains
a dimension rather than a new architecture.

### 3.8 Environment lifecycle machinery already exists — F14

Teardown is a PostDelete hook (`atlas-pr-cleanup-<N>` in `argocd`) running a
phased `cleanup.sh` that reclaims databases, topics, consumer groups, Redis keys,
GHCR tags and the bot branch, reporting `failed_phases`
(`docs/runbooks/ephemeral-pr-deployments.md` §9.4). Bootstrap restores a canonical
data baseline from MinIO rather than re-ingesting WZ (§9.1).

### 3.9 Socket services are bound to ports by tenant — F15

This finding determines the sparse floor.

`atlas-login` and `atlas-channel` each identify themselves by a `SERVICE_ID`
environment variable (`login/main.go:54`, `channel/main.go:176`) and read their
registration from `atlas-configurations`' `services` table
(`services/service/rest.go`):

```go
type LoginRestModel struct   { Tenants []LoginTenantRestModel }   // { Id, Port }
type ChannelRestModel struct { Tenants []ChannelTenantRestModel } // { Id, IPAddress, Worlds[]{ Channels[]{ Port } } }
```

A login service instance binds **one port per tenant**; a channel service binds
one port per (tenant, world, channel) and advertises an `IPAddress` that the
client is redirected to.

**Consequence.** The contract *technically* supports multiple tenants of the same
version on one service instance. But making `atlas-login-main` serve an ephemeral
tenant would require adding that tenant + port to `main`'s service registration
row and having `main`'s login bind a new port — i.e. **mutating `main`'s
configuration and redeploying `main` to serve a PR.** That violates NG6 and G7,
and makes `main` writable by PR CI. See D6.

---

## 4. Decisions taken

| # | Decision | Rationale |
|---|---|---|
| **D1** | **Shared datastores, tenant-scoped rows** for the data plane. Sparse environments use `main`'s databases, Kafka topics, and Redis keyspace; isolation is `tenant_id` on the data plus `environment` on the operation. | Avoids making every service's persistence layer environment-aware. |
| **D2** | **Environment CRD reconciled by Argo CD**, watched by services through a shared library and cached in memory; nginx configuration generated from the same source. | Declarative, colocated with the overrides, natural home for readiness. |
| **D3** | **POC plus full migration of all 64 Deployments.** | Partial migration leaves an ambiguous middle state where some services leak and some don't. |
| **D4** | **Fail closed.** An operation whose environment cannot be resolved is dropped and alerted, never executed by the baseline. | Duplicate or misdirected processing is strictly worse than non-processing. |
| **D5** | **The control plane is environment-scoped, not tenant-scoped.** `atlas-configurations`' `tenants`, `templates` and `services` tables — and `atlas-tenants`' tenant table — gain an **environment** dimension. They do not gain a tenant column. | These tables exist to provision and serve *many* tenants (F11, F15). Tenant-scoping them is a category error. |
| **D6** | **`atlas-login` and `atlas-channel` are mandatory overrides in every sparse environment.** The sparse floor is two workloads, never zero. | Serving an ephemeral tenant from `main`'s socket services requires mutating and redeploying `main` (F15) — violating NG6/G7. |
| **D7** | **The tenant-scoped Redis API is mandatory for data-plane state.** The non-tenant constructors are withdrawn from domain use; genuinely cross-tenant state uses an explicitly environment-scoped API. | Four callsites checked, one already unscoped (F12). The generic API permits a leak the type system should forbid. |

### 4.1 What D1 + D5 mean together

The data plane is tenant-scoped and shares `main`'s datastores; environment is
carried on the *operation* and never persisted on gameplay rows. The control
plane is environment-scoped and environment *is* persisted there.

This produces a useful property: because the tenant-registry rows are themselves
environment-scoped, **"which environment does tenant T belong to?" is answered by
the row that defines T.** Tenant→environment derivation (FR-7.3) is structural
rather than a side table, and there is no separate "register the ephemeral tenant
with the baseline" step — the baseline registry holds every tenant, each tagged
with its environment.

### 4.2 What D6 means for the sparse floor

A sparse environment always deploys `atlas-login` and `atlas-channel` plus its
changed services. The worst case for a docs-only PR is 2 workloads instead of 64;
a typical single-service PR is 3. This also **simplifies** environment context
origination (input §7.1): game traffic enters through the PR's own socket
services, so environment is established at the edge the client actually connects
to, rather than being injected into a shared entry point.

Design must confirm whether any other service is socket-bound or otherwise
port-registered and therefore belongs in the mandatory floor —
`services` currently recognizes a third type, `drops-service`
(`services/processor.go:59-61`), whose registration semantics must be checked.

---

## 5. User stories

- As an **Atlas developer**, I want my one-service PR to spin up in under a
  minute so that I can iterate without waiting for 64 Deployments.
- As an **Atlas developer**, I want a request I send into my PR environment to
  execute my changed code and the baseline's code for everything else.
- As an **Atlas developer**, I want to connect a game client to my PR
  environment's own login service, as I do today.
- As an **Atlas developer**, I want a monster spawned by `atlas-monsters-main` on
  behalf of my PR to be *my* monster.
- As a **platform operator**, I want to distinguish `atlas-monsters-main`
  processing `main` from the same pod processing `pr-123` in logs and metrics.
- As a **platform operator**, I want a PR's teardown to guarantee no delayed work
  later executes against `main`.
- As a **platform operator**, I want to be certain no PR can modify `main`'s
  configuration or force `main` to redeploy.
- As a **release engineer**, I want changes sparse mode cannot safely validate to
  escalate automatically to a fully-isolated environment.

---

## 6. Functional requirements

### FR-1 — Environment model and registry

- **FR-1.1** An `Environment` custom resource carries: identity, baseline
  reference, service overrides, associated tenant(s), and lifecycle phase.
- **FR-1.2** `Resolve(environment, service) → deployment`; falls back to the
  baseline's deployment when no override exists.
- **FR-1.3** `EnvironmentsOwnedBy(service, deployment) → [environment]`.
- **FR-1.4** `IsOwner(deployment, environment, service) → bool` and
  `IsEnvironmentActive(environment) → bool`.
- **FR-1.5** The baseline must not be hard-coded to `main`; it is a field on the
  resource. A second baseline must require no code change.
- **FR-1.6** All four queries resolve from an in-process cache. No REST, Kafka,
  database, or Kubernetes API call on the resolution path.
- **FR-1.7** Cache staleness has a defined bound; behavior on exceeding it is
  fail-closed (D4).
- **FR-1.8** With no Environment resources present, every query returns the legacy
  answer: the local deployment owns everything. The feature is inert for `main`.

### FR-2 — Execution context

- **FR-2.1** Environment lives in `context.Context` alongside tenant, using the
  same `WithContext` / `FromContext` idiom.
- **FR-2.2** Environment is established at exactly four origination points: a
  socket connection to an override login/channel (D6); an inbound REST call
  carrying the header; an inbound Kafka message carrying the header; and an
  autonomous loop iterating owned environments (FR-6).
- **FR-2.3** An operation reaching domain code without a resolvable environment is
  rejected before the domain call. It does not default to baseline.
- **FR-2.4** Environment context is immutable for the duration of an operation.

### FR-3 — REST

- **FR-3.1** REST calls carry environment as a header alongside the four tenant
  headers, set centrally in `libs/atlas-rest`. No service sets it by hand.
- **FR-3.2** A service handling a request for `pr-123` emits `pr-123` on every
  downstream call, regardless of which deployment it is.
- **FR-3.3** The ingress resolves the upstream from `(environment, service)`.
- **FR-3.4** The ingress uses generated configuration derived from the Environment
  resources; no per-request registry lookup.
- **FR-3.5** If an `ACTIVE` environment declares an override and that override is
  unavailable, the ingress returns an error. It must not fall back to baseline.
- **FR-3.6** A request naming an unknown environment, or one in `DEACTIVATING`
  or `DELETED`, is rejected. A request naming an environment in `PROVISIONING`
  is **admitted**.

  *Amended after implementation.* As originally written this rejected every
  non-`ACTIVE` environment, which made the lifecycle unsatisfiable: an
  environment's own setup writes — atlas-pr-bootstrap's service-config rows —
  happen during `PROVISIONING`, but FR-5.2 flips the phase to `ACTIVE` only
  after that setup completes. The two rules deadlocked, and the first real
  sparse environment could not bootstrap (400 on every service-config POST).
  Admitting `PROVISIONING` resolves it. This weakens the REST gate only;
  ownership and routing are unaffected, because those are governed by
  `Registry.IsOwner`, which still requires `ACTIVE` (FR-5.2). Note that
  in-process confinement to the caller's own rows exists only in
  atlas-configurations and atlas-tenants, which implement the scope layer;
  the gate itself has never provided it.

### FR-4 — Kafka

- **FR-4.1** Messages carry environment as a message header, written and parsed
  centrally, mirroring the tenant treatment (F8).
- **FR-4.2** A consumer processing a message for `pr-123` emits `pr-123` on every
  message it produces.
- **FR-4.3** Override deployments use consumer groups distinct from the baseline
  via the existing `KAFKA_CONSUMER_GROUP` mechanism (F9).
- **FR-4.4** An **ownership gate** runs in consumer infrastructure before the
  domain handler. If this deployment is not the effective implementation for
  `(message.environment, service)`, the message is acknowledged without domain
  processing.
- **FR-4.5** The gate lives in `libs/atlas-kafka`. No domain processor contains
  environment-conditional logic; enforceable by a repo guard.
- **FR-4.6** Exactly one deployment processes any environment-scoped message.
- **FR-4.7** A message whose environment is unknown to the cached registry is not
  processed by any deployment, is acknowledged, and emits a distinct alertable
  metric and log line (D4).
- **FR-4.8** Sparse environments consume the **baseline's** topics — which means
  they must name them the way the baseline names them, suffixed with the
  *baseline's* environment id (`-main` today). Per-environment topic suffixing
  (F4, `-<ATLAS_ENV>`) applies to isolated mode only.

  *Corrected 2026-08-20.* This requirement originally read "the **unsuffixed**
  baseline topics", treating "unsuffixed" as a synonym for "the baseline's".
  It is not: `overlays/main` suffixes all 170 topics with `-main`, so an
  unsuffixed name addresses a topic nobody publishes to. See
  [bug-sparse-baseline-scoping.md](bug-sparse-baseline-scoping.md).
- **FR-4.9** New ephemeral consumer groups must not skip messages by starting at
  `latest`. Design must define offset initialization and prove a message produced
  between group creation and first poll is not lost.
- **FR-4.10** An environment does not become `ACTIVE` until override consumer
  groups exist and their offsets are initialized.

### FR-5 — Lifecycle

- **FR-5.1** Phases: `PROVISIONING → ACTIVE → DEACTIVATING → DELETED`.
- **FR-5.2** During `PROVISIONING`, baseline deployments continue to own the
  environment's services; overrides receive no work; the ingress does not route.
- **FR-5.3** Transition to `ACTIVE` requires every override Deployment ready, every
  override consumer group initialized (FR-4.9), and the mandatory socket services
  (D6) bound and registered.
- **FR-5.4** Activation is atomic with respect to ownership: no observable window
  where both baseline and override own a service, or neither does.
- **FR-5.5** `DEACTIVATING` stops new work being routed or gated to the environment
  *before* any override workload is destroyed.
- **FR-5.6** Design must define drain semantics for in-flight REST requests,
  unacknowledged Kafka messages, open game sockets, and scheduled work.
- **FR-5.7** After `DELETED`, surviving delayed or persisted work for that
  environment is discarded and never executes against the baseline.

### FR-6 — Autonomous processing

- **FR-6.1** Every background loop iterates the environments it owns and executes
  independently within each. The 18 ticker-bearing files (§3.1) are the starting
  inventory; design must confirm completeness.
- **FR-6.2** Autonomously originated work carries the environment of the iteration
  that produced it, on both REST and Kafka.
- **FR-6.3** A baseline deployment must not originate work for an environment that
  overrides its service.
- **FR-6.4** Ownership changes take effect on the next iteration; a loop must not
  cache an ownership set across ticks.
- **FR-6.5** Per-environment execution is isolated for faults and latency.
- **FR-6.6** Iteration cost scales with the number of *owned* environments. A
  deployment owning only `main` performs the same work it does today.

### FR-7 — Tenant and control-plane integration

- **FR-7.1** The ephemeral tenant ↔ environment relationship is **one-to-one** for
  this task. `main` is exempt and may contain many tenants.
- **FR-7.2** An environment's tenant set is recorded on the Environment resource
  and mirrored by the environment column on control-plane tenant rows (D5).
- **FR-7.3** Environment must be derivable from tenant identity, so autonomous
  work starting from persisted tenant-owned state resolves its environment. Per
  §4.1 this is structural: the tenant-registry row carries the environment.
- **FR-7.4** Environment is **not** persisted on data-plane domain objects;
  `tenant_id` remains their only scoping column. Environment **is** persisted on
  control-plane rows.
- **FR-7.5** Control-plane reads are environment-scoped: a service resolving
  templates, service registrations, or the tenant list receives only rows for its
  execution environment.
- **FR-7.6** Tenant lifecycle is bound to environment lifecycle.
- **FR-7.7** A mismatch between the environment header and the environment implied
  by the tenant is a hard error, not a reconciliation.
- **FR-7.8** A PR must be unable to modify any control-plane row belonging to
  another environment, `main` included (G7). This must be enforced, not merely
  conventional.

### FR-8 — Data isolation prerequisites (blocking)

**None of the routing work is safe to enable until FR-8.1–8.4 are complete.**

- **FR-8.1** Every Postgres query path must be tenant-scoped (data plane) or
  environment-scoped (control plane). The audit must be a sweep, not a sample, and
  its result recorded per service.
- **FR-8.2** The control-plane tables identified in F11 —
  `atlas-configurations`' `tenants`, `templates`, `services`, and `atlas-tenants`'
  tenant table — gain an environment dimension, with a migration assigning all
  existing rows to the baseline environment (D5). `main`'s resolution behavior
  must be unchanged (NG6).
- **FR-8.3** Every data-plane Redis callsite migrates to the tenant-scoped API
  (D7). The known-unscoped `atlas-storage` `npc-context` cache (F12) is fixed as
  part of this; note it is a live cross-tenant defect independent of this project.
- **FR-8.4** The non-tenant-scoped Redis constructors are withdrawn from
  data-plane use — removed, unexported, or guarded so a new use fails CI. A
  narrow, explicitly-named environment-scoped API replaces them for legitimate
  cross-tenant control-plane state such as `atlas-data`'s `ingestrun` (F12).
- **FR-8.5** A repo guard fails CI on a new non-environment-scoped control-plane
  entity, a new non-tenant-scoped data-plane entity, or a new bare Redis keyFn.
- **FR-8.6** Any remaining resource whose isolation depends on deployment identity
  is enumerated with a disposition: scope it, or force isolated mode.

### FR-9 — Mode selection and escalation

- **FR-9.1** Two modes: **sparse** (default) and **isolated** (existing).
- **FR-9.2** Affected-service determination must account for direct service
  changes, shared-library changes, deployment/config changes, Kafka contract
  changes, and schema changes. It must be conservative.
- **FR-9.3** The following **force isolated mode** automatically:
  - changes to `deploy/k8s/base` or the ingress configuration;
  - changes to Kafka infrastructure or message contracts;
  - database migrations;
  - changes to `atlas-configurations` or `atlas-tenants` — the control plane
    itself, which sparse environments share;
  - changes to cross-cutting libraries whose blast radius cannot be bounded
    (`libs/atlas-kafka`, `libs/atlas-rest`, `libs/atlas-tenant`, `libs/atlas-redis`,
    and the environment library);
  - any case where affected-service determination is unreliable.
- **FR-9.4** The sparse override set is always the changed services **plus**
  `atlas-login` and `atlas-channel` (D6), plus any other member of the mandatory
  floor design identifies (§4.2).
- **FR-9.5** Mode is overridable per PR by an explicit label, in both directions.
- **FR-9.6** A scheduled full-stack isolated deployment continues to run.
- **FR-9.7** The chosen mode, the override set, and the reason are reported on the
  PR.

### FR-10 — Observability

- **FR-10.1** Environment is a structured-log field emitted by the logging setup,
  not by callers.
- **FR-10.2** Environment appears in Kafka processing logs, REST access logs, and
  error reporting.
- **FR-10.3** Environment is a metric label where cardinality permits; design must
  state the budget and the exclusions.
- **FR-10.4** The ownership gate emits counters for processed, skipped-not-owner,
  and dropped-unresolvable (FR-4.7). The third is alertable.
- **FR-10.5** Operators can filter logs for one shared deployment serving one
  environment. Loki has no `app` label in this cluster — selectors use
  `service_name`.

---

## 7. API surface

### 7.1 REST header

One new header alongside the four tenant headers (F6), following their flat
uppercase-underscore convention so the ingress's `underscores_in_headers on` and
explicit `proxy_set_header` handling apply unchanged. Set only by the central
decorator; read only by the central parser.

### 7.2 Kafka header

One message header mirroring the REST header, written by the producer wrapper and
read alongside `TenantHeaderParser`. Message *bodies* are unchanged — no domain
schema is touched, which is what keeps the migration mechanical across 64
services.

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
    atlas-login:     { deployment: atlas-login-pr-123 }      # mandatory (D6)
    atlas-channel:   { deployment: atlas-channel-pr-123 }    # mandatory (D6)
    atlas-character: { deployment: atlas-character-pr-123 }
status:
  phase: ACTIVE
  overridesReady: true
  consumerGroupsReady: true
  socketsRegistered: true
```

### 7.4 Control-plane API changes

`atlas-configurations`' `services`, `templates` and `tenants` resources become
environment-scoped (D5). Reads are filtered by the caller's execution environment
(FR-7.5); writes are constrained to the caller's own environment (FR-7.8). The
`LoginRestModel` / `ChannelRestModel` tenant-and-port contract (F15) is
**unchanged** — D6 removes the need to extend it.

### 7.5 Error semantics

| Condition | Behavior |
|---|---|
| Unknown / `DEACTIVATING` / `DELETED` environment on a REST request | Reject (FR-3.6) |
| `PROVISIONING` environment on a REST request | Admit — setup writes happen in this phase (FR-3.6) |
| Override declared but unavailable | Error; never fall back (FR-3.5) |
| Unknown environment on a Kafka message | Ack, do not process, alert (FR-4.7) |
| Registry cache stale beyond bound | Fail closed (FR-1.7) |
| Control-plane write targeting another environment | Reject (FR-7.8) |

---

## 8. Data model

### 8.1 Data plane — no new columns

Environment is not persisted on gameplay objects (FR-7.4); it is derived from
tenant (FR-7.3). This keeps the change out of 84 entity definitions.

### 8.2 Control plane — new environment dimension

| Table | Service | Current key | Required |
|---|---|---|---|
| `tenants` | atlas-configurations | `(Id, Region, Major, Minor)` | + environment |
| `templates` | atlas-configurations | `(Region, Major, Minor)` | + environment |
| `services` | atlas-configurations | `Type` | + environment |
| tenant table | atlas-tenants | tenant id | + environment |

Each needs a backfill assigning existing rows to the baseline environment. All
are read on hot configuration paths, so design must confirm `main`'s resolution
behavior is byte-for-byte unchanged (NG6).

Note the uniqueness consequence: `services` is currently keyed by `Type`, which
implies at most one login-service registration. With an environment dimension the
key becomes `(Type, environment)` — which is precisely what lets `pr-123` have
its own login registration without touching `main`'s (D6, G7).

### 8.3 Tenant registration

Because control-plane tenant rows are environment-scoped (§4.1), provisioning an
ephemeral tenant is an insert into the shared registry tagged with its
environment, and teardown is its deletion. Ordering against FR-5 is a design
obligation: a tenant visible before activation would be picked up prematurely by
baseline autonomous loops (F13).

---

## 9. Service impact

### 9.1 Shared libraries — where the work concentrates

| Library | Change |
|---|---|
| `libs/atlas-tenant` (or a new environment lib) | environment context type, `WithContext` / `FromContext` |
| `libs/atlas-rest` | header decorator + inbound parser (FR-3.1, FR-3.2) |
| `libs/atlas-kafka` | header write/parse, **ownership gate** (FR-4.4); group naming already sufficient (F9) |
| new environment registry lib | CRD watch, in-memory cache, the four queries (FR-1) |
| `libs/atlas-redis` | withdraw the bare constructors, migrate data-plane callsites, add an environment-scoped API for control-plane state, make the `ATLAS_ENV` prefix inert in sparse mode (D7, FR-8.3/8.4) |
| `libs/atlas-routine` | per-environment iteration helper (FR-6) |

A typical service should change **only its `main.go` wiring and its background
loops**. If a domain package needs editing, the abstraction is in the wrong place
(NG5).

### 9.2 Services needing individual attention

- **`atlas-configurations`** — the largest single change: three tables gain an
  environment dimension, with migrations and environment-filtered reads/writes
  (D5, FR-8.2, FR-7.5, FR-7.8). Blocking.
- **`atlas-tenants`** — environment dimension on the tenant table; `GetAll()`
  semantics now environment-filtered (F13).
- **`atlas-login` / `atlas-channel`** — mandatory overrides (D6). Their existing
  `SERVICE_ID`-plus-tenant-ports contract is unchanged, but each sparse
  environment needs its own service registration row and its own port/IP
  allocation, and they are the origination point for environment context on game
  traffic (FR-2.2). Their templated consumer groups already pass variadic args to
  `consumergroup.Resolve` (F9); the interaction with per-environment group naming
  needs explicit design.
- **`atlas-storage`** — fix the unscoped `npc-context` cache (F12). This is a live
  cross-tenant defect today, independent of this project.
- **`atlas-monsters`, `atlas-summons`, `atlas-doors`, `atlas-npc-shops`,
  `atlas-data`** — Redis migration to the tenant-scoped or environment-scoped API
  (FR-8.3/8.4).
- **`atlas-saga-orchestrator`** — saga continuation must restore environment across
  steps; it is a persisted-work path with a ticker.
- **The 18 ticker-bearing files** (§3.1) — per-environment iteration (FR-6.1).

### 9.3 Deployment and CI

- A sparse overlay producing only the override set plus the Environment resource.
- Port/IP allocation for each sparse environment's login and channel services.
- `.github/workflows/pr-validation.yml` gains affected-service determination and
  mode selection (FR-9).
- `cleanup.sh` gains deactivate-before-destroy ordering (FR-5.5), tenant
  deregistration, and control-plane row cleanup.
- Under D1, per-env DB creation and topic pre-creation become no-ops for sparse
  environments but must remain for isolated mode.

---

## 10. Non-functional requirements

- **NFR-1 (Performance).** Environment resolution on the hot path is an in-memory
  map lookup. No I/O (FR-1.6, NG4).
- **NFR-2 (Overhead).** For a deployment owning only `main`, added per-operation
  cost is negligible and no extra background work is performed (FR-6.6).
- **NFR-3 (Capacity).** A sparse environment's incremental footprint is the
  mandatory floor plus its overrides. Several concurrent sparse environments cost
  a small multiple of that, not a multiple of 64.
- **NFR-4 (Startup).** A sparse environment reaches `ACTIVE` substantially faster
  than the isolated equivalent. The ~60s baseline restore (§3.8) leaves the
  critical path when the environment shares `main`'s data.
- **NFR-5 (Correctness).** Cross-environment leakage and duplicate processing are
  P0. Tests must assert the *new* contract explicitly — per CLAUDE.md, existing
  tests may currently pin the old silent-drop behavior.
- **NFR-6 (Multi-tenancy).** Environment composes with tenancy; it does not
  replace or bypass it (NG7).
- **NFR-7 (Backward compatibility).** With no Environment resources present,
  behavior is byte-for-byte today's behavior (FR-1.8, NG6).
- **NFR-8 (Security).** Under D1, ephemeral tenants share `main`'s datastores. No
  ephemeral tenant may read or write another tenant's data through any endpoint —
  the tenant boundary now carries weight the database boundary used to. Under D5,
  no environment may read or write another environment's control-plane rows.

---

## 11. Open questions

1. **Header spelling and precedence.** Which name, and what happens when the
   environment header and the tenant-derived environment disagree (FR-7.7)?
2. **Consumer offset initialization.** What mechanism satisfies FR-4.9, and how is
   "initialized" observed for the activation gate (FR-5.3)?
3. **Activation atomicity.** How is FR-5.4 achieved given registry caches across
   many pods update independently? Is a two-phase ownership handover needed?
4. **Staleness bound.** The numeric bound in FR-1.7 and the behavior at it.
5. **nginx configuration distribution.** Generated ConfigMap plus reload, or
   `njs`-based map lookup? ~60s ConfigMap propagation interacts badly with FR-5.4.
6. **Drain semantics** (FR-5.6) — in particular an unacknowledged Kafka message
   and an open game socket belonging to a `DEACTIVATING` environment.
7. **Control-plane scoping shape.** Does an environment column on the existing
   tables suffice, or is a per-environment overlay-with-baseline-fallback better?
   Overlay-with-fallback would let a sparse environment inherit `main`'s templates
   without copying them, at the cost of a more complex read path.
8. **Port/IP allocation for mandatory socket overrides.** Each sparse environment
   needs unique login and channel ports. How are they allocated, advertised in
   `ChannelTenantRestModel.IPAddress`, and reclaimed? The existing PR overlay has
   an `lb-allocate` patch that may already solve this.
9. **Is `drops-service` socket-bound?** It is the third registered `ServiceType`
   (`services/processor.go:59-61`); if it is port-bound it joins the mandatory
   floor (§4.2).
10. **Metrics cardinality budget** (FR-10.3).
11. **Sequencing of the 64-service migration.** Is there an intermediate state
    where some services are environment-aware and others are not that is *not*
    dangerous?
12. **Does the shared control plane defeat the purpose for some PRs?** Any PR
    touching `atlas-configurations` or `atlas-tenants` escalates to isolated
    (FR-9.3). How common is that in practice?

---

## 12. Acceptance criteria

### Prerequisites (must be complete before routing is enabled)

- [ ] Every service's Postgres access is verified tenant-scoped (data plane) or
      environment-scoped (control plane), recorded per service (FR-8.1).
- [ ] The four control-plane tables carry an environment dimension, migrated with
      `main`'s resolution behavior unchanged (FR-8.2).
- [ ] Every data-plane Redis callsite uses the tenant-scoped API; the
      `atlas-storage` `npc-context` defect is fixed (FR-8.3).
- [ ] The bare Redis constructors are withdrawn from data-plane use and a narrow
      environment-scoped API serves legitimate cross-tenant state (FR-8.4).
- [ ] Repo guards fail CI on new unscoped entities or bare Redis keyFns (FR-8.5).
- [ ] Every remaining deployment-scoped resource has a disposition (FR-8.6).

### Core capability

- [ ] The Environment CRD, controller, and registry library exist; all four
      queries resolve from an in-memory cache with no I/O.
- [ ] Environment propagates over REST and Kafka via central plumbing; no domain
      processor references environment (FR-4.5, guard-enforced).
- [ ] The ingress routes on `(environment, service)` with no per-request registry
      call, and fails rather than falling back when an override is down.
- [ ] The Kafka ownership gate guarantees exactly-one-processor and drops + alerts
      on unresolvable environments.
- [ ] All 64 Deployments are environment-aware (D3).
- [ ] All 18 ticker-bearing background loops iterate owned environments.

### The §17 proof — environment ≠ deployment

- [ ] Given `pr-123` overriding `atlas-character` (plus the mandatory login and
      channel), **only those workloads are additionally deployed.** The remaining
      61 are not.
- [ ] A game client connects to `pr-123`'s own login service and its session
      remains in `environment=pr-123`, `tenant=ephemeral-T123` throughout.
- [ ] A request entering `pr-123` executes through a mix of override and baseline
      deployments and never leaves `pr-123`.
- [ ] The equivalent `main` request is unaffected and behaviorally unchanged.
- [ ] **A baseline deployment originates autonomous work separately for `main` and
      for `pr-123`**, and the `pr-123` work reaches the `pr-123` override.
- [ ] A baseline deployment originates **no** autonomous work for an environment
      that overrides its service (FR-6.3).

### Isolation guarantees

- [ ] `main`'s configuration is not mutated and `main` does not redeploy at any
      point in a sparse environment's lifecycle (G7, FR-7.8).
- [ ] A PR cannot read or write another environment's control-plane rows.
- [ ] A PR cannot read or write another tenant's data-plane rows.

### Lifecycle and teardown

- [ ] An environment becomes `ACTIVE` only when overrides, consumer groups, and
      socket registrations are all ready, with no window of double or zero
      ownership.
- [ ] Teardown deactivates routing before destroying workloads, and reclaims
      control-plane rows and allocated ports.
- [ ] After deletion, no delayed, scheduled, or in-flight work executes against
      `main`. Verified by test, not inspection.

### Mode selection

- [ ] Affected-service determination is conservative and covers direct, library,
      deployment, contract, and schema changes.
- [ ] Every FR-9.3 condition escalates to isolated mode automatically, including
      control-plane changes.
- [ ] The override set always includes the mandatory floor (FR-9.4).
- [ ] Isolated mode still works, unchanged; the scheduled full-stack deployment
      still runs.

### Verification

- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] Tests assert the new contract including negative cases — not-owner skip,
      unresolvable drop, no-fallback-on-override-down, cross-environment
      control-plane write rejection (NFR-5).
- [ ] Multiple concurrent sparse environments coexist with resource use
      proportional to their override sets (NFR-3).
