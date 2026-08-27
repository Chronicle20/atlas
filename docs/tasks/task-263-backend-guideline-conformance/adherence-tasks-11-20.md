# Plan Audit — task-263-backend-guideline-conformance (Tasks 11-20)

**Plan Path:** docs/tasks/task-263-backend-guideline-conformance/plan.md
**Audit Date:** 2026-08-27
**Branch:** task-263-backend-guideline-conformance
**Base Branch:** main
**Scope:** Tasks 11-20 only (range shard)

## Executive Summary

All 10 tasks (11-20) are IMPLEMENTED. Evidence is unusually strong: per-task/per-batch commits,
independent reviewer re-derivations recorded in progress.md (as of commit ea15a4477, before the
scaffolding-removal commit f8de7249c), and `tools/verify.sh` gate PASS records for every task.
Spot-checks against the live tree (chair/rest.go, cashshop/rewardpool/rest.go,
query-aggregator/validation/builder.go) confirm the claimed code is actually present, not just
claimed in reports. Task 18 surfaced a genuine 17-package gap (SKIPPED packages with neither
hand-work nor an exemption) via its own escalation criterion, and it was correctly closed by an
inserted Task 18b rather than silently absorbed or left open — this is the sweep working as
designed, not a defect. Task 20 hit a real gate FAIL (gofumpt non-conformance in codemod output)
that was diagnosed as a codemod defect and fixed via a two-stage fix (876d4cb63 partial +
852c472b4 continuation), with the fix verified before being called done.

Because a flagless `tools/verify.sh` run is in progress against this working tree, no build/test
commands were run here (per instructions); build/test PASS status is taken from progress.md's own
gate records, which were themselves independently re-run by reviewers in several cases.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 11 | Apply `transform` to atlas-channel tier-A | IMPLEMENTED | Commit `ed54e564e`; 23 rows (15 APPLIED/8 SKIPPED), reviewer APPROVED (review-task-11.md), gate PASS after gofumpt fix `20765f8d3`. Spot-check: `chair/rest.go:36 func Transform`. |
| 12 | Apply `transform` to remaining tier-A services | IMPLEMENTED | 29 services across 3 sequential batches (all committed, e.g. `81922b270`..`0f4dc6e17`), ledgers `ledger-transform-rest-{1,2,3}.tsv` totalling 61 APPLIED/10 SKIPPED, each batch reviewed APPROVED and gated PASS. `handwork-dom04.tsv` produced per plan Step 3. |
| 13 | W3 hand work — the four #1498 packages | IMPLEMENTED | progress.md "Task 13 — implemented, reviewed APPROVED"; `channel/data/tradeability`, `channel/monsterbook`, `inventory/data/tradeability` all confirmed to have `Transform*` functions (exemptions.md:225-226 cross-references this). Gate initially FAIL (gofumpt), fix queued and cleared. |
| 14 | W3 hand work — remaining NO-RESTMODEL packages | IMPLEMENTED | Split into batches A-D, each implemented + reviewed (A: APPROVED_WITH_FINDINGS, B: APPROVED_WITH_FINDINGS, C: APPROVED, D: APPROVED with 1 non-blocking). `services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go:43 func TransformReward` confirmed live. |
| 15 | Tier B1 hand work (multiple Extract*) | IMPLEMENTED | Fanned out per-service (atlas-channel, atlas-consumables, atlas-inventory, atlas-npc-shops, atlas-doors, atlas-drops, atlas-guilds, atlas-party-quests, atlas-trades), each with its own gate PASS + review APPROVED entry in progress.md (e.g. "Task 15 batch atlas-trades — commits 5e2b905ac + 21c22e538, gate PASS, review APPROVED"). |
| 16 | Tier B2 hand work (non-flat/displaced Extract) | IMPLEMENTED | ~10 batches (channel-a/b/c/d, messages-storage, character, doors-summons-reactors, login-monsters-npcconv-qagg, misc-a, misc-b), each implementer-DONE and reviewer-APPROVED (channel-c and channel-d APPROVED_WITH_FINDINGS, non-blocking). One gate FAIL (lint&format, 24 modules) on gofumpt, fixed at `6a575d0`, re-gate PASS at `37a8799f1`. |
| 17 | Tier C hand work (no Extract at all) | IMPLEMENTED | Commits `7d81d43`,`9680019`,`a71607d`,`dc1eab8`; gate PASS `37a8799f1..8203e0d03`; review CHANGES_REQUIRED (1 blocking) → fixed at `e3faf58` (maps/location Transform mapping RestModel.Id from CharacterId), blocking finding cleared. |
| 18 | Close out DOM-04 and confirm the ledger | IMPLEMENTED (with correctly-handled escalation) | Steps 1-3 run; residue analysis found 17 genuinely forgotten SKIPPED packages with neither hand-work nor exemption (a real design-§10 gap). Per CLAUDE.md producibility bar, this was NOT silently deferred — Task 18b was inserted and closed all 17 (commits `56298756d`+fix `04a1c2262`, `89c396334`, `21c1f2367`; gate PASS `8203e0d03..21c1f2367`). Re-run of Steps 1-2 confirmed residue dropped 30→13, matching the three accepted classes exactly. DOM-04 formally closed. Three pre-existing (not introduced by this branch) `Extract`-drops-field defects were surfaced (`spawnPoint` in atlas-channel/character and atlas-cashshop/character; x/y/stance in atlas-npc-shops/character) and correctly NOT fixed in-branch (would be a behavior change, out of scope), with a recommendation for a separate follow-up task. |
| 19 | W1 `relocate` subcommand | IMPLEMENTED | `codemod/relocate.go`+`relocate_test.go` wired into `main.go` (commit `8c568d6b7`); `TestRelocate` covers all 7 brief cases + 1 more; dry run 64 APPLIED/9 SKIPPED, no service file touched. Gate PASS (thin — codemod module not in verify.sh's set, but `GOWORK=off go test` is the real coverage). Review APPROVED_WITH_FINDINGS (2 non-blocking: missing fixture for resolveBuilderType fallback; the 64/9→73 vs design's 72-count reconciliation carried forward, explicitly assigned to Task 21-C, not silently dropped). |
| 20 | Apply `relocate`, first half | IMPLEMENTED | 7 modules relocated, one commit each (`2301a7f71` query-aggregator, `e25c5b5f3` login, `af27afccb` pets, `7f58dc448` maps, `0a573f178` consumables, `767c65d1a` npc-shops, `c9582f0ec` atlas-script-core), preceded by codemod fix `a433d6513` for two real defects (comment loss, builder.go overwrite). Ledger 33 APPLIED/1 SKIPPED. Review APPROVED (0 blocking) with independent re-derivation of the asymmetry check. Gate initially FAIL (lint&format guard, gofumpt non-conformance in codemod output, 12 targets across 6 modules) — root-caused to the codemod's `format.Source` not being gofumpt-clean; fixed in the codemod itself (not hand-patched) via `876d4cb63` (partial, correctly reverted on cap-out rather than committing unverified output) then continuation `852c472b4`, gate FAIL cleared and confirmed (`tools/lint.sh --check --go` 0 issues ×6). Spot-check: `services/atlas-query-aggregator/.../validation/builder.go:48 func NewValidationContextBuilder` present and correctly formatted. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 20's fix round was recorded transiently as PARTIAL but was completed and closed within the same task before commit to the final state; no open PARTIAL remains)

