# Tenant-Aware Job Skill Enumeration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-27
---

## 1. Overview

The set of skills belonging to a job is not constant across MapleStory client
versions. GMS v95.1 adds and removes skills on certain jobs relative to earlier
versions. Atlas currently answers "which skills does job N have?" from a
hardcoded table in `libs/atlas-constants/job/constants.go` — 82 `Job` literals
carrying ~540 skill references, with no tenant or version input. Every tenant,
from GMS v12 through JMS v185, receives the same answer, and that answer is a
v83-shaped list.

This is not a v95.1-only problem. Probing the live cluster (§9.2), skill
`5101001` — "Haste (Super)", under job image 510 — resolves on GMS v48 and v61
and is absent from v72 onward. Job skill sets have been drifting across the
whole version range the project supports; v95.1 is the case that surfaced it.

The fix does not require inventing a version matrix. Each tenant's own
`Skill.wz` archive *is* the authoritative per-version job→skill mapping, and
atlas-data already ingests it: `data/workers/skill.go` walks
`Skill.wz/{jobId}.img`, and `skill/reader.go` parses the job id off the image
filename and iterates that image's `skill` subproperty — the exact mapping — and
then discards the job id, persisting only the individual skill documents. This
task persists that mapping as a first-class `JOB` document type, reads it back
through the existing tenant-scoped `document.Storage`, and deletes the hardcoded
lists.

This is a hard cutover: the constants lists are removed in the same change. JOB
documents exist only after a `Skill.wz` re-ingest or a re-published baseline, so
rollout has a required data step, documented here as a runbook and executed by
the operator at deploy time.

