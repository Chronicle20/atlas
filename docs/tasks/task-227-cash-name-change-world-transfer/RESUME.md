# task-227 — resume state

**Updated 2026-08-14, controller session 3.** The blocker this file previously
opened with is RESOLVED — both of its sub-blockers were answered and the user
ruled. Read "Current state" first; the historical blocker narrative below is kept
for the evidence chain, not as an open question.

## Current state

**Updated end of controller session 3.** Ledger
`.superpowers/sdd/plan/progress.md` (git-ignored but it HAS survived two clears) is
the authority on per-task state; this file is the narrative.

### Landed this session
- `7183e75aa` docs — currency derivation, plan Phase H, Task 28 rewrite
- `5f69feac0` docs — plan Phase I (the pre-v95 credential)
- `ce5b8e246` **code** — Task 40: birth date in atlas-account + atlas-channel mirror.
  `tools/verify.sh --quick` PASS (exit 0, 86 modules, all guards). Task review was
  in flight at handoff — **check the ledger for its verdict before assuming clean.**

### Resolved blockers (do not relitigate)
- **Currency: DERIVED.** Client hard-codes NX Prepaid for both buy ops; pass
  `isPoints=false`, `currency=0`. Evidence in `buy-currency-derivation.md`.
- **User ruling: Option A** — `BUY_*` routes through `RequestPurchase`, insert-first,
  done body emitted from the purchase-outcome consumer with the real `AssetId`.
  Planned as Phase H, Tasks 37/38/39.
- **User ruling: add a birth date to atlas-account** for the pre-v95 credential.
  Planned as Phase I; Task 40 is DONE, Task 41 folded into Task 26 (see ordering).

### Controller ruling on ordering — READ BEFORE RESUMING
**Task 40 runs before Task 26, and Task 41 is folded INTO Task 26.** As originally
planned, Task 26 would have had to answer availability checks with no working
pre-v95 credential comparison, landing either a security hole or a stub — both
forbidden. Task 40 is now done, so **Task 26 must implement BOTH version paths in
one commit**: `Spw()` vs `PIC()` on v95/jms, `BirthDate()` vs the new stored birth
date on v48–v92, fail-closed when the stored value is unset.

### Next action
Resume at **Task 26**, using `.superpowers/sdd/plan/task-26-brief-cont.md` PLUS
the Task 41 content now folded in. **Note the continuation brief contains one
error of mine**: it says to compare `.PIC()` against the decoded SPW, which is
correct ONLY on v95/jms. Correct that when you re-brief. Two prior Task 26
dispatches produced NO code — both were aborted by the turn-budget hook bug, not
by real problems.

### The turn-budget hook — THE USER OWNS THIS FIX
Do NOT touch `.claude/hooks/turn-budget.sh` in either tree. The registered hook is
the MAIN repo's copy (`$CLAUDE_PROJECT_DIR` resolves to the main repo even for
worktree sessions). Until the user's fix lands, the controller MUST zero the counter
immediately before every dispatch:
`printf '0' > /tmp/claude-turn-budget/<YOUR session id>` — read your own session id
from your scratchpad path; do not reuse a previous session's.

### Still outstanding
Tasks 26, 27, 28 (rewritten), 29–36, 37–39. Flagless `tools/verify.sh` owed at
branch end (Task 35). Commit `4a5d9ff65` (client cancel path) remains UNREVIEWED by
user ruling — deferred to the branch-end whole-branch review.

---

## Historical: the blocker as first written


## Historical: the blocker as first written

## Where the plan stands

plan.md Tasks 1–25 are complete and committed. Plus one out-of-plan task the
user ruled in (the client cancel path). Branch head at pause: `4a5d9ff65`.

| Commit | What |
|---|---|
| `972ff8f3a` | Task 23 + deploy-notes.md |
| `420cf2573`, `6de1ae5b7` | Task 24 (pendingchange REST client) |
| `ceb6639ba` | fix: preserve decoded reason on 409 |
| `7ea605251` | fix: SA4004 in atlas-character |
| `98213d81e` | Task 25 (BUY_* handlers) |
| `723f60ab8` | docs: cancel-path IDA derivations |
| `4a5d9ff65` | client cancel path (out of plan, user-ruled) |

Last gate run: `tools/verify.sh --quick` **PASS, exit 0** — but that was at
`98213d81e`. `--quick` skips docker bake and `-race`, so it is NOT a pre-PR pass.
The flagless run is still owed (plan Task 35).

