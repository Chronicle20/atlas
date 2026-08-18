# Bug: the world-transfer coupon (5401000) is never consumed on APPLY

> **Status: FIXED in `5ffaad846`** — "fix(task-227): consume the world-transfer
> coupon on APPLY". Gate `tools/verify.sh --quick --base e9122b595` PASS.
> Review `APPROVED_WITH_FINDINGS`, 0 blocking (see
> `review-world-transfer-coupon-fix.md`); the single non-blocking finding — no
> `StatusExpired` coverage — was fixed and amended into the same commit.
> **Not yet re-tested live**, and the flagless `tools/verify.sh` has not re-run
> since this commit (the branch's last flagless PASS was `88b53f027`). See
> Resolution at the bottom.

Live evidence from `atlas-pr-1370` (env `aa3d`), tenant
`d606f1cb-ba79-45ca-a989-cf0dc956fee7`, character 1 (`Hard`), account 1
(`Atlas`), 2026-08-17. Found during a post-test database audit of the
name-change + world-transfer flows.

## Symptom

The world transfer applied successfully (world 0 → 1), but the character still
holds the 5401000 World Transfer coupon in their cash inventory, unexpired.
The coupon can be reused, so a player gets unlimited world transfers from a
single purchase.

## Evidence

Character state — the transfer landed:

```
atlas-characters-aa3d . characters
 id | account_id | world | name | level | job_id
  1 |          1 |     1 | Hard |   200 |    322
```

```
atlas-characters-aa3d . character_pending_changes
 type           | status  | destination_world_id | source_world_id | resolved_at
 WORLD_TRANSFER | APPLIED |                    1 |               0 | 2026-08-17 18:46:32.269893+00
```

The coupon is still live in the character's cash inventory (inventory_type 5),
10 minutes after the transfer resolved:

```
-- at now() = 2026-08-17 18:56:30.803636+00
atlas-inventory-aa3d . assets JOIN compartments
 id | slot | template_id | quantity |          expiration           | deleted_at | character_id | inventory_type
 32 |    1 |     5401000 |        1 | 2026-11-15 18:46:03.245581+00 |            |            1 |              5
```

Contrast with the name change on the same character, which *does* consume its
coupon. Full cash-inventory history for character 1:

```
 id | slot | template_id |          created_at           |          deleted_at
 27 |    1 |     5400000 | 2026-08-16 15:29:27.431719+00 | 2026-08-16 15:54:08.818319+00
 28 |    2 |     5400000 | 2026-08-16 15:29:48.65901+00  | 2026-08-16 15:54:06.91234+00
 29 |    1 |     5400000 | 2026-08-16 16:01:51.244688+00 | 2026-08-16 18:58:28.422008+00
 30 |    1 |     5400000 | 2026-08-16 18:58:36.189648+00 | 2026-08-16 19:18:23.042953+00
 31 |    2 |     5400000 | 2026-08-16 19:18:08.899811+00 | 2026-08-16 19:18:23.095089+00
 32 |    1 |     5401000 | 2026-08-17 18:46:11.878084+00 |          (still live)
```

Every 5400000 is soft-deleted; the single 5401000 is not.

## Root cause

Coupon consumption is wired only into the **name-change** apply path.

`services/atlas-character/atlas.com/character/pending_change/producer.go:94`

```go
var nameChangeCouponTemplateIds = []uint32{5400000}
```

`services/atlas-character/atlas.com/character/pending_change/processor.go:426`
— inside `applyNameChange`, and this is the **only** call site of
`consumeCouponsCommandProvider`:

```go
for _, templateId := range nameChangeCouponTemplateIds {
    if err := buf.Put(sagamsg.EnvCommandTopic, consumeCouponsCommandProvider(m, templateId)); err != nil {
        return err
    }
}
```

The world-transfer apply path (`startWorldTransfer`, same file) dispatches the
five-step saga built by `worldTransferCommandProvider` and never enqueues a
consume step. The saga that ran confirms it — `df3a97f3-fddb-5d44-9e69-37019b308d24`,
`saga_type=world_transfer`, `status=completed`, five steps:

```
validate_world_transfer      completed
leave_guild_for_transfer     completed
leave_party_for_transfer     completed
sever_buddies_for_transfer   completed
change_character_world       completed
```

No `destroy_all_assets` / consume arm. Compare the name-change apply, which
emits a separate saga `4d27705e-833d-5778-ad56-89645908c9f4`:

```
stepId: consume_name_change_coupons
action: destroy_all_assets
payload: {templateId: 5400000, characterId: 1}
status: completed
```

`consumeCouponsCommandProvider(m Model, templateId uint32)` is already generic
over the template id, so the mechanism exists — it is simply never invoked for
`WORLD_TRANSFER`.

The channel-side item-use handler does not compensate for this: the 5401000 arm
in `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
routes to the request/cancel flow and never destroys the item. Consumption at
APPLY is the intended design (see the comment at `producer.go:138-149` — the
purchase path materialises the coupon after the request, so apply is the only
reliable point), and that design was implemented for one of the two flows.

## Fix

### Ruling on approach (decided before dispatch — do not re-litigate)

Move coupon consumption **into `Resolve`, on the APPLIED branch, keyed by
`m.Type()`**, and delete the loop from `applyNameChange`. Do not bolt a second
ad-hoc consume onto the world-transfer callback.

Rationale — `Resolve` is already the single chokepoint every terminal
transition passes through, and it already carries the exact symmetric case
directly above:

```go
if status != StatusApplied && m.HasAsset() {
    ... awardAssetCommandProvider(m)   // refund on every non-APPLIED exit
}
```

Consumption is the APPLIED-side mirror of that refund and belongs beside it.
Three consequences that make this the right call rather than merely the tidier
one:

- **Idempotency comes for free.** `Resolve` already guards on `moved`, so a
  redelivered resolve emits nothing (design §3.10). A consume placed on the
  world-transfer REST callback would sit outside that guard.
- **It removes the asymmetry that *is* the root cause.** A future third change
  type gets consumption by construction instead of by remembering.
- **The ordering objection does not apply.** `applyNameChange` currently emits
  the consume before `Resolve` so that a failed enqueue aborts the transaction;
  inside `Resolve` the emit is on the same `*message.Buffer` in the same
  transaction, so that property is preserved.

### Files

| File | Change |
|---|---|
| `services/atlas-character/atlas.com/character/pending_change/producer.go:94` | Add `worldTransferCouponTemplateIds = []uint32{5401000}` next to the existing `nameChangeCouponTemplateIds`. Carry over the grounding comment style — 5401000 is **not** an assumed value (see Grounding below). |
| `services/atlas-character/atlas.com/character/pending_change/producer.go:150` | `consumeCouponsCommandProvider` hardcodes the step id `"consume_name_change_coupons"`. Make the step id type-appropriate (e.g. pass it in, or derive `consume_world_transfer_coupons`). Keep `sagaTransactionId(m, sagaPurposeConsumeCoupon+":"+templateId)` deterministic — that determinism is what makes a redelivery a no-op rather than a second destroy. |
| `services/atlas-character/atlas.com/character/pending_change/processor.go:303-310` | In `Resolve`, add the APPLIED-side branch beside the existing refund branch: on `status == StatusApplied`, emit one `consumeCouponsCommandProvider(m, templateId)` per template id for `m.Type()`. |
| `services/atlas-character/atlas.com/character/pending_change/processor.go:425-430` | Delete the now-duplicated consume loop from `applyNameChange` (leave the surrounding comment's intent — consumption-at-apply — reflected at the new site). |

Type → template ids: `TypeNameChange` → `nameChangeCouponTemplateIds`,
`TypeWorldTransfer` → `worldTransferCouponTemplateIds`. Constants are at
`entity.go:14-15`.

### Grounding for 5401000

Not invented and not remembered — `derivation.md` §3 reads
`CCashShop::ProcessBuy` across all nine GMS versions and finds 5401000 compared
as an EXACT id, the world-transfer sibling of 5400000 (§3 line 314:
"**`5400000` is name change. `5401000` is world transfer.**"). jms_v185 has no
5400000 at all and maps 5401000 to the world-transfer flow (§1.5), so the
world-transfer coupon id is universal across every configured version. The same
value is already used at
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1112`.

### Tests

`services/atlas-character/atlas.com/character/pending_change/coupon_consumption_test.go`
asserts on the literal string `consume_name_change_coupons` at lines 32, 40 and
69. If the step id changes, these must move with it — and the name-change
assertions must keep passing, since that path's observable behaviour must not
regress.

Add the world-transfer counterpart in the same file: a WORLD_TRANSFER record
resolved to APPLIED emits exactly one consume step for 5401000; resolved to
REJECTED or CANCELLED emits none. Use the project Builder pattern for setup —
no `*_testhelpers.go`.

Module-local verification only: `go build ./... && go test ./...` from
`services/atlas-character/atlas.com/character`.

## Not yet answered

- **Refund on a failed world transfer is a separate, pre-existing gap.** The
  refund branch is gated on `m.HasAsset()`, and `asset_id` is NULL on the
  purchase path (deliberately — see `bug-purchase-path-sets-assetid.md`), so a
  REJECTED world transfer refunds nothing today. That is out of scope for this
  fix and must not be "fixed" opportunistically here; the coupon is still in
  the character's inventory on that path, so nothing is lost — it is only the
  APPLIED path that leaks.
- **The two `compensating` sagas** (secondary observation 1 below) are not
  addressed by this change.
- **The already-leaked row** — `atlas-inventory-aa3d.assets` id 32 — is live
  test data in `atlas-pr-1370`. Nothing in this fix retroactively destroys it;
  it will need a manual delete or a fresh character to re-test cleanly.

## Secondary observations (not this bug)

1. **Two sagas stuck in `compensating`.** `e9be7f6d-e314-58dc-8157-640cf70e1dd9`
   (created 2026-08-16 15:29:17, `timeout_at` 15:34:17) and
   `bb3f7c0d-0020-5ad9-b02e-4a7b1dfc4556` (created 16:01:48, `timeout_at`
   16:06:48) are both `cash_shop_operation` / `consume_pending_change_coupon`
   with the step still `pending`, 30h past their timeout. These are residue of
   the already-documented `assetId` bug (see
   `bug-purchase-path-sets-assetid.md`); the later flows correctly leave
   `asset_id` NULL. Worth noting only because `compensating` appears to have no
   terminal sweep — `saga/store.go:230` re-picks `active`/`compensating` on
   startup recovery, so these rows will be retried indefinitely.

2. **Orphan world-0 storage row.** `atlas-storage-aa3d.storages` holds both a
   world-0 and a world-1 row for account 1; account 1 now has no character in
   world 0. The world-0 row is empty (capacity 4, 0 mesos, 0 `storage_assets`),
   so nothing is stranded. Storage is account+world scoped, so this may well be
   intended — flagged for a decision, not as a defect.

## What was checked and found clean

Swept all 35 `*-aa3d` databases.

- **No stale name copies anywhere.** Scanning every `text`/`varchar` column in
  every database for the literal `Atlas` returned only legitimate hits:
  `accounts.name` (the account is named Atlas), `bans.login_history.account_name`,
  and `character_pending_changes.requested_name` (historical request records).
  Scanning for `Hard` hits exactly `characters.name`,
  `character_rankings.name`, and the pending-change history.
- **World scoping is correct** in `maps.character_locations` (world 1),
  `characters.session_history` (world 1 post-transfer),
  `rankings.character_rankings` (world 1, restamped by the 18:51:07 recompute
  cycle — note this table is eventually consistent and lagged the transfer by
  ~5 minutes, which is the recompute interval, not a defect).
- **Severances applied.** `guilds.members`, `guilds.characters`,
  `buddies.buddies`, `families.family_members` are all empty; the saga's
  `sever_buddies_for_transfer` step carried `buddyIds: [2]` and both directions
  are gone.
- **Character-scoped data survived intact**: 28 skills, 5 quest statuses, 40
  key bindings, 5 inventory compartments, equipment and consumables all still
  bound to character 1.
- **No stuck outbox entries.** All 19 databases with an `outbox_entries` table
  report `unsent=0`, `witherror=0`.
- **No stranded world-scoped economy state**: `merchant.shops`,
  `mts.listings`, `mts.holdings`, `mts.wish_entries`, `mts.mts_transactions`,
  `trades.trade_ledger_sides`, `storage.storage_assets` are all empty.
- **The source of truth is correct.** `GET /api/characters/1` returns
  `worldId: 1`, `name: "Hard"`.

## Resolution

Fixed in `5ffaad846` (single commit; the review follow-up was amended in, not
added as a second commit).

Implemented exactly as the `## Fix` ruling specified — consumption moved into
`Resolve`'s APPLIED branch keyed by `m.Type()`, `worldTransferCouponTemplateIds
= []uint32{5401000}` added, the saga step id derived from change type
(`consume_world_transfer_coupons` vs `consume_name_change_coupons`), and the
duplicated loop deleted from `applyNameChange`. An unrecognised change type
consumes nothing rather than guessing.

Files: `producer.go`, `processor.go`, `coupon_consumption_test.go` — all in
`services/atlas-character/atlas.com/character/pending_change/`.

### Verification

- **Gate**: `tools/verify.sh --quick --base e9122b595` → PASS (go build/vet, go
  analyzer guards, skill/job id guard, scope guard, producer seam guard, env
  domain guard, lint & format guard).
- **Module tests**: `go build ./... && go test ./...` from
  `services/atlas-character/atlas.com/character` → all packages pass.
- **Review**: `APPROVED_WITH_FINDINGS`, 0 blocking, 1 non-blocking (missing
  `StatusExpired` coverage) — fixed and amended in. Full reasoning in
  `review-world-transfer-coupon-fix.md`. The review confirmed the new
  `TestApplyConsumesTheWorldTransferCoupon` fails against pre-fix code, so it is
  a genuine regression pin rather than vacuous coverage, and that it drives the
  real production path (`ResolveAndEmit`, the saga-completion callback at
  `resource.go:281-286`) rather than the name-change-only `ApplyForCharacter`.
- **Seam check**: atlas-saga-orchestrator keys step dispatch off `action`, not
  `stepId` (`saga/processor.go` uses `StepId()` only for logging and
  intra-saga uniqueness), so the step-id rename does not affect the orchestrator.
  The `action` (`sharedsaga.DestroyAllAssets`) and payload shape are unchanged.

### Still outstanding

- **Not re-tested live.** The fix has not been exercised against
  `atlas-pr-1370`. A clean re-test needs a character that has not already
  leaked a coupon — `atlas-inventory-aa3d.assets` id 32 is still live for
  character 1 and this fix does not retroactively destroy it.
- **Flagless `tools/verify.sh` has not re-run** since this commit. The branch's
  last flagless PASS was `88b53f027`; per CLAUDE.md the flagless gate must exit
  0 before the branch is called ready for PR. `--quick` skips the bake and
  `-race`.
