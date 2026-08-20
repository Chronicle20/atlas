# Sparse ephemeral environments — operator runbook

This runbook covers day-to-day operation of sparse-mode ephemeral PR
environments (task-232): how mode is chosen, what the mandatory floor is,
how to read the ownership-gate counters, the P0 leakage alert, and how to
find a PR's logs in Loki. See
`docs/tasks/task-232-sparse-ephemeral-environments/design.md` for the full
design and `prd.md` for the FR/NFR numbering referenced below.

## Mode selection (FR-9)

**Sparse is the default mode.** `tools/mode-select.sh`'s no-trigger branch
resolves to `sparse` — only the changed services (plus the mandatory floor,
below) are deployed as their own workloads; everything else is served by the
`main` baseline via the environment registry. Escalating to isolated is the
documented exception path, not a coin-flip alternative:

- **Sparse** (default) — as above.
- **Isolated** (escalation) — a full-stack deployment, unaffected by the
  registry/gate machinery this runbook covers. Used automatically when a
  trigger below fires, or explicitly via `atlas:isolated` (see
  `docs/runbooks/ephemeral-pr-deployments.md` §9.15 for both override
  labels and the full trigger list).

**Mode escalates to isolated automatically (FR-9.3)** when a PR touches any
of:

- `deploy/k8s/base` or the ingress configuration;
- Kafka infrastructure or message contracts;
- database migrations;
- `atlas-configurations` or `atlas-tenants` — the control plane itself,
  which sparse environments share;
- cross-cutting libraries whose blast radius cannot be bounded:
  `libs/atlas-kafka`, `libs/atlas-rest`, `libs/atlas-tenant`,
  `libs/atlas-redis`, `libs/atlas-env`;
- any case where affected-service determination is unreliable.

Measured over ~2500 commits on `main`, roughly one commit in eleven hits an
escalation trigger — so isolated mode is a first-class, regularly-exercised
path, not a vestigial fallback. Mode is overridable per PR via an explicit
label in either direction (FR-9.5), and the chosen mode, the override set,
and the reason are reported on the PR (FR-9.7).

### The mandatory floor (D6)

The sparse override set is always the changed services **plus
`atlas-login` and `atlas-channel`** (FR-9.4) — the mandatory floor. These
two are always deployed as their own workloads in a sparse environment
regardless of whether the PR touched them, because they are the entry point
a client actually connects through. `atlas-drops` does **not** join the
floor: it has no tenant/port identity of its own (it reads task intervals
from `ServiceTypeDrops` once at startup) and is an ordinary override
candidate like any other service.

## Reading the three gate counters

Every Kafka consumer applies the ownership gate (`libs/atlas-kafka/consumer/gate.go`)
before handing a message to domain handlers. It emits exactly three
counters, each labelled `{service, environment}`:

| Counter | Meaning | Expected in steady state |
|---|---|---|
| `atlas_kafka_gate_processed_total` | Message passed the gate and was handed to domain handlers. | Grows normally with traffic. |
| `atlas_kafka_gate_skipped_not_owner_total` | Acknowledged without processing — this deployment is not the environment's owner for this service (FR-4.4). | Grows in sparse mode; a baseline pod sees every environment's traffic and skips what it doesn't own. Not a problem by itself. |
| `atlas_kafka_gate_dropped_unresolvable_total` | Acknowledged and **dropped** — the message's environment could not be resolved to any owning deployment (FR-4.7, D4), or the header disagreed with the tenant it named (FR-7.7). | **Should be zero.** See the alert below. |

`skipped_not_owner` growing is routine and expected — it is how one baseline
pod correctly ignores traffic belonging to environments it does not own.
`dropped_unresolvable` growing means a message nobody processed: the
operation silently disappeared.

### Fan-out cost is unmeasured at the time sparse became the default

