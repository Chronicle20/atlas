# Backend Audit — task-102-mts-marketplace (whole-branch, adversarial)

- **Diff range:** `6c6f52abfcdb…e3696ccf22f3` (merge-base BASE..HEAD)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*, anti-patterns, patterns-saga)
- **Date:** 2026-07-10
- **Build:** PASS — `go build ./...` clean in atlas-mts, atlas-saga-orchestrator, atlas-channel, atlas-cashshop, atlas-tenants; `go vet ./...` clean in atlas-mts.
- **Tests:** author-verified `-race` clean (not re-run per audit instructions; targeted probe only).
- **redis-key-guard:** PASS (exit 0 run correctly from repo root; the `GOWORK=off` FAIL is the documented false-positive).
- **Overall:** NEEDS-WORK — build/tests pass; one Important correctness risk (pre-existing platform primitive) plus minor deviations. No money-loss defect that is definitively introduced by this branch.

## Summary counts

- Critical: 0
- Important: 1
- Minor: 4

### Important (one-liners)
- **I1** — Every "atomic" money/custody composition in atlas-mts is built on `database.ExecuteTransaction`, which is a **no-op** (verified empirically): a mid-`fn` DB error leaves earlier writes committed → possible item loss / stranded bid escrow / orphaned history rows.

---

## I1 (Important) — Money/custody "atomic" compositions are not atomic (`ExecuteTransaction` no-op)

**Evidence — the primitive:** `libs/atlas-database/transaction.go:9-18`.
`isTransaction(db)` returns `true` whenever `db.Statement.ConnPool != nil`, which is the case for any root `*gorm.DB` after `db.WithContext(ctx)` (the ConnPool is the raw `*sql.DB`, i.e. autocommit — not a transaction). `ExecuteTransaction` therefore runs `fn(db)` directly and **never** calls `db.Transaction(fn)`.

**Empirically verified** in this worktree: a throwaway test using `atlas-mts`'s own sqlite harness created a row inside `ExecuteTransaction` and returned an error from `fn`; the row **survived** (`Statement!=nil:true ConnPool!=nil:true; NOT ATOMIC: row survived despite fn error (count=1)`). So the "compose N mutations in one tx, all-or-nothing" contract these handlers document does not hold.

**Where the branch relies on it (money path):**
- `services/atlas-mts/atlas.com/mts/listing/processor.go:512` — `transitionToSellerHolding` (Cancel/Expire): `UpdateState(active→terminal)` + `holding.CreateHolding` + `bid.UpdateState(held→released)`.
- `listing/processor.go:977` — `PlaceBid`: CAS advance + `bid.CreateBid` + prior-bid release-mark.
- `listing/processor.go:1154` — `SettleAuction`: CAS `active→settling` + winning-bid `held→won`.
- `listing/processor.go:440` — `ReleaseHighBidEscrow`: read + `bid.UpdateState(held→released)`.
- `services/atlas-mts/atlas.com/mts/kafka/consumer/custody/consumer.go:354` — `handleMtsMoveListingToHolding`: `UpdateState(active/settling→sold)` + `holding.CreateHolding` + two `transaction.CreateTransaction` rows.
- Also `custody/consumer.go:103, :242, :577, :614`.

**Concrete failure scenario:** In `transitionToSellerHolding` (seller Cancel of an auction with a live high bid): `UpdateState(active→cancelled)` autocommits, then `holding.CreateHolding` fails (constraint / connection blip). Because there is no real transaction, the listing is now `cancelled` with **no seller holding** and **no escrow release emitted** → the seller's item is gone and the high bidder's escrowed NX is stranded. The same class applies to `handleMtsMoveListingToHolding`: the listing can flip `sold` and then the holding/history insert fail, delivering nothing while the buyer's prepaid was already debited by the preceding saga step. The `errMoveLostRace` guard (`custody/consumer.go:402-412`) only covers the `affected==0` race arm — it does **not** cover a mid-`fn` insert error after a winning transition.

