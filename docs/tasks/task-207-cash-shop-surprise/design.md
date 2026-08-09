# Cash Shop Surprise — Design

Task: task-207-cash-shop-surprise
PRD: [`prd.md`](prd.md) (approved)
RE checklist: [`re-checklist.md`](re-checklist.md) (answered in full — §1 below)
Status: proposed
Date: 2026-08-09

---

## 0. Summary of what changed relative to the PRD

The RE pass required by `re-checklist.md` ran against all ten IDBs before any
architecture was chosen. It invalidated three premises the PRD was written on.
Every correction below is backed by a decompile citation in §1.

| PRD statement | Reality | Consequence |
|---|---|---|
| FR-5.1: on v84/v87/v95 the success result "MUST reuse the existing `GachaponOpenDone` codec" | `GachaponOpenDone` is the `CCashShop::OnCashItemResCashGachaponOpenDone` arm, which calls **`CUICashGachapon`** — the Cash Gachapon UI the PRD lists as a **non-goal** (§2). The Surprise box's result never passes through `CASHSHOP_OPERATION` on any version. | The Surprise box uses the standalone `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` opcode on **every** supported version. `GachaponOpenDone` is not touched. |
| FR-6.1/6.3: failures reported via `BuyFailed` with distinct numeric error codes | The standalone result opcode carries its own two-arm mode byte: SUCCESS and FAILED. The FAILED arm has an **empty body** and maps to one fixed StringPool message. There is no error-code field anywhere on this packet. | Failure = the native FAILED mode. No error code on the wire; the distinct reasons are logged server-side. |
| FR-3.5/3.6 + §6: new `box_template_id` column + new roll-by-template endpoint | The incubator pool already keys a pool by the **item id as the pool slug** (`PoolFormDialog.tsx:154` — `id: String(values.eggItemId)`), and `POST /gachapons/{id}/rewards/select` already rolls by that slug. | No new column, no new endpoint. A `cash-surprise` pool's slug **is** the box template id. |

Two further findings that expand scope in the PRD's favour:

- **v79 is not the empty column the PRD assumed.** v79 has a live
  `CUICashItemGachapon`, and its cash-locker double-click handler compares the
  clicked item's template id against the literal **5222000**
  (`sub_4A68B0` @ `0x4a6a00`, constant `0x4FAE70` = 5222000) before
  constructing the dialog. But v79's `CCashShop::OnPacket` has **no** result
  case. See §5 for the decision this forces.
- **jms_v185 has the full feature**, serverbound included — the PRD listed its
  trigger as "absent from registry" (it is absent from the *registry*, not from
  the *binary*). It sends `0xA7`.

---

## 1. Reverse-engineering results

All ten IDBs were resolved from `idb_list` by binary name and read via
`func_query`/`decompile`. Nothing below is inferred from a registry entry or
from a symbol name alone.

### 1.1 Per-version matrix (verified)

| Version | IDB session | `CUICashItemGachapon` | Send opcode | Result opcode(s) | SUCCESS mode | FAILED mode | Fail StringPool id |
|---|---|---|---|---|---|---|---|
| gms_v48 | `93cc947e` | **absent** | — | `0x101` → 1-byte flag store (`0x4536b9`) | — | — | — |
| gms_v61 | `415bf585` | **absent** | — | `0x100` → 1-byte flag store (`0x46128a`) | — | — | — |
| gms_v72 | `c8acae95` | **absent** | — | `0x124` → 1-byte flag store (`0x470d24`) | — | — | — |
| gms_v79 | `1438cecd` | **present** (`0x8ef2a0`+) | `0x9F` (`0x8efda6`) | **none** | — | — | — |
| gms_v83 | `41f13e0d` | present | `0xA1` (`0x99a9a7`) | `0x14D` only (`0x478e2b`) | `0xE5` (229) | `0xE4` (228) | 538 |
| gms_v84 | `5881cf84` | present | `0xA5` (`0x9db82d`) | `0x154`, `0x155` (`0x47bf59`) | `0xEE` (238) | `0xED` (237) | 541 |
| gms_v87 | `d51ecbd3` | present | `0xA9` (`0xa215f6`) | `0x15E`, `0x15F` (`0x4844a4`) | `0xF4` (244) | `0xF3` (243) | 548 |
| gms_v92 | `acdfccff` | present | `0xB6` (`0x944110`) | `0x180`, `0x181` (`0x495770`) | `0xBE` (190) | `0xBD` (189) | 556 |
| gms_v95 | `79906a1e` | present | `0xB9` (`0x96aa40`) | `0x188`, `0x189` (`0x4997e0`) | `0xC1` (193) | `0xC0` (192) | 556 (`0x22C`) |
| jms_v185 | `b6864e54` | present | `0xA7` (`0xa6e309`) | `0x16D` only (`0x48b21d`) | `0xEB` (235) | `0xEA` (234) | 579 (`0x243`) |

