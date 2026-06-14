# Canonical Version Migration — Runbook

> **See also:** [ephemeral-pr-deployments.md](ephemeral-pr-deployments.md) for
> per-PR bootstrap procedures that consume the per-version baselines produced
> by this migration.

Operational guide for the one-time, per-environment migration introduced by
task-095 (version-scoped canonical fallback).

## Background

Prior to task-095, canonical (baseline) game data was stored under the
all-zeros sentinel tenant id `00000000-0000-0000-0000-000000000000`, shared
by every version. From task-095 onward, canonical data is stored under a
deterministic, per-version id derived as:

```
canonical.TenantId(region, major, minor)   // UUIDv5
```

Each game version gets its own isolated canonical dataset. When a tenant has
no per-tenant rows for a given resource, atlas-data falls back to that
version's canonical dataset rather than the global sentinel.

**Ordering guarantee:** steps 1–4 are purely additive. The legacy sentinel
rows are not removed until step 5, so no tenant ever falls back to an empty
canonical dataset during the cutover.

**`atlas-pr-bootstrap` impact:** the bootstrap is **baseline-only** — it
provisions PR-env data exclusively by restoring the published canonical
baseline for the env's version and **fails fast** (before bringing services
up) when none exists. Publishing the per-version baseline (step 4) is
therefore a **prerequisite** for any ephemeral env on that version, not an
optimization. See [ephemeral-pr-deployments.md](ephemeral-pr-deployments.md) §9.1.

---

## Migration Steps

### Step 1 — Deploy the new atlas-data image

Deploy the task-095 image to the target environment. Existing per-tenant rows
are untouched; tenants that have their own ingested data continue to serve
normally throughout the migration.

```sh
kubectl rollout status deployment/atlas-data -n atlas --timeout=300s
```

### Step 2 — Ingest shared (canonical) data per version

For each live version, trigger a shared-scope ingest. The `X-Atlas-Operator:
1` header is required; pass the version's region/major/minor via the standard
tenant headers.

```sh
# Template — repeat for each version below.
curl -X POST "https://<atlas-data-host>/api/data/process?scope=shared" \
  -H "X-Atlas-Operator: 1" \
  -H "TENANT_ID: <tenant-uuid>" \
  -H "REGION: <region>" \
  -H "MAJOR_VERSION: <major>" \
  -H "MINOR_VERSION: <minor>"
```

Run for all six live versions:

| Version     | REGION | MAJOR_VERSION | MINOR_VERSION |
|-------------|--------|---------------|---------------|
| GMS 83.1    | GMS    | 83            | 1             |
| GMS 84.1    | GMS    | 84            | 1             |
| GMS 87.1    | GMS    | 87            | 1             |
| GMS 92.1    | GMS    | 92            | 1             |
| GMS 95.1    | GMS    | 95            | 1             |
| JMS 185.1   | JMS    | 185           | 1             |

Use any active tenant UUID for the `TENANT_ID` header that belongs to the
target version — it is only used to derive the region/major/minor; the ingest
writes under the version-scoped canonical id, not the caller's tenant id.

### Step 3 — Verify per version

For each version, confirm the canonical dataset was written:

```sh
# Returns JSON:API with documentCount > 0 when ingest succeeded.
curl "https://<atlas-data-host>/api/data/status?scope=shared" \
  -H "X-Atlas-Operator: 1" \
  -H "TENANT_ID: <tenant-uuid>" \
  -H "REGION: <region>" \
  -H "MAJOR_VERSION: <major>" \
  -H "MINOR_VERSION: <minor>"
```

Spot-check fallback for a tenant of that version that has **no** per-tenant
data (e.g. a freshly provisioned ephemeral tenant):

```sh
curl "https://<atlas-data-host>/api/data/status" \
  -H "TENANT_ID: <ephemeral-tenant-uuid>" \
  -H "REGION: <region>" \
  -H "MAJOR_VERSION: <major>" \
  -H "MINOR_VERSION: <minor>"
```

The response should reflect the canonical row count, not zero.

### Step 4 — Re-publish baselines per version

Publish the version-correct dump and sha256 sidecar so the ephemeral baseline-only
bootstrap can restore the correct canonical dataset:

```sh
curl -X POST "https://<atlas-data-host>/api/data/baseline/publish" \
  -H "X-Atlas-Operator: 1" \
  -H "TENANT_ID: <tenant-uuid>" \
  -H "REGION: <region>" \
  -H "MAJOR_VERSION: <major>" \
  -H "MINOR_VERSION: <minor>"
```

Repeat for each version in the table above.

### Step 5 — Legacy cleanup (idempotent, safe to re-run)

Remove the all-zeros sentinel rows from every indexed table. This step is
safe to re-run; it deletes only the legacy sentinel rows. Do **not** delete
from `tenant_baselines` — canonical data never had a row there.

```sql
DELETE FROM documents             WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM monster_search_index  WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM npc_search_index      WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM reactor_search_index  WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM map_search_index      WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM item_string_search_index WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
```

---

## Rollback / Safety

Steps 1–4 are purely additive and non-destructive — they write new rows under
new tenant ids and do not modify any existing data. They can be safely rolled
back by redeploying the previous atlas-data image (tenants fall back to their
own per-tenant data or get an empty response for the canonical scope, which is
the pre-task-095 behaviour).

Only step 5 deletes data, and only the legacy all-zeros sentinel rows. Do not
run step 5 until steps 2–3 have been verified for every live version.
