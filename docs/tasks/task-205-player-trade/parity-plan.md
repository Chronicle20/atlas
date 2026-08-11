# Task-205 follow-on — meso custody parity

**Status:** not started. Written 2026-08-11 as a handoff from the session that
produced commits `a3279ee73`..`0bf941fc5`.

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

- **P1 — the transaction isolation level `libs/atlas-database` opens with.**
  Gates the size of the read-then-CAS windows in the meso paths. If it is
  READ COMMITTED, several windows in the audits are real; under SERIALIZABLE
  some collapse. Check `libs/atlas-database` and the live Postgres
  (`SHOW default_transaction_isolation`), and record the answer here.
- **P2 — does the v83 client's `TRADE_CONFIRM` pass `CanSendExclRequest`?**
  Gates whether a meso stake can still be in flight when a room reaches
  CONFIRM, i.e. whether Task 3's window is reachable from an unmodified client.
  This needs an IDA read of `CTradingRoomDlg`, not reasoning. Note the server
  has no guard either way, so Task 3 is worth doing regardless — but the answer
  decides whether it is BLOCKING or defence-in-depth.

Record both answers in this file before starting Task 2.

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

### Task 1 — Defence A: net the in-flight stake before submitting a new one ✅ *confirmed*

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

**The fix.** The delta must be against *committed + in flight*, the way
`StagedQuantityFrom` nets already-staged quantity for items. Decide deliberately
between two shapes and record the choice:
- net against `Amount + PendingDelta`, letting stakes compose; or
- refuse a second stage while one is in flight (simpler, but re-echoes a stale
  number over what the player just typed — the reason supersession was chosen
  originally; see the comment in `addMeso`).

Whichever is chosen, the superseded stake's debit must be accounted for, never
silently dropped.

**Tests.** Two stages in flight conserve meso exactly; the superseded stake's
status is not silently dropped; the fast-retype sequence above balances.
Mutation-verify by restoring the committed-only delta.

### Task 2 — Defence B: a durable claim latch for meso rows

**The defect (relayed, unconfirmed).** The boot sweep and the orphaned-stake
refund both act on the same meso row with no arbitration; the item twin claims
on both paths via `returning_at`. Also: `discardResolvedMeso` decides from a
**pre-CAS** `row.Amount()` while `CommitMesoStake` assigns
`amount = pending_amount` unconditionally, so a teardown zeroing in that window
makes the guard misclassify and leave a stale non-zero row for the next boot to
refund again.

**The fix.** Give the meso row the same latch discipline the item row has, and
make every decision that ends meso custody read its inputs from the same
statement that acts on them. P1's answer determines how wide these windows are.

**Tests.** Two paths racing one row refund exactly once; the pre-CAS
misclassification is unreachable. Mutation-verify both.

### Task 3 — Defence C: a settlement-time custody check for meso

**The defect (relayed, unconfirmed).** Nothing gates CONFIRM or `settle` on a
pending meso stake. `settlementPayload` uses `pt.MesoStaged()`, which only
advances when a stake resolves, then `dischargeSettledMesos` zeroes the row.
A reduction in flight delivers more than the giver still has staked (**mints**);
a raise resolving after the room is SETTLING is **destroyed**. Which one occurs
is timing. The item twin is protected: `settlementPayload` errors when a staged
item has no escrow row.

**The fix.** The meso equivalent — settlement must fail rather than settle
against a stake whose outcome is unknown. Gate at CONFIRM, at `settle`, or both;
decide and justify.

**Tests.** Settlement with a stake in flight does not mint or destroy, in both
orderings. Mutation-verify.

### Task 4 — register `TradeStaging` in the timeout reverse-walk ✅ *confirmed*

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

**Tests.** A timed-out staging saga rolls back and the asset returns.
Mutation-verify by removing the registration.

### Task 5 — route a failed `trade_unwind`

**The defect (relayed, unconfirmed).** A `trade_unwind` transaction id owns none
of the three id spaces the saga-status consumer probes, so `SAGA_FAILED` for an
unwind falls through all three and is swallowed. Items stay latched by
`returning_at` — cleared only on a completed release — so they are **stranded
permanently and invisible to every future boot sweep**. Meso was already zeroed
in the same transaction that submitted the unwind, so it is **destroyed with no
record**.

**The fix.** Route it. Consider whether "discharge before the return is
confirmed" is the right posture at all, or whether both columns should discharge
only on the unwind's terminal success — that is the deeper question this finding
raises, and it may subsume Task 2.

**Tests.** A failed unwind leaves both columns recoverable. Mutation-verify.

### Task 6 — fence `RestoreItem` against redelivery

**The defect (relayed, unconfirmed).** `RestoreItem` clears `deleted_at` and
`returning_at` for whatever row bears the id, with no fencing token. A
redelivered `RESTORE_TRADE_ESCROW` arriving after the unwind already released
the row and granted the item resurrects it live and unclaimed, and the next boot
sweep grants it again. `CreateItem` has an `OnConflict…DoNothing` guard for
exactly this at-least-once posture; restore has none. The existing doc comment
argues this is impossible, but its reasoning covers only the reverse walk's own
ordering, not a redelivery racing a different saga's release —
**treat that comment as suspect, not as evidence.**

**Tests.** A redelivered restore after a completed release is inert.
Mutation-verify.

### Task 7 — re-run both audits, then the full gate

Re-run the symmetry matrix and the conservation audit against the result, as
fresh read-only passes. A BLOCKING finding means the pass is not done.

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
