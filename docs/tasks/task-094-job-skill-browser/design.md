# Jobs & Skills Browser — Design

Task: task-094-job-skill-browser
Status: Approved (design phase)
Created: 2026-06-13
PRD: `docs/tasks/task-094-job-skill-browser/prd.md`

---

## 1. Summary

A read-only **Jobs & Skills browser** added entirely to `atlas-ui`. No backend
changes — the `atlas-data` job→skills and skill-definition endpoints, the
service modules (`jobsService`, `skillsService`), and the React Query hooks
(`useJobSkills`, `useSkillDefinition`) already exist. This feature is
overwhelmingly **presentation + a small static data file + three pure helpers**.

### Resolved design decisions (PRD open questions)

| OQ | Decision | Rationale |
|----|----------|-----------|
| OQ-1 version filter | **Static `minMajorVersion` per job node (FR-2.2a)** | Deterministic, zero network cost, unit-testable. Actual version numbers sourced empirically during implementation (§4.2). |
| OQ-2 skill detail surface | **In-page expandable panel** within the job's skill list | No new route; keeps the skill list and its bonus table on one screen. |
| OQ-3 skill-type heuristic | `statups → Buff`, else `action → Active`, else `Passive` (§4.4) | Single documented helper, degrades safely, verified by unit tests against real v83 skill JSON. |
| OQ-4 hierarchy source | **Hand-authored TS** (`jobs-hierarchy.ts`) | The Go `libs/atlas-constants/job` constants carry skill lists but **no archetype/class grouping and no version metadata**, so a generator buys nothing. |
| OQ-5 name coverage | Reuse `jobNameMap`; hierarchy supplies grouping; admin/GM/beginner handled explicitly (§4.1) | — |
| OQ-6 statup labels | Dedicated `STATUP_LABELS` map with raw-key fallback (§4.5) | Buff-stat enum strings are not all presentable. |
| Hierarchy UI | **Single-page accordion tree** (Archetype → Class → job-tier) | Dataset is tiny and static; the whole filtered tree fits one page. |
| Skill fan-out (perf) | `useQueries` parallel fetch, shared cache keys | Avoids the N+1 serial waterfall when a job grants ~13 skills. |

### Routes

- `/jobs` — `JobsPage`: filtered accordion tree; job-tier leaves link to detail.
- `/jobs/:jobId` — `JobDetailPage`: skill list, each row expandable to a full
  skill panel (description, element, type, master level, per-level bonus table).

No `/jobs/:jobId/skills/:skillId` route (skill detail is the in-page panel).

---

## 2. Architecture

```
JobsPage (/jobs)                 JobDetailPage (/jobs/:jobId)
  │ useTenant                       │ useTenant
  │ JOB_HIERARCHY (static)          │ useJobSkills(tenant, jobId) ──► skillIds[]
  │ filterHierarchy(tree, major)    │ useJobSkillDefinitions(tenant, skillIds)
  │   → pruned tree                 │   = useQueries → SkillDefinitionWithIcon[]
  │ Accordion (Archetype>Class>     │ skill rows (sorted asc by id)
  │   job-tier Link → /jobs/:id)    │   each: icon, title, maxLevel, type badge,
  └─ no network calls               │         description snippet
                                    │   expanded panel:
                                    │     deriveSkillType(def)
                                    │     buildLevelTable(def.effects)
                                    │       → { columns, rows }  (non-zero only)
                                    └─ loading / empty / error states
```

### Module inventory (all under `services/atlas-ui/src`)

