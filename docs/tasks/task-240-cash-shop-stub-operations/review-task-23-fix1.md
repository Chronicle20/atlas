# Review: task-23 fix round 1 (commit cda6eb58c)

## Scope

Commit `cda6eb58c` (range `8cfbc1200..cda6eb58c`) only. This commit touches a
single file:

```
services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go | 49 ++++++++++++++++++++++
1 file changed, 49 insertions(+)
```

Confirmed via `git diff 8cfbc1200..cda6eb58c --stat`. No production file is
touched by this commit.

Brief: `.superpowers/sdd/plan/task-23-brief-fix1.md`.
Originating finding: `docs/tasks/task-240-cash-shop-stub-operations/review-task-23.md` blocking f1.
Implementer report: `.superpowers/sdd/plan/task-23-report.md`, "Fix round 1".

## Findings

### PASS — the new test drives `handleStatusEventEquipSlotIncreased` directly

`TestEquipSlotIncreasedAnnouncesWireSlotIndexZeroNotTheCanonicalPosition`
(consumer_test.go, added lines) calls
`handleStatusEventEquipSlotIncreased(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[...]{...})`
directly — a real invocation of the production handler under test, not a call
into the packet body builder. This is the layer the original (inadequate)
test never reached.

### PASS — `SlotIndex` is seeded with the canonical `-59`, not `0`

The event fixture in the new test sets `SlotIndex: -59` (the field is typed
`int16` at `kafka/message/cashshop/kafka.go:404`, so `-59` is a valid,
representable canonical value — not silently truncated or coerced). Seeding
`0` would have made the assertion pass regardless of whether
`consumer.go:544` forwarded `e.Body.SlotIndex` or hardcoded `0`; seeding
`-59` makes the two behaviors diverge in the announced bytes, which is the
crux of the finding this commit closes.

### PASS — asserts the announced bytes via the real Decode path, and pins `Days` distinctly from slot index

`decodeEnableEquipSlotExtSuccess` (added helper) pulls the last announced
`CashShopOperationWriter` body and runs it through
`cashpkt.EnableEquipSlotExtSuccess{}.Decode(...)` — the same decoder that
round-trips `Encode` byte-for-byte
(`libs/atlas-packet/cash/clientbound/shop_operation_result_slots.go`:
`Encode` writes `mode, slotIndex(uint16), days(uint16)`; `Decode` reads the
same three fields in the same order). The test then asserts
`body.SlotIndex() == 0` and, separately, `body.Days() == 30` — a value
distinct from the slot-index field, so a swap between the two uint16 wire
positions could not slip through unnoticed. This is a genuine bytes-level
assertion on the actual wire encoding, not a re-derivation of the expected
value from the same code path being tested.

Verified the reasoning behind "a pass-through regression would encode
65477" independently, without mutating any code: `int16(-59)` reinterpreted
as `uint16` is `65477` (`-59 & 0xFFFF == 65477`), matching the doc comment
at `consumer.go:530-541` and the new test's failure message. Also ran the
new test in isolation against the current (correct) production code:

```
=== RUN   TestEquipSlotIncreasedAnnouncesWireSlotIndexZeroNotTheCanonicalPosition
--- PASS: TestEquipSlotIncreasedAnnouncesWireSlotIndexZeroNotTheCanonicalPosition (0.01s)
PASS
```

### PASS — production diff is genuinely empty

`git diff 8cfbc1200..cda6eb58c --stat` shows exactly one file touched:
`consumer_test.go` (+49/-0). `consumer.go:544` still reads
`cashpkt.CashShopEnableEquipSlotExtSuccessBody(0, e.Body.Days)` — the literal
`0`, not `e.Body.SlotIndex` — confirmed by direct read of the current file at
that line. No trace of the implementer's reported temporary flip-and-revert
experiment survived into this commit.

### PASS — pre-existing encoder-level test left intact

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_equip_slot_test.go`
does not appear in the commit's diff at all (only `consumer_test.go` is
touched), and `TestCashShopEnableEquipSlotExtSuccessBodyEncodes` is still
present there, unmodified. It continues to cover the encoder-byte-layout
concern at its own layer; the new consumer-level test is additive, not a
replacement.

## Not evaluable

None — the fix is fully within scope of this single test file and the
consumer/packet code it exercises, all of which was reviewed above.

## Verdict

APPROVED. The regression guard is real: it drives the actual handler, seeds
the canonical (non-zero, non-trivial) `-59` value, asserts on genuinely
decoded wire bytes, distinctly pins `Days`, leaves production code and the
pre-existing encoder test untouched, and the failure-mode arithmetic behind
the pinned assertion checks out independently.
