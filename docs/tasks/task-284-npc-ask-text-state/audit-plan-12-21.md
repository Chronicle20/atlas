# Plan Audit — task-284-npc-ask-text-state (Tasks 12-21)

**Plan Path:** docs/tasks/task-284-npc-ask-text-state/plan.md
**Audit Date:** 2026-08-29
**Branch:** task-284-npc-ask-text-state
**Base Branch:** main
**Range Audited:** Plan Tasks 12-21 only (commit range `9cd1ec5af..e6a1540cb`)

## Executive Summary

All ten plan tasks in this range (12-21) are fully implemented with direct evidence in the branch diff. The messageType-table derivation (Task 12) is backed by a provenance file and all four template guards pass. The two packet-audit promotions (Task 13) land with byte-fixture tests and both STATUS.md cells show ✅. The atlas-ui work (Tasks 14-15) matches the plan's type/transition/editorOps/inspector spec exactly, including the non-trivial match-indexed `setTransitionTarget`, and both `npm run build` and `npm test` (282 files / 2352 tests) are clean. All 70 seed content files (Tasks 16-20, 7 NPCs × 10 tenant directories) are present, byte-identical across directories, pass `catalog-lint`, and their JSON bodies match the plan's state tables verbatim, including the intentional asymmetries (2111018's same-turn advance-then-prompt, the doll-cave map-id derivation cited to a live `atlas-data` query, and the documented occupancy/password ordering deviation). Task 21's documentation lands in two commits as the dispatch brief specified (`b957190ba` + the `e6a1540cb` fix round), and both are needed together to close the schema/spec/domain.md work correctly. `go build`/`go test` are clean for `atlas-npc-conversations` and `atlas-packet`; no gitleaks-style path leaks were found in any task-284-owned file.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 12 | messageType tables for gms_87_1, gms_92_1, jms_185_1 | DONE | Commit `9368d1f19`. `messagetype-derivation.md` records per-version binary/md5/addresses. Tables added at `template_gms_87_1.json`, new handler registered in `template_gms_92_1.json` at opcode `0x42` (confirmed against `docs/packets/registry/gms_v92.yaml:2495-2498`, `NPC_TALK_MORE` opcode 66 decimal = 0x42), `template_jms_185_1.json` updated. All four guards (`template-opcode-order-guard.sh`, `template-duplicate-binding-guard.sh`, `template-movement-types-guard.sh`, `operator-cancel-path-guard.sh`) pass. |
| 13 | Promote `NpcAskTextConversationDetail` on v84 and v92 | DONE | Commits `c91c1e0fc`, `21fb8a561`. `conversation_test.go:236-270` adds `TestAskTextConversationDetailEncodeV84`/`V92` with `packet-audit:verify` markers; both pass (`go test -run AskText`). `docs/packets/audits/STATUS.md:1022` row shows ✅ across all 10 version columns. |
| 14 | atlas-ui types, state metadata, transitions, editor ops | DONE | Commit `17d5b6361`. `conversation.ts` adds `AskTextMatch`/`AskTextState`, union member, `StateModel.askText`. `stateMeta.ts`/`transitions.ts`/`editorOps.ts` all updated per plan; `setTransitionTarget` promoted to `export` and gains `"match"`/`"fallback"` kinds addressed by index. `transitions.test.ts` and `editorOps.test.ts` cover every assertion table in the plan (edge count/order/labels/no-dedup/empty-matches; rename rewire; delete rewire matching askNumber's documented quirk; setTransitionTarget by match index and fallback). |
| 15 | atlas-ui askText inspector panel | DONE | Commit `a4f2490ed`. `ConversationInspector.tsx` adds read-only KV view + `AskTextForm` modelled on `AskNumberForm`, plus a `MatchesEditor` with add/remove/reorder and value↔valueFromContext toggle clearing the sibling field. `AskTextForm.test.tsx` (197 lines) covers it. |
| 16 | Content: npc-2111024.json (Magatia lab door) | DONE | 10 files present, byte-identical (md5 `d7916acb...`), body matches plan's state table exactly (`loadPassword`/`askPassword`/`openDoor`/`warpJenu`/`warpAlca`/`wrongPassword`), `sp_jenu`/`sp_alca` mapId branch correct. |
| 17 | Content: npc-2111017/18/19.json (Magatia lab pipes) | DONE | 30 files present, byte-identical per NPC id. Diffs between 2111017/2111018/2111019 match the plan's specified asymmetry exactly — 2111018's `advanceTo3` routes to `askPassword` in the same turn while 2111017/2111019's `advanceTo1`/`advanceTo2` route to `null`. |
| 18 | Content: npc-1063011.json (Thief/Puppeteer merge) | DONE | 10 files present, byte-identical. Doll-cave map id `105070300` is documented in `conversion-notes.md:98-109` as derived from a live `GET /api/data/maps/105070300/portals` query confirming portal 3 (`in00`) carries `scriptName: "enterDollcave"` — not invented, and the plan's "stop and ask" condition did not trigger because the map id was reachable. State table (puppeteerPreCheck/puppeteerBlocked/askPassword/thiefGate/puppeteerGate1/puppeteerGate2/etc.) matches the plan verbatim. |
| 19 | Content: npc-2091009.json (Sealed Shrine) | DONE | 10 files present, byte-identical. Body matches plan exactly, including the documented occupancy/password ordering deviation (password checked before occupancy, since askText's matches gate first). |
| 20 | Content: npc-1092019.json (Nautilus seagull quiz) + conversion-notes.md | DONE | 10 files present, byte-identical. Text transcribed verbatim including the "intellingence" misspelling and the `\r\n\r\n` + leading double-space in `finalHint`. `seagullProgress == 1` arm correctly omitted (falls through to `null` at `branchProgress`'s catch-all), not stubbed. `conversion-notes.md` covers all required items: the omitted nine-Barts arm, the occupancy/password ordering swap, the 1063011 merge and its map-id dependency, the portal-owned exclusions, and the nine referenced-but-unseeded quest ids. |
| 21 | Documentation | DONE | Commits `b957190ba` + `e6a1540cb` (fix round, taken together per dispatch instructions). Both JSON schemas gain the `askText` enum entry and property block matching `RestAskTextModel` field names; `contextKey` correctly dropped from `required` in the fix round to match the REST default-fill behavior. `domain.md` documents the state. `npc_conversation_conversion_spec.md` gains the `sendGetText → askText` mapping (lines 222-296) and the `local:get_quest_progress` reference (lines 573, 913-948). `docs/research/missing-features/npc-content.md` created, covering all 8 converted scripts and all blocked ones (2101014/1052014/2010009/changeName/2030013_old) plus the nine-Barts omission. No `/home/` literal found in any task-284-owned doc file. |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range.

## Out-of-plan additions in range (judged for correctness only, per dispatch instructions)

- **`7bc963455`** (genericAction AND-semantics fix): `processGenericActionState` previously evaluated only `Conditions()[0]`. Fixed to loop the full slice with short-circuit on first false. `processor_condition_and_test.go` (254 lines) added. `go test ./...` for `atlas-npc-conversations` passes. This also retroactively tightens the two-condition gates already authored in Tasks 18/19 (`puppeteerGate1`/`puppeteerGate2`, `checkQuest`), which depend on true AND semantics to behave as specified.
- **`bbf4c4782`** (askText out-of-range re-prompt): previously a length violation returned a bare error that the sole caller discarded, silently stalling the conversation. Fixed to set `nextStateId` to the current state (re-prompt) without storing the rejected input, mirroring `processAskNumberState`'s established pattern. `processor_asktext_reprompt_test.go` (216 lines) added; `processor_asktext_test.go` updated for the new signature. Tests pass.

Both fixes are correctly implemented, tested, and documented in the Task 21 fix-round commit (`e6a1540cb`).

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/npc`) | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` all packages ok, including `conversation` (Tasks 16-20 engine deps) and the two out-of-plan fix commits' new test files. |
| atlas-packet (`libs/atlas-packet`) | PASS | PASS | `go build ./...` exit 0; `go test ./... -count=1` clean, including `TestAskTextConversationDetailEncodeV84`/`V92` (Task 13). |
| atlas-configurations templates | N/A (JSON) | N/A | Verified via the four template guards instead (all PASS); no Go changes in this range. |
| Seed content (`deploy/seed/...`) | N/A | PASS | `tools/catalog-lint` run against `deploy/seed` — clean, no output. |
| atlas-ui (`services/atlas-ui`) | PASS | PASS | `npm run build` — clean (only a pre-existing chunk-size warning, unrelated to askText). `npm test` — 282 test files / 2352 tests, all passed. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (scoped to Tasks 12-21; final merge decision should also incorporate the 1-11 shard's findings)

## Action Items

None. No gaps found in Tasks 12-21.