**Fairness caveat:** This is a **pre-existing, platform-wide** defect (`bug_execute_transaction_noop`; fix is task-119's `TxCommitter` check), not introduced by this branch, and it affects ~18 services. Its practical probability is low (requires a DB error between two writes inside one `fn`), and replay-idempotency guards + the saga compensator mask many partial-failure paths. But this is the money service, and the code's comments assert atomicity ("guarantees the cancel can never half-complete", "one local DB transaction") that is currently false. **Recommendation:** land task-119 before/with this branch, or add an explicit `db.Transaction(...)` wrapper at these call sites.

---

## Minor findings

### M1 (Minor, DOM-02/03 deviation) — no `Make(Entity)` / `Model.ToEntity()` in entity.go
Domain packages `listing`, `bid`, `holding`, `wish`, `transaction` have no `Make()` or `ToEntity()` in `entity.go`. Entity↔model mapping is done via `modelFromEntity` in `provider.go` (e.g. `listing/provider.go`) and an inline builder inside `administrator.go`'s create. Functionally equivalent and consistent with the wider Atlas convention, but diverges from `file-responsibilities.md`. No functional defect.

### M2 (Minor, DOM-05 deviation) — no `TransformSlice()` in rest.go
No `TransformSlice()` exists; the browse handler transforms via `model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())` (`listing/resource.go:244`). The rule's intent (no hand-rolled transform loops in resource.go) is satisfied; the literal helper is absent.

### M3 (Minor, DOM-17) — create-listing error mapping collapses DB errors to 400
`handleCreateListing` maps **every** `List()` error to `http.StatusBadRequest` (`listing/resource.go:141-145`), including an internal DB failure from `getActiveCountBySeller` (`listing/processor.go:643`). A transient DB error is reported to the client as a 400 validation failure rather than 500.

### M4 (Minor, cosmetic) — garbled comment
`listing/processor.go:687-690` has a dangling/garbled sentence ("The client previews (and the flat meso fee charged to the seller to create a listing.").

---

## Checks that PASS (with evidence)

- **Server-authoritative pricing (no client-trusted money):** Buy/PlaceBid/Settle read `listValue`/`buyNowPrice`/`currentBid`/`commissionRate` from the listing row, never from the caller — `listing/processor.go:798-813, :941-962, :1132`.
- **Debit-first settle & commission-as-sink:** `expandMtsSettlePurchase` orders buyer-prepaid debit (`-MarkedUpPrice`) first, then seller-points credit (`+ListValue`), then move; the delta stays the sink — `saga/processor.go` (expandMtsSettlePurchase), `listing/processor.go:836-856, :866`.
- **Escrow self-guard by bid state:** `ReleaseHighBidEscrow` releases only a `StateHeld` bid (winner is `StateWon` before the move) — `listing/processor.go:435-484`; sibling-offer release best-effort and idempotent — `listing/processor.go:389-422`.
- **Saga step completion / compensation coverage:** every MTS async action has a completion + reverse-walk inverse, plus a full late-compensation table (`lateCompensableActions`) with MTS custody inverses (`RestoreMtsHolding`, `RemoveMtsListing`, `RestoreListingFromHolding`) — `saga/compensator.go:1229-1294, :1357-1585`. Move-lost-race forces buyer-debit compensation — `custody/consumer.go:402-412`.
- **DOM-25 (config-resolved wire values):** `failNoticeOr` soft-resolves the semantic reason key from the tenant `noticeFailReasons` table with a documented fallback — `services/atlas-channel/.../kafka/consumer/mts/consumer.go:516-560` (the reference implementation cited in anti-patterns.md).
- **Multi-tenancy:** `tenant.MustFromContext` / `tenant.FromContext` used throughout; **no** manual `Where("tenant_id …")` in non-test code; holdings copy `tenant_id` from the listing row for the cross-tenant sweep — `listing/processor.go:529, :592`.
- **DOM-06/07:** processors take `logrus.FieldLogger`; handlers pass `d.Logger()`; no `logrus.StandardLogger()` in non-test code.
- **DOM-08/15/12/04(SUB):** all POST use `RegisterInputHandler[T]`; no `db.Create/Save/Delete` in any `resource.go`; no `os.Getenv` in handlers; no manual JSON decode.
- **DOM-10:** test DB registers tenant callbacks — `services/atlas-mts/.../test/database.go:28`.
- **DOM-11:** providers lazy (`database.Query`/`SliceQuery`, `model.SliceMap`).
- **DOM-21:** reuses `inventory.TypeFromItemId`, `item.Id`, `world.Id` (custody/consumer.go:127-129) rather than reinventing classification/id types.

## Scope note
This is a ~34k-line, multi-service branch. The audit concentrated (per instruction) on the money/correctness path in `atlas-mts` (listing/bid/holding/wish/transaction/wallet/saga + custody consumer), the orchestrator MTS expansion + compensator, and the channel DOM-25 surface, all read in full. The remaining channel ITC socket handlers, packet codecs (libs/atlas-packet), and configuration/mock parity were sampled, not exhaustively line-audited.

## Final resolution (post-review fixes)

- **I1 (Important) — DEFERRED to task-119 (merge-ordering decision, NOT fixed here).** `ExecuteTransaction` being a no-op is the pre-existing platform-wide bug (`bug_execute_transaction_noop`) affecting ~18 services; the fix is a shared `TxCommitter` primitive that must not be patched on a feature branch. atlas-mts's money compositions inherit the platform behavior. Recommendation carried to the PR: land task-119 with/before this branch, or merge with the non-atomicity documented as a known platform gap.
- **M4 (Minor) — FIXED.** Rewrote the garbled listing-fee comment in `listing/processor.go` (Step 1 debit).
- **M1 / M2 / M3 (Minor) — DEFERRED (convention-consistent / low-risk).** entity mapping via `modelFromEntity`, `model.SliceMap(Transform)`, and the `List()`→400 mapping are left as-is; noted for a future cleanup pass.
