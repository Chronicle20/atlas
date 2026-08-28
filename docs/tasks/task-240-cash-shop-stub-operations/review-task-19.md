# Review: Task 19 — ring purchase transaction, REST surface, Kafka wiring

Commit range: `8806b742e..c05b2a750` (one commit,
`c05b2a750 feat(cashshop): implement atomic ring pair purchase (task-240 task 19)`)

Brief: `.superpowers/sdd/plan/task-19-brief.md`
Report: `.superpowers/sdd/plan/task-19-report.md`

## Scope

`git diff --stat 8806b742e..c05b2a750` touches exactly the files the brief
names: `cashshop/ring.go` (new), `cashshop/ring_test.go` (new),
`ring/rest.go` / `ring/resource.go` (new), `ring/rest_test.go` (new),
`kafka/message/cashshop/kafka.go`, `kafka/consumer/cashshop/consumer.go`,
`kafka/producer/cashshop/producer.go`, `main.go`, `cashshop/processor.go`
(one-line interface addition), and `context.md`. No scope drift.

## 1. The implementer's own concern — adjudicated

**Verdict: correctly dismissed. Non-blocking.**

The report flags that `partner locker full`, `insufficient funds`, and
`unknown commodity` all reject before any DB write, so none of them proves
mid-transaction rollback (buyer wallet debit / buyer asset already landed,
partner-side write then fails).

I read `cashshop/ring.go`'s statement order directly. All three of those
rejection reasons resolve in Steps 2-4 (`ci, err := p.comP.GetById(...)`,
character resolution, both-compartment capacity checks —
`cashshop/ring.go:105-148`), which run entirely *before* Step 5 (wallet
debit, `cashshop/ring.go:153-169`) and Step 6 (asset creates,
`cashshop/ring.go:176-185`). So yes: as coded, none of the three named
"creates neither" subtests can reach a state where a write has landed. The
report's own accounting is accurate.

The reachable post-write failure points I confirmed by reading the code are
real: `astP.CreateGift` (partner asset, `ring.go:181`), `ring.CreatePair`
(`ring.go:191`), and `purchaserecord.Record` (`ring.go:201`) can each fail
for a genuine DB-level reason (constraint violation, connection loss) *after*
the wallet debit and buyer-asset create have already executed against `tx`.
So a mid-transaction failure ordering is reachable in principle — the
question is whether it is protected.

I traced the transaction boundary rather than trusting the doc comment:

- `database.ExecuteTransaction` (`libs/atlas-database/transaction.go:9-14`)
  only opens a *new* `db.Transaction(fn)` when the handle is not already
  inside one (`isTransaction`, checked via `gorm.TxCommitter`); otherwise it
  runs `fn` directly on the existing `tx`. Every sub-processor `ring.go`
  constructs (`asset.NewProcessor`, `wallet.NewProcessor`,
  `compartment.NewProcessor`, `ring.NewProcessor`,
  `purchaserecord.Record`) is passed the SAME `tx` from the outer
  `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)` call at
  `ring.go:83`. There is no code path that writes through a different
  handle.
- `message.Emit` (`kafka/message/message.go:46-59`) propagates the inner
  closure's error unchanged (`err := f(b); if err != nil { return err }`),
  so `reject()`'s `errRingRejected` reaches the outer `db.Transaction(fn)`
  call as a non-nil return, which triggers gorm's standard rollback of
  every statement executed on that `tx` — including ones issued before the
  failing step.

**Reproduction (instruction 2).** I mutated `cashshop/ring.go` locally
(never committed) to insert an unconditional `return reject("FORCED_TEST_
FAILURE")` immediately after the buyer's asset create succeeds
(`ring.go:176-180`), before the partner's asset create runs. Result: `go
test ./cashshop/... -run TestPurchaseRing -v` turned every *happy-path*
subtest RED (`"[]" should have 1 item(s), but has 0"` on the buyer's own
compartment), i.e. the already-executed buyer asset create was NOT present
after the call returned. That is direct evidence the transaction wrapper
correctly discards a completed write when a later step in the same
`PurchaseRingAndEmit` invocation fails — the exact scenario the concern
raises. I reverted the mutation immediately after
(`git status --porcelain -- services/atlas-cashshop` clean, confirmed).