## Skipped / Deferred Tasks

None. No task in the 11-20 range was skipped. Task 18's 17-package gap was the one place work
could have been silently dropped, and the branch's own record shows it was not — it was escalated,
sized, and closed via Task 18b, with the closure independently re-verified (residue count dropped
exactly as predicted, 30→13).

Three defects were explicitly identified as out-of-scope and deferred to a **separate future task**
(not folded into task-263, and this is the correct call per the branch's own behavior-preservation
constraint, not evasion):
1. `atlas-channel/atlas.com/channel/character` — `Extract` never sets `Model.spawnPoint` (pre-existing).
2. `atlas-cashshop/atlas.com/cashshop/character` — same `spawnPoint` drop (pre-existing).
3. `atlas-npc-shops/atlas.com/npc/character` — `Extract` never sets `x`/`y`/`stance` (pre-existing).

These are pre-existing REST->domain defects unrelated to this branch's Transform/relocate work;
fixing them would change live behavior, which the PRD forbids. Flagging for completeness only —
not a task-263 completion gap.

One open reconciliation thread from Task 19's review (64+9=73 dry-run groups vs design's
documented ≈72 estimate) was explicitly carried forward to Task 21-C rather than resolved or
buried; this is Task 21 scope, outside this shard's range, and is noted here only so the next
shard (21-30) does not miss it.

## Build & Test Results

Not independently run in this audit — a flagless `tools/verify.sh` was reported running against
this working tree at audit time, and re-running any build/test command was explicitly disallowed
to avoid corrupting that gate's result. Build/test status below is as recorded in
`progress.md` (commit `ea15a4477`), itself independently re-run by reviewers in most cases.

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| atlas-channel | PASS | PASS | Tasks 11, 13, 15-18; gofumpt fix required once (Task 11), cleared |
| atlas-messages, atlas-consumables, atlas-monster-death, atlas-inventory, atlas-character, atlas-npc-shops, atlas-query-aggregator, atlas-pets, atlas-monsters, atlas-maps, +~20 more tier-A services | PASS | PASS | Task 12 batches 1-3, each reviewer-verified |
| atlas-cashshop, atlas-dragons, atlas-drops, atlas-messengers, atlas-npc-conversations, atlas-parties, atlas-summons, atlas-saga-orchestrator, atlas-tenants | PASS | PASS | Task 14 NO-RESTMODEL batches A-D |
| ~15 services (B1/B2 tiers) | PASS | PASS | Tasks 15-16, one gofumpt gate FAIL fixed at `6a575d0` |
| codemod module (task-local) | PASS | PASS | Task 19; `GOWORK=off go test ./...` — not covered by main verify.sh module set |
| atlas-query-aggregator, atlas-login, atlas-pets, atlas-maps, atlas-consumables, atlas-npc-shops, atlas-script-core | PASS (after fix) | PASS (after fix) | Task 20; initial gate FAIL on gofumpt (codemod defect), fixed and re-verified `tools/lint.sh --check --go` 0 issues ×6 |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this range; overall branch recommendation depends on
  shards covering Tasks 1-10 and 21-27)

## Action Items

None required for Tasks 11-20 specifically. For awareness of downstream shards:

1. Confirm Task 21-C actually performed the 64/9-vs-72 (73 vs 72) reconciliation carried forward
   from Task 19's review, and that its resolution is recorded before Task 25's exemptions.md was
   finalized (outside this shard's range — flag to the 21-30 shard).
2. Confirm the three pre-existing `Extract`-drops-field defects (spawnPoint x2, x/y/stance x1)
   were recorded as out-of-scope findings in the final PR description / task-263 close-out
   materials rather than lost (outside this shard's range — flag to the 21-30 shard / final review).
