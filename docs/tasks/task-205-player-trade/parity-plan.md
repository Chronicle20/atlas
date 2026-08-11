# Task-205 follow-on — meso custody parity

**Status:** COMPLETE. Prerequisites P1 and P2 settled; Tasks 1–7 done. Both
audits re-run independently: symmetry 0 BLOCKING, conservation 1 BLOCKING (F-8)
which is fixed and mutation-verified. Full gate green.

Written 2026-08-11 as a handoff from the session that produced commits
`a3279ee73`..`0bf941fc5`.

**Read first:** `custody-symmetry-matrix.md` and `conservation-audit.md` in this
folder. They are the evidence for everything below and were produced by two
independent read-only audits with different lenses. This plan is the response to
them; it does not restate their traces.

---

## Why this pass exists

Nine defects were found on this branch. The first was the reported bug. **Every
one after it lived on the same seam:** trade custody was built as two parallel
mechanisms — items and meso — and a guarantee kept being implemented on the item
side and not the meso side.

The symmetry audit reduced the six remaining blocking findings to **one design
gap with six symptoms**. The item column has three structural defences the meso
column lacks:

| | Defence | Item implementation | Meso |
|---|---|---|---|
| **A** | Net what is already in flight before submitting a new movement | `StagedQuantityFrom` | **absent** |
| **B** | A durable single-claimant latch on the return path | `returning_at` + `ClaimItemForReturn` | **absent** |
| **C** | A settlement-time check that the custody row still exists | `settlementPayload` errors on a missing row | **absent** |

Fixing the six symptoms individually is what produced defects five and nine.
**This pass closes the three gaps and then re-derives whether the symptoms are
gone**, rather than patching each symptom.

## What "done" means

1. The three defences exist on both sides, or an asymmetry is documented with a
   reason rooted in a real difference between an asset and a balance.
2. Both audits re-run against the result and report no BLOCKING finding.
3. Every guard added is **mutation-verified**: remove it, watch a named test
   fail with a message that states the consequence, restore, green.

---

## Prerequisites — two facts nobody could establish from the code

Both audits flagged these as UNVERIFIED and both gate real decisions. Settle
them **before** designing the fixes; the answers change what Task 2 has to do.

**Both are now settled (2026-08-11). Neither answer lets any task shrink.**

### P1 — transaction isolation: **READ COMMITTED** ✅

Live Postgres is **18.4**; `SHOW default_transaction_isolation` returns
`read committed`. Nothing in the repo overrides it:

- `libs/atlas-database/transaction.go:13` — `ExecuteTransaction` calls
  `db.Transaction(fn)` with **no** `*sql.TxOptions`, so GORM passes the driver
  default through.
- `DSNBuilder.Build()` (`libs/atlas-database/connection.go:55`) emits only
  `host/user/password/dbname/port/sslmode/TimeZone` — no `options=` clause.
- A tree-wide grep for `TxOptions`, `LevelSerializable`, `Serializable`, and
  `default_transaction_isolation` matches nothing outside this document.

**Consequence:** every read-then-CAS window the audits flagged in the meso
paths is real. Task 2 does **not** get to collapse any of them — each decision
that ends meso custody must read its inputs in the same statement that acts on
them (single-statement CAS or an explicit row lock), not in a prior `SELECT`.
This applies directly to the `discardResolvedMeso` pre-CAS `row.Amount()`
misclassification.

### P2 — `TRADE_CONFIRM` is **not** excl-gated ✅ — Task 3 is BLOCKING

Read from the v83 IDB (`GMS/v83_Me/MapleStory_dump.exe.i64`). All three trade
sends use serverbound opcode **123 (0x7B)** and differ only in the mode byte:

| Mode | Function | `CanSendExclRequest`? | Arms the excl latch? |
|---|---|---|---|
| `0x0F` put item | `CTradingRoomDlg::PutItem` @ `0x7c359f` | **yes** — early-returns 0 | yes |
| `0x10` put money | `CTradingRoomDlg::PutMoney` @ `0x7c37ca` | **yes** — whole body is inside it | yes |
| `0x11` confirm | `CTradingRoomDlg::Trade` @ `0x7c39a0` | **no — absent entirely** | no |