| File | Kind | Responsibility |
|------|------|----------------|
| `lib/jobs-hierarchy.ts` | **new** static data + helper | The archetype→class→job-tier tree with `minMajorVersion`; `filterHierarchy()`; pure. |
| `lib/skills/skill-type.ts` | **new** pure helper | `deriveSkillType(def): SkillType` + `SKILL_TYPE` union. |
| `lib/skills/level-table.ts` | **new** pure helper | `buildLevelTable(effects): LevelTable`; `FIELD_LABELS`, `STATUP_LABELS`. |
| `lib/hooks/api/useJobSkillDefinitions.ts` | **new** hook | Parallel `useQueries` over skillIds → `SkillDefinitionWithIcon[]`. |
| `services/api/skills.service.ts` | **edit** | Add `maxLevel` to `SkillDefinition` + resource + mapper (FR-6.1); broaden `SkillEffect` (FR-6.2). |
| `pages/JobsPage.tsx` | **new** page | Accordion hierarchy browser. |
| `pages/JobDetailPage.tsx` | **new** page | Skill list + expandable skill panels. |
| `App.tsx` | **edit** | Two lazy routes. |
| `components/app-sidebar.tsx` | **edit** | "Jobs" entry under Operations. |
| `lib/breadcrumbs/*` | **edit (if registry-driven)** | Label `/jobs` and `/jobs/:jobId` (job name). |
| `__tests__` (colocated) | **new** | Unit + component tests (§7). |

Reused as-is: `useJobSkills`, `useSkillDefinition`/`skillDefinitionKeys`,
`jobsService`, `getAssetIconUrl`, `useTenant`, shadcn `Collapsible`/`Table`/
`Card`/`Badge`.

---

## 3. Static job hierarchy (`lib/jobs-hierarchy.ts`)

```ts
import { getJobNameById } from "@/lib/jobs";

export type Archetype = "Adventurer" | "Cygnus" | "Legend" | "Admin";

export interface JobNode {
  jobId: number;          // key for /api/data/jobs/{id}/skills
  name: string;           // resolved from jobNameMap where possible
  minMajorVersion: number;// FR-2.2a version gate
}
export interface ClassNode { name: string; jobs: JobNode[]; }
export interface ArchetypeNode { name: Archetype; classes: ClassNode[]; }

export const JOB_HIERARCHY: ArchetypeNode[] = [ /* authored below */ ];

/** Prune job-tiers above the tenant version, then drop empty classes/archetypes (FR-2.3). */
export function filterHierarchy(tree: ArchetypeNode[], major: number): ArchetypeNode[];
```

**Authoring rules:**

- Names come from `getJobNameById(jobId)`; where the map lacks a usable label
  (rare admin/GM cases) the node supplies its own `name` literal. `jobs.ts` is
  *not* duplicated — the hierarchy references its ids.
- Coverage maps to the ids in `jobs.ts`:
  - **Adventurer**: Beginner(0) + Warrior(100s) / Magician(200s) / Bowman(300s)
    / Thief(400s) / Pirate(500s) tiers, plus Maple Leaf Brigadier(800).
    *No Dual Blade ids exist in `jobs.ts`; none are added (out of v83/v84 data).*
  - **Cygnus**: Noblesse(1000) + Dawn Warrior / Blaze Wizard / Wind Archer /
    Night Walker / Thunder Breaker (1100–1512).
  - **Legend**: Legend(2000) + Aran(2100–2112) + Evan(2001, 2200–2218).
  - **Admin**: GM(900), Super GM(910). (Tiny; rendered last. May be hidden
    behind a constant if operators prefer — see §9.)
- `minMajorVersion` is **the verification-sensitive field** (§4.2). The tree
  declares it per node; the implementer fills concrete values from data, not
  memory.

---

## 4. Behavior detail

### 4.1 No-tenant / empty states

`JobsPage` and `JobDetailPage` both gate on `activeTenant`. With no tenant they
render the standard "select a tenant" empty state and fire **no** requests
(FR-5.4).

### 4.2 Version filter & sourcing `minMajorVersion`

`filterHierarchy` keeps a `JobNode` iff `minMajorVersion <= tenant.majorVersion`,
then removes classes with no surviving jobs and archetypes with no surviving
classes (FR-2.3). Pure, unit-tested. Tenant switch already clears React Query
caches and re-renders with the new `majorVersion` (FR-2.4) — but note the
hierarchy itself needs no cache; it re-filters on the new `activeTenant` value.

**Sourcing the version numbers (implementation step, not guessed here):**
Per CLAUDE.md "Verification Over Memory", the concrete `minMajorVersion` values
MUST be grounded, not recalled. Procedure:

1. Anchor the v83 baseline empirically: against a local **v83** tenant, probe
   `GET /api/data/jobs/{id}/skills` for a representative job of each archetype
   (e.g. a Warrior tier, Dawn Warrior, Aran, Evan). Any archetype that returns
   data on v83 → those nodes get `minMajorVersion: 83`.
2. Cross-check the curve against project memory
   `reference_maplestory_version_timeline` (v83 baseline; **Evan ≈ 84**; Dual
   Blade ≈ 88; Big Bang ≈ 93). Evan tiers (2001/22xx) → `84`.
3. For Cygnus and Aran, if a later tenant (e.g. v92/v95) is available, confirm
   the floor by probing; otherwise set the floor to the lowest version known to
   return data and leave a comment citing the source. Do **not** hardcode a
   memory-only number without a comment naming its basis.

The mechanism is correct regardless of the exact integers; wrong integers only
mis-show/hide a branch and are a one-line fix.

### 4.3 Job detail data flow & fan-out

`useJobSkills(tenant, jobId)` → `number[]` skill ids. Feed into the **new**
`useJobSkillDefinitions(tenant, skillIds)`:

```ts
// useQueries: one query per skillId, SAME key as useSkillDefinition so cache is shared.
return useQueries({
  queries: skillIds.map((skillId) => ({
    queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
    queryFn: async (): Promise<SkillDefinitionWithIcon> => { /* getSkillById + iconUrl */ },
    enabled: !!tenant?.id && skillId > 0,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
    retry: /* skip on 404, else <3 */,
  })),
});
```

Parallel, individually cached, reuses `skillDefinitionKeys` so a skill already
viewed elsewhere is a cache hit (NFR perf, no N+1 waterfall). The page derives
`isLoading`/`isError` by aggregating the result array, and renders rows for the
defs that have resolved. Skills are sorted **ascending by id** (FR-3.4); an
empty `skillIds` shows a "this job grants no skills" empty state.

### 4.4 `deriveSkillType` (`lib/skills/skill-type.ts`)

```ts
export type SkillType = "Passive" | "Active" | "Buff";

export function deriveSkillType(def: Pick<SkillDefinition, "action" | "effects">): SkillType {
  const effects = def.effects ?? [];
  const hasStatups = effects.some(e => (e.statups?.length ?? 0) > 0);
  const sustained = effects.some(e => e.overTime === true);
  if (hasStatups || sustained) return "Buff";   // boosters, blessings, Maple Warrior, Magic Guard…
  if (def.action === true) return "Active";       // attacks, casts with animation
  return "Passive";                               // masteries, passive recovery
}
```

Degrades safely: missing/empty `effects` or `action` never throws — falls
through to `action`-then-`Passive`. **Verified by unit tests** against real v83
skill JSON (FR-3.5, acceptance): one mastery (Passive), one booster/blessing
(Buff), one attack (Active). Fixtures are captured from a v83 tenant via
`GET /api/data/skills/{id}` (per memory `reference_atlas_data_wz_inspection`),
not authored from MapleStory recall.

### 4.5 Per-level bonus table (`lib/skills/level-table.ts`)

```ts
export interface LevelColumn { key: string; label: string; }      // scalar or statup column
export interface LevelTable { columns: LevelColumn[]; rows: Array<Record<string, string>>; }

export function buildLevelTable(effects: SkillEffect[]): LevelTable;
```

Algorithm:

1. **Rows**: one per `effects[i]`, level `i + 1` (FR-4.2). First column is always
   "Level".
2. **Scalar columns**: iterate a curated `FIELD_LABELS` ordered list of numeric
   `SkillEffect` keys (the magnitude fields — see below). Include a column iff at
   least one level has a non-zero / non-null value (FR-4.3). Cell = formatted
   value, blank for zero/absent.
3. **Statup columns**: collect the union of distinct `statups[].type` across all
   levels; one column per type (FR-4.5), label via `STATUP_LABELS` (raw key
   fallback), cell = that level's `amount` or blank.
