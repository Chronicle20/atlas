# Sparse ephemeral environments — operator runbook

This runbook covers day-to-day operation of sparse-mode ephemeral PR
environments (task-232): how mode is chosen, what the mandatory floor is,
how to read the ownership-gate counters, the P0 leakage alert, and how to
find a PR's logs in Loki. See
`docs/tasks/task-232-sparse-ephemeral-environments/design.md` for the full
design and `prd.md` for the FR/NFR numbering referenced below.

## Mode selection (FR-9)

Every PR deploys in one of two modes:

- **Sparse** (default) — only the changed services (plus the mandatory
  floor, below) are deployed as their own workloads; everything else is
  served by the `main` baseline via the environment registry.
- **Isolated** (existing) — a full-stack deployment, unaffected by the
  registry/gate machinery this runbook covers.

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
