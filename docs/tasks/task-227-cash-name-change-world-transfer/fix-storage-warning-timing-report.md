# Fix report: move storage-stranding warning to BUY time (step 1 of the suggested fix order)

Scope: Ruling 1 only, from `bug-world-transfer-eligibility-reasons.md`. Steps
2-5 (inFamily fixes, check_unavailable, CHECK-time rejection split, BUY-time
errors alias table) are explicitly out of scope for this task and untouched.

## What was implemented

`warnIfStrandingStorage` (the FR-4.7 courtesy notice) no longer fires from
`CashShopCheckTransferWorldPossibleHandleFunc`
(`cash_shop_check_transfer_world_possible.go`). It now fires from
`handleBuyWorldTransfer` (`cash_shop_operation.go`), on the rejection-free
(successful) path, after the `RequestPurchase` call and before the function
returns.

- The function itself moved from the CHECK file to the operation file
  (same package `handler`, so no export/rename was needed).
- The `checkPossibleAccountCharactersInWorldFunc` seam stayed in
  `cash_shop_check_transfer_world_possible.go` (that file already owns the
  sibling account/PIC seams) — its doc comment was updated to say it now
  feeds the BUY-time warning, not the CHECK handler.
- POP_UP is still the message mode; nothing about the PINK_TEXT-is-a-silent-
  no-op reasoning changed, only *when* the POP_UP fires.
- The stranding predicate is unchanged: `isLast := len(chars) == 1 &&
  chars[0].Id() == characterId`, keyed on `s.WorldId()` — the SOURCE world.
  At BUY time a destination world is known (`sp.TargetWorld()`), but nothing
  about "stranding" now considers it; the predicate still asks only whether
  the character is the account's last one in the world it is leaving.
- Fail-open is unchanged: a lookup error is logged (`Unable to determine
  whether world transfer strands storage...`) and swallowed; it never
  affects the purchase.
- Both the CHECK handler's doc comment (inline, at the removed call site) and
  `handleBuyWorldTransfer`'s doc comment were updated to cite the bug file
  and explain the modal-collision reasoning for the move.

## Tests updated / added

`cash_shop_check_transfer_world_possible_test.go`:
- Removed `TestWorldTransferCheckWarnsWhenStrandingStorage`,
  `TestWorldTransferCheckNoWarningWhenAnotherCharacterRemains`,
  `TestWorldTransferCheckLookupErrorFailsOpen`, and
  `TestWorldTransferStorageWarningUsesPopUpNotPinkText` — all four pinned the
  bug (CHECK-time emission).
- Added `TestWorldTransferCheckNeverEmitsStorageWarning`, with three
  subtests (would-be-stranded character, another character remains,
  lookup-error) asserting the CHECK handler answers ALLOWED and never writes
  a `WORLD_MESSAGE` in any of the three scenarios that used to trigger the
  warning.

`cash_shop_operation_imprint_test.go`:
- Added `installCharactersInWorldSeam` (swaps
  `checkPossibleAccountCharactersInWorldFunc` for a test, returns a restore
  func) and `worldMessageBuyRecorder` (a `writer.Producer` fake that
  resolves `chatpkt.WorldMessageWriter`'s "operations" table the same way
  `checkPossibleWriterOptions` does, so the POP_UP mode byte can be
  asserted, plus `storageWarningWasAnnounced` / `storageWarningModeByte`
  helpers).
- `TestBuyWorldTransferCreatesAPendingRequest` now pins the
  characters-in-world seam to `(nil, nil)` (never stranding) with a comment
  explaining why — that test is about the pending-record/purchase-command
  contract, not the warning, and its assertion (d) ("0 packets announced
  directly") needed to stay meaningful independent of the new warning path.
- Added `TestBuyWorldTransferWarnsWhenStrandingStorage` — asserts the
  warning IS written, as POP_UP (mode byte `0x01`), on the successful BUY
  path.
- Added `TestBuyWorldTransferNoWarningWhenAnotherCharacterRemains` — no
  warning when a sibling character remains in the source world.
- Added `TestBuyWorldTransferStorageWarningLookupFailsOpen` — a lookup
  error suppresses the warning but does not block the `REQUEST_PURCHASE`
  command (asserted via the existing `installCapturingProducer` kafka
  capture).
- `TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid` needed no
  change: that path returns before the warning line is ever reached
  (transaction-id parse failure), so it was never affected by the move.

## Verification (module-local, per Contract 2)

```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```

`go build ./...` — clean, no output.

`go test ./...` — all packages `ok`, including
`ok atlas-channel/socket/handler 0.964s`. No skipped tests, no flag-adjusted
run.

Targeted re-run of the touched tests (verbose):

```
go test ./socket/handler/... -run 'TestBuyWorldTransfer|TestWorldTransferCheckNeverEmitsStorageWarning' -v
```

```
=== RUN   TestWorldTransferCheckNeverEmitsStorageWarning
=== RUN   TestWorldTransferCheckNeverEmitsStorageWarning/would-be-stranded_character
=== RUN   TestWorldTransferCheckNeverEmitsStorageWarning/another_character_remains
=== RUN   TestWorldTransferCheckNeverEmitsStorageWarning/last-character_lookup_errors
--- PASS: TestWorldTransferCheckNeverEmitsStorageWarning (0.00s)
=== RUN   TestBuyWorldTransferCreatesAPendingRequest
--- PASS: TestBuyWorldTransferCreatesAPendingRequest (0.01s)
=== RUN   TestBuyWorldTransferWarnsWhenStrandingStorage
--- PASS: TestBuyWorldTransferWarnsWhenStrandingStorage (0.00s)
=== RUN   TestBuyWorldTransferNoWarningWhenAnotherCharacterRemains
--- PASS: TestBuyWorldTransferNoWarningWhenAnotherCharacterRemains (0.00s)
=== RUN   TestBuyWorldTransferStorageWarningLookupFailsOpen
--- PASS: TestBuyWorldTransferStorageWarningLookupFailsOpen (0.00s)
=== RUN   TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid
--- PASS: TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid (0.00s)
PASS
ok  	atlas-channel/socket/handler	0.035s
```

`go vet ./socket/handler/...` — clean, no output.

## Files changed

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible_test.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation_imprint_test.go`

Commit: `c642cda59` — "fix(task-227): move storage-stranding warning from
CHECK to BUY_WORLD_TRANSFER"

## Self-review

- Confirmed no other caller references `warnIfStrandingStorage` from the old
  location (it's package-private, same package, single call site moved).
- Confirmed `chatpkt`/`writer` imports were cleanly removed from the CHECK
  file (goimports hook did this automatically) and cleanly present in the
  operation file (already imported for `announceCashShopRejection`).
- Confirmed the stranding predicate's semantics were not touched — same
  `s.WorldId()`, same `len(chars)==1 && chars[0].Id()==characterId` check,
  same fail-open error log text/behavior.
- Confirmed `TestBuyWorldTransferAbortsPurchaseWhenTransactionIdInvalid` and
  `TestBuyNameChangeAbortsPurchaseWhenTransactionIdInvalid` were unaffected
  (they return before the new warning call).
- Did not touch `deploy/`, `atlas-character`, or `atlas-configurations`
  files that appear modified in `git status` — those are other agents'
  in-flight work on steps 2-5 of the same worktree; left untouched and
  unstaged, only the four files above were `git add`ed and committed.

## Concerns

None. The change is a single, well-scoped move plus a doc-comment/test
update; fail-open and POP_UP-mode contracts are unchanged and covered by
tests.
