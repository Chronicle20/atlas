# task-205 — Conservation Re-Audit (parity-plan Task 7)

Read-only re-run of `conservation-audit.md` against the branch tip after
parity-plan Tasks 1–6 (`efec8332a..01106e56e`). No code, test, config or
template was permanently modified. One throwaway diagnostic test
(`escrow/zzz_conservation_probe_test.go`) was added, run, and deleted to
empirically confirm one finding against production code; it left no trace
(`git status` clean).

**Question asked of every path:** does
`sum(what left players' pockets) == sum(what escrow holds) + sum(what players received)`,
for items and for meso, everywhere?

All paths are repo-relative. Line numbers are as of the branch tip at the time
of this audit (`01106e56e test(task-205): stop the timeout-routing tests
stomping shared cache state`).

---

## 1. Paths walked and their conservation argument

### 1.1 The six symptoms from the original audit, re-verified against the fix

| Original finding | Fix (parity-plan task) | Re-verified | Evidence |
|---|---|---|---|
| **F-1** — `TradeStaging` saga timeout dispatches no rollback | Task 4 | ✅ fixed | `saga/timer.go:163-167` — `reverseWalkSagaTypes` now lists `CharacterCreation, MtsOperation, TradeTransaction, TradeStaging`; `dispatchTimeoutRollbacks` (`timer.go:177-195`) is a `switch` over that list, and `case TradeStaging` calls `c.DispatchTradeStagingRollbacks(s)`. `saga/compensator.go:305-307` (step-failure path) was already correct. `TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout` iterates the same list, so a future addition that forgets `timer.go` fails it. |
| **F-2** — settlement ignores an in-flight meso stake (mints on reduction, destroys on raise) | Task 3 | ✅ fixed for the *settlement* path | `trade/settlement.go:598-614` `assertMesoCustodyAgrees`, called per side from `settlementPayload` (`:564`), refuses to build a payload if `InFlightMesoDelta != 0` or `committed != pt.MesoStaged()`. The caller (`settle`, `:495-498`) turns the error into a failed settlement, which unwinds. **However, see new finding F-8 below: the underlying mechanism this gate was built to police (a stake resolving out of order) can mint currency BEFORE settlement is ever attempted, entirely bypassing this gate.** |
| **F-3** — `RestoreItem` resurrects a row a different saga already released | Task 6 | ✅ fixed | `escrow/administrator.go:269-276` — `RestoreItem` now takes the restoring saga's `txId` and restores only `WHERE returning_tx_id IS NULL OR returning_tx_id = ?`. `returnOrphanedStage` (`trade/processor.go:1117-1158`) mints one id and uses it for both the claim and the submitted unwind (the plan's "latent Task-5 defect" is fixed — one `unwindTxId` at line 1136, used at 1137 and 1151). `TestRestoreItemCannotResurrectAReturnedRow` / `TestRestoreItemStillUndoesAnUnclaimedRelease` cover both halves. |
| **F-5** — a failed `trade_unwind` is routed nowhere | Task 5 | ✅ fixed | `kafka/consumer/saga/consumer.go:151-162` — the unwind probe (`p.UnwindFailed`) is inserted BEFORE the settlement probe, with a comment explaining why the order is load-bearing. `trade/settlement.go:1686-1703` `UnwindFailed` calls `ReleaseItemReturnClaims` + `RestoreMesoRefunds`. `TestAFailedUnwindLeavesBothColumnsRecoverable` exercises redelivery of the same FAILED event twice and asserts the second delivery does not re-restore. |
| **F-4** — compensation-vs-unwind race across two topics strands items until restart | *not in scope* | still present | `trade/settlement.go:1115-1125` `unwindRecord` still reads `ItemsByRoom`/`MesosByRoom` fresh and returns `nil` if both are empty — no retry, no periodic sweep. `main.go:125-130` still documents "no reservation-refresh loop"; `main.go:140` runs `ReconcileAtBoot` exactly once. Not a ledger-identity violation (the original audit classified it SERIOUS, not BLOCKING: the asset is conserved, only unreachable until the next restart) and the parity plan never scoped it. Unchanged; re-confirmed present. |
| **F-6** — template-addressed compensating destroy can hit the recipient's own instance | *explicitly out of scope* | still present | `saga/compensator.go:1937-1956` unchanged (id-less accept contract). Explicitly named in parity-plan's "Deliberately out of scope" section. |
| **F-7** — `emitUnwind` refunds meso onto the room's stale channel | *not in scope* | still present, impact still unverified | `trade/settlement.go:727-728` still builds `TradeUnwindMeso` from `room.Field().WorldId()/ChannelId()`, unlike `refundOrphanedStake` (`processor.go:1517`) and `unwindStranded` (`settlement.go:1504`), which both read the owner's CURRENT field and explain why. Whether atlas-character's award-mesos credit is routed by character id (making this cosmetic) or by the payload's world/channel (making it meso-destroying) was UNVERIFIED in the original audit and remains UNVERIFIED here — I did not read atlas-character's `RequestChangeMeso` far enough to settle it (I read the balance-check and persistence half at `services/atlas-character/atlas.com/character/character/processor.go:824-859`; the credit itself is a direct DB `SetMeso` keyed by `characterId`, not by world/channel, so the *balance* update is unaffected by a stale channel — only the on-screen announcement (`mesoChangedStatusEventProvider`) would render wrong. On this reading F-7 is cosmetic, not conservation-breaking, but I have not traced the client-side rendering to be certain it is not silently dropped for a channel the character has left). |

