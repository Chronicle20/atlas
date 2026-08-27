# review-currency-fixes

Task: task-269-ring-pair-behavior · Unit: `94a3c365c..d228fea7a` (`67d31d6bc`,
`8ee623194`, `d228fea7a`)

Reviewed against:
- `docs/tasks/task-269-ring-pair-behavior/bug-ring-purchase-wrong-currency.md`
- `docs/tasks/task-269-ring-pair-behavior/bug-package-purchase-wrong-currency.md`

## Scope

`git diff --stat 94a3c365c..d228fea7a`:

```
docs/.../fix-ring-purchase-wrong-currency.md      | 118 ++++
services/atlas-channel/.../cashshop/processor.go       |  73 ++++++++++---
services/atlas-channel/.../cashshop/processor_test.go  |  34 ++++++
services/atlas-channel/.../handler/cash_shop_package.go|  42 ++++----
services/atlas-channel/.../handler/cash_shop_ring.go   |  28 ++---
```

Reviewed the diff hunks for all five files, the full current bodies of
`processor.go`, `cash_shop_ring.go`, `cash_shop_package.go`, and read the
consumer-side contract in `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go`
and `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`
(`effectivePurchaseCurrency`) since correctness of the mapping genuinely
depends on those. Also read `ShopOperationBuyCouple` / `ShopOperationBuyPackage`
(`libs/atlas-packet/cash/serverbound/`) decode paths (untouched by this diff)
to trace what `option`/`isPoints`/`currency` actually carry per wire version.

## Findings

### 1. Fallback ordering across wire versions — PASS

Traced every version's decoded `option`/`isPoints`/`currency` triple through
`resolveOptionCurrency`:

- **GMS < 83 (ring)**: `shop_operation_buy_couple.go` `decodeGMS`'s
  `legacyGMS` branch sets `isPoints`/`currency` from the wire and never
  touches `option` (zero-value `0`). `resolveOptionCurrency(0, isPoints,
  currency)` falls to `resolvePurchaseCurrency(isPoints, currency)` — byte
  for byte the pre-fix call. No regression.
- **GMS < 83 (package)**: `shop_operation_buy_package.go`'s `legacyGMS`
  branch decodes only `serialNumber`; `pointType`/`option` stay zero-value.
  `resolveOptionCurrency(0, false, 0)` → `resolvePurchaseCurrency(false, 0)`
  → `0`, identical to the pre-fix literal-`0` passthrough
  (`cash_shop_package.go:83`, pre-fix diff hunk). No regression.
- **JMS (ring)**: `decodeJMS` sets none of `isPoints`/`currency`/`option`
  (all zero-value). Falls to `resolvePurchaseCurrency(false, 0)` = `0`,
  matching the documented "JMS keeps today's prepaid behavior." No
  regression.
- **GMS >= 83 (ring and package)**: `option` is decoded and now threaded
  through end to end (see finding 3).

`resolvePurchaseCurrency`'s only other call site is `RequestPurchase`
(`processor.go:112-116`), untouched by this diff — confirmed by
`grep -n resolvePurchaseCurrency` isolation and reading that function.

### 2. `handleBuyOtherPackage` (gift package, mode 31) — PASS, unchanged

`cash_shop_package.go:148`: `RequestPackagePurchase(..., false,
walletCurrencyPrepaid, 0, sp.SerialNumber(), recipient.Id(),
sender.Name())` — `option` is hardcoded `0`, `isPoints=false`,
`currency=walletCurrencyPrepaid(3)`. `resolveOptionCurrency(0, false, 3)`
falls to `resolvePurchaseCurrency(false, 3)`, whose only branch
(`isPoints && currency==0`) cannot fire, so it returns `3` unchanged. The
new `option` parameter is provably inert for this arm, matching D3a's
pin to NX Prepaid.

### 3. Wallet-currency convention (1/2/3, never 4) — PASS

`resolveOptionCurrency` (`processor.go:151-168`) maps wire `option` 1→1,
2→2, 4→**3** (not 4), default→`resolvePurchaseCurrency`. Confirmed against
the consumer: `wallet.Model.Balance`/`Purchase`
(`services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-58`) treat
`1` as credit, `2` as points, and **everything else** (including a raw `3`
or a raw `0`) as prepaid — so option 4 → wallet currency 3 lands on the
correct bucket, and `effectivePurchaseCurrency`
(`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:87-92`)
persists `3` unchanged on the asset row (no 0-vs-3 ambiguity). No path in
`resolveOptionCurrency` or `resolvePurchaseCurrency` can emit a bare `4`;
the option-4→prepaid case is deliberately renormalized to `3` before it
ever reaches the command body. No stale "everything-else-arm" collision
found.

