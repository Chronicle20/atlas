# Sparse-Mode Tenant Provisioning — Design

Task: task-242-sparse-tenant-provisioning
PRD: `docs/tasks/task-242-sparse-tenant-provisioning/prd.md` (v1, approved)
Status: Draft for plan phase
Created: 2026-08-19

---

## 1. Scope of this document

The PRD names three defects and five requirement groups. This document
settles *how* each is implemented, records the alternatives rejected, and
reports four things the PRD could not have known because they only surface
once the code is read end to end:

- **F1** — the `main` environment record is a hard prerequisite of FR-1, not
  merely of FR-4.4. Without it, setting `ATLAS_ENVIRONMENT=main` makes main
  *worse*, not better.
- **F2** — main's environment heartbeat is currently dead, which is a second,
  independent reason sparse Kafka traffic is dropped.
- **F3** — main's own periodic per-tenant loops already stop doing work
  whenever any environment record exists. This is live today.
- **F4** — the PRD's "full per-tenant baseline restore" premise is
  contradicted by the merged fix in `c5e88320a`, and two acceptance criteria
  built on it are unachievable. §6 restates them.

Everything in §3–§5 is grounded in file:line evidence quoted inline. Where a
claim is a hypothesis to confirm against the live cluster, it says so.

---

## 2. The two planes, restated for this task

Task-232's model, as it actually behaves in code:

| Plane | Boundary | Where it lives |
|---|---|---|
| Control | `environment` column + `scope.Strict` | `atlas-tenants`, `atlas-configurations` only |
| Data | `tenant_id` | every data-plane service's tables |

Two consequences drive the whole design:

1. **Only two services enforce the control-plane boundary.**
   `libs/atlas-rest/server/handler.go:50-59` is explicit that every other
   service "has no such layer: admitting the header there means the request
   is served exactly as any untagged request would be". So stamping
   `ENVIRONMENT` at the ingress changes observable behaviour in exactly two
   services, and is inert everywhere else. That is what makes FR-4 a
   low-blast-radius change.

2. **The empty environment is a real, load-bearing value.**
   `scope.Strict` (`services/atlas-tenants/atlas.com/tenants/scope/scope.go:29`)
   applies *no filter* for `""`, and `decide` in
   `libs/atlas-kafka/consumer/gate.go:57` returns `gateProcess` for an
   untagged message. Isolated mode and local compose depend on this. Every
   change below must leave `""` reachable.

---

## 3. What is actually broken (evidence)

### 3.1 FR-1 — baseline pods do not know their own environment

`env.Self()` reads `ATLAS_ENVIRONMENT` (`libs/atlas-env/env.go:29,78-81`).
`deploy/k8s/overlays/main` sets `ATLAS_ENV` (a different variable, used for
consumer groups and pod labels — `overlays/main/patches/atlas-env-env.yaml`)
but never `ATLAS_ENVIRONMENT`. `grep -rn ATLAS_ENVIRONMENT deploy/` returns
hits only under `pr-sparse` and `pr-cleanup`. So on every main pod,
`r.self == ""`.

Three separate failures follow from that one fact.

**(a) Ownership — the defect the PRD names.**
`MapRegistry.IsOwner` (`libs/atlas-env/registry.go:216-217`) ends with
`return rec.Baseline == r.self`. The sparse record carries
`baseline: PLACEHOLDER_BASELINE_ENVIRONMENT` → `main`
(`overlays/pr-sparse/environment-record.yaml:71`, CI derivation at
`.github/workflows/pr-validation.yml:921`). `"main" == ""` is false, so a
baseline pod owns nothing in `pr-<N>` and `decide` returns `gateSkipNotOwner`
(`gate.go:68`).

