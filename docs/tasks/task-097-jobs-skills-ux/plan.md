# Jobs & Skills Browser UX Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix eight UX defects (FR-1…FR-8) on the atlas-ui Jobs browser (`/jobs`) and job detail page (`/jobs/:jobId`) — expand affordance, advancement-tree layout, Beginner-skill names, master-level label, copyable id, in-page scroll, skill-description markup, and corrected version-gate floors — entirely in `services/atlas-ui`.

**Architecture:** Consolidate the two drifting job-hierarchy structures into one parent-edge graph with per-root version floors (`lib/jobs/job-advancement-tree.ts`); push every testable decision into small pure helpers (tree shaping, blank-name fallback, description markup parser) and keep the two pages as thin presentation wiring those helpers to shadcn primitives.

**Tech Stack:** Vite + React 19 + React Router 7, TanStack React Query 5, shadcn/ui (Collapsible, Tooltip, Table, Badge), Tailwind 4, Vitest + Testing Library. No backend changes.

---

## Conventions for every task

- Work inside the worktree: `cd .worktrees/task-097-jobs-skills-ux/services/atlas-ui`.
- Before any `npm`/`npx`: `source ~/.nvm/nvm.sh && nvm use 22`.
- Run a single test file with `npx vitest run <path>`; the full suite with `npm run test`.
- After each task commit, confirm branch: `git -C ../.. branch --show-current` → `task-097-jobs-skills-ux`.
- All paths below are relative to `services/atlas-ui/`.

---

## File Structure

| File | Responsibility |
|---|---|
| `src/lib/jobs/job-advancement-tree.ts` (new) | Single source of truth: `JOB_GRAPH` (parent edges + names), `BRANCH_FLOORS` (per-root floors), helpers `rootOf`/`floorOf`/`visibleRoots`/`childrenOf`/`JOB_ROOTS`/`jobTreePath`. (FR-2, FR-8) |
| `src/lib/skills/beginner-skill-names.ts` (new) | Curated id→name map (1000–1012) + `resolveSkillName(id, serverName)` blank-name fallback. (FR-3) |
| `src/lib/skills/format-skill-description.ts` (new) | Pure MapleStory-markup parser → `FormattedDescription` (lines of styled segments + captured master-level header). (FR-7) |
| `src/pages/JobsPage.tsx` (modify) | Recursive indented advancement tree, chevron expand affordance, page-local scroll. (FR-1, FR-2, FR-6) |
| `src/pages/JobDetailPage.tsx` (modify) | Blank-name fallback, `Master Lv` label + tooltip, copyable job id, page-local scroll, formatted description. (FR-3, FR-4, FR-5, FR-6, FR-7) |
| `src/components/features/characters/SkillsSection.tsx` (modify) | Re-point `jobTreePath` import to the new module. |
| Removed: `src/lib/utils/job-tree.ts`, `src/lib/jobs-hierarchy.ts` (+ their `__tests__`) | Consolidated into `job-advancement-tree.ts`. |

---

## Task 1: Consolidated job advancement tree (`lib/jobs/job-advancement-tree.ts`)

Builds the single source of truth (FR-2.3, FR-8). `JOB_GRAPH` is ported **verbatim**
from the existing `lib/utils/job-tree.ts` `JOB_TREE` (ids/names already sourced to
`libs/atlas-constants/job/constants.go`). Floors corrected per FR-8.1/8.2.

**Files:**
- Create: `src/lib/jobs/job-advancement-tree.ts`
- Test: `src/lib/jobs/__tests__/job-advancement-tree.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/jobs/__tests__/job-advancement-tree.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  JOB_GRAPH, BRANCH_FLOORS, JOB_ROOTS,
  childrenOf, rootOf, floorOf, visibleRoots, jobTreePath,
} from "@/lib/jobs/job-advancement-tree";

describe("job-advancement-tree", () => {
  it("exposes the seven branch roots ascending", () => {
    expect(JOB_ROOTS).toEqual([0, 800, 900, 910, 1000, 2000, 2001]);
  });

  it("derives children from parent edges, ascending", () => {
    expect(childrenOf(0)).toEqual([100, 200, 300, 400, 500]);
    expect(childrenOf(100)).toEqual([110, 120, 130]);
    expect(childrenOf(112)).toEqual([]); // 4th job is a leaf
  });

  it("walks to the branch root", () => {
    expect(rootOf(112)).toBe(0);    // Hero -> Beginner
    expect(rootOf(1112)).toBe(1000); // Dawn Warrior 4 -> Noblesse
    expect(rootOf(2112)).toBe(2000); // Aran 4 -> Legend
    expect(rootOf(2218)).toBe(2001); // Evan 10 -> Evan root
    expect(rootOf(99999)).toBe(99999); // unknown id returns itself
  });

  it("uses the corrected per-branch floors, inherited from the root", () => {
    expect(BRANCH_FLOORS).toEqual({ 0: 83, 800: 83, 900: 83, 910: 83, 1000: 83, 2000: 80, 2001: 84 });
    expect(floorOf(112)).toBe(83);  // Adventurer
    expect(floorOf(1112)).toBe(83); // Cygnus corrected 92 -> 83
    expect(floorOf(2112)).toBe(80); // Aran corrected 88 -> 80
    expect(floorOf(2218)).toBe(84); // Evan
  });

  it("shows Cygnus + Aran on v83 and hides Evan until v84", () => {
    const r83 = visibleRoots(83);
    expect(r83).toContain(0);
    expect(r83).toContain(1000);     // Cygnus visible on v83
    expect(r83).toContain(2000);     // Aran visible on v83
    expect(r83).not.toContain(2001); // Evan hidden on v83
    expect(visibleRoots(84)).toContain(2001); // Evan visible on v84
  });

  it("jobTreePath returns root->node inclusive", () => {
    expect(jobTreePath(112).map((j) => j.name)).toEqual([
      "Beginner", "Warrior", "Fighter", "Crusader", "Hero",
    ]);
    expect(jobTreePath(0).map((j) => j.name)).toEqual(["Beginner"]);
    expect(jobTreePath(99999)).toEqual([]);
  });

  it("has no orphan parent references", () => {
    for (const e of Object.values(JOB_GRAPH)) {
      if (e.parent !== null) {
        expect(JOB_GRAPH[e.parent]).toBeDefined();
      }
    }
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/lib/jobs/__tests__/job-advancement-tree.test.ts`
Expected: FAIL — cannot resolve `@/lib/jobs/job-advancement-tree`.

