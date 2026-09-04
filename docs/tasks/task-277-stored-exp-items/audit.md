# Plan Audit — task-277-stored-exp-items

**Plan Path:** docs/tasks/task-277-stored-exp-items/plan.md
**Audit Date:** 2026-08-28
**Branch:** task-277-stored-exp-items
**Base Branch:** main (merge-base bda6566f3f87da03c6e65719f2e8163acfbf5bb6)
**Commit range audited:** bda6566f3f87..7ec1cb118

## Executive Summary

All 14 plan tasks were implemented, matching the plan/design faithfully. Independent evidence checks confirm the two highest-risk items named in this audit's brief: the Task 13 fix round correctly removed the contradictory `exp` table row and corrected the false `gms_v12` positive-absence claim (verified byte-for-byte against `docs/packets/feature-na-evidence.yaml`, which carries only `gms_v48`/`gms_v61` entries), and the Task 14 corpus-guard fix (3387→3403) matches exactly 16 new bindings (2 handlers × 8 in-scope seed templates). Opcode-to-hex conversions in all 8 routed templates were independently spot-checked against `docs/packets/registry/*.yaml` and match exactly. No stubs, skips, or new TODOs were introduced. `tools/verify.sh` was reported flagless-green at 648b103ca (not re-run per instructions).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `SolomonItemUse` audit-only wrapper (`libs/atlas-packet`) | DONE | `libs/atlas-packet/inventory/serverbound/solomon_item_use.go` — thin wrapper over `ItemUse` bound to `CharacterItemUseSolomonHandle`, with `packet-audit:fname CWvsContext::SendExpUpItemUseRequest`. Tests present (`solomon_item_use_test.go`). |
| 2 | `StoredExperienceUse` codec | DONE | `libs/atlas-packet/character/serverbound/stored_experience_use.go` — decodes the 4-byte tick only, `packet-audit:fname CWvsContext::SendTempExpUseRequest`. Tests present. |
| 3 | Packet registry rows + positive-absence evidence | DONE | `USE_SOLOMON_ITEM`/`USE_GACHA_EXP` rows present in all 8 in-scope registry yamls (gms_v72/79/83/84/87/92/95, jms_v185); opcodes cross-checked against Task 12 seed templates (all match exactly, e.g. gms_v79: 155/156 = 0x9B/0x9C). gms_v79 IDA-address hex→decimal defect (9822440/9822960 vs correct 9821928/9822448) was caught and fixed in the same task per ledger (progress.md:46-48); committed registry now shows correct decimals (9821928 at gms_v79.yaml:2537). |
| 4 | Evidence records and coverage-matrix promotion | DONE | Commit 640b6958d touches 8 pairs of `.verify`/na-evidence yamls + IDA export json for all 8 columns (26 files, 463 insertions). Ledger records an APPROVED_WITH_FINDINGS review (0 blocking, 4 deferred minors) with an honest disclosure that 6/8 columns were not directly re-confirmed by IDA in this session, corroborated instead by cross-checks — documented, not hidden. |
| 5 | `atlas-data` — expose `info/maxLevel` and `spec/exp` | DONE | `services/atlas-data/atlas.com/data/consumable/reader.go:61` (`m.MaxLevel = ...`), `:149` (`m.Spec[SpecTypeExperience] = ...`); `rest.go` adds `SpecTypeExperience` const and `MaxLevel uint32 json:"maxLevel"` field. `reader_test.go` has 4-subtest table `TestReadMaxLevelAndExperienceSpec` covering absent-field paths distinctly. |
| 6 | `atlas-consumables` — consume the two new fields | DONE | `data/consumable/model.go:120-126` (`MaxLevel()` accessor), `rest.go:23,120,198` (RestModel field + Transform both directions). `rest_test.go:135` (`TestExtractMaxLevelAndExperienceSpec`) round-trips both zero and non-zero cases. |
| 7 | Classification 237 constant + credit command seam | DONE | `libs/atlas-constants/item/constants.go` adds `ClassificationConsumableExpUpItem = Classification(237)` with derivation note citing `is_exp_up_item` at gms_v95 0x5078be. Kafka `CommandCreditStoredExperience`/`CommandRedeemStoredExperience` + body structs at `services/atlas-character/.../kafka/message/character/kafka.go:38-39,196-209`. |
| 8 | `atlas-character` — `CREDIT_STORED_EXPERIENCE` | DONE | `character/processor.go:1470-1511` — saturating add (`math.MaxUint32-current` guard, FR-17 comment), single-tx read-then-update, no-op event suppression when `amount==0`/`adjusted==current`. Consumer wired at `kafka/consumer/character/consumer.go` (`handleCreditStoredExperience`). |
| 9 | `atlas-character` — `REDEEM_STORED_EXPERIENCE` | DONE | `character/processor.go:1519-1571` — zero-balance no-op (FR-11/FR-13 replay-safety), level>50 gate (`storedExperienceMaxLevel = byte(50)`), zero-then-award ordering inside one transaction so a failed award rolls the zero back. Consumer wired (`handleRedeemStoredExperience`). |
| 10 | `atlas-consumables` — `ConsumeSolomon` consumer | DONE | `consumable/solomon.go` — parallel read group (fields/data/character), eligibility gates in the correct order (spec/exp presence/positivity, maxLevel, existing non-zero balance) all BEFORE `ConsumeItem` commit, credit call after commit. Routed ahead of the reward-table fallback in `processor.go` (`routesToSolomon(itemId)` branch), with an explicit comment on why it must precede that fallback. `solomon_test.go` (288 lines) present. |
| 11 | `atlas-channel` — the two socket handlers | DONE | `CharacterItemUseSolomonHandleFunc` in `character_item_use.go` (decodes shared `ItemUse`, calls `RequestItemConsume`) and `CharacterUseStoredExperienceHandleFunc` in `character_stored_experience_use.go` (decodes `StoredExperienceUse`, calls `RedeemStoredExperience` via a test-seam processor func). Test file present (`character_stored_experience_use_test.go`, 101 lines). |
| 12 | Seed templates — route both handlers on 8 in-scope versions | DONE | Commit 4e333abb8 touches exactly the 8 in-scope template files (gms_72/79/83/84/87/92/95, jms_185), 18 lines each = 2 opcode entries, 16 total. Opcodes independently verified to match Task 3's registry rows exactly for all 8 columns. gms_12/48/61 templates untouched — confirmed via `git diff --stat` filter (no other template files in the commit). |
| 13 | Documentation and re-ingest follow-up | DONE (after fix round) | Initial pass (6559f3bad) had 2 blocking review findings (contradictory `exp` table row; false `gms_v12` na-evidence claim introduced by the controller's own dispatch prompt, not the implementer). Fix commit 439e02fdd verified: `exp` row removed from the Wholly-missing table with a footnote pointer (`items-and-consumables.md:87`), and the corrected sentence at `:147` now correctly attributes na-evidence only to `gms_v48`/`gms_v61` and separately, correctly, describes `gms_v12` as "out of scope... not na-evidenced absence." Cross-checked against `docs/packets/feature-na-evidence.yaml:252-290` — exactly 4 entries, `gms_v48` and `gms_v61` only, zero `gms_v12` entries. Match confirmed. |
| 14 | Full verification gate | DONE (after fix round) | First flagless run failed on a real defect: `corpus_test.go` guard was stale at 3387. Fix (7b197cd46) bumped it to 3403 with a task-277 clause appended to the narrative naming the exact 16 bindings; 3403-3387=16, matching Task 12's actual seed-template diff exactly. Ledger records the fix-round review as APPROVED (0 blocking) and a subsequent flagless `verify.sh` pass at 648b103ca. Not re-run in this audit per instructions. |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Task 13 and Task 14 each went through one fix round (both recorded transparently in the controller ledger, `.superpowers/sdd/plan/progress.md`), and both fix rounds are independently confirmed landed and correct by this audit (see Task 13/14 rows above). No task was silently skipped, stubbed, or deferred without an explicit ledger entry.

One item is worth naming as a residual, non-blocking limitation rather than a plan gap: Task 4's ledger entry (progress.md:69) discloses that only 2 of 8 matrix columns were directly re-confirmed via IDA in this session; the other 6 rely on cross-checks (packet-audit verify markers, Task 3 registry rows, Task 1/2 fixtures) rather than a fresh IDA read. This is disclosed in the ledger as an accepted risk, not a hidden gap, and does not correspond to any unchecked plan task.

## Build & Test Results

Not re-run in this audit (explicitly out of scope per the dispatch brief — `tools/verify.sh` was reported flagless-green at 648b103ca and this audit's job is code-level adherence verification, not re-verification). Evidence of test coverage per affected package was confirmed by file presence and content inspection:

| Service/Lib | Evidence of tests | Notes |
|---|---|---|
| libs/atlas-packet | Present | `solomon_item_use_test.go`, `stored_experience_use_test.go` |
| atlas-data | Present | `reader_test.go` (`TestReadMaxLevelAndExperienceSpec`, 4 subtests) |
| atlas-consumables | Present | `rest_test.go`, `producer_test.go`, `solomon_test.go` (288 lines) |
| atlas-character | Present | `processor_stored_experience_test.go` (327 lines) |
| atlas-channel | Present | `character_stored_experience_use_test.go` (101 lines), `producer_test.go` |
| atlas-configurations | Present | `corpus_test.go` fixed and passing per ledger (648b103ca) |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. All 14 plan tasks are implemented with file-level evidence, both fix rounds (Task 13 docs, Task 14 corpus guard) are confirmed landed correctly, and no stubs/skips/placeholders were introduced.