`CWvsContext::CanSendExclRequest` @ `0x485bf7` is
`!this[2089] && (…) && get_update_time() - this[2090] >= a2`; both staging
functions set exactly those two fields immediately after `SendPacket`
(`v24[2089] = 1; v13[2090] = get_update_time()`). So they are genuine excl
requests. `Trade` builds its packet, sends it, and touches neither field — it
is gated only by the `SP_413_ARE_YOU_SURE_YOU_WANT_TO_TRADE` confirmation
dialog.

**Consequence:** the client's excl latch serializes staging against *staging*,
but imposes **no ordering at all between staging and confirm**. An unmodified
v83 client can send `TRADE_CONFIRM` with a meso stake still in flight — the
player only has to click through a modal. Task 3's window is reachable in
normal play, so **Task 3 is BLOCKING, not defence-in-depth.**

Two secondary results worth carrying into the tasks:

- The item and meso staging paths have **identical** client-side gating. The
  asymmetry this whole pass is fixing is therefore purely server-side; no
  proposed fix can be justified by "the client protects the item column."
- P2 independently corroborates Task 1's mechanism. `PutMoney` arms the latch
  and the debit's own response clears it, which is exactly what lets a second
  `PutMoney` supersede the first.

---

## Verification discipline for this pass

The previous session's gate passed while the branch was badly broken. These are
not optional:

- **Mutation-verify every guard.** A test that cannot fail against the bug is
  not a test of it. Paste the failing and passing output in the commit or report.
- **A stubbed seam is zero coverage.** Two defects shipped behind tests that
  stubbed the very seam under test.
- **Re-read tests that assert the behaviour you are changing.** One defect
  shipped because the existing tests pinned the old contract, so green meant
  "still does the old thing".
- **Trace every event into its consumer.** Nothing links across module
  boundaries; three defects lived in seams between services.
- **Verify each finding yourself before fixing it.** The findings below are
  relayed from the audits. Two were independently confirmed (marked ✅); the
  rest were not. Read the code and confirm the trace before changing anything —
  fixing a misdiagnosed defect is how two of the nine were created.

---

## Tasks

### Task 1 — Defence A: net the in-flight stake before submitting a new one ✅ *confirmed* — **DONE**

**The defect.** `addMeso` derives its delta from `escrow.MesoByOwner`, which
returns only the committed `Amount` column
(`escrow/provider.go` — `return e.Amount, true, nil`). The in-flight stake is
ignored, and `ArmMesoStake` overwrites it, so the superseded stake's terminal
status matches no row and is dropped — but its `award_mesos` already moved real
meso.

Type 100 (debit 100, stake armed) → the debit's own `STAT_CHANGED` unlocks the
client → type 200 (delta computed as 200−0, debit 200, stake overwritten).
Debited 300, escrow says 200, **100 destroyed**.

`Participant.PendingMesoTxId()` and `PendingMesoAmount()` exist for exactly this
and have **no production callers** — confirmed by grep.

**The fix — DONE.** Chosen shape: **compose, and resolve every stake
independently.** Recorded here because the alternative (refuse while one is in
flight) was live and rejected: it re-echoes a stale number over what the player
just typed, which is why supersession was chosen originally.

The decisive arithmetic: if each stake resolves on its own — success adds its
own delta, failure adds nothing — then the committed total is always the sum of
the deltas that actually moved, and conservation holds with no "superseded"
special case at all. The defect was never supersession as such; it was that
resolution keyed on a SINGLE slot, so a superseded stake's success was dropped
though its debit had moved real meso.

That makes the pending slot a child table. Implemented as:

- `trade_escrow_meso_stakes` — one row per in-flight stake, replacing
  `pending_stake_id`/`pending_amount`/`pending_delta`. The migration BACKFILLS
  any armed slot into it before dropping the columns; dropping them outright
  would strand a debit that has already left a player's pocket.
- `CommitMesoStake` claims by DELETEing the stake row and adds `delta` to the
  committed total as a SQL expression. The delete IS the compare-and-set, so
  redeliveries are inert; the relative add is what lets siblings resolve in any
  order without clobbering each other. **P1 forced this**: under READ COMMITTED
  a read-then-assign loses whichever sibling committed in the window.
