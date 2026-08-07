# Inkwell Token Shop — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-06
---

## 1. Overview

Inkwell (NPC `9000069`) is a token vendor: his commodities are paid for with Perfect Pitch
(item `4310000`) rather than mesos. Today his shop opens in-game with **zero items listed**, and the
Web UI shows all nine commodities priced as "Free". Both symptoms trace to the same seed data, and a
second latent defect would block the feature even after the data is corrected.

**Defect A — data.** `deploy/seed/**/npc-shops/shops/shop-9000069.json` seeds all nine commodities
with `mesoPrice: 0`, `tokenPrice: 0`, `tokenTemplateId: 0` on every one of the eleven version
directories. It is the only shop in the entire seed catalog that has commodities but no prices. The
GMS v83 client discards such rows: `CShopDlg::SetShopDlg` @ `0x7529ad` decodes each commodity but
only inserts it into the buy list under `if ( v99 || v100 )`, where `v99` is `mesoPrice` and `v100`
is `tokenPrice`. Nine items decoded, nine items dropped, empty dialog. The UI's "Free" label is a
faithful render of the same zeros (`NpcShopCommodityWidget.tsx:94-98` gates the price line on
`tokenPrice > 0 && tokenTemplateId > 0`), not an independent display bug.

**Defect B — code.** `services/atlas-npc-shops/atlas.com/npc/shops/processor.go:484` is a landed
stub — `// TODO: implement TokenItem purchasing.` returning `ErrorGenericErrorWithReason` with the
reason `"not implemented"`. `Buy()` implements the rechargeable and meso paths only; any commodity
with `MesoPrice() == 0` falls through to that stub. Correcting the seed data alone would make the
items visible but unbuyable. This is already tracked at `docs/TODO.md:15` and `docs/TODO.md:270`
(the latter's `shops/processor.go:430` line reference is stale).

Alongside the fix, the NPC shop commodity dialogs in atlas-ui gain item-search pickers for the two
template-id fields, replacing raw numeric entry with search-by-name-or-id — matching the mechanism
already used by the Character Template equipment/weapon/starting-item sections.

## 2. Goals

Primary goals:

- Inkwell's shop renders all nine commodities in-game on every version where the client supports
  token pricing, in the correct display order.
- A character can buy from Inkwell, paying the configured token price in Perfect Pitch, with correct
  failure handling when they hold too few.
- Token purchasing is implemented generically — driven by the commodity row's `tokenTemplateId`, not
  hardcoded to `4310000` — so any future token vendor works without code changes.
- The Add/Edit Commodity dialogs let an operator find items by name instead of memorising template
  ids, without changing the CRUD payloads.
- `docs/TODO.md:15` and `docs/TODO.md:270` are closed.

Non-goals:

- **Per-commodity bundle quantity.** atlas-channel's `commodities.Model.Quantity()`
  (`npc/shops/commodities/model.go`) returns a hardcoded `0` with a TODO, so every shop on every
  version ships `quantity: 0` on the wire. This was verified harmless: the v83 client reads the field
  at `ITEM+56` and uses it in exactly one place — `sub_4284BE(itemId) || v8[14] > 1` in
  `CShopDlg::SendBuyRequest` — to *suppress* the "how many?" prompt for bundle purchases. Shipping
  `0` lands stackable items on the quantity-prompt branch bounded by `slotMax`, which is the correct
  behaviour for every commodity Atlas can currently express. Implementing it properly is an
  unmodelled feature requiring a new entity column, REST field, UI field and a `Buy()` that grants N
  items per purchase. Out of scope here; the TODO stays.
- Repricing, reordering or otherwise altering any shop other than `9000069`.
- Changes to `libs/atlas-packet` — the v83 wire encoding is already correct and verified.
- Enforcement of `period` / `levelLimit` server-side (not enforced on the meso path today either;
  parity is preserved, not extended).
- Adding NPC `9000069` to, or removing it from, any non-shop seed catalog (npc definitions, spawns,
  conversations).

## 3. User Stories

- As a player on a v83+ tenant, I want Inkwell's shop to list the items he sells so that I can see
  what my Perfect Pitch can buy.
- As a player, I want to spend Perfect Pitch on an Inkwell item and receive it, so the currency has a
  purpose.
- As a player without enough Perfect Pitch, I want a clear "you need N more" refusal rather than a
  generic error or a silent failure.
- As an operator configuring a shop in the Web UI, I want to search for an item by name or id and
  pick it, so I don't have to look up template ids by hand.
- As an operator editing an existing commodity, I want to see which item the row refers to by name,
  and be prevented from accidentally repointing it at a different item.

## 4. Functional Requirements

### FR-1 — Seed data corrections

**FR-1.1 — Remove pre-v83 Inkwell shops.** Inkwell does not exist in these client versions. Delete:

```
deploy/seed/gms/12_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/48_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/61_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/72_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/79_1/npc-shops/shops/shop-9000069.json
```

Independently corroborated: the v79 client's `CShopDlg::SetShopDlg` @ `0x6d3459` (v79 IDB) reads
only `v98 = Decode4` (mesoPrice) — there is no `tokenPrice` field on the wire at all — and guards
insertion with `if ( v98 )` alone. A token-priced shop is structurally unrepresentable pre-v83, so
these files could never have worked. The seed catalog is directory-driven with no manifest or index,
so deleting the file is sufficient — no other registration to update.

**FR-1.2 — Correct the remaining six.** For each of:

```
deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/84_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/87_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/92_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/95_1/npc-shops/shops/shop-9000069.json
deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json
```

set every commodity's `tokenTemplateId` to `4310000` and `tokenPrice` per the table below, leaving
`mesoPrice`, `discountRate`, `period` and `levelLimit` at `0`. JMS receives identical treatment to
GMS v83+: the version gates in `shop_list.go:55-68` are `t.Region() == "GMS"` conditioned, so the
JMS branch writes the full field set including `tokenTemplateId`.

**FR-1.3 — Commodity order.** The commodity array order is the wire order is the client's display
order (`ShopList.Encode` iterates the slice; the client appends to its list in decode order). The
current seed array is the exact reverse of the intended display positions. Rewrite the array in
position order 1→9:

| Position | templateId | tokenPrice |
|---|---|---|
| 1 | 2022503 | 5 |
| 2 | 2000004 | 5 |
| 3 | 2022514 | 10 |
| 4 | 2000005 | 10 |
| 5 | 3010116 | 25 |
| 6 | 1122017 | 30 |
| 7 | 2049000 | 45 |
| 8 | 2049100 | 70 |
| 9 | 1003016 | 100 |

The nine template ids are exactly the nine already seeded — no additions, no removals.

### FR-2 — Token purchasing in atlas-npc-shops

Replace the stub at `services/atlas-npc-shops/atlas.com/npc/shops/processor.go:484`.

**FR-2.1 — Trigger.** The token path runs when `cm.MesoPrice() == 0 && cm.TokenPrice() > 0`. The
existing rechargeable and meso branches are unchanged and continue to take precedence in their
current order. If both `MesoPrice()` and `TokenPrice()` are zero, or `TokenTemplateId()` is zero
while `TokenPrice() > 0`, the commodity is misconfigured: log and return `ErrorGenericError`.

**FR-2.2 — Ignore the client-supplied price.** The v83 client sends the *meso* price in the buy
packet's final field (`CShopDlg::SendBuyRequest` @ `0x7566f4` ends with
`COutPacket::Encode4(&v66, v8[6])`, where `v8[6]` is `ITEM+24` = mesoPrice), which is `0` for a token
item. The `discountPrice` parameter is therefore meaningless on this path and MUST NOT be used to
compute cost. Charge `cm.TokenPrice() * quantity` from the commodity row.

