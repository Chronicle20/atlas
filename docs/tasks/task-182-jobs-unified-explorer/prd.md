# Jobs Unified Explorer — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-24
---

## 1. Overview

The Jobs area of atlas-ui currently spans two pages: `JobsPage` renders the job
advancement graph as a collapsible text tree of links, and `JobDetailPage`
renders a flat list of a job's skills with expandable rows. Answering a routine
operator question — "what does Power Guard do at level 15 on this tenant?" —
takes a page navigation, a row expansion, and a scan of a wide per-level table,
and comparing skills across jobs costs a full round-trip each time.

This task replaces both pages with a single unified explorer: a three-pane
layout with a branches rail, a tier-aligned advancement flow plus filterable
skill list, and a persistent skill-detail panel with a level slider. The design
was validated interactively; the approved mock is committed alongside this PRD
as [`ux-mock.html`](ux-mock.html) and is the visual reference for layout,
styling (existing Nord theme tokens only), and interactions.

The explorer also corrects the presentation of the GM line: in game, GM and
Super GM present as an advancement line from Beginner, so the display graph
adopts Beginner › GM › Super GM rather than showing GM and Super GM as
standalone roots.

## 2. Goals

Primary goals:
- One screen for browsing job family trees, a job's skills, and per-skill
  detail — no navigation round-trips.
- Tier-aligned advancement visualization: same-tier jobs line up vertically
  across a branch's paths.
- Instant "value at level N" answers via a level slider, backed by the full
  per-level table.
- Preserve tenant version-floor gating exactly as it behaves today.
- Preserve deep-linking: `/jobs/:jobId` keeps working, and selection state is
  shareable via URL.

Non-goals:
- No backend or atlas-data changes; the explorer consumes existing endpoints.
- No changes to skill-description formatting (`format-skill-description.ts`).
- No new jobs or branches beyond the current `JOB_GRAPH`.
- No redesign of any other page or of the app shell.

## 3. User Stories

- As a server operator, I want to see a class branch's full advancement paths
  at a glance so that I can orient myself without expanding tree nodes.
- As a server operator, I want to click any job in the advancement flow and see
  its skills immediately so that I can walk a class line (e.g. Warrior →
  Fighter → Crusader → Hero) without leaving the page.
- As a server operator, I want a skill's stats at a chosen level so that I can
  verify live behavior against configured data without reading a 30-row table.
- As a server operator, I want to filter a job's skills by name or id so that I
  can find one skill quickly.
- As a server operator, I want to share a link to a specific job and skill so
  that a teammate opens exactly what I'm looking at.
- As a server operator on a legacy tenant (e.g. GMS v12), I want branches that
  don't exist at that version hidden so that the page reflects the tenant.

## 4. Functional Requirements

### 4.1 Routing & URL state

- FR-1.1 `/jobs` renders the unified explorer with a default selection: the
  first visible branch (Warrior on v62+ tenants) selected, no skill selected.
- FR-1.2 `/jobs/:jobId` renders the explorer with that job selected — the rail
  highlights its branch, the flow highlights the job, and the skill list shows
  its skills. Unknown or version-hidden `jobId` falls back to FR-1.1 behavior
  (no crash, no empty screen).
- FR-1.3 Selecting a job updates the URL to `/jobs/:jobId` (react-router
  navigation, no full reload). Selecting a skill sets `?skill=<skillId>`;
  clearing selection removes it. Loading a URL with `?skill=` preselects that
  skill once definitions load, including the detail panel/sheet.
- FR-1.4 The `JobDetailPage` component and its route registration are removed;
  `App.tsx` routes both `/jobs` and `/jobs/:jobId` to the explorer page.
- FR-1.5 Breadcrumbs continue to resolve for both routes (verify against
  `lib/breadcrumbs/`; adjust the jobs entries if they referenced the removed
  detail page).

### 4.2 Display graph (`job-advancement-tree.ts`)

- FR-2.1 `JOB_GRAPH` changes: `900 (GM)` gets `parent: 0`; `910 (Super GM)`
  gets `parent: 900`. This intentionally diverges from
  `atlas-constants/job/constants.go` (where both are roots) to match in-game
  presentation; the file's provenance comment is updated to say so.
- FR-2.2 `BRANCH_FLOORS` entries for 900/910 are removed (as non-roots they now
  inherit the Beginner root floor of 1 — unchanged effective floor).
- FR-2.3 Existing exports (`childrenOf`, `rootOf`, `floorOf`, `visibleRoots`,
  `visibleChildrenOf`, `jobTreePath`) keep their contracts; tests in
  `job-advancement-tree.test.ts` are updated for the new parentage (e.g.
  `JOB_ROOTS` no longer contains 900/910; `jobTreePath(910)` is
  `[0, 900, 910]`).