Sparse mode makes every override deployment's shared-topic consumer group
receive **every** environment's traffic on the unsuffixed baseline topics —
the gate above is what filters it back down per-environment. Task 53 was
scoped to measure that fan-out cost (consumer lag, gate-skip rate) against
`atlas_kafka_gate_*` under concurrent sparse environments, but those counters
are introduced by this same branch and are not yet deployed — there is no
live data to measure against yet. **Task 53 is deferred to a post-deploy
follow-up**, not skipped: this doc will be updated with the real numbers once
they exist. Until then, treat the fan-out cost as an open question, not an
assumed-acceptable one, and watch for the symptom below.

**Symptom to watch for:** consumer lag climbing on the shared, unsuffixed
baseline topics as the number of concurrent sparse environments increases —
check `atlas_kafka_gate_skipped_not_owner_total` growth rate per topic
alongside standard consumer-lag metrics. If lag grows materially with
concurrency, the mitigation options (in priority order to evaluate, none
implemented yet) are:

- escalate the hot service to isolated mode for PRs that touch it;
- reinstate topic suffixing for that specific service;
- partition consumer groups by environment for that topic.

## The alert (FR-10.4)

```promql
atlas_kafka_gate_dropped_unresolvable_total > 0
```

is the **P0 signal for cross-environment leakage**. It should be zero in
steady state in every environment. A non-zero value means an operation
carried an environment nobody could resolve, and the message it names was
**not processed by anyone** — it was acknowledged and discarded, not
retried. Triage:

1. Check whether the named environment (label `environment`) is mid-teardown
   or mid-startup — a registry projection lag can cause a brief, transient
   blip around activation/deactivation (FR-5.5–5.7).
2. If the environment is expected to be active and the counter keeps
   climbing, check `atlas_kafka_gate_processed_total{service=...}` for the
   owning deployment of that service/environment pair — it may be crash-looping
   or never started.
3. Check whether the registry itself is stale (`Registry.Stale()` — more
   than `StaleAfter` = 120s since the last observed environment-status
   message). A stale registry fails every foreign environment closed by
   design (§4.3): only `env.Self()`'s own traffic keeps processing. This
   protects `main` at the cost of every ephemeral environment's traffic
   until the projection catches up.
4. A `mismatched` drop (header ENVIRONMENT disagreeing with the tenant it
   names) is a producer bug, not a registry problem — trace it to whichever
   service emitted the message.

## Verifying a consumer group is seeded (FR-4.9, FR-5.3)

Every override deployment in a sparse environment joins the shared baseline
topics with a brand-new consumer group (`KAFKA_CONSUMER_GROUP`, resolved by
`libs/atlas-kafka/consumergroup/resolver.go`). The sync-wave-0
`atlas-kafka-precreate` Job commits an end-of-log offset for that group on
every subscribed topic **before any Deployment starts** — while the group is
still empty and therefore resettable (design §6.3). This is what keeps a new
group from either replaying `main`'s entire retention window (the library's
`FirstOffset` default) or losing traffic produced between group creation and
first poll (`LastOffset`).

The seeding logic lives in `deploy/k8s/base/kafka-precreate.sh` (mounted into
the Job via the `atlas-kafka-precreate-script` ConfigMap) and is skipped
entirely whenever `KAFKA_CONSUMER_GROUP` is unset — that is `main`, whose
groups already exist and carry real committed offsets that must never be
reset. `deploy/k8s/base/atlas-kafka-precreate_test.sh` asserts that skip
without touching Kafka, and exercises `seed_group` against a real broker when
`BOOTSTRAP_SERVERS` is set.

To verify a group was actually seeded in a live environment:

```sh
kubectl -n atlas-pr-<N> logs job/atlas-kafka-precreate
```

should show `seeding end-of-log offsets for override consumer groups`
followed by `override consumer group offsets verified`. The Job's own
verification pass (`verify_group_offsets`) fails the Job — visible to Argo
CD's health check, which holds every Deployment (sync-wave 10) until this
Job completes — if any subscribed topic lacks a committed offset for any
override group. A completed, healthy `atlas-kafka-precreate` Job is therefore
itself the observable readiness signal (FR-5.3): no separate inference is
needed.

To check a specific group by hand:

```sh
kafka-consumer-groups.sh --bootstrap-server <broker> \
    --group "<Service Name> [pr-<N>]" --describe
```

