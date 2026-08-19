# Review: Task 17 — BUY_PACKAGE / BUY_OTHER_PACKAGE (atlas-channel)

Commit range: `89e0eb4..96bc6ea` (`96bc6ea27 feat(channel): implement BUY_PACKAGE and route BUY_OTHER_PACKAGE`)

Brief: `.superpowers/sdd/plan/task-17-brief.md`
Report: `.superpowers/sdd/plan/task-17-report.md` (implementer status: DONE, with a flagged
design-judgment concern)

## Method

`git diff --stat 89e0eb4..96bc6ea`, then full reads of every changed file. Independent mutation
testing on three claims (byte-swap on the result-body test, field-order on the new codec, and
dispatcher-arm removal), each with a restore-and-verify-clean-diff cycle.

## 1. `resolvePurchaseCurrency` routing — semantically identical to the ruling's intent

PASS. `cashshop/processor.go:121-127`:

```go
func resolvePurchaseCurrency(isPoints bool, currency uint32) uint32 {
	const walletCurrencyMaplePoints = uint32(2)
	if isPoints && currency == 0 {
		return walletCurrencyMaplePoints
	}
	return currency
}
```

- `handleBuyPackage` → `RequestPackagePurchase(..., sp.PointType(), 0, ...)` →
  `processor.go:268` calls `resolvePurchaseCurrency(sp.PointType(), 0)` — the literal call the
  brief specified, now reachable because it moved server-side of the package boundary rather
  than being called directly from the handler package (which was impossible, as the implementer
  correctly diagnosed: `resolvePurchaseCurrency` is unexported in `cashshop`).
- `handleBuyOtherPackage` → `RequestPackagePurchase(..., false, walletCurrencyPrepaid, ...)` →
  `resolvePurchaseCurrency(false, 3)`. The only branch requires `isPoints == true`; with
  `isPoints = false` the branch can never fire, so this returns `3` unchanged — a provable no-op
  passthrough, confirmed by reading the function body (not asserted from memory).
- No path exists that could route a raw client `option` (1/2/4) into this call —
  `ShopOperationBuyOtherPackage` (the new codec) has no `option` field at all (`grep` of the new
  codec file confirms only `spw/serialNumber/name/message`), and `handleBuyOtherPackage`'s call
  site hardcodes `false, walletCurrencyPrepaid` literally (`cash_shop_package.go:151`).

This is a reasonable resolution of the brief's uncallable literal instruction, and the
implementer's report accurately describes the design choice and its tradeoff rather than
asserting it away. Agreed with the implementer's own self-assessment.

## 2. Currency 3 reaches the wire, unmapped

PASS. Traced end to end:

- `handleBuyOtherPackage` → `RequestPackagePurchase(..., walletCurrencyPrepaid=3, ...)`
  (`cash_shop_package.go:151`)
- `processor.go:270`: `currency = resolvePurchaseCurrency(false, 3)` → `3` (see above)
- `processor.go:271` → `RequestPackagePurchaseCommandProvider(..., currency=3, ...)`
  (`cashshop/producer.go:186-200`) → `RequestPackagePurchaseCommandBody.Currency = 3` on the wire
  (`kafka/message/cashshop/kafka.go`)
- Field names/order/types are byte-for-byte identical to atlas-cashshop's Task 16 type
  (`services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:136-142` diffed
  against the new channel-side copy — identical).
- Consumer side: `services/atlas-cashshop/.../kafka/consumer/cashshop/consumer.go:224` passes
  `c.Body.Currency` straight into `PurchasePackageAndEmit`, which eventually calls
  `effectivePurchaseCurrency(currency)` (`cashshop/processor.go:84-90`, atlas-cashshop):
  `switch currency { case walletCurrencyCredit(1), walletCurrencyPoints(2): return currency;
  default: return walletCurrencyPrepaid(3) }`. For input `3` this hits `default` and returns `3`
  — an identity mapping, not a silent reinterpretation. No 0/1/2-only switch anywhere on the
  path (grep swept `cashshop/*.go` and `socket/handler/cash_shop*.go` on the channel side and
  confirmed no `switch currency`/bare `case 0/1/2` pattern outside this one, which is a no-op for
  3).

No blocking defect. Currency 3 is preserved unchanged from handler to persisted asset.

