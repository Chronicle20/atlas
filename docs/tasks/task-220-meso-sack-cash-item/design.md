# Meso Sack Cash Item — Design

Version: v1
Status: Approved
Created: 2026-08-13
PRD: [`prd.md`](./prd.md)

---

## 1. Summary

Meso sacks (classification `520`, `Item.wz/Cash/0520.img`) are the last unimplemented
common cash-slot type in `CharacterCashItemUseHandleFunc`. This design adds the
type-19 arm end to end:

1. `atlas-data` parses `info/meso` into a first-class `Meso` field on the cash
   document.
2. `atlas-channel` grows a `CashSlotItemTypeCurrencySack` branch that resolves the
   amount server-side and creates a two-step `meso_sack_use` saga
   (`DestroyAsset` → `AwardMesos`).
3. `atlas-character` emits a `MESO_OVERFLOW` error status event on the previously
   silent `RequestChangeMeso` overflow path, so the award step fails fast instead
   of stranding the saga until timeout.
4. `atlas-saga-orchestrator` gains a `meso_sack_use` reverse-walk compensator that
   refunds the sack **and emits the saga-failed event with the real character id**
   — the existing generic emitter would emit `characterId: 0` and the player would
   never be told anything (§5.2, the single largest correction to the PRD).

The wire is unchanged: FR-6.1's IDA pass (§3) confirms across all ten versions that
the classification-520 arm of `CWvsContext::SendConsumeCashItemUseRequest` encodes
**no sub-body**. No codec, no template, no `libs/atlas-packet` change.

---

## 2. Architecture

```
client                atlas-channel                atlas-data      saga-orchestrator      atlas-character
  │  CASH_ITEM_USE
  │   (itemId, slot)
  ├──────────────────────▶
  │              slot/template guard
  │              GetCashSlotItemType → 19
  │                      ├── GET /cash/items/{id} ─▶│
  │                      │◀──── { "meso": N } ─────┤
  │              N == 0 / err ─▶ reject + EnableActions ──┐
  │                      │                                │ (terminal)
  │              POST /sagas  meso_sack_use               │
  │                      ├──────────────────────────────▶ │
  │                      │        consume_meso_sack (DestroyAsset)  ─▶ destroy 1×itemId
  │                      │        award_mesos    (AwardMesos)       ─▶ RequestChangeMeso
  │                      │                                             ├─ ok  → MESO_CHANGED + STAT_CHANGED
  │◀─ meso chat line ────┤◀───────────────────────────────────────────┘         (ExclRequestSent=true)
  │◀─ STAT_CHANGED ──────┤        (unlocks the client — §5.1)
  │                      │
  │                      │        overflow → ERROR{MESO_OVERFLOW} ─▶ step failed
  │                      │        compensateMesoSackUse: CreateItem(sack) + SAGA FAILED{characterId, MESO_OVERFLOW}
  │◀─ pink text ─────────┤
  │◀─ EnableActions ─────┤
```

Three properties fall out of this shape and are worth stating explicitly:

- **Consume-first.** Every sibling cash-item-use arm (`item_tag_use`,
  `sealing_lock_use`, `incubator_use`, `point_reset`, `note_send`) destroys before
  it grants. Award-first would pay a player who then fails to lose the sack.
- **The success unlock is free.** `atlas-character`'s `statChangedProvider` hard-codes
  `ExclRequestSent: true`
  (`services/atlas-character/atlas.com/character/character/producer.go:238`), and
  the meso credit already emits a `STAT_CHANGED{TypeMeso}`. The client's
  exclusive-request gate is released by the packet that renders the new balance —
  correctly ordered by construction. See §5.1.
- **The failure unlock is not free.** It requires the compensator fix in §5.2.

---

## 3. FR-6.1 — Wire verification (all ten versions)

Method: locate `CWvsContext::SendConsumeCashItemUseRequest` per IDB, decompile the
client's own `get_cashslot_item_type` to learn which **case number** classification
520 maps to *on that build*, walk that case's arm to its terminal `jmp`, and confirm
the arm reaches the shared `CanSendExclRequest` → `SendPacket` tail without executing
a single `COutPacket::Encode*` of its own.