**FR-2.3 — Token item resolution.** The item to consume is `cm.TokenTemplateId()` — never a
hardcoded constant. The v83 client hardcodes `4310000` for its *local* pre-check
(`TSecType<long>::GetData(v70 + 12, 0) == &loc_41C3F0`, `0x41C3F0` = `4310000`), but the server is the
authority and stays version- and vendor-agnostic. The seed sets the value explicitly (FR-1.2).

**FR-2.4 — Balance check and consumption.** Resolve the token item's inventory type via
`inventory.TypeFromItemId(item.Id(cm.TokenTemplateId()))` rather than assuming ETC — Perfect Pitch is
an ETC item and the v83 client scans inventory type 4, but the derivation keeps other token items
correct. Then, over that compartment's assets (`c.Inventory().CompartmentByType(it).Assets()`, with
`asset.TemplateId()`, `asset.Slot()`, `asset.Quantity()`):

1. Sum the quantity held across **all** slots holding the token item. A single `FindFirstByItemId`
   is insufficient — the cost may exceed one stack.
2. If the total is less than the required cost, return `ErrorNeedMoreItems` (`"NEED_MORE_ITEMS"`,
   `kafka/message/shops/kafka.go:54`). This matches the client's own refusal string SP_5433
   ("YOU NEED %d MORE %s") and is preferred over `ErrorNotEnoughMoney`, which is the meso-path error.
