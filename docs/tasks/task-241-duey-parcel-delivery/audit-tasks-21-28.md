# Plan Audit — task-241-duey-parcel-delivery (Tasks 21-28)

**Plan Path:** docs/tasks/task-241-duey-parcel-delivery/plan.md
**Audit Date:** 2026-08-19
**Branch:** task-241-duey-parcel-delivery
**Base Branch:** main
**Scope:** Plan tasks 21-28 only (sibling shards cover 1-20)

## Executive Summary

All eight plan tasks in this range (21-28) are implemented, gated, and reviewed. Task 28 (the packet coverage-matrix campaign) is the largest unit audited: `docs/packets/audits/status.json` confirms all 16 op×version cells (`PARCEL` and `DUEY_ACTION` across gms_v72/79/83/84/87/92/95 and jms_v185) read `verified`, all four required `go run ./tools/packet-audit …` checks exit 0 live, and `PARCEL`/`DUEY_ACTION` are absent from `docs/packets/dispatcher-lint-baseline.yaml`. Two real defects were caught and fixed mid-campaign (RULING 24, a v92+ wire-format gap; RULING 25/27, a template-fname mismatch introduced by this branch) — both are documented and resolved on the branch, not swept under a "pre-existing" label. All affected modules build and test clean (`atlas-channel`, `atlas-parcel`, `atlas-character`, `atlas-constants`, `atlas-packet`). No task in this range was skipped or left partial.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 21 | atlas-channel SHOW_PARCEL consumer | DONE | `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go:77` (`handleShowParcelCommand`); registered in `main.go:267,601`. Commits `a6acc9f3..3e9d27635` + gate fix `1d09cda4c`. Review APPROVED_WITH_FINDINGS, 0 blocking. |
| 22 | Quick Delivery Ticket classification-533 branch | DONE | `libs/atlas-constants/item/duey.go:19` (`QuickDeliveryTicketId = uint32(5330000)`, placed in a new file rather than `constants.go` — equivalent outcome); `services/atlas-channel/.../character_cash_item_use_duey.go:46,74` (`dueyCouponEnabled`, `handleDueyCouponUse`); dispatch branch at `character_cash_item_use.go:799-800`. Commits `98173c1c4..bc09d238b`. Review APPROVED_WITH_FINDINGS, fix round landed. |
| 23 | atlas-parcel expiry / return-to-sender sweep | DONE | `services/atlas-parcel/atlas.com/parcel/parcel/task.go:21,56` (`DefaultExpiryInterval`, `NewExpiryTask`); RISK-4 resolved and documented at `docs/tasks/task-241-duey-parcel-delivery/context.md:238` ("RISK-4 resolution"). Commits `b4f2ea0c7`, `d9941beb4`, `b3981f49a`. Review APPROVED_WITH_FINDINGS, 0 blocking. |
| 24 | atlas-parcel notification sweep | DONE | `services/atlas-parcel/atlas.com/parcel/parcel/notification_task.go:27,60` (`DefaultNotificationInterval`, `NewNotificationTask`); `kafka/message/parcel/kafka.go:7,11` (`EnvStatusEventTopic`, `StatusEventParcelArrived`). Commit `a2093bc27`. Gate initially FAILED (unused `withBatch`), review CHANGES_REQUIRED (stale line citations); both closed in fix commit `eac953e3c`. |
| 25 | atlas-channel parcel arrival alarm | DONE | `services/atlas-channel/.../kafka/consumer/parcel/consumer.go:207-209` (`handleParcelArrivedEvent`, guards `e.Type`). Commit `b8efda7b1`. Review APPROVED, 0 findings — envelope/body field names hand-diffed against Task 24's producer. |
| 26 | atlas-character world-transfer gate 12 `parcel_pending` | DONE | `pending_change/processor_eligibility.go:53,211,242,405` (`gateDeps.parcelPending`, gate appended in both `evaluate` functions, `checkParcelPending`); `requests.go:208` (`parcelPending` REST client). Test coverage: `processor_eligibility_test.go:479` (`TestEligibilityGate12ParcelPending`), `requests_test.go:101` (`TestParcelPending`). Commit `65d9c2704`. Review APPROVED_WITH_FINDINGS, 0 blocking (1 non-blocking folded into Task 28's commit per ledger). |
| 27 | Surface `parcel_pending` in every seed template | DONE | All nine `template_gms_{48,61,72,79,83,84,87,92,95}_1.json` files contain exactly one `parcel_pending` occurrence each (verified live via `grep -c`), each read from its own file's `mts_listings_open` value per the plan's anti-copy instruction. Commit `283a618da`. Review APPROVED, 0 findings. |
| 28 | Promote the coverage matrix, close out packet record | DONE | `docs/tasks/task-241-duey-parcel-delivery/coverage-manifest.yaml` present and complete. `docs/packets/audits/status.json` confirms all 16 cells `verified` (both `PARCEL` and `DUEY_ACTION` rows, all 8 versions, live-parsed). All 5 `go run ./tools/packet-audit …` commands exit 0 live (matrix --check, dispatcher-lint, fname-doc --check, operations --check). `PARCEL` absent from `dispatcher-lint-baseline.yaml` (grep exit 1 = no match). 8 batch commits + retro-fit + a routing pre-fix (Task 28a) + two mid-campaign defect fixes, each gated and reviewed; final commit `d2a701e94` records batch 8's clean review. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. One deliberate, user-ruled deferral surfaced *during* Task 28 and is explicitly out of scope for this branch, not a silently dropped plan task:

- **`packet-audit seed-fname --write` writer-churn defect** (reorders JSON keys across all 10 seed templates on any write, unrelated to fname correctness). Diagnosed as pre-existing at the merge base (`d9ec287b8`) via a controlled reproduction, and the user explicitly ruled it deferred to a follow-up task (RULING 26 in `progress.md`) after confirming it does not affect Task 28's accuracy and cost exactly one already-absorbed incident (RULING 25's hand-applied fix). This is not a plan-241 task; it is correctly out of scope here.

