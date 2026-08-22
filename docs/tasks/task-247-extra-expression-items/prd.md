# Extra-Expression (Emote) Cash Items — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

`Item.wz/Cash/0516.img` defines fifteen "extra expression" cash items —
`05160000`–`05160014` in GMS v83.1 data — named *Queasy*, *Panicky*,
*Sweetness*, *Smoochies*, *Wink*, *Ouch*, *Sparkling Eyes*, *Flaming*, *Ray*,
*Goo Goo* and five more. Every one carries the same `desc` in
`String.wz/Cash.img`: *"On the KeyConfig, configure this expression on a button
of your choice. Press the button and watch your character display …"* Their
`info` node holds only `icon`, `iconRaw`, and `cash=1` — there is **no `spec`
node**, so they are not consumables in the `atlas-consumables` sense. They are
permanent unlocks that extend the character's emote palette beyond the seven
free base emotes.

The backlog entry that produced this task
(`docs/research/missing-features/items-and-consumables.md:31`) framed the gap as
*"`ClassificationExpression` → type 6 — no dispatch arm for type 6"* in
`CharacterCashItemUseHandleFunc`. **That framing is wrong, and this task
corrects it.** Client analysis (§1.1) shows that type-6 items never produce a
cash-item-use packet at all: the client converts the item use into an ordinary
emote request. Adding a type-6 arm to `character_cash_item_use.go` would be dead
code.

The gap that actually exists is on the emote path. `CharacterExpressionHandleFunc`
(`services/atlas-channel/.../socket/handler/character_expression.go:20`) forwards
whatever `emote` value the client sends straight into the expression pipeline
with **no validation of any kind** — no range check, and no check that the
character owns the cash item the extra expression requires. Separately, the
expression Kafka contract drops the `duration` and `byItemOption` fields that
GMS v95 clients send and expect echoed back, so the broadcast is hardcoded to
`duration=0, byItemOption=false` (the standing TODO at
`services/atlas-channel/.../kafka/consumer/expression/consumer.go:57`).

### 1.1 Client-derived behaviour (GMS v95 IDB `ecc757f4`, GMS v87 IDB `c0829805`)

