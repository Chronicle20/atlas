# Task-084 Multi-Version Tenant Provisioning — Context

Companion to `plan.md`. Key files, decisions, and gotchas for the implementer.

## What this task is

Make N game versions coexist in one environment without two hand-maintained
places silently disagreeing. Three moving parts:

1. **One port formula** (`major×100` login, `+1` channel) in a single sourced
   shell helper, consumed by both the bootstrap image and the LB generator.
2. **A declared version set** (`deploy/k8s/base/versions.json`) driving a
   build-time generator (`tools/gen-lb-ports.sh`) that rewrites marker-delimited
   port blocks in the two base manifests, with a CI `--check` drift guard.
3. **Additive bootstrap**: `upsert_service_config` reads the live `services`
   config and upserts only its canonical tenant (keyed by id), so a co-resident
   version is never clobbered (the root-cause "login then hang" bug).

**No Go module changes.** Pure shell + JSON + k8s/CI config + docs.

## ⚠️ One intentional deviation from design.md

`design.md` §4.1 puts the shared helper at `tools/lib/version-ports.sh` and
plans `COPY tools/lib/version-ports.sh` into the bootstrap image, claiming the
image builds from repo-root context. **That is wrong for this repo:**

- `docker-bake.hcl:123` → `target "atlas-pr-bootstrap"` has
  `context = "services/atlas-pr-bootstrap"` (relative COPYs only).
- `services/atlas-pr-bootstrap/test/dockerfile_test.bats` enforces
  `^COPY scripts/<name> /atlas/<name>$`.

So a `COPY tools/...` line cannot reach outside the service-dir context.

**Plan's resolution:** the single shared helper lives at
`services/atlas-pr-bootstrap/scripts/version-ports.sh`. The image COPYs it like
any other script; `tools/gen-lb-ports.sh` sources it by repo-relative path. One
physical file, both consumers source it → FR-1.2 single-definition preserved,
minimal churn (no bake-context or Dockerfile-test-convention change). This is
the **only** departure from the design.

## Key files (read these first)

| File | Role |
|------|------|
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | The PR-env bootstrap. `upsert_service_config` (~272–342) is the clobbering function being reworked. `MAJOR_VERSION` is overwritten from `canonical/tenant.json` (~207) *before* the service-config step, so deriving the port from it is correct. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Shared `log`/`require_env`/`retry`/`http_ok`. **Note:** `log` goes to stderr (a `log` on stdout corrupts `$()` capture — PR-544). `set -uo pipefail` here; bootstrap re-asserts `set -e`. |
| `services/atlas-pr-bootstrap/canonical/services/{login,channel,drops}-service.json` | Canonical service-config templates keyed by pinned `data.id` (service UUID). login holds the stale `8300`; channel holds the worlds shell + stale `8301`; drops is tenant-agnostic (no `tenants[]`). |
| `services/atlas-pr-bootstrap/Dockerfile` | `context = services/atlas-pr-bootstrap`, relative COPYs. Add the two sourced helpers here. |
| `services/atlas-pr-bootstrap/test/dockerfile_test.bats` | Asserts every `scripts/*.sh` is COPY'd; chmod-required for all but the **skip list** (currently only `lib.sh`). Sourced helpers must be added to that skip list. |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | Sparse — only env-validation. New merge logic is tested via the *extracted* `service-config.sh` (network-free), not by running bootstrap.sh. |
| `deploy/k8s/base/atlas-login.yaml` / `atlas-channel.yaml` | The LB/Deployment manifests. **Currently incomplete** (see below). |
| `tools/gen-routes.sh` | Existing generator precedent (sed-based, committed-generated-file). New generator mirrors the convention but adds `--check`. |
| `tools/task-numbers_test.sh` | Precedent for a hermetic throwaway-git-repo shell test harness — the generator test mirrors it. |
| `.github/workflows/pr-validation.yml` | `redis-key-guard` job (~84) is the pattern for the new job; `pr-validation-complete` (~460) is the final gate to wire into. |
| `docs/runbooks/ephemeral-pr-deployments.md`, `docs/onboarding.md` | Operator docs to update. |

## Current manifest state (matters for FR-3.3)

The PRD says PR #711 "already backfilled" the LB ports. **It did not, in this
worktree.** Current checked-in ports:

- `atlas-login.yaml`: `1200, 8300, 8700, 9200, 9500, 18500` (+`8080` static) — **missing `8400`**.
- `atlas-channel.yaml`: `1201, 8301, 8701, 18501` (+`8080`) — **missing `8401, 9201, 9501`**.

