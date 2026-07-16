# Task 131 add-on — UI: surface possible random rewards on the item detail page

Status: Approved for planning
Created: 2026-07-15
Parent: task-131 (random reward items); this is a frontend-only add-on on the
`task-131-random-reward-items` branch.

---

## 1. Summary

When an item carries a reward table (a reward box — see task-131), the atlas-ui
**item detail page** should show the possible random rewards and each one's
chance, instead of leaving the box opaque. The reward data is already served by
the backend and already reaches the browser; this add-on types it and renders a
card. Frontend-only: no backend, API, or reward-*use*-flow change.

## 2. Data — already available, just untyped

`atlas-data`'s consumable endpoint returns per-item:

```
attributes.rewards: [{ itemId, count, prob, effect, worldMsg, period }]
```

(`services/atlas-data/atlas.com/data/consumable/rest.go:102,125`.)
`itemsService.getConsumable` deserializes via the generic JSON:API path
(`api.getOne<ConsumableData>('/api/data/consumables/{id}')`), which passes the
whole `attributes` object through — so `rewards` is present at runtime today; the
only gap is the TypeScript type (`ConsumableAttributes` in
`src/types/models/item.ts` omits it).

### 2.1 `prob` is a weight, not a percentage (key correctness point)

The task-131 WZ sweep (design §2.5) found reward `prob` sums like 20 … 19,864
with no common denominator — the values are **weights**, not authored
percentages. So the UI must compute each reward's chance as `prob / Σprob`, never
render the raw `prob` as a percent. `Σprob` is computed per item over its own
reward list.

## 3. Changes (frontend-only)

### 3.1 Type — `src/types/models/item.ts`

- Add:
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
- Add `rewards?: RewardModel[]` to `ConsumableAttributes` (optional; absent/old
  data → treat as `[]`).

No `items.service` change — the generic deserializer already carries the field;
this only makes it visible to the type system.

### 3.2 New component — `src/components/features/items/PossibleRewardsCard.tsx`

Mirrors the existing `RecipesByItemCard` / `DroppedByWidget` patterns.

- Props: `{ rewards: RewardModel[] }`.
- Renders `null` when `rewards.length === 0` (parent also gates, belt-and-braces).
- `total = Σ rewards[].prob`. Chance for a row = `total > 0 ? prob / total : 0`,
  formatted to **3 decimal places** (e.g. `12.400%`). *(Refined during
  implementation from the originally-specified 1 decimal: the rarest canonical
  reward is ~0.005% (1 / 19,864), which 1–2 decimals would round to a false
  `0.0%`/`0.01%`. 3 decimals renders it faithfully with nothing rounding to a
  false `0.000%`.)*
- Rows sorted by chance **descending** (stable; ties keep input order).
- Card header: `Possible Rewards (N)` where `N = rewards.length`.
- **Per-row content (Rich detail level, per approval):**
  - Item **icon + name** via `useItemData(itemId)` (arg is a `number`;
    returns `{ name?, iconUrl?, isLoading }`), the row linking to `/items/{itemId}`
    (link pattern from `MonsterDropWidget` / `RecipesByNpcCard`). Per-row loading
    fallback (icon placeholder + id) like `DroppedByWidget`.
  - **Chance** only, e.g. `12.400%`. *(The raw weight `· w9900` and the item id
    were dropped during implementation as noise — the computed chance is the
    player-facing signal; the underlying weight/id add clutter without value.)*
  - **`×count`** shown only when `count > 1`.
  - **`time-limited`** badge when `period > 0`.
  - **"announces"** indicator (small badge/icon) when `worldMsg` is non-empty —
    signals the reward triggers a world-wide message on drop.
  - `effect` is **not** surfaced (server-internal playback path).

### 3.3 Wire-in — `src/pages/ItemDetailPage.tsx` (`ConsumableSection`)

Add, alongside the existing conditional cards (Scroll Effects, Spec):

```tsx
{a.rewards && a.rewards.length > 0 && (
  <PossibleRewardsCard rewards={a.rewards} />
)}
```

### 3.4 Tests

- `src/components/features/items/__tests__/PossibleRewardsCard.test.tsx`,
  mirroring `RecipesByItemCard.test.tsx`:
  - chance is computed from weights (`prob/Σprob`), not read raw;
  - rows sorted by chance descending;
  - `×count` shown only when `count > 1`; `time-limited` only when `period > 0`;
    "announces" only when `worldMsg` non-empty;
  - `total === 0` guard (no divide-by-zero / NaN);
  - empty `rewards` → renders nothing.
  - `useItemData` mocked (as sibling tests mock their data hooks).
- If the item-model test enumerates attributes, extend it for `rewards`.

## 4. Non-goals

- No backend / atlas-data / API change.
- No change to the reward-*use* flow (task-131 core — already implemented).
- No raw `effect` / `worldMsg` string dump (only the derived "announces"
  indicator).
- No new network calls beyond what `useItemData` already performs per item.

## 5. Verification

- `atlas-ui` build (`tsc -b` type-checks tests too — project memory), lint (no
  new errors vs baseline), and the new component test pass (source nvm 22 first).
- Manual: open a reward-box item's detail page (e.g. `2022503` Golden Pig's Egg,
  `2022309`) — the card lists rewards with sane percentages summing to ~100%, and
  a non-reward consumable shows no card.
