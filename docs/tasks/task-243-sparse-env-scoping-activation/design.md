# Sparse Environment Service Binding and Activation — Design

Version: v1
Status: Approved
Created: 2026-08-20
PRD: `prd.md` (this folder)
Diagnosis: `diagnosis.md` (this folder)

---

## 1. Corrections to the PRD

Two of the PRD's premises are wrong on the evidence in the tree. Both narrow
the work; neither changes a goal.

### 1.1 Task-232 Task 45 is implemented — it is one wire short

PRD FR-4.3 and §7 state that offset seeding is unimplemented, citing zero
occurrences of `seed_group` in `deploy/k8s/base/atlas-kafka-precreate.yaml`.
The grep is accurate and the conclusion is not: the script was deliberately
moved out of that manifest into a ConfigMap-mounted file so it could be
sourced and tested (`kafka-precreate.sh:6`, `base/kustomization.yaml:88`).
The mechanism exists and landed in `a3586c14f`:

| Function | Location | Role |
|---|---|---|
| `seed_group` | `deploy/k8s/base/kafka-precreate.sh:132` | one multi-`--topic` `--reset-offsets --to-latest --execute` per group |
| `seed_override_offsets` | `:178` | iterates newline-delimited `KAFKA_CONSUMER_GROUP`, seeds the full topic union |
| `verify_group_offsets` | `:206` | `--describe` per group; `exit 1` if any seeded topic has no committed offset |
| `main` | `:246` | `precreate_topics; seed_override_offsets; verify_group_offsets` |

`verify_group_offsets` is precisely the *observed* signal FR-4.3 demands, and
Argo CD's health check on the wave-0 Job already carries it.

**The actual gap is wiring.** `seed_override_offsets` returns early whenever
`KAFKA_CONSUMER_GROUP` is unset — the NG6 guard that keeps the pass inert on
`main`. The precreate Job takes its environment from `envFrom: configMapRef:
name: atlas-env`, and `KAFKA_CONSUMER_GROUP` is **never** a key in that
ConfigMap: it is set per-Deployment by
`overlays/pr-sparse/patches/consumer-group-env.yaml`. So on every sparse sync
the Job logs `KAFKA_CONSUMER_GROUP unset — skipping override offset seeding
(main, NG6)` and exits 0 having seeded nothing.