4. Header label via `FIELD_LABELS` (e.g. `MPConsume`→"MP Cost",
   `weaponAttack`→"Weapon Atk", `cooldown`→"Cooldown (ms)",
   `duration`→"Duration (ms)"); uncovered keys fall back to the raw JSON key
   (FR-4.4).

Pure, no React. The **primary unit-test target**: non-zero-column omission,
statup column derivation, level-row mapping, empty `effects` → empty table.

**Scope of `FIELD_LABELS`** (numeric magnitudes worth a column): `weaponAttack`,
`magicAttack`, `weaponDefense`, `magicDefense`, `accuracy`, `avoidability`,
`speed`, `jump`, `hp`, `mp`, `hpR`, `mpR`, `mhpr`, `mmpr`, `MHPRRate`,
`MMPRRate`, `MPConsume`, `HPConsume`, `duration`, `cooldown`, `damage`,
`attackCount`, `mobCount`, `prop`, `x`, `y`, `fixDamage`, `bulletCount`,
`bulletConsume`, `morphId`, `moneyConsume`, `itemConsume`, `itemConsumeAmount`.
Complex/structured fields (`monsterStatus`, `cardStats`, `cureAbnormalStatuses`,
`lt`/`rb`) are **out of scope for the v1 table** (YAGNI — they aren't
per-level magnitudes); booleans (`overTime`, `skill`, `repeatEffect`) are
consumed by `deriveSkillType`, not shown as columns.

### 4.6 Skill list row & expandable panel

