# Version-Correct Job Hierarchy — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07
---

## 1. Overview

The atlas-ui Jobs page presents a job advancement hierarchy that is structurally
version-blind. Its shape (parent edges and display names) comes from a
hardcoded, v83-keyed table — `JOB_GRAPH` in
`services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` — and its visibility
gate comes from `GET /api/data/jobs`, which reports whatever job images exist in
the tenant's ingested `Skill.wz`. Neither input knows what a client version
actually *released*. The result is three classes of wrong rendering: a v0.48
tenant shows wire ids 500/510 as "Pirate → Brawler" when that version binds them
to GM → Super GM; a v0.72 tenant shows the Cygnus Knights branch because the WZ
carries an unreleased stub; a v0.79 tenant shows Aran for the same reason.

task-187 already built the correct source of truth for two of these three
problems. `GET /api/data/job-availability` returns the tenant version's
*released* job identities with version-correct wire ids and display names,
derived from `constants.For(region, major, minor)`. The presets picker adopted
it (`usePresetJobOptions`); the Jobs page never did. What the endpoint does not
yet supply is the advancement hierarchy itself — it returns a flat `{id, name}`
list — and the parent edges are exactly what diverges at v0.48. So closing this
gap requires extending the endpoint, not just re-pointing the page at it.

Two independent data defects surfaced during the investigation that produced
this PRD, both verified against the WZ and the live baseline on 2026-08-07.
First, every Evan advancement stage (`2200`–`2218`) reports zero skills at
v0.84+ because a silent-swallow bug in the JOB reader lets `Skill.wz/Dragon/`'s
identically-named animation images overwrite the real job documents. Second,
`availability.csv` tracks release status at whole-class granularity, so Cygnus
4th job is marked released even though those images are empty in the WZ at every
supported version. Both are backend data-correctness bugs that the UI is
faithfully rendering; fixing the UI alone would not fix them.

## 2. Goals

Primary goals:

- The Jobs page hierarchy is correct for every supported client version: correct
  parent edges, correct display names, and correct visibility, with no
  version-specific knowledge hardcoded in the frontend.
- `GET /api/data/job-availability` becomes the single authority for
  version-aware job identity — id, name, **and** advancement parent — so any
  client gets the same answer without reimplementing per-version semantics.
- A numeric `Skill.wz` image with no `skill` node can never blank a previously
  ingested JOB document.
- `availability.csv` release status is accurate at advancement-tier granularity,
  not just class granularity, and is verified against the WZ rather than
  asserted from a patch log alone.

Non-goals:

- Re-ingesting `Skill.wz` and republishing baselines to repair the already-bad
  Evan JOB documents in live environments. The reader fix lands here with tests;
  the data operation is tracked as a separate operational follow-up (see §9).
  Evan will continue to show zero skills in every environment until that
  follow-up runs — this is an accepted, explicitly-scoped-out outcome.
- Changing which skills belong to a job, or any skill-effect parsing. The skill
  ingest path is correct and stays untouched.
- Adding Cygnus 4th-job content (skills, presets, or advancement) that the game
  never shipped. See §9 regarding the stale `docs/TODO.md` entry.
- Redesigning the Jobs page layout, rail grouping, or skill-detail panel. This
  is a correctness change to the data feeding the existing UI.
- Resistance, Dual Blade, and Mechanic release classes. They have no identity in
  the namespace (`classOf` never returns them) and remain inert.

## 3. User Stories

- As an operator browsing a v0.48 tenant, I want wire id 500 to render as "Gm"
  with Super Gm beneath it, so that the Jobs page reflects what that client
  actually binds those ids to instead of showing a Pirate branch that version
  never had.
- As an operator browsing a v0.72 or v0.79 tenant, I want unreleased classes
  (Cygnus, Aran) hidden, so that an unreleased WZ stub is not mistaken for a
  playable class.
- As an operator browsing any tenant, I want Cygnus Knights to end at 3rd job,
  so that the UI does not advertise a 4th-job tier that has no skills at any
  version we support.
- As an operator browsing a v0.84+ tenant, I want each Evan stage to list its
  skills, so that the Evan class is inspectable like every other class.
- As a backend developer adding a future client version, I want release status
  and advancement edges to come from generated per-version tables, so that a new
  version's divergence is a data change rather than a frontend patch.

## 4. Functional Requirements

### FR-1 — JOB ingest must not blank documents (atlas-data)

