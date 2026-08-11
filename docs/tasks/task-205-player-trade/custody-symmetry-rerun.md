# Task-205 — custody symmetry re-audit (post parity-plan Tasks 1–6)

Read-only re-run of `custody-symmetry-matrix.md` against the parity-plan's
fixes. Branch `task-205-player-trade`, commits `efec8332a..HEAD`
(`01106e56e` at time of writing). Paths are repo-relative. Shorthand as in the
original matrix:

| Short path | Real path |
|---|---|
| `trade/processor.go` | `services/atlas-trades/atlas.com/trades/trade/processor.go` |
| `trade/settlement.go` | `services/atlas-trades/atlas.com/trades/trade/settlement.go` |
| `escrow/*.go` | `services/atlas-trades/atlas.com/trades/escrow/*.go` |
| `saga/consumer.go` | `services/atlas-trades/atlas.com/trades/kafka/consumer/saga/consumer.go` |
| `orch/processor.go` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` |
| `orch/compensator.go` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` |
| `orch/timer.go` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go` |
| `saga/producer.go` (trades) | `services/atlas-trades/atlas.com/trades/saga/producer.go` |

## 0. Result

**No BLOCKING finding.** The three structural defences (A: in-flight netting,
B: single-claimant return latch, C: settlement-time custody check) now exist
on both the item and meso columns, and every read-then-act pair I traced on
the meso side is inside a single statement or a locked transaction, matching
the item column's shape. One **NON-BLOCKING** finding and two
**OBSERVATIONS** are recorded below; everything else the original nine-defect
audit and the parity plan named is either fixed and verifiably tested, or was
never actually a defect (see §4, "claims re-checked and found not to hold").

`go build ./...`, `go vet ./...` and `go test -race -count=1` are clean in
`atlas-trades` (`./trade/...`, `./escrow/...`) and `atlas-saga-orchestrator`
(`./saga/...`) as of this pass (§5).

## 1. What I checked, and how

I re-read every file the parity plan and original matrix cite, then walked
each of the seven areas in the assignment against the CURRENT code (not the
plan's prose, which is a claim to verify, not evidence):

1. Staging / netting in flight — `trade/processor.go` `addMeso`,
   `escrow/provider.go` `EffectiveMesoByOwner`/`InFlightMesoDelta`,
   `escrow/administrator.go` `ArmMesoStake`.
2. The return path and its claim — `escrow/administrator.go`
   `ClaimMesoForReturn`/`ClaimItemForReturn`, and all three unwind call sites
   in `trade/settlement.go` (`emitUnwind`, `unwindRecord`, `unwindStranded`).
3. Settlement — `trade/settlement.go` `settlementPayload`,
   `assertMesoCustodyAgrees`.
4. Resolution of in-flight movements — `escrow/administrator.go`
   `CommitMesoStake`/`AbandonMesoStake`, `trade/processor.go`
   `resolveMesoStake`.
5. Teardown, boot sweep, orphan handling — `trade/settlement.go`
   `ReconcileAtBoot`/`ReconcileEscrow`/`unwindStranded`, `trade/processor.go`
   `refundOrphanedStake`/`discardOrphanedMeso`.
6. Failed-unwind recovery — `trade/settlement.go` `UnwindFailed`/
   `UnwindSucceeded`, `escrow/administrator.go` `RestoreMesoRefunds`/
   `DiscardMesoRefunds`/`ReleaseItemReturnClaims`.
7. The compensating restore and its fence — `escrow/administrator.go`
   `RestoreItem`.

Then I went one layer further than the original audit into the
**orchestrator's own saga machinery**, because the parity plan's Task 5/6
fixes assume specific things about how `trade_unwind` fails and compensates
that neither prior audit traced end to end:

- Confirmed `BuildUnwind` (`saga/producer.go:96-103`, trades side) submits
  `trade_unwind` under `SagaType TradeTransaction`, **not** a distinct
  `TradeUnwind` saga type — `TradeUnwind` is only the composite **step
  action** `expandTradeUnwind` (`orch/processor.go:1646`) expands. This means
  a `trade_unwind` saga's step failures and timeouts route through the
  **same** `compensateTradeTransaction` / `DispatchTradeTransactionRollbacks`
  reverse-walk a settlement failure uses (`orch/compensator.go:295-297,
  1834-1974`), which already reverses completed `ReleaseFromTrade` (→
  `RestoreTradeEscrowAndEmit`), `AcceptToCharacter` (→ `RequestDestroyItem`)
  and `AwardMesos` (→ negated re-credit) steps **before** `SAGA_FAILED` is
  emitted for that transaction id. I traced this because it is the load-bearing
  fact behind whether `UnwindFailed`'s atlas-trades-side recovery
  (§2, Row 13) is redundant, wrong, or correct on a PARTIAL completion
  (some legs of one unwind landed, others didn't) — see §3, item (c).
- Confirmed `RestoreTradeEscrowAndEmit` (`orch/trade/processor.go:92-96`) is a
  **fire-and-forget Kafka command**, not a synchronous DB write, and traced
  the resulting cross-topic race between it and `SAGA_FAILED` landing at
  atlas-trades — see §3, item (a).
- Read `stepCompletedWithResultOnce`'s commit-time terminal gate
  (`orch/processor.go:576-590`) and `DispatchTradeStagingRollbacks`'s
  per-action switch (`orch/compensator.go:2448-2494`) to check whether the
  meso stage saga (`stage_meso`, single-step `AwardMesos`, also `SagaType
  TradeStaging`) has the same late-completion defence the two-step item stage
  has — see §3, item (b) / the one finding below.

I ran the actual test suites rather than trusting the parity plan's claimed
mutation results (§5), and spot-read three of the named tests
(`TestConcurrentMesoStakesConserve`'s siblings, `TestAFailedUnwindLeavesBoth-
ColumnsRecoverable`, `TestClaimMesoForReturnIsExclusive`) to confirm they
exercise the real store (`escrow.MesoByOwner(p.db, ...)`), not the fakeable
`escrowStore` seam the original audit flagged as a coverage risk.

## 2. Per-guarantee item vs. meso symmetry table

| Guarantee | ITEM | MESO | Symmetric? |
|---|---|---|---|
| **A — net in-flight before a new movement** | `StagedQuantityFrom` (`trade/processor.go:1251`) | `EffectiveMesoByOwner` = committed + Σ in-flight stake deltas (`escrow/provider.go:101-126`, read at `trade/processor.go:1328`) | **Yes.** Both are read through the real store, never the fake (`trade/processor.go:1324-1327` comment states this explicitly and matches). |
| **B — durable single-claimant return latch** | `ClaimItemForReturn`, CAS inside the UPDATE's WHERE (`escrow/administrator.go:122-130`) | `ClaimMesoForReturn`, `SELECT...FOR UPDATE` + zero in one transaction (`escrow/administrator.go:479-507`) | **Yes**, by different mechanics for the reason the code documents: an UPDATE's own WHERE can CAS a boolean-shaped claim; taking a whole signed total cannot be expressed that way, so it needs the explicit row lock. Both close the READ COMMITTED window (P1). |
| **B2 — claim ownership named, so a failed claimant can release exactly its own** | `ReturningTxId` (`escrow/entity.go:154-165`), `ReleaseItemReturnClaims(txId)` (`escrow/administrator.go:143-150`) | `MesoRefundEntity` (`escrow/entity.go:287-319`), `RestoreMesoRefunds(txId)` / `DiscardMesoRefunds(txId)` (`escrow/administrator.go:171-226`) | **Yes.** Both are transaction-scoped, both are written in the same DB transaction that performs the claim (`emitUnwind`/`unwindRecord`/`unwindStranded` all call `RecordMesoRefund` right after `ClaimMesoForReturn`, inside `p.emit`'s transaction — `trade/settlement.go:713-724, 1149-1160, 1511-1524`). |
| **C — settlement-time custody check** | `settlementPayload` errors when a staged item names no escrow row (`trade/settlement.go:573-577`) | `assertMesoCustodyAgrees` errors when anything is in flight OR the committed total disagrees with the room's figure (`trade/settlement.go:598-614`), called per side from `settlementPayload:564` | **Yes.** Both turn into a failed settlement via the same caller (`settle`, `trade/settlement.go:495-497`), which unwinds and returns everything on either error. |
| **Resolution of in-flight movements** | n/a (items have no in-flight commit step; accept is direct) | `CommitMesoStake` deletes the stake row (the CAS) and adds `delta` as a SQL expression in the same transaction (`escrow/administrator.go:403-431`); `AbandonMesoStake` deletes without touching `Amount` (`:441-447`) | No item twin needed — asset custody has no "delta" concept, it either exists in escrow or it doesn't. Not an asymmetry. |
| **Teardown / orphan return** | `emitUnwind`/`unwindRecord`/`unwindStranded` claim every row via `ClaimItemForReturn` before building the payload (`trade/settlement.go:638-652` `claimItemsForReturn`) | Same three functions claim every row via `ClaimMesoForReturn` (`:713, 1149, 1511`); the room-gone branch of `resolveMesoStake` uses `discardOrphanedMeso`, a **relative** subtract inside `DeleteResolvedMeso`'s own WHERE, never a read-then-assign (`trade/processor.go:1562-1570`, `escrow/administrator.go:580-593`) | **Yes.** F3/F5 from the original audit (pre-CAS misread, sweep-vs-orphan double refund) are both closed: nothing on the meso side reads a total and later assigns from it; every discharge is `amount ± X` inside one statement. |
| **Failed-unwind recovery** | `UnwindFailed` → `ReleaseItemReturnClaims(txId)`, scoped to non-deleted rows by GORM's default soft-delete scope, so an already-`DeleteItem`'d row (its `release_from_trade` completed) is correctly excluded (`escrow/administrator.go:143-150`) | `UnwindFailed` → `RestoreMesoRefunds(txId)`, which re-adds each recorded refund via `gorm.Expr("amount + ?", ...)` (`escrow/administrator.go:181-217`) | **Yes, and mutually consistent with the orchestrator's own reverse-walk** — see §3(c) for the trace confirming these two layers don't double-count. |
| **Restore fence** | `RestoreItem` fenced on `(returning_tx_id IS NULL OR returning_tx_id = txId)` — the RESTORING saga's own id, not merely "is claimed" (`escrow/administrator.go:269-276`) | n/a — meso has no soft-delete/restore concept; `ClaimMesoForReturn`'s row survives zeroed rather than being deleted, so there is nothing to "restore," only to re-credit via `RestoreMesoRefunds`, which is the mechanism above | Not a gap: the item column needs a restore because `DeleteItem` is destructive at the row level; the meso column's "delete" (`ClaimMesoForReturn`) only zeros a still-live row, so its inverse is an add, already covered above. |
| **DeleteResolvedMeso retirement** | `DeleteItem` is unconditional (nothing to condition on — a released item has nothing left to owe) | `DeleteResolvedMeso` retires only `amount = 0 AND NOT EXISTS` any stake, tested **inside** the DELETE's WHERE, not via a prior read (`escrow/administrator.go:580-593`) | **Yes** in effect: neither retires a row that still owes something (stake existence and non-zero amount are both re-checked at commit time by the DB, not by Go-side state read earlier), and nothing is left behind once `amount == 0` and no stake exists — see §4 for the one edge case (a stray persistent negative row) I traced and found to be hygiene-only, not a conservation defect. |

## 3. Load-bearing traces beyond the original audit's scope

### (a) `RestoreTradeEscrowAndEmit` is async; does the item claim/restore race `SAGA_FAILED`?

`RestoreTradeEscrowAndEmit` (`orch/trade/processor.go:92-96`) only enqueues a
`RESTORE_TRADE_ESCROW` Kafka command; it does not wait for it to land.
`compensateTradeTransaction` (`orch/compensator.go:1834-1871`) dispatches the
whole reverse-walk and then immediately transitions the saga to `Failed` and
emits `SAGA_FAILED` — there is no ordering guarantee between the two
different topics (`tradeCustody.EnvCommandTopic` vs. `sagaMsg.EnvStatusEvent-
Topic`).

Traced both interleavings:

- `RESTORE_TRADE_ESCROW` lands first: `RestoreItem` un-soft-deletes the row
  and clears `returning_at`/`returning_tx_id` in one UPDATE
  (`escrow/administrator.go:269-276`). `UnwindFailed`'s later
  `ReleaseItemReturnClaims(txId)` finds `returning_tx_id` already `NULL` — 0
  rows affected, correctly a no-op.
- `SAGA_FAILED` lands first: `ReleaseItemReturnClaims(txId)` runs against a
  row that is still soft-deleted (its `release_from_trade` already ran) —
  GORM's default scope excludes it, so this call is also a no-op.
  `RestoreItem`, when the `RESTORE_TRADE_ESCROW` command lands afterward,
  still fences correctly because its own WHERE re-checks `returning_tx_id`
  independently of whatever `ReleaseItemReturnClaims` did or didn't do.

Both orderings converge on the same correct end state (row live, unclaimed).
**No finding** — the fence is self-sufficient regardless of delivery order,
which is exactly what Task 6's design note ("the token is the restoring
saga's own transaction id") is for.

### (b) Meso stage's single-step saga and the late-completion race

`DispatchTradeStagingRollbacks` (`orch/compensator.go:2448-2494`) — the
reverse-walk both `compensateTradeStaging` (step-failure path) and the
timeout path (`orch/timer.go:191-195`, added by parity-plan Task 4) dispatch
for `SagaType TradeStaging` — switches only on `AcceptToTrade` and
`ReleaseFromCharacter`. It has **no case for `AwardMesos`**, which is the
sole step of a `stage_meso` saga (`saga/producer.go:81-88`, trades side, also
`SagaType TradeStaging`).

I traced whether this is reachable the same way Task 4's original defect was
(ordinary Kafka latency, no race needed) or requires genuine concurrency.
For a **two-step** item stage, Task 4's repro needs no race: step 1
(`release_from_character`) completes, the lifecycle stays `Pending` because
step 2 is still outstanding, and step 2 simply never arrives inside the
timeout window — ordinary latency. For the **one-step** `stage_meso` saga
there is no second step to wait on: the moment `AwardMesos` completes,
`stepCompletedWithResultOnce` marks it `Completed` and immediately calls
`p.Step`, which (finding no more pending steps) advances the lifecycle away
from `Pending` in the same call — so the "step completed, lifecycle still
Pending" window that `DispatchTradeStagingRollbacks` exists to catch does not
persist by ordinary latency for a single-step saga.

The only way to reach it is the narrower race `stepCompletedWithResultOnce`'s
own commit-time gate is built to close: `GetCache().GetLifecycle(...)` is
checked (`orch/processor.go:582`) before the step is marked; if the lifecycle
has *already* left `Pending` (the timeout won the CAS first), the step goes
through `absorbLateTerminal` instead, which — I confirmed by reading the
late-compensation switch at `orch/compensator.go:~2273` — **does** have an
unconditional `AwardMesos` case (`c.charP.AwardMesosAndEmit(...,
-payload.Amount, false)`), so a step landing *after* the lifecycle transition
is defended. The gap is only the interval between `GetLifecycle`'s read and
the subsequent `TryTransition`/`MarkEarliestPendingStepWithResult` calls in
the SAME function invocation — a true TOCTOU window on the order of a single
goroutine scheduling gap, not a Kafka round trip.

**Classified NON-BLOCKING.** It is a genuine asymmetry — the item legs of a
`TradeStaging` saga have an explicit reverse-walk case for this exact shape
of race and the meso leg does not — but I could not establish it is reachable
by anything short of a scheduler-level race inside one process invocation,
which is a different (and far weaker) reachability bar than every BLOCKING
finding in the original audit needed. Recorded here rather than escalated
because a `case AwardMesos:` arm would be nearly free to add (the negated
`AwardMesosAndEmit` call already exists verbatim in the late-compensation
switch) and would close the theoretical gap outright — worth doing as a small
follow-up, not worth blocking on.

### (c) Does the orchestrator's own reverse-walk double-count against `UnwindFailed`'s recovery?

Because `trade_unwind` runs as `SagaType TradeTransaction`
(`saga/producer.go:96-103`, trades side — confirmed by direct read, this was
not stated as a fact anywhere in the parity plan or original matrix), a
PARTIAL completion within one unwind (e.g., two `release_from_trade` steps
and one `award_mesos_refund` step complete, then a later
`accept_to_character` step fails) is reverse-walked by
`DispatchTradeTransactionRollbacks` **before** `SAGA_FAILED` is emitted:
completed `ReleaseFromTrade` steps are restored via `RestoreTradeEscrowAndEmit`
(un-soft-delete, claim cleared), and completed `AwardMesos` (refund) steps are
reversed via a negated `AwardMesosAndEmit` — i.e. the real in-game credit the
player received is deducted back out, restoring them to "never refunded."

Then atlas-trades' `UnwindFailed` runs on `SAGA_FAILED`:
`ReleaseItemReturnClaims` is a no-op for the already-restored items (per (a)
above, order-independent), and `RestoreMesoRefunds` re-adds the escrow claim
for the meso leg whose real payout was **just undone** by the orchestrator's
own reverse-walk. The two layers are consistent, not double-counting: one
undoes the real currency movement, the other restores atlas-trades' own
bookkeeping of what is still owed, and both target the same "never happened"
end state. **No finding** — but flagged because the parity plan's Task 5
write-up describes `UnwindFailed`'s recovery as though it were the *only*
mechanism putting things back ("the point is to return custody to a state the
ordinary paths can act on"), when in fact for `trade_unwind` specifically it
is the SECOND of two coordinated layers, the first being the standard
`TradeTransaction` reverse-walk it inherits by construction. Worth stating
explicitly in the design doc so a future reader doesn't assume `UnwindFailed`
is reversing real currency by itself (it never touches `AwardMesosAndEmit`;
it only touches the escrow bookkeeping row).

One residual gap I could **not** rule out from static reading: the reverse-
walk's individual inverse dispatches (`AwardMesosAndEmit`,
`RestoreTradeEscrowAndEmit`) are each wrapped in `claimTradeRollback` and, on
a dispatch error, are logged and skipped ("continuing chain",
`orch/compensator.go:1929-1936`) rather than retried or escalated. If the
negated `AwardMesosAndEmit` call for a completed refund leg fails to even
reach the Kafka producer, the real payout is never undone, while atlas-
trades' `RestoreMesoRefunds` unconditionally restores the escrow claim
anyway — that combination would mint. I could not determine from the code
alone how likely a producer-level emit failure is at that call site, and it
is not specific to task-205 (`AwardMesos`'s reverse-walk in
`DispatchTradeTransactionRollbacks` is shared with ordinary `trade_settlement`
failures and would have the identical exposure there). Recorded under §6 as
not verified rather than asserted.

## 4. Claims re-checked and found not to hold (or already closed)

- **Signed `Amount` narrowing to `uint32`.** Grepped every `uint32(...)` cast
  in `escrow/*.go` and `trade/*.go` (excluding tests) against every
  `.Amount()` call site. The only narrowing casts of a *stake's* `Amount`
  (`MesoStakeEntity.Amount`, already `uint32` — the player's absolute typed
  figure) are safe by construction. The confirmed total
  (`MesoEntity.Amount`, `int64`, signed) is never cast to `uint32` anywhere in
  production code — every consumer (`ReconcileEscrow`, `assertMesoCustody-
  Agrees`, `DischargeMeso`, `ClaimMesoForReturn`) compares or adds it as
  `int64`. **No finding.**
- **A remaining read-then-write pair on meso custody outside one statement or
  lock.** Walked every exported `escrow/administrator.go` function that
  mutates `MesoEntity`/`MesoStakeEntity`: `ArmMesoStake` (transaction, CAS via
  `OnConflict DoNothing` on the owner row + plain insert of the stake),
  `CommitMesoStake`/`AbandonMesoStake` (delete-is-the-CAS, relative SQL
  expression add), `ClaimMesoForReturn` (`SELECT...FOR UPDATE` + zero, one
  transaction), `DischargeMeso` (relative SQL expression, no prior read),
  `DeleteResolvedMeso` (condition inside the DELETE's WHERE). `UpsertMeso` is
  the one exception, but it is documented and used only by
  `retireEscrowMesos`/`clearRefundedMesos` as a blunt zero-assignment on a row
  the caller has already exclusively claimed by `ClaimMesoForReturn` moments
  earlier in the same transaction path — not a race window. **No finding.**
- **`RestoreItem`'s fence.** Re-derived both cases from the doc comment and
  confirmed against the code: unwind's own reverse walk (ids match, restores
  unlatched) and settlement's own release (unclaimed, restores) both pass;
  a stale restore from a different saga (ids differ) is refused. Cross-
  checked against §3(a)'s ordering trace. **No finding.**
- **`DeleteResolvedMeso` retiring an owing row / leaving a non-owing row
  behind.** Confirmed both conditions (`amount = 0` and `NOT EXISTS` a stake)
  are evaluated by Postgres inside the DELETE's own WHERE, not from a value
  read by Go beforehand, closing exactly the class of bug F3 was in the
  original audit. One edge case traced and found harmless: if `Amount` were
  ever left at a persistent (non-transient) negative value with no stakes
  outstanding — which the design says should not happen post-resolution, since
  every input is validated non-negative — the row would neither be swept by
  `ReconcileEscrow` (`m.Amount() <= 0` skip) nor retired by
  `DeleteResolvedMeso` (`amount = 0` only), and would sit forever. That is a
  hygiene leak (an unbounded row count in a pathological case), not a
  conservation defect — nothing is minted, destroyed, duplicated or stranded.
  Recorded as an OBSERVATION, not a finding.

## 5. Build / test verification run this pass

```
$ cd services/atlas-trades/atlas.com/trades && go build ./...
(clean)
$ go test ./trade/... ./escrow/... -race -timeout 180s
ok  	atlas-trades/trade	3.085s
ok  	atlas-trades/escrow	1.340s

$ cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...
(clean)
$ go test ./saga/... -race -timeout 240s -count=1
ok  	atlas-saga-orchestrator/saga	1.551s
ok  	atlas-saga-orchestrator/saga/mock	1.036s
```

Confirmed the following named tests exist and read the **real** store
(`escrow.MesoByOwner(p.db, ...)`, `p.db` directly), not the fakeable
`escrowStore` seam:

- `TestAFailedUnwindLeavesBothColumnsRecoverable`,
  `TestASucceededUnwindDiscardsItsRefundRecords`
  (`trade/processor_settlement_test.go:1826-1975`)
- `TestSettlementRefusesWhileAMesoStakeIsInFlight`
  (`trade/processor_settlement_test.go`, grep-confirmed present)
- `TestClaimMesoForReturnIsExclusive` (`escrow/administrator_test.go`,
  grep-confirmed present)
- `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout`,
  `TestTradeStagingTimeoutDispatchesItsReverseWalk`
  (`orch/saga/timer_test.go`, grep-confirmed present)

I did not re-run each test with its guard manually reverted (full mutation
re-verification of all ~15 named tests was out of budget for this pass); I
instead read the test bodies for the five above and confirmed each asserts
against the real DB row after the recovery call, which is what makes a
reverted guard capable of failing them. I did not do this for every test the
parity plan lists as mutation-verified — see §6.

## 6. Not verified / would need more than reading

- **Full mutation re-verification of every test the parity plan lists.** I
  spot-checked five (§5) rather than reverting all ~15 named guards myself.
  The ones I did not re-verify by hand: `TestSupersededMesoStakeFailureDoes-
  NotMint`, `TestRetypingTheMesoBoxMidSagaConservesMeso`, `TestAddMesoRefuses-
  WhenTheEscrowedTotalCannotBeRead`, `TestMigrationLiftsAnArmedStakeOutOfThe-
  OldSlot`, `TestReleaseItemReturnClaimsUnlatchesOnlyItsOwnTransaction`,
  `TestRestoreItemCannotResurrectAReturnedRow`, `TestRestoreItemStillUndoes-
  AnUnclaimedRelease`.
- **§3(c)'s residual gap**: whether a `claimTradeRollback`/dispatch-level
  failure on a completed `AwardMesos` refund leg (Kafka producer error at the
  reverse-walk call site) is reachable in the live fleet and, if so, at what
  rate. Static reading cannot bound this; it needs either a chaos test against
  the real Kafka producer or log-mining for `claimTradeRollback` /
  `AwardMesos reversal dispatch failed` in production.
- **§3(b)'s TOCTOU window**: whether `stepCompletedWithResultOnce`'s
  `GetLifecycle` read (`orch/processor.go:582`) and the timeout's
  `TryTransition` (`orch/timer.go:96`) can genuinely interleave in the live
  scheduler/cache implementation, or whether some lock I did not find
  serializes them. I read the code path but did not instrument or load-test
  it.
- **Kafka partition ordering** between `RESTORE_TRADE_ESCROW`
  (`tradeCustody.EnvCommandTopic`) and `SAGA_FAILED`
  (`sagaMsg.EnvStatusEventTopic`) — confirmed by reading they are different
  topics with no stated ordering guarantee; I did not check producer keying
  configuration for either to see if they happen to share a partition scheme
  that would incidentally order them (irrelevant to correctness here, since
  §3(a) showed both orderings converge, but noted for completeness).
