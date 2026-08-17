# Fix report: world-transfer coupon (5401000) not consumed on APPLY

## What was implemented

Followed the bug file's `## Fix` ruling exactly: moved coupon consumption
into `Resolve`, on the `APPLIED` branch, keyed by `m.Type()`, and deleted the
now-duplicated consume loop from `applyNameChange`. No ad-hoc consume was
bolted onto the world-transfer callback.

### `services/atlas-character/atlas.com/character/pending_change/producer.go`

- Added `worldTransferCouponTemplateIds = []uint32{5401000}` next to
  `nameChangeCouponTemplateIds`, with a grounding comment matching the style
  of the existing one (cites `derivation.md` §3's exact-id comparison across
  nine GMS versions, §1.5's jms_v185 mapping, and the existing use of 5401000
  at `atlas-channel character_cash_item_use.go:1112`).
- Added `couponTemplateIdsForType(changeType string) []uint32` — a small
  switch mapping `TypeNameChange` → `nameChangeCouponTemplateIds`,
  `TypeWorldTransfer` → `worldTransferCouponTemplateIds`, and any unknown
  type → `nil` (consumes nothing rather than guessing).
- Added `couponConsumptionStepId(changeType string) string` — returns
  `"consume_world_transfer_coupons"` for `TypeWorldTransfer` and
  `"consume_name_change_coupons"` otherwise, so the step id stays
  type-appropriate.
- `consumeCouponsCommandProvider` now derives its step id via
  `couponConsumptionStepId(m.Type())` instead of the hardcoded
  `"consume_name_change_coupons"` literal. `SetTransactionId` still uses
  `sagaTransactionId(m, sagaPurposeConsumeCoupon+":"+templateId)`, unchanged
  — that determinism is what makes a redelivered resolve a no-op instead of a
  second destroy, and it was already generic over template id (and therefore
  over `m` / its type), so no further change was needed there.
- Updated the doc comment on `consumeCouponsCommandProvider` to describe it
  as generic over change type rather than name-change-specific.

### `services/atlas-character/atlas.com/character/pending_change/processor.go`

- In `Resolve`, added the APPLIED-side branch immediately after the existing
  refund branch (`status != StatusApplied && m.HasAsset()`):

  ```go
  if status == StatusApplied {
      for _, templateId := range couponTemplateIdsForType(m.Type()) {
          if err := mb.Put(sagamsg.EnvCommandTopic, consumeCouponsCommandProvider(m, templateId)); err != nil {
              return Model{}, false, err
          }
      }
  }
  ```

  This sits inside the same `moved`-guarded block as the refund, on the same
  `*message.Buffer`/transaction, so a redelivered resolve emits nothing (the
  existing `if !moved { return ... }` guard above short-circuits first) and a
  failure to enqueue still aborts the transaction.
- Deleted the consume loop from `applyNameChange` (previously right before
  the `Resolve(buf)(m.Id(), StatusApplied, "")` call) and left a short comment
  in its place pointing at the new site in `Resolve`, preserving the original
  intent note (consumption happens at apply, not at request acceptance,
  because the purchase path materialises the coupon after the request).

### `services/atlas-character/atlas.com/character/pending_change/coupon_consumption_test.go`

Existing name-change tests (`TestApplyConsumesTheNameChangeCoupons`,
`TestRejectedApplyLeavesTheCouponsAlone`) were left untouched — they still
assert on the literal `consume_name_change_coupons` string, which is
unchanged for `TypeNameChange`, and both still pass.

Added the world-transfer counterpart, as directed:

- `TestApplyConsumesTheWorldTransferCoupon` — creates a `TypeWorldTransfer`
  pending change (via `p.CreateAndEmit(...)` with
  `withTransferEligibilityGates(passingGateDeps())` to satisfy the transfer
  eligibility gates without a live dependency), asserts no consume is emitted
  at creation time, then calls `p.ResolveAndEmit(m.Id(), StatusApplied, "")`
  directly (this is the real production call path — the world-transfer saga's
  terminal event drives `Resolve`, not `ApplyForCharacter`, which only starts
  the saga) and asserts exactly one `consume_world_transfer_coupons` /
  `destroy_all_assets` / `5401000` emission, and that
  `consume_name_change_coupons` never appears.
- `TestNonAppliedWorldTransferResolutionLeavesTheCouponAlone` — table test
  over `StatusRejected` and `StatusCancelled`, asserting zero
  `consume_world_transfer_coupons` emissions on either.

Used the project's existing Builder/helper pattern (`seedCharacter`,
`newProcessorTestDB`, `testLogger`, `testContext`,
`countOutboxMessagesMatching`, `passingGateDeps`,
`withTransferEligibilityGates`) already present in this package's test files
— no new `*_testhelpers.go` file was created.

## What was tested and the results

Module-local verification, per the brief and Contract 2, from
`services/atlas-character/atlas.com/character`:

```
$ go build ./... && go test ./pending_change/...
ok  	atlas-character/pending_change	166.724s
```

Then the full module (also module-local, same module root):

```
$ go build ./... && go test ./...
ok  	atlas-character
ok  	atlas-character/character
ok  	atlas-character/configuration
ok  	atlas-character/kafka/consumer/character
ok  	atlas-character/kafka/consumer/teleportrock
ok  	atlas-character/kafka/message/character
ok  	atlas-character/location
ok  	atlas-character/pending_change
ok  	atlas-character/session
ok  	atlas-character/session/history
ok  	atlas-character/skill
ok  	atlas-character/teleport_rock
[exited with code 0]
```

All green, output pristine (no `[no test files]` regressions beyond the
pre-existing ones for packages with no tests).

## Files changed

- `services/atlas-character/atlas.com/character/pending_change/producer.go`
- `services/atlas-character/atlas.com/character/pending_change/processor.go`
- `services/atlas-character/atlas.com/character/pending_change/coupon_consumption_test.go`

## Self-review findings

- Confirmed `nameChangeCouponTemplateIds` behaviour is byte-for-byte
  unchanged for `TypeNameChange` — same template id, same step id string —
  so the name-change path's observable behaviour did not regress (both
  pre-existing tests still pass unmodified).
- Confirmed the ordering property called out in the bug file's rationale
  holds: the consume emit and the `resolvedEventProvider` emit are both on
  the same `mb *message.Buffer` inside the same `ExecuteTransaction` /
  `outbox.EmitProvider` transaction as before, so a failed enqueue still
  aborts the whole write.
- Confirmed `TestAppliedResolutionEmitsNoRefund` and
  `TestPurchasePathResolutionEmitsNoAssetRefund` (pre-existing, in
  `refund_idempotency_test.go`) still pass with the new APPLIED-side
  consume branch present — they only assert on `award_asset`, which is
  unaffected.
- Left the "Not yet answered" items (refund-on-failed-transfer gap, the two
  stuck `compensating` sagas, the already-leaked live row in `aa3d`) alone,
  as instructed — none of them were touched.
- No `TODO`, stub, or placeholder introduced. No new domain type/constant
  needed beyond the coupon literal, which is grounded per the bug file, not
  invented.

## Issues or concerns

None. The fix follows the ruling in the bug file exactly, is scoped to the
four files/hunks it named, and both the module-local pending_change suite
and the full atlas-character module build/test cleanly.

## Status

DONE — commit `5ffaad846b7bac4189795dc2ede0617a6fea0cf5` on branch
`task-227-cash-name-change-world-transfer` in worktree
`.worktrees/task-227-cash-name-change-world-transfer` (repo root).

## Review follow-up (post-report)

Review of the original commit `e62ca6e99` came back
`APPROVED_WITH_FINDINGS` — 0 blocking, 1 non-blocking:

> `coupon_consumption_test.go` covers REJECTED and CANCELLED for
> WORLD_TRANSFER, but not StatusExpired. Low risk (the gate is a single
> `status == StatusApplied` check with no per-status branching) but cheap to
> pin.

### Fix applied

Extended the existing table in
`TestNonAppliedWorldTransferResolutionLeavesTheCouponAlone` (rather than
adding a new test function) to include `StatusExpired` alongside
`StatusRejected` and `StatusCancelled`, and updated its doc comment
accordingly. No new test infrastructure needed — the same
`ResolveAndEmit`/`countOutboxMessagesMatching` shape covers the new subtest.

### Verification

```
$ cd services/atlas-character/atlas.com/character
$ go build ./... && go test ./...
ok  	atlas-character	(cached)
ok  	atlas-character/character	(cached)
ok  	atlas-character/configuration	(cached)
ok  	atlas-character/kafka/consumer/character	(cached)
ok  	atlas-character/kafka/consumer/teleportrock	(cached)
ok  	atlas-character/kafka/message/character	(cached)
ok  	atlas-character/location	(cached)
ok  	atlas-character/pending_change	181.669s
ok  	atlas-character/session	(cached)
ok  	atlas-character/session/history	(cached)
ok  	atlas-character/skill	(cached)
ok  	atlas-character/teleport_rock	(cached)
[exit code 0]
```

All green, `atlas-character/pending_change` re-ran fresh (not cached) and
includes the new `StatusExpired` subtest via
`TestNonAppliedWorldTransferResolutionLeavesTheCouponAlone/EXPIRED`.

### Commit

The `coupon_consumption_test.go` change was amended into the existing commit
(not a second commit), per the coordinator's instruction — the fix stays one
commit. New SHA: `5ffaad846b7bac4189795dc2ede0617a6fea0cf5` (short:
`5ffaad8`). Branch and worktree confirmed unchanged
(`task-227-cash-name-change-world-transfer`,
`.worktrees/task-227-cash-name-change-world-transfer`).