The case number is **not** constant across versions — this is the finding that makes
the per-version walk necessary rather than ceremonial:

| Tenant | IDB | send fn | 520 → client case | case-arm entry | sub-body | `update_time` | send tail |
|---|---|---|---|---|---|---|---|
| gms_v48 | `GMS_v48_1_DEVM.exe` | `0x70e495` | **17** | `0x70f53c` | none | trailing `Encode4` @`0x711d9f` | `0x711d60` |
| gms_v61 | `GMS_v61.1_U_DEVM.exe` | `0x832a5d` | **18** | `0x833e44` | none | trailing `Encode4` @`0x83672e` | `0x833519` |
| gms_v72 | `GMS_v72.1_U_DEVM.exe` | `0x904fe2` | 19 | `0x90679c` | none | trailing `Encode4` @`0x909123` | `0x905294` |
| gms_v79 | `GMS_v79_1_DEVM.exe` | `0x95634a` | 19 | `0x957bc4` | none | trailing `Encode4` @`0x95a54e` | `0x9565fc` |
| gms_v83 | `MapleStory_dump.exe` | `0xa0a63f` | 19 | `0xa0bfd8` | none | trailing `Encode4` @`0xa0ea5c` | `0xa0a8f1` |
| gms_v84 | `GMS_v84.1_U_DEVM` | `0xa54a2f` | 19 | `0xa563a3` | none | trailing `Encode4` @`0xa58e50` | `0xa54ce8` |
| gms_v87 | `GMSv87_4GB.exe` | `0xa9fef9` | 19 | `0xaa1917` | none | leading (header) | `0xaa01c0` → send `0xaa43ac` |
| gms_v92 | `GMS_v92_1_DEVM.exe` | `0x9bfe10` | 19 | `0x9c1de7` | none | leading (header) | `0x9bff51` → send `0x9c4dab` |
| gms_v95 | `GMS_v95.0_U_DEVM.exe` | `0x9eb3e0` | 19 | `0x9ed2dc` | none | leading (header) | `0x9f063c` → send `0x9f066b` |
| jms_v185 | `MapleStory_dump_SCY.exe` | `0xaef2f5` | 19 | `0xaf0735` | none | leading (header) | `0xaf2ad2` → send `0xaf2afc` |

Client type tables read from: v48 `0x477471`, v61 `0x48a522`, v72 `0x49fb33`,
v79 `0x47ec3e`, v95 `get_cashslot_item_type@0x488c70`, jms_v185
`get_cashslot_item_type@0x49a1ee`.

**Conclusion: FR-6.1 is satisfied with no codec work.** The `update_time` split
(trailing ≤ v84, leading ≥ v87 and JMS) exactly matches the existing
`cashsb.UpdateTimeFirst(t)` gate, which the common `cashsb.ItemUse` header already
consumes. The arm reads zero additional bytes. `libs/atlas-packet` is untouched and
`FR-6.2` holds trivially.

### 3.1 Three corroborating findings from the same pass

**(a) Atlas's unconditional `19` is correct — because the type never rides the wire.**
`GetCashSlotItemType` (`character_cash_item_use.go:894`) returns `19` for
`ClassificationCurrencySack` on every version, which *disagrees* with the v48 (17)
and v61 (18) clients. This is harmless and must not be "fixed": Atlas derives the
type from the server-resolved template id, so `19` is simply Atlas's internal name
for "classification 520". Only two things could break it, and neither does — no
other classification maps to `19` in Atlas's table (verified: `19` is returned from
exactly one site, line 894), and the arm decodes no sub-body, so there is nothing to
mis-align. **Do not add a version gate here.**

**(b) The client itself reads `info/meso`.** On v83 the arm fetches the item's WZ
property and formats string-pool entry `0x31D` into a `CUtilDlg::YesNo` confirmation
(`0xa0c0b7`–`0xa0c15b`); v61 does the same with `0x309` (`0x833f34`–`0x833fe4`), v48
with the `0x70f5fe` fetch. The server paying `info/meso` therefore matches exactly
what the player was shown before they clicked Yes. This is direct evidence for
FR-1.1 choosing `info/meso` over any other node.

