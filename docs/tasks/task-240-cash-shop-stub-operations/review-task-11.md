# Review: Task 11 — REBATE_LOCKER_ITEM (atlas-cashshop side)

- Commit range: `fb2c946f8..669272766` (single commit, `669272766`)
- Brief: `.superpowers/sdd/plan/task-11-brief.md` (Controller corrections C1–C5 authoritative over the body)
- Implementer report: `.superpowers/sdd/plan/task-11-report.md`
- Module: `services/atlas-cashshop/atlas.com/cashshop`

## Scope reviewed

`git diff --stat fb2c946f8..669272766` (16 files, +580/-36):

- `cashshop/rebate.go` (new), `cashshop/rebate_test.go` (new)
- `cashshop/processor.go` (Purchase now threads currency into asset creation)
- `cashshop/inventory/asset/{entity,model,processor,administrator,resource,rest}.go` (C1/C3: new `Currency` column + signature ripple)
- `cashshop/inventory/compartment/processor.go`, `coupon/granter.go`, `surprise/processor.go`, `kafka/consumer/item/consumer.go` (C3 non-purchase call sites, pass `0`)
- `kafka/message/cashshop/kafka.go`, `kafka/producer/cashshop/producer.go`, `kafka/consumer/cashshop/consumer.go` (command/event wiring)