- `EffectiveMesoByOwner` (committed + Σ in-flight deltas) is what `addMeso` nets
  against — the meso twin of `StagedQuantityFrom`.
- The committed column is now **signed and widened**. Stakes resolve in status
  order, not arm order, so a player who types 1000 then 500 arms +1000 and −500,
  and if the reduction lands first the total passes through −500 legitimately.
  Held unsigned it underflowed. Refund paths treat non-positive as nothing owed.
- `discardResolvedMeso` → `discardOrphanedMeso`, which discharges by a RELATIVE
  subtraction instead of deciding from a pre-CAS read and assigning. That is
  **Task 2's pre-CAS misclassification, fixed here** because the signature had
  to change anyway and leaving a known-broken guard was not an option.

**Seam note, worth carrying forward.** The netting read deliberately goes
through the same real escrow processor that arms the stake, NOT the fake-able
`escrowStore` seam. Arming already bypassed that seam, so a fake answering the
read would have reported nothing in flight and silently restored the bug — a
green test proving nothing. For the same reason the six test call sites that
seeded committed escrow into the fake alone now seed both stores, and
`databasetest.FailReadsOn` (new, the counterpart of `FailWritesOn`) injects the
unreadable-escrow failure at the real store.

**Tests — all mutation-verified** (guard removed → named test fails with a
message stating the consequence → restored → green):
- `TestConcurrentMesoStakesConserve`, `TestSupersededMesoStakeFailureDoesNotMint`
  (escrow layer: conserves on success, does not mint on superseded failure).
- `TestRetypingTheMesoBoxMidSagaConservesMeso` (end to end). Mutation: reverting
  to the committed-only delta yields "the two stakes debited 300000 in total,
  but the player only ever typed 200000".
- `TestAddMesoRefusesWhenTheEscrowedTotalCannotBeRead` (re-pointed at the real
  store).
- `TestMigrationLiftsAnArmedStakeOutOfTheOldSlot`. Mutation: dropping the
  backfill yields "the armed stake was dropped with the columns; the player's
  debit is stranded".
- `TestArmMesoStakeSupersedesPriorStake` was **deleted and replaced**. It pinned
  the defect as a contract — asserting the superseded stake's commit must report
  false. Its replacement is `TestArmMesoStakeKeepsPriorStakeOutstanding`.

### Task 2 — Defence B: a durable claim latch for meso rows — **DONE**

Both halves are now closed. The `discardResolvedMeso` half went with Task 1: it
is `discardOrphanedMeso` and discharges by relative subtraction rather than
deciding from a pre-CAS read and assigning, and `CommitMesoStake` no longer
assigns at all. The second half — the two return paths racing one row — is
below.

**The defect (relayed; the racing-paths trace confirmed by reading all three
unwind call sites).** The boot sweep and the orphaned-stake
refund both act on the same meso row with no arbitration; the item twin claims
on both paths via `returning_at`.

**The fix — DONE.** `ClaimMesoForReturn` is the meso twin of
`ClaimItemForReturn`: it takes the participant's whole escrowed total and
reports what THIS caller won, zero if another path got there first. All three
unwind paths (`emitUnwind`, `unwindRecord`, `unwindStranded`) now CLAIM instead
of reading a total and zeroing afterwards.

Two implementation notes that were not free choices:

- **It is a row lock, not `RETURNING`.** An UPDATE's `RETURNING` yields the row
  as it stands AFTER the assignment — the zero — not the amount taken. So the
  claim does `SELECT … FOR UPDATE` and zeroes inside the same transaction; a
  competitor blocks on the lock and then reads the zero. **P1 settled the
  isolation as READ COMMITTED, so this window was real** and no read-then-act
  pair could be left unfenced on the grounds that isolation closes it.
- **In `unwindStranded` the field lookup happens BEFORE the claim.** Claiming
  zeroes the row, so a `FieldOf` failure after it would drop the only record
  that the meso is owed. Leaving the row unclaimed hands it to the next sweep
  intact.