### 4. Test honesty and stale references

- `TestResolveOptionCurrency` (`processor_test.go`, current name after the
  `d228fea7a` rename) asserts the new contract directly: option 1→1,
  2→2, **4→3** (the case that would have silently mis-debited before this
  fix), option-0 fallback cases, and an unmapped option value. This is a
  real test of the new behavior — it did not exist before `67d31d6bc` and
  fails without `resolveOptionCurrency`'s switch.
  **Non-blocking**: no test exercises the handler layer end-to-end (that
  `sp.Option()` actually reaches `RequestRingPurchase`/`RequestPackagePurchase`
  through `handleBuyCouple`/`handleBuyFriendship`/`handleBuyPackage`). Both
  the ring and package fix reports acknowledge this and give a reason
  (`handleRingPurchase`/`handleBuyPackage` have no injected mock/producer
  seam, and the repo forbids test-only constructors) — a defensible call,
  not a defect, but it means the plumbing between decode and the resolver
  is confirmed only by reading, not by a red/green test.
- `grep -rn resolveRingPurchaseCurrency` across the four touched Go files
  returns nothing — the rename in `d228fea7a` left no stale call site or
  doc comment inside the reviewed diff. Confirmed doc comments on
  `resolveOptionCurrency`, `RequestPackagePurchase`, `RequestRingPurchase`,
  and `handleRingPurchase` all describe the *current* shared-resolver
  behavior, not the pre-rename `resolveRingPurchaseCurrency` name.

### 5. Doc-trail completeness — non-blocking, but flagged

- `resolveRingPurchaseCurrency` **is** still referenced by name in three
  files under `docs/tasks/task-269-ring-pair-behavior/`:
  `bug-ring-purchase-wrong-currency.md`, `bug-package-purchase-wrong-currency.md`,
  and `fix-ring-purchase-wrong-currency.md` (the last is committed, in
  `67d31d6bc`/`8ee623194`, and pre-dates the `d228fea7a` rename — expected
  to go stale, and not corrected by the later commit). Not blocking (these
  are historical fix-report narratives, not the interface itself), but
  worth a follow-up note if anyone edits that file later expecting current
  names.
- More significant: `git status --porcelain` shows `bug-package-purchase-wrong-currency.md`,
  `bug-ring-purchase-wrong-currency.md`, and `fix-package-purchase-wrong-currency.md`
  are **untracked** in this worktree — none of the three commits under
  review actually commits them. Only `fix-ring-purchase-wrong-currency.md`
  is in git history. Concretely: `d228fea7a` (the package fix) changed only
  the four Go files; its own implementation report
  (`fix-package-purchase-wrong-currency.md`, present on disk, dated the
  same time as the commit) was never `git add`ed, and
  `bug-package-purchase-wrong-currency.md`'s own "## Resolution" section
  still reads "_(pending)_" even though the code fix is landed. The task
  description characterizes the range as "three code/docs commits," but
  only two of the three (`67d31d6bc`, `8ee623194`) actually commit docs;
  `d228fea7a` is code-only, and its companion doc trail exists only in the
  working tree. Not a correctness defect, but a real gap in the done-ness
  bar this repo's CLAUDE.md sets ("Done means verified" / doc commit
  hygiene) — flagged, not blocking, since it does not affect production
  behavior.

## Build/test verification

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```
Exit 0, no output.

```
cd services/atlas-channel/atlas.com/channel && go test ./cashshop/... ./socket/handler/...
```
All `ok` (or `[no test files]` for `cashshop/wallet`), no failures.

## Not evaluable

- Live re-test against a real client on GMS 83.1 (the bug doc's own "Gate:
  pending" note) — outside a static code review's reach.
- Whether `option` can legitimately arrive as `0` on a genuine GMS >= 83
  confirm (both bug docs flag this as "not yet answered," defensive-only).

## Verdict

APPROVED_WITH_FINDINGS — the currency-resolution logic itself is correct
end to end (traced through every GMS/JMS version and confirmed against the
`atlas-cashshop` consumer contract), `handleBuyOtherPackage` is provably
unchanged, and the new test asserts the actual new contract. The two
findings below are process/doc-completeness gaps, not code defects.
