# Task-205 — ITEM vs MESO custody symmetry matrix

Read-only audit. Every claim below is grounded in `file:line` on this branch
(`task-205-player-trade`, HEAD `0bf941fc5`). Paths are repo-relative.

Shorthand used throughout:

| Short path | Real path |
|---|---|
| `trade/processor.go` | `services/atlas-trades/atlas.com/trades/trade/processor.go` |
| `trade/settlement.go` | `services/atlas-trades/atlas.com/trades/trade/settlement.go` |
| `trade/model.go` | `services/atlas-trades/atlas.com/trades/trade/model.go` |
| `escrow/*.go` | `services/atlas-trades/atlas.com/trades/escrow/*.go` |
| `saga/consumer.go` | `services/atlas-trades/atlas.com/trades/kafka/consumer/saga/consumer.go` |
| `orch/processor.go` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` |
| `orch/compensator.go` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` |

---

## 0. The one-sentence version

The item column has **three** structural defences the meso column does not have:
an in-flight amount that is netted before a new movement is submitted
(`StagedQuantityFrom`), a durable single-claimant latch on the return path
(`returning_at`), and a hard settlement-time check that the custody row still
exists (`settlementPayload`). Every finding below is one of those three absences
becoming reachable.

---

## 1. Findings, ranked

### F1 — BLOCKING. A second meso stage while one is in flight double-debits the player; the superseded stake's debit is never accounted for anywhere.

**What.** `addMeso` derives the movement from the **committed** escrow total only:

- `trade/processor.go:1305` — `escrowed, _, err := p.esc.MesoByOwner(room.Id(), characterId)`
- `escrow/provider.go:60-72` — `MesoByOwner` returns `e.Amount`, i.e. the committed column. `PendingAmount` / `PendingDelta` are not consulted.
- `trade/processor.go:1311` — `delta := int64(amount) - int64(escrowed)`
- `trade/processor.go:1333` — `ArmMesoStake(...)`, which **overwrites** any stake already armed (`escrow/administrator.go:228-236`, `DoUpdates` on `pending_stake_id`).

The first stake's `award_mesos` has already moved real meso in atlas-character.
Its terminal status then finds nothing: `MesoStakeById` matches on
`pending_stake_id` (`escrow/provider.go:86`), which now holds the *newer* stake,
so `resolveMesoStake` returns `(false, nil)` (`trade/processor.go:1397-1401`) and
`saga/consumer.go:93-104` falls through to `SettlementSucceeded`, which finds no
settlement record and drops it (`trade/settlement.go:952-957`).

**Nothing gates this.** `Participant.PendingMesoTxId()` / `PendingMesoAmount()`
(`trade/model.go:232,235`) exist and are read by **no production code** — verified
by grep over `trade/*.go` excluding tests. The in-flight guard is state that is
never consulted.

**Item twin does the right thing.** `stageableQuantity`
(`trade/processor.go:1239`) subtracts `pt.StagedQuantityFrom(inventoryType,
sourceSlot)`, and pending staged items are in `pt.Items()`
(`trade/processor.go:948,964-966`; `StagedItem` is born pending,
`trade/model.go:109`). So a second stage out of the same stack cannot
over-commit. **This is an asymmetry and it is a bug, not a justified difference.**

**Repro.**
1. A opens a trade with B. A's escrow meso row is absent (escrowed = 0).
2. A types `100` in the meso box. `delta = 100 - 0 = 100`; `award_mesos(-100)` is submitted; stake1 armed with `pending_amount=100, pending_delta=100`.
3. atlas-character debits 100 and publishes `stat.TypeMeso` with `ExclRequestSent: true` — this is what unlocks the client (design §5A.5). **The client is now free to send another PUT_MONEY, and stake1's SAGA_COMPLETED has not yet reached atlas-trades.**
4. A types `200`. `MesoByOwner` still reads `amount = 0`. `delta = 200 - 0 = 200`; `award_mesos(-200)` is submitted; stake2 overwrites stake1 on the row.
5. stake1's SAGA_COMPLETED arrives → no row answers to it → dropped silently.
6. stake2's SAGA_COMPLETED arrives → `amount = 200`.

A has been debited **300**. The escrow record says **200**. Cancel refunds 200
(A is down 100 permanently); settle delivers `tax(200)` to B (A is down 100
permanently). Meso destroyed, on the reference client, with no race against an
internal window — only against a Kafka round trip that the client is explicitly
unblocked before.