- [ ] **Step 3: Write the module**

Create `src/lib/jobs/job-advancement-tree.ts`:

```ts
export interface JobEntry {
  id: number;
  name: string;
  parent: number | null;
}

// Structural source of truth for the job advancement graph.
// Ported verbatim from the former lib/utils/job-tree.ts JOB_TREE, whose ids/names
// derive from libs/atlas-constants/job/constants.go::Jobs (v83 conventions).
// Order per branch: branch leader (parent: null) -> 1st -> 2nd -> 3rd -> 4th.
export const JOB_GRAPH: Record<number, JobEntry> = {
  // Beginner branch
  0: { id: 0, name: "Beginner", parent: null },
  // Warrior
  100: { id: 100, name: "Warrior", parent: 0 },
  110: { id: 110, name: "Fighter", parent: 100 },
  111: { id: 111, name: "Crusader", parent: 110 },
  112: { id: 112, name: "Hero", parent: 111 },
  120: { id: 120, name: "Page", parent: 100 },
  121: { id: 121, name: "White Knight", parent: 120 },
  122: { id: 122, name: "Paladin", parent: 121 },
  130: { id: 130, name: "Spearman", parent: 100 },
  131: { id: 131, name: "Dragon Knight", parent: 130 },
  132: { id: 132, name: "Dark Knight", parent: 131 },
  // Magician
  200: { id: 200, name: "Magician", parent: 0 },
  210: { id: 210, name: "Wizard (F/P)", parent: 200 },
  211: { id: 211, name: "Mage (F/P)", parent: 210 },
  212: { id: 212, name: "Arch Mage (F/P)", parent: 211 },
  220: { id: 220, name: "Wizard (I/L)", parent: 200 },
  221: { id: 221, name: "Mage (I/L)", parent: 220 },
  222: { id: 222, name: "Arch Mage (I/L)", parent: 221 },
  230: { id: 230, name: "Cleric", parent: 200 },
  231: { id: 231, name: "Priest", parent: 230 },
  232: { id: 232, name: "Bishop", parent: 231 },
  // Bowman
  300: { id: 300, name: "Bowman", parent: 0 },
  310: { id: 310, name: "Hunter", parent: 300 },
  311: { id: 311, name: "Ranger", parent: 310 },
  312: { id: 312, name: "Bowmaster", parent: 311 },
  320: { id: 320, name: "Crossbowman", parent: 300 },
  321: { id: 321, name: "Sniper", parent: 320 },
  322: { id: 322, name: "Marksman", parent: 321 },
  // Thief
  400: { id: 400, name: "Rogue", parent: 0 },
  410: { id: 410, name: "Assassin", parent: 400 },
  411: { id: 411, name: "Hermit", parent: 410 },
  412: { id: 412, name: "Night Lord", parent: 411 },
  420: { id: 420, name: "Bandit", parent: 400 },
  421: { id: 421, name: "Chief Bandit", parent: 420 },
  422: { id: 422, name: "Shadower", parent: 421 },
  // Pirate
  500: { id: 500, name: "Pirate", parent: 0 },
  510: { id: 510, name: "Brawler", parent: 500 },
  511: { id: 511, name: "Marauder", parent: 510 },
  512: { id: 512, name: "Buccaneer", parent: 511 },
  520: { id: 520, name: "Gunslinger", parent: 500 },
  521: { id: 521, name: "Outlaw", parent: 520 },
  522: { id: 522, name: "Corsair", parent: 521 },
  // Special / Admin (standalone roots)
  800: { id: 800, name: "Maple Leaf Brigadier", parent: null },
  900: { id: 900, name: "GM", parent: null },
  910: { id: 910, name: "Super GM", parent: null },
  // Noblesse / Cygnus Knights
  1000: { id: 1000, name: "Noblesse", parent: null },
  1100: { id: 1100, name: "Dawn Warrior 1", parent: 1000 },
  1110: { id: 1110, name: "Dawn Warrior 2", parent: 1100 },
  1111: { id: 1111, name: "Dawn Warrior 3", parent: 1110 },
  1112: { id: 1112, name: "Dawn Warrior 4", parent: 1111 },
  1200: { id: 1200, name: "Blaze Wizard 1", parent: 1000 },
  1210: { id: 1210, name: "Blaze Wizard 2", parent: 1200 },
  1211: { id: 1211, name: "Blaze Wizard 3", parent: 1210 },
  1212: { id: 1212, name: "Blaze Wizard 4", parent: 1211 },
  1300: { id: 1300, name: "Wind Archer 1", parent: 1000 },
  1310: { id: 1310, name: "Wind Archer 2", parent: 1300 },
  1311: { id: 1311, name: "Wind Archer 3", parent: 1310 },
  1312: { id: 1312, name: "Wind Archer 4", parent: 1311 },
  1400: { id: 1400, name: "Night Walker 1", parent: 1000 },
  1410: { id: 1410, name: "Night Walker 2", parent: 1400 },
  1411: { id: 1411, name: "Night Walker 3", parent: 1410 },
  1412: { id: 1412, name: "Night Walker 4", parent: 1411 },
  1500: { id: 1500, name: "Thunder Breaker 1", parent: 1000 },
  1510: { id: 1510, name: "Thunder Breaker 2", parent: 1500 },
  1511: { id: 1511, name: "Thunder Breaker 3", parent: 1510 },
  1512: { id: 1512, name: "Thunder Breaker 4", parent: 1511 },
  // Legend / Aran
  2000: { id: 2000, name: "Legend", parent: null },
  2100: { id: 2100, name: "Aran 1", parent: 2000 },
  2110: { id: 2110, name: "Aran 2", parent: 2100 },
  2111: { id: 2111, name: "Aran 3", parent: 2110 },
  2112: { id: 2112, name: "Aran 4", parent: 2111 },
  // Evan (separate root per job/constants.go)
  2001: { id: 2001, name: "Evan", parent: null },
  2200: { id: 2200, name: "Evan 1", parent: 2001 },
  2210: { id: 2210, name: "Evan 2", parent: 2200 },
  2211: { id: 2211, name: "Evan 3", parent: 2210 },
  2212: { id: 2212, name: "Evan 4", parent: 2211 },
  2213: { id: 2213, name: "Evan 5", parent: 2212 },
  2214: { id: 2214, name: "Evan 6", parent: 2213 },
  2215: { id: 2215, name: "Evan 7", parent: 2214 },
  2216: { id: 2216, name: "Evan 8", parent: 2215 },
  2217: { id: 2217, name: "Evan 9", parent: 2216 },
  2218: { id: 2218, name: "Evan 10", parent: 2217 },
};

// Version-introduction floor per branch ROOT id. A node inherits its root's floor
// (gating is per-branch, never per-node). This is a DISPLAY-curation choice — the
// atlas-data /jobs/{id}/skills endpoint is NOT version-gated (live probe 2026-06-14)
// — so floors hide classes that did not exist in the live game at the tenant's
// version; they are not a data-availability gate.
//   0    Adventurers          — v83 baseline
//   800  Maple Leaf Brigadier  — v83 baseline (special)
//   900  GM                    — admin, always present
//   910  Super GM              — admin, always present
//   1000 Cygnus (Noblesse)     — reference_maplestory_version_timeline: KoC exist in v83
//   2000 Legend (Aran)         — product owner (PRD FR-8.1): Aran introduced v80
//   2001 Evan                  — reference_maplestory_version_timeline: Evan introduced v84
export const BRANCH_FLOORS: Record<number, number> = {
  0: 83, 800: 83, 900: 83, 910: 83, 1000: 83, 2000: 80, 2001: 84,
};

/** Branch root ids (parent === null), ascending. */
export const JOB_ROOTS: number[] = Object.values(JOB_GRAPH)
  .filter((e) => e.parent === null)
  .map((e) => e.id)
  .sort((a, b) => a - b);

/** Direct children of a node, ascending by id. */
export function childrenOf(id: number): number[] {
  return Object.values(JOB_GRAPH)
    .filter((e) => e.parent === id)
    .map((e) => e.id)
    .sort((a, b) => a - b);
}

/** Walk parent edges to the branch root. Returns the id itself if it is a root or unknown. */
export function rootOf(id: number): number {
  let cur = JOB_GRAPH[id];
  if (!cur) return id;
  while (cur.parent != null) {
    const next = JOB_GRAPH[cur.parent];
    if (!next) break;
    cur = next;
  }
  return cur.id;
}

/** Version floor for a node = its root's BRANCH_FLOORS entry (0 = fail-open if unfloored). */
export function floorOf(id: number): number {
  return BRANCH_FLOORS[rootOf(id)] ?? 0;
}

/** Root ids visible at the given tenant major version, ascending. */
export function visibleRoots(major: number): number[] {
  return JOB_ROOTS.filter((r) => floorOf(r) <= major);
}

/** Root -> node advancement path (inclusive), for breadcrumbs. */
export function jobTreePath(jobId: number): JobEntry[] {
  const path: JobEntry[] = [];
  let cur: JobEntry | undefined = JOB_GRAPH[jobId];
  while (cur) {
    path.unshift(cur);
    cur = cur.parent != null ? JOB_GRAPH[cur.parent] : undefined;
  }
  return path;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/lib/jobs/__tests__/job-advancement-tree.test.ts`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add src/lib/jobs/job-advancement-tree.ts src/lib/jobs/__tests__/job-advancement-tree.test.ts
