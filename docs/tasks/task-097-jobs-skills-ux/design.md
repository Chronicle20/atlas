# Jobs & Skills Browser UX Overhaul — Design

Task: task-097-jobs-skills-ux
PRD: `docs/tasks/task-097-jobs-skills-ux/prd.md` (v1, approved)
Scope: **frontend-only** (`services/atlas-ui`). No Go / atlas-data / API changes.
Status: Design

---

## 1. Summary

task-094 shipped `/jobs` (`JobsPage`) and `/jobs/:jobId` (`JobDetailPage`). This task
fixes eight UX defects (FR-1…FR-8) without touching any backend. The work decomposes
into four small, independently-testable pure helpers plus two page edits:

| Unit | Kind | FRs |
|---|---|---|
| `lib/jobs/job-advancement-tree.ts` (consolidated tree + version floors) | pure data + fns | FR-2, FR-8 |
| `lib/skills/beginner-skill-names.ts` (curated id→name fallback) | pure data + fn | FR-3 |
| `lib/skills/format-skill-description.ts` (markup parser) | pure fn | FR-7 |
| `JobsPage.tsx` (recursive indented tree + expand affordance) | page | FR-1, FR-2, FR-6 |
| `JobDetailPage.tsx` (fallback label, master-level label, copyable id, scroll, description render) | page | FR-3, FR-4, FR-5, FR-6, FR-7 |

The guiding principle: **push every testable decision into a pure helper** and keep the
pages as thin presentation that wires helpers to shadcn primitives. This keeps Vitest
coverage on the logic (name fallback, floor gating, markup parsing, tree shaping) and
out of brittle DOM assertions.

---

## 2. Key findings from scoping (verified, not from memory)

These drove the decisions below and are recorded so the plan phase doesn't re-litigate them.

- **Beginner skill names** — verified against `libs/atlas-constants/skill/constants.go`
  (ids 2903–2915). The 13 Beginner skills served on v83 (`/jobs/0/skills` →
  `1000…1012`) map to: 1000 Three Snails, 1001 Recovery, 1002 Nimble Feet,
  1003 Soul of Craftsman, 1004 Monster Riding, 1005 Echo of Hero, 1006 Jump Down,
  1007 Maker, 1008 Multi Pet, 1009 Bamboo, 1010 Invincible, 1011 Berserk,
  1012 Bless of Nymph. Resolves **OQ-3** — the map is sourced, not invented.
- **Scroll fix** — `MapDetailPage.tsx:44` already uses
  `flex flex-col flex-1 min-h-0 overflow-y-auto space-y-6 p-10 pb-16`. Both Jobs pages
  use the **identical** class string **minus `overflow-y-auto`**. So FR-6 is literally
  adding that one token, matching the established page-local pattern — no `app-shell.tsx`
  change (confirmed `app-shell.tsx:27-28` wraps `<Outlet/>` in `overflow-hidden`).
- **Cygnus version floor** — `reference_maplestory_version_timeline` (user-corrected
  2026-06-12) states "**v83** — Atlas baseline. Explorer Knights of Cygnus exist."
  The current `CYGNUS = 92` is therefore wrong against the sanctioned project source;
  FR-8.2 mandates correcting it to **83**.
- **Job 0 path** — `useJobSkills.ts:18` enables on `jobId >= 0`, so id 0 already
  fetches. The stale "job 0 returns 0" comment in `jobs-hierarchy.ts` is the only
  remaining trace of the old assumption; no runtime short-circuit exists (FR-3.1 is a
  verify-and-delete-comment, not a code path fix).

---

## 3. FR-2 / FR-8 — Job hierarchy: single source of truth

### Problem

Two structures encode the job hierarchy and **drift is the defect**:

- `lib/utils/job-tree.ts` `JOB_TREE` — `Record<id, {id, name, parent}>`. Has the
  **advancement edges** (parent pointers) and names, but **no version floors**.
- `lib/jobs-hierarchy.ts` `JOB_HIERARCHY` — `archetype → class → JobNode[]`. Has the
  **version floors** (`minMajorVersion`) but is **flat within a class** (no edges) and
  duplicates the job set.