**Tested?** NO. `trade/processor_staging_test.go:1051`
(`TestAddMesoStagesTheDeltaAgainstEscrowNotTheRoom`) seeds the **committed**
escrow via `escrowOf(t, p).setMeso(...)` and never arms a pending stake — it
asserts the current behaviour, not conservation.
`escrow/administrator_test.go:472` (`TestArmMesoStakeSupersedesPriorStake`)
exercises the DB compare-and-set only; it never asks what happened to the
superseded stake's money. There is no test anywhere that calls `AddMeso` twice
without resolving the first (grep for `supersede` in `trade/` returns only
comments).

---

### F2 — BLOCKING. A meso stake that resolves while the room is SETTLING has its debit destroyed. Whether meso is conserved depends purely on the arrival order of two independent Kafka statuses.

**What.** Nothing prevents Confirm/Attest/settle while a meso stake is in flight
(`trade/settlement.go:203-271`, `288-319`, `426-503` — no reference to any
pending-stake state). `settle` builds the payload from the in-memory participant:

- `trade/settlement.go:544-547` — `MesoStaged/MesoTax/MesoDelivered` come from `pt`, which only advances when a stake **settles** (`trade/model.go:254-256`).

So an in-flight stake is silently excluded from the trade. Then:

- If the stake's status arrives **after** `SettlementSucceeded`: the room is gone, `resolveMesoStake` takes the room-gone branch (`trade/processor.go:1421-1436`) and `refundOrphanedStake` returns the delta. Correct. **This is the tested ordering** (`trade/processor_settlement_test.go:1680`, `TestSettlementSuccessKeepsARowWhoseStakeIsStillInFlight` — the test explicitly drives `MesoStageSucceeded` *after* `SettlementSucceeded`).
- If it arrives **before**: `p.reg.Get` finds the SETTLING room (`trade/processor.go:1421`), `WithSettledMeso` moves `mesoStaged` to the new total *after the payload has already been submitted*, and then `dischargeSettledMesos` (`trade/settlement.go:1330-1340` → `retireEscrowMesos:1350`) zeroes and deletes the row. The extra delta is debited, undelivered, unrefunded and unrecorded. **Destroyed.**

**Item twin cannot produce this.** `settlementPayload` refuses to settle when a
staged item has no escrow row (`trade/settlement.go:550-553`), so the in-flight
item case either settles correctly (row already written) or aborts the whole
trade. Meso has no equivalent check at any point in the settlement path.

**Repro.**
1. A stages 5 000 meso; stake commits; `amount = 5000`, room `MesoStaged = 5000`.
2. A types 9 000. Stake armed (`pending_amount=9000, pending_delta=4000`); `award_mesos(-4000)` submitted; atlas-character debits 4 000 and unlocks A's client.
3. A presses Trade; B presses Trade; both clients auto-reply TRANSACTION. `settle` runs with `MesoStaged = 5000` and submits `trade_settlement` delivering `tax(5000)` to B.
4. The **stake's** SAGA_COMPLETED lands first: `CommitMesoStake` sets `amount = 9000`; the SETTLING room is found; `mesoStaged` becomes 9 000 (cosmetic).
5. The **settlement's** SAGA_COMPLETED lands: `dischargeSettledMesos` zeroes + deletes the row.

A debited 9 000, B received `tax(5000)`. 4 000 meso destroyed with no error
logged anywhere.

**Tested?** NO for this ordering. The mirror ordering is tested and passes,
which is exactly what makes this an invariant held by emergent ordering rather
than by a guard.

---

### F3 — BLOCKING. `discardResolvedMeso`'s pre-CAS `Amount()` test misreads a concurrent teardown and leaves a stale non-zero row the next boot refunds a second time.

**What.** `resolveMesoStake` reads the row (`trade/processor.go:1393`), then
commits it (`:1409`). `CommitMesoStake` assigns `amount = pending_amount`
**unconditionally** (`escrow/administrator.go:257`, `gorm.Expr("pending_amount")`).
The room-gone branch then calls `discardResolvedMeso`, which decides whether to
retire the row from `row.Amount()` — the value read **before** the CAS
(`trade/processor.go:1530-1533`):

```go
if row.Amount() != 0 {
    return nil          // "a room whose process died before any teardown refunded it"
}
```

That premise fails when a teardown commits *between* the read and the CAS. The
teardown's `clearRefundedMesos` (`trade/settlement.go:1302` →
`retireEscrowMesos:1350` → `UpsertMeso(...,0)`) zeroes the row in its own
transaction, and the window is not tight: `emitUnwind` reads `MesosByRoom`
(`trade/settlement.go:626`), then issues one `ClaimForReturn` UPDATE per escrowed
item (`:633` → `claimItemsForReturn:583-597`), and only then zeroes (`:659`).