**(b) F3 — main's own periodic loops stop, today.**
`MapRegistry.EnvironmentsOwnedBy` (`registry.go:225-247`) returns `[""]`
*only* while `len(r.records) == 0`. The moment any sparse PR registers a
record, main's pods have a non-empty record set, and the loop admits a record
only when `rec.Baseline == r.self`. With `self == ""` and every record's
baseline `"main"`, the result is an **empty slice** — and there is no record
named `main` to fall back on. `libs/atlas-service/foreach.go:86` then iterates
nothing, so every `ForEachOwnedEnvironment` ticker in main silently does no
work for the lifetime of any sparse PR environment.

This is not a sparse-environment bug. It is a live main-environment
regression, and it is the strongest argument for treating FR-1 as urgent.
*Hypothesis to confirm on the cluster: at least one main service's periodic
loop measurably stalled while PR #1411's environment record existed.*

**(c) F2 — the heartbeat never fires on main.**
`environments.StartHeartbeat` republishes `envlib.Self()`
(`services/atlas-configurations/.../environments/heartbeat.go:27`). With
`Self() == ""`, `Republish("")` calls `GetByName("")`, which fails, so the
30-second heartbeat logs `environment heartbeat failed` and publishes
*nothing*. Consumers treat any message on the topic as liveness (same file,
lines 13-17), so every pod's registry ages into `Stale()`. Once stale,
`decide` (`gate.go:63`) drops any `msgEnv != self` as
`gateDropUnresolvable` — a *different* verdict and a *different* metric from
the `skipped_not_owner` the PRD's NFR section expects to observe.

Practical consequence for acceptance: **watch both**
`atlas_kafka_gate_skipped_not_owner_total` and
`atlas_kafka_gate_dropped_unresolvable_total`. The PRD's observability
section names only the first.

### 3.2 FR-2 — bootstrap adopts the baseline's tenant

`services/atlas-pr-bootstrap/scripts/bootstrap.sh:234-238` lists
`/api/tenants` with **no** `ENVIRONMENT` header and selects on
`(region, majorVersion, minorVersion)`. Against a shared database, the
baseline's canonical tenant always matches that triple, so `existing` is
main's tenant id and the sparse environment adopts it. Everything downstream
— config clone, service configs, seeds, gameplay — then writes under main's
tenant, and `sweep-tenant` has nothing distinct to reclaim.

The header is the whole fix: `getAll`
(`services/atlas-tenants/atlas.com/tenants/tenant/provider.go:27-32`) already
applies `scope.Strict` to the caller's environment, and `Entity.Environment`
is written from request context on create. No service change is required.

### 3.3 FR-3 — nothing ever writes `environments.tenant`

The column exists (`environments/entity.go:22`), the PATCH route exists
(`environments/resource.go:29`), `UpdateByName` already carries omitted
fields forward from the existing record (`environments/processor.go:236-255`),
and `cleanup.sh:355-358` reads it. No caller writes it. `bootstrap.sh` never
mentions `/api/configurations/environments` at all.

### 3.4 FR-4 — browser traffic is untagged

`deploy/k8s/base/atlas-ingress.yaml:50` is
`proxy_set_header ENVIRONMENT $http_environment;` — unconditional
passthrough. A browser sends no such header, so the SPA's `GET /api/tenants`
arrives with `env == ""` and `scope.Strict` applies no filter: the unfiltered
union of every environment's tenants.

---

## 4. Design decisions

### D1 — The `main` environment record is a first-class prerequisite, created by a deploy-time Job

**Decision.** Add `deploy/k8s/overlays/main/environment-record.yaml`: an
idempotent GET-then-POST Job that creates
`{name: main, baseline: main, namespace: atlas-main, tenant: "", overrides: {}, phase: ACTIVE}`.

**Why this is not optional.** Three independent mechanisms need it:

| Mechanism | Without the record |
|---|---|
| `EnvironmentsOwnedBy` (`registry.go:225`) | main's own periodic loops iterate nothing (F3) — and setting `self="main"` does **not** fix this on its own |
| `StartHeartbeat` (`heartbeat.go:27`) | `Republish("main")` fails; no heartbeat; every registry goes stale (F2) |
| `ParseEnvironment` (`handler.go:68-73`) | an ingress stamping `main` 400s every request (FR-4.4) |

