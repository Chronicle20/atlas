# MiuMiu Travel Store (cash item category 545) — Design

Task: task-221-miumiu-travel-store
PRD: [`prd.md`](prd.md)
Status: Draft for review
Created: 2026-08-13

---

## 0. Executive summary

The design phase decompiled all nine GMS client builds plus JMS185 to close the
PRD's six blocking open questions. Three of them change the shape of the task:

1. **The Remote Gachapon Ticket (5451000) is unreachable on every audited
   client.** No build emits `CASH_ITEM_USE` for its cash-slot type. FR-3 and its
   dependent work (`atlas-npc-conversations` seed, `atlas-reward-pools` reuse,
   the type-59/60 codec) collapse into a single greppable warn + unlock. This is
   a scope *reduction*, not a deferral — there is nothing on the wire to
   implement.
2. **The remote-store sub-body is empty on every version that has an arm**, so
   FR-7.2's "new serverbound codec" also dissolves. No `libs/atlas-packet`
   change is required for the serverbound side.
3. **Three GMS builds cannot send the op at all** (gms_12, gms_48, gms_61), so
   the flow is version-gated to gms_{72,79,83,84,87,92,95}. The NPC-shop
   *protocol* fixes (FR-6) still apply to gms_{87,92,95} — those benefit every
   NPC shop, not just this one — and gms_48's missing writers now have
   IDB-derived opcodes, so FR-6.3 resolves as "register", not "n-a".

What remains is: one classification-545 handler arm, a two-step saga, an
unlock registry, an `atlas-data` field, ten shop seed files, and the
gms_{48,87,92,95} template registrations.

---

## 1. Evidence: what the client actually does

All addresses below are from the IDBs enumerated by `idb_list`; each was read in
this session. Method: locate `CWvsContext::SendConsumeCashItemUseRequest`, then
enumerate its jump-table `case`/`default case` annotations over the function's
address range. A type that appears in the `default case` list has **no arm** —
the client computes the type, finds no dispatch entry, and sends nothing.

### 1.1 Cash-slot type for classification 545

