# Jobs Unified Explorer — Design

Task: task-182-jobs-unified-explorer
Status: Approved PRD → design
Inputs: [`prd.md`](prd.md), approved visual mock [`ux-mock.html`](ux-mock.html)

## 1. Overview

Replace `JobsPage` (collapsible link tree) and `JobDetailPage` (flat skill list)
with one three-pane explorer page:

```
┌─────────────┬──────────────────────────────┬────────────────────┐
│ Branches    │ Advancement (tier grid)      │ Skill Detail       │
│ rail        ├──────────────────────────────┤  header/badges     │
│ (grouped,   │ Skills (search + rows)       │  description       │
│  gated)     │                              │  level slider      │
│             │                              │  all-levels table  │
└─────────────┴──────────────────────────────┴────────────────────┘
```

Everything is frontend-only in `services/atlas-ui`. No endpoint, hook-contract,
or Go changes. The approved mock is the visual contract; this document decides
*how* the React implementation is structured, where state lives, and what
changes in the shared graph module.

## 2. Architecture

### 2.1 File layout

```
src/pages/JobsPage.tsx                        # rewritten: URL state + composition only
src/pages/JobDetailPage.tsx                   # DELETED (and its test)
src/App.tsx                                   # /jobs/:jobId → JobsPage
src/lib/jobs/job-advancement-tree.ts          # GM reparent, floors cleanup, new helpers
src/components/features/jobs/
  branch-rail.tsx                             # left pane
  advancement-flow.tsx                        # tier-aligned chip grid
  skill-list.tsx                              # search + rows + states
  skill-detail.tsx                            # detail content (pane AND sheet body)
  skill-icon.tsx                              # moved from JobDetailPage (img + Sparkles fallback)
  rail-groups.ts                              # RAIL_GROUPS constant (groups, entry ids, accent var names)
  __tests__/                                  # colocated component tests
src/hooks/use-media-query.ts                  # generalized matchMedia hook (new)
src/index.css                                 # --c-warrior … --c-special accent tokens (light + dark)
```

Naming follows the existing feature-component convention (kebab-case files
under `components/features/<domain>/`, PascalCase named exports).

### 2.2 Data flow

Unchanged from today, just re-plumbed:

```
useTenant() ─→ activeTenant.attributes.majorVersion ─→ visibility gating (pure)
useJobSkills(tenant, jobId) ─→ skill id list
useJobSkillDefinitions(tenant, ids) ─→ definitions (per-skill useQueries, 30 min stale)
buildLevelTable(def.effects) ─→ slider readout + all-levels table
```

Selecting a different job changes only the `useJobSkills` key; definition
queries for already-seen skills stay cached (NFR: no refetch of unchanged
definitions). All graph work (`JOB_GRAPH`, chains, floors) is pure computation
memoized with `useMemo`.

## 3. Key decisions

### D1 — Selection state lives in the URL (router as single source of truth)

**Chosen:** the selected job is the route param (`/jobs/:jobId`), the selected
skill is a search param (`?skill=<id>`). The page derives everything else:

- `jobId = Number(params.jobId)`; absent (`/jobs`) → default entry (first
  visible rail entry, Warrior 100 on v62+).