Every row's `CURRENT-OFFSET` column should be a number, never `-` (which
means the group has no committed offset on that partition — the failure mode
this Job exists to prevent).

### `KAFKA_CONSUMER_GROUP` must be resolved, post-substitution group names

Whoever wires `KAFKA_CONSUMER_GROUP` into the Job's `atlas-env` ConfigMap
(newline-delimited, one group per line — group names contain spaces and
brackets, e.g. `"Account Service [pr-123]"`, so a space-delimited list is
ambiguous): the value must be the **resolved, post-substitution** group
name(s) actually joined at runtime, never a raw literal copied from
`deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`.
`libs/atlas-kafka/consumergroup/resolver.go` documents templated callers
(`atlas-login`, `atlas-channel`) whose `KAFKA_CONSUMER_GROUP` is a **format
string** such as `"Channel Service - %s [pr-123]"`, substituted per
channel/login instance by `Resolve(defaultName, args...)` at runtime — this
Job never performs that substitution. Copying the template literal in
unresolved seeds a group name containing a literal `%s` that no runtime
consumer ever joins: the seeding pass reports success and silently seeds
nothing real. A templated service needs one resolved entry per
channel/login instance it actually runs, not the template string itself.

## Loki selectors

**Loki has no `app` label in this cluster.** Select on `service_name` and
filter by the `environment` structured-log field (emitted by every
`CreateLogger` logger — see below):

```
{service_name="atlas-monsters"} | environment="pr-123"
```

## The `environment` log field (FR-10.1)

`environment` is added by a logrus hook in `libs/atlas-service/logger.go`,
registered beside the existing `service.name` hook and **before** the
field-key normalizer hook (which must stay last so it can normalize keys
added by every earlier hook). It is sourced from `env.Self()` — this
process's own `ATLAS_ENVIRONMENT` — never from a caller's `WithField` call
(FR-10.1: emitted by the logging setup, not by callers).

On `main` (no `ATLAS_ENVIRONMENT` set), the hook is a no-op: log records stay
byte-identical to today's (NFR-7). On an ephemeral deployment, every log
record carries `environment=pr-123` (or whatever id the deployment was
given) without any call-site change.

The nginx ingress (`deploy/k8s/base/atlas-ingress.yaml`) also passes the
`ENVIRONMENT` header through to `$http_environment` and includes it in the
access log as `env=$http_environment`, so an inbound request's environment
is visible at the ingress layer even before it reaches a service log.

## Metric cardinality budget (FR-10.3, design §14)

`environment` is a label on exactly the three gate counters above plus the
existing REST and Kafka message counters. It is **excluded** from anything
already labelled by topic × partition × handler, where it would multiply an
already-large product. Budget: ~10 concurrent environments, i.e. a bounded
small multiple on a handful of series.

Do not add `environment` as a label to any metric already carrying
topic/partition/handler cardinality, and do not source an `environment`
label from message or request header data — always from `env.Self()`
(process-local, trusted). Header-derived data is unvalidated cardinality: a
producer bug can turn an unbounded string into an unbounded label set.

## Login/channel port and IP allocation (Task 46)

Each environment's `atlas-login-lb` / `atlas-channel-lb` request a dynamic IP
from MetalLB's `pr-pool` (`deploy/k8s/overlays/pr/patches/lb-allocate.yaml`,
mirrored in `pr-sparse/patches/lb-allocate.yaml`). Ports are derived from
`majorVersion` alone (`services/atlas-pr-bootstrap/scripts/version-ports.sh`),
so two environments on the same client version bind the identical port on
different IPs — correct for a LoadBalancer Service, and confirmed as such in
Task 46. Only the IP varies per environment; the port never does.

`pr-pool` is confirmed live (`kubectl -n metallb-system get ipaddresspool -o
yaml`) as `192.168.23.190-192.168.23.209` — 20 addresses. Each sparse
environment consumes **two** (login + channel), so the pool ceiling is **10
concurrent environments**, matching Task 29's metric-cardinality budget.

