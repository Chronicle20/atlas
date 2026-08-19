# Review — Task 13: GIFT — `atlas-cashshop` side

Commit range: `a6948029b..a759bf12d` (single commit `a759bf12d`).

## Scope

`git diff --stat a6948029b..a759bf12d` — 10 files, 690 insertions(+), 9
deletions(-):

```
cashshop/gift.go                            | 167 ++++++++++
cashshop/gift_test.go                       | 345 +++++++++++++++++++++
cashshop/inventory/asset/administrator.go   |   4 +-
cashshop/inventory/asset/entity.go          |  26 +-
cashshop/inventory/asset/model.go           |  32 ++
cashshop/inventory/asset/processor.go       |  51 ++-
cashshop/processor.go                       |   1 +
kafka/consumer/cashshop/consumer.go         |  18 ++
kafka/message/cashshop/kafka.go             |  34 ++
kafka/producer/cashshop/producer.go         |  21 ++
```

Matches the brief's `### Files` list plus the two C6-declared additions
(`inventory/asset/processor.go`, `cashshop/processor.go`) and the two
registration-only files the brief itself calls out. No scope mismatch.

## C1 — `ledger.Claim` signature

`gift.go:66` calls `ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestGiftPurchase, characterId)` —
matches the controller-verified signature exactly (`ctx` first).
`ErrAlreadyProcessed` is handled as a no-op success (`gift.go:67-69`), matching
the doc comment's stated idempotency contract. PASS.

## C2 — `ErrorOperationGift` constant

`gift.go:57` — `cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationGift, reason, transactionId)`
used on every `reject(...)` path (single `reject` closure, all seven call
sites route through it). No literal `"GIFT"` string anywhere on the path.
PASS.

## C3 — `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL` bare string literal

`gift.go:115` — `return reject("CANNOT_GIFT_RECIPIENT_INVENTORY_FULL")`, a
bare string literal, matching `Purchase`'s existing shape
(`cashshop/processor.go:185`/`:209`) per C3. Confirmed no Go constant was
added anywhere (`grep -rn "CANNOT_GIFT_RECIPIENT_INVENTORY_FULL"` — only
`gift.go` and `gift_test.go`, no `libs/atlas-constants/` addition). Confirmed
it is NOT the `"INVENTORY_FULL"` string `Purchase` uses — different literal.

**Mutation check:** changed the literal to `"INVENTORY_FULL"` in `gift.go`
and reran `TestGift/recipient_locker_full_charges_nothing` — the test goes
RED (`+INVENTORY_FULL` diff on the assertion). Restored; `git diff --stat`
confirmed empty afterward. PASS, and the test genuinely pins the exact
string.

## C4 — no `Currency` field, no 0/1/2 switch

`GiftPurchasedBody` (`kafka/message/cashshop/kafka.go`) and
`RequestGiftPurchaseCommandBody` carry no `Currency` field — verified by
reading the full struct definitions in the diff. `gift.go` reuses the
existing `walletCurrencyCredit` constant (`cashshop/processor.go:55`)
unconditionally, with a doc comment (`gift.go:38-44`) explaining why a gift
is always credit-funded. No `switch` on currency anywhere in `gift.go`.
PASS.

## C5 — fixture discrimination (mutation-tested, not just read)

1. **`recipient locker full` — sender's compartment must not also be full.**
   `gift_test.go:231` seeds the sender's compartment open (capacity 55,
   empty) while the recipient's is seeded at capacity 1 with one asset
   (`gift_test.go:232-233`), with an explicit comment stating why. Confirmed
   by mutation: changed `cicP.GetByAccountIdAndType(recipient.AccountId(), ...)`
   to `sender.AccountId()` in `gift.go` — three subtests went RED
   (`delivers_to_the_recipient_locker`, `recipient_locker_full_charges_nothing`,
   `replay_delivers_once`), proving the fixture and code both key off the
   right compartment. Restored; diff confirmed empty. PASS.