Registry cross-check: v83 `0xA1`, v95 `0xB9`, v83 `0x14D`, v95 `0x188`, jms
`0x16D`, v48 `0x101`, v61 `0x100`, v72 `0x124` all agree with the binaries. The
v83 serverbound entry's `provenance: csv-import`
(`docs/packets/registry/gms_v83.yaml:2856-2860`) can be upgraded — the value is
now IDA-derived. Two registry entries are **missing** and must be added:
gms_v79 serverbound `0x9F` and jms_v185 serverbound `0xA7`.

Two IDB functions were renamed during this pass (per the "name symbols while
reversing" rule): v83 `sub_99A9A7` →
`CUICashItemGachapon__OnButtonClicked_send_0xA1`, v92 `sub_944110` →
`CUICashItemGachapon__OnButtonClicked_send_0xB6`.

### 1.2 Serverbound body — identical on every version that has it

```
CUICashItemGachapon::OnButtonClicked(nId == 2000):
    COutPacket(<send opcode>)
    EncodeBuffer(&m_liItemSN, 8)      // the box's cash SN (LARGE_INTEGER)
    SendPacket
```

Guarded by `if (m_nState < 1)` — the client self-gates re-clicks and hides the
Open button until the dialog is reopened. Only **v79** additionally calls
`CWvsContext::SetExclRequestSent`; on every version in scope the send does
**not** arm the excl-request gate, so no `EnableActions`/unlock is owed
(cf. the EXCL-request contract note in project memory).

### 1.3 Clientbound body — identical on every version that has it

```
CCashShop::OnCashItemGachaponResult(iPacket):
    mode = Decode1()
    if mode == <SUCCESS>:
        DecodeBuffer(liSN, 8)         // SN of the consumed box
        remain = Decode4()            // box's new quantity; 0 removes the locker row
        DecodeBuffer(newItem, 0x37)   // 55-byte GW_CashItemInfo, appended to m_aCashItemInfo
        if (m_pUICashItemGachapon)    // dialog open → hand the rest to the UI
            CUICashItemGachapon::OnCashItemGachaponResult(iPacket):
                itemId  = Decode4()   // rewarded template id (icon + chat log)
                count   = Decode1()
                jackpot = Decode1()   // selects CashGachaponJackpot vs CashGachaponNormal sfx
    elif mode == <FAILED>:
        StringPool::GetString(<fixed id>) ; CUtilDlg::Notice(...)   // no body read
```

Notes that shape the design:

- The trailing three fields are read by the **UI object**, not by `CCashShop`.
  jms is the only build that guards the no-dialog case explicitly
  (`CWnd::InvalidateRect` instead). On GMS the trailing fields are simply left
  in the buffer when no dialog is open — harmless. The server always writes
  them.
- The 55-byte blob is byte-identical to `CashInventoryItem.EncodeBytes`
  (`shop_inventory.go:27-39`: 8+4+4+4+4+2+13+8+4+4 = 55). Reuse it verbatim.
- The FAILED arm reads **nothing**. There is no error code to carry.
- On failure the client shows a notice but does **not** re-enable the dialog's
  Open button (`m_nState`/`m_bCloseButtonEnabled` are only reset by the SUCCESS
  path). This is native behaviour; the player closes and reopens the dialog. We
  replicate it rather than inventing a recovery packet.

### 1.4 `GACHAPON_OPEN_*` is a different feature (PRD §9 Q4 dissolved)

`CCashShop::OnCashItemResCashGachaponOpenDone` (v95 `0x494ac0`) calls
`CUICashGachapon::OnCashGachaponOpenResult` (`0x968ee0`) — the Cash **Gachapon**
UI, i.e. `CASH_GACHAPON_BUTTON` / `CUICashGachapon::OnButtonClicked`, explicitly
a **non-goal** in PRD §2. Its wire shape also differs from the Surprise result
(`isCashItem` gate byte before the blob; `resultCode`/`resultParam2` trailing
instead of `itemId`/`count`/`jackpot`). Q4 therefore needs no answer for this
task, and `shop_operation_result_gachapon.go` is not modified.

### 1.5 Matrix corrections this pass produces

- Row `CASHSHOP_CASH_GACHAPON_OPEN_RESULT` (`STATUS.md:486`) shares the fname
  `CCashShop::OnCashItemGachaponResult` and lists v83 `0x14E`. **v83's
  dispatcher has no `0x14E` case** (`0x478e2b`) — that cell is `n-a`, not
  merely unverified. The alias (two consecutive opcodes routed to the same
  handler) exists on v84 (`0x154`/`0x155`), v87 (`0x15E`/`0x15F`), v92
  (`0x180`/`0x181`), v95 (`0x188`/`0x189`) and **not** on v83 or jms.
- task-183's `arm-catalog.md:251` attributes jms `COutPacket(0xA7)` (reached by
  double-clicking item `5222002` in the Cash Locker) to
  `SendChangeMaplePoint`. In this jms build the only sender of `0xA7` in the UI
  code segment (`0xA00000`–`0xB00000`) is
  `CUICashItemGachapon::OnButtonClicked` (`0xa6e39c`), and `5222002` is a
  `522` — gachapon-coupon — classification. The attribution is very likely
  wrong; this task should correct the note. (Scope caveat: the opcode search
  was bounded to the UI segment, not the whole image.)