git commit -m "feat(atlas-ui): add consolidated job advancement tree with corrected floors (task-097)"
```

---

## Task 2: Re-home `jobTreePath`, remove `lib/utils/job-tree.ts`

`jobTreePath` is consumed by `SkillsSection.tsx` and `lib/utils/__tests__/job-tree.test.ts`.
Task 1 reproduced it in the new module (with the orphan/path cases folded in), so the
old module and its test can go once the one external import is re-pointed.

**Files:**
- Modify: `src/components/features/characters/SkillsSection.tsx:5`
- Remove: `src/lib/utils/job-tree.ts`, `src/lib/utils/__tests__/job-tree.test.ts`

- [ ] **Step 1: Re-point the import**

In `src/components/features/characters/SkillsSection.tsx`, change line 5:

```ts
import { jobTreePath } from "@/lib/utils/job-tree";
```

to:

```ts
import { jobTreePath } from "@/lib/jobs/job-advancement-tree";
```

(No other change — the returned `JobEntry` shape `{ id, name, parent }` is identical.)

- [ ] **Step 2: Delete the superseded module and its test**

```bash
git rm src/lib/utils/job-tree.ts src/lib/utils/__tests__/job-tree.test.ts
```

- [ ] **Step 3: Verify nothing else references the old module**

Run: `grep -rn "lib/utils/job-tree\|JOB_TREE" src --include='*.ts' --include='*.tsx'`
Expected: no matches.

- [ ] **Step 4: Run the affected tests**

Run: `npx vitest run src/components/features/characters/__tests__/SkillsSection.test.tsx`
Expected: PASS (empty-state and render cases unchanged — same `jobTreePath` behaviour).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(atlas-ui): re-home jobTreePath onto JOB_GRAPH, drop lib/utils/job-tree (task-097)"
```

