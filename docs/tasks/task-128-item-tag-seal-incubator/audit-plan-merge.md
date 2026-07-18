# Plan Audit (Post-Merge + Version-Support) — task-128-item-tag-seal-incubator

**Plan Path:** docs/tasks/task-128-item-tag-seal-incubator/plan.md
**Audit Date:** 2026-07-15
**Branch:** task-128-item-tag-seal-incubator
**Base Branch:** main
**Scope:** (1) post-merge integrity of the original feature; (2) the new gms_48/61/72/79 version-support work (commit `b2725b065`, merge `1b4de9f58`).

## Executive Summary

Every original task-128 deliverable survived the ~79-commit merge with `main` and the three follow-up commits — the saga types, compensators, IncubatorResult writer, the three channel dispatch arms, the incubator-rewards tenant config resource, and the seed-template rows are all present and wired. Nothing was dropped or half-implemented at the code level. The new version-support work (gms_61/72/79 wired + verified, gms_48 deliberately excluded) is real and complete: templates, byte fixtures, markers, evidence, audit reports, and the status.json matrix promotions are all in place. One genuine post-merge defect exists: **duplicate `CharacterCashItemUseHandle` handler rows** in the gms_87/95/jms seed templates (main added the same handler at the same opcode that Task 18 already added). It is runtime-harmless but is unresolved merge drift that should be deduped. `go test ./incubator/... ./cash/serverbound/...` is green.

## Part 1 — Post-Merge Integrity of the Original Feature

| Deliverable | Status | Evidence |
|---|---|---|
| Saga types `ItemTagUse` / `SealingLockUse` / `IncubatorUse` | DONE | libs/atlas-saga/model.go:28-30 |
| `compensateCashItemUse` | DONE | services/atlas-saga-orchestrator/.../saga/compensator.go:1199 (iface :51) |
| `DispatchCashItemUseRollbacks` | DONE | compensator.go:1256 (iface :83) |
| Compensator saga-type dispatch to cash-item-use | DONE | compensator.go:222-225 (`ItemTagUse \|\| SealingLockUse \|\| IncubatorUse` → `compensateCashItemUse`) |
| `IncubatorResult` clientbound writer | DONE | libs/atlas-packet/incubator/clientbound/result.go:11,40 |
| Item-tag dispatch arm | DONE | services/atlas-channel/.../socket/handler/character_cash_item_use.go:122 (`SagaType: saga.ItemTagUse` :148) |
| Sealing-lock dispatch arm | DONE | character_cash_item_use.go:184 (`SagaType: saga.SealingLockUse` :220) |
| Incubator dispatch arm | DONE | character_cash_item_use.go:252 (`SagaType: saga.IncubatorUse` :297) |
| Cash-slot type consts (25/26/27/64/65) | DONE | character_cash_item_use.go:417-421 |
| `incubator-rewards` tenant config resource | DONE | services/atlas-tenants/atlas.com/tenants/configuration/{rest,provider,processor,resource,seed}.go; 6 seed files in services/atlas-tenants/configurations/incubator-rewards/ (cap-1002000, orange-potion, red-potion, scroll-2040000, sword-1302000, white-potion) |
| Seed-template writer/handler rows — original 5 versions | DONE (with drift, see below) | Each of gms_83/84/87/95/jms_185 carries exactly one `IncubatorResult` writer row |

**Original-5 template row detail (writer / handler counts, branch vs. main):**

| Template | IncubatorResult writer | CharacterCashItemUseHandle handler (branch) | same on main |
|---|---|---|---|
| gms_83 | 1 | 1 | 1 |
| gms_84 | 1 | 1 | 1 |
| gms_87 | 1 | **2 (DUP)** | 1 |
| gms_95 | 1 | **2 (DUP)** | 1 |
| jms_185 | 1 | **2 (DUP)** | 1 |

### Finding 1 (post-merge drift, LOW severity) — duplicate handler rows in gms_87/95/jms

`main` already carried a `CharacterCashItemUseHandle` handler in all five templates at the same opcode that Task 18 added independently. After the merge, gms_87/95/jms each contain **two byte-identical handler entries at the same opcode** (gms_87 `0x52`, gms_95 `0x55`, jms_185 `0x47`):

- gms_87: template_gms_87_1.json:363-365 and :846-848 (both opcode `0x52`, `LoggedInValidator`, `CharacterCashItemUseHandle`, same `handlers` array).
- gms_95 (`0x55`) and jms_185 (`0x47`) mirror this.

Runtime impact: **none.** `libs/atlas-opcodes/producer.go:44 BuildHandlerMap` keys `result[uint16(op)] = ...` (last-wins) and does not error on duplicate opcodes; both rows are identical so the resolved handler is unchanged. This is config-cleanliness drift the merge should have resolved, not a functional regression. Task 18's handler-row edits for gms_87/95/jms are now redundant with `main`.

