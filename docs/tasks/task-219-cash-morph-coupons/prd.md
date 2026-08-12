# Cash Transformation (Morph) Coupons — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12
---

## 1. Overview

Cash-inventory transformation coupons (`Item.wz/Cash/0530.img.xml`, item classification 530) are
purchasable cash items that transform the character into a Morph.wz creature for ten minutes and heal
50 HP. Today they do nothing at all: the client sends the use request, and `atlas-channel`'s cash-item
handler falls through every branch to a terminal warn
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:640`,
`"Character [%d] attempting to use cash item [%d] in slot [%d] of type [%d]"`). No item is consumed and
no effect is applied. The item is listed as "Wholly missing #7 / Transformation-morph coupons" in
`docs/research/missing-features/items-and-consumables.md:38`.

The effect itself is already built. Task 140 (morph potion routing) landed the morph applier in
`atlas-consumables`: `computeEffectPlan`/`ApplyItemEffects`
(`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:232-259`) turn a `morph`
spec into a `TemporaryStatTypeMorph` statup whose amount is the morph id, applied through the ordinary
`atlas-buffs` pipeline, with death cancellation inherited from the respawn saga's `CancelAllBuffs`. The
morph ids used by the coupons are drawn from the same Morph.wz namespace as the use-tab morph potions —
`Cash/0530.img.xml` carries `morph` values 1, 2 and 3, and `Consume/0221.img.xml` carries 1, 2, 3, 10,
11, 12, … — so no new effect semantics are required.

What is missing is plumbing, in three places. `atlas-data`'s cash reader never parses the `morph` and
`hp` spec children, so those values are dropped at ingest and no downstream service can ever observe
them. `atlas-channel` has no arm for classification 530. `atlas-consumables` has no consume branch for
it — and cannot reuse `ConsumeStandard`, which hard-codes the Use compartment
(`processor.go:471`, `inventory2.TypeValueUse`) and fetches from the *consumable* data resource, where
cash items do not exist. This task closes all three gaps.

## 2. Goals

Primary goals:

- Using a `Cash/0530` coupon transforms the character for the item's `time` duration, heals its `hp`
  amount, and decrements the coupon from the Cash compartment.
- The `morph` and `hp` spec values on cash items survive WZ ingest and are exposed over the atlas-data
  cash REST resource.
- The feature is version-neutral: it is selected by item classification alone, with no region/major
  version literals, so it fires on every tenant whose template registers `CharacterCashItemUseHandle`.

Non-goals:

- The other fifteen unimplemented cash slot-item types enumerated in the same backlog table (meso sacks
  520, jukebox 510, pet name tag 517, Duey 533, name change / world transfer 540, Maple Life 543, MiuMiu
  545, expiration extenders 550, karma scissors 552, store permit, cosmetic coupons, emotes, pet skill
  items). Each is its own task.
- `morphRandom` on cash items. No item in `Cash/0530.img.xml` carries a `morphRandom` table in any
  inspected WZ set; the weighted-random selector added by task 140
  (`consumables/consumable/morph.go`) is not wired into this path.
- Registering `CharacterCashItemUseHandle` in `template_gms_12_1.json`. It is the one template of eleven
  that omits the handler (verified by grep across
  `services/atlas-configurations/seed-data/templates/`), so gms_12 is a documented no-op for the entire
  cash-item-use family, not a regression introduced here.
- Any new clientbound packet writer. Morph rides the existing temporary-stat pipeline.
- Re-ingesting live tenant WZ data. See §9 / §10 — the reader change only affects newly ingested data,
  and the operational re-ingest is explicitly deferred to a follow-up.
- Anti-cheat for attacking while morphed. Ruled out and justified in task 140's PRD §2.

## 3. User Stories

- As a player, I want using a transformation coupon from my Cash inventory to transform my character for
  ten minutes, so that the item I paid for does what its description says instead of silently doing
  nothing.
- As a player, I want the coupon to be removed from my Cash inventory when it takes effect, and not
  removed when it fails, so that I never lose a paid item to a no-op.
- As a player, I want to be able to act immediately after using a coupon, rather than having the client
  wedge because the server never released its request lock.
- As a player, I want using a second coupon while already transformed to replace the current
  transformation and restart the timer.

## 4. Functional Requirements

### 4.1 Client contract (IDA-verified)

Verified against GMS v83 `MapleStory_dump.exe`
(`E:\Programs\Nexon\IDBs_v9\GMS\v83_Me\MapleStory_dump.exe.i64`), function
`CWvsContext::SendConsumeCashItemUseRequest` @ `0xa0a63f`:

- Header, common to all types: `Encode2(nPOS)` @ `0xa0a6bf`, `Encode4(nItemID)` @ `0xa0a6cb`, then
  `get_consume_cash_item_type` @ `0xa0a6d1` drives a 58-case jump table keyed on `type - 12`
  (cases 12..69).
- Classification 530 resolves to type **40** (GMS < 95) / **41** (GMS >= 95) in Atlas's
  `GetCashSlotItemType` (`character_cash_item_use.go:923-929`). Type 40 passes the client's own
  whitelist in `get_consume_cash_item_type` @ `0x4863d5` (the `result != 40` guard lets it through), so
  the request is genuinely sent on v83.
- **The case-40 arm (`0xa0caf0`–`0xa0cb37`) contains no `Encode*` call at all.** It runs three client-side
  predicates — the first, `sub_A0ECCD` @ `0xa0eccd`, is literally `itemId / 10000 == 530` — calls
  `play_item_sound(nItemID, 0x29)` @ `0xa0cb30`, then jumps to the shared send tail. The per-type
  sub-body is therefore **empty**.
- Shared tail: `CWvsContext::CanSendExclRequest(0x1F4, 0)` @ `0xa0a8fa`; on success `loc_A0EA53`
  appends `Encode4(get_update_time())` @ `0xa0ea5c` (the trailing-updateTime layout), calls
  `SendPacket`, then **`CWvsContext::SetExclRequestSent(1)`** @ `0xa0ea6f`.

Requirements that follow:

- **FR-1.1**: No new wire layout. The request is `int16 source, int32 itemId` plus `updateTime`, whose
  position is already handled by `cashsb.UpdateTimeFirst`
  (`libs/atlas-packet/cash/serverbound/item_use.go:21-23`; leading on GMS >= v87 and JMS, trailing on
  GMS <= v84). The common `cashsb.ItemUse` header decode is sufficient for the leading-updateTime
  versions.
- **FR-1.2**: For trailing-updateTime versions a sub-body reader MUST consume the trailing `int32`
  updateTime and nothing else, following the shape of the existing per-type codecs in
  `libs/atlas-packet/cash/serverbound/`. It MUST read nothing when `UpdateTimeFirst(t)` is true. A
  round-trip encode/decode test MUST pin both variants.
- **FR-1.3**: The type byte MUST NOT be used to select this arm. The collision is cross-version, not
  within one tenant: on GMS >= 95, `ClassificationGachaponCoupon` maps to 40 — the same byte
  classification 530 (transformation) uses pre-95 — and pre-95, `ClassificationPetEvolution` maps to
  41 — the same byte transformation uses on GMS >= 95
  (`character_cash_item_use.go:933-938`, `:974-979`; **corrected during task-8's PRD walk from this
  PRD's original, backwards "pre-95 gachapon" claim** — verified against source, not IDA, at
  task-8 time), so the arm MUST gate on
  `item.ClassificationTransformationCoupon` (`libs/atlas-constants/item/constants.go:97`), exactly as
  the existing Vicious Hammer / teleport rock arms gate on classification. No raw `530` literal.

### 4.2 atlas-data — cash spec parse

- **FR-2.1**: `services/atlas-data/atlas.com/data/cash/rest.go` MUST gain `SpecTypeMorph` (`"morph"`)
  and `SpecTypeHp` (`"hp"`) alongside the existing `SpecTypeTime`.
- **FR-2.2**: `services/atlas-data/atlas.com/data/cash/reader.go` MUST parse `spec/morph` and `spec/hp`
  into the `Spec` map, using the same "omit when zero" convention as the existing `expR`/`drpR`/`time`
  parses at `reader.go:130-138`. `spec/time` is already parsed (`reader.go:136-138`) and needs no change.
- **FR-2.3**: A reader unit test MUST pin all three of `5300000`/`5300001`/`5300002` shaped fixtures,
  asserting `morph` = 1/2/3, `hp` = 50, `time` = 600000. These values are read from
  `Item.wz/Cash/0530.img.xml`; they are byte-identical in the v83 corpus and in a post-v95 corpus, so
  the fixture is version-stable.
- **FR-2.4**: No other cash classification's parse output may change. Adding two keys to the `Spec` map
  is additive; a regression test over an existing 0521 (EXP coupon) or 0519 (pet skill) fixture MUST
  confirm unchanged output.

### 4.3 atlas-consumables — the consume branch

- **FR-3.1**: `services/atlas-consumables/atlas.com/consumables/cash/rest.go` MUST mirror FR-2.1 with
  `SpecTypeMorph` and `SpecTypeHp` so the values survive the REST hop.
- **FR-3.2**: `RequestItemConsume` (`consumable/processor.go:260-319`) MUST route
  `item.ClassificationTransformationCoupon` to a new `ConsumeMorphCoupon` consumer, placed alongside the
  existing classification branches. It MUST NOT be added to `usesStandardConsumer`: `ConsumeStandard`
  hard-codes `inventory2.TypeValueUse` (`processor.go:467,471,473`) and fetches from the *consumable*
  data resource (`processor.go:465`), neither of which is correct for a cash item.
- **FR-3.3**: `ConsumeMorphCoupon` MUST follow the `ConsumeCashPetFood` precedent
  (`processor.go:566-588`): read the item via `cash.NewProcessor(l, ctx).GetById(uint32(itemId))`,
  and on any error route to `ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)`
  so the reservation is released and the coupon is **not** consumed.
- **FR-3.4**: On success it MUST commit the reservation via
  `compartment.NewProcessor(l, ctx).ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot)`
  before applying effects, matching the ordering in `ConsumeStandard` (`processor.go:471-476`).
- **FR-3.5**: Effects applied, in this order:
  1. HP recovery of the `hp` spec value via `character.NewProcessor(l, ctx).ChangeHP(field, characterId, amount)`.
  2. A `TemporaryStatTypeMorph` statup whose amount is the `morph` spec value, applied via
     `buff.NewProcessor(l, ctx).Apply(field, characterId, -int32(itemId), 0, duration, statups)(characterId)`
     — the same source encoding (`-itemId`) and the same call shape as `ApplyItemEffects`
     (`processor.go:257`).
  The compartment id passed to `RequestItemConsume`'s reservation is already derived correctly by
  `inventory2.TypeFromItemId` (`processor.go:268`), which maps 5xxxxxx to Cash.
- **FR-3.6**: `duration` MUST be the raw `time` spec value in **milliseconds**, passed through
  unscaled. The `atlas-buffs` `ApplyCommandBody.Duration` contract is milliseconds
  (`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`), and
  `effectPlan.duration` is documented as "WZ `time` spec in ms" at `processor.go:156`. No
  seconds→milliseconds scaling is permitted; `tools/buff-duration-guard.sh` enforces this.
- **FR-3.7**: If the `morph` spec is absent or zero, no morph statup is applied. If `hp` is absent or
  zero, no HP change is issued. Neither case is an error, and neither blocks the other effect or the
  item consumption — a 530 item with an empty spec node consumes and does nothing, mirroring
  `computeEffectPlan`'s per-spec independence.
- **FR-3.8**: Using a coupon while already morphed replaces the active morph and restarts the timer.
  This is the default overwrite behaviour of the `atlas-buffs` apply path and requires no special
  handling; a test MUST pin that the second apply is issued unconditionally (no "already morphed"
  rejection).

### 4.4 atlas-channel — the handler arm

- **FR-4.1**: `CharacterCashItemUseHandleFunc` MUST gain an arm, gated per FR-1.3, that decodes the
  FR-1.2 sub-body and calls
  `consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, 1, updateTime)`
  — the same delegation the pet-consumable arm performs at `character_cash_item_use.go:62-69`.
- **FR-4.2**: The arm MUST sit behind the existing ownership check
  (`cashItemInSlotFunc` + template-id equality, `character_cash_item_use.go:51-57`), which every arm
  already inherits by position. No additional ownership logic.
- **FR-4.3 (exclusive-request lock)**: The client sets its exclusive-request lock when it sends
  (`SetExclRequestSent(1)` @ `0xa0ea6f`, gated by `CanSendExclRequest` @ `0xa0a8fa`). The morph outcome
  does **not** warp the character, so the lock must be released by a server response. Design MUST
  determine and pin which response clears it — the inventory-operation packet emitted by the consume
  commit, the stat-change from the `hp` recovery, or an explicit `session.EnableActions` — and MUST NOT
  assume the release is free. Note that nothing in the consumable path calls `session.EnableActions`
  today: the only callers are the chair and trade consumers
  (`kafka/consumer/chair/consumer.go:78,130`, `kafka/consumer/trade/consumer.go:192`). If an explicit
  unlock is required, it MUST be emitted only on the success path, never on the
  item-not-found / mismatch early return, which sends nothing today.

## 5. API Surface

No new REST endpoints, Kafka topics, or message types.

One existing resource changes shape, additively: `GET /data/{tenant}/cash-items/{id}` (JSON:API type
`cash_items`) gains two optional keys in its `spec` object.

```
GET /data/{tenantId}/cash-items/5300000
{
  "data": {
    "type": "cash_items",
    "id": "5300000",
    "attributes": {
      "slotMax": 200,
      "tradeBlock": true,
      "spec": { "morph": 1, "hp": 50, "time": 600000 }
    }
  }
}
```

Error cases are unchanged: an unknown id 404s, and the `spec` keys are simply absent for items whose WZ
`spec` node lacks them.

Internally, `atlas-consumables` emits the existing `COMMAND_TOPIC_CHARACTER_BUFF` apply command and the
existing compartment consume/error commands. No new command bodies.

## 6. Data Model

No database schema change and no migration. The cash-item projection in `atlas-data` is WZ-derived and
rebuilt by ingest; adding two keys to an existing JSONB `spec` map requires no DDL.

The one operational consequence: values added to a reader are only materialised for **newly ingested**
WZ. Tenants whose cash data was ingested before this change will continue to serve a `spec` object
without `morph`/`hp`, and the feature will be an inert no-op for them until the WZ is re-ingested. Per
the scoping decision recorded in §9, that re-ingest is an operational follow-up, not an acceptance
criterion of this task.

## 7. Service Impact

- **`atlas-data`** — `data/cash/rest.go` (two new `SpecType` constants), `data/cash/reader.go` (two new
  spec parses), `data/cash/reader_test.go` (FR-2.3, FR-2.4 fixtures).
- **`atlas-consumables`** — `cash/rest.go` (mirror the two `SpecType` constants),
  `consumable/processor.go` (new `ConsumeMorphCoupon` + one routing branch in `RequestItemConsume`),
  plus tests for routing, effect application, the zero-spec cases, and the re-use/replace case.
- **`atlas-channel`** — `socket/handler/character_cash_item_use.go` (one classification-gated arm),
  `libs/atlas-packet/cash/serverbound/` (one trailing-updateTime sub-body codec + round-trip test),
  and whatever FR-4.3 resolves to.
- **`atlas-buffs`, `atlas-character`, `atlas-configurations`** — no changes. The morph temporary stat,
  the HP change command, and the tenant socket templates are all already in place. In particular no
  template edit is required: `CharacterCashItemUseHandle` is already registered in ten of eleven
  templates (all but `template_gms_12_1.json`), so the opcode-order and duplicate-binding template
  guards are not implicated.

## 8. Non-Functional Requirements

- **Multi-tenancy**: no new tenant configuration. All behaviour derives from per-tenant WZ data served
  by `atlas-data` and from the tenant's own socket template registration. Tenant context flows through
  `tenant.MustFromContext(ctx)` on every hop, unchanged.
- **Version coverage**: the implementation carries **no** region or major-version literal. Selection is
  by item classification (530) on the channel side and by classification on the consumables side; the
  only version-dependent code touched is the pre-existing `UpdateTimeFirst` split, which is already
  IDA-verified per version. Consequently the feature is live on gms_48/61/72/79/83/84/87/92/95 and
  jms_185 wherever the tenant's WZ carries `Cash/0530` items, and is an inert no-op on gms_12 (handler
  not registered) and on any version whose WZ lacks the items.
- **Client wire values (DOM-25)**: no client-interpreted wire constant is introduced. The morph id is
  game data read from WZ, and the cash slot-item type byte is never written to the wire by the server.
- **Failure semantics**: a data-fetch failure, a compartment-consume failure, or a missing item MUST
  leave the coupon in the player's inventory. Effect-application failures after a successful consume are
  logged and not rolled back, matching the existing `ApplyItemEffects` convention
  (`processor.go:243-258`).
- **Observability**: the terminal warn at `character_cash_item_use.go:640` must no longer fire for
  classification 530. Failures on the new arm log at debug/error following the conventions of the
  neighbouring arms.
- **Testing/verification**: `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
  `atlas-data`, `atlas-consumables`, `atlas-channel`, and `libs/atlas-packet`; `tools/lint.sh --check`,
  `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/buff-duration-guard.sh` clean from
  the repo root. `docker buildx bake` is required only if a `go.mod` is touched, which is not expected.

