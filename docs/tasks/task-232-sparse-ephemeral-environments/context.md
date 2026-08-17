# task-232 — Implementation context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task
up cold: it names the files that matter, the decisions already taken, and the
places where the obvious approach is wrong.

Verified against worktree `task-232-sparse-ephemeral-environments`, branch
point `c8d44127c`. Every file:line below was read during plan authoring.

---

## 1. The one-sentence version

Environment becomes a property of the operation — carried on
`context.Context`, one flat header on REST and one message header on Kafka,
resolved through a per-process in-memory registry projected from a compacted
Kafka topic — so a PR can deploy three workloads instead of 64 and have
requests execute through a mix of its own and `main`'s deployments without
ever leaving its environment.

## 2. The structural discovery the PRD did not have

> Every inter-service REST call in Atlas already goes through an ingress whose
> address is a single environment variable, and every environment already
> deploys its own ingress.

`BASE_SERVICE_URL` is
`http://atlas-ingress.atlas-main.svc.cluster.local:80/api/`
(`deploy/k8s/overlays/main/kustomization.yaml:39`) and
`http://atlas-ingress.atlas-pr-<N>.svc.cluster.local:80/api/`
(`deploy/k8s/overlays/pr/kustomization.yaml:160`).
`libs/atlas-rest/requests/url.go` resolves every outbound call to it.

REST routing therefore needs no service mesh, no `njs` map, no regenerated
ConfigMap, and no per-request registry lookup — only the originating process
choosing *which ingress* to send to, which is a namespace substitution in one
string. That is Task 23, and it is what closes the leak the PRD identified but
did not localise.

## 3. Key files, by mechanism

| Mechanism | Files you will actually edit |
|---|---|
| **Context (M1)** | `libs/atlas-env/` (new module). Mirrors `libs/atlas-tenant/processor.go:12-17,56-97` exactly |
| **Registry (M2)** | `libs/atlas-env/registry.go` (queries), `libs/atlas-service/envregistry.go` (the Kafka projection), `services/atlas-configurations/atlas.com/configurations/environments/` (the source of records) |
| **REST (M3)** | `libs/atlas-rest/requests/url.go`, `requests/header.go:29`, `server/handler.go:34`, `server/register.go` (all four registrars), `deploy/k8s/base/atlas-ingress.yaml:38-41`, `tools/gen-routes.sh` |
| **Kafka (M4)** | `libs/atlas-kafka/producer/header.go`, `producer/provider.go:14-22`, `consumer/header.go:28-66`, `consumer/manager.go:611-615` (the gate) |
| **Autonomous (M5)** | `libs/atlas-service/foreach.go` (new), then the 8 class-1 files from `grep -rl NewTicker services` |
| **Control plane (D5)** | `services/atlas-configurations/atlas.com/configurations/{tenants,templates,services}/entity.go`, `services/atlas-tenants/atlas.com/tenants/tenant/entity.go` |
| **Redis (D7)** | `libs/atlas-redis/keys.go:15-46`, the six services in `isolation-audit.md` §4.2 |
| **Deployment** | `deploy/k8s/overlays/pr-sparse/` (new), `deploy/k8s/base/atlas-kafka-precreate.yaml`, `services/atlas-pr-bootstrap/scripts/{bootstrap,cleanup,service-config,sweep-orphans}.sh` |
| **CI** | `tools/mode-select.sh` (new), `.github/actions/detect-changes/`, `.github/workflows/pr-validation.yml` |

## 4. Decisions already taken — do not relitigate

From the PRD (D1–D7) and the design (V1–V4, and P1–P4 which this plan adds):

| # | Decision |
|---|---|
| D1 | Sparse environments share `main`'s Postgres, Kafka topics and Redis keyspace. Isolation is `tenant_id` on the data plus `environment` on the operation |
| D5 | The control plane is **environment**-scoped, not tenant-scoped. Adding a `tenant_id` column to the table that lists tenants is a category error |
| D6 | `atlas-login` and `atlas-channel` are mandatory overrides in every sparse environment. The floor is two workloads, never zero |
| D7 | The tenant-scoped Redis API is mandatory for data-plane state; the bare constructors are withdrawn |
| V1 | The registry is a **compacted Kafka topic projected into memory**, not a CRD watched by client-go. 64 services have never talked to the Kubernetes API, and a ConfigMap's ~60 s propagation cannot be *observed* as stale |
| V2 | FR-5.4 is refined: at most one owner ever, and zero owners only during windows in which no operation for that environment can exist |
| V3 | `templates` reads fall back to the baseline row. They are version-keyed, ~76 KB, and already treated as a shared read-only source by the PR bootstrap. Implemented as three **key-aware** helpers in the `templates` package — a single-key lookup, an anti-join collection read, and a by-id visibility rule — not as one generic GORM scope (see §5.5) |
| V4 | The `services` table key is **unchanged**. It is already `Id`-keyed and consumers select by `SERVICE_ID` |
| P1 | Environment records live in `atlas-configurations`, not a new 65th service |
| P2 | `env.Id` is validated at ingest (`^[a-z][a-z0-9-]{0,30}[a-z0-9]$`), never per-operation; the namespace comes from the record's field, never derived from the id |
| P3 | The `NS_*` ingress variables are **derived** by `tools/gen-routes.sh`, never hand-listed |
| P4 | `libs/atlas-env` is a leaf module; its Kafka projection lives in `libs/atlas-service`, or the module graph is mutually recursive |

