# Reward Pools UI — representing the merged Gachapon + Incubator systems

**Status:** design approved in-session (2026-07-17), pending spec review → writing-plans.
**Ships on:** the `task-128-item-tag-seal-incubator` branch (extends PR #909).
**Origin:** task-128 reconciled the incubator (Pigmy Egg) onto the gachapon service and
renamed it `atlas-reward-pools` (`design-incubator-gachapon-reconciliation.md`). The UI
was never redesigned around the merge — it is still the pre-merge "Gachapons" surface
with a raw `kind` column bolted on. This doc specifies the proper admin surface.

## Approved decisions

| Decision | Choice |
|---|---|
| Scope | Presentation redesign **and** full pool CRUD (pool + items + global items) |
| Navigation | One **"Reward Pools"** sidebar entry / page; kind filter tabs; `/gachapons` routes redirect |
| Item editing | Add a real `PATCH` item endpoint to `atlas-reward-pools` (no delete+recreate) |
| Global pool | Fully managed in the UI (own tab, CRUD) |
| Ship vehicle | Extend the task-128 branch |
| Architecture | Kind-adaptive single surface — one list route, one detail route, sections branch on `kind` |

## 1. Current-state gaps (grounded)

- Sidebar/nav/breadcrumbs say **"Gachapons"** (`app-sidebar.tsx:61`, `App.tsx:86-87`,
  `breadcrumbs/routes.ts:193-198`) while the service concept is now a reward-pool
  authority with two kinds.
- List page (`GachaponsPage.tsx` + `gachapons-columns.tsx`) renders `kind` as a raw
  string column and shows Common/Uncommon/Rare weight columns that are meaningless
  (zero) for incubator pools. No kind filtering.
- Detail page (`GachaponDetailPage.tsx`) unconditionally renders the Tier Weights card,
  shows NPC ids as bare numbers, and renders the pool from `GET
  /gachapons/{id}/prize-pool` with Item/Qty/Tier columns — no `weight`, no chance %.
- **The detail page is empty for every incubator pool**: `reward/processor.go`
  `GetPrizePool` iterates only the tiers `common/uncommon/rare`, but incubator items
  carry `weight` with no tier, so the merged pool comes back empty ("No items in the
  prize pool"). The reward `RestModel` (`reward/rest.go`) also has no `weight` field.
- The UI `GachaponRewardAttributes` type has no `weight` even though
  `item/rest.go:11` serves it.
- No editing surface anywhere: the branch retired the `incubator-rewards` tenant-config
  form (per the reconciliation design §3.4) but the promised replacement — pool admin
  "folded into the gachapon admin filtered to `kind=incubator`" — was never built. The
  backend's existing write endpoints (pool POST/PATCH/DELETE, item POST/DELETE, global
  item POST/DELETE) are unreachable from the UI.
- The shared **global pool** (`/global-items`, merged into every classic gachapon roll)
  has zero UI.

## 2. Backend deltas (small, same service, same branch)

All in `services/atlas-reward-pools/atlas.com/reward-pools/`:

1. **Item PATCH** — `PATCH /gachapons/{gachaponId}/items/{itemId}` in
   `item/resource.go`, mirroring the gachapon PATCH shape
   (`gachapon/resource.go:28,122-131`): `RegisterInputHandler[RestModel]` →
   `processor.Update(id, itemId, quantity, tier, weight)`. Update mutates
   quantity/tier/weight (and itemId); `gachaponId` is not re-parented.
2. **Global-item PATCH** — `PATCH /global-items/{itemId}` in `global/resource.go`,
   same shape, so the Global Pool tab gets real edits too.
3. **Kind-aware prize pool** — `GetPrizePool` (`reward/processor.go`) branches on
   `gachapon.KindIncubator` exactly like `SelectReward` does: for incubator pools
   return the machine's items with their weights (tier empty); add
   `Weight uint32 \`json:"weight"\`` to `reward/rest.go` and thread it through the
   model/builder. This fixes the endpoint for any consumer even though the redesigned
   UI reads the items endpoints directly (§3.3).
4. **Pool PATCH covers NPC ids** — `handleUpdateGachapon` currently updates only
   `name` + tier weights (`gachapon/resource.go:125`). Extend `Update` to also accept
   `npcIds` so machine locations / the incubator success NPC are editable. `kind` and
   `id` remain immutable after creation.

Verification: `go build/vet/test -race` on the module, `docker buildx bake
atlas-reward-pools`, plus the repo guards (`redis-key-guard`, `goroutine-guard`).

## 3. Frontend design

### 3.1 Naming, routing, navigation

- Sidebar: `Gachapons` → **`Reward Pools`**, url `/reward-pools` (`app-sidebar.tsx:61`).
- Routes in `App.tsx`: `RewardPoolsPage` at `/reward-pools`, `RewardPoolDetailPage` at
  `/reward-pools/:id`; `<Navigate replace>` redirects from `/gachapons` and
  `/gachapons/:id` (deep links + muscle memory keep working).
- Breadcrumbs (`lib/breadcrumbs/routes.ts`): replace the `/gachapons` entries with
  `/reward-pools` ("Reward Pools") and add `/reward-pools/[id]` (label = pool name).
- Setup page (`SetupPage.tsx:186`): label "Gachapons" → "Reward Pools". The seed group
  name and the REST base path stay `gachapons` — the backend deliberately kept the
  resource identity (reconciliation design §4); the UI maps the concept, not the URL.

### 3.2 Types, services, hooks

Rename/extend the existing modules (straight moves, no aliases):

- `types/models/reward-pool.ts` — `RewardPoolKind = "gachapon" | "incubator"`;
  `RewardPoolAttributes { name; kind: RewardPoolKind; npcIds: number[];
  commonWeight; uncommonWeight; rareWeight }`; `RewardPoolData`.
- `types/models/reward-pool-item.ts` — `{ gachaponId: string; itemId: number;
  quantity: number; tier: string; weight: number }` (matches `item/rest.go`; row id =
  the numeric item record id from `GetID()`).
- `types/models/global-reward-item.ts` — `{ itemId; quantity; tier }`.
- `services/api/reward-pools.service.ts` (replaces `gachapons.service.ts`; base path
  stays `/api/gachapons`, plus `/api/global-items`):
  - pools: `getAll` (drain-all), `getById`, `create`, `update`, `remove`
  - items: `getItems(poolId)`, `createItem`, `updateItem`, `removeItem`
  - global: `getGlobalItems`, `createGlobalItem`, `updateGlobalItem`, `removeGlobalItem`
  - Writes to `RegisterInputHandler` endpoints use the JSON:API envelope
    `{data: {type, attributes}}` with the verified resource types: `"gachapons"`,
    `"gachapon-items"`, `"global-gachapon-items"` (each `RestModel.GetName()`).
- `lib/hooks/api/useRewardPools.ts` — tenant-safe query keys (`rewardPoolKeys`),
  list/detail/items/global queries, and mutations that invalidate the affected keys
  `onSettled`. The pool collection is small (a handful of machines + ten eggs), so the
  list uses **drain-all** (`fetchAll`, task-117) instead of server paging — tab
  filtering and counts are then exact on the client. Drop `useGachaponsPage`/`Pager`.
- `lib/schemas/reward-pools.schema.ts` — zod:
  - pool: `name` non-empty; `kind` enum (create only); gachapon → three tier weights
    `int ≥ 0` with `sum > 0`; incubator → `id` (egg item id) required numeric string
    on create; `npcIds` array of positive ints.
  - item: `itemId` positive int; `quantity` positive int; gachapon-kind → `tier`
    enum(`common|uncommon|rare`); incubator-kind → `weight` positive int.
- Chance math in a pure util `lib/utils/reward-pool-chance.ts` (unit-testable),
  mirroring `reward/processor.go` `selectItem` exactly:
  - incubator: `weight / Σ(pool weights)`.
  - gachapon: `tierChance(tier) × withinTier`, where `tierChance = tierWeight /
    (common+uncommon+rare)` and the within-tier pool = machine items of that tier
    **plus global items of that tier (always weight 0** —
    `getMergedPool`, `reward/processor.go:139-142`). Within-tier:
    if the pool's Σweight > 0 → `weight / Σweight` (zero-weight items, including
    all global items, get **0%**); else uniform `1 / N`.
  - The detail page surfaces the footgun: when a gachapon tier mixes weighted and
    zero-weight items, show a warning banner on that tier group ("weighted items
    exclude the unweighted ones from this tier's roll").

### 3.3 List page — `RewardPoolsPage`

shadcn `Tabs`: **All · Gachapons · Incubators · Global Pool**.

- Pool tabs share one `DataTableWrapper` over the client-filtered pool list. Columns:
  - **Name** — for incubator pools, the egg's item icon (`getAssetIconUrl`) + resolved
    item name (`useItemName(pool.id)` — the pool id *is* the egg item id) with the
    seed name as fallback; for gachapons, the machine name. Links to detail.
  - **Kind** — badge, not raw text: `secondary` "Gachapon", incubator gets a
    distinct-variant badge "Incubator".
  - **Details** — kind-appropriate summary: gachapon → tier-weight ratio
    (`C/U/R 70·25·5`) and NPC count; incubator → item count is *not* shown (would
    require N+1 fetches); show the egg id instead.
- Header: **"New Pool"** button → create dialog. Step 1 picks kind; fields then adapt
  (gachapon: name, tier weights, npcIds; incubator: egg item id — which becomes the
  pool id — plus name/region label and success NPC id).
- **Global Pool tab**: table of global items — Item (icon + name link), Qty, Tier
  badge — with Add / Edit / Delete via the shared item dialog (tier variant), plus a
  short caption explaining these merge into every gachapon machine's roll (they never
  apply to incubators, per `SelectReward`).
