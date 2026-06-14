# Jobs & Skills Browser — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-13
---

## 1. Overview

Atlas operators and content authors currently have no way to browse the **jobs and
skills** available within a tenant's MapleStory region/version. Skill data is rich
and fully ingested into `atlas-data` (names, descriptions, elements, max levels, and
~60 stat fields per level), and a job→skill mapping endpoint already exists, but
none of it is surfaced in the web UI. To answer "what does Bow Master's Hurricane do
at level 30?" today you have to hit the REST API by hand or read WZ files.

This feature adds a **read-only Jobs & Skills browser** to `atlas-ui`. A user picks a
job from a navigable hierarchy (archetype → class → job tier), then sees every skill
that job grants. Each skill shows its icon, title, master level, derived type
(passive / active / buff), description, and a per-level bonus table. The page is
tenant- and version-aware: switching the active tenant re-scopes the view, and only
jobs that actually exist in that tenant's MapleStory version are shown.

The job *hierarchy structure* (the named archetype/class/tier tree) is owned by the
frontend as a static definition derived from job IDs (extending the existing
`src/lib/jobs.ts` name map). All *skill data* and the *job→skill mapping* come from
the existing `atlas-data` REST endpoints. No new backend endpoints are required.

## 2. Goals

Primary goals:

- Add a **Jobs browser** page to `atlas-ui` that renders the job hierarchy
  (archetype → class → job tier) for the active tenant.
- Filter the hierarchy to **only jobs that exist in the tenant's MapleStory version**
  (e.g. hide Cygnus Knights, Dual Blade, and Legends on a v83 tenant).
- Provide a **Job detail view** listing all skills the job grants, each with icon,
  title, master level, derived type, and description.
- Provide a **Skill detail view** (or expandable section) showing the full per-level
  bonus breakdown as an auto-generated table that only includes stat columns with at
  least one non-zero value across levels.
- Reuse existing UI patterns: tenant context/headers, React Query hooks, the
  list+detail page convention (`ItemsPage`/`ItemDetailPage`), `getAssetIconUrl` for
  icons.

Non-goals:

- No editing of jobs or skills (read-only).
- No new `atlas-data` REST endpoints, no WZ ingest changes, no new backend job-list
  or job-hierarchy endpoint. The hierarchy is a frontend static definition.
- No backend skill "type" classification — type is derived client-side.
- No character skill assignment, skill-point allocation, or skill-build simulation.
- No skill-book / mastery-book unlock mechanics, prerequisites, or job-advancement
  requirements.
- No skill animation playback.

## 3. User Stories

- As an operator, I want to browse all jobs available in my tenant's version grouped
  by archetype and class, so I can understand the job landscape without reading WZ.
- As a content author, I want to open a job and see every skill it grants with icons
  and titles, so I can reference skills when building presets, quests, or shops.
- As a content author, I want to open a skill and see its description, master level,
  type, and exact per-level bonuses, so I can verify game values against intent.
- As an operator running a v83 tenant, I do **not** want to see Cygnus/Legend/Dual
  Blade jobs that don't exist in my version, so the view reflects reality.
- As a user, when I switch the active tenant, I want the jobs/skills view to re-scope
  to that tenant's version automatically, so I'm never looking at stale data.

## 4. Functional Requirements

### 4.1 Job hierarchy (frontend static tree)

- FR-1.1 The UI MUST define a static job hierarchy with three navigable levels:
  - **Archetype**: Adventurer, Cygnus (Knights of Cygnus), Legend (Aran/Evan/etc.),
    Admin/GM. (Archetype membership is derived from job-ID ranges.)
  - **Class** (branch): e.g. under Adventurer → Warrior (Swordman), Magician, Bowman
    (Archer), Thief (Rogue), Pirate, and Dual Blade where applicable.
  - **Job tier**: the specific advancement within a class, e.g. Archer (1st) →
    Hunter / Crossbowman (2nd) → Ranger / Sniper (3rd) → Bow Master / Marksman (4th).
- FR-1.2 The hierarchy MUST be built from job IDs and human-readable names. It SHOULD
  extend/reuse the existing `src/lib/jobs.ts` `jobNameMap` rather than duplicating
  names. Where the existing map lacks archetype/class grouping, the new static
  structure supplies it.
- FR-1.3 Each leaf node MUST carry its numeric **job ID** so the detail view can query
  the job→skills endpoint.
- FR-1.4 The static definition is version-agnostic (it describes the universe of
  jobs). Version filtering (FR-2) decides which nodes are visible.