| Fact | Evidence |
|---|---|
| Double-clicking a cash item checks the *etc* type first | `CDraggableItem::OnDoubleClicked` @`0x50814b` calls `get_etc_cash_item_type`; if non-zero it calls `CWvsContext::SendEtcCashItemUseRequest` @`0x508165` and returns — no `SendConsumeCashItemUseRequest` |
| `get_etc_cash_item_type` is a whitelist over `get_cashslot_item_type` | `get_etc_cash_item_type` @`0x49c780`: passes through types `{1,2,3,4,5,6,7,36,37,40,42,46,55,58,59,60,63,77}`, else returns 0 |
| Type 6 sends an emote change, not an item use | `CWvsContext::SendEtcCashItemUseRequest` @`0xa02bf0`, `case 6:` @`0xa02c86` → `CWvsContext::SendEmotionChange(this, nItemID % 100 + 8, 0, -1)` |
| Argument order | `CWvsContext::SendEmotionChange` @`0x9f9320` is `(unsigned int nEmotion, unsigned __int8 bByItemOption, int nDuration)` — so the call passes `bByItemOption = 0`, `nDuration = -1` |
| Wire body | `SendEmotionChange` @`0x9f93c0`–`0x9f93d8`: `Encode4(CAvatar::GetEmotion())` + `Encode4(nDuration)` + `Encode1(bByItemOption)` |
| Client-side guards on the emote request | `nEmotion <= 0x17` (23) and `timeGetTime() - m_tLastEmotionChange >= 2000` @`0x9f9386`; blocked entirely while morphed (`AddChatMorphedMsg` @`0x9f9360`) |
| Same mapping on v87 | `CWvsContext::SendEtcCashItemUseRequest` @`0xab4f91` → `SendEmotionChange(a2 % 100 + 8)` (single argument; v87's `SendEmotionChange` encodes only `Encode4(emotionId)`) |
| These items are also key-mappable | `CDraggableItem::MapFuncKey` @`0x501756` / `UnmapFuncKey` @`0x501a7d` / `CUIKeyConfig::CompareValidateFuncKeyMappedInfo` @`0x7d955d` all call `get_etc_cash_item_type`; `CUserLocal::UseFuncKeyMapped` @`0x932e20` calls `SendEmotionChange` at `0x933859` and `0x933898` |

**Derived emote↔item mapping:** `emote = itemId % 100 + 8`, therefore
`itemId = 5159992 + emote`. `5160000` → emote 8 … `5160014` → emote 22. The
client's own `nEmotion <= 23` cap admits emote 23 (`5160015`), which has no entry
in v83.1 data — a uniform ownership rule handles it without a special case.

## 2. Goals

Primary goals:

- Reject expression requests for emote ids the client could never legitimately send.
- Gate extra expressions (emote 8–23) on the character actually owning the
  corresponding `516xxxx` cash item in their CASH compartment.
- Carry `duration` and `byItemOption` end-to-end from the serverbound
  `ExpressionRequest` through the expression Kafka contract to the clientbound
  `CharacterExpression` broadcast, closing the `consumer.go:57` TODO.
- Correct the false premise recorded in
  `docs/research/missing-features/items-and-consumables.md` so the backlog stops
  advertising a dispatch arm that must not be written.

Non-goals:

- No `CashSlotItemType(6)` dispatch arm in `character_cash_item_use.go`. Stock
  clients never route these items there; a defensive arm was explicitly
  considered and rejected.
- No wire-format change to `character/serverbound/ExpressionRequest` or
  `character/clientbound/CharacterExpression`. Both codecs are already correct
  and evidence-verified; only the Go constructor surface may change.
- No server-side re-implementation of the client's 2000 ms emote cooldown or
  morph block. Those are UX gates on a purely cosmetic action, not exploit
  surface, and would require new per-character state in `atlas-expressions`.
- No consumption of the item, no inventory mutation, no `atlas-consumables`
  involvement. These items have no `spec` node and are permanent.
- No cash-shop purchase-path work for the `516` range.
- Other unimplemented cash types (cube/potential, jukebox trigger, cosmetic
  coupons, character-creation items) remain out of scope.

## 3. User Stories

- As a player who bought a *Wink* expression item, I want pressing my mapped
  KeyConfig button to show the wink emote to everyone in my map, so the item I
  paid for does something.
- As a player who does **not** own any extra-expression item, I want to still be
  able to use the seven free base emotes without interruption.
- As a server operator, I want a client that fabricates an emote id it has not
  purchased to be rejected server-side, so cash items retain their value.
- As a server operator, I want an out-of-range emote id to be dropped rather
  than broadcast, so a malformed or hostile client cannot push undefined
  expression ids to every other client in the map.
- As a GMS v95 player, I want the emote's duration to behave the way the client
  asked for it, rather than always being sent back as `0`.

## 4. Functional Requirements

### 4.1 Emote range validation (`atlas-channel`)

- **FR-1.1** `CharacterExpressionHandleFunc` MUST reject and drop any request
  whose decoded `emote` is greater than `23`. The client's own guard is
  `nEmotion <= 0x17` (`SendEmotionChange` @`0x9f9386`), so a larger value cannot
  originate from a stock client.
- **FR-1.2** A rejected request MUST be logged at warn level, naming the
  character id and the offending emote id, and MUST NOT emit any Kafka message.
- **FR-1.3** Emote ids `0`–`7` MUST continue to pass through unchanged with no
  ownership check. These are the free base emotes.

### 4.2 Extra-expression ownership gating (`atlas-channel`)

- **FR-2.1** For an emote id in `[8, 23]`, the handler MUST resolve the required
  cash item id as `itemId = 5159992 + emote` and verify the character owns at
  least one asset with that template id in their CASH compartment.
- **FR-2.2** The required item id MUST be derived from a named constant/helper
  rather than a bare literal at the call site. Follow repository convention:
  check `libs/atlas-constants/item/` for an existing home before adding one.
  `item.ClassificationExpression` (= `516`) already exists at
  `libs/atlas-constants/item/constants.go:96`.
- **FR-2.3** The helper MUST be total over the gated range and MUST NOT
  special-case emote 23. `5159992 + 23 = 5160015` simply resolves to an item the
  character cannot own in v83.1 data, so the ownership check fails naturally.
- **FR-2.4** If the character does not own the required item, the request MUST
  be dropped: logged at warn level with character id, emote id, and required
  item id, and no Kafka message emitted.
- **FR-2.5** If the ownership lookup itself errors (character service
  unreachable, decoration failure), the request MUST be dropped and the error
  logged. Fail closed — a lookup failure MUST NOT be treated as ownership.
- **FR-2.6** Ownership MUST be resolved through the existing character
  inventory surface. No item-keyed lookup exists on
  `character.Processor` (it exposes only `GetItemInSlot`), but the decorated
  model path does:
  `NewProcessor(l, ctx).GetById(p.InventoryDecorator)(characterId)` →
  `.Inventory().Cash().FindFirstByItemId(uint32(itemId))`
  (`services/atlas-channel/.../compartment/model.go:58`). Design may instead add
  a narrow processor method; either is acceptable, a filesystem-wide new
  abstraction is not.
- **FR-2.7** The lookup MUST sit behind a package-level test seam following the
  established precedent in this package (`cashItemInSlotFunc`,
  `requestItemConsumeFunc`, `karmaCharacterProcessorFunc` in
  `character_cash_item_use.go:1012-1036`), so handler tests assert which branch
  a request reached without a live character service.

### 4.3 Duration and `byItemOption` propagation

- **FR-3.1** `atlas-channel`'s expression Kafka `Command`
  (`services/atlas-channel/.../kafka/message/expression/kafka.go`) MUST carry
  `duration` and `byItemOption`, populated from the decoded
  `ExpressionRequest.Duration()` and `.ByItemOption()`.
- **FR-3.2** `atlas-expressions`' `Command` and `StatusEvent`
  (`services/atlas-expressions/.../kafka/message/expression/kafka.go`) MUST
  carry the same two fields, and `handleChangeCommand` MUST thread them into
  `Processor.ChangeAndEmit`.
- **FR-3.3** `atlas-channel`'s expression `Event`
  (same file as FR-3.1) MUST carry them, and the consumer at
  `services/atlas-channel/.../kafka/consumer/expression/consumer.go:62` MUST
  pass them to `charpkt.NewCharacterExpression` instead of the current
  hardcoded `0` / implicit `false`. The TODO comment at lines 57–60 MUST be
  removed, not left stale.
- **FR-3.4** `libs/atlas-packet/character/clientbound.NewCharacterExpression`
  MUST expose `byItemOption`, which today is unreachable — the struct field
  exists but the constructor takes only three arguments
  (`expression.go:41`). Design chooses between widening the constructor
  (updating the three existing call sites in `v61_test.go:724`,
  `v72_test.go:506`, `v79_test.go:520`) or adding a `With…` variant. Either
  way this is a **Go-surface change only**; the encoded bytes for a given
  input MUST NOT change.
- **FR-3.5** Sign handling MUST be explicit. `ExpressionRequest.duration` is
  `int32` and the type-6 case sends `-1`; `CharacterExpression.duration` is
  `uint32`. The conversion MUST preserve the bit pattern so `-1` reaches the
  wire as `0xFFFFFFFF`, matching what the client encoded.
- **FR-3.6** On versions where the fields are not on the wire, behaviour MUST be
  unchanged from today. `ExpressionRequest` decodes `duration`/`byItemOption`
  only for `GMS && MajorVersion > 87`; on GMS ≤ 87 and JMS they stay at their
  zero values and the existing writer gates continue to govern what is emitted.
- **FR-3.7** The revert sweep MUST be unaffected in observable behaviour.
  `revertExpression` (`services/atlas-expressions/.../expression/task.go`)
  emits expression `0`; it MUST emit `duration = 0`, `byItemOption = false`.
- **FR-3.8** The registry `Model`
  (`services/atlas-expressions/.../expression/model.go`) MUST NOT be extended
  with these fields. Nothing reads them back — the revert path per FR-3.7 uses
  fixed values — so persisting them would be dead state.

### 4.4 Backlog correction

- **FR-4.1** `docs/research/missing-features/items-and-consumables.md` MUST be
  updated: the row at line 31 and the itemisation at line 80 currently assert a
  missing type-6 dispatch arm. Both MUST be corrected to record that type 6 is
  routed by the client through `SendEtcCashItemUseRequest` → `SendEmotionChange`
  and never reaches the cash-item-use handler, citing the addresses in §1.1.

### 4.5 Version coverage

- **FR-5.1** The gating and validation in §4.1–§4.2 MUST apply on every
  configured version — all ten GMS templates and JMS v185 — since the emote
  request path is version-independent above the codec.
- **FR-5.2** No template registration change is required: `CharacterExpressionHandle`
  is already wired (`services/atlas-channel/.../main.go:948`).

## 5. API Surface

No REST endpoints are added or modified.

Kafka contract changes (both topics are internal service-to-service):

**`COMMAND_TOPIC_EXPRESSION`** — producer `atlas-channel`, consumer `atlas-expressions`:

```json
{
  "transactionId": "uuid",
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "expression": 8,
  "duration": -1,
  "byItemOption": false
}
```

**`EVENT_TOPIC_EXPRESSION`** — producer `atlas-expressions`, consumer `atlas-channel`:
same two additional fields alongside the existing payload.

Both additions are backward compatible: an old consumer ignores the new keys, and
an old producer's message decodes with the new fields at their zero values, which
reproduces today's `duration=0, byItemOption=false` behaviour exactly.

> **Pre-existing discrepancy to confirm during design (not necessarily in scope):**
> `atlas-expressions`' `Command` declares `TransactionId`, but `atlas-channel`'s
> `Command` struct does not and `SetCommandProvider`
> (`services/atlas-channel/.../character/expression/producer.go`) never sets it,
> so every command arrives with the zero UUID. This predates the task; fix it
> only if design concludes it is trivially adjacent.

## 6. Data Model

No new entities, no database tables, no migrations. `atlas-expressions` state
lives in a Redis TTL registry (`defaultTTL = 5 * time.Second`,
`expression/registry.go:16`) and per FR-3.8 its `Model` is unchanged.

Multi-tenancy is unchanged: the registry is already keyed per tenant via
`atlas.TTLRegistry[uint32, Model]` with `tenant.MustFromContext`, and the new
Kafka fields carry no tenant-scoped data.

## 7. Service Impact

| Service / library | Change |
|---|---|
| `services/atlas-channel` | `socket/handler/character_expression.go`: range check, ownership gate, ownership test seam, duration/byItemOption forwarding. `character/expression/producer.go`: populate the two new command fields. `kafka/message/expression/kafka.go`: extend `Command` and `Event`. `kafka/consumer/expression/consumer.go`: pass the fields to the writer; delete the stale TODO. |
| `services/atlas-expressions` | `kafka/message/expression/kafka.go`: extend `Command` and `StatusEvent`. `kafka/consumer/expression/consumer.go`: thread the fields through `handleChangeCommand`. `expression/processor.go`: widen `Change`/`ChangeAndEmit`. `expression/producer.go`: emit the fields. `expression/task.go`: revert emits fixed zero values. |
| `libs/atlas-packet` | `character/clientbound/expression.go`: expose `byItemOption` on the constructor surface. No wire change. Existing `v61`/`v72`/`v79` byte tests updated for the new signature if the constructor is widened. |
| `libs/atlas-constants` | Possibly one helper mapping emote id → required `516xxxx` item id, under `item/`. Check for an existing equivalent first. |
| `services/atlas-consumables` | **No change.** These items are non-consuming. |
| `services/atlas-data` | **No change.** `ClassificationExpression` already classifies as `"expression"` (`item/classify.go:192`). |
| `services/atlas-ui` | **No change.** |

## 8. Non-Functional Requirements

- **Performance.** The ownership check adds one character-service round trip per
  extra-expression request. It MUST run only on the `[8, 23]` branch — base
  emotes 0–7 stay on a zero-lookup path. The client's own 2000 ms cooldown bounds
  the request rate per character.
- **Security.** Fail closed on lookup error (FR-2.5). The gate is the only thing
  standing between a modified client and free use of paid cosmetics.
- **Observability.** Every rejection path logs at warn with character id, emote
  id, and (for FR-2.4) the required item id, so operators can distinguish "player
  doesn't own it" from "client sent garbage" from "lookup broke".
- **Multi-tenancy.** Tenant resolution is via `tenant.MustFromContext` throughout;
  no new tenant-scoped storage is introduced.
- **Backward compatibility.** Both Kafka field additions are additive and
  zero-value-safe, so a partial rollout of `atlas-channel` and
  `atlas-expressions` degrades to today's behaviour rather than breaking.

## 9. Open Questions

1. **Base-emote duration on GMS v95.** `CUserLocal::UseFuncKeyMapped`
   @`0x932e20` calls `SendEmotionChange` twice (`0x933859`, `0x933898`) — one of
   these is the base-emote path. The arguments it passes have not been derived.
   Since FR-3.1 forwards whatever the client encoded rather than synthesising a
   value, this does not block implementation, but design should decompile
   `UseFuncKeyMapped` to confirm no base-emote case sends a duration Atlas would
   now start echoing where it previously sent `0`. If it does, that is a
   behaviour change to call out explicitly, not to smuggle in.
2. **Should client-supplied `duration` be clamped?** FR-3.1 trusts the value. It
   is cosmetic and bounded by the client's own animation handling, but design may
   choose an upper clamp if there is evidence a large value misbehaves.
3. **Constructor vs. `With…` variant for FR-3.4.** A design-phase call; both
   satisfy the requirement.
4. **`TransactionId` gap** in `atlas-channel`'s expression `Command` (§5). In
   scope only if trivially adjacent.

## 10. Acceptance Criteria

- [ ] An `ExpressionRequest` with `emote > 23` is dropped, logged at warn, and
      emits no Kafka message. Covered by a handler test.
- [ ] An `ExpressionRequest` with `emote` in `0`–`7` is forwarded with no
      ownership lookup performed. Covered by a handler test asserting the seam
      was not called.
- [ ] An `ExpressionRequest` with `emote = 8` from a character holding `5160000`
      in their CASH compartment is forwarded. Covered by a handler test.
- [ ] The same request from a character **not** holding `5160000` is dropped and
      logged. Covered by a handler test.
- [ ] The same request when the ownership lookup returns an error is dropped —
      not forwarded. Covered by a handler test.
- [ ] The emote→item mapping is asserted for the boundaries: `emote 8 → 5160000`,
      `emote 22 → 5160014`, `emote 23 → 5160015`. Covered by a unit test on the
      helper.
- [ ] `duration` and `byItemOption` decoded from a GMS v95 `ExpressionRequest`
      reach the clientbound `CharacterExpression` unchanged, including
      `duration = -1` arriving on the wire as `0xFFFFFFFF`. Covered by an
      end-of-pipeline test or a byte-level test on the writer.
- [ ] On GMS v83/v87 and JMS v185, the emitted `CharacterExpression` bytes are
      byte-identical to `main`'s output for the same input. Covered by the
      existing evidence tests still passing unmodified in their assertions.
- [ ] The TODO block at `services/atlas-channel/.../kafka/consumer/expression/consumer.go:57-60`
      is deleted.
- [ ] `character_cash_item_use.go` is **unchanged** — no type-6 arm was added.
- [ ] `docs/research/missing-features/items-and-consumables.md` lines 31 and 80
      are corrected per FR-4.1, citing the §1.1 addresses.
- [ ] All new/changed code follows the repository Builder pattern for test setup;
      no `*_testhelpers.go` files are introduced.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `plan-adherence-reviewer` both report
      clean before the PR is opened.
