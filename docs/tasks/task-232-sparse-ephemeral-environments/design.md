# Sparse Ephemeral Environments — Design

Status: Draft for review
Created: 2026-08-15
Inputs: [prd.md](prd.md) (approved, v2), [isolation-audit.md](isolation-audit.md)
Verified against: worktree `task-232-sparse-ephemeral-environments`, branch point `c8d44127c`

---

## 0. What this document decides

The PRD fixed the requirements and seven decisions (D1–D7). This document
picks the mechanisms, answers the twelve open questions with evidence from the
repository, and records four places where the verified code contradicts a PRD
premise strongly enough to change the design.

Every file:line citation below was read during this design pass. Where a claim
is a hypothesis rather than a reading, it says so.

The design rests on one structural discovery that the PRD did not have:

> **Every inter-service REST call in Atlas already goes through an ingress
> whose address is a single environment variable, and every environment
> already deploys its own ingress.**
>
> `BASE_SERVICE_URL` is `http://atlas-ingress.atlas-main.svc.cluster.local:80/api/`
> in `deploy/k8s/overlays/main/kustomization.yaml:39` and
> `http://atlas-ingress.atlas-pr-<N>.svc.cluster.local:80/api/` in
> `deploy/k8s/overlays/pr/kustomization.yaml:160`. `libs/atlas-rest/requests/url.go`
> resolves every outbound call to that value (or a per-domain override).

That means REST routing does not need a service mesh, an `njs` map, a
regenerated ConfigMap, or a per-request registry lookup. It needs the
originating process to pick *which ingress* to send to, based on the
environment on the operation — a string format, in memory, with no I/O. The
routing table inside each environment's ingress stays static for that
environment's whole life. §4 develops this.

---

## 1. Corrections to the PRD's picture

These are readings, not opinions. Each changes something downstream.

### C1 — `services` is keyed by `Id`, not by `Type`

PRD §8.2 and isolation-audit §3 state the `services` table is keyed by `Type`,
and derive from that a need to change the key to `(Type, environment)`.

The entity is:

```go
type Entity struct {
    Id   uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
    Type ServiceType     `gorm:"type:varchar;not null"`
    Data json.RawMessage `gorm:"type:json;not null"`
}
```
— `services/atlas-configurations/atlas.com/configurations/services/entity.go`

and consumers select their row by UUID from a `SERVICE_ID` environment
variable (`services/atlas-login/atlas.com/login/main.go:54`,
`services/atlas-channel/atlas.com/channel/main.go:176`,
`services/atlas-drops/atlas.com/drops/main.go:61`).

Multiple rows of the same `Type` are already representable. **No key change is
required.** A sparse environment gets its own rows with fresh UUIDs and its
own `SERVICE_ID` values; `main`'s row is never read, written, or merged into.
The `environment` column is still added — but for teardown scoping, listing,
and write authorisation (FR-7.8), not for resolution.

This *strengthens* D6 rather than weakening it: the isolation D6 wants is
already available at the row level.

### C2 — the PR bootstrap currently merges into a **pinned** service UUID

`services/atlas-pr-bootstrap/scripts/bootstrap.sh:303-320` and
`scripts/service-config.sh` implement `upsert_service_config`: it reads the
live row for a canonical, *pinned* `SERVICE_ID`, merges this PR's tenant entry
into its `tenants[]` array id-keyed, and writes the merged result back. Today
that is safe because each PR has its own `atlas-configurations` database.

Under D1 (shared control-plane database) this exact code would merge the PR's
tenant and port into **`main`'s** service row — the NG6/G7 violation D6 was
written to prevent, arriving through the bootstrap script rather than through
service code. The sparse bootstrap must therefore **create rows, not merge
entries**. This is the single most dangerous line of existing code in the
migration and gets its own plan task and its own negative test.

### C3 — `drops-service` is not socket-bound; the mandatory floor stays at two

Open question 9. `ServiceTypeDrops` decodes to `service.GenericRestModel`
(`services/processor.go:128-136`), whose only payload is
`Tasks []task.RestModel` — `{Type, Interval, Duration}`
(`services/atlas-configurations/atlas.com/configurations/services/task/rest.go`).
No tenants, no ports, no advertised IP. `atlas-drops` reads it once at startup
for timer configuration.

**`drops-service` does not join the mandatory floor.** The floor is exactly
`atlas-login` + `atlas-channel`, as D6 states. `atlas-drops` does need an
environment-scoped service row so it reads *its* environment's task intervals,
but it is an ordinary override candidate.

### C4 — background loops cache the tenant list at process start

`services/atlas-transports/atlas.com/transports/main.go:93-151` — the pattern
F13 cites — loads `tenants` once before the ticker and the ticker goroutine
closes over that slice. A tenant created after the pod started is invisible
until restart.

This directly blocks the project: a baseline pod must pick up a newly
provisioned ephemeral tenant **without redeploying `main`** (G7, NG6). FR-6.4
already forbids caching ownership across ticks; this finding says the
violation is not hypothetical — it is the current implementation in the file
the PRD used as the reference pattern. Every loop in the §3.1 inventory must
be checked for it, not just refactored around it.

---

## 2. Architecture in one page

Five mechanisms. Each is independently testable and none requires
environment-conditional logic in a domain processor (NG5).

