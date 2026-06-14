# Jobs & Skills Browser UX Overhaul — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-14
Follows: task-094 (Jobs & Skills Browser, merged PR #767)
---

## 1. Overview

task-094 shipped a Jobs browser (`/jobs`) and a per-job skill detail page
(`/jobs/:jobId`) in atlas-ui. It works, but its first real use surfaced a set of
UX defects that make it hard to navigate, hard to read, and — in the case of the
Beginner job and version-gated branches — actively wrong about what content
exists. This task is a focused, **frontend-only** overhaul of those two pages to
fix discoverability, labeling, copyable IDs, scroll layout, the Beginner-skill
gap, the job-advancement visualization, the version-gate floors, and skill
description rendering.

No Go service, atlas-data, or API changes are in scope. Every fix is in
`services/atlas-ui`. The one genuine data limitation discovered during scoping
(Beginner skill definitions carry empty `name`/`description` in atlas-data) is
worked around on the client with a fallback label, **not** by changing the
ingest pipeline.

### Scoping investigation (live probe, 2026-06-14)

Probed live `atlas-data` in `atlas-main` against the **GMS v83** tenant
(`ec876921-c363-4cc6-9c51-5bb8d57f9553`, REGION=GMS, MAJOR_VERSION=83,
MINOR_VERSION=1):

- `GET /api/data/jobs/0/skills` → `{"skills":[1000,1001,1002,1003,1004,1005,1006,1007,1008,1009,1010,1011,1012]}`
  — **13 Beginner skills**, served on v83. (The task-094 in-code note claiming
  "job 0 returns 0, handled as empty" is **stale**; the data was re-ingested
  since.) Skill `1004` carries a `MONSTER_RIDING` statup → Monster Rider.
- `GET /api/data/skills/1000` and `/skills/1004` → **`"name":""`, `"description":""`**.
  Beginner skill definitions exist but have **empty name/description** text.
  Contrast `GET /api/data/skills/1000000` → `"name":"Improved HP Recovery"`,
  `"description":"[Master Level : 16]\nRecover additional HP…"`.
- Confirms task-094's standing finding that `/jobs/{id}/skills` is **not**
  version-gated (v83 serves Aran job 2100 → 4 skills, etc.).

Conclusions that drive requirements:
- Beginner skills **are** reachable; the page must stop treating job 0 as empty
  and must supply a **fallback label** for skills whose `name` is empty.
- The version-gate floors in `jobs-hierarchy.ts` are an unsourced UI-curation
  guess. `ARAN = 88` is wrong — **Aran was introduced in v80** (per product
  owner) — which is why Aran disappeared on sub-v88 tenants.
- Skill descriptions contain MapleStory markup (`\n`, `[Master Level : N]`,
  `#c…#` color/reference codes) that must be handled, not rendered raw.

## 2. Goals

Primary goals:
- Make the job hierarchy obviously expandable (clear affordance), and present it
  as an indented collapsible **advancement tree** rather than a flat link list.
- Render Beginner job skills correctly, including a readable fallback for skills
  whose definition `name` is empty in atlas-data.
- Disambiguate the master-level indicator and make the job id copyable, matching
  existing domain-object conventions.
- Fix in-page scrolling on both pages so expanding skills never clips content.
- Correct the version-gate floors (Aran = v80) and make the gating behaviour
  defensible and documented.
- Render skill description text with MapleStory markup handled (newlines,
  master-level header, `#…#` directives).

Non-goals:
- No changes to Go services, atlas-data ingest, or REST API shapes.
- No attempt to populate the empty Beginner skill names server-side.
- No change to which skills a job grants (server-owned).
- No new authentication, routing framework, or global layout rewrite.
- No redesign of unrelated pages (the scroll fix is page-local; see FR-5).

## 3. User Stories

- As an operator browsing jobs, I want each hierarchy node to visibly look
  expandable so I don't have to click random text to discover structure.
- As an operator, I want to see the job advancement line (Beginner → 1st → 2nd →
  3rd → 4th) so the progression is legible, not a flat wrapped list.
- As an operator viewing the Beginner job, I want to see its skills (Recovery,
  Nimble Feet, Three Snails, Monster Rider, …) with a readable label even when
  the underlying data has no name.
- As an operator, I want the "Lv" indicator to clearly mean Master Level, the
  same wherever it appears.
- As an operator, I want to copy a job's id from the detail page the way I copy
  other domain-object ids.
- As an operator with many skills on a 4th-job character, I want the page to
  scroll so I can read every expanded skill and its level table.
- As an operator on any tenant version, I want jobs that existed at that version
  (including Aran on v83) to appear, and jobs that didn't to be hidden for the
  right reasons.
- As an operator reading a skill description, I want clean formatted text, not
  raw `\n` and `#c…#` control codes.

## 4. Functional Requirements

Numbered to map onto the original six issues plus the two discovered during
scoping (FR-7 description formatting, FR-8 version floors).

### FR-1 — Hierarchy expand affordance (`JobsPage.tsx`)
- FR-1.1 Every collapsible node (archetype and class level) MUST show an explicit
  expand/collapse affordance: a chevron icon that rotates with state, plus a
  pointer cursor and a hover background, so the node reads as interactive.
- FR-1.2 The affordance MUST be keyboard-focusable and toggle on Enter/Space
  (shadcn `Collapsible` trigger already supports this; styling must not remove it).
- FR-1.3 Collapsed vs expanded state MUST be visually distinguishable at a glance.

### FR-2 — Advancement-tree layout (`JobsPage.tsx`)
- FR-2.1 Replace the flat `archetype → class → flex-wrap list of job links` with
  an **indented collapsible tree** that expresses the advancement line
  (Beginner/branch-leader → 1st job → 2nd → 3rd → 4th), using parent→child
  relationships.
- FR-2.2 Each job node MUST remain a navigation target to `/jobs/:jobId` (a
  click on a leaf/job node navigates to its detail page; expanding/collapsing a
  branch node does not navigate).
- FR-2.3 The tree SHOULD reuse the existing `lib/utils/job-tree.ts` `JOB_TREE`
  (which already encodes `parent` for every job) rather than the flatter
  `jobs-hierarchy.ts` shape, OR `jobs-hierarchy.ts` is reshaped to carry the
  advancement edges. Design phase picks the single source of truth; the two
  structures must not drift.
- FR-2.4 Indentation depth MUST reflect advancement tier so the progression is
  readable without expanding everything.

### FR-3 — Beginner skills render (`JobDetailPage.tsx`, hierarchy, hook)
- FR-3.1 The Beginner job (id 0) MUST be navigable and MUST render its skills;
  no code path may treat job 0 as inherently empty. (`useJobSkills` already
  enables `jobId >= 0`; verify nothing downstream short-circuits id 0.)
- FR-3.2 For any skill whose definition `name` is empty/blank, the row MUST show
  a **fallback label**. The fallback is a curated client-side name map for known
  Beginner skill ids (1000–1012: Three Snails, Recovery, Nimble Feet, Legendary
  Spirit / Monster Rider, etc. — exact ids verified during design against
  `libs/atlas-constants` and/or the live skill data), falling back to the skill
  id (e.g. `Skill 1000`) when not in the map.
- FR-3.3 The empty-name fallback MUST be generic (driven by "name is blank"), not
  special-cased to job 0, so any future blank-name skill also degrades cleanly.

### FR-4 — Master-level label clarity (`JobDetailPage.tsx`)
- FR-4.1 The per-row level indicator currently labeled `Lv {maxLevel}` MUST be
  relabeled to read unambiguously as **Master Level** (e.g. `Master Lv N`),
  consistent with the expanded detail which already says "Master Level".
- FR-4.2 The label SHOULD carry a tooltip clarifying it is the skill's maximum
  (master) level.
- FR-4.3 The same value/term MUST be used in both the row and the expanded panel
  — no two names for one number.

### FR-5 — Copyable job id (`JobDetailPage.tsx`)
- FR-5.1 The job id, currently a plain `<Badge variant="outline">{jobId}</Badge>`
  in the page header, MUST become a **copyable id** using the established
  pattern (`components/common/CopyableIdHeader.tsx` / the tooltip-with-`copyable`
  convention used by Monster/Map headers), so clicking copies the id to the
  clipboard.
- FR-5.2 The interaction MUST match other domain objects (hover/focus reveals the
  id; copy on click), not introduce a new bespoke copy UI.

### FR-6 — In-page scrolling (page-local) (`JobDetailPage.tsx`, `JobsPage.tsx`)
- FR-6.1 When the skill list plus an expanded level table exceeds the viewport,
  the page content MUST scroll vertically. Root cause: `AppShell` wraps the
  outlet in `overflow-hidden` and the page provides no internal
  `overflow-y-auto` scroll region.
- FR-6.2 The fix MUST be **page-local** — add an internal scroll container to
  these two pages. It MUST NOT modify `app-shell.tsx` or otherwise risk
  regressing scroll behaviour on the other ~44 routes.
- FR-6.3 The level table's existing horizontal `overflow-auto` MUST continue to
  work; vertical page scroll and horizontal table scroll coexist.

### FR-7 — Skill description formatting (`JobDetailPage.tsx`)
- FR-7.1 Skill description text MUST be rendered with MapleStory markup handled,
  not dumped raw. Minimum handling:
  - `\n` → line break.
  - Leading `[Master Level : N]` header rendered as a header/secondary line (or
    suppressed, since master level is already shown — design decides).
  - `#c…#` and other `#x…#` directives: at minimum **strip** the control markers
    so the human-readable text remains (e.g. `#cAt least Level 3 on Sacrifice#`
    → `At least Level 3 on Sacrifice`); optionally render `#c…#` as colored text.
  - `#` reset markers handled so no stray `#` leaks into output.
- FR-7.2 The parser MUST be a small, unit-tested pure helper (e.g.
  `lib/skills/format-skill-description.ts`) so the directive set is testable and
  extensible. Unknown directives degrade by stripping the markers, never by
  showing them.
- FR-7.3 Applies to both the per-skill description and any reference text that
  carries the same markup.

### FR-8 — Version-gate floor correction & policy (`jobs-hierarchy.ts`)
- FR-8.1 The `minMajorVersion` floors are currently unsourced guesses. `ARAN`
  MUST be corrected to **80** (Aran introduced in v80), which restores Aran on
  v83+ tenants.
- FR-8.2 The remaining floors (CYGNUS, EVAN, ADV, ADMIN) MUST be re-verified
  during design against `reference_maplestory_version_timeline` and corrected if
  wrong; any floor that cannot be sourced MUST be documented as best-effort with
  its basis.
- FR-8.3 Decision to settle in design (Open Question OQ-1): keep curated floors
  (corrected) vs. drop version-gating entirely and show all jobs (since the data
  is not version-gated). Default recommendation: **keep curation with corrected
  floors** (the filter is already unit-tested and prevents showing jobs that did
  not exist at the tenant's version), and document the floors' basis in-code.
- FR-8.4 Whatever policy is chosen, the in-code comment block MUST be updated so
  it no longer asserts the stale `ARAN=88` rationale or the stale "job 0 returns
  0" claim.

## 5. API Surface

No new or modified endpoints. Existing reads consumed unchanged:
- `GET /api/data/jobs/{jobId}/skills` → `{ data: { attributes: { skills: number[] } } }`
- `GET /api/data/skills/{skillId}` → skill definition (`name`, `description`,
  `maxLevel`, `effects[]`, `element`, `action`). Note: `name`/`description` may
  be empty strings for some ids (Beginner skills) — clients MUST tolerate this.

## 6. Data Model

No backend data model changes. Client-side data structures only:
- A curated **Beginner skill name map** (skill id → display name) for FR-3.2,
  colocated with the skill helpers. Verified against `libs/atlas-constants`
  and/or live data during design; never invented from memory.
- Possible consolidation of the job hierarchy/tree source of truth (FR-2.3):
  either adopt `lib/utils/job-tree.ts`'s `JOB_TREE` (already has `parent` edges)
  for the page, or extend `jobs-hierarchy.ts`. Exactly one structure should own
  the advancement edges after this task.

## 7. Service Impact

- **atlas-ui** (only): `pages/JobsPage.tsx`, `pages/JobDetailPage.tsx`,
  `lib/jobs-hierarchy.ts`, `lib/utils/job-tree.ts`, `lib/hooks/api/useJobSkills.ts`
  (verify id-0 path), new `lib/skills/format-skill-description.ts`, a curated
  beginner-skill-name map, and reuse of `components/common/CopyableIdHeader.tsx`
  / tooltip primitives. Colocated `__tests__` updated/added.
- No other Atlas service is touched. No Dockerfile/`go.work`/k8s changes.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all data reads continue to flow through the tenant-aware
  `api` client (four tenant headers); no tenant assumptions hardcoded. Version
  curation keys off `activeTenant.attributes.majorVersion` only.
- **Performance:** no new network calls introduced for the tree (it's static
  client data); skill definition fetches remain the existing parallel
  `useJobSkillDefinitions`. The description formatter is pure/synchronous.
- **Accessibility:** expand affordances and copyable id remain keyboard- and
  screen-reader-operable (focus ring, Enter/Space toggle, accessible copy
  control).
- **Testing:** Vitest + Testing Library. New pure helpers (description
  formatter, beginner-name fallback, tree shaping, corrected floors) are
  unit-tested; page behaviours (expand affordance present, Beginner renders with
  fallback label, scroll container present, copyable id, formatted description)
  covered at the page level. Gate on `npm run build` + full `npm run test` green
  + no new lint errors (lint baseline is pre-existing-broken; see
  `reference_atlas_ui_npm_nvm_and_lint_baseline`).
- **Conventions:** follow `services/atlas-ui/CLAUDE.md` and
  frontend-dev-guidelines (named page exports, `@/` alias, no `next/*`, React
  Query for server state, `import.meta.env.VITE_*`).

## 9. Open Questions

- **OQ-1 (FR-8.3):** Keep corrected version-gate floors, or drop version-gating
  entirely and show every job (data isn't gated)? Default: keep + correct.
- **OQ-2 (FR-7.1):** For `#c…#` color directives — strip-only (simplest) or
  render actual colors? Default: strip markers for v1, leave colored rendering as
  a stretch if cheap.
- **OQ-3 (FR-3.2):** Source the Beginner skill-name map from a verified list —
  confirm the exact id→name mapping for 1000–1012 against `libs/atlas-constants`
  / live data during design (do not cite from memory per repo policy).
- **OQ-4 (FR-7.1):** Suppress the `[Master Level : N]` description header (since
  master level is shown separately) or keep it? Default: suppress to avoid
  duplication.

## 10. Acceptance Criteria

- [ ] Archetype and class nodes in `/jobs` show a rotating chevron, pointer
      cursor, and hover state; collapsed/expanded states are visually distinct;
      keyboard toggle works.
- [ ] `/jobs` presents an indented advancement tree (branch leader → 1st → 2nd →
      3rd → 4th) with correct indentation by tier; job nodes navigate to
      `/jobs/:jobId`; a single hierarchy/tree structure owns the edges.
- [ ] Navigating to `/jobs/0` shows the 13 Beginner skills; skills with empty
      `name` show a curated fallback label (verified ids) or `Skill <id>`, never
      blank text; the fallback is driven by blank-name, not hardcoded to job 0.
- [ ] The row master-level indicator reads as Master Level (e.g. `Master Lv N`),
      matches the expanded panel term, and has a clarifying tooltip.
- [ ] The job id in the detail header is copyable via the existing
      CopyableId/tooltip pattern (click copies; keyboard/hover reveals).
- [ ] On a job with many skills, expanding several (incl. level tables) scrolls
      the page vertically; the fix is page-local (no `app-shell.tsx` change);
      level tables still scroll horizontally.
- [ ] Skill descriptions render with `\n` as line breaks and `#…#` directives
      stripped/handled (no raw `#c…#` or stray `#`); handled by a unit-tested
      pure helper; unknown directives degrade by stripping.
- [ ] `ARAN` floor is `80`; Aran appears on the v83 tenant; other floors
      re-verified and their basis documented; stale `ARAN=88` / "job 0 returns 0"
      comments removed.
- [ ] `npm run build` clean, full `npm run test` green, no new lint errors.