---

## Task 3: Beginner skill name fallback (`lib/skills/beginner-skill-names.ts`)

Implements FR-3.2/3.3. The trigger is a blank server `name` (any job), never a
hardcoded job id. Map ids/names verified against
`libs/atlas-constants/skill/constants.go` (`Beginner*Id`, 1000–1012).

**Files:**
- Create: `src/lib/skills/beginner-skill-names.ts`
- Test: `src/lib/skills/__tests__/beginner-skill-names.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/skills/__tests__/beginner-skill-names.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { BEGINNER_SKILL_NAMES, resolveSkillName } from "@/lib/skills/beginner-skill-names";

describe("resolveSkillName", () => {
  it("prefers a non-blank server name", () => {
    expect(resolveSkillName(1000, "Improved HP Recovery")).toBe("Improved HP Recovery");
  });

  it("falls back to the curated name when the server name is blank", () => {
    expect(resolveSkillName(1000, "")).toBe("Three Snails");
    expect(resolveSkillName(1004, "")).toBe("Monster Riding");
    expect(resolveSkillName(1012, "")).toBe("Bless of Nymph");
  });

  it("treats whitespace-only as blank", () => {
    expect(resolveSkillName(1001, "   ")).toBe("Recovery");
  });

  it("falls back to Skill <id> for an unknown blank-name id", () => {
    expect(resolveSkillName(99999, "")).toBe("Skill 99999");
    expect(resolveSkillName(99999, undefined)).toBe("Skill 99999");
  });

  it("covers the 13 Beginner skills", () => {
    expect(Object.keys(BEGINNER_SKILL_NAMES)).toHaveLength(13);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/lib/skills/__tests__/beginner-skill-names.test.ts`
Expected: FAIL — cannot resolve module.

- [ ] **Step 3: Write the module**

Create `src/lib/skills/beginner-skill-names.ts`:

```ts
// Curated display names for Beginner skills whose atlas-data definition carries an
// empty `name` (verified live, 2026-06-14: GET /api/data/skills/1000 -> name:"").
// Ids + names sourced from libs/atlas-constants/skill/constants.go (Beginner*Id,
// ids 1000-1012). This is a display hint table, not a data change.
export const BEGINNER_SKILL_NAMES: Record<number, string> = {
  1000: "Three Snails",
  1001: "Recovery",
  1002: "Nimble Feet",
  1003: "Soul of Craftsman",
  1004: "Monster Riding",
  1005: "Echo of Hero",
  1006: "Jump Down",
  1007: "Maker",
  1008: "Multi Pet",
  1009: "Bamboo",
  1010: "Invincible",
  1011: "Berserk",
  1012: "Bless of Nymph",
};

/**
 * Display name for a skill. Uses the server-provided name when non-blank; otherwise
 * the curated map; otherwise `Skill <id>`. Driven by a blank server name (FR-3.3),
 * so any future blank-name skill degrades cleanly — never special-cased to job 0.
 */
export function resolveSkillName(id: number, serverName: string | undefined): string {
  if (serverName != null && serverName.trim() !== "") {
    return serverName;
  }
  return BEGINNER_SKILL_NAMES[id] ?? `Skill ${id}`;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/lib/skills/__tests__/beginner-skill-names.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/lib/skills/beginner-skill-names.ts src/lib/skills/__tests__/beginner-skill-names.test.ts
git commit -m "feat(atlas-ui): add blank-name skill fallback with curated Beginner names (task-097)"
```

---

## Task 4: Skill description markup parser (`lib/skills/format-skill-description.ts`)

Implements FR-7. Pure tokenizer returning a structured model: lines of styled
segments, with a leading `[Master Level : N]` captured and suppressed (OQ-4). `#c…#`
sets a `color` marker but text is rendered plain in v1 (OQ-2); all other `#x…#` and
bare `#` markers are stripped, inner text kept; unknown directives degrade by stripping.

**Files:**
- Create: `src/lib/skills/format-skill-description.ts`
- Test: `src/lib/skills/__tests__/format-skill-description.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/skills/__tests__/format-skill-description.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { formatSkillDescription } from "@/lib/skills/format-skill-description";

describe("formatSkillDescription", () => {
  it("returns an empty model for blank input", () => {
    expect(formatSkillDescription(undefined)).toEqual({ lines: [] });
    expect(formatSkillDescription("")).toEqual({ lines: [] });
    expect(formatSkillDescription("   ")).toEqual({ lines: [] });
  });

  it("splits on newlines into lines of segments", () => {
    const r = formatSkillDescription("Line one\nLine two");
    expect(r.lines).toHaveLength(2);
    expect(r.lines[0]).toEqual([{ text: "Line one" }]);
    expect(r.lines[1]).toEqual([{ text: "Line two" }]);
  });

  it("captures and suppresses a leading [Master Level : N] header", () => {
    const r = formatSkillDescription("[Master Level : 16]\nRecover additional HP");
    expect(r.masterLevelHeader).toBe(16);
    expect(r.lines).toEqual([[{ text: "Recover additional HP" }]]);
  });

  it("strips #c...# color markers and marks the segment color", () => {
    const r = formatSkillDescription("#cAt least Level 3 on Sacrifice#");
    expect(r.lines).toEqual([[{ text: "At least Level 3 on Sacrifice", color: "highlight" }]]);
  });

  it("strips unknown #x...# directives keeping the inner text", () => {
    const r = formatSkillDescription("see #ebold text# here");
    expect(r.lines).toEqual([[{ text: "see " }, { text: "bold text" }, { text: " here" }]]);
  });

  it("consumes bare # reset markers, leaking no '#'", () => {
    const r = formatSkillDescription("#cred#then plain");
    const flat = r.lines.flat().map((s) => s.text).join("");
    expect(flat).not.toContain("#");
    expect(flat).toBe("red" + "then plain");
  });

  it("keeps the master-level header even if the body is empty", () => {
    expect(formatSkillDescription("[Master Level : 30]")).toEqual({ lines: [], masterLevelHeader: 30 });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/lib/skills/__tests__/format-skill-description.test.ts`