Two real defects were found and *fixed on this branch* (not deferred), which is the behavior this audit is checking for:

1. **RULING 24** — the v92+ equip potential/socket trailer (12 bytes) was ungated in `model/asset.go`, so PARCEL cells for v92/v95 would have promoted `verified` against wire-incorrect fixtures. Fixed in commit `a8adafb12` (`MajorAtLeast(92)` gate added, fixtures re-derived from the IDB).
2. **RULING 25/27** — `TestSeedFName_RealTemplatesInsertionCoverage` failed on 7 of 8 versions because a branch-introduced commit (`6bb465f86`, DUEY_ACTION serverbound codecs) diverged from the registry's canonical fname. Root-caused to the correct side (registry canonical, templates derived) and fixed in commit `681ebfdcc` (one line per affected template, controller-verified against each version's registry primary).

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| services/atlas-channel/atlas.com/channel | PASS | PASS | `go test ./kafka/consumer/parcel/... ./parcel/... ./socket/handler/...` — all ok |
| services/atlas-parcel/atlas.com/parcel | PASS | PASS | `go test ./...` — ok (no-test packages reported `[no test files]`, expected) |
| services/atlas-character/atlas.com/character | PASS | PASS | `go test ./pending_change/... -count=1` — ok (245s runtime, exit 0) |
| libs/atlas-constants | PASS | PASS | `go test ./...` — all ok |
| libs/atlas-packet (parcel/*) | PASS | PASS | `go test ./parcel/...` — clientbound and serverbound both ok |
| tools/packet-audit (matrix, dispatcher-lint, fname-doc, operations) | N/A | PASS | All 4 commands exit 0 live against current HEAD |

Repo-wide flagless `tools/verify.sh` was running separately per the dispatch instructions and was not re-launched here.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the sibling shards' verdicts for tasks 1-20, and pending the separately-running flagless `tools/verify.sh`)

## Action Items

None required for tasks 21-28. For completeness, carry forward into the PR description (already tracked in `progress.md`, not new findings):

1. `duey_action.yaml` has no call-site roster field at all (the `CTabQuickSend::SendQuickDelivery` omission noted across multiple batches is a missing field, not a missing entry) — worth a follow-up, not blocking.
2. `packet-audit export -ida-database <session>` is broken on GUI-adopted IDA sessions; every batch fell back to a surgical raw-text export splice. Worth a tooling follow-up.
3. `CPet::OnNameChanged` is duplicated (annotated + bare-stub entries) in four committed IDA exports (gms_v48/61/72/79); JSON parsers keep the stub. Pre-existing, deliberately untouched to avoid moving unrelated PET-family matrix cells.
4. `seed-fname --write` reorders JSON keys across all 10 seed templates regardless of whether any fname changed — user-ruled deferred to a follow-up task (RULING 26).