- FR-2.4 A pure helper produces, for a branch entry node, its advancement
  chains (every root-to-leaf path below it, version-filtered) for the flow
  renderer; unit-tested including the Beginner›GM›Super GM line and the Evan
  10-tier chain.

### 4.3 Branches rail (left pane)

- FR-3.1 The rail lists branch entries in four labeled groups:
  Explorers (Warrior 100, Magician 200, Bowman 300, Rogue 400, Pirate 500),
  Cygnus Knights (Noblesse 1000), Legends (Legend/Aran 2000, Evan 2001),
  Special (Maple Leaf Brigadier 800, GM 900).
- FR-3.2 Each entry shows a branch accent dot, the branch name, and the count
  of jobs in its subtree (per the mock).
- FR-3.3 Entries whose version floor exceeds the active tenant's major version
  are hidden (existing `floorOf` semantics; e.g. GMS v12 shows only Warrior,
  Magician, Bowman, Rogue, GM).
- FR-3.4 Selecting an entry selects its node as the current job and clears the
  skill selection and filter.
- FR-3.5 With no active tenant, the page shows the existing "Select a tenant to
  browse its jobs and skills." card in place of the explorer (current
  behavior preserved).

### 4.4 Advancement flow (middle pane, top)

- FR-4.1 The flow renders the selected branch as a tier-aligned grid: ancestor
  chips (e.g. Beginner) and the branch root span all path rows vertically
  centered; each advancement path is a row; tier-k jobs occupy the same grid
  column so same-tier chips align and share the column's width.
