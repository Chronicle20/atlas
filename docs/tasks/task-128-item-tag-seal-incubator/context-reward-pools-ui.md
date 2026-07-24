# Context — Reward Pools UI (plan-reward-pools-ui.md)

Key files, decisions, and dependencies for executing `plan-reward-pools-ui.md`.
Design: `design-reward-pools-ui.md`. Everything below was verified against source
on this branch (`task-128-item-tag-seal-incubator`) on 2026-07-17.

## Why this exists

task-128 reconciled the incubator (Pigmy Egg) onto the gachapon service
(`design-incubator-gachapon-reconciliation.md`) and renamed it
`atlas-reward-pools`, but the UI is still the pre-merge "Gachapons" surface and
the branch **deleted the only incubator editing UI** (`incubator-rewards` tenant
form, removed in `5da125b76` / `acf149ca7`) without a replacement. This plan
builds the replacement: one kind-adaptive "Reward Pools" surface with full CRUD.

## Approved decisions (user, 2026-07-17)

1. Scope: presentation redesign **and** full CRUD (pools + items + global items).
2. One "Reward Pools" sidebar entry; `/reward-pools` routes; `/gachapons` redirects.
3. Item edits get a real backend PATCH (no delete+recreate).
4. Global pool fully managed in the UI.
5. Ships on the task-128 branch (extends PR #909).
6. Kind-adaptive single surface (one list route, one detail route).

## Backend map (`services/atlas-reward-pools/atlas.com/reward-pools/`)

- Module name is `atlas-reward-pools` (short form, like atlas-transports).
- `gachapon/` — pool resource. `RestModel.GetName() = "gachapons"`; attrs
  `name, npcIds []uint32, commonWeight, uncommonWeight, rareWeight, kind`.
  `kind ∈ {"gachapon","incubator"}` (`gachapon.KindIncubator`); entity stores
  npcIds as a custom `int64Array` (`type:integer[]`); pool id is a client-suppliable
  string (`handleCreateGachapon` uses `rm.Id` — incubator pool id = egg item id).
  Existing routes: GET all (paged), POST, GET/PATCH/DELETE by id.
  **PATCH currently updates only name+weights** (`processor.go:50`, `administrator.go:41`) — Task 3 adds npcIds.
- `item/` — per-pool items at `/gachapons/{gachaponId}/items`.
  `RestModel.GetName() = "gachapon-items"`; numeric autoincrement record id
  (`GetID()` = stringified); attrs `gachaponId, itemId, quantity, tier, weight`.
  Routes: GET (paged, optional `?tier=`), POST, DELETE `/{itemId}`.
  **No PATCH** — Task 1 adds it. `rest.ParseItemId` already exists (`rest/handler.go:109`).
- `global/` — shared pool at `/global-items`. `GetName() = "global-gachapon-items"`;
  attrs `itemId, quantity, tier` — **no weight** (merged into gachapon rolls with
  weight 0, `reward/processor.go` `getMergedPool`). Routes: GET/POST/DELETE.
  **No PATCH** — Task 2 adds it.
- `reward/` — the roll. `SelectReward` branches on `KindIncubator` (whole-machine
  weighted, no tiers, no global merge) vs classic (selectTier → merged pool →
  `selectItem`: weight-proportional if Σweight>0 else uniform).
  **`GetPrizePool` does NOT branch on kind** → returns empty for incubator pools;
  `reward/rest.go` has no `weight`. Task 4 fixes both. atlas-channel only calls
  `rewards/select` (`channel/incubator/requests.go`) — additive change is safe.
- Tests: `test.SetupTestDB` (sqlite `file::memory:?cache=shared`),
  `test.CreateTestContext()`, `test.TestTenantId`, per-domain fixtures
  `test.Create{Gachapon,Item,Global,Reward}Processor(t)` → `(processor, db, cleanup)`.

## Frontend map (`services/atlas-ui/src/`)

Current (to be replaced in Phase 4): `pages/GachaponsPage.tsx` (paged table),
`pages/GachaponDetailPage.tsx` (always shows Tier Weights; pool via prize-pool →
empty for incubators), `pages/gachapons-columns.tsx` (raw kind column),
`services/api/gachapons.service.ts`, `lib/hooks/api/useGachapons.ts`,
`types/models/gachapon{,-reward}.ts`.

Integration points:
- Sidebar entry: `components/app-sidebar.tsx:61`. Routes: `App.tsx:23-24,86-87`.
  Breadcrumbs: `lib/breadcrumbs/routes.ts:193-198` + `GACHAPONS` const at `:488`.
  Setup label: `pages/SetupPage.tsx:186` (seed hooks in `useSeed.ts` stay — backend
  seed group name remains `"gachapons"`).
- `ItemNameCell` (`components/item-name-cell.tsx`): props `{ itemId: string; tenant: Tenant | null }`.
- `useItemName(itemId: string)` (`lib/hooks/api/useItemStrings.ts`) → item name string.
- `useNPC(tenant, npcId)` (`lib/hooks/api/useNpcs.ts:57`).
- `getAssetIconUrl(tenantId, region, majorVersion, minorVersion, "item", numericId)`
  (`lib/utils/asset-url.ts:5`; call shape in `components/features/items/ItemHeader.tsx:28-35`).
- shadcn on hand: `tabs`, `dialog`, `alert-dialog`, `alert`, `table`, `badge`,
  `tooltip`. **Check `radio-group` exists before Task 9**; fall back to two styled
  buttons or add the shadcn radio-group if absent.
- Retired CRUD-dialog pattern source:
  `git show 5da125b76~1:services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx`.
- JSON:API envelope required on all writes: `{data:{id?,type,attributes}}`
  (known bug pattern: bare attrs → 400).
- Drain-all via `fetchAll` from `services/api/pagination` (pool collection is
  small; tabs need exact counts; drops `useGachaponsPage`/`Pager`).

## Chance math (must mirror `reward/processor.go` exactly)

- Incubator: `weight / Σweight`; zero-total → 0.
- Gachapon: `tierWeight/(c+u+r)` × within-tier share, where the within-tier pool
  = machine items + global items (weight 0). If tier Σweight > 0 →
  `weight/Σweight` (zero-weight rows: 0%, "excluded"); else uniform `1/N`.
- Footgun surfaced in UI: a gachapon tier mixing weighted and zero-weight rows
  silently excludes the zero-weight rows (including ALL global items) — warning
  banner per affected tier.

## Verification gates

- Go: build/vet/test -race in the module; `docker buildx bake atlas-reward-pools`
  (mandatory); `tools/redis-key-guard.sh` + `tools/goroutine-guard.sh`.
- UI: nvm 22; `npm run build` (type-checks tests) + `npm run test`; lint gate is
  no-NEW-errors (baseline pre-broken).
- Code review before updating PR #909: backend-guidelines-reviewer +
  frontend-guidelines-reviewer + plan-adherence-reviewer (pin to Sonnet).

## Non-goals (do not do)

- No REST path / resource-type / Kafka topic / DB renames.
- No server-side `filter[kind]`.
- No `incubatorInfo.img` ingestion; egg/region labels stay seed-provided.
- No player-facing roll simulator.
