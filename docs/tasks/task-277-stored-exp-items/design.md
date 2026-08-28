# Stored-EXP Items (Writ of Solomon / stored "gachapon" EXP) — Design

Phase: 2 (design). Input: `docs/tasks/task-277-stored-exp-items/prd.md` (approved).
Status: Draft for approval.

---

## 0. Executive summary — and one PRD correction

The evidence pass (§1) resolved every FR-2 unknown and **inverted the feature model the
PRD assumed**. The two ops are not "grant EXP" plus "redeem a counter fed by something
else". They are the two halves of one feature:

| Op | Client fn | What it really does |
|---|---|---|
| `USE_SOLOMON_ITEM` | `CWvsContext::SendExpUpItemUseRequest` | **Credits** the stored-EXP counter (`GW_CharacterStat::nTempEXP`) from a Writ of Solomon item (`itemId/10000 == 237`). Does *not* touch character EXP. |
| `USE_GACHA_EXP` | `CWvsContext::SendTempExpUseRequest` | **Redeems** the whole stored counter into real character EXP and zeroes it. |

This is not inference. It is the client's own UI copy, decoded out of the v95 StringPool
(§1.4):

- StringPool `0x130E`, shown by `SendExpUpItemUseRequest` when `nTempEXP != 0` and the
  request is **suppressed**: `"You'll need to use all your remaining EXP in order to use
  a new Writ of"` (truncated in the pool; the sentence continues in a sibling entry).
- StringPool `0x130F`, the confirm dialog `CUIStatusBar::TryUseTempExp` puts up before
  calling `SendTempExpUseRequest`: `"Do you wish to charge the EXP \r\nearned from The
  Writs of Solomon?"`
- StringPool `0xC97`, the rejection on both paths when the level bound fails:
  `"Your level is too high to use the selected item."`

Consequences for the PRD:

- **FR-7 is wrong as written.** Solomon use must credit `gachapon_experience`, not call
  `AwardExperience`. Awarding EXP directly would make the client's own interlock
  (`nTempEXP != 0` blocks a second Writ) permanently inert and the status-bar EXP-charge
  affordance permanently empty.
- **FR-15 / Open Question 1 are resolved, and FR-18 does not fire.** The accrual source is
  the Writ of Solomon item itself. No monster-kill diversion, no gachapon reward pool, no
  new producing service. The "gachapon experience" name in `characters` is a community
  misnomer for the client's `nTempEXP`; we keep the column name (no migration) and note the
  mapping in code.
- **FR-3 / Open Question 2 are resolved against the matrix.** gms_v72 and gms_v79 *do*
  send both ops. Their ⬜ is a registry gap, not an absence. The version spread is **eight**
  columns, not six. gms_v48 and gms_v61 are genuinely absent and get positive-absence
  evidence entries.

Everything else in the PRD survives intact; the acceptance criteria that name
`AwardExperience` for the Solomon path move to the redeem path, where they belong.

---

## 1. Evidence

All addresses are from the live IDBs listed in `idb_list`; the checked-in exports under
`docs/packets/ida-exports/` do **not** contain either function, so this pass used live IDA.

### 1.1 Request layouts (FR-1, FR-2)

`CWvsContext::SendExpUpItemUseRequest(nPOS, nItemID)` — gms_v95 @`0x9db1c0`:

```
COutPacket::COutPacket(&oPacket, 181);
COutPacket::Encode4(&oPacket, get_update_time());   // updateTime / tick
COutPacket::Encode2(&oPacket, nPOS);                // inventory slot
COutPacket::Encode4(&oPacket, nItemID);             // item id
CClientSocket::SendPacket(...);
CWvsContext::SetExclRequestSent(this, 1);
play_item_sound(nItemID, SE_ITEM_USE);
```

Body = `updateTime:int32, slot:int16, itemId:int32`. **Byte-identical to the existing
`inventory/serverbound.ItemUse`** (`libs/atlas-packet/inventory/serverbound/item_use.go`).

`CWvsContext::SendTempExpUseRequest()` — gms_v95 @`0x9db430`:

```
COutPacket::COutPacket(&oPacket, 182);
COutPacket::Encode4(&oPacket, get_update_time());   // updateTime / tick
CClientSocket::SendPacket(...);
CWvsContext::SetExclRequestSent(this, 1);
```

Body = `updateTime:int32`. Nothing else.