### 1.2 Lifecycles re-walked from the original audit's L-1..L-21 table

Spot-re-verified the ones the fixes touch most directly (netting, claims, discharge, delete-conditions); did not re-derive the ones the fixes left untouched (tax splitting, partial stacks, restart ordering, double-delivery on items) since no code on those paths changed. Confirmed still correct by inspection:

- **L-2 / L-20 (tax only applied at settle, never on a refund)** — unchanged; `configuration.Tax` is called only from `settle` (`settlement.go:445`); every refund path (`emitUnwind`, `unwindRecord`, `unwindStranded`, `refundOrphanedStake`) still passes a raw amount.
- **L-9 (meso terminal status delivered twice)** — now `CommitMesoStake`/`AbandonMesoStake` are per-stake-row compare-and-sets (`escrow/administrator.go:403-447`), a strengthening of the original single-column CAS. `TestCommitMesoStakeMismatchedIdIsNoOp`, `TestCommitMesoStakeTwiceOnlyAppliesOnce` cover it.
- **L-12 (meso stake in flight vs teardown)** — `refundOrphanedStake` (`processor.go:1506-1531`) still reads the stake's own persisted `Delta()`, not a re-derived figure; `discardOrphanedMeso` (`processor.go:1562-1570`) now discharges *relatively* rather than assigning — a strict improvement (was flagged UNVERIFIED-adjacent in the original audit's §4.1 misclassification note, now fixed per Task 1/2).
- **L-16 (same escrow row, two return paths)** — `ClaimItemForReturn` (`escrow/administrator.go:122-130`) unchanged and still a single-statement CAS; its meso twin `ClaimMesoForReturn` (`escrow/administrator.go:479-507`) is new and closes the previously-flagged gap (original audit §4.5's "the one gap I would close first"). Verified as a `SELECT ... FOR UPDATE` + update inside one transaction, not a `RETURNING`-based read-and-take (the doc comment explicitly explains why `RETURNING` would be wrong). `TestClaimMesoForReturnIsExclusive` pins two *sequential* claims (the plan is honest that sqlite with `MaxOpenConns(1)` cannot exercise the actual interleaving — see §3 below).

### 1.3 New pressure tests run against this branch specifically (per the audit brief)

1. **Concurrent meso stakes composing, adversarial order, one failing.** Walked in depth — see **F-8** below. This is the one that broke.
2. **Signed int64 passing through negative — consumer misuse.** Checked every `uint32(...)` cast and every `<= 0` / `== 0` comparison against `MesoEntity.Amount` or a `ClaimMesoForReturn`/`InFlightMesoDelta` result:
   - `escrow/administrator.go:493` (`ClaimMesoForReturn`) and `trade/settlement.go:1357` (`ReconcileEscrow`) both gate on `<= 0` before ever casting to `uint32` — correct, no truncation-driven mint.
   - `escrow/administrator.go:587` (`DeleteResolvedMeso`) gates on `amount = 0` **exactly**, not `<= 0`. A row left at a negative `Amount` (which F-8 shows is reachable) can never satisfy this condition and is therefore **never retired** — see **F-9** below (non-blocking, but real).
   - `trade/settlement.go:610` compares `committed != int64(pt.MesoStaged())` — `uint32→int64` is a widening, lossless cast; a negative `committed` simply fails the equality (refuses to settle), no mint or destroy.
   - No consumer was found that reads a negative `Amount`, casts it to `uint32`, and pays it out. The truncation attack surface itself is clean; the defect is upstream of it (see F-8).
3. **`trade_unwind` redelivery — FAILED after success, FAILED twice, success and failure both delivered.** Traced `UnwindFailed`/`UnwindSucceeded` end to end (`trade/settlement.go:1686-1721`) and confirmed by reading (not re-deriving) `TestAFailedUnwindLeavesBothColumnsRecoverable` and `TestASucceededUnwindDiscardsItsRefundRecords` (`trade/processor_settlement_test.go:1826-1912`), both of which exercise exactly this sequence end to end against the real escrow store (not the `fakeEscrow` seam — `UnwindFailed`/`UnwindSucceeded` are called directly on the processor and read back through `escrow.MesoByOwner(p.db, ...)`, the real DB accessor). Confirmed correct: the claim-then-record design (`MesoRefundEntity`) makes a redelivered FAILED inert once resolved either way, in either order.
4. **Teardown racing the boot sweep; orphaned stage racing a teardown; a claim whose unwind never resolves.** `ClaimItemForReturn` / `ClaimMesoForReturn` are both single-statement, single-transaction CASes — the item one was already race-proof pre-existing; the meso one is new and, per the plan's own honest limitation (§3 below), is proven exclusive only via two *sequential* calls in the sqlite harness, not genuine concurrent interleaving. A claim whose unwind never resolves (crashes before its terminal status) leaves the row/records latched — recoverable only by an operator or a process restart that happens to redeliver the saga's status; there is no ticker (same root cause as F-4). Not a new finding — same shape as F-4, now extended to meso via `MesoRefundEntity`.
5. **`RestoreItem` staleness / post-grant redelivery.** Re-verified: `RestoreItem(db, tenantId)(id, txId)` (`escrow/administrator.go:269-276`) is fenced on `returning_tx_id`, and `returnOrphanedStage`'s latent id-mismatch bug (claim stamped with one id, saga submitted with another) is fixed — confirmed by reading `processor.go:1136-1151`, one `unwindTxId` used for both. `TestRestoreItemCannotResurrectAReturnedRow` / `TestRestoreItemReleasesTheReturnClaim` (corrected to share the saga id, per the plan's own note) both pass.

