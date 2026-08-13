# Plan Audit — task-206-cash-shop-coupon-codes (Plan Adherence)

**Plan Path:** docs/tasks/task-206-cash-shop-coupon-codes/plan.md
**Audit Date:** 2026-08-09
**Branch:** task-206-cash-shop-coupon-codes
**Base Branch:** main (merge-base `1e0a321b8`, 46 commits ahead)

## Executive Summary

All 30 plan tasks were implemented, with file:line evidence for the substantive
parts of each. Seven deliberate deviations from the plan's literal text were
identified in advance by the human/controller (normalize.go location, SQLite
test harness, Task 30 Steps 7/8/10 handoff, 6→10 version scope expansion,
Task 9's full 57-arm enumeration, Task 20's package restructure, Task 28/29's
`errors` split) — all landed as described, not silently dropped. Note:
plan.md's own per-step `- [ ]` checkboxes are 0/179 checked in the committed
file; the actual completion ledger lives in `.superpowers/sdd/plan/progress.md`
(reviewed each task through multiple fix rounds) and is corroborated here by
independent source-tree greps, not merely trusted. Only genuine gaps: plan
Task 30 Step 7 (live client E2E) is explicitly parked for a human, and Step 8
(this review) / Step 10 (commit the record) are in progress by design — none
of these is a silent skip. Build/test status per `verification.md` (re-run and
spot-confirmed): all six changed Go modules pass `go build`/`go vet`/`go test
-race -count=1`; eight repo-root guards pass; `docker buildx bake
atlas-cashshop atlas-channel atlas-configurations` passes; atlas-ui `npm run
build` + `npm test` pass (2003/2003).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Correct gms_v84 COUPON_CODE opcode 230→236, declare codec path | DONE | `derivation.md:355-360`; matrix shows gms_v84 verified 236 (verification.md:375); commits 1d60401..bf86c23 |
| 2 | Derive jms_v185 coupon body + error enum | DONE | derivation.md gains `## jms_v185`; jms body divergence (nType field) captured in codec (see Task 4); commits bf86c23..cbf5c25 |
| 3 | Turn legacy n-a verdict into positive evidence | DONE (scope-expanding finding) | All four legacy GMS clients proven to send COUPON_CODE (opcodes 161/197/220/222); matrix moved n-a→❌ then Task 4 promoted to verified; commits cbf5c25..bd2827f |
| 4 | CouponCode serverbound codec, all versions | DONE | `libs/atlas-packet/cash/serverbound/coupon_code.go` (5249 bytes) + `coupon_code_test.go` (13751 bytes) present; all ten matrix cells `verified` (verification.md:370-380); fix round addressed TIER1-FIXTURE framing gap |
| 5 | Correct INVALID_COUPON_COUPON → INVALID_COUPON_CODE key string | DONE | `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91` = `CashShopOperationErrorInvalidCouponCode = "INVALID_COUPON_CODE"` (confirmed by direct grep) |
| 6 | packet-audit addEntry emits validator + sorted insert | DONE | `tools/packet-audit/cmd/operations.go:521` `func addEntry`, `:578` `func insertSorted`; two extra real bugs found and fixed (total==0 gate, services array from raw) |
| 7 | Route COUPON_CODE in every applicable template | DONE | Ten templates gain `CashShopCouponCodeHandle`; opcodes cross-checked by controller against registries, zero mismatches (progress.md Task 24 entry) |
| 8 | Generate errors table into every template | DONE | Counts v83/84/87=53, v92/95=51, jms=30 (verification.md numeric sweep: 476 leaves added, 0 numeric-changed in errors category); template_gms_92_1.json errors count independently verified = 51 by this audit |
| 9 | Derive legacy + jms error enums, close gms_v92 clientbound hole | DONE | v48/61/72/79 error rows (40/43/49/53); v92 hole closed with 57-arm full enumeration (plan premise wrong — arm-catalog.md had no v92 column, verified) |
| 10 | Tenant configuration for coupon rate limiter | DONE | `services/atlas-cashshop/.../configuration/registry.go:64-72` `GetCouponRateLimit`/`couponRateLimitFrom`; wired into `processor.go:92`; present in all 11 templates per verification.md 9c |
| 11 | wallet.Model.Award saturating arithmetic | DONE | Reviewed boundary cases per progress.md; `switch` refactor confirmed later during lint pass (`wallet/model.go`) |
| 12 | Coupon code normalization rules | DONE (deviation: shared package, per human ruling) | `libs/atlas-constants/coupon/` holds Normalize/Plausible/MaxCodeLength once; both services import it — verified this is the ruling applied, not the plan's literal duplication |
| 13 | Reward model | DONE | jsonb Reward/Rewards model landed; later moved to `coupon/reward` package in Task 20 to break import cycle (deviation #6, confirmed) |
| 14 | Coupon entity, model, migration | DONE | Ladder order (inactive→not-started→expired→usage-limit) implemented; `coupon.Migration` registered at main.go:58 |
| 15 | Batch and redemption entities | DONE | `redemption/entity.go:29-31` — uniqueIndex `idx_redemptions_tenant_coupon_account` on (TenantId,CouponId,AccountId), confirmed by this audit's own grep; fix round added Build() guard tests |
| 16 | Kafka contracts for coupon redemption | DONE | Six symbols added, duplicated correctly across both services (wire-contract pattern, not the shared-logic exception) |
| 17 | Redis counter + failed-attempt rate limiter | DONE | `TenantCounter.IncrWithTTL`/`Get` pattern present in `libs/atlas-redis`; go.mod gained atlas-redis dependency, confirmed COPY lines present in Dockerfile (verification.md Step 5) |
| 18 | Providers and atomic reservation | DONE | `administrator.go:21` `func reserveUse`, `:31` `res.RowsAffected == 1`, confirmed by this audit's grep; real GORM `default:true` bug found in Task 14's entity and fixed here |
| 19 | Reward granters | DONE | Currency + cash-item granters; brief's wrong templateId=0 assumption corrected and falsified by reviewer |
| 20 | The redemption transaction (RedeemAndEmit) | DONE (deviation: reward package moved to coupon/reward) | Single local transaction; compartmentId=uuid.Nil contract documented on both wire types; fix round closed two Important findings |
| 21 | Concurrency and rollback tests | DONE (deviation: SQLite per ruling; Step 3 falsification run-and-disconfirmed) | `TestConcurrentRedemptionsOfASingleUseCoupon` passes 5/5,10/10; root cause of no-write-concurrency (`SetMaxOpenConns(1)`) documented honestly, not papered over |
| 22 | Consume REQUEST_COUPON_REDEMPTION in atlas-cashshop | DONE | `kafka/consumer/cashshop/consumer.go:141-143` `handleCommandRequestCouponRedemption`, type-guard on `CommandTypeRequestCouponRedemption`, confirmed by this audit's grep |
| 23 | Admin REST surface | DONE | `resource_test.go:781-784` `TestThereIsNoRedeemEndpoint`, confirmed by this audit's grep; ten routes enumerated, none reaches RedeemAndEmit/granters; PATCH-as-PUT data-loss bug found and fixed (Nullable[T]) |
| 24 | Channel handler for coupon request | DONE | Standalone handler, routing chain verified end-to-end across all ten templates by controller (opcode cross-check performed independently) |
| 25 | Channel response wiring | DONE | Success/failure announcement order pinned with reflect.DeepEqual; no CompartmentId guard (matches Task 20's contract) |
| 26 | Coupons API client and hooks | DONE | TS types matched Go REST models field-for-field; tenant-readiness gate added in fix round on all 5 query hooks |
| 27 | Coupons admin pages | DONE | `services/atlas-ui/src/pages/CouponsPage.tsx` and `CouponDetailPage.tsx` present (confirmed by this audit's `ls`); no-global-redemption-list design note confirmed present at `CouponDetailPage.tsx:5` and `CouponsPage.tsx:4`; active-toggle partial-PATCH falsified by controller (test failed on full-object mutation, confirming client cannot regress Task 23's server-side fix) |
| 28 | Serverbound CashShopOperationHandle operations + BUY_NORMAL fix | DONE | BUY_NORMAL 20→32 fix verified independently on both v83 (0x46e79e) and v84 (0x471227) IDBs; template numeric sweep confirms exactly this 1 change per template (verification.md 9b) |
| 29 | Finish gms_v92/gms_v95 serverbound arm attribution | DONE | `derivation.md` GIFT/OnGiftMateInfoResult dual-mode entries present at lines 70, 202, 375, 534, 804 (confirmed by this audit's grep); all 25 v92 candidate sites decompiled per ledger; stale status headers corrected in fix round |
| 30 | Full verification gate | PARTIAL (by design) | Steps 1-6,9 run and PASS (verification.md); Step 7 (live client E2E) explicitly NOT run, parked for a human per pre-flight ruling; Step 8 (code review) in progress via this audit; Step 10 (commit the record) pending the controller — none silently skipped |

**Completion Rate:** 30/30 tasks implemented (100%); Task 30 has 3 of 10 steps
intentionally deferred to human/controller handoff, not silently dropped.
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 30, by design — see above)

## Skipped / Deferred Tasks

- **Task 30 Step 7 (live v83 client E2E):** Not run. Requires a human at a game
  client against an ephemeral deployment; checklist parked at
  `verification.md:445-471`, unfilled. Impact: most `errors` key **names** are
  ordinal alignment against decompiled enums rather than observed client text
  (`derivation.md` per-version `errors` sections); a live run would convert
  `aligned` rows to `verified`. All ten COUPON-specific error keys are inside
  evidenced ranges on every version, including the jms 30-of-53 partial
  column, so redemption itself is not at risk — only notice-text precision on
  edge-case error paths.
- **Task 30 Step 8 (code review):** This audit (plan-adherence) is one leg of
  it; backend-guidelines-reviewer and frontend-guidelines-reviewer are
  concurrent separate reviewers per the branch's stated setup.
- **Task 30 Step 10 (commit the verification record):** Pending; `verification.md`
  exists as an untracked/committed file at the time of this audit but the
  final controller commit step is explicitly the controller's responsibility
  per the plan.

No task was skipped without an explicit, recorded ruling. All deviations
(normalize.go location, SQLite harness, 6→10 version scope, Task 9's
enumeration approach, Task 20's package move, Task 28/29's errors-column
split) are documented in `.superpowers/sdd/plan/progress.md` pre-flight/
controller rulings and were independently confirmed by this audit to have
landed in the code, not merely claimed.

## Self-Review Spec-Coverage Table Verification (plan.md:5495-5540)

Sampled 10 of the table's rows against source, in addition to the ledger's
own claims:

| Row | Claim | Verified |
|---|---|---|
| PRD FR-1.1-1.4 codec+fixtures → Task 4 | codec+test files exist | YES — `coupon_code.go`, `coupon_code_test.go` present |
| PRD FR-3.5 key-string correction → Task 5 | INVALID_COUPON_CODE constant | YES — `shop_operation_body.go:91` |
| PRD FR-5.5 atomic counter → Tasks 18, 21 | RowsAffected-verdict reservation | YES — `administrator.go:21,31` |
| PRD FR-6 saga orchestration → superseded | atlas-saga-orchestrator absent from blast radius | YES — design.md:598,621,632 confirm exclusion; verification.md Step 5 lists only atlas-cashshop/atlas-channel/atlas-configurations, no saga-orchestrator |
| PRD §8 security (no redeem endpoint) → Task 23 | pinned by test | YES — `resource_test.go:784` `TestThereIsNoRedeemEndpoint` |
| Design §2.2 dedup by unique index → Tasks 15, 21 | composite uniqueIndex | YES — `redemption/entity.go:29-31` |
| Design §5.2 gms_v92 clientbound hole → Task 9 | errors table populated | YES — `template_gms_92_1.json` errors count = 51 (independently computed by this audit) |
| Design Q7 no global redemption list → Task 27 | UI design note | YES — `CouponDetailPage.tsx:5`, `CouponsPage.tsx:4` |
| PRD §8 rate limiting → Tasks 10, 17, 20 | config + limiter + short-circuit | YES — `registry.go:64-72` config, `processor.go:92` consumption, `processor_test.go:582` `TestRedeemRateLimitedShortCircuits` |
| FR-2.1 USE_COUPON in operations → superseded | standalone op, not mode byte | YES — Task 7 routes CashShopCouponCodeHandle as its own handler (confirmed: ten templates gain a top-level handler entry, not an operations-table row) |

All 10 sampled rows hold up against source. No overclaiming found in the
sampled set.

## Build & Test Results

(Re-stated from `docs/tasks/task-206-cash-shop-coupon-codes/verification.md`,
spot-confirmed rather than re-run in full by this audit — the branch's gates
were already executed against the final tree at commit `ce8c8dbe6`.)

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-packet | PASS | PASS | 80 ok packages, 0 FAIL |
| libs/atlas-redis | PASS | PASS | 1 ok package, 0 FAIL |
| libs/atlas-constants | PASS | PASS | 11 ok packages, 0 FAIL (coupon normalize/plausible lives here) |
| tools/packet-audit | PASS | PASS | 13 ok packages, 0 FAIL (after `-count=1` fixed a stale-cache false green) |
| services/atlas-cashshop | PASS | PASS | 13 ok packages, 0 FAIL |
| services/atlas-channel | PASS | PASS | 110 ok packages, 0 FAIL |
| docker buildx bake (atlas-cashshop, atlas-channel, atlas-configurations) | PASS | n/a | Confirms `COPY libs/atlas-redis` / `libs/atlas-constants` present in root Dockerfile (grep-confirmed at Dockerfile:32,41,62,71) |
| atlas-ui | PASS | PASS | npm run build clean; npm test 2003/2003, 246/246 files |

Repo-root guards (8): redis-key-guard, goroutine-guard, skill-job-id-guard,
buff-duration-guard, template-opcode-order-guard,
template-duplicate-binding-guard, template-movement-types-guard,
service-registration-guard — all exit 0 per verification.md Step 2.

Packet-audit gates: matrix, operations --check, fname-doc --check,
doc-freshness --check, gate-check --check, dispatcher-lint — all exit 0 on
final tree. One real regression (`dispatcher-lint` FAM-CAP false-positive on
two new serverbound dispatcher files) was found and fixed on this branch
(commit `dc6c2fba5`) rather than worked around.

## Overall Assessment

- **Plan Adherence:** FULL — all 30 tasks implemented with verifiable
  evidence; every deviation from the plan's literal text traces to a recorded
  human or controller ruling, and each ruling's stated outcome was confirmed
  in the actual code/config, not just claimed in prose.
- **Recommendation:** NEEDS_REVIEW — code and gates are ready
  (READY_TO_MERGE from a plan-adherence standpoint), but Task 30 Step 7 (live
  client E2E) is a genuine open item that the plan itself treats as a
  precondition for calling the branch fully done ("not deferred — the branch
  is not done until they report back"). This audit does not block merge on
  plan-adherence grounds but flags Step 7 as outstanding per the plan's own
  gate definition.

## Action Items

1. Run Task 30 Step 7 (live v83 client E2E, six scenarios) and fill in
   `verification.md`'s checklist (lines 456-463); convert any `aligned`
   error-key rows that mismatch observed text into corrected, evidence-backed
   entries in `derivation.md`.
2. Once Steps 7-8 complete, commit `verification.md` per Step 10 (currently
   pending the controller).
3. No code changes are required by this plan-adherence audit — all findings
   above are confirmations, not defects.