| # | Mechanism | Where | Answers |
|---|---|---|---|
| **M1** | `environment` on `context.Context`, a string, same idiom as tenant | new `libs/atlas-env` | FR-2 |
| **M2** | Environment registry projected from a compacted Kafka topic into memory, exactly like the existing configuration projection | new `libs/atlas-env` + new `atlas-environments` service | FR-1 |
| **M3** | Outbound REST targets *the ingress of the operation's environment*; each ingress has a static per-service upstream table | `libs/atlas-rest/requests/url.go` + `deploy/k8s/base/atlas-ingress.yaml` | FR-3 |
| **M4** | Kafka carries `ENVIRONMENT` as a header; an ownership gate in `Consumer.processMessage` decides process-or-ack | `libs/atlas-kafka` | FR-4 |
| **M5** | Background loops iterate `(environment, tenant)` pairs resolved fresh each tick, from the registry | `libs/atlas-service` helper | FR-6 |

Data flow for one request entering `pr-123`:

```
game client ──socket──▶ atlas-login-pr-123          (environment = pr-123, from ENV var)
                              │
                              │ REST, env on context
                              ▼
             BASE_SERVICE_URL for env pr-123  =  atlas-ingress.atlas-pr-123
                              │
              ┌───────────────┴───────────────┐
              │ static routes.conf per service │
              ▼                                ▼
   atlas-character.atlas-pr-123     atlas-monsters.atlas-main   (override)   (baseline)
              │                                │
              │ ENVIRONMENT: pr-123 header travels on every hop
              ▼                                ▼
   Kafka produce (header)            Kafka produce (header)
              │
              ▼
   every consumer of the topic runs the ownership gate:
     effective impl for (pr-123, atlas-inventory)? ─ yes ▶ handle
                                                    ─ no  ▶ ack, count, drop
```

The baseline pod `atlas-monsters-main` never learns it is "serving a PR". It
reads the environment off the operation and hands it back out. That is the
PRD's central claim — environment is a property of the operation — expressed
as five pieces of plumbing.

---

## 3. M1 — environment on the context

New module `libs/atlas-env`, package `env`. Deliberately **not** folded into
`libs/atlas-tenant`: the registry (M2) needs a Kafka consumer, and
`libs/atlas-tenant` is imported by `libs/atlas-kafka` — folding it in creates
an import cycle.

```go
package env

const Key = "ENVIRONMENT"   // context key AND REST/Kafka header name

type Id string               // "main", "pr-123"

func WithContext(ctx context.Context, id Id) context.Context
func FromContext(ctx context.Context) model.Provider[Id]
func MustFromContext(ctx context.Context) Id
func Self() Id               // this process's own environment, from ATLAS_ENVIRONMENT
```

Three deliberate choices:

- **A plain string, not a struct.** Tenant is a four-field model because all
  four fields are semantically load-bearing. Environment is one identifier;
  a struct would invite fields (baseline, phase) that belong in the registry,
  not on every operation.
- **`Key` is both the context key and the header name**, matching
  `libs/atlas-tenant/processor.go:12-17` where `ID = "TENANT_ID"` serves both.
- **`Self()` reads an env var and never fails.** A pod always knows its own
  environment even when the registry is unreachable. This is what keeps `main`
  fully functional during a registry outage (NFR-7).

`env.Id("")` is the legacy value and means "not environment-aware". FR-1.8's
inertness falls out: an empty environment resolves to the local deployment
for every query.

---

## 4. M2 — the registry

### 4.1 Three approaches

**(a) Environment CRD watched by every service (PRD D2).**
Argo CD reconciles a CRD; each of 64 services runs a client-go informer.
Push updates, sub-second, clear staleness signal from the watch connection.
Cost: client-go in 64 `go.mod` files, a ServiceAccount + Role + RoleBinding
per service, RBAC to read across namespaces, and a new failure mode (API
server pressure from 64×N informers) that Atlas has never had. No existing
Atlas service talks to the Kubernetes API.

**(b) ConfigMap mounted as a file, polled.**
No RBAC, no client-go, trivially testable. But kubelet propagation is up to
~60s with no way to distinguish a stale file from a current one — so FR-1.7's
staleness bound cannot be *observed*, only assumed. Fails D4 in the one
direction that matters.

**(c) Compacted Kafka topic projected into memory — RECOMMENDED.**
A small `atlas-environments` service owns environment records and publishes
them to a compacted topic. Every service consumes that topic under a
per-process consumer group replaying from `FirstOffset`, projecting into an
in-memory map, and gates readiness on catch-up.

(c) wins because **this exact machinery already exists and is already wired
into the fleet-canonical bootstrap.** `libs/atlas-service/projection.go`
implements it for configuration:

```go
// Per-process group id so each container start replays the full
// compacted log from FirstOffset; a shared group would resume from the
// previous run's committed offset and leave the in-memory State empty.
groupId := fmt.Sprintf("%s - projection - %s", pc.baseGroupId, uuid.New().String())
```
— `libs/atlas-service/projection.go:74-77`

with a catch-up gate (`AwaitProjectionCatchUp`, line 89) and a readiness gate
(`WithReadinessGate`, `bootstrap.go:30`). The environment registry is a second
projection alongside the first: same topic mechanics, same catch-up gate, same
readiness composition, zero new infrastructure concepts, and no new dependency
in any service's `go.mod` beyond `libs/atlas-env`.

It also gives what (b) cannot: an **observable** staleness bound. See §4.3.

**Deviation from D2 is explicit and is the largest in this document.** The
registry is a Kafka projection, not a CRD watch. The Environment *resource*
still exists as a declarative YAML artifact in the overlay (§4.4) — what
changes is how 64 pods learn about it.

### 4.2 The four queries

All four resolve from the projected map. No I/O (FR-1.6, NG4).