The layout is invariant on every version that carries the op (checked on v72 @`0x90cb20` /
`0x90cd28`, v79 @`0x95dee8` / `0x95e0f0`, v83 @`0xa12685` / `0xa1288f`, v84 @`0xa5cac2` /
`0xa5cccc`, v87 @`0xaa80ac` / `0xaa82b6`, v92 @`0x9b0ab0` / `0x9b0d20`, jms185 @`0xaf883d` /
`0xaf8a40`). The jms_v185 body is the same `Encode4 / Encode2 / Encode4` — Open Question 6
resolves to "no payload divergence", only an opcode shift.

**No version gate is needed on either codec.** There is no `MajorAtLeast` branch to write;
the divergence is entirely in the opcode, which the template already owns.

### 1.2 Opcodes and the v72 / v79 resolution (FR-3)

Opcode read directly off the `COutPacket` ctor argument at each send site:

| Column | `USE_SOLOMON_ITEM` | `USE_GACHA_EXP` | Matrix today | Action |
|---|---|---|---|---|
| gms_v48 | absent | absent | ⬜ | keep ⬜, add na-evidence |
| gms_v61 | absent | absent | ⬜ | keep ⬜, add na-evidence |
| gms_v72 | **0x9C (156)** | **0x9D (157)** | ⬜ | **add registry rows, implement, verify** |
| gms_v79 | **0x9B (155)** | **0x9C (156)** | ⬜ | **add registry rows, implement, verify** |
| gms_v83 | 0x9D (157) | 0x9E (158) | ❌ | implement, verify |
| gms_v84 | 0xA1 (161) | 0xA2 (162) | ❌ | implement, verify |
| gms_v87 | 0xA5 (165) | 0xA6 (166) | ❌ | implement, verify |
| gms_v92 | 0xB2 (178) | 0xB3 (179) | ❌ | implement, verify |
| gms_v95 | 0xB5 (181) | 0xB6 (182) | ❌ | implement, verify |
| jms_v185 | 0x71 (113) | 0x72 (114) | ❌ | implement, verify |

The v83/v84/v87/v92/v95/jms185 opcodes match the existing registry rows exactly — no
registry correction there.

**Positive-absence evidence for v48 / v61.** Both IDBs carry full MSVC-mangled symbols
(`?SendPacket@CClientSocket@@QAEXABVCOutPacket@@@Z` resolves in both, as does
`?GetItemInfo@CItemInfo@@...`), and in both, `?SendExpUpItemUseRequest@CWvsContext@@QAEXJJ@Z`,
`?SendTempExpUseRequest@CWvsContext@@QAEXXZ` **and** `?GetMaxLEV@CItemInfo@@QAEJJ@Z` are all
absent. The whole stored-EXP subsystem — sender, redeemer, and the WZ accessor the sender
gates on — does not exist in those builds. That is the na-evidence text. gms_v12 stays out
of scope (no matrix column, cash-item-use family unregistered there).

### 1.3 Client-side gating (FR-2, FR-5)

`SendExpUpItemUseRequest` refuses to send, in order:

1. `nJob / 10000 == 190` → StringPool `0xF11` (the tamed-mob / riding job guard shared by
   many senders).