## 9. Open Questions

1. **Which response clears the client's exclusive-request lock (FR-4.3).** Resolvable in the design
   phase by reading the client's unlock path and the packets the consume commit already emits; the
   fallback (an explicit `session.EnableActions` on success) is known-safe because the outcome does not
   warp. Not blocking.
2. **Whether the design keeps `ConsumeMorphCoupon`'s effect application inline or extracts a shared
   pure planner** with `computeEffectPlan`. Option (a) — a dedicated consumer with inline application —
   was chosen at spec time for the smaller diff; if design finds the duplication objectionable, a shared
   planner over an adapter type is an acceptable substitute provided FR-3.5..FR-3.8 still hold.

Resolved during the spec interview, recorded here so design does not relitigate:

- Effect application lives in `atlas-consumables`, not `atlas-channel` (keeps consume and effect in one
  transaction).
- Live re-ingest and REST verification of `5300000` is an **operational follow-up**, not an acceptance
  criterion.
- Re-use while morphed replaces the morph and restarts the timer.
- No version literals; classification gating only.
- The serverbound sub-body is empty (§4.1) — resolved by IDA before this PRD was written.

## 10. Acceptance Criteria

- [x] `atlas-data`'s cash reader parses `spec/morph` and `spec/hp`; unit tests pin morph 1/2/3, hp 50,
      time 600000 for `5300000`/`5300001`/`5300002` shaped fixtures (FR-2.3). —
      `services/atlas-data/atlas.com/data/cash/reader.go` + `TestReaderMorphCoupons`
      (`cash/reader_test.go:970`).