```go
type Registry interface {
    Resolve(e env.Id, service string) (deployment string, err error)   // FR-1.2
    EnvironmentsOwnedBy(service string) []env.Id                       // FR-1.3
    IsOwner(e env.Id, service string) bool                             // FR-1.4
    IsActive(e env.Id) bool                                            // FR-1.4
    Stale() bool                                                       // FR-1.7
}
```

`Resolve` is a two-level map lookup: `overrides[e][service]`, falling back to
`baselines[e]`'s deployment. `EnvironmentsOwnedBy` drops the `deployment`
parameter from FR-1.3 — a process only ever asks about itself, and `env.Self()`
plus `SERVICE_NAME` already identify it. Passing a deployment you are not
would be a bug with no legitimate caller.

The baseline is a field on each environment record, never the literal `"main"`
(FR-1.5).

### 4.3 Staleness (open question 4)

`atlas-environments` publishes a heartbeat record to the compacted topic every
**30 s**. A process whose most recently observed registry message is older
than **120 s** (4 missed heartbeats) sets `Stale() == true`.

Behaviour when stale, which is the part that matters:

| Operation | Stale behaviour |
|---|---|
| Environment == `env.Self()` | **Proceed.** A pod's own environment comes from an env var and cannot go stale. |
| Environment != `env.Self()`, inbound | **Fail closed** — reject (REST) / ack-and-drop with the alertable counter (Kafka). |
| Autonomous iteration | **Only `env.Self()`** is iterated. |

The consequence is the important one: a registry outage degrades a
*baseline* pod to exactly its pre-project behaviour — it serves `main` and
nothing else — rather than taking it down. NFR-7 holds under failure, not
just under absence.

### 4.4 The environment record

Declarative YAML committed in the overlay, applied by Argo CD, read by
`atlas-environments` and published to the topic. Shape is the PRD's §7.3 with
`tenant` promoted to the authoritative field:

```yaml
name: pr-123
baseline: main
namespace: atlas-pr-123
tenant: <ephemeral tenant uuid>
overrides:
  atlas-login:     atlas-pr-123
  atlas-channel:   atlas-pr-123
  atlas-character: atlas-pr-123
phase: ACTIVE
```

`overrides` maps service name → **namespace**, not to a Deployment name.
Deployment names are identical across namespaces in Atlas (`atlas-character`
everywhere); the namespace is what actually varies and what M3 needs.

---

## 5. M3 — REST routing

### 5.1 Outbound: pick the environment's ingress

`libs/atlas-rest/requests/url.go` gains an environment-aware form. `RootUrl`
today:

```go
func RootUrl(domain string) string {
    if val, ok := os.LookupEnv(strings.ToUpper(domain) + ServiceSuffix); ok {
        return val
    }
    return os.Getenv(BaseService)
}
```

The environment-aware resolution substitutes the namespace of the operation's
environment into the base URL. Since every service constructs request URLs
through `requests.RootUrl(...)` and every request goes through
`requests.GetRequest`/`PostRequest`/… which already take a `context.Context`,
the environment is available at the point of resolution without touching a
single service's client code beyond threading the same `ctx` it already
threads.

Resolution is `fmt.Sprintf` over a namespace read from the in-memory registry.
No I/O (NFR-1). An unknown or inactive environment returns an error before the
call is made (FR-3.6). An environment whose namespace is gone yields a
connection error, never a baseline call (FR-3.5, G4).

**This is what closes the leak the PRD identified but did not localise.** A
baseline pod serving `pr-123` has `BASE_SERVICE_URL` pointing at
`atlas-main`'s ingress. Without M3 its downstream call lands in `main` and the
operation silently changes environment mid-flight. With M3 it targets
`atlas-ingress.atlas-pr-123`.

### 5.2 Inbound: one header, one central parser

`ENVIRONMENT`, flat uppercase-underscore, alongside the four tenant headers
(open question 1). Set only by a new `EnvHeaderDecorator` in
`libs/atlas-rest/requests/header.go` beside `TenantHeaderDecorator`
(`header.go:29-43`); read only by a new `ParseEnvironment` in
`libs/atlas-rest/server/handler.go` beside `ParseTenant`
(`handler.go:34-100`), composed in the four `Register*Handler` functions in
`libs/atlas-rest/server/register.go`.

That file is the whole inbound surface — four functions, every service's every
route. `RegisterSimpleHandler` and `RegisterSimpleInputHandler` (the
tenant-free control-plane routes, e.g. `atlas-drops` fetching its own service
config with a bare `rt.Context()`) get environment parsing but not tenant
parsing. For those routes the header is the *only* source of environment,
which is why the header exists at all rather than deriving environment from
tenant everywhere.

nginx needs one line added to `deploy/k8s/base/atlas-ingress.yaml:38-41`:

```
proxy_set_header ENVIRONMENT $http_environment;
```

`underscores_in_headers on` is already set (line 27).

### 5.3 The ingress upstream table

`deploy/k8s/base/routes.conf.template.generated` is 114 `location` blocks of
the form:

```
location ~ ^/api/parties(/.*)?$ {
  set $u "atlas-parties.${POD_NAMESPACE}.svc.cluster.local:8080";
  proxy_pass http://$u$request_uri;
}
```

`${POD_NAMESPACE}` is substituted by nginx's envsubst at container start,
restricted by `NGINX_ENVSUBST_FILTER: POD_NAMESPACE`
(`atlas-ingress.yaml:88-89`).

`tools/gen-routes.sh` changes to emit a **per-service** variable:

```
set $u "atlas-parties.${NS_PARTIES}.svc.cluster.local:8080";
```