FR-2.3 requires exactly one structure to own the advancement edges after this task.

### Decision: consolidate onto a tree keyed by parent edges, with per-root version floors

Adopt `JOB_TREE`'s parent-edge shape as the structural source of truth and **attach
version floors to branch-root nodes only** (inherited down the subtree). `JOB_HIERARCHY`
and its `filterHierarchy` are **removed**; their floor knowledge migrates into the tree.

Rationale:
- The advancement tree the PRD wants (Beginner → 1st → 2nd → 3rd → 4th) **is** the
  parent-edge graph. `JOB_TREE` already encodes it correctly, including the natural
  roots (`parent: null`): `0` (Beginner → all five Adventurer branches), `800`, `900`,
  `910`, `1000` (Noblesse → Cygnus), `2000` (Legend → Aran), `2001` (Evan).
- Version gating is **per-branch**, so floors belong on roots, not on every node. A
  single `BRANCH_FLOORS: Record<rootId, number>` map is the entire gating policy — small,
  obvious, unit-testable.
- One structure, one set of names, no drift. `jobs-hierarchy.ts` going away is a net
  reduction in surface area.

New module `lib/jobs/job-advancement-tree.ts`:

```ts
export interface JobNode {
  id: number;
  name: string;
  parent: number | null;
  children: number[];   // derived once from parent edges
}

// The full graph, keyed by id. Names + parent edges ported verbatim from
// the existing JOB_TREE (already sourced to libs/atlas-constants/job/constants.go).
export const JOB_GRAPH: Record<number, JobNode> = { /* … */ };

// Version-introduction floor per branch ROOT id. A node inherits its root's floor.
// Basis is documented per entry; floors are a display-curation choice (the
// atlas-data endpoint is NOT version-gated — verified probe), NOT a data gate.
export const BRANCH_FLOORS: Record<number, number> = {
  0:    83, // Adventurers — v83 baseline
  800:  83, // Maple Leaf Brigadier (special) — v83 baseline
  900:  83, // GM        — admin, always present
  910:  83, // Super GM  — admin, always present
  1000: 83, // Cygnus    — reference_maplestory_version_timeline: Cygnus exist in v83
  2000: 80, // Aran      — product owner: Aran introduced v80
  2001: 84, // Evan      — reference_maplestory_version_timeline: Evan introduced v84
};

export const JOB_ROOTS: number[];                 // ids with parent === null
export function rootOf(id: number): number;        // walk parents to the root
export function floorOf(id: number): number;       // BRANCH_FLOORS[rootOf(id)]
export function visibleRoots(major: number): number[]; // roots whose floor <= major
export function childrenOf(id: number): number[];
```

### Corrected floors (FR-8.1, FR-8.2) — and their basis

| Branch | Old | New | Basis |
|---|---|---|---|
| Adventurer (root 0) | 83 | 83 | v83 baseline (unchanged) |
| Special (800) | 83 | 83 | baseline (unchanged) |
| Admin (900/910) | 83 | 83 | always present (unchanged) |
| **Cygnus (1000)** | **92** | **83** | `reference_maplestory_version_timeline`: KoC exist in v83 |
| **Aran (2000)** | **88** | **80** | product owner (PRD FR-8.1): Aran introduced v80 |
| Evan (2001) | 84 | 84 | `reference_maplestory_version_timeline`: Evan introduced v84 |

On the v83 tenant this makes Adventurer + Cygnus + Aran + Admin visible, Evan hidden —
matching the PRD's "Aran should appear on v83" requirement and the project timeline.
The stale `ARAN=88` and "job 0 returns 0" comments are deleted (FR-8.4).

### OQ-1 resolution — keep gating, corrected

Keep curated version-gating (PRD default). Dropping it would show Evan on a v83 tenant,
which is wrong for the live game at that version even though the data is ungated. The
floor map is tiny and unit-tested; correctness, not removal, is the fix.

### Tree rendering (FR-2.1–2.4, FR-1)

`JobsPage` renders a **recursive indented tree** as a forest over `visibleRoots(major)`.
A single recursive `JobTreeNode` component:

- Indents by `depth` (e.g. `style={{ paddingLeft: depth * 16 }}` or a depth→class map),
  so advancement tier is readable without expanding (FR-2.4).
- **Leaf node** (no children): a `Link` to `/jobs/:id` with hover underline. No chevron.
- **Branch node** (has children): a shadcn `Collapsible` whose row contains:
  - a **chevron `CollapsibleTrigger`** (rotates with state, pointer cursor, hover bg,
    keyboard-focusable Enter/Space) — toggles expand only, does **not** navigate
    (FR-1.1–1.3, FR-2.2);
  - the **job name as a `Link`** to `/jobs/:id` — navigates, does not toggle.
  This dual-control row resolves the branch-node tension cleanly: every node is a real
  job with a detail page (so the name always navigates), while expansion is owned by the
  chevron. `defaultOpen` on the top tier(s) for discoverability.

Collapsed/expanded distinction (FR-1.3) comes from the rotating chevron
(`transition-transform` + `rotate-90` when open) plus child visibility.

### Alternatives considered

- **Keep `jobs-hierarchy.ts`, bolt edges on.** Rejected — leaves two structures and the
  archetype/class grouping that the flat-list defect lives in. The PRD explicitly wants
  the advancement line, which is the edge graph, not the archetype buckets.
- **Per-node `minMajorVersion`.** Rejected — floors are per-branch; per-node is 60+
  redundant fields that can disagree. Root-inherited floor is the minimal correct model.

---

## 4. FR-3 — Beginner skill name fallback

### Decision: generic blank-name fallback backed by a curated id→name map

New `lib/skills/beginner-skill-names.ts`:

```ts
// Sourced from libs/atlas-constants/skill/constants.go (Beginner* ids 1000–1012).
export const BEGINNER_SKILL_NAMES: Record<number, string> = {
  1000: "Three Snails", 1001: "Recovery", 1002: "Nimble Feet",
  1003: "Soul of Craftsman", 1004: "Monster Riding", 1005: "Echo of Hero",
  1006: "Jump Down", 1007: "Maker", 1008: "Multi Pet", 1009: "Bamboo",
  1010: "Invincible", 1011: "Berserk", 1012: "Bless of Nymph",
};

/** Resolve a display name: server name if non-blank, else curated map, else `Skill <id>`. */
export function resolveSkillName(id: number, serverName: string | undefined): string;
```

`SkillRow` (and the loading/empty logic) call `resolveSkillName(def.id, def.name)` for
both the row label and the icon `alt`. The trigger is **`name` is blank** (FR-3.3), not
`jobId === 0`, so any future blank-name skill (any job) degrades to its curated name or
`Skill <id>` — never empty text. The curated map is a hint table; absence falls through
to `Skill <id>`.

`useJobSkills` already enables id 0 (verified), and no downstream code special-cases 0,
so FR-3.1 is satisfied by deleting the stale comment — no logic change.

---

## 5. FR-7 — Skill description markup parser

### Decision: a pure tokenizer returning a structured model, rendered by the page

New `lib/skills/format-skill-description.ts`. MapleStory description markup present in
the data (verified examples in the PRD): `\n` newlines, a leading `[Master Level : N]`
header, and `#x…#` directives (`#c…#` color, `#e…#`/`#n…#` emphasis, bare `#` resets).

Parse into a structured model rather than returning a single string, so the page can
render line breaks and (optionally) color without re-parsing:

```ts
export interface DescSegment { text: string; color?: string; }
export interface FormattedDescription {
  lines: DescSegment[][];   // outer = lines (split on \n), inner = styled segments
  masterLevelHeader?: number; // parsed from leading [Master Level : N], then removed
}
export function formatSkillDescription(raw: string | undefined): FormattedDescription;
```

Directive handling (FR-7.1):
- `\n` (and literal `\r\n`) → line boundary.
- Leading `[Master Level : N]` → captured into `masterLevelHeader`, removed from body
  (**OQ-4: suppress** — master level is already shown in the row + panel; avoid the
  duplicate). Captured rather than discarded so a later change can re-surface it cheaply.
- `#c…#` → segment with a color marker; **OQ-2: strip-to-text for v1** (segment carries
  the inner text; `color` is populated but the page renders plain text in v1). Keeping
  `color` in the model means enabling colored rendering later is a page-only change, no
  parser rework.