`get_cashslot_item_type` (the function Atlas's `GetCashSlotItemType` ports):

| Build | Function | Category-545 result |
|---|---|---|
| gms_v83 | `?get_cashslot_item_type@@YAJJ@Z` @ `0x48645b` | `case 545: return a1/1000 != 5451 ? 37 : 59;` |
| gms_v79 | `get_cashslot_item_type` @ `0x47ec3e` | identical (`37` / `59`) |
| gms_v72 | `get_cashslot_item_type` @ `0x49fb33` | present (not re-read field-by-field; arm evidence below is decisive) |
| gms_v48 | — | **no such function**; v48 uses an earlier, smaller type enum |
| gms_v61 | — | **no such symbol** in the IDB |
| jms185 | `?get_cashslot_item_type@@YAJJ@Z` @ `0x49a1ee` | `case 545: return 36;` — **no 5451 branch at all** |

Atlas's port (`character_cash_item_use.go:997-1010`) matches GMS exactly:
`37` (GMS < 95) / `38` (GMS ≥ 95) for the store, `59` / `60` for the ticket.
It is **wrong for JMS**, which uses `36` — see §7.3.

### 1.2 Does an arm exist? (`SendConsumeCashItemUseRequest` jump table)

| Build | Function addr | Store type | Store arm | Ticket type | Ticket arm |
|---|---|---|---|---|---|
| gms_v48 | `0x70e495` | n/a (different enum; max case 46) | — | n/a | — |
| gms_v61 | `0x832a5d` | 37 | **absent** — `37` is in `default case, cases 27,34,35,37,38,40,42-44` @ `0x832af3` | 59 | absent (out of table range) |
| gms_v72 | `0x904fe2` | 37 | **present** @ `0x907472` (`call get_field`) | 59 | absent — `default case, cases 28,35,36,38,39,41,43-45,54-59,62` @ `0x905083` |
| gms_v79 | `0x95634a` | 37 | **present** @ `0x95889a` (`call get_field`) | 59 | absent — `default …,54-59,62,67` @ `0x9563eb` |
| gms_v83 | `0xa0a63f` | 37 | **present** @ `0xa0cda7` (`call get_field`) | 59 | absent — full 49-entry case enumeration over `jumptable 00A0A6E6` contains 12-27,29-34,37,40,42,46-53,60,61,63-66,68,69; **no 59** |
| gms_v84 | `0xa54a2f` | 37 | **present** (37 not in default list) | 59 | absent — `default …,54-59,62,67-69` @ `0xa54ad0` |
| gms_v87 | `0xa9fef9` | 37 | **present** | 59 | absent — `default …,54-59,62,67-69` @ `0xa9ffa8` |
| gms_v92 | `0x9bfe10` | 37 | **present** | 59 | absent — `default …,54-59,62,67-69` @ `0x9bff2c` |
| gms_v95 | `0x9eb3e0` | 38 | **present** @ `0x9ee50a` (`call get_field`) | 60 | absent — `default case, cases 36,37,39,40,42,44,46,55-60,63,68-70,76,77` @ `0x9eb4fd` |
| jms185 | `0xaef2f5` | 36 | **present** (36 not in `default case, cases 34,35,37,40,45,46,53,54,62,65-68` @ `0xaef3a2`) | — | n/a |

**OQ-3 resolved: negative.** The Remote Gachapon Ticket has no send path on any
audited build. The question "does the sub-body carry a town selection?" is moot;
there is no sub-body because there is no packet.

**OQ-1 resolved.** The store arm (v83 `0xa0cda7`, verbatim):

```
loc_A0CDA7:                       ; case 37
    call    get_field
    mov     ecx, eax
    call    sub_A0ECAB            ; (pField[0x144] >> 18) & 1
    test    eax, eax
    jz      short loc_A0CDEC
    ; ---- blocked on this field: StringPool string 0x114 -> CHATLOG_ADD, return
loc_A0CDEC:
    call    sub_A14A1C            ; shared "another exclusive dialog is open?" probe
    test    eax, eax
    jz      loc_A0A8F1            ; -> shared encode+send tail (cases 33,69)
    ; ---- blocked: StringPool string 0x96 -> CHATLOG_ADD, return
```

The arm performs **no `Encode*` of its own** — it falls straight into the tail
shared with cases 33 and 69. v95's `0x9ee50a` has the same shape (`call
get_field`, then `jz $LN232_14`, the cases-33/72/73 tail). **The sub-body is
empty.** The only bytes on the wire are the ones
`CharacterCashItemUseHandleFunc` already reads (update-time, source slot, item
id). FR-7.2 has nothing to codec.

**OQ-6 resolved.** `sub_A0ECAB` @ `0xa0ecab` decompiles to
`return (this[81] >> 18) & 1;` — bit 18 of the `CField` attribute dword at
offset `0x144`. The client blocks the send itself on restricted fields and shows
string 0x114. The exact `FieldLimit` enum name is **not resolved** (no
`GetFieldLimit`/`CField…Limit` symbol in the v95 PDB-backed IDB), and Atlas
models no field-limit bitmask anywhere (`grep -i fieldlimit` over `libs/`,
`atlas-data`, `atlas-maps` is empty). **Design decision: Atlas does not
replicate this check.** The client is the gate; a crafted packet that bypasses
it opens a shop the player could have reached by walking to town. Replicating it
would require ingesting a field-limit bitmask Atlas does not currently parse —
disproportionate to the risk.

**OQ-2 resolved: the server must unlock.** `CShopDlg::SetShopDlg` @ `0x7529ad`
(v83) was decompiled in full: it decodes the shop body, populates the item
arrays, and calls `SetSellItems`/`SetScrollBar`/`SetAvatar`/`SetNPC`/
`InvalidateRect`. It never touches `m_bExclRequestSent`. Its only caller,
`CShopDlg::OnPacket` @ `0x756da7` (`nType == 0x131`), does not either.
Meanwhile the send tail sets the flag (the task-123 megaphone audit recorded the
shared epilogue as `call SetExclRequestSent; Encode4; SendPacket`, and every
arm above funnels into it). So the client stays locked unless the server sends
`EnableActions`. See §4.3 for where.

**OQ-4 resolved: gms_48 has `CShopDlg`.** `?OnPacket@CShopDlg@@SAXJAAVCInPacket@@@Z`
@ `0x5b7a38` branches on `nType == 229` → allocate `CShopDlg` + `SetShopDlg`,
and `nType == 230` → transaction-result switch (the same Decode1 result codes
and StringPool notices as v83's `0x132` arm). v48 also has a *separate*
`CTrunkDlg` family (`SendGetItemRequest` @ `0x5ebf4f` etc.), so this is the shop
dialog, not the storage bank — the decompiler renders the enclosing type name as
`CStoreBankDlg` from a stale local type, which is a naming artifact, not the
function's identity.

> **gms_48 opcodes: `OPEN_NPC_SHOP` = `0xE5` (229), `CONFIRM_SHOP_TRANSACTION` = `0xE6` (230).**

Cross-check that the reading method is sound: the same procedure on v83 yields
`0x131` / `0x132`, which is exactly what `template_gms_83_1.json` already binds
(`"opCode": "0x131"` → `NPCShop`/`CShopDlg::SetShopDlg`; `"opCode": "0x132"` →
`NPCShopOperation`/`CShopDlg::OnPacket`). These two v48 cells still go through
`/verify-packet` with a byte fixture before the matrix moves off ⬜.

### 1.3 Version enablement matrix (the design's central table)

| Template | 545 → type | Client sends? | Remote store enabled | Why |
|---|---|---|---|---|
| gms_12 | — | no | **no** | no `CharacterCashItemUseHandle` in the template at all (PRD non-goal) |
| gms_48 | different enum | no | **no** | pre-`get_cashslot_item_type` generation; no arm evidence |
| gms_61 | 37 | **no** | **no** | case 37 is in the `default case` list @ `0x832af3` |
| gms_72 | 37 | yes | **yes** | arm @ `0x907472` |
| gms_79 | 37 | yes | **yes** | arm @ `0x95889a` |
| gms_83 | 37 | yes | **yes** | arm @ `0xa0cda7` |
| gms_84 | 37 | yes | **yes** | 37 absent from default list @ `0xa54ad0` |
| gms_87 | 37 | yes | **yes** | 37 absent from default list @ `0xa9ffa8` |
| gms_92 | 37 | yes | **yes** | 37 absent from default list @ `0x9bff2c` |
| gms_95 | 38 | yes | **yes** | arm @ `0x9ee50a` |
| jms_185 | **36** | yes | see §7.3 | `get_cashslot_item_type` @ `0x49a1ee` returns 36 |

The Remote Gachapon Ticket column is uniformly **no**.

---

## 2. Architecture

### 2.1 Component map

```
atlas-channel                                   atlas-saga-orchestrator
─────────────                                   ───────────────────────
CharacterCashItemUseHandleFunc
  └─ classification-545 arm  ──creates saga──▶  Saga{ open_npc_shop,
     (character_cash_item_use_                          destroy_asset_from_slot }
      remote_merchant.go)                              │
  └─ remotemerchant.Registry.Put(charId)               │ step 1 dispatch
                                                       ▼
                                        COMMAND_TOPIC_NPC_SHOP / ENTER
                                                       │
                                                       ▼
                                                 atlas-npc-shops
                                                       │
                                        EVENT_TOPIC_NPC_SHOP_STATUS
                                          ENTERED │ ERROR
                          ┌────────────────────────┴───────────────────┐
                          ▼                                            ▼
   atlas-channel npc/shop consumer                   saga-orchestrator npcshop consumer
     • announce NPCShop packet                         • ENTERED → step 1 complete
     • if Registry has charId:                         • ERROR   → step 1 failed
         EnableActions + clear                                    → saga failed
                                                                  → step 2 never runs
                                                       │ step 1 complete
                                                       ▼
                                              destroy_asset_from_slot
                                              (cash compartment, source slot)
```

Two consumers on `EVENT_TOPIC_NPC_SHOP_STATUS`, in different consumer groups:
the channel one already exists (`kafka/consumer/npc/shop/consumer.go:59`,
`:91`); the saga-orchestrator one is new.

### 2.2 Why a saga, and what the alternative was

**Chosen: a two-step saga created by the channel arm.**

The repo already does exactly this for cash items whose effect must land before
the item is destroyed —
`character_cash_item_use_point_reset.go:55-96` builds
`Saga{DestroyAsset, TransferAP|TransferSP}` and hands it to
`saga.NewProcessor(l, ctx).Create(...)`. Ordering, retries, and compensation are
the saga's job, and `destroy_asset_from_slot`
(`libs/atlas-saga/payloads.go:107-115`) is the canonical way to consume from a
specific slot with a compensator that can re-create the asset.

**Rejected alternative — channel-side registry only.** The channel arm would
call `shops.NewProcessor(l, ctx).EnterShop(...)` directly, stash the pending
`{itemId, slot}` in a registry, and destroy the item from inside the existing
`handleEnteredStatusEvent`. Precedent exists (`shopscanner` registry +
auto-enter after warp, `kafka/consumer/character/consumer.go:272-282`). This is
materially less machinery — no new `Action`, no `EventKind`s, no acceptance-table
entry, no new consumer group.

It was rejected on **durability**: the pending state would live in one channel
pod's memory. A pod restart between `ENTER` and `ENTERED` leaves the item
unconsumed. That failure direction is player-favourable, so it is not a
correctness disaster — but Atlas already owns a durable mechanism for exactly
this, and duplicating item-consumption logic outside the saga fragments where
"consume an item only if the effect landed" lives. If the reviewer prefers the
lighter path, the only pieces that change are §3.2 and §3.3; §3.1, §3.4 and
§3.5 are unaffected.

**Rejected alternative — consume first, open second** (Cosmic's shape). Any
failure after the destroy costs the player a paid item. The PRD forbids it.

### 2.3 Why the unlock needs its own registry

The saga cannot unlock the client: it has no session. The channel's existing
`handleEnteredStatusEvent` has the session but is shared with the ordinary
"talk to an NPC" shop path, and NFR "no wire change to an already-verified
version" forbids adding an unconditional `EnableActions` there — v61/72/79/83/84
shop cells are ✅ today and must stay byte-identical for the NPC-talk flow.

So the unlock is **conditional on a remote-initiated open**. A small
`remotemerchant` registry in atlas-channel (singleton `sync.Once` +
`sync.RWMutex`, per the repo's registry idiom) records
`(tenant, characterId) → {itemId, slot, at}`. `handleEnteredStatusEvent` and
`handleErrorStatusEvent` consult it after their existing work; a hit sends
`EnableActions` and clears the entry. A TTL sweep (same shape as the other
channel registries) clears stale entries so a dropped event cannot leave a
character permanently locked — the sweep also sends `EnableActions` on eviction.

This registry carries **presentation state only**. Losing it costs an unlock,
not an item; the item's fate is entirely the saga's.

---

## 3. Detailed design

### 3.1 atlas-channel — the handler arm

New file `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant.go`,
following the megaphone / point-reset file-per-arm precedent.

Dispatch is **classification-first**, inserted into
`CharacterCashItemUseHandleFunc` next to the existing
`ClassificationMegaphones` branch (`character_cash_item_use.go:503-507` explains
why: the cash-slot type byte collides — 37 is also the wedding-ticket bucket and
59/60 are also triple-megaphone buckets):

```go
if category == item.ClassificationRemoteMerchant {
    handleRemoteMerchantUse(l, ctx, wp)(s, t, itemId, source, it)
    return
}
```

No `Decode` call — §1.2 established the sub-body is empty.

`handleRemoteMerchantUse` order of operations:

1. **Ticket short-circuit.** `itemId/1000 == 5451` → distinct warn naming the
   item id and version, `EnableActions`, return. Never consumed. (Reaching this
   branch means a crafted packet; no client emits it.)
2. **Version gate.** Resolve `constants.For(t.Region(), t.MajorVersion(),
   t.MinorVersion())` and consult a `remoteMerchantEnabled(t)` predicate
   implementing §1.3's table with the `MajorAtLeast` idiom — never a raw
   `> N` comparison (`[[bug_majorversion_gt83_is_off_by_one_v87]]`). Disabled →
   distinct greppable warn (NFR "no silent drops"), `EnableActions`, return.
3. **Cash-slot type sanity.** Assert `it` equals the version's expected value
   (37 / 38); mismatch → warn + `EnableActions`. Validation only, never
   dispatch (FR-1.2).
4. **Ownership re-validation.** `cashItemInSlotFunc(l, ctx, characterId, source)`
   must return `itemId` (the seam the incubator/vicious-hammer arms already
   use). Mismatch → warn + `EnableActions`.
5. **Resolve the NPC.** `atlas-data` cash-item read → `Npc`. Zero or error →
   warn + `EnableActions`. Never hard-coded (DOM-25,
   `[[feedback_client_wire_values_config_resolved]]`).
6. **Register the pending unlock**, then **create the saga**. Registry first, so
   a very fast `ENTERED` cannot race ahead of the registry write.

Every rejection path logs its reason and sends `EnableActions`; the success path
logs item id, cash-slot type, resolved NPC template id, and saga transaction id
(NFR Observability).

### 3.2 libs/atlas-saga — new action and payload

```go
// model.go, in a new "NPC shop actions" block
OpenNpcShop Action = "open_npc_shop"

// payloads.go
// OpenNpcShopPayload represents the payload required to open an NPC shop for a character.
type OpenNpcShopPayload struct {
    CharacterId   uint32     `json:"characterId"`
    WorldId       world.Id   `json:"worldId"`
    ChannelId     channel.Id `json:"channelId"`
    NpcTemplateId uint32     `json:"npcTemplateId"`
}
```

`world.Id`/`channel.Id` are the shared `libs/atlas-constants` types (DOM-21).
`unmarshal.go` gains the `open_npc_shop` case; `unmarshal_completeness_test.go`
and `payloads_test.go` keep coverage green.

A new `SagaType` (`RemoteMerchant`) names the saga; `character_extractor.go`
gains an `OpenNpcShopPayload` case so the orchestrator can attribute the saga to
a character.

### 3.3 atlas-saga-orchestrator — handler, events, acceptance

- **Handler.** `handleOpenNpcShop` mirrors `handleShowStorage`
  (`handler.go:1858-1871`) but is **not** self-completing: it produces
  `COMMAND_TOPIC_NPC_SHOP` / `ENTER` with
  `shops.Command[CommandShopEnterBody]{CharacterId, Type: "ENTER",
  Body: {NpcTemplateId}}` and returns, leaving the step `Pending`. This is a new
  producer for an existing contract
  (`services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go`).
- **Event kinds.** `EventKindNpcShopEntered`, `EventKindNpcShopError`.
- **Acceptance table.** `sharedsaga.OpenNpcShop: {EventKindNpcShopEntered,
  EventKindNpcShopError}` — the same two-entry success/error shape as
  `AcceptToStorage` etc. `event_acceptance_test.go`'s coverage assertion stays
  green because the new action is added to both the table and the test's action
  list.
- **Consumer.** New consumer for `EVENT_TOPIC_NPC_SHOP_STATUS` mapping
  `ENTERED` → `EventKindNpcShopEntered`, `ERROR` → `EventKindNpcShopError`.
  Its group id is distinct from atlas-channel's, so both services see every
  event.
- **Idempotency.** A redelivered `ENTERED` for an already-completed step is a
  no-op (the orchestrator's step matching already ignores events for
  non-`Pending` steps); `destroy_asset_from_slot` is slot+item-id scoped and
  carries `TemplateId` so its compensator can restore.
- **Compensation.** If `destroy_asset_from_slot` fails after the shop opened,
  the step's compensator emits `COMMAND_TOPIC_NPC_SHOP` / `EXIT`
  (FR-4.5). This is the one place where the saga earns its keep over the
  registry-only alternative.

The saga the channel builds:

```go
saga.Saga{
    SagaType: saga.RemoteMerchant,
    Steps: []saga.Step{
        {Status: saga.Pending, Action: saga.OpenNpcShop,
         Payload: saga.OpenNpcShopPayload{CharacterId, WorldId, ChannelId, NpcTemplateId}},
        {Status: saga.Pending, Action: saga.DestroyAssetFromSlot,
         Payload: saga.DestroyAssetFromSlotPayload{
            CharacterId, InventoryType: 5 /* cash */, Slot: source,
            Quantity: 1, ShowEffect: false, TemplateId: uint32(itemId)}},
    },
}
```

Step 1 failing means step 2 never runs — the item survives, which is FR-4.4.

### 3.4 atlas-data — expose `info/npc` for cash items

`services/atlas-data/atlas.com/data/cash/reader.go` parses `slotMax`,
`protectTime`, `stateChangeItem`, `rate`, the `spec` subtree — but **not**
`npc`. (`consumable/reader.go:75` already does
`m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))`, and `consumable/rest.go:74`
exposes it — the cash domain simply never got the field.)

Change: add `Npc uint32` to the cash model, read it with
`GetIntegerWithDefault("npc", 0)`, and expose it as `"npc"` on the cash REST
model — additive, JSON:API-shaped, mirroring the consumable domain verbatim.
`reader_test.go` gains a case asserting 5450000 → 9090000 and 5451000 → 0.

Verified WZ ground truth (`Item.wz/Cash/0545.img.xml`): `npc` = `9090000` for
5450000, `0` for 5451000 — exactly the PRD's table.

### 3.5 Shop seed data

Ten files, `deploy/seed/gms/{12,48,61,72,79,83,84,87,92,95}_1/npc-shops/shops/shop-9090000.json`,
in the existing envelope (verified against `shop-1001000.json`):

```json
{ "data": { "attributes": { "commodities": [ {...} ], "npcId": 9090000, "recharger": true },
            "id": "9090000", "type": "npc-shop" } }
