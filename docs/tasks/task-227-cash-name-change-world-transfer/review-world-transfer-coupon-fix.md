# Review: fix(task-227) consume the world-transfer coupon on APPLY

Commit range reviewed: `e9122b595..e62ca6e99` (single commit `e62ca6e99`).

Scope: `services/atlas-character/atlas.com/character/pending_change/{producer.go,processor.go,coupon_consumption_test.go}` — matches `git diff --stat` exactly, matches the bug file's `## Fix` → `### Files` table exactly (all four named hunks touched, no others).

## 1. Regression risk to the name-change path

- `applyNameChange` (processor.go:400-445) no longer emits the consume loop directly; it now falls through to `Resolve(buf)(m.Id(), StatusApplied, "")` at processor.go:441, unchanged call site.
- `Resolve`'s new APPLIED branch (processor.go:317-323) calls `couponTemplateIdsForType(m.Type())`. For `TypeNameChange` this returns `nameChangeCouponTemplateIds = []uint32{5400000}` (producer.go, unchanged var), so exactly one consume command is still emitted, keyed to `templateId=5400000`.
- `couponConsumptionStepId` (producer.go, new) returns `"consume_name_change_coupons"` for anything that is not `TypeWorldTransfer` — i.e. the `default` arm — so the step-id string is byte-for-byte unchanged for name-change. Verified against `TestApplyConsumesTheNameChangeCoupons` (coupon_consumption_test.go:20-49), which still asserts on the literal string and still asserts `destroy_all_assets` / `5400000` counts of 1. Ran `go test ./pending_change/... -run TestApplyConsumesTheNameChangeCoupons` — not individually re-run here, but the full package run (`go test ./pending_change/...`) passed, and this test is unmodified per the diff stat (no change to lines 1-49 of the test file other than new tests appended below).
- **Ordering/transaction property preserved.** The original comment claimed the emit happened *before* `Resolve` so a failed enqueue aborts the transaction. The new site is inside `Resolve`, which is itself always invoked with the same `buf`/`mb *message.Buffer` inside the same `outbox.EmitProvider` transaction as before (`applyNameChange` still wraps the whole thing in `database.ExecuteTransaction` + `message.Emit`, processor.go:401-402). A `mb.Put` error inside the new branch (processor.go:319-321) returns `Model{}, false, err` immediately, propagating up through `Resolve`, `applyNameChange`, and aborting the transaction — same property, different call frame. Confirmed by reading processor.go:294-330 directly, not inferred.

Verdict: no regression on the name-change path. PASS.

## 2. Idempotency contract

- `Resolve` guards on `moved` at processor.go:300-303: `if !moved { ...; return m, false, nil }` — this returns *before* both the refund branch and the new consume branch. The new branch at processor.go:317-323 sits strictly after that guard, so a redelivered resolve (second call with the same terminal id) short-circuits at line 302 and never reaches the consume emit. Confirmed by direct line read, not description.
- `sagaTransactionId(m, sagaPurposeConsumeCoupon+":"+strconv.FormatUint(uint64(templateId), 10))` (producer.go:190) is unchanged by this diff — the only change to that line is the step-id argument to `AddStep`, not the transaction-id derivation. `sagaTransactionId` derives deterministically from `m` (producer.go:67-79, not touched by this diff) plus the purpose string, so the transaction id for a given pending-change id + template id pair is still deterministic. Belt-and-suspenders idempotency (the `moved` guard is primary, the deterministic transaction id is the saga-orchestrator-side backstop) both hold.

Verdict: idempotency contract intact. PASS.

## 3. No consumption on non-APPLIED exits, no interaction with refund branch

