# Review: Task 3 — Failure routing (`ErrorEventBody.Operation`)

Range: `66e9c0f26..3a0ac66df` (1 commit, `3a0ac66df`)
Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Scope

`git diff --stat 66e9c0f26..3a0ac66df` — 5 files, 155 insertions / 2 deletions:

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` (+16)
- `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` (+19)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` (+34/-2)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_failure_routing_test.go` (+72, new)
- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` (+16)

Matches the brief's file list exactly. No files touched outside the declared surface.

## 1. Interfaces produced — checked character for character against the brief

Both `kafka/message/cashshop/kafka.go` copies (atlas-cashshop and atlas-channel) add, verbatim:

```go
const (
	ErrorOperationGift            = "GIFT"
	ErrorOperationBuyNormal       = "BUY_NORMAL"
	ErrorOperationRebate          = "REBATE"
	ErrorOperationCouple          = "COUPLE"
	ErrorOperationFriendship      = "FRIENDSHIP"
	ErrorOperationBuyPackage      = "BUY_PACKAGE"
	ErrorOperationGiftPackage     = "GIFT_PACKAGE"
	ErrorOperationEnableEquipSlot = "ENABLE_EQUIP_SLOT"
)
```

Confirmed identical in both files (`git diff` hunks for both are byte-for-byte the same constant block). PASS.

`services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go:31`:

```go
func ErrorStatusEventForOperationProvider(characterId uint32, operation string, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message]
```

Matches the brief's **Interfaces produced** signature exactly (parameter names, order, and return type). PASS.

## 2. Backward compatibility — the load-bearing property

`failureBodyForOperation`'s `default` arm (`services/atlas-channel/.../consumer.go:311`) returns `cashpkt.CashShopInventoryCapacityIncreaseFailedBody(reason)` — the exact builder `handleStatusEventError` called unconditionally before this change.

`ErrorStatusEventProvider` (`services/atlas-cashshop/.../producer.go:13-24`) is untouched; `ErrorStatusEventForOperationProvider` is a new, separate function added after it (diff shows only an insertion, no modification to the original function body).

The first two subtests in `consumer_failure_routing_test.go`:

```go
{"empty operation keeps today's arm", "", "INVENTORY_FULL", 110, 25},
{"unknown operation keeps today's arm", "SOMETHING_ELSE", "INVENTORY_FULL", 110, 25},
```

both assert mode byte 110 (the `CashShopOperationInventoryCapacityIncreaseFailed` value from the options map) and error byte 25. Ran and confirmed PASS (see §6). PASS.

## 3. `ErrorEventBody` mirrors across the seam

Extracted the `ErrorEventBody` struct block from both files and diffed them directly:

```
diff <(sed -n '/type ErrorEventBody struct/,/^}/p' services/atlas-cashshop/.../kafka.go) \
     <(sed -n '/type ErrorEventBody struct/,/^}/p' services/atlas-channel/.../kafka.go)
```

Output: empty (exit 0) — the two struct definitions, including the new `Operation string \`json:"operation,omitempty"\`` field, comment, and constant block, are byte-identical. PASS.

## 4. No literal mode bytes in production code

Grepped `consumer.go` for the ten numeric literals from the brief's options table (110, 108, 159, 151, 153, 163, 155, 157, 118) — zero matches in the handler or the new `failureBodyForOperation` switch. All literals live only in `consumer_failure_routing_test.go`'s `options` map and assertions. `failureBodyForOperation` dispatches purely on the `ErrorOperation*` string constants, and each arm delegates to an existing `cashcb.CashShop*FailedBody` constructor that resolves its own mode via `atlas_packet.ResolveCode`/`WithResolvedCode` (confirmed all nine constructor names — `CashShopGiftFailedBody`, `CashShopBuyNormalFailedBody`, `CashShopRebateFailedBody`, `CashShopCoupleFailedBody`, `CashShopFriendshipFailedBody`, `CashShopBuyPackageFailedBody`, `CashShopGiftPackageFailedBody`, `CashShopEnableEquipSlotExtFailedBody`, `CashShopInventoryCapacityIncreaseFailedBody` — exist in `libs/atlas-packet/cash/clientbound/shop_operation_body.go` with the `packet.Encode` signature). PASS.