Consequence for scope: PRD Open Question 5 ("is Task 45 absorbed here or
executed as a prerequisite?") dissolves. It is absorbed, and it is one
CI-rendered ConfigMap key (§6), not a task.

### 1.2 The four zero-UUID services make (canonical id, environment) an unsafe derivation key

PRD §6 proposes deriving the deterministic id from *(canonical id,
environment)*. Four services pin the nil UUID as their canonical
`SERVICE_ID`:

```
deploy/k8s/base/atlas-drops.yaml:32              00000000-0000-0000-0000-000000000000
deploy/k8s/base/atlas-world.yaml:32              00000000-0000-0000-0000-000000000000
deploy/k8s/base/atlas-character-factory.yaml:32  00000000-0000-0000-0000-000000000000
deploy/k8s/base/atlas-drop-information.yaml:48   00000000-0000-0000-0000-000000000000
```

`uuid5(NS, "00000000-…-000000000000/pr-1411")` is one value for all four. Any
two of them in a single override set would derive the same id and collide on
the very unique index this design adds — turning a provisioning bug into a
migration-time failure.

The derivation key is therefore **(service type, environment)**, which is also
exactly the unique index's key (§4). See D1.

---

## 2. Decisions

| # | Decision | Rejected alternative |
|---|---|---|
| D1 | `SERVICE_ID = uuidv5(ATLAS_SERVICE_NS, "<service-type>/<environment>")`, derived in **one** place: `deploy/k8s/overlays/pr/scripts/derive-service-id.sh`, called by CI. | (canonical id, environment) — collides across the four nil-UUID services (§1.2). |
| D2 | CI renders the derived id into both the override Deployment's `SERVICE_ID` env and the bootstrap Job's env. `bootstrap.sh` consumes it; `kubectl set env` is deleted. | Bootstrap deriving independently — two implementations that must stay byte-identical forever, whose divergence is silent and reproduces today's symptom exactly. |
| D3 | A **full** unique index on `services (type, environment)`, built by a hand-written migration package after a defensive dedupe. | Partial index exempting the baseline (recommended, overruled — see §4.3 risk). |
| D4 | `Reconcile` treats a projected-empty tenant environment as *legacy* and trusts the header. Genuine non-empty disagreement still drops. | Suppressing empty environments in `ApplyTenant` — loses the distinction between "not projected yet" and "projected as legacy", and leaves no tombstone path for a tenant that legitimately becomes environment-less. |
| D5 | Activation is the last step of `bootstrap.sh`, inside the existing `Sync` hook. | A PostSync Job — PostSync fires only once Argo declares the Sync phase Healthy, the chicken-and-egg documented at `sync-bootstrap.yaml:40-56`. A wedged environment would silently never activate. |
| D6 | FR-1.5 is a readiness gate: the pod stays non-Ready until its own service-config row is projected, and `base/atlas-login.yaml` / `atlas-channel.yaml` gain the `readinessProbe` that `/readyz` never had. | Log-only (a silently idle pod still reports Ready — how this hid for five deploys); panic-on-deadline (turns an ordering race into an outage). |
| D7 | `decide()` returns a typed `gateReason` alongside its verdict; the existing drop counter gains a `reason` label. `decide()` stays pure. | Three separate counters — triples the metric surface and breaks any dashboard on the aggregate. |
| D8 | The activation window (FR-5.4) is accepted and documented, not eliminated. | Two-phase handover (larger than the rest of this task); post-activation settle delay (two minutes on every provision). |
| D9 | Existing sparse environments are **torn down and recreated**. Teardown is the primary reclaim mechanism; the migration's dedupe is a defensive backstop, not a data-repair path. | In-place reconciliation — a throwaway code path that must itself be correct and tested. |

---

## 3. Durable service-config binding (FR-1.1 – FR-1.4)

### 3.1 The derivation

One script, one namespace constant, one algorithm:

```sh
# deploy/k8s/overlays/pr/scripts/derive-service-id.sh
# Usage: derive-service-id.sh <service-type> <environment>
#   derive-service-id.sh login-service pr-1411
#
# ATLAS_SERVICE_NS is a fixed, arbitrary v4 UUID minted once for this
# namespace and NEVER regenerated: every derived id in every environment
# depends on it. It appears here and nowhere else.
ATLAS_SERVICE_NS=<minted-once-at-implementation-time>

python3 - "$ATLAS_SERVICE_NS" "$1/$2" <<'PY'
import sys, uuid
print(uuid.uuid5(uuid.UUID(sys.argv[1]), sys.argv[2]))
PY
```

It is a sibling of the existing `gen-consumer-group-patch.sh` /
`gen-db-name-suffix.sh` generators, runs on the GitHub runner (`python3` is
present), and is unit-testable offline against pinned expected values.

Properties, mapped to the requirements:

- **FR-1.3** (establishable with no prior API call): pure function of two
  strings.
- **FR-2.2** (stable across runs): same inputs, same output, forever — across
  bootstrap runs, across Argo syncs, and across CI re-runs on new commits.
  A CI-time *random* id would satisfy FR-1.2/1.3 but not this: the resolved
  branch is regenerated on every push, so each commit would mint a new row.
- **FR-1.4** (no change outside sparse): the derivation is invoked only on
  the sparse path. `overlays/pr` and `overlays/main` render byte-identical
  manifests.

### 3.2 CI rendering

`.github/workflows/pr-validation.yml`'s `update-pr-overlay` job already
computes `PLACEHOLDER_NS_OVERRIDES`, `PLACEHOLDER_OVERRIDES_JSON` and
`PLACEHOLDER_DELETE_BLOCK` from a single override-set computation
(`:996-1028`). The service-id rendering joins that block — same source of
truth, same substitution pass, so the override set and the ids it implies can
never drift.

For each service type in the override set that carries a `SERVICE_ID` in
base, CI emits:

1. A `patches/service-id.yaml` entry — a strategic-merge patch setting that
   Deployment's `SERVICE_ID` to the derived value. **This is what makes the
   binding survive `selfHeal` (FR-1.2): the value is in the manifest Argo
   renders, so there is nothing for Argo to revert.**
2. A `SERVICE_ID_<TYPE>` env entry on the bootstrap Job in
   `sync-bootstrap.yaml`, so bootstrap POSTs the same id it rendered.

Both are produced by the same loop over the same list. The workflow's
existing unfilled-`PLACEHOLDER_` sweep (`:1035-1043`, `:1154-1162`) catches a
missed substitution before Argo ever sees it.

The patch generator lives beside the other generators as
`overlays/pr/scripts/gen-service-id-patch.sh`, so the mapping
service → service-type is declared once and regenerated rather than
hand-maintained.

### 3.3 bootstrap.sh

`create_service_config` loses its id generation and its Deployment patch:

```diff
-create_service_config() {
-    local tmpl="$1" shape="$2" deployment="$3" body svc_id
-    body=$(build_service_config "$shape" "$tmpl") || return 1
-    svc_id=$(echo "$body" | jq -r '.data.id')
+create_service_config() {
+    local tmpl="$1" shape="$2" svc_id="$3" body
+    #  svc_id is the CI-rendered SERVICE_ID_<TYPE>. Absent means the CI
+    #  rendering did not run — fail the hook rather than mint one here,
+    #  which would resurrect the two-derivation-sites defect (D2).
     ...
-    kubectl set env deployment/"$deployment" SERVICE_ID="$svc_id" ...
```

Consequences:

- `new_uuid()` (`service-config.sh:34-51`) becomes dead and is removed, along
  with its `_SC_UUID_PROC` test seam and the bats cases covering it. The
  guard it existed to provide is replaced by the required-env-var check
  above, which fails the Job instead of warning.
- `build_service_config`'s sparse arm takes the id as a parameter rather than
  calling `new_uuid`.
- The POST becomes an upsert on a known id (§4.2).
- **The three hard-coded `create_service_config` calls become a loop over the
  override set.** Today bootstrap creates login/channel/drops rows
  unconditionally and patches Deployments that may not exist in a sparse
  namespace (the `|| log warn` that swallowed the failure). Creating a row
  only for a service actually deployed here also removes the orphan rows that
  the new unique index would otherwise have to tolerate.
- `atlas-pr-bootstrap`'s Role keeps `deployments` `get`/`list` (the readiness
  observation in §5 needs them) but drops `patch`/`update`, which existed
  only for `kubectl set env` and the restart loop. **The restart loop still
  needs them for `atlas-drops`/`atlas-world`/`atlas-character-factory`, so
  the grant is narrowed only if the sparse override set excludes those; the
  implementation determines this rather than assuming it.** NFR "Security"
  is satisfied either way by an explicit re-examination.