### 1.6 WZ presence (PRD §9 Q7) — answered against the live baseline

`GET /api/data/cash/items/5222000` per tenant (ten tenants, correct
`TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers):

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|
| 404 | 404 | 404 | 200 | 200 | 200 | 200 | 200 | 200 | 200 |

`5222002` is present only on jms_v185. Where present, the record is
`{"slotMax":0,"spec":{}}` — the box carries **no** WZ spec node, confirming the
drop table is entirely server-owned. The WZ axis and the binary axis agree
exactly: absent on v48/v61/v72, present from v79 onward.

---

## 2. Architecture

### 2.1 Chosen shape

```
client ──0xA1/0xA5/0xA9/0xB6/0xB9/0x16D-trigger──▶ atlas-channel
                                                    │ CashItemGachaponHandle
                                                    │ decode: cashId (int64)
                                                    ▼
                                    COMMAND_TOPIC_CASH_SHOP  (new type OPEN_SURPRISE)
                                                    ▼
                                              atlas-cashshop
                                 ┌──────────────────┴───────────────────┐
                                 │ 1. resolve box asset by cashId,      │
                                 │    assert ownership + templateId     │
                                 │ 2. POST /gachapons/{templateId}/     │  REST
                                 │    rewards/select  ──────────────────┼──▶ atlas-reward-pools
                                 │ 3. resolve commodity (local)         │
                                 │ 4. ONE tx: idempotency insert +      │
                                 │    decrement/release box +           │
                                 │    create reward asset               │
                                 └──────────────────┬───────────────────┘
                                                    ▼
                                 EVENT_TOPIC_CASH_SHOP_STATUS (SURPRISE_OPENED | SURPRISE_FAILED)
                                                    ▼
                                              atlas-channel
                                        CashItemGachaponResult writer
                                                    ▼
                                                  client
```

This is the same command → in-service transaction → status event → writer shape
that the existing cash-shop purchase already uses
(`cashshop/processor.go:98` `Purchase`, `kafka/consumer/cashshop/consumer.go:96`
`handleStatusEventPurchase`). No new architectural pattern is introduced.

### 2.2 Alternatives considered

**A. Saga through atlas-saga-orchestrator** (PRD §9 Q3's other branch).
Rejected. A saga buys cross-service rollback, and there is nothing to roll back:
the roll is a pure read against reward-pools that mutates nothing, and the
consume+grant both live in one database. `MoveFromCashInventory` uses a saga
because it spans atlas-cashshop **and** the character inventory; this operation
does not. Ordering is roll → (consume + grant) so a failed grant loses nothing.

**B. Channel does the roll and sends a fully-resolved command.**
Rejected. It puts a currency-adjacent decision in the edge service and makes the
command replayable-with-a-different-outcome; atlas-cashshop would have to trust
a reward id chosen elsewhere. Keeping the roll behind the same service that
writes the asset means one place enforces "rolled ⇒ granted".

**C. Reuse `CASHSHOP_OPERATION`'s `GACHAPON_OPEN_*` arms.**
Rejected on evidence — §1.4. Those arms belong to a different UI class and a
different feature.

### 2.3 Failure vehicle (supersedes FR-6.1/6.2/6.3)

Send the native FAILED mode of the same opcode. Rationale:

- It is what the client implements for this feature. `BuyFailed` targets
  `CCashShop::OnCashItemResBuyFailed`, which raises the cash-shop purchase
  notice — a second, wrong dialog on top of the gachapon UI.
- The FAILED arm has no error-code field, so FR-6.3's "distinct numeric codes"
  is not expressible on this wire. The distinct reasons
  (`BOX_NOT_FOUND`, `NOT_OWNED`, `NOT_A_SURPRISE_BOX`, `LOCKER_FULL`,
  `POOL_EMPTY`, `POOL_MISSING`, `COMMODITY_MISSING`, `INTERNAL`) are carried on
  the status event and written to the log (NFR "Observability"), not to the
  client.
- FR-6.4 (box left unconsumed on failure) is satisfied structurally: the
  consume only happens inside the success transaction.

### 2.4 Capacity accounting (FR-2.3)

The reward is created before/while the box is consumed, so the peak slot count
matters, not the net. Rule:

- box quantity > 1 → the box row survives, so the grant needs **one free slot**:
  require `len(assets) < capacity`.
- box quantity == 1 → the box row is released, so the grant is slot-neutral:
  `len(assets) <= capacity` is sufficient.

Implement as a single helper with both cases unit-tested; the check runs inside
the transaction against the same read used for the write.

---

## 3. Packet layer

### 3.1 New serverbound codec

`libs/atlas-packet/cash/serverbound/item_gachapon_button.go`

```go
// packet-audit:fname CUICashItemGachapon::OnButtonClicked
type CashItemGachaponButton struct {
    cashId int64
}
func NewCashItemGachaponButton(cashId int64) CashItemGachaponButton
func (m CashItemGachaponButton) CashId() int64
func (m CashItemGachaponButton) Operation() string
Encode: w.WriteInt64(m.cashId)
Decode: m.cashId = r.ReadInt64()
```

`EncodeBuffer(&m_liItemSN, 8)` is byte-identical to a little-endian int64, so no
version gate and no `WriteByteArray` special case. Immutable struct + `New…`
constructor + both directions, matching every sibling in the package.

### 3.2 New clientbound codecs

`libs/atlas-packet/cash/clientbound/item_gachapon_result.go`

```go
const CashItemGachaponResultWriter = "CashItemGachaponResult"

const (
    CashItemGachaponModeSuccess = "SUCCESS"
    CashItemGachaponModeFailed  = "FAILED"
)

// packet-audit:fname CCashShop::OnCashItemGachaponResult#SUCCESS
type CashItemGachaponSuccess struct {
    mode    byte
    sn      int64             // consumed box's cash SN
    remain  int32             // box's new quantity (0 removes the row)
    newItem CashInventoryItem // 55-byte GW_CashItemInfo blob
    itemId  int32             // rewarded template id (UI icon + chat log)
    count   byte
    jackpot byte
}

// packet-audit:fname CCashShop::OnCashItemGachaponResult#FAILED
type CashItemGachaponFailed struct {
    mode byte
}
```

Body providers use the established DOM-25 idiom
(`vega_scroll.go:171` precedent):

```go
func CashItemGachaponSuccessBody(sn int64, remain int32, item CashInventoryItem,
        itemId int32, count byte, jackpot byte) ... {
    return atlas_packet.WithResolvedCode("operations", CashItemGachaponModeSuccess,
        func(mode byte) packet.Encoder {
            return NewCashItemGachaponSuccess(mode, sn, remain, item, itemId, count, jackpot)
        })
}
func CashItemGachaponFailedBody() ... // same, CashItemGachaponModeFailed
```

The mode byte is **never** hard-coded — it differs on every single version
(§1.1), which is precisely the DOM-25 failure mode the rule exists for.

`newItem` uses the existing `CashInventoryItem` type. No change is made to
`shop_operation_result_gachapon.go`, `shop_operation_result_failed.go`, or any
already-✅ codec: this task adds files, it does not edit shared ones.

### 3.3 Template routing (six templates)

New handler entry, at its sorted `opCode` position, in
`services/atlas-configurations/seed-data/templates/`:

| template | handler `opCode` | writer `opCode` |
|---|---|---|
| `template_gms_79_1.json` | — (see §5) | — |
| `template_gms_83_1.json` | `0xA1` | `0x14D` |
| `template_gms_84_1.json` | `0xA5` | `0x154` |
| `template_gms_87_1.json` | `0xA9` | `0x15E` |
| `template_gms_92_1.json` | `0xB6` | `0x180` |
| `template_gms_95_1.json` | `0xB9` | `0x188` |
| `template_jms_185_1.json` | `0xA7` | `0x16D` |

```json
{ "opCode": "0xA1", "validator": "LoggedInValidator",
  "handler": "CashItemGachaponHandle",
  "fname": "CUICashItemGachapon::OnButtonClicked", "services": ["channel"] }
```

```json
{ "opCode": "0x14D", "writer": "CashItemGachaponResult",
  "fname": "CCashShop::OnCashItemGachaponResult",
  "options": { "operations": { "SUCCESS": 229, "FAILED": 228 } },
  "services": ["channel"] }
```

`validator` must be non-empty or the handler is silently dropped
(`bug_socket_handler_missing_validator_silently_dropped`); `LoggedInValidator`
matches every sibling cash-shop handler. `fname` is required on writers
(`bug_seed_template_writers_require_fname`). The v48/v61/v72 templates get
nothing. Both `tools/template-opcode-order-guard.sh` and
`tools/template-duplicate-binding-guard.sh` must pass — insert at sorted
position, never adjacent to a semantically-related entry.

Live tenants also need the new opcodes pushed into their socket configuration,
or the handler is present in code and absent at runtime
(`bug_new_opcodes_not_in_live_tenant_config`).

---

## 4. Service designs

### 4.1 `atlas-reward-pools`

**Kind.** Add `KindCashSurprise = "cash-surprise"` to the closed union
(`gachapon/builder.go:15-16`) and accept it at `builder.go:77`. `DefaultKind`
stays `gachapon`.

**Selection.** `reward/processor.go` currently branches
`KindIncubator` → whole-machine, flat `item.Weight`, **no** global merge; else →
tiered + global merge. `cash-surprise` takes the **same branch as incubator**
(PRD §9 Q2 resolved: flat per-entry weights). This satisfies FR-3.2 (no global
merge) and FR-3.4 (weighted) with no second selection path. Rewrite the branch
condition as an explicit "kind uses flat weights" predicate covering both kinds,
so the third kind doesn't accrete a copy of the body.

**Pool identity.** A `cash-surprise` pool's slug (`gachapons.id`) **is** the box
template id, exactly as an incubator pool's slug is the egg item id. This
removes the PRD's proposed `box_template_id` column, its uniqueness index, and
the proposed new endpoint: `POST /gachapons/5222000/rewards/select` already
exists (`reward/resource.go:24`) and already 404s when no pool matches.
FR-3.5's "`npcIds` may be empty" is already true — `npcIds` is
`not null` but an empty array is valid, and incubator pools already leave it
empty.

**Commodity ids (FR-3.3).** One additive column:
`gachapon_items.commodity_id uint32 not null default 0`. `reward.Model` and its
builder/REST model gain `CommodityId`. For `cash-surprise` entries `item_id` is
advisory (display only) and `commodity_id` is authoritative; the builder
rejects a `cash-surprise` entry with `commodity_id == 0`. Existing rows read 0
and are untouched — no backfill (matches how `Weight` was added).

**Empty pool (FR-3.7).** `SelectReward` already returns
`errors.New("no items available in pool for tier: ")` for an empty pool. Replace
the anonymous error with a sentinel `ErrEmptyPool`, map it to **409** in
`reward/resource.go`, and keep 404 for "no such pool". Both are then
distinguishable by the caller — 404 → `POOL_MISSING`, 409 → `POOL_EMPTY`.

**Determinism (NFR).** Follow the existing precedent: `crypto/rand` stays inside
`selectItem`, and determinism is achieved by unit-testing the pure
`selectWeightedIndex(pool, roll)` at its boundaries. No RNG injection is added.

### 4.2 `atlas-cashshop`

New `SurpriseProcessor` (or a method group on the existing cashshop processor,
matching how `Purchase` sits there today) with the
pure-`(mb)` / `…AndEmit` pairing the codebase uses everywhere.

```
OpenSurpriseAndEmit(transactionId, accountId, characterId, cashId) error
```

Sequence:

1. **Resolve.** Find the compartment for the character's job type
   (Explorer/Cygnus/Legend, same three-way as `Purchase`) and the asset whose
   `cashId` matches. Reject when absent, when its compartment does not belong to
   `accountId` (FR-2.1), or when its `templateId` is not a configured Surprise
   box id (FR-2.2). Failure ⇒ `SURPRISE_FAILED`, no state change.
2. **Capacity.** §2.4. Failure ⇒ `SURPRISE_FAILED`, no state change.
3. **Roll.** `POST /gachapons/{templateId}/rewards/select` against
   atlas-reward-pools, via the `requests.Provider` idiom already used by
   `atlas-channel/incubator/requests.go:30`. 404/409/transport error ⇒
   `SURPRISE_FAILED`, no state change. Nothing is mutated before this point, so
   FR-4.1's "partial application" hazard cannot arise from a failed roll.
4. **Resolve the commodity.** `commodity.Processor.GetById(commodityId)` gives
   `itemId`, `count`, `period`. Missing commodity ⇒ `SURPRISE_FAILED`.
5. **One transaction** (`database.ExecuteTransaction` + `message.Emit(outbox…)`,
   the `PurchaseAndEmit` shape):
   - insert the idempotency row (§4.3); unique violation ⇒ commit nothing,
     return success-without-effect;
   - decrement the box by 1, releasing the row at 0 (`asset.UpdateQuantity` /
     `asset.Release`) — FR-4.3;
   - create the reward with `asset.Create(mb)(compartmentId, itemId,
     commodityId, count, 0, characterId)` — FR-4.2. `asset.Create` already
     derives `expiration` from the commodity's `period`
     (`asset/processor.go:84-95`), so the PRD's expiration requirement is met by
     reuse.
   - emit `SURPRISE_OPENED` carrying everything the writer needs.

   **Deviation from FR-4.2's letter:** the PRD says "via the existing
   compartment `Accept` path". `compartment.Accept` is the saga-facing inbound
   path and emits `ACCEPTED` status events for a saga to correlate. The
   in-service creation path is `asset.Create`, which is what `Purchase` uses and
   which produces the identical flattened row (`cashId`, `commodityId`,
   `templateId`, `quantity`, `expiration`, `purchasedBy`). FR-4.2's intent —
   "a normal flattened cash asset, fully populated" — is satisfied; its named
   mechanism is not the right one here.

**Surprise box ids (FR-2.2).** Tenant configuration, not a constant, under the
existing `configuration/tenant/cashshop` surface: a list of template ids, default
`[5222000]`. A list rather than a scalar so a tenant can designate additional
boxes. FR-4.5's recursion risk (a pool that can award a Surprise box) is
honoured-by-configuration, not blocked in code; the risk is called out here and
in the UI copy.

### 4.3 Idempotency (FR-4.4)

New table, owned by atlas-cashshop:

```
cash_surprise_openings(
    tenant_id      uuid   not null,
    transaction_id uuid   not null,
    account_id     uint32 not null,
    asset_id       uint32 not null,
    created_at     timestamptz not null,
    primary key (tenant_id, transaction_id)
)
```

The insert is the **first** statement in the transaction. A redelivered command
hits the primary-key violation, the transaction aborts, and nothing is granted.
`transactionId` is minted by atlas-channel per click and travels on the command,
so a Kafka redelivery replays the same id while a genuine second click gets a
new one.

This is deliberately a real ledger row rather than an optimistic
compare-and-set on the box quantity: a CAS would still consume a *second* box on
redelivery when the player holds a stack.

### 4.4 `atlas-channel`

- `socket/handler/cash_item_gachapon.go` — `CashItemGachaponHandle`: decode
  `CashItemGachaponButton`, mint a `transactionId`, produce the command. No
  validation lives here (the edge does not own the locker).
- `kafka/consumer/cashshop/consumer.go` — two new handlers:
  `handleStatusEventSurpriseOpened` announces
  `CashItemGachaponSuccessBody(...)` on the `CashItemGachaponResult` writer;
  `handleStatusEventSurpriseFailed` announces `CashItemGachaponFailedBody()`.
  Both follow `handleStatusEventPurchase`'s tenant guard +
  `IfPresentByCharacterId` shape.
- The success event carries the reward's asset id; the consumer reads the asset
  back (as `handleStatusEventPurchase` does) to build the `CashInventoryItem`
  blob, keeping one place responsible for that mapping.

### 4.5 `atlas-ui`

- `RewardPoolKind` widens to `"gachapon" | "incubator" | "cash-surprise"`.
- `KindBadge` becomes a lookup rather than a ternary (it currently has no
  default branch) — a third badge, "Cash Surprise".
- `PoolFormDialog` gains a third radio option and a third form, mirroring the
  incubator form exactly: `boxItemId` (→ pool slug, as `eggItemId` already is)
  and `name`. No `npcIds` field for this kind — FR-7.2 satisfied by the existing
  per-kind form split rather than by conditionally hiding fields.
- `PoolItemDialog` currently switches on `weighted = kind === "incubator"`;
  `cash-surprise` is also weighted, and adds a required `commodityId` field.
  New `cashSurpriseItemSchema` = weight schema + `commodityId`, with the
  resolved item shown for operator sanity-checking (FR-7.3).

---

## 5. Version coverage decisions

**Implement (6 columns): v83, v84, v87, v92, v95, jms_v185.** Serverbound and
clientbound both, per §1.

**`n-a` with proof (3 columns): v48, v61, v72.** Three independent proofs agree:
no `CUICashItemGachapon` anywhere in the binary; the standalone opcode's handler
is a 3-line "read one byte into a `CWvsContext` flag" stub that is not a
gachapon result at all; item 5222000 is absent from the ingested WZ data
(§1.6). Recorded per the `n-a` consistency gate as absence-verified-against-the-
binary, not inferred from a missing registry entry.

**v79 — recommended `n-a`, with the request path documented.** This is the one
genuine judgement call, so the evidence first:

- *For implementing:* the UI class exists, the cash-locker double-click handler
  hard-codes template **5222000** and constructs the dialog
  (`sub_4A68B0` @ `0x4a6a00`), the Open button sends `0x9F` with the 8-byte SN,
  and 5222000 is present in v79's WZ data. The request path is fully live.
- *Against:* there is **no clientbound result**. `CCashShop::OnPacket`
  (`0x471da6`) enumerates 301–309 with no gachapon case, and
  `CCashShop::OnCashItemResult` (`0x4720ed`) enumerates 47 modes with no
  gachapon arm. `CUICashItemGachapon::Update` (`0x8ef7ce`) renders the reward
  from `m_nSelectedItemID`, and nothing in the binary writes it.
  (Scope of the negative: the two cash-shop dispatch tables are the complete
  set of `CCashShop` clientbound entry points; a result arriving through some
  wholly unrelated router was not exhaustively excluded.)

Recommendation: **do not route v79.** Granting a paid item with zero client
feedback, on a path where the player can reopen the dialog and click again, is
worse than the feature being absent — and the dialog cannot be closed out by any
packet we can send. Record `n-a` for the clientbound cell with the proof above,
record `n-a` for the serverbound cell on the grounds that the request is
unanswerable, and **add the discovered `0x9F` entry to
`docs/packets/registry/gms_v79.yaml`** so the finding is not lost. This is an
evidence-based `n-a`, not a convenience one — the PRD's FR-5.5 expected v79 to be
out of scope, and it is, for a different and better-documented reason.

*Open decision for the user:* if you would rather v79 grant silently (item
lands in the locker, visible after reopening the Cash Shop), say so and the
serverbound half is routed with a documented "no client acknowledgement"
caveat. The design assumes not.

**Matrix outcome.** Every one of the ten columns ends implemented-and-verified
or `n-a`-with-proof (FR-5.6). Additionally, `CASHSHOP_CASH_GACHAPON_OPEN_RESULT`
× v83 is corrected to `n-a` (§1.5).

---

## 6. Testing

| Layer | What | How |
|---|---|---|
| Packet | Serverbound decode round-trip | Byte fixture per implemented version, `packet-audit:verify` marker |
| Packet | Clientbound SUCCESS/FAILED encode | Byte fixture per version; the mode byte comes from an `options.operations` map in the test, proving it is not hard-coded (`vega_scroll_test.go` pattern) |
| Packet | No-regression | Encode of every already-✅ cash codec unchanged — this task adds files only |
| reward-pools | `cash-surprise` rolls flat-weighted, excludes the global pool | Mirror `TestSelectRewardIncubatorKind`; add the control case proving `gachapon`/`incubator` behaviour is byte-identical after the change |
| reward-pools | Empty pool → `ErrEmptyPool` → 409; missing pool → 404 | Processor + resource tests |
| reward-pools | Weighted boundary | `selectWeightedIndex` table test (deterministic, no RNG stub) |
| cashshop | Capacity rule, both branches (qty 1 vs qty > 1) | Unit test on the helper + integration through the open path |
| cashshop | Atomicity: forced grant failure leaves the box intact | Transaction test |
| cashshop | Idempotency: same `transactionId` twice grants once | Integration test |
| cashshop | Ownership rejection: asset belonging to another account | Integration test, asserts no state change |
| channel | Command produced on decode; status events → correct writer | Handler + consumer tests |
| atlas-ui | Third kind renders its badge; form omits `npcIds`; item dialog requires `commodityId` | vitest, extending the existing `PoolFormDialog`/`PoolItemDialog` suites |

Live acceptance (PRD §10): v83 tenant first (primary target), then one v84+
tenant — but exercising the **standalone opcode** on both, since §1.4 removed the
dispatcher-arm path the PRD's second bullet assumed.

Build gates: `go test -race`, `go vet`, `docker buildx bake atlas-cashshop
atlas-channel atlas-reward-pools`, `tools/lint.sh --check`,
`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/template-opcode-order-guard.sh`,
`tools/template-duplicate-binding-guard.sh`, and `npm run build` + vitest for
atlas-ui.

---

## 7. Risks

| Risk | Mitigation |
|---|---|
| The mode byte differs on all six implemented versions; hard-coding one silently mis-dispatches on the others | Byte lives only in `options.operations`; per-version fixture test asserts the configured value reaches the wire |
| New opcodes exist in the seed templates but not in a live tenant's socket config → handler never fires | Reseed/patch live tenant socket configs as part of rollout; this exact failure is a known recurring bug |
| A `cash-surprise` pool configured with a Surprise box as a reward creates an infinite box | Not blocked in code (FR-4.5); called out in the UI copy and in the pool documentation |
| The commodity referenced by a pool entry is deleted → grants fail after the roll | Roll and commodity resolution both happen **before** the transaction; failure path leaves the box intact and logs `COMMODITY_MISSING` |
| Cash-shop purchase today has no idempotency; this path introduces a ledger only for itself | Deliberate — scoped to this task. The ledger table is generic enough to be reused if purchase is hardened later |
| v79 decision reversed late | The serverbound codec and registry entry are produced either way; only template routing and the channel handler's version reach change |

---

## 8. Open questions carried forward

1. **v79 routing** (§5) — recommendation stated, user's call.
2. **jms `0xA7` attribution** (§1.5) — this task should correct
   task-183's `arm-catalog.md` note; confirm that is in scope for this branch
   rather than a documentation-only follow-up.

PRD §9 Q1 (cash-only) stands as written. Q2 → flat weights (§4.1). Q3 → single
in-service transaction, roll first (§2.1, §4.2). Q4 → dissolved (§1.4).
Q5 → dissolved; the wire carries no error code (§2.3). Q6 → answered (§1.1,
§5). Q7 → answered (§1.6).