```

Commodity shape per existing seeds: `discountRate`, `levelLimit`, `mesoPrice`,
`period`, `templateId`, `tokenPrice`, `tokenTemplateId`. The 26 rows are the
PRD's FR-5.2 table.

`"recharger": true` makes `atlas-npc-shops` append the Redis-backed
star/bullet listings (`shops/cache.go`, `data/consumable/processor.go:GetRechargeable`),
which is what the item description promises.

Seeding all ten versions — including the three where the item cannot be used —
is deliberate: the shop belongs to NPC 9090000, and a future NPC-talk route to
the same merchant should find it. An unreachable shop row is inert.

**OQ-5 is an implementation-time sweep, not a guess.** Only one WZ set is
available locally (`~/source/Cosmic/wz`, a v83-era set), so per-version
existence cannot be settled from this worktree. The plan phase gets an explicit
task: for each of the ten versions, query the ingested `atlas-data` for each of
the 26 template ids and drop the misses from that version's file, recording the
drop list in the task folder. Candidates most likely to be absent pre-v83:
`5041000` (Teleport Rock), `2022189`/`2022190`/`2022191`/`2022195`,
`2002023`-`2002025`. **No row is seeded without a per-version existence check.**

### 3.6 Template registrations

Current state re-counted from `services/atlas-configurations/seed-data/templates/`
matches the PRD's table. Changes:

| Template | Add | opCode | Source |
|---|---|---|---|
| gms_48 | `NPCShop` writer (`CShopDlg::SetShopDlg`) | `0xE5` | IDB `0x5b7a38`, §1.2 |
| gms_48 | `NPCShopOperation` writer (`CShopDlg::OnPacket`) | `0xE6` | IDB `0x5b7a38`, §1.2 |
| gms_87 | `NPCShopHandle` | `0x040` | `docs/packets/audits/STATUS.md:572` |
| gms_92 | `NPCShopHandle` | `0x043` | STATUS.md:572 |
| gms_92 | `NPCShop` writer | `0x164` | STATUS.md:381 |
| gms_92 | `NPCShopOperation` writer | `0x165` | STATUS.md:383 |
| gms_95 | `NPCShopHandle` | `0x042` | STATUS.md:572 |

Each handler entry copies gms_83's shape verbatim
(`template_gms_83_1.json:520-536`) — `"validator": "LoggedInValidator"`,
`"fname": "CShopDlg::SetRet"`, `"options.operations"` =
`{BUY:0, SELL:1, RECHARGE:2, LEAVE:3}`, `"services": ["channel"]`. The
non-empty validator is mandatory
(`[[bug_socket_handler_missing_validator_silently_dropped]]`). Each writer entry
copies gms_83's `0x131`/`0x132` blocks including the full
`NPCShopOperation` operations table.

The gms_48 `operations` values are **not** assumed to match gms_83: v48's
`CShopDlg::OnPacket` Decode1 switch has its own arm-to-message mapping, and the
`operations` tables are exactly the kind of thing that drifts across generations
(`[[bug_gms_61_72_79_interaction_operations_wrong]]`,
`[[bug_operations_mode_tables_missing_v87_v95_jms]]`). The plan phase derives
v48's table from `0x5b7a38` arm by arm before writing the entry.

Ordering and duplicate guards (`tools/template-opcode-order-guard.sh`,
`tools/template-duplicate-binding-guard.sh`) run on every edited template.

**FR-6.6 — live tenant reconciliation.** Template edits are inert for
already-provisioned tenants
(`[[bug_new_opcodes_not_in_live_tenant_config]]`). Follow
`[[reference_reconcile_live_tenant_socket_to_template]]`, then verify by
**reading back the live configuration**, not by asserting the template file
changed.

### 3.7 Packet coverage

`/verify-packet` per cell, each with a committed byte fixture and evidence
record:

- serverbound `NPC_SHOP` × {v87, v92, v95} — newly reachable via FR-6.1.
- clientbound `OPEN_NPC_SHOP`, `CONFIRM_SHOP_TRANSACTION` × v92 — newly bound.
- clientbound `OPEN_NPC_SHOP`, `CONFIRM_SHOP_TRANSACTION` × v48 — ⬜ → verified
  at `0xE5`/`0xE6`.

No serverbound codec is added (§1.2), so there is no new `libs/atlas-packet`
cell to promote for the cash-item side. `packet-audit matrix --check` and
`fname-doc --check` must exit 0.

---

## 4. Behaviour walk-throughs

### 4.1 Happy path (gms_83, MiuMiu in cash slot 3)

Client checks field bit 18, finds it clear, sets `m_bExclRequestSent`, sends
`CASH_ITEM_USE{updateTime, source=3, itemId=5450000}` with no sub-body →
channel arm validates version/type/ownership → `atlas-data` yields npc 9090000 →
registry records `{5450000, 3}` → saga created → `open_npc_shop` produces
`ENTER` → `atlas-npc-shops` emits `ENTERED` → channel announces `NPCShop`
(existing code) and, seeing the registry hit, sends `EnableActions` and clears →
saga-orchestrator completes step 1 and runs `destroy_asset_from_slot` → one
5450000 leaves the cash compartment. Buy/sell/recharge then flow through the
already-registered `NPCShopHandle`.

### 4.2 Shop missing / service error

`atlas-npc-shops` emits `ERROR` → channel writes the existing shop-error packet
(existing code), sees the registry hit, sends `EnableActions`, clears →
saga-orchestrator fails step 1 → saga fails → `destroy_asset_from_slot` never
runs → **item still in inventory, client unlocked, reason logged.**

### 4.3 Character already in a shop (FR-2.3)

The authoritative state lives in `atlas-npc-shops`. Design decision: **no
channel-side pre-check.** A channel-side guard would need to mirror
`atlas-npc-shops`' state and could disagree with it. The service's `ERROR` reply
is the single source of truth and routes into §4.2 — item kept, client unlocked.
The plan phase adds one verification task: confirm `atlas-npc-shops` actually
replies `ERROR` (rather than silently re-entering) when a character with an open
shop receives `ENTER`; if it silently re-enters, the fix belongs in
`atlas-npc-shops`, not in a channel-side mirror.

### 4.4 Remote Gachapon Ticket

No client sends it. If one ever did (crafted packet), the arm logs
`… attempted remote gachapon ticket [5451000]; no client build emits this op —
ignoring without consuming.` and unlocks. Nothing is destroyed.

---

## 5. Testing

| Layer | Test |
|---|---|
| atlas-channel arm | Table test over §1.3's eleven templates asserting enabled/disabled routing; ownership-mismatch → `EnableActions` + no saga; ticket id → warn + no saga; `atlas-data` error → `EnableActions` + no saga. Uses the existing `cashItemInSlotFunc` package-var seam and the Builder pattern (no `*_testhelpers.go`). |
| libs/atlas-saga | `open_npc_shop` round-trips through `unmarshal.go`; `unmarshal_completeness_test.go` stays green. |
| saga-orchestrator | `event_acceptance_test.go` coverage assertion includes the new action; step-matching test: `ERROR` fails the saga and `destroy_asset_from_slot` never dispatches; duplicate `ENTERED` is a no-op; destroy-failure compensator emits `EXIT`. |
| atlas-channel registry | ENTERED-with-hit unlocks and clears; ENTERED-without-hit does **not** unlock (protects the NPC-talk path); TTL eviction unlocks. |
| atlas-data | `cash/reader_test.go`: 5450000 → npc 9090000, 5451000 → npc 0. |
| Seed data | Existing seed-shape test extended, or a new one, asserting `shop-9090000.json` parses, carries `recharger: true`, and contains only version-present template ids. |
| Templates | The three guard scripts, plus `/verify-packet` byte fixtures for §3.7's cells. |

Cross-service seams get event-level tests, not stubs
(`[[feedback_green_tests_miss_cross_service_seams]]`): the saga tests assert on
the produced `COMMAND_TOPIC_NPC_SHOP` body, and the channel tests assert on the
consumed `EVENT_TOPIC_NPC_SHOP_STATUS` body — the two halves of the contract
that no compiler checks.

---

## 6. PRD deltas

| PRD item | Design outcome |
|---|---|
| FR-1.3 (derive sub-body; possible new codec) | **Empty sub-body**, no codec. Evidence §1.2. |
| FR-1.5 (unlock only if `SetShopDlg` doesn't) | **Server must unlock.** §1.2 OQ-2; mechanism §2.3. |
| FR-3.1-3.4 (remote gachapon) | **Removed.** No client emits the op. Replaced by a warn + unlock. Drops the `atlas-npc-conversations` / `atlas-reward-pools` work entirely. |
| FR-2.3 (already-in-shop guard) | Service-side (`ERROR`), no channel mirror. §4.3. |
| FR-5.2 (26 commodities) | Kept, with a **mandatory** per-version existence sweep before seeding. §3.5. |
| FR-6.3 (gms_48 ⬜ cells) | **Register**, not `n-a`: `0xE5` / `0xE6`. §1.2 OQ-4. |
| FR-7.2 (serverbound codec) | Dissolved — nothing to encode. |
| Non-goal "map restrictions" | Confirmed client-side (field attr bit 18); Atlas does not replicate. §1.2 OQ-6. |
| Service Impact: `libs/atlas-packet` | **No change.** |
| Service Impact: `atlas-npc-conversations`, `atlas-reward-pools` | **No change.** |
| New: version gate | gms_12 / gms_48 / gms_61 excluded from the use-flow with IDB evidence. §1.3. |

---

## 7. Risks and open items

**7.1 gms_48 shop opcodes are IDB-derived, not yet fixture-verified.** `0xE5` /
`0xE6` come from reading `CShopDlg::OnPacket` @ `0x5b7a38`; the same procedure
reproduces v83's known-correct `0x131`/`0x132`, and v48 has a distinct
`CTrunkDlg` family so the dialog identity is not confused. Still, the cells only
move off ⬜ through `/verify-packet` with a byte fixture. If v48's writer body
layout differs from v83's, that is a codec-level version gate discovered during
verification, not a design change.

**7.2 gms_48 `NPCShopOperation` operations table.** Derived from the v48 arm
switch during the plan phase, never copied from gms_83. Copying would reproduce
the class of bug recorded in
`[[bug_gms_61_72_79_interaction_operations_wrong]]`.

**7.3 JMS185 maps classification 545 to cash-slot type 36, and Atlas returns
37.** `get_cashslot_item_type` @ `0x49a1ee` returns `36` with no `5451` branch;
JMS's send arm for 36 exists. Atlas's `GetCashSlotItemType` gates only on
`Region()=="GMS" && MajorVersion()>=95`, so a JMS tenant gets `37`. The PRD
scopes shop seeds and template work to the ten GMS versions and says nothing
about `template_jms_185_1.json`, so **this task does not enable JMS**; the arm's
version gate (§3.1 step 2) returns false for JMS, and the mismatch is inert
because nothing consumes the JMS value today. It is recorded here so the
follow-up — JMS type `36` plus a JMS template/shop pass — is a known, bounded
piece of work rather than a rediscovery.

**7.4 The unlock registry is per-pod, in-memory.** Losing it costs one
`EnableActions`, never an item. The TTL sweep bounds the worst case; a player
whose unlock is lost recovers by any action that re-issues `EnableActions`.

**7.5 OQ-5 remains open until the plan phase's existence sweep runs.** It is
explicitly a task, not an assumption; no commodity row ships unverified.