- **FR-1.1** `job.Read` (`services/atlas-data/atlas.com/data/job/reader.go`)
  MUST yield **no model** — not a model with an empty skill list — when the
  image's root node has no `skill` child. This matches how it already treats a
  non-numeric image (`MobSkill.img`), and it is the specific defect: the current
  `if ssxml, err := exml.ChildByName("skill"); err == nil` swallows the error
  and emits an empty document.
- **FR-1.2** A numeric image that HAS a `skill` node with zero children MUST
  still yield a model with an empty skill list. "The job exists with zero
  skills" stays representable and distinguishable from "the job is absent" —
  this is the pre-existing FR-2.4 contract from task-185, and it is exactly the
  legitimate Cygnus 4th-job case (`1112.img` has an empty-but-present `skill`
  node). FR-1.1 and FR-1.2 differ only in whether the node is present; the
  implementation must not collapse the two.
- **FR-1.3** Ingesting a `Skill.wz` whose tree contains a subdirectory with
  images named identically to top-level job images (v0.84+ `Skill.wz/Dragon/`
  holds `2200.img`–`2218.img`) MUST leave the top-level images' JOB documents
  intact, regardless of directory walk order.
- **FR-1.4** Observability: the ingest summary MUST make a skipped image
  visible. A numeric image skipped under FR-1.1 is expected and benign for
  `Dragon/`, but silence here is what let this bug live — the run summary must
  distinguish "images seen" from "documents written".

### FR-2 — Tier-accurate release availability (libs/atlas-constants)

- **FR-2.1** Cygnus 4th-job identities (canonical tokens `1112`, `1212`, `1312`,
  `1412`, `1512`, and their `1112xxxx`-style skill tokens) MUST resolve to a
  release class distinct from the Cygnus tiers 1–3, and that class MUST be
  `released=false` for every supported version (gms 12/48/61/72/79/83/84/87/92/95,
  jms 185).
- **FR-2.2** `classOf` (`libs/atlas-constants/gen/availability.go`) MUST return
  the new label for those tokens and continue to return `Cygnus` for the rest of
  the `1000`–`1599` range. The existing floor-by-10000 relationship that lets
  one range table serve both the job and skill domains MUST be preserved.
- **FR-2.3** Every other release class in `availability.csv` (`GM`, `SuperGM`,
  `Pirate`, `Aran`, `Evan`) MUST be audited for the same tier-level over-claim,
  using the WZ as the evidence base: for each (class, version) marked
  `released=true`, confirm the corresponding job images actually carry skills.
  Findings MUST be recorded in the task folder with per-version evidence, and
  any confirmed mismatch fixed the same way as FR-2.1. Classes verified as
  correct MUST be recorded as such — a silent pass is not a result.
- **FR-2.4** `availability.csv` rows added or changed MUST carry a `meymink`
  justification consistent with the file's existing convention, and MUST cite
  the WZ evidence where the patch log and the WZ disagree. Where they conflict,
  the WZ wins for *content* questions ("does this tier have skills") and the
  patch log wins for *release-date* questions.
- **FR-2.5** The generated per-version sets MUST be regenerated, and
  `libs/atlas-constants/gen/audit_validate_test.go` MUST pass against the
  amended CSV.

### FR-3 — Advancement edges in the availability API (atlas-data + libs)

- **FR-3.1** `libs/atlas-constants/job` MUST gain a version-aware advancement
  parent relation, keyed by `Identity` (no such table exists today — `Job` holds
  only `id` and `fourthJob`). The relation MUST be expressible per version so
  that v0.48's Gm → Super Gm edge and v0.61+'s Pirate → Brawler edge coexist
  without either being a special case in consuming code.
- **FR-3.2** The relation MUST preserve task-182's display convention: GM and
  Super GM present as an advancement line beneath Beginner (Beginner → GM →
  Super GM), rather than as the two independent roots that
  `libs/atlas-constants/job/constants.go` models. Moving this convention from
  the UI into the shared library is a deliberate transfer of ownership; it MUST
  be documented at the definition site so it is not later "corrected" back.
- **FR-3.3** The `job-availability` resource MUST expose the parent as an
  attribute. Parent MUST be the version's **wire id**, consistent with the
  resource's existing id semantics, and MUST be null/absent for a root.
- **FR-3.4** A parent MUST always refer to a job that is itself available at
  that version. The API MUST NOT emit an edge pointing at an unavailable job;
  if the natural parent is unavailable, the entry is a root for that version.