Expected: FAIL — cannot resolve module.

- [ ] **Step 3: Write the module**

Create `src/lib/skills/format-skill-description.ts`:

```ts
export interface DescSegment {
  text: string;
  /** Set to "highlight" for #c...# regions; the page renders plain text in v1. */
  color?: string;
}

export interface FormattedDescription {
  /** Outer array = visual lines (split on newlines); inner = styled segments. */
  lines: DescSegment[][];
  /** Parsed from a leading [Master Level : N]; the header is removed from `lines`. */
  masterLevelHeader?: number;
}

const MASTER_LEVEL_RE = /^\s*\[Master Level\s*:\s*(\d+)\]\s*(?:\r\n|\r|\n)?/;
const HIGHLIGHT = "highlight";

function tokenizeLine(line: string): DescSegment[] {
  const segments: DescSegment[] = [];
  let buf = "";
  let color: string | undefined;

  const flush = () => {
    if (buf.length > 0) {
      segments.push(color ? { text: buf, color } : { text: buf });
      buf = "";
    }
  };

  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === "#") {
      const next = line[i + 1];
      if (next !== undefined && /[a-zA-Z]/.test(next)) {
        // opener like #c, #e, #z... — start a new segment, set color only for #c
        flush();
        color = next.toLowerCase() === "c" ? HIGHLIGHT : undefined;
        i += 1; // consume the letter; the loop's i++ consumes the '#'
        continue;
      }
      // bare '#' reset
      flush();
      color = undefined;
      continue;
    }
    buf += ch;
  }
  flush();
  if (segments.length === 0) segments.push({ text: "" });
  return segments;
}

export function formatSkillDescription(raw: string | undefined): FormattedDescription {
  if (raw == null || raw.trim() === "") {
    return { lines: [] };
  }

  let body = raw;
  let masterLevelHeader: number | undefined;
  const m = body.match(MASTER_LEVEL_RE);
  if (m) {
    masterLevelHeader = Number(m[1]);
    body = body.slice(m[0].length);
  }

  if (body.trim() === "") {
    return masterLevelHeader !== undefined ? { lines: [], masterLevelHeader } : { lines: [] };
  }

  const lines = body.split(/\r\n|\r|\n/).map(tokenizeLine);
  return masterLevelHeader !== undefined ? { lines, masterLevelHeader } : { lines };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/lib/skills/__tests__/format-skill-description.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/lib/skills/format-skill-description.ts src/lib/skills/__tests__/format-skill-description.test.ts
git commit -m "feat(atlas-ui): add skill-description markup parser (task-097)"
```

---

## Task 5: `JobsPage` advancement tree + expand affordance + scroll (FR-1, FR-2, FR-6)

Replaces the flat archetype/class/flex-wrap list with a recursive indented tree over
`visibleRoots(major)`. Branch nodes are `Collapsible`s with a rotating-chevron trigger
(toggle only) plus the job name as a navigating `Link`; leaf nodes are a plain `Link`.
Adds page-local `overflow-y-auto`.

**Files:**
- Modify: `src/pages/JobsPage.tsx`
- Test: `src/pages/__tests__/JobsPage.test.tsx` (rewrite)

- [ ] **Step 1: Rewrite the page test to the new structure**

Replace the entire contents of `src/pages/__tests__/JobsPage.test.tsx`:

```ts
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({ useTenant: () => useTenantMock() }));

import { JobsPage } from "@/pages/JobsPage";

const tenant = (major: number) =>
  ({ id: "t1", attributes: { region: "GMS", majorVersion: major, minorVersion: 1 } } as unknown as Tenant);

function renderPage() {
  return render(
    <MemoryRouter>
      <JobsPage />
    </MemoryRouter>,
  );
}

describe("JobsPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows a select-a-tenant empty state when no tenant is active", () => {
    useTenantMock.mockReturnValue({ activeTenant: null });
    renderPage();
    expect(screen.getByText(/select a tenant/i)).toBeInTheDocument();
    expect(screen.queryByText("Beginner")).not.toBeInTheDocument();
  });

  it("renders Cygnus + Aran roots on a v83 tenant, Evan hidden", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    renderPage();
    expect(screen.getByText("Beginner")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.getByText("Noblesse")).toBeInTheDocument(); // Cygnus root visible on v83
    expect(screen.getByText("Legend")).toBeInTheDocument();   // Aran root visible on v83
    expect(screen.queryByText("Evan")).not.toBeInTheDocument();
  });

  it("reveals the Evan root on a v84 tenant", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(84) });
    renderPage();
    expect(screen.getByText("Evan")).toBeInTheDocument();
  });

  it("gives branch nodes a toggle affordance and links job names to detail pages", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    renderPage();
    expect(screen.getByLabelText(/toggle beginner/i)).toBeInTheDocument();
    expect(screen.getByText("Warrior").closest("a")).toHaveAttribute("href", "/jobs/100");
  });

  it("scrolls in-page via a local overflow container (no app-shell change)", () => {
    useTenantMock.mockReturnValue({ activeTenant: tenant(83) });
    const { container } = renderPage();
    expect(container.firstChild).toHaveClass("overflow-y-auto");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/pages/__tests__/JobsPage.test.tsx`
Expected: FAIL — current page still imports `jobs-hierarchy`, has no chevron toggle / `overflow-y-auto`, and hides Cygnus/Legend on v83.

