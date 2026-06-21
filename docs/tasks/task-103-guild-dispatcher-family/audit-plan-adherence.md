# Plan Audit (Plan Adherence) — task-103-guild-dispatcher-family

**Plan Path:** docs/tasks/task-103-guild-dispatcher-family/plan.md
**Audit Date:** 2026-06-18
**Branch:** task-103-guild-dispatcher-family
**Base Branch:** main
**Impl commit range:** 72cb42c6c..HEAD (13 commits)

## Executive Summary

Tasks 0–10 are faithfully implemented and grounded. All four packet-audit gates exit 0
(dispatcher-lint, matrix --check, fname-doc --check, operations --check). The three
changed Go modules (libs/atlas-packet, tools/packet-audit, services/atlas-channel) build,
vet, and test clean. Every guild + BBS arm is ✅ in STATUS.md across the supported versions,
with ⬜ only for genuinely version-absent cells (jms BBS, jms NEW_YEAR). The catch-all
structs and caller-selectable selectors are gone; byte fixtures are real assertions with
IDA citations. All 7 controller-approved deviations are present and sound. One additional,
sound, lint-clean deviation was found that was NOT in the supplied approved list (Info 0x1A
kept literal) — flagged for confirmation, not a failure. Tasks 11 (live config) and 12 (PR)
are correctly absent (controller-owned). **Verdict: PASS** (with one deviation to confirm).

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 0 | Baseline green | DONE | All 4 gates exit 0; 3 modules build/vet/test clean (verified live) |
| 1 | Enumerate switches; guild.yaml + guild_bbs.yaml | DONE | docs/packets/dispatchers/guild.yaml + guild_bbs.yaml exist; context.md §10 enumerated arm table with per-version IDA addresses (OnGuildResult v83 0xa37490 … jms 0xb22518) |
| 2 | Split catch-alls into discrete structs | DONE | operation.go: no `type ErrorMessage`/`ErrorMessageWithTarget` remain (grep EXIT=1). Discrete structs + NewXxx ctors; mode-only + target-bearing arms. operation_test.go: 174 verify markers, real `bytes.Equal(got, want)` assertions (TestModeOnlyErrorArms, TestTargetBearingInviteErrors lines 287–345) |
| 3 | Per-mode fixed-key bodies; remove selectors | DONE* | operation_body.go: `GuildErrorBody`/`GuildErrorBody2` removed (grep shows only a descriptive comment in channel consumer). One `WithResolvedCode("operations", GuildOperation*, …)` body func per arm (lines 59–272). *Deviation: GuildInfoBody kept 0x1A literal — see Deviations §A |
| 4 | BBS clientbound config-driven | DONE (approved deviation #4) | bbs.go: mode injected via ctor (`NewBBSThreadList(mode byte, …)`), no `mode:0x` literal; bbs_body.go passes fixed package consts `clientbound.GuildBBSMode*` (literal-mode kept, documented: no template registers a GuildBBS operations table). +BBSEntryNotFound added |
| 5 | Migrate atlas-channel call sites | DONE | guild/consumer.go: `guildErrorBodies` dispatch map (15 entries) + drop-on-unmapped replaces dynamic `GuildErrorBody(errCode)`. invite/consumer.go:181 → `GuildInviteDeniedBody`. socket/writer/guild_bbs.go → `GuildBBSThreadListBody`/`GuildBBSThreadBody` |
| 6 | AgreementResponse codec | DONE (approved deviation #1) | operation_agreement_response.go: codec UNCHANGED (`Encode4(unk)+Encode1(agreed)`), comment cites v83@0x530666/v87@0x557e6e/v95@0x52d780/jms@0x56da47 = byte-correct, "no wire change made." run.go:1554 + STATUS row 823 ✅ all 5 |
| 7 | run.go per-mode #-entries; remove phantom roots | DONE | run.go: 30+ `OnGuildResult#<Arm>` entries (1371–1494), 3 BBS `#`-entries (1527–1535). No bare `OnGuildResult`/`OnGuildBBSPacket` root case (grep EXIT=1). Duplicate `#AgreementResponse` clientbound alias removed (1577) |
| 8 | Serverbound v83/v87 + v84 fold; fixtures | DONE (approved deviations #2,#3,#7) | Serverbound rows 823/829/830/831 ✅ all 5. v84 fixtures cite REAL IDB `@0xa82e2b` (not folded). Invite gate widened to `MajorAtLeast(84)` w/ IDA citation (operation.go:769,786). gms_84 GuildBBSHandle 0x9B→0x9F (template diff) |
| 9 | Reconcile seed templates; regenerate matrix | DONE | operations --check exit 0; matrix --check exit 0; STATUS.md/status.json regenerated (toolSha commit fb02434e0). All guild/BBS rows ✅ / ⬜ |
| 10 | De-baseline + full gate sweep | DONE | dispatcher-lint-baseline.yaml: guild removed (only OnPartyResult + OnFriendResult remain). All 4 gates exit 0. Modules clean |
| 11 | Live config runbook | NOT DONE (intentional) | live-config-runbook.md absent — controller-owned per audit scope |
| 12 | Code review + PR | NOT DONE (intentional) | No PR for branch — controller-owned per audit scope |

**Completion Rate:** 11/11 in-scope tasks (Tasks 0–10) (100%)
**Skipped without approval:** 0
**Partial:** 0

## Controller-Approved Deviations — all verified sound

1. **AgreementResponse no codec change** — VERIFIED. Codec retains `unk uint32`; struct
   comment (operation_agreement_response.go:12–25) cites per-version IDA showing
   `Encode1(op)+Encode4(unk)+Encode1(agreed)`; the prior "extra unk" verdict marked STALE.
   run.go:1554 comment matches.
2. **v84 real IDB** — VERIFIED. v84 clientbound fixtures cite `ida=0xa82e2b`; agreement
   v84 comment cites `@0x53c8cd`. context.md §10 F2 records "v84 confirmed from the LIVE
   v84 IDB (port 13337), not folded." (Minor cosmetic nit: agreement v84 marker line uses
   `ida=0x0` while the adjacent comment gives the real `@0x53c8cd` — citation present, just
   not on the marker line.)
3. **Invite v84 gate 87→84** — VERIFIED. operation.go:769,786 `(t.IsRegion("GMS") &&
   t.MajorAtLeast(84)) || JMS`, IDA-cited (v84 case 5 @0xa82e2b reads trailing ints).
4. **BBS literal-mode** — VERIFIED. bbs_body.go documents version-stable modes + no
   template registers a GuildBBS operations table; dispatcher-lint clean.
5. **Unkeyed v95-only arms (0x4B, 0x52, NPCsay) omitted** — VERIFIED. context.md §10 F7
   flags them as no-Atlas-key / stop-and-ask; no struct invented; not in operation_body.go
   const set; STATUS shows no dangling rows for them.
6. **Foreign broadcasts + NEW_YEAR v84 added** — VERIFIED. STATUS rows 269/270
   (GUILD_NAME_CHANGED / GUILD_MARK_CHANGED) ✅ all 5; row 667 (NEW_YEAR_CARD_REQUEST)
   ✅ v83/v84/v87/v95, ⬜ jms. Files emblem_changed_foreign.go / name_changed_foreign.go +
   tests exist.
7. **gms_84 BBS 0x9B→0x9F + registry realign** — VERIFIED & grounded against the matrix:
   STATUS row 662 BBS_OPERATION serverbound shows v83=0x9B, v84=0x9F, v87=0xA3, v95=0xB3 —
   the +4 step from v83→v84 matches the v84→v87 +4 step, i.e. the task-100 reshift pattern.
   On `main` the handler was the unshifted 0x9B (the carryover bug). Does not regress
   task-100 (the reshift is in the same direction as its neighbors).

## Additional Deviation Found (NOT in the supplied approved list) — flag for confirmation

### A. GuildInfoBody / Info 0x1A kept literal (plan Task 3 Step 3 not literally followed)

- **Plan text:** Task 3 Step 3 directed folding `GuildInfoBody` into
  `WithResolvedCode("operations", GuildOperationInfo, …)`, making `Info` take `mode byte`
  and adding a `GuildOperationInfo` key (or stop-and-note if the table has no Info key).
- **What landed:** `clientbound/info.go:70` still writes `WriteByte(0x1A)` literal;
  `GuildInfoBody` (operation_body.go:283) calls `NewInfo(...).Encode` directly — no
  `WithResolvedCode`, no `GuildOperationInfo` const.
- **Justification present in code** (operation_body.go:276–282): "Info is NOT one of the
  OnGuildResult dispatcher arms (it has no operations key in guild.yaml — it is the separate
  GUILDDATA::Decode path)." Confirmed: no `INFO`/`GUILD_INFO` key exists in any seed
  template operations table, so a `WithResolvedCode` on it would resolve to nothing (the
  bug_operations_mode_tables_missing trap). run.go:1494 documents the wire @
  GUILDDATA::Decode@0x4fb760.
- **Assessment:** Sound, grounded, and the same disposition class as approved deviation #4
  (literal mode kept where no operations table registers the key). dispatcher-lint passes
  (Info is not a `#`-mapped error/notice arm subject to INV-2/INV-3). This is the plan's own
  documented fallback branch ("if the table has no Info key … stop-and-note"), exercised as
  keep-literal rather than add-key. **Not a failure**, but it was not in the deviation list
  handed to this audit — surfacing it so the controller can confirm the keep-literal choice
  (vs. adding a GUILD_INFO operations key) is the intended one.

## Gate & Build/Test Results (run live during audit)

| Gate / Module | Result | Notes |
|---|---|---|
| packet-audit dispatcher-lint | PASS (exit 0) | clean; only OnFriendResult + OnPartyResult baselined; guild absent |
| packet-audit matrix --check | PASS (exit 0) | no orphan/dangling/stale/drift |
| packet-audit fname-doc --check | PASS (exit 0) | OK (212 structs w/o report carry no fname) |
| packet-audit operations --check | PASS (exit 0) | OK (2 absent-writer notes: jms CharacterStatusMessage/NoteOperation — unrelated to guild) |
| libs/atlas-packet (guild) | PASS | go build + vet clean; go test ./guild/... ok |
| tools/packet-audit | PASS | go build clean; go test ./... all ok |
| services/atlas-channel | PASS | go build + vet clean; go test ./... all ok (from atlas.com/channel) |
| STATUS guild/bbs/agreement ❌/🟡 grep | PASS | `grep guild|bbs|agreement | grep ❌|🟡` returns nothing (EXIT=1) |
| TODO/stub/501 in changed Go | PASS | 2 TODO hits are PRE-EXISTING (invite/consumer.go:154 from commit 2c43c0e9e; run.go:2201 OnTournamentSetPrize) — neither touched in 72cb42c6c..HEAD |

NOTE: docker buildx bake atlas-channel and `go test -race` were not re-run in this read-only
audit pass (non-race `go test` is clean; bake is a controller PR-time gate). No go.mod was
added/removed for a new lib, so a missing `COPY libs/...` is not a risk introduced here.

## Especially-Scrutinized Items — results

- **Every guild+BBS arm ✅ across 5 versions / version-absent = ⬜:** CONFIRMED. STATUS
  rows 80 (BBS ✅×4, jms ⬜), 85 (GuildOperation ✅×5), 625/667 (serverbound ✅, NEW_YEAR
  jms ⬜), 823–831 (per-struct serverbound ✅×5), 269/270 (foreign ✅×5).
- **Real byte fixtures w/ markers + IDA citations (not mode-byte-only false pass):**
  CONFIRMED. TestModeOnlyErrorArms / TestTargetBearingInviteErrors assert full bodies via
  `bytes.Equal`; 174 markers in operation_test.go; serverbound round-trips per version.
- **GuildErrorBody/GuildErrorBody2 selectors gone + guildErrorBodies map:** CONFIRMED.
  Selectors removed; explicit 15-entry map with unmapped→Warn+drop (guild/consumer.go:139–171).
- **No TODO/stub/501 landed:** CONFIRMED (both TODO hits pre-existing, neither guild-related).

## Overall Assessment

- **Plan Adherence:** FULL (Tasks 0–10), with one sound non-listed deviation (Info 0x1A
  literal) to confirm.
- **Recommendation:** READY_TO_MERGE pending controller sign-off on Deviation A and the
  controller-owned Tasks 11–12.

## Action Items

1. Confirm Deviation A (Info 0x1A kept literal vs. adding a GUILD_INFO operations key) is the
   intended disposition. If keep-literal is accepted, optionally tighten the plan's Task 3
   Step 3 wording to record it (no code change needed).
2. (Cosmetic) AgreementResponse v84 verify marker carries `ida=0x0` while the adjacent
   comment has the real `@0x53c8cd`; consider moving the address onto the marker line for
   consistency with the other v84 markers (`ida=0xa82e2b`).
3. Controller: execute Task 11 (live config runbook) and Task 12 (review + PR) — out of audit
   scope, correctly absent.