**(c) v92/v95 recognise the random-payout family, and we deliberately diverge.**
`sub_9A1AB0` on v92 is literally `get_cashslot_item_type(id) == 19 && id/1000 == 5202`
(`0x9a1ab0`); when true the client shows a *different*, amount-less confirmation
string (`0x17F7`) instead of the amount-bearing one (`0x32E`). So on v92/v95 the
client tells the player "you will receive a random amount" while this task pays the
flat `info/meso` value. That is the PRD's accepted non-goal, now with a wire-level
citation: the deviation is cosmetic (dialog text), never structural (both branches
converge on the same zero-byte send at `0x9c1fd7`).

**(d) JMS routes the Maple Point sacks away from 19 entirely.** JMS's
`get_cashslot_item_type` special-cases `5200009`/`5200010` → type **49**, everything
else in 520 → 19 (`0x49a1ee`, `case 520`). GMS v95 does not. Atlas maps both to `19`
regardless, so on JMS a Maple Point sack would enter our branch and be rejected by
FR-2.4's zero-amount guard rather than by type. Same observable outcome (nothing
consumed, nothing awarded, client unlocked); no code change needed, but the
divergence is recorded here so a future reader does not "correct" the table.

---

## 4. Component design

### 4.1 `atlas-data` — parse `info/meso` (FR-1)

`services/atlas-data/atlas.com/data/cash/reader.go`, alongside the existing
`slotMax`/`protectTime` reads:

```go
m.Meso = uint32(i.GetIntegerWithDefault("meso", 0))
```

`services/atlas-data/atlas.com/data/cash/rest.go`:

```go
Meso uint32 `json:"meso,omitempty"`
```

A first-class field, **not** a `Spec` entry: `Spec` is the effect map consumed by the
consumable pipeline, and an award amount is not an effect. `omitempty` matches
`protectTime`/`stateChangeItem`.

Verified against local WZ (`GMS/83.1/Item.wz/Cash/0520.img.xml`): `05200000` →
`<int name="meso" value="1000000"/>`, `05200001` → `5000000`, `05200002` →
`10000000`. Note the node set is *only* `icon`/`iconRaw`/`meso`/`cash` — there is no
`slotMax`, no `spec`, no `tradeBlock`. The reader's `GetIntegerWithDefault(..., 0)`
already tolerates that; FR-1.2's "absent ⇒ zero" is the default behaviour, not new
code.

**Rollout (FR-1.4).** Cash items live in `document.Storage[string, RestModel]` under
kind `CASH`. The field is additive to the JSONB document, so **no existing document
gains it on deploy** — every tenant version must be re-ingested before the feature
functions. Until then FR-2.4 makes the branch inert (amount `0` ⇒ reject, nothing
consumed), which is the correct failure mode for a half-rolled-out deploy. The plan
must carry a per-tenant re-ingest + verification step.

### 4.2 `atlas-channel` — cash view model (FR-3)

`services/atlas-channel/atlas.com/channel/data/cash/rest.go` gains
`Meso uint32 \`json:"meso"\`` so `cashData.NewProcessor(l, ctx).GetById(...)` returns
it. This mirror is already partial (it carries only the four fields the channel
uses); adding one more field is the established pattern.

### 4.3 `atlas-channel` — the type-19 branch (FR-2, FR-4, FR-5)

New constant beside the others:

```go
CashSlotItemTypeCurrencySack = CashSlotItemType(19)
```

New file `socket/handler/character_cash_item_use_meso_sack.go` holding
`handleMesoSackUse`, dispatched from `CharacterCashItemUseHandleFunc` as:

```go
if it == CashSlotItemTypeCurrencySack {
    handleMesoSackUse(l, ctx, wp)(s, itemId)
    return
}
```

No sub-body decode (§3), so the arm takes no `*request.Reader`. Placement: with the
other `it == …` arms, before the classification-first megaphone block — type 19 does
not collide with any megaphone alias, so ordering is unconstrained; keeping it with
its peers is the readable choice.

`handleMesoSackUse` in full shape:

```go
enableActions := func() {                       // FR-5.1 — identical to the
    _ = session.Announce(l)(ctx)(wp)(           // vega-scroll / point-reset /
        statpkt.StatChangedWriter)(             // incubator rejection idiom
        statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
}

cd, err := cashData.NewProcessor(l, ctx).GetById(uint32(itemId))
if err != nil { warn; enableActions(); return }          // FR-2.3 / FR-2.5

if cd.Meso == 0 || cd.Meso > math.MaxInt32 {             // FR-2.4
    warn(characterId, itemId, reason); enableActions(); return
}
```

The `math.MaxInt32` half of that guard is new relative to the PRD and is not
optional: `AwardMesosPayload.Amount` is `int32`
(`libs/atlas-saga/payloads.go:75`) while the WZ value is `uint32`. A sack above
2^31−1 would silently wrap to a negative award — i.e. *take* mesos. Fail closed.

Then the saga:

```go
saga.Saga{
    TransactionId: uuid.New(),
    SagaType:      saga.MesoSackUse,
    InitiatedBy:   "CASH_ITEM_USE",
    Steps: []saga.Step{
        { StepId: "consume_meso_sack", Action: saga.DestroyAsset,
          Payload: saga.DestroyAssetPayload{
              CharacterId: s.CharacterId(), TemplateId: uint32(itemId),
              Quantity: 1, RemoveAll: false } },
        { StepId: "award_mesos", Action: saga.AwardMesos,
          Payload: saga.AwardMesosPayload{
              CharacterId: s.CharacterId(),
              WorldId: f.WorldId(), ChannelId: f.ChannelId(),
              ActorId: uint32(itemId), ActorType: "ITEM",
              Amount: int32(cd.Meso), ShowEffect: true } },
    },
}
```

**Open question 2 resolved — `DestroyAsset`, not `DestroyAssetFromSlot`.** Three
reasons, in order of weight:

1. The orchestrator's cash-item-use rollback family inverts `DestroyAsset` →
   `RequestCreateItem` (`compensator.go:1436`); `DestroyAssetFromSlot` is inverted
   only inside `DispatchCashItemUseRollbacks`, which we are not reusing (§5.2 adds a
   dedicated compensator). Using the template-keyed action keeps the inverse trivial.
2. Every sibling arm consumes *the cash item itself* by template
   (`item_tag_use`, `sealing_lock_use`, `field_effect_use`, `point_reset`,
   `note_send`); `DestroyAssetFromSlot` is used only for the *target* of an
   operation (the incubated egg). A meso sack has no target.
3. The slot is already proven: the pre-branch guard at
   `character_cash_item_use.go:52-57` resolved the CASH-compartment slot and
   asserted its template equals the claimed `itemId` (FR-2.6). Re-passing the slot
   buys nothing and would make the compensator's "restore to the same slot"
   question live for no benefit. A refund landing in the first free CASH slot is
   correct and matches every other refund path in the system.

`ActorType: "ITEM"` is a new value in a free-form string field (existing values:
`"NPC"`, `"STORAGE"`, `"CHARACTER"`); nothing validates it, and it is the honest
description of the actor.

### 4.4 `libs/atlas-saga` + re-exports (FR-4.4)

```go
MesoSackUse Type = "meso_sack_use"
```

in `libs/atlas-saga/model.go`, re-exported as `saga.MesoSackUse` from
`services/atlas-channel/.../saga/model.go` and
`services/atlas-saga-orchestrator/.../saga/model.go`, plus
`SagaTypeMesoSackUse = "meso_sack_use"` in the channel's
`kafka/message/saga/kafka.go` (the consumer compares against that copy, not the
`Type`). Saga types are free strings — `unmarshal.go` and `validation.go` have no
per-type registry, so nothing else needs touching.

### 4.5 `atlas-character` — `MESO_OVERFLOW` (FR-7)

`processor.go:838-841` currently returns `ErrMesoOverflow` with no emission. Mirror
the `ErrNotEnoughMeso` path immediately above it — set a `rejectEmit` closure inside
the transaction, fire it after rollback:

```go
StatusEventErrorTypeMesoOverflow = "MESO_OVERFLOW"   // kafka.go, beside NOT_ENOUGH_MESO

// producer.go
func mesoOverflowErrorStatusEventProvider(transactionId, characterId, worldId, amount) …
    // StatusEvent[StatusEventMesoErrorBody]{ Type: ERROR, Body: {Error: MESO_OVERFLOW, Amount: amount} }
```

Reusing `StatusEventMesoErrorBody` verbatim is what makes FR-7.2 hold: the
orchestrator's `handleCharacterMesoErrorEvent` decodes that exact body and is gated
by the acceptance table, not by the error string.

`RequestChangeMeso` keeps returning `ErrMesoOverflow` after emitting — the emission
is additive. Semantics unchanged: reject, never clamp (FR-7.3).

**One deliberate asymmetry.** The `ErrNotEnoughMeso` path swallows the error
(`return nil`) after emitting, because the emission *is* the response. The overflow
path must keep returning the error so the REST/command handler still logs a failure;
the saga is driven by the event either way. Do not "harmonise" these.

### 4.6 `atlas-saga-orchestrator` (FR-7.2 — **PRD correction**)

The PRD expected no orchestrator change. Two are required.

**(a) Thread the error code.** `handleCharacterMesoErrorEvent`
(`kafka/consumer/character/consumer.go:166`) calls `StepCompleted(tx, false)` and
drops `e.Body.Error` on the floor. Change to:

```go
_ = p.StepCompletedWithResult(e.TransactionId, false,
        map[string]any{"errorCode": e.Body.Error})
```

exactly as `handleCharacterApTransferErrorEvent` already does. Backward compatible —
every existing consumer of this step ignores the result map, and the acceptance-table
registration is unchanged (`AwardMesos: {MesoChanged, MesoError}`,
`event_acceptance.go:132`, with `MesoError → OutcomeFailure` at line 327). The
existing `NOT_ENOUGH_MESO` behaviour is byte-identical apart from a populated result
map.

**(b) A dedicated compensator.** See §5.2 — this is the load-bearing one.

---

## 5. Client feedback

### 5.1 Unlock on every outcome (FR-5.1) — open question 5 resolved

| Outcome | Unlock source |
|---|---|
| Success | `STAT_CHANGED{TypeMeso}` from `atlas-character`, `ExclRequestSent: true` (`producer.go:238`), rendered by `handleStatusEventStatChanged` → `NewStatChanged(updates, true)` |
| Handler rejection (lookup failure, zero/oversized amount) | handler-local `enableActions()` |
| Saga failure | pink text + `enableActions()` in the saga-failed arm (§5.2) |

**The handler sends no unlock on the success path.** The PRD's open question asked
whether to unlock immediately or defer until the meso stat update lands; the answer
is that the stat update *is* the unlock, so the ordering is correct by construction
and an extra empty `StatChanged` would be a redundant packet racing the real one.
This also matches every sibling arm — none of them unlocks on success.

### 5.2 Meso-ceiling message (FR-5.2) — and the characterId-0 bug

The naive implementation ("add a `SagaTypeMesoSackUse` arm to `handleFailedEvent`")
**cannot work as written.** `handleFailedEvent` resolves the session via
`session.NewProcessor(...).GetByCharacterId(sc.Channel())(e.Body.CharacterId)`
(`kafka/consumer/saga/consumer.go:253`), and `EmitSagaFailed` populates that field
from `ExtractCharacterCreationIds`, which returns `0` for any saga without a
`CreateCharacter` step (`saga/producer.go:138-152`). Every `meso_sack_use` failure
would emit `characterId: 0`, the session lookup would miss, and the player would get
silence *and* stay input-locked — precisely the bug this task exists to remove.

This is not speculation: `compensateNoteSend` documents the same trap in a comment
and works around it by calling `EmitSagaFailedByIds` directly with the sender id
(`compensator.go:1636`).

Design: `compensateMesoSackUse`, modelled on `compensatePointReset` (identical
lifecycle idioms) but emitting through `EmitSagaFailedByIds`:

```go
if s.SagaType() == MesoSackUse { return c.compensateMesoSackUse(s, failedStep) }   // dispatch, ~compensator.go:276

func (c *CompensatorImpl) compensateMesoSackUse(s Saga, failedStep Step[any]) error {
    c.DispatchMesoSackRollbacks(s)                       // DestroyAsset → RequestCreateItem
    if !GetCache().TryTransition(ctx, tx, Compensating, Failed) { … return nil }
    SagaTimers().Cancel(tx); GetCache().Remove(ctx, tx)

    characterId := mesoSackCharacterId(s)                // AwardMesos ?? DestroyAsset payload
    errorCode   := mesoSackErrorCode(failedStep)         // result["errorCode"] ?? ErrorCodeUnknown
    return EmitSagaFailedByIds(c.l, c.ctx, tx, string(s.SagaType()),
            0, characterId, errorCode, reason, failedStep.StepId())
}
```

`DispatchMesoSackRollbacks` is the same three-line reverse-walk as
`DispatchPointResetRollbacks`: iterate steps backwards, skip anything not
`Completed`, invert `DestroyAsset` via `RequestCreateItem`. The failed `AwardMesos`
step committed nothing (`RequestChangeMeso` rejects inside its transaction) and has
no inverse, so a completed-only walk is exactly right. Reusing
`DispatchPointResetRollbacks` directly would be tempting and wrong — it is named and
documented for its saga type; a sibling function keeps the two independent.

`mesoSackCharacterId` prefers the `AwardMesosPayload` (present on every
`meso_sack_use` saga by construction) and falls back to the `DestroyAssetPayload`,
matching `compensateNoteSend`'s belt-and-braces shape. `ExtractCharacterId`
(`saga/character_extractor.go`) already handles both payload types and can be used
per step rather than hand-rolling the switch.

Channel side, a new arm in `handleFailedEvent` beside the point-reset arm:

```go
if e.Body.SagaType == saga.SagaTypeMesoSackUse {
    msg := mesoSackFailureMessage(e.Body.ErrorCode)
    _ = session.Announce(…)(chatpkt.WorldMessageWriter)(
            writer.WorldMessagePinkTextBody("", "", msg))(s)
    _ = session.Announce(…)(statpkt.StatChangedWriter)(
            statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
    return
}
```

**Open question 1 resolved — the copy.** Searching the v83 arm's string-pool
references turned up no meso-ceiling string the client owns (`0x31D` is the
confirmation prompt; the ceiling case never arises client-side because the client
never knows the player's headroom). The message is therefore server-authored, exactly
as `pointreset.ErrorMessage` is:

| `errorCode` | Message |
|---|---|
| `MESO_OVERFLOW` | `You cannot hold any more mesos.` |
| anything else | `You are unable to use this item right now.` |

Both go out as pink text via `WorldMessagePinkTextBody("", "", msg)`, the same
channel the point-reset rejections use. The generic fallback matters: a
`meso_sack_use` saga can also fail on the destroy step or by timeout
(`SAGA_TIMEOUT`), and claiming "you cannot hold any more mesos" then would be a lie.

---

## 6. Testing

**`atlas-data`** — reader unit tests over a synthetic `0520.img` node set: `meso`
present (1000000), absent (⇒ 0), and explicitly `0`. Assert the field is populated
independently of `Spec` (i.e. `Spec` gains no `meso` key).

**`atlas-channel`** — table tests on `handleMesoSackUse` using the existing
package-var seam pattern (`cashItemInSlotFunc` in `character_cash_item_use.go`,
`itemInSlotFunc` in `teleport_rock_use.go`) to stub the cash lookup and capture the
created saga:

- amount `1000000` ⇒ exactly two steps, in order `consume_meso_sack` (DestroyAsset,
  qty 1, template = itemId) then `award_mesos` (AwardMesos, `Amount: 1000000`,
  `ShowEffect: true`); no `StatChanged` written.
- amount `0` (Maple Point sack `5200009`) ⇒ no saga created, exactly one
  `StatChanged` written with `ExclRequestSent`.
- amount `> math.MaxInt32` ⇒ same rejection shape as `0`.
- cash lookup error (404) ⇒ same rejection shape.

Plus a `handleFailedEvent` test asserting the `MESO_OVERFLOW` arm writes pink text
then `StatChanged`, and the unknown-code arm writes the generic text.

Builders only — no `*_testhelpers.go`.

**`atlas-saga-orchestrator`** — mirror `point_reset_compensation_test.go`:
a completed `consume_meso_sack` + failed `award_mesos` saga must produce exactly one
`RequestCreateItem` for the sack's template/character, and exactly one saga-failed
emission whose `characterId` is **non-zero and equal to the payload's** (this
assertion is the regression guard for §5.2) and whose `errorCode` is `MESO_OVERFLOW`.

