# Plan Audit — task-241-duey-parcel-delivery (Tasks 1-10)

**Plan Path:** docs/tasks/task-241-duey-parcel-delivery/plan.md
**Audit Date:** 2026-08-19
**Branch:** task-241-duey-parcel-delivery
**Base Branch:** main
**Scope:** Plan Tasks 1-10 only (atlas-parcel skeleton/model/processor/REST, service
registration, and the PARCEL/DUEY_ACTION packet record + codecs). Tasks 11-28 are
covered by sibling shards and are explicitly out of scope here.

## Executive Summary

All 10 tasks in this range are implemented, tested, reviewed, and gated. Every
task has verifiable code on disk plus a corresponding execution-ledger entry
in `.superpowers/sdd/plan/progress.md` with commit hashes that check out
against `git log`. Two review-cycle CHANGES_REQUIRED rounds (Task 3, Task 4)
and one blocking packet-audit gate (Task 7) were caught and fixed before the
range closed; the fix commits are present and independently re-reviewed
APPROVED. `go build ./... && go test ./... -count=1` is green on
`services/atlas-parcel/atlas.com/parcel`, `libs/atlas-packet`, and
`tools/packet-audit` (the third because Tasks 7-10 append to its `run.go`).
The plan.md checkboxes themselves are all still unchecked (`- [ ]`) for this
range — see note under Task Completion — but this is consistent with how the
rest of the plan tracks completion on this branch (via the ledger, not the
markdown checkboxes) and is not evidence the work was skipped: concrete code,
tests, and reviewed commits exist for every task.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-parcel module skeleton, entity and migration | DONE | `services/atlas-parcel/atlas.com/parcel/go.mod`, `main.go`, `rest/handler.go`, `parcel/entity.go`, `parcel/asset_data.go`, `parcel/entity_test.go` all present. Commit `c16e1e256`. Review APPROVED_WITH_FINDINGS (0 blocking, 2 non-blocking/deferred). `progress.md:63-81`. |
| 2 | atlas-parcel model, builder, provider, administrator | DONE | `parcel/model.go`, `parcel/builder.go`, `parcel/builder_test.go`, `parcel/provider.go`, `parcel/administrator.go` present. Commit `b179059b5`, 10/10 subtests. Review APPROVED_WITH_FINDINGS (0 blocking, 1 non-blocking/deferred — `ByRecipient`'s free-form `status` param). `progress.md:83-106`. |
| 3 | atlas-parcel processor — state machine and injectable clock | DONE | `parcel/processor.go`, `parcel/processor_custody.go`, `parcel/processor_test.go` present. Initial commit `91dc1cae9` (22/22 subtests). Review CHANGES_REQUIRED (2 blocking): `processor.go:97` hardcoded `world.Id(0)` in `HasInFlight`, and `GetById` not mapping to `ErrNotFound`. Fixed in commit `7449ce973` — `HasInFlight` now calls `ReceivableByRecipientAnyWorld` (`provider.go:88`), `GetById` maps `gorm.ErrRecordNotFound`→`ErrNotFound` (`processor.go:67-73`), and `UpdateStatusIfPending` is a genuine SQL CAS (`processor.go:183-191`). Scoped re-review APPROVED, 0 blocking. `progress.md:108-127, 135-152, 181-223`. |
| 4 | atlas-parcel REST surface | DONE | `rest/resource.go`, `rest/handler.go` present (module builds/tests green — see Build & Test Results). Initial commit `cae523317`. Review CHANGES_REQUIRED (1 blocking): `filter[worldId]` silently defaulted to world 0 instead of being required, returning a clean 200 in the wrong world. Fixed in commit `6879aa7e0` (sequenced behind Task 3's fix per Ruling 4) — `filter[worldId]` now required (400 if absent), 404 path swapped to `parcel.ErrNotFound`. Scoped re-review APPROVED, 0 blocking, all 8 `TestParcelResource` subtests green. `progress.md:128-134, 154-208`. |
| 5 | Register atlas-parcel across build, CI, k8s and databases | DONE | Confirmed via `git diff main...HEAD --stat`: `deploy/k8s/base/atlas-parcel.yaml`, `atlas-ingress.yaml`, `env-configmap.yaml` (+`COMMAND_TOPIC_PARCEL`, `COMMAND_TOPIC_PARCEL_CUSTODY`, `EVENT_TOPIC_PARCEL_STATUS`, `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`), `kustomization.yaml`, `routes.conf.template.generated`, `deploy/shared/routes.conf`, overlays `main`/`pr`/`pr-sparse` (all three, incl. the `consumer-group-env.yaml` mirror), `.github/config/services.json`, `docker-bake.hcl`, and 8 per-version `npc-9010009.json` seed templates. Commit `37a0fb601` + lint fix `86c949d99`. A real gap (pr-sparse mirror drift, missed by the initial review) was caught by the gate and fixed in `2604430f3`, independently verified via `tools/pr-sparse-mirror-guard.sh`. Review APPROVED, 0 blocking. `progress.md:269-352`. |
| 6 | Packet record — dispatcher yamls, registry entries, template tables | DONE | `docs/packets/dispatchers/parcel.yaml`, `docs/packets/dispatchers/duey_action.yaml` present; per-version registry entries confirmed (e.g. `docs/packets/registry/gms_v79.yaml:2921` cites the live-decompiled v79 DUEY_ACTION opcode). Two commits `a590457c6` (partial, controller-diagnosed misrouted-agent-type blocker, not a code defect) + `91a6e82aa` (continuation, complete). Independently re-derived by an IDA-capable audit (`task-6-ida-audit.md`) which confirmed every checked byte at the cited address, including v79 DUEY_ACTION=0x3F. Ruling 5 (jms_v185 14-key notice shortfall genuinely underdetermined, left honestly unset) is a documented, evidenced scope gap, not a silent omission. `progress.md:354-497`. |
| 7 | `PARCEL` struct and the OPEN / OPEN_QUICK arms | DONE | `libs/atlas-packet/parcel/parcel.go`, `parcel/clientbound/parcel.go`, `parcel_body.go`, `parcel_test.go` present; `tools/packet-audit/cmd/run.go` carries the 2 appended candidate cases (`run.go:1345-1352`). Commit `797baebef`. Review CHANGES_REQUIRED (1 blocking): `matrix --check` stale (self-hash drift from the `run.go` edit — not regenerated after the final edit). Fixed in commit `29ec83a32`, independently re-verified by the controller (`matrix --check` exit 0, generated files changed exactly the `Tool:` hash line) and by a second `packet-verifier` review (APPROVED, 0 blocking) that re-derived the two comment-provenance corrections (`sub_6F0DC3` unnamed, `+21 sentAt` v83 corroboration) from the live IDB. `progress.md:497-624, 686-737`. |
| 8 | `PARCEL` bodyless result arms | DONE | Fifteen bodyless result structs appended in `clientbound/parcel.go`/`parcel_body.go`, `run.go` gained 15 appended cases (verified zero-deleted-lines by the reviewer), new `parcel_result_test.go`. Commit `f1e7d0229`. Review APPROVED_WITH_FINDINGS (0 blocking, 1 non-blocking — an untested but harmless additive `OpenQuick.Decode`, folded into Task 9 as test coverage rather than a fix round). Ruling 5 (no jms values invented for the 14 unset keys) independently confirmed by diff — `parcel.yaml` diff empty for this commit. `progress.md:756-800`. |
| 9 | `PARCEL` arms that carry a body — removed, arrived, alarms | DONE | Four body-carrying arms (`PARCEL_REMOVED`, `PARCEL_ARRIVED`, `ALARM_NAMED`, `ALARM_GENERIC`) landed, `TestParcelOpenQuickDecode` added closing Task 8's coverage gap. Commit `707e97efc`. Review APPROVED — 0 blocking, 0 non-blocking, 0 not_evaluable, "cleanest review on the branch so far." `ParcelRemovedKindClaimed=0` flagged by the implementer as a documented CHOSEN (not derived) sentinel, independently re-checked by the reviewer against the live IDB (only one pinning literal, `kind==3`) and accepted. `progress.md:823-865`. |
| 10 | `DUEY_ACTION` serverbound codecs | DONE | `libs/atlas-packet/parcel/serverbound/action.go`, `action_send.go`, `action_parcel_id.go`, `action_test.go` plus per-version test fixtures (`v72_test.go` … `v185_test.go`) present. TWO commits: `6bb465f86` (codecs, 4 appended `run.go` cases, 0 deleted lines) + `586ee9ebb` (matrix regeneration, a necessarily separate commit — `toolTreeSHA()` hashes the committed tree, so bundling regeneration with the edit that changes the tool tree is structurally impossible, confirmed from `tools/packet-audit/cmd/matrix.go:491-492`). Review APPROVED — 0 blocking, both disclosed deviations (no `WithResolvedCode` layer for a serverbound decode-only family; two-commit split) independently verified against `DISPATCHER_FAMILY.md`, `storage/serverbound/operation.go`, and the tool source. `progress.md:900-980`. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (two tasks — 3 and 4 — went through one CHANGES_REQUIRED fix round each, both independently re-reviewed APPROVED before being marked complete; not left partial)

## Skipped / Deferred Tasks

None. No task in this range was skipped, silently deferred, or left with an
open blocking finding. All CHANGES_REQUIRED findings (Task 3 x2, Task 4 x1,
Task 7 x1) were fixed and re-reviewed clean within this range's commit
history before the range's last-gated commit (`707e97efc`, gate run 8,
PASS).

Two items worth carrying forward for context, both explicitly non-blocking
and already routed to their correct owning task rather than dropped:
- **Task 3 minor (deferred to final whole-branch review):** `atlas-parcel.Create`
  trusts channel-side validation for the 10-parcel mailbox cap; no
  server-side backstop exists in the custody service, so two concurrent
  sends to a full mailbox could theoretically both pass the channel-side
  count check. Ruled correctly out of Task 3/16/17 scope by the plan's own
  design (validation lives channel-side); flagged for the whole-branch
  review, not silently dropped.
- **Task 6 (Ruling 5, verified twice):** `parcel.yaml`'s `jms_v185` column is
  populated for only 7 of 21 keys; the other 14 are genuinely underdetermined
  from the JMS binary (fewer notice slots exist in JMS than GMS) and are left
  honestly unset rather than guessed. This is a documented scope gap, not a
  silent omission — Tasks 8/9 were explicitly told not to invent jms values
  for those keys and both were independently confirmed compliant.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| services/atlas-parcel/atlas.com/parcel | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages pass or report `[no test files]` (e.g. `atlas-parcel/parcel`, `atlas-parcel/kafka/consumer/custody`). |
| libs/atlas-packet | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages pass, including `parcel/clientbound` and `parcel/serverbound`. |
| tools/packet-audit | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all green (touched by Tasks 7-10's `run.go` edits). |
| packet-audit `matrix --check` | PASS (exit 0) | — | Confirmed fresh at current HEAD, matching the Task 10 ledger claim that Ruling 6's two-commit regeneration closed the branch's oldest open item. |
| packet-audit `dispatcher-lint` | PASS (`dispatcher-lint: clean`) | — | Confirms Task 7's FAM-CAP fix held through Tasks 8-10. |

Repo-wide `tools/verify.sh` was not re-run by this shard (per instructions —
it runs separately at the controller level). Gate runs 3-9 in the ledger
(ranges `b8b0bea22..91a6e82aa`, `..f1e7d0229`, `..707e97efc`) all reconcile to
PASS for the code in this range, net of a repeatedly-recurring but
independently-diagnosed environmental `golangci-lint` lock collision on the
shared machine (unrelated to this branch's code, and confirmed to clear on
its own by gate run 6).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch
  recommendation depends on sibling shards covering Tasks 11-28)

## Action Items

None required for Tasks 1-10. No blocking findings remain open in this
range.
