# task-243 — Implementation Context

Read this before Task 1 of `plan.md`. It records what the plan assumes, three
places where the plan's *mechanism* differs from `design.md` (no decision is
changed), and the facts a task's brief cannot carry on its own.

---

## The one-line problem

An override Deployment bakes the *baseline's* pinned `SERVICE_ID`
(`deploy/k8s/base/atlas-login.yaml:45`), because the sparse row's id does not
exist until bootstrap mints it at runtime. `bootstrap.sh` compensates with
`kubectl set env`, and Argo's `selfHeal: true` reverts that on the next
reconcile. The fix is to make the id **derivable** — a pure function of
(service type, environment) — so CI can render it into the manifest, where
there is nothing for Argo to revert.

Everything else in this task follows from that or sits behind it.

---

## Deviations from `design.md`

All three are mechanism, forced by invariants already in the tree. Flag them in
review; none re-opens a decision.

### Deviation 1 — the derivation script lives in `tools/`, not `overlays/pr/scripts/`

`design.md` §3.1 placed `derive-service-id.sh` beside the other overlay
generators. The plan puts it at `tools/derive-service-id.sh`.

Reason: `tools/verify.sh:488-500` gates every changed `tools/*.sh` through
`shell-guard.sh --require-shellcheck` and auto-selects its `_test.sh` suite.
Nothing gates `deploy/k8s/overlays/pr/scripts/`. This script is the single
source of every sparse environment's service binding; an ungated single point
of failure is the wrong trade. CI can invoke it from either path
(`pr-validation.yml:1105` already invokes an overlay script), so nothing else
changes. Its single-site property (D2) is untouched.

### Deviation 2 — `KAFKA_CONSUMER_GROUP` is patched onto the precreate Job, not added to the `atlas-env` ConfigMap

`design.md` §6 puts the newline-delimited group list in the sparse `atlas-env`
ConfigMap. The plan patches it onto the `atlas-kafka-precreate` Job instead.

Reason: `atlas-env` is consumed via `envFrom` by **every** container in the
namespace (`deploy/k8s/base/atlas-kafka-precreate.yaml:45-47` is one of many).
A container-level `env` entry wins over `envFrom`, so services covered by
`patches/consumer-group-env.yaml` would be fine — but any container **not**
covered by that patch would silently take the whole multi-line group list as
its own consumer group. Patching the one Job that reads the variable is
contained and has an exact precedent: the inline JSON-6902 patch at
`deploy/k8s/overlays/pr-sparse/kustomization.yaml:168-188`, which adds
`ATLAS_MODE`/`ATLAS_ENVIRONMENT` to the bootstrap Job for the same reason
(`sync-bootstrap.yaml` is mirror-locked).

### Deviation 3 — the `%s` in the templated consumer groups is the SERVICE_ID, not a channel number

`design.md` §6 says CI must emit "one resolved entry per login/channel instance
actually run, derived from the same instance count the overlay uses to size
those Deployments."

That is wrong on the evidence. Both templated callers pass
`serviceId.String()`:

- `services/atlas-login/atlas.com/login/main.go:49` — `consumerGroupIdTemplate = "ChannelConnect Service - %s"`, resolved at `:55-56` with `serviceId.String()`
- `services/atlas-channel/atlas.com/channel/main.go:175` — `consumerGroupIdTemplate = "Channel Service - %s"`, resolved at `:181-182` with `serviceId.String()`

`SERVICE_ID` is one env var per pod, so there is exactly **one** group per
Deployment, and the value to fill `%s` with is the id the same CI loop just
derived. This makes §6's "single subtle part" mechanically trivial and removes
the instance-count coupling entirely. It also means the group name is only
correct if the derived `SERVICE_ID` is — the two are rendered together in one
loop by construction (Task 12).

Related correction: `design.md` §6's example group names read `[pr-1411]`. The
actual suffix is `PLACEHOLDER_ATLAS_ENV` → `sha256("pr-<N>")[:4]`
(`pr-validation.yml:879`), a 4-hex token. Task 12 reads the resolved value out
of the already-substituted `patches/consumer-group-env.yaml` rather than
constructing it, so this never has to be re-derived.

---

## Corrections `design.md` already made to the PRD

Both stand; do not re-investigate them.

