# Plan Audit — task-129-vicious-hammer-use

**Plan Path:** docs/tasks/task-129-vicious-hammer-use/plan.md
**Audit Date:** 2026-07-03
**Branch:** task-129-vicious-hammer-use
**Base Branch:** main (merge-base 38d4d0ba2)

## Executive Summary

All 16 plan tasks were faithfully implemented; nothing was silently skipped, stubbed, or left as a `// TODO`. The four affected Go modules (atlas-constants, atlas-packet, atlas-consumables, atlas-channel) build, vet, and test clean, and every packet-audit gate (dispatcher-lint, operations --check, fname-doc --check, matrix --check) plus redis-key-guard passes with exit 0. The three documented IDA-verified corrections to the plan's version-uniform assumptions (v84 opcodes 0x10B/0x16C, v87 modes 63/64, v95 modes 65/66) are correctly reflected across the codec comments, dispatcher yaml, seed templates, registry, and audit matrix. jms and gms_v92 are documented as out-of-scope with no code, as planned.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-constants ClassificationViciousHammer (557) | DONE | libs/atlas-constants/item/constants.go:106; test constants_test.go:77 |
| 2 | serverbound ItemUseViciousHammer tail codec | DONE | libs/atlas-packet/cash/serverbound/item_use_vicious_hammer.go:18-54 (itemTI/slotPosition/updateTime) |
| 3 | serverbound ItemUpgradeUpdate codec + Handle const | DONE | libs/atlas-packet/field/serverbound/item_upgrade_update.go:12,21-49 |
| 4 | ViciousHammer clientbound dispatcher (Open/Success/Failure + body funcs) | DONE | field/clientbound/vicious_hammer.go:36-143; field/vicious_hammer_body.go:23-45; WithResolvedCode used, no hard-coded modes |
| 5 | packet-audit wiring (yaml, run.go candidates, retire stubs) | DONE | docs/packets/dispatchers/vicious_hammer.yaml; tools/packet-audit/cmd/run.go:2428-2442; FieldViciousHammer.{yaml,json,md} stubs removed (git diff -8/-11/-13 lines each version) |
| 6 | Cash() equip data + AddHammersApplied change | DONE | data/equipable/model.go:20,87; rest.go:149; equipable/processor.go:136; test processor_test.go |
| 7 | Kafka message types + event producer | DONE | kafka/message/consumable/kafka.go:20,47,72,94; consumable/producer.go:45 |
| 8 | hammer request/consume flow (validate + atomic apply) | DONE | consumable/processor.go:868-1023 (RequestViciousHammer:946, ConsumeViciousHammer:989, ViciousHammerError:929, resolveViciousHammerTarget:883, viciousHammerErrorCode:901); consumer.go:71,37; test vicious_hammer_test.go |
| 9 | channel Kafka types/producer/processor | DONE | channel kafka/message/consumable/kafka.go:18,44,53,75; producer.go:52; processor.go:38 |
| 10 | token helpers + hammer arm in CharacterCashItemUseHandle | DONE | vicious_hammer_token.go:9,13; character_cash_item_use.go:109-113,126-138,490-492,527-549; stale `// TODO for v83 ... updateTime` removed |
| 11 | ItemUpgradeUpdateHandle handler + registration | DONE | socket/handler/item_upgrade_update.go:21; main.go:868 |
| 12 | hammer-result Kafka consumer | DONE | kafka/consumer/consumable/consumer.go:110 (handler), :53 (registration) |
| 13 | seed templates — handler entries + operations tables | DONE | template_gms_{83,84,87,95}_1.json: handler opcodes 0x104/0x10B/0x112/0x128, writer opcodes 0x162/0x16C/0x177/0x1A9, operations SUCCESS 61/61/63/65; all validators = LoggedInValidator; JSON valid |
| 14 | packet verification v83 + v95 | DONE | verify markers item_upgrade_update_test.go:17,20 + vicious_hammer_test.go:43-45,52-54; audits/gms_v83/, gms_v95/ reports; export splices; STATUS.md cells ✅ |
| 15 | packet verification v84 + v87; jms disposition | DONE | markers item_upgrade_update_test.go:18-19 + vicious_hammer_test.go:46-51; audits/gms_v84/, gms_v87/; jms ITEM_UPGRADE_UPDATE stays ❌ documented version-absent |
| 16 | full verification gates + rollout doc | DONE | rollout.md present; all module gates + packet-audit gates + redis-key-guard pass (see below) |

