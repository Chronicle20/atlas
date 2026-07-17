# Plan Audit — task-123-megaphones-maple-tv

**Plan Path:** docs/tasks/task-123-megaphones-maple-tv/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-123-megaphones-maple-tv
**Base:** main (c9490b724..0950d4617, 34 commits)

## Executive Summary

All 21 plan tasks landed with file:line evidence; Amendment A1 (DOM-25 config-resolution for the three client wire codes) is fully and correctly implemented — no surviving hardcoded literal at any reject/render call site. All four `go build`, all `go vet`, and all `go test` runs are clean across the six affected modules/services, and all four `packet-audit` gates (`dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check`) plus `redis-key-guard.sh` exit 0. The only genuine gaps are the ones the plan itself flags as incomplete: 4 of 6 serverbound USE_CASH_ITEM sub-bodies remain BLOCKED for v83 (2/6 verified) and 2 of 6 for v95 (4/6 verified) with v84/v87/jms serverbound cells entirely unattempted, all honestly reflected as ❌ in the coverage matrix (never falsely ✅); and Task 21 Step 5 (live-tenant acceptance) was not performed, as expected for a human/deploy step. Plan.md's own checkboxes (all 100 `- [ ]`) were never ticked during execution — a process/bookkeeping gap only, not a functional one; every task's actual code deliverable was independently verified against source.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Serverbound sub-body structs | ✅ DONE | `libs/atlas-packet/cash/serverbound/item_use_{megaphone,super_megaphone,item_megaphone,triple_megaphone,maple_tv,avatar_megaphone}.go` all present with matching `_test.go`; commit 3ca0c6dfc |
| 2 | Saga lib type/actions/payloads/unmarshal + channel re-export | ✅ DONE | `libs/atlas-saga/model.go:29,177-178`; `payloads.go:925,964,976,991`; `unmarshal.go:540,546`; channel re-export `saga/model.go:30-33,56,85-86`; `TvMessageType string` (A1) at `payloads.go:1001`; commit 387fee6cc |
| 3 | Discrete `WorldMessageMegaphone`; re-cut `WorldMessageItemMegaphone` | ✅ DONE | `chat/clientbound/world_message.go:165,203,211`; writer call site updated `socket/writer/world_message.go:90-91`; commit c4e5f7247 |
| 4 | WorldMessage per-mode body functions | ✅ DONE | `chat/world_message_body.go:22-41`, all four funcs use `atlas_packet.WithResolvedCode`; commit ce744e43a |
| 5 | Avatar megaphone clientbound + A1 body | ✅ DONE | `chat/clientbound/avatar_megaphone.go:29,109,139,145`; A1 file `chat/avatar_megaphone_body.go` resolves `errorCodes` via `ResolveCode`, never a literal; commit 4f654d04e |
| 6 | Maple TV clientbound + A1 body | ✅ DONE | `tv/clientbound/{set_message,clear_message,send_message_result}.go`; A1 file `tv/tv_body.go` resolves `messageTypes`/`errorCodes`; `TvResultReason` consts match the corrected 1/2/3 mapping; commit 11a7b06bd |
| 7 | atlas-world broadcast model/registry (CAS) | ✅ DONE | `broadcast/model.go`, `broadcast/registry.go:59,63` (`Upsert` create-on-missing via `ErrNotFound`); adjudicated deviation (merged single-CAS Upsert vs. plan's two-CAS) correctly in place; commit af5c64376 + CAS-retry fix 065b121ce |
| 8 | Kafka messages/producer/processor + sweep | ✅ DONE | `kafka/message/broadcast/kafka.go` (`TvMessageType string` at :31,55,77 — A1); `kafka/producer/broadcast/producer.go` with test; `broadcast/processor.go`; commit 04324b530 |
| 9 | Consumer, REST, leader-gated sweep, main wiring | ✅ DONE | `kafka/consumer/broadcast/consumer.go`; `broadcast/{resource,rest,task}.go`; `main.go:69-71` (`InitRegistry`), `:93` (`InitResource`), `:137-152` (`lock.New("world-broadcast-sweep", ...)`); commit 567cc2931 |
| 10 | Orchestrator action handlers/producers | ✅ DONE | `kafka/message/{megaphone,broadcast}/kafka.go` (TvMessageType string at :34 — A1); `saga/handler.go:162-163` (interface), `:904-907` (dispatch), `:3093,3122` (handler impls); commit 3a6201fb9 |
| 11 | Channel messages + world-broadcast REST client | ✅ DONE | `kafka/message/megaphone/kafka.go`, `kafka/message/worldbroadcast/kafka.go` (TvMessageType string at :40 — A1); `worldbroadcast/{processor,requests,rest}.go` + rest_test.go; commit 0d78da2cb |
| 12 | USE_CASH_ITEM megaphone handler branches | ✅ DONE | `character_cash_item_use.go:164-169` classification-first dispatch; `character_cash_item_use_megaphone.go` full three-branch implementation; `socket/model/snapshot.go` all four converter funcs present; TODO at :108 confirmed removed (grep clean); commit d9e361f78 |
| 13 | Megaphone broadcast consumer | ✅ DONE | `kafka/consumer/megaphone/consumer.go`; writer wrappers `socket/writer/world_message.go:82-94`; commit eec73b471 |
| 14 | World-broadcast status consumer | ✅ DONE | `kafka/consumer/worldbroadcast/consumer.go:76-160` — QUEUED ack via `TvSendMessageResultSuccessBody()`, STARTED via `TvSetMessageBody`/`NewSetAvatarMegaphone`, ENDED via `TvClearMessageBody()`/`NewClearAvatarMegaphone()`; commit 221767c80 |
| 15 | Channel main wiring + deploy topic vars | ✅ DONE | `main.go:31,57` imports, `:233-234` InitConsumers, `:528,531` InitHandlers, `:762-767` writer registration; three topic vars present in all three deploy files; commit c8c10110b |
| 16 | Seed templates (5 versions) | ✅ DONE | All 6 writers present in gms_83/84/87/95; jms correctly omits `AvatarMegaphoneResult` (D9); `errorCodes`/`messageTypes` options tables seeded per Amendment A1.2 with the corrected per-version codes (83/84, 86/87, 88/89, 96/97; TvSendMessageResult 1/2/3 everywhere); `CharacterCashItemUseHandle` present in all 5; commit f25492f1c |
| 17 | Live-tenant rollout runbook | ✅ DONE | `docs/tasks/task-123-megaphones-maple-tv/rollout.md` (357 lines) — writer/handler deltas, the three DOM-25(d) options tables explicitly called out, PATCH procedure, restart order, pitfalls; commit c6603adf0 |
| 18 | WorldMessage dispatcher family enrollment | ✅ DONE | `docs/packets/dispatchers/worldmessage.yaml` with per-version IDA-derived mode tables + evidence comments; 4 `#`-entries in `tools/packet-audit/cmd/run.go:1469-1478`; `dispatcher-lint` exits 0; commit b413d8126 |
| 19 | Packet verification gms_v83/v95 | ⚠️ PARTIAL (by design — see below) | Clientbound rows (WorldMessage 4 arms, avatar 3, TV 3) all ✅ for v83+v95 in STATUS.md; serverbound: v83 2/6 (Megaphone, SuperMegaphone), v95 4/6 (+ItemMegaphone, +AvatarMegaphone); TripleMegaphone/MapleTV BLOCKED both versions — explicitly documented in task-19-v83-report.md/task-19-v95-report.md, not silently dropped |
| 20 | Packet verification gms_v84/v87/jms + final matrix | ⚠️ PARTIAL (by design — see below) | Clientbound rows ✅ across v84/v87/jms (jms AVATAR_MEGAPHONE_RESULT correctly ⬜/absent, SET_TV/CLEAR_TV/ENABLE_TV jms mostly ✅ except SEND_TV jms ❌); serverbound USE_CASH_ITEM sub-bodies 0/6 attempted for v84/v87/jms per prompt's known-incomplete list — matrix shows ❌, not fabricated ✅; commits 5366cf13e, 04d29316a |
| 21 | Final gates, docs, TODO sweep, acceptance | ⚠️ PARTIAL | Step 1 gates: all pass (see Build & Test below). Step 2 docker bakes: not run in this audit (not requested; go build/vet/test all clean). Step 4 TODO sweep: clean (only pre-existing unrelated TODO at `cash/clientbound/shop_operation_body.go:80`, untouched by this branch). Step 5 live acceptance: genuinely NOT done — no deploy, no observations doc — correctly absent rather than falsely claimed |

**Completion Rate:** 21/21 tasks structurally implemented; 2 of the 21 (19, 20) carry a plan-sanctioned partial-verification footnote that is honestly reflected in the matrix, not silently claimed done.
**Skipped without approval:** 0
**Partial implementations:** 3 (Tasks 19, 20, 21 — all for reasons the plan itself anticipates: IDA-blocked cells and the human live-acceptance step)

## Skipped / Deferred Tasks

None skipped. The three PARTIAL items are all pre-adjudicated by the prompt's "known INCOMPLETE items" list and verified here to be honestly represented:

- **Serverbound USE_CASH_ITEM sub-bodies** (`ItemUseItemMegaphone`, `ItemUseTripleMegaphone`, `ItemUseMapleTV`, `ItemUseAvatarMegaphone` for v83; `ItemUseTripleMegaphone`/`ItemUseMapleTV` for v95; all six for v84/v87/jms): coverage matrix (`docs/packets/audits/STATUS.md`) shows these as ❌, matching source-of-truth `// packet-audit:verify` marker absence in `item_use_triple_megaphone_test.go` and `item_use_maple_tv_test.go` (zero markers in either file). `matrix --check` passes, meaning the matrix is internally consistent with the code — no drift, no false promotion. Impact: the wire shapes for Triple Megaphone and Maple TV serverbound decode remain Cosmic-derived and unverified against any IDB; a future dedicated IDA pass is required (task-19/20 reports recommend budgeting more time than the other 12 cells combined took).
- **Task 21 Step 5 (live acceptance)**: requires a deployed tenant; not performable in this worktree. No file in the task folder falsely claims it was done.

## Amendment A1 Adherence — VERIFIED CLEAN

Grepped every reject/render call site and the full diff for the three DOM-25 wire values:

- `character_cash_item_use_megaphone.go:206` → `tvpkg.TvSendMessageResultErrorBody(tvpkg.TvResultQueueTooLong)` (not `NewTvSendMessageResultError(2)`)
- `character_cash_item_use_megaphone.go:300` → `chatpkg.AvatarMegaphoneResultBody(chatpkg.AvatarMegaphoneWaitingLine)` (not `NewAvatarMegaphoneResult(83, "")`)
- `kafka/consumer/worldbroadcast/consumer.go:80,113` → `TvSendMessageResultSuccessBody()` / `TvSetMessageBody(tvpkg.TvMessageType(e.TvMessageType), ...)` — semantic key in, resolved byte out
- `TvMessageType` is `string` (never `byte`) in all five locations it's carried: `libs/atlas-saga/payloads.go:1001`, `services/atlas-world/atlas.com/world/kafka/message/broadcast/kafka.go:31/55/77`, `services/atlas-world/atlas.com/world/broadcast/model.go:26`, orchestrator `kafka/message/megaphone/kafka.go:34`, channel `kafka/message/worldbroadcast/kafka.go:40`. Codebase-wide grep for `TvMessageType byte` returns zero hits.
- `grep -rn "NewAvatarMegaphoneResult(\|NewTvSendMessageResultError(" services/` (excluding tests) returns zero call sites outside `libs/atlas-packet` codec internals.

No violation found.

## Build & Test Results

| Module/Service | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| libs/atlas-packet | PASS | PASS | PASS | full `go test ./...`; race-mode spot-check on chat/tv/cash/serverbound also PASS |
| libs/atlas-saga | PASS | PASS | PASS | |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | full package list incl. `worldbroadcast`, `socket/handler`, `socket/writer`, `socket/model` |
| services/atlas-world/atlas.com/world | PASS | PASS | PASS | incl. `broadcast` (race-mode PASS), `kafka/producer/broadcast`, `kafka/consumer/channel` |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | PASS | `saga` package (handler dispatch) PASS |
| services/atlas-configurations/atlas.com/configurations | PASS | PASS | PASS | `templates`, `tenants`, `seeder` all PASS |
| packet-audit dispatcher-lint | — | — | PASS (exit 0) | "dispatcher-lint: clean" |
| packet-audit matrix --check | — | — | PASS (exit 0) | no output, exit 0 |
| packet-audit fname-doc --check | — | — | PASS (exit 0) | "fname-doc check OK" |
| packet-audit operations --check | — | — | PASS (exit 0) | "operations check OK (0 absent-writer notes)" |
| tools/redis-key-guard.sh | — | — | PASS (exit 0) | clean across all 60 services |

Docker `buildx bake` for atlas-channel/world/saga-orchestrator/configurations was NOT run in this audit (out of the requested scope — go build/vet/test/gates were the specified checks); no new shared libs were added on this branch so Dockerfile COPY risk is low, but this remains an outstanding pre-PR step per CLAUDE.md §Build & Verification.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (FULL on code/config/docs; the two verification-scope tasks carry plan-anticipated partial coverage, honestly reflected)
- **Recommendation:** NEEDS_REVIEW — code is clean and A1-compliant; before PR, run `docker buildx bake` for the four touched services (mandatory per CLAUDE.md, not yet executed) and decide whether to schedule the follow-up IDA pass for TripleMegaphone/MapleTV serverbound (and v84/v87/jms serverbound entirely) before or after merge.

## Action Items

1. Run `docker buildx bake atlas-channel atlas-world atlas-saga-orchestrator atlas-configurations` from the worktree root — mandatory per CLAUDE.md, not covered by this audit's go build/test pass.
2. Schedule a dedicated follow-up pass for the BLOCKED serverbound cells (`ItemUseTripleMegaphone`, `ItemUseMapleTV` on v83/v95; all six sub-bodies on v84/v87/jms) — task-19/20 reports estimate this needs more IDA time than the rest of the campaign combined.
3. Perform Task 21 Step 5 live-tenant acceptance after deploy, per `rollout.md`'s procedure, and record observations in the task folder.
4. Optional bookkeeping: tick the plan.md checkboxes (all 100 are still `- [ ]`) to reflect actual completion state, or note in the PR description that tracking was done via commit-per-task instead.