- `#e…#`, `#n…#`, other `#x…#` → strip the markers, keep inner text.
- Bare `#` reset markers → consumed, never leaked (FR-7.1 last bullet).
- **Unknown directives degrade by stripping** the `#…#` markers and keeping inner text
  (FR-7.2) — never render raw `#`.

The page renders `lines.map(line => <p>{line.map(seg => seg.text)}</p>)` (or `<br/>`
joins), with the empty-description case still showing "No description available."

### Alternatives considered

- **Regex `.replace()` chain returning a string.** Rejected — can't express line breaks
  as JSX without re-splitting, and color rendering would need a second pass. The
  structured model is barely more code and is the FR-7.2 "small, unit-tested pure
  helper" the PRD asks for, with a clean extension path.
- **A markdown/3rd-party lib.** Rejected — YAGNI; the directive set is tiny, closed, and
  game-specific. No dependency.

---

## 6. FR-4 — Master-level label

`SkillRow`'s row indicator currently reads `Lv {maxLevel}` (`JobDetailPage.tsx:82-84`)
while the expanded panel says `Master Level: {maxLevel}` (`:91`) — two names for one
number. Decision: row shows **`Master Lv {N}`** wrapped in a shadcn `Tooltip`
("Skill's maximum (master) level"), and the expanded panel keeps `Master Level: {N}`.
Both read the same `def.maxLevel` value (FR-4.1–4.3). Single term, one tooltip, no new
component.

---

## 7. FR-5 — Copyable job id

The detail header currently uses `<Badge variant="outline">{jobId}</Badge>`
(`JobDetailPage.tsx:118`). The header also has a back-chevron `Link` and the job title.

`CopyableIdHeader` (`components/common/CopyableIdHeader.tsx`) renders its own
title-with-tooltip and an `actions` slot but **does not** include a back button, and it
makes the *title* the copy target. The Jobs detail header needs: back chevron + title +
copyable id. Decision: **reuse the `TooltipContent copyable` primitive directly** in the
existing header row rather than forcing `CopyableIdHeader` (whose layout assumes the
title is the hover target and has no back-button slot).

Replace the `Badge` with the established copyable pattern: a focusable id trigger wrapped
in `Tooltip` whose `TooltipContent` carries the `copyable` prop (the same convention
Monster/Map/Account headers use — `TooltipContent copyable` is already in the codebase).
Hover/focus reveals + click-to-copy, keyboard operable (FR-5.1–5.2), no bespoke copy UI.

This keeps the back-chevron + title layout intact while matching the domain-wide copy
interaction. (If the shared `copyable` tooltip content is not trivially reusable in this
layout, the plan phase may extract a tiny `CopyableId` inline component — but the default
is to reuse the existing `Tooltip`/`TooltipContent copyable` primitives, not invent UI.)

---

## 8. FR-6 — In-page scrolling

Root cause confirmed: `app-shell.tsx:27-28` wraps `<Outlet/>` in `overflow-hidden`; the
Jobs pages provide no internal scroll region. Both pages' outer div is
`flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16` — identical to `MapDetailPage:44`
except that page adds `overflow-y-auto`.

Decision: add **`overflow-y-auto`** to the outer div of `JobsPage.tsx` and
`JobDetailPage.tsx`. Page-local, one token each, matches the existing working pattern,
**no `app-shell.tsx` change** (FR-6.2). The `LevelTable`'s `overflow-auto` wrapper
(`JobDetailPage.tsx:47`) is unaffected — horizontal table scroll and vertical page
scroll coexist because they're separate containers (FR-6.3).

---

## 9. Testing strategy (NFR §8)

Vitest + Testing Library. Gate: `npm run build` clean + full `npm run test` green + no
**new** lint errors (lint baseline is pre-existing-broken per
`reference_atlas_ui_npm_nvm_and_lint_baseline`; `npm` needs nvm 22 first). Note
`npm run build` type-checks `*.test.ts` too — update test call sites in the same commit
when a signature changes.

