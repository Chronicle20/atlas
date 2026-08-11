# task-205 — Conservation Audit (ledger lens)

Read-only audit. No code, test, config or template was modified.

**Question asked of every path:** for one staged item and one staked meso amount,
does the total in existence at the end equal the total at the start?

**Ledger identities held to:**

- **Item** — exists in exactly one place: the owner's compartment, `trade_escrow_items`,
  or the recipient's compartment. Never zero places, never two.
- **Meso** — `debited from giver` == `credited to receiver` + `tax destroyed` + `refunded to giver`.

All paths are repo-relative. Line numbers are as of the branch tip
(`0bf941fc5 fix(task-205): discharge the escrow meso a settled trade delivered`).

---

## 1. Findings, ranked

### F-1 — BLOCKING — a `TradeStaging` saga that TIMES OUT dispatches no rollback: the staged item (or the staked meso) is destroyed

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go:89-169`
is the 30 s timeout backstop every saga gets
(`saga/processor.go:325` schedules it; `saga/model.go:340-342` — `DefaultSagaTimeout = 30 * time.Second`;
`services/atlas-trades/atlas.com/trades/saga/producer.go:43-50` sets no override).

`handleSagaTimeout` dispatches a reverse-walk for exactly three saga types:

- `timer.go:127` — `CharacterCreation`
- `timer.go:136` — `MtsOperation`
- `timer.go:146` — `TradeTransaction`

`TradeStaging` is **not** in that list. Both of atlas-trades' staging sagas are
`TradeStaging`:

- item stage — `services/atlas-trades/atlas.com/trades/saga/producer.go:61-68` (`BuildStage`)
- meso stake — `services/atlas-trades/atlas.com/trades/saga/producer.go:81-88` (`BuildStageMeso`)

The step-driven compensator *does* handle it
(`saga/compensator.go:305-307` → `compensateTradeStaging` at `compensator.go:2414`,
→ `DispatchTradeStagingRollbacks` at `compensator.go:2451`), but that path is only
reached from `CompensateFailedStep`, i.e. when a step reports **failure**. A step that
simply never answers within 30 s goes through `handleSagaTimeout` instead, which
falls straight through to `c.Remove(ctx, txId)` (`timer.go:160`) and `EmitSagaFailed`
(`timer.go:166`).

#### F-1a — item destroyed

Sequence:

1. Player A sends `PUT_ITEM` for 1× template T.
   `trade/processor.go:977-987` submits the `transfer_to_trade` composite.
2. Orchestrator expands it into `release_from_character` then `accept_to_trade`
   (`saga/processor.go:1478-1515`).
3. `release_from_character` **completes** — atlas-inventory deletes/decrements the asset
   (`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1812-1834`).
4. `accept_to_trade` does not answer within 30 s (atlas-trades custody consumer lagging,
   partition stall, pod restart).
5. `handleSagaTimeout` fires. `s.SagaType() == TradeStaging`, so **no** reverse-walk is
   dispatched. Cache entry removed, `SAGA_FAILED` emitted.
6. atlas-trades `handleSagaFailed` (`kafka/consumer/saga/consumer.go:112-144`) →
   `StageFailed` → `trade/processor.go:1060-1093`: item is still `Pending()`, so the
   dialog slot is freed and `ITEM_REFUSED` is sent.

Arithmetic: compartment −1 T, escrow +0, recipient +0. **One item destroyed.**

If `accept_to_trade` lands *late*, the saga is already `Remove`d from the cache, so
`AcceptEvent` (`saga/processor.go:431-438`) reports `SkipReasonSagaNotFound` and drops
it — the escrow row is written but nothing references it. Item is then **stranded**
(recovered only by the next boot sweep, `trade/settlement.go:1214`). Either branch fails
the identity.

#### F-1b — meso destroyed

Same root cause on `BuildStageMeso`:

1. A raises the stake 0 → 500. `trade/processor.go:1332-1366` arms the stake
   (`ArmMesoStake`, delta +500) and submits `award_mesos` with `Amount: -500`.
2. atlas-character debits 500. The step's status is slow; the saga times out at 30 s.
3. No rollback dispatched (`TradeStaging`). `SAGA_FAILED` emitted.
4. atlas-trades `MesoStageFailed` → `resolveMesoStake` (`trade/processor.go:1391-1419`)
   → `AbandonMesoStake` (`escrow/administrator.go:271-283`) clears the pending stake and
   writes **nothing** into `Amount`.

Arithmetic: A −500; escrow records 0; nothing refunded; nobody credited.
**500 meso destroyed.**

The late `award_mesos` COMPLETED event would normally be caught by `CompensateLateStep`
(`AwardMesos` is in `lateCompensableActions`, `compensator.go:2078`), but the saga was
already evicted at `timer.go:160`, so `claimLateCompensation` (`compensator.go:2187-2192`)
cannot find it — `AcceptEvent` drops the event before that anyway.

---

### F-2 — BLOCKING — settlement ignores an in-flight meso stake: mints meso in one direction, destroys it in the other

`trade/settlement.go:441-447` computes the tax split from `pt.MesoStaged()`, and
`settlementPayload` (`trade/settlement.go:542-548`) puts `pt.MesoStaged()` /
`MesoTax()` / `MesoDelivered()` on the payload.

`mesoStaged` is only advanced when a stake's terminal status resolves
(`trade/model.go:249-260`, `WithSettledMeso`). The in-flight stake lives in
`pendingMesoTxId` / `pendingMesoAmount` (`trade/model.go:152-153`) — and
**nothing reads those fields.** `grep` over `services/atlas-trades` shows
`PendingMesoTxId()` / `PendingMesoAmount()` have no non-test callers:
neither `confirm` (`trade/settlement.go:203-271`) nor `settle`
(`trade/settlement.go:426-503`) consults them. The room's escrow row is likewise
only read at stage time (`trade/processor.go:1305`), never at settlement.

#### F-2a — reduction in flight → meso MINTED

1. A stages 500. `award_mesos −500` completes. `CommitMesoStake` → escrow `Amount = 500`,
   room `mesoStaged = 500`. (A is −500.)
2. A retypes the box to 100. `trade/processor.go:1311` computes `delta = 100 − 500 = −400`;
   `ArmMesoStake(amount=100, delta=−400)`; `award_mesos` with `Amount: int32(-delta) = +400`
   (`trade/processor.go:1364`). atlas-character **credits A 400**. (A is now −100.)
3. Before that stake's terminal status is applied, both sides confirm and attest.
   `settle` reads `pt.MesoStaged()` — still **500**.
4. With tax 0: `MesoDelivered = 500`; `expandTradeSettlement`
   (`saga/processor.go:1604-1624`) credits **B 500**.
5. Terminal success → `dischargeSettledMesos` (`trade/settlement.go:1330-1340`) zeroes
   the row; `DeleteResolvedMeso` refuses while the stake is armed
   (`escrow/administrator.go:322-328`).
6. The stake resolves with the room gone → `refundOrphanedStake`
   (`trade/processor.go:1480-1505`): `delta = −400 ≤ 0` → refunds nothing (correct in
   isolation). `discardResolvedMeso` retires the row.

Arithmetic: A −100, B +500, tax destroyed 0. **+400 minted.**

#### F-2b — raise in flight resolving while the room is still `SETTLING` → meso DESTROYED

Same shape, other direction:

1. A stages 100 (committed; A −100).
2. A retypes 500 → `delta = +400`, `award_mesos −400`. A is now −500.
3. Both confirm/attest before the stake resolves. `settle` uses `mesoStaged = 100`;
   `MesoDelivered = 100`.
4. The stake's COMPLETED status arrives **while the room is still `StateSettling`**.
   `resolveMesoStake` finds the room (`trade/processor.go:1421`), so the
   room-gone/refund branch at `trade/processor.go:1422-1436` is **not** taken;
   `WithSettledMeso` sets `mesoStaged = 500` — too late, the payload is already out.
5. Settlement succeeds → `dischargeSettledMesos` zeroes and deletes the row.

Arithmetic: A −500, B +100, tax destroyed 0. **400 destroyed.**

If instead the stake resolves *after* the room is removed, `refundOrphanedStake`
returns the +400 delta and the path balances — so the outcome is timing-dependent,
which is itself disqualifying for a ledger invariant.

**Reachability note (stated honestly):** the reference client arms
`m_bExclRequestSent` on `PUT_MONEY` and is unlocked only by the server's
`MESO_STAGED` / `MESO_REFUSED` re-echo, so an unmodified client is unlikely to send
`TRADE_CONFIRM` with its own stake unresolved. I did **not** verify from the client
binary whether `TRADE_CONFIRM` passes through `CWvsContext::CanSendExclRequest`
(UNVERIFIED). The server, however, has no guard at all: a modified client, or the
counterparty's confirm landing on a room whose *other* side has an unresolved stake in
a `2-of-2` sequence, breaks the identity with no error anywhere.

---

### F-3 — BLOCKING (precondition: one Kafka redelivery) — `RestoreItem` is idempotent with itself but not with a later release: a redelivered restore resurrects a row whose item was already handed back

`escrow/administrator.go:144-151`:

```go
db.Unscoped().Model(&ItemEntity{}).
    Where("tenant_id = ? AND id = ?", tenantId, id).
    Updates(map[string]interface{}{"deleted_at": nil, "returning_at": nil})
