# Follow-up: evaluate destination-independent eligibility at check time

**Status:** filed, not yet a task. Run `/spec-task` against this document to give it a
number and a worktree; `tools/task-numbers.sh next` picks the number. Filed by user
ruling during task-227 Task 26 — the ruling was **ship task-227 as-is and queue this**,
not "defer because it is hard."

**Origin:** task-227 Task 26 (commits `3264eebbd`, `768966c32`).

## The gap, precisely

The cash shop's two availability-check handlers —

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go`

— validate the account credential and then answer **ALLOWED unconditionally.** They
evaluate no eligibility gate of any kind.

The gates are not missing from the system; they are enforced later, at
`services/atlas-character/atlas.com/character/pending_change/processor.go:205`
`Create()`, which runs `evaluateTransferEligibility` (world transfer) or
`CheckNameValidity` (name change) and returns `IneligibleError`. With task-227's
insert-first ordering, `Create()` runs **before** any charge, so the player is never
billed for a change that then fails.

**So this is a UX defect, not a security or money defect.** The cash shop tells the
player "yes, you may rename / transfer," and the subsequent purchase then fails with a
reason the check should have surfaced a step earlier.

## Why Task 26 could not close it

Structural, and verified rather than assumed:

- The only exposed entry point is `GET /characters/{characterId}/transfer-eligibility`
  (`pending_change/resource.go:37,69`), and `handleGetTransferEligibility` requires a
  `destinationWorldId` on **every** call.
- The serverbound `WORLD_TRANSFER` check body carries `characterId + credential` only
  (jms: credential only). It carries **no destination world** — re-derived from the
  client in task-227 Task 16. The target world arrives later, on `BUY_WORLD_TRANSFER`
  (`ShopOperationBuyWorldTransfer.TargetWorld`).
- Passing a placeholder is not an option: world `0` is a real world, and the endpoint
  would answer about it.
- `NAME_TRANSFER`'s check body has the same shape — task-227 Task 15 proved it carries
  **no candidate name**, so name validity is equally unevaluable at check time.

## What the work is

`evaluateTransferEligibility` (`pending_change/eligibility.go:105`) is one ordered
table mixing both kinds of gate. Split it:

**Destination-independent — evaluable at check time:**
`is_gm`, `banned`, `is_guild_master`, `in_family`, `trade_open`, `merchant_open`,
`mts_listings_open`

**Destination-dependent — must stay at purchase time:**
`world_same`, `world_unknown` / `world_full`, `no_character_slot` (in destination),
`name_taken` (in destination)

Then:

1. Extract the destination-independent prefix into its own method, with
   `evaluateTransferEligibility` calling it first so there is exactly ONE definition of
   each gate. Do not fork the table — a second copy will drift, and a drifted
   eligibility table is the failure mode this whole area is most exposed to.
2. Expose the prefix over REST (a query-parameter variant of the existing route, or a
   sibling route — decide during design; the existing route's contract must not change).
3. Add the atlas-channel client for it, mirroring the existing `pendingchange` REST
   client.
4. Call it from both check handlers, after credential validation, and map each
   rejection reason onto the already-verified clientbound reason codes.

Preserve the table's documented ordering property — cheapest and most local first, so
an obviously-invalid request never fans out to eight services — and its **fail-closed**
posture: a remote gate that cannot be verified rejects, and must never be treated as
silently eligible.

## Acceptance

- Both check handlers answer with a real reason code when a destination-independent
  gate rejects, and ALLOWED only when all of them pass.
- No gate is defined in two places — assert it by having the purchase-time path call
  the same prefix method, and by a test that a new gate added to the prefix is
  observed by both paths.
- The four destination-dependent gates still run at `Create()` and still reject there;
  a test pins that the check op does **not** attempt them.
- Fail-closed preserved: a remote-gate error at check time answers unavailable.
- No change to the credential validation task-227 landed, and no change to the
  existing `transfer-eligibility` route's contract.

## Not in scope

The clientbound reason-code codecs already exist and are verified per version
(task-227 Phase D, Tasks 17-22). This task consumes them; it does not add or re-derive
any packet.