That 10-environment figure is the **mechanical** limit (IP addresses, a hard
resource bound) — it is not a validated statement about fan-out cost at that
concurrency. See "Fan-out cost is unmeasured" above: the Kafka fan-out cost
of 10 concurrent sparse environments sharing unsuffixed baseline topics has
not been measured (Task 53, deferred). Treat 10 as "the pool physically
cannot exceed this," not "10 is known to be safe for consumer lag."

### NetworkPolicy dependency (design §16)

Sparse mode's `M3` (baseline routing for un-overridden services) depends on
**cross-namespace ingress** — a sparse PR namespace's requests reach services
still running in the `main` namespace. As of this writing there are **no**
NetworkPolicy resources in `deploy/k8s/` (repo) and none in any `atlas-*`
namespace in the live cluster (cluster-wide NetworkPolicies exist only in
`argocd`, `cattle-fleet-local-system`, and `cattle-resources-system`, none of
which select atlas pods), so this dependency is currently unconstrained. It
is not enforced anywhere today, only assumed. **If a NetworkPolicy is ever
added that selects pods in an `atlas-*` namespace, it must explicitly permit
this cross-namespace ingress or sparse routing breaks silently** — validate
any future atlas-namespace NetworkPolicy against M3 before it merges.

### Failure mode: pool exhaustion

When the pool has no free address left, `atlas-channel-lb` never gets a
LoadBalancer IP and `services/atlas-pr-bootstrap/scripts/bootstrap.sh` fails
loudly during the `lb-discover` step:

```
atlas-channel-lb has no allocated LoadBalancer IP — MetalLB pool exhausted?
```

Diagnostic — find every Service stuck `<pending>`:

```sh
kubectl get svc -A | grep pending
```

Remedy: tear down an idle PR environment to free its two addresses, or widen
the `pr-pool` `IPAddressPool` range in the cluster's MetalLB configuration.

## Tenant-keyed teardown reclaim (task-49, design §7.5)

Sparse mode's databases are **shared with `main`** (D1) — a sparse PR never
gets its own Postgres instance, so `cleanup.sh`'s `drop-dbs` phase has
nothing to `DROP` there and skips entirely (`skipped (sparse)`). Everything
that PR's tenant wrote still lives in main's tables when the PR closes.

`cleanup.sh`'s `sweep-tenant` phase (after `drop-control-plane`, sparse-mode
only) reclaims those rows: it reads this environment's tenant id off the
control-plane environment record (the same record `deactivate` and
`drop-control-plane` already read — no separate env var to keep in sync),
then runs

```sh
services/atlas-pr-bootstrap/scripts/sweep-orphans.sh \
    --sweep-tenant <tenant-uuid> --apply
```

which `DELETE`s `WHERE tenant_id = '<tenant-uuid>'` from every table listed
in `services/atlas-pr-bootstrap/scripts/tenant-tables.txt` — a **checked-in,
generated** `<database> <table>` list, one row per data-plane table the
FR-8.1 query-scope audit
(`docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md`,
filtered to `Plane == Data`) found. Control-plane tables are **not** in this
list — `drop-control-plane` already owns those, scoped by environment id
rather than tenant id.