## 3. `BUY_OTHER_PACKAGE` dispatch — BLOCKING, disproven by mutation

**`TestBuyOtherPackageIsDispatched` does not exercise the dispatcher.** It calls
`isCashShopOperation(logrus.New())(options, 33, CashShopOperationBuyOtherPackage)` directly
(`cash_shop_package_test.go:114-127`) — a pure lookup helper that predates this diff and was
never broken by it (the op-33 binding already existed in every GMS template per the brief's own
framing: "It is already bound in every GMS template's `operations` table"). It does not call
`CashShopOperationHandleFunc`, the function the new arm was actually added to.

Mutation performed: I removed the newly-added dispatch arm at
`socket/handler/cash_shop_operation.go:201-206` —

```go
if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyOtherPackage) {
	sp := &cashsb.ShopOperationBuyOtherPackage{}
	sp.Decode(l, ctx)(r, readerOptions)
	handleBuyOtherPackage(l, ctx, wp)(s, sp)
	return
}
```

— and re-ran `go test ./socket/handler/... -run 'TestBuyOtherPackageIsDispatched|TestCashShopPackage' -v`.
Result: **both tests still PASS** with the arm deleted (verified output: `--- PASS:
TestBuyOtherPackageIsDispatched (0.00s)`). This means mode 33 falling straight through to
`Unhandled Cash Shop Operation` — the exact defect this task exists to eliminate — would ship
undetected by the test suite. I restored the file and confirmed `git diff --stat` is empty
afterward.

