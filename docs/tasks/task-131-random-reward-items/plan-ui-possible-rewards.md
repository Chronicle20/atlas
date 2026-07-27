# Possible Rewards (item detail page) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On the atlas-ui item detail page, show a "Possible Rewards" card for reward-box items (items with a reward table), listing each possible reward with its computed drop chance.

**Architecture:** Frontend-only. The reward array already reaches the browser via the generic JSON:API deserializer in `itemsService.getConsumable` — it is only missing from the TypeScript type. Add the type, a self-contained `PossibleRewardsCard` component (one row per reward, each row resolving its own item name/icon via `useItemData`), and render it conditionally inside the existing `ConsumableSection`.

**Tech Stack:** Vite + React 19 + TypeScript, TanStack React Query, react-router-dom v7, shadcn/ui (Card, Badge), Tailwind 4, Vitest + `@testing-library/react`.

Spec: `docs/tasks/task-131-random-reward-items/design-ui-possible-rewards.md`.

## Global Constraints

- **Frontend-only.** No backend, `atlas-data`, API, or `items.service` change. No change to the reward-*use* flow (task-131 core).
- **`prob` is a weight, not a percentage.** Render each reward's chance as `prob / Σprob` over that item's own reward list; guard `Σprob === 0` (no NaN). Never render raw `prob` as a percent.
- **Node 22 for atlas-ui commands.** Run `nvm use 22` (source nvm first) before `npm` commands. Work from `services/atlas-ui/`.
- **Rich row detail** (per approval): icon + name (links to `/items/{itemId}`), chance % + raw weight, `×count` when `count > 1`, `time-limited` badge when `period > 0`, `announces` badge when `worldMsg` is non-empty. Do not surface the `effect` string.
- **Named exports; `@/` alias; plain `<img>` with width/height/loading; `vi.*` (not `jest.*`) in new tests.**

## File Structure

- Modify `src/types/models/item.ts` — add `RewardModel` + `rewards?` on `ConsumableAttributes`.
- Create `src/components/features/items/PossibleRewardsCard.tsx` — the card + per-row widget.
- Create `src/components/features/items/__tests__/PossibleRewardsCard.test.tsx` — component test.
- Modify `src/pages/ItemDetailPage.tsx` — render the card in `ConsumableSection`.

All paths below are relative to `services/atlas-ui/`.

---

### Task 1: Reward type + `PossibleRewardsCard` component (TDD)

**Files:**
- Modify: `src/types/models/item.ts` (add `RewardModel`; add `rewards?` to `ConsumableAttributes` at `:89-102`)
- Create: `src/components/features/items/PossibleRewardsCard.tsx`
- Test: `src/components/features/items/__tests__/PossibleRewardsCard.test.tsx`

**Interfaces:**
- Consumes: `useItemData(itemId: number)` from `@/lib/hooks/useItemData` → `{ name?: string; iconUrl?: string; isLoading: boolean }`. `Card/CardContent/CardHeader/CardTitle` from `@/components/ui/card`. `Badge` from `@/components/ui/badge` (variants include `secondary`).
- Produces:
  - `RewardModel` (in `@/types/models/item`): `{ itemId: number; count: number; prob: number; effect: string; worldMsg: string; period: number }`.
  - `ConsumableAttributes.rewards?: RewardModel[]`.
  - `PossibleRewardsCard({ rewards }: { rewards: RewardModel[] })` — React component; renders `null` when `rewards` is empty.

- [ ] **Step 1: Add the `RewardModel` type and `rewards` field**

In `src/types/models/item.ts`, add the interface just before `ConsumableAttributes` (currently at `:89`):

```ts
export interface RewardModel {
  itemId: number;
  count: number;
  prob: number;
  effect: string;
  worldMsg: string;
  period: number;
}
```

Then add `rewards` to `ConsumableAttributes` (after `spec`):

```ts
export interface ConsumableAttributes {
  price: number;
  unitPrice: number;
  slotMax: number;
  reqLevel: number;
  quest: boolean;
  tradeBlock: boolean;
  notSale: boolean;
  timeLimited: boolean;
  success: number;
  cursed: number;
  rechargeable: boolean;
  spec: Record<string, number>;
  rewards?: RewardModel[];
}
```

- [ ] **Step 2: Write the failing component test**