- Branch entry = walk `jobTreePath(jobId)` and find the rail entry whose id is
  on the path (mock's `branchEntryOf`). Beginner (0) maps to the default entry
  but stays the selected job.
- Selecting a job → `navigate(\`/jobs/${id}\`)` (push, so browser Back walks
  selection history per acceptance criteria). Selecting a skill →
  `setSearchParams({skill})` (push); clearing removes the param.
- Ephemeral state stays local component state, NOT URL: skill filter text and
  slider level. They reset on job/skill change respectively and are not
  shareable by design (PRD only requires job+skill in the URL).

**Alternative considered:** `useState` selection mirrored to the URL via
effects. Rejected: two sources of truth, back-button and deep-link handling
need reconciliation effects, and it's the pattern the tenant/query pages
already avoid (`useSearchParams` precedent in BansPage, MapsPage, etc.).

**Invalid/hidden jobId (FR-1.2):** if `jobId` is not in `JOB_GRAPH` or
`floorOf(jobId) > major`, render the default selection and normalize the URL
with `navigate("/jobs", { replace: true })` — replace, not push, so Back
doesn't bounce. Same normalization runs on tenant switch (FR-7.3): the
existing `TenantProvider` cache clear is untouched; the explorer just
re-evaluates visibility and replaces the URL if the current selection fell
below the new floor. A `?skill=` that never resolves to one of the loaded
definitions for the selected job is ignored and stripped (replace) once
definitions settle.

### D2 — Tier alignment via CSS Grid with explicit cell placement

**Chosen:** port the mock's approach 1:1 — a single `display: grid` container,
`width: max-content; margin-inline: auto` (centering, FR-4.3), inside an
`overflow-x: auto` scroller:

- A pure helper (new, in `job-advancement-tree.ts`):
  `advancementChains(entryId, major): number[][]` — every root-to-leaf path
  below the entry node, DFS in ascending child order, dropping any chain that
  contains a below-floor node (FR-4.4, matches `visibleChildrenOf` semantics).
- Rendering: ancestors (`jobTreePath(entry).slice(0,-1)`) and the entry chip
  are placed at `gridColumn: i+1`, `gridRow: 1 / span chains.length`,
  vertically self-centered ("anchor" cells). Chain node k of chain r goes to
  `gridColumn: anchorCols + 1 + k`, `gridRow: r + 1`. Implicit auto columns
  size to the widest chip per tier; non-anchor chips stretch to fill their
  column — which is exactly the "same-tier chips align and share width"
  requirement (FR-4.1), with zero measurement code.

**Alternatives considered:**
- *Nested flexbox rows* — cannot equalize column widths across rows without JS
  measurement; rejected.
- *`<table>`* — semantically wrong (these are buttons, not tabular data) and
  fights the spanning-anchor layout; rejected.
- *A graph-layout lib* — massive overkill for ≤10 tiers × ≤5 fixed paths;
  rejected (PRD: no new deps implied, mock proves grid suffices).

Tier tags (FR-4.2) come from `jobTreePath(id).length - 1`: depth 0 with
children → "Base", otherwise ordinal ("1st" … "10th"). Pure function
`tierLabel(jobId)` exported next to `advancementChains`, unit-tested (Evan
depth 10, GM line depths 1/2).

### D3 — One `SkillDetail` component rendered in two hosts (pane vs Sheet)

**Chosen:** `skill-detail.tsx` renders the full detail content (header, badges,
description, slider box, collapsible all-levels table) given
`{ def, accentVar }`. The page decides the host by viewport:

- `useMediaQuery("(min-width: 1150px)")` — a new generalized hook copying the
  `useSyncExternalStore` pattern of the existing `use-mobile.tsx` (which stays
  untouched; it hard-codes 768px for the sidebar).
- Wide: third grid column renders a Card wrapping `<SkillDetail/>` (or the
  "Select a skill to inspect it" empty state).
- Narrow: the third column is not rendered at all; a `Sheet` (existing
  `components/ui/sheet.tsx`, `side="right"`) is `open={!!selectedSkill}`, body
  is the same `<SkillDetail/>`. `onOpenChange(false)` clears `?skill=` (job
  selection untouched) — covering close button, overlay click, and Escape in
  one place (FR-6.6).

Slider reset-on-skill-change (FR-6.4) is `key={def.id}` on `SkillDetail`, so
its internal `useState(1)` level remounts fresh per skill — no effect
juggling. Skills with `maxLevel ≤ 1` or an empty `buildLevelTable` render "No
per-level data." in place of the slider box.

**Alternative considered:** CSS-only hiding (the mock's `display:none`) —
rejected: it leaves detail content mounted-but-invisible below 1150px,
violating "information always reachable" (FR-6.6), and renders two DOM copies
if combined with a Sheet.

### D4 — Branch accents as theme-level CSS custom properties

**Chosen:** add the nine `--c-*` tokens from the mock (darkened Nord variants
under `:root`, Nord originals under `.dark`) to `src/index.css` next to the
existing theme tokens. `rail-groups.ts` maps each entry id to its token name
(`{ id: 100, accent: "--c-warrior" }`). Components receive the *token name*
and set a scoped `style={{ "--acc": \`var(${accent})\` }}` on their subtree;
all component CSS uses `hsl(var(--acc) / …)` exactly like the mock
(FR-7.1: scoped custom property, no per-component hard-coding).

**Alternative considered:** Tailwind utility classes per branch
(`text-red-600` etc.) — rejected: nine branches × light/dark × ~6 usage sites
explodes into class-map lookups, and the values wouldn't be the theme's Nord
palette.

### D5 — Display graph changes stay inside `job-advancement-tree.ts`

- `JOB_GRAPH`: `900.parent: null → 0`, `910.parent: null → 900` (FR-2.1).
- `BRANCH_FLOORS`: drop the `900: 1` and `910: 1` rows (FR-2.2) — as non-roots
  they inherit Beginner's floor 1; `floorOf` needs no change (it already walks
  to the root).
- Provenance comment updated: note the intentional divergence from
  `atlas-constants/job/constants.go` for the GM line (in-game presentation).
- New pure exports: `advancementChains(entryId, major)` and `tierLabel(id)`
  (D2). Existing exports keep their contracts (FR-2.3); `JOB_ROOTS` shrinks by
  two, which the updated tests pin (`visibleRoots` excludes 900/910,
  `jobTreePath(910) === [0, 900, 910]`).
- Subtree job count for the rail badge (FR-3.2): `subtreeCount(entryId,
  major)` counting version-visible nodes (entry + visible descendants) — also
  a pure export with tests (GM entry → 2; Warrior → 10; Explorers unaffected
  by the Pirate node-floor since Pirate is its own entry).

Breadcrumbs (FR-1.5): `routes.ts` keeps both `/jobs` and `/jobs/[id]`
patterns with the existing `getJobNameById` label resolver — nothing
references the `JobDetailPage` component, so verification is a test that both
paths still resolve (no code change expected).

## 4. Component contracts

| Component | Props (essentials) | Owns locally |
|---|---|---|
| `JobsPage` | — (reads router + tenant) | derived selection, URL writes |
| `BranchRail` | `groups` (visible, with counts), `selectedEntryId`, `onSelect(id)` | nothing |
| `AdvancementFlow` | `entryId`, `major`, `selectedJobId`, `accent`, `onSelect(id)` | nothing (chains via `useMemo`) |
| `SkillList` | `defs`, `state` (loading/error/empty/defs-failed), `selectedSkillId`, `accent`, `onSelect(id)` | filter text |
| `SkillDetail` | `def`, `accent` (keyed by `def.id`) | slider level, collapsible open |
| `SkillIcon` | `def`, `name` | `failed` flag |

The page is the only component that touches the router or React Query; panes
are presentational and independently testable with plain props (isolation per
design guidelines).

All interactive elements are `<button>`s with `aria-pressed` for selection
(rail items, flow chips, skill rows) and `focus-visible` rings via existing
utility classes; the slider is a native `<input type="range">` with an
`aria-label` (FR-7.2, NFR accessibility).

## 5. States & error handling

Skill-list states replicate `JobDetailPage` today, verbatim strings (FR-5.4):

1. `skillsQuery.isLoading || (ids.length > 0 && defsLoading)` → skeleton rows.
2. `skillsQuery.isError` → "Failed to load this job's skills."
3. `ids.length === 0` → "This job grants no skills."
4. `definitions.length === 0 && defsError` → "Skill details unavailable."
5. Filter with no matches → "No skills match …" (new, per mock).

No-tenant renders the existing "Select a tenant to browse its jobs and
skills." card in place of the whole explorer (FR-3.5). Partial definition
failures (some 404) behave as today: missing defs simply don't render rows; a
`?skill=` pointing at a missing def is stripped per D1.

## 6. Testing

- **Graph unit tests** (`job-advancement-tree.test.ts`, updated + extended):
  reparented GM invariants, `advancementChains` (Warrior 3×[1st..4th], Evan
  1×10, GM `[[900,910]]` from entry 900 — plus chains from Beginner include
  the GM line), version filtering (v12 drops Pirate chains), `tierLabel`,
  `subtreeCount`.
- **Page tests** (`JobsPage.test.tsx`, rewritten; `JobDetailPage.test.tsx`
  deleted): MemoryRouter deep-link in (`/jobs/110`, `/jobs/110?skill=…`),
  selection writes URL, invalid id normalizes to `/jobs`, version gating of
  rail entries per tenant fixture (v12/v62/v83/v84), no-tenant card.
- **Component tests**: rail (groups, counts, `aria-pressed`), flow (chip
  `gridColumn`/`gridRow` assignment asserts tier alignment structurally),
  skill list (all five states, filter by name and id), detail (slider drives
  readout + highlighted row, reset on skill change via rerender with new def,
  "No per-level data." branch), sheet behavior with `matchMedia` mocked narrow
  (existing setup already stubs `matchMedia`).
- Vitest-native (`vi.*`), colocated under `__tests__/`; gates: `npm run
  test`, `npm run lint`, `npm run build`, `tools/lint.sh --check`.

## 7. Risks / notes

- **Chip `<button>` inside a grid cell with a `›` separator** (mock renders
  separator + chip in one cell): keep the separator inside the cell as
  presentational `aria-hidden` text so column sizing matches the mock.
- **`useJobSkills(tenant, 0)` for Beginner**: skill ids for job 0 exist in
  atlas-data; the hook is already id-agnostic. `enabled` guards in the
  definitions hook skip `skillId ≤ 0` only, so Beginner works unchanged.
- **Push-vs-replace history**: only job/skill *selections* push; all
  normalizations (invalid id, stale skill param, tenant switch) replace —
  keeps Back useful without trapping the user in redirect loops.
- **No new dependencies**; bundle impact is a few KB of component code.