This answers PRD **Q2**: the record does **not** exist. Nothing in
`deploy/k8s/overlays/main/`, `deploy/k8s/base/`, or CI creates one — the only
`environments` POST in the repo is `pr-sparse/environment-record.yaml`. CI
derives the baseline *name* from the namespace
(`pr-validation.yml:916-921`) and never consults the registry, exactly as the
PRD suspected.

**The Job must bypass the ingress.** Once FR-4 lands, main's ingress stamps
`ENVIRONMENT: main` on everything — including the very request that would
create the `main` record, which `ParseEnvironment` would then 400. The Job
therefore POSTs to the ClusterIP directly:
`http://atlas-configurations.atlas-main.svc.cluster.local:8080/api/configurations/environments`.
An in-cluster caller that sets no header is the legacy `""` environment,
which `ParseEnvironment` admits unconditionally.

**Placement.** `argocd.argoproj.io/sync-wave: "11"` (base pushes every
Deployment to wave 10 — `base/kustomization.yaml:93-106`), with
`sync-options: Force=true,Replace=true` and a bounded retry loop, mirroring
`pr-sparse/environment-record.yaml`'s idiom.

**Known window.** On a *fresh* `atlas-main` bring-up, atlas-ingress becomes
Ready at wave 10 and starts stamping `main` before the wave-11 Job creates
the record. Every REST call 400s for that interval. Argo gates wave 11 on
wave-10 health, so the window is seconds and bring-up-only, and it
self-heals. Accepted.

**Alternative rejected: seed the record from atlas-configurations' migration.**
The service already has `environmentBaseline()`
(`environmentcol/migration.go:34-40`) and owns the `environments` table, so a
migration-time upsert would close the bring-up window entirely — the record
would exist before the first request is served. Rejected because it puts a
deployment topology fact into service code and touches a service PRD §7
declares unchanged; the window it closes is bounded and recoverable. **If the
bring-up window is judged unacceptable, this is the alternative to take** —
see §9.

**`tenant` stays `""` on main's record.** `do_sweep_tenant` is sparse-only
(`cleanup.sh:335-341`) and never runs against main. Populating it would add a
value nothing reads and one more way to point a sweep at main's tenant.

### D2 — FR-1 is one ConfigMap literal

Add `ATLAS_ENVIRONMENT=main` to the `atlas-env` `configMapGenerator` literals
in `deploy/k8s/overlays/main/kustomization.yaml` (the `behavior: replace`
block starting line 36).

Every service Deployment consumes the whole ConfigMap via
`envFrom: configMapRef: name: atlas-env` (`base/atlas-ban.yaml:21-23` and
uniformly across base), so one literal reaches all 64 deployments. No
per-Deployment patch, and nothing to keep in sync with
`patches/atlas-env-env.yaml`.

`overlays/pr` has no `configMapGenerator` entry for `atlas-env` and never
defines `ATLAS_ENVIRONMENT`, so isolated pods keep `Self() == ""` and
FR-1.5 holds by construction.

**FR-1.2 conformance.** The literal `main` equals
`BASELINE_NAMESPACE#atlas-` for `namespace: atlas-main`
(`overlays/main/kustomization.yaml:8`), matching `pr-validation.yml:921`.
A rendered-manifest assertion pins this (§7).

**Free side effect, worth stating.** `BackfillEnvironment` in both services
(`environmentcol/migration.go:16`, `tenant/environment_migration.go:12`)
already defaults to `"main"` when `ATLAS_ENVIRONMENT` is unset, so main's
existing rows are *already* stamped `main`. Setting the variable is a no-op
for the backfill, not a re-stamp — and because the migration re-runs on every
boot, the pod restart this change causes also sweeps up any rows written with
`environment = ''` since the last restart. That ordering is what makes D5
safe.

### D3 — The ingress default is one envsubst'd `map`, driven by the ConfigMap key that already exists