Cross-referenced (contract only, not owned by this unit): `wallet/model.go` (`Balance`/`Purchase`/`Award` currency convention), `services/atlas-channel/atlas.com/channel/cashshop/processor.go:112-124` (`resolvePurchaseCurrency`), `ledger/ledger.go` (Task 10's `Claim`), `design.md:18` (BUY_NORMAL wire shape).

## Priority item: BUY_NORMAL round-trip asymmetry — CONFIRMED, blocking

The suspected defect is real. Verified by direct code trace and by a throwaway
proof test run against this branch (not left in the tree):

1. `services/atlas-channel/atlas.com/channel/cashshop/processor.go:112-124` —
   `resolvePurchaseCurrency(isPoints, currency)` returns `currency` unchanged
   whenever `isPoints` is false. `design.md:18` records BUY_NORMAL as
   `isPoints=false, currency=0`. So a genuine BUY_NORMAL purchase reaches
   atlas-cashshop's `Purchase` with wire currency `0` — not a legacy artifact,
   a normal, common value.

2. `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-57` —
   `Balance`/`Purchase`/`Award` all route currency `1` to credit, `2` to
   points, and **everything else, including `0`, to prepaid**. So
   `w.Purchase(0, price)` at `cashshop/processor.go:170` debits **prepaid**.

3. `cashshop/processor.go:217-224` — the same raw `currency` value used to
   debit is passed straight into `Create`/`CreateWithCashId` and persisted
   onto `asset.Entity.Currency` (`inventory/asset/entity.go`). For a BUY_NORMAL
   purchase this row is written with `Currency = 0`.

4. `cashshop/rebate.go:136-138` — `RebateAndEmit` reads that stored value and
   applies the C2 default-bucket translation unconditionally:
   ```go
   currency := am.Currency()
   if currency == 0 {
       currency = 1
   }
   ...
   w = w.Award(currency, ci.Price())
   ```
   `wallet.Model.Award` (`wallet/model.go:73-100`) routes currency `1` to
   **credit**. So the rebate always credits credit/NX for a `Currency == 0`
   asset, regardless of which bucket was actually debited.

**Proof test (throwaway, run and removed — not committed):**

```go
// seed wallet: Credit=0, Prepaid=500
NewProcessor(l, ctx, db).PurchaseAndEmit(characterId, 0 /* BUY_NORMAL wire currency */, serialNumber, transactionId, "")
// -> credit=0 prepaid=0   (prepaid debited, confirms step 2/3)
// asset.Currency() == 0   (confirms step 3's persisted value)
NewProcessor(l, ctx, db).RebateAndEmit(characterId, accountId, cashId, uuid.New())
// -> credit=500 prepaid=0 (rebate credited CREDIT, not the prepaid bucket that was debited)
```

Actual test output:
```
after purchase: credit=0 prepaid=0
persisted asset.Currency = 0
after rebate: credit=500 prepaid=0
ROUND-TRIP BROKEN: rebate refunded credit/NX, not the prepaid bucket that was debited
```

This is a genuine free currency-conversion exploit: buy on prepaid (wire
currency 0, the ordinary BUY_NORMAL case), rebate, and the refund lands on
credit/NX instead. `rebate_test.go`'s "the currency actually round-trips"
subtest (`rebate_test.go:253-273`) only exercises `Currency == 2` (points) —
a value the C2 translation never touches — so it cannot catch this. Every
other subtest that uses `Currency == 0` (`rebate_test.go:92,159,189,211`)
deliberately treats it as the legacy/gift case the correction intends, and
none of them seed a purchase through the real `Purchase` path with wire
currency 0, so nothing in the new test suite pins the actual BUY_NORMAL
behavior.

**Root cause, confirmed as diagnosed:** `Currency == 0` is overloaded on the
asset row. It means both "legacy row, column didn't exist yet" and "a real
purchase whose effective bucket was prepaid" (any BUY_NORMAL buy, and any
buy where `resolvePurchaseCurrency` leaves the wire value at `0`). C2's
"treat 0 as the default credit/NX bucket" ruling is correct only for the
first meaning and silently wrong for the second. No read-site (rebate-side)
translation can disambiguate these, because the information distinguishing
them is discarded before it reaches the asset row: `wallet.Model.Purchase`'s
own `else` branch already conflates "currency 0" with "currency 3, 4, ... N"
into a single prepaid debit, and `Award`'s `default` branch does the same on
credit — the fix belongs at the boundary where `Purchase`/`Create` persist
the effective bucket (e.g. normalize `0` to a distinct explicit "prepaid"
code, or a distinct explicit "legacy/unknown" sentinel that is not the same
value as prepaid's own currency id), not in `rebate.go`'s read.

This blocks the unit: the corrected behavior is provably wrong for the most
common purchase path in the game (BUY_NORMAL), and no test in the diff would
fail today to catch it — the verification note in the brief ("Add a subtest
proving the currency actually round-trips … Without it C1 is unpinned") was
followed literally but with a currency value (`2`) that cannot exercise the
`0`-translation branch at all.

## Normal review

### Requirement coverage (brief + C1–C5)

| Requirement | Status | Evidence |
|---|---|---|
| `ledger.Claim` first statement, `ErrAlreadyProcessed` → success-without-effect | PASS | `rebate.go:70-76`; matches `ledger/ledger.go:44` real signature (C5) |
| Asset resolved within requesting account's compartments only | PASS | `rebate.go:83-107` scans `cicP.GetByAccountId(accountId)` only |
| Reject `CommodityId == 0`, expired, unresolvable commodity | PASS | `rebate.go:112-124`; each covered by a subtest (`rebate_test.go:227-247`, `:205-225`) |
| Delete asset | PASS | `rebate.go:127-131`, `astP.Delete(buf)(am.Id())` |
| Credit wallet on the purchase-recorded currency (C1) | **FAIL for `Currency==0`, PASS for `Currency==2`** | see priority item above; `rebate_test.go:253-273` only pins the non-zero case |
| `LOCKER_REBATED` event on success | PASS | `rebate.go:155`, asserted `rebate_test.go:111-117` |
| `purchaserecord` row survives rebate (C4) | PASS | `rebate.go` never touches `purchaserecord`; `rebate_test.go:136-150` asserts count unchanged |
| `Currency` column added via `AutoMigrate`, no new migration entry, no `main.go` change | PASS | `entity.go` diff only adds a struct field + doc comment; `grep` below confirms no `main.go`/migration-list touch |
| C3 signature ripple: `Create`/`CreateWithCashId` + `*AndEmit` gain `currency uint32` | PASS | `asset/processor.go` diff, all four signatures updated together |
| C3 non-purchase call sites pass `0` | PASS | `compartment/processor.go:278`, `surprise/processor.go:247`, `coupon/granter.go:131`, `kafka/consumer/item/consumer.go:52`, `asset/resource.go:100` (REST create) all pass literal `0`, each with a comment explaining why |
| C3 purchase call sites pass the real debited currency | PASS | `cashshop/processor.go:222,224` pass the same `currency` `w.Purchase` was called with |
| Command/event wiring | PASS | `kafka/message/cashshop/kafka.go` adds `CommandTypeRequestLockerRebate`/`RequestLockerRebateCommandBody`/`StatusEventTypeLockerRebated`/`LockerRebatedBody`; `kafka/producer/cashshop/producer.go` adds `LockerRebatedStatusEventProvider`; `kafka/consumer/cashshop/consumer.go:65-67,178-190` registers `handleCommandRequestLockerRebate` in `InitHandlers` |

```
$ grep -rn "asset.Migration\|Migration(" services/atlas-cashshop/atlas.com/cashshop/main.go | head
```
Confirmed no `main.go` change is in the diff (`git diff --stat` above lists no `main.go`), consistent with the AutoMigrate-only constraint.

### Rejection cases (brief table)

All five rejection subtests in `rebate_test.go` (new-transaction-id-on-gone-
asset, foreign-account, expired, no-commodity-id, plus the redelivery replay)
assert `ERROR` with `Operation == cashshop.ErrorOperationRebate` and the
wallet unchanged. Each is backed by a fixture that isolates exactly the
condition under test (e.g. the foreign-account case seeds the asset in
account 99's compartment and requests as account 42 — `rebate_test.go:183-203`).
These are honest tests: each fixture, if the corresponding `if` were deleted
or inverted in `rebate.go`, would fail (rejection subtests assert the wallet
is untouched and an `ERROR` fires; removing the guard would cause a credit
and no error).

### Ledger claim ordering / `ErrAlreadyProcessed`

`rebate.go:70-76` calls `ledger.Claim` as the very first statement inside the
`message.Emit`-wrapped closure, before any read. `ErrAlreadyProcessed` returns
`nil` with no `reject()` call, so no error and no event — matches the brief
and C5's corrected signature. The "replay refunds once" subtest
(`rebate_test.go:121-134`) reuses the same `TransactionId` and asserts no
double-credit and no second outbox row — this is a real regression test: if
`Claim`'s `OnConflict{DoNothing}` guard were removed, the second call would
re-delete (no-op, asset already gone → falls into `am == nil` → reject, so
this particular test would actually still pass because the second run hits
the "gone asset" rejection path first). Note: the "replay" subtest and the
"new transaction id on a gone asset" subtest are not fully independent proofs
of the ledger claim specifically, since a second `RebateAndEmit` on an
already-deleted asset would be correctly rejected by the asset-not-found path
even if `ledger.Claim` were skipped entirely. This is a minor test-strength
gap, not a defect — non-blocking.

### Currency column / doc comment (C2)

`inventory/asset/entity.go`'s new `Currency` field doc comment states the
"`0 == default bucket`" convention as required by C2. As shown above, the
convention is actually implemented, but is incorrect for real BUY_NORMAL
purchases — the doc comment states the (wrong) design decision faithfully;
the defect is in the decision C2 encoded, now shown to conflict with
`resolvePurchaseCurrency`'s real wire behavior, not in whether the code
matches the doc.

### Test suite

`go build ./... && go test ./...` on the module: all packages pass (verified
directly, not from a stale run). `rebate_test.go`'s 7 subtests all pass.

## Not evaluable

- `ledger.Claim`'s own correctness (uniqueness semantics, tenant scoping) is
  Task 10's surface, not this unit's; only its call-site usage was reviewed
  here.
- `resolvePurchaseCurrency` and the channel-side packet parsing are outside
  atlas-cashshop and outside this unit's diff; read only for the contract
  needed to evaluate the priority item.

## Verdict rationale

The priority item is a confirmed, reproducible, blocking defect: a BUY_NORMAL
purchase (the ordinary, non-points buy path — the majority of real cash shop
purchases) debits prepaid and its rebate credits credit/NX, a free currency
conversion with no test in the diff that would catch it. Everything else in
the unit — ledger ordering, rejection cases, C3's signature ripple and call
sites, C4 (purchase record survives), wiring — is implemented correctly and
tested honestly.

verdict: CHANGES_REQUIRED
