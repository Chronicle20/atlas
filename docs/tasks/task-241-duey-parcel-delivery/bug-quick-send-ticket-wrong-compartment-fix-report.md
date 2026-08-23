# Fix report — quick DUEY_ACTION SEND ticket lookup queried the wrong compartment

Task: task-241-duey-parcel-delivery
Brief: `docs/tasks/task-241-duey-parcel-delivery/bug-quick-send-ticket-wrong-compartment.md`, `## Fix` section

## What I implemented

`services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send.go`

- Extracted the `hasTicket` closure's compartment lookup into a new named
  function `hasQuickDeliveryTicket(l, ctx, characterId)`.
- `hasQuickDeliveryTicket` derives the compartment type via
  `inventory.TypeFromItemId(item.Id(item.QuickDeliveryTicketId))` instead of
  the hard-coded `inventory.TypeValueETC`. For item 5330000 this resolves to
  `TypeValueCash` (5), matching the live data in the bug report. Kept a
  defensive fallback to `TypeValueCash` if `TypeFromItemId`'s `ok` were ever
  false (it isn't for this constant — classification 533 always maps to
  `t=5` — but the fallback keeps the function total rather than silently
  passing a zero-value `Type`).
- Added a package-level var seam, `dueyQuickTicketCompartmentFetch`,
  wrapping `compartment.NewProcessor(l, ctx).GetByType(...)`, in the style
  of the existing `dueyCouponSagaCreateFunc` seam
  (`character_cash_item_use_duey.go:19`). This is what lets a test intercept
  and assert the actual `GetByType` argument.
- `handleDueyActionSend`'s `hasTicket` closure now just calls
  `hasQuickDeliveryTicket(l, ctx, characterId)`.
- Error and not-found behavior unchanged: on error or a miss, `sendParcel`
  still rejects with `PARCEL[INCORRECT_REQUEST]` and starts no saga.

`services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send_test.go`

- Added `TestHasQuickDeliveryTicketQueriesCashCompartment`, which swaps
  `dueyQuickTicketCompartmentFetch` for a spy, calls
  `hasQuickDeliveryTicket` directly, and asserts the captured `inventory.Type`
  argument equals `inventory.TypeValueCash` (5) and the `characterId` is
  passed through unchanged. This reaches the actual `GetByType` argument the
  production wiring sends — the pre-existing `TestDueyActionSend` table
  stubs `hasTicket` at the `dueySendDeps` boundary and could never have
  caught this bug (confirmed below).
- Added `"atlas-channel/compartment"` to the test file's imports (the
  formatter initially auto-resolved to the wrong same-named package,
  `atlas-channel/cashshop/inventory/compartment` — corrected by hand; see
  Self-review).

Did not touch `ReceivableAt`/24h logic (`dueyReceivableDelay`,
`buildParcelSendSaga`) or the client-lockup item — both out of scope per the
brief.

## TDD evidence

RED — with the fix's `it, ok := inventory.TypeFromItemId(...)` line
temporarily replaced by `it, ok := inventory.TypeValueETC, true` (i.e.
reproducing the original bug):

```
$ go test ./socket/handler/... -run 'TestHasQuickDeliveryTicket' -v
=== RUN   TestHasQuickDeliveryTicketQueriesCashCompartment
    duey_action_send_test.go:469: GetByType inventoryType = 4, want 5 (TypeValueCash)
--- FAIL: TestHasQuickDeliveryTicketQueriesCashCompartment (0.00s)
FAIL
```

This confirms the new test fails against the pre-fix code path (type 4 =
ETC, the bug) and would have caught the production defect — unlike the
pre-existing `TestDueyActionSend` table, whose `dueySendFixture.hasTicket`
stub is asserted only as a bool and never reaches `GetByType`.

GREEN — reverted to the fix (`inventory.TypeFromItemId(item.Id(item.QuickDeliveryTicketId))`):

```
$ go build ./... && go test ./socket/handler/... -run 'TestDueyActionSend|TestHasQuickDeliveryTicket' -v
=== RUN   TestDueyActionSend
... (13 subtests) ...
--- PASS: TestDueyActionSend (0.00s)
=== RUN   TestHasQuickDeliveryTicketQueriesCashCompartment
--- PASS: TestHasQuickDeliveryTicketQueriesCashCompartment (0.00s)
PASS
ok  	atlas-channel/socket/handler	0.015s
```

## Module-local verification

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Full module build and test suite pass; output pristine (all `ok`, no `FAIL`,
no stray warnings — full output captured during the session, tail shown
above in the GREEN run plus a full-suite pass with all packages `ok`).

## Files changed

- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send_test.go`

## Self-review findings

- The Go formatter/import-organizer (a `PostToolUse` hook) initially
  rewrote the test file's new import to `atlas-channel/cashshop/inventory/compartment`
  — a different, same-named `compartment` package used by the cash-shop
  module, not the one `duey_action_send.go` actually calls
  (`atlas-channel/compartment`, whose `Model`/`GetByType`/`FindFirstByItemId`
  are the ones in play here). Caught by inspecting the diff and confirmed
  by `grep`'ing both packages' `Model` definitions; corrected to
  `atlas-channel/compartment` before building. Worth flagging since an
  auto-import guess on an ambiguous package name silently compiles wrong
  code in some cases — here it happened to fail to compile since the two
  `compartment.Model` types are structurally different, so it was caught at
  `go build`, not silently.
- Verified `item.QuickDeliveryTicketId` is `uint32` and `inventory.TypeFromItemId`
  takes `item.Id` (`uint32` alias); the `item.Id(...)` conversion is correct
  and matches `TestQuickDeliveryTicketId` / `TestGetCashSlotItemTypeDueyCoupon`'s
  existing use of the same constant elsewhere in this package.
- Confirmed via the RED/GREEN cycle above that the new test genuinely
  exercises the argument the bug was about, not just a restated version of
  the stubbed-bool table.
- No other callers of the old `hasTicket` closure's compartment logic exist
  in this file; `grep` for `TypeValueETC` in this file found only the
  removed line.

## Issues or concerns

None. Scope matched the brief exactly: two files, no change to
`ReceivableAt`/24h behavior, no attempt at the client-lockup item (still
under `## Not yet answered`, controller's call).