**Regenerate the list with `tools/gen-tenant-tables.sh`** whenever the audit
or a service's `deploy/k8s/base/atlas-<service>.yaml` `DB_NAME` changes; do
not hand-edit `tenant-tables.txt`. `tools/gen-tenant-tables.sh --check`
fails if the checked-in list has drifted from what the audit + manifests
would generate — a table missing from the list is silently never reclaimed,
so this check runs alongside the other generated-artifact drift guards
(`tools/verify.sh`'s "tenant tables drift" step).

**What this does NOT guarantee — correctness does not depend on it.** A row
whose tenant no longer resolves to a live environment is inert: no request
path, and no other sweeper, ever looks it up again (the unknown-environment
rule). `sweep-tenant`'s own failure is recorded in `failed_phases` but never
aborts the rest of teardown — this is storage reclamation, not a
correctness gate. Known incompleteness at the time this was written:

- **`atlas-families` and `atlas-marriages`** are not currently deployed —
  neither has a `deploy/k8s/base/atlas-<service>.yaml` manifest, so there
  are no rows for them to reclaim in any environment. Their audit tables
  (`family_members`, `marriages`, `proposals`, `ceremonies`) are **inert,
  not missed**; this is not an outstanding reclamation gap. The generator
  cannot resolve a `DB_NAME` for an undeployed service and skips them with
  a warning rather than guessing one. The situation is self-correcting: if
  either service gains a base manifest, the generated `tenant-tables.txt`
  changes and `tools/gen-tenant-tables.sh --check` fails until it is
  regenerated, so the guard catches the transition automatically.
- Any table added to a service's schema after the last audit sweep is not
  in `tenant-tables.txt` until the audit is refreshed and the generator
  re-run.

Manual invocation (e.g. reclaiming a specific tenant without a full
teardown run):

```sh
DB_HOST=... DB_PORT=5432 DB_USER=... DB_PASSWORD=... \
    services/atlas-pr-bootstrap/scripts/sweep-orphans.sh \
    --sweep-tenant <tenant-uuid>          # list mode — prints what would delete
    # add --apply to actually delete
```

## Upgrading to durable service-config binding (task-243, D9)

`servicesuniq.Migration` builds a unique index on `services (type,
environment)` at `atlas-configurations` startup, after `services.Migration`
(plain `AutoMigrate`, no index) has run. That index rejects any pre-existing
duplicate group (task-232's non-idempotent bootstrap left several — five
`login-service` rows and five `channel-service` rows were observed for one
environment, one per bootstrap re-run), so a duplicate present at rollout
crash-loops the control plane for the baseline **and** every environment
routed through it.

**Every existing sparse environment must be torn down before this change is
deployed.** This is not optional cleanup — it is the primary mitigation
(design §4.3 Layer 1). Run `cleanup.sh` for each one. Its environment-scoped
reclaim goes through `ProcessorImpl.DeleteById`, which enqueues the
nil-value tombstone a compacted topic requires to stop replaying the deleted
row; a raw `DELETE` against the `services` table does not do this and must
never be used as a substitute. The rollout is a recreate, not an in-place
reconciliation: open a fresh PR in sparse mode afterward rather than trying
to migrate a torn-down environment's state forward.

### Pre-flight: checking for duplicate service-config groups (§4.3 Layer 3)

Before rolling `atlas-configurations`, inspect the baseline database for any
`(type, environment)` group the startup migration would have to dedupe. The
migration's own read-only check (`servicesuniq.Preflight`) is not exposed as
a standalone CLI; run the equivalent query directly against the baseline
database:

```sql
SELECT type, environment, COUNT(*) AS count
FROM services
GROUP BY type, environment
HAVING COUNT(*) > 1;
```

Every row named is a group the startup migration will dedupe automatically —
keeping the row whose id equals the derived id if one exists, else the row
with the newest `service_history.created_at`, else the lowest id — and
tombstone the rest.

That automatic rule is a mechanical tiebreak, not a judgment about whether
the duplicate is legitimate. Read the named groups by hand: **if any group
represents a genuine co-resident row rather than a non-idempotent-bootstrap
artifact — including a legitimate co-resident `login-service` row on
`main`, which the PRD records as an accepted multi-tenant case — the dedupe
rule cannot resolve it correctly.** The unique index forbids re-creating
that row once deleted. Stop the rollout and re-decide D3 rather than letting
the migration silently delete a row that was supposed to stay.

### Precondition: `ATLAS_ENVIRONMENT` must be rolled to baseline deployments (§5.4)

Activation depends on `IsOwner`, which reads `r.self` from
`ATLAS_ENVIRONMENT`. A ConfigMap change does not roll the Deployments that
consume it via `envFrom`, so a baseline deployment can be running with
`Self() == ""` long after the ConfigMap itself was updated. **This is a PRD
non-goal and a separate defect** — it is not fixed by this change — but it
is a live-environment precondition of the end-to-end acceptance test:
without it, the baseline cannot resolve its own environment and the
ownership gate behaves incorrectly for baseline traffic.

Check for it before running the end-to-end test:

```sh
kubectl -n <namespace> get pods -l app=<service> -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[0].env[?(@.name=="ATLAS_ENVIRONMENT")].value}{"\n"}{end}'
```

A pod whose `ATLAS_ENVIRONMENT` column is empty predates the ConfigMap
change reaching it. Roll every affected baseline Deployment (a normal
rolling restart is sufficient — the value comes from `envFrom` on pod start,
not a live watch) before running the test; do not treat this runbook as the
place that fixes the underlying drift, only as the place that names it as a
pre-test step.

## Verifying a service-config binding (FR-1.1–FR-1.4)

To confirm an override Deployment is bound to its own environment's
service-config row rather than a stale or wrong one:

1. Read `SERVICE_ID` off the running override Deployment:

   ```sh
   kubectl -n <namespace> get deployment <deployment> \
       -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SERVICE_ID")].value}'
   ```

2. Re-derive the expected id independently:

   ```sh
   tools/derive-service-id.sh <service-type> <environment>
   ```

3. Confirm the two values match. If they do not, the binding is wrong —
   Argo's `selfHeal` may have reverted a hand-patched value, or the
   deployment predates this design.
4. Confirm the `services` row with that id actually carries the expected
   environment: `GET /api/configurations/services/{id}` and check
   `attributes.environment` equals the environment's own name. A match on
   step 3 alone is not sufficient — the id must also resolve to a row scoped
   to the right environment, not a stray row that happens to share the id.

### Verifying a seeded consumer group for that binding

Once a binding is confirmed, confirm the group it joins actually has
committed offsets — the full procedure and failure semantics are covered
above under "Verifying a consumer group is seeded"; the one-line check:

```sh
kafka-consumer-groups.sh --bootstrap-server <broker> \
    --group "<resolved group name>" --describe
```

Every row's `CURRENT-OFFSET` column must be a number. A `-` means the group
was never seeded before this Deployment started consuming — the FR-4.4
failure the wave-0 precreate Job exists to prevent.

## Readiness probe timing (design §8)

`atlas-login` and `atlas-channel`'s `readinessProbe` blocks
(`initialDelaySeconds: 10, periodSeconds: 10, failureThreshold: 30`) give a
5-minute catch-up budget before an override pod whose service-config row has
not yet been projected (`projection.State.HasService` still false) is marked
NotReady. These are conservative defaults chosen because the baseline's
actual projection catch-up time had not been measured at the time this
design was written; design §8 calls for the numbers to be validated against
a live baseline and tightened here if the measurement supports it.

**No live measurement has been recorded here yet** — record it once a
sparse environment has been created against this rollout: the wall-clock
time from pod start to the first successful `/api/readyz` response, taken
from `kubectl -n <namespace> describe pod <pod>` events or from Loki
(`{service_name="atlas-login"} | environment="pr-<N>"`). Until a real
number is recorded, treat the 5-minute budget as untightened and do not
narrow `failureThreshold`/`initialDelaySeconds` from an assumption.

## The activation window (D8)

The transition from `PROVISIONING` to `ACTIVE` is not instantaneous across
every pod's cached view of the registry — registry caches update per-pod off
the environment-status topic, so for up to one heartbeat interval different
pods can disagree about which environment owns a given service. This window
is accepted and bounded, not eliminated:

- During the window, a message for the affected service/environment pair is
  either dropped as not-yet-`ACTIVE` (a pod still on the old cached state)
  or processed by the baseline — **never processed twice**, because
  `IsOwner` is a single-winner predicate on a record every pod eventually
  converges to.
- In practice the window is empty: the ingress does not route traffic for an
  environment while it is `PROVISIONING`, so there is no traffic for the
  affected environment before the flip to `ACTIVE` completes.

No operator action is required for this window; it is documented here so a
transient, self-resolving disagreement between pods immediately around
activation is not mistaken for the P0 leakage alert above.