- [x] An existing non-0530 cash fixture (0521 EXP coupon or 0519 pet-skill pouch) produces byte-identical
      reader output before and after the change (FR-2.4). — `TestReaderMorphHpAdditiveOnly`
      (`cash/reader_test.go:1023`), asserting `SpecTypeMorph`/`SpecTypeHp` stay absent on a 0521 EXP
      coupon fixture.
- [x] The `atlas-consumables` cash REST model round-trips `morph` and `hp` (FR-3.1), pinned by a JSON
      test. — `TestRestModelSpecRoundTripsMorphKeys` (`consumables/cash/rest_test.go:14`) and
      `TestSpecTypeWireValues` (`:53`).
- [x] Using a `Cash/0530` coupon decrements exactly one coupon from the **Cash** compartment (not the Use
      compartment), pinned by a unit test asserting `inventory2.TypeValueCash` (FR-3.4). —
      `TestConsumeMorphCouponSuccess` (`consumable/morph_coupon_test.go:263`, assertion at `:282`).
- [x] Using a coupon issues a `TemporaryStatTypeMorph` statup with amount = the item's `morph` value,
      source = `-itemId`, duration = the item's `time` value in **milliseconds, unscaled** (FR-3.5,
      FR-3.6). — same test, `:296-308` (`sourceId`, `duration = 600000`, `statups[0]`).