### 3.4 Why this closes the loop

The override's `SERVICE_ID` now names a row whose `environment` column equals
its own. The service-status projection filters by service id
(`subscriber.go:112`), which is strictly stronger than filtering by
environment — so the override projects only its own row, binds only its own
tenant's socket, and emits messages whose `ENVIRONMENT` header and tenant
agree. The PRD's non-goal (adding an `ENVIRONMENT` header to the
service-status topic) holds: with a correct `SERVICE_ID` the header would be
pure defence-in-depth. Diagnosis "Fault 6" is resolved by binding, not by
headers.

---

## 4. Idempotent provisioning (FR-2.1 – FR-2.4)

### 4.1 Two independent guarantees

- **By construction:** the id is a function of (type, environment), so N
  POSTs of the same row are N writes to one id.
- **By constraint:** a unique index on `services (type, environment)` makes
  the invariant unbreakable by any future writer, including one that does not
  know about the derivation.

### 4.2 The upsert

`POST /api/configurations/services` with an explicit `.data.id` already
honours the supplied id ("Use ID from input if provided and valid, otherwise
generate a new one"). Bootstrap's sparse path becomes:

- `GET /api/configurations/services/{derived-id}` — present?
- present → `PATCH` with the merged attributes, skipping the PATCH when the
  live attributes already match (the same guard `upsert_service_config`
  already applies at `bootstrap.sh:466-472`, which also dodges the PATCH
  handler panic on tenant-agnostic configs).
- absent → `POST` with `ENVIRONMENT: $ATLAS_ENVIRONMENT`.

The `ENVIRONMENT` header stays mandatory and server-owned: `administrator.go`
takes `env.MustFromContext(ctx)` for the column and ignores
`attributes.environment` in the body. Omitting it is what put sparse rows in
the legacy `''` environment, invisible to `cleanup.sh`'s scoped reclaim.

FR-2.3 (never touch another environment's rows) is satisfied because every
write is keyed by an id that already encodes this environment, and every
create carries this environment's header.

### 4.3 The unique index, and its risk

`services.Migration` is `db.AutoMigrate(&Entity{}, &HistoryEntity{})`, run at
`atlas-configurations` startup against the **shared baseline database**.
Postgres refuses to build a unique index over existing duplicates, and
AutoMigrate surfaces that as an error — so a duplicate present at rollout
crash-loops the control plane for the baseline **and** every environment
routed through it.

This risk was raised and the full-table index was chosen deliberately (D3).
The design mitigates it in three layers rather than accepting it:

**Layer 1 — process (primary).** Every sparse environment is torn down before
this ships (D9). `cleanup.sh`'s environment-scoped reclaim
(`cleanup.sh:293`, `_dcp_reclaim … /api/configurations/services`) goes through
`ProcessorImpl.DeleteById`, which enqueues the nil-value tombstone
(`processor.go:228`) that a compacted topic requires. This is what FR-2.4
asks for and it already exists; nothing new is built for it. Teardown removes
the known duplicates (5 `login-service`, 5 `channel-service`, 7
`drops-service` rows for `pr-1411`) before the index is ever attempted.

**Layer 2 — a hand-written migration, not AutoMigrate.** A new
`servicesuniq` package, modelled on the existing `environmentcol`
precedent (`environmentcol/migration.go` — raw `db.Exec`, ordered explicitly
in `main.go`'s migration list). It runs **after** `services.Migration` and
after `environmentcol.Migration` (which backfills `environment` to the
baseline, so every row carries a non-empty environment by the time this
runs), and does two things in order:

1. **Dedupe.** For each `(type, environment)` group with more than one row,
   keep the row whose id equals the derived id if one exists, else the row
   with the newest `service_history.created_at`, else the lowest id. Delete
   the losers *through the same transaction that enqueues a tombstone into
   the outbox*, so consumers stop replaying them from the compacted topic —
   never raw `DELETE` alone, which is the operator error the diagnosis
   records.
2. **Index.** `CREATE UNIQUE INDEX IF NOT EXISTS idx_services_type_env ON
   services (type, environment)`.

Both steps are idempotent, so the migration is safe on every restart.

**Layer 3 — a pre-flight the operator can run.** The dedupe's SELECT half is
exposed as a read-only check so the duplicate set can be inspected before the
rollout rather than discovered by a crash-loop. If it reports a group the
dedupe rule cannot resolve unambiguously, the migration fails loudly with
that group named — a failed migration on a *known* row set is recoverable;
a silent wrong-row deletion is not.

**Residual risk, stated plainly.** The index decides on the baseline's behalf
that a second `services` row of one type in one environment is illegal. PRD
§6 records that a co-resident multi-tenant `login-service` row on `main` is
legitimate. If `main` currently holds such a row, the dedupe will delete it
and the index will forbid re-creating it. The pre-flight check exists so this
is discovered before the rollout, but the constraint is the chosen design and
the implementation does not soften it. `main`'s actual row set must be
inspected with the pre-flight before this migration is deployed.

---

## 5. Activation (FR-4.1 – FR-4.6)

### 5.1 Owner and placement

The final step of `bootstrap.sh`, in the existing `Sync` hook at wave 10
(D5). Bootstrap is the only actor that already has every input:

- It runs strictly after the wave-0 precreate Job, which Argo health-checks —
  so `verify_group_offsets` has already proved every override group carries a
  committed offset (FR-4.3, once §6 wires it).
- It already holds `deployments/status` RBAC and already runs `kubectl
  rollout status … --timeout=180s` per override in its restart loop.
- It already PATCHes the environment record (`bootstrap.sh:109-125`), today
  deliberately re-asserting the *current* phase because "it must not promote
  it". That constraint is exactly what this task lifts.

### 5.2 The gate

```
ATLAS_STEP=activate                     # sparse mode only
for each service in the override set:
    kubectl rollout status deployment/<svc> --timeout=<T>   # FR-4.2: ready
    -> failure is FATAL (exit 1), not `|| log warn`
assert every mandatory socket service (login, channel) is in the override
    set and Ready                                            # FR-4.2 / D6
GET  /api/configurations/environments/pr-<N>
if phase == ACTIVE: log "already active"; exit 0             # idempotent
if phase != PROVISIONING: fail loudly                        # DEACTIVATING/DELETED
PATCH phase=ACTIVE
```

Three properties this shape buys:

- **Observed, not assumed (FR-4.2).** Readiness comes from
  `rollout status`; offset initialization comes from the wave-0 Job's own
  exit code, which Argo enforces as a precondition of wave 10 existing at
  all. Neither is inferred.
- **FR-1.5 composes for free.** With D6's readiness gate, a pod whose
  service-config row never arrived is never Ready, so `rollout status` times
  out and activation fails — the environment stays `PROVISIONING` rather than
  advertising a capability it does not have. The two requirements reinforce
  each other rather than being checked twice.
- **Idempotent (NFR).** Every provisioning step re-runs on every Argo sync;
  `UpdateByName` already accepts a same-phase transition and rejects skips
  and reverts, so a re-sync of an ACTIVE environment is a no-op and a
  re-sync during teardown cannot resurrect it.

Failures fail the hook. The `kubectl … || log warn` idiom is removed from
every step on this path — NFR "Failure visibility" is the requirement, and
the swallowed `kubectl set env` warning is the precedent.

### 5.3 Teardown (FR-4.5) and the window (FR-5.4 / D8)

`cleanup.sh` already PATCHes `DEACTIVATING` before destroying any override
workload. Unchanged.

The activation window is accepted and documented. Registry caches update
per-pod off the environment-status topic, so for up to one heartbeat interval
pods disagree about ownership. The bound on the consequence is what makes it
acceptable: during the window a message is either dropped as not-yet-ACTIVE
or processed by the baseline — **never processed twice**, because `IsOwner`
is a single-winner predicate on a record every pod eventually converges to.
In practice the window is empty: the ingress does not route during
PROVISIONING (FR-5.2), so no traffic for the environment exists before the
flip. This closes task-232 PRD open question 3 with an answer rather than
leaving it open.

### 5.4 The precondition this design depends on

Activation is only meaningful if `IsOwner` can succeed, and `IsOwner` reads
`r.self` from `ATLAS_ENVIRONMENT`. The diagnosis found 47 pods across 26
baseline deployments running with `Self() == ""`, because a ConfigMap change
does not roll the Deployments consuming it via `envFrom`. **That is a PRD
non-goal and stays out of scope**, but it is a live-environment precondition:
the end-to-end acceptance criterion cannot pass until those deployments are
rolled. The runbook records this as a pre-test step, and the separate defect
is filed as its own task.

---

## 6. Wiring the offset seeding (FR-4.3, FR-4.4)

One key, rendered by CI into the sparse `atlas-env` ConfigMap:

```
KAFKA_CONSUMER_GROUP: |
  Account Service [pr-1411]
  ...
  Login Service - 0 [pr-1411]
  Channel Service - 0 - 1 [pr-1411]
```

Constraints the existing script imposes on the value, none of them optional:

- **Newline-delimited.** Group names contain spaces and brackets, so a
  space-delimited list is ambiguous (`kafka-precreate.sh:181`).
- **Override groups only.** The value must be absent on `main`, where the
  unset guard is what keeps `--reset-offsets` provably away from real
  committed offsets (NG6, `:176`).
- **Resolved names, never templates.** `consumergroup.Resolve` `Sprintf`s
  `%s` at runtime for `atlas-login` and `atlas-channel`
  (`resolver.go:38-50`), so `"Channel Service - %s [pr-1411]"` seeds a group
  no consumer ever joins — and the pass would report success having done
  nothing. CI must emit **one resolved entry per login/channel instance
  actually run**, derived from the same instance count the overlay uses to
  size those Deployments. This is the single subtle part of §6 and carries
  its own test.

Because the ids are per-Deployment today, the generator that produces
`patches/consumer-group-env.yaml` is the natural source: it already knows
every group name for every service in the environment. CI collects that
output into the ConfigMap key rather than deriving group names a second time.

With the key present, `seed_override_offsets` seeds and
`verify_group_offsets` proves — and FR-4.4's prohibition (never activate
without seeding) is enforced structurally, since a failed verification exits
the wave-0 Job non-zero and Argo never reaches wave 10.

---

## 7. Legacy tenant reconciliation (FR-3.1 – FR-3.3)

### 7.1 Reconcile

`libs/atlas-env/tenants.go`, one arm:

```diff
 	tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
 	if !known {
 		return headerEnv, nil
 	}
+	// A tenant projected with an empty environment is LEGACY — a
+	// pre-#1427 tenant-status event carried no environment attribute and
+	// ApplyTenant stores unconditionally (registry.go:67), so "" means
+	// "this tenant predates environment scoping", not "this tenant
+	// definitely belongs to no environment". Everywhere else in this
+	// codebase "" means legacy, don't filter (FR-1.8); treating it as a
+	// hard mismatch here was the asymmetry that dropped every message a
+	// sparse environment produced against a legacy tenant.
+	if tenantEnv == "" {
+		return headerEnv, nil
+	}
 	if headerEnv == "" {
 		return tenantEnv, nil
 	}
 	if headerEnv != tenantEnv {
 		return "", fmt.Errorf(...)
 	}
```

FR-3.2 is preserved by construction: the mismatch arm is now reached only
when both sides are non-empty, which is the genuine-disagreement case.

`ApplyTenant` is **not** changed (D4). Suppressing empty environments there
would make a legacy tenant indistinguishable from an unprojected one and
would leave no way for a tenant to legitimately become environment-less.

FR-4.6 is untouched: this change only widens the set of messages that resolve
to the header's environment. A deployment with no environment records
projected has no tenants projected either, takes the `!known` arm, and
behaves exactly as today — asserted by test, not by inspection.

### 7.2 Gate observability (D7)

`decide()` stays a pure function of registry state; it gains a second return
value:

```go
type gateReason string

const (
    reasonMismatched gateReason = "mismatched"  // FR-7.7
    reasonStale      gateReason = "stale"       // registry staleness, design §4.3
    reasonNotActive  gateReason = "not_active"  // FR-4.7 / D4
)

func decide(...) (gateVerdict, gateReason)
```

- `atlas_kafka_gate_dropped_unresolvable_total` gains a `reason` label —
  three values, no cardinality risk, and the aggregate still sums.
- The drop log line names the arm instead of emitting identical text for all
  three. The diagnosis records that this ambiguity cost several hours.
- `gateSkipNotOwner`, currently *silent* (counter only), gains a debug log.
  A message nobody processes must leave a trace.

The call site in the consumer loop is the only consumer of the new return
value; no decision logic moves.

---

## 8. Fail-loud binding (FR-1.5, D6)

`atlas-login` and `atlas-channel` are projection-driven: if their row never
arrives they bind no socket and report Ready. That is how a wrong
`SERVICE_ID` survived five deploys.

`atlas-login/main.go:179` already mounts `/readyz` via
`restserver.MountReadiness("/readyz", rt.Ready)` — and
`base/atlas-login.yaml` has **no `readinessProbe`**, so nothing ever calls
it. Two halves:

1. **Register a readiness component** in the configuration projection: not
   ready until a service-config row keyed `service:<SERVICE_ID>` has been
   applied. The projection subscriber already recognises exactly that key
   (`subscriber.go:99`), so the signal is a boolean set at the point the row
   lands — no polling, no new query path.
2. **Add the `readinessProbe`** to `base/atlas-login.yaml` and
   `base/atlas-channel.yaml` pointing at `/readyz`.

Blast radius is the reason this is called out rather than assumed benign: the
probe applies to **every** overlay, baseline included. A baseline pod is
Ready today the moment its HTTP server binds; after this it is Ready only
once its own row is projected. That is the intended semantic — a login server
with no socket bound is not serving — but it must be validated against the
baseline before merge, with `initialDelaySeconds` / `failureThreshold` set
from the observed projection-catch-up time rather than guessed. If the
baseline's catch-up is not reliably bounded, the probe ships behind the
sparse overlay only and the base change is deferred with that stated; the
readiness component itself ships unconditionally.

Silent fallback is removed either way: a pod that has no row logs it at
error, exports it, and is not Ready.

---

## 9. Component map

| Component | Change | Requirements |
|---|---|---|
| `overlays/pr/scripts/derive-service-id.sh` | **new** — the single derivation site | FR-1.3, FR-2.2, D1 |
| `overlays/pr/scripts/gen-service-id-patch.sh` | **new** — renders the per-service strategic-merge patch | FR-1.2 |
| `overlays/pr-sparse/patches/service-id.yaml` | **new** — generated; `SERVICE_ID` per override Deployment | FR-1.1, FR-1.2 |
| `.github/workflows/pr-validation.yml` | derives ids in the existing override-set block; renders `SERVICE_ID_*` onto the bootstrap Job; renders `KAFKA_CONSUMER_GROUP` into sparse `atlas-env` | FR-1.3, FR-4.3 |
| `overlays/pr-sparse/kustomization.yaml` | `atlas-env` gains the newline-delimited `KAFKA_CONSUMER_GROUP` key | FR-4.3 |
| `atlas-pr-bootstrap/scripts/service-config.sh` | id becomes a parameter; `new_uuid` removed | FR-2.2 |
| `atlas-pr-bootstrap/scripts/bootstrap.sh` | upsert on the derived id; `kubectl set env` deleted; per-override loop; activation step | FR-1.1, FR-2.1, FR-4.1, FR-4.2 |
| `overlays/pr-sparse/sync-bootstrap.yaml` | `SERVICE_ID_*` env; Role re-examined | FR-1.3, NFR security |
| `atlas-configurations/.../servicesuniq/` | **new** — dedupe + unique index, raw SQL after `environmentcol` | FR-2.1, D3 |
| `libs/atlas-env/tenants.go` | legacy-empty arm in `Reconcile` | FR-3.1, FR-3.2 |
| `libs/atlas-kafka/consumer/gate.go` | typed reason; `reason` label; not-owner debug log | FR-3.3, NFR observability |
| `atlas-login`, `atlas-channel` (projection + base manifests) | readiness component; `readinessProbe` | FR-1.5 |
| `docs/runbooks/sparse-environments.md` | teardown-before-upgrade; the `ATLAS_ENVIRONMENT` roll precondition; how to verify a seeded group and a bound row | D9, §5.4 |

Untouched, deliberately: the `NS_*` routing table, ingress header defaulting,
tenant provisioning (task-242), the service-status topic's headers (PRD
non-goal, and §3.4 explains why it stays unnecessary), and the ConfigMap
propagation defect (PRD non-goal, filed separately).

---

## 10. Data flow, end to end

```
CI (update-pr-overlay)
  override set ──┬─> NS_* / delete block / overrides JSON     (existing)
                 ├─> derive-service-id.sh per type
                 │     └─> patches/service-id.yaml            SERVICE_ID (Deployment)
                 │     └─> sync-bootstrap.yaml env            SERVICE_ID_<TYPE> (Job)
                 └─> resolved group names
                       └─> atlas-env ConfigMap                KAFKA_CONSUMER_GROUP
                                    │
Argo sync                           │
  wave 0  atlas-kafka-precreate  ───┘
            precreate_topics; seed_override_offsets; verify_group_offsets
            └─ exit != 0 stops the sync here                  (FR-4.4 structural)
  wave 1  atlas-environment-record  POST phase=PROVISIONING
  wave 10 override Deployments  ──  start with the derived SERVICE_ID
          atlas-pr-bootstrap (Sync hook, parallel)
            upsert services row @ derived id, ENVIRONMENT header
            rollout status per override                       (FR-4.2 ready)
            PATCH phase=ACTIVE                                (FR-4.1)
                                    │
Runtime                             │
  login projects service:<derived-id> only  ─> binds only its own tenant
  emits ENVIRONMENT: pr-N + own tenant      ─> Reconcile agrees
  gate: not mismatched, not stale, ACTIVE, owner ─> gateProcess
```

---

## 11. Testing

**Unit / module-local**

- `derive-service-id.sh`: pinned expected values for several
  (type, environment) pairs; the four nil-UUID services derive four
  *distinct* ids (the §1.2 regression).
- `service-config.sh` bats: sparse build takes the supplied id verbatim;
  absent id is a hard failure, not a warning; isolated-mode output is
  byte-identical to today's.
- `servicesuniq`: dedupe keeps the derived-id row; keeps newest on tie;
  deletes losers *and* enqueues their tombstones; index creation is
  idempotent; a second run is a no-op. SQLite shadow entities, per the
  `environmentcol/migration_test.go` precedent.
- `Reconcile`: empty tenant environment + non-empty header → header trusted;
  two different non-empty → still `ErrEnvironmentMismatch`; unknown tenant →
  header trusted (unchanged).
- `decide`: each of the three drop arms returns its own reason; verdicts are
  unchanged from today for every existing case.
- **FR-4.6 regression, by test not inspection:** a registry with zero records
  projected produces identical verdicts before and after every change in this
  task, for a message with an empty header and for one with a non-empty
  header.
- Readiness component: not ready with no row; ready after the matching
  `service:<id>` key is applied; still not ready after a *different*
  service's row is applied.

**Rendered-manifest**

- Sparse render: each override Deployment's `SERVICE_ID` equals
  `derive-service-id.sh <type> pr-<N>`; the bootstrap Job's `SERVICE_ID_<TYPE>`
  matches it.
- **`overlays/main` and `overlays/pr` renders are byte-identical to
  pre-change, except the intended `readinessProbe` lines** (FR-1.4 and the
  acceptance criterion on isolated mode).
- The workflow's unfilled-`PLACEHOLDER_` sweep still passes.
- `KAFKA_CONSUMER_GROUP` is present in sparse `atlas-env`, absent in `main`'s,
  newline-delimited, and contains no `%s`.

**Integration / live**

- Bootstrap run three times against one environment → exactly one row per
  service type (FR-2.1), same ids each time (FR-2.2), baseline rows untouched
  (FR-2.3).
- Argo hard refresh + re-sync → `SERVICE_ID` unchanged (FR-1.2). This is the
  criterion today's `kubectl set env` fails.
- The override's login socket binds no tenant whose `environment` differs
  from the pod's.
- Environment reaches `ACTIVE` with no manual PATCH; and does **not** reach
  it while any override is unready or any group lacks committed offsets
  (tested by holding one Deployment unready).
- Teardown deactivates first and reclaims every row the environment created.
- **End to end: connect to a freshly created sparse PR environment, create an
  account, log in, reach character select — with no manual database,
  `kubectl`, or Argo intervention.** This is the criterion the task exists
  for; §5.4's `ATLAS_ENVIRONMENT` roll is a documented precondition of the
  live run, not of the code.

---

## 12. Rollout

Ordering matters, because the migration is the one irreversible step.

1. Run the §4.3 Layer-3 pre-flight against the baseline database and read the
   duplicate set. If it names a group the dedupe rule cannot resolve —
   including a legitimate co-resident `login-service` row on `main` — stop
   and re-decide D3 before merging.
2. Tear down every existing sparse environment (D9). `cleanup.sh` reclaims
   their rows through the tombstone-emitting API path.
3. Roll the 26 baseline deployments missing `ATLAS_ENVIRONMENT` (§5.4) — a
   live-state precondition, not part of this change.
4. Merge. `atlas-configurations` restarts, `servicesuniq` dedupes and builds
   the index.
5. Open a fresh PR in sparse mode and run the end-to-end criterion.

Rollback: reverting the code restores random ids and the `kubectl set env`
patch, but the unique index survives a code revert. If the index must go, it
is dropped explicitly — the migration is idempotent, not reversible, and that
is the residual cost of D3.

---

## 13. Requirements coverage

| Requirement | Where |
|---|---|
| FR-1.1 – FR-1.4 | §3 |
| FR-1.5 | §8, and §5.2 (composes with the activation gate) |
| FR-2.1 – FR-2.3 | §4.1, §4.2, §4.3 |
| FR-2.4 | §4.3 Layer 1 — existing `cleanup.sh` + `DeleteById` tombstone |
| FR-3.1, FR-3.2 | §7.1 |
| FR-3.3 | §7.2 |
| FR-4.1, FR-4.2 | §5.1, §5.2 |
| FR-4.3, FR-4.4 | §6 (and §1.1 — the mechanism already exists) |
| FR-4.5 | §5.3 — unchanged |
| FR-4.6 | §7.1, and the §11 regression test |
| NFR backward compat | §3.1 (FR-1.4), §11 rendered-manifest tests |
| NFR observability | §7.2 |
| NFR idempotency | §4.2, §5.2 |
| NFR failure visibility | §3.3, §5.2 — every `|| log warn` on the provisioning path becomes fatal |
| NFR security | §3.3 — Role re-examined once `kubectl set env` is gone |
| PRD OQ1 | D1 — CI, one site |
| PRD OQ2 | D3 — full unique index, dedupe first |
| PRD OQ3 | D5 — `bootstrap.sh` tail |
| PRD OQ4 | D8 — window accepted and documented; closes task-232 OQ3 |
| PRD OQ5 | §1.1 — absorbed; it is one ConfigMap key, not a task |
| PRD OQ6 | D9 — recreate |
| PRD OQ7 | D6 — readiness gate + probe |