and the ingress Deployment gains one `NS_<SERVICE>` env var per service, with
`NGINX_ENVSUBST_FILTER` widened to the `NS_` prefix. In `main` and in isolated
mode every `NS_*` is `$(POD_NAMESPACE)` — byte-identical behaviour (NFR-7). In
a sparse environment the overlay sets overridden services to the PR namespace
and everything else to `atlas-main`.

Consequences, all good:

- The routing table is **static for an environment's entire life**. The
  override set is fixed when the environment is created. There is no reload,
  no ConfigMap propagation delay, and open question 5 dissolves — neither a
  regenerated ConfigMap nor `njs` is needed.
- No per-request registry lookup (FR-3.4) — the lookup happened at
  `kustomize build` time.
- An override that is down produces a 502 from its own environment's ingress.
  That is an error, not a fallback (FR-3.5).

---

## 6. M4 — Kafka

### 6.1 Header

Mirrors §5.2: an `EnvHeaderDecorator` in `libs/atlas-kafka/producer/header.go`
wired into `ProviderImpl` (`producer/provider.go:14-22`), which is the
canonical producer used at **211** call sites; and an `EnvHeaderParser` in
`libs/atlas-kafka/consumer/header.go` beside `TenantHeaderParser`.

Four call sites bypass `ProviderImpl` and call `producer.Produce` directly:

- `services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go:97`
- `services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go:28`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go:103`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go:175`

All four already pass `(sd, td)` decorators explicitly and simply need the
third. A repo guard (§10) then forbids new direct `producer.Produce` calls, so
the set cannot grow.

Message **bodies** are untouched. No domain schema changes anywhere — this is
what keeps the 64-service migration mechanical.

### 6.2 The ownership gate

One choke point:

```go
func (c *Consumer) processMessage(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) bool {
    wctx := ctx
    for _, p := range c.headerParsers {
        wctx = p(wctx, msg.Headers)
    }
    ...
```
— `libs/atlas-kafka/consumer/manager.go:611-615`

`processMessage` is called from all three consumption paths
(`engine_reader.go:125,231`, `partition.go:200,220`) and is the only place a
domain handler is reached. The gate goes immediately after header parsing and
before the tracing span:

```
env := env.FromContext(wctx)          // empty ⇒ legacy ⇒ process (FR-1.8)
if !registry.IsActive(env)      → ack, drop, alertable counter   (FR-4.7)
if !registry.IsOwner(env, self) → ack, skip, skipped counter     (FR-4.4)
                                → process                        (FR-4.6)
```

Returning `true` (success) on both drop paths is deliberate: the message is
acknowledged and the offset advances. A drop is not a failure — a failure
would block the prefix-commit cursor (`consumer/cursor.go`) and wedge the
partition.

Exactly-one-processor (FR-4.6) follows from `IsOwner` being a pure function of
the registry: for a given `(environment, service)` exactly one deployment
satisfies it, and every pod projects the same log.

### 6.3 Consumer groups and offsets (open questions 2 and FR-4.9)

Override deployments already get distinct group IDs — `KAFKA_CONSUMER_GROUP`
is resolved at runtime by `libs/atlas-kafka/consumergroup/resolver.go:38-50`
and patched per-PR by `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`
(F9). Nothing changes there.

What changes is that sparse environments consume the **unsuffixed baseline
topics** (FR-4.8), so a new group on a long-lived topic has a real offset
problem:

- `kafka.FirstOffset` is the library default
  (`libs/atlas-kafka/consumer/config.go:37`). A new override group would
  replay the entire retained history of `main`'s traffic and gate-drop
  substantially all of it — correct, but a startup cost measured in the
  topic's whole retention window.
- `LastOffset` loses anything produced between group creation and first poll —
  precisely FR-4.9's stated worry.

**Recommendation: pre-seed committed offsets at provisioning time.** Extend
the existing sync-wave-0 Job `deploy/k8s/base/atlas-kafka-precreate.yaml`
(which already enumerates every `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` from the
`atlas-env` ConfigMap and runs `kafka-topics.sh`) to also run
`kafka-consumer-groups.sh --reset-offsets --to-latest --execute` for each
override group on each topic, while the group is empty and therefore
resettable.

This is provably lossless, not merely likely-lossless:

1. Offsets are committed at end-of-log at instant *T*.
2. Committed offsets take precedence over `startOffset`, so the replay
   question disappears.
3. No message for this environment can exist before *T*, because the
   environment's tenant does not exist and its ingress does not route until
   activation, which is strictly after *T*.
4. Every message after *T* is consumed.

It also gives FR-5.3 an **observable** readiness signal: the group exists and
has a committed offset on every partition of every subscribed topic. That is a
`kafka-consumer-groups.sh --describe` check, not an inference.

Argo CD already health-checks this Job and holds Deployments (sync-wave 10)
until it completes, so the ordering is enforced by existing machinery.

---

## 7. M5 — autonomous work, and the activation model

### 7.1 The iteration helper

Every loop becomes: for each environment I own, for each tenant in that
environment, do the work. Both dimensions resolved **fresh each tick**
(FR-6.4) — which is exactly what C4 says the current code does not do.

The helper lives in `libs/atlas-service` beside `Bootstrap`, so the wiring is
one call in `main.go` and no domain package is touched (NG5):

```go
service.ForEachOwnedEnvironment(l, rt, "atlas-transports", func(ctx context.Context) {
    // ctx already carries environment AND tenant
})
```

Semantics:

- Owned environments come from `registry.EnvironmentsOwnedBy(service)`,
  re-read per tick. A baseline that owns only `main` gets exactly one
  iteration and performs exactly today's work (FR-6.6, NFR-2).
