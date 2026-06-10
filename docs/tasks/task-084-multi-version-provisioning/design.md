# Multi-Version Tenant Provisioning — Design

Task: task-084-multi-version-provisioning
Status: Draft
Created: 2026-06-10
Based on: `prd.md` (approved)

---

## 1. Problem Recap

Running N game versions side-by-side requires two hand-maintained places to agree, with no
single source of truth:

1. the login/channel **`services` config `tenants[]`** (drives socket bind + per-tenant Kafka
   consumer registration), and
2. the **k8s LoadBalancer / Deployment port set** in `deploy/k8s/base/atlas-{login,channel}.yaml`.

Two distinct failure modes follow:

- **Bootstrap clobber (runtime).** `atlas-pr-bootstrap`'s `upsert_service_config` rebuilds the
  live `services` config `tenants[]` from a static canonical template and writes it back wholesale.
  Any second version added by hand is reconciled away on the next bootstrap run; its socket
  listener can linger long enough to accept a connection and read the login packet, but its
  per-tenant consumers are drained, so the client logs in and hangs.
- **LB drift (deploy).** The per-version port set is hand-edited in two files. Today's checked-in
  base manifests are already inconsistent (see §3): the bind side and LB side can silently
  disagree, which is exactly the `bug_new_tenant_version_lb_socket_ports` memory.

This task removes both classes of error by (a) making bootstrap an **additive, id-keyed upsert**,
(b) deriving **every port from `majorVersion`** via one formula, and (c) **generating the LB/Deployment
port set from one declared version list** with a CI drift guard.

---

## 2. Scope-Setting Decisions (resolving PRD §9 open questions)

| # | Open question | Decision | Rationale |
|---|---------------|----------|-----------|
| Q1 | Declared version-set location/format | New `deploy/k8s/base/versions.json` (array of `{region, majorVersion, minorVersion}`), with a sibling `versions.schema.json`. | The game-version axis is **orthogonal** to `.github/config/services.json`, which enumerates Go services/libs for the CI build matrix + docker-bake. Mixing version exposure into it conflates "what we compile" with "what versions a cluster exposes" and bloats that schema. Co-located with the manifests it drives, mirroring how `deploy/shared/routes.conf` sits beside its generated output. |
| Q2 | Generator mechanism + drift-check home | A bash+jq script `tools/gen-lb-ports.sh` that rewrites **marker-delimited port blocks** inside the two base YAMLs, plus a `--check` mode. CI runs it in `--check` mode in `pr-validation.yml` (next to `redis-key-guard`). | Mirrors the established `gen-routes.sh` + committed-generated-file + `git diff --exit-code` convention and the `tools/*.sh` ecosystem (`build-services.sh`, `redis-key-guard.sh`, `task-numbers.sh`). Kustomize exec-generator plugins add CI sandbox friction for no benefit. Marker blocks keep the rest of each YAML (env, selector, `loadBalancerIP`, the static `8080` port) human-owned and reviewable. |
| Q3 | Canonical-template port handling | **Remove** the literal port from `canonical/services/{login,channel}-service.json`; construct the canonical tenant entry (id + derived port[s]) entirely in `bootstrap.sh` from `MAJOR_VERSION`. | The template port (8300/8301) is the root of FR-2.3's "binds 8300/8301 regardless of the PR's actual version" bug. Deriving in-script removes the second source. The template keeps only `type` + `tasks` (the tenant-agnostic shell). |
| Q4 | Same-`major` coexistence | **Reject at generate time and at version-set lint.** Two entries that derive the same `loginPort` (same `majorVersion`) are a hard error in `gen-lb-ports.sh` and the version-set validator. | FR-1.3: the derivation is a function of `majorVersion` only; two same-major tenants collide on the port. Documented as unsupported; failing loudly beats a silent last-writer-wins LB. |
| Q5 | Persistent-env tenant creation relationship | The declared version set drives **LB exposure only**. Creating the tenant row + per-tenant config stays the UI template-clone flow (persistent) / bootstrap (ephemeral). The runbook ties them together: "add to `versions.json` + redeploy" exposes the port; the tenant/config still must exist (clone or bootstrap). | Keeps this task to provisioning plumbing; per PRD non-goals we do not auto-project `tenants[]` from the registry nor change template-clone. |

---

## 3. Pre-Work: reconcile the base manifests (FR-3.3 enabler)

**Finding (must be surfaced):** the PRD states PR #711 "already backfilled the current LB ports by
hand (gms-84 8400/8401, gms-92 9201, gms-95 9501)." In this worktree's base manifests that backfill
is **not present** — the LB YAMLs were last touched by #522, not #711:

- `atlas-login.yaml` exposes `1200, 8300, 8700, 9200, 9500, 18500` → **missing `8400` (gms-84)**.
- `atlas-channel.yaml` exposes `1201, 8301, 8701, 18501` → **missing `8401, 9201, 9501`
  (gms-84, gms-92, gms-95)**.

So FR-3.3 ("generator output is a no-op diff vs. today's checked-in ports for
`[12, 83, 84, 87, 92, 95, 185]`") **cannot hold literally** against the current incomplete state.

**Resolution.** The first generator run is intentionally **not** a pure no-op against the broken
state — it *completes* the set. We re-interpret FR-3.3 precisely:

1. **Step 0 (this task):** define `versions.json` = `[12, 83, 84, 87, 92, 95, 185]`, run
   `gen-lb-ports.sh`, and commit the result. This produces the *intended* post-#711 complete set
   (login `+8400`; channel `+8401, +9201, +9501`). This commit is the backfill #711 was supposed to
   land; doing it via the generator proves the generator reproduces the convention.
2. **Invariant going forward (the real FR-3.3/FR-3.5 guarantee):** from that commit on, re-running
   the generator against the committed `versions.json` is a **no-op diff**, and CI fails on any
   drift. The "no-op" guarantee is anchored to the generated baseline, not to the pre-existing
   broken state.

The design doc and runbook will state this re-interpretation explicitly so the plan/execution phase
doesn't chase a literal no-op against a known-incomplete file.

---

## 4. Component Architecture

Five units, each with one purpose and a defined interface.

### 4.1 Port derivation (single source of truth) — FR-1

A pure function of `majorVersion`:

```
loginPort(major)   = major * 100
channelPort(major) = loginPort(major) + 1
```

**Where it lives.** A tiny shared shell helper `tools/lib/version-ports.sh` exposing
`derive_login_port <major>` and `derive_channel_port <major>` (echo the integer). Both consumers
source it:

- `tools/gen-lb-ports.sh` sources it directly (repo-root context, CI + local).
- `atlas-pr-bootstrap` sources it at runtime. The image already ships `scripts/lib.sh`; the
  Dockerfile gains one `COPY tools/lib/version-ports.sh /atlas/version-ports.sh` line and
  `bootstrap.sh` sources it alongside `lib.sh`.

This gives FR-1.2 *literal* single-definition sharing — no port arithmetic is written twice. The
build context for the bootstrap image is repo-root (`docker_context: "."`), so the COPY is
straightforward.

> Alternative considered: duplicate the one-line formula in each consumer and guard parity with a
> test. Rejected — a shared sourced helper is the same effort and removes the duplication outright,
> which is the explicit FR-1.2 intent.

### 4.2 Declared version set — `deploy/k8s/base/versions.json` — FR-3.1

```json
{
  "$schema": "./versions.schema.json",
  "description": "Game versions this environment exposes on the login/channel LoadBalancers.",
  "versions": [
    { "region": "gms", "majorVersion": 12,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 83,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 84,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 87,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 92,  "minorVersion": 1 },
    { "region": "gms", "majorVersion": 95,  "minorVersion": 1 },
    { "region": "jms", "majorVersion": 185, "minorVersion": 1 }
  ]
}
```

- `region` + `majorVersion` drive the port **name** (`atlas-login-<region>-<major>`), matching the
  existing convention exactly (`atlas-login-gms-83`, `atlas-login-jms-185`). `minorVersion` is
  carried for completeness/operator clarity and to match the naming if a future minor-bearing name
  is needed; it does **not** affect the port (FR-1.3).
- `versions.schema.json` constrains types and requires the triple; the generator additionally
  rejects duplicate `majorVersion` (Q4 / FR-1.3).

> Naming nuance: the current names are `gms-83`, not `gms-831`. The generator uses
> `<region>-<major>` for the port `name` (stable, matches checked-in YAML). The `minorVersion`
> field exists in the manifest but is not interpolated into the name, preserving the no-op diff.

### 4.3 LB/Deployment generator — `tools/gen-lb-ports.sh` — FR-3.2 / FR-3.5

**Interface:** `gen-lb-ports.sh [--check]`.

- No flag → rewrite the marker-delimited blocks in both base YAMLs in place.
- `--check` → generate to a temp file, `diff` against the checked-in YAML, exit non-zero on drift
  (the CI guard). Mirrors `gen-routes.sh` semantics.

**What it owns.** Two marker-delimited regions per file:

```yaml
        ports:
        # BEGIN generated:version-ports (tools/gen-lb-ports.sh)
        - containerPort: 1200
        - containerPort: 8300
        ...
        # END generated:version-ports
        - containerPort: 8080        # static, human-owned (metrics/http)
```

and the `Service.ports` list likewise wrapped in `# BEGIN/END generated:version-ports`. Everything
outside the markers (image, env, `SERVICE_ID`, `loadBalancerIP`, the static `8080` port,
`DRAIN_DEADLINE_MS`) is untouched.

**Algorithm.** For each version in `versions.json`, sorted deterministically (by `majorVersion`):
emit `containerPort: <loginPort>` (login YAML) / `<channelPort>` (channel YAML) in the Deployment
block, and the named `Service.ports` entry (`port`/`targetPort` = derived, `name` =
`atlas-<svc>-<region>-<major>`). Ports come **only** from §4.1; no literals in the script.
Determinism (FR-3.3): stable sort + fixed field order → identical bytes for identical input.

**Error handling.**
- Duplicate `majorVersion` → `exit 1` with the colliding versions (Q4).
- Missing/invalid `versions.json` or absent markers in a target YAML → `exit 1` with a clear
  message (fail closed; never emit a half-written manifest).
- `--check` drift → `exit 1` printing the diff (CI failure).

**CI wiring (FR-3.5).** A `gen-lb-ports` job in `.github/workflows/pr-validation.yml` (sibling to
`redis-key-guard`) runs `./tools/gen-lb-ports.sh --check`. Added to the final gate's `needs`. This is
the single mechanism that prevents the two must-agree places from diverging.

### 4.4 Additive bootstrap upsert — `atlas-pr-bootstrap/scripts/bootstrap.sh` — FR-2

Rework `upsert_service_config` from **template-rewrite-and-overwrite** to
**read-live → upsert-by-id → write-merged**:

1. **Build the canonical tenant entry in-script** from the resolved `TENANT_ID`, `MAJOR_VERSION`
   (→ derived port[s] via §4.1), and — for channel — `LB_IP`:
   - login: `{ "id": TENANT_ID, "port": loginPort }`
   - channel: `{ "id": TENANT_ID, "ipAddress": LB_IP,
     "worlds": [{ "id": 0, "channels": [{ "id": 0, "port": channelPort }] }] }`
   The world/channel shell still comes from the canonical template (so multi-world layouts remain
   template-controlled); only `id`, `ipAddress`, and the derived `port` are set in-script.
2. **GET** `/api/configurations/services/{svc_id}`.
3. **Absent (first run, FR-2.7):** POST a config whose `tenants` = `[canonicalEntry]`. (`type`,
   `tasks` from the template.)
4. **Present:** take the **live** attributes, and upsert `canonicalEntry` into `tenants[]`
   **keyed by `id`** (FR-2.2):
   - if an entry with `id == TENANT_ID` exists → replace it **at its existing index** (preserve
     array order → byte-stable → idempotent);
   - else → append.
   Every other entry is left byte-for-byte intact, including its `ipAddress` (FR-2.4).
   jq sketch:
   ```
   .data.attributes.tenants |=
     ( if any(.[]; .id == $tid)
       then map(if .id == $tid then $entry else . end)
       else . + [$entry] end )
   ```
   with `$entry` = the canonical entry, merged onto live `type`/`tasks`.
5. **Idempotency guard (FR-2.5):** keep the existing canonicalized-attribute compare
   (`jq -cS '.data.attributes'`); PATCH only when merged ≠ live. Second identical run → equal →
   skip PATCH → no config-status churn, and the drops-service PATCH-panic workaround
   (`reflect.Value.Set using unaddressable value`) remains exercised (drops has no `tenants[]`, so
   the merge is a no-op and the compare always matches).
6. **drops-service (FR-2.6):** unchanged — `has("tenants")` is false, the merge step is skipped,
   compare matches, no PATCH.

The three call sites stay; the `rewrite_ip` parameter is replaced by a per-service "shape"
(login vs channel) that selects which canonical entry to build.

**Why this is correct for coexistence (FR-4.2):** because the write is now a merge that preserves
foreign `tenants[]` entries, a canonical-version re-run never removes another version's entry →
the projection never sees that tenant disappear → its listener and consumers are never drained.

### 4.5 Coexistence verification + runbook — FR-4 / FR-5

`atlas-login`/`atlas-channel` need **no code change** (PRD §7): the #522 projection already binds
per-tenant listeners and registers/drains per-tenant consumers from the `services` config. This
task stabilizes the *inputs*; FR-4 is a **verification obligation**, not a re-architecture. If
verification surfaces a genuine drain-on-merge-rewrite bug in the projection, that becomes in scope
(contingency, not expected).

Verification artifacts:
- **Bats unit tests** (`test/bootstrap_test.bats`) for the new merge: foreign-entry preservation,
  id-keyed replace-in-place, order stability, first-run POST, second-run no-op, channel `ipAddress`
  preservation, derived-port correctness, drops no-op.
- **Generator tests** (`tools/gen-lb-ports_test.sh`, mirroring `task-numbers_test.sh`): no-op against
  the committed baseline, duplicate-major rejection, add-a-version diff, `--check` exit codes.
- **Manual k8s repro** (documented in runbook): v83 canonical + v84 hand-added; re-run bootstrap;
  assert both tenants' `projection.applied op=add` present and **no** `op=drain` for v84; a v84
  client completes the login handshake without hanging.

Runbook (FR-5.2): `docs/runbooks/ephemeral-pr-deployments.md` (add-a-version flow; how a hand-added
second version now survives bootstrap; the watch-for `projection.applied op=add` sequence) and
`docs/onboarding.md` (the one-list-edit + redeploy workflow, additive-bootstrap guarantee,
persistent-vs-ephemeral tenant creation per Q5).

---

## 5. Data Flow

**Deploy time (LB exposure):**
```
versions.json ──> gen-lb-ports.sh ──> atlas-{login,channel}.yaml (marker blocks)
       │                 │
       └── version-ports.sh (derive) ──┘            CI: gen-lb-ports.sh --check ⟶ fail on drift
```

**Bootstrap time (services config):**
```
MAJOR_VERSION ─┐
TENANT_ID ─────┼─> build canonicalEntry ─> GET live config ─> merge-by-id ─> (POST | PATCH | skip)
LB_IP (chan) ──┘        │                                                         │
              version-ports.sh (derive)                          idempotency compare guard
```

**Runtime (unchanged, verified):**
```
services config tenants[] ──(#522 projection)──> per-tenant listener bind + consumer register/drain
```

---

## 6. Testing Strategy

| Unit | Test | Asserts |
|------|------|---------|
| version-ports.sh | bats | `derive_login_port 83 == 8300`, `derive_channel_port 83 == 8301`, `12→1200/1201`, `185→18500/18501`. |
| gen-lb-ports.sh | `gen-lb-ports_test.sh` | no-op vs committed baseline; duplicate-major → exit 1; add v99 → expected diff; `--check` exit codes; markers required. |
| bootstrap merge | bats | preserve-foreign; id-keyed replace-in-place + order; first-run POST shape; second-run no-op (no PATCH); channel ipAddress preserved on foreign + set on canonical; derived port; drops no-op. |
| coexistence | manual k8s repro (runbook) | v83+v84 both `op=add`, no v84 `op=drain` on re-run; v84 login handshake completes. |

No Go module changes expected (shell + k8s/CI config), so the Go build/test/bake gates are
trivially satisfied; redis-key-guard unaffected. The acceptance gate stays as PRD §10.

---

## 7. Risks & Mitigations

- **Marker-block fragility.** A future hand-edit inside the generated block would be silently
  reverted by the generator. *Mitigation:* CI `--check` fails the PR that introduces the drift,
  pointing the operator at `versions.json`; the markers name the owning script.
- **Bootstrap-image build context.** Sourcing `tools/lib/version-ports.sh` requires it in the image.
  *Mitigation:* one `COPY` line; build context is already repo-root. Covered by the existing
  `dockerfile_test.bats`.
- **jq order-preservation correctness.** Idempotency depends on replace-in-place (not
  filter-then-append). *Mitigation:* explicit order-stability bats test on the second run.
- **FR-3.3 literal no-op expectation.** Addressed in §3 by re-anchoring "no-op" to the generated
  baseline; called out so the plan phase doesn't treat the current incomplete YAML as the target.
- **Same-major collision.** Rejected loudly at generate + lint (Q4) rather than silently producing a
  colliding LB.

---

## 8. Out of Scope (per PRD non-goals)

Packet/wire layer; handler/writer opcode projection internals; new game content; template-clone
rewrite; login/channel consuming `tenant.status` directly; `atlas-configurations` auto-projecting
`tenants[]`; a runtime LB port-reconciler controller; region/version filtering on the tenants REST
API.

---

## 9. Deliverables Checklist (maps to PRD §10)

- [ ] `tools/lib/version-ports.sh` (+ bats) — single port derivation (FR-1).
- [ ] `deploy/k8s/base/versions.json` + `versions.schema.json` — declared set (FR-3.1).
- [ ] `tools/gen-lb-ports.sh` + `gen-lb-ports_test.sh` — generator + drift `--check` (FR-3.2/3.5).
- [ ] Regenerated `atlas-{login,channel}.yaml` marker blocks completing `[12,83,84,87,92,95,185]`
      (FR-3.3 per §3 re-anchor).
- [ ] `pr-validation.yml` `gen-lb-ports --check` job in the final gate (FR-3.5).
- [ ] `bootstrap.sh` additive id-keyed merge + derived port; template port removed; idempotency
      guard preserved (FR-2).
- [ ] Bootstrap bats coverage for the merge (FR-2/FR-4 unit level).
- [ ] Manual coexistence repro documented (FR-4).
- [ ] `docs/runbooks/ephemeral-pr-deployments.md` + `docs/onboarding.md` updated (FR-5).
