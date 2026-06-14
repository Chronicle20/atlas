# Jobs & Skills Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only Jobs & Skills browser to `atlas-ui`: pick a job from a version-filtered archetype→class→tier tree, see every skill it grants with icon/title/master-level/type, and expand a skill to its full per-level bonus table.

**Architecture:** Frontend-only. A static job hierarchy (`jobs-hierarchy.ts`) with per-node `minMajorVersion` drives a pure `filterHierarchy`. Two pages — `JobsPage` (zero-network accordion tree) and `JobDetailPage` (skill list via existing `useJobSkills` + a new parallel `useJobSkillDefinitions`). Three pure helpers: `deriveSkillType`, `buildLevelTable`, `filterHierarchy`. The existing `atlas-data` endpoints, service modules, and `useSkillDefinition` cache keys are reused unchanged; the existing `skill-effect-format.ts` statup label map and `getJobNameById` are reused, not duplicated.

**Tech Stack:** Vite + React 19 + React Router v7, TanStack React Query 5 (`useQueries`), shadcn/ui (`Accordion`/`Collapsible`/`Table`/`Badge`/`Card`), Vitest + Testing Library, Tailwind 4.

---

## Conventions for every task

- **Working dir:** `services/atlas-ui`. All paths below are relative to it.
- **Run tests:** `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null && npm run test -- <file>` (single file) or `npm run test` (all).
- **Build:** `source ~/.nvm/nvm.sh && nvm use 22 >/dev/null && npm run build` — type-checks `*.test.ts` too.
- **Named exports only** on pages/components; `@/` alias; `import.meta.env.VITE_*`; no `next/*`.
- Commit after each task with the message shown in its final step.

---

## Task 1: Extend `skills.service.ts` — `maxLevel` + broadened `SkillEffect`

Implements FR-6.1, FR-6.2. Adds the optional `maxLevel` mapping and the extra
numeric `SkillEffect` keys the per-level table enumerates. All additions are
**optional** so no existing call site (`SkillTooltipContent`, `SkillWidget`,
`useSkillData`, their tests) breaks.

**Files:**
- Modify: `src/services/api/skills.service.ts`
- Test: `src/services/api/__tests__/skills.service.test.ts` (create)

- [ ] **Step 1: Write the failing test**

Create `src/services/api/__tests__/skills.service.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";

const getOneMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: { getOne: (...args: unknown[]) => getOneMock(...args) },
}));

import { skillsService } from "@/services/api/skills.service";

describe("skillsService.getSkillById", () => {
  beforeEach(() => vi.clearAllMocks());

  it("maps maxLevel and effects from the resource", async () => {
    getOneMock.mockResolvedValue({
      id: "1101004",
      type: "skills",
      attributes: {
        name: "Iron Body",
        description: "Hardens the body.",
        action: false,
        element: "",
        animationTime: 0,
        maxLevel: 20,
        effects: [{ weaponDefense: 16, statups: [{ type: "WeaponDefense", amount: 16 }] }],
      },
    });

    const def = await skillsService.getSkillById("1101004");

    expect(def.maxLevel).toBe(20);
    expect(def.effects[0].weaponDefense).toBe(16);
    expect(def.effects[0].statups?.[0]).toEqual({ type: "WeaponDefense", amount: 16 });
  });

  it("defaults maxLevel to undefined and effects to [] when absent", async () => {
    getOneMock.mockResolvedValue({
      id: "9",
      type: "skills",
      attributes: { name: "X", action: true, element: "", animationTime: 0 },
    });
    const def = await skillsService.getSkillById("9");
    expect(def.maxLevel).toBeUndefined();
    expect(def.effects).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/services/api/__tests__/skills.service.test.ts`
Expected: FAIL — `def.maxLevel` is `undefined` because the mapper doesn't set it (first assertion fails).

- [ ] **Step 3: Implement the change**

In `src/services/api/skills.service.ts`, broaden `SkillEffect`, add `maxLevel`
to both `SkillDefinition` and `SkillResource.attributes`, and map it. Replace the
existing `SkillEffect`, `SkillDefinition`, `SkillResource`, and `getSkillById`
with:

```ts
// Mirrors atlas-data effect.RestModel (services/atlas-data/.../skill/effect/rest.go).
// JSON keys verified against source — keep casing exact (e.g. MPConsume, hpR).
export interface SkillEffect {
  weaponAttack?: number;
  magicAttack?: number;
  weaponDefense?: number;
  magicDefense?: number;
  accuracy?: number;
  avoidability?: number;
  speed?: number;
  jump?: number;
  hp?: number;
  mp?: number;
  hpR?: number;
  mpR?: number;
  MHPRRate?: number;
  MMPRRate?: number;
  mhpr?: number;
  mmpr?: number;
  HPConsume?: number;
  MPConsume?: number;
  duration?: number;
  overTime?: boolean;
  cooldown?: number;
  x?: number;
  y?: number;
  mobCount?: number;
  moneyConsume?: number;
  morphId?: number;
  prop?: number;
  itemConsume?: number;
  itemConsumeAmount?: number;
  damage?: number;
  attackCount?: number;
  fixDamage?: number;
  bulletCount?: number;
  bulletConsume?: number;
  statups?: SkillEffectStatup[];
}

export interface SkillDefinition {
  id: number;
  name: string;
  description: string; // "" when atlas-data not yet upgraded
  action: boolean;
  element: string;
  animationTime: number;
  maxLevel?: number; // optional: older atlas-data responses omit it
  effects: SkillEffect[];
}

interface SkillResource {
  id: string;
  type: string;
  attributes: {
    name: string;
    description?: string;
    action: boolean;
    element: string;
    animationTime: number;
    maxLevel?: number;
    effects?: SkillEffect[];
  };
}
```

Keep `SkillEffectStatup` and `getSkillName` as they are. Update `getSkillById`'s
return object to add one line:

```ts
  async getSkillById(id: string): Promise<SkillDefinition> {
    const skill = await api.getOne<SkillResource>(`${BASE_PATH}/${id}`);
    return {
      id: parseInt(skill.id, 10),
      name: skill.attributes.name,
      description: skill.attributes.description ?? "",
      action: skill.attributes.action,
      element: skill.attributes.element,
      animationTime: skill.attributes.animationTime,
      maxLevel: skill.attributes.maxLevel,
      effects: skill.attributes.effects ?? [],
    };
  },
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/services/api/__tests__/skills.service.test.ts`
Expected: PASS (both cases).

- [ ] **Step 5: Run the full build to confirm no consumer broke**

Run: `npm run build`
Expected: PASS. (Every addition is optional; `SkillTooltipContent`/`SkillWidget`/`useSkillData` and their tests still compile.) If a mock relied on the exact old shape, fix it in this same commit.

- [ ] **Step 6: Commit**

```bash
git add src/services/api/skills.service.ts src/services/api/__tests__/skills.service.test.ts
git commit -m "feat(atlas-ui): map skill maxLevel and broaden SkillEffect (task-094)"
```

---

## Task 2: Extract a shared `fetchSkillDefinitionWithIcon` fetcher

So Task 3's batch hook and the existing `useSkillDefinition` share one fetch
function and the same cache key (NFR perf, no duplication). Pure refactor — the
existing hook's behavior is unchanged.

**Files:**
- Modify: `src/lib/hooks/api/useSkillDefinition.ts`
- Test: `src/lib/hooks/api/__tests__/useSkillDefinition.test.tsx` (already exists — must still pass)

- [ ] **Step 1: Add the exported fetcher and route the hook through it**

In `src/lib/hooks/api/useSkillDefinition.ts`, add an exported async function and
make the hook's `queryFn` call it. Replace the file body with:

```ts
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import type { Tenant } from "@/services/api/tenants.service";
import { skillsService, type SkillDefinition } from "@/services/api/skills.service";
import { getAssetIconUrl } from "@/lib/utils/asset-url";

export interface SkillDefinitionWithIcon extends SkillDefinition {
  iconUrl: string;
}

export const skillDefinitionKeys = {
  all: ["skill-definition"] as const,
  detail: (tenantId: string | undefined, skillId: number) =>
    ["skill-definition", tenantId, skillId] as const,
};

/** Shared fetcher: skill definition + deterministic icon URL. Reused by the batch hook. */
export async function fetchSkillDefinitionWithIcon(
  tenant: Tenant,
  skillId: number,
): Promise<SkillDefinitionWithIcon> {
  const def = await skillsService.getSkillById(skillId.toString());
  return {
    ...def,
    iconUrl: getAssetIconUrl(
      tenant.id,
      tenant.attributes.region,
      tenant.attributes.majorVersion,
      tenant.attributes.minorVersion,
      "skill",
      skillId,
    ),
  };
}

/** Retry policy shared with the batch hook: never retry a 404. */
export function skillDefinitionRetry(failureCount: number, error: Error): boolean {
  const msg = error?.message?.toLowerCase() ?? "";
  if (msg.includes("404") || msg.includes("not found")) return false;
  return failureCount < 3;
}

export function useSkillDefinition(
  tenant: Tenant | null | undefined,
  skillId: number,
): UseQueryResult<SkillDefinitionWithIcon, Error> {
  return useQuery({
    queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
    queryFn: () => {
      if (!tenant) throw new Error("Tenant is required");
      return fetchSkillDefinitionWithIcon(tenant, skillId);
    },
    enabled: !!tenant?.id && skillId > 0,
    staleTime: 30 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
    retry: skillDefinitionRetry,
  });
}
```

- [ ] **Step 2: Run the existing hook test to verify no regression**

Run: `npm run test -- src/lib/hooks/api/__tests__/useSkillDefinition.test.tsx`
Expected: PASS (unchanged behavior).

- [ ] **Step 3: Commit**

```bash
git add src/lib/hooks/api/useSkillDefinition.ts
git commit -m "refactor(atlas-ui): extract shared skill-definition fetcher (task-094)"
```

---

## Task 3: New `useJobSkillDefinitions` aggregate hook

Parallel `useQueries` over a job's skill ids, reusing `skillDefinitionKeys.detail`
so a skill viewed elsewhere is a cache hit. Implements §4.3 fan-out.

**Files:**
- Create: `src/lib/hooks/api/useJobSkillDefinitions.ts`
- Test: `src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`:

```tsx
import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useJobSkillDefinitions } from "../useJobSkillDefinitions";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getSkillByIdMock = vi.fn();
vi.mock("@/services/api/skills.service", () => ({
  skillsService: { getSkillById: (...a: unknown[]) => getSkillByIdMock(...a) },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useJobSkillDefinitions", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches every skill id in parallel and exposes an icon url", async () => {
    getSkillByIdMock.mockImplementation((id: string) =>
      Promise.resolve({ id: Number(id), name: `Skill ${id}`, description: "", action: true, element: "", animationTime: 0, effects: [] }),
    );

    const { result } = renderHook(() => useJobSkillDefinitions(fakeTenant, [1101000, 1101001]), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.definitions).toHaveLength(2);
    expect(result.current.definitions.map((d) => d.id).sort()).toEqual([1101000, 1101001]);
    expect(result.current.definitions[0].iconUrl).toContain("/skill/");
  });

  it("returns an empty result for no skill ids and fires no requests", async () => {
    const { result } = renderHook(() => useJobSkillDefinitions(fakeTenant, []), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.definitions).toEqual([]);
    expect(getSkillByIdMock).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`
Expected: FAIL with "Cannot find module '../useJobSkillDefinitions'".

- [ ] **Step 3: Implement the hook**

Create `src/lib/hooks/api/useJobSkillDefinitions.ts`:

```ts
import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import {
  fetchSkillDefinitionWithIcon,
  skillDefinitionKeys,
  skillDefinitionRetry,
  type SkillDefinitionWithIcon,
} from "@/lib/hooks/api/useSkillDefinition";

export interface UseJobSkillDefinitionsResult {
  /** Resolved definitions, sorted ascending by skill id. */
  definitions: SkillDefinitionWithIcon[];
  isLoading: boolean;
  isError: boolean;
}

export function useJobSkillDefinitions(
  tenant: Tenant | null | undefined,
  skillIds: number[],
): UseJobSkillDefinitionsResult {
  const results = useQueries({
    queries: skillIds.map((skillId) => ({
      queryKey: skillDefinitionKeys.detail(tenant?.id, skillId),
      queryFn: () => {
        if (!tenant) throw new Error("Tenant is required");
        return fetchSkillDefinitionWithIcon(tenant, skillId);
      },
      enabled: !!tenant?.id && skillId > 0,
      staleTime: 30 * 60 * 1000,
      gcTime: 24 * 60 * 60 * 1000,
      retry: skillDefinitionRetry,
    })),
  });

  return useMemo(() => {
    const definitions = results
      .map((r) => r.data)
      .filter((d): d is SkillDefinitionWithIcon => d != null)
      .sort((a, b) => a.id - b.id);
    return {
      definitions,
      isLoading: results.some((r) => r.isLoading),
      isError: results.length > 0 && results.every((r) => r.isError),
    };
  }, [results]);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx`
Expected: PASS (both cases).

- [ ] **Step 5: Commit**

```bash
git add src/lib/hooks/api/useJobSkillDefinitions.ts src/lib/hooks/api/__tests__/useJobSkillDefinitions.test.tsx
git commit -m "feat(atlas-ui): add useJobSkillDefinitions parallel fetch hook (task-094)"
```

---

## Task 4: `deriveSkillType` pure helper

Implements FR-3.5. Single documented heuristic, degrades safely.

**Files:**
- Create: `src/lib/skills/skill-type.ts`
- Test: `src/lib/skills/__tests__/skill-type.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/skills/__tests__/skill-type.test.ts`. The fixtures mirror real
v83 skill shapes (a mastery, a booster, an attack); Task 7 confirms them against
live data.

```ts
import { describe, it, expect } from "vitest";
import { deriveSkillType } from "@/lib/skills/skill-type";

describe("deriveSkillType", () => {
  it("classifies a stat-up effect as Buff", () => {
    // mirrors a booster/armor skill: action true but the point is the stat bonus
    expect(
      deriveSkillType({ action: true, effects: [{ statups: [{ type: "WeaponDefense", amount: 16 }] }] }),
    ).toBe("Buff");
  });

  it("classifies an overTime effect with no statups as Buff", () => {
    expect(deriveSkillType({ action: false, effects: [{ overTime: true }] })).toBe("Buff");
  });

  it("classifies an action skill with no statups/overTime as Active", () => {
    // mirrors an attack: damage + attackCount, no buff
    expect(
      deriveSkillType({ action: true, effects: [{ damage: 120, attackCount: 1 }] }),
    ).toBe("Active");
  });

  it("classifies a non-action skill as Passive", () => {
    // mirrors a mastery: no action animation, no statups
    expect(deriveSkillType({ action: false, effects: [{ accuracy: 10 }] })).toBe("Passive");
  });

  it("degrades safely with missing fields", () => {
    expect(deriveSkillType({ action: false, effects: [] })).toBe("Passive");
    expect(deriveSkillType({ action: true } as never)).toBe("Active");
    expect(deriveSkillType({} as never)).toBe("Passive");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/skills/__tests__/skill-type.test.ts`
Expected: FAIL with "Cannot find module '@/lib/skills/skill-type'".

- [ ] **Step 3: Implement the helper**

Create `src/lib/skills/skill-type.ts`:

```ts
import type { SkillDefinition } from "@/services/api/skills.service";

export const SKILL_TYPE = ["Passive", "Active", "Buff"] as const;
export type SkillType = (typeof SKILL_TYPE)[number];

/**
 * Derives a display type from existing skill fields (atlas-data has no explicit
 * type). Order matters: a stat-up / sustained effect is a Buff regardless of
 * action; otherwise an action animation means Active; otherwise Passive.
 * Never throws on missing fields.
 */
export function deriveSkillType(
  def: Pick<SkillDefinition, "action" | "effects">,
): SkillType {
  const effects = def?.effects ?? [];
  const hasStatups = effects.some((e) => (e?.statups?.length ?? 0) > 0);
  const sustained = effects.some((e) => e?.overTime === true);
  if (hasStatups || sustained) return "Buff";
  if (def?.action === true) return "Active";
  return "Passive";
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/lib/skills/__tests__/skill-type.test.ts`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add src/lib/skills/skill-type.ts src/lib/skills/__tests__/skill-type.test.ts
git commit -m "feat(atlas-ui): add deriveSkillType helper (task-094)"
```

---

## Task 5: `buildLevelTable` + reuse the existing statup label map

Implements FR-4.2–FR-4.5. Exports the pre-existing statup label map as
`STATUP_LABELS` (no second map) and adds a scalar `FIELD_LABELS` map.

**Files:**
- Modify: `src/lib/utils/skill-effect-format.ts` (export the label map under a shared name)
- Create: `src/lib/skills/level-table.ts`
- Test: `src/lib/skills/__tests__/level-table.test.ts`

- [ ] **Step 1: Export the existing statup label map**

In `src/lib/utils/skill-effect-format.ts`, the module-private `const LABELS` is
the statup-type label map. Export it under a shared name without changing
`formatStatup`. Change the declaration line:

```ts
// was: const LABELS: Record<string, string> = {
export const STATUP_LABELS: Record<string, string> = {
```

…and update the one reference inside `formatStatup` from `LABELS[s.type]` to
`STATUP_LABELS[s.type]`. Nothing else changes.

- [ ] **Step 2: Write the failing test**

Create `src/lib/skills/__tests__/level-table.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { buildLevelTable } from "@/lib/skills/level-table";
import type { SkillEffect } from "@/services/api/skills.service";

describe("buildLevelTable", () => {
  it("returns just a Level column for empty effects", () => {
    const t = buildLevelTable([]);
    expect(t.columns.map((c) => c.key)).toEqual(["level"]);
    expect(t.rows).toEqual([]);
  });

  it("emits one row per level with level numbers 1..n", () => {
    const effects: SkillEffect[] = [{ MPConsume: 10 }, { MPConsume: 12 }];
    const t = buildLevelTable(effects);
    expect(t.rows).toHaveLength(2);
    expect(t.rows[0].level).toBe("1");
    expect(t.rows[1].level).toBe("2");
  });

  it("omits a column that is zero/absent across every level", () => {
    const effects: SkillEffect[] = [{ MPConsume: 10, weaponAttack: 0 }, { MPConsume: 12 }];
    const t = buildLevelTable(effects);
    const keys = t.columns.map((c) => c.key);
    expect(keys).toContain("MPConsume");
    expect(keys).not.toContain("weaponAttack");
  });

  it("includes a column with at least one non-zero level", () => {
    const effects: SkillEffect[] = [{ weaponAttack: 0 }, { weaponAttack: 5 }];
    const t = buildLevelTable(effects);
    const col = t.columns.find((c) => c.key === "weaponAttack");
    expect(col?.label).toBe("Weapon Atk");
    expect(t.rows[0].weaponAttack).toBe("");
    expect(t.rows[1].weaponAttack).toBe("5");
  });

  it("derives one column per distinct statup type, labelled and valued per level", () => {
    const effects: SkillEffect[] = [
      { statups: [{ type: "WeaponAttack", amount: 10 }] },
      { statups: [{ type: "WeaponAttack", amount: 12 }, { type: "Accuracy", amount: 3 }] },
    ];
    const t = buildLevelTable(effects);
    const watk = t.columns.find((c) => c.key === "statup:WeaponAttack");
    const acc = t.columns.find((c) => c.key === "statup:Accuracy");
    expect(watk?.label).toBe("Weapon Attack"); // from reused STATUP_LABELS
    expect(acc?.label).toBe("Accuracy");
    expect(t.rows[0]["statup:WeaponAttack"]).toBe("10");
    expect(t.rows[0]["statup:Accuracy"]).toBe(""); // absent on level 1
    expect(t.rows[1]["statup:Accuracy"]).toBe("3");
  });

  it("falls back to the raw key for an unlabeled field", () => {
    const effects: SkillEffect[] = [{ morphId: 1000 }];
    const t = buildLevelTable(effects);
    const col = t.columns.find((c) => c.key === "morphId");
    expect(col?.label).toBe("Morph ID");
    const unknownStatup = buildLevelTable([{ statups: [{ type: "Zzz", amount: 1 }] }]);
    expect(unknownStatup.columns.find((c) => c.key === "statup:Zzz")?.label).toBe("Zzz");
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `npm run test -- src/lib/skills/__tests__/level-table.test.ts`
Expected: FAIL with "Cannot find module '@/lib/skills/level-table'".

- [ ] **Step 4: Implement `level-table.ts`**

Create `src/lib/skills/level-table.ts`:

```ts
import type { SkillEffect } from "@/services/api/skills.service";
import { STATUP_LABELS } from "@/lib/utils/skill-effect-format";

export interface LevelColumn {
  key: string;
  label: string;
}
export interface LevelTable {
  columns: LevelColumn[];
  rows: Array<Record<string, string>>;
}

/**
 * Ordered numeric SkillEffect magnitude fields → human labels. Keys are the
 * exact atlas-data effect.RestModel JSON keys (camelCase, casing significant).
 * Structured fields (lt/rb/monsterStatus/cardStats/cureAbnormalStatuses) and
 * booleans (overTime/skill/repeatEffect) are intentionally excluded.
 */
export const FIELD_LABELS: Array<[keyof SkillEffect, string]> = [
  ["weaponAttack", "Weapon Atk"],
  ["magicAttack", "Magic Atk"],
  ["weaponDefense", "Weapon Def"],
  ["magicDefense", "Magic Def"],
  ["accuracy", "Accuracy"],
  ["avoidability", "Avoid"],
  ["speed", "Speed"],
  ["jump", "Jump"],
  ["hp", "HP"],
  ["mp", "MP"],
  ["hpR", "HP Recovery %"],
  ["mpR", "MP Recovery %"],
  ["mhpr", "HP Recovery"],
  ["mmpr", "MP Recovery"],
  ["MHPRRate", "Max HP Recovery %"],
  ["MMPRRate", "Max MP Recovery %"],
  ["HPConsume", "HP Cost"],
  ["MPConsume", "MP Cost"],
  ["duration", "Duration (ms)"],
  ["cooldown", "Cooldown (ms)"],
  ["damage", "Damage %"],
  ["attackCount", "Attack Count"],
  ["mobCount", "Mob Count"],
  ["prop", "Chance %"],
  ["x", "X"],
  ["y", "Y"],
  ["fixDamage", "Fixed Damage"],
  ["bulletCount", "Bullets"],
  ["bulletConsume", "Bullet Cost"],
  ["morphId", "Morph ID"],
  ["moneyConsume", "Meso Cost"],
  ["itemConsume", "Item ID"],
  ["itemConsumeAmount", "Item Qty"],
];

function isPresent(v: number | undefined): boolean {
  return typeof v === "number" && v !== 0;
}

/**
 * Builds a per-level table: one row per effect (level i+1), one "Level" column
 * plus one column per scalar field with a non-zero value at any level, plus one
 * column per distinct statup type. All-zero/absent columns are omitted.
 */
export function buildLevelTable(effects: SkillEffect[]): LevelTable {
  const columns: LevelColumn[] = [{ key: "level", label: "Level" }];

  // Scalar columns: keep a field iff some level has a non-zero value.
  for (const [field, label] of FIELD_LABELS) {
    if (effects.some((e) => isPresent(e?.[field] as number | undefined))) {
      columns.push({ key: field as string, label });
    }
  }

  // Statup columns: union of distinct types across all levels, in first-seen order.
  const statupTypes: string[] = [];
  for (const e of effects) {
    for (const s of e?.statups ?? []) {
      if (!statupTypes.includes(s.type)) statupTypes.push(s.type);
    }
  }
  for (const type of statupTypes) {
    columns.push({ key: `statup:${type}`, label: STATUP_LABELS[type] ?? type });
  }

  const rows = effects.map((e, i) => {
    const row: Record<string, string> = { level: String(i + 1) };
    for (const [field] of FIELD_LABELS) {
      const v = e?.[field] as number | undefined;
      if (columns.some((c) => c.key === field)) {
        row[field as string] = isPresent(v) ? String(v) : "";
      }
    }
    for (const type of statupTypes) {
      const found = (e?.statups ?? []).find((s) => s.type === type);
      row[`statup:${type}`] = found ? String(found.amount) : "";
    }
    return row;
  });

  return { columns, rows };
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `npm run test -- src/lib/skills/__tests__/level-table.test.ts src/lib/utils/__tests__/skill-effect-format.test.ts`
Expected: PASS — the new table test and the **existing** `skill-effect-format` test (proves the `LABELS`→`STATUP_LABELS` rename didn't break `formatStatup`).

- [ ] **Step 6: Commit**

```bash
git add src/lib/utils/skill-effect-format.ts src/lib/skills/level-table.ts src/lib/skills/__tests__/level-table.test.ts
git commit -m "feat(atlas-ui): add buildLevelTable, reuse statup label map (task-094)"
```

---

## Task 6: Static `jobs-hierarchy.ts` + `filterHierarchy`

Implements FR-1.x, FR-2.x. Authors only the archetype/class grouping + per-node
`minMajorVersion`; leaf names resolve via `getJobNameById` (no duplication).

**Files:**
- Create: `src/lib/jobs-hierarchy.ts`
- Test: `src/lib/__tests__/jobs-hierarchy.test.ts`

- [ ] **Step 1: Write the failing test**

Create `src/lib/__tests__/jobs-hierarchy.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { JOB_HIERARCHY, filterHierarchy, jobNodeName } from "@/lib/jobs-hierarchy";

describe("jobs-hierarchy", () => {
  it("resolves leaf names via getJobNameById", () => {
    expect(jobNodeName({ jobId: 112, minMajorVersion: 83 })).toBe("Hero");
  });

  it("v83 keeps Adventurer but drops Cygnus, Legend, and Evan", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 83);
    const names = tree.map((a) => a.name);
    expect(names).toContain("Adventurer");
    expect(names).not.toContain("Cygnus");
    expect(names).not.toContain("Legend");
  });

  it("removes a class with no surviving jobs and an archetype with no surviving classes", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 83);
    // No Cygnus archetype at all (all its jobs are minMajorVersion > 83)
    expect(tree.find((a) => a.name === "Cygnus")).toBeUndefined();
    // Every surviving class has at least one job
    for (const arch of tree) {
      for (const cls of arch.classes) {
        expect(cls.jobs.length).toBeGreaterThan(0);
      }
    }
  });

  it("a high version keeps Cygnus and Legend archetypes", () => {
    const tree = filterHierarchy(JOB_HIERARCHY, 95);
    const names = tree.map((a) => a.name);
    expect(names).toContain("Cygnus");
    expect(names).toContain("Legend");
  });

  it("does not mutate the source tree", () => {
    const before = JOB_HIERARCHY.length;
    filterHierarchy(JOB_HIERARCHY, 83);
    expect(JOB_HIERARCHY.length).toBe(before);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/__tests__/jobs-hierarchy.test.ts`
Expected: FAIL with "Cannot find module '@/lib/jobs-hierarchy'".

- [ ] **Step 3: Implement the hierarchy**

Create `src/lib/jobs-hierarchy.ts`. `minMajorVersion` integers carry a cited
basis and are **confirmed in Task 7**; the mechanism is correct regardless of the
exact integer.

```ts
import { getJobNameById } from "@/lib/jobs";

export type Archetype = "Adventurer" | "Cygnus" | "Legend" | "Admin";

export interface JobNode {
  jobId: number; // key for /api/data/jobs/{id}/skills
  minMajorVersion: number; // FR-2.2a version gate
}
export interface ClassNode {
  name: string;
  jobs: JobNode[];
}
export interface ArchetypeNode {
  name: Archetype;
  classes: ClassNode[];
}

/** Display name for a leaf — reuses jobNameMap, never duplicates it. */
export function jobNodeName(node: JobNode): string {
  return getJobNameById(node.jobId) ?? `Job ${node.jobId}`;
}

// minMajorVersion basis:
//  - 83: v83 baseline — all Adventurer jobs, GM/Super GM, and Maple Leaf
//        Brigadier exist in v83 data (confirmed by Task 7 probe).
//  - Evan (2001/22xx) = 84  — reference_maplestory_version_timeline (Evan ≈ v84).
//  - Aran (2000/21xx) = 88  — timeline floor (Aran predates Dual Blade ≈ v88);
//        confirm/adjust via Task 7 probe.
//  - Cygnus (1000–1512) = 92 — Knights of Cygnus; floor only needs to exceed 83
//        so it hides on the v83 baseline. Confirm via Task 7 probe.
const ADV = 83;
const ADMIN = 83;
const CYGNUS = 92;
const ARAN = 88;
const EVAN = 84;

export const JOB_HIERARCHY: ArchetypeNode[] = [
  {
    name: "Adventurer",
    classes: [
      { name: "Beginner", jobs: [{ jobId: 0, minMajorVersion: ADV }] },
      {
        name: "Warrior",
        jobs: [100, 110, 111, 112, 120, 121, 122, 130, 131, 132].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Magician",
        jobs: [200, 210, 211, 212, 220, 221, 222, 230, 231, 232].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Bowman",
        jobs: [300, 310, 311, 312, 320, 321, 322].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Thief",
        jobs: [400, 410, 411, 412, 420, 421, 422].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      {
        name: "Pirate",
        jobs: [500, 510, 511, 512, 520, 521, 522].map((jobId) => ({ jobId, minMajorVersion: ADV })),
      },
      { name: "Special", jobs: [{ jobId: 800, minMajorVersion: ADV }] },
    ],
  },
  {
    name: "Cygnus",
    classes: [
      { name: "Noblesse", jobs: [{ jobId: 1000, minMajorVersion: CYGNUS }] },
      { name: "Dawn Warrior", jobs: [1100, 1110, 1111, 1112].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Blaze Wizard", jobs: [1200, 1210, 1211, 1212].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Wind Archer", jobs: [1300, 1310, 1311, 1312].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Night Walker", jobs: [1400, 1410, 1411, 1412].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
      { name: "Thunder Breaker", jobs: [1500, 1510, 1511, 1512].map((jobId) => ({ jobId, minMajorVersion: CYGNUS })) },
    ],
  },
  {
    name: "Legend",
    classes: [
      { name: "Legend", jobs: [{ jobId: 2000, minMajorVersion: ARAN }] },
      { name: "Aran", jobs: [2100, 2110, 2111, 2112].map((jobId) => ({ jobId, minMajorVersion: ARAN })) },
      {
        name: "Evan",
        jobs: [2001, 2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218].map((jobId) => ({ jobId, minMajorVersion: EVAN })),
      },
    ],
  },
  {
    name: "Admin",
    classes: [{ name: "GM", jobs: [900, 910].map((jobId) => ({ jobId, minMajorVersion: ADMIN })) }],
  },
];

/**
 * Prune job-tiers above the tenant's major version, then drop classes with no
 * surviving jobs and archetypes with no surviving classes (FR-2.3). Pure — does
 * not mutate the source tree.
 */
export function filterHierarchy(tree: ArchetypeNode[], major: number): ArchetypeNode[] {
  return tree
    .map((arch) => ({
      name: arch.name,
      classes: arch.classes
        .map((cls) => ({
          name: cls.name,
          jobs: cls.jobs.filter((j) => j.minMajorVersion <= major),
        }))
        .filter((cls) => cls.jobs.length > 0),
    }))
    .filter((arch) => arch.classes.length > 0);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/lib/__tests__/jobs-hierarchy.test.ts`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add src/lib/jobs-hierarchy.ts src/lib/__tests__/jobs-hierarchy.test.ts
git commit -m "feat(atlas-ui): add static job hierarchy + version filter (task-094)"
```

---

## Task 7: Ground `minMajorVersion` against live data (Verification Over Memory)

CLAUDE.md mandates game-data values be verified, not recalled. Probe a live v83
tenant for one representative job per archetype and confirm the floors. This task
**adjusts constants in `jobs-hierarchy.ts` if reality differs** — the unit test
from Task 6 must still pass afterward (it only asserts "v83 hides Cygnus/Legend",
which holds for any floor > 83).

**Files:**
- Possibly modify: `src/lib/jobs-hierarchy.ts` (constants only)

- [ ] **Step 1: Identify a live v83 tenant and its headers**

Find a v83 tenant (region/major/minor) to probe. Reference memory
`reference_atlas_data_wz_inspection` for the throwaway-curl-pod approach, e.g.:

```bash
# From a pod/curl with the four tenant headers set for a known v83 tenant:
#   TENANT_ID, REGION, MAJOR_VERSION=83, MINOR_VERSION
# Adventurer (Warrior Hero) — expect a non-empty skills array:
curl -s -H "TENANT_ID: $T" -H "REGION: $R" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: $M" \
  "$BASE/api/data/jobs/112/skills"
# Cygnus (Dawn Warrior 4), Aran (2112), Evan (2001) — expect empty/404 on v83:
for j in 1112 2112 2001; do
  curl -s -o /dev/null -w "job $j -> %{http_code}\n" \
    -H "TENANT_ID: $T" -H "REGION: $R" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: $M" \
    "$BASE/api/data/jobs/$j/skills"; done
```

- [ ] **Step 2: Record the outcome and adjust constants if needed**

- If Adventurer (112) returns data on v83 → `ADV`/`ADMIN = 83` confirmed.
- If Cygnus/Aran/Evan return empty/404 on v83 → their floors being `> 83` is
  confirmed (any value works for hiding on v83). If a later tenant (v92/v95) is
  available, probe it to tighten `CYGNUS`/`ARAN` to the lowest version that
  returns data, and update the constant + its comment to cite the probe result.
- If reality contradicts an authored floor, edit the `ADV`/`CYGNUS`/`ARAN`/`EVAN`
  constants in `src/lib/jobs-hierarchy.ts` and replace the comment basis with the
  probe finding (date + tenant version). **Do not leave a memory-only number.**

- [ ] **Step 3: Re-run the hierarchy test**

Run: `npm run test -- src/lib/__tests__/jobs-hierarchy.test.ts`
Expected: PASS.

- [ ] **Step 4: Commit (only if constants changed)**

```bash
git add src/lib/jobs-hierarchy.ts
git commit -m "chore(atlas-ui): ground job minMajorVersion against live v83 data (task-094)"
```

If no probe environment is reachable, record that in the task's notes and leave
the cited-basis comments as-is — the mechanism is verified by the unit test; the
integers remain a one-line follow-up fix.

---

## Task 8: `JobsPage` — version-filtered accordion tree

Implements FR-5.1/FR-5.4 and the §3/§4.2 browser. Zero network calls.

**Files:**
- Create: `src/pages/JobsPage.tsx`
- Test: `src/pages/__tests__/JobsPage.test.tsx`

- [ ] **Step 1: Confirm the shadcn collapsible component exists**

Run: `ls src/components/ui/collapsible.tsx`
Expected: exists. (There is **no** `accordion.tsx`; this plan uses `Collapsible`,
which the sidebar already uses — no new dependency. Do not add an accordion.)

- [ ] **Step 2: Write the failing test**

Create `src/pages/__tests__/JobsPage.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { Tenant } from "@/services/api/tenants.service";

const useTenantMock = vi.fn();
vi.mock("@/context/tenant-context", () => ({ useTenant: () => useTenantMock() }));

import { JobsPage } from "@/pages/JobsPage";

const v83 = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as unknown as Tenant;

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
    expect(screen.queryByText("Adventurer")).not.toBeInTheDocument();
  });

  it("renders Adventurer branches but not Cygnus/Legend on a v83 tenant", () => {
    useTenantMock.mockReturnValue({ activeTenant: v83 });
    renderPage();
    expect(screen.getByText("Adventurer")).toBeInTheDocument();
    expect(screen.getByText("Warrior")).toBeInTheDocument();
    expect(screen.queryByText("Cygnus")).not.toBeInTheDocument();
    expect(screen.queryByText("Dawn Warrior")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `npm run test -- src/pages/__tests__/JobsPage.test.tsx`
Expected: FAIL with "Cannot find module '@/pages/JobsPage'".

- [ ] **Step 4: Implement the page**

Create `src/pages/JobsPage.tsx`:

```tsx
import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Briefcase } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { JOB_HIERARCHY, filterHierarchy, jobNodeName } from "@/lib/jobs-hierarchy";

export function JobsPage() {
  const { activeTenant } = useTenant();

  const tree = useMemo(
    () => (activeTenant ? filterHierarchy(JOB_HIERARCHY, activeTenant.attributes.majorVersion) : []),
    [activeTenant],
  );

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
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
          <CardContent className="space-y-2">
            {tree.map((archetype) => (
              <Collapsible key={archetype.name} defaultOpen>
                <CollapsibleTrigger className="text-lg font-semibold py-1">
                  {archetype.name}
                </CollapsibleTrigger>
                <CollapsibleContent className="pl-4 space-y-1">
                  {archetype.classes.map((cls) => (
                    <Collapsible key={cls.name} defaultOpen>
                      <CollapsibleTrigger className="font-medium py-1">{cls.name}</CollapsibleTrigger>
                      <CollapsibleContent className="pl-4 flex flex-wrap gap-2 py-1">
                        {cls.jobs.map((job) => (
                          <Link
                            key={job.jobId}
                            to={`/jobs/${job.jobId}`}
                            className="text-sm text-primary underline-offset-2 hover:underline"
                          >
                            {jobNodeName(job)}
                          </Link>
                        ))}
                      </CollapsibleContent>
                    </Collapsible>
                  ))}
                </CollapsibleContent>
              </Collapsible>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npm run test -- src/pages/__tests__/JobsPage.test.tsx`
Expected: PASS (both cases).

- [ ] **Step 6: Commit**

```bash
git add src/pages/JobsPage.tsx src/pages/__tests__/JobsPage.test.tsx
git commit -m "feat(atlas-ui): add JobsPage hierarchy browser (task-094)"
```

---

## Task 9: `JobDetailPage` — skill list + expandable per-level panel

Implements FR-3.x, FR-4.x, FR-5.3, §4.6/§4.7.

**Files:**
- Create: `src/pages/JobDetailPage.tsx`
- Test: `src/pages/__tests__/JobDetailPage.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `src/pages/__tests__/JobDetailPage.test.tsx`:

```tsx
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
  };
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

  it("renders a skill row with title, master level and a type badge", () => {
    useJobSkillsMock.mockReturnValue({ data: [1101004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [def({})], isLoading: false, isError: false });
    renderAt();
    expect(screen.getByText("Iron Body")).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
    expect(screen.getByText("Passive")).toBeInTheDocument();
  });

  it("falls back to a placeholder icon when the image fails", () => {
    useJobSkillsMock.mockReturnValue({ data: [1101004], isLoading: false, isError: false });
    useJobSkillDefsMock.mockReturnValue({ definitions: [def({})], isLoading: false, isError: false });
    renderAt();
    const img = screen.getByAltText("Iron Body") as HTMLImageElement;
    fireEvent.error(img);
    expect(screen.getByTestId("skill-icon-fallback-1101004")).toBeInTheDocument();
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

Run: `npm run test -- src/pages/__tests__/JobDetailPage.test.tsx`
Expected: FAIL with "Cannot find module '@/pages/JobDetailPage'".

- [ ] **Step 3: Implement the page**

Create `src/pages/JobDetailPage.tsx`:

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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";

function SkillIcon({ def }: { def: SkillDefinitionWithIcon }) {
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
      alt={def.name}
      width={32}
      height={32}
      loading="lazy"
      className="object-contain"
      onError={() => setFailed(true)}
    />
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
  return (
    <Collapsible>
      <div className="flex items-center gap-3 py-2 border-b">
        <SkillIcon def={def} />
        <CollapsibleTrigger asChild>
          <button className="flex-1 text-left">
            <span className="font-medium">{def.name}</span>
          </button>
        </CollapsibleTrigger>
        <Badge variant="secondary">{type}</Badge>
        <span className="text-sm text-muted-foreground w-16 text-right">Lv {def.maxLevel ?? "—"}</span>
      </div>
      <CollapsibleContent className="py-3 pl-11 space-y-3">
        <p className="text-sm">{def.description || "No description available."}</p>
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
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Link to="/jobs" className="text-muted-foreground hover:text-foreground">
          <ChevronLeft className="h-5 w-5" />
        </Link>
        <h2 className="text-2xl font-bold tracking-tight">{jobName}</h2>
        <Badge variant="outline">{jobId}</Badge>
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
              <p className="text-center py-8 text-destructive">Failed to load this job's skills.</p>
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

Run: `npm run test -- src/pages/__tests__/JobDetailPage.test.tsx`
Expected: PASS (all five cases).

- [ ] **Step 5: Commit**

```bash
git add src/pages/JobDetailPage.tsx src/pages/__tests__/JobDetailPage.test.tsx
git commit -m "feat(atlas-ui): add JobDetailPage skill list + per-level table (task-094)"
```

---

## Task 10: Wire routes, sidebar, and breadcrumbs

Implements FR-5.1/FR-5.2 and §6.

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/components/app-sidebar.tsx`
- Modify: `src/lib/breadcrumbs/routes.ts`

- [ ] **Step 1: Add lazy imports + routes in `App.tsx`**

After the `ItemDetailPage` lazy import line, add:

```tsx
const JobsPage = lazy(() => import("@/pages/JobsPage").then(m => ({ default: m.JobsPage })));
const JobDetailPage = lazy(() => import("@/pages/JobDetailPage").then(m => ({ default: m.JobDetailPage })));
```

After the `<Route path="/items/:id" element={<ItemDetailPage />} />` line, add:

```tsx
                    <Route path="/jobs" element={<JobsPage />} />
                    <Route path="/jobs/:jobId" element={<JobDetailPage />} />
```

- [ ] **Step 2: Add the sidebar entry in `app-sidebar.tsx`**

In the `Operations` group's `children` array, after the `Items` entry add:

```tsx
            {
                title: "Jobs",
                url: "/jobs"
            },
```

- [ ] **Step 3: Add breadcrumb configs in `routes.ts`**

Add the import at the top of `src/lib/breadcrumbs/routes.ts`:

```ts
import { getJobNameById } from '@/lib/jobs';
```

Add two entries to `ROUTE_CONFIGS` (near the Item routes):

```ts
  {
    pattern: '/jobs',
    label: 'Jobs',
    parent: '/',
  },
  {
    pattern: '/jobs/[id]',
    label: 'Job Details',
    parent: '/jobs',
    labelResolver: (params) => getJobNameById(Number(params.id)) ?? `Job ${params.id}`,
  },
```

> Note: `JobDetailPage`'s route param is `:jobId`, but the breadcrumb registry
> extracts the first dynamic segment as `params.id` via its `[id]` pattern — so
> the `[id]` pattern above is correct for breadcrumb label resolution.

- [ ] **Step 4: Build to confirm wiring compiles**

Run: `npm run build`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/App.tsx src/components/app-sidebar.tsx src/lib/breadcrumbs/routes.ts
git commit -m "feat(atlas-ui): wire Jobs routes, sidebar, breadcrumbs (task-094)"
```

---

## Task 11: Final verification gate

**Files:** none (verification only).

- [ ] **Step 1: Capture the lint baseline (before-state)**

Run: `npm run lint 2>&1 | tail -5` and note the error count (baseline is
pre-existing-broken per project memory; we gate on **no new** errors, not zero).

- [ ] **Step 2: Run the full test suite**

Run: `npm run test`
Expected: PASS — all new tests plus the pre-existing suite green.

- [ ] **Step 3: Run the production build**

Run: `npm run build`
Expected: PASS (tsc -b type-checks tests too).

- [ ] **Step 4: Confirm no new lint errors**

Run: `npm run lint 2>&1 | tail -5`
Expected: error count ≤ the Step 1 baseline. If new errors appear in any file
this task created/modified, fix them.

- [ ] **Step 5: Manual smoke check (optional, if a dev environment is up)**

Run `npm run dev`, select a v83 tenant, open `/jobs`, confirm Adventurer tree
shows and Cygnus/Legend do not; click a Warrior tier, confirm the skill list
renders with icons + type badges; expand a skill and confirm the per-level table
shows non-zero columns only.

- [ ] **Step 6: Final commit (only if Step 4 required fixes)**

```bash
git add -A
git commit -m "chore(atlas-ui): satisfy lint/build gate for jobs browser (task-094)"
```

---

## Self-Review

**Spec coverage (PRD §4 + design §10):**
- FR-1.x job hierarchy → Task 6 (`JOB_HIERARCHY`, archetype→class→tier, jobId per leaf, names via `getJobNameById`).
- FR-2.x version filter → Task 6 (`filterHierarchy`, empty-branch pruning) + Task 7 (grounded `minMajorVersion`).
- FR-3.1–3.4 skill list → Task 3 (fetch) + Task 9 (rows: icon/title/maxLevel/type/snippet, sorted by id, empty state).
- FR-3.5 skill type → Task 4 (`deriveSkillType`).
- FR-4.x per-level table → Task 5 (`buildLevelTable`, non-zero columns, statup columns, field labels) + Task 9 (render, empty/missing handling).
- FR-5.x nav/loading/errors → Tasks 8, 9, 10 (routes, sidebar, breadcrumbs, no-tenant/loading/error states).
- FR-6.1/6.2 service extension → Task 1 (`maxLevel`, broadened `SkillEffect`).
- NFR multi-tenancy → Tasks 2/3 (tenant-keyed cache via `skillDefinitionKeys`).
- NFR perf → Task 3 (`useQueries`, shared keys, 30-min staleTime), Task 9 (`loading="lazy"`).
- NFR resilience → Task 9 (`onError` icon fallback, `?? "—"`, per-skill isolation).
- Testing → every task is TDD; Task 11 gates build + test + no-new-lint.

**Placeholder scan:** No TBD/TODO/"add error handling" left; every code step shows full code; every test step shows assertions; commands have expected output.

**Type consistency:** `SkillDefinitionWithIcon` (Task 2) is consumed by Tasks 3/9; `fetchSkillDefinitionWithIcon`/`skillDefinitionRetry` (Task 2) used in Task 3; `SkillType`/`deriveSkillType` (Task 4) used in Task 9; `LevelTable`/`LevelColumn`/`buildLevelTable` (Task 5) used in Task 9; `STATUP_LABELS` (Task 5) reused from `skill-effect-format.ts`; `JOB_HIERARCHY`/`filterHierarchy`/`jobNodeName`/`JobNode`/`ArchetypeNode` (Task 6) used in Task 8; `useJobSkills`/`jobSkillsKeys` (existing) used in Task 9. Names match across tasks.