2. **`delivers to the recipient locker` — zero new rows in account 1's
   compartment asserted explicitly.** `gift_test.go:170-172`:
   `senderSeededCcm.Assets()` asserted `Empty` with an explicit message.
   Covered transitively by the same compartment-swap mutation above (which
   also broke this subtest). PASS.

3. **`records the purchase for the sender` — both directions asserted.**
   `gift_test.go:337-343` asserts `Get(...,1,...) == 1` AND
   `Get(...,2,...) == 0`. **Mutation check:** changed
   `purchaserecord.Record(tx, p.t.Id(), sender.AccountId(), serialNumber)` to
   `recipient.AccountId()` — the subtest went RED
   (`the SENDER bought it` assertion failed, since sender's count stayed 0).
   Restored; diff confirmed empty. PASS.

All three C5 requirements hold under actual mutation, not just by reading
subtest names.

## C6 — out-of-list file edits

Both declared in the implementer's report and verified independently:

- `cashshop/inventory/asset/processor.go` — new `CreateGift` method added to
  the `Processor` interface and impl (`processor.go:30-31` interface,
  `:138-183` impl). The single existing call to `create(...)` inside
  `Create` was updated to pass two trailing `""` args
  (`processor.go:101` diff: `..., expiration, "", "")()`), so every existing
  caller (`Create`, `CreateAndEmit`, and their three downstream callers —
  `Purchase`, `Expire`'s replacement path, `surprise`'s reward path) is
  behaviourally unchanged: they persist `GiftFrom`/`GiftMessage` as empty
  strings, same as before the column existed. `CreateGift` is a straight
  copy of `Create`'s body with two extra params threaded through to
  `create(...)`. This is genuinely additive — no existing method signature
  or behaviour changed. PASS on the additive-only claim.
- `cashshop/processor.go` — one new interface line,
  `GiftAndEmit(...) error`, mirroring the existing `RebateAndEmit` entry.
  Purely additive. PASS.

Neither edit duplicates logic that an existing path should have been
extended to cover instead — extending `Create`/`CreateAndEmit`'s public
signature would have forced three unrelated call sites (`Purchase`,
`Expire`, `surprise`) to pass `"", ""`, which is a worse shape than one new
small method. The implementer's own YAGNI reasoning in the report is sound.

## Migration / column widths

`entity.go` adds `GiftFrom string \`gorm:"size:13;not null;default:''"\`` and
`GiftMessage string \`gorm:"size:73;not null;default:''"\``, both additive
with defaulted-empty values — no backfill needed, confirmed by reading the
column tags directly (not trusting the report).

**Width verification, done independently, not trusted from the brief or
report:**

- `libs/atlas-packet/cash/clientbound/shop_inventory.go:35` —
  `model.WritePaddedString(w, m.GiftFrom, 13)` → 13 confirmed.
- `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:43-44` —
  `model.WritePaddedString(w, m.BuyCharacterName, 13)` and
  `model.WritePaddedString(w, m.Text, 73)` → 73 confirmed (this is
  `GiftListEntry.Text`, matching the report's citation).

Both widths hold exactly as claimed. PASS.

## Transaction ordering (brief's Step 4, C1 signature applied)

Read end-to-end in `gift.go`:

1. `ledger.Claim` first statement (`:66`). ✓
2. Resolve commodity by serial (`:75`). ✓
3. Resolve sender and recipient accounts (`:82`, `:87`). ✓
4. Resolve recipient compartment + capacity check (`:98-116`), using
   `recipient.AccountId()` — confirmed by mutation above. ✓
5. Check + debit sender's wallet (`:119-135`), `sender.AccountId()`. ✓
6. Create asset in recipient's compartment (`ccm.Id()`, resolved from
   `recipient.AccountId()` at step 4) with `purchasedBy = characterId` (the
   sender) — `gift.go:141`:
   `astP.CreateGift(buf)(ccm.Id(), ci.ItemId(), serialNumber, walletCurrencyCredit, ci.Count(), 0, characterId, senderName, giftMessage)`,
   matching `CreateGift`'s signature
   `(compartmentId, templateId, commodityId, currency, quantity, petId, purchasedBy, giftFrom, giftMessage)` —
   `characterId` lands in the `purchasedBy` slot. ✓
7. `purchaserecord.Record(tx, p.t.Id(), sender.AccountId(), serialNumber)`
   (`:149`) — sender's account, confirmed by mutation above. ✓
8. `buf.Put(...GiftPurchasedStatusEventProvider(...))` (`:155`), only
   reached on the success path. ✓

**No-partial-state check:** `GiftAndEmit` runs everything inside
`database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {...})`.
`libs/atlas-database/transaction.go:9-14` shows `ExecuteTransaction` calls
`db.Transaction(fn)` when not already inside a tx, which is GORM's standard
commit-on-nil/rollback-on-error semantics. Every `reject(...)` returns
`errGiftRejected` (a non-nil error) from the closure, so a rejection AFTER
the debit (e.g. the asset-create-fails branch at `:142-145`, or the
purchase-record-fails branch at `:149-152`) rolls back the debit along with
everything else in the same transaction. No debit-without-asset, no
asset-without-debit is possible. `rejectEmit` is fired only after the
transaction has already resolved (success or rollback), on the direct
producer path, exactly mirroring `Purchase`/`RebateAndEmit`. PASS.

## Rejection ordering (recipient-capacity before sender-balance)

The implementer's report explicitly flags this as a self-declared deviation
worth reviewer attention. Assessment:

- The brief's own Step 4 numbered list places "resolve recipient
  compartment + check capacity" (4) before "debit the sender's wallet" (5),
  with no separate numbered step for a balance-sufficiency check — the
  implementer folded the balance check into step 5 (check-then-debit in one
  step, mirroring `Purchase`'s existing pattern) and therefore checks
  capacity before balance. This is a literal, defensible reading of the
  brief text as written; it is not a deviation from the brief, it is the
  brief's own ordering.
- On the merits (not just per the brief): checking recipient capacity
  before debiting the sender is the more defensible economic order
  regardless — it avoids ever debiting the sender for a gift that cannot
  physically land, which is the same reason `Purchase` checks-before-debits
  for its own compartment. If a real player fails both checks
  simultaneously, seeing "recipient's locker is full" over "you don't have
  enough cash" is arguably *more* actionable (funds are still theirs;
  fixing capacity requires the recipient's action) and matches the general
  principle of failing on the condition that is least within the current
  actor's control to silently work around by retrying. I did not find a
  correctness defect in this ordering.
- **The absence of a test pinning the dual-failure case is a genuine, minor
  gap** — none of the six subtests exercises "sender lacks funds AND
  recipient's locker is full simultaneously," so the ordering is
  unverified by the suite one way or the other. This is real but
  non-blocking: the brief's own test table (which is the operative spec for
  Step 1) does not list this fixture, so its absence is not a broken
  requirement, just an uncovered edge the report is right to flag for
  awareness.

**Verdict on this point: the landed order is defensible, not a defect; the
missing dual-failure test is a legitimate non-blocking gap, correctly
self-reported.**

## Build / test verification (run independently)

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```

All packages `ok` or `[no test files]`, no failures. `TestGift` itself
(`go test ./cashshop/... -run TestGift -v`) — all 6 subtests PASS.

Three targeted mutations (compartment-check target, C3 literal, purchase-
record account) each independently turned the relevant subtest(s) RED, and
each was restored with `git diff --stat` confirmed empty afterward.

## Not evaluable

- The channel-side producer of `RequestGiftPurchaseCommandBody` and the
  consumer of `GiftPurchasedBody`/`GIFT_PURCHASED` events are out of scope
  for this task (a separate task on the channel side); this review did not
  trace into those consumers because they are not part of this commit and
  the brief does not claim them for Task 13.

## Verdict

APPROVED. All six controller corrections (C1-C6) hold under direct
inspection and, where the review protocol calls for it, under mutation
testing. The two out-of-brief file edits are genuinely additive and
correctly declared. Column widths were independently re-verified against
the packet library rather than trusted. The one self-declared ordering
question is defensible and its residual gap (no dual-failure test) is
non-blocking.
