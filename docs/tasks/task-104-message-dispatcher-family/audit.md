# Plan Audit — task-104-message-dispatcher-family

> NOTE: This file contains modular reviewer sections. The plan-adherence section below
> was written by the plan-adherence-reviewer. Do not clobber other reviewers' sections.

## Plan Adherence Review

**Plan Path:** docs/tasks/task-104-message-dispatcher-family/plan.md
**Audit Date:** 2026-06-19
**Branch:** task-104-message-dispatcher-family
**Base Branch:** main
**Review Range:** `2cac0557b` .. `32351ad4d`

### Executive Summary

All 14 plan tasks (0–13) are implemented and verified. The four packet-audit gates
(`dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check`) all exit 0,
and `go build`/`go vet`/`go test -race` pass clean in all four changed modules
(`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel/atlas.com/channel`,
`services/atlas-configurations/atlas.com/configurations`). The `SHOW_STATUS_INFO` row in
`docs/packets/audits/STATUS.md:61` is ✅ on all five versions. The two written-plan deviations
(jms mode-0xF implemented as a new `StatusMessageJMSCounterNotice` arm; jms `opcodes` line + Task 8
evidence/reports produced) are user-directed / producible-prerequisite work, not unrequested scope
creep, and were executed cleanly. No TODO/stub/501 and no `0x0` placeholder addresses landed.
**Recommendation: READY_TO_MERGE.**

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Pre-flight (worktree/baseline gate snapshot) | DONE | Read-only; outcomes consistent with later state — no SP call site exists, gates clean. |
| 1 | IDA enumeration & grounding (5 versions, jms 0xF) | DONE | `context.md:206-335` "## Enumeration results": per-version switch addrs, per-arm delegate table, inner fan-out discriminators, v84 per-arm semantic confirmation, jms 0xF resolution. Marker addresses match the table (spot-checked IncreaseMeso, SP). Commit `b287d62d2`. |
| 2 | Correct per-version mode table in dispatcher yaml | DONE | `docs/packets/dispatchers/character_status_message.yaml:31-46`: v83 omits `INCREASE_SKILL_POINT`, fame=4..skill_expire=13; v84+ SP=4, fame=5..skill_expire=14; `JMS_COUNTER_NOTICE: { jms_v185: 15 }`. Commit `8c7442268`. |
| 3 | Rewire run.go — `#`-entries, no bare root | DONE | `tools/packet-audit/cmd/run.go`: no bare `case "CWvsContext::OnMessage":` (grep exit 1); 25 `#`-entry cases (24 GMS arms + `#JMSCounterNotice` at line 487). Commits `502fa7b11`, `2d7ba2cd4`. |
| 4 | Export decomposition (GMS delegates + jms splice) | DONE | All 5 exports spliced with real delegate addresses; v83 omits `#IncreaseSkillPoint` (grep 0), v84/87/95/jms include it; jms has `OnMessage#JMSCounterNotice` @`0xb0931c` (`gms_jms_185.json:10812`). Zero `0x0` placeholder addresses added by the task. Commits `cf9845a1f`, `2d7ba2cd4`. |
| 5 | Per-arm fixtures + verify markers + struct citations | DONE | `status_message_test.go`: 120 markers total — v83=23 (no SP), v84/87/95=24, jms=25; SP markers omit v83; `#JMSCounterNotice` jms-only. `status_message.go` gains the jms struct with per-version citation. Commits `c1b4273ad`, `2d7ba2cd4`. |
| 6 | Reconcile seed templates from yaml | DONE | `template_gms_83_1.json:1457-1469` drops `INCREASE_SKILL_POINT` and shifts fame..skill_expire down one (writer opCode `0x27` intact). `operations --check` exits 0. Commit `f8582ec8a`. |
| 7 | dispatcher-lint clean, no baseline entry | DONE | `dispatcher-lint: clean` (exit 0); `grep OnMessage docs/packets/dispatcher-lint-baseline.yaml` → no match (exit 1). |
| 8 | Evidence records | DONE (deviation #2b) | 120 evidence YAMLs added under `docs/packets/evidence/{version}/character.clientbound.StatusMessage*.yaml`, count matching markers (23/24/24/24/25). Commit `fc5d76f51`. Plan framed this as conditional; it was required for promotion and was produced. |
| 9 | Regenerate coverage matrix | DONE | `STATUS.md:61` SHOW_STATUS_INFO ✅ on all 5 versions (v83/v84/v87 0x027, v95 0x026, jms 0x025); `matrix --check` exits 0. Commit `1692d4d82`. |
| 10 | Call-site & validator verification | DONE | No literal-mode `NewStatusMessage` construction in `services/atlas-channel/atlas.com/channel/` (grep exit 1); no SP emission anywhere (grep exit 1); gms_83 writer opcode intact; channel builds clean. |
| 11 | Full build/vet/test gates | DONE | build/vet/test -race exit 0 in all four modules; all four packet-audit gates exit 0. (Docker bake / redis-key-guard not re-run in this audit — see Notes.) |
| 12 | Live-config runbook (authored, not executed) | DONE | `runbook-live-config.md` (6.5KB): v83 correction map, jms writer addition, v84/87/95 verify-only, channel restart, post-restart verification, rollback, execution-gating. Commit `32351ad4d`. |
| 13 | Code review before PR | IN PROGRESS | This audit is part of Task 13. `audit.md` created here. |

**Completion Rate:** 14/14 tasks substantively complete (Task 13 is the review step in flight).
**Skipped without approval:** 0
**Partial implementations:** 0

### Intentional Deviations (judged adherent)

1. **jms mode 0xF implemented as a new arm `StatusMessageJMSCounterNotice`** (key `JMS_COUNTER_NOTICE`,
   jms mode 15, delegate `0xB0931C`, wire `[mode byte][int32]`). The plan's D7 gate said stop-and-ask;
   the user chose "implement as a new arm." Done cleanly: discrete struct + body func via
   `WithResolvedCode("operations", "JMS_COUNTER_NOTICE", ...)` (no hardcoded mode byte), jms-only
   template/yaml/export/marker/report/evidence. Round-trip test passes. This raises the family to 25
   arms / 16 jms cases. Grounded in `context.md` §5 (the encrypted message text is correctly left out
   of the structural name).
2a. **jms `opcodes: { jms_v185: "0x25" }` added to the dispatcher yaml** because the jms template had
    no CharacterStatusMessage writer at all (would resolve every mode to 99 — the operations-table-
    missing bug family). Mirrors the `character_interaction.yaml` precedent; registry-confirmed
    opcode 37=0x25. Required to actually meet acceptance for jms.
2b. **Task 8 evidence + per-arm reports produced** (120 each), matching the existing InventoryFull /
    FieldEffect exemplar precedent that carries evidence+reports. Required for cell promotion.

### Findings

- No `// TODO`, stub handler, 501, or `panic(...)` introduced (diff scan clean; the only TODO/0x0
  string matches are negative assertions in plan/design/runbook prose).
- No `0x0` placeholder delegate addresses landed in any of the five exports (per-version count of
  task-104-added `0x0` lines = 0). Every OnMessage delegate uses a real IDA address that matches the
  grounded `context.md` table.
- Marker IDA addresses spot-checked against `context.md` §2 (IncreaseMeso v83 `0xa221f3` / v95
  `0x9fe910`; SP v84 `0xa6cefa` / v95 `0x9f8570`) — exact match.
- D5 safeguard holds: v83 `INCREASE_SKILL_POINT` is absent from the operations table AND no consumer
  emits SP, so the v83 SP arm is genuinely ⬜-by-absence, not a fabricated byte.

### Build & Test Results

| Module | Build | Vet | Test -race | Notes |
|--------|-------|-----|-----------|-------|
| libs/atlas-packet | PASS | PASS | PASS | exit 0 |
| tools/packet-audit | PASS | PASS | PASS | exit 0 |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | exit 0 |
| services/atlas-configurations/atlas.com/configurations | PASS | PASS | PASS | exit 0 |

**Packet-audit gates:** dispatcher-lint=0, matrix --check=0, fname-doc --check=0, operations --check=0.
(`operations --check` emits an informational absent-writer note for `NoteOperation` on jms — unrelated
to this task, exits 0.)

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

### Action Items

None blocking. Optional, per CLAUDE.md "Build & Verification" (not re-run during this read-only audit):
1. Run `docker buildx bake atlas-channel` from the worktree root (no go.mod was touched, so expected
   clean — but it is the mandated final check).
2. Run `GOWORK=off tools/redis-key-guard.sh` (no Redis introduced — expected clean).