**Result.** The teardown refunds the committed part `C`; the orphan path refunds
the delta `D`; total `C + D = P` — arithmetically correct. But the row is left
carrying `P` with no room referencing it. Nothing else ever reads it. The next
boot's `ReconcileEscrow` (`trade/settlement.go:1256-1263`) sees a non-zero
room-less row and refunds `P` again. **Mint.**

**Repro.**
1. A has 100 meso committed in room R; A types 500 → stake armed (`pending_amount=500, pending_delta=400`), `award_mesos(-400)` submitted and executed.
2. The stake's SAGA_COMPLETED reaches `resolveMesoStake`; it reads the row (`amount = 100`).
3. Concurrently B disconnects → `TeardownCharacter` → `claimRoom` removes R from the registry → `emitUnwind` reads mesos (100), claims A's items, zeroes the row, submits `trade_unwind` refunding 100. Commits.
4. Step 2's `CommitMesoStake` now runs: `amount = 500`, pending cleared.
5. Room gone → `refundOrphanedStake` refunds `pending_delta = 400`.
6. `discardResolvedMeso` sees `row.Amount() == 100 != 0` → returns early. Row survives with `amount = 500`.
7. Restart atlas-trades. `ReconcileEscrow` refunds 500 to A.

Net: A debited 500, refunded 100 + 400 + 500 = 1 000. **500 minted.**

**Item twin has no equivalent hole**: `ClaimItemForReturn`
(`escrow/administrator.go:120-128`) decides by `RowsAffected` inside the UPDATE,
never by a read the caller made first — and its doc comment says so explicitly.
The meso path has no such latch on any of its refund routes.

**Tested?** NO. `trade/processor_staging_test.go:1382`
(`TestATeardownKeepsARefundedRowUntilItsStakeResolves`) and `:1452`
(`TestAnOrphanedStakeLeavesNothingForTheNextBootToRefund`) both drive the
teardown to completion **before** resolving the stake, i.e. the ordering where
`row.Amount()` is already 0. The interleaved ordering has no test.

---

### F4 — BLOCKING. A failed `trade_unwind` is handled by nobody. Items are latched into permanent limbo; meso is destroyed outright.