**Why still non-blocking.** I checked whether "mirrors gift_test.go" is a
lazy analogy or a real precedent, per the brief's explicit instruction not
to accept the analogy at face value. `cashshop/gift.go` also debits one
account's wallet and creates an asset in a *different* account's
compartment (sender pays, recipient receives) — the same cross-account
shape as ring. `cashshop/gift.go`'s own rejection ordering
(`CANNOT_GIFT_RECIPIENT_INVENTORY_FULL` at line 115, before the wallet debit
at line 131) has the identical "all checks precede all writes" structure,
and `gift_test.go` has the identical gap: no test forces a failure after the
wallet debit. So this is not a case where "the existing pattern never had a
two-account write to protect" — gift.go did, and the codebase's established
answer for it is (a) push all domain validation ahead of any write, and (b)
rely on the DB transaction wrapper plus a dedicated unit test at the
atomic-write primitive itself (`ring.TestCreatePairIsAtomic`,
`ring/administrator_test.go:181-208`, using the `failWritesForCharacter`
GORM-callback technique) rather than end-to-end failure injection in every
domain transaction. Given (1) I directly proved the rollback holds for this
exact code path, and (2) the design consistently follows an established,
repo-wide convention rather than a coincidental gap, this is not a
correctness defect.

**Non-blocking suggestion.** The `failWritesForCharacter`-style callback
technique already exists in `ring/administrator_test.go` and could be
adapted cheaply (target `cash_shop_asset`/`ring` writes keyed to the
partner) to pin this specific guarantee with a real regression test rather
than relying on my one-off manual mutation. Worth doing in a follow-up if
someone touches this file again, not worth blocking this commit on.

## 2. OQ-R1 compliance

**PASS.** `ring.go:174-181` and `ring.go:191-194` mint both halves from the
same resolved `ci.ItemId()` unconditionally — no `+1`, no gender-based
offset, no derived template id anywhere in the diff (grepped the whole
commit; the only per-half distinction is `CreateGift` vs `Create`, which
only affects `GiftFrom`/`GiftMessage`, not `TemplateId`).

There is deliberately no runtime detection of "this commodity needs
distinct halves" → typed `COUPLE_FAILED`/`FRIENDSHIP_FAILED` rejection. I
verified this is not a silently dropped requirement: `context.md` §7
(`docs/tasks/task-240-cash-shop-stub-operations/context.md:188-208`)
documents exactly why — no field reachable from `atlas-cashshop` (commodity
catalog, `item.Classification`, or elsewhere) distinguishes a couple-ring
commodity needing distinct halves from an ordinary same-template ring, so
writing that branch today would mean inventing the very detection rule
OQ-R1 forbids guessing at. The two operation labels
(`ErrorOperationCouple`/`ErrorOperationFriendship`) that a future
distinct-halves rejection would use already exist and are exercised by
every other rejection on this path, so the deferred branch can be added
later without touching the wire contract. This matches CLAUDE.md's bar for
a genuine external blocker (no data source exists to test the condition)
rather than an avoidable "documented gap." Non-blocking.

## 3. Idempotency via the shared Task 11 ledger

**PASS.** `ring.go:96` calls `ledger.Claim(p.ctx, tx, transactionId,
cashshop.CommandTypeRequestRingPurchase, characterId)` — the same
`ledger` package Task 11 introduced (confirmed via
`ledger/ledger.go:44` `func Claim(...)` and `ErrAlreadyProcessed`), not a
second mechanism. `ErrAlreadyProcessed` short-circuits to `return nil`
(success-without-effect) before any read or write, matching `gift.go`'s
convention.

The `replay creates one pair` subtest
(`cashshop/ring_test.go:339-368`) actually proves it: it calls
`PurchaseRingAndEmit` twice with the same `transactionId` and asserts
exactly one buyer asset, one partner asset, one pair (matched `PairId`s),
and a wallet credit of `2500` (not `5000`, i.e. not double-charged) — the
claim genuinely gates the second call rather than the test being a no-op
either way.

## 4. Multi-tenancy

**PASS.** Both REST handlers are tenant-scoped:
- `handleGetRings` (`ring/resource.go:51-91`) resolves
  `t := tenant.MustFromContext(d.Context())` and passes `t` into
  `byCharacterIdPagedProvider`, whose query is
  `db.Where("tenant_id = ? AND character_id = ?", t.Id(), characterId)`
  (`ring/resource.go:26-30`).
