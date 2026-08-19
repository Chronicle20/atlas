# Review: Task 22 — Quick Delivery Ticket (classification 533)

Commit range: `1d09cda4c..98173c1c4` (single commit `98173c1c4`)
Brief: `.superpowers/sdd/plan/task-22-brief.md`
Report: `.superpowers/sdd/plan/task-22-report.md`

## Scope

`git diff --stat 1d09cda4c..98173c1c4` (excluding the unrelated
review/ledger docs that landed in the same commit range from prior tasks):

- `libs/atlas-constants/item/constants_test.go` — new `TestQuickDeliveryTicketId`
- `libs/atlas-constants/item/duey.go` — doc comment only, no logic change
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go` — `quickDeliveryEnabled` JMS clause
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer_test.go` — JMS subtest + helper
- `services/atlas-channel/atlas.com/channel/saga/model.go` — re-export `ShowParcel`/`ShowParcelPayload`
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — dispatch branch, region-aware `GetCashSlotItemType`, chalkboard classification guard
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go` — new handler
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey_test.go` — new tests

`scope_confirmed`: matches the brief's file inventory plus Extra Required Work
items A/B/C from the controller addendum. No unrelated edits found.

## Findings

### 1. Chalkboard collision fix — logic verified correct, but untested (BLOCKING)

Traced every `it == CashSlotItemType(32)` producer in
`character_cash_item_use.go` between the chalkboard check (line 101) and the
classification-first dispatch block (line 791):

- `CashSlotItemTypeChalkboard = CashSlotItemType(32)` (`:976`), returned
  unconditionally by the `ClassificationChalkboard` (537) arm of
  `GetCashSlotItemType` (`:1433`).
- The new Duey/JMS arm at `:1428` (`ClassificationDueyCoupon` (533), JMS only)
  is the only other producer of 32.

No other `it ==` comparison in the file's linear dispatch chain equals 32, so
the new guard `if item.GetClassification(itemId) == item.ClassificationChalkboard { ...; return }`
(falling through otherwise) is logically sound: a genuine chalkboard
(classification 537) still enters `chalkboard.NewProcessor(...).AttemptUse`
on every region/version — the guard only ever excludes items whose
classification is *not* 537, and 32 has no third producer. A JMS Quick
Delivery Ticket (classification 533) correctly falls through to the
classification-first dispatch and reaches `handleDueyCouponUse`.

However: **no test pins the chalkboard side of this collision.** The Duey
side is covered (`TestHandleDueyCouponUse/jms_emits_show_parcel_quick`,
`character_cash_item_use_duey_test.go:82-119`, which implicitly proves the
ticket falls through past the chalkboard guard). But there is zero test
anywhere in `socket/handler/` — before or after this commit — that exercises
a genuine chalkboard item and asserts `chalkboard.AttemptUse` is still
invoked. `grep -rln "Chalkboard" services/atlas-channel/atlas.com/channel/socket/handler/` finds only
`character_cash_item_use.go` and the unrelated `chalkboard_close.go`; no
`_test.go` file mentions it.