**What.** No consumer handles a terminal failure of a `trade_unwind` saga.
`saga/consumer.go:112-144` routes `SAGA_FAILED` through `StageFailed` (no escrow
row named by the unwind's transaction id → `false`), `MesoStageFailed`
(`MesoStakeById` on the unwind tx id → no match → `false`), then
`SettlementFailed` (no settlement record → dropped, `trade/settlement.go:952-957`).
The failure is logged nowhere in atlas-trades.

Both columns have already discharged their record **before** the unwind is known
to have succeeded, and they discharge it differently:

- **Items**: `claimItemsForReturn` stamps `returning_at` before the payload is built (`trade/settlement.go:583-597`). It is cleared **only** by `RestoreItem` (`escrow/administrator.go:144-151`), which is dispatched only for a `ReleaseFromTrade` step that reached `Completed` (`orch/compensator.go:1916,1957-1971`). An unwind that fails at expansion (`orch/processor.go:1651-1655` rejects `Amount > MaxInt32`) or whose first `release_from_trade` fails leaves the latch set forever. `ClaimItemForReturn` requires `returning_at IS NULL`, so every future boot sweep claims-and-loses on that row (`trade/settlement.go:1384`) and skips it. The item is **stranded permanently**, recoverable only by hand-editing the table.
- **Meso**: `clearRefundedMesos` zeroes the row **in the same transaction that buffers the unwind** (`trade/settlement.go:659-662`, `1074-1077`, `1412-1415`). If the unwind's `award_mesos` never completes, the reverse-walk has nothing to reverse (`orch/compensator.go:1920-1936` acts on `Completed` steps only) and the row now reads `amount = 0`, which the boot sweep skips (`trade/settlement.go:1257`). The escrowed meso is **destroyed with no record that it ever existed**.

**Repro (meso).**
1. A stages 10 000 meso in room R (committed).
2. B logs out → teardown → `emitUnwind` zeroes A's row to 0 and submits `trade_unwind` with one `award_mesos(+10000)`.
3. atlas-character is unavailable; the saga exhausts retries and emits SAGA_FAILED.
4. atlas-trades drops the event. A's row says 0. Boot sweep skips it. 10 000 meso gone.

**Repro (item).**
1. A stages an item; escrow row E exists.
2. A logs out → teardown claims E (`returning_at` stamped) and submits `trade_unwind`.
3. The `release_from_trade` step fails (custody consumer error → `errorStatusProvider`, `escrow/processor.go:87`). The saga fails; no step reached `Completed`, so `RestoreTradeEscrow` is never dispatched.
4. E survives with `returning_at` set and `deleted_at` NULL. Every subsequent boot sweep reads it, fails the claim, and skips it (`trade/settlement.go:1384`, `escrow/administrator.go:124`). The item is unreachable forever.

**Tested?** NO. There is no test of a failed `trade_unwind` in either service
(no test name in `trade/` or the orchestrator mentions an unwind failure).
`escrow/administrator_test.go:1013` (`TestRestoreItemReleasesTheReturnClaim`)
covers only the path where the release *did* complete.

---

### F5 — SERIOUS. The boot sweep and the orphaned-stake path can both refund the same meso row. Items are arbitrated by `returning_at`; meso is arbitrated by nothing.

**What.** `ReconcileEscrow` snapshots `AllMesos` (`trade/settlement.go:1219`) and
refunds `m.Amount()` (`:1396-1408`), while `resolveMesoStake`'s room-gone branch
independently refunds `pending_delta` (`trade/processor.go:1480-1505`). Neither
checks the other. Items on the identical pair of paths **are** arbitrated:
`unwindStranded` claims (`trade/settlement.go:1384`) and `returnOrphanedStage`
claims (`trade/processor.go:1124`).

The sweep is not isolated from live traffic: consumers are registered at
`services/atlas-trades/atlas.com/trades/main.go:92-123` and `ReconcileAtBoot` is
spawned afterwards at `:139-143`, on its own goroutine.

**Repro.**
1. A stakes 500 on top of 100 committed (`pending_delta = 400`); the debit lands.
2. atlas-trades restarts before the stake's SAGA_COMPLETED is consumed.
3. On boot, the redelivered SAGA_COMPLETED is consumed first: `CommitMesoStake` sets `amount = 500`; room gone → `refundOrphanedStake` refunds 400.
4. `ReconcileEscrow`'s `AllMesos` snapshot (taken after step 3) reads `amount = 500` and refunds 500.

A debited 500, refunded 900. **400 minted.** (`discardResolvedMeso` would have
retired the row in step 3 only if `row.Amount()` had been 0 — here it was 100,
so the row survived for the sweep to find; see F3, same guard.)

**Tested?** NO. `trade/settlement_reconcile_test.go:302,336`
(`TestReconcileEscrowSkipsARowAlreadyClaimedForReturn`,
`...SubmitsNothingWhenEveryRowIsClaimed`) pin exactly this arbitration **for
items**. There is no meso equivalent, because there is no meso claim to test.

---

### F6 — SERIOUS. The boot sweep can sweep a **live** room's escrow, and for meso that mints.

**What.** `ReconcileAtBoot`'s exclusion set is built solely from unresolved
settlement records (`trade/settlement.go:1184-1191`). A room created after boot
but before `AllItems`/`AllMesos` runs is in neither set. Consumers are live by
then (`main.go:92-123` precedes `:139`).

- Item case is **defended by accident but defended**: the sweep claims and returns the row; the room still lists the item; at settlement `settlementPayload` finds no escrow row and errors (`trade/settlement.go:550-553`), so the trade aborts with LEAVE 8 instead of settling half a trade. No duplication.
- Meso case is **not defended**: the sweep refunds and zeroes the row, but the settlement's meso leg is derived entirely from `pt.MesoStaged()` (`trade/settlement.go:545-547`) and the orchestrator's expansion is credit-only (`orch/processor.go:1606-1624`). The giver is refunded **and** the receiver is credited. **Mint.**

**Repro.**
1. atlas-trades restarts. Consumers come up.
2. A and B open a trade; A stakes 1 000 meso; the stake commits and writes the escrow row.
3. `ReconcileEscrow`'s `AllMesos` runs a moment later, sees a row whose room id is in no settlement record, refunds 1 000 to A and zeroes the row.
4. A and B confirm and settle. B is credited `tax(1000)`.

A is whole; B gained `tax(1000)` from nothing.

**Tested?** NO. All `ReconcileEscrow` tests construct the escrow state directly
with no live room (`trade/settlement_reconcile_test.go:259-702`).

---

### F7 — SERIOUS. An in-flight meso stake that is never resolved is invisible to every recovery path.

**What.** `ReconcileEscrow` skips zero-amount meso rows outright:
`trade/settlement.go:1256-1258` — `if m.Amount() == 0 { continue }`. A row whose
`amount` is 0 but which carries an armed stake records a debit that **has already
been taken from the player** (`PendingDelta`, `escrow/entity.go:184-192`). No
path sweeps it: the boot sweep skips it, no room references it, and its terminal
status is by hypothesis lost (orchestrator crash mid-saga, or a saga that never
terminates).

The item column has no pending-based exclusion at all — `AllItems`
(`escrow/provider.go:110-116`) returns every live row and every one is returned.

**Justified or bug?** Bug. The `amount == 0` filter exists to avoid emitting
zero-value refund legs; it was not written to consider `pending_delta`.

**Tested?** `trade/settlement_reconcile_test.go:642`
(`TestReconcileEscrowIgnoresAnAlreadyZeroedMesoRow`) pins the *skip* as intended
behaviour with no pending stake armed — it asserts current behaviour and would
keep passing while the pending-stake case bleeds meso.

---

### F8 — MINOR. `MesoStakeById` is untenanted while the CAS that follows it is tenanted; a tenant-header mismatch silently forfeits the debit.

`escrow/provider.go:83-99` deliberately reads across tenants. The very next call,
`CommitMesoStake` / `AbandonMesoStake`, is scoped to `p.t.Id()`
(`escrow/processor.go:137-143` → `administrator.go:255,274`). If the saga status
event's tenant header is absent or wrong, the row is found but the CAS matches
nothing → `matched == false` → logged as "already resolved"
(`trade/processor.go:1416-1419`) and the stake is abandoned in place. The item
path routes through the tenant-scoped `ItemById` (`trade/processor.go:189`), so
the same mismatch surfaces as "not a stage" and falls through the routing
instead of masquerading as a resolved stake. Asymmetric, low probability, but
the failure is silent on the meso side and merely mis-routed on the item side.

### F9 — MINOR. `stageFailed` has no orphan arm; `stageSucceeded` does.

`trade/processor.go:1060-1064` returns `(false, nil)` when no dialog slot claims
the escrow id, where `stageSucceeded` calls `returnOrphanedStage`
(`:1010`). Justified in the normal case (a failed staging saga's rows are
removed by `DispatchTradeStagingRollbacks`, `orch/compensator.go:2473-2487`), but
that dispatch logs-and-continues on failure (`:2482-2486`), so a dropped
`RemoveTradeEscrow` leaves a row nothing reports until the next boot. Acceptable
as-is; noted because it is the one item-side path with no orphan handling.

### F10 — MINOR. Dead in-flight accessors and a dangling doc comment.

`Participant.PendingMesoTxId()` / `PendingMesoAmount()` (`trade/model.go:232,235`)
are unreferenced outside tests — the state F1 and F2 needed is present and unused.
`trade/model.go:283-285` is a doc comment for a `WithRelocatedItems` that no
longer exists.

---

## 2. The matrix

Legend: **Impl** = what performs it. **Discharge** = what ends the custody
record, and whether it is guaranteed on every path including a restart.
**Test** = named test, or NO with the reason. **Asym** = asymmetry verdict.

### Row 1 — Stage submit

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/processor.go:892-988` `putItem` → `sgp.Stage` (`transfer_to_trade`), expanded to `release_from_character`+`accept_to_trade` at `orch/processor.go:1437-1517`; the row is written by the custody consumer (`kafka/consumer/custody/consumer.go:65-81`) → `escrow.Accept` (`escrow/processor.go:73-81`) | `trade/processor.go:1271-1367` `addMeso` → `ArmMesoStake` (`escrow/administrator.go:210-238`) then `sgp.StageMeso` (bare `award_mesos`) |
| **Discharge** | n/a (custody begins). Row + saga ack land in one `message.Buffer` (`escrow/processor.go:74-80`), so a row without an ack is impossible. | n/a. Arm + saga command commit in one `p.emit` transaction (`trade/processor.go:1333,1358`, `emit` at `:404-411`). Atomicity is symmetric and correct. |
| **Test** | `processor_staging_test.go:461` `TestPutItemSubmitsATransferToTradeKeyedByItsEscrowRow`; `:519,540` pending-slot holding | `processor_staging_test.go:1051` `TestAddMesoStagesTheDeltaAgainstEscrowNotTheRoom` — **does not cover an in-flight stake** |
| **Asym** | **BUG — F1.** Item nets in-flight quantity (`StagedQuantityFrom`, `:1239`); meso nets nothing. |

### Row 2 — Stage confirm (SAGA_COMPLETED)

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/processor.go:1002-1047` `stageSucceeded` → clear `pending`, emit `ITEM_STAGED` | `trade/processor.go:1391-1466` `resolveMesoStake(settled=true)` → `CommitMesoStake` then room, emit `MESO_STAGED` |
| **Discharge** | n/a | n/a |
| **Test** | `processor_staging_test.go:850` `TestStageSucceededClearsPendingAndAnnouncesOnce` | `processor_staging_test.go:1135` `TestMesoStageSucceededCommitsTheStakeAndAnnouncesIt` |
| **Asym** | Item verifies the escrow row exists before announcing (`:1025-1031`); meso has no equivalent because the row *is* the thing being committed. Justified. |

### Row 3 — Stage refuse (SAGA_FAILED, and every local refusal)

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/processor.go:1060-1093` `stageFailed`; local refusals via `refuseStage` (`:883-890`) → `ITEM_REFUSED` | `trade/processor.go:1391-1466` `resolveMesoStake(settled=false)` → `AbandonMesoStake`; local refusals via `refuseMeso` (`:1264-1269`) → `MESO_REFUSED` |
| **Discharge** | Dialog slot freed via `WithoutItem`; the escrow row (if any) is the orchestrator's to remove (`orch/compensator.go:2473-2487`) | Stake cleared without committing (`escrow/administrator.go:271-283`) |
| **Test** | `processor_staging_test.go:688` `TestEveryPutItemRefusalPathAnswersTheClient`; `:910`, `:954` | `processor_staging_test.go:1185` `TestMesoStageFailedReEchoesTheLastValidAmount`; `:1580` |
| **Asym** | Symmetric and both unlock the client (design §5A.6). No finding. |

### Row 4 — Stage superseded / redelivered

| | ITEM | MESO |
|---|---|---|
| **Impl** | No supersede concept — each stage mints its own escrow id (`:947`). Redelivery is inert via `Pending()` (`:1012`) and `OnConflict DoNothing` (`escrow/administrator.go:82`) | Supersede by overwrite (`escrow/administrator.go:228-236`); redelivery inert via the `pending_stake_id` CAS (`:252-283`) |
| **Discharge** | — | **The superseded stake's DEBIT is never discharged.** The CAS makes the *status* inert; the money already moved. |
| **Test** | `processor_staging_test.go:850` (redelivery) | `escrow/administrator_test.go:472` `TestArmMesoStakeSupersedesPriorStake` — DB-level only, asserts nothing about conservation. **NOT tested as a contract.** |
| **Asym** | **BUG — F1.** |

### Row 5 — Settlement success

| | ITEM | MESO |
|---|---|---|
| **Impl** | `orch/processor.go:1571-1583` `release_from_trade` per item, then `accept_to_character` (`:1586-1602`) | `orch/processor.go:1606-1624` credit-only `award_mesos` to the counterparty |
| **Discharge** | `release_from_trade` → `DeleteItem` soft-delete (`escrow/administrator.go:92-96`). Guaranteed by the saga; a restart is covered by `ReconcileSettlements` (`trade/settlement.go:1430-1457`). | `dischargeSettledMesos` (`trade/settlement.go:1330-1340`) — zero + conditional delete, run **before** `sp.Resolve` (`:981-983`) so the row cannot outlive the record. This is the fix for the original bug and it is correct **for the orderings it anticipates**. |
| **Test** | `processor_settlement_test.go:853,897,792` | `processor_settlement_test.go:1577,1596,1622,1680` — a genuinely good set |
| **Asym** | **BUG — F2.** Item settlement hard-fails when the custody row is missing (`trade/settlement.go:550-553`); meso settlement never reads the meso escrow at all, so a stake resolving mid-settlement is silently dropped. |

### Row 6 — Settlement failure

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/settlement.go:1036-1078` `unwindRecord`, driven from the durable record's room id | same function, `:1060-1070` |
| **Discharge** | Claimed (`:1048`) then released by the unwind saga | Zeroed by `clearRefundedMesos` (`:1074`) **before** the unwind is known to succeed |
| **Test** | `processor_settlement_test.go:967` `TestSettlementFailureUnwindsTheEscrowBackToBothOwners` | `processor_settlement_test.go:1730` `TestSettlementFailureStillRefundsTheEscrowedMeso` |
| **Asym** | **BUG — F4.** Both discharge before the return lands; the item outcome is "stranded", the meso outcome is "destroyed". |

### Row 7 — Teardown / cancel

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/settlement.go:1474-1485` `teardownRoom` → `emitUnwind` (`:621-663`); also `abandonSettlement` (`:385-402`) | same |
| **Discharge** | `claimItemsForReturn` latch (`:633`) then the unwind's `release_from_trade` | `clearRefundedMesos` zero + conditional delete (`:659`) |
| **Test** | `processor_staging_test.go:1742,1829,1899,1938,1977` | `processor_staging_test.go:1382` `TestATeardownKeepsARefundedRowUntilItsStakeResolves` |
| **Asym** | **BUG — F3.** The teardown's meso zero can be overwritten by a concurrently-committing stake, and `discardResolvedMeso` then misclassifies the row. The item latch is immune by construction. |

### Row 8 — Orphaned resolve (row outlives its room)

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/processor.go:1110-1146` `returnOrphanedStage` | `trade/processor.go:1421-1436` room-gone branch → `refundOrphanedStake` (`:1480-1505`) + `discardResolvedMeso` (`:1530-1539`) |
| **Discharge** | `ClaimForReturn` CAS (`:1124`) then the unwind | `discardResolvedMeso` — **conditional on a stale pre-CAS read** |
| **Test** | `processor_staging_test.go:1003,1899,1938` | `processor_staging_test.go:1253,1452,1512,1544` |
| **Asym** | **BUG — F3 / F5.** |

### Row 9 — Boot recovery sweep

| | ITEM | MESO |
|---|---|---|
| **Impl** | `trade/settlement.go:1214-1280` `ReconcileEscrow` → `unwindStranded` (`:1378-1416`); `AllItems` (`escrow/provider.go:110`) returns **every** row | same function; `AllMesos` (`:119`) but filtered by `if m.Amount() == 0 { continue }` (`trade/settlement.go:1257`) |
| **Discharge** | claim then unwind | zero then unwind |
| **Test** | `settlement_reconcile_test.go:259,302,336,363,397,445` | `settlement_reconcile_test.go:517,570,618,642` |
| **Asym** | **BUG — F5, F6, F7.** Item rows are claim-arbitrated against the live orphan path and are swept regardless of pending state; meso rows are neither. |

### Row 10 — Saga compensation / reverse-walk

| | ITEM | MESO |
|---|---|---|
| **Impl** | staging: `orch/compensator.go:2451-2515` (`AcceptToTrade`→`RemoveTradeEscrow` hard delete, `ReleaseFromCharacter`→re-grant from the snapshot). settlement/unwind: `:1912-1974` (`ReleaseFromTrade`→`RestoreTradeEscrow`, `AcceptToCharacter`→`DestroyItem`). Late inverses at `:2299-2314`. | `:1920-1936` (`AwardMesos`→`AwardMesos(-Amount)`) only. There is **no** custody-row inverse because there is no meso custody command. |
| **Discharge** | `RestoreItem` also clears `returning_at` (`escrow/administrator.go:149`) — a deliberate, correct coupling | The escrow meso row is not part of any saga, so no compensation ever touches it |
| **Test** | `TestExpandTransferToTradeOrdersReleaseBeforeAccept`, `TestExpandTradeSettlementOrdersReleasesBeforeAccepts`, `TestExpandTradeUnwindOrdersReleasesBeforeAccepts`, `TestExpandTradeSettlementEmitsNoNegativeAward` (orchestrator) | Only the `AwardMesos` negation is covered by the generic trade-rollback tests |
| **Asym** | Justified in shape (a balance has no row to restore), **but** it means the meso escrow row's lifecycle is maintained entirely by hand-written statements in atlas-trades with no saga to make them atomic with the money movement. That is the structural root of F2/F3/F4. |

### Row 11 — Concurrent-claim arbitration

| | ITEM | MESO |
|---|---|---|
| **Impl** | `ClaimItemForReturn` (`escrow/administrator.go:120-128`) — a real CAS decided by `RowsAffected`, durable in `returning_at` (`escrow/entity.go:124-150`), used by all three return paths (`trade/settlement.go:583`, `:1048`, `:1384`; `trade/processor.go:1124`) | **NOTHING.** `CommitMesoStake`/`AbandonMesoStake` CAS on `pending_stake_id` arbitrate *stake resolution*, not *refund*. No refund path claims anything. |
| **Discharge** | — | — |
| **Test** | `escrow/administrator_test.go:867,912,948,981,1013`; `settlement_reconcile_test.go:302,336` | none exists |
| **Asym** | **BUG — F3, F5.** This is the single largest missing guarantee on the meso side. |

### Row 12 — Restart survival

| | ITEM | MESO |
|---|---|---|
| **Impl** | Rows durable; `returning_at` durable so an in-flight return is not re-swept (`escrow/administrator.go:110-115`); `ReconcileAtBoot` orders settlements first (`trade/settlement.go:1183-1201`) | Rows durable; `pending_stake_id/amount/delta` durable so a status can resolve with no room (`escrow/entity.go:169-192`) |
| **Discharge** | Boot sweep returns everything not owned by an in-flight settlement | Boot sweep returns non-zero rows only |
| **Test** | `settlement_reconcile_test.go:397,445`; `escrow/administrator_test.go:912` `TestClaimItemForReturnSurvivesTheProcess` | `processor_settlement_test.go:1389` `TestTerminalStatusAfterARestartCompletesFromTheRecord` (settlement, not stake) |
| **Asym** | **BUG — F7.** A pending stake that survives a restart with `amount == 0` is invisible to the only recovery path there is. Also **F5/F6**: the sweep races live consumers because it is spawned after them (`main.go:92-143`). |

### Row 13 — Unwind failure (added; the design does not name this operation)

| | ITEM | MESO |
|---|---|---|
| **Impl** | **NONE.** `saga/consumer.go:112-144` has no arm for a `trade_unwind` transaction id | **NONE.** Same |
| **Discharge** | `returning_at` already latched → row permanently unclaimable | `amount` already zeroed → row invisible to the sweep |
| **Test** | none | none |
| **Asym** | **BUG — F4.** Both columns are broken; the meso column is worse (destroys rather than strands). |

---

## 3. Answers to the four targeted hunts

**Paths that END custody without discharging the record.**
Row 5/MESO in the stake-resolves-first ordering (F2): the debit is consumed by
`dischargeSettledMesos` with no delivery and no refund. Row 4/MESO (F1): the
superseded stake's debit leaves no record at all. Row 13 (F4): both columns
discharge before the return lands, so a failed unwind ends custody for the
record while the asset never arrives.

**Paths that READ the escrow tables and act without checking for a mid-flight actor.**
`addMeso`'s `MesoByOwner` (F1). `discardResolvedMeso`'s use of the pre-CAS
`row.Amount()` (F3). `ReconcileEscrow`'s `AllMesos` snapshot vs. a concurrently
committing stake and vs. live rooms (F5, F6). `emitUnwind`/`unwindRecord`/
`unwindStranded` read `MesosByRoom` and act with no claim (F5). Every one of
these is meso-side; the item side routes all four through `ClaimForReturn`.

**Implemented for one column and not the other.**
`returning_at` claim (item only). In-flight netting at submit (item only, via
`StagedQuantityFrom`). Custody-row existence check at settlement (item only,
`settlementPayload:550`). Pending-state exclusion from the boot sweep (meso only,
and it is a bug — F7). Saga-owned custody commands with compensating inverses
(item only — justified in shape, but it is why the meso row's invariants are
hand-maintained).

**Invariants that hold only by emergent ordering.**
F2 is the clearest: conservation of a mid-settlement meso stake depends on which
of two independent `EVENT_TOPIC_SAGA_STATUS` messages is consumed first, and the
test suite pins only the safe order. F3 depends on a teardown not interleaving
between one read and one write. F6 depends on the boot sweep winning a race
against a player who logs in and stages within the reconcile window.

**Restart between two steps leaving inconsistent state.**
F7 (pending stake + zero amount survives a restart and is swept by nothing) and
F5 (a stake status redelivered after a restart double-refunds against the sweep).

---

## 4. Not verified / would need more than reading

- **Transaction isolation.** F3's and F5's windows assume Postgres READ COMMITTED with no `SELECT … FOR UPDATE` on the meso row. I read the queries (`escrow/provider.go:60-99`) and they carry no locking clause, and `database.ExecuteTransaction` is used without an explicit isolation level in `trade/processor.go:405` — but I did not read `libs/atlas-database` to confirm the level it opens with. If it opened SERIALIZABLE the interleavings would abort rather than corrupt. **UNVERIFIED**; worth 5 minutes on `libs/atlas-database`.
- **Kafka partitioning between the stake status and the settlement status.** F2 assumes the two `EVENT_TOPIC_SAGA_STATUS` messages can be consumed in either order. They carry different transaction ids, so same-partition ordering is not guaranteed unless the producer keys by something common; I did not read the orchestrator's status producer keying. **UNVERIFIED** — but note that even same-partition ordering would only fix F2 if the *settlement's* status were guaranteed to precede the stake's, which is the reverse of causality.
- **Saga retry/timeout semantics for `trade_unwind`.** F4 assumes an `award_mesos` that ultimately fails produces a terminal SAGA_FAILED that atlas-trades then ignores. I verified nothing consumes it (`saga/consumer.go:112-144`); I did not verify how many retries precede it.
- **Whether the reference client can actually issue a second `PUT_MONEY` before atlas-trades sees the first stake's terminal status.** F1's step 3 relies on design §5A.5's own claim that `atlas-character`'s `statChangedProvider` hard-sets `ExclRequestSent: true` on the meso path, which unlocks the client independently of the saga completing. That is the design's stated mechanism, not something I re-derived from the binary. If the unlock were instead gated on the saga's terminal status, F1 would narrow to a modified-client exposure — it would not disappear, because the saga status and the stat change are separate messages. **PARTIALLY VERIFIED** (from design.md §5A.5, not from IDA).
- I did not audit atlas-channel's trade status consumer, the REST surface, or the packet layer — out of scope for a custody-symmetry pass.