- **FR-3.5** The endpoint MUST remain paginated and MUST keep its current
  ordering (ascending by wire id).

### FR-4 — Jobs page sources version-correct data (atlas-ui)

- **FR-4.1** Job visibility MUST be the intersection of release availability
  (`GET /api/data/job-availability`) and WZ presence (`GET /api/data/jobs`). A
  job renders only if both report it.
- **FR-4.2** Display names MUST come from the availability response, which is
  version-correct. `JOB_GRAPH`'s name table and the duplicate `jobNameMap` in
  `services/atlas-ui/src/lib/jobs.ts` MUST no longer be the name authority for
  this page; any remaining consumer of `jobName`/`getJobNameById` must be
  identified and either migrated or explicitly justified.
- **FR-4.3** Parent edges MUST come from the availability response.
  `JOB_GRAPH`'s hardcoded `parent` field MUST no longer drive the Jobs page
  hierarchy. Derived helpers (`rootOf`, `childrenOf`, `jobTreePath`,
  `tierLabel`, `advancementChains`, `subtreeCount`) MUST operate on the
  API-supplied graph.
- **FR-4.4** Both queries' pending and error states MUST mean "unknown", never
  "empty". `TenantProvider` clears the query cache on every tenant switch, so an
  empty set immediately after a switch is the normal pending state. The existing
  `isSuccess` gating discipline (task-182 design D10 — do not redirect a valid
  `/jobs/112` to `/jobs` on a pending set) MUST be preserved across both
  queries.
- **FR-4.5** If either query fails, the page MUST surface a load error rather
  than silently rendering a partial or version-blind hierarchy.
- **FR-4.6** Rail grouping (`rail-groups.ts`) MUST continue to drop empty
  groups, and MUST do so against the intersected set — e.g. the Cygnus Knights
  group disappears entirely on a v0.72 tenant, and the Explorers rail on a v0.48
  tenant has no Pirate entry while the Special group gains GM.
- **FR-4.7** The Jobs page MUST NOT hardcode any version comparison
  (`major <= 48` or similar). All version-specific behavior arrives as data.

## 5. API Surface

### Modified: `GET /api/data/job-availability`

Existing behavior (unchanged): returns the requesting tenant's released job
identities as JSON:API `job-availability` resources, `id` = the version's wire
id, paginated, ascending by wire id. Tenant resolved from the four standard
headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).

Added: an advancement parent attribute.

```json
{
  "data": [
    { "type": "job-availability", "id": "0",
      "attributes": { "name": "Beginner", "parent": null } },
    { "type": "job-availability", "id": "500",
      "attributes": { "name": "Gm", "parent": 0 } },
    { "type": "job-availability", "id": "510",
      "attributes": { "name": "Super Gm", "parent": 500 } }
  ],
  "meta": { "total": 40, "page": { "number": 1, "size": 250, "last": 1 } },
  "links": { "self": "...", "first": "...", "last": "..." }
}
```

The same tenant on gms 72.1 returns `{"id": "500", "attributes": {"name":
"Pirate", "parent": 0}}` and separately `{"id": "900", "attributes": {"name":
"Gm", "parent": 0}}` — the wire id 500 divergence is resolved server-side.

Error cases are unchanged: a malformed `page[...]` parameter is a 400; missing
or malformed tenant headers fail per the existing shared tenant middleware. An
unprovisioned version yields an empty `data` array with `meta.total: 0`, not an
error.