- Tenants are filtered to the environment, from the tenant registry
  (§8.2) — also per tick.
- Each `(environment, tenant)` body runs under its own panic recovery via
  `routine.Go` (`libs/atlas-routine/routine.go:15`), so one environment's
  fault does not stop another's (FR-6.5).
- A tenant whose environment is not in the registry is **skipped**, not
  treated as baseline (D4). §7.3 explains why this specific rule is what makes
  activation safe.

### 7.2 Triage of the 18 ticker files

The §3.1 inventory is not homogeneous. Three classes, each with a different
disposition; the plan must classify all 18 explicitly rather than mechanically
wrapping them:

| Class | Files (from `grep -rl NewTicker services`) | Disposition |
|---|---|---|
| **Per-tenant domain work** | `atlas-transports/…/main.go`, `atlas-marriages/…/scheduler/ceremony_timeout.go`, `…/proposal_expiry.go`, `atlas-asset-expiration/…/task/periodic.go`, `atlas-mts/…/task/periodic.go`, `atlas-party-quests/…/main.go`, `atlas-saga-orchestrator/…/main.go`, `atlas-login/…/main.go` | Wrap in `ForEachOwnedEnvironment` |
| **Process-local caches / watchdogs** (no tenant loop, no emitted work) | `atlas-cashshop/…/reservation/cache.go`, `atlas-channel/…/chakra/registry.go`, `atlas-channel/…/live_mirror.go`, `atlas-channel/…/npc/shop/consumer.go`, `atlas-data/…/ingest/heartbeat.go`, `atlas-data/…/rest/watchdog.go` | No change; record the reasoning |
| **Control-plane projections** | `atlas-channel/…/configuration/projection/loop.go`, `atlas-login/…/configuration/projection/loop.go`, `atlas-character-factory/…/configuration/bridge.go`, `atlas-world/…/configuration/bridge.go` | Environment-scoped, not tenant-scoped (D5) |

The class-2 verdicts are the ones that need evidence in the plan: "it looks
local" is not a disposition. Each needs a citation showing it neither reads
tenant-scoped state nor emits a message.

### 7.3 Activation atomicity (open question 3)

FR-5.4 asks for no window in which both baseline and override own a service,
or neither does. Achieving that literally would require distributed consensus
across 64 pods' independent projection caches. It is not necessary, because
of an asymmetry:

> **Before an environment is ACTIVE, no operation can carry its environment.**
> The tenant does not exist, no ingress routes to it, and no socket accepts
> for it. There is no work to own.

So the real invariant is:

> **At most one deployment ever owns `(environment, service)`, and zero owners
> occurs only during windows in which no operation for that environment can
> exist.**

That is strictly weaker than FR-5.4 as written and strictly sufficient for
G4/NFR-5. It is achievable with the projection model as designed. **This is a
deliberate refinement of FR-5.4 and is flagged in §11.**

The mechanism that makes it hold is the skip-on-unknown-environment rule in
§7.1, and one ordering constraint:

- **Activation order:** environment record published → override pods ready →
  consumer offsets seeded → tenant row created → ingress route enabled →
  phase `ACTIVE`.
- If a baseline pod sees the tenant before the environment (possible: they
  travel on different topic keys and therefore different partitions), it finds
  an unknown environment and **skips**. It never claims ownership by default.
  The window self-heals as the environment record arrives.
- Once the environment record is visible, `IsOwner` returns false for the
  overridden service on that baseline pod, permanently and consistently.

Per §4.1 of the PRD the tenant row itself carries its environment, so the
tenant is never ambiguous — only possibly-early.

### 7.4 Deactivation and drain (open question 6, FR-5.5–5.7)

Deactivation runs the activation order in reverse, and its governing rule is
that work is **discarded, never handed back**:

| Resource | Drain semantics |
|---|---|
| In-flight REST | The environment's own ingress stops routing. Established connections complete under nginx's existing `proxy_read_timeout 30` (`atlas-ingress.yaml:22`). Bounded, no handover. |
| Unacknowledged Kafka | Override pods terminate; the override consumer group is deleted. Uncommitted messages are never consumed by anyone: baseline pods see an environment that is no longer in the registry and ack-and-drop (FR-4.7). |
| Open game sockets | Override login/channel pods terminate; clients disconnect. No baseline handover is possible — the tenant is gone. |
| Scheduled / persisted work | §7.5. This is the one that needs work. |

The Kafka row of that table is worth stating plainly: FR-5.7 ("no delayed work
executes against the baseline") is satisfied *by the gate's drop path*, not by
draining. Teardown does not have to be complete to be correct.

### 7.5 The teardown cost of D1 — stated honestly

Today teardown drops the PR's databases (`do_drop_dbs` in
`services/atlas-pr-bootstrap/scripts/cleanup.sh:240`) and everything the
environment ever wrote vanishes with them. Under D1 the databases are shared,
so teardown becomes "delete every row belonging to the ephemeral tenant across
~85 shared databases" — a materially harder problem, and the largest cost this
design pays for D1's benefits.

Two things make it tractable:

1. **Orphaned rows are inert, not dangerous.** A `sagas` row
   (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/entity.go`
   persists the full tenant) whose tenant no longer resolves to a known
   environment is skipped by the sweeper under §7.1's rule. It is storage
   waste, not a correctness defect. Correctness does not depend on teardown
   completeness — which is exactly what makes this acceptable rather than
   merely unfortunate.
2. **A tenant-keyed sweeper, not a per-service cleanup script.** Every
   data-plane table has `tenant_id` (84 of 89 `entity.go` files — F11), and
   FR-8.1's sweep will confirm the rest. A generic "delete rows for tenant T"
   pass driven from the entity registry is one mechanism, not 85. The existing
   `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` is the place for it.

`cleanup.sh`'s `PHASES` list (line 240) gains `deactivate` (first, before any
destruction — FR-5.5) and `drop-control-plane` (environment-scoped
`configurations` rows and the `atlas-tenants` row). `drop-dbs` and
`drop-topics` remain, no-ops in sparse mode and load-bearing in isolated mode.

---

## 8. Control plane (D5)

### 8.1 Two scoping strategies, split by row kind (open question 7)

The PRD's open question 7 asks whether an `environment` column suffices or
whether an overlay-with-baseline-fallback read is better. The answer differs
by table, and the split is principled:

| Table | Strategy | Why |
|---|---|---|
| `atlas-configurations.services` | **Strict** — rows for my environment only | Per-environment instances (ports, IPs, SERVICE_IDs). Inheriting `main`'s would be actively wrong. |
| `atlas-configurations.tenants` | **Strict** | A tenant belongs to exactly one environment (FR-7.1). |
| `atlas-tenants.tenants` | **Strict** | Same. |
| `atlas-configurations.templates` | **Baseline fallback** — my environment's row if present, else the baseline's | Version-keyed and identical across environments; ~76 KB each (`bootstrap.sh:280`). Copying per environment is pure duplication. |

Fallback reads are a deviation from FR-7.5's "only rows for its execution
environment" and are flagged in §11. The justification is concrete: templates
are keyed by `(Region, MajorVersion, MinorVersion)` — a *version* identity, not
an *environment* identity — and the PR bootstrap already treats them as a
shared read-only source it clones from
(`bootstrap.sh:266-287`). Fallback preserves that relationship, keeps
provisioning from copying 76 KB per environment, and still lets a PR override
a template locally by inserting its own row. A PR that *changes* template
semantics escalates to isolated mode anyway (FR-9.3).

`main`'s behaviour is unchanged in both strategies: `main`'s environment is the
baseline, its rows carry `environment = 'main'` after backfill, and a strict
read and a fallback read return the same rows for it (NG6).

### 8.2 Write authorisation (FR-7.8)

A PR must be structurally unable to write another environment's rows. Two
layers, because one is not enough:

1. **Application:** every `atlas-configurations` and `atlas-tenants`
   administrator function takes the environment from the request context and
   scopes both the `WHERE` clause and the inserted value. A write whose target
   row's environment differs from the caller's is rejected with a distinct
   error (PRD §7.5).
2. **Fixture:** a negative test that a `pr-*` environment's PATCH against a
   `main`-owned service row returns an error and leaves the row byte-identical.

Layer 1 alone is a convention enforced by review. Layer 2 is what makes G7
verified rather than intended. C2 is why this needs both.

### 8.3 Tenant → environment derivation (FR-7.3)

Structural, per PRD §4.1: the tenant row carries its environment, so
"which environment does tenant T belong to?" is answered by the row that
defines T — projected into memory alongside environments and available with no
I/O. This is the recovery path for **all** persisted and delayed work: the
saga sweeper, the outbox drainer, and every background loop reconstruct
environment from the tenant they read off a row.

### 8.4 The outbox needs no schema change

`libs/atlas-outbox/entity.go` stores `Headers datatypes.JSON` and
`libs/atlas-outbox/headers.go` round-trips them byte-exactly through base64.
The `ENVIRONMENT` header rides along with no migration.

One consequence to state rather than discover: under D1, an override and its
baseline share the service's `outbox_entries` table, so either may drain the
other's rows. This is **correct** — the row carries the environment header, the
published message carries it onward, and the gate routes it to the right
consumer. Draining is not originating. It does mean the `pg_try_advisory_lock`
in `libs/atlas-outbox/lock.go` (a single constant key per database) now
serialises drainers across environments. That is a throughput coupling, not a
correctness one, and it is the kind of "deliberately global" resource
isolation-audit §4.3 asks to be enumerated with a stated intent.

---

## 9. Redis (D7, FR-8.3/8.4)

`libs/atlas-redis/keys.go:15` computes the key prefix in a package-level `var`
initializer from `os.Getenv("ATLAS_ENV")`. It cannot vary per operation
without restructuring, and D1 removes it as an isolation boundary in sparse
mode anyway.

Disposition:

- **Data plane** migrates to the tenant-scoped constructors
  (`NewTenantRegistry`, `NewTenantSet`, …), which build keys via
  `tenantEntityKey` and are correct in a shared keyspace. The affected
  namespaces are enumerated in isolation-audit §4.2.
- **`atlas-storage`'s `npc-context` cache** (`storage/cache.go:31-36`, keyed on
  `strconv(characterId)` alone) is a **live cross-tenant defect today**,
  independent of this project. It is fixed first, on its own commit, so the fix
  is reviewable without the migration around it.
- **Control plane** (`atlas-data`'s `ingestrun`) moves to a new, narrowly
  named environment-scoped constructor.
- **The bare constructors** are unexported or removed; `tools/rediskeyguard`
  (which already exists) is extended to fail CI on new bare `keyFn` use
  (FR-8.5).
- **`ATLAS_ENV` prefixing** stays in force for isolated mode and becomes inert
  in sparse mode by not setting the variable in the sparse overlay. No code
  change to `computeKeyPrefix`; the empty-string branch is already the
  legacy/main path.

`NewGlobalIDGenerator`, `NewLock`, `NewLockWithTTL` each need a stated intent
per surviving call site (isolation-audit §4.3). A lock that is global by
intent is now global *across environments* — either correct or a
cross-environment stall, and the difference cannot be assumed.

---

## 10. Guards

Guards are how this survives contact with 64 services and future PRs.
`tools/lint.sh --check` and `tools/verify.sh` already run a guard suite;
these join it.

| Guard | Fails on | Requirement |
|---|---|---|
| `env-domain-guard` | `atlas-env` imported from a domain package (anything below `main.go` / `kafka/` / `rest/`) | FR-4.5, NG5 |
| `producer-seam-guard` | a new direct `producer.Produce(` outside `libs/` | FR-4.1 |
| `control-plane-scope-guard` | a new `atlas-configurations` / `atlas-tenants` entity without an environment column | FR-8.5 |
| `tenant-scope-guard` | a new `services/*/entity.go` primary entity without `TenantId`, absent an allowlist entry with a reason | FR-8.5 |
| `rediskeyguard` (extend existing) | new bare-constructor Redis use in a data-plane package | FR-8.4/8.5 |
| `env-bootstrap-guard` | a service `main.go` that does not wire the environment registry | §12 gating |

The last one is the gate on enabling sparse mode at all (open question 11).

---

## 11. Deviations from the PRD

Four, each with its reasoning above. They need explicit sign-off.

| # | PRD says | Design does | Why | §|
|---|---|---|---|---|
| **V1** | D2: Environment CRD watched by services | Compacted Kafka topic projected into memory | The projection machinery, its catch-up gate and its readiness composition already exist in `libs/atlas-service`; a CRD watch puts client-go and RBAC into 64 services that have never talked to the API server, and gives no better staleness signal | §4.1 |
| **V2** | FR-5.4: no window of double **or zero** ownership | At most one owner ever; zero owners only while no operation for that environment can exist | Literal FR-5.4 needs consensus across 64 independent caches. The weaker invariant is sufficient for G4/NFR-5 because pre-ACTIVE environments have no reachable work | §7.3 |
| **V3** | FR-7.5: control-plane reads return only rows for the execution environment | `templates` reads fall back to the baseline row | Templates are version-keyed, ~76 KB, and already treated as a shared read-only source by the PR bootstrap. Strict scoping means copying them per environment for no benefit | §8.1 |
| **V4** | §8.2: `services` key becomes `(Type, environment)` | Key unchanged; `environment` added as a scoping column only | The table is already `Id`-keyed and consumers select by `SERVICE_ID`. Per-environment rows are already representable | §1 C1 |

---

## 12. Sequencing (open question 11)

The intermediate state is safe, and this is the property that makes a
64-service migration possible at all:

> **Every mechanism is inert when no Environment record exists** (FR-1.8). An
> empty environment string resolves to the local deployment for all four
> registry queries; the gate processes everything; `RootUrl` returns exactly
> what it returns today.

So services can be migrated onto `main` incrementally, each merge changing
nothing observable. The dangerous state is not "some services migrated" — it is
**"sparse mode enabled while some services are unmigrated."**

The gate is therefore not a code-freeze but `env-bootstrap-guard` (§10):
sparse mode is not selectable until every service wires the registry, and CI
proves it. Ordering:

1. **Prerequisites (FR-8, blocking).** Query-path sweep; `atlas-storage` fix;
   Redis migration; control-plane environment columns and backfill; guards.
   Nothing routing-related is safe before these land — as FR-8 states.
2. **Libraries.** `libs/atlas-env`; the `atlas-environments` service; header
   plumbing in `libs/atlas-rest` and `libs/atlas-kafka`; the gate; the
   `libs/atlas-service` iteration helper. All inert.
3. **Services.** 64 × (wire the registry in `main.go`; convert background
   loops). Mechanical, independently mergeable, no domain code touched.
4. **Deployment.** `gen-routes.sh` per-service namespace variables; the sparse
   overlay; the consumer-group offset seeding in the precreate Job; bootstrap
   changed from merge-into-pinned-row to create-own-row (C2); `cleanup.sh`
   phases.
5. **Mode selection.** Affected-service determination and escalation.
6. **Enable.** Flip the default once `env-bootstrap-guard` passes.

Step 3 is where the tool-call budget goes and where fresh-context implementers
per service batch are the right unit.

---

## 13. Mode selection (FR-9) and open question 12

`tools/cideps` already computes the affected-module set from a changed-file
list **including transitive module dependencies** — it is what
`.github/workflows/pr-validation.yml`'s `detect-changes` job
(lines 46-65) feeds every downstream matrix from. Affected-service
determination for FR-9.2 is that same computation, not a new one.

Escalation to isolated mode is a path-prefix rule over the changed files,
evaluated before the override set is computed, per FR-9.3.

**How often does that fire?** Measured over the 2492 commits between
2026-02-15 and 2026-08-15 on `main`:

| Trigger | Commits | Share |
|---|---|---|
| touches `services/atlas-configurations` or `services/atlas-tenants` | 127 | 5.6 % |
| touches `libs/atlas-{kafka,rest,tenant,redis}` | 85 | 3.7 % |

Roughly **one commit in eleven** hits an escalation trigger (the two sets
overlap, and `deploy/k8s/base` and migration triggers are not counted, so the
true rate is somewhat higher). Caveat: this counts commits, not PRs, and a PR
bundling many commits escalates if *any* one of them qualifies — so the
per-PR rate will be higher than 9 %. The measurement is a lower bound.

The answer to open question 12 is therefore: **no, the shared control plane
does not defeat the purpose** — the large majority of PRs are single-service
or few-service changes that sparse mode serves — but the escalation rate is
high enough that isolated mode must stay a first-class, tested, scheduled path
(FR-9.6), not a vestigial fallback.

---

## 14. Observability (FR-10)

- **Log field.** `environment` is added by a logrus hook in
  `libs/atlas-service/logger.go` beside the existing `service.name` hook
  (`logger.go:31-43`) — emitted by the logging setup, never by callers
  (FR-10.1). It reads from the entry's context where present and falls back to
  `env.Self()`.
- **Metric cardinality (open question 10).** `environment` is a label on
  exactly three new counters — gate `processed`, `skipped_not_owner`,
  `dropped_unresolvable` — and on the existing REST and Kafka message
  counters. It is **excluded** from anything already labelled by topic ×
  partition × handler, where it would multiply an already-large product.
  Budget: ~10 concurrent environments, so a bounded small multiple on a
  handful of series.
- **Alert.** `dropped_unresolvable > 0` is the P0 signal for cross-environment
  leakage (FR-10.4). It should be zero in steady state in every environment.
- **Log filtering.** Loki has no `app` label in this cluster; selectors are
  `{service_name="atlas-monsters"} | environment="pr-123"` (FR-10.5).

---

## 15. Testing

The gate is where the correctness lives, so that is where the tests go.
Per CLAUDE.md, existing tests may currently pin the *old* silent behaviour —
each negative case below must be a new assertion of the new contract, not an
adjustment of an old one.

**Unit — `libs/atlas-env`, `libs/atlas-kafka`, `libs/atlas-rest`**

- `Resolve` falls back to baseline; honours override; never hard-codes `main`.
- Empty environment ⇒ every query returns the legacy answer (FR-1.8). This is
  the NFR-7 test and must exist before any service is migrated.
- Gate: owner ⇒ processed; non-owner ⇒ acked, not processed, counter
  incremented; unknown environment ⇒ acked, not processed, alertable counter.
- Gate: stale registry ⇒ `env.Self()` processed, foreign environments dropped.
- Header round-trips REST → Kafka → REST unchanged across three hops.
- Tenant-derived environment vs. header mismatch ⇒ hard error, both transports
  (FR-7.7).

**Integration**

- Two deployments of one service, one baseline and one override, on one topic:
  every message is processed exactly once, by the right one (FR-4.6).
- A PATCH from a `pr-*` environment against a `main`-owned control-plane row
  is rejected and the row is byte-identical afterwards (FR-7.8, G7). This is
  the C2 regression test.
- After an environment is removed from the registry, messages for it are
  dropped and no baseline handler runs (FR-5.7).
- `main` with no Environment records: golden-path behaviour byte-identical to
  the pre-change build (NFR-7, NG6).

**End-to-end (the §17 proof)**

- `pr-123` overriding `atlas-character`: exactly three workloads deployed.
- A client connects to `pr-123`'s login; the session stays
  `environment=pr-123, tenant=T123` through a mixed override/baseline path.
- A baseline `atlas-monsters-main` originates autonomous work separately for
  `main` and `pr-123`, and none for an environment that overrides it.

---

## 16. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| **C2**: bootstrap merges the PR tenant into `main`'s service row under a shared DB | **Highest** — silently mutates `main`, violating G7, and would not be caught by `tools/verify.sh` | Rewrite to create-own-row; the §15 negative test is the gate |
| Tenant-scoped Postgres audit (FR-8.1) is a **sweep**, not a sample; 89 entities, and a `tenant_id` column proves nothing about the `WHERE` clause | High — a missed query path is silent cross-tenant corruption in `main` | Per-service recorded result; `tenant-scope-guard` prevents regression while the sweep is in flight |
| Teardown completeness under D1 (§7.5) | Medium | Correctness does not depend on it (orphans are inert); tenant-keyed sweeper for storage |
| Registry unavailability | Medium | Fail closed to `env.Self()`; `main` degrades to today's behaviour (§4.3) |
| Consumer-group offset seeding depends on the group being empty at reset time | Medium | Runs at sync-wave 0, before any Deployment starts; Argo CD already enforces the ordering |
| Cross-namespace ingress calls (M3) blocked by NetworkPolicy | Low | `grep -rn NetworkPolicy deploy/k8s/` returns nothing today; must be re-checked before enabling, since a future NetworkPolicy would break M3 silently |
| 64-service migration drift — some services migrated, sparse enabled | Medium | `env-bootstrap-guard` gates mode selection (§12) |

---

## 17. Still open

Not blocking the plan, but needing a decision during it:

1. **`atlas-environments` as a new service vs. a package inside
   `atlas-configurations`.** Both work. A new service is cleaner (the
   environment registry is not tenant configuration) but adds a 65th
   Deployment and its own migration path. Leaning new service; the deciding
   factor is whether FR-9.3's "changes to `atlas-configurations` escalate to
   isolated" should also capture environment-record changes — if they share a
   service, it does, and that is arguably correct.
2. **`env.Id` validation.** Is any string an environment, or is it constrained
   to `main|pr-\d+`? A constrained form makes the namespace derivation total
   and cheap; an unconstrained one keeps FR-1.5's second-baseline promise
   fully open. Leaning constrained-with-an-escape-hatch.
3. The exact list of `NS_*` variables in §5.3 — one per service, or one per
   *routed* service (114 `location` blocks vs. 64 Deployments; the counts
   differ and the mapping must be derived from `tools/gen-routes.sh`, not
   assumed).