3. Verify a free slot exists in the purchased item's compartment before consuming anything, and
   return `ErrorInventoryFull` if not — mirroring the meso path's ordering so tokens are never
   consumed for an item that cannot be received.
4. Consume across slots via `p.compP.RequestDestroyItem(mb)(characterId, it, slot, quantity)`
   (slot-based — see the Sell path at `processor.go:570` for the call shape), issuing one call per
   slot drawn from, in ascending slot order, until the cost is met.
5. Grant the item via `p.compP.RequestCreateItem(mb)(c.Id(), itemTemplateId, quantity)`.

All Kafka emissions go through the supplied `*message.Buffer` so the whole purchase is emitted
atomically by the enclosing `message.Emit`, consistent with the surrounding code.

**FR-2.5 — Quantity.** The v83 client always sends quantity `1` on the token branch (its `a2` stays
at the `*a2 = 1` initialisation; only the meso branch prompts). The server must still honour the
supplied `quantity` by multiplying the cost, so a hand-crafted or future-version multi-buy is charged
correctly rather than under-charged.

**FR-2.6 — TODO.md.** Remove the now-satisfied entries at `docs/TODO.md:15` and `docs/TODO.md:270`.

### FR-3 — Web UI item pickers

Target: `services/atlas-ui/src/components/features/npc/NpcShopCommodityDialog.tsx`, which currently
renders all seven fields from a single `FIELDS` array of numeric `<Input>`s (lines 32-101). The two
template-id fields must break out of that loop; the remaining five stay as numeric inputs.

**FR-3.1 — New picker component.** `ItemSearchCombobox.tsx`
(`components/features/characters/templates/`) is the mechanism to model, but it is an *action*
control: an "Add" button in a popover that fires `onAdd(id)` and holds no value. A value-bearing
sibling is required — a field that displays the currently-selected item (icon, name, id), opens the
same search popover, and calls `onChange(id)`. Prefer extracting the shared search/popover internals
over duplicating them; the existing `ItemSearchCombobox` behaviour and its tests must not regress.

Reuse as-is:
- `POOL_SEARCH_CONFIGS.items` (`templates/poolSearchConfig.ts`) — the unfiltered, all-compartment
  search, which is what a shop commodity needs (shops sell equips, use, setup and etc items).
- `itemsService.getItemName(id)` (`services/api/items.service.ts`) — resolves a single id to a name
  for display.
- The debounce, "Load more" pagination, manual "Use id N" escape hatch, and icon rendering via
  `getAssetIconUrl`.

**FR-3.2 — Add dialog.** `Template ID` and `Token Template ID` both become pickers. Both must retain
a way to commit a raw numeric id (the existing "Use id N" row satisfies this) so an operator is never
blocked by a missing or unnamed item.

**FR-3.3 — Edit dialog.** `Template ID` renders as the item's **name** (resolved from the existing
value) and is not editable — preserving today's `disabled` behaviour (line 79) while making it
legible. `Token Template ID` is a fully editable picker, same as Add.