Create `src/components/features/items/__tests__/PossibleRewardsCard.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { PossibleRewardsCard } from "../PossibleRewardsCard";
import type { RewardModel } from "@/types/models/item";

// Mock the data hook so no tenant/network is needed; name echoes the itemId.
vi.mock("@/lib/hooks/useItemData", () => ({
  useItemData: (itemId: number) => ({
    name: `Item ${itemId}`,
    iconUrl: undefined,
    isLoading: false,
  }),
}));

function mk(over: Partial<RewardModel>): RewardModel {
  return { itemId: 1000, count: 1, prob: 100, effect: "", worldMsg: "", period: 0, ...over };
}

function wrap(children: React.ReactNode) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe("PossibleRewardsCard", () => {
  it("renders nothing when there are no rewards", () => {
    const { container } = render(wrap(<PossibleRewardsCard rewards={[]} />));
    expect(container.firstChild).toBeNull();
  });

  it("computes chance as prob/total and shows the count in the title", () => {
    render(wrap(<PossibleRewardsCard rewards={[mk({ itemId: 1, prob: 30 }), mk({ itemId: 2, prob: 10 })]} />));
    expect(screen.getByText("Possible Rewards (2)")).toBeInTheDocument();
    expect(screen.getByText("75.0%")).toBeInTheDocument(); // 30 / 40
    expect(screen.getByText("25.0%")).toBeInTheDocument(); // 10 / 40
  });

  it("sorts rows by chance descending", () => {
    render(wrap(<PossibleRewardsCard rewards={[mk({ itemId: 1, prob: 10 }), mk({ itemId: 2, prob: 90 })]} />));
    const pcts = screen.getAllByText(/%$/).map((el) => el.textContent);
    expect(pcts).toEqual(["90.0%", "10.0%"]);
  });

  it("guards total=0 without producing NaN", () => {
    render(wrap(<PossibleRewardsCard rewards={[mk({ itemId: 1, prob: 0 }), mk({ itemId: 2, prob: 0 })]} />));
    expect(screen.getAllByText("0.0%").length).toBe(2);
  });

  it("shows raw weight, ×count, time-limited and announces when applicable", () => {
    render(wrap(<PossibleRewardsCard rewards={[mk({ itemId: 1, prob: 100, count: 3, period: 7200, worldMsg: "/name got /item" })]} />));
    expect(screen.getByText("w100")).toBeInTheDocument();
    expect(screen.getByText("×3")).toBeInTheDocument();
    expect(screen.getByText("time-limited")).toBeInTheDocument();
    expect(screen.getByText("announces")).toBeInTheDocument();
  });

  it("omits ×count, time-limited and announces when not applicable", () => {
    render(wrap(<PossibleRewardsCard rewards={[mk({ itemId: 1, count: 1, period: 0, worldMsg: "" })]} />));
    expect(screen.queryByText("time-limited")).toBeNull();
    expect(screen.queryByText("announces")).toBeNull();
    expect(screen.queryByText(/^×/)).toBeNull();
  });
});
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
cd services/atlas-ui && nvm use 22 >/dev/null && npx vitest run src/components/features/items/__tests__/PossibleRewardsCard.test.tsx
```
Expected: FAIL — `Failed to resolve import "../PossibleRewardsCard"` (component not created yet).

- [ ] **Step 4: Implement `PossibleRewardsCard`**

Create `src/components/features/items/PossibleRewardsCard.tsx`:

```tsx
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useItemData } from "@/lib/hooks/useItemData";
import type { RewardModel } from "@/types/models/item";

interface PossibleRewardsCardProps {
  rewards: RewardModel[];
}

interface RewardRow extends RewardModel {
  chance: number; // 0..1, computed from prob / Σprob
}

export function PossibleRewardsCard({ rewards }: PossibleRewardsCardProps) {
  if (rewards.length === 0) return null;

  const total = rewards.reduce((sum, r) => sum + r.prob, 0);
  const rows: RewardRow[] = rewards
    .map((r) => ({ ...r, chance: total > 0 ? r.prob / total : 0 }))
    .sort((a, b) => b.chance - a.chance);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          Possible Rewards ({rewards.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {rows.map((row, idx) => (
            <RewardRowWidget key={`${row.itemId}-${idx}`} reward={row} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function RewardRowWidget({ reward }: { reward: RewardRow }) {
  const { name, iconUrl, isLoading } = useItemData(reward.itemId);
  const pct = (reward.chance * 100).toFixed(1);
  const displayName =
    isLoading && !name ? `Item #${reward.itemId}` : name || `Item #${reward.itemId}`;

  return (
    <Link
      to={`/items/${reward.itemId}`}
      className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors"
    >
      <div className="h-8 w-8 shrink-0 flex items-center justify-center">
        {iconUrl && (
          <img
            src={iconUrl}
            alt={name || String(reward.itemId)}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {displayName}
          {reward.count > 1 && (
            <span className="ml-1 text-muted-foreground">×{reward.count}</span>
          )}
        </p>
        <p className="text-xs font-mono text-muted-foreground">{reward.itemId}</p>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {reward.period > 0 && <Badge variant="secondary">time-limited</Badge>}
        {reward.worldMsg !== "" && <Badge variant="secondary">announces</Badge>}
        <div className="text-right">
          <p className="text-sm font-medium tabular-nums">{pct}%</p>
          <p className="text-xs font-mono text-muted-foreground">w{reward.prob}</p>
        </div>
      </div>
    </Link>
  );
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
cd services/atlas-ui && nvm use 22 >/dev/null && npx vitest run src/components/features/items/__tests__/PossibleRewardsCard.test.tsx
```
Expected: PASS (6 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/types/models/item.ts \
        services/atlas-ui/src/components/features/items/PossibleRewardsCard.tsx \
        services/atlas-ui/src/components/features/items/__tests__/PossibleRewardsCard.test.tsx
git commit -m "feat(task-131): PossibleRewardsCard component + RewardModel type (ui add-on)"
```

---

### Task 2: Wire the card into the item detail page + verify

**Files:**
- Modify: `src/pages/ItemDetailPage.tsx` (import `PossibleRewardsCard`; render in `ConsumableSection`, currently at `:263-311`)

**Interfaces:**
- Consumes: `PossibleRewardsCard` from Task 1; `ConsumableData.attributes.rewards?: RewardModel[]`.
- Produces: no new exports.

- [ ] **Step 1: Import the component**

In `src/pages/ItemDetailPage.tsx`, add to the item-feature imports (next to `RecipesByItemCard`, around `:27`):

```tsx
import { PossibleRewardsCard } from "@/components/features/items/PossibleRewardsCard";
```

- [ ] **Step 2: Render the card in `ConsumableSection`**

In the `ConsumableSection` function, inside the returned `<>...</>`, add the card after the existing Scroll Effects / Spec cards (i.e., just before the closing `</>`):

```tsx
      {a.rewards && a.rewards.length > 0 && (
        <PossibleRewardsCard rewards={a.rewards} />
      )}
```

- [ ] **Step 3: Type-check + build the app**

```bash
cd services/atlas-ui && nvm use 22 >/dev/null && npm run build
```
Expected: PASS — `tsc -b` resolves `a.rewards` (now typed) and the new component; `vite build` succeeds. If `tsc` complains that `rewards` is possibly undefined, the `a.rewards && a.rewards.length > 0` guard already narrows it; confirm the guard is present.

- [ ] **Step 4: Run the full test + lint gates**

```bash
cd services/atlas-ui && nvm use 22 >/dev/null && npx vitest run src/components/features/items/ && npm run lint
```
Expected: component tests PASS; `npm run lint` introduces **no new** errors versus the pre-change baseline (the repo has pre-existing lint warnings — compare counts, do not require zero).

- [ ] **Step 5: Manual smoke (optional but recommended)**

Run `npm run dev`, open a reward-box item's detail page (e.g. `/items/2022503` Golden Pig's Egg, or `/items/2022309`) on a v83 tenant. Expect the "Possible Rewards" card listing rewards with percentages that sum to ~100%. Open a plain consumable (no reward table) and confirm **no** card appears.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/ItemDetailPage.tsx
git commit -m "feat(task-131): render PossibleRewardsCard on the item detail page (ui add-on)"
```

---

## Self-Review

- **Spec coverage:** type addition (§3.1) → Task 1 Step 1; component with chance=prob/Σprob, sort, rich rows, empty→null (§3.2) → Task 1 Steps 2/4; wire-in gated on `rewards.length` (§3.3) → Task 2 Steps 1/2; tests (§3.4) → Task 1 Step 2; verification (§5) → Task 2 Steps 3/4/5. All covered.
- **Placeholder scan:** none — every step has concrete code/commands.
- **Type consistency:** `RewardModel`/`ConsumableAttributes.rewards` defined in Task 1 Step 1 are used verbatim in the component (Task 1 Step 4) and the page wire-in (Task 2 Step 2); `PossibleRewardsCard` prop shape (`{ rewards: RewardModel[] }`) matches its usage.
- **Weight-not-percent** invariant is enforced in code (Task 1 Step 4) and asserted in tests (Task 1 Step 2, "computes chance" + "guards total=0").