So the first generator run (Task 6) is **not** a literal no-op — it *completes*
the set (this is the #711 backfill, done via the generator) and normalizes
trailing whitespace. FR-3.3's "no-op" guarantee is re-anchored to *that*
generated baseline: from Task 6 onward, `gen-lb-ports.sh --check` is a no-op and
CI fails on drift. Per `design.md` §3.

Intended complete set (sorted by major): login
`1200, 8300, 8400, 8700, 9200, 9500, 18500`; channel
`1201, 8301, 8401, 8701, 9201, 9501, 18501`.

## Design decisions locked in (from design.md §2)

- **Q1** version set = new `deploy/k8s/base/versions.json` (+ advisory schema),
  *not* an extension of `.github/config/services.json` (that file is the
  Go-service/CI-build matrix — orthogonal axis).
- **Q2** generator = bash+jq script with marker blocks + `--check`; CI guard in
  `pr-validation.yml`. Not a kustomize exec-generator.
- **Q3** remove the literal port from the canonical templates; build the
  canonical tenant entry in-script. (Reconciled: login template → `tenants: []`;
  channel template → keep the `worlds` shell because `build_channel_entry` reads
  it, with the port as an overwritten `0` placeholder.)
- **Q4** two tenants sharing a `majorVersion` (port collision) = hard error at
  generate time. The generator rejects duplicate majors.
- **Q5** the declared version set drives **LB exposure only**; tenant-row +
  per-tenant-config creation stays bootstrap (ephemeral) / template-clone
  (persistent). Documented in the runbook.

## Port name convention (do not break)

Service port `name` is `atlas-<svc>-<region>-<major>` — e.g.
`atlas-login-gms-83`, `atlas-login-jms-185`. `minorVersion` is carried in
`versions.json` for operator clarity but is **not** interpolated into the name
or the port (preserves the existing names / FR-1.3).

## Marker contract

Two labelled regions per file (distinct labels so each renders the right shape):

- `# BEGIN/END generated:container-ports` → Deployment `- containerPort: N`
  (8-space indent). `8080` stays **outside** the markers.
- `# BEGIN/END generated:service-ports` → named `Service.ports` entries
  (2-space indent). `loadBalancerIP` stays **outside**.

Generator indentation is fixed (8 / 2 spaces) — the YAML structure is stable and
the `--check` guard catches any structural drift.

## Bootstrap merge invariants (FR-2)

- **Replace-in-place, not filter-then-append** → preserves array order → byte
  stable → idempotent. (`merge_tenant_entry` uses `map(if .id==entry.id ...)`.)
- Merge onto the **live** attributes fetched by GET, never the template, so
  foreign entries (incl. their `ipAddress`) survive verbatim.
- Keep the **skip-PATCH-when-equal** guard: it provides idempotency *and* dodges
  the known atlas-configurations PATCH panic on tenant-agnostic configs
  (`reflect.Value.Set using unaddressable value`). drops-service has no
  `tenants[]` → merge is a no-op → compare matches → no PATCH.
- `build_login_entry` / `build_channel_entry` / `merge_tenant_entry` are pure and
  depend on env `TENANT_ID` / `MAJOR_VERSION` / `LB_IP` — set by bootstrap before
  the service-config step; injected directly by the bats tests.

## Verification

- Unit: `bats services/atlas-pr-bootstrap/test/` (version-ports, service_config,
  dockerfile, bootstrap env-guards, + existing suites) and
  `tools/gen-lb-ports_test.sh` (hermetic, throwaway git repo).
- Image: `docker buildx bake atlas-pr-bootstrap` then run the entrypoint shell to
  confirm both helpers source and derive correctly (`8400`/`8401`).
- LB: `tools/gen-lb-ports.sh --check` must be a clean no-op after Task 6.
- No Go/TS changes → backend/frontend guideline reviewers N/A; run
  `plan-adherence-reviewer` before PR (CLAUDE.md: review before PR).

## Related memory

- `bug_new_tenant_version_lb_socket_ports` — the exact failure class this task
  removes: per-version LB port must agree across the `services` config bind side
  and the k8s LB side; was busted for gms-84 / gms-92 / gms-95.
- `bug_new_opcodes_not_in_live_tenant_config` — adjacent gotcha: projection does
  not hot-reload *handlers/writers*; this task only touches listener/consumer
  lifecycle inputs, which the #522 projection *does* reconcile live.
