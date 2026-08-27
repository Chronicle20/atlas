# Plan Audit — task-240-cash-shop-stub-operations (Tasks 13-24)

**Plan Path:** docs/tasks/task-240-cash-shop-stub-operations/plan.md
**Audit Date:** 2026-08-19
**Branch:** task-240-cash-shop-stub-operations
**Base Branch:** main
**Scope:** Plan Tasks 13 through 24 (including the 24a/24b/24c split) only. Tasks 1-12 are covered by a separate shard.
**Range audited:** merge-base `d9ec287b8` .. `3bc7ebd21`

## Executive Summary

All 12 plan tasks in this range (13-24, with 24 split into 24a/24b/24c) were
implemented with real, non-stub business logic, verified through both the
implementer's tests and an independent per-task code review, and every
blocking finding raised during those reviews was fixed in a follow-up commit
before the branch's final state. The task-24 packet-coverage/seam-trace/
verification closer was executed in full: the seam trace found and fixed a
genuine test-coverage gap (five untested producer→consumer contracts) and a
genuine functional bug (`handleGetRing` returning 500 instead of 404 across
tenants), and a real cross-service atomicity hazard (the equip-slot extension
HTTP write living inside the purchase DB transaction) was diagnosed in 24a and
fixed in 24c by moving it behind the existing outbox pattern with an
idempotency key. The flagless verification gate (`tools/verify.sh`) is
recorded as passing in full (`gate-final5.log`, `VERIFY_EXIT=0`), and this was
independently corroborated by re-running `go build ./... && go test ./...
-count=1` for `atlas-cashshop` in this audit (all packages `ok`, no `FAIL`).
No placeholder comments, stubbed handlers, or unimplemented-status responses
were found in the diff for this range; the two places the plan legitimately
specifies stub/deferred behavior (task 15's `BuyOtherPackage` option/oneADay
fields carried but not acted on; task 24c's fire-and-forget failure path for
`CompleteEquipSlotExtension`) are explicitly scoped as such in the plan and
the implementer's own write-up, not silent omissions.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 13 | GIFT — `atlas-cashshop` side | DONE | `services/atlas-cashshop/atlas.com/cashshop/cashshop/gift.go` (new), `gift_test.go`; commit `a759bf12d`. Reviewed `review-task-13.md`, verdict APPROVED. Transaction order (ledger claim → recipient compartment capacity → sender debit → recipient asset → purchase record → event) matches the brief. |
| 14 | GIFT — `atlas-channel` side | DONE | `cashshop/processor.go`, `cashshop/producer.go`, `kafka/consumer/cashshop/consumer.go` in `atlas-channel`; commit `08358225e`. `review-task-14.md`: "No blocking defects found," all controller corrections verified. |
| 15 | Cash package catalogue in `atlas-data` | DONE (fixed) | `services/atlas-data/atlas.com/data/cashpackage/*` (new); commit `d05dd7ff0`. `review-task-15.md` found one **blocking** defect: the single-item GET route was unreachable because `rest.ParseItemId` read the wrong mux var. Fixed by follow-up commit `863252fcb` ("route packageId through a dedicated parser, not itemId"), which adds `rest.ParsePackageId` and a reproducing integration test (`resource_test.go`, 194 new lines). `89e0eb4c0` additionally fixed an unchecked `resp.Body.Close()` error in the new test. Both fixes land before the task-16 commit that depends on this catalogue. |
| 16 | Package purchase — `atlas-cashshop` side | DONE | `cashshop/package.go`, `package_test.go`; commit `e2f5aaece`. `review-task-16.md` verdict `APPROVED_WITH_FINDINGS`, blocking: 0, non_blocking: 1 (a test-honesty nit — the "capacity checked before charge" test doesn't discriminate ordering because the whole transaction rolls back regardless; atomicity itself is not in question). Non-blocking, correctly left unfixed. |
| 17 | BUY_PACKAGE / BUY_OTHER_PACKAGE — `atlas-channel` side | DONE (fixed) | `cashshop/processor.go`, new codec `CashShopOperationBuyOtherPackage`; commit `96bc6ea27`. `review-task-17.md` found the production code correct but flagged two test-honesty gaps (a dispatch test that doesn't touch the dispatcher, a codec round-trip test that can't detect field-order defects by construction). Fixed by `20746105d` ("make the BUY_OTHER_PACKAGE dispatch and codec tests able to fail"), landed same day, no production-code change needed. |
| 18 | The ring-pairing domain | DONE | `services/atlas-cashshop/atlas.com/cashshop/ring/{entity,model,administrator,provider,resource,rest}.go` + tests; commit `8806b742e`. `review-task-18.md`: "None blocking. None non-blocking beyond what is already self-flagged and resolved." FR-RING-4 atomicity independently reproduced by mutation testing per the review. |
| 19 | Ring purchase transaction and REST surface | DONE | `cashshop/ring.go` (new), `ring/rest.go`, `ring/resource.go`, Kafka wiring; commit `c05b2a750`. `review-task-19.md`: no blocking findings, mutation-tested, `go build`/`go test -run TestPurchaseRing` reproduced clean by the reviewer. |
| 20 | BUY_COUPLE / BUY_FRIENDSHIP — `atlas-channel` side | DONE | `cash_shop_ring.go` handler, consumer/producer wiring; commit `27d1c91e8`. `review-task-20.md`: "None blocking." One non-blocking finding (raw-error passthrough to `giftRejectionReason`, inherited unchanged from the pre-existing `handleGift` pattern) — explicitly picked up and fixed later in task-24a item 3 (see below), beyond what task 20 itself was obligated to fix. |
| 21 | Derive the equip-slot extension facts | DONE | `docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md` (new, 486 lines); commit `9d3026337`. `review-task-21.md` verdict `APPROVED_WITH_FINDINGS`, blocking: 0, non_blocking: 2 (an imprecise wording of the "FILETIME 0" sentinel, and an unflagged disagreement with a prior packet-audit-pinned IDA reading of address `0x468e43` — see task-24 discussion below, this second item is the seed of the v72 reconciliation done in task 24). Both non-blocking, doc-only. |
| 22 | Equip-slot extension domain in `atlas-character` | DONE (fixed) | `services/atlas-character/atlas.com/character/equipslot/*` (new); `Timestamp`→`EquipSlotExtExpire` rename in `libs/atlas-packet`; commit `7b887ee0e`. `review-task-22.md` found one blocking issue: a gofmt/gofumpt-misaligned struct literal that would fail the lint gate. Fixed immediately by `202a574d2` ("gofumpt data_test.go after the equip-slot rename"), same-day follow-up. |
| 23 | ENABLE_EQUIP_SLOT (mode 9/10) end to end | DONE (fixed) | `cashshop/equipslot.go` (new, purchase transaction), `character/rest.go` POST route, `atlas-channel` request-forwarding (`4c5e6bdff`) and `EQUIP_SLOT_INCREASED` announcement (`8cfbc1200`); commits `460875a7a`, `4c5e6bdff`, `8cfbc1200`. `review-task-23.md` found one blocking issue: the byte-level regression test for the R1 wire-vs-canonical hazard (`SlotIndex` 0-vs-canonical) didn't actually exercise the call site with the hazard. Fixed by `cda6eb58c` ("guard the equip-slot canonical/wire boundary at the consumer"), reviewed and APPROVED in `review-task-23-fix1.md` (new test drives the real handler, seeds a non-zero canonical value, asserts on genuinely decoded wire bytes). |
| 24a | Sweep every arm + cross-service seam trace + deferred-finding cleanup | DONE (fixed) | Commit `160191c25`. Sweep confirmed no `l.Infof`-and-return stub arm remains in `cash_shop_operation.go` (`grep -n 'l.Infof'` → zero matches) and `CashShopOperationBuyOtherPackage` is both declared and dispatched. Seam trace found 5 of 6 new producer→consumer event contracts (`GIFT_PURCHASED`, `PACKAGE_PURCHASED` x2, `RING_PURCHASED` x2, `LOCKER_REBATED`) had **zero** pinning tests before this task; fixed by adding 6 new consumer-level tests that decode the real wire packet. Also fixed a real bug found by a new cross-tenant test: `handleGetRing` returned HTTP 500 instead of 404 for another tenant's ring (item 2). `review-task-24a.md` found one blocking issue — 6 of 7 touched files failed the `gofumpt`/lint-gate format check despite the report's claim of a clean pass. Fixed by follow-up `a294a1e2f` (confirmed clean via `gofmt -l` in this audit). |
| 24b | Coverage manifest and matrix promotion | DONE | `docs/tasks/task-240-cash-shop-stub-operations/coverage-manifest.yaml` (new); orphan-marker resolution for `CashShopOperationBuyOtherPackage` x `gms_v95` fixed via `02ab37435` (surgical export correction per `VERIFYING_A_PACKET.md` SS10, IDA-confirmed via `lookup_funcs` against the live IDB, not decompiler guesswork); commit `71fc76306` for the manifest, `02ab37435` for the promotion fix. `review-tail-artifacts.md` (which specifically re-reviewed the hand-spliced export entry) verdict `APPROVED_WITH_FINDINGS`, blocking: 0; the one non-blocking item is a disclosed structural limit (this review environment has no IDA-Pro access to independently re-derive the raw address claim), not a defect. |
| 24c | Move the equip-slot extension write out of the DB transaction | DONE | Diagnosed as a real cross-service atomicity hazard in task-24a item 4 (the `ExtendEquipSlot` HTTP write to `atlas-character` lived inside the same `database.ExecuteTransaction` as the wallet debit — a late rollback after a successful write would leave a free, uncharged extension). Fixed in commit `bcceb3784`: the write is now minted as an outbox command (`EXTEND_EQUIP_SLOT`) inside the transaction and applied by a same-service consumer only after commit, keyed by `transactionId` persisted on `equipslot.Entity` for redelivery-safe dedupe. `review-task-24c.md` verdict: "APPROVED. No blocking findings." Also fixed: verification gate re-run, `context.md` final state (`71fc76306`), final agent-ledger rows (`ac79c40ca`, `3bc7ebd21`), and one corpus-guard count bump (`b0a318962`). |

**Completion Rate:** 12/12 tasks (100%) — counting 24a/24b/24c as the three constituent parts of plan Task 24.
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range are `SKIPPED`, `DEFERRED`, or left `PARTIAL`. Every blocking finding raised by a per-task review in this range was addressed by a same-branch follow-up commit before the plan's tail (task-24) closed:

- Task 15's unreachable GET route → fixed by `863252fcb`.
- Task 17's two test-honesty gaps → fixed by `20746105d`.
- Task 22's gofumpt violation → fixed by `202a574d2`.
- Task 23's non-discriminating regression test → fixed by `cda6eb58c`.
- Task 24a's format-gate failure → fixed by `a294a1e2f`.
- The cross-service equip-slot atomicity hazard surfaced by task-24a's own mandate ("fix any disagreement you find here rather than filing it") → fixed by task 24c (`bcceb3784`).

Non-blocking findings that were legitimately left as-is (correctly, per each review's own reasoning):
- Task 16's test-ordering-doesn't-discriminate nit (atomicity itself is not in question; only the test's rhetorical claim is imprecise).
- Task 21's two doc-wording/citation nits in `derivation-equip-slot.md`.
- Task 24's tail-artifact review's disclosed inability to independently re-derive an IDA address claim without IDA-Pro tooling — flagged transparently rather than silently accepted, consistent with the plan's evidence-and-grounding standard.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-cashshop | PASS | PASS | Re-run in this audit: `go build ./...` clean; `go test ./... -count=1` — all packages `ok` (including `cashshop`, `ring`, `purchaserecord`, `ledger`, `kafka/consumer/cashshop`), no `FAIL`. |
| atlas-channel | PASS | PASS | Confirmed via `gate-final5.log` (the branch's own last flagless `tools/verify.sh` run, `VERIFY_EXIT=0`) — `atlas-channel` and every `cashshop/*` subpackage report `ok`. |
| atlas-character | PASS | PASS | Confirmed via `gate-final5.log` — `atlas-character`, `atlas-character/equipslot`, `pending_change`, `session`, etc. all report `ok`. |
| atlas-data | PASS | PASS | Confirmed via `gate-final5.log` — `atlas-data/cashpackage` and all sibling packages report `ok`. |
| Repo-wide (`tools/verify.sh`, flagless) | PASS | PASS | `gate-final5.log`: "All checks passed." / `VERIFY_EXIT=0`, including all 88 modules' `-race` build/vet/test, the analyzer/scope/producer-seam/service-registration/env-domain/env-bootstrap guards, and the lint & format guard. This is the most recent gate log in the plan's working directory and postdates every fix commit in this range. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; final PR readiness also depends on the 1-12 shard's findings, which are out of this audit's scope)

## Action Items

None required for tasks 13-24. All findings raised by per-task reviews in this range were either fixed on-branch (confirmed by follow-up commits and, where applicable, a follow-up review approving the fix) or correctly left as documented non-blocking observations.