Pure-helper unit tests (the bulk of coverage):
- `job-advancement-tree`: `floorOf`/`rootOf` for a sampled node per branch;
  `visibleRoots(83)` includes Cygnus + Aran, excludes Evan; `visibleRoots(84)` includes
  Evan; `childrenOf(0)` returns the five Adventurer 1st-jobs; no orphan ids; corrected
  floor values asserted explicitly.
- `beginner-skill-names`: blank server name → curated; non-blank → server; unknown blank
  id → `Skill <id>`; whitespace-only treated as blank.
- `format-skill-description`: `\n` → lines; leading `[Master Level : N]` captured +
  suppressed; `#cText#` → `Text` (+ color set); unknown `#x…#` stripped; bare `#`
  consumed; `undefined`/`""` → empty model.

Page-level tests (behavioural, minimal DOM):
- `JobsPage`: branch node renders a chevron trigger + a name `Link`; leaf renders a
  `Link` only; tier indentation present; Evan branch absent on v83, present on v84.
- `JobDetailPage`: `/jobs/0` renders 13 rows with Beginner fallback labels (no blank
  text); row shows `Master Lv`; id is copyable (tooltip/`copyable` present); outer
  container has `overflow-y-auto`; a description with `\n`/`#c…#` renders broken lines
  with no raw `#`.

The existing `jobs-hierarchy.test.ts` is **replaced** by `job-advancement-tree.test.ts`
(the module it tested is removed). `job-tree.test.ts` is folded in or updated since
`JOB_TREE` is superseded by `JOB_GRAPH`.

---

## 10. File-by-file impact

New:
- `services/atlas-ui/src/lib/jobs/job-advancement-tree.ts` + `__tests__/`
- `services/atlas-ui/src/lib/skills/beginner-skill-names.ts` + `__tests__/`
- `services/atlas-ui/src/lib/skills/format-skill-description.ts` + `__tests__/`

Modified:
- `services/atlas-ui/src/pages/JobsPage.tsx` — recursive indented tree, chevron
  affordance, `overflow-y-auto` (FR-1, FR-2, FR-6).
- `services/atlas-ui/src/pages/JobDetailPage.tsx` — name fallback, `Master Lv` label +
  tooltip, copyable id, `overflow-y-auto`, formatted description (FR-3, FR-4, FR-5,
  FR-6, FR-7).

Removed:
- `services/atlas-ui/src/lib/jobs-hierarchy.ts` + its `__tests__/jobs-hierarchy.test.ts`
  (consolidated into `job-advancement-tree`).
- `services/atlas-ui/src/lib/utils/job-tree.ts` — superseded by `JOB_GRAPH`. Check
  callers first: `jobTreePath` may be used by a breadcrumb or another page. If it has
  external callers, re-home `jobTreePath` onto `JOB_GRAPH` rather than deleting blindly
  (the plan phase greps usages before removal).

No other Atlas service, no Dockerfile / `go.work` / k8s changes (frontend-only).

---

## 11. Open questions — resolved

| OQ | Decision |
|---|---|
| OQ-1 (drop vs keep version gating) | **Keep**, with corrected floors (§3). |
| OQ-2 (`#c…#` strip vs color) | **Strip to text for v1**; parser keeps `color` in the model so colored rendering is a later page-only change (§5). |
| OQ-3 (Beginner name map source) | Verified against `libs/atlas-constants/skill/constants.go`; map in §4. |
| OQ-4 (suppress `[Master Level : N]` header) | **Suppress** (captured then removed) to avoid duplicating the master-level shown in row/panel (§5). |

## 12. Risks / notes

- **`job-tree.ts` removal** must be preceded by a usage grep — `jobTreePath` could feed a
  breadcrumb. The plan phase confirms callers before deleting; if used, port it onto
  `JOB_GRAPH`.
- **`CopyableIdHeader` fit** — its layout makes the title the copy target and has no
  back-button slot, so the design reuses the lower-level `Tooltip`/`TooltipContent
  copyable` primitives directly. If the plan finds a cleaner shared extraction, that's
  acceptable so long as it reuses the existing copy convention, not a new one.
- **Lint baseline** is pre-existing-broken; gate on *no new* errors, not a clean lint.
