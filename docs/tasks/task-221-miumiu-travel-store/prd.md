# MiuMiu Travel Store (cash item category 545) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13

---

## 1. Overview

Cash item category 545 (`item.ClassificationRemoteMerchant`) is the "remote
merchant" family: consumable cash items that give the player access to a
town-bound service from wherever they are standing. WZ `Item.wz/Cash/0545.img.xml`
holds exactly two items on the GMS ≤ v95 data set:

| Item id | String.wz name | `info/npc` | `info/cash` |
|---|---|---|---|
| 5450000 | Miu Miu the Traveling Merchant | 9090000 | 1 |
| 5451000 | Remote Gachapon Ticket | 0 | 1 |

Neither is implemented. `GetCashSlotItemType`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:997-1010`)
already classifies both — 5450000 → cash-slot type 37 (GMS < 95) / 38 (GMS ≥ 95),
5451000 → type 59 (GMS < 95) / 60 (GMS ≥ 95) — but no arm in
`CharacterCashItemUseHandleFunc` matches either, so both fall through to the
terminal `l.Warnf("Character [%d] attempting to use cash item …")` at line 641.
The player's client stays locked on `m_bExclRequestSent` and the item is never
consumed.

Implementing the family is more than one handler arm, because the services the
items are supposed to reach are themselves incomplete:

- **No shop exists for NPC 9090000.** `deploy/seed/gms/*/npc-shops/shops/` seeds
  99 shops per version and 9090000 is one of three documented missing merchants
  (`docs/research/missing-features/npc-content.md` §B). Cosmic seeds it as shop
  id 1338 with 26 commodities.
- **The NPC-shop serverbound opcode is unregistered on three live templates.**
  `NPCShopHandle` appears in `template_gms_{48,61,72,79,83,84}_1.json` and is
  **absent** from `gms_87`, `gms_92`, `gms_95`. `gms_92` is additionally missing
  both the `NPCShop` and `NPCShopOperation` clientbound writers. On those
  tenants a remote store would open (or not even render) and then be
  non-transactable — the item would be destroyed for nothing.
- **`atlas-saga-orchestrator` has no npc-shop step.** The closest precedent,
  `ShowStorage`, is self-completing (`event_acceptance.go:169`), which cannot
  express "consume the item only once the shop actually opened."

This task implements the family end-to-end: the handler arms, the missing shop
data, the missing template registrations, and a saga step that gates item
consumption on the shop-entered acknowledgement.

## 2. Goals

Primary goals:

- Using 5450000 opens NPC 9090000's shop from any map and consumes one copy of
  the item, on every tenant template where the NPC-shop protocol is complete
  after this task.
- Using 5451000 opens the gachapon flow without travelling to a gachapon NPC,
  and consumes one copy of the item.
- The cash item is consumed **only after** the target service acknowledges it
  opened. A missing shop, a missing gachapon pool, or a service error leaves the
  item in the player's cash inventory and unlocks the client.
- NPC 9090000 has a seeded shop, with rechargeable star/bullet listings, in
  every GMS seed version directory.
- `NPCShopHandle` / `NPCShop` / `NPCShopOperation` are registered in every
  template where the coverage matrix has a known opcode, closing the v87/v92/v95
  hole for **all** NPC shops, not just the remote one.

Non-goals:

- Any other unimplemented cash-slot type (transformation coupons, meso sacks,
  pet name tag, Duey coupon, Maple Life, karma scissors, store permit, cosmetic
  coupons, expiration extenders). Each is its own row in
  `docs/research/missing-features/items-and-consumables.md` §7.
- The cash-shop gachapon (`CASH_ITEM_GACHAPON`, `cash_item_gachapon.go`) — a
  different opcode and a different feature.
- Map-based restrictions on remote-store use. The String.wz description claims
  "Some maps restrict the use of this item"; no restriction data exists in
  `Cash/0545.img.xml` and the reference implementation enforces none. Out of
  scope unless the design-phase IDA pass finds a client-side or WZ-side list.
- `gms_12`: its template registers no `CharacterCashItemUseHandle` at all, so
  the op cannot arrive. Out of scope by construction.

## 3. User Stories

- As a player far from town with a Miu Miu the Traveling Merchant in my cash
  inventory, I want to use it and get a general store window, so that I can
  restock potions and recharge throwing stars without walking back.
- As a player, if the remote store fails to open (bad data, service down), I
  want to keep my cash item and regain control of my character, so that a server
  fault does not cost me a paid item.
- As a player with a Remote Gachapon Ticket, I want to run a gachapon without
  travelling to a gachapon NPC, so that the ticket is worth its cash price.
- As a v87/v92/v95 tenant operator, I want NPC shops to be buyable/sellable at
  all, so that seeded shop content is reachable on my version.

## 4. Functional Requirements

### FR-1 — Handler dispatch (atlas-channel)

- **FR-1.1** Add an arm to `CharacterCashItemUseHandleFunc` for the remote
  merchant family, dispatched **classification-first** on
  `item.GetClassification(itemId) == item.ClassificationRemoteMerchant` (545),
  not on the cash-slot type. The type values collide: 37 is also the wedding
  ticket bucket (`GetCashSlotItemType`'s `ClassificationWeddingTicket` branch),
  and 59/60 collide with the triple-megaphone buckets. This follows the existing
  precedent documented at `character_cash_item_use.go:503-507`.
- **FR-1.2** Within the arm, dispatch by item id: `itemId/1000 == 5451` → remote
  gachapon; otherwise → remote store. Do not use the cash-slot type byte to make
  this distinction, only to validate it.
- **FR-1.3** Decode whatever sub-body the client sends for cash-slot types
  37/38 and 59/60 before any other read. The shape must be derived from the
  GMS v83 and v95 IDBs (`CWvsContext::SendConsumeCashItemUseRequest`'s dispatch
  table) during the design phase — see OQ-1. If a version's dispatch table has
  no arm for the type, that version emits nothing for the item and the arm must
  be gated off for it rather than decoding speculatively.
- **FR-1.4** Server-side re-validation: reject (log + `EnableActions`) if the
  item in `source` slot of the cash compartment is not the item id the request
  claims, mirroring the `cashItemInSlotFunc` seam already used by the incubator
  and vicious-hammer arms.
- **FR-1.5** On any rejection path, send `EnableActions` (`StatChanged` with
  `exclRequestSent = true`) so the client unlocks. On the success path, do
  **not** send `EnableActions` unless the design-phase IDA check confirms
  `CShopDlg::SetShopDlg` does not itself clear `m_bExclRequestSent` — see
  `[[reference_exclrequest_unlock_contract]]` and OQ-2.

### FR-2 — Remote store flow (5450000)

- **FR-2.1** The target NPC template id is read from the item's WZ
  `info/npc` value via `atlas-data`, not hard-coded. `atlas-data`'s cash-item
  reader must expose it; if it does not today, add the parse. (Cosmic
  hard-codes shop 1338; Atlas must be data-driven per DOM-25 /
  `[[feedback_client_wire_values_config_resolved]]`.)
- **FR-2.2** Opening the shop and consuming the item is one saga (see FR-4),
  ordered: open shop → destroy asset. The item is consumed only after the
  `ENTERED` acknowledgement.
- **FR-2.3** If the character already has a shop, storage, trade, or other
  exclusive dialog open, the request is rejected with `EnableActions` and no
  consumption. (Cosmic's `player.getShop() != null` guard.) The authoritative
  "already in a shop" state lives in `atlas-npc-shops`; the design phase decides
  whether the guard is the service's `ERROR` reply or a channel-side check.

### FR-3 — Remote gachapon flow (5451000)

- **FR-3.1** The item's WZ `info/npc` is `0`, so the target gachapon is **not**
  data-derivable from the item. The design phase must determine from the IDB
  whether the client's type-59/60 sub-body carries a town/gachapon selection
  (OQ-3). Two acceptable outcomes:
  - The sub-body carries a selection → resolve it to one of the eight seeded
    gachapon NPCs (9100100-9100106, 9100117) and run that pool.
  - The sub-body carries no selection → the server opens a selection dialog
    (an `atlas-npc-conversations` conversation) whose terminal states are
    `gachaponAction` nodes, one per town.
- **FR-3.2** Whichever path, the gachapon roll reuses the existing
  `gachaponAction` machinery (`deploy/seed/gms/*/npc-conversations/npc/npc-91001*.json`,
  `atlas-reward-pools`) — no second gachapon implementation.
- **FR-3.3** The ticket item consumed is 5451000 itself, not the ordinary
  Gachapon Ticket 5220000. The existing `gachaponAction` node carries
  `ticketItemId`; the remote flow must pass 5451000 so the player is not charged
  twice, and must not require 5220000 in inventory.
- **FR-3.4** Consumption is gated on a successful roll, consistent with FR-2.2
  and with the existing `failureState` behaviour (inventory-full → no
  consumption).

### FR-4 — Saga step (atlas-saga-orchestrator + libs/atlas-saga)

- **FR-4.1** Add a saga `Action` for opening an NPC shop (working name
  `open_npc_shop`) with a payload carrying `characterId`, `worldId`,
  `channelId`, `npcTemplateId`.
- **FR-4.2** The handler dispatches `COMMAND_TOPIC_NPC_SHOP` / `ENTER`
  (`shops.CommandShopEnterBody{NpcTemplateId}`), the same command
  `atlas-channel`'s `npc/shops` processor already produces
  (`services/atlas-channel/atlas.com/channel/npc/shops/processor.go:50`).
- **FR-4.3** The step is **not** self-completing. Add `EventKind`s for the
  npc-shop status topic (`npcshop.entered`, `npcshop.error`) and an
  `acceptanceTable` entry mapping the new action to them, plus a consumer for
  `EVENT_TOPIC_NPC_SHOP_STATUS` in `atlas-saga-orchestrator`. The
  `event_acceptance_test.go` coverage test must stay green.
- **FR-4.4** `StatusEventTypeError` fails the step, which fails the saga, which
  means the following `destroy_asset_from_slot` step never runs — the item
  survives. The failure must surface to the player as `EnableActions` (and, if
  the error carries a reason, the existing shop error writer path).
- **FR-4.5** If the destroy step fails after the shop opened, compensate by
  emitting `COMMAND_TOPIC_NPC_SHOP` / `EXIT` so the player is not left in a shop
  they did not pay for.

### FR-5 — Shop data (deploy/seed)

- **FR-5.1** Add `shop-9090000.json` to **all ten** version directories under
  `deploy/seed/gms/*/npc-shops/shops/` (12_1 … 95_1), following the existing
  envelope: `{"data":{"type":"npc-shop","id":"9090000","attributes":{"npcId":9090000,"recharger":true,"commodities":[…]}}}`.
- **FR-5.2** Commodities are the 26 rows Cosmic seeds for shop 1338
  (`Cosmic/src/main/resources/db/data/102-shopitems-data.sql:3340-3365`), mapped
  to Atlas's commodity shape with `discountRate: 0`, `levelLimit: 0`,
  `period: 0`, `tokenPrice: 0`, `tokenTemplateId: 0`:

  | templateId | mesoPrice | | templateId | mesoPrice |
  |---|---|---|---|---|
  | 2010003 | 100 | | 2022191 | 1000 |
  | 2061000 | 1 | | 2022189 | 1000 |
  | 2060000 | 1 | | 2010004 | 310 |
  | 2030000 | 400 | | 2010001 | 106 |
  | 2022195 | 15000 | | 2010002 | 50 |
  | 2020015 | 10200 | | 2010000 | 30 |
  | 2020014 | 8100 | | 2002025 | 1500 |
  | 2020013 | 5600 | | 2002024 | 1500 |
  | 2020012 | 4500 | | 2002023 | 3800 |
  | 2022190 | 3000 | | 5041000 | 1500000 |
  | 2001002 | 4000 | | 2022000 | 1650 |
  | 2001001 | 2300 | | 2022003 | 1100 |
  | 2001000 | 3200 | | 2022192 | 600 |

  Any commodity whose `templateId` does not exist in a given version's WZ data
  must be dropped from that version's file rather than seeded blind; the design
  phase verifies existence against `atlas-data` per version.
- **FR-5.3** `"recharger": true`, so `atlas-npc-shops` appends the Redis-backed
  rechargeable star/bullet listings (`shops/cache.go`,
  `data/consumable/processor.go:GetRechargeable`). Cosmic's 1338 rows already
  include throwing stars (2070xxx are appended, not seeded) — the item
  description explicitly promises star and bullet recharge.

### FR-6 — Template registration (atlas-configurations)

Current state, counted over
`services/atlas-configurations/seed-data/templates/`:

| Template | `CharacterCashItemUseHandle` | `NPCShopHandle` | `NPCShop` writer | `NPCShopOperation` writer |
|---|---|---|---|---|
| gms_12 | ✗ | ✗ | ✗ | ✗ |
| gms_48 | ✓ | ✓ | ✗ | ✗ |
| gms_61/72/79/83/84 | ✓ | ✓ | ✓ | ✓ |
| gms_87 | ✓ | **✗** | ✓ | ✓ |
| gms_92 | ✓ | **✗** | **✗** | **✗** |
| gms_95 | ✓ | **✗** | ✓ | ✓ |

- **FR-6.1** Register `NPCShopHandle` in `gms_87` (opcode `0x040`), `gms_92`
  (`0x043`), `gms_95` (`0x042`), taken from the `NPC_SHOP` serverbound row of
  `docs/packets/audits/STATUS.md:572`.
- **FR-6.2** Register the `NPCShop` writer (`OPEN_NPC_SHOP`, STATUS.md:381) and
  `NPCShopOperation` writer (`CONFIRM_SHOP_TRANSACTION`, STATUS.md:383) in
  `gms_92` — `0x164` and `0x165` respectively.
- **FR-6.3** `gms_48` has the handler but neither writer, and the matrix has
  **no** opcode (⬜) for `OPEN_NPC_SHOP` / `CONFIRM_SHOP_TRANSACTION` on the v48
  column. Derive them from the v48 IDB during the design phase. If the v48
  client genuinely lacks `CShopDlg::SetShopDlg`, record the cells as `n-a` in the
  matrix with evidence and exclude gms_48 from FR-2/FR-3 — do not leave the
  question open.
- **FR-6.4** Every template edit must keep both `handlers` and `writers` arrays
  in strictly ascending `opCode` order (`tools/template-opcode-order-guard.sh`)
  and must not introduce a duplicate `(name, opCode)` pair
  (`tools/template-duplicate-binding-guard.sh`).
- **FR-6.5** Every handler entry needs a non-empty validator; a handler with a
  missing validator is silently dropped at load
  (`[[bug_socket_handler_missing_validator_silently_dropped]]`). Mirror the
  validator used by `NPCShopHandle` in gms_83.
- **FR-6.6** Newly registered opcodes must also be applied to the **live tenant
  socket configurations**, not just the seed templates — a template-only change
  does nothing for an already-provisioned tenant
  (`[[bug_new_opcodes_not_in_live_tenant_config]]`,
  `[[reference_reconcile_live_tenant_socket_to_template]]`).

### FR-7 — Packet coverage

- **FR-7.1** The serverbound `NPC_SHOP` cells for v87/v92/v95 and the
  clientbound `OPEN_NPC_SHOP` / `CONFIRM_SHOP_TRANSACTION` cells for v92 are
  currently ❌. Any cell this task makes reachable must be promoted through the
  single-cell verify procedure (`/verify-packet`,
  `docs/packets/audits/VERIFYING_A_PACKET.md`) — a registration change with no
  byte fixture is not a verified cell.
- **FR-7.2** If FR-1.3 requires a new serverbound sub-body codec for cash-slot
  types 37/38 or 59/60, it is written per
  `docs/packets/IMPLEMENTING_A_PACKET.md` with both `Encode` and `Decode` and a
  byte fixture per applicable version.

## 5. API Surface

No new REST endpoints. Existing surfaces used:

- `GET /npc-shops/{npcId}` (atlas-npc-shops) — resolve NPC 9090000's shop.
  Already consumed by `atlas-channel`'s `npc/shops` processor.
- `atlas-data` cash-item resource — must expose `info/npc` for category-545
  items (FR-2.1). If a new field is added to the consumable/cash rest model, it
  is additive and JSON:API-shaped.

New Kafka contracts:

- **Saga step payload** (`libs/atlas-saga`): `OpenNpcShopPayload{CharacterId,
  WorldId, ChannelId, NpcTemplateId}` for action `open_npc_shop`.
- **Consumed**: `EVENT_TOPIC_NPC_SHOP_STATUS` in `atlas-saga-orchestrator`
  (`ENTERED` / `ERROR`), an existing topic gaining a second consumer group.
- **Produced**: `COMMAND_TOPIC_NPC_SHOP` `ENTER` / `EXIT` from
  `atlas-saga-orchestrator` (existing contract, new producer).

## 6. Data Model

No database migrations.

New seed data:

- `deploy/seed/gms/{12,48,61,72,79,83,84,87,92,95}_1/npc-shops/shops/shop-9090000.json`
  — 10 files, `type: "npc-shop"`, `id: "9090000"`, `recharger: true`, ≤26
  commodities (per-version filtered per FR-5.2).

Modified config data:

- `services/atlas-configurations/seed-data/templates/template_gms_{87,92,95}_1.json`
  (and `gms_48` pending FR-6.3): new `handlers` / `writers` entries.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-channel` | New classification-545 arm in `character_cash_item_use.go` + a new `character_cash_item_use_remote_merchant.go` (following the megaphone / point-reset / vicious-hammer file-per-arm precedent); saga emission; ownership re-validation. |
| `libs/atlas-packet` | Serverbound sub-body codec(s) for cash-slot types 37/38 and 59/60, if FR-1.3's IDA pass finds a non-empty body. |
| `libs/atlas-saga` | New `Action` constant + payload struct. |
| `atlas-saga-orchestrator` | New action handler, new `EventKind`s, `acceptanceTable` entry, `EVENT_TOPIC_NPC_SHOP_STATUS` consumer, `COMMAND_TOPIC_NPC_SHOP` producer, compensation step. |
| `atlas-npc-shops` | No code change expected; new seed file consumed. Verify the "already in a shop" path returns `ERROR` rather than silently re-entering. |
| `atlas-data` | Expose cash-item `info/npc` if not already parsed (FR-2.1). |
| `atlas-npc-conversations` / `atlas-reward-pools` | Only if FR-3.1 lands on the selection-dialog branch: a new remote-gachapon conversation seed reusing `gachaponAction`. |
| `atlas-configurations` | Template registrations (FR-6). |
| `deploy/seed` | 10 new shop files. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Everything resolves through `tenant.MustFromContext(ctx)`.
  Shop seeds are per-version directories; template edits are per-template. No
  cross-tenant shop state.
- **Version gating.** All version branches use the
  `constants.For(region, major, minor)` / `MajorAtLeast` idiom, never a raw
  `> N` comparison (`[[bug_majorversion_gt83_is_off_by_one_v87]]`).
- **No wire change to an already-verified version.** v61/72/79/83/84 NPC-shop
  behaviour must be byte-identical after this task; their matrix cells stay ✅.
- **Idempotency.** The saga must not consume the cash item twice under Kafka
  redelivery (`[[bug_kafka_redelivery_dupes_nonidempotent_handlers]]`). The
  destroy step is slot+item-id scoped, and the shop-enter step must tolerate a
  duplicate `ENTERED`.
- **Observability.** Log the item id, cash-slot type, resolved NPC template id,
  and saga id on entry; log the rejection reason on every `EnableActions` path.
- **No silent drops.** A category-545 item on a version where the flow is
  disabled must log a distinct, greppable warn — not fall into the generic
  catch-all at line 641.

## 9. Open Questions

- **OQ-1** (blocking FR-1.3): what does the v83/v95 client send in the
  `SendConsumeCashItemUseRequest` arms for cash-slot types 37/38 and 59/60? The
  reference implementation reads no extra bytes for type 545, suggesting an
  empty sub-body, but this is **unverified** against the IDB. Design phase must
  decompile and record addresses.
- **OQ-2** (blocking FR-1.5): does `CShopDlg::SetShopDlg` clear
  `m_bExclRequestSent`, or must the server send `EnableActions` alongside the
  shop packet? Answer decides whether the success path unlocks explicitly.
- **OQ-3** (blocking FR-3.1): does the type-59/60 sub-body carry a
  town/gachapon selection? If not, which of the eight seeded gachapon NPCs (or
  what selection UI) does the remote ticket target? No reference implementation
  exists — Cosmic routes item type 545 unconditionally to shop 1338, which is
  wrong for 5451000.
- **OQ-4** (blocking FR-6.3): does the v48 client implement `CShopDlg` at all?
  The coverage matrix has ⬜ (unknown, not ❌) for both v48 clientbound shop ops.
- **OQ-5**: does any commodity in FR-5.2's list not exist in the pre-v83 WZ
  sets (12/48/61/72/79)? 5041000 (Teleport Rock) in particular.
- **OQ-6**: should the remote store be usable inside instanced/PQ fields, or
  should it be blocked there? No WZ data expresses a restriction; deferred to
  design unless the IDA pass finds a client-side check.

## 10. Acceptance Criteria

- [ ] Using 5450000 on a gms_83 tenant opens NPC 9090000's shop window with the
      seeded commodities plus rechargeable listings, and removes exactly one
      copy from the cash inventory.
- [ ] Buying, selling, and recharging inside that window work on gms_83 and on
      gms_87, gms_92, gms_95 (previously impossible — `NPCShopHandle` was
      unregistered).
- [ ] Using 5450000 when `atlas-npc-shops` returns `ERROR` leaves the item in
      inventory and the client unlocked; the failure is logged with a reason.
- [ ] Using 5451000 runs a gachapon roll (per FR-3.1's resolved branch) and
      consumes exactly one 5451000 — never a 5220000 — and consumes nothing when
      the roll fails for inventory-full.
- [ ] `shop-9090000.json` exists in all ten `deploy/seed/gms/*/npc-shops/shops/`
      directories with `recharger: true`, and each version's commodity list
      contains only item ids that exist in that version's data.
- [ ] `NPCShopHandle` is present in gms_87/92/95 templates; `NPCShop` and
      `NPCShopOperation` writers are present in gms_92; every entry sits at its
      sorted `opCode` position with a non-empty validator.
- [ ] OQ-4 is resolved for gms_48 — either the writers are registered with
      IDB-derived opcodes, or the matrix cells are recorded `n-a` with evidence.
- [ ] Live tenant socket configurations are reconciled to the updated templates
      (FR-6.6), verified by reading back the live config, not by asserting the
      template changed.
- [ ] Every packet cell this task touches is promoted via `/verify-packet` with
      a committed byte fixture and evidence record; `packet-audit matrix --check`
      and `fname-doc --check` exit 0.
- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` clean in every
      changed module.
- [ ] `docker buildx bake atlas-channel atlas-saga-orchestrator atlas-npc-shops
      atlas-data` succeeds from the worktree root.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
      `tools/goroutine-guard.sh`, `tools/template-opcode-order-guard.sh`,
      `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh`, and
      `tools/skill-job-id-guard.sh` all clean from the repo root.
- [ ] Code review (`superpowers:requesting-code-review`) run and findings
      resolved before the PR is opened.
