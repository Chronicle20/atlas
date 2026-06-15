# Baseline-Only Ephemeral Bootstrap — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-14
---

## 1. Overview

Ephemeral PR environments are seeded by `atlas-pr-bootstrap`. Today its data step has three modes
(`BOOTSTRAP_MODE=auto|baseline|full`): `baseline` restores a published canonical baseline (fast, ~60s),
`full` uploads a ~1 GB `atlas.zip` WZ bundle and runs a full ingest (~10 min), and `auto` probes for a
baseline and **falls back to `full`** when absent. A `fetch-wz-canonical` init container unconditionally
downloads `atlas-canonical/atlas.zip` into an `/opt/wz` volume before the main bootstrap runs.

This design has two problems, both observed in production:

1. **MinIO bloat.** The `full` path re-ingests a complete WZ tree into each env's own tenant prefix in the
   shared MinIO buckets. This was the original driver of the multi-gigabyte per-tenant duplication that
   task-095 (version-scoped canonical fallback) and the canonical-version-migration runbook set out to
   eliminate. As long as `full` remains a silent fallback, the bloat can recur.
2. **Hard `atlas.zip` dependency.** `fetch-wz-canonical` fetches `atlas.zip` *unconditionally* — even when a
   baseline exists and `full` will never run. When the object is absent (e.g., after a deliberate
   clean-slate MinIO wipe), the init container 404s (`curl: (22) … 404`), the bootstrap Job fails, the
   env's databases are never seeded, and downstream services crash-loop. This happened on 2026-06-14:
   wiping `atlas-canonical/atlas.zip` broke every new ephemeral env even though valid per-version baselines
   existed.

With task-095 making canonical data version-scoped and `baseline/publish` producing correct per-version
baselines, the intended provisioning contract is now simply: **ephemeral envs restore from a published
canonical baseline for their version.** This task removes the `full`/`atlas.zip` path entirely, makes the
bootstrap baseline-only, and fails fast with an actionable message when no baseline exists — turning
cold-start canonical provisioning into an explicit operator step (the task-095
`canonical-version-migration.md` runbook) rather than an automatic, bloat-producing full ingest.