Each row (collapsed): icon, title (`name`), master level (`maxLevel ?? "—"`),
type badge (`deriveSkillType`), description snippet (truncated). Expanding
(shadcn `Collapsible` per row) reveals: full `description` (or "No description
available" when `""` — FR-4.6), `element`, type, master level, and the
`buildLevelTable` output rendered as a shadcn `Table` with sticky header. An
empty table shows "No per-level data".

**Icon 404 handling** (FR-3.3 / NFR resilience): `<img>` with `loading="lazy"`,
`alt={name}`, and `onError` swapping to a lucide placeholder (e.g. `Sparkles`)
— same defensive pattern other pages use. Never a broken image or crash.

### 4.7 Loading / error

`JobDetailPage` shows skeleton rows while `useJobSkills` or the aggregate
`useJobSkillDefinitions` is loading; an error state (job lookup failed) mirrors
existing browser pages. Individual skill-def failures degrade to a row showing
title + "details unavailable" rather than failing the whole list.

---

## 5. Service-layer extension (`skills.service.ts`, FR-6)

- Add `maxLevel?: number` to `SkillDefinition`, `maxLevel?: number` to
  `SkillResource.attributes`, and map `maxLevel: skill.attributes.maxLevel` in
  `getSkillById`. **Optional** (not required) so older `atlas-data` responses
  and existing `SkillDefinition` mock-construction sites keep compiling; the UI
  renders `maxLevel ?? "—"` (FR-4.6).
- Broaden the `SkillEffect` interface to add the numeric keys the table
  enumerates that aren't already present: `prop`, `damage`, `attackCount`,
  `mobCount`, `x`, `y`, `fixDamage`, `bulletCount`, `bulletConsume`, `mhpr`,
  `mmpr`, `MHPRRate`, `MMPRRate`, `morphId`, `moneyConsume`, `itemConsume`,
  `itemConsumeAmount`. All optional, JSON keys matching `effect.RestModel`
  exactly (e.g. `MPConsume`, `HPConsume`, `hpR`, `mpR`).

**Blast-radius note** (memory `reference_atlas_ui_build_typechecks_tests`):
`npm run build` type-checks `*.test.ts` too, and `SkillEffect`/`SkillDefinition`
are consumed by `SkillTooltipContent`, `SkillWidget`, `useSkillData` and their
tests. Because every addition is **optional**, no existing call site breaks;
still, the implementation must run the build to confirm and fix any mock that
relied on the exact shape in the same commit.

---

## 6. Routing, sidebar, breadcrumbs

- `App.tsx`: `const JobsPage = lazy(...)`, `const JobDetailPage = lazy(...)`;
  routes `<Route path="/jobs" .../>` and `<Route path="/jobs/:jobId" .../>`
  inside the `AppShell` group, alphabetically near `/items`.
- `app-sidebar.tsx`: add `{ title: "Jobs", url: "/jobs" }` to the **Operations**
  group, positioned near Items/Monsters/Maps (content browsers).
- Breadcrumbs: if the breadcrumb registry is config-driven, add labels for
  `/jobs` ("Jobs") and `/jobs/:jobId` (resolve to `getJobNameById(jobId)` with
  the numeric id as fallback), matching how other detail pages title their crumb.

---

## 7. Testing strategy

Gate (per memory `reference_atlas_ui_npm_nvm_and_lint_baseline`): `npm run build`
+ `npm run test` green, **no new lint errors** (lint baseline is
pre-existing-broken; do not gate on clean lint). Source nvm 22 before `npm`.

**Unit (pure helpers — the core):**
- `filterHierarchy`: v83 hides Cygnus/Legend(Evan) per their `minMajorVersion`;
  a class with all jobs filtered out is removed; an archetype with all classes
  removed is removed (FR-2.3); a later version keeps them.
- `deriveSkillType`: real-fixture Passive (mastery), Buff (booster/blessing),
  Active (attack); degrade cases (no effects, no action) don't throw.
- `buildLevelTable`: omits all-zero columns; includes a column with one non-zero
  level; derives one statup column per distinct type; one row per level; empty
  effects → `{ columns: ["Level"], rows: [] }`; raw-key fallback for an
  unlabeled field.
- `STATUP_LABELS`/`FIELD_LABELS`: known key → label; unknown key → raw key.

**Component:**
- `JobsPage`: no tenant → empty state, zero requests; v83 tenant → tree renders
  Adventurer branches and does **not** render Cygnus/Legend; empty branches absent.
- `JobDetailPage`: loading skeleton; empty skill list state; aggregate error
  state; icon `onError` falls back to placeholder; type badge text; expand
  reveals the per-level table.

---

## 8. Non-functional adherence

- **Multi-tenancy**: every query key includes `tenant.id`
  (`jobSkillsKeys`, `skillDefinitionKeys`); tenant switch clears caches
  (existing contract). The static hierarchy holds no tenant data. No
  cross-tenant leakage.
- **Performance**: parallel `useQueries`, 30-min `staleTime`, shared cache keys,
  `loading="lazy"` icons. The hierarchy page makes zero network calls.
- **Resilience**: optional fields + `?? "—"` fallbacks, `onError` icon swap,
  per-skill failure isolation, empty/loading/error states throughout.
- **Consistency**: named page exports, `@/` alias, `import.meta.env.VITE_*`,
  shadcn components, React Query as sole server-state source.
- **Accessibility**: `alt` on icons, `<th>` headers on the per-level table.

---

## 9. Out of scope / YAGNI

- No editing, no new backend endpoints, no WZ/ingest changes.
- No Dual Blade nodes (ids absent from `jobs.ts`; no v83/v84 data).
- No `monsterStatus` / `cardStats` / `cureAbnormalStatuses` / `lt`/`rb`
  rendering in the per-level table v1.
- No skill detail route, no character skill assignment, no prerequisites, no
  animation playback.
- The Admin (GM) archetype is rendered (it's in the data) but is the lowest
  priority; if operators object it can later hide behind a constant — not built
  speculatively now.

---

## 10. Acceptance mapping

All PRD §10 acceptance criteria are covered: sidebar entry + route (§6);
archetype→class→tier tree (§3, §4.2); v83 hides Cygnus/DB/Legends + no empty
branches (§4.2, FR-2.3 in `filterHierarchy`); tenant switch re-scopes (§4.2,
§8); job → skill list with icon/title/maxLevel/type/snippet (§4.3, §4.6); icon
404 → placeholder (§4.6); skill panel with full detail + per-level table (§4.6);
one row per level, non-zero columns only (§4.5); statups per level (§4.5);
`deriveSkillType` unit-tested on 3 real skills (§4.4, §7); `maxLevel` +
broadened `SkillEffect` exposed (§5); empty/loading/error states (§4.1, §4.7);
build + test green (§7).
```
