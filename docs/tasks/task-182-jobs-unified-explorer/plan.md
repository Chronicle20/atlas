# Jobs Unified Explorer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `JobsPage` (collapsible link tree) and `JobDetailPage` (flat skill list) with one three-pane explorer: branches rail, tier-aligned advancement flow + filterable skill list, and a persistent skill-detail panel with a level slider.

**Architecture:** Frontend-only change in `services/atlas-ui`. Selection state lives in the URL (`/jobs/:jobId` + `?skill=`); the page is the only component touching the router/React Query; the four panes are presentational components under `src/components/features/jobs/`. Tier alignment uses CSS Grid with explicit cell placement (ported 1:1 from the approved mock `docs/tasks/task-182-jobs-unified-explorer/ux-mock.html`). The GM line is reparented under Beginner in the display graph.

**Tech Stack:** React 19, react-router-dom v7, TanStack React Query 5 (existing hooks only), shadcn/ui primitives, Tailwind 4, Vitest + Testing Library.

## Global Constraints

- **No new dependencies.** No graph-layout libs, no measurement code.
- **No backend/endpoint changes.** Consume `useJobSkills(tenant, jobId)` and `useJobSkillDefinitions(tenant, ids)` as-is.
- **Styling:** existing theme tokens + shadcn primitives only. Branch accents are new `--c-*` CSS custom properties in `src/index.css` (light: darkened Nord variants; dark: Nord originals — exact values in Task 3), applied via a scoped `--acc` custom property (`style={{ "--acc": "var(--c-warrior)" }}`), never hard-coded per component.
- **Breakpoint:** detail pane is a persistent third column at viewport width ≥ 1150px; below that it renders in a right-side `Sheet` instead (third column not rendered at all).
- **Verbatim state strings** (parity with today's `JobDetailPage`): "Failed to load this job's skills.", "This job grants no skills.", "Skill details unavailable.", "Select a tenant to browse its jobs and skills.", "No per-level data.", "Select a skill to inspect it", and the new filter-miss "No skills match “…”."
- **History discipline:** job/skill *selections* push; all normalizations (invalid id, stale `?skill=`, tenant switch) replace.
- **Strict TS, no `any`.** Tests are Vitest-native (`vi.*`), colocated under `__tests__/`.
- **Gates:** `npm run test`, `npm run lint`, `npm run build` clean in `services/atlas-ui`; `tools/lint.sh --check` clean from the repo root.
- All paths below are relative to `services/atlas-ui/` unless prefixed with `docs/` or `tools/`.

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `src/lib/jobs/job-advancement-tree.ts` | Modify | GM reparent, floors cleanup, new pure helpers (`advancementChains`, `tierLabel`, `subtreeCount`) |
| `src/lib/jobs/__tests__/job-advancement-tree.test.ts` | Modify | Updated invariants + new helper tests |
| `src/index.css` | Modify | Nine `--c-*` accent tokens + `--acc-fg` (light + dark) |
| `src/components/features/jobs/rail-groups.ts` (+test) | Create | `RAIL_GROUPS` constant, `branchEntryOf`, `visibleRailGroups` |
| `src/hooks/use-media-query.ts` (+test) | Create | Generalized matchMedia hook (`use-mobile.tsx` untouched) |
| `src/components/features/jobs/skill-icon.tsx` (+test) | Create | `SkillIcon` moved from `JobDetailPage` (img + Sparkles fallback) |
| `src/components/features/jobs/branch-rail.tsx` (+test) | Create | Left pane |
| `src/components/features/jobs/advancement-flow.tsx` (+test) | Create | Tier-aligned chip grid |
| `src/components/features/jobs/skill-list.tsx` (+test) | Create | Search + rows + 5 states |
| `src/components/features/jobs/skill-detail.tsx` (+test) | Create | Detail content (pane AND sheet body) |
| `src/pages/JobsPage.tsx` (+test rewrite) | Rewrite | URL state + composition only |
| `src/pages/JobDetailPage.tsx` (+test) | Delete | Folded into the explorer |
| `src/App.tsx` | Modify | `/jobs/:jobId` → `JobsPage`; drop `JobDetailPage` lazy import |
| `src/lib/breadcrumbs/__tests__/routes.test.ts` | Modify | Pin `/jobs` + `/jobs/[id]` resolution (no `routes.ts` change expected) |

Component tests live in `src/components/features/jobs/__tests__/`.

---

### Task 1: Reparent the GM line in the display graph

**Files:**
- Modify: `src/lib/jobs/job-advancement-tree.ts`
- Test: `src/lib/jobs/__tests__/job-advancement-tree.test.ts`

**Interfaces:**
- Consumes: nothing new.
- Produces: `JOB_GRAPH[900].parent === 0`, `JOB_GRAPH[910].parent === 900`; `BRANCH_FLOORS` without 900/910 keys; `JOB_ROOTS === [0, 800, 1000, 2000, 2001]`. All existing exports (`childrenOf`, `rootOf`, `floorOf`, `visibleRoots`, `visibleChildrenOf`, `jobTreePath`) keep their contracts.

- [ ] **Step 1: Update the failing tests first**

In `src/lib/jobs/__tests__/job-advancement-tree.test.ts`, replace these four test bodies (leave the others untouched):

```ts
  it("exposes the five branch roots ascending (GM line is no longer a root)", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500, 900]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(900)).toEqual([910]); // GM advances to Super GM
    expect(childrenOf(112)).toEqual([]); // 4th job is a leaf
  });

  it("uses the corrected per-branch floors, inherited from the root", () => {
    expect(BRANCH_FLOORS).toEqual({
      0: 1,
      800: 83,
      1000: 83,
      2000: 80,
      2001: 84,
    });
    expect(floorOf(112)).toBe(1); // Adventurer — present since launch
    expect(floorOf(900)).toBe(1); // GM — inherits Beginner's floor as a non-root
    expect(floorOf(910)).toBe(1); // Super GM — likewise
    expect(floorOf(1112)).toBe(83); // Cygnus corrected 92 -> 83
    expect(floorOf(2112)).toBe(80); // Aran corrected 88 -> 80
    expect(floorOf(2218)).toBe(84); // Evan
  });

  it("shows base Adventurers + the GM line on legacy sub-83 versions (GMS v12/v48)", () => {
    const r12 = visibleRoots(12);
    expect(r12).toContain(0); // Adventurers — the jobs page was empty before this
    expect(r12).not.toContain(1000); // Cygnus (v83) hidden
    expect(r12).not.toContain(2000); // Aran (v80) hidden
    expect(r12).not.toContain(800); // Maple Leaf Brigadier (special) hidden
    // GM/Super GM are children of Beginner now — visible via the tree, not as roots.
    expect(visibleChildrenOf(0, 12)).toContain(900);
    expect(visibleChildrenOf(900, 12)).toContain(910);
  });
```

Also extend the `jobTreePath` test with the GM-line invariant (inside the existing `it("jobTreePath returns root->node inclusive", ...)`):

```ts
    expect(jobTreePath(910).map((j) => j.id)).toEqual([0, 900, 910]);
    expect(jobTreePath(910).map((j) => j.name)).toEqual([
      "Beginner",
      "GM",
      "Super GM",
    ]);
```

And the `rootOf` test gains:

```ts
    expect(rootOf(910)).toBe(0); // Super GM -> GM -> Beginner
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ui && npm run test -- src/lib/jobs`
Expected: FAIL — `JOB_ROOTS` still contains 900/910, `childrenOf(0)` lacks 900, `BRANCH_FLOORS` still has 900/910 keys.

- [ ] **Step 3: Apply the graph change**

In `src/lib/jobs/job-advancement-tree.ts`:

Replace the `// Special / Admin (standalone roots)` block:

```ts
  // Special / Admin. Maple Leaf Brigadier stays a standalone root. In-game, GM
  // and Super GM present as an advancement line from Beginner, so the DISPLAY
  // graph adopts Beginner > GM > Super GM — an intentional divergence from
  // libs/atlas-constants/job/constants.go, where 900/910 are roots (task-182).
  800: { id: 800, name: "Maple Leaf Brigadier", parent: null },
  900: { id: 900, name: "GM", parent: 0 },
  910: { id: 910, name: "Super GM", parent: 900 },
```

In `BRANCH_FLOORS`, delete the `900: 1,` and `910: 1,` lines, and delete the two comment lines above the constant that describe them (`//   900  GM ... (floor 1)` and `//   910  Super GM ... (floor 1)`). As non-roots they now inherit the Beginner root floor of 1 — unchanged effective floor.

Update the file-header provenance comment (lines 7–10) to:

```ts
// Structural source of truth for the job advancement graph.
// Ported from the former lib/utils/job-tree.ts JOB_TREE, whose ids/names derive
// from libs/atlas-constants/job/constants.go::Jobs (v83 conventions). One
// intentional divergence: constants.go has GM (900) and Super GM (910) as
// roots, but in-game they present as an advancement line from Beginner, so
// this display graph parents 900 under 0 and 910 under 900 (task-182).
// Order per branch: branch leader -> 1st -> 2nd -> 3rd -> 4th.
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-ui && npm run test -- src/lib/jobs`
Expected: PASS (all tests in the file, including untouched ones — `Pirate`, orphan-reference sweep, etc.)

Note: `src/pages/__tests__/JobsPage.test.tsx` may now fail (the old tree page rendered 900/910 as roots). That page is rewritten in Task 10; if its test breaks here, that is expected — do NOT fix the old page. Confirm only the graph tests pass and move on.

- [ ] **Step 5: Commit**

```bash
git add src/lib/jobs/job-advancement-tree.ts src/lib/jobs/__tests__/job-advancement-tree.test.ts
git commit -m "feat(atlas-ui): reparent GM line under Beginner in job display graph (task-182)"
```

---

### Task 2: Graph helpers — `advancementChains`, `tierLabel`, `subtreeCount`

**Files:**
- Modify: `src/lib/jobs/job-advancement-tree.ts` (append after `jobTreePath`)
- Test: `src/lib/jobs/__tests__/job-advancement-tree.test.ts` (append new `describe` blocks)

**Interfaces:**
- Consumes: `childrenOf`, `floorOf`, `visibleChildrenOf`, `jobTreePath` (Task 1 file).
- Produces (exact signatures later tasks rely on):
  - `advancementChains(entryId: number, major: number): number[][]` — descendant chains **excluding** `entryId` itself; a leaf entry yields `[]`.
  - `tierLabel(jobId: number): string` — `"Base"` / `""` / `"1st"`…`"10th"`.
  - `subtreeCount(entryId: number, major: number): number` — visible nodes including the entry.

- [ ] **Step 1: Write the failing tests**

Append to `src/lib/jobs/__tests__/job-advancement-tree.test.ts` (add `advancementChains`, `tierLabel`, `subtreeCount` to the import list):

```ts
describe("advancementChains", () => {
  it("returns one chain per advancement path below the entry, ascending, entry excluded", () => {
    expect(advancementChains(100, 83)).toEqual([
      [110, 111, 112],
      [120, 121, 122],
      [130, 131, 132],
    ]);
  });

  it("handles the Evan 10-tier single chain", () => {
    expect(advancementChains(2001, 84)).toEqual([
      [2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218],
    ]);
  });

  it("renders the GM line as a single chain below the GM entry", () => {
    expect(advancementChains(900, 1)).toEqual([[910]]);
  });

  it("includes the GM line among Beginner's chains", () => {
    const chains = advancementChains(0, 83);
    expect(chains).toContainEqual([900, 910]);
    expect(chains).toContainEqual([100, 110, 111, 112]);
    // Warrior/Magician have 3 paths each; Bowman/Rogue/Pirate 2 each; + GM line
    expect(chains).toHaveLength(13);
  });

  it("drops any chain containing a below-floor node (Pirate on v12)", () => {
    const chains = advancementChains(0, 12);
    expect(chains).toHaveLength(11); // 13 minus Pirate's 2 paths
    for (const chain of chains) {
      expect(chain).not.toContain(500);
    }
  });

  it("returns [] for a leaf entry", () => {
    expect(advancementChains(112, 83)).toEqual([]);
    expect(advancementChains(800, 83)).toEqual([]);
  });
});

describe("tierLabel", () => {
  it("labels tree roots with children as Base, childless roots empty", () => {
    expect(tierLabel(0)).toBe("Base");
    expect(tierLabel(1000)).toBe("Base");
    expect(tierLabel(800)).toBe("");
  });

  it("labels advancement depth ordinally, including the GM line and Evan depth 10", () => {
    expect(tierLabel(100)).toBe("1st");
    expect(tierLabel(110)).toBe("2nd");
    expect(tierLabel(111)).toBe("3rd");
    expect(tierLabel(112)).toBe("4th");
    expect(tierLabel(900)).toBe("1st");
    expect(tierLabel(910)).toBe("2nd");
    expect(tierLabel(2218)).toBe("10th");
  });

  it("returns empty for unknown ids", () => {
    expect(tierLabel(99999)).toBe("");
  });
});

describe("subtreeCount", () => {
  it("counts version-visible nodes including the entry", () => {
    expect(subtreeCount(100, 83)).toBe(10); // Warrior line
    expect(subtreeCount(900, 1)).toBe(2); // GM + Super GM
    expect(subtreeCount(1000, 83)).toBe(21); // Noblesse + 5 paths x 4
    expect(subtreeCount(800, 83)).toBe(1);
    expect(subtreeCount(2001, 84)).toBe(11); // Evan root + 10 tiers
  });

  it("excludes below-floor subtrees and below-floor entries", () => {
    // Beginner + Warrior 10 + Magician 10 + Bowman 7 + Rogue 7 + GM line 2
    expect(subtreeCount(0, 12)).toBe(37);
    expect(subtreeCount(0, 62)).toBe(44); // Pirate's 7 nodes join at v62
    expect(subtreeCount(500, 12)).toBe(0); // Pirate entry itself below floor
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-ui && npm run test -- src/lib/jobs`
Expected: FAIL with unresolved imports (`advancementChains` is not exported).

- [ ] **Step 3: Implement the helpers**

Append to `src/lib/jobs/job-advancement-tree.ts`:

```ts
/**
 * Every advancement chain below entryId: one array per root-to-leaf path of
 * the subtree, EXCLUDING entryId itself, DFS in ascending child order. A
 * chain containing any below-floor node is dropped entirely (matches
 * visibleChildrenOf semantics). A leaf entry yields [].
 */
export function advancementChains(entryId: number, major: number): number[][] {
  const walk = (id: number): number[][] => {
    const kids = childrenOf(id);
    if (kids.length === 0) return [[]];
    const out: number[][] = [];
    for (const k of kids) {
      for (const rest of walk(k)) out.push([k, ...rest]);
    }
    return out;
  };
  return walk(entryId)
    .filter((chain) => chain.length > 0)
    .filter((chain) => chain.every((id) => floorOf(id) <= major));
}

function ordinal(n: number): string {
  if (n === 1) return "1st";
  if (n === 2) return "2nd";
  if (n === 3) return "3rd";
  return `${n}th`;
}

/**
 * Tier tag for a flow chip: "Base" for a tree root with children, "" for a
 * childless root or unknown id, else the ordinal advancement depth
 * ("1st" … "10th") measured from the tree root.
 */
export function tierLabel(jobId: number): string {
  const depth = jobTreePath(jobId).length - 1;
  if (depth < 0) return "";
  if (depth === 0) return childrenOf(jobId).length > 0 ? "Base" : "";
  return ordinal(depth);
}

/** Count of version-visible nodes in entryId's subtree, entry included (0 if the entry itself is below floor). */
export function subtreeCount(entryId: number, major: number): number {
  if (JOB_GRAPH[entryId] === undefined || floorOf(entryId) > major) return 0;
  return (
    1 +
    visibleChildrenOf(entryId, major).reduce(
      (n, k) => n + subtreeCount(k, major),
      0,
    )
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-ui && npm run test -- src/lib/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/lib/jobs/job-advancement-tree.ts src/lib/jobs/__tests__/job-advancement-tree.test.ts
git commit -m "feat(atlas-ui): advancement chains, tier labels, subtree counts for jobs explorer (task-182)"
```

---

### Task 3: Branch accent tokens + rail groups module

**Files:**
- Modify: `src/index.css` (`:root` and `.dark` blocks)
- Create: `src/components/features/jobs/rail-groups.ts`
- Test: `src/components/features/jobs/__tests__/rail-groups.test.ts`

**Interfaces:**
- Consumes: `JOB_GRAPH`, `floorOf`, `jobTreePath`, `subtreeCount` from `@/lib/jobs/job-advancement-tree`.
- Produces:
  - CSS tokens `--c-warrior|magician|bowman|thief|pirate|cygnus|aran|evan|special` and `--acc-fg` in both themes.
  - `interface RailEntry { id: number; accent: string }` (accent = token name, e.g. `"--c-warrior"`).
  - `RAIL_GROUPS: RailGroup[]` where `interface RailGroup { label: string; entries: RailEntry[] }`.
  - `branchEntryOf(jobId: number): RailEntry` — entry whose id is on `jobTreePath(jobId)`; falls back to the Warrior entry.
  - `visibleRailGroups(major: number): VisibleRailGroup[]` with `interface VisibleRailEntry extends RailEntry { name: string; count: number }`, `interface VisibleRailGroup { label: string; entries: VisibleRailEntry[] }` — version-gated, empty groups dropped.

- [ ] **Step 1: Add the CSS tokens**

In `src/index.css`, inside the `:root {` block (after the existing `--nord-15` line), add:

```css
    /* Branch accents for the jobs explorer (Nord aurora/frost, darkened for the light ground) — task-182 */
    --c-warrior: 354 42% 48%;
    --c-magician: 213 32% 44%;
    --c-bowman: 92 28% 38%;
    --c-thief: 311 20% 45%;
    --c-pirate: 14 51% 46%;
    --c-cygnus: 40 71% 38%;
    --c-aran: 193 43% 38%;
    --c-evan: 179 25% 38%;
    --c-special: 210 34% 44%;
    --acc-fg: 210 20% 98%;
```

Inside the `.dark {` block (at the end, before its closing brace), add:

```css
    /* Branch accents: Nord originals read well on the dark ground — task-182 */
    --c-warrior: 354 42% 56%;
    --c-magician: 213 32% 63%;
    --c-bowman: 92 28% 65%;
    --c-thief: 311 20% 63%;
    --c-pirate: 14 51% 63%;
    --c-cygnus: 40 71% 73%;
    --c-aran: 193 43% 67%;
    --c-evan: 179 25% 65%;
    --c-special: 210 34% 63%;
    --acc-fg: 220 16% 22%;
```

- [ ] **Step 2: Write the failing tests**

Create `src/components/features/jobs/__tests__/rail-groups.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  RAIL_GROUPS,
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";

describe("RAIL_GROUPS", () => {
  it("defines the four labeled groups with the PRD's entries in order", () => {
    expect(RAIL_GROUPS.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(RAIL_GROUPS[0].entries.map((e) => e.id)).toEqual([
      100, 200, 300, 400, 500,
    ]);
    expect(RAIL_GROUPS[1].entries.map((e) => e.id)).toEqual([1000]);
    expect(RAIL_GROUPS[2].entries.map((e) => e.id)).toEqual([2000, 2001]);
    expect(RAIL_GROUPS[3].entries.map((e) => e.id)).toEqual([800, 900]);
  });

  it("maps every entry to a --c-* accent token name", () => {
    for (const g of RAIL_GROUPS) {
      for (const e of g.entries) {
        expect(e.accent).toMatch(/^--c-[a-z]+$/);
      }
    }
    expect(RAIL_GROUPS[0].entries[0].accent).toBe("--c-warrior");
    expect(RAIL_GROUPS[3].entries[1].accent).toBe("--c-special");
  });
});

describe("branchEntryOf", () => {
  it("resolves any node to the rail entry on its path", () => {
    expect(branchEntryOf(112).id).toBe(100); // Hero -> Warrior
    expect(branchEntryOf(910).id).toBe(900); // Super GM -> GM entry
    expect(branchEntryOf(1512).id).toBe(1000); // Thunder Breaker 4 -> Noblesse
    expect(branchEntryOf(2218).id).toBe(2001); // Evan 10 -> Evan
    expect(branchEntryOf(800).id).toBe(800);
  });

  it("falls back to the Warrior entry for Beginner and unknown ids", () => {
    expect(branchEntryOf(0).id).toBe(100);
    expect(branchEntryOf(99999).id).toBe(100);
  });
});

describe("visibleRailGroups", () => {
  it("gates entries by version floor and drops empty groups (GMS v12)", () => {
    const groups = visibleRailGroups(12);
    expect(groups.map((g) => g.label)).toEqual(["Explorers", "Special"]);
    expect(groups[0].entries.map((e) => e.id)).toEqual([100, 200, 300, 400]); // no Pirate
    expect(groups[1].entries.map((e) => e.id)).toEqual([900]); // no Brigadier (v83)
  });

  it("adds Pirate at v62, Cygnus/Aran/Brigadier at v83, Evan at v84", () => {
    expect(
      visibleRailGroups(62)[0].entries.map((e) => e.id),
    ).toContain(500);
    const v83 = visibleRailGroups(83);
    expect(v83.map((g) => g.label)).toEqual([
      "Explorers",
      "Cygnus Knights",
      "Legends",
      "Special",
    ]);
    expect(v83[2].entries.map((e) => e.id)).toEqual([2000]); // Evan hidden
    expect(v83[3].entries.map((e) => e.id)).toEqual([800, 900]);
    expect(visibleRailGroups(84)[2].entries.map((e) => e.id)).toEqual([
      2000, 2001,
    ]);
  });

  it("decorates entries with display name and visible subtree count", () => {
    const v83 = visibleRailGroups(83);
    const warrior = v83[0].entries[0];
    expect(warrior.name).toBe("Warrior");
    expect(warrior.count).toBe(10);
    const gm = v83[3].entries.find((e) => e.id === 900);
    expect(gm?.name).toBe("GM");
    expect(gm?.count).toBe(2);
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `rail-groups` does not exist.

- [ ] **Step 4: Implement the module**

Create `src/components/features/jobs/rail-groups.ts`:

```ts
import {
  JOB_GRAPH,
  floorOf,
  jobTreePath,
  subtreeCount,
} from "@/lib/jobs/job-advancement-tree";

export interface RailEntry {
  /** JOB_GRAPH node whose advancement flow the entry shows. */
  id: number;
  /** Theme token name for the branch accent, e.g. "--c-warrior" (src/index.css). */
  accent: string;
}

export interface RailGroup {
  label: string;
  entries: RailEntry[];
}

// Rail entries per PRD FR-3.1; accents are scoped via style={{ "--acc": `var(${accent})` }}.
export const RAIL_GROUPS: RailGroup[] = [
  {
    label: "Explorers",
    entries: [
      { id: 100, accent: "--c-warrior" },
      { id: 200, accent: "--c-magician" },
      { id: 300, accent: "--c-bowman" },
      { id: 400, accent: "--c-thief" },
      { id: 500, accent: "--c-pirate" },
    ],
  },
  { label: "Cygnus Knights", entries: [{ id: 1000, accent: "--c-cygnus" }] },
  {
    label: "Legends",
    entries: [
      { id: 2000, accent: "--c-aran" },
      { id: 2001, accent: "--c-evan" },
    ],
  },
  {
    label: "Special",
    entries: [
      { id: 800, accent: "--c-special" },
      { id: 900, accent: "--c-special" },
    ],
  },
];

/**
 * The rail entry whose node lies on jobId's advancement path. Beginner (0) and
 * unknown ids fall back to the first entry (Warrior) — the caller keeps the
 * job selection itself.
 */
export function branchEntryOf(jobId: number): RailEntry {
  const path = jobTreePath(jobId).map((e) => e.id);
  for (const g of RAIL_GROUPS) {
    for (const e of g.entries) {
      if (path.includes(e.id)) return e;
    }
  }
  return RAIL_GROUPS[0].entries[0];
}

export interface VisibleRailEntry extends RailEntry {
  name: string;
  count: number;
}

export interface VisibleRailGroup {
  label: string;
  entries: VisibleRailEntry[];
}

/** Version-gated rail groups with display name + visible-subtree count; empty groups dropped. */
export function visibleRailGroups(major: number): VisibleRailGroup[] {
  return RAIL_GROUPS.map((g) => ({
    label: g.label,
    entries: g.entries
      .filter((e) => floorOf(e.id) <= major)
      .map((e) => ({
        ...e,
        name: JOB_GRAPH[e.id]?.name ?? `Job ${e.id}`,
        count: subtreeCount(e.id, major),
      })),
  })).filter((g) => g.entries.length > 0);
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/index.css src/components/features/jobs/rail-groups.ts src/components/features/jobs/__tests__/rail-groups.test.ts
git commit -m "feat(atlas-ui): branch accent tokens and rail groups for jobs explorer (task-182)"
```

---

### Task 4: `useMediaQuery` hook

**Files:**
- Create: `src/hooks/use-media-query.ts`
- Test: `src/hooks/__tests__/use-media-query.test.ts`

**Interfaces:**
- Produces: `useMediaQuery(query: string): boolean`. `src/hooks/use-mobile.tsx` stays untouched (it hard-codes the sidebar's 768px breakpoint).

Note: `src/test/setup.ts` globally stubs `window.matchMedia` with `matches: false` and no-op listeners — so in every component test the explorer defaults to the **narrow** layout unless a test installs its own stub.

- [ ] **Step 1: Write the failing test**

Create `src/hooks/__tests__/use-media-query.test.ts`:

```tsx
import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, afterEach } from "vitest";
import { useMediaQuery } from "@/hooks/use-media-query";

const originalMatchMedia = window.matchMedia;

function stubMatchMedia(initial: boolean) {
  let matches = initial;
  const listeners = new Set<() => void>();
  window.matchMedia = ((query: string) =>
    ({
      get matches() {
        return matches;
      },
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: (_: string, cb: () => void) => listeners.add(cb),
      removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList) as typeof window.matchMedia;
  return {
    set(next: boolean) {
      matches = next;
      listeners.forEach((cb) => cb());
    },
  };
}

afterEach(() => {
  window.matchMedia = originalMatchMedia;
});

describe("useMediaQuery", () => {
  it("returns the current match state synchronously", () => {
    stubMatchMedia(true);
    const { result } = renderHook(() =>
      useMediaQuery("(min-width: 1150px)"),
    );
    expect(result.current).toBe(true);
  });

  it("re-renders when the media query flips", () => {
    const media = stubMatchMedia(false);
    const { result } = renderHook(() =>
      useMediaQuery("(min-width: 1150px)"),
    );
    expect(result.current).toBe(false);
    act(() => media.set(true));
    expect(result.current).toBe(true);
    act(() => media.set(false));
    expect(result.current).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/hooks`
Expected: FAIL — module `use-media-query` does not exist.

- [ ] **Step 3: Implement the hook**

Create `src/hooks/use-media-query.ts`:

```ts
import * as React from "react";

// Generalized matchMedia subscription — same useSyncExternalStore pattern as
// use-mobile.tsx (which stays hard-coded to the sidebar's 768px breakpoint):
// the snapshot is read synchronously on first render instead of flashing a
// default until an effect commits.
export function useMediaQuery(query: string): boolean {
  const subscribe = React.useCallback(
    (onChange: () => void) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onChange);
      return () => mql.removeEventListener("change", onChange);
    },
    [query],
  );
  const getSnapshot = React.useCallback(
    () => window.matchMedia(query).matches,
    [query],
  );
  return React.useSyncExternalStore(subscribe, getSnapshot);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/hooks`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/hooks/use-media-query.ts src/hooks/__tests__/use-media-query.test.ts
git commit -m "feat(atlas-ui): generalized useMediaQuery hook (task-182)"
```

---

### Task 5: `SkillIcon` feature component

**Files:**
- Create: `src/components/features/jobs/skill-icon.tsx`
- Test: `src/components/features/jobs/__tests__/skill-icon.test.tsx`

**Interfaces:**
- Consumes: `SkillDefinitionWithIcon` from `@/lib/hooks/api/useSkillDefinition`.
- Produces: `SkillIcon({ def, name }: { def: SkillDefinitionWithIcon; name: string })` — real icon `<img>` with the Sparkles fallback on load error (`data-testid="skill-icon-fallback-<id>"`), moved verbatim from `JobDetailPage` (which is deleted in Task 10; do not edit it here).

- [ ] **Step 1: Write the failing test**

Create `src/components/features/jobs/__tests__/skill-icon.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { SkillIcon } from "@/components/features/jobs/skill-icon";

const def: SkillDefinitionWithIcon = {
  id: 1001004,
  name: "Power Strike",
  description: "",
  action: true,
  element: "",
  animationTime: 0,
  maxLevel: 20,
  effects: [],
  iconUrl: "http://assets.test/skills/1001004/icon",
};

describe("SkillIcon", () => {
  it("renders the real icon image", () => {
    render(<SkillIcon def={def} name="Power Strike" />);
    const img = screen.getByRole("img", { name: "Power Strike" });
    expect(img).toHaveAttribute("src", def.iconUrl);
  });

  it("falls back to the Sparkles glyph when the image errors", () => {
    render(<SkillIcon def={def} name="Power Strike" />);
    fireEvent.error(screen.getByRole("img", { name: "Power Strike" }));
    expect(
      screen.getByTestId("skill-icon-fallback-1001004"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `skill-icon` does not exist.

- [ ] **Step 3: Implement the component** (verbatim move from `JobDetailPage.tsx` lines 39–68)

Create `src/components/features/jobs/skill-icon.tsx`:

```tsx
import { useState } from "react";
import { Sparkles } from "lucide-react";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";

export function SkillIcon({
  def,
  name,
}: {
  def: SkillDefinitionWithIcon;
  name: string;
}) {
  const [failed, setFailed] = useState(false);
  if (failed) {
    return (
      <span
        data-testid={`skill-icon-fallback-${def.id}`}
        className="inline-flex h-8 w-8 items-center justify-center text-muted-foreground"
      >
        <Sparkles className="h-4 w-4" />
      </span>
    );
  }
  return (
    <img
      src={def.iconUrl}
      alt={name}
      width={32}
      height={32}
      loading="lazy"
      className="object-contain"
      onError={() => setFailed(true)}
    />
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/jobs/skill-icon.tsx src/components/features/jobs/__tests__/skill-icon.test.tsx
git commit -m "feat(atlas-ui): extract SkillIcon into jobs feature components (task-182)"
```

---

### Task 6: `BranchRail` component

**Files:**
- Create: `src/components/features/jobs/branch-rail.tsx`
- Test: `src/components/features/jobs/__tests__/branch-rail.test.tsx`

**Interfaces:**
- Consumes: `VisibleRailGroup` from `./rail-groups` (Task 3); shadcn `Card`.
- Produces: `BranchRail({ groups, selectedEntryId, onSelect }: { groups: VisibleRailGroup[]; selectedEntryId: number; onSelect: (id: number) => void })`. Selection is conveyed with `aria-pressed`; accent applied via scoped `--acc`.

- [ ] **Step 1: Write the failing test**

Create `src/components/features/jobs/__tests__/branch-rail.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { BranchRail } from "@/components/features/jobs/branch-rail";
import { visibleRailGroups } from "@/components/features/jobs/rail-groups";

describe("BranchRail", () => {
  it("renders group labels, entry names, and subtree counts", () => {
    render(
      <BranchRail
        groups={visibleRailGroups(83)}
        selectedEntryId={100}
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("Explorers")).toBeInTheDocument();
    expect(screen.getByText("Cygnus Knights")).toBeInTheDocument();
    expect(screen.getByText("Legends")).toBeInTheDocument();
    expect(screen.getByText("Special")).toBeInTheDocument();
    const warrior = screen.getByRole("button", { name: /Warrior 10/ });
    expect(warrior).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: /^GM 2$/ })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("fires onSelect with the entry id", () => {
    const onSelect = vi.fn();
    render(
      <BranchRail
        groups={visibleRailGroups(83)}
        selectedEntryId={100}
        onSelect={onSelect}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Magician/ }));
    expect(onSelect).toHaveBeenCalledWith(200);
  });

  it("scopes the branch accent token per entry", () => {
    render(
      <BranchRail
        groups={visibleRailGroups(83)}
        selectedEntryId={100}
        onSelect={() => {}}
      />,
    );
    const warrior = screen.getByRole("button", { name: /Warrior 10/ });
    expect(warrior.style.getPropertyValue("--acc")).toBe("var(--c-warrior)");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `branch-rail` does not exist.

- [ ] **Step 3: Implement the component**

Create `src/components/features/jobs/branch-rail.tsx`:

```tsx
import type { CSSProperties } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { VisibleRailGroup } from "@/components/features/jobs/rail-groups";

interface BranchRailProps {
  groups: VisibleRailGroup[];
  selectedEntryId: number;
  onSelect: (id: number) => void;
}

export function BranchRail({
  groups,
  selectedEntryId,
  onSelect,
}: BranchRailProps) {
  return (
    <Card className="flex min-h-0 flex-col">
      <CardHeader className="pb-1">
        <CardTitle className="text-[15px]">Branches</CardTitle>
      </CardHeader>
      <CardContent className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {groups.map((g) => (
          <div key={g.label}>
            <h3 className="mx-2 mb-1 mt-2.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {g.label}
            </h3>
            {g.entries.map((e) => (
              <button
                key={e.id}
                type="button"
                aria-pressed={selectedEntryId === e.id}
                onClick={() => onSelect(e.id)}
                style={{ "--acc": `var(${e.accent})` } as CSSProperties}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[13.5px] hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring aria-pressed:bg-[hsl(var(--acc)/0.14)] aria-pressed:font-medium"
              >
                <span
                  aria-hidden
                  className="h-2 w-2 flex-none rounded-[3px] bg-[hsl(var(--acc))]"
                />
                <span className="truncate">{e.name}</span>
                <span className="ml-auto text-[11.5px] tabular-nums text-muted-foreground">
                  {e.count}
                </span>
              </button>
            ))}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/jobs/branch-rail.tsx src/components/features/jobs/__tests__/branch-rail.test.tsx
git commit -m "feat(atlas-ui): BranchRail pane for jobs explorer (task-182)"
```

---

### Task 7: `AdvancementFlow` component

**Files:**
- Create: `src/components/features/jobs/advancement-flow.tsx`
- Test: `src/components/features/jobs/__tests__/advancement-flow.test.tsx`

**Interfaces:**
- Consumes: `JOB_GRAPH`, `advancementChains`, `jobTreePath`, `tierLabel` (Tasks 1–2); `cn` from `@/lib/utils`.
- Produces: `AdvancementFlow({ entryId, major, selectedJobId, accent, onSelect }: { entryId: number; major: number; selectedJobId: number; accent: string; onSelect: (id: number) => void })`. Every chip is a `<button>` with `aria-pressed`; each grid cell carries `data-testid="flow-cell-<jobId>"` with explicit `gridColumn`/`gridRow` inline styles (anchors span all rows). Grid layout per D2: anchor cells at column `i+1` row `1 / span rows`; chain node k of chain r at column `anchorCols + 1 + k`, row `r + 1`.

- [ ] **Step 1: Write the failing test**

Create `src/components/features/jobs/__tests__/advancement-flow.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { AdvancementFlow } from "@/components/features/jobs/advancement-flow";

function cell(id: number): HTMLElement {
  return screen.getByTestId(`flow-cell-${id}`);
}

describe("AdvancementFlow", () => {
  it("tier-aligns the Magician branch: same-tier jobs share a grid column", () => {
    render(
      <AdvancementFlow
        entryId={200}
        major={83}
        selectedJobId={200}
        accent="--c-magician"
        onSelect={() => {}}
      />,
    );
    // anchors: Beginner (col 1), Magician (col 2), spanning all 3 path rows
    expect(cell(0).style.gridColumn).toBe("1");
    expect(cell(0).style.gridRow).toBe("1 / span 3");
    expect(cell(200).style.gridColumn).toBe("2");
    // 2nd-job tier column: Wizard (F/P) / Wizard (I/L) / Cleric vertically aligned
    expect(cell(210).style.gridColumn).toBe("3");
    expect(cell(210).style.gridRow).toBe("1");
    expect(cell(220).style.gridColumn).toBe("3");
    expect(cell(220).style.gridRow).toBe("2");
    expect(cell(230).style.gridColumn).toBe("3");
    expect(cell(230).style.gridRow).toBe("3");
    // 4th-job tier aligned likewise (2 anchors + chain positions 3..5)
    expect(cell(212).style.gridColumn).toBe("5");
    expect(cell(232).style.gridColumn).toBe("5");
  });

  it("renders the GM line Beginner > GM > Super GM with tier tags", () => {
    render(
      <AdvancementFlow
        entryId={900}
        major={83}
        selectedJobId={900}
        accent="--c-special"
        onSelect={() => {}}
      />,
    );
    expect(cell(0).style.gridColumn).toBe("1");
    expect(cell(900).style.gridColumn).toBe("2");
    expect(cell(910).style.gridColumn).toBe("3");
    expect(cell(910).style.gridRow).toBe("1");
    expect(screen.getByText("Base")).toBeInTheDocument();
    expect(screen.getByText("1st")).toBeInTheDocument();
    expect(screen.getByText("2nd")).toBeInTheDocument();
  });

  it("marks the selected chip pressed and fires onSelect on click", () => {
    const onSelect = vi.fn();
    render(
      <AdvancementFlow
        entryId={100}
        major={83}
        selectedJobId={110}
        accent="--c-warrior"
        onSelect={onSelect}
      />,
    );
    expect(screen.getByRole("button", { name: /Fighter/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    fireEvent.click(screen.getByRole("button", { name: /Page/ }));
    expect(onSelect).toHaveBeenCalledWith(120);
  });

  it("omits version-hidden paths (no Pirate content below floor)", () => {
    render(
      <AdvancementFlow
        entryId={500}
        major={83}
        selectedJobId={500}
        accent="--c-pirate"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: /Brawler/ })).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `advancement-flow` does not exist.

- [ ] **Step 3: Implement the component**

Create `src/components/features/jobs/advancement-flow.tsx`:

```tsx
import { useMemo } from "react";
import type { CSSProperties } from "react";
import {
  JOB_GRAPH,
  advancementChains,
  jobTreePath,
  tierLabel,
} from "@/lib/jobs/job-advancement-tree";
import { cn } from "@/lib/utils";

interface AdvancementFlowProps {
  entryId: number;
  major: number;
  selectedJobId: number;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
  onSelect: (id: number) => void;
}

function FlowChip({
  id,
  selected,
  onSelect,
}: {
  id: number;
  selected: boolean;
  onSelect: (id: number) => void;
}) {
  const tier = tierLabel(id);
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={() => onSelect(id)}
      className={cn(
        "inline-flex items-center gap-1.5 whitespace-nowrap rounded-md border px-2.5 py-1 text-[13px] font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        selected
          ? "border-[hsl(var(--acc))] bg-[hsl(var(--acc))] text-[hsl(var(--acc-fg))]"
          : "bg-card hover:border-[hsl(var(--acc))]",
      )}
    >
      {JOB_GRAPH[id]?.name ?? `Job ${id}`}
      {tier ? (
        <span
          className={cn(
            "rounded px-1 py-px text-[10px] font-semibold tracking-wide",
            selected
              ? "bg-[hsl(var(--acc-fg)/0.22)] text-[hsl(var(--acc-fg))]"
              : "bg-secondary text-muted-foreground",
          )}
        >
          {tier}
        </span>
      ) : null}
    </button>
  );
}

/**
 * Tier-aligned advancement grid (design D2, ported from the approved mock):
 * ancestors + the entry are "anchor" cells spanning every path row, vertically
 * centered; chain node k of path r lands at column anchors+1+k, row r+1, so
 * same-tier chips share an implicit auto column and align with zero
 * measurement code.
 */
export function AdvancementFlow({
  entryId,
  major,
  selectedJobId,
  accent,
  onSelect,
}: AdvancementFlowProps) {
  const anchors = useMemo(
    () => jobTreePath(entryId).map((e) => e.id),
    [entryId],
  );
  const chains = useMemo(
    () => advancementChains(entryId, major),
    [entryId, major],
  );
  const rows = Math.max(chains.length, 1);
  const anchorCols = anchors.length;
  const sep = (
    <span aria-hidden className="mx-px flex-none text-muted-foreground/55">
      ›
    </span>
  );
  return (
    <div className="overflow-x-auto pb-0.5">
      <div
        className="mx-auto grid w-max gap-x-1 gap-y-1.5"
        style={{ "--acc": `var(${accent})` } as CSSProperties}
      >
        {anchors.map((id, i) => (
          <div
            key={`anchor-${id}`}
            data-testid={`flow-cell-${id}`}
            className="flex items-center gap-1 self-center whitespace-nowrap"
            style={{ gridColumn: `${i + 1}`, gridRow: `1 / span ${rows}` }}
          >
            {i > 0 ? sep : null}
            <FlowChip
              id={id}
              selected={selectedJobId === id}
              onSelect={onSelect}
            />
          </div>
        ))}
        {chains.map((chain, r) =>
          chain.map((id, k) => (
            <div
              key={`chain-${id}`}
              data-testid={`flow-cell-${id}`}
              className="flex items-center gap-1 whitespace-nowrap [&>button]:flex-1 [&>button]:justify-center"
              style={{
                gridColumn: `${anchorCols + 1 + k}`,
                gridRow: `${r + 1}`,
              }}
            >
              {sep}
              <FlowChip
                id={id}
                selected={selectedJobId === id}
                onSelect={onSelect}
              />
            </div>
          )),
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/jobs/advancement-flow.tsx src/components/features/jobs/__tests__/advancement-flow.test.tsx
git commit -m "feat(atlas-ui): tier-aligned AdvancementFlow grid (task-182)"
```

---

### Task 8: `SkillList` component

**Files:**
- Create: `src/components/features/jobs/skill-list.tsx`
- Test: `src/components/features/jobs/__tests__/skill-list.test.tsx`

**Interfaces:**
- Consumes: `SkillIcon` (Task 5), `deriveSkillType`, `resolveSkillName`, shadcn `Badge`/`Input`/`Skeleton`.
- Produces:
  - `type SkillListState = "loading" | "error" | "empty" | "defs-failed" | "ready"`
  - `SkillList({ jobName, defs, state, selectedSkillId, accent, onSelect }: { jobName: string; defs: SkillDefinitionWithIcon[]; state: SkillListState; selectedSkillId: number | null; accent: string; onSelect: (id: number) => void })`
  - Filter text is local state; the page remounts the component with `key={jobId}` so it resets on job change (FR-3.4).

- [ ] **Step 1: Write the failing test**

Create `src/components/features/jobs/__tests__/skill-list.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { SkillList } from "@/components/features/jobs/skill-list";

function def(
  id: number,
  name: string,
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  return {
    id,
    name,
    description: "",
    action: true,
    element: "",
    animationTime: 0,
    maxLevel: 20,
    effects: [],
    iconUrl: `http://assets.test/skills/${id}/icon`,
    ...over,
  };
}

const defs = [
  def(1001004, "Power Strike"),
  def(1001005, "Slash Blast"),
  def(1001003, "Iron Body"),
];

function renderList(over?: Partial<Parameters<typeof SkillList>[0]>) {
  return render(
    <SkillList
      jobName="Warrior"
      defs={defs}
      state="ready"
      selectedSkillId={null}
      accent="--c-warrior"
      onSelect={() => {}}
      {...over}
    />,
  );
}

describe("SkillList", () => {
  it("renders rows with name, monospace id, type badge, and master level", () => {
    renderList();
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
    const row = screen.getByRole("button", { name: /Power Strike/ });
    expect(row).toHaveTextContent("1001004");
    expect(row).toHaveTextContent("Active");
    expect(row).toHaveTextContent("Master 20");
  });

  it("marks the selected row pressed and fires onSelect", () => {
    const onSelect = vi.fn();
    renderList({ selectedSkillId: 1001005, onSelect });
    expect(
      screen.getByRole("button", { name: /Slash Blast/ }),
    ).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: /Power Strike/ }));
    expect(onSelect).toHaveBeenCalledWith(1001004);
  });

  it("filters by case-insensitive name substring and by id substring", () => {
    renderList();
    const input = screen.getByLabelText("Filter skills");
    fireEvent.change(input, { target: { value: "power" } });
    expect(
      screen.getByRole("button", { name: /Power Strike/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Slash Blast/ }),
    ).not.toBeInTheDocument();
    fireEvent.change(input, { target: { value: "1001005" } });
    expect(
      screen.getByRole("button", { name: /Slash Blast/ }),
    ).toBeInTheDocument();
    fireEvent.change(input, { target: { value: "zzz" } });
    expect(screen.getByText(/No skills match/)).toBeInTheDocument();
  });

  it("renders the loading, error, empty, and defs-failed states verbatim", () => {
    const { rerender } = renderList({ state: "loading", defs: [] });
    expect(screen.getByTestId("skill-list-loading")).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="error"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(
      screen.getByText("Failed to load this job's skills."),
    ).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="empty"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("This job grants no skills.")).toBeInTheDocument();
    rerender(
      <SkillList
        jobName="Warrior"
        defs={[]}
        state="defs-failed"
        selectedSkillId={null}
        accent="--c-warrior"
        onSelect={() => {}}
      />,
    );
    expect(screen.getByText("Skill details unavailable.")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `skill-list` does not exist.

- [ ] **Step 3: Implement the component**

Create `src/components/features/jobs/skill-list.tsx`:

```tsx
import { useState } from "react";
import type { CSSProperties } from "react";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import { SkillIcon } from "@/components/features/jobs/skill-icon";

export type SkillListState =
  | "loading"
  | "error"
  | "empty"
  | "defs-failed"
  | "ready";

interface SkillListProps {
  jobName: string;
  defs: SkillDefinitionWithIcon[];
  state: SkillListState;
  selectedSkillId: number | null;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
  onSelect: (id: number) => void;
}

// Filter text is intentionally local (not URL) state; the page remounts this
// component with key={jobId} so the filter resets on job change (FR-3.4).
export function SkillList({
  jobName,
  defs,
  state,
  selectedSkillId,
  accent,
  onSelect,
}: SkillListProps) {
  const [filter, setFilter] = useState("");
  const q = filter.trim().toLowerCase();
  const filtered = defs.filter((d) => {
    if (!q) return true;
    return (
      resolveSkillName(d.id, d.name).toLowerCase().includes(q) ||
      String(d.id).includes(q)
    );
  });

  let body;
  if (state === "loading") {
    body = (
      <div data-testid="skill-list-loading" className="space-y-2 px-1 py-1">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  } else if (state === "error") {
    body = (
      <p className="py-8 text-center text-destructive">
        Failed to load this job&#39;s skills.
      </p>
    );
  } else if (state === "empty") {
    body = (
      <p className="py-8 text-center text-muted-foreground">
        This job grants no skills.
      </p>
    );
  } else if (state === "defs-failed") {
    body = (
      <p className="py-8 text-center text-destructive">
        Skill details unavailable.
      </p>
    );
  } else if (filtered.length === 0) {
    body = (
      <p className="py-8 text-center text-muted-foreground">
        No skills match &ldquo;{filter}&rdquo;.
      </p>
    );
  } else {
    body = filtered.map((d) => {
      const name = resolveSkillName(d.id, d.name);
      return (
        <button
          key={d.id}
          type="button"
          aria-pressed={selectedSkillId === d.id}
          onClick={() => onSelect(d.id)}
          className="flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring aria-pressed:bg-[hsl(var(--acc)/0.13)]"
        >
          <SkillIcon def={d} name={name} />
          <span className="min-w-0">
            <span className="block truncate text-[13.5px] font-medium">
              {name}
            </span>
            <span className="block font-mono text-[11px] text-muted-foreground">
              {d.id}
            </span>
          </span>
          <span className="ml-auto flex flex-none items-center gap-2">
            <Badge variant="secondary">{deriveSkillType(d)}</Badge>
            <span className="whitespace-nowrap text-xs tabular-nums text-muted-foreground">
              Master {d.maxLevel ?? "—"}
            </span>
          </span>
        </button>
      );
    });
  }

  return (
    <div
      className="flex min-h-0 flex-1 flex-col"
      style={{ "--acc": `var(${accent})` } as CSSProperties}
    >
      <div className="flex flex-none items-center gap-3 px-4 pb-2 pt-3">
        <h4 className="text-[13px] font-semibold text-muted-foreground">
          {jobName} — Skills
        </h4>
        <Input
          type="search"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter skills…"
          aria-label="Filter skills"
          className="ml-auto h-8 w-[190px] text-[13px]"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">{body}</div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/jobs/skill-list.tsx src/components/features/jobs/__tests__/skill-list.test.tsx
git commit -m "feat(atlas-ui): SkillList pane with search and load states (task-182)"
```

---

### Task 9: `SkillDetail` component

**Files:**
- Create: `src/components/features/jobs/skill-detail.tsx`
- Test: `src/components/features/jobs/__tests__/skill-detail.test.tsx`

**Interfaces:**
- Consumes: `SkillIcon` (Task 5), `buildLevelTable`, `deriveSkillType`, `resolveSkillName`, `formatSkillDescription`, shadcn `Badge`/`Collapsible`/`Table`/`Tooltip` (the `TooltipContent copyable` treatment used for ids elsewhere).
- Produces: `SkillDetail({ def, accent }: { def: SkillDefinitionWithIcon; accent: string })`. **Hosts must render it with `key={def.id}`** so the internal slider level (useState) remounts fresh per skill (design D3). Slider is a native `<input type="range">` with `aria-label="Skill level"`. `maxLevel ≤ 1` or an empty `buildLevelTable` ⇒ "No per-level data." and no slider/table.

- [ ] **Step 1: Write the failing test**

Create `src/components/features/jobs/__tests__/skill-detail.test.tsx`:

```tsx
import { render, screen, fireEvent, within } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import type { SkillEffect } from "@/services/api/skills.service";
import { SkillDetail } from "@/components/features/jobs/skill-detail";

function makeDef(
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  const effects = Array.from(
    { length: 20 },
    (_, i) => ({ damage: 105 + 8 * i, MPConsume: 4 + i }) as SkillEffect,
  );
  return {
    id: 1001004,
    name: "Power Strike",
    description: "Strikes a single enemy with a concentrated, powerful blow.",
    action: true,
    element: "",
    animationTime: 600,
    maxLevel: 20,
    effects,
    iconUrl: "http://assets.test/skills/1001004/icon",
    ...over,
  };
}

describe("SkillDetail", () => {
  it("renders header, badges, and description", () => {
    render(<SkillDetail def={makeDef()} accent="--c-warrior" />);
    expect(screen.getByText("Power Strike")).toBeInTheDocument();
    expect(screen.getByText("ID 1001004")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Master Lv 20")).toBeInTheDocument();
    expect(
      screen.getByText(/Strikes a single enemy/),
    ).toBeInTheDocument();
  });

  it("drives the stat readout and table highlight from the slider", () => {
    render(<SkillDetail def={makeDef()} accent="--c-warrior" />);
    expect(screen.getByText("Level 1")).toBeInTheDocument();
    const slider = screen.getByLabelText("Skill level");
    fireEvent.change(slider, { target: { value: "5" } });
    expect(screen.getByText("Level 5")).toBeInTheDocument();
    // readout shows the level-5 row: damage 105 + 8*4 = 137
    expect(screen.getByTestId("stat-readout")).toHaveTextContent("137");
    // the open all-levels table highlights row 5
    const table = screen.getByRole("table");
    const rows = within(table).getAllByRole("row");
    expect(rows[5]).toHaveAttribute("data-selected", "true"); // rows[0] = header
    expect(rows[1]).toHaveAttribute("data-selected", "false");
  });

  it("shows 'No per-level data.' for maxLevel <= 1", () => {
    render(
      <SkillDetail
        def={makeDef({ maxLevel: 1, effects: [] })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("No per-level data.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Skill level")).not.toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("shows 'No per-level data.' when the level table is empty", () => {
    render(
      <SkillDetail
        def={makeDef({ effects: [] })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("No per-level data.")).toBeInTheDocument();
  });

  it("resets the slider when remounted for another skill (key pattern)", () => {
    const { rerender } = render(
      <SkillDetail key={1001004} def={makeDef()} accent="--c-warrior" />,
    );
    fireEvent.change(screen.getByLabelText("Skill level"), {
      target: { value: "9" },
    });
    expect(screen.getByText("Level 9")).toBeInTheDocument();
    rerender(
      <SkillDetail
        key={1001005}
        def={makeDef({ id: 1001005, name: "Slash Blast" })}
        accent="--c-warrior"
      />,
    );
    expect(screen.getByText("Level 1")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: FAIL — module `skill-detail` does not exist.

- [ ] **Step 3: Implement the component**

Create `src/components/features/jobs/skill-detail.tsx`:

```tsx
import { useMemo, useState } from "react";
import type { CSSProperties } from "react";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { buildLevelTable } from "@/lib/skills/level-table";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import { formatSkillDescription } from "@/lib/skills/format-skill-description";
import { SkillIcon } from "@/components/features/jobs/skill-icon";
import { cn } from "@/lib/utils";

interface SkillDetailProps {
  def: SkillDefinitionWithIcon;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
}

// Hosts render this with key={def.id} so the slider level resets per skill
// (design D3) — no effect juggling.
export function SkillDetail({ def, accent }: SkillDetailProps) {
  const [level, setLevel] = useState(1);
  const name = resolveSkillName(def.id, def.name);
  const type = deriveSkillType(def);
  const formatted = formatSkillDescription(def.description);
  const table = useMemo(() => buildLevelTable(def.effects), [def.effects]);
  const maxLevel = def.maxLevel ?? 0;
  const hasLevels = maxLevel > 1 && table.rows.length > 0;
  const row = table.rows[level - 1];
  const statColumns = table.columns.filter((c) => c.key !== "level");

  return (
    <div
      className="space-y-3 px-4 pb-5 pt-1"
      style={{ "--acc": `var(${accent})` } as CSSProperties}
    >
      <div className="flex items-start gap-3">
        <SkillIcon def={def} name={name} />
        <div>
          <h5 className="text-base font-semibold">{name}</h5>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span
                  tabIndex={0}
                  className="cursor-help rounded font-mono text-[11.5px] text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  ID {def.id}
                </span>
              </TooltipTrigger>
              <TooltipContent copyable>
                <p>{def.id}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Badge variant="secondary">{type}</Badge>
        <Badge variant="secondary">Master Lv {def.maxLevel ?? "—"}</Badge>
        {def.element ? (
          <Badge variant="outline">Element {def.element}</Badge>
        ) : null}
      </div>

      {formatted.lines.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No description available.
        </p>
      ) : (
        <div className="max-w-[60ch] space-y-1 text-sm">
          {formatted.lines.map((line, i) => (
            <p key={i}>
              {line.map((seg, j) => (
                <span key={j}>{seg.text}</span>
              ))}
            </p>
          ))}
        </div>
      )}

      {hasLevels ? (
        <>
          <div className="rounded-lg border bg-[hsl(var(--sidebar-background))] p-3">
            <div className="mb-1.5 flex items-baseline gap-2">
              <span className="text-[13px] font-semibold">Level {level}</span>
              <span className="text-xs tabular-nums text-muted-foreground">
                / {maxLevel}
              </span>
            </div>
            <input
              type="range"
              min={1}
              max={maxLevel}
              value={level}
              onChange={(e) => setLevel(Number(e.target.value))}
              aria-label="Skill level"
              className="mb-2.5 mt-0.5 w-full accent-[hsl(var(--acc))]"
            />
            {row ? (
              <div
                data-testid="stat-readout"
                className="grid grid-cols-2 gap-x-3.5 gap-y-1.5"
              >
                {statColumns.map((c) => (
                  <div
                    key={c.key}
                    className="flex justify-between gap-2.5 text-[13px]"
                  >
                    <span className="text-muted-foreground">{c.label}</span>
                    <span className="font-medium tabular-nums">
                      {row[c.key] ?? ""}
                    </span>
                  </div>
                ))}
              </div>
            ) : null}
          </div>

          <Collapsible defaultOpen>
            <CollapsibleTrigger className="group flex cursor-pointer items-center gap-1.5 rounded text-[12.5px] font-medium text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <ChevronRight className="h-3.5 w-3.5 transition-transform group-data-[state=open]:rotate-90" />
              All {table.rows.length} levels
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-2 overflow-x-auto rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      {table.columns.map((c) => (
                        <TableHead key={c.key}>{c.label}</TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {table.rows.map((r, i) => (
                      <TableRow
                        key={i}
                        data-selected={i === level - 1}
                        className={cn(
                          i === level - 1 && "bg-[hsl(var(--acc)/0.14)]",
                        )}
                      >
                        {table.columns.map((c) => (
                          <TableCell key={c.key} className="tabular-nums">
                            {r[c.key] ?? ""}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CollapsibleContent>
          </Collapsible>
        </>
      ) : (
        <p className="text-sm text-muted-foreground">No per-level data.</p>
      )}
    </div>
  );
}
```

Note on the highlight test: shadcn `TableRow` forwards unknown props to `<tr>`, so `data-selected` lands on the DOM row. `rows[5]` is the level-5 body row because `rows[0]` is the header row.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-ui && npm run test -- src/components/features/jobs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/components/features/jobs/skill-detail.tsx src/components/features/jobs/__tests__/skill-detail.test.tsx
git commit -m "feat(atlas-ui): SkillDetail panel with level slider and all-levels table (task-182)"
```

---

### Task 10: `JobsPage` rewrite, routing, and `JobDetailPage` removal

**Files:**
- Rewrite: `src/pages/JobsPage.tsx`
- Rewrite: `src/pages/__tests__/JobsPage.test.tsx`
- Modify: `src/App.tsx` (lines 71–75 lazy imports; line 277 route)
- Delete: `src/pages/JobDetailPage.tsx`, `src/pages/__tests__/JobDetailPage.test.tsx`
- Modify: `src/lib/breadcrumbs/__tests__/routes.test.ts` (append jobs assertions; `src/lib/breadcrumbs/routes.ts` itself needs **no change** — its `/jobs` + `/jobs/[id]` entries reference `getJobNameById`, not the deleted page)

**Interfaces:**
- Consumes: everything from Tasks 1–9 plus `useTenant`, `useJobSkills(tenant, jobId)` (→ `UseQueryResult<number[]>`), `useJobSkillDefinitions(tenant, ids)` (→ `{ definitions, isLoading, isError }`), `getJobNameById` is NOT needed (names come from `JOB_GRAPH`), shadcn `Sheet`.
- Produces: `JobsPage()` — the only router/React-Query-aware component. URL contract (design D1): job = route param, skill = `?skill=`; selections push, normalizations replace.

- [ ] **Step 1: Rewrite the page test**

Replace the entire contents of `src/pages/__tests__/JobsPage.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import type { SkillEffect } from "@/services/api/skills.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => useTenantMock(),
}));

const useJobSkillsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobSkills", () => ({
  useJobSkills: (...args: unknown[]) => useJobSkillsMock(...args),
}));

const useJobSkillDefinitionsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobSkillDefinitions", () => ({
  useJobSkillDefinitions: (...args: unknown[]) =>
    useJobSkillDefinitionsMock(...args),
}));

const useMediaQueryMock = vi.fn();
vi.mock("@/hooks/use-media-query", () => ({
  useMediaQuery: () => useMediaQueryMock(),
}));

import { JobsPage } from "@/pages/JobsPage";

const tenant = (major: number) =>
  ({
    id: "t1",
    attributes: { region: "GMS", majorVersion: major, minorVersion: 1 },
  }) as unknown as Tenant;

function def(
  id: number,
  name: string,
  over?: Partial<SkillDefinitionWithIcon>,
): SkillDefinitionWithIcon {
  return {
    id,
    name,
    description: "",
    action: true,
    element: "",
    animationTime: 0,
    maxLevel: 20,
    effects: Array.from(
      { length: 20 },
      (_, i) => ({ damage: 10 + i }) as SkillEffect,
    ),
    iconUrl: `http://assets.test/skills/${id}/icon`,
    ...over,
  };
}

const warriorDefs = [def(1001004, "Power Strike"), def(1001005, "Slash Blast")];
const fighterDefs = [def(1101007, "Power Guard"), def(1101006, "Rage")];

function LocationProbe() {
  const location = useLocation();
  return (
    <div data-testid="location">{location.pathname + location.search}</div>
  );
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/jobs" element={<JobsPage />} />
        <Route path="/jobs/:jobId" element={<JobsPage />} />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
  useMediaQueryMock.mockReturnValue(true); // wide by default in these tests
  useJobSkillsMock.mockImplementation((_t: unknown, jobId: number) => ({
    data:
      jobId === 110
        ? fighterDefs.map((d) => d.id)
        : warriorDefs.map((d) => d.id),
    isLoading: false,
    isError: false,
  }));
  useJobSkillDefinitionsMock.mockImplementation(
    (_t: unknown, ids: number[]) => ({
      definitions: [...warriorDefs, ...fighterDefs].filter((d) =>
        ids.includes(d.id),
      ),
      isLoading: false,
      isError: false,
    }),
  );
});

describe("JobsPage", () => {
  it("shows the select-a-tenant card when no tenant is active", () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    renderAt("/jobs");
    expect(screen.getByText(/select a tenant/i)).toBeInTheDocument();
    expect(screen.queryByText("Branches")).not.toBeInTheDocument();
  });

  it("defaults /jobs to the Warrior entry with no skill selected", () => {
    renderAt("/jobs");
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
    expect(useJobSkillsMock).toHaveBeenCalledWith(expect.anything(), 100);
    expect(
      screen.getByText("Select a skill to inspect it"),
    ).toBeInTheDocument();
  });

  it("deep-links /jobs/110 to Fighter in the Warrior branch", () => {
    renderAt("/jobs/110");
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toHaveAttribute("aria-pressed", "true"); // rail highlights the branch
    expect(screen.getByRole("button", { name: /Fighter/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("Fighter — Skills")).toBeInTheDocument();
  });

  it("deep-links ?skill= to an open detail panel once definitions load", () => {
    renderAt("/jobs/110?skill=1101007");
    expect(screen.getByText("ID 1101007")).toBeInTheDocument();
    expect(screen.getByLabelText("Skill level")).toBeInTheDocument();
  });

  it("selecting a job pushes /jobs/:id and clears the skill selection", () => {
    renderAt("/jobs/100?skill=1001004");
    fireEvent.click(screen.getByRole("button", { name: /Fighter/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
    expect(screen.getByTestId("location")).not.toHaveTextContent("skill=");
  });

  it("selecting a skill writes ?skill= to the URL", () => {
    renderAt("/jobs/110");
    fireEvent.click(screen.getByRole("button", { name: /Power Guard/ }));
    expect(screen.getByTestId("location")).toHaveTextContent(
      "/jobs/110?skill=1101007",
    );
  });

  it("normalizes an unknown jobId to /jobs with the default selection", async () => {
    renderAt("/jobs/99999");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
    expect(screen.getByText("Warrior — Skills")).toBeInTheDocument();
  });

  it("normalizes a version-hidden jobId (Evan on v83) to /jobs", async () => {
    renderAt("/jobs/2200");
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent(/^\/jobs$/),
    );
  });

  it("strips a ?skill= that does not resolve for the job", async () => {
    renderAt("/jobs/110?skill=424242");
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });

  it("gates rail entries by tenant version (GMS v12)", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(12) });
    renderAt("/jobs");
    // "Warrior 10" is the rail entry; the flow chip reads "Warrior 1st"
    expect(
      screen.getByRole("button", { name: /Warrior 10/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^GM 2$/ })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Pirate/ }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Cygnus Knights")).not.toBeInTheDocument();
    expect(screen.queryByText("Legends")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Maple Leaf Brigadier/ }),
    ).not.toBeInTheDocument();
  });

  it("renders skill-list error state from the hook", () => {
    useJobSkillsMock.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    });
    useJobSkillDefinitionsMock.mockReturnValue({
      definitions: [],
      isLoading: false,
      isError: false,
    });
    renderAt("/jobs/100");
    expect(
      screen.getByText("Failed to load this job's skills."),
    ).toBeInTheDocument();
  });

  it("below 1150px renders the detail in a dismissible sheet that clears ?skill=", async () => {
    useMediaQueryMock.mockReturnValue(false); // narrow
    renderAt("/jobs/110?skill=1101007");
    // detail content is in the sheet (dialog), not a third column
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("ID 1101007");
    fireEvent.click(screen.getByRole("button", { name: /close/i }));
    await waitFor(() =>
      expect(screen.getByTestId("location")).not.toHaveTextContent("skill="),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/jobs/110");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/pages/__tests__/JobsPage.test.tsx`
Expected: FAIL — the old tree page doesn't render "Branches", rail buttons, etc.

- [ ] **Step 3: Rewrite the page**

Replace the entire contents of `src/pages/JobsPage.tsx`:

```tsx
import { useEffect, useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Briefcase } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useJobSkills } from "@/lib/hooks/api/useJobSkills";
import { useJobSkillDefinitions } from "@/lib/hooks/api/useJobSkillDefinitions";
import { useMediaQuery } from "@/hooks/use-media-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { JOB_GRAPH, floorOf } from "@/lib/jobs/job-advancement-tree";
import {
  branchEntryOf,
  visibleRailGroups,
} from "@/components/features/jobs/rail-groups";
import { BranchRail } from "@/components/features/jobs/branch-rail";
import { AdvancementFlow } from "@/components/features/jobs/advancement-flow";
import {
  SkillList,
  type SkillListState,
} from "@/components/features/jobs/skill-list";
import { SkillDetail } from "@/components/features/jobs/skill-detail";
import { cn } from "@/lib/utils";

export function JobsPage() {
  const { jobId: jobIdParam } = useParams<{ jobId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const isWide = useMediaQuery("(min-width: 1150px)");

  const major = activeTenant?.attributes.majorVersion ?? 0;
  const groups = useMemo(() => visibleRailGroups(major), [major]);
  const defaultJobId = groups[0]?.entries[0]?.id ?? 100;

  const parsedJobId = jobIdParam !== undefined ? Number(jobIdParam) : null;
  const jobIdValid =
    parsedJobId !== null &&
    Number.isInteger(parsedJobId) &&
    JOB_GRAPH[parsedJobId] !== undefined &&
    floorOf(parsedJobId) <= major;
  const jobId = jobIdValid ? parsedJobId : defaultJobId;

  // FR-1.2 / FR-7.3: unknown or version-hidden jobId (incl. after a tenant
  // switch) normalizes to /jobs with replace, so Back doesn't bounce.
  useEffect(() => {
    if (activeTenant && parsedJobId !== null && !jobIdValid) {
      navigate("/jobs", { replace: true });
    }
  }, [activeTenant, parsedJobId, jobIdValid, navigate]);

  const entry = branchEntryOf(jobId);
  const jobName = JOB_GRAPH[jobId]?.name ?? `Job ${jobId}`;

  const skillsQuery = useJobSkills(activeTenant, jobId);
  const skillIds = useMemo(
    () => skillsQuery.data ?? [],
    [skillsQuery.data],
  );
  const {
    definitions,
    isLoading: defsLoading,
    isError: defsError,
  } = useJobSkillDefinitions(activeTenant, skillIds);

  const loading =
    skillsQuery.isLoading || (skillIds.length > 0 && defsLoading);

  const skillParam = searchParams.get("skill");
  const selectedSkillId = skillParam !== null ? Number(skillParam) : null;
  const selectedDef =
    definitions.find((d) => d.id === selectedSkillId) ?? null;

  // D1: a ?skill= that never resolves for this job is stripped (replace) once
  // definitions settle.
  useEffect(() => {
    if (activeTenant && skillParam !== null && !loading && !selectedDef) {
      setSearchParams({}, { replace: true });
    }
  }, [activeTenant, skillParam, loading, selectedDef, setSearchParams]);

  const state: SkillListState = loading
    ? "loading"
    : skillsQuery.isError
      ? "error"
      : skillIds.length === 0
        ? "empty"
        : definitions.length === 0 && defsError
          ? "defs-failed"
          : "ready";

  const selectJob = (id: number) => navigate(`/jobs/${id}`); // push; drops ?skill=
  const selectSkill = (id: number) => setSearchParams({ skill: String(id) });
  const clearSkill = () => setSearchParams({});

  const detail = selectedDef ? (
    <SkillDetail
      key={selectedDef.id}
      def={selectedDef}
      accent={entry.accent}
    />
  ) : null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 p-10 pb-6">
      <div className="flex flex-none items-center gap-2">
        <Briefcase className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Jobs</h2>
      </div>

      {!activeTenant ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Select a tenant to browse its jobs and skills.
          </CardContent>
        </Card>
      ) : (
        <div
          className={cn(
            "grid min-h-0 flex-1 gap-3.5",
            isWide
              ? "grid-cols-[200px_minmax(340px,1fr)_minmax(480px,42%)]"
              : "grid-cols-[200px_minmax(0,1fr)]",
          )}
        >
          <BranchRail
            groups={groups}
            selectedEntryId={entry.id}
            onSelect={selectJob}
          />

          <Card className="flex min-h-0 flex-col">
            <CardHeader className="pb-1">
              <CardTitle className="text-[15px]">Advancement</CardTitle>
            </CardHeader>
            <div className="flex-none border-b px-4 pb-3.5">
              <AdvancementFlow
                entryId={entry.id}
                major={major}
                selectedJobId={jobId}
                accent={entry.accent}
                onSelect={selectJob}
              />
            </div>
            <SkillList
              key={jobId}
              jobName={jobName}
              defs={definitions}
              state={state}
              selectedSkillId={selectedSkillId}
              accent={entry.accent}
              onSelect={selectSkill}
            />
          </Card>

          {isWide ? (
            <Card className="flex min-h-0 flex-col overflow-hidden">
              <CardHeader className="pb-1">
                <CardTitle className="text-[15px]">Skill Detail</CardTitle>
              </CardHeader>
              <div className="min-h-0 flex-1 overflow-y-auto">
                {detail ?? (
                  <div className="px-6 py-14 text-center text-muted-foreground">
                    Select a skill to inspect it
                  </div>
                )}
              </div>
            </Card>
          ) : (
            <Sheet
              open={selectedDef !== null}
              onOpenChange={(open) => {
                if (!open) clearSkill();
              }}
            >
              <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
                <SheetHeader>
                  <SheetTitle className="sr-only">Skill detail</SheetTitle>
                </SheetHeader>
                {detail}
              </SheetContent>
            </Sheet>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Update routing and delete the old page**

In `src/App.tsx`:
1. Delete the `JobDetailPage` lazy-import block (lines 74–76):
```tsx
const JobDetailPage = lazy(() =>
  import("@/pages/JobDetailPage").then((m) => ({ default: m.JobDetailPage })),
);
```
2. Change the route at line 277 from `element={<JobDetailPage />}` to:
```tsx
                    <Route path="/jobs/:jobId" element={<JobsPage />} />
```

Then delete the old page and its test:

```bash
git rm src/pages/JobDetailPage.tsx src/pages/__tests__/JobDetailPage.test.tsx
```

Verify no references remain:

```bash
grep -rn "JobDetailPage" src/
```
Expected: no output.

- [ ] **Step 5: Pin the breadcrumb resolution**

Append to `src/lib/breadcrumbs/__tests__/routes.test.ts`, inside the top-level `describe("Route Configuration", ...)` block:

```ts
  describe("Jobs routes (task-182 explorer)", () => {
    it("resolves /jobs and /jobs/[id] with the job-name label resolver", () => {
      expect(findRouteConfig("/jobs")?.label).toBe("Jobs");
      const detail = findRouteConfig("/jobs/110");
      expect(detail).toBeTruthy();
      expect(detail?.parent).toBe("/jobs");
      expect(detail?.labelResolver?.({ id: "110" })).toBe("Fighter");
    });
  });
```

(If `labelResolver`'s param type rejects a plain object literal, match the signature used in `routes.ts` — it is `(params: Record<string, string>) => string`.)

- [ ] **Step 6: Run the full test suite**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS — including the rewritten JobsPage tests, all Task 1–9 tests, breadcrumbs, and no references to the deleted page.

- [ ] **Step 7: Commit**

```bash
git add src/pages/JobsPage.tsx src/pages/__tests__/JobsPage.test.tsx src/App.tsx src/lib/breadcrumbs/__tests__/routes.test.ts
git commit -m "feat(atlas-ui): unified jobs explorer replaces JobsPage tree + JobDetailPage (task-182)"
```

(`git rm` from Step 4 stages the deletions; they land in this commit.)

---

### Task 11: Full verification gates

**Files:** none (verification only; fix-forward anything found and amend the responsible commit or add a `fix(atlas-ui):` commit).

- [ ] **Step 1: Unit tests**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS, zero failures.

- [ ] **Step 2: Lint**

Run: `cd services/atlas-ui && npm run lint`
Expected: exit 0. (If the environment needs Node via nvm: `source ~/.nvm/nvm.sh && nvm use 22` first — see project memory `reference_atlas_ui_npm_nvm_and_lint_baseline`.)

- [ ] **Step 3: Build (type-checks the app AND new-style tests)**

Run: `cd services/atlas-ui && npm run build`
Expected: `tsc -b` + `vite build` succeed. Watch for strict-TS errors in the new components (e.g. `CSSProperties` casts for `--acc`).

- [ ] **Step 4: Repo-wide lint guard**

Run from the repo root: `tools/lint.sh --check`
Expected: exit 0 (Prettier + ESLint over atlas-ui; Go modules untouched by this task).

- [ ] **Step 5: Verify acceptance criteria that tests can't cover, against the running dev server**

Run: `cd services/atlas-ui && npm run dev` and check manually (or note as verified-by-test):
- `/jobs` renders the three-pane layout in both themes (accent tokens present in both `:root` and `.dark`).
- Evan (v84 tenant) flow scrolls horizontally; grid stays centered when narrower than the pane.
- Browser Back walks selection history (job selections push).

- [ ] **Step 6: Final commit if anything was fixed**

```bash
git status --short   # must be clean (or commit fixes with a fix(atlas-ui) message)
```

---

## Self-Review Notes

- **Spec coverage:** FR-1.1–1.5 (Task 10), FR-2.1–2.4 (Tasks 1–2), FR-3.1–3.5 (Tasks 3, 6, 10), FR-4.1–4.5 (Tasks 2, 7), FR-5.1–5.5 (Tasks 5, 8, 10), FR-6.1–6.6 (Tasks 4, 9, 10), FR-7.1–7.3 (Tasks 3, 6–10; tenant-switch normalization is the same effect as FR-1.2, keyed on `major` via `jobIdValid`). Design D1–D5 all mapped.
- **Design §6 discrepancy resolved:** design §6 writes "GM `[[900,910]]` from entry 900"; the normative D2 text ("every root-to-leaf path **below** the entry") and the mock's `chainsFrom` both exclude the entry, so `advancementChains(900, …) === [[910]]` and `[900, 910]` appears among *Beginner's* chains. Tasks 2/7 follow D2.
- **Type consistency:** `accent` is always the token *name* (`"--c-warrior"`); every consumer wraps it as `` `var(${accent})` `` into a scoped `--acc`. `SkillListState` defined once in `skill-list.tsx`, imported by the page. `VisibleRailGroup` defined once in `rail-groups.ts`.
- **Test-environment defaults:** the global `matchMedia` stub returns `matches: false` → narrow layout; the page test explicitly mocks `useMediaQuery` both ways instead.