### Relationship to other work
- **task-095 (version-scoped canonical fallback, #778)** made canonical data and baselines version-correct.
  This task makes the bootstrap *consume* baselines exclusively.
- **task-071 (gamedata MinIO consolidation)** introduced `BOOTSTRAP_MODE` and the auto→full fallback being
  removed here.
- **Out of scope:** the external PostDelete DB-drop / superuser-termination issue that leaves orphaned
  per-env databases lives in the cluster-infra repo, not here.

## 2. Goals

Primary goals:
- The ephemeral PR bootstrap provisions data **only** by restoring a published canonical baseline for the
  env's `(region, majorVersion, minorVersion)`.
- The `full` ingest path, the `atlas.zip`/`WZ_CANONICAL` dependency, and the `fetch-wz-canonical` init
  container (plus its `/opt/wz` volume/PVC) are removed entirely.
- When no baseline exists for the env's version, the bootstrap **fails fast, before bringing services up**,
  with a clear, actionable error pointing at the canonical-provisioning runbook.
- The runbook documentation is updated to reflect baseline-only provisioning (no `atlas.zip` upload step).

Non-goals:
- Removing or changing the `PATCH /api/data/wz` or `POST /api/data/process` endpoints in atlas-data —
  operators still use them (shared scope) to provision canonical data. Only the *bootstrap's* use of them
  is removed.
- Changing `predelete-purge` behavior.
- Fixing the external PostDelete `DROP DATABASE` orphaned-DB leak (separate cluster-infra repo; tracked
  separately).
- Changing which version(s) an env bootstraps, or adding multi-version bootstrap.
- Changing `baseline/publish` or `baseline/restore` themselves (task-095 owns those).

## 3. User Stories

- As a **platform operator**, I want a PR env to come up fast by restoring the canonical baseline, with no
  multi-gigabyte WZ re-ingest, so ephemeral envs never re-introduce per-tenant MinIO bloat.
- As a **platform operator**, I want a missing baseline to fail the bootstrap loudly with a message telling
  me to publish one, instead of silently full-ingesting or dying on a confusing `atlas.zip` 404.
- As a **developer opening a PR**, I want my env to either come up correctly or fail with a clear reason,
  not crash-loop with empty databases.

## 4. Functional Requirements

### FR-1 Baseline-only data provisioning
- **FR-1.1** `bootstrap.sh` data step restores the canonical baseline via `POST /api/data/baseline/restore`
  for the env's `(region, major, minor)` and tenant. No other data-ingest path remains.
- **FR-1.2** Remove `BOOTSTRAP_MODE` entirely (always baseline). Remove `WZ_CANONICAL` and all `full`-mode
  logic (the `PATCH /api/data/wz` upload + `POST /api/data/process` calls in the bootstrap) and the
  `auto`→`full` fallback in `resolve_mode`.
- **FR-1.3** Remove the `fetch-wz-canonical` init container and the `/opt/wz` volume mount from
  `sync-bootstrap.yaml`. Remove the backing volume and, if present and unused elsewhere, the
  `atlas-wz-canonical-readonly` PVC.

### FR-2 Fail-fast on missing baseline
- **FR-2.1** Before performing data-affecting work (ideally as an early preflight in `bootstrap.sh`), probe
  for the baseline (`HEAD atlas-canonical/baseline/regions/<region>/versions/<major>.<minor>/documents.dump.sha256`,
  the existing `canonical_baseline_exists` check).
- **FR-2.2** If the baseline is absent, the bootstrap exits non-zero with an actionable message naming the
  version and pointing at `docs/runbooks/canonical-version-migration.md` (e.g.
  `"no canonical baseline for GMS 84.1 — publish one (see canonical-version-migration runbook) before
  deploying this env"`). The Argo CD Job surfaces the failure; the env does not come up half-seeded.
- **FR-2.3** The failure is deterministic and idempotent — re-syncing after a baseline is published
  succeeds without manual cleanup.

### FR-3 Documentation
- **FR-3.1** `docs/runbooks/ephemeral-pr-deployments.md` is updated: remove the `atlas.zip` bucket/upload
  section, the `fetch-wz-canonical` description, and the `BOOTSTRAP_MODE` table; document baseline-only
  provisioning and link to `canonical-version-migration.md` for cold-start.

## 5. API Surface

No atlas-data API changes. The bootstrap stops calling `PATCH /api/data/wz` and `POST /api/data/process`;
it continues to call `POST /api/data/baseline/restore` (unchanged contract). The baseline-presence probe is
an anonymous `HEAD` against MinIO (unchanged).

## 6. Data Model

No schema changes. (Canonical/baseline storage is owned by task-095.)

## 7. Service Impact

- **`services/atlas-pr-bootstrap/scripts/bootstrap.sh`** — remove `WZ_CANONICAL`, `BOOTSTRAP_MODE`,
  `resolve_mode` fallback, and the `full` branch; keep/relocate the `canonical_baseline_exists` probe to an
  early preflight that hard-fails per FR-2; data step becomes baseline-restore only. (Check `lib.sh` /
  `cleanup.sh` for stale references.)
- **`deploy/k8s/overlays/pr/sync-bootstrap.yaml`** — delete the `fetch-wz-canonical` init container, the
  `/opt/wz` mount, and the backing volume/PVC.
- **`docs/runbooks/ephemeral-pr-deployments.md`** — rewrite per FR-3.
- **No code change** to atlas-data; the `wzinput` handler / `/api/data/wz` endpoint remain for operator use.

## 8. Non-Functional Requirements

- **No bloat:** after this change, no ephemeral env can write a per-tenant WZ/asset tree to shared MinIO via
  bootstrap (the only path that did so is removed). Verifiable: a freshly bootstrapped env produces no
  `tenants/<id>/` prefix in `atlas-wz`/`atlas-assets`.
- **Fast + deterministic:** baseline restore (~60s) is the only path; no ~10-min ingest, no ~1 GB download.
- **Observability:** the fail-fast path logs a single clear, greppable error line; the success path logs the
  baseline restore as today.
- **Idempotency:** re-running bootstrap (Argo re-sync) is safe — restore is idempotent; the preflight is a
  read-only probe.
- **Backward compatibility:** existing envs are unaffected at runtime; this only changes how *new* envs
  bootstrap. The `atlas-canonical/atlas.zip` object becomes unreferenced (operators may delete it; not
  required by this task).

## 9. Open Questions

- **OQ-1** Does the `/opt/wz` mount use an `emptyDir` or a named PVC (`atlas-wz-canonical-readonly`)? Confirm
  in design and remove whichever is present; ensure no other manifest references it.
- **OQ-2** Are there sibling scripts (`lib.sh`, `cleanup.sh`, helm/chart values) that set `BOOTSTRAP_MODE` or
  `WZ_CANONICAL` and must be cleaned up to avoid dangling env wiring?
- **OQ-3** Should the preflight also assert the baseline `documents.dump` (not just the `.sha256` sidecar)
  exists, to avoid a half-published baseline passing the probe? (Lean yes — cheap extra HEAD.)

## 10. Acceptance Criteria

- [ ] `bootstrap.sh` has no `BOOTSTRAP_MODE`, `WZ_CANONICAL`, `full`-mode, or `auto`→`full` fallback;
      data provisioning is baseline-restore only.
- [ ] `sync-bootstrap.yaml` has no `fetch-wz-canonical` init container, no `/opt/wz` mount, and no orphaned
      volume/PVC.
- [ ] With a published baseline for the env's version, a fresh env bootstraps successfully via baseline
      restore and its `atlas-data` DB is populated.
- [ ] With NO baseline for the env's version, the bootstrap Job fails fast (non-zero, before service
      bring-up) with a message naming the version and referencing the canonical-version-migration runbook;
      Argo surfaces the failure.
- [ ] A freshly bootstrapped env creates no `tenants/<id>/` prefix in `atlas-wz`/`atlas-assets` (no full
      ingest path exists).
- [ ] `docs/runbooks/ephemeral-pr-deployments.md` reflects baseline-only provisioning; no `atlas.zip`
      upload instructions remain; links to `canonical-version-migration.md`.
- [ ] No references to `atlas.zip` / `WZ_CANONICAL` / `BOOTSTRAP_MODE` remain in `atlas-pr-bootstrap` or the
      `pr` overlay (grep clean), except where intentionally documented as removed.