**Completion Rate:** 16/16 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. jms and gms_v92 dispositions are documented (no code) exactly as the plan's Global Constraints require, not silent skips: STATUS.md shows jms ITEM_UPGRADE_UPDATE (0x114) ❌ and VICIOUS_HAMMER clientbound ⬜ (registry-absent); the dispatcher yaml header records both omissions.

## Intentional Corrections (verified correct, NOT deviations)

- v84 serverbound 0x104→**0x10B**, clientbound 0x169→**0x16C**: template_gms_84_1.json handler @1177, writer @2759; marker ida=0x8562d1; yaml header documents decoder sub_85676C behind IDB-mislabeled forwarder 0x5443af (CField::OnCharacterSale).
- v87 modes 61/62→**63/64**: template_gms_87_1.json operations SUCCESS 63; yaml SUCCESS gms_v87:63 FAILURE:64.
- v95 modes 61/62→**65/66**: template_gms_95_1.json operations SUCCESS 65; yaml gms_v95:65/66; codec comments corrected to OnItemUpgradeResult 0x7c0fd0 (ShowResult disproven).
- Dead writer wrapper services/atlas-channel/.../socket/writer/vicious_hammer.go deleted (file absent; commit 7e847bcfdb).

## Build & Test Results

| Module | Build | Vet | Tests | Notes |
|--------|-------|-----|-------|-------|
| libs/atlas-constants | PASS | PASS | PASS | item tests ok |
| libs/atlas-packet | PASS | PASS | PASS | field + cash serverbound suites ok |
| atlas-consumables | PASS | PASS | PASS | equipable + consumable suites ok |
| atlas-channel | PASS | PASS | PASS | socket/handler + writer suites ok |

**Repo/packet gates:** dispatcher-lint clean (exit 0); operations --check OK (exit 0); fname-doc --check OK (exit 0); matrix --check exit 0; redis-key-guard exit 0.

Docker bakes (plan Task 16 Step 2) were NOT re-run in this audit (no Go image built here); no new shared lib was added, so no Dockerfile/go.work edits were required — go.work.sum is the only workspace change.

## Findings

### Minor (confirmed, pre-existing pattern)

1. **consumable/processor.go:972-974** — `RequestViciousHammer` discards the `RegisterHandler` error: `_, err = consumer.GetManager().RegisterHandler(...)` is immediately overwritten by `err = p.cpp.RequestReserve(...)` without an intervening check. This is mirrored verbatim from the pre-existing `RequestScroll` (lines 219-221) and `ConsumeScroll` (572-574) flows, so it is a pattern-level nit, not a task-129 regression. If the reservation-status consumer fails to register, the OneTime callback would never fire and the dialog could stall, but the failure surface is identical to the established scroll path.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required for merge. Optional cleanup (repo-wide, out of scope for this task): check the `RegisterHandler` error before calling `RequestReserve` in all three consumable flows (RequestScroll, ConsumeScroll, RequestViciousHammer) so a failed handler registration surfaces instead of being silently overwritten.

---

## Controller triage of review findings (task-129)

Backend-guidelines findings are in `audit-backend.md`. Both reviews returned **no Critical/Important**. Minor dispositions:

1. **DOM-21 raw `category == 557`** (`socket/handler/character_cash_item_use.go:488`) — left as-is: it sits in a uniform block of ~20 raw `category == NNN` sibling guards; using the constant for only this one would be a local-idiom outlier. Functionally correct.
2. **Magic cap/notice literals in `handleViciousHammerOpen`** — left: notice codes carry inline comments; the plan explicitly called the `viciousHammerCap` hoist optional.
3. **`RegisterHandler` error discarded in `RequestViciousHammer`** — left: verbatim mirror of the pre-existing `RequestScroll`/`ConsumeScroll` pattern (not a task-129 regression).
4. **Consume-after-mutate ordering in `ConsumeViciousHammer`** — left: mirrors `ConsumeScroll`; a symptom of the project-wide `ExecuteTransaction` no-op, atomicity provided by the reserve→consume-callback + compensating failure event.
5. **Emit paths untested (only pure helpers)** — left: the plan's chosen TDD scope; testing emit paths would require Kafka/producer mocks the project's DOM-24 says to avoid.

Verdict: **merge-ready.** All 16 plan tasks implemented and verified; per-version opcode/mode divergence (v84 0x10B/0x16C, v87 63/64, v95 65/66) IDA-verified and corrected during execution.