---

## 2. Findings

### F-8 — **BLOCKING** — a meso reduction, netted against an unconfirmed sibling increase, mints real currency if that increase later fails or resolves too late

This is new on this branch. It is the direct consequence of Task 1's fix
(composing independent stakes) not being paired with any check that a
*negative* delta's real-money credit is backed by the sibling debit it was
netted against having actually landed.

**The mechanism.** `addMeso` (`trade/processor.go:1283-1390`) computes a new
stake's delta as `target − EffectiveMesoByOwner`
(`processor.go:1328-1334`), where `EffectiveMesoByOwner` = committed +
Σ(deltas of every stake still IN FLIGHT) (`escrow/processor.go:201-203`,
`escrow/provider.go:114-128`). It then submits `AwardMesosPayload{Amount:
int32(-delta)}` (`processor.go:1381-1388`) — negative delta ⇒ positive
`Amount` ⇒ a real **credit** to the player's wallet, via
`RequestChangeMeso` (`services/atlas-character/atlas.com/character/character/processor.go:824-859`),
which applies unconditionally once its own (rare) overflow check passes —
crucially, **with no dependency on whether the sibling stake it was netted
against ever lands.**

`CommitMesoStake` / `AbandonMesoStake` then resolve each stake **entirely
independently** (`escrow/administrator.go:403-447`) — by design, per Task 1,
so that a genuinely-landed debit is never dropped as "superseded." But the
same independence means: if the EARLIER, larger stake (the one the later
reduction's delta was computed against) subsequently **fails** — a perfectly
ordinary outcome, since `RequestChangeMeso` re-checks the player's *live*
balance at execution time and rejects with `ErrNotEnoughMeso` if it has
dropped since the stage's arm-time check
(`character/processor.go:832-838`, confirmed routed to a saga step failure
via `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go:166-184`)
— then the reduction's credit was never actually backed by anything. The
credit already reached the wallet; the debit it assumed would land never
does.

**Reproducing sequence** (empirically confirmed against the real
`escrow` package — see below):

1. Character 100 holds ≥1000 real meso. Player types **1000** into the trade
   meso box. `addMeso`: `EffectiveMesoByOwner = 0`, `delta = +1000`. Stake
   **A** is armed (`ArmMesoStake(..., amount=1000, delta=1000)`,
   `processor.go:1356`); `award_mesos` submitted with `Amount = -1000` (a
   debit).
2. Before A's terminal status arrives, the player retypes the box to
   **200**. `addMeso`: `EffectiveMesoByOwner = committed(0) + inflight(A's
   +1000) = 1000`, `delta = 200 − 1000 = −800`. Stake **B** is armed
   (`amount=200, delta=-800`); `award_mesos` submitted with `Amount = +800`
   (a **credit**).
3. Meanwhile — no attacker action needed, purely ordinary async timing —
   character 100's real balance drops below 1000 between step 1's affordability
   check and A's actual execution (an NPC shop purchase, a quest-item
   turn-in, a cash-shop spend; anything using the same live balance while the
   trade dialog is open). A's `RequestChangeMeso(-1000)` is rejected
   (`character/processor.go:832-838`), emits a meso-error status, and the
   orchestrator marks the step FAILED
   (`saga-orchestrator/kafka/consumer/character/consumer.go:181-183`) →
   `SAGA_FAILED` → `MesoStageFailed(A)` → `resolveMesoStake` →
   `AbandonMesoStake(A)`. **A never touched the wallet at all** — there was
   nothing to compensate.