## 5. Ordering inside `handleStatusEventError`

Read the full function body (`consumer.go:322-361`). Structure, unchanged from before except the final call site:

1. Tenant guard.
2. `resolvePendingChange` branch — on match, cancels the pending record and returns early on its own arm (`TypeNameChange` → pink text; `TypeWorldTransfer` → `CashShopTransferWorldFailedBody`). Both `return` inside the branch, so a match never falls through.
3. New: `failureBodyForOperation(e.Body.Operation, e.Body.Error)` call, unconditionally reached only when step 2 did not return.

This is the exact ordering the brief requires — the new routing call is a strict continuation of the existing fallthrough path, not an insertion before or a subsumption of the pending-change branch. Confirmed empirically: `TestErrorWithPendingNameChangeCancelsRecordAndAnswersPinkText` and `TestErrorWithPendingWorldTransferCancelsRecordAndAnswersFailedArm` (pre-existing tests, not modified by this diff) both still PASS, proving priority was preserved. PASS.

## 6. Test quality

Read `consumer_failure_routing_test.go` in full. All ten subtests from the brief's table are present with the exact operation/reason/expected-byte values. Each subtest:

- Computes `body := encode(...)(options)` from a shared, independently-defined `options` map (not derived from the input under test).
- Asserts `len(body) == 2` first (`t.Fatalf` before checking bytes).
- Asserts `body[0]` (mode) against a literal expected value from the brief's table.
- Asserts `body[1]` (error code) against a literal expected value from the brief's table.

None of the ten is vacuous — each asserts a value that was not computed by the code under test, and would fail under a broken `failureBodyForOperation` (e.g. a mis-wired `switch` arm, a swapped constant, or a default-arm regression). No `*_testhelpers.go` file was created; the test uses table-driven cases and a pure function call, consistent with the Builder-pattern constraint (no builder needed here — nothing to build). PASS.

## 7. Module-local suites (run directly, not taken from the report)

### `services/atlas-channel/atlas.com/channel`

```
$ go build ./... && go test ./...
```
All packages `ok` or `[no test files]`; zero `FAIL` lines.

```
$ go test ./kafka/consumer/cashshop/... -v
```
```
=== RUN   TestFailureBodyForOperation
=== RUN   TestFailureBodyForOperation/empty_operation_keeps_today's_arm
=== RUN   TestFailureBodyForOperation/unknown_operation_keeps_today's_arm
=== RUN   TestFailureBodyForOperation/gift
=== RUN   TestFailureBodyForOperation/buy_normal
=== RUN   TestFailureBodyForOperation/rebate
=== RUN   TestFailureBodyForOperation/couple
=== RUN   TestFailureBodyForOperation/friendship
=== RUN   TestFailureBodyForOperation/buy_package
=== RUN   TestFailureBodyForOperation/gift_package
=== RUN   TestFailureBodyForOperation/enable_equip_slot
--- PASS: TestFailureBodyForOperation (0.00s)
... (all other cashshop consumer tests, including the three pending-change tests)
PASS
ok  	atlas-channel/kafka/consumer/cashshop	(cached)
```

### `services/atlas-cashshop/atlas.com/cashshop`

```
$ go build ./... && go test ./...
```
All packages `ok` or `[no test files]`, including `ok atlas-cashshop/kafka/message/cashshop 0.008s` and `ok atlas-cashshop/kafka/consumer/cashshop 0.448s`; zero `FAIL` lines.

## Findings

No blocking findings. No non-blocking findings — every constraint in the brief (interface names, backward compatibility, ordering, no-literal-bytes, mirror-struct, test quality) was independently verified against the diff, not assumed from the report.

## Not evaluable

None — the full review surface (the 5 changed files plus the read-only `shop_operation_body.go` contract they depend on) was reviewed directly.
