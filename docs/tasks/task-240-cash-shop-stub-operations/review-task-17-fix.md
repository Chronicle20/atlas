# Review: Task 17 fix round 1 (test-only)

Range: `96bc6ea27..20746105d` (single commit `20746105d`).
Prior review: `docs/tasks/task-240-cash-shop-stub-operations/review-task-17.md` (CHANGES_REQUIRED, two can't-fail-test findings). This round claims to close both, test-only.

## Scope

`git diff --stat 96bc6ea27..20746105d`:

```
.../shop_operation_buy_other_package_test.go       | 26 +++++++
.../socket/handler/cash_shop_package_test.go       | 84 ++++++++++++++++++--
2 files changed, 103 insertions(+), 7 deletions(-)
```

Both files are `_test.go`. No production file appears in the diff. Matches the brief: test-only fix round.

## Finding 1 — dispatch test now drives the real entry point

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_package_test.go`: `TestBuyOtherPackageIsDispatched` was rewritten to:

- Build a full wire packet (`buyOtherPackagePacket`) with mode byte + spw/serialNumber/name/message body.
- Stand up an `httptest.Server` (`newBuyOtherPackageTestServer`) that answers `accounts/` (empty PIC/birthDate, so `verifySecondaryCredential` passes unconditionally) and everything else with an empty JSON:API character list (so `GetByName` resolves `ErrEmptySlice` — "unknown recipient").
- Call `CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, pkt, options)` directly — the real dispatcher entry point, not `isCashShopOperation`.
- Assert an observable effect: exactly one announced packet, writer `cashcb.CashShopOperationWriter` (GIFT_PACKAGE_FAILED).

**Reproduced independently** (not taken on the implementer's word):

- Baseline: `go test ./socket/handler/... -run TestBuyOtherPackageIsDispatched -v` → PASS, log shows the GIFT_PACKAGE_FAILED path being exercised (`"Character [54321] attempted to gift a package to unknown recipient"`, `"gift package failure reason [INCORRECT_NAME]"`).
- Mutation: deleted the dispatch arm at `cash_shop_operation.go:201-206` (`if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyOtherPackage) { ... handleBuyOtherPackage(...) ...}`). Re-ran the same test → **FAIL**: `"Unhandled Cash Shop Operation [33] issued by character [54321]"`, `announced packets = 0, want 1`. This is exactly the RED the brief demanded.
- Reverted via `git checkout --`, re-ran → PASS again. Tree confirmed clean afterward (`git status --porcelain` shows only the pre-existing `go.work.sum` diff and pre-existing untracked review docs from earlier rounds, neither touched by this session).

Verdict: finding 1 from the original review is genuinely closed. The test now fails without the fix and passes with it, and it asserts an observable effect of `handleBuyOtherPackage`, not the `isCashShopOperation` helper's boolean.

## Finding 2 — byte-pin test derived, not back-fitted

`libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package_test.go` adds `TestShopOperationBuyOtherPackageV95Bytes`, which pins the literal wire bytes for `{spw: "ABCD", serialNumber: 0x05060708, name: "Bob", message: "Hi"}` against `want := "040041424344" + "08070605" + "0300426f62" + "02004869"`.

- **Field order matches derivation.md D3a** (`docs/tasks/task-240-cash-shop-stub-operations/derivation.md:208-214,481`): `spw str, serialNumber u32, name str, message str`; `CCashShop::OnGiftPackage @ 0x4907b0`; no `pointType`/`option`. This matches both the production `Encode`/`Decode` order in `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go:44-61` and the pinned byte layout.
- **Independently re-derived the expected hex by hand** (Python, not trusting the implementer's comment): `enc_ascii("ABCD") + le32(0x05060708) + enc_ascii("Bob") + enc_ascii("Hi")` → `040041424344080706050300426f6202004869` — byte-identical to the `want` literal in the test. This rules out "whatever Encode currently emits" back-fitting, since the value was reconstructed from the field-order spec alone, not read off program output.
- **Reproduced the mutation proof.** Swapped `name`/`message` consistently in both `Encode` (`shop_operation_buy_other_package.go:49-50`) and `Decode` (`:59-60`). Re-ran both tests:
  - `TestShopOperationBuyOtherPackageRoundTrip` — still PASS (all 13 variants), confirming the round-trip test alone is blind to this defect class, as claimed.
  - `TestShopOperationBuyOtherPackageV95Bytes` — FAIL: `got 04004142434408070605020048690300426f62, want 040041424344080706050300426f6202004869`. Exactly the RED required.
  - Reverted via `git checkout --`, re-ran → both PASS.

Verdict: finding 2 is genuinely closed. The byte-pin catches a self-consistent field-swap that the round-trip test cannot, and its expected value is externally derivable from the recorded field order rather than circularly copied from current `Encode` output.

## `packet-audit:verify` marker check

The new test's doc comment cites `derivation.md D3a (§4, CCashShop::OnGiftPackage @ 0x4907b0, the same address the round-trip test's own packet-audit:verify marker above already cites)`.

Checked directly:
- `git diff 96bc6ea27..20746105d | grep packet-audit` → only one hit, and it is inside an added *comment* (`"packet-audit:verify marker above already cites"`), not a new marker directive.
- `git show 96bc6ea27:.../shop_operation_buy_other_package_test.go` shows the marker `// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyOtherPackage version=gms_v95 ida=0x4907b0` was already present **before** this fix commit (part of the Task 17 implementation, already reviewed).
- `derivation.md:214` independently records the same address (`CCashShop::OnGiftPackage @ 0x4907b0`).

No new `packet-audit:verify` marker was added by this commit, and the address it references was already recorded on both the sibling test and in `derivation.md` prior to this round. The implementer's claim holds.

## Production code

`git diff --stat` confirms zero non-test files changed. This round is test-only, as ruled.

## Not evaluable

None — both required mutation proofs were reproduced independently, the byte-pin was independently re-derived, and the marker claim was checked against the actual diff and pre-existing sources.

Out of scope per the brief (not re-raised): `handleStatusEventPackagePurchased` having no dedicated test (already dispositioned not-evaluable in the original review; not part of this fix round).

## Verdict

Both blocking findings from the original Task 17 review are closed by real, reproducible mutation-testing evidence, no production code was touched, and no new unverified IDA address was introduced. APPROVED.