```

It carries no fencing token — no generation, no "restore only the release with
transaction X". It un-deletes and **un-claims** whatever row bears the id, whenever it
arrives. The custody consumer has no dedupe
(`kafka/consumer/custody/consumer.go:94-103`), and `escrow/entity.go:74-78` states the
at-least-once posture explicitly for the *accept* path (which is why `CreateItem` has an
`OnConflict … DoNothing` guard at `administrator.go:79-84`). No equivalent guard exists
for restore.

Sequence:

1. `trade_settlement` releases escrow row X (`saga/processor.go:1570-1583`) — row X
   soft-deleted.
2. A later step fails. `DispatchTradeTransactionRollbacks` (`compensator.go:1957-1971`)
   dispatches `RestoreTradeEscrow(X)`.
3. The custody consumer processes it — X is live and **unclaimed** (`returning_at` NULL).
4. atlas-trades `SettlementFailed` → `unwindRecord` (`trade/settlement.go:1036-1078`)
   reads X, claims it, submits `trade_unwind`.
5. The unwind's `release_from_trade` soft-deletes X; `accept_to_character` grants the item
   to its owner. **Item delivered.**
6. Kafka redelivers step 3's `RESTORE_TRADE_ESCROW` command (rebalance, uncommitted
   offset, handler retry). `RestoreItem` sets `deleted_at = NULL, returning_at = NULL`.
   X is live and unclaimed again.
7. Next process start: `ReconcileEscrow` (`trade/settlement.go:1214-1280`) reads X via
   `escrow.AllItems` (`escrow/provider.go:110-116`), finds no owning settlement record,
   and unwinds it — granting the same item to the owner a **second** time.

Arithmetic: escrow −1, owner +2. **One item minted.**

The doc comment at `administrator.go:132-143` argues the duplicate "requires an accept
that succeeded, and an accept that succeeded is never followed by a restore of its own
release (the reverse walk removes the granted item first)". That reasoning holds for the
reverse walk's *own* ordering. It does not hold for a **redelivery** of the restore
arriving after a *different* saga (the unwind, minted with its own transaction id at
`trade/settlement.go:1053`) released the row. The settlement's compensator has no
visibility into that unwind.

---

### F-4 — SERIOUS — the settlement compensation and the unwind that follows it are racing on separate topics: items are silently stranded until the next restart

`compensateTradeTransaction` (`compensator.go:1834-1871`) dispatches
`RestoreTradeEscrow` commands (line 1842) and *then* emits `SAGA_FAILED` (line 1858).
These travel on **different topics** — the custody command topic
(`kafka/message/custody/kafka.go:23-29`) and the saga status topic — with no ordering
between them.

atlas-trades reacts to `SAGA_FAILED` in `completeSettlement`
(`trade/settlement.go:1000-1014`) → `unwindRecord`, which reads the escrow **fresh**
(`trade/settlement.go:1037-1044`). Soft-deleted rows are invisible to `ItemsByRoom`
(`escrow/provider.go:15-27` — default GORM scope).

Sequence:

1. Settlement fails after both `release_from_trade` steps completed (rows soft-deleted).
2. `SAGA_FAILED` reaches atlas-trades before the `RESTORE_TRADE_ESCROW` commands reach
   the custody consumer.
3. `unwindRecord` reads `ItemsByRoom` → **empty** → `len(items)==0 && len(mesos)==0` →
   returns `nil` at `trade/settlement.go:1045-1047`. Nothing is returned.
4. The restores land moments later. Both rows are live and unclaimed. Nothing is watching
   for them: there is **no periodic reconciliation ticker** — `main.go:139-143` runs
   `ReconcileAtBoot` exactly once, and `main.go:125-130` explicitly records that the
   refresh loop was deleted.

Arithmetic: giver −1 item, recipient +0, escrow +1, no error surfaced. The identity is
not violated (the item exists in exactly one place), but the asset is unreachable by any
player until the pod restarts. **Recoverable only by hand or by restart.**

The mirror case is F-3 step 4 — the same race resolved the other way — so which outcome
you get is pure timing.

---

### F-5 — SERIOUS — a failed `trade_unwind` is not routed anywhere in atlas-trades: its escrow rows sit until the next boot

`kafka/consumer/saga/consumer.go:112-144` routes a `SAGA_FAILED` by ownership:
`StageFailed` (escrow row id) → `MesoStageFailed` (pending stake id) →
`SettlementFailed` (settlement record id). An unwind's transaction id is minted fresh
(`trade/settlement.go:638`, `:1053`, `:1389`, `trade/processor.go:1138`) and owns **none**
of those three id spaces:

- `StageFailed` → `findStagedItem` misses → returns `(false, nil)` (`trade/processor.go:1060-1064`).
- `MesoStageFailed` → `MesoStakeById` misses → `(false, nil)` (`trade/processor.go:1397-1401`).
- `SettlementFailed` → `GetByTransactionId` → `ErrRecordNotFound` → logged as
  "already resolved. Ignoring" and `nil` (`trade/settlement.go:951-958`).

So a failed unwind is **silently swallowed**. Its own reverse walk
(`DispatchTradeTransactionRollbacks`, since `BuildUnwind` types it `TradeTransaction` —
`saga/producer.go:96-103`) destroys anything already accepted and restores the escrow
rows with `returning_at` cleared, which is correct — but nothing then re-submits the
unwind. The rows wait for `ReconcileAtBoot`.

Concrete trigger (question 10 of the brief): the owner's inventory is full when the
unwind's `accept_to_character` runs. Arithmetic: item in escrow, owner has nothing,
recipient has nothing — conserved but stranded, with no operator signal beyond a log line.

---

### F-6 — SERIOUS — `AcceptToCharacter → RequestDestroyItem(template, quantity)` can destroy an asset that belongs to the recipient

`compensator.go:1937-1956`. The inverse of a settlement accept is a destroy addressed by
**template id and quantity**, because the created asset's id is never reported back
(documented at `compensator.go:1888-1895`).

Two arithmetic consequences:

1. **Wrong instance.** The recipient already owned an instance of the same template. The
   destroy removes *an* instance — possibly the recipient's own scrolled/hammered one,
   leaving the plain traded copy behind. Item **count** is conserved; item **value** is
   not. The alternative (permanent loss) is worse, and the trade-off is explicitly
   documented — but it is a real, silent value transfer.
2. **Recipient's own asset destroyed.** If the recipient consumed/dropped/traded away the
   received item before the compensation lands, the destroy consumes a *different* stack
   of the same template that the recipient owned outright. Meanwhile the escrow row is
   restored and the giver eventually gets their item back. Net: giver whole, recipient
   short one asset they owned before the trade began. **A third asset destroyed.**

Both are inherent to the id-less accept contract, not to task-205's code — but the trade
flow is the first one that runs this inverse on a *two-party* swap, where the recipient
is a second real player rather than the same character.

---

### F-7 — MINOR — a teardown refunds escrowed meso onto the room's stale channel

`emitUnwind` (`trade/settlement.go:649-655`) builds each `TradeUnwindMeso` from
`room.Field().WorldId()` / `ChannelId()`. Three of the four §3.3 teardown triggers are
*channel changes and map changes* (`kafka/consumer/character/consumer.go:76,92,109`), so
the room's field is exactly the one the player has just left.

The two other refund paths deliberately read the player's **current** field and say why:

- `refundOrphanedStake` — `trade/processor.go:1487-1494` ("a teardown is very often a
  channel change — announcing onto the room's old channel would credit the meso where the
  player no longer is").
- `unwindStranded` — `trade/settlement.go:1396-1408`.

`emitUnwind` was not brought in line. Whether the *credit* is lost or only the on-screen
announcement is UNVERIFIED (see §4). If atlas-character routes the award by character id
regardless of the channel on the payload, this is cosmetic; if not, it is a meso-destroying
path on the most common teardown trigger.

---

## 2. Lifecycles that DO balance

Each of these was walked step by step against the code and the arithmetic closes.

| # | Lifecycle | Arithmetic | Evidence |
|---|---|---|---|
| L-1 | **Stage → settle**, both sides carry items, no tax | A: −1 item +1 item; B: −1 +1; escrow ends empty | `saga/processor.go:1570-1602` — all `release_from_trade` precede all `accept_to_character`; release soft-deletes the row (`escrow/administrator.go:92-96`) before any grant, so no window exists in which the row and an inventory copy coexist |
| L-2 | **Stage → settle** with meso and tax | A −100, B −200; B +95, A +190; destroyed 15 = 5+10 = Σtax | `configuration/tax.go:33-44` guarantees `tax + delivered == amount` exactly (integer floor, no rounding leak); `saga/processor.go:1604-1624` is **credit-only** — the debit already happened at stage time; `dischargeSettledMesos` (`trade/settlement.go:1330-1340`) retires the rows so no later sweep re-refunds |
| L-3 | **Stage → teardown**, every §3.3 trigger | escrow → 0, owners restored in full, untaxed | All four triggers land on `TeardownCharacter` (`consumer/character/consumer.go:76,92,109`, `consumer/session/consumer.go:66`, `consumer/trade/consumer.go:202`) → `teardownRoom` (`trade/settlement.go:1474-1485`) → `emitUnwind`. `expandTradeUnwind` (`saga/processor.go:1646-1708`) refunds `m.Amount` raw — **no tax on a refund**, and the refund arm is a deliberately separate composite for exactly that reason (`saga/processor.go:1633-1637`) |
| L-4 | **Stage → settlement FAILS → compensation → unwind** | reverse walk: accepts destroyed, meso credits negated, escrow rows restored; then unwind returns everything; tax destroyed = 0 | `compensator.go:1912-1974` walks newest-first so an accept is undone before its matching release is restored; `unwindRecord` (`trade/settlement.go:1036-1078`) reads the escrow **fresh** rather than replaying the record's snapshot, so a row whose restore failed is not conjured back into existence (`trade/settlement.go:1027-1031`) — subject to F-4's race |
| L-5 | **Restart before the escrow row is written** | item still in the compartment (release not yet run) or returned by the late-stage path | `StageSucceeded` with no room → `returnOrphanedStage` (`trade/processor.go:1110-1146`) claims and unwinds the row that appears after the sweep |
| L-6 | **Restart after the row is written, before atlas-trades knows** | boot sweep returns it once | `ReconcileAtBoot` (`trade/settlement.go:1183-1201`) captures the `owned` exclusion set **before** the settlement pass, which is the correct order — the comment at `:1176-1182` names the exact double-delivery it prevents |
| L-7 | **Restart after settlement submission** | settlement record survives; outcome re-derived from the orchestrator | `settlement.Submit` is written in the same transaction as the saga command (`trade/settlement.go:481-492`); `ReconcileSettlements` (`trade/settlement.go:1430-1457`) treats an UNKNOWN outcome as "leave it", never as failure |
| L-8 | **Terminal status delivered twice — item stage** | one announcement, one unwind | `stageSucceeded` checks `!i.Pending()` (`trade/processor.go:1012-1017`); the orphan branch claims via `ClaimForReturn` (`trade/processor.go:1124-1131`) |
| L-9 | **Terminal status delivered twice — meso stake** | one commit | `CommitMesoStake` / `AbandonMesoStake` are single-statement compare-and-sets on `pending_stake_id` (`escrow/administrator.go:252-283`); the second delivery matches zero rows |
| L-10 | **Terminal status delivered twice — settlement** | one ledger row, one outcome | ledger write is idempotent per settlement transaction and happens **before** the arbiter; `sp.Resolve`'s rows-affected picks the single winner (`trade/settlement.go:960-993`) |
| L-11 | **Terminal status delivered twice — custody accept / release** | one row, one delete | `CreateItem` has `OnConflict{id} DoNothing` (`escrow/administrator.go:79-84`); `DeleteItem` treats a no-match as success (`escrow/administrator.go:86-96`) |
| L-12 | **Meso stake in flight while a teardown happens** | committed part refunded by the teardown, delta refunded by the stake's own resolution | `clearRefundedMesos` **zeroes rather than deletes** a row with an armed stake (`trade/settlement.go:1282-1308`); `refundOrphanedStake` refunds `PendingDelta`, not the absolute total (`trade/processor.go:1480-1505`) — the persisted signed delta at `escrow/entity.go:184-192` is what makes this correct, and a re-derived delta was the documented prior bug |
| L-13 | **Meso stake in flight, then FAILS, room gone** | nothing was debited, nothing refunded, row retired | `resolveMesoStake` `!settled` branch → `discardResolvedMeso` (`trade/processor.go:1422-1430`, `:1530-1539`) |
| L-14 | **Partial stack (1 of 40) → settlement** | giver 40→39, receiver +1 | `expandTransferToTrade` overrides the snapshot quantity with the *staged* amount (`saga/processor.go:1512`); atlas-inventory's `Release` decrements rather than deletes when `quantity < asset.Quantity()` (`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1825-1834`); `assetDataFromSnapshot` carries the staged quantity through (`saga/processor.go:1723`) |
| L-15 | **Partial stack → unwind** | giver 39 → 40 | same snapshot, `expandTradeUnwind` (`saga/processor.go:1673-1686`); remainder in the source slot is never touched, because escrow holds only the staged quantity |
| L-16 | **Same escrow row reached by two return paths at once** | exactly one unwind | `ClaimItemForReturn` is a compare-and-set inside the `UPDATE`'s `WHERE` (`escrow/administrator.go:120-128`), and **every** return path goes through `claimItemsForReturn` — `emitUnwind` (`trade/settlement.go:633`), `unwindRecord` (`:1048`), `unwindStranded` (`:1384`), `returnOrphanedStage` (`trade/processor.go:1124`) |
| L-17 | **Teardown vs. settlement racing the same room** | mutually exclusive | `claimRoom` / `RemoveIf` is a state-checked atomic removal (`trade/settlement.go:1508-1515`), and `teardownCharacter` refuses a `SETTLING` room outright (`trade/processor.go:801-804`) |
| L-18 | **Settlement submitted, then the command fails before publish** | nothing moved; room closed; escrow returned | `recoverAbandonedSettlement` → `abandonSettlement` → `emitUnwind` (`trade/settlement.go:370-402`); the saga was never published so there is nothing to compensate |
| L-19 | **Staging saga fails at `accept_to_trade` (step-reported failure)** | item re-granted from the snapshot | `compensateTradeStaging` → `DispatchTradeStagingRollbacks` (`compensator.go:2451-2515`): `AcceptToTrade → RemoveTradeEscrow` (hard delete, so it can never be restored — `escrow/administrator.go:159-163`) then `ReleaseFromCharacter → RequestAcceptAsset` with the full snapshot |
| L-20 | **Tax on failure paths** | never taken | Tax is computed only inside `settle` (`trade/settlement.go:444-447`) and frozen onto the participant. Every refund path (`emitUnwind`, `unwindRecord`, `unwindStranded`, `refundOrphanedStake`) passes the **raw** `Amount` / `PendingDelta`. There is no code path that applies `configuration.Tax` to a refund |
| L-21 | **`accept_to_character` fails on a full inventory at settlement** | conserved, then unwound | `canCarry` pre-check simulates atlas-inventory's merge-then-spill (`trade/settlement.go:716-761`), and every read failure is a **refusal** rather than an assumption. If the check is beaten by a concurrent inventory change, L-4's compensation applies — subject to F-4 and F-5 |

---

## 3. Coverage map against the brief

| Brief item | Status |
|---|---|
| 1. stage → settle, both sides, ± tax | ✅ balanced — L-1, L-2 |
| 2. stage → teardown, each §3.3 trigger | ✅ balanced — L-3 |
| 3. stage → settlement FAILS → compensation | ⚠️ balanced in principle (L-4) but see **F-4** (race strands), **F-3** (redelivery mints), **F-6** (wrong instance destroyed) |
| 4. restart at each point | ✅ L-5, L-6, L-7; "between a saga's expanded steps" — see §4 |
| 5. terminal status delivered twice | ✅ L-8…L-11 |
| 6. meso stake in flight vs teardown / settlement / restart | teardown ✅ L-12, L-13; **settlement ❌ F-2**; restart ✅ L-6 |
| 7. partial stack through settlement and unwind | ✅ L-14, L-15 |
| 8. same escrow row, two paths | ✅ L-16 |
| 9. tax == staged − delivered, never on a refund | ✅ L-2, L-20 |
| 10. `accept_to_character` fails on a full inventory | settlement ✅ L-21; unwind **F-5**; boot recovery **F-5** |
| — | staging-saga timeout: **F-1** (not enumerated in the brief; found while walking item 4) |

---

## 4. What I could not verify

Marked UNVERIFIED rather than guessed.

1. **Whether `TRADE_CONFIRM` is gated by `CWvsContext::CanSendExclRequest` in the
   reference client.** This decides whether F-2 is reachable from an unmodified client or
   only from a modified one. It does not change the finding — the server has no guard
   either way — but it changes the exploitability. Needs: the v83 client IDB, reading
   `CTradingRoomDlg::Trade @0x7c39a0` (already cited at `trade/settlement.go:825-828`) for
   an `m_bExclRequestSent` test on entry.

2. **Whether `AwardMesosPayload.WorldId` / `ChannelId` affect the credit or only the
   announcement** in atlas-character. This decides whether F-7 is cosmetic or
   meso-destroying. Needs: reading atlas-character's award-mesos command handler and its
   stat-update emission.

3. **Whether the orchestrator re-arms saga timers for sagas recovered from
   `PostgresStore` at boot** (`saga/store.go:227-231` is labelled "for startup recovery").
   If it does not, an orchestrator restart mid-settlement leaves the saga with no
   backstop at all — a *different* stranding shape from F-1. I read the store's recovery
   query but not the boot wiring that consumes it.

4. **Whether a redelivered `ACCEPTED` custody status double-advances an already-`Completed`
   step in the orchestrator.** `CreateItem`'s conflict clause makes the *row* idempotent
   (L-11), and the escrow ack is emitted unconditionally at
   `escrow/processor.go:79`. I did not read `stepCompletedWithResultOnce`'s guard closely
   enough to state that a second `ACCEPTED` for a `Completed` step is inert rather than
   advancing the saga past a step it should still be waiting on.

5. **`ReconcileAtBoot` running concurrently with live consumers.** It is launched in a
   goroutine at `main.go:139-143` while every consumer is already registered
   (`main.go:92-123`). The claim column protects the item side (L-16). I did **not**
   exhaustively walk whether a `MesoStakeById` resolution racing `ReconcileEscrow`'s
   `AllMesos` read can double-refund: `AllMesos` reads `Amount` (`escrow/provider.go:119-125`)
   with no claim mechanism analogous to `returning_at`, and `clearRefundedMesos` runs
   inside the emitting transaction — but the two run under different tenants' processors
   and I could not convince myself of the isolation level in play. **This is the one gap
   I would close first**, because a meso row has no claim latch at all.

---

## 5. Summary

Three findings break a ledger identity outright:

- **F-1 (BLOCKING)** — a timed-out `TradeStaging` saga destroys the item or the meso,
  because `timer.go` dispatches rollbacks for three saga types and `TradeStaging` is not
  one of them. This needs no race and no modified client — just a 30 s stall.
- **F-2 (BLOCKING)** — settlement reads `mesoStaged` and never `pendingMesoTxId`, so a
  stake in flight at settlement mints meso (reduction) or destroys it (raise resolving
  while `SETTLING`).
- **F-3 (BLOCKING, one redelivery)** — `RestoreItem` clears `deleted_at` *and*
  `returning_at` with no fencing token, so a redelivered restore resurrects a row whose
  item the unwind already handed back; the boot sweep then hands it back again.

Three more strand assets or destroy the wrong one: **F-4** (compensation racing the unwind
across two topics), **F-5** (a failed `trade_unwind` is routed nowhere), **F-6**
(template-addressed destroy hits the recipient's own instance). **F-7** is minor.

Twenty-one lifecycles were walked and balance, including every tax path, both partial-stack
paths, all four teardown triggers, all four double-delivery shapes, and the two-path claim
on a single escrow row. The claim latch (`returning_at`) and the persisted signed
`pending_delta` are both doing real conservation work and are correct as written; the
meso row's lack of an equivalent latch is the residual gap flagged in §4.5.

No path was found that balances only because two bugs cancel.