### 4.2 Version filtering

- FR-2.1 The visible hierarchy MUST be filtered to the **active tenant's
  `majorVersion`** so that jobs introduced in later versions are hidden on older
  tenants. (Reference: v83 baseline; Evan ≈ v84, Dual Blade/Blade Lord ≈ v88, Big
  Bang ≈ v93 — see project memory `reference_maplestory_version_timeline`.)
- FR-2.2 The filter MUST be derivable without a new backend endpoint. Two acceptable
  mechanisms (design phase to choose; FR-2.2a preferred):
  - FR-2.2a **Static version map** — each job node in the static tree declares the
    minimum `majorVersion` in which it exists; the UI hides nodes whose minimum
    exceeds the tenant's version. Deterministic, no extra network calls.
  - FR-2.2b **Data probe** — treat a job as "exists" iff
    `GET /api/data/jobs/{jobId}/skills` returns a non-empty skill list for the tenant.
    More accurate to actual ingested data but adds N requests; only acceptable if
    batched/cached and gated behind 4.5 loading states.
- FR-2.3 A class or archetype node with **no** visible job-tier descendants for the
  current version MUST be hidden (don't render empty branches).
- FR-2.4 Switching the active tenant MUST re-evaluate the filter (tenant change
  already clears React Query caches per the tenant contract).

### 4.3 Job detail — skill list

- FR-3.1 Selecting a job MUST fetch its skills via
  `GET /api/data/jobs/{jobId}/skills` (existing `jobsService.getSkillsByJobId`),
  which returns an array of skill IDs.
- FR-3.2 For each returned skill ID, the UI MUST fetch the skill definition via
  `GET /api/data/skills/{id}` (existing `skillsService.getSkillById`).
- FR-3.3 The skill list MUST display, per skill:
  - **Icon** via `getAssetIconUrl(tenantId, region, major, minor, 'skill', skillId)`
    with a graceful fallback when the icon 404s.
  - **Title** (skill `name`).
  - **Master level** (skill `maxLevel`).
  - **Type** — derived client-side (see FR-3.5).
  - A short **description** snippet (full description on the skill detail view).
- FR-3.4 Skills SHOULD be presented in a stable order (e.g. ascending skill ID) and
  the list MUST handle an empty result (job exists but grants no skills) gracefully.
- FR-3.5 **Skill type is derived on the frontend** from existing fields, since
  `atlas-data` exposes no explicit type. The derivation MUST be a single documented
  helper. Suggested heuristic (design phase to finalize):
  - If the skill's effects carry `statups` and/or a positive `duration`/`overTime`,
    classify as **Buff**.
  - Else if `action === true` (has an attack/cast animation), classify as **Active**.
  - Else classify as **Passive**.
  - The helper MUST degrade safely (default to "Active" or "—" rather than throwing)
    when fields are missing on older `atlas-data` responses.

### 4.4 Skill detail — per-level bonuses

- FR-4.1 The skill detail view (route or expandable panel) MUST show:
  - Icon, full title, master level, derived type, element, and full description.
  - A **per-level bonus table** built from the skill's `effects[]` array.
- FR-4.2 The table MUST render one **row per level** (`effects[i]` → level `i+1`).
- FR-4.3 The table MUST render one **column per stat field that is non-zero/non-empty
  for at least one level**. Columns whose values are zero/empty/absent across every
  level MUST be omitted ("auto table, non-zero only").
- FR-4.4 Field → column-header mapping MUST use human-readable labels (e.g.
  `MPConsume` → "MP Cost", `weaponAttack` → "Weapon Atk", `cooldown` → "Cooldown
  (ms)", `duration` → "Duration (ms)"). A field-label map MUST cover the common
  fields; uncovered fields MAY fall back to the raw JSON key.
- FR-4.5 `statups` (array of `{type, amount}`) MUST be rendered legibly per level
  (e.g. one column per distinct statup `type`, value = `amount`), since these carry
  the primary buff magnitudes.
- FR-4.6 The view MUST handle skills whose `effects` is empty or whose
  `description`/`maxLevel` is missing (older `atlas-data` responses return
  `description: ""`), showing a clear empty/unknown state rather than erroring.

### 4.5 Navigation, loading, and errors

- FR-5.1 The feature MUST add route(s) under `atlas-ui` App.tsx and a sidebar entry
  (grouped near other content browsers — Items, Monsters, Maps).
- FR-5.2 Suggested routes: `/jobs` (hierarchy browser), `/jobs/:jobId` (job detail +
  skill list), and a skill detail surface — either `/jobs/:jobId/skills/:skillId` or
  an in-page expandable panel (design phase to choose).
- FR-5.3 All data fetches MUST go through React Query hooks under
  `src/lib/hooks/api/` keyed by the active tenant ID (per the tenant contract), with
  loading skeletons and error states consistent with existing browser pages.
- FR-5.4 When no tenant is active, the page MUST show the standard "select a tenant"
  empty state rather than firing requests.

### 4.6 Service-layer extension

- FR-6.1 `skillsService.getSkillById` currently does **not** map `maxLevel` from the
  REST response (it maps name/description/action/element/animationTime/effects). This
  feature MUST extend `SkillDefinition` and the mapper to include `maxLevel` (REST
  field `maxLevel`, `uint8`).
- FR-6.2 The `SkillEffect` TypeScript interface (currently a partial subset) MUST be
  extended to cover the additional `effect.RestModel` fields the per-level table
  surfaces (e.g. `weaponAttack`, `magicAttack`, `cooldown`, `damage`, `attackCount`,
  `prop`, `mobCount`, `x`, `y`, `MPConsume`, `HPConsume`, etc.) so the auto-table can
  enumerate them type-safely.

## 5. API Surface

**No new or modified backend endpoints.** The feature consumes existing `atlas-data`
JSON:API endpoints (tenant headers `TENANT_ID`, `REGION`, `MAJOR_VERSION`,
`MINOR_VERSION` injected by the API client):

| Method & path | Existing client method | Response (attributes) |
|---|---|---|
| `GET /api/data/jobs/{jobId}/skills` | `jobsService.getSkillsByJobId(jobId)` | `{ skills: number[] }` |
| `GET /api/data/skills/{skillId}` | `skillsService.getSkillById(id)` | `{ name, description, action, element, animationTime, maxLevel, effects[] }` |

Skill icons are served as static assets, not JSON:API:

- `GET {VITE_ASSET_BASE_URL|/api/assets}/{tenantId}/{region}/{major}.{minor}/skill/{skillId}/icon.png`
  via `getAssetIconUrl(..., 'skill', skillId)`.

`skill.RestModel` (atlas-data, `services/atlas-data/.../skill/rest.go`):
`name`, `description`, `action (bool)`, `element (string)`, `animationTime (uint32)`,
`maxLevel (uint8)`, `effects ([]effect.RestModel)`.

`effect.RestModel` carries ~60 per-level fields (see
`services/atlas-data/.../skill/effect/rest.go`) including `weaponAttack`,
`magicAttack`, `weaponDefense`, `magicDefense`, `accuracy`, `avoidability`, `speed`,
`jump`, `hp`, `mp`, `hpR`, `mpR`, `MPConsume`, `HPConsume`, `duration`, `overTime`,
`cooldown`, `damage`, `attackCount`, `mobCount`, `prop`, `x`, `y`, `morphId`,
`fixDamage`, `bulletCount`, `bulletConsume`, `statups ([]{type,amount})`,
`monsterStatus (map[string]uint32)`, `cardStats`, etc.

## 6. Data Model

No persistent data model changes. All entities are read from `atlas-data` at request
time. The only "new data" is a **frontend static job-hierarchy definition**, e.g.:

```ts
// src/lib/jobs-hierarchy.ts (illustrative — finalized in design)
interface JobNode {
  jobId: number;           // numeric job ID; key for /api/data/jobs/{id}/skills
  name: string;            // reuse jobNameMap where possible
  minMajorVersion: number; // FR-2.2a version gate
}
interface ClassNode { name: string; jobs: JobNode[]; }
interface ArchetypeNode { name: 'Adventurer'|'Cygnus'|'Legend'|'Admin'; classes: ClassNode[]; }
```

Tenant scoping is implicit via the four tenant headers; React Query keys MUST include
`activeTenant.id` so cached job/skill data never leaks across tenants.

## 7. Service Impact

- **`atlas-ui`** (primary, all changes):
  - New static hierarchy definition (e.g. `src/lib/jobs-hierarchy.ts`), reusing
    `src/lib/jobs.ts` names.
  - New pages: `JobsPage` (hierarchy), `JobDetailPage` (skill list); skill detail as a
    page or panel.
  - New React Query hooks under `src/lib/hooks/api/` (e.g. `useJobSkills` already
    exists; add `useSkillDefinition`, and a batch hook for a job's skill set).
  - Extend `skillsService`/`SkillDefinition` to map `maxLevel` and broaden
    `SkillEffect` (FR-6).
  - New `deriveSkillType` helper + field-label map for the auto-table.
  - Routes in `App.tsx`; sidebar entry in `app-sidebar.tsx`.
- **`atlas-data`**: **no changes.** (Endpoints already exist.)
- **Asset service**: no changes; `'skill'` category already supported by
  `getAssetIconUrl`.

## 8. Non-Functional Requirements

- **Multi-tenancy**: every fetch is tenant-scoped via headers; every React Query key
  includes `activeTenant.id`; tenant switch clears caches (existing contract). No
  cross-tenant leakage.
- **Performance**: a job can grant many skills; per-skill detail fetches MUST be
  batched/parallelized and cached (React Query `staleTime` similar to existing
  `useJobSkills` 30-min). Avoid N+1 waterfalls where a single job view triggers
  dozens of serial requests. Icons use `loading="lazy"`.
- **Resilience**: skill icon 404s and missing/empty `description`, `maxLevel`, or
  `effects` MUST degrade gracefully (placeholders / "—"), never crash the page.
- **Consistency**: follow `atlas-ui/CLAUDE.md` conventions — named page exports, `@/`
  alias, `import.meta.env.VITE_*`, shadcn/ui components, React Query as the single
  source of server state.
- **Accessibility**: icons have `alt` text; the per-level table has proper header
  semantics.
- **Testing**: Vitest unit tests for `deriveSkillType`, the non-zero-column table
  builder, and the version filter. Component tests for the empty/loading/error states.
  Gate on `npm run build` + `npm run test` + no-new-lint-errors (lint baseline is
  pre-existing-broken per project memory).

## 9. Open Questions

- **OQ-1 (version filter mechanism)**: FR-2.2a (static `minMajorVersion` per node) vs
  FR-2.2b (probe `/jobs/{id}/skills`). PRD leans 2.2a for determinism and zero extra
  network cost. Design phase to confirm, and to source accurate minimum versions
  (cross-check `reference_maplestory_version_timeline`).
- **OQ-2 (skill detail surface)**: dedicated route `/jobs/:jobId/skills/:skillId` vs
  in-page expandable panel. Affects routing and shareable URLs.
- **OQ-3 (skill type heuristic)**: confirm the passive/active/buff derivation against
  a sample of real v83 skills (e.g. a known passive like a mastery, a known buff like
  a booster, a known attack) before locking FR-3.5.
- **OQ-4 (hierarchy source of truth)**: should the static hierarchy be hand-authored
  in TS, or generated once from `libs/atlas-constants/job` to avoid drift? PRD scopes
  hand-authored TS; design may propose a generator if low-cost.
- **OQ-5 (job names without grouping)**: `jobNameMap` has ~90 IDs but no
  archetype/class grouping; confirm coverage gaps (especially admin/GM jobs) during
  design.
- **OQ-6 (statup label readability)**: statup `type` strings come straight from the
  buff-stat enum; confirm they're presentable or need a label map (FR-4.5).

## 10. Acceptance Criteria

- [ ] A "Jobs" entry appears in the `atlas-ui` sidebar and routes to a jobs browser.
- [ ] The jobs browser renders the hierarchy as Archetype → Class → Job tier for the
      active tenant.
- [ ] On a v83 tenant, jobs that don't exist in v83 (Cygnus, Dual Blade, Legends) are
      NOT shown; empty class/archetype branches are not rendered.
- [ ] Switching the active tenant re-scopes the visible hierarchy without a manual
      refresh.
- [ ] Selecting a job lists every skill it grants, each showing icon, title, master
      level, derived type, and a description snippet.
- [ ] Skill icons render via `getAssetIconUrl(..., 'skill', skillId)` and a 404 shows
      a placeholder, not a broken image or crash.
- [ ] Opening a skill shows full description, master level, type, element, and a
      per-level bonus table.
- [ ] The per-level table has one row per level and only includes columns with at
      least one non-zero/non-empty value across levels.
- [ ] `statups` magnitudes are visible per level in the table.
- [ ] `deriveSkillType` correctly labels at least one known passive, one known buff,
      and one known active skill (covered by a unit test).
- [ ] `skillsService`/`SkillDefinition` expose `maxLevel`; the broadened
      `SkillEffect` interface covers the fields the table surfaces.
- [ ] Empty/loading/error states are present for no-tenant, no-skills, and missing
      skill-data cases.
- [ ] `npm run build` and `npm run test` pass in `atlas-ui`; no new lint errors
      introduced.