`nginx.conf` is mounted directly at `/etc/nginx/nginx.conf` via `subPath`
(`base/atlas-ingress.yaml:249-251`), so it is **not** envsubst'd;
`routes.conf.template` is (lines 252-254, filter at line 105-108).

**Decision.** Add a second template and a `map`:

1. New file `deploy/k8s/base/env-default.conf.template`, added to the base
   `configMapGenerator` alongside `routes.conf.template`:

   ```nginx
   map $http_environment $atlas_environment {
       ""      "${ATLAS_ENVIRONMENT_DEFAULT}";
       default $http_environment;
   }
   ```

2. Mount it into `/etc/nginx/templates/` so the nginx entrypoint renders it
   to `/etc/nginx/conf.d/env-default.conf`.

3. In `nginx.conf`'s `http {}` block, above `log_format`:
   `include /etc/nginx/conf.d/env-default.conf;`

4. Change the server block's line 50 to
   `proxy_set_header ENVIRONMENT $atlas_environment;` and the access log's
   `env=$http_environment` to `env=$atlas_environment`, so the log records
   the *resolved* environment (PRD NFR-Observability).

5. Extend `NGINX_ENVSUBST_FILTER` to
   `"POD_NAMESPACE|NS_|ATLAS_ENVIRONMENT_DEFAULT"` (line 108).

6. On the ingress container, add:

   ```yaml
   - name: ATLAS_ENVIRONMENT_DEFAULT
     valueFrom:
       configMapKeyRef:
         name: atlas-env
         key: ATLAS_ENVIRONMENT
         optional: true
   ```

**Why this shape wins.** The per-overlay default becomes *zero overlay
patches*. It is sourced from the same `atlas-env` / `ATLAS_ENVIRONMENT` key
that `env.Self()` reads, so the ingress default **cannot drift from what the
pods believe they are**. And each overlay lands where FR-4.2 wants it for
free:

| Overlay | `atlas-env.ATLAS_ENVIRONMENT` | Rendered default |
|---|---|---|
| base / local compose | absent | `""` → today's passthrough-with-empty |
| `overlays/main` | `main` (D2) | `main` |
| `overlays/pr-sparse` | `pr-<N>` (`kustomization.yaml:266`) | `pr-<N>` |
| `overlays/pr` (isolated) | absent | `""` — unchanged, and never stamps an unregistered id |

FR-4.3 (caller wins) is the `default` arm of the map. Empty-valued
`proxy_set_header` is not emitted by nginx, so the base case is byte-identical
to today.

**Mirror guard.** Nothing here touches any of the nine files in
`tools/pr-sparse-mirror-guard.sh`'s `MIRRORS` array (lines 33-43) — in
particular not `patches/ingress-host.yaml`, which the PRD flagged. The
`pr-sparse` side needs no new file at all.

**Alternatives rejected.**

- *Patch `data.nginx.conf` per overlay.* A strategic merge on a ConfigMap
  data key replaces the whole string — three full copies of a 60-line nginx
  config, drifting independently. Rejected.
- *A separate small ConfigMap holding the `map`, replaced per overlay.* Works,
  but needs a new volume, a new mount, and a patch in each overlay; and the
  default can then disagree with `ATLAS_ENVIRONMENT`. Strictly worse than
  sourcing one env var.
- *Put `proxy_set_header` in `routes.conf.template`.* `gen-routes.sh` owns
  that file, and a second same-level `proxy_set_header ENVIRONMENT` would
  accumulate rather than override. Rejected.
- *Resolve `env == ""` to `env.Self()` in `atlas-rest`.* This is PRD Q1,
  explicitly deferred; it would also change isolated mode and every
  in-cluster legacy caller. Out of scope.

**Rollout note.** `atlas-ingress-configmap` is a plain resource, not a
generated one, so editing `nginx.conf` does not roll the pods. The
`configMapGenerator`-managed `env-default.conf.template` *is* hashed, so
adding it does trigger a rollout — but the plan must still call for an
explicit `kubectl rollout restart deployment/atlas-ingress` in the runbook
for the `nginx.conf` half.