This matters because the collision fix is exactly the kind of change the
report itself calls "not in the brief" and self-authored under TDD pressure
to make the JMS test pass — the established repo precedent for this same
pattern (the teleport-rock/megaphone type-12 collision) *does* carry a test
for the non-primary side
(`TestCharacterCashItemUseHandleFunc_MegaphoneEnum12NotInvoked`,
`character_cash_item_use_test.go:171-201`, asserting the megaphone alias does
NOT invoke `useRockFunc`). The chalkboard fix has no analogous "still invokes
AttemptUse for a real chalkboard" test. My manual trace found no bug, but a
future edit to `GetClassification` or the chalkboard classification table
would regress this silently — no test would fail. Given the review brief's
explicit framing ("a regression here would silently break an existing
shipped feature"), the absence of that test is a blocking gap, not merely a
style nit — the collision fix is the one piece of scope the implementer
introduced beyond the brief's literal text, and it ships with no regression
coverage on the higher-blast-radius side (chalkboard) of the two.

**Location**: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:101-112`
(the fix); missing test file location would be
`character_cash_item_use_test.go` or a new chalkboard-specific test file.

### 2. JMS gate condition — identical in both packages, doc comments rewritten (PASS)

- `dueyCouponEnabled` (`character_cash_item_use_duey.go:41-44`):
  `(t.IsRegion("GMS") && t.MajorAtLeast(72)) || (t.IsRegion("JMS") && t.MajorAtLeast(185))`
- `quickDeliveryEnabled` (`kafka/consumer/parcel/consumer.go:178-181`): identical
  expression, byte for byte.

`quickDeliveryEnabled`'s doc comment (`consumer.go:153-177`) was fully
rewritten: it no longer describes Task 22 as not-yet-landed or the JMS clause
as a future sync point. It states the two functions are "the same condition
as `socket/handler/character_cash_item_use_duey.go`'s `dueyCouponEnabled`, in
a different package" and carries the identical IDA citation (address, the
verbatim jump-table annotation, and the `parcel.yaml` dispatcher-slot
reference) required by the controller addendum ("Do NOT paraphrase the
annotation; quote it" — both files quote it verbatim, character-for-character
identical between the two doc comments).

`consumer_test.go:207-229` (`open with mailbox jms quick enabled`) exercises
a JMS v185 tenant through `newJmsQuickEnabledTenant` (`:58-67`) and asserts
`quickEnabled == true` in the non-quick `ParcelOpenBody`. This is a real
RED→GREEN test: before this commit `quickDeliveryEnabled` had no JMS clause,
so a JMS v185 tenant would have produced `quickEnabled == false` and the
assertion `if !quickEnabled { t.Error(...) }` would have failed.

### 3. Handler consumes nothing — FR-26 (PASS)

`TestHandleDueyCouponUse/consumes_nothing`
(`character_cash_item_use_duey_test.go:121-142`) asserts a single saga is
created and iterates its steps checking none has
`Action == saga.DestroyAssetFromSlot || Action == saga.DestroyAsset`. This is
a genuine absence assertion, not merely "a step exists." The handler itself
(`character_cash_item_use_duey.go:96-115`) builds exactly one step
(`ShowParcel`), confirming the assertion is meaningful and not vacuously
true because the saga is empty (`len(sg.Steps) != 1` is checked in the
sibling subtests, establishing the saga is non-trivial).

### 4. `item.QuickDeliveryTicketId` duplicate revert — clean (PASS)

`git diff 1d09cda4c..98173c1c4 -- libs/atlas-constants/item/constants.go` is
empty — no stray edit landed there. `duey.go`'s only change is the doc
comment (Task 22 landed vs. pending), the constant declaration itself is
untouched. `constants_test.go` gained one new test
(`TestQuickDeliveryTicketId`, `:106-113`) and no other change. No leftover
duplicate declaration found anywhere in `libs/atlas-constants`.

### 5. `SagaType: saga.InventoryTransaction` for a non-inventory, single-display step

`handleDueyCouponUse` (`character_cash_item_use_duey.go:88`) sets
`SagaType: saga.InventoryTransaction` for a saga whose only step is
`ShowParcel` — a UI-announce action, not an inventory operation. The report
states this matches "the npc-conversations service's own precedent for
single 'show' steps." I did not verify this precedent's file
(`operation_executor.go` in a sibling service) because it is outside this
unit's diff and the brief does not specify a `SagaType` value for this saga —
this is a genuine gap between what I can verify from the diff and what the
report claims. Not evaluable from this unit's scope alone; flagging as a
non-blocking note since the saga type field does not appear to be consumed
by any logic this commit touches (the saga's `Steps` and their `Action` are
what atlas-saga's dispatcher keys off).

## Not evaluable

- The `operation_executor.go:createSagaForOperation` precedent cited in the
  report for reusing `saga.InventoryTransaction` on a non-inventory saga —
  that file lives in the npc-conversations service, outside this diff's
  scope, and I did not read it.
- Whether the atlas-saga shared library's `ShowParcel` action/`ShowParcelPayload`
  type actually exist with the shape re-exported in `saga/model.go` (i.e.
  whether `sharedsaga.ShowParcel`/`sharedsaga.ShowParcelPayload` are Task 19's
  real types) — this is Task 19's contract; the code compiles per the
  report's `go build` evidence, which is sufficient grounding for this
  reviewer's purposes, but I did not independently re-derive the library's
  contract from source.

## Test value provenance

- Item id `5330000` (`QuickDeliveryTicketId`) — traces to Task 17's
  pre-existing constant, unchanged by this commit.
- Cash-slot type bytes 31 (GMS) and 32 (JMS) — 31 traces to the existing,
  unchanged `ClassificationDueyCoupon` arm; 32 traces to the controller
  addendum's quoted IDA evidence (`get_cashslot_item_type @0x49a1ee`,
  `case 533: return 32;`), reproduced verbatim in both doc comments.
- Tenant rows GMS v83 / GMS v61 / JMS v185 — v83 and v61 match the existing
  `remoteMerchantEnabled` precedent's span (72 floor, 61 below floor); JMS
  185 matches the addendum's `MajorAtLeast(185)` floor exactly.
- `docs/packets/dispatchers/parcel.yaml:67,85` (`jms_v185: 10` for OPEN,
  `jms_v185: 27` for OPEN_QUICK) — this file is outside the diff; I did not
  re-verify these line numbers against the current file, only that the doc
  comments cite them consistently with the addendum's text.

## Verdict rationale

Everything the review focus called out was independently verified except
one: the chalkboard side of the type-32 collision is logically correct by
manual trace of every `it == 32` producer in the file, but has zero test
coverage (neither pre-existing nor added by this commit), which is a
material gap for a fix that touches an existing shipped feature's dispatch
path.