- FR-4.2 Every chip is clickable and selects that job. The selected job's chip
  is filled with the branch accent; chips carry tier tags ("Base" for the
  branch root's tree root, "1st"–"4th" and beyond by depth), including on the
  GM line.
- FR-4.3 The flow is horizontally centered in its card and horizontally
  scrollable when wider than the pane (Evan's 10-tier chain, Cygnus's five
  paths).
- FR-4.4 Version-hidden jobs are excluded from paths (a path is omitted when
  any node in it is below floor — matches `visibleChildrenOf` semantics).
- FR-4.5 The GM branch renders as the single line Beginner › GM › Super GM.

### 4.5 Skill list (middle pane, bottom)

- FR-5.1 The list shows the selected job's skills using existing hooks:
  `useJobSkills(tenant, jobId)` then `useJobSkillDefinitions(tenant, ids)`.
- FR-5.2 Each row shows: the real skill icon (`iconUrl` from
  `useJobSkillDefinitions`, with the existing Sparkles fallback on load error),
  name via `resolveSkillName`, id (monospace), type badge via
  `deriveSkillType`, and master level.
- FR-5.3 A search input filters rows by case-insensitive name substring or id
  substring, client-side.
- FR-5.4 States: loading → skeleton rows; skills-fetch error → existing error
  message; zero skills → "This job grants no skills."; definitions all failed →
  "Skill details unavailable." (parity with current `JobDetailPage` states).
- FR-5.5 Clicking a row selects the skill (highlight via `aria-pressed` +
  accent tint) and populates the detail panel; on narrow viewports it opens
  the detail sheet (FR-6.6).

### 4.6 Skill detail (right pane)

- FR-6.1 Empty state (no skill selected) shows a muted "Select a skill to
  inspect it" placeholder.
- FR-6.2 Header: skill icon, name, id (with the existing copyable-tooltip
  treatment used for ids elsewhere), badges for type and master level.
- FR-6.3 Description rendered via the existing `formatSkillDescription`
  pipeline.
- FR-6.4 Level slider from 1 to `maxLevel` with a stat readout for the chosen
  level. Values come from `buildLevelTable(def.effects)`: the readout renders
  the table's columns for the selected level's row (no new derivation logic).
  Slider state persists while the same skill stays selected and resets to 1 on
  skill change. Skills with `maxLevel` ≤ 1 or an empty level table show "No
  per-level data." instead of the slider.
- FR-6.5 Below the slider, a collapsible "All levels" section (default open)
  renders the full level table with the slider's current level row
  highlighted; the table scrolls horizontally inside its own container.
- FR-6.6 Responsive behavior: at viewport width ≥ 1150px the detail pane is a
  persistent third column. Below 1150px the third column is not rendered;
  selecting a skill opens the detail content in a right-side `Sheet`
  (existing `components/ui/sheet.tsx`), dismissible by close button, overlay
  click, or Escape; dismissing clears `?skill=` but keeps the job selection.
  The information is always reachable — never silently hidden.

### 4.7 Visual & interaction conventions

- FR-7.1 Styling uses existing theme tokens and shadcn primitives only (Card,
  Badge, Input, Sheet, Skeleton, Tooltip, Collapsible, Table); branch accent
  colors are defined from the Nord palette values already used by the theme
  (per the mock: darkened variants in light mode, Nord originals in dark) and
  applied via a scoped CSS custom property, not hard-coded per component.
- FR-7.2 All interactive elements are keyboard-operable with visible focus
  (`focus-visible` ring), and toggling selection state is conveyed with
  `aria-pressed` as in the mock.
- FR-7.3 Tenant switching behaves as today: `TenantProvider` clears the query
  cache; the explorer additionally resets selections that are no longer
  visible at the new tenant's version (fall back per FR-1.2).

## 5. API Surface

No new or modified endpoints. Consumed as-is:

- `GET /jobs/{jobId}/skills` via `jobsService` / `useJobSkills` (skill id list).
- `GET /skills/{skillId}` via `skillsService` / `useJobSkillDefinitions`
  (definition + effects), icon URL via `getAssetIconUrl`.

Error cases already handled by the hooks (404 non-retry policy) are unchanged.

## 6. Data Model

No backend data model changes. Frontend display-graph change only
(FR-2.1/2.2): GM(900) and Super GM(910) become descendants of Beginner(0) in
`JOB_GRAPH`. No migration; no persisted state beyond URL params.

## 7. Service Impact

`services/atlas-ui` only:

- `src/pages/JobsPage.tsx` — rewritten as the unified explorer (page-level
  composition; panes as colocated or `components/features/jobs/` components).
- `src/pages/JobDetailPage.tsx` (+ its test) — removed; route folded into the
  explorer.
- `src/App.tsx` — `/jobs/:jobId` now renders the explorer page.
- `src/lib/jobs/job-advancement-tree.ts` (+ tests) — GM reparenting, floors
  cleanup, chain-derivation helper.
- `src/lib/breadcrumbs/` — verify/adjust jobs entries.
- New feature components under `src/components/features/jobs/` (rail, flow,
  skill list, skill detail) with tests.

No Go services, deploy files, or shared Dockerfile changes.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all skill fetches go through the existing tenant-scoped
  hooks; no direct fetches. Version gating derives from
  `activeTenant.attributes.majorVersion` only.
- **Performance:** no new network calls beyond today's pages; skill definitions
  stay cached per React Query config (30 min staleTime). Selecting between
  jobs must not refetch unchanged definitions. The flow/tree derivation is
  pure computation over a ~70-node graph — memoize with `useMemo`.
- **Type safety:** strict TS, no `any`; new helpers fully typed.
- **Testing:** Vitest + Testing Library coverage for the graph helpers
  (including GM reparenting), URL-state behavior (deep link in, selection out),
  version gating of rail/flow, skill list states, slider/readout behavior, and
  the narrow-viewport sheet. `npm run test`, `npm run lint`, `npm run build`
  clean; `tools/lint.sh --check` clean.
- **Accessibility:** keyboard operability + `aria-pressed`/labels per FR-7.2;
  slider is a native range input with an accessible label.

## 9. Open Questions

None — decisions captured during spec: keep deep links + URL-encoded selection
(FR-1.3), reparent `JOB_GRAPH` directly (FR-2.1), use real skill icons
(FR-5.2), defined narrow-viewport sheet behavior (FR-6.6), keep tier tags on
the GM line (FR-4.2).

## 10. Acceptance Criteria

- [ ] `/jobs` shows the three-pane explorer matching `ux-mock.html` layout in
      both themes, inside the unchanged app shell.
- [ ] `/jobs/110` deep-links to Fighter selected in the Warrior branch;
      `/jobs/110?skill=1101007` additionally opens Power Guard detail.
- [ ] Selecting jobs/skills updates the URL; browser back restores the prior
      selection.
- [ ] Magician branch shows Wizard (F/P) / Wizard (I/L) / Cleric vertically
      aligned in one tier column, with 3rd/4th tiers aligned likewise.
- [ ] GM branch renders Beginner › GM › Super GM as one line with tier tags;
      900/910 are no longer roots (`visibleRoots` excludes them) and
      `jobTreePath(910)` is Beginner → GM → Super GM.
- [ ] On a GMS v12 tenant, Pirate, Cygnus, and Legends entries are absent;
      on v62, Pirate appears; on v83+, all but Evan; v84+, all.
- [ ] Skill rows show real icons (Sparkles fallback on error), searchable by
      name and id; all four list states render correctly.
- [ ] Detail panel: slider changes update the stat readout and highlight the
      matching row in the open "All levels" table; skills without per-level
      data show "No per-level data."
- [ ] Below 1150px the detail opens as a dismissible Sheet and nothing is
      unreachable; dismissing clears `?skill=`.
- [ ] `JobDetailPage.tsx` deleted; no route or import references remain.
- [ ] `npm run test`, `npm run lint`, `npm run build` (in
      `services/atlas-ui`) and `tools/lint.sh --check` (repo root) all pass.