- Empty state per tab (e.g. incubators: "No incubator pools — seed defaults from
  Setup, or create one").

### 3.4 Detail page — `RewardPoolDetailPage` (kind-adaptive)

Header: pool name + kind badge + monospace id. For incubators the header shows the
egg item icon + resolved name linking to `/items/{id}`.

- **Gachapon sections:**
  - *Tier Weights card* — three rows with weight **and computed tier %**; Edit button
    → pool dialog (name + weights + npcIds).
  - *NPCs card* — resolved NPC names via `useNPC`, chips linking `/npcs/{id}`;
    editable in the same pool dialog.
  - *Pool table* — from `getItems` + `getGlobalItems`, grouped by tier. Columns:
    Item (icon + name, links `/items/{id}`), Qty, Tier badge, **Chance %** (util
    above; tooltip shows the raw math), Actions (Edit/Delete). Global-sourced rows
    are badged **"Global"** and read-only here (edited on the Global Pool tab).
- **Incubator sections:**
  - *Egg card* — egg item (icon/name/id), success NPC (npcIds[0], resolved + linked),
    total pool weight. Edit → pool dialog (name + success NPC).
  - *Pool table* — Item, Qty, **Weight**, **Chance %** (`weight/Σweight`), Actions.
    No tier column, no global rows.
- **Shared:** "Add Item" header button → item dialog (kind decides tier-vs-weight
  field); Delete-item `AlertDialog`; a Danger Zone card with "Delete Pool"
  (`AlertDialog`, navigates back to the list on success). Toasts via `sonner` +
  `createErrorFromUnknown`; loading/empty guards as on the current pages.

Files removed: `GachaponsPage.tsx`, `GachaponDetailPage.tsx`, `gachapons-columns.tsx`,
`gachapons.service.ts`, `useGachapons.ts`, `types/models/gachapon{,-reward}.ts` and
their tests — replaced by the reward-pool modules above (SetupPage's seed hooks keep
working; only the label changes).

## 4. Tests

- **Backend:** item/global PATCH handler + processor tests; `GetPrizePool` incubator
  branch (weighted items returned, weight in rest model); pool `Update` npcIds.
- **Frontend (Vitest):**
  - service: URL + JSON:API envelope per write; drain-all list.
  - hooks: mutation invalidation of `rewardPoolKeys`.
  - `reward-pool-chance` util: incubator weights; gachapon tier×uniform incl. global
    counts; zero-sum guards.
  - list page: tab filtering, kind badges, incubator egg-name rendering, Global Pool
    tab CRUD wiring.
  - detail page: gachapon vs incubator section branching; global rows badged and
    non-editable; chance column values.
  - dialogs: zod validation (kind-adaptive fields), submit → mutation payload.
  - routing: `/gachapons` → `/reward-pools` redirect.
- Gates: nvm 22, `npm run build` (type-checks tests), `npm run test`, no new lint
  errors; Go gates per §2.

## 5. Non-goals

- No rename of the REST base path (`/gachapons`), resource types, Kafka topic, or DB —
  deliberate (reconciliation design §4/§5).
- No player-facing drop-rate simulator or roll-testing button.
- No `incubatorInfo.img` ingestion — egg/region labels remain seed-provided
  (`design-incubator-pigmy.md` Phase 7 stays deferred).
- No server-side `filter[kind]` query param — client-side filtering over the drained
  collection is sufficient at this cardinality.