Note the attribute shape must accommodate a null parent. `RestModel.Id` is
currently `uint16` with `json:"-"` (id is carried by JSON:API's `GetID`); the
parent field needs a representation where "root" is unambiguous and not
confusable with wire id 0 (Beginner is a legitimate id 0). A nullable pointer
type is the obvious answer; the design phase should confirm it round-trips
through api2go's marshalling.

No new endpoints. `GET /api/data/jobs` and `GET /api/data/jobs/{id}/skills` are
unchanged in shape; the JOB documents they serve become correct for Evan once
the FR-1 fix is re-ingested.

## 6. Data Model

No database schema changes. The `documents` table
(`tenant_id, type, document_id, content`, unique on the first three, upserted
with `DoUpdates: content`) is untouched — FR-1 changes *what gets written*, not
the storage contract.

Changed data artifacts:

- `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv` — new
  rows for the Cygnus 4th-job class across all eleven version keys, plus any
  rows amended by the FR-2.3 audit. This file is a generator input, not
  documentation-only; it is consumed by
  `libs/atlas-constants/gen/availability.go` at generation time via a relative
  path.
- `libs/atlas-constants/job/version_*_gen.go` — regenerated `available_*` maps.
  Cygnus `*Stage4` identities drop out of v79/v83/v84/v87/v92/v95/jms185.
- `libs/atlas-constants/job` — a new per-version advancement parent relation
  (FR-3.1). Whether this is a generated per-version map or a version-blind
  identity relation combined with the existing identity→wire binding is a design
  decision; the second is smaller if the *identity-level* parent never varies by
  version. The v0.48 case suggests it does not: Gm's parent is Beginner in both
  worlds, and only the wire id differs. This should be confirmed in design
  rather than assumed here.

Migration notes: none required for the schema. The Evan JOB documents already
persisted in every v0.84+ environment are stale and will remain so until a
re-ingest; nothing in this task mutates them.

## 7. Service Impact

**atlas-data** — `job/reader.go` (FR-1.1/1.2), the SKILL worker's job-pass
observability in `data/workers/skill.go` (FR-1.4), and the `jobavailability`
package's `RestModel` and processor (FR-3.3/3.4). Test coverage in
`job/reader_test.go` and `jobavailability/resource_test.go`.

**libs/atlas-constants** — the availability generator and its CSV input (FR-2),
the new advancement relation (FR-3.1/3.2), and all regenerated per-version
files. `audit_validate_test.go` gates the CSV.

**atlas-ui** — `pages/JobsPage.tsx`, `lib/jobs/job-advancement-tree.ts`,
`lib/jobs.ts`, `components/features/jobs/rail-groups.ts`,
`services/api/availability.service.ts`, `lib/hooks/api/useJobAvailability.ts`,
and the colocated tests for each. `usePresetJobOptions` already consumes the
endpoint and must keep working unchanged — the added attribute must be additive.

No Kafka topics, no other services, no deploy-surface changes.

## 8. Non-Functional Requirements

**Multi-tenancy** — every read path is tenant-scoped via the four standard
headers and `tenant.MustFromContext`. The Jobs page now issues two tenant-scoped
queries; both must be keyed by tenant id in React Query so a tenant switch
cannot serve one tenant's hierarchy against another's availability. The existing
`queryClient.clear()` on switch covers this, and the query-key convention in
`jobAvailabilityKeys.list(tenantId)` must be matched by the jobs query.

**Performance** — `job-availability` is served from in-memory generated tables
with no database or WZ access; adding a parent attribute does not change its
cost. The UI's second query is cached with the existing 30-minute
`staleTime` / 24-hour `gcTime`, so the intersection costs one extra request per
tenant per half hour, not per navigation.

**Correctness over convenience** — FR-1.1 and FR-2.1 are both cases where the
previous code chose a permissive default (swallow the error; assume the whole
class shipped) and produced silently wrong data. Neither replacement may
reintroduce a silent fallback: a skipped image is logged, and an unclassified
identity keeps its existing explicit behavior.

**Observability** — the existing `job documents: written=N` summary and its
zero-documents warning stay. FR-1.4 extends it so that images seen and documents
written are separately visible, making a future recurrence diagnosable from logs
alone rather than requiring a WZ walk.

**Testing** — the WZ-derived facts in this PRD (Evan images carry skills;
`Dragon/` images carry none; Cygnus 4th-job `skill` nodes are present but empty)
must be pinned as fixtures or table-driven test cases, not left as prose. A
regression here is silent by nature.

## 9. Open Questions

1. **Does the identity-level parent ever vary by version?** If Gm's parent is
   Beginner at every version and only the wire id differs, FR-3.1 collapses to a
   version-blind identity relation plus the existing identity→wire binding —
   materially less generated code. The v0.48 case supports this, but it has not
   been checked against every divergent id in `divergences.csv`. Resolve in
   design before choosing the table shape.
2. **Null-parent representation through api2go.** FR-3.3 needs "no parent" to be
   distinguishable from wire id 0 (Beginner). The marshalling behavior of a
   nullable numeric attribute in this codebase's api2go version should be
   confirmed with a test before the shape is fixed.
3. **Scope of the FR-4.2 name migration.** `jobName` and `getJobNameById` have
   consumers beyond the Jobs page (preset badges, class pickers, rankings, item
   `formatReqJob`). This task makes the Jobs page correct; whether the other
   consumers migrate now or are left on the static table needs a decision. They
   are wrong at v0.48 in exactly the same way, so leaving them is knowingly
   shipping a partial fix.
4. **Re-ingest and baseline republish (explicitly out of scope, §2).** Needs a
   tracked follow-up covering v0.84/87/92/95 and jms 185, plus baseline
   republish, plus live verification that `GET /api/data/jobs/2200/skills`
   returns Evan's skills. Until it runs, Evan shows zero skills everywhere.
5. **Stale TODO entry.** `docs/TODO.md` line ~405 proposes adding "Cygnus /
   Aran / Resistance / Legend 4th-job presets". The Cygnus half of that is
   invalid — verified: no Cygnus 4th-job skills exist at any supported version.
   The entry should be corrected as part of FR-2, but the Aran/Legend half may
   still be valid and should not be deleted wholesale.
6. **jms 185 provenance.** The jms 185 rows in `availability.csv` are carried
   from the GMS patch log with an explicit "NOT independently confirmed"
   caveat, and jms 185 returned byte-identical Cygnus/Evan results to GMS in
   this investigation. The FR-2.3 audit should note whether jms 185 is genuinely
   verified or is inheriting GMS assumptions.

## 10. Acceptance Criteria

**Ingest correctness (FR-1)**

- [ ] A numeric `Skill.wz` image with no `skill` node produces no JOB document
      (unit test).
- [ ] A numeric image with a present-but-empty `skill` node still produces a
      JOB document with an empty skill list (unit test, distinct from the above).
- [ ] A simulated `Skill.wz` tree containing both `2200.img` (with skills) and
      `Dragon/2200.img` (no `skill` node) yields a JOB document `2200` carrying
      the real skills, in both possible walk orders (unit test).
- [ ] The SKILL worker's run summary distinguishes images seen from documents
      written.

**Availability accuracy (FR-2)**

- [ ] `Set.Available` returns false for all five Cygnus 4th-job identities at
      every supported version (unit test pinning the fix, in the style of
      `identity_test.go`'s v48/v72 tests).
- [ ] Cygnus tiers 1–3 remain available at v79+ and unavailable at v72 and
      earlier — no regression from the split.
- [ ] The FR-2.3 audit of `GM`, `SuperGM`, `Pirate`, `Aran`, and `Evan` is
      written up in the task folder with per-version WZ evidence, and states an
      explicit verdict for each class — including the classes found correct.
- [ ] `go test ./...` in `libs/atlas-constants` passes, including
      `audit_validate_test.go` against the amended CSV.
- [ ] `docs/TODO.md`'s 4th-job preset entry is corrected with respect to Cygnus.

**API (FR-3)**

- [ ] `GET /api/data/job-availability` on gms 48.1 returns id `500` named `Gm`
      with parent `0`, and id `510` named `Super Gm` with parent `500`.
- [ ] The same endpoint on gms 72.1 returns id `500` named `Pirate` and id `900`
      named `Gm`.
- [ ] No returned parent refers to a job absent from that version's response.
- [ ] Roots are unambiguously distinguishable from wire id 0.
- [ ] `usePresetJobOptions` behavior is unchanged (the attribute is additive).

**UI (FR-4)**

- [ ] On a v0.48 tenant the Explorers rail has no Pirate entry, and the Special
      group shows Gm with Super Gm beneath it.
- [ ] On a v0.72 tenant the Cygnus Knights rail group is absent entirely.
- [ ] On a v0.79 tenant the Legends rail group is absent entirely.
- [ ] On a v0.83+ tenant every Cygnus branch ends at 3rd job.
- [ ] No file under `services/atlas-ui/src` compares a tenant major version to a
      literal in order to decide job naming, parenting, or visibility.
- [ ] A pending availability or jobs query does not redirect a valid
      `/jobs/{id}` route (task-182 D10 preserved).
- [ ] A failed query surfaces the load-error card.

**Verification gates (CLAUDE.md §Build & Verification)**

- [ ] `go test -race ./...` clean in `libs/atlas-constants` and
      `services/atlas-data`.
- [ ] `go vet ./...` clean in both modules.
- [ ] `npm run build` clean in `services/atlas-ui` (type-checks tests too;
      `npm run test` alone does not type-check).
- [ ] `tools/lint.sh --check` clean from the repo root.
- [ ] `docker buildx bake atlas-data` succeeds if any `go.mod` was touched.
- [ ] Code review run before the PR is opened.