The code itself is correct (the arm is present, wired to `handleBuyOtherPackage`, and returns —
confirmed by direct read, `cash_shop_operation.go:201-206`), so this is a test-honesty finding,
not a functional one: the "acceptance criterion" test the brief asked for (Step 2:
`TestBuyOtherPackageIsDispatched` … "This is the acceptance criterion 'grep shows it referenced
beyond its declaration', turned into a test") does not actually turn that criterion into a test
of the *dispatcher* — a future regression that deletes or reorders this arm would pass CI.

## 4. New mode-33 codec — matches D3a; test does NOT pin byte layout (BLOCKING per brief item 4)

Order and shape confirmed correct: `shop_operation_buy_other_package.go` decodes exactly `spw
string, serialNumber uint32, name string, message string`, in that order, matching D3a §4 with no
`pointType`/`option`.

The test, however, is a round-trip test only (`TestShopOperationBuyOtherPackageRoundTrip`,
`shop_operation_buy_other_package_test.go`) — it encodes and decodes through the type's own
`Encode`/`Decode`, then compares the decoded struct back to the input struct. This is a weaker
guarantee than the sibling files' convention: e.g.
`shop_operation_buy_package_test.go` has both a round-trip test *and*
`TestShopOperationBuyPackageV79Bytes`, which asserts a literal hex string independent of the
type's own decoder.

Mutation performed: I edited both `Encode` and `Decode` in the new file to swap the write/read
order of `name` and `message` (both are strings, so this compiles and stays internally
self-consistent), then re-ran `go test ./cash/serverbound/... -run
TestShopOperationBuyOtherPackage -v`. Result: **all 13 variant subtests still PASS.** A
field-order defect that would desync this codec from the real client wire layout is invisible to
this test — round-tripping through a codec's own encode/decode pair can never detect a
mismatch against the actual spec, only an internal inconsistency between the two functions. I
restored the file (`cp` from backup) and confirmed `git diff --stat -- cash/serverbound/shop_operation_buy_other_package.go`
is empty afterward.

The report does not mention this gap or attempt to justify the round-trip-only choice; it treats
the codec as done once the round-trip test passes. This is exactly the "test that passes either
way" class the review brief calls out, applied to a case the brief explicitly flagged in advance
(item 4: "confirm ... its test pins the byte layout rather than round-tripping through its own
encoder").

## 5. `option` documentation — PASS

`cash_shop_package.go:71-84` (handleBuyPackage doc comment): explicitly cites "derivation.md D4a,
§6: 1 = NX Credit, 2 = Maple Point, 4 = NX Prepaid" and states `option` "is currently unconsumed
server-side" and "is not unused/spare, only not yet wired." Grep swept the diff for
`unused|spare|reserved` near `option` — the only hit is this comment's own explicit denial of
those labels. Matches D4a exactly.

## 6. Self-gift rejection / `giftRejectionReason` reuse — PASS

`grep -n "func giftRejectionReason" services/atlas-channel/atlas.com/channel/socket/handler/`
returns exactly one definition, `cash_shop_gift.go:30` (pre-existing, from Task 14).
`handleBuyOtherPackage` calls it four times (`cash_shop_package.go`) for credential mismatch,
unknown recipient, cross-world recipient, and sender-resolution failure. No second mapper was
introduced. The self-gift check (`recipient.AccountId() == s.AccountId()`) reuses the existing
`errGiftOwnAccount` sentinel through the same mapper rather than a new comparison/error path.

## 7. Scope — PASS, no drift

`git diff --stat 89e0eb4..96bc6ea` shows exactly 9 files, matching the brief's Files list plus its
"Also touched" set one-for-one:

- `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go` (+test) — brief-named
- `services/atlas-channel/.../socket/handler/cash_shop_package.go` (+test) — brief-named
- `services/atlas-channel/.../socket/handler/cash_shop_operation.go` — brief's "Also touched"
- `services/atlas-channel/.../cashshop/producer.go` — brief's "Also touched"
- `services/atlas-channel/.../cashshop/processor.go` — brief's "Also touched"
- `services/atlas-channel/.../kafka/message/cashshop/kafka.go` — brief's "Also touched"
- `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go` — brief-named (Step 5)

No file outside this set was touched. The report's Files-changed list matches the actual diff
exactly (`git status --porcelain` confirms only pre-existing/other-task artifacts remain
untracked/modified — `go.work.sum` and sibling `review-task-*.md`/`agent-ledger.tsv` files not
part of this diff).

## Mutation-testing summary (independent reproduction)

| Mutation | Expected | Observed | Restored clean? |
|---|---|---|---|
| Swap 154/156 in `TestCashShopPackageResultBodies`'s own operations map | RED | Both subtests FAIL with full byte-slice mismatch (`0x9c` vs `0x9a`, both directions) | Yes, `git diff --stat` empty |
| Remove the new `BUY_OTHER_PACKAGE` dispatch arm from `cash_shop_operation.go` | RED (dispatch test should fail) | **GREEN — no test failure** (blocking finding #3) | Yes, `git diff --stat` empty |
| Swap `name`/`message` encode+decode order in the new codec (self-consistent) | RED (byte-layout regression should be caught) | **GREEN — all 13 variants pass** (blocking finding #4) | Yes, `git diff --stat` empty |

Two of three of my own independent mutations found gaps the implementer's own mutation evidence
(the 154/156 swap) did not surface, because that swap only exercised the one test that already
had strong byte-pinning. The dispatcher-wiring and codec-byte-order tests were both weaker than
their names/purpose imply.

## Not evaluable

- No dedicated unit test exists for `handleStatusEventPackagePurchased`
  (`kafka/consumer/cashshop/consumer.go`) in this diff; correctness there was reviewed by direct
  read only (branches on `RecipientCharacterId`, projects `AssetIds` via
  `asset.NewProcessor().GetById`, matches `handleStatusEventPurchase`'s established pattern). The
  brief's Step 5 did not require a test for this handler, so this is not scored as a defect, but
  it means the consumer-side wiring is unverified by any test in this diff — flagged for
  awareness, not blocking.

## Verdict rationale

Both blocking findings are test-honesty gaps, not functional defects — the production code is
correct on every axis checked (currency routing, wire encoding, dispatch wiring, doc comments,
mapper reuse, scope). But per this branch's explicit standing bar ("do not accept ... a claim
that a test 'proves' something unless you have mutated the code to confirm it"), a test that
does not fail when the thing it claims to guard is removed is not evidence, and two of the tests
in this diff are exactly that: `TestBuyOtherPackageIsDispatched` doesn't touch the dispatcher, and
`TestShopOperationBuyOtherPackageRoundTrip` can't detect a field-order defect by construction.
Both are inexpensive to fix (call `CashShopOperationHandleFunc` directly for #3; add a
`TestShopOperationBuyOtherPackageBytes`-style literal-hex assertion for #4, mirroring
`shop_operation_buy_package_test.go`'s own convention) and do not require touching production
code.