- [x] Using a coupon issues an HP change of the item's `hp` value in the same operation (FR-3.5). —
      same test, `:290` (`hpChanges[0].amount == 50`).
- [x] A 530 item whose `morph` is absent/zero applies no morph but still consumes and still heals; a 530
      item whose `hp` is absent/zero applies no HP change but still morphs (FR-3.7). —
      `TestConsumeMorphCouponZeroSpecs` (`consumable/morph_coupon_test.go:358`).
- [x] Using a second coupon while morphed issues a second apply unconditionally — no rejection, no
      "already morphed" branch (FR-3.8). — `TestConsumeMorphCouponReuseWhileMorphedApplies`
      (`consumable/morph_coupon_test.go:397`).
- [x] A cash-data fetch failure leaves the coupon in inventory (reservation released via `ConsumeError`),
      pinned by a test (FR-3.3). — `TestConsumeMorphCouponCashFetchFailureKeepsCoupon`
      (`consumable/morph_coupon_test.go:319`).
- [x] Classification 530 no longer reaches the terminal warn at `character_cash_item_use.go:640`;
      a handler test asserts the arm is entered for `5300000` and **not** entered for a
      `ClassificationGachaponCoupon` id that shares type byte 40 on GMS >= 95 (corrected — this
      PRD originally said "pre-95"; source (`character_cash_item_use.go:933-938`) shows gachapon
      maps to 40 on GMS >= 95, colliding with transformation's pre-95 byte 40) (FR-1.3). —
      `TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume`
      (`character_cash_item_use_test.go:371`) for the entry case;
      `TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions` (`:456`, table rows at
      `:464-465`) for both collision non-entries, as corrected above.
- [x] The sub-body codec round-trips: nothing consumed when `UpdateTimeFirst(t)` is true, exactly one
      trailing `int32` consumed when it is false (FR-1.2). —
      `TestItemUseMorphCouponUpdateTimeFirstRoundTrip` and
      `TestItemUseMorphCouponNoUpdateTimeFirstRoundTrip`
      (`libs/atlas-packet/cash/serverbound/item_use_morph_coupon_test.go:12,28`).
- [x] The exclusive-request lock question (FR-4.3) is resolved with cited evidence — either "response X
      already clears it" with the file/IDA reference, or an explicit success-path unlock with a test —
      and is not left as a prose assumption. — design.md §1.2 plus the landed code comment at
      `character_cash_item_use.go:654-659`: the non-silent `INVENTORY_OPERATION` emitted by the
      consume commit already clears the client's exclusive-request lock
      (`CWvsContext::OnInventoryOperation @0xa1ead9`, gated on the packet's leading `bOnExclRequest`
      byte, IDA-verified), so no explicit unlock is emitted. Corroborated at task-8 time by the
      "no EnableActions" grep over `services libs` (Step 4 of the task-8 brief) finding zero real
      `EnableActions` calls added — only this design-decision comment and its counterpart at `:121`.
- [x] The diff contains no region or major-version literal in any of the three services (FR-1.3, §8), and
      no raw `530` numeric literal outside `libs/atlas-constants`. — task-8 Step 4 greps: `MajorVersion()`/
      `Region() ==` restricted to `services libs`: no hits; the `\b530\b` hits restricted to `services
      libs` are all inside comments or `GetClassification(...) == 530` test assertions, none a raw
      classification-selection literal.
- [x] No seed template is modified; gms_12's non-registration is recorded as a documented no-op. —
      `git diff --stat main...HEAD -- services/atlas-configurations/seed-data/templates/` is empty;
      the no-op is recorded in `docs/research/missing-features/items-and-consumables.md`
      (Present-but-partial #6) and `docs/TODO.md` (task-219 follow-up section).
- [x] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `atlas-data`, `atlas-consumables`,
      `atlas-channel`, `libs/atlas-packet`; `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
      `tools/goroutine-guard.sh`, `tools/buff-duration-guard.sh` clean from the repo root. — full sweep
      output in `task-8-report.md`; all clean except `tools/lint.sh --check`'s `ui:node-version` target,
      which fails for a known, pre-existing, unrelated environment reason (node v24 present, v22
      required) — all Go fmt/lint targets (86/86) report "0 issues.".
- [x] A follow-up item is filed for the operational re-ingest + live `GET /cash-items/5300000`
      verification on each tenant, referencing §6. — `docs/TODO.md`, "task-219 follow-up: cash WZ
      re-ingest for morph-coupon `spec/morph`/`spec/hp`".