`clearRefundedMesos` became `retireClaimedMesos` and **no longer zeroes** — the
claim already did, under the lock. Re-assigning zero afterwards was worse than
redundant: a sibling stake committing its delta in between would have had that
delta silently clobbered. Only the conditional `DeleteResolvedMeso` remains.

**Tests.** `TestClaimMesoForReturnIsExclusive` (mutation-verified: drop the
zeroing and the second claim takes 5000 again, "the player would be refunded
twice") and `TestClaimMesoForReturnIgnoresANonPositiveRow`.

**Honest limit on the verification, stated rather than papered over.** The
double refund needs two transactions interleaving a read and a write.
`databasetest.NewInMemoryTenantDB` caps sqlite at `MaxOpenConns(1)`, so
transactions cannot interleave there and a concurrency test could not fail
against the bug — which by this plan's own rule makes it not a test of it. The
guarantee is therefore pinned at the level where it is genuinely decidable: the
claim is exclusive, proven by two sequential claims. The interleaving itself is
closed by the row lock, which is a Postgres behaviour this harness cannot
exercise.

### Task 3 — Defence C: a settlement-time custody check for meso — **BLOCKING** (P2) — **DONE**

**Reachability confirmed by P2:** the v83 client applies no excl gate to
`TRADE_CONFIRM`, so a stake genuinely can be in flight at CONFIRM. This is not
defence-in-depth.

**The defect (trace relayed, unconfirmed).** Nothing gates CONFIRM or `settle`
on a pending meso stake. `settlementPayload` uses `pt.MesoStaged()`, which only
advances when a stake resolves, then `dischargeSettledMesos` zeroes the row.
A reduction in flight delivers more than the giver still has staked (**mints**);
a raise resolving after the room is SETTLING is **destroyed**. Which one occurs
is timing. The item twin is protected: `settlementPayload` errors when a staged
item has no escrow row.

**The fix — DONE.** `assertMesoCustodyAgrees`, called from `settlementPayload`
per side, refuses to build a payload when either (a) anything is still in
flight, or (b) the committed total disagrees with what the room is about to
deliver. The caller turns the error into a failed settlement, which unwinds and
returns everything — the same outcome the item branch already produces.

**Gated at `settle`, NOT at CONFIRM, and the reason is in the client.**
`CTradingRoomDlg::Trade` sets its own confirmed flag (`v1[111] = 1`) and
disables both trade buttons BEFORE it sends, and no server packet re-enables
them short of leaving the room. Refusing the confirm would therefore wedge the
dialog. Failing the settlement is recoverable and symmetric with items.

**Tests.** `TestSettlementRefusesWhileAMesoStakeIsInFlight`, table-driven over
both directions. Mutation-verified — with the gate removed, both cases report a
settlement saga submitted while custody was still moving:

    a settlement saga was submitted with -1500 meso still in flight …
    a settlement saga was submitted with  1500 meso still in flight …

`TestSettlementSuccessKeepsARowWhoseStakeIsStillInFlight` was re-documented
rather than deleted: the gate makes its state unreachable through the normal
flow, but it still pins the DISCHARGE behaviour for a stake that outlives its
room by another route, which the orphan path depends on.

### Task 4 — register `TradeStaging` in the timeout reverse-walk ✅ *confirmed* — **DONE**

**The defect.** `saga/timer.go` dispatches its rollback for exactly three saga
types — `CharacterCreation`, `MtsOperation`, `TradeTransaction`. Both staging
sagas are typed `TradeStaging` (`atlas-trades/saga/producer.go`). Every saga
gets a 30s backstop. So `release_from_character` completes, `accept_to_trade`
stalls past it, **nothing rolls back**: compartment −1, escrow +0, item
destroyed. The meso twin destroys the debit the same way. No race — a slow
consumer suffices. The step-*failure* path is handled correctly; only the
timeout is missing.

**Note the trap this sits in.** `TradeStaging` was introduced deliberately, with
a comment explaining that reusing `TradeTransaction` would send a staging
failure through the swap's pairing logic and destroy the item. A new saga type
must be registered everywhere saga types are enumerated. **Grep for every
`SagaType()` comparison in the orchestrator** and check each one, rather than
fixing only `timer.go`.

**The fix — DONE, and not by adding a fourth `if`.** The chain of independent
`if s.SagaType() == …` statements IS the defect shape: it is an enumeration
with no way to notice an omission. It is now one `switch` in
`dispatchTimeoutRollbacks`, driven by a named `reverseWalkSagaTypes` list that
the test reads too. `DispatchTradeStagingRollbacks` already existed — the
timeout path simply never called it.

**Every `SagaType()` comparison was checked FOR WHETHER IT HANDLES
`TradeStaging`** — which is narrower than an earlier draft of this section
claimed, and the difference matters. Asking "does this site handle
`TradeStaging`" is not the same as asking "which OTHER types have a bespoke
reverse walk but no timeout arm", and only the first question was asked.

The plan-adherence review asked the second one. **Eleven saga types have a
bespoke step-failure compensator; `reverseWalkSagaTypes` covers four.** The
seven uncovered — `PetEvolution`, `ItemTagUse`, `SealingLockUse`,
`IncubatorUse`, `PointReset`, `NoteSend`, `SkillBookUse` — each have the
identical unrolled-timeout bug this task exists to close, since every saga gets
the same 30s backstop. All seven already had a pure `Dispatch…Rollbacks` half
written (`compensator.go:1230, 1324, 1428, 1559, 1658`), so **all seven were
wired in** rather than deferred — a decision put to the user, who chose to fix
them now rather than ship a trade PR alongside seven live instances of the class
it exists to close.

The guard was also strengthened, because the review exposed a second gap: a test
that ITERATES `reverseWalkSagaTypes` can only catch a type present in the list
but missing from the switch. A type nobody listed is invisible to it — which is
exactly how the seven survived. `noReverseWalkSagaTypes` now names the five
types that deliberately have no whole-saga walk, `allSagaTypes` names all
sixteen, and `TestEverySagaTypeIsClassified` fails when the three disagree. So
adding a saga type without deciding its timeout behaviour now fails a test
instead of silently destroying value on the backstop. Both layers are
mutation-verified.

The per-site results for the `TradeStaging` question:

| Site | Handles `TradeStaging`? | Verdict |
|---|---|---|
| `saga/timer.go` | **no → FIXED** | the defect |
| `saga/compensator.go:305` | yes | step-failure path, already correct |
| `saga/error_mapper.go:13` | no → falls to `ErrorCodeUnknown` | **left alone deliberately**: `compensateTradeStaging` already emits `ErrorCodeUnknown` explicitly, so the two agree, and the code is diagnostic detail on the FAILED event, never a branch the client sees |
| `saga/producer.go:28,34,105,184` | n/a | enrich the emitted event with type-specific ids; nothing at risk for staging |
| `saga/processor.go:814` | n/a | `CharacterCreation && Completed` only |
| `saga/compensator.go:252-320, 2114, 2284, 2303` | yes at 305 | the rest are other types' bespoke arms |

**Tests.** `TestTradeStagingTimeoutDispatchesItsReverseWalk` names the defect,
and `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout` guards the CLASS —
it iterates `reverseWalkSagaTypes`, so it fails for the next type added to the
compensator and forgotten here. Mutation-verified: dropping the `TradeStaging`
arm fails both, with "compartment -1, escrow +0, item destroyed".

### Task 5 — route a failed `trade_unwind` — **DONE**

**The defect (relayed, unconfirmed).** A `trade_unwind` transaction id owns none
of the three id spaces the saga-status consumer probes, so `SAGA_FAILED` for an
unwind falls through all three and is swallowed. Items stay latched by
`returning_at` — cleared only on a completed release — so they are **stranded
permanently and invisible to every future boot sweep**. Meso was already zeroed
in the same transaction that submitted the unwind, so it is **destroyed with no
record**.

**Confirmed.** `completeSettlement` treats an id that names no settlement record
as already-resolved and returns nil (`trade/settlement.go`), so the failure was
swallowed with a debug line. There is also no `TradeUnwind` arm in the
orchestrator's compensator, so it falls to `compensateStorageOperation`, which
performs no rollback.

**The open question was put to the user and answered: keep
discharge-before-confirm, and make the failure routable.** The alternative —
discharging only on the unwind's terminal success, which would have subsumed
Task 2 by turning the meso claim into a latch — was presented and not chosen.

**The fix — DONE.** The blocker was that nothing recorded WHAT an unwind
claimed: `returning_at` was a bare timestamp, and after Task 2 the meso is
already zeroed. So:

- `ItemEntity.ReturningTxId` names the unwind holding each latch, and
  `ClaimItemForReturn` stamps it. The unwind's id is now minted BEFORE anything
  is claimed, at all three sites.
- `MesoRefundEntity` (`trade_escrow_meso_refunds`) durably records what an
  unwind took from each participant, written in the same transaction that takes
  it. It has to carry the AMOUNT, not just the claim, because the meso claim is
  destructive — once submitted, nothing else names the figure that was owed.
- `UnwindFailed` releases that transaction's latches and adds back its recorded
  amounts (relative, so a stake committing meanwhile is not clobbered; the row
  is re-created if the retire removed it). It re-submits nothing: the point is
  to return custody to a state the ordinary paths can act on.
- `UnwindSucceeded` discards the records. Without it they accumulate, and a
  redelivered FAILED event — which at-least-once permits — would restore meso
  already paid out, minting it.
- A fourth probe in the saga-status consumer, **ordered before the settlement
  probe**, because that one swallows unknown ids and would make an unwind look
  handled.

**Tests, all mutation-verified.**
`TestAFailedUnwindLeavesBothColumnsRecoverable` (including the redelivery),
`TestASucceededUnwindDiscardsItsRefundRecords`, and
`TestReleaseItemReturnClaimsUnlatchesOnlyItsOwnTransaction` — the last pins the
scoping, and widening it to the tenant releases 2 rows instead of 1, which would
grant another unwind's item twice.

### Task 6 — fence `RestoreItem` against redelivery — **DONE**

**The defect (relayed, unconfirmed).** `RestoreItem` clears `deleted_at` and
`returning_at` for whatever row bears the id, with no fencing token. A
redelivered `RESTORE_TRADE_ESCROW` arriving after the unwind already released
the row and granted the item resurrects it live and unclaimed, and the next boot
sweep grants it again. `CreateItem` has an `OnConflict…DoNothing` guard for
exactly this at-least-once posture; restore has none. The existing doc comment
argues this is impossible, but its reasoning covers only the reverse walk's own
ordering, not a redelivery racing a different saga's release —
**treat that comment as suspect, not as evidence.**

**Confirmed, and the doc comment WAS wrong.** Its argument covered only the
ordering within one reverse walk. The reachable sequence needs nothing beyond
Kafka's own at-least-once guarantee: a failed settlement's reverse walk emits
RESTORE for row X; the failed settlement's unwind then claims X, releases it,
and the owner is granted the item; the restore is redelivered and resurrects X
live and unclaimed for the next boot sweep to grant again.

**The fix — DONE, and the first fence attempt was wrong.** Fencing on "the row
is claimed" broke a legitimate case that an existing test correctly pinned: an
unwind whose release completed and whose accept then FAILED must be restored,
and that row is claimed too. Both cases carry a claim, so the claim alone cannot
separate them.

The discriminator is WHOSE claim it is. `RestoreItem` now takes the restoring
saga's transaction id — which the command already carried — and restores only
when the row is unclaimed or claimed by that same saga:

- unwind's own reverse walk → ids match → restores, unlatched. Correct: the item
  was never granted.
- stale restore from a different saga → ids differ → inert.
- settlement's own release → unclaimed → restores. Unaffected, which is why the
  fence could not key on "was soft-deleted".

**A latent defect surfaced while doing this**, introduced by Task 5:
`returnOrphanedStage` stamped its claim with one id and submitted the unwind
with a different one, so neither the Task 5 recovery nor this fence could ever
find that claim. Both now use a single value.

**Tests.** `TestRestoreItemCannotResurrectAReturnedRow` and
`TestRestoreItemStillUndoesAnUnclaimedRelease`, plus the pre-existing
`TestRestoreItemReleasesTheReturnClaim`, which was corrected to share the saga
id as the real flow does. Mutation-verified: dropping the fence resurrects the
returned row.

### Task 7 — re-run both audits, then the full gate — **DONE**

Both re-run as fresh read-only passes against HEAD, by agents that did not write
the code. Reports: `custody-symmetry-rerun.md`, `conservation-rerun.md`.

**Symmetry: 0 BLOCKING.** All three defences confirmed present on both columns,
and every meso read-then-act pair traced to a single statement or a locked
transaction.

**Conservation: 1 BLOCKING (F-8), now fixed.** Composition nets against
committed PLUS in flight, but a NEGATIVE delta is a payout — `award_mesos`
carries `-delta` — so lowering the box while a raise was in flight credited the
player against a debit that had not landed:

    stake 1000  -> delta +1000, debit submitted, unresolved
    retype 200  -> delta  -800, CREDIT of 800 submitted
    the debit FAILS -> player is 800 up, debited nothing

Nothing recalls it. Escrow lands at -800, which every consumer correctly reads
as "nothing owed", so Task 3's gate can refuse to DELIVER but cannot claw back
money already in the wallet. **This was a hole in Task 1's chosen shape, missed
while implementing it and found only by the independent pass.**

Fixed by refusing a PAYOUT while anything is in flight. Raises still compose
deliberately: if an earlier debit fails, later debits took only what they took,
so the committed total still equals what landed, and no payout was ever made
against the unconfirmed part. Only the narrow lower-mid-saga case serialises.
Test: `TestLoweringTheMesoBoxNeverPaysOutAgainstAnUnconfirmedStake`,
mutation-verified.

**Symmetry NON-BLOCKING, also fixed:** `DispatchTradeStagingRollbacks` had no
`AwardMesos` arm, so a MESO staging saga — whose only step is the debit —
reverse-walked nothing and the debit stood with no escrow behind it. The same
class as Task 4, and the inverse already existed verbatim on the late-completion
path. Test: `TestStagingReverseWalkReversesItsMesoDebit`, mutation-verified.

**Two observations recorded, not actioned:**

1. `trade_unwind` runs under `SagaType TradeTransaction`, not a type of its own,
   so the orchestrator ALREADY reverse-walks its failed steps before
   `SAGA_FAILED` reaches `UnwindFailed`. Both delivery orderings were traced and
   converge correctly. Task 5 above reads as though `UnwindFailed` were the sole
   recovery mechanism; it is the atlas-trades half of two, and a future reader
   should not assume it reverses currency by itself.
2. A row left persistently negative with no stakes outstanding would never be
   retired (`DeleteResolvedMeso` requires exactly 0). With F-8 fixed this should
   be unreachable, and leaving such a row visible is the right behaviour for a
   state that would indicate a bug — so the condition is deliberately NOT
   widened to `<= 0`, which would silently discard a real debt.

**One item the auditor flagged UNVERIFIED, carried forward honestly rather than
closed:** whether a Kafka producer-level failure when dispatching the
orchestrator's negated `AwardMesos` reversal (logged and skipped, not retried)
could combine with atlas-trades' `RestoreMesoRefunds` to mint. It was not
asserted as a defect and is not proven safe either.

**Full gate, all green:** `go build` / `go vet` / `go test -race` across all 13
changed modules; `docker buildx bake atlas-trades` (required — the branch moved
its `go.mod`); all nine applicable repo guards; `tools/lint.sh`.

Then the standard gate: `go build` / `go vet` / `go test -race -timeout 240s`
in `libs/atlas-saga`, `atlas-trades`, `atlas-saga-orchestrator`, `atlas-channel`,
`atlas-tenants`; all guards in the repo CLAUDE.md; `tools/lint.sh --check` with
node 22; and `docker buildx bake` for every service whose `go.mod` moved.

---

## Deliberately out of scope

- **The two decisions already made.** Automatic boot return stays (the escrow
  table is the custody source of truth, and release-before-grant is now pinned
  by tests in all three expanders). Meso rows are deleted only once fully
  resolved.
- **A read-only escrow REST surface.** Worth doing — nothing in the system can
  currently answer "what are we holding?" — but it is an addition, not a fix.
  Recorded here so it is not lost.
- **`AcceptToCharacter → RequestDestroyItem`** destroying a recipient's own
  instance of a template (conservation audit F-6). Real, but not introduced by
  task-205 and not specific to trade; it needs its own task.
