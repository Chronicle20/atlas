# Baseline-Only Ephemeral Bootstrap — Design

Task: task-098-baseline-only-bootstrap
Status: Approved design
PRD: `docs/tasks/task-098-baseline-only-bootstrap/prd.md`

## 1. Summary

Make `atlas-pr-bootstrap` provision ephemeral PR data **only** by restoring a
published canonical baseline. Delete the `full`-mode WZ-ingest path, the
`BOOTSTRAP_MODE`/`WZ_CANONICAL` knobs, the `auto→full` fallback, and the
`fetch-wz-canonical` init container plus its `/opt/wz` volume. Add an early,
read-only preflight that hard-fails — before any data-affecting work — when no
baseline exists for the env's canonical version, pointing the operator at the
`canonical-version-migration.md` runbook. Rewrite the runbook section to match.

This is a deletion-plus-guard change across three artifacts: one bash script,
one Kubernetes manifest, and one runbook. No Go code, no API, no schema changes.

## 2. Current State (verified against the worktree)

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh`
  - Defaults `WZ_CANONICAL=/opt/wz/atlas.zip`, `BOOTSTRAP_MODE=auto` (lines 31–32).
  - `canonical_baseline_exists()` (lines 127–132) — anonymous `HEAD` of
    `…/versions/<major>.<minor>/documents.dump.sha256`, 200 ⇒ present.
  - `resolve_mode()` (lines 137–155) — `auto` probes the baseline and **falls
    back to `full`** with a WARN; explicit `baseline|full` pass through.
  - `wait-ready` block probes `…/api/data/wz` (line 166) in addition to
    `…/api/data/status`.
  - Data-ingest step (lines 373–420) runs `resolve_mode` **after** REGION/
    MAJOR/MINOR have been overwritten with the canonical values from
    `/atlas/canonical/tenant.json` (lines 211–213); branches `baseline`
    (restore) vs `full` (`PATCH /api/data/wz` + `POST /api/data/process`).
- `deploy/k8s/overlays/pr/sync-bootstrap.yaml`
  - Main container `volumeMounts` `wz-canonical → /opt/wz` (readOnly).
  - `initContainers: [fetch-wz-canonical]` curls `atlas-canonical/atlas.zip`
    into `/opt/wz`.
  - `volumes: [wz-canonical]` is an **`emptyDir` (sizeLimit 8Gi)** — *not* a PVC.
- `docs/runbooks/ephemeral-pr-deployments.md` §9.1 — documents the `atlas.zip`
  upload, the `fetch-wz-canonical` init container, `WZ_CANONICAL`, and the
  `BOOTSTRAP_MODE` table.

### Open questions, resolved from the code

- **OQ-1 (emptyDir vs PVC):** It is an **`emptyDir`**. `grep -rn
  'atlas-wz-canonical' deploy` returns nothing — the named PVC referenced by
  the old task-063 design no longer exists in the tree. So removal is: delete
  `initContainers`, the `wz-canonical` `volumeMount`, and the `volumes` block.
  There is **no PVC to delete**.
- **OQ-2 (dangling env wiring):** Clean. The only `configMapKeyRef`/`envFrom`
  inputs to the Job are `atlas-pr-bootstrap-tenant` (TENANT_ID/REGION/MAJOR/
  MINOR) and `atlas-env-tokens`; neither sets `BOOTSTRAP_MODE` or
  `WZ_CANONICAL`. `cleanup.sh`'s `full` is an unrelated local DB-name variable.
  `lib.sh:9` only *mentions* `resolve_mode` in a comment — cosmetic, will be
  updated. No Helm/chart values set the removed envs.
- **OQ-3 (half-published baseline):** Yes — the preflight HEADs **both** the
  `documents.dump.sha256` sidecar **and** the `documents.dump` object, so a
  baseline that published the sidecar but not the dump (or vice versa) fails the
  preflight rather than passing it and breaking the restore later.

## 3. Approach

### 3.1 Considered alternatives

**A — Keep `BOOTSTRAP_MODE`, just drop the `auto→full` fallback.** Smallest
diff: `auto` would hard-fail instead of falling back; `full` stays as an
explicit opt-in. *Rejected.* The PRD's NFR is "no ephemeral env can write a
per-tenant WZ tree to shared MinIO via bootstrap" — as long as a `full` branch
and the `atlas.zip` mount exist, the bloat path is one env-var away from
recurring, which is exactly the regression the PRD exists to foreclose. Leaving
the init container also keeps the hard `atlas.zip` 404 failure mode alive.

**B — Remove `full` entirely; bootstrap is baseline-only with a fail-fast
preflight. (Recommended, and what the PRD specifies.)** Delete the mode switch,
the `full` branch, `WZ_CANONICAL`, the init container, and the volume. Data
provisioning is unconditionally `POST /api/data/baseline/restore`. A missing
baseline is surfaced by an early read-only probe that exits non-zero with an
actionable message. This makes cold-start canonical provisioning an explicit
operator step (the migration runbook) rather than a silent, bloat-producing
fallback.

**C — B, plus move baseline existence enforcement server-side into atlas-data.**
Out of scope and heavier: the PRD explicitly forbids atlas-data changes, and the
bootstrap is the right place to gate because it owns the env's lifecycle and can
fail the Argo Job before bringing consumers up.

We take **B**.

### 3.2 Preflight: placement and version source

The preflight must fail **before any data-affecting work** (tenant create,
config clone, service restarts, restore). Two design choices:

**Where.** Run it as the **first action after env validation** — immediately
after the `require_env` / TENANT_ID-shape check at the top of `bootstrap.sh`,
*before* the `wait-ready` block. The probe is an anonymous `HEAD` straight to
MinIO (`minio.minio` namespace, always-on, not gated by the Argo sync wave), so
it has no dependency on atlas-data being up. Failing here means the Job dies
before it touches tenants, configurations, or any deployment — the cleanest
possible "do not come up half-seeded."

**Which version to probe.** The restore later keys on the **canonical**
`(region, major, minor)` read from `/atlas/canonical/tenant.json` (the script
overwrites the env-injected REGION/MAJOR/MINOR with these at lines 211–213). To
guarantee *the preflight probes exactly the version the restore will request*,
the preflight reads those three values from `/atlas/canonical/tenant.json` up
front (the file is baked into the image, always present, no network needed)
rather than trusting the env-injected values, which are the *initial* configmap
values and could in principle differ. Probing the env values would risk a
false-pass preflight followed by a late restore failure on a different version.

To keep this unit testable without a cluster, the canonical tenant path is read
through an overridable variable:
`CANONICAL_TENANT_JSON="${CANONICAL_TENANT_JSON:-/atlas/canonical/tenant.json}"`,
used by both the new preflight and the existing tenant-create step (single
source of truth for the path).

### 3.3 Preflight semantics (transient-down vs genuinely-absent)

`canonical_baseline_exists()` today maps any non-200 (including connection
failure `000`) to "absent". For a fail-fast that an operator will act on, a
transient MinIO blip during cold cluster start must **not** be misreported as
"no baseline — go publish one". The preflight therefore distinguishes:

- HTTP **200** on *both* the `.sha256` and the `documents.dump` HEAD ⇒ present;
  proceed.
- HTTP **404** (object genuinely missing) ⇒ **fail fast** with the actionable
  message (FR-2.2).
- Connection-level failure / `000` ⇒ MinIO unreachable; **retry** a bounded
  number of times (reuse the existing `retry` helper) before giving up with a
  distinct "MinIO unreachable" error, so the operator isn't sent to publish a
  baseline that already exists.

The existing `canonical_baseline_exists` helper is refactored into a small
`baseline_object_status <url> → 200|404|000` primitive plus a
`preflight_baseline` wrapper that applies the rules above over the two object
URLs. `resolve_mode` is deleted.

### 3.4 Data-ingest step after the change

`resolve_mode`/`$mode` and the `case` are gone. The step becomes: if
`documentCount == 0`, `POST /api/data/baseline/restore` (unchanged body and
headers, lines 386–402) and `retry 60 5 data_processing_done`; else skip
(already-restored idempotency, unchanged). Because the preflight already proved
the baseline exists, the restore has no "what if absent" branch.

### 3.5 wait-ready cleanup

Drop the `retry 60 5 http_ok_tenant "$ATLAS_UI_BASE/api/data/wz"` probe (line
166). Bootstrap no longer calls `/api/data/wz`; atlas-data readiness is already
covered by the `/api/data/status` probe on the same service. The endpoint itself
is untouched (operators still use it; out of scope per PRD §5).

## 4. Manifest change (`sync-bootstrap.yaml`)

Delete three blocks, leaving the Job otherwise identical:

1. The main container's `volumeMounts: [{ name: wz-canonical, mountPath:
   /opt/wz }]`.
2. The entire `initContainers:` list (only member is `fetch-wz-canonical`).
3. The entire `volumes:` list (only member is the `wz-canonical` emptyDir).

All ServiceAccount/Role/RoleBinding, env wiring, hook annotations, and the
`imagePullSecrets` are unchanged. The Job's RBAC (deployments patch for the
rolling restart, services get for LB discovery) stays as-is.

## 5. Documentation change (`ephemeral-pr-deployments.md`)

Rewrite §9.1:

- Delete the "upload `atlas.zip`" subsection, the `fetch-wz-canonical` init
  container snippet, the `WZ_CANONICAL` note, and the `BOOTSTRAP_MODE` table.
- Replace with a short "Data provisioning is baseline-only" section: ephemeral
  envs restore the published canonical baseline for their `(region, major,
  minor)`; a missing baseline fails the bootstrap Job fast with a greppable
  error; cold-start provisioning of a new version is the
  **canonical-version-migration** runbook (link it).
- Keep the MinIO stand-up subsection (MinIO is still where baselines live) but
  drop the `mc cp … atlas.zip` step.
- Fix the cross-reference at the top of the file that currently says baselines
  are "consumed by the `auto`-mode bootstrap" → "consumed by the baseline-only
  bootstrap".

The `canonical-version-migration.md` runbook needs a one-line touch-up: its
"**`atlas-pr-bootstrap` impact:** none … `auto` mode restores …" note (lines
29–31) should read that bootstrap is baseline-only and that publishing the
baseline (step 4) is now a **prerequisite** for any ephemeral env on that
version, not merely an optimization.

## 6. Error handling

| Condition | Behavior |
|---|---|
| Baseline `.sha256` + `documents.dump` both 200 | Proceed (success path). |
| Either object 404 | Exit non-zero, single greppable line naming the version + runbook (FR-2.2). |
| MinIO unreachable (`000`) after bounded retry | Exit non-zero with a distinct "MinIO unreachable" message (not "publish a baseline"). |
| Baseline present but restore later fails | Unchanged from today — `curl -fsS` non-zero fails the Job; `data_processing_done` retry bounds the wait. |
| Re-sync after a baseline is published | Preflight passes; restore runs (idempotent on `documentCount`). FR-2.3. |

The preflight is read-only (two HEADs), so re-runs are safe and deterministic.

## 7. Testing & verification

`atlas-pr-bootstrap` is a shell container (no `go.mod`); the Go bake/test rules
in CLAUDE.md don't apply. Verification is shell + manifest + docs:

1. **bats (`test/bootstrap_test.bats`)** — keep the two existing `require_env`
   tests. Add:
   - *fail-fast on absent baseline*: a `PATH`-shim `curl` that returns `404` for
     the HEAD probes; set the required envs and `CANONICAL_TENANT_JSON` to a
     fixture `tenant.json`; assert the script exits non-zero, prints the
     version-named actionable message referencing the migration runbook, and
     does so **before** reaching `kubectl`/`wait-ready` (assert no `kubectl`
     shim was invoked).
   - *MinIO-unreachable is distinct*: shim `curl` to emit `000`; assert the
     distinct "unreachable" message and non-zero exit.
   - The shim pattern mirrors the existing tests (run the real script with a
     doctored `PATH`); no cluster needed.
2. **shellcheck** — `bootstrap.sh`, `lib.sh` clean (matches existing CI).
3. **`kustomize build deploy/k8s/overlays/pr`** — renders without the removed
   volume/init container and without dangling references.
4. **grep gate** — `BOOTSTRAP_MODE`, `WZ_CANONICAL`, `fetch-wz-canonical`,
   `/opt/wz`, `resolve_mode`, and `atlas-canonical/atlas.zip` return **no hits**
   in `services/atlas-pr-bootstrap/` and `deploy/k8s/overlays/pr/` (the
   acceptance grep; historical `docs/tasks/task-063|071/` references are
   pre-existing and out of scope).
5. **Docker image** — `atlas-pr-bootstrap`'s Dockerfile only `COPY`s the
   scripts; a local `docker build` of the service confirms the changed scripts
   still package. (No Go bake target applies.)

## 8. Scope / non-goals (restated)

- No atlas-data code or API change; `/api/data/wz` and `/api/data/process`
  remain for operator use.
- No change to `baseline/publish` or `baseline/restore` (task-095 owns them).
- No change to `predelete-purge` or the external PostDelete DB-drop leak
  (cluster-infra repo, tracked separately — see
  `bug_ephemeral_db_teardown_leak_superuser`).
- The now-unreferenced `atlas-canonical/atlas.zip` object may be deleted by an
  operator; this task does not delete it.

## 9. Files touched

| File | Change |
|---|---|
| `services/atlas-pr-bootstrap/scripts/bootstrap.sh` | Remove `WZ_CANONICAL`/`BOOTSTRAP_MODE`/`resolve_mode`/`full` branch + the `/api/data/wz` wait-ready probe; add early `preflight_baseline` (canonical-version, both objects, transient-aware); restore-only data step; `CANONICAL_TENANT_JSON` var. |
| `services/atlas-pr-bootstrap/scripts/lib.sh` | Update the `resolve_mode` comment reference (cosmetic). |
| `services/atlas-pr-bootstrap/test/bootstrap_test.bats` | Add fail-fast + MinIO-unreachable tests with `curl`/`kubectl` PATH shims. |
| `deploy/k8s/overlays/pr/sync-bootstrap.yaml` | Delete `volumeMounts` entry, `initContainers`, and `volumes`. |
| `docs/runbooks/ephemeral-pr-deployments.md` | Rewrite §9.1 to baseline-only; drop `atlas.zip`/`BOOTSTRAP_MODE`/init-container; link migration runbook. |
| `docs/runbooks/canonical-version-migration.md` | One-line note: bootstrap is baseline-only; publishing a baseline is a prerequisite per version. |