4. B's `RequestChangeMeso(+800)` succeeds independently (a credit has no
   affordability check to fail against) → `SAGA_FAILED` never fires for B →
   `SAGA COMPLETED` → `MesoStageSucceeded(B)` → `CommitMesoStake(B)` adds
   B's delta (−800) to the committed escrow total.

**Arithmetic:** character 100's real wallet gained **+800** meso (B's credit,
unconditional). A's assumed 1000 debit never happened. Escrow's own
bookkeeping ends at `Amount = −800` — which every consumer in the codebase
(`ClaimMesoForReturn`, `ReconcileEscrow`, `assertMesoCustodyAgrees`) correctly
treats as "nothing owed" and refuses to touch further. So nothing ever claws
the 800 back: it is not "in escrow" (there is no positive custody standing
behind it), it already landed in the player's live balance, and every path
that could reach the row again only ever *adds* to or *refuses against* a
negative figure — none of them *debit* the player to correct it.
`assertMesoCustodyAgrees` at settlement time (`settlement.go:598-614`) will
correctly refuse to settle this room (`committed(−800) != mesoStaged`), which
prevents the trade from *delivering* the wrong number — but the settlement
gate runs downstream of where the currency was already minted, and the
subsequent unwind only returns what's *positively* held in escrow (nothing).
**800 meso minted, permanently, with no error, log line, or recoverable trail
beyond the escrow row's own negative figure.**