**FR-3.4 — Payload invariance.** Form state stays `CommodityAttributes` with numeric ids. The
create/update requests must be byte-for-byte what they are today — the picker changes how a value is
chosen, never what is submitted. A `tokenTemplateId` of `0` (no token) must remain expressible.

**FR-3.5 — Loading and failure states.** Name resolution is async: the dialog must render a sensible
placeholder while loading and fall back to displaying the raw id if lookup fails, never blocking
submission or showing a perpetual spinner.

## 5. API Surface

No new or modified endpoints. The existing shop-commodity routes are unchanged in shape and
semantics:

- `POST /npcs/{npcId}/shop/relationships/commodities` — add commodity
- `PUT  /npcs/{npcId}/shop/relationships/commodities/{commodityId}` — update commodity
- `DELETE /npcs/{npcId}/shop/relationships/commodities/{commodityId}` — remove commodity
- `GET  /npcs/{npcId}/shop?include=commodities` — read (used by atlas-channel)

The `commodities` JSON:API resource keeps all ten attributes (`templateId`, `mesoPrice`,
`discountRate`, `tokenTemplateId`, `tokenPrice`, `period`, `levelLimit`, `unitPrice`, `slotMax`).
FR-3 is a presentation change only.

Behavioural change on the buy command path: a token-priced commodity now emits a success outcome or
`NEED_MORE_ITEMS` / `INVENTORY_FULL` / `GENERIC_ERROR`, where it previously always emitted
`GENERIC_ERROR_WITH_REASON("not implemented")`.

## 6. Data Model

No schema change. `commodities.Entity` (`commodities/entity.go:9-21`) already carries
`TokenTemplateId` and `TokenPrice` as `uint32 NOT NULL DEFAULT 0`, tenant-scoped by `TenantId`. No
migration is required.

Semantics clarified by this task (worth capturing in the design doc): `tokenTemplateId` is the
server's authoritative statement of *which item* is the currency for that commodity. It is only
transmitted to clients on GMS ≥ 95 (`shop_list.go:58`), but it is load-bearing on **all** versions
because the server reads it to decide what to consume. It is also what the Web UI gates its price
line on. A row with `tokenPrice > 0` and `tokenTemplateId == 0` is therefore invalid data, not a
version-appropriate omission.

**Live data.** Seed files are the catalog, not the live rows — correcting the files does not repair
an already-seeded tenant. Per the operating decision for this task, the live tenant will be corrected
by **re-seeding after the PR merges to main**, not by a migration or a targeted PATCH. Note for the
operator: `libs/atlas-seeder/seed.go:88-134` (`runSubdomain`) performs `DeleteAllForTenant` followed
by `BulkCreate` — re-seeding the npc-shops group is a **destructive full replace** of that tenant's
shops and commodities, so any hand-edits made through the Web UI will be lost. This is accepted.

## 7. Service Impact

| Service / area | Change |
|---|---|
| `deploy/seed/gms/{12,48,61,72,79}_1/npc-shops/` | Delete `shop-9000069.json` (FR-1.1) |
| `deploy/seed/gms/{83,84,87,92,95}_1/npc-shops/`, `deploy/seed/jms/185_1/npc-shops/` | Token prices, `tokenTemplateId`, reorder (FR-1.2, FR-1.3) |
| `services/atlas-npc-shops` | Implement the token branch in `shops/processor.go` `Buy()` (FR-2) |
| `services/atlas-ui` | New value-bearing item picker + `NpcShopCommodityDialog` rework (FR-3) |
| `docs/TODO.md` | Remove lines 15 and 270 (FR-2.6) |
| `libs/atlas-packet` | **No change** — v83 wire encoding already correct |
| `services/atlas-channel` | **No change** — `Quantity()` TODO is an explicit non-goal |

## 8. Non-Functional Requirements

- **Multi-tenancy.** All commodity reads and inventory mutations remain tenant-scoped through the
  existing `tenant.MustFromContext(ctx)` / GORM tenant callbacks. No new cross-tenant surface.
- **Atomicity.** Token consumption and item grant must be emitted through the same
  `*message.Buffer`, so a partial purchase (tokens taken, item not granted) cannot be published.
