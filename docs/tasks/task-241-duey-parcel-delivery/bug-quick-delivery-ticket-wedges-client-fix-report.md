# Fix report — Quick Delivery Ticket wedges the client's exclusive-request lock

Task: task-241-duey-parcel-delivery

## What I implemented

`handleDueyCouponUse` in
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go`
now calls the existing `enableActions()` closure on the success path — after
`dueyCouponSagaCreateFunc` returns without error and the "Quick delivery
dialog open requested." log line is emitted — releasing the client's
`m_bExclRequestSent` lock. `enableActions()` was already defined at the top
of the handler and used on the three failure paths (unsupported version,
saga-create error); the success path was the only branch that returned
without calling it.

Also updated the handler's doc comment: the "consumes nothing" paragraph now
states that precisely because the ticket is not consumed here, no
INVENTORY_OPERATION follows, and PARCEL[OPEN_QUICK] itself is neither
STAT_CHANGED, INVENTORY_OPERATION, nor SET_FIELD — so the handler must call
`session.EnableActions` itself or the lock is never released.

## Regression test

Added `t.Run("unlocks the client's exclusive-request lock", ...)` to
`TestHandleDueyCouponUse` in
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey_test.go`.
It uses the `gaugeProducerRecorder` seam already established in this package
(`character_damage_test.go`, and used the same way by
`character_cash_item_use_expiration_extender_test.go`'s enable-actions
assertion) to capture the producer call:

- asserts exactly one announce (`rec.calls == 1`)
- asserts the writer name is `statpkt.StatChangedWriter`
- decodes the plaintext body's leading byte (exclRequestSent bool) == `1`
  and the following 4-byte updateMask == `0`, confirming an empty `Update`
  set — matching `session.EnableActions`'s own encode
  (`NewStatChanged(make([]statpkt.Update, 0), true)`).

## Fallout from wiring in a real producer

Because the success path now calls `wp(...)`, the two existing
`TestHandleDueyCouponUse` subtests ("emits show_parcel quick", "jms emits
show_parcel quick") and the "consumes nothing" subtest previously called
`CharacterCashItemUseHandleFunc(..., nil)`. A nil `writer.Producer` panics
the first time it's invoked. I swapped all four to
`rec := &gaugeProducerRecorder{}` / `rec.producer()` (the package's
established seam — no new mocking style introduced), preserving every
existing assertion.

The same nil-producer fallout hit
`character_cash_item_use_chalkboard_test.go`'s `"JMS Quick Delivery Ticket
does not invoke AttemptUse"` subtest (it drives the coupon success path to
prove the JMS chalkboard/coupon cash-slot-type collision resolves to the
coupon branch, not chalkboard). Fixed the same way.

## Tested

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./socket/handler/... -run 'Duey|CashItemUse' -v
go test ./...
```

- `go build ./...` — clean, no output.
- `go test ./socket/handler/... -run 'Duey|CashItemUse' -v` — all
  `TestHandleDueyCouponUse` subtests pass, including the new
  `unlocks_the_client's_exclusive-request_lock`; all `TestCharacterCashItemUseHandleFunc_*`,
  `TestDueyActionReceive`, `TestDueyActionDiscard`, `TestDueyActionClose`,
  `TestDueyActionSend` subtests pass unchanged (`duey_action_send.go`/
  `duey_action_send_test.go` untouched, per the brief).
- `go test ./...` (full module) — all packages `ok`, including
  `atlas-channel/socket/handler` (this run also caught and fixed the
  chalkboard-test nil-producer panic above).

## Files changed

- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go`
  — success path now calls `enableActions()`; doc comment updated.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey_test.go`
  — three existing subtests switched from a nil producer to
  `gaugeProducerRecorder`; new regression subtest added; `statpkt` import
  added.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_chalkboard_test.go`
  — one subtest switched from a nil producer to `gaugeProducerRecorder` for
  the same reason.

`duey_action_send.go` was not touched, per the brief (already fixed in
`7c09672dd`).

## Self-review

- Scope held to the handler named in the brief; the ReceivableAt/24h logic
  (FR-12) and the ticket-compartment gate (`7c09672dd`) were not revisited.
- No new mocking style introduced — reused `gaugeProducerRecorder`, already
  established for exactly this "assert on the enable-actions announce"
  pattern in three sibling test files.
- Fixed the chalkboard test collateral rather than leaving the module
  build/test red — that fix is a one-line producer swap, not a scope change
  to that test's assertions.
- Considered whether `PARCEL[OPEN_QUICK]` itself might warp the field (the
  doc's converse guard: don't double-unlock a field-changing outcome). It
  does not — no field change accompanies this response, so the unlock is
  correct here, matching the brief's own reasoning.

## Issues or concerns

None. The "Not yet answered" section of the bug report (possible second lock
on the `PARCEL[INCORRECT_REQUEST]` send-reject path) is explicitly deferred
to re-testing after this fix per the brief; I did not investigate it further
since it is out of this task's scope.

## Resolution

Fixed. `handleDueyCouponUse`'s success path now announces
`session.EnableActions`, matching every other path in the handler and the
`session.EnableActions` doc contract.