## BLOCKING — decision needed before any further implementation

### The defect

Task 25's `BUY_NAME_CHANGE` / `BUY_WORLD_TRANSFER` arms emit a done body
containing a `CashInventoryItem` with `CashId = 0`. **This is confirmed unsafe**,
by IDA derivation (`cash-inventory-item-zero-fields.md`, committed):

- Both done-body handlers `DecodeBuffer` the item into the client's cash locker
  array at `this+290` (v83: `0x47bccb`, `0x47bfa2`).
- `CCSWnd_Locker::OnMouseButton` (`0x4b053b`) reads the clicked slot's `CashId`
  and passes it to `CCashShop::OnMoveCashItemLtoS` (`0x472632`), which re-resolves
  the slot by **scanning the locker for a CashId match** and **echoes that CashId
  back to the server** on the locker-withdraw op.
- Our own code closes the loop: `MoveFromCashInventory` →
  `saga.WithdrawFromCashShopPayload{CashId: serialNumber}`.

So two `CashId == 0` items coexisting in the locker make the withdraw ambiguous,
and in all cases a fabricated id is echoed. `Expiration = 0` is **UNVERIFIED** —
no read path was found, which is not the same as proof it is unused.

There is no small fix: **nothing in the codebase fabricates a CashId.** Every
sibling site (`cash_shop_entry.go:86-95`, `kafka/consumer/cashshop/consumer.go`
:135-142, :181-188, :274-281) resolves a real `asset.Model` and reads
`.CashId()`. These two arms cannot, because **this flow never creates a cash
asset at all.**

Root cause is design.md §3.1/§5.1, which route `BUY_*` straight to the
pending-change record with no asset creation. **Task 25 implemented the design
faithfully.**

### The user's ruling, and why it stalled

User chose: route `BUY_*` through `cashshop.RequestPurchase` so a real asset
exists, then emit the done body from the purchase-success consumer using the real
`AssetId` — mirroring `consumer.go:135-142`.

A source investigation then found **the chosen shape cannot be built as drawn.**
The user picked it believing it matched the sibling pattern; it does not. Three
findings, all file:line confirmed:

1. **Currency is undeterminable — a genuine blocker.**
   `RequestPurchase(characterId, serialNumber uint32, isPoints bool, currency uint32, zero uint32)`
   treats currency strictly as a client-declared wallet selector
   (`atlas-cashshop cashshop/processor.go:98-127`; `wallet/model.go:37-40` maps
   1→credit, 2→points, else prepaid). Neither `BUY_*` codec carries a currency
   field, and **no commodity record carries one** — atlas-cashshop's
   `commodity.Model` has `Price` but no wallet type. No server-side rule infers
   it anywhere. Needs an IDA derivation of real client behaviour, or an explicit
   product decision. **Do not let an implementer guess this.**

2. **No correlation carrier exists.** Moving the done body to the consumer means
   `requestedName` / `targetWorld` must cross the async boundary. Today:
   `RequestPurchaseCommandBody{Currency, SerialNumber}`,
   `PurchaseEventBody{TemplateId, Price, CompartmentId, AssetId, ItemId}`,
   `StatusEvent{WorldId, CharacterId, Type, Body}` — no transaction id, no op
   tag. `handleStatusEventPurchase` keys only off `CharacterId` and cannot tell a
   name-change buy from any other concurrent BUY. `ErrorEventBody{Error,
   CashItemId}` is equally op-blind. Closest precedent
   (`OpenSurpriseCommandBody.TransactionId`) carries an opaque UUID and its
   success event does not echo it back. `session.Model` has no attribute bag.
   => requires **new fields on three message bodies** — a cross-service
   wire-format change, the first of its kind here.

3. **Ordering: insert-first is the only option that reuses existing machinery.**
   `Resolve()` mints a refund only when `status != StatusApplied && m.HasAsset()`
   (`pending_change/processor.go:287-306`), and `HasAsset()` is **false on the
   purchase path by construction** (`entity.go:69-74`). atlas-cashshop has **no
   void/refund command at all** (`Expire` deletes the asset and never credits the
   wallet). So purchase-first + a name-taken 409 = player charged with nothing,
   reversible only by building new refund machinery. Insert-first + purchase
   failure = release the unpaid PENDING row via the already-tested cancel path,
   minting no spurious refund.
   *(Correction: an earlier controller note claimed the refund machinery already
   supported purchase-first. That was wrong — it was inferred from
   `refund_idempotency_test.go` without checking `HasAsset()`.)*