- **No unpriced purchases.** The implementation must never fall through to a path that grants an item
  without charging. Every branch either charges, refuses with a typed error, or returns an error.
- **Version safety.** No behavioural change for GMS < 83, which no longer has an Inkwell shop to
  serve. No wire-format change on any version.
- **Observability.** Follow the existing logging shape in `Buy()` — an `Errorf` per refusal with
  characterId/itemTemplateId/slot, and a `Debugf` on success.
- **Frontend.** Search requests stay debounced and tenant-scoped; no additional request per keystroke,
  and no request at all until the popover is open and a term is entered (the existing `enabled`
  gating).
- **Grounding.** Every client-behaviour claim in this PRD is cited to a decompiled address in a
  specific IDB. Any new claim added during design or implementation must be similarly cited or
  marked unverified.

## 9. Open Questions

None blocking. Two items to resolve during design:

1. **Picker extraction shape.** Whether the value-bearing field and the existing "Add" combobox
   share an extracted internal (preferred) or the new component is written standalone. Decide against
   the actual `ItemSearchCombobox` internals and its existing test suite.
2. **Multi-slot consumption granularity.** Whether `RequestDestroyItem` is called once per drawn slot
   (simplest, matches the slot-based API) or whether a batching affordance exists that would be
   preferable. Confirm against the compartment consumer's handling of multiple destroy commands in
   one buffer.

## 10. Acceptance Criteria

Data:

- [ ] `shop-9000069.json` is deleted from the five pre-v83 GMS seed directories.
- [ ] The remaining six `shop-9000069.json` files each list nine commodities in position order 1→9,
      every one with `tokenTemplateId: 4310000` and the `tokenPrice` from the FR-1.3 table.
- [ ] No other shop's seed file is modified.

Backend:

- [ ] The `// TODO: implement TokenItem purchasing.` stub is gone; no `TODO` or `"not implemented"`
      remains in `Buy()`.
- [ ] Buying a token commodity with sufficient tokens consumes `tokenPrice × quantity` of
      `tokenTemplateId` (across multiple slots when one stack is insufficient) and grants the item.
- [ ] Buying with insufficient tokens emits `NEED_MORE_ITEMS` and consumes nothing.
- [ ] Buying with a full destination compartment emits `INVENTORY_FULL` and consumes nothing.
- [ ] The client-supplied `discountPrice` provably does not influence the amount charged.
- [ ] The meso and rechargeable paths are behaviourally unchanged (existing tests still pass).
- [ ] Unit tests cover: sufficient/insufficient tokens, multi-slot consumption, inventory-full,
      misconfigured row (`tokenPrice > 0`, `tokenTemplateId == 0`), and `quantity > 1` cost.

Frontend:

- [ ] Add Commodity: both `Template ID` and `Token Template ID` are search pickers accepting name or
      id, each with a raw-id escape hatch.
- [ ] Edit Commodity: `Template ID` displays the item name and cannot be changed; `Token Template ID`
      is an editable picker.
- [ ] Create and update request payloads are unchanged from current behaviour, verified by test.
- [ ] `tokenTemplateId: 0` remains expressible.
- [ ] Existing `ItemSearchCombobox` tests still pass.

Verification gates (per CLAUDE.md, in the worktree):

- [ ] `go test -race ./...` clean in every changed Go module.
- [ ] `go vet ./...` clean in every changed Go module.
- [ ] `go build ./...` clean in `services/atlas-npc-shops`.
- [ ] `tools/lint.sh --check` clean from the repo root.
- [ ] `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` clean.
- [ ] atlas-ui `npm run build` (type-checks tests) and `npm run test` clean.
- [ ] `docker buildx bake atlas-npc-shops` only if that service's `go.mod` was touched.
- [ ] `docs/TODO.md` lines 15 and 270 removed.

Manual (post-merge, on the live tenant):

- [ ] Re-seed the npc-shops group, then confirm Inkwell's shop lists nine items in the order above
      with Perfect Pitch prices, and that a purchase debits the correct number of Perfect Pitch.
