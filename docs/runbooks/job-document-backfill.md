# Runbook — JOB document backfill (task-185)

**Audience:** operator, at deploy time.
**Status:** required before task-185 ships to a tenant.

## Why

task-185 made the job→skill mapping tenant data instead of a compiled-in table.
`JOB` documents are written only by a `Skill.wz` ingest, so **until a tenant is
re-ingested or restored from a re-published baseline**:

- `GET /api/data/jobs` returns 200 with an empty array and `meta.total: 0`
- `GET /api/data/jobs/{jobId}/skills` returns 404 for every job
- the atlas-ui Jobs page renders an empty branch rail

This is a deliberate hard cutover. A transitional fallback to the old constants
list was considered and rejected: it would have kept the deleted table alive for
another release and would have masked a failed ingest as a success.

## Scope

All 11 versions registered in `deploy/k8s/base/versions.json`:

GMS 12.1, 48.1, 61.1, 72.1, 79.1, 83.1, 84.1, 87.1, 92.1, 95.1, JMS 185.1.

## Per-version procedure

Run these in order, one version at a time. Do not batch — step 3 is the gate.

1. **Re-ingest the version's canonical dataset** from its already-uploaded WZ
   archives:

   ```
   POST /api/data/process?scope=shared
   X-Atlas-Operator: 1
   ```

2. **Poll until the job reports `succeeded`:**

   ```
   GET /api/data/process
   ```

3. **Verify — this is the gate.** For a tenant on that version:

   ```
   GET /api/data/jobs
   ```

   - `meta.total` must be `> 0`.
   - Spot-check the returned id set against the expectation table below.

   `GET /api/data/status` is **not** sufficient: it reports only an aggregate
   `documentCount` with no per-type breakdown, so it cannot distinguish "JOB
   documents were written" from "skills were written and JOB was not."

   If `meta.total` is 0, check the ingest logs for
   `Skill.wz ingest produced no JOB documents` — the worker warns explicitly on
   this case. Do not proceed to step 4.

4. **Publish the baseline** so ephemeral PR environments (baseline-only, and
   they fail fast without one) pick up the `JOB` documents:

   ```
   POST /api/data/baseline/publish
   X-Atlas-Operator: 1
   Content-Type: application/vnd.api+json
   ```

   Request body:

   ```json
   {
     "data": {
       "type": "baselinePublishes",
       "attributes": {
         "region": "GMS",
         "majorVersion": 83,
         "minorVersion": 1
       }
     }
   }
   ```

## Expected job-set changes

Derived from probing `GET /api/data/skills/{id}` across the live tenants for one
representative skill per job image (task-185 design §3). The Jobs page changes
visibly on five of the ten currently-provisioned versions:

| Version | Change vs the retired floor table |
|---|---|
| GMS 48 | GM (900) + Super GM (910) **disappear** — no `9xx` skills exist at this version |
| GMS 61 | no change — GM/Super GM stay |
| GMS 72 | Maple Leaf Brigadier (800) + the whole Cygnus branch (1000) **appear** |
| GMS 79 | Maple Leaf Brigadier + Cygnus **appear** (same cause) |
| GMS 83/84/87/92/95 | no change |
| JMS 185 | Super GM (910) **disappears**; GM (900) stays |

These are correct outcomes, not ingest failures. Every one of them moves the UI
toward the tenant's actual data.

**Caveat on the table's provenance:** it probes `SKILL` documents as a *proxy*
for the presence of a per-job image, because no `JOB` document existed anywhere
when it was produced. Two ways the proxy can be wrong — a job image can exist
with zero skills (the job then appears with an empty skill list, which the Jobs
page renders as its "empty" state), and a representative skill can be absent
while its image exists. Step 3's `GET /api/data/jobs` check is the authoritative
verification; this table only sets the expectation it is checked against.

## GMS 12.1 — extra step

GMS 12.1 is registered in `versions.json` but has **no provisioned tenant and no
ingested data** in the current cluster: the live tenant list holds exactly ten
tenants (GMS 48/61/72/79/83/84/87/92/95 + JMS 185), and probing
`/api/data/skills/1000` for GMS 12.1 returns 404, as do maps, monsters,
consumables, equipment, and npcs.

It must be **provisioned and ingested**, not merely re-published, before its
step-3 verification can pass.

This is a data-state gap, not an archive-capability gap: v12's monolithic
`Data.wz` root does contain a `Skill/` directory, and task-172's live v12 ingest
ran the SKILL worker successfully (skill `1001003` "Iron Body" with full effects,
plus 175 skill icons). The categories v12 genuinely lacks are Quest, Morph,
TamingMob, and `Item/Cash`.

## Rollback

There is no data rollback: `JOB` documents are additive and no existing document
type changes. Rolling back the *code* restores the old compiled-in list and
leaves the `JOB` rows in place, harmlessly.