### D4 — Bootstrap: environment-scoped tenant resolution

Replace `bootstrap.sh:234-238` with two extractable top-level functions.
`test/data_ingest_test.bats:14-24` establishes the harness contract:
bootstrap.sh is not sourceable, so bats extracts helpers with
`sed -n '/^name()/,/^}/p'`. **Both functions must therefore open with
`name()` at column 0 and close with `}` at column 0.**

```
find_environment_tenant   # echoes this environment's tenant id, or empty
create_environment_tenant # POSTs canonical payload, echoes the new id
```

The `ENVIRONMENT` header is carried in a bash array assembled once, near the
`ATLAS_MODE` gate that `create_service_config` already keys on
(`bootstrap.sh:404,462`):

```sh
ENV_HEADER=()
if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
    require_env ATLAS_ENVIRONMENT
    ENV_HEADER=(-H "ENVIRONMENT: $ATLAS_ENVIRONMENT")
fi
```

An array, not a string, because a string would word-split on the space in
`ENVIRONMENT: pr-1411`. Expand as `"${ENV_HEADER[@]}"`; the image ships
bash ≥ 4.4 so an empty array under `set -u` is safe, but the plan should pin
that with a bats case that runs the isolated path.

Isolated mode gets an empty array, so its curl argv is byte-identical to
today — FR-2.5 satisfied by construction rather than by a parallel code path.

Idempotency (FR-2.4) is the same GET-first shape as today; only the scope
changes. FR-2.6 needs no work: `TENANT_ID` is already reassigned in place and
every later step reads it.

**Why not gate on "does an environment record exist" like `cleanup.sh` does?**
`cleanup.sh:135-137` prefers the live check because teardown must not trust a
build-time flag. Bootstrap is different: it is *establishing* the state, and
it must send the header on the very first call, before the record could tell
it anything. `ATLAS_MODE` is the right signal here, and it is the signal the
neighbouring `create_service_config` already uses.

### D5 — Bootstrap: recording the tenant, via a shared env-record helper

`cleanup.sh` already solved GET-then-PATCH-with-every-attribute, and its
comment (`cleanup.sh:143-152`) documents exactly why a partial body is
dangerous: `RestModel` has no pointer fields, so an omitted attribute decodes
as its zero value. Note that the *processor* now backfills omitted fields from
the existing record (`processor.go:243-255`), so the two layers disagree
about who is responsible — the safe move is to keep sending everything.

There is a second, harder constraint the PRD does not mention:
`UpdateByName` calls `validatePhase(input.Phase)` **before** any backfill
(`processor.go:224-226`), and `phaseIndex("")` is `-1`. **A PATCH with no
`phase` is a 400.** The body must carry the record's current phase — which
`validatePhaseTransition` accepts, since `ti == fi` is explicitly legal
(`processor.go:82-88`). That is also what makes FR-3.4's idempotency free.

**Decision.** Extract `services/atlas-pr-bootstrap/scripts/env-record.sh`
with two functions:

```
env_record_get                                              # echoes the JSON:API document, or nothing
env_record_patch <phase> <baseline> <namespace> <tenant> <overrides_json>
```

`cleanup.sh`'s `_dcp_env_get` and `_dcp_patch_phase` become one-line
delegations, preserving their names so `cleanup_test.bats` (30 KB of pinned
assertions) keeps passing unchanged. `bootstrap.sh` sources the new file
alongside `service-config.sh` and adds `record_environment_tenant`, which
GETs, extracts `baseline`/`namespace`/`overrides`/`phase`, and PATCHes with
`tenant` replaced.