- [ ] **Step 3: Rewrite the page**

Replace the entire contents of `src/pages/JobsPage.tsx`:

```tsx
import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Briefcase, ChevronRight } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { JOB_GRAPH, childrenOf, visibleRoots } from "@/lib/jobs/job-advancement-tree";

function JobTreeNode({ id, depth }: { id: number; depth: number }) {
  const entry = JOB_GRAPH[id];
  const name = entry?.name ?? `Job ${id}`;
  const children = childrenOf(id);
  const indent = { paddingLeft: depth * 16 } as const;

  if (children.length === 0) {
    return (
      <div style={indent} className="py-1">
        <Link
          to={`/jobs/${id}`}
          className="text-sm text-primary underline-offset-2 hover:underline"
        >
          {name}
        </Link>
      </div>
    );
  }

  return (
    <Collapsible defaultOpen={depth === 0}>
      <div style={indent} className="flex items-center gap-1 py-1">
        <CollapsibleTrigger
          aria-label={`Toggle ${name}`}
          className="group flex h-6 w-6 items-center justify-center rounded hover:bg-muted cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ChevronRight className="h-4 w-4 transition-transform group-data-[state=open]:rotate-90" />
        </CollapsibleTrigger>
        <Link
          to={`/jobs/${id}`}
          className="text-sm font-medium text-primary underline-offset-2 hover:underline"
        >
          {name}
        </Link>
      </div>
      <CollapsibleContent>
        {children.map((childId) => (
          <JobTreeNode key={childId} id={childId} depth={depth + 1} />
        ))}
      </CollapsibleContent>
    </Collapsible>
  );
}

export function JobsPage() {
  const { activeTenant } = useTenant();

  const roots = useMemo(
    () => (activeTenant ? visibleRoots(activeTenant.attributes.majorVersion) : []),
    [activeTenant],
  );

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 overflow-y-auto p-10 pb-16">
      <div className="flex items-center gap-2">
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
        <Card>
          <CardHeader>
            <CardTitle>Job Hierarchy</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            {roots.map((rootId) => (
              <JobTreeNode key={rootId} id={rootId} depth={0} />
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/pages/__tests__/JobsPage.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/pages/JobsPage.tsx src/pages/__tests__/JobsPage.test.tsx
git commit -m "feat(atlas-ui): render Jobs as an indented advancement tree with expand affordance + scroll (task-097)"
```

---

## Task 6: Remove the superseded `jobs-hierarchy` module

`JobsPage` no longer imports it (Task 5). The only other consumer is its own test.

**Files:**
- Remove: `src/lib/jobs-hierarchy.ts`, `src/lib/__tests__/jobs-hierarchy.test.ts`

- [ ] **Step 1: Confirm there are no remaining consumers**

Run: `grep -rn "jobs-hierarchy\|JOB_HIERARCHY\|filterHierarchy\|jobNodeName" src --include='*.ts' --include='*.tsx'`
Expected: matches only inside `src/lib/jobs-hierarchy.ts` and `src/lib/__tests__/jobs-hierarchy.test.ts` (both about to be deleted).

- [ ] **Step 2: Delete the module and its test**

```bash
git rm src/lib/jobs-hierarchy.ts src/lib/__tests__/jobs-hierarchy.test.ts
```

- [ ] **Step 3: Run the full suite to confirm nothing broke**

Run: `npm run test`
Expected: PASS (no test references the removed module).

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor(atlas-ui): remove jobs-hierarchy, consolidated into job-advancement-tree (task-097)"
```

---

## Task 7: `JobDetailPage` — fallback name, Master Lv label, copyable id, scroll, formatted description (FR-3, FR-4, FR-5, FR-6, FR-7)

**Files:**
- Modify: `src/pages/JobDetailPage.tsx`
- Test: `src/pages/__tests__/JobDetailPage.test.tsx` (extend)

- [ ] **Step 1: Extend the page test**

Replace the entire contents of `src/pages/__tests__/JobDetailPage.test.tsx`:

```ts
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";

const useTenantMock = vi.fn();
const useJobSkillsMock = vi.fn();
const useJobSkillDefsMock = vi.fn();

vi.mock("@/context/tenant-context", () => ({ useTenant: () => useTenantMock() }));
vi.mock("@/lib/hooks/api/useJobSkills", () => ({
  useJobSkills: (...a: unknown[]) => useJobSkillsMock(...a),
  jobSkillsKeys: { all: ["job-skills"], detail: () => [] },
}));
vi.mock("@/lib/hooks/api/useJobSkillDefinitions", () => ({
  useJobSkillDefinitions: (...a: unknown[]) => useJobSkillDefsMock(...a),
}));

import { JobDetailPage } from "@/pages/JobDetailPage";

const v83 = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as unknown as Tenant;

function def(over: Partial<SkillDefinitionWithIcon>): SkillDefinitionWithIcon {
  return {
    id: 1101004, name: "Iron Body", description: "Hardens the body.", action: false,
    element: "", animationTime: 0, maxLevel: 20, effects: [{ weaponDefense: 16 }],
    iconUrl: "/api/assets/x/GMS/83.1/skill/1101004/icon.png", ...over,
  } as SkillDefinitionWithIcon;
}

