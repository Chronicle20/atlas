# Task 8 — Fix Round 1 Re-review

Range: `4d121b8..3cc3c89` (fix commit `3cc3c89` on top of the already-clean
`4d121b8`). Scope: the single blocking finding from
`docs/tasks/task-240-cash-shop-stub-operations/review-task-8.md` — the
APPLY_WISHLIST read-failure branch answering with `CashShopLoadWishFailedBody`
(mode 93) instead of the latch-clearing pair member.

This is a narrow re-review of the fix only. Everything else in `4d121b8` was
already confirmed clean by the prior review and is not re-litigated here.

## Diff surface

```
services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_apply_wishlist_test.go | 110 ++++++++++
services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go           |   9 +-
2 files changed, 118 insertions(+), 1 deletion(-)
```

No `libs/atlas-packet` files touched by this commit.

## 1. Is the emitted mode byte on the error path now 99, not 93?

Yes. `cash_shop_operation.go:195` now calls
`cashcb.CashShopSetWishFailedBody("unknown_error")` in the
`wishlist.NewProcessor(...).GetByCharacterId` error branch, replacing the
prior `CashShopLoadWishFailedBody("unknown_error")`.

The mode is resolved from the tenant `operations` table, not hard-coded:
`libs/atlas-packet/cash/clientbound/shop_operation_body.go:249`

```go
mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationSetWishFailed)
```

`CashShopOperationSetWishFailed = "SET_WISH_FAILED"`
(`shop_operation_body.go:27`), and the v95 template maps that key to 99:
`services/atlas-configurations/seed-data/templates/template_gms_95_1.json:4823`
— `"SET_WISH_FAILED": 99`. **PASS.**

## 2. Is mode 99 genuinely the right pair member?

Confirmed independently against `derivation.md` §3 (D2b), not taken on the
fix's word. Evidence 1 (`derivation.md:155-162`) is a full sweep of
`mov dword ptr [reg+1Ch], 0` (the request-in-flight latch clear) over the
whole `CCashShop` code range, finding exactly two wishlist arms that clear
it:

```
0x494d70  CCashShop::OnCashItemResSetWishDone(CInPacket &)+10   mov dword ptr [esi+1Ch], 0
0x4969c7  CCashShop::OnCashItemResSetWishFailed(CInPacket &)+7  mov dword ptr [esi+1Ch], 0
```

`OnCashItemResSetWishDone` = mode 98 = `UPDATE_WISHLIST` (the existing
success arm, unchanged by this fix). `OnCashItemResSetWishFailed` = mode 99
= `SET_WISH_FAILED` per the dispatch table at `derivation.md:190-195`
(`case 0x63u → OnCashItemResSetWishFailed` (99)). `OnCashItemResLoadWishDone`
and `OnCashItemResLoadWishFailed` (92/93) appear in **neither** latch-clearing
range. So SET_WISH_FAILED (99) is indeed the SetWish pair's failure member,
symmetric with UPDATE_WISHLIST (98) as its success member — exactly what the
handler's success arm above the fixed branch already answers with. **PASS.**
The derivation supports the fix's premise; it does not point to a different
arm.

## 3. Does the new test truly pin the failure arm?

`TestApplyWishlistReadErrorAnswersSetWishFailedNotLoadWishFailed`
(`cash_shop_apply_wishlist_test.go:139-172`) does a full dispatch through
`CashShopOperationHandleFunc`, forcing the wishlist read to fail via an
`httptest.NewServer` that always returns 500, pointed at by
`CASHSHOP_SERVICE_URL` (`t.Setenv`, line 149). It asserts the announced mode
byte both ways:

```go
if got.body[0] == 93 {
    t.Fatalf("mode byte = %d (LOAD_WISH_FAILED) ... want 99 ...", got.body[0])
}
if got.body[0] != 99 {
    t.Fatalf("mode byte = %d, want 99 (SET_WISH_FAILED)", got.body[0])
}
```

I independently re-verified the RED claim rather than trusting the report: I
reverted the single line in `cash_shop_operation.go` back to
`CashShopLoadWishFailedBody`, ran the test in isolation, and got:

```
cash_shop_apply_wishlist_test.go:166: mode byte = 93 (LOAD_WISH_FAILED) --
does not clear the client's request-in-flight latch; want 99 (SET_WISH_FAILED)
--- FAIL: TestApplyWishlistReadErrorAnswersSetWishFailedNotLoadWishFailed (0.02s)
```

then restored the file with `git checkout --`. The test is genuinely capable
of catching a regression back to 93 — not tautological. **PASS.**

## 4. Identifier reality (Global Constraint)

Both identifiers exist verbatim, pre-dating this fix (task-183 Wave 1.1
comment at `shop_operation_body.go:24`), so nothing was invented by this
commit:

- `CashShopOperationSetWishFailed = "SET_WISH_FAILED"` — `shop_operation_body.go:27`
- `func CashShopSetWishFailedBody(...)` — `shop_operation_body.go:246-253`

**PASS.**

## 5. Regression check

- Success path still answers `UPDATE_WISHLIST` (mode 98) via
  `cashcb.CashShopWishListUpdateBody(sns)` — `cash_shop_operation.go:205`
  (unchanged from `4d121b8`).
- The `inferred` D2b hedge doc comment (lines 176-181) is present and
  unmodified; the new failure-arm comment (lines 182-188) is added
  immediately below it without altering the original text.
- Empty-wishlist body construction (`sns := make([]uint32, len(wl))` loop,
  lines 199-202) is untouched.
- No Kafka arm appears in the diff (`git diff ... | grep -i kafka` → no
  output).
- No `libs/atlas-packet` file is touched by this commit (`git diff --stat
  4d121b8..3cc3c89 -- libs/atlas-packet` → empty).
- Test-setup convention: reuses the existing Builder-style
  `newCashItemUseTestSession` helper (`character_cash_item_use_test.go:61`),
  no new `*_testhelpers.go` file introduced.

All **PASS**, no regressions found.

## Build / test verification (run myself)

From `services/atlas-channel/atlas.com/channel`, test cache cleared first:

```
go clean -testcache && go build ./... && go test ./...
```

All packages pass, including `ok atlas-channel/socket/handler 1.053s`.

## Recurrence check — same brief-vs-derivation contradiction elsewhere in this file

Swept the rest of `cash_shop_operation.go` for other failure/success arm
pairs answered from this handler: `BUY_COUPLE`/`BUY_PACKAGE`/`BUY_FRIENDSHIP`
arms (lines 165-190ish) only log and do not yet announce any reply body
(stubbed per Task 17/D4a, out of this fix's scope), and the
`GET_PURCHASE_RECORD` pair at lines 223/229
(`CashShopPurchaseRecordFailedBody` / `CashShopPurchaseRecordDoneBody`) uses
a single self-consistent operation-key pair with no wishlist-latch-style
split identified in the derivation for that opcode — no second instance of
the brief-vs-derivation contradiction was found in this file's diff surface
or its immediate neighbors.

## Not evaluable

None — the fix's surface (one line changed, one comment added, one new test
file) was fully within the review scope and every claim was independently
verified against source and a live test run.

## Verdict

The blocking finding is closed: the error path now answers with
`CashShopSetWishFailedBody` (mode 99, `SET_WISH_FAILED`), confirmed as the
correct latch-clearing pair member for `UPDATE_WISHLIST` (mode 98) per
`derivation.md` D2b evidence 1, resolved through the tenant `operations`
table (not hard-coded), backed by identifiers that exist verbatim, and pinned
by a test independently confirmed RED-before/GREEN-after. No regressions in
the surrounding code.