- **Task-232 Task 45 is implemented.** `seed_group` /
  `seed_override_offsets` / `verify_group_offsets` all exist in
  `deploy/k8s/base/kafka-precreate.sh:132-246`. The PRD's grep against
  `atlas-kafka-precreate.yaml` was accurate; the script had been moved into a
  ConfigMap-mounted file. The gap is one wire (Deviation 2's patch), not a task.
- **The derivation key is (service type, environment), not (canonical id,
  environment).** Four base Deployments pin the nil UUID as `SERVICE_ID`
  (`atlas-drops`, `atlas-world`, `atlas-character-factory`,
  `atlas-drop-information`), so a canonical-id key collides across all four.
  Task 1's test asserts the four derive four distinct ids.

---

## Pinned values

The namespace constant appears in exactly two places —
`tools/derive-service-id.sh` (Task 1) and `servicesuniq`'s keeper rule
(Task 4) — and each cross-references the other:

```
ATLAS_SERVICE_NS = c8f90111-a0cf-513e-95e6-c54609e5dec0
                 = uuid5(NAMESPACE_DNS, "service-config.atlas.chronicle20")
```

It is reproducible rather than arbitrary so it can be re-derived if the line is
lost, but it must **never be regenerated** — changing it re-keys every sparse
environment's service-config row.

Derived ids used as test fixtures across Tasks 1, 4, 5, 6 and 8:

| type | environment | id |
|---|---|---|
| `login-service` | `pr-1411` | `6439ca9c-d28d-5db9-821b-8dd93d318a25` |
| `channel-service` | `pr-1411` | `5a86d8e6-3167-5e74-9fc5-021d94001da2` |
| `drops-service` | `pr-1411` | `cbce66aa-facb-5766-8583-84c3478a6ba2` |
| `world-service` | `pr-1411` | `f80c02bc-2ac4-598e-a8e6-298e7e1d72b5` |
| `character-factory` | `pr-1411` | `a0bb4ad4-0c2b-5941-b297-fa4b6cf9403e` |
| `drops-information-service` | `pr-1411` | `87d2d5a6-f37d-5a1e-8e81-bfed3a239e69` |
| `login-service` | `pr-999` | `e7ae96a2-c484-5617-8e28-2178b60a8378` |
| `login-service` | `main` | `78d4984e-22dd-5284-8729-61627a5e603f` |
| `channel-service` | `pr-999` | `2e3b50b4-fb89-5af0-bb51-19749ecb734f` |
| `channel-service` | `main` | `dff6f040-d4aa-51fa-914b-ff1dff6f6a76` |

Every value above was computed with `python3`'s `uuid.uuid5` against the
namespace constant. If Task 1's test disagrees with this table, the script is
wrong — not the table.

The six base Deployments that carry `SERVICE_ID` / `SERVICE_TYPE`, verified by
grep over `deploy/k8s/base/*.yaml`:

| Deployment | `SERVICE_TYPE` | base `SERVICE_ID` |
|---|---|---|
| `atlas-login` | `login-service` | `e7fb1d7e-47b8-46bd-97dc-867d93530856` |
| `atlas-channel` | `channel-service` | `e7fb1d7e-47b8-46bd-97dc-867d93530000` |
| `atlas-drops` | `drops-service` | nil UUID |
| `atlas-world` | `world-service` | nil UUID |
| `atlas-character-factory` | `character-factory` | nil UUID |
| `atlas-drop-information` | `drops-information-service` | nil UUID |

---

## Invariants a task will trip over

- **`sync-bootstrap.yaml` is mirror-locked.** `tools/pr-sparse-mirror-guard.sh`
  binds nine files byte-identical between `overlays/pr` and `overlays/pr-sparse`.
  `sync-bootstrap.yaml` and `patches/consumer-group-env.yaml` are both on that
  list. Any edit to either must land identically in both copies (Task 9's RBAC
  comment) — or, better, land as an overlay-only inline patch in
  `pr-sparse/kustomization.yaml` (Tasks 11–12).
- **`kustomize build deploy/k8s/overlays/pr-sparse` runs unsubstituted** in
  `tools/overlay-env-guard.sh` and `tools/sparse-baseline-scoping-guard.sh`.
  New CI tokens must therefore be YAML *comments* when unfilled — which is why
  Task 11 uses `#PLACEHOLDER_…` list-position comments rather than a
  `patches: - path:` entry pointing at a generated file that does not exist in
  the repo.
- **The unfilled-`PLACEHOLDER_` sweep** (`pr-validation.yml:1035-1043`) fails
  the job on any surviving token inside `$OVERLAY_DIR`. The two new tokens live
  only in `pr-sparse/`, and isolated mode never substitutes there — so they must
  **not** be added to `overlays/pr/kustomization.yaml`.
- **`ENVIRONMENT` is server-owned on writes.** `administrator.go`'s INSERT takes
  `env.MustFromContext(ctx)` and the body's `attributes.environment` is ignored.
  Omitting the header is what put every sparse row in the legacy `''`
  environment, invisible to `cleanup.sh`'s scoped reclaim. `bootstrap.sh`'s
  `ENV_HEADER` array (`:42-60`) already handles this and is empty in isolated
  mode — use it, do not hand-roll a `-H`.
- **A no-op PATCH on a tenant-agnostic service config panics the handler**
  (`reflect.Value.Set using unaddressable value`). `upsert_service_config`
  dodges it with a `jq -cS` equality check before PATCHing
  (`bootstrap.sh:466-472`); Task 9's sparse upsert must keep that guard.
- **`env_record_patch` zeroes omitted attributes.** Send all five every time,
  reading the other four out of the current record — the pattern
  `record_environment_tenant` already uses (`bootstrap.sh:114-126`).
- **`bats` must be installed** or the `atlas-pr-bootstrap` suite is a hard
  failure, not a skip (`tools/verify.sh:520-528`). Tasks 8–10 all touch that
  service.

---

## Task sizing

Every task is at or under six files and one service. Three notes:

- **Tasks 5 and 6** are the same mechanical change in two services. They are
  split rather than batched because they are two review surfaces and two module
  roots; Task 6 explicitly says to read Task 5's diff and mirror it.
- **Task 7** (the `readinessProbe`) is split off from Tasks 5–6 deliberately.
  Its blast radius is every overlay including the baseline — `design.md` §8
  requires it be validated against the baseline before merge, and a separate
  task keeps that gate visible instead of buried in a Go diff.
- **Tasks 9 and 10** both edit `bootstrap.sh`. They are sequential, not
  parallel. Task 10 depends on Task 9's override-set read.

Nothing was deliberately left oversized.

---

## What is out of scope, and stays out

- Adding an `ENVIRONMENT` header to `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`.
  The projection already filters by service id
  (`.../projection/subscriber.go:111`), which is strictly stronger. With a
  correct `SERVICE_ID` the header is pure defence-in-depth.
- The `NS_*` REST routing table, ingress header defaulting, tenant provisioning
  — task-242 owns those.
- Making sparse the default PR mode (task-232 Task 54).
- **The ConfigMap-propagation defect.** A ConfigMap change does not roll the
  Deployments consuming it via `envFrom`; `#1427` added `ATLAS_ENVIRONMENT`,
  which the entire ownership gate depends on, and it reached zero running pods.
  This is real and dangerous, it is a **PRD non-goal**, and it must be filed as
  its own task. It is also a *live-state precondition* of this task's
  end-to-end acceptance criterion — Task 13 records that in the runbook, and
  the criterion cannot be claimed until the affected baseline deployments are
  rolled.
- Retroactively repairing PR #1411's live state. It has been hand-patched during
  diagnosis; tear it down and recreate it to validate this work.

---

## Residual risk, stated plainly

Task 4's unique index on `services (type, environment)` decides on the
baseline's behalf that a second row of one type in one environment is illegal.
`prd.md` §6 records that a co-resident multi-tenant `login-service` row on
`main` is legitimate. If `main` holds such a row, the dedupe deletes it and the
index forbids re-creating it.

The Layer-3 pre-flight exists so this is discovered *before* the rollout rather
than as an `atlas-configurations` crash-loop that takes the baseline and every
environment routed through it down with it. `design.md` §12 step 1 is not
optional: run the pre-flight against the baseline database and read the
duplicate set before merging. If it names a group the dedupe rule cannot
resolve, stop and re-decide D3.

The migration is idempotent, not reversible. Reverting the code restores random
ids; the index survives the revert and must be dropped explicitly if it has to
go.