function renderAt(jobId = "112") {
  return render(
    <MemoryRouter initialEntries={[`/jobs/${jobId}`]}>
      <Routes>
        <Route path="/jobs/:jobId" element={<JobDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("JobDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTenantMock.mockReturnValue({ activeTenant: v83 });
  });

  it("shows a skeleton while skill ids are loading", () => {
    useJobSkillsMock.mockReturnValue({ data: undefined, isLoading: true, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [], isLoading: true, isError: false });
    renderAt();
    expect(screen.getByTestId("job-detail-loading")).toBeInTheDocument();
  });

  it("shows an empty state when the job grants no skills", () => {
    useJobSkillsMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [], isLoading: false, isError: false });
    renderAt();
    expect(screen.getByText(/grants no skills/i)).toBeInTheDocument();
  });

  it("renders a skill row with a Master Lv indicator and a type badge", () => {
    useJobSkillsMock.mockReturnValue({ data: [1101004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [def({})], isLoading: false, isError: false });
    renderAt();
    expect(screen.getByText("Iron Body")).toBeInTheDocument();
    expect(screen.getByText(/Master Lv/i)).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
    expect(screen.getByText("Passive")).toBeInTheDocument();
  });

  it("renders Beginner skills with a curated fallback when the server name is blank", () => {
    useJobSkillsMock.mockReturnValue({ data: [1000, 1004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({ id: 1000, name: "" }), def({ id: 1004, name: "" })],
      isLoading: false, isError: false,
    });
    renderAt("0");
    expect(screen.getByText("Three Snails")).toBeInTheDocument();
    expect(screen.getByText("Monster Riding")).toBeInTheDocument();
  });

  it("renders the job id as a copyable (focusable) header element", () => {
    useJobSkillsMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [], isLoading: false, isError: false });
    renderAt("112");
    expect(screen.getByText("112")).toHaveAttribute("tabindex", "0");
  });

  it("uses a page-local overflow container for scrolling", () => {
    useJobSkillsMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [], isLoading: false, isError: false });
    const { container } = renderAt();
    expect(container.firstChild).toHaveClass("overflow-y-auto");
  });

  it("renders a markup description with line breaks and no raw '#'", () => {
    useJobSkillsMock.mockReturnValue({ data: [1101004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({ description: "Line one\n#cColored#" })],
      isLoading: false, isError: false,
    });
    renderAt();
    fireEvent.click(screen.getByRole("button", { name: /iron body/i }));
    expect(screen.getByText("Line one")).toBeInTheDocument();
    expect(screen.getByText("Colored")).toBeInTheDocument();
    expect(screen.queryByText(/#c/)).toBeNull();
  });

  it("expanding a skill reveals its per-level table", () => {
    useJobSkillsMock.mockReturnValue({ data: [1101004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({
      definitions: [def({ effects: [{ weaponDefense: 16 }, { weaponDefense: 18 }] })],
      isLoading: false, isError: false,
    });
    renderAt();
    fireEvent.click(screen.getByRole("button", { name: /iron body/i }));
    expect(screen.getByText("Weapon Def")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/pages/__tests__/JobDetailPage.test.tsx`
Expected: FAIL — current page renders `Lv` not `Master Lv`, the id is a non-focusable `Badge`, the outer div has no `overflow-y-auto`, and the description is dumped raw.

- [ ] **Step 3: Rewrite the page**

Replace the entire contents of `src/pages/JobDetailPage.tsx`:

```tsx
import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ChevronLeft, Sparkles } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useJobSkills } from "@/lib/hooks/api/useJobSkills";
import { useJobSkillDefinitions } from "@/lib/hooks/api/useJobSkillDefinitions";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { getJobNameById } from "@/lib/jobs";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { buildLevelTable } from "@/lib/skills/level-table";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import { formatSkillDescription, type FormattedDescription } from "@/lib/skills/format-skill-description";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";

function SkillIcon({ def, name }: { def: SkillDefinitionWithIcon; name: string }) {
  const [failed, setFailed] = useState(false);
  if (failed) {
    return (
      <span data-testid={`skill-icon-fallback-${def.id}`} className="inline-flex h-8 w-8 items-center justify-center text-muted-foreground">
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

function SkillDescription({ formatted }: { formatted: FormattedDescription }) {
  if (formatted.lines.length === 0) {
    return <p className="text-sm text-muted-foreground">No description available.</p>;
  }
  return (
    <div className="text-sm space-y-1">
      {formatted.lines.map((line, i) => (
        <p key={i}>
          {line.map((seg, j) => (
            <span key={j}>{seg.text}</span>
          ))}
        </p>
      ))}
    </div>
  );
}

function LevelTable({ def }: { def: SkillDefinitionWithIcon }) {
  const table = buildLevelTable(def.effects);
  if (table.rows.length === 0) {
    return <p className="text-sm text-muted-foreground">No per-level data.</p>;
  }
  return (
    <div className="rounded-md border overflow-auto">
      <Table>
        <TableHeader className="sticky top-0 bg-background z-10">
          <TableRow>
            {table.columns.map((c) => (
              <TableHead key={c.key}>{c.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {table.rows.map((row, i) => (
            <TableRow key={i}>
              {table.columns.map((c) => (
                <TableCell key={c.key}>{row[c.key] ?? ""}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function SkillRow({ def }: { def: SkillDefinitionWithIcon }) {
  const type = deriveSkillType(def);
  const name = resolveSkillName(def.id, def.name);
  const formatted = formatSkillDescription(def.description);
  return (
    <Collapsible>
      <div className="flex items-center gap-3 py-2 border-b">
        <SkillIcon def={def} name={name} />
        <CollapsibleTrigger asChild>
          <button className="flex-1 text-left">
            <span className="font-medium">{name}</span>
          </button>
        </CollapsibleTrigger>
        <Badge variant="secondary">{type}</Badge>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                tabIndex={0}
                className="text-sm text-muted-foreground w-24 text-right cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
              >
                Master Lv <span>{def.maxLevel ?? "—"}</span>
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p>Skill&#39;s maximum (master) level</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <CollapsibleContent className="py-3 pl-11 space-y-3">
        <SkillDescription formatted={formatted} />
        <div className="flex gap-4 text-xs text-muted-foreground">
          <span>Type: {type}</span>
          {def.element ? <span>Element: {def.element}</span> : null}
          <span>Master Level: {def.maxLevel ?? "—"}</span>
        </div>
        <LevelTable def={def} />
      </CollapsibleContent>
    </Collapsible>
  );
}

export function JobDetailPage() {
  const { jobId } = useParams<{ jobId: string }>();
  const { activeTenant } = useTenant();
  const numericJobId = Number(jobId);
  const jobName = getJobNameById(numericJobId) ?? `Job ${jobId}`;

  const skillsQuery = useJobSkills(activeTenant, numericJobId);
  const skillIds = skillsQuery.data ?? [];
  const { definitions, isLoading: defsLoading, isError: defsError } = useJobSkillDefinitions(activeTenant, skillIds);

  const loading = skillsQuery.isLoading || (skillIds.length > 0 && defsLoading);

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 overflow-y-auto p-10 pb-16">
      <div className="flex items-center gap-2">
        <Link to="/jobs" className="text-muted-foreground hover:text-foreground">
          <ChevronLeft className="h-5 w-5" />
        </Link>
        <h2 className="text-2xl font-bold tracking-tight">{jobName}</h2>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                tabIndex={0}
                className="inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {jobId}
              </span>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{jobId}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      {!activeTenant ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Select a tenant to browse its jobs and skills.
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Skills</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div data-testid="job-detail-loading" className="space-y-2">
                {[0, 1, 2].map((i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : skillsQuery.isError ? (
              <p className="text-center py-8 text-destructive">Failed to load this job&#39;s skills.</p>
            ) : skillIds.length === 0 ? (
              <p className="text-center py-8 text-muted-foreground">This job grants no skills.</p>
            ) : definitions.length === 0 && defsError ? (
              <p className="text-center py-8 text-destructive">Skill details unavailable.</p>
            ) : (
              <div>
                {definitions.map((def) => (
                  <SkillRow key={def.id} def={def} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/pages/__tests__/JobDetailPage.test.tsx`
Expected: PASS (all cases including fallback names, Master Lv, copyable id, scroll, formatted description).

- [ ] **Step 5: Commit**

```bash
git add src/pages/JobDetailPage.tsx src/pages/__tests__/JobDetailPage.test.tsx
git commit -m "feat(atlas-ui): job detail fallback names, Master Lv label, copyable id, scroll, formatted description (task-097)"
```

---

## Task 8: Full verification gate

Run the project gate before declaring the branch done (PRD §8, repo CLAUDE.md).
Frontend-only — no Go build / `docker buildx bake` / `go.work` / k8s steps.

- [ ] **Step 1: Ensure Node 22**

Run: `source ~/.nvm/nvm.sh && nvm use 22`
Expected: `Now using node v22.x`.

- [ ] **Step 2: Full test suite**

Run: `npm run test`
Expected: all suites PASS (including the three new helper suites and both rewritten page suites).

- [ ] **Step 3: Build + typecheck**

Run: `npm run build`
Expected: clean (tsc -b + vite build); no type errors in the new helpers, pages, or test files.

- [ ] **Step 4: Lint — no new errors**

Run: `npm run lint`
Expected: only the pre-existing baseline errors (~48); **zero new** errors attributable to the files this task touched. (Baseline is known-broken — ref `reference_atlas_ui_npm_nvm_and_lint_baseline`; do not chase pre-existing failures.)

- [ ] **Step 5: Confirm no stray references remain**

Run: `grep -rn "jobs-hierarchy\|lib/utils/job-tree\|JOB_TREE\|JOB_HIERARCHY" src --include='*.ts' --include='*.tsx'`
Expected: no matches.

- [ ] **Step 6: Acceptance walk-through (manual, optional but recommended)**

`npm run dev`, select a v83 tenant, and confirm against PRD §10:
- `/jobs` shows an indented tree with rotating chevrons on branch nodes; Cygnus
  (Noblesse) and Aran (Legend) appear, Evan does not.
- `/jobs/0` lists the 13 Beginner skills with curated names (Three Snails, …),
  never blank.
- A skill row reads `Master Lv N`; the job id in the detail header copies on click.
- Expanding many skills scrolls the page; level tables still scroll horizontally.
- A skill description renders with line breaks and no raw `#c…#`.

---

## Self-Review (completed by plan author)

**Spec coverage** (FR-1…FR-8 + acceptance criteria):

| Requirement | Task |
|---|---|
| FR-1 expand affordance (chevron, pointer, focus, distinct states) | Task 5 |
| FR-2 advancement-tree layout, single source of truth, nav targets, indentation | Tasks 1, 5 |
| FR-3 Beginner render + generic blank-name fallback | Tasks 3, 7 (and verify no id-0 short-circuit — context.md finding #3) |
| FR-4 Master Lv label + tooltip + shared term | Task 7 |
| FR-5 copyable job id via existing pattern | Task 7 |
| FR-6 page-local scroll, no app-shell change, table scroll coexists | Tasks 5, 7 |
| FR-7 description markup parser, unit-tested, unknown directives stripped | Task 4, rendered in Task 7 |
| FR-8 corrected floors (Cygnus 83, Aran 80), documented basis, stale comments removed | Tasks 1, 6 |
| AC: single hierarchy/tree owns the edges | Tasks 1, 2, 6 (both old modules removed) |
| AC: build clean, tests green, no new lint | Task 8 |

**Placeholder scan:** No TBD/TODO/"add error handling"/"similar to Task N" — every code step shows complete code; every test step shows complete assertions.

**Type consistency:** `JobEntry {id,name,parent}`, `JOB_GRAPH`, `BRANCH_FLOORS`,
`visibleRoots`/`childrenOf`/`rootOf`/`floorOf`/`jobTreePath` used identically in
Tasks 1/2/5; `resolveSkillName(id, serverName)` identical in Tasks 3/7;
`FormattedDescription { lines, masterLevelHeader }` and `DescSegment { text, color }`
identical in Tasks 4/7; `formatSkillDescription(raw)` identical in Tasks 4/7. No drift.
