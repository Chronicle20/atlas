# Plan Audit — task-083-gms-v84-tenant-support

**Plan Path:** docs/tasks/task-083-gms-v84-tenant-support/plan.md
**Audit Date:** 2026-06-09
**Branch:** task-083-gms-v84-tenant-support
**Base (merge-base with main):** 0ba66e8ce
**HEAD:** fb6d44f0f

## Executive Summary

Every task implemented in this session (Phases A, B, C, F, and Task E1) is genuinely present in the
code/docs with cited evidence — none was claimed-but-not-implemented, and no TODO/stub was left behind.
The Phase F build/vet/test gates were reproduced and pass clean on all five changed modules. The one
intentional deviation from the plan's literal text (B4 / the atlas-packet boundary family using
`MajorAtLeast(87)` instead of `MajorAtLeast(84)`) is the documented mid-session off-by-one correction
and is correct, not a miss. Tasks D1, D2, E2, E3 are legitimately deferred (live cluster + v84 WZ
archives + real v84 client). Overall: faithful execution of the in-session scope.

## Task Completion (implemented phases)

| Task | Status | Evidence |
|---|---|---|
| A0 scaffold delta doc | DONE | `v84-packet-delta.md` exists, 1435 lines, all 7 `## 0..6` sections present |
| A1 IDB anchors (§0) | DONE | §0 table: v83/v84/v95/v87/JMS rows, `CWvsContext::OnPacket`/`ProcessPacket`/`SendPacket` addrs per IDB |
| A2 inbound opcode map (§1) | DONE | §1 populated, ~95 rows, SAME/SHIFTED/ADDED/REMOVED classified, OQ-7 low-confidence subsection |
| A3 outbound opcode map (§2) | DONE | §2 populated, ~127 rows incl. ADDED 0x7D–0x7F analysis + completeness-vs-template subsection |
| A4 structure delta + usesPin (§3,§4) | DONE | §3.1 per-flow entries + §3.2 spot-checks; §4 `usesPin=false` with v84 `sub_60D368` vs v83 `0x5F83EE` evidence |
| B1 tenant helpers | DONE | `libs/atlas-tenant/tenant.go:88-106` IsRegion/MajorAtLeast/MajorAtMost/MajorInRange; tests in tenant_test.go |
| B2 audit table (§5) | DONE | §5 has 413 table rows; action tally 365 unchanged / 90 resolved / 2 migrate+correct |
| B3 appliesAutoAP (character) | DONE | `processor.go:55` `IsRegion("GMS") && MajorAtMost(94)`; call site line 1342; test passes |
| B4 usesChooseGender (account) | DONE (documented override) | `processor.go:125` `IsRegion("GMS") && MajorAtLeast(87)` (off-by-one correction, NOT MajorAtLeast(84)); call site 172; test passes |
| B5 atlas-packet predicates | DONE | 50 changed predicate lines; `>83` family migrated to `MajorAtLeast(87)` so v84 == v83; 5 version_bounds_test.go files assert old==new |
| C1 template_gms_84_1.json | DONE | region GMS / major 84 / minor 1 / usesPin false; 93 handlers, 112 writers; 72 writer opcodes SHIFTED, 0 handler opcodes shifted |
| C2 symbol gate + seeder test | DONE | `tools/template-symbol-check.sh` (exec, passes v83 + v84); seeder_test.go TestExtractMetadataGmsV84 + TestGmsV84DistinctFromV83 pass |
| E1 provisioning runbook (§6) | DONE | §6 Steps 0–5 with concrete kubectl/curl/redis-cli commands; Step 4 OQ-6 restart sequence repo-verified |
| F1 build/test/vet gates | DONE | Reproduced clean — see below |

**Completion rate (in-session scope):** 14/14 tasks DONE.
**Claimed-but-not-implemented:** 0.
**TODOs/stubs introduced:** 0 (two pre-existing unrelated TODOs in character/processor.go are not from this task).

## Deferred (correctly, not silently skipped)

- **D1 / D2** (v84 WZ ingest + atlas-data serving) — require the v84 WZ archive files and a live cluster.
- **E2** (live v84 client playthrough) — requires a running stack and a real v84 game client.
- **E3** (v83 regression) — requires the deployed stack.

These are blocked on operator-supplied infra/client and are documented as such in the runbook (§6). Not implementation gaps.

## Build & Test Results (reproduced this audit)

| Module | Build | Vet | Test (-race) | Notes |
|---|---|---|---|---|
| libs/atlas-tenant | PASS | PASS | PASS | helper tests pass |
| libs/atlas-packet | PASS | PASS | PASS | 58 pkgs ok, 0 fail; version_bounds tests pass |
| services/atlas-account | PASS | PASS | PASS | TestUsesChooseGender pass |
| services/atlas-character | PASS | PASS | PASS | 4 pkgs ok; TestAppliesAutoAP pass |
| services/atlas-configurations | PASS | PASS | PASS | seeder v84 idempotency/distinct tests pass |

Gates: `tools/template-symbol-check.sh` → OK on both v83 and v84 templates.
(Not reproduced: `docker buildx bake` and `redis-key-guard.sh` — bake requires the build daemon; redis-guard
has no new redis usage in this diff. F1 step list documents both as run.)

## Deviations from plan (all justified)

1. **B4 / atlas-packet `>83` boundaries → `MajorAtLeast(87)`** (plan literal said `MajorAtLeast(84)`).
   This is the explicit mid-session decision: Phase A found the `>83` checks are a systematic off-by-one;
   v84 must match v83, so the boundary moves to 87. Consistently applied across atlas-packet and recorded
   in §5 and the commit messages. CORRECT.
2. **Plan checkboxes left at 0/92 checked.** The implementer tracked progress via commit messages, not
   by ticking boxes. Cosmetic only — every task has commit-level evidence. Minor hygiene note.

## Overall Assessment

- **Plan Adherence (in-session scope):** FULL
- **Recommendation:** READY_TO_MERGE for the documented A/B/C/F+E1 scope; remaining D/E tasks are the
  operator's live-infra gate, not a code deficiency.

## Action Items

1. (Optional hygiene) Tick the plan.md checkboxes for the completed A/B/C/F/E1 steps so plan state matches reality.
2. (Out of session) Execute D1/D2/E2/E3 once the v84 WZ archives and a v84 client are available, per the §6 runbook.