This is not a corner case requiring a modified client or an operator error:
it needs two ordinary retypes of the trade box (the client permits this —
P2's finding that `PutMoney` only locks until its own `STAT_CHANGED` clears
it, not until the trade-level stage completes, is exactly what makes a second
`ADD_MESO` reachable before the first resolves) and one unrelated spend that
drops the player below their first-typed amount before that first saga
executes — both are things ordinary play produces, not adversarial ones.

**Empirical confirmation.** I temporarily added
`escrow/zzz_conservation_probe_test.go`, calling `ArmMesoStake`,
`AbandonMesoStake`, and `CommitMesoStake` — the real, unmodified production
functions — through exactly steps 1–4 above (using the trade package's own
delta arithmetic, computed by hand to isolate the escrow half), ran it, and
deleted it (the working tree is clean; `git status` confirms no residue):

```
escrow.Amount after A abandoned / B committed = -800
--- PASS: TestZZZProbeReductionAgainstUnconfirmedIncreaseGoesNegativeWithNoOffsettingDebit (0.00s)
```

This confirms the escrow-side half of the arithmetic against real code, not
just against reasoning about it. The wallet-side half (that B's credit
unconditionally reaches the player, and A's debit rejection touches nothing)
is confirmed by reading `character/processor.go:824-859` directly, not by
execution — atlas-character is a different service/module and was not
brought under test here.

**What would close it.** The gap is structural: a stake's delta is computed
optimistically against sibling stakes that have not yet confirmed. Two shapes
of fix are visible from the code as read (neither implemented, and I did not
design one — this audit is read-only): (a) refuse to arm a *reduction* whose
delta would go negative against **committed** custody alone, forcing a
reduction to net only against what is already durably committed, never
against another stake's optimistic in-flight delta; or (b) require the
credit leg of a reduction to also carry the offsetting debit as a single
atomic unit rather than two independently-resolving sagas. Both change the
semantics Task 1 documented deliberately ("compose, and resolve every stake
independently"), so this is not a small patch to the existing design — it
undoes the assumption that in-flight deltas are as good as committed ones for
netting purposes.

---

### F-9 — **NON-BLOCKING** — a meso row driven negative by F-8 is never retired

`DeleteResolvedMeso` (`escrow/administrator.go:580-593`) only deletes a row
when `amount = 0` (exact equality) **and** no stake is outstanding. A row
left at a negative `Amount` (reachable via F-8, or via any other path that
legitimately produces a transient negative per `MesoEntity`'s own doc
comment) can never satisfy `amount = 0` and is therefore never retired by
this function. It is not a conservation violation on its own —
`ReconcileEscrow` (`settlement.go:1357`) and `ClaimMesoForReturn`
(`escrow/administrator.go:493`) both correctly skip/no-op on a non-positive
row, so nothing is minted or destroyed by the row's mere persistence — but it
is a permanent orphan: it survives every future boot sweep
(`escrow.AllMesos` is unfiltered per the row's own doc comment at
`entity.go` around the `MesoEntity` block) and grows that scan's cost
forever, for a row that will never again do anything useful. This is a
housekeeping consequence of F-8, not a new independent defect; fixing F-8
by construction (a delta that can never be netted below what is durably
committed) would prevent the negative state that triggers it. If F-8 is
fixed by some other means that still permits a legitimate transient
negative, this needs its own fix (e.g. `DeleteResolvedMeso` deleting on
`amount <= 0`, mirroring every other consumer's condition, rather than exact
zero — though changing that changes what "resolved" means and I did not
verify that no caller depends on the exact-zero contract for column
hygiene).

---

### Findings carried forward, unchanged, from the original audit

- **F-4 (SERIOUS, non-blocking for conservation)** — still present, not in
  scope for this pass. See §1.1.
- **F-6 (SERIOUS)** — still present, explicitly out of scope per the parity
  plan. See §1.1.
- **F-7 (MINOR, impact still not fully settled)** — still present. See §1.1
  for the additional read I did on `RequestChangeMeso` narrowing (not
  closing) the original audit's UNVERIFIED item.

---

## 3. What I could not verify

1. **F-7's true impact** — whether atlas-character's stat-update client
   packet (as opposed to the balance write, which I confirmed is keyed by
   character id and unaffected) silently drops or misrenders when addressed
   to a channel/world the character has since left. I read
   `mesoChangedStatusEventProvider`'s existence and that the balance mutation
   itself does not depend on it, but not the socket-side renderer.
2. **True concurrent interleaving of `ClaimMesoForReturn` / `ClaimItemForReturn`
   under genuine Postgres READ COMMITTED.** Same limitation the parity plan
   states honestly for Task 2: `databasetest.NewInMemoryTenantDB` pins sqlite
   to `MaxOpenConns(1)`, so no test in this repo — mine included — can
   exercise the actual row-lock contention the design comments argue for.
   The claims are provably exclusive against two *sequential* callers; the
   *concurrent* case rests on the `SELECT ... FOR UPDATE` semantics being
   correctly implemented by Postgres, which I did not independently test
   against a live cluster.
3. **Whether F-8 is reachable through a THIRD path that needs no external
   balance change at all** — e.g., a saga-orchestrator-level timeout on the
   earlier stake (30s `DefaultSagaTimeout`) racing a fast retype+resolve of
   the later one, with no shop purchase or other external actor. I traced
   that `DispatchTradeStagingRollbacks` on timeout ultimately also emits
   `SAGA_FAILED` for the stake (same `AbandonMesoStake` destination), so I
   believe this path exists too and needs no external balance change — only
   ordinary Kafka/consumer latency — but I did not walk `EmitSagaFailed`'s
   full timeout-side code path with the same rigor as the direct-rejection
   path in §F-8, so I am stating this as a probable second trigger rather
   than a confirmed one.
4. **Whether atlas-character's own idempotency (if any) on `RequestChangeMeso`
   changes F-8's arithmetic under a REDELIVERED command.** I read the happy
   path and the rejection path but not whether a redelivered
   `CommandRequestChangeMeso` for the same `TransactionId` could double-apply
   B's credit (an independent, possibly compounding mint on top of F-8). This
   is outside atlas-trades and I did not chase it further given the scope of
   this audit.

---

## 4. Summary

**Counts:** 1 BLOCKING (new), 1 NON-BLOCKING (new, a consequence of the
BLOCKING one), 3 carried forward unchanged from the original audit (2
SERIOUS/non-blocking, explicitly or implicitly out of scope for this pass; 1
MINOR).

**The BLOCKING finding, one line:** composing independently-resolving meso
stakes (Task 1's fix) lets a reduction's credit reach the player's real
wallet before the increase it was netted against is confirmed, so when that
increase fails, the credit stands with no offsetting debit — a real,
reproducible mint of meso, confirmed empirically against the production
`escrow` package.

The six symptoms the original audit found (F-1, F-2, F-3, F-5, and the two
explicitly-scoped-out F-4/F-6) are correctly addressed or correctly left
as documented, non-blocking exceptions, by Tasks 1–6 as claimed. This pass
did not find them regressed. It found one new BLOCKING defect introduced by
the very mechanism (independent per-stake composition) that Task 1 built to
fix the original F-1b/F-2. No path was found that balances only because two
bugs cancel; F-8 does not cancel with anything — it is additive currency
creation with no corresponding loss anywhere in the system.