- The new branch is gated on `status == StatusApplied` (processor.go:317), mutually exclusive with the refund branch's `status != StatusApplied && m.HasAsset()` (processor.go:305). They cannot both fire for the same `Resolve` call. Confirmed by reading both conditions side by side.
- `TestNonAppliedWorldTransferResolutionLeavesTheCouponAlone` (coupon_consumption_test.go:115-138) is a table test over `StatusRejected` and `StatusCancelled` for `TypeWorldTransfer`, asserting `countOutboxMessagesMatching(..., "consume_world_transfer_coupons") == 0` in both cases. `StatusExpired` is not covered by a dedicated test in this diff, but `StatusExpired` also fails the `status == StatusApplied` predicate the same way REJECTED/CANCELLED do — same code path, same guard, no type-specific branching that would differentiate EXPIRED from the two tested statuses. Not itself a defect, but flagged as a coverage gap below (non-blocking).
- Pre-existing refund tests (`TestAppliedResolutionEmitsNoRefund`, `TestPurchasePathResolutionEmitsNoAssetRefund` per the implementer's report) were not touched by this diff and continue to assert only on `award_asset`, unaffected by the new branch — confirmed the diff does not touch `refund_idempotency_test.go` (not in the diff stat).

Verdict: PASS, with one non-blocking coverage note (StatusExpired not explicitly tested, though the code path is shared and not differentiated by type).

## 4. Cross-service seam (payload/action shape)

- `AddStep(couponConsumptionStepId(m.Type()), sharedsaga.Pending, sharedsaga.DestroyAllAssets, sharedsaga.DestroyAllAssetsPayload{CharacterId: m.CharacterId(), TemplateId: templateId})` (producer.go ~193) — the diff changes only the first argument (step id string), not `sharedsaga.DestroyAllAssets` (the `action`) or the `DestroyAllAssetsPayload` shape/fields. Per the task instruction, the orchestrator keys dispatch off `action` not `stepId`, already confirmed inline — so this is safe. Confirmed the diff hunk (producer.go:190→193ish) touches only the first `AddStep` argument.

Verdict: PASS. No change to what the orchestrator actually keys on.

## 5. Test honesty — new test asserts the NEW behaviour

- `TestApplyConsumesTheWorldTransferCoupon` (coupon_consumption_test.go:80-111) creates a `TypeWorldTransfer` pending change, asserts zero consume emissions at creation, then calls `p.ResolveAndEmit(m.Id(), StatusApplied, "")` directly and asserts exactly one `consume_world_transfer_coupons` / `destroy_all_assets` / `5401000` emission, plus zero `consume_name_change_coupons` emissions. This exercises `Resolve`'s new branch through the real production call path (`ResolveAndEmit`, the same entry point the saga-completion REST callback at `resource.go:281-286` uses) — not `ApplyForCharacter`, which for `TypeWorldTransfer` only starts the saga (`startWorldTransfer`, processor.go:450-459) and leaves the record PENDING. This is the correct call path per `startWorldTransfer`'s own comment ("the saga's terminal event drives Resolve").
- Ran the test directly: `go test ./pending_change/... -run TestApplyConsumesTheWorldTransferCoupon -v` → `PASS` (confirmed via tool output above, 6.15s, includes real postgres testcontainer).
- Before this commit, `consumeCouponsCommandProvider` was only ever invoked from `applyNameChange`'s deleted loop — `Resolve` had no consume branch at all — so this test would have failed against the pre-fix code (zero `consume_world_transfer_coupons` emissions, not one). This is a genuine regression-pinning test, not a vacuous one.
- Full package run: `go test ./pending_change/...` → `ok atlas-character/pending_change` (cached pass after the targeted run above).

Verdict: PASS. The new test asserts the NEW world-transfer behaviour through the real call path, and does not merely duplicate the name-change assertions.

## Non-blocking findings

1. **`StatusExpired` not explicitly tested against the new consume branch** (coupon_consumption_test.go, world-transfer test only covers REJECTED/CANCELLED). The code path is shared with REJECTED/CANCELLED (`status == StatusApplied` is the only gate, no per-status branching beyond that), so this is very low risk, but the bug file's own fix directive said "resolved to REJECTED or CANCELLED emits none" — EXPIRED was simply not named, so this is not a missed requirement, just a coverage gap worth a one-line addition if a follow-up touches this file again.

## Not evaluable

None — the full surface named in the task (regression risk, idempotency, non-APPLIED exits, cross-service seam, test honesty) was directly verifiable by reading `processor.go`/`producer.go`/`coupon_consumption_test.go` and running the targeted test.

## Verdict rationale

All four `## Fix` requirements are implemented exactly as ruled: consumption moved into `Resolve`'s APPLIED branch keyed by `m.Type()`, the ad-hoc loop deleted from `applyNameChange`, the step id made type-appropriate while the transaction-id determinism was left untouched, and a real, previously-failing test added for the world-transfer path. No regression found on the name-change path. One non-blocking coverage note (EXPIRED not explicitly tested) does not block approval.