Recommendation: delete the Task-128-added handler rows in gms_87/95/jms (keep main's), leaving one per template. No such duplicate exists for the `IncubatorResult` writer rows (exactly one each) or in gms_83/84.

## Part 2 — New Version-Support Work (commit b2725b065)

### gms_61 / gms_72 / gms_79 — claimed wired + verified

| Claim | Status | Evidence |
|---|---|---|
| Writer wired into seed templates at opcode 0x42 | DONE | template_gms_{61,72,79}_1.json each: one `{"opCode":"0x42","writer":"IncubatorResult"}`; all valid JSON |
| Byte fixtures in result_test.go | DONE | result_test.go:49-51 (`{"GMS",61,short}`,`72`,`79`) — 2-field `short` body, matches v83 |
| packet-audit:verify markers | DONE | result_test.go:27-29 (gms_v61 ida=0x8490d7, gms_v72 ida=0x9203de, gms_v79 ida=0x9722d8) |
| Pinned evidence | DONE | docs/packets/evidence/gms_v{61,72,79}/incubator.clientbound.IncubatorResult.yaml (each with IDA function/address + decompile_sha256) |
| Audit reports | DONE | docs/packets/audits/gms_v{61,72,79}/IncubatorResult.json — all `"Verdict": 0` (match), 2 rows (int itemId + short count) |
| status.json cells promoted to verified | DONE | status.json diff: gms_v61/72/79 `incomplete`→`verified` (opcode 66=0x42); STATUS.md:79 shows `0x042 ✅` for all three |

### gms_48 — claimed correctly NOT wired

| Claim | Status | Evidence |
|---|---|---|
| Writer NOT wired into gms_48 template | DONE | template_gms_48_1.json: `IncubatorResult` count = 0 (handler `CharacterCashItemUseHandle` still present = 1, from its own bring-up) |
| Matrix cell left incomplete | DONE | STATUS.md:79 gms_48 `0x02A ❌`; status.json gms_v48 stays incomplete |
| Divergence documented in deploy-runbook | DONE | deploy-runbook.md:49-63 caveat 3 — OnIncubatorResult @0x71f72a opcode 0x2A is a mode-prefix dispatcher (switch on `Decode1()-6`), flat writer would misparse; needs a dispatcher-family writer |

### Observation (not a defect) — extended-body gate narrowed vs. original plan

The IncubatorResult writer's extended (5-field) body gate is `t.Region()=="GMS" && t.MajorVersion() >= 95` (result.go:46) — i.e. **v95-only**. The original plan (Task 4) had `>=87 || JMS`, which would have made v87 and jms 5-field. The test agrees with the current writer (result_test.go:54 `{"GMS",87,short}`, :56 `{"JMS",185,short}`, :55 `{"GMS",95,extended}`). The change is documented as live-IDA re-verification in commit `b2725b065` and result.go:16-21 / result_test.go:13-20 (v83/84/87/jms all read the 2-field body; only v95 extended). This is an intentional, documented correction made during the version-support session, internally consistent, with markers/evidence/audit reports present. IDA addresses were not independently re-derived in this audit (no live IDA session); flagged as verified-by-artifact, not verified-by-reviewer.

### Finding 2 (consistency, LOW severity) — missing `verifies:` block in new evidence

The original-5 evidence yamls (gms_v83/84/87/95, jms_v185) contain a `verifies:` block pointing at `result_test.go#TestIncubatorResult`; the new gms_v61/72/79 evidence yamls do not. Not enforced by `matrix --check` (the three cells promoted to `verified` regardless), but inconsistent with plan Task 19 Step 3 and with the other five records.

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-packet (incubator/clientbound, cash/serverbound) | PASS | PASS | `go test ./incubator/... ./cash/serverbound/... -count=1` → ok |

(Full-suite build/test was reported green upstream and not re-run here per instructions; spot-check confirms the incubator packet code compiles and tests pass post-merge.)

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — all original deliverables intact; all version-support claims verified real.
- **Recommendation:** NEEDS_FIXES (minor) — dedupe the three duplicate handler rows before merge; the two LOW findings are cleanup, not functional blockers.

## Action Items

1. Remove the duplicate `CharacterCashItemUseHandle` handler rows added by Task 18 in `template_gms_87_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json` (keep main's single row per template) so each has exactly one handler at its opcode.
2. (Optional) Add the `verifies:` block to `docs/packets/evidence/gms_v{61,72,79}/incubator.clientbound.IncubatorResult.yaml` to match the original-5 records.
3. (Optional / already documented) If independent IDA confirmation is desired, re-verify the v87/jms 2-field body decision (writer gate `>=95`) against the live IDBs — the artifacts assert it but no reviewer re-derivation was performed.