- `handleGetRing` (`ring/resource.go:93-117`) calls
  `GetById(db.WithContext(d.Context()), t.Id(), ringId)`.
- The underlying queries (`ring.GetByCharacterId`, `ring.GetById`,
  `ring.CreatePair` via `ProcessorImpl.CreatePair` →
  `CreatePair(db, p.t.Id(), ...)`, `ring/processor.go:43-44`) are all
  pre-existing Task 18 code, unmodified here, and already filter on
  tenant id.

Non-blocking note: `ring/rest_test.go` only unit-tests `Transform`/
`RestModel` (`GetName`/`GetID`/`SetID`), not the HTTP handlers, so there is
no test proving cross-tenant isolation at the REST layer specifically. This
matches the existing convention in this module (`wishlist/resource.go` has
no `resource_test.go` at all), so it is not a regression against the
pattern being copied, but it is a real gap in what's actually proven by a
test versus by code inspection.

## 5. Kafka contract

**PASS**, exact match to the brief's interfaces:
- `CommandTypeRequestRingPurchase = "REQUEST_RING_PURCHASE"`
  (`kafka/message/cashshop/kafka.go:23`)
- `RequestRingPurchaseCommandBody` (`kafka.go:154-162`) — field set matches
  the brief exactly (`TransactionId`, `Currency`, `SerialNumber`,
  `PartnerCharacterId`, `SenderName`, `Message`, `RingType`).
- `StatusEventTypeRingPurchased = "RING_PURCHASED"` (`kafka.go:176`)
- `RingPurchasedBody` (`kafka.go:355-363`) — field set matches the brief
  exactly (`TransactionId`, `CompartmentId`, `AssetId`, `PartnerName`,
  `TemplateId`, `Quantity`, `RingType`, `PairId`).
- Handler registered: `consumer.go:77` wires
  `handleCommandRequestRingPurchase(db)` on `CommandTypeRequestRingPurchase`;
  `consumer.go:233-243` guards on `c.Type` and delegates to
  `PurchaseRingAndEmit`, logging (not re-emitting) on error, matching every
  other handler in the file.
- Producer wired: `producer.go:164-181`
  `RingPurchasedStatusEventProvider` builds the exact `RingPurchasedBody`,
  called from `ring.go:207`.
- `main.go:148` registers `ring.InitResource(GetServer())(db)`;
  `main.go:65` adds `ring.Migration` to the migration set (pre-existing from
  Task 18, unchanged here — confirmed already present).

## 6. FR-RING-3 — partner asset carries GiftFrom/GiftMessage

**PASS.** `ring.go:181`: `astP.CreateGift(buf)(partnerCcm.Id(), ci.ItemId(),
serialNumber, effectiveCurrency, ci.Count(), 0, characterId, senderName,
ringMessage)` — `CreateGift`'s signature
(`cashshop/inventory/asset/processor.go:141`) threads `giftFrom`/
`giftMessage` into `create(...)` (`administrator.go:26`), the same Task 13
columns `gift.go` writes through. The buyer's own asset correctly uses
`astP.Create` (no gift fields) since the buyer isn't receiving a gift from
anyone.

## Standard backend expectations

- No `*_testhelpers.go` files; test setup uses inline builders/seed helpers
  matching `gift_test.go`/`processor_test.go` conventions
  (`ring_test.go:44-118`).
- No new domain constant invented outside `libs/atlas-constants`; reused
  service-local `ring.Type`/`ring.State` and existing
  `ErrorOperationCouple`/`ErrorOperationFriendship`.
- `RestModel`/`Transform` follow the existing JSON:API accessor pattern
  (`ring/rest.go`), consistent with other REST resources in this module.

## Verification performed

- `go build ./...` — clean, both before and after my temporary mutation
  (mutation reverted).
- `go test ./cashshop/... -run TestPurchaseRing -v` — all 7 subtests PASS
  on the unmodified commit.
- Confirmed working tree clean after experimentation:
  `git status --porcelain -- services/atlas-cashshop` returns nothing.

## Not evaluable

None — the review surface (the single commit's diff, plus `ring.go`'s
transaction boundary and the `database.ExecuteTransaction` /
`message.Emit` contracts it depends on) was fully traceable from the repo.