**`atlas-character`** — a `RequestChangeMeso` overflow test asserting one
`StatusEvent[StatusEventMesoErrorBody]` with `Type: ERROR`, `Error: MESO_OVERFLOW`,
and that the character's meso is unchanged.

**Manual / live**, once per tenant after re-ingest: `5200000` on a fresh character
(credits 1,000,000, one sack gone, chat line renders, client responsive);
`5200001`/`5200002`; a near-ceiling character (mesos unchanged, sack retained, pink
text, client responsive); `5200009` on v87 (nothing consumed, warn logged, client
responsive); `5202000` on v92 (pays flat `info/meso`).

---

## 7. Files touched

| Service / lib | File | Change |
|---|---|---|
| `atlas-data` | `data/cash/reader.go` | parse `info/meso` |
| | `data/cash/rest.go` | `Meso uint32` |
| | `data/cash/reader_test.go` | new |
| `atlas-channel` | `data/cash/rest.go` | `Meso uint32` |
| | `socket/handler/character_cash_item_use.go` | `CashSlotItemTypeCurrencySack` const + dispatch |
| | `socket/handler/character_cash_item_use_meso_sack.go` | new — the arm |
| | `saga/model.go` | re-export `MesoSackUse` |
| | `kafka/message/saga/kafka.go` | `SagaTypeMesoSackUse` |
| | `kafka/consumer/saga/consumer.go` | failure arm + message mapper |
| `atlas-character` | `kafka/message/character/kafka.go` | `StatusEventErrorTypeMesoOverflow` |
| | `character/producer.go` | `mesoOverflowErrorStatusEventProvider` |
| | `character/processor.go` | emit on the overflow path |
| `atlas-saga-orchestrator` | `saga/model.go` | re-export `MesoSackUse` |
| | `kafka/consumer/character/consumer.go` | thread `errorCode` |
| | `saga/compensator.go` | dispatch arm + `compensateMesoSackUse` + `DispatchMesoSackRollbacks` |
| | `kafka/message/character/kafka.go` | `StatusEventErrorTypeMesoOverflow` (mirror) |
| `libs/atlas-saga` | `model.go` | `MesoSackUse Type` |

No `go.mod` changes ⇒ no mandatory `docker buildx bake`. No template changes ⇒ none
of the template guards apply. No new shared lib ⇒ no `Dockerfile`/`go.work` edits.

---

## 8. Risks and non-obvious traps

- **Re-ingest is a hard prerequisite.** Deploying the code without re-ingesting a
  tenant's WZ leaves `meso` absent, and FR-2.4 turns every sack use into a logged
  rejection. Loud in logs, invisible to a smoke test that never uses a sack.
  The plan must gate "live" on per-tenant field verification.
- **`int32` truncation.** Covered by the `math.MaxInt32` guard (§4.3); if that guard
  is ever dropped, a large sack becomes a meso *thief*.
- **characterId-0 on saga failure.** Covered by §5.2. The regression test asserting
  a non-zero `characterId` on the failed event is the thing that keeps it fixed.
- **Do not version-gate the type constant.** §3.1(a). The v48/v61 client tables
  disagree with `19` and that is fine; a "fix" would break the branch on those
  builds.
- **v92/v95 random sacks show an amount-less prompt.** §3.1(c). Accepted; a gaussian
  roll over `mesomin`/`mesomax`/`mesostdev` is a clean follow-up that changes only
  the handler's amount resolution, not the saga or the wire.
- **`5202000` on JMS v185 is unverified.** Only that one id exists there and no local
  JMS WZ tree is extracted; whether it carries a base `info/meso` is unknown. FR-2.4
  fails closed either way, so this is a verification item, not a blocker.
