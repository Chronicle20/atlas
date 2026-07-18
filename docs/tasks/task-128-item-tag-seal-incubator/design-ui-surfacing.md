# task-128 follow-on — surface item-tag / seal / incubator in atlas-ui (+ MTS)

**Status:** design approved, pending spec review → writing-plans.
**Ships on:** the `task-128-item-tag-seal-incubator` branch (extends PR #909).

## Goal

Surface the task-128 backend feature in the admin web-ui and marketplace:

1. **Incubator-rewards admin page** — manage the per-tenant reward pool from the UI (currently only seed-template + REST).
2. **Inventory tag/seal indicators** — show item-tag ownership and sealing-lock status when viewing a character's equipment/inventory.
3. **MTS listing tag/seal indicators** — show the same on items up for sale in the Marketplace.

## Approved decisions

| Decision | Choice |
|---|---|
| Scope | All three (incubator page + inventory indicators + MTS indicators) |
| Indicator UX | Corner icon (lock/tag) + faint gold tint on sealed + tooltip detail |
| Incubator editing | Read-only table + Add/Edit zod dialog + delete confirm + "Seed defaults" button; computed chance % column |
| Ship as | Extend PR #909 (task-128 branch) |

## Data grounding (verified in-repo)

- **Inventory asset REST model** `services/atlas-inventory/atlas.com/inventory/asset/rest.go:12-17` already serves `owner string json:"owner"` (the item-tag name), `flag uint16 json:"flag"`, `expiration time.Time json:"expiration"`, `ownerId uint32`. The UI reads this via `GET /api/characters/{characterId}/inventory` (`services/atlas-ui/src/services/api/inventory.service.ts`).
- **Lock bit** `libs/atlas-constants/asset/flag.go:6` → `FlagLock Flag = 0x01`. Backend derivation: `asset/model.go:81` `Locked() = HasFlag(flag, FlagLock)`.
- **UI `Asset` type** (`inventory.service.ts:58-97`) already has `flag: number` and `expiration: string` but **not** `owner` — needs adding (data already on the wire).
- **MTS listing REST** `services/atlas-mts/atlas.com/mts/listing/rest.go` has `SellerName`, `Flags uint16 json:"flags"` (line 45), but **no owner name**; `listing/model.go:80` stores `flags`. The transfer `AssetData.Flag` (`libs/atlas-saga/payloads.go:512`) is commented "for stackables" — so the equip lock bit and tag owner are **not** reliably carried into the listing snapshot today.
- **atlas-ui MTS listing type** `services/atlas-ui/src/services/api/mts-listings.service.ts:25-66` has `flags: number` but no `owner`.

---

## A. Incubator-rewards admin page (atlas-ui only)

Backend is a **per-tenant collection** with per-row CRUD: `GET /api/tenants/{tenantId}/configurations/incubator-rewards` (list), `POST` (create), `PATCH /{id}`, `DELETE /{id}`, plus `POST /seed`. Each row: `{ itemId: uint32, quantity: uint32, weight: uint32 }` with a string `id`.

Pattern source: mts-config files (service/hook/schema/page/route/nav skeleton) + a true-collection CRUD shape (`api.getList`/`post`/`delete` from `lib/api/client.ts`), rather than mts-config's singleton `getOne`/`patch`.

**New files**
- `src/services/api/incubator-rewards.service.ts`
  - `export const INCUBATOR_REWARDS_RESOURCE_TYPE = "incubator-rewards";`
  - `interface IncubatorRewardAttributes { itemId: number; quantity: number; weight: number; }`
  - `interface IncubatorReward { id: string; attributes: IncubatorRewardAttributes; }`
  - `path(tenantId) = /api/tenants/${tenantId}/configurations/incubator-rewards`
  - `list(tenantId) → getList<IncubatorReward>(path)`; `create(tenantId, attrs) → post(path, {data:{type,attributes}})`; `update(tenantId, id, attrs) → patch(\`${path}/${id}\`, {data:{id,type,attributes}})`; `remove(tenantId, id) → delete(\`${path}/${id}\`)`; `seed(tenantId) → post(\`${path}/seed\`)`.
- `src/lib/hooks/api/useIncubatorRewards.ts`
  - `incubatorRewardsKeys` (tenant-scoped, mirrors `mtsConfigKeys`).
  - `useIncubatorRewards(tenantId)` — list `useQuery`, `enabled: !!tenantId`.
  - `useCreateIncubatorReward`, `useUpdateIncubatorReward`, `useDeleteIncubatorReward`, `useSeedIncubatorRewards` — mutations that `invalidateQueries` on the list key `onSettled`.
- `src/lib/schemas/incubator-rewards.schema.ts`
  - `incubatorRewardSchema = z.object({ itemId: z.number().int().positive(), quantity: z.number().int().positive(), weight: z.number().int().positive() })`; `type IncubatorRewardFormData = z.infer<...>`.
- `src/pages/tenants-incubator-rewards-form.tsx`
  - `tenantId` from `useParams()`.
  - Renders a shadcn `<Table>`: columns **Item** (resolve `itemId` → name via the existing `ItemNameCell` used by the Marketplace), **Qty**, **Weight**, **Chance** (`weight / Σweight` as a %), and a per-row actions cell (`Edit`, delete-`X`).
  - Header actions: **"Seed defaults"** (calls `useSeedIncubatorRewards`, behind an `AlertDialog` confirm since it repopulates the pool) and **"+ Add"**.
  - Add/Edit use a shared `<Dialog>` with a `useForm` + `zodResolver(incubatorRewardSchema)`; three number inputs (`valueAsNumber` coercion); submit calls create/update; toasts on success/error via `sonner` + `createErrorFromUnknown`.
  - Delete uses an `AlertDialog` confirm.
  - Loading / empty guards mirror `tenants-mts-config-form.tsx`.
- `src/pages/TenantsIncubatorRewardsPage.tsx` — `<TenantDetailLayout><IncubatorRewardsForm/></TenantDetailLayout>`.

**Edits**
- `src/App.tsx` — `const TenantsIncubatorRewardsPage = lazy(...)`; `<Route path="/tenants/:id/incubator-rewards" element={<TenantsIncubatorRewardsPage/>} />` inside the AppShell group.
- `src/components/features/tenants/TenantDetailLayout.tsx` — add `{ title: "Incubator Rewards", href: \`/tenants/${id}/incubator-rewards\` }` to `sidebarNavItems`.

**Tests**: service (CRUD URL/envelope), hook (invalidation), schema (positive-int validation), form (add/edit dialog submit, delete confirm, seed confirm, chance computation).

---

## B. Inventory tag/seal indicators (atlas-ui only)

**Data typing**
- `src/services/api/inventory.service.ts` — add `owner: string;` to `Asset.attributes` (after `ownerId`).

**Derivation util**
- New `src/lib/utils/asset-flags.ts`:
  - `export const FLAG_LOCK = 0x01;` (doc comment: mirrors `libs/atlas-constants/asset/flag.go:6`).
  - `isSealed(a: Asset) = (a.attributes.flag & FLAG_LOCK) !== 0`.
  - `isTagged(a: Asset) = a.attributes.owner.trim() !== ""`.
  - `ZERO_DATE = "0001-01-01T00:00:00Z"` reused from the existing tooltip sentinel.

**Cell rendering** (`EquipmentCell.tsx`, `InventoryCard.tsx`)
- Absolute-positioned corner badges over the item image: lucide `Lock` (bottom-right) when sealed; lucide `Tag` (top-right) when tagged.
- When sealed, add a faint gold `ring`/border to the cell container (e.g. `ring-1 ring-amber-400/60`), theme-aware.

**Tooltip** (`AssetTooltipContent.tsx`)
- When tagged: an `Owner: <name>` line.
- When sealed (`isSealed`): a seal line — **"Sealed"** if `expiration === ZERO_DATE`, else **"Sealed until <formatted date>"**. **Suppress the existing "EXPIRES:" line for a sealed item** (locked items unlock at expiry, they are not destroyed — so it must not read as a normal expiration). Non-sealed items with an expiration keep the current "EXPIRES:" behavior unchanged.

**Tests**: `asset-flags` util; `EquipmentCell`/`InventoryCard` badge + gold-ring rendering (tagged / sealed / both / neither); `AssetTooltipContent` line logic for the four cases (tagged, permanent-seal, timed-seal, plain-expiration-not-sealed).

---

## C. MTS listing tag/seal indicators (backend + atlas-ui)

The listing snapshot must carry the item's **owner name** and **lock flag** from the asset at transfer time. Thread both.

**Backend**
- `libs/atlas-saga/payloads.go` — on the TransferToMts listing-snapshot payload (the struct around lines 655-675 with `Flags uint16`), add `Owner string json:"owner"`, and ensure `Flags` is sourced from the equip asset flag (not only the stackable path). Populate both where the payload is built from the asset (atlas-channel and/or atlas-saga-orchestrator transfer step).
- `services/atlas-mts/atlas.com/mts/listing/model.go` — add `owner string` field + `Owner()` getter + builder wiring.
- `services/atlas-mts/atlas.com/mts/listing/rest.go` — add `Owner string json:"owner"` to `RestModel`; map both transform directions.
- Persistence: `owner` column via GORM AutoMigrate; baseline/restore already uses name-keyed column lists so an added column is safe.
- Populate `owner`/`flags` when the listing is created from the transfer payload.

**Frontend**
- `src/services/api/mts-listings.service.ts` — add `owner: string` to `MtsListingAttributes`.
- Marketplace row (`MarketplacePage.tsx` / `ItemNameCell`) — render the `Tag` icon + owner when `owner !== ""`, and the `Lock` icon when `(flags & 0x01) !== 0`, reusing the `asset-flags` helpers (share `FLAG_LOCK`).

**Tests**: atlas-mts listing model/rest owner round-trip; saga payload owner/flag threading; atlas-ui marketplace-row indicator test.

**Build-time verification**: confirm the transfer path actually carries the equip `FlagLock` into `listing.flags`. If it already does, the seal side is frontend-only and only the owner needs threading; if not, thread the flag exactly like owner (as specced above).

---

## Cross-cutting / verification

- **atlas-ui**: run under nvm 22; `npm run build` type-checks `*.test.ts` (update any changed call sites in the same commit); gate on build + test + no-new-lint-errors.
- **Go**: `go build/vet/test` on atlas-mts + any touched service; `docker buildx bake atlas-mts` (and any other service whose code changed).
- **DOM-21**: reuse `libs/atlas-constants/asset` semantics; do not redefine the lock bit in Go — only the UI mirrors `0x01` (documented).
- Code review before PR update (backend-guidelines for the Go changes, frontend-guidelines for the TS changes).

## Risks / open items

- **C is the largest surface** (atlas-mts + saga + transfer path + migration). If de-risking #909 becomes preferable, C can be reduced to seal-only-on-MTS (pending the flag-carry verification) with the tag-owner column deferred — but the approved scope is all three.
- The `flags`-carries-`FlagLock` assumption for equips is the one unverified point; resolved at build time with a clear fallback.