2. `CWvsContext::IsAbleToConsume(nItemID, 0)` must pass.
3. `characterLevel > CItemInfo::GetMaxLEV(nItemID)` → StringPool `0xC97` ("Your level is
   too high to use the selected item."). **Upper bound only — there is no lower bound.**
4. `GW_CharacterStat::nTempEXP != 0` → StringPool `0x130E` (must spend the banked balance
   before charging another Writ).
5. `IsAttract`, a modeless-dialog guard, and `CanSendExclRequest(500, 0)`.

`CUIStatusBar::TryUseTempExp` (v95 @`0x870aa0`, the sole caller of
`SendTempExpUseRequest`) gates on:

1. the click landing in the EXP-bar rect `{28,18,336,31}`;
2. `characterLevel <= 0x32` (**50**);
3. `nTempEXP > 0`;
4. a `CUtilDlg::YesNo` confirm (StringPool `0x130F`).

`SendTempExpUseRequest` itself re-checks the job guard and `characterLevel <= 50`, emitting
StringPool `0xC97` above 50.

### 1.4 Item identity and WZ fields (FR-2, Open Question 3)

`CDraggableItem::OnDoubleClicked` (v95 @`0x506e10`) reaches the Solomon sender at
`0x5078d9` only through `is_exp_up_item(nItemID)` @`0x5078be`, which is exactly:

```c
BOOL __cdecl is_exp_up_item(int nItemID) { return nItemID / 10000 == 237; }
```

So the family is **classification 237** — `2370000`–`2379999`.

`CItemInfo::GetMaxLEV` (v95 @`0x5acb70`) reads one property off the item's `info` node,
named by StringPool index `0x788`. Decoding that entry (`StringPool::ms_aString` @`0xc5a878`,
16-byte key `?ms_aKey@StringPool@@` @`0xb98830`, per-entry seed byte, `rotatel` by
`((seed>>3)%16)*8 + (seed&7)` bits, then `out = enc ^ key[i%16]` with the `enc == key`
identity case) yields **`maxLevel`**. Neighbouring entries decode to `reqLevel`,
`reqMobLevel`, `reqMap`, `Etc/QuestInfo.img`, confirming the decoder.

The **amount** banked per Writ is not read by the client at all —
`is_exp_up_item` has exactly one xref, and no tooltip path reads it — so the amount is
server-authoritative and lives in WZ. This repo already enumerated it from v83
`Item.wz/Consume/*.img.xml`: `docs/research/missing-features/items-and-consumables.md` §5
records spec field **`exp`**, 13 occurrences, item ids **`2370000`–`2370012`**, described as
"EXP potions … no effect". That is the Writ family and its amount field.

So the two WZ reads this task needs are:

- `info/maxLevel` (int) — the upper level bound the client checks.
- `spec/exp` (int) — the EXP amount the Writ banks.

Neither is parsed by `services/atlas-data/atlas.com/data/consumable/reader.go` today.

### 1.5 Response packets (Open Question 4)

There is **no dedicated clientbound response**. The only client-visible effects are the
`STAT_CHANGED` stat updates and the excl-request unlock. Both senders call
`CWvsContext::SetExclRequestSent(1)`, so every terminal outcome — success *and* rejection —
must reach the client with `bExclRequestSent` set, or the character is soft-locked out of
further exclusive requests. Atlas already owns that seam
(`services/atlas-channel/atlas.com/channel/session/enable_actions.go`).

`stat.TypeGachaponExperience` is already a 4-byte stat in
`libs/atlas-packet/stat/clientbound/changed.go:86,137`, and `GACHAPON_EXPERIENCE` is already
present in the stat-index table of every seed template (verified by grep on gms_72/79/83/95
and jms_185). Nothing on the encode side needs to change.

---

## 2. Architecture

### 2.1 Data flow — Solomon use (credit)

```
client  --USE_SOLOMON_ITEM(tick, slot, itemId)-->  atlas-channel
   CharacterItemUseSolomonHandleFunc
     decode inventory/serverbound.ItemUse
     consumable.Processor.RequestItemConsume(field, charId, itemId, slot, 1, tick)
                                   |
                            atlas-consumables
     Step 0  classification gate: item2.GetClassification(itemId) == 237
     Step 1  reserve slot/itemId/qty=1 in the USE compartment (existing saga)
     Step 2  on reservation success -> ConsumeSolomon ItemConsumer:
               read consumable data (spec/exp, info/maxLevel)
               validate level <= maxLevel  and  stored balance == 0
               if invalid -> cancel reservation, EnableActions, log, stop
               if valid   -> commit CONSUME  +  emit CREDIT_STORED_EXPERIENCE
                                   |
                             atlas-character
     credit: gachapon_experience = min(cur + amount, MaxUint32)   [one tx]
     emit STAT_CHANGED{GACHAPON_EXPERIENCE}
                                   |
                             atlas-channel fan-out (existing) -> STAT_CHANGED packet
```

### 2.2 Data flow — stored-EXP redeem

```
client  --USE_GACHA_EXP(tick)-->  atlas-channel
   CharacterUseStoredExperienceHandleFunc
     decode character/serverbound.StoredExperienceUse
     character.Processor.RedeemStoredExperience(field, charId)
                                   |
                             atlas-character
     ONE transaction:
       read gachapon_experience
       if 0            -> no write, no event, unlock, return   (FR-11)
       if level > 50   -> no write, no event, unlock, return   (client parity)
       AwardExperience(mb)(tx, ITEM distribution, amount)      -- same tx
       SetGachaponExperience(0)                                -- same tx
     emit EXPERIENCE_CHANGED + STAT_CHANGED{EXPERIENCE, GACHAPON_EXPERIENCE}
```

### 2.3 Ownership decisions

| Concern | Owner | Why |
|---|---|---|
| Wire decode | `libs/atlas-packet` | invariant; no version gate needed |
| Session/handler dispatch, excl unlock | `atlas-channel` | every other item-use op is wired the same way |
| Item ownership, reservation, consumption | `atlas-consumables` | resolves Open Question 5 — the reserve/consume saga already gives FR-8's exactly-once and the ownership check for free; the dedicated opcode is a *routing* difference, not an ownership one |
| Eligibility (`maxLevel`, balance interlock) | `atlas-consumables` | it is the service that already holds both the consumable's WZ data and the character read |
| Counter credit / redeem, EXP award, level cap | `atlas-character` | sole writer of `characters`; FR-14 falls out because we reuse `AwardExperience` verbatim |
| WZ field exposure | `atlas-data` | `spec/exp`, `info/maxLevel` |

---

## 3. Alternatives considered

### A. Where does the Solomon item get consumed?

1. **Through `atlas-consumables`' existing reserve/consume saga (chosen).** A new
   `ConsumeSolomon` `ItemConsumer` branch in `RequestItemConsume`, alongside
   `ConsumeMorphCoupon`, `ConsumeSummoningSack`, `ConsumeMonsterCard`. Gets ownership
   validation, the 30s reservation, and single-commit semantics with no new machinery, and
   keeps the failure path (cancel reservation) identical to every sibling.
2. Channel calls `atlas-inventory` directly and then produces the credit command. Rejected:
   duplicates the reservation protocol, and a crash between destroy and credit silently eats
   the Writ. Violates FR-8 by construction.
3. New `atlas-experience` service. Rejected as unwarranted scope for two ops.

### B. Where does the level/balance eligibility check live?

1. **In `atlas-consumables`, inside the consumer, after reservation and before commit
   (chosen).** The reservation is cancellable, so a rejection leaves the item untouched
   (FR-6), and the check sits next to the WZ read that supplies `maxLevel`.
2. In `atlas-channel` before dispatch. Rejected: channel would need the consumable data
   resource and a character read it does not otherwise take, and a check there races the
   reservation.
3. In `atlas-character` at credit time. Rejected: the item is already destroyed by then.

### C. Redeem transactionality (FR-10, FR-13)

1. **Compose inside one `database.ExecuteTransaction`, calling
   `p.WithTransaction(tx).AwardExperience(buf)(...)` and the counter zero on the same `tx`
   (chosen).** Note the trap: `ProcessorImpl.AwardExperience` today opens its *own*
   `ExecuteTransaction` internally (`character/processor.go:753`). The redeem path must use
   the `WithTransaction` form and, if the inner call still nests its own transaction, the
   nesting must be verified to join the outer one rather than commit independently — this is
   an explicit implementation checkpoint, not an assumption.
2. Two commands (award, then zero). Rejected: a crash between them double-grants on retry.
3. Optimistic-concurrency column. Rejected: the single-writer transaction plus the
   read-inside-tx already serialises; a version column is new schema for no gain.

### D. Codec placement

`USE_SOLOMON_ITEM`'s body is byte-identical to `ItemUse`. The repo has settled precedent for
exactly this (`ReturnScrollItemUse`, `SummonBagItemUse`): **one audit-only wrapper per op**,
so each op gets its own packet id, evidence key and audit report rather than a manufactured
✅ pinned to the potion sender's decompile. We follow it. `USE_GACHA_EXP` is a genuinely new
one-field body and gets a real struct.

---

## 4. Components

### 4.1 `libs/atlas-packet`

`inventory/serverbound/solomon_item_use.go` — audit-only wrapper, mirroring
`return_scroll_item_use.go`:

```go
// packet-audit:fname CWvsContext::SendExpUpItemUseRequest
type SolomonItemUse struct{ ItemUse }
func NewSolomonItemUse() SolomonItemUse {
    return SolomonItemUse{ItemUse: NewItemUse(CharacterItemUseSolomonHandle)}
}
```

with `CharacterItemUseSolomonHandle = "CharacterItemUseSolomonHandle"` added to the const
block in `item_use.go`. The handler decodes the shared `ItemUse`, exactly as the town-scroll
and summon-bag handlers do.

`character/serverbound/stored_experience_use.go` — new immutable struct with both `Encode`
and `Decode`:

```go
// packet-audit:fname CWvsContext::SendTempExpUseRequest
type StoredExperienceUse struct{ updateTime uint32 }
```

`Operation() == CharacterUseStoredExperienceHandle`. Byte-fixture tests with
`packet-audit:verify` markers for all eight columns on both ops.

### 4.2 `libs/atlas-constants`

`item/constants.go`: `ClassificationConsumableExpUpItem = Classification(237)`, documented
with the `is_exp_up_item` derivation and the v95 address. Checked first per repo convention —
237 is currently unnamed.

### 4.3 `services/atlas-data`

`consumable/reader.go`: `m.MaxLevel = uint32(i.GetIntegerWithDefault("maxLevel", 0))` and
`m.Spec[SpecTypeExperience] = s.GetIntegerWithDefault("exp", 0)` with
`SpecTypeExperience = SpecType("exp")` in `rest.go`; matching model accessor and REST field.
Scope is held to these two fields — the broader unparsed-`spec` gap stays a separate task per
the PRD's non-goals.

**Ingest-order caveat (same class as task-219's morph coupon):** tenants whose Item.wz was
ingested before this change will not carry `exp` / `maxLevel` until re-ingested. An
operational re-ingest follow-up goes in `docs/TODO.md`.

### 4.4 `services/atlas-consumables`

- `RequestItemConsume`: new branch
  `item2.GetClassification(itemId) == item2.ClassificationConsumableExpUpItem` →
  `ConsumeSolomon(transactionId, f, characterId, slot, itemId)`. Placed **before** the
  reward-table fallback, for the same reason `routesToMorphCoupon` is.
- `ConsumeSolomon` consumer: reads the consumable data and the character; rejects
  (cancel reservation + `EnableActions` + structured log naming character id, item id and the
  failing rule) when `spec/exp <= 0`, when `level > maxLevel` (and `maxLevel > 0`), or when
  the character's stored balance is non-zero; otherwise commits the CONSUME and emits the
  credit command.
- `character.Processor`: add `CreditStoredExperience(f field.Model, characterId uint32,
  amount uint32) error`, producing on `EnvCommandTopic` exactly like `ChangeHP`/`ChangeMP`.
- `character.Model` gains `gachaponExperience` (the REST resource already serves it) so the
  interlock can be evaluated without a second call.

### 4.5 `services/atlas-character`

New commands on the existing character command topic:

- `CommandCreditStoredExperience = "CREDIT_STORED_EXPERIENCE"`, body
  `{channelId, amount uint32, reason string}`. Clamps at `math.MaxUint32` (FR-17); emits
  `STAT_CHANGED` with `stat.TypeGachaponExperience` and values key `gachapon_experience`
  (the key the channel snapshot registry already maps, `snapshot/registry.go:264`).
- `CommandRedeemStoredExperience = "REDEEM_STORED_EXPERIENCE"`, body `{channelId}`. One
  transaction: read balance → zero it → `AwardExperience` with a single
  `ExperienceDistributionTypeItem` distribution → emit `EXPERIENCE_CHANGED` and
  `STAT_CHANGED{EXPERIENCE, GACHAPON_EXPERIENCE}`. Zero balance and level > 50 are silent
  no-ops (FR-11).

Processor: `CreditStoredExperience` / `RedeemStoredExperience` in the established
`...AndEmit` + `(mb *message.Buffer)` pair shape. `SetGachaponExperience` already exists on
the builder; add the `dynamicUpdate` setter if absent.

### 4.6 `services/atlas-channel`

- `socket/handler/character_item_use.go`: `CharacterItemUseSolomonHandleFunc`, decoding
  `NewItemUse(CharacterItemUseSolomonHandle)` and calling `RequestItemConsume` — one line
  different from the town-scroll handler.
- `socket/handler/character_stored_experience_use.go`: new handler decoding
  `StoredExperienceUse` and producing `REDEEM_STORED_EXPERIENCE`.
- `main.go`: both `handlerMap` registrations.

### 4.7 `services/atlas-configurations`

Both handler names added to the eight in-scope templates: `template_gms_72_1.json`,
`gms_79`, `gms_83`, `gms_84`, `gms_87`, `gms_92`, `gms_95`, `jms_185_1`. **Not** gms_12,
gms_48, gms_61.

### 4.8 `docs/packets`

- Registry rows for gms_v72 (`156`/`157`) and gms_v79 (`155`/`156`) with
  `provenance: ida-verified` and the send-site addresses.
- `feature-na-evidence.yaml` entries for `USE_SOLOMON_ITEM` × gms_v48, `USE_GACHA_EXP` ×
  gms_v48, and the same pair on gms_v61, carrying the §1.2 positive-absence argument.
- Evidence records + regenerated `STATUS.md` / `status.json`; `packet-audit matrix`,
  `fname-doc --check`, `operations --check` all exit 0.

---

## 5. Error handling

| Case | Behaviour |
|---|---|
| Slot/item mismatch, item not owned | reservation fails; nothing consumed; `EnableActions`; logged |
| `level > maxLevel` | reservation cancelled; item intact; `EnableActions`; logged with the rule name |
| Stored balance already non-zero | reservation cancelled; item intact; `EnableActions`; logged (mirrors StringPool `0x130E`) |
| `spec/exp` missing or ≤ 0 (un-reingested tenant) | reservation cancelled; item intact; logged at Warn naming the item id — never destroy a Writ for zero EXP |
| Redeem with zero balance | no write, no event, `EnableActions`, no disconnect |
| Redeem above level 50 | no write, no event, `EnableActions`, logged |
| Duplicate/replayed redeem | second one reads 0 inside the transaction and is a no-op |
| Unknown character | logged, no write |

No new clientbound error packet is introduced — §1.5 established none exists.

---

## 6. Testing

Unit / integration (all asserted by test, per the PRD's acceptance list):

- Byte fixtures per op × version (8 columns × 2 ops) with `packet-audit:verify` markers,
  round-tripping `Encode`/`Decode`.
- `ConsumeSolomon`: eligible → CONSUME committed once + credit command with `spec/exp`;
  over-`maxLevel` → reservation cancelled, no credit; non-zero balance → cancelled, no
  credit; missing `spec/exp` → cancelled, no credit.
- Credit: clamps at `MaxUint32` instead of wrapping; emits `STAT_CHANGED` carrying
  `GACHAPON_EXPERIENCE`.
- Redeem: non-zero balance grants exactly the balance and leaves the column at 0 **in the
  same transaction** (asserted by injecting a failure after the award and observing the
  balance un-zeroed); zero balance is a total no-op; level > 50 is a no-op; two concurrent
  redeems grant once.
- `STAT_CHANGED` from redeem contains both `EXPERIENCE` and `GACHAPON_EXPERIENCE`.
- Template grep: both handlers present in all eight in-scope templates and absent from the
  other three.

Gates: flagless `tools/verify.sh` exits 0; `packet-audit matrix` / `fname-doc --check` /
`operations --check` exit 0; code review before PR.

---

## 7. Risks

1. **Nested transaction in `AwardExperience`.** The single-transaction redeem hinges on
   `WithTransaction(tx)` genuinely joining the outer transaction. Verify before building on
   it; if it does not, hoist the EXP arithmetic into the redeem transaction rather than
   splitting the write.
2. **Un-reingested tenants.** `spec/exp` absent ⇒ every Writ is rejected rather than eaten.
   That is the safe failure, but it is a visible "nothing happens" until re-ingest. Recorded
   in `docs/TODO.md`.
3. **`maxLevel` on the 237 family is unconfirmed per item.** The client reads `info/maxLevel`
   (§1.4) but the repo's WZ enumeration (§5 of the research doc) only recorded the `spec`
   side. Treat a missing/zero `maxLevel` as "no upper bound" rather than as "reject", and
   confirm the field's presence on `2370000`–`2370012` during implementation against tenant
   WZ.
4. **Naming.** `gachapon_experience` stays as the column/stat name (no migration, no churn
   across atlas-login / atlas-cashshop / atlas-npc-shops), while the new Kafka commands and
   Go symbols use "stored experience". The mismatch is deliberate and gets a comment at the
   column and at each new command.

---

## 8. Open items for the user

- **PRD FR-7 changes meaning** (§0): Solomon use credits the counter instead of awarding
  EXP. This design implements the evidence-backed behaviour. Say the word if you want the
  PRD amended before planning.
- Everything else the PRD flagged as unknown is now resolved from client evidence; FR-18's
  escalation condition did not occur.