FR-3.2 (target the baseline's ingress) needs no new plumbing: the PATCH goes
to `$ATLAS_UI_BASE`, which in sparse mode is the PR's own ingress, whose
`NS_ATLAS_CONFIGURATIONS` already points at `atlas-main`
(`overlays/pr-sparse/ns-overrides.yaml`). atlas-configurations is never
overridden.

FR-3.5: `record_environment_tenant` returns non-zero on failure and is called
bare under `set -e` (restored at `bootstrap.sh:22`), with an explicit
`|| exit 1` at the call site for the same belt-and-braces reason
`bootstrap.sh:470-476` documents.

**Sequencing inside bootstrap.** The PATCH runs immediately after the tenant
id is resolved and *before* the tenant-config clone. If bootstrap dies
mid-run, the record already names the tenant, so teardown can reclaim whatever
was written. The reverse order leaks on every partial failure.

### D6 — Everything bootstrap does becomes environment-tagged, and that is fine (checked, not assumed)

Once the sparse ingress defaults to `pr-<N>` (D3), **every** bootstrap call
through `$ATLAS_UI_BASE` is tagged, not only the ones that opt in. Each
control-plane read was checked:

| Bootstrap call | Scoping | Verdict |
|---|---|---|
| `GET /api/configurations/templates?region=…` | `templates.OverlaySingle` / `OverlayCollection` — **baseline fallback**, `templates/overlay.go:42-75` | Finds main's template via the `environment IN (e, baseline)` fallback. **Not** broken. |
| `GET`/`POST /api/configurations/tenants/{id}` | `scope.Strict` (`configurations/tenants/provider.go:19-28`) | Self-consistent: bootstrap creates and reads under `pr-<N>`. |
| `POST /api/configurations/services` | `scope.Strict`, header already sent explicitly | Unchanged from PR #1418. |
| `GET`/`POST /api/tenants` | `scope.Strict` | This is the fix (D4). |
| everything else (`/api/data/*`, seeds, health) | no scope layer (`handler.go:50-59`) | Header is inert. |

The templates row is the one that could have broken the whole task — a strict
scope there would have made every sparse bootstrap fail with "no template
found ... cluster setup issue" (`bootstrap.sh:290-293`). It does not, because
task-232 deviation V3 gave templates a baseline-fallback strategy precisely
for this case. **Worth an explicit regression test**, because it is one
`scope.Strict` refactor away from becoming a hard outage.

There is also a *latent* bug this closes for free: today the sparse
tenant-config row is created with `environment = ''` (no header), while main's
pods reading it under a `pr-<N>` operation apply `scope.Strict("pr-<N>")` and
would not find it. After this change both sides agree on `pr-<N>`.

### D7 — Data isolation is by tenant, not by duplicated documents (PRD correction)

The PRD §1/§6 assume "a sparse environment's tenant gets its own full
baseline restore (~48k documents)". That is **not** what will happen, and
must not be made to happen.

`c5e88320a` ("stop the data-ingest guard restoring into a populated
database") changed the guard to
`if [ "$docs" = "0" ] && [ "$canon" = "0" ]` (`bootstrap.sh:541`). The
reasoning is in the comment at lines 517-526: `baseline.Rewriter` rewrites
only `tenant_id` and copies every other column *including primary keys*, so a
COPY into a database that already holds the canonical rows **fails on
`documents_pkey` and rolls back to a wiped target tenant**. A sparse
environment shares main's `atlas-data`, so `canon` is ~49k and the restore is
correctly skipped.

The sparse tenant reads the corpus anyway:
`document/storage.go:45,73-97` falls back to
`canonical.TenantId(region, major, minor)` when the caller's tenant owns no
rows, and that id is derived from the version triple
(`canonical/canonical.go:32`) — not from the environment. A sparse tenant
deliberately shares main's version triple (FR-2.3), so it reads the identical
dataset while owning zero rows of its own.

**This does not weaken isolation.** `atlas-data` documents are version-keyed,
read-mostly WZ reference data; the canonical tenant is a shared corpus by
design. Mutable gameplay state — characters, inventories, guilds, drops,
seeds — lives in the other services' tables, is `tenant_id`-keyed, is written
under the environment's own tenant, and is exactly what
`sweep-orphans.sh --sweep-tenant` reclaims via `tenant-tables.txt`.

It also *strengthens* the answer to PRD **Q4**: there is no ~48k-document
multiplier per concurrent sparse environment, so the ceiling on simultaneous
sparse environments is not gated on `atlas-data` volume. Measure it anyway
during acceptance, but the expected answer is "negligible".

Two acceptance criteria must change — see §6.

---

## 5. Rollout ordering (PRD Q3, answered)

The dependency chain is strict and one-directional:

```
1. main environment record exists, phase=ACTIVE          (D1)
2. ATLAS_ENVIRONMENT=main on main's pods                 (D2)
      → Self()="main"; heartbeat starts; ownership resolves;
        boot-time BackfillEnvironment re-stamps any '' rows
3. ingress default = main                                (D3)
      → browser traffic tagged; scope.Strict engages
```

Step 3 before step 1 is the failure the PRD's FR-4.4 warns about (every
request 400s). Step 2 before step 1 leaves main's own loops iterating nothing
— strictly worse than today, because today `self=""` at least still processes
untagged messages while step 2 alone would not fix the empty-slice problem in
`EnvironmentsOwnedBy`.

**In a single Argo sync this ordering is *mostly* free**, because Argo runs
Deployments at wave 10 and the record Job at wave 11 — but that is the
*reverse* of what we want for steps 1↔3. The resolution is the one in D1:
the ingress serving 400s between wave-10 health and wave-11 completion is a
bounded, self-healing bring-up window, and nothing outside that window is
affected. For the *existing* live `atlas-main`, the operator-facing sequence
is:

1. Merge and sync — the wave-11 Job creates the `main` record.
2. `kubectl rollout restart deployment/atlas-ingress -n atlas-main` to pick up
   the `nginx.conf` change (D3's rollout note).

If the bring-up window proves unacceptable in practice, switch D1 to the
migration-seeder alternative; that change is local and does not disturb
anything else in this design.

**Backwards compatibility.** Isolated mode gets no `ATLAS_ENVIRONMENT`, no
ingress default, and an empty `ENV_HEADER` array — three independent reasons
its behaviour is unchanged. Local compose is unaffected for the same reasons.

---

## 6. Corrections to the PRD's acceptance criteria

Two live-round-trip criteria are unachievable as written (D7). Replace:

| PRD criterion | Replacement |
|---|---|
| "`atlas-data` document count under the PR tenant reaches the full baseline-restore total, with the baseline's own count unchanged" | "`GET /api/data/status` under the PR tenant reports **0** owned documents; `GET /api/data/status?scope=shared` reports the full canonical total; main's own counts are unchanged; bootstrap logs `data already reachable (tenant=0, canonical=<N>); skipping ingest`" |
| PRD §6 "each sparse environment adds a full per-tenant `atlas-data` restore" | Struck. Sparse environments add **no** `atlas-data` rows; they read the shared canonical corpus. Per-environment row growth is confined to the tenant-keyed gameplay and seed tables. |

Two additions:

- Watch `atlas_kafka_gate_dropped_unresolvable_total` alongside
  `skipped_not_owner` (F2 — a stale registry produces the *other* verdict).
- Confirm main's periodic per-tenant work continues while a sparse
  environment is live (F3). This regression is invisible to every other
  criterion in the list.

Everything else in PRD §10 stands.

---

## 7. Test strategy

**bats (`services/atlas-pr-bootstrap/test/`)** — run by `tools/verify.sh:521-528`
when the service changes.

New `tenant_provisioning_test.bats`, following `data_ingest_test.bats`'s
sed-extraction harness and its `declare -F` guard (a missing extraction exits
127, which would satisfy every "must fail" assertion and turn the suite green
for the wrong reason):

- scoped GET returns nothing → exactly one POST is issued, both carrying
  `ENVIRONMENT: pr-<N>` (assert on recorded curl argv)
- scoped GET returns a tenant → **no** POST; the id is reused (FR-2.4)
- a baseline tenant matching the version triple is **not** adopted when the
  scoped listing is empty (the core regression)
- `ATLAS_MODE=isolated` → recorded argv contains no `ENVIRONMENT` header at
  all, and `ENV_HEADER` empty-array expansion does not trip `set -u` (FR-2.5)
- `create_environment_tenant` fails loudly on a POST that returns no id

New `env_record_test.bats` for the extracted helper:

- `env_record_patch` sends all five attributes (regression pin for
  `cleanup.sh:143-152`)
- `record_environment_tenant` carries the record's **current** phase, not an
  empty one (the `validatePhase` 400 in D5)
- re-running against a record already holding the same tenant is a same-phase
  PATCH, not an error (FR-3.4)
- a failing PATCH propagates non-zero (FR-3.5)

`cleanup_test.bats` must keep passing **unmodified** — that is the acceptance
test for the D5 delegation refactor.

**Rendered manifests.** `kustomize build` assertions (a new
`tools/`-side check or a bats case, whichever the plan prefers):

- `overlays/main` renders `ATLAS_ENVIRONMENT: main` in the `atlas-env`
  ConfigMap, and that value equals `namespace` minus `atlas-` (FR-1.2)
- `overlays/pr` renders **no** `ATLAS_ENVIRONMENT` anywhere (FR-1.5)
- `overlays/main` renders an `atlas-environment-record` Job whose POST body
  carries `"phase": "ACTIVE"` and targets the ClusterIP, not the ingress
- the ingress container carries `ATLAS_ENVIRONMENT_DEFAULT` with
  `optional: true`, and `NGINX_ENVSUBST_FILTER` contains it

**Guards.** `tools/pr-sparse-mirror-guard.sh` must pass (nothing mirrored is
touched). `tools/gen-routes.sh --check` must pass (the `NS_*` block is
unchanged; the new template is a sibling of `routes.conf.template`, not part
of its generated content).

**Go.** One regression test in `atlas-configurations` pinning that a template
lookup from a non-baseline environment still resolves the baseline's row
(D6's sharp edge). `templates/overlay_test.go` is the home.

**Gate.** Flagless `tools/verify.sh` must exit 0 before the branch is called
done.

---

## 8. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Bring-up window where main's ingress stamps `main` before the record exists | Medium | Bounded by Argo wave gating; self-heals; D1's alternative is a one-file swap if unacceptable |
| A future refactor makes `templates` strictly scoped | High if it happens | The D6 regression test in `overlay_test.go` |
| `nginx.conf` edits do not roll the ingress pods automatically | Low | Explicit restart step in the runbook; the hashed template CM does roll |
| `ENV_HEADER` word-splitting or `set -u` on an empty array | Low | Array not string; isolated-path bats case |
| A PATCH omitting `phase` 400s | Low | D5 sends the current phase; pinned by bats |
| Tenant-created Kafka event does not reach main's consumers with the right environment | Medium | Not designed here; the live round-trip (`atlas-ban` processes a PR-env operation) is what proves it |
| main's control-plane rows written with `environment=''` between the last restart and this change become invisible | Low | Boot-time `BackfillEnvironment` re-runs on the D2 restart and sweeps them (D2) |

---

## 9. Open question for the user

**Only one, and it does not block planning** — the design commits to a
recommendation either way.

D1 places the `main` environment record in a wave-11 Argo Job, which leaves a
seconds-long window on a fresh `atlas-main` bring-up where the ingress stamps
`main` before the record exists and every REST call 400s. The alternative —
having `atlas-configurations` upsert its own baseline record during migration
— closes the window completely but edits a service PRD §7 lists as unchanged.

Recommendation: **ship the Job**, and keep the seeder in reserve. The window
is bring-up-only and self-healing, and the Job keeps this task's blast radius
inside deploy config plus one shell script.

PRD Q1 (tighten the legacy `env == ""` path) and Q4 (concurrent-environment
ceiling) are answered or deferred in §4/§6 and need no decision here.