The change also lets atlas-ui stop guessing. `job-advancement-tree.ts` currently
hides job branches using a hand-maintained `BRANCH_FLOORS` / `NODE_FLOORS`
version-floor table, introduced because the backend endpoint was not
version-aware (see the comment at `job-advancement-tree.ts:114`, and task-182,
whose goal was explicitly to "preserve tenant version-floor gating exactly as it
behaves today"). Once the backend can report which jobs exist for a tenant, the
floors are redundant and are retired.

## 2. Goals

Primary goals:

- `GET /data/jobs/{jobId}/skills` returns the skill set that job actually has in
  the requesting tenant's client version, derived from that tenant's ingested
  `Skill.wz`.
- A new `GET /data/jobs` returns every job present for the tenant, as a JSON:API
  compound document whose `skills` relationship can be materialized into
  `included`.
- `libs/atlas-constants/job` stops carrying job→skill data and returns to being
  a constants library with no version-varying content.
- atlas-ui derives job-tree visibility from tenant data instead of the
  `BRANCH_FLOORS` / `NODE_FLOORS` tables, which are deleted.

Non-goals:

- Pruning or migrating character-held skills that a version removes. Nothing
  validates skill-belongs-to-job at grant time, and `job.IdFromSkillId` is pure
  arithmetic, so a stale holder is not an error path. Out of scope.
- `JOB_GRAPH` job topology (parent/child advancement edges) and job display
  names. Neither is carried in `Skill.wz`; both stay hardcoded in the UI.
- `PRESET_JOBS` in the character-preset editor (a curated id→name map). Names
  come from `String.wz`, which is a separate concern.
- Any change to skill *content* (effects, per-level tables). Only enumeration
  changes.
- A transitional fallback to the constants list. Explicitly rejected — see §7.5.

## 3. User Stories

- As a server operator running a GMS v95.1 tenant, I want the jobs explorer to
  show exactly the skills that version's client has, so that what I see matches
  what players get.
- As a server operator running a legacy tenant (GMS v12/v48), I want job
  branches that did not exist in that version to be absent because the data says
  so, not because a hardcoded floor table happens to be correct.
- As a server operator building a character preset, I want "add all job skills"
  to grant the skills valid for my tenant's version, not a v83 list.
- As a developer bringing up a new client version, I want the job→skill mapping
  to come from the WZ ingest I am already running, so that I do not have to
  hand-maintain a per-version skill table.

## 4. Functional Requirements

### FR-1 — `JOB` document type

- **FR-1.1** A new document type `JOB` is persisted through the existing
  `document.Storage` abstraction, keyed by job id.
- **FR-1.2** The stored document carries the job id and the ordered list of
  skill ids found in that job's `Skill.wz` image. It carries no skill detail —
  skill content remains in `SKILL` documents, unduplicated.
- **FR-1.3** `JOB` documents are tenant-scoped and inherit
  `document.Storage.ByIdProvider`'s canonical fallback, which resolves against
  `canonical.TenantId(region, major, minor)`.
- **FR-1.4** `JOB` documents require no changes to `baseline/dump.go` or
  `tenantpurge/purge.go`; both operate on the whole `documents` table rather
  than a per-type list, so the new type participates in baseline publish and
  tenant purge automatically. This must be verified, not assumed.

### FR-2 — Ingest

- **FR-2.1** `JOB` documents are written by the existing `SKILL` worker
  (`data/workers/skill.go`), which already resolves and serializes `Skill.wz`.
  No new entry is added to `data.Workers`; this folds into `SKILL` the same way
  `MOB_SKILL` already does.
- **FR-2.2** The writer reuses the per-image job id that `skill/reader.go`
  already parses from the image filename. It must not re-derive job id by
  dividing skill ids.
- **FR-2.3** Non-numeric images in `Skill.wz` (`MobSkill.img`, and any sibling
  such as `BFSkill`) must produce no `JOB` document.
- **FR-2.4** A job image containing zero skills produces a `JOB` document with
  an empty skill list — distinct from the job being absent entirely.
- **FR-2.5** Monolithic-archive tenants are handled by the existing runtime.
  GMS v12 ships an all-in-one `Data.wz`, which `data/workers/runtime.go` already
  serves to workers as a sub-view (`monolithFile`, `ErrCategoryAbsent`). Because
  the `JOB` writer runs inside the `SKILL` worker, it inherits this with no
  monolith-specific code. **Verified** — see §9.1: v12's `Data.wz` root contains
  a `Skill/` directory, and task-172's live v12 ingest ran the SKILL worker
  successfully. No monolith-specific work is required.

### FR-3 — Read path

- **FR-3.1** `job.ProcessorImpl.GetSkillsForJob` reads the tenant's `JOB`
  document instead of `constJob.Jobs`. All references to `constJob` leave the
  processor.
- **FR-3.2** A job id with no `JOB` document for the tenant (and none via
  canonical fallback) yields HTTP 404. "Unknown job id" and "job does not exist
  in this tenant's version" are deliberately the same response; the UI treats
  404 as "not visible."
- **FR-3.3** `job.InitResource` takes the `*gorm.DB` handle, mirroring
  `skill.InitResource`.

### FR-4 — Job listing endpoint

- **FR-4.1** `GET /data/jobs` returns every `JOB` document for the tenant.
- **FR-4.2** The response is a JSON:API compound document. Each `jobs` resource
  declares a to-many `skills` relationship. The reference implementation to
  follow is `services/atlas-npc-shops/atlas.com/npc/shops/rest.go`, which
  implements `GetReferences` / `GetReferencedIDs` / `GetReferencedStructs` for a
  `shops` → `commodities` to-many.
- **FR-4.3** Relationship linkage (`GetReferencedIDs`) is derived from the
  stored skill id list and is always present. Full skill resources are emitted
  in `included` (`GetReferencedStructs`) only when the client requests them.
- **FR-4.4** Populating `included` must not change what is *persisted*. The
  stored `JOB` document holds skill ids only; resolution of ids to skill
  resources happens in the read/resource layer. The design must state explicitly
  how the persisted shape and the response shape are kept distinct, given that
  `document/db_storage.go:123` marshals the same model type for storage.
- **FR-4.5** The endpoint is paginated consistently with `GET /data/skills`,
  which already uses `paginate.ParseParams`.

### FR-5 — atlas-constants reduction

- **FR-5.1** `Job.skills`, `Job.Skills()`, and `Job.Buffs()` are deleted.
  `Job.Buffs()` currently has zero callers repo-wide.
- **FR-5.2** The 82 exported package-level `Job` value vars are deleted. A
  repo-wide enumeration of every `job.*` / `constJob.*` identifier used outside
  `libs/atlas-constants` confirms no external consumer references them; external
  use is limited to the `*Id` constants, `Jobs`, `Id`, `Type*`, and the function
  helpers.
- **FR-5.3** `Jobs` remains, as a map literal with inline values:
  `HeroId: {id: HeroId, fourthJob: true}`. Its key set remains the "does this
  job exist" check used by the two preset validators in atlas-configurations.
  All 82 job ids and the 23 `fourthJob` markers are preserved exactly.
- **FR-5.4** The `Job` struct is retained as `{id, fourthJob}`. It is not
  collapsed to a `map[Id]bool`: `FromSkillId` returns `(Job, bool)` and external
  callers use `.Id()` (atlas-skills) and `.IsFourthJob()`
  (`atlas-character/skill/model.go:34`).
- **FR-5.5** The `Id` constant block and `Type` constants are untouched.
- **FR-5.6** `libs/atlas-constants/go.mod` gains no dependency. The library must
  not import `atlas-tenant`.
- **FR-5.7** `libs/atlas-constants/README.md`'s `job` row is updated if its
  description implies skill data.

### FR-6 — atlas-ui version-floor retirement

- **FR-6.1** `BRANCH_FLOORS`, `NODE_FLOORS`, and `floorOf` are deleted from
  `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts`.
- **FR-6.2** The `major: number` parameter is removed from `visibleRoots`,
  `visibleChildrenOf`, `advancementChains`, and `subtreeCount` (same file) and
  from `visibleRailGroups` (`components/features/jobs/rail-groups.ts:78`),
  replaced by the set of job ids available for the tenant.
- **FR-6.3** `JobsPage.tsx` sources that set from `GET /data/jobs` rather than
  `activeTenant.attributes.majorVersion`, and its per-job visibility check
  (`JobsPage.tsx:45`) uses set membership.
- **FR-6.4** `JOB_GRAPH`, `JOB_ROOTS`, `childrenOf`, `rootOf`, `jobTreePath`,
  and `tierLabel` are unchanged — they encode topology, not version data.
- **FR-6.5** The stale comment block at `job-advancement-tree.ts:110-129`,
  which documents floor rationale and asserts the backend is not version-gated,
  is removed with the code it describes.
- **FR-6.6** `lib/jobs/__tests__/job-advancement-tree.test.ts:39` asserts
  `BRANCH_FLOORS` exactly and is rewritten against the new data-driven behavior.

### FR-7 — atlas-ui compound-document support

- **FR-7.1** `types/api/responses.ts` has no `included` field today.
  `ApiResponse` gains an optional `included`, and a client primitive is added
  that surfaces it — `api.getList` currently returns `r.data` only
  (`lib/api/client.ts:352`) and discards the rest of the envelope.
- **FR-7.2** A jobs service method fetches the compound document and composes
  jobs with their included skills.
- **FR-7.3** Existing per-job consumers keep working. `JobSkillsAddButton`
  fetches `/jobs/{id}/skills` through the React Query cache keyed on
  `activeTenant.id`; that key is already tenant-scoped and needs no change.

## 5. API Surface

### `GET /data/jobs`

Tenant-scoped list of jobs present in the tenant's ingested data.

Query parameters: pagination consistent with `GET /data/skills`; `include=skills`
to materialize skill resources into `included`.

Response `200`, JSON:API compound document:

```json
{
  "data": [
    {
      "type": "jobs",
      "id": "112",
      "attributes": { "skills": [1121000, 1121001] },
      "relationships": {
        "skills": {
          "data": [
            { "type": "skills", "id": "1121000" },
            { "type": "skills", "id": "1121001" }
          ]
        }
      }
    }
  ],
  "included": [
    { "type": "skills", "id": "1121000", "attributes": { "name": "...", "...": "..." } }
  ]
}
```

`included` is present only when requested. The exact attribute set of an
included `skills` resource is whatever `skill.RestModel` already serializes; this
task does not change it.

### `GET /data/jobs/{jobId}/skills`

Unchanged path, shape, and resource type. Behavior changes from "hardcoded list"
to "this tenant's ingested list."

- `200` — the job exists for the tenant.
- `404` — the job id is unknown *or* absent from this tenant's version.
- `400` — the path segment is not a valid job id (existing `rest.ParseJobId`
  behavior, unchanged).

## 6. Data Model

No relational schema change. `JOB` rides on the existing generic `documents`
table (`document/entity.go`):

| Column | Value for a `JOB` row |
|---|---|
| `tenant_id` | owning tenant (or the canonical tenant for a published baseline) |
| `type` | `JOB` |
| `document_id` | the job id |
| `content` | the marshaled job resource: job id + skill id list |

Uniqueness is `(tenant_id, type, document_id)`, so `JOB` and `SKILL` cannot
collide. `document_id` is `uint32` and every job id fits. `DbStorage.Add`
derives `document_id` by parsing `GetID()` (`document/db_storage.go:141`), which
the existing `job.RestModel` already returns as the job id.

Migration: none in DDL. The data migration is the baseline re-publish in §7.5.

## 7. Service Impact

### 7.1 `libs/atlas-constants`

`job/constants.go` drops from 1,236 lines to approximately 180 — the 82 `Job`
literals spanning lines 7–1058 collapse into inline map values. The `skill`
import leaves `constants.go` (`model.go` still needs it for `mpEaterSkillIds`,
`FromSkillId`, and `IdFromSkillId`, so the package-level dependency remains).
`job/model.go` loses `Skills()` and `Buffs()`.

Consumers verified unaffected — each was checked against the identifiers it
actually uses:

| Consumer | Uses | Affected |
|---|---|---|
| `atlas-skills/skill/processor.go:369-400` | `IdFromSkillId`, `Is`, `Advancement`, `IsFourthJob` | No |
| `atlas-character/skill/model.go:34` | `FromSkillId(...).IsFourthJob()` | No |
| `atlas-character/character/processor.go:1045` | `FromSkillId(...).Id()` | No |
| `atlas-configurations` preset validators (×2) | `Jobs[id]` existence | No |
| `atlas-data/job/processor.go:30` | `Jobs[id].Skills()` | **Yes — rewritten** |

### 7.2 `services/atlas-data`

- `job/` gains a document registry, storage, reader, and the listing resource,
  mirroring the structure of `skill/`.
- `job/processor.go` is rewritten against storage.
- `job/resource.go` gains the list route; `InitResource` gains `*gorm.DB`, so
  `main.go:184`'s route-initializer wiring changes.
- `job/rest.go` gains the JSON:API relationship interfaces.
- `data/workers/skill.go` gains a second registration pass.

### 7.3 `services/atlas-ui`

`lib/jobs/job-advancement-tree.ts`, `components/features/jobs/rail-groups.ts`,
`components/features/jobs/advancement-flow.tsx`, `pages/JobsPage.tsx`,
`types/api/responses.ts`, `lib/api/client.ts`, `services/api/jobs.service.ts`,
plus `lib/jobs/__tests__/job-advancement-tree.test.ts`.

### 7.4 Unaffected services

atlas-character, atlas-skills, atlas-configurations, atlas-channel,
atlas-messages, atlas-consumables. No code change.

### 7.5 Rollout — baseline re-publish (operator-executed)

Hard cutover means the endpoints return 404 for any tenant whose data predates
this change, until that tenant's `Skill.wz` is re-ingested or it is restored
from a re-published baseline.

The design phase must produce a runbook covering all 11 registered versions in
`deploy/k8s/base/versions.json` — GMS 12.1, 48.1, 61.1, 72.1, 79.1, 83.1, 84.1,
87.1, 92.1, 95.1, and JMS 185.1 — with a per-version verification step
confirming `JOB` documents exist before the version is considered done.
Executing the runbook is the operator's deploy-time responsibility and is not
part of this task's implementation.

GMS 12.1 needs an extra step: it is registered in `versions.json` but has no
tenant provisioned and no ingested data in the current cluster (§9.1). It must
be provisioned and ingested, not merely re-published, before its verification
step can pass.

A transitional fallback to the constants list was considered and rejected: it
would keep the deleted table alive for another release and would silently mask a
failed ingest as a successful one.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every read resolves the tenant from context via the
  existing `document.Storage` path. No endpoint may read job skills without a
  tenant in context. The canonical fallback keys on `(region, major, minor)`,
  so versions do not bleed into one another.
- **Performance.** Per-job reads must go through `document.Storage`'s registry
  cache rather than scanning the `SKILL` document set. `GET /data/jobs` returns
  ~82 rows and must not trigger a full skill-table drain when `included` is not
  requested.
- **Observability.** A `Skill.wz` ingest that yields zero `JOB` documents must
  log at warn or above. Silent success here is the failure mode that a
  transitional fallback would have hidden.
- **Testing.** Reader tests over a fixture job image, including the empty-skill
  and non-numeric-image cases (FR-2.3, FR-2.4). `job/resource_test.go` currently
  asserts against the hardcoded list for job 112 and becomes a seeded-storage
  test. UI tests cover data-driven visibility replacing floor-based visibility.
- **Verification gates.** Per CLAUDE.md: `go test -race ./...`, `go vet ./...`,
  and `go build ./...` clean in `libs/atlas-constants` and
  `services/atlas-data`; `docker buildx bake atlas-data` from the worktree root
  if its `go.mod` is touched; `tools/lint.sh --check`; and for atlas-ui,
  `npm run build` (not `vitest` alone).

## 9. Resolved Questions and Open Questions

### 9.1 Resolved — GMS v12's `Data.wz` contains a `Skill` category

Verified against task-172's recorded findings. `design.md:43-45` enumerates the
v12 `Data.wz` root as `Character/`, `Effect/`, `Etc/`, `Item/`, `Map/`, `Mob/`,
`Npc/`, `Reactor/`, **`Skill/`**, `Sound/`, `String/`, `UI/`, plus root-level
`smap.img`/`zmap.img`. The categories v12 genuinely lacks are Quest, Morph,
TamingMob, and `Item/Cash` — `Skill` is not among them.

This is not a static-listing inference. Task-172 ran a live v12 ingest
(`e2e-results.md:105-120`) in which the SKILL worker resolved its monolithic
sub-view and ingested real skills — `1001003` = "Iron Body" with full effects,
plus 175 skill icons. Icon extraction in `data/workers/skill.go` walks
`file.Root().Images()`, skips non-numeric image names via `imgID`, and reads
each image's `skill` subproperty, so numeric per-job images carrying a `skill`
subproperty demonstrably exist in v12. That is exactly the structure the `JOB`
writer consumes. The run's only skipped category was `QUEST`.

**Caveat for the rollout runbook (§7.5):** no GMS 12.1 tenant is currently
provisioned — the live tenant list holds 10 tenants (GMS 48/61/72/79/83/84/87/
92/95 + JMS 185) and task-172's v12 verification tenant is gone. Probing
`/api/data/skills/1000` for GMS 12.1 returns 404, as do maps, monsters,
consumables, equipment, and npcs, so the version has no ingested data in this
cluster at all. v12 must be provisioned and ingested before it can be verified,
and that is a data-state gap, not an archive-capability gap.

### 9.2 Resolved — the per-job-image layout holds on every live version

Probed `/api/data/skills/{id}` across all 10 live tenants for skills drawn from
five different job images — `1000` (job 0), `1001003` (100), `2001002` (200),
`4101004` (410), `1121000` (112). All five resolve `200` on all ten versions:
GMS 48, 61, 72, 79, 83, 84, 87, 92, 95, and JMS 185. `skill/reader.go`'s
assumption holds across every generation the project supports.

The same probe produced direct live evidence for this task's premise. Skill
`5101001` ("Haste (Super)", job image 510) returns `200` on GMS v48 and v61 and
`404` on v72 and every later version. A skill present on a job in one version
and absent in another is precisely what the hardcoded list cannot express, and
it is not limited to the v95.1 case that motivated this task.

### 9.3 Open — persisted-vs-response model split

FR-4.4. One type with a `json:"-"` detail field populated on read (the `shops`
pattern), or two distinct types. A design decision, not a requirement.

### 9.4 Open — non-UI consumers of `GET /data/jobs/{id}/skills`

The repo search found only atlas-ui. Worth one more pass across scripts and seed
data before deleting the constants list.

## 10. Acceptance Criteria

- [ ] A `JOB` document type is persisted per tenant, written by the `SKILL`
      worker, carrying job id + skill id list.
- [ ] `MobSkill.img` and other non-numeric images produce no `JOB` document.
- [ ] A job image with zero skills produces an empty-list `JOB` document, not an
      absent one.
- [ ] `GET /data/jobs/{jobId}/skills` returns the tenant's ingested skill list,
      and 404s for a job absent from the tenant's version.
- [ ] `GET /data/jobs` returns all of the tenant's jobs as a JSON:API compound
      document with a `skills` to-many relationship and optional `included`.
- [ ] Two tenants on different versions receive different skill lists for the
      same job id, demonstrated by test.
- [ ] `Job.skills`, `Job.Skills()`, `Job.Buffs()`, and the 82 exported `Job`
      vars are gone; `job/constants.go` is under ~200 lines.
- [ ] `Jobs` still contains all 82 job ids with all 23 `fourthJob` markers
      intact.
- [ ] `libs/atlas-constants/go.mod` is unchanged and does not depend on
      `atlas-tenant`.
- [ ] atlas-skills, atlas-character, and atlas-configurations compile and pass
      tests with no source changes.
- [ ] `BRANCH_FLOORS`, `NODE_FLOORS`, and `floorOf` are deleted; no `major`
      version parameter remains in the job-tree visibility path.
- [ ] The jobs page renders the tenant's actual job set, verified on a legacy
      version and on GMS v95.1.
- [ ] `baseline` publish and `tenantpurge` include `JOB` documents, verified
      rather than assumed.
- [ ] A baseline re-publish runbook covering all 11 registered versions exists,
      with a per-version verification step.
- [ ] All CLAUDE.md verification gates in §8 pass.