## 5. Six places where the obvious approach is wrong

1. **`upsert_service_config` merges into a pinned `main` row.**
   `services/atlas-pr-bootstrap/scripts/bootstrap.sh:303-320` +
   `scripts/service-config.sh`. Safe today because each PR has its own
   database; under D1 it silently mutates `main`'s service row. This is the
   highest-severity item in the whole plan (Task 47) and `tools/verify.sh`
   cannot see it. It gets its own negative test.

2. **Background loops cache the tenant list at process start.**
   `services/atlas-transports/atlas.com/transports/main.go:93-151` — the very
   file the PRD cites as the reference pattern — loads tenants once and the
   ticker goroutine closes over the slice. A tenant created after the pod
   started is invisible until restart, which directly blocks the project
   (a baseline pod must pick up a new ephemeral tenant **without** redeploying
   `main`). Every loop must be *checked* for this, not merely wrapped.

3. **`atlas-storage`'s `npc-context` cache keys on character id alone.**
   `services/atlas-storage/atlas.com/storage/storage/cache.go:31-36`.
   Character ids are per-tenant sequences, so this is a **live cross-tenant
   collision today** inside multi-tenant `main`; the `ATLAS_ENV` prefix masks
   it only across environments. Fixed on its own commit (Task 4), independent
   of this project.

4. **A `tenant_id` column is not isolation.** FR-8.1 is a query-path sweep,
   not a schema sweep. 84 of 89 `entity.go` files have the column; what
   matters is the `WHERE` clause of every read and the assignment of every
   write. Tasks 1–3 record the result per service, and Task 15's guard keeps
   it from regressing.

5. **"Resolve the namespace" is two questions, not one.** The registry
   exposes `EnvironmentNamespace(e)` — the environment's own namespace, where
   its own ingress lives — and `ServiceNamespace(e, service)` — the effective
   implementation's namespace, which falls back to the baseline. Outbound
   REST (Task 23) must use the **first**. A record's `overrides` map does not
   list `atlas-ingress`, so `ServiceNamespace(e, "atlas-ingress")` returns the
   baseline's namespace and a baseline pod handling a `pr-123` operation would
   send its next call into `main` — the exact leak M3 exists to close. Task 18
   carries a test that fails if the two queries are ever collapsed.

6. **Template fallback is key-aware, not an `ORDER BY`.** A `CASE`-ordered
   `WHERE environment IN (e, baseline)` resolves a single-key `First()` and
   nothing else. On a collection read it returns the union: with `main` at
   83.1 + 95.1 and `pr-123` at 83.1, the union is three rows where the overlay
   is two (`pr-123`/83.1 and `main`/95.1). Task 13 uses a `NOT EXISTS`
   anti-join for collections, the ordered form only for the one exact version
   key, and a plain own-or-baseline visibility rule for by-id lookups.

## 6. Ordering constraints that actually bind

- **Phase A blocks everything.** FR-8 says so and it is not a formality: with
  D1 removing all three deployment-scoped substrates, an unscoped query path
  becomes silent cross-tenant corruption **in `main`**.
- **Task 8 depends on Task 17** (`libs/atlas-env` must exist before
  `libs/atlas-redis` can import it). If you work strictly in numeric order,
  run Task 8 after Task 17.
- **Tasks 13 and 14 depend on Tasks 11/12 (schema), 17 (`env`), and 22
  (the request context that carries the environment).**
- **Phase C depends on Tasks 20, 23, 25 and 30** (the recipe).
- **Task 52 depends on all of Phase C.** `env-bootstrap-guard` is written in
  Task 28 and deliberately left *out* of `verify.sh` until then, because it is
  expected to fail for the whole of Phases B–C.
- **Task 53 (Kafka fan-out measurement) gates Task 54.** It needs a live
  cluster and up to ten concurrent sparse environments, so it cannot run
  before Phase D works end to end.
- **Task 54 (flip the default) depends on every guard passing** and on Task
  53's verdict, listed as executable commands in its Step 1.

Everything else is genuinely parallel. Within Phase A, Tasks 4–7 (Redis) and
Tasks 11–14 (control plane) do not touch each other.

## 7. Tasks deliberately left large, and why

The writing-plans and CLAUDE.md guidance is to split anything touching more
than ~6 files or more than one service. Five tasks break that rule on purpose:

| Task | Why it stays whole |
|---|---|
| **1–3** (query sweep) | Already split into thirds. Each is read-only and produces table rows; the cost is reading, not editing, and splitting further would fragment one audit across six reports |
| **13** (`atlas-configurations` scoped reads/writes) | Four `provider.go`/`administrator.go` pairs plus `scope/`. Splitting reads from writes would land a half-scoped service, which is worse than a large task |
| **33–40** (service batches, 8 services each) | The same three mechanical edits repeated. Batching is explicitly the exception CLAUDE.md allows. The per-service build is the checkpoint |
| **34** (contains `atlas-character`) and **32** (`atlas-channel`) | These two services are much larger than the rest. `atlas-channel` has its own task for that reason. If Task 34 exceeds the tool-call budget it should hand back `PARTIAL` with the remaining service list rather than compressing — the continuation is cheap because the recipe is written down |
| **55** (the proof) | It is one coherent measurement against one live environment. Splitting it would mean deploying the environment more than once |

## 8. What "done" means

`tools/verify.sh` **flagless** exits 0 — `--quick` does not count, and the
bake step is load-bearing here because Phase B adds a new `libs/atlas-env`
dependency to four shared libraries and every service Dockerfile must
`COPY` it. `go build` against `go.work` cannot catch a missing `COPY`.

Beyond the gate, the PRD §12 checklist is signed off against
`proof.md` (Task 55) and `query-scope-audit.md` §4 (Task 16). A green
`verify.sh` is not sufficient: the gate cannot see a producer/consumer seam,
and the correctness of this project lives almost entirely in seams.

## 9. Artifacts this plan produces besides code

| File | Task | Purpose |
|---|---|---|
| `query-scope-audit.md` | 1–3, 10, 16, 46 | The FR-8.1 sweep, the FR-8.6 dispositions, the global-Redis intents, and the prerequisite sign-off |
| `service-wiring-recipe.md` | 30 | The three edits Tasks 31–40 apply verbatim |
| `ticker-dispositions.md` | 42 | All 18 ticker files, classified, with evidence for every "no change" |
| `docs/runbooks/sparse-environments.md` | 29, 45, 46, 49, 53, 54 | The operator runbook: modes, the floor, the gate counters, the alert, the Loki selectors, the MetalLB pool ceiling |
| `kafka-fanout-measurement.md` | 53 | Broker egress, consumer CPU, lag and drop ratio at 1 / 6 / 11 concurrent environments, with a pre-stated threshold |
| `proof.md` | 55 | The §17 acceptance proof, with pasted evidence |

## 10. Environment-specific gotchas worth knowing

- **Loki has no `app` label in this cluster.** Selectors are
  `{service_name="atlas-monsters"} | environment="pr-123"`. An `app=`
  selector silently returns zero rows.
- **`world.Id` / `channel.Id` are `byte`, `_map.Id` is `uint32`.** Not
  relevant to most of this plan, but `atlas-channel`'s socket work touches
  them.
- **`gh` needs the token env cleared:** `env -u GH_TOKEN -u GITHUB_TOKEN gh …`.
- **`ForEachOwnedEnvironment` is serial on purpose.** Every class-1 ticker
  loop today is `for _, t := range tenants { work }`. Making the helper
  concurrent would turn a one-second tick into a burst of goroutines across
  every tenant of every environment — a behavioural change with real blast
  radius and nothing to do with environment isolation. Fault isolation comes
  from a per-iteration `recover`, not from a goroutine.
  `ForEachOwnedEnvironmentConcurrently` exists for loops that already
  parallelised their tenants; Tasks 41–42 must state per loop which variant
  they used and what the pre-existing shape was.
- **Never sweep the filesystem for a Go dependency.**
  `go list -m -f '{{.Dir}}' <module>` answers in ~0.02 s;
  `find /` takes ~2 minutes on this WSL2 host.
- **MetalLB `pr-pool` is 20 addresses**, two per sparse environment, so the
  concurrency ceiling is 10 — which is also the metric-cardinality budget in
  Task 29. Confirm the range from the cluster in Task 46 rather than from the
  comment in `lb-allocate.yaml`.

## 11. The one cost this design does not yet bound

Sparse environments consume the **unsuffixed** baseline topics (FR-4.8), so
every override consumer group reads its service's entire topic and gate-drops
the traffic belonging to other environments. Ten concurrent environments
overriding `atlas-character` means eleven consumer groups each seeing
essentially all character messages. The trade is `60 pods × environment`
before, `topic traffic × overrides-of-that-service` after.

That is very likely still a large win — a dropped message costs a header
parse and two map lookups, not a domain handler — but it is superlinear in
exactly the dimension the project is designed to increase, and nothing in the
design measures it. **Task 53 does, and it gates Task 54.** State the
acceptance threshold before reading the numbers; if the verdict is "needs
mitigation", the cheapest remedy is escalating PRs that touch the identified
high-volume service to isolated mode — a one-line addition to Task 50's
escalation table.