**Also discovered: a pre-existing hole unrelated to this fix.**
`pending_change/processor.go:250-256` states the purchase path's entitlement is
consumed by atlas-cashshop off the `PENDING_CHANGE_CREATED` event. **That
consumer does not exist** — grep for `PENDING_CHANGE_CREATED` /
`PendingChangeCreated` across `services/atlas-cashshop/` returns nothing. And
`TransactionId` is minted inside atlas-character and never returned (channel-side
`pendingchange.RestModel` has no such field). This is a *different* unbuilt
design from the user's chosen channel-driven shape.

### Options to put to the user on resume

- **A — build it properly**: insert-first, add a correlation id to the three
  message bodies, new consumer arm, plus a currency decision. Authentic, but new
  cross-service surface.
- **B — minimal safe**: stop emitting the `CashInventoryItem`-bearing done body.
  No fabricated CashId reaches the client; player sees no locker entry (likely
  inauthentic). Small, safe, unblocks the branch.
- **C — reconcile with design.md's own shape** (atlas-cashshop consumes
  `PENDING_CHANGE_CREATED`) rather than the channel-driven shape. Also unbuilt;
  larger; but it is what the design document actually says.

Currency (blocker 1) must be resolved for A regardless of ordering.

## Also outstanding

- **plan.md Task 28 must be REWRITTEN before it is executed.** As written it
  builds `cancel-path-guard.sh` asserting "no client cancel path exists" — now
  proven **false**. A green guard asserting a falsehood is worse than none.
  Replacement property (true, checkable, preserves the original security intent):
  *"the OPERATOR cancel route is not reachable from any socket handler"* — assert
  no file under `services/atlas-channel/.../socket/handler/` references the
  operator DELETE path or the `operator_cancelled` reason. Cite both derivation
  docs in the guard's header so nobody "restores" the old property.
  `4a5d9ff65` already flagged this in design.md but did **not** touch plan.md.
- **`4a5d9ff65` is UNREVIEWED and UNVERIFIED.** Its implementer was stopped
  immediately after committing, before writing its report — so
  `.superpowers/sdd/plan/client-cancel-report.md` does not exist. The commit
  message is detailed and self-describing (read it). It claims the
  cross-character red run was captured: expected 404, **actual 204** before the
  fix — confirming the operator DELETE route's missing ownership check was a real
  hole, not theoretical. **Re-verify that claim**; it was not independently
  reviewed. Run a review + `atlas-verifier` on this commit on resume.
- Plan Tasks 26, 27, 29–36 not started. **Task 26 must register the three
  check-result writers in `produceWriters()`** (`atlas-channel/.../main.go:629`)
  — six clientbound writers added in Phase D are still absent from it, so those
  codecs are verified in the matrix but no code path can emit them.
- Flagless `tools/verify.sh` (bake + `-race`) owed at branch end.

## Standing rules earned on this branch — do not relearn these

1. **Never two `atlas-verifier`s at once** — concurrent golangci-lint produces
   `parallel golangci-lint is running` and phantom failures in unrelated modules.
   Cost a full round already.
2. **Verifier must report EVERY failing module**, not just the first block. Grep
   the log for all `LINT FAIL` markers. A second failure in another module hid
   behind the first and shipped.
3. **Serialize implementers** — one at a time in this worktree. Two committing to
   one branch is a hand-untangle.
4. **Treat briefs and reports as claims, not authority.** Ten instances on this
   branch where prose contradicted source; twice it would have inverted runtime
   behaviour. Several were introduced by the controller, not the implementers.
   What catches it: deriving from source/IDA and demanding red/green evidence.
   What fails: prose copied plan → report → brief with nobody opening the file.
5. **Writer registration follows whoever EMITS** (supersedes the earlier "Tasks
   26 and 27 own all six together").
6. `pendingchange`'s `assetId` param **is an item TEMPLATE id** — passing
   `com.ItemId` is CORRECT. One agent flagged this as a bug by inferring from the
   parameter name; the doc comment and Tasks 6/7 settle it. Do not "fix" it.

## Phase F warning (Tasks 29–32)

Four services mirror `NAME_CHANGED` by hand. Highest-risk remaining surface for
the seam class the verify gate cannot see. Brief it as **"enumerate consumers
from source, do not sample."**
