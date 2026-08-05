# Plan Audit — task-167-homing-beacon-bullseye (Plan Adherence)

**Plan Path:** docs/tasks/task-167-homing-beacon-bullseye/plan.md
**Audit Date:** 2026-08-05
**Branch:** task-167-homing-beacon-bullseye
**Base Branch:** main (24 commits, `main..HEAD`)

## Executive Summary

All 14 plan tasks were faithfully executed; 82/82 checkbox items map to real, verifiable
work, though the checkboxes in `plan.md` themselves were never ticked (`grep -c '\[x\]'`
= 0) — a documentation-hygiene gap, not a substance gap. Five deliberate divergences from
the plan's literal text are present, all owner-approved and all independently
re-verified by this audit against the resulting code and evidence docs: (1) Task 4's
`charstatus` package was built then folded into the pre-existing `characterstatus`
consumer (commits `f427f207d`→`48ffcdfc9`); (2) Task 7b corrected the v61 two-state group
to 6 members/88 bytes (commit `be0927c38`), making the empty-CTS length 106, not the
plan's stale 128; (3) Task 8's movement filter grew from 2 hypothesized branches to 5
version-specific branches (v84/v87/v92/v95/JMS), each cited against
`evidence/movement-filter.md`; (4) four pre-existing cancel fixtures were re-baselined
because they pinned the F1 bug Task 8 exists to fix; (5) v79/v84 two-state block sizes
remain genuinely `UNVERIFIED` (IDBs lack ctor/RTTI/vtable symbols across two passes),
with 110 bytes used as a bracketed inference — documented, not silently shipped. `go
build`/`go vet`/`go test -race` are clean in all three changed modules
(`services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel`,
`libs/atlas-packet`), `docker buildx bake atlas-buffs atlas-channel` both succeed,
and `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and
`tools/skill-job-id-guard.sh` are all clean. The one plan item genuinely not done is
Task 14 Step 6 (live v83-tenant acceptance) — it requires a running tenant and was not
performed; this is correctly a known gap, not a false completion claim.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-buffs no-expiry domain model | DONE | `buff/model.go:20` (`noExpiry` field), `:31-40` (`NoExpiry()`/`Expired()`), commit `711a75a4d`. Matches plan verbatim. |
| 2 | noExpiry through registry/processor/Kafka | DONE | `character/registry.go:70` (`Apply(..., noExpiry bool)`), `character/processor.go:21,48`, `kafka/message/character/kafka.go:49-112` (3 `NoExpiry` fields), `kafka/consumer/character/consumer.go:55` (reject-guard), commits `b975222c6`, `5c65f25d6`. |
| 3 | REST projection of no-expiry | DONE | `buff/rest.go:18,48` (`NoExpiry bool` + `Transform` wiring), commit `bcad01365`. |
| 4 | MAP_CHANGED consumer cancels the beacon | DONE (superseded implementation, documented) | Plan specified a new `kafka/consumer/charstatus` package; built as specified in `f427f207d`, then consolidated into the pre-existing `kafka/consumer/characterstatus` consumer (which already owned `EVENT_TOPIC_CHARACTER_STATUS`/`MAP_CHANGED` for berserk tracking) in `48ffcdfc9`. Final: `kafka/consumer/characterstatus/consumer.go:87-96` (`handleStatusEventMapChanged` calls `CancelByStatTypes` for `HOMING_BEACON` independently of the berserk `HandleTransfer` call, each error logged-and-continued). `charstatus` package no longer exists (`find … -path '*charstatus*'` returns nothing). Commit message explicitly documents the fold. |
| 5 | Populated GuidedBullet block (pre-95) | DONE | `NewGuidedBulletTemporaryStatWithOptions` at `model/character_temporary_stat.go:542` (evolved to take a `narrowTimeField` 4th param feeding Task 7b's v61 fix); `getBaseTemporaryStats` case `twoStateGuidedBullet` at `:1243-1253`; commit `bfb99bf33`. |
| 6 | v95 two-state group extension (closes Task 41b) | DONE | `twoStateStat.conditional` field + `twoStatePartyBooster`/`twoStateGuidedBullet` kinds at `:1005-1022`; v95 branch in `twoStateBaseStats` at `:1064-1073` (4 unconditional + PartyBooster/GuidedBullet conditional); `decodeBaseTemporaryStats` takes the decoded mask at `:1191-1221`; commit `b1502a53b`. |
| 7 | IDA — movement filter + two-state group, all 9 versions | DONE | `evidence/movement-filter.md` has all 9 sections (v61,72,79,83,84,87,92,JMS,v95) + a JMS provenance caveat; `evidence/two-state-group-per-version.md` has all 9 version sections plus a gms_12/gms_48 n/a section and a "read-style generalisation" correction section; commit `d0a7299e5` + follow-up `fa278d8be` (added a missing JMS v185 section, caught by the task's own coverage audit — see progress.md). |
| 7b | (not in original plan) v61 two-state group correction | DONE, owner-approved divergence | `isGmsV61` gate at `model/character_temporary_stat.go:1032-1034`; 6-member/88-byte group (`:1074-1083`); commit `be0927c38`. Independently adjudicated in `.superpowers/sdd/plan/task-7b-undead-bit-adjudication.md` — confirms raw bit 65 (Undead) is a provably inert hole for v61 across 3 IDA sites; the repo's compensating re-OR of the Undead bit in `EncodeMask`'s `isGmsV61` branch is the technically correct choice, not just test-preserving. |
| 8 | Accurate cancel masks + conditional movement byte | DONE, plan under-specified (5 branches not 2) | `CancelMask`/`EncodeCancelMask` at `:694-715`; `movementAffectingStatNames` at `:737-843` with 5 version-specific branches (v84, v87, v92, v95, JMS) each carrying an evidence citation and an explicit "why included despite being unresolved in that IDB" rationale (one-directional-failure argument: over-inclusion is inert on a length-delimited packet, omission desyncs); `buff_cancel.go` mask/byte gating; commits `be0927c38`(base)..`28a046b00` (comment-fix round). Four pre-existing fixtures were re-baselined (`TestBuffCancelV72/V79/V61/V48ByteFixture`) — reviewed per progress.md and independently re-derived correct by the code-review agent, `packet-audit:verify` markers left untouched. |
| 9 | atlas-channel noExpiry mirror + buff model | DONE | `kafka/message/buff/kafka.go:17` (`CommandTypeCancelByTypes`), `:42-56` (`NoExpiry`/`CancelByTypesCommandBody`), `character/buff/model.go:59,63` (`NoExpiry()`/`NewBuff` trailing param), `character/buff/rest.go:18,47`, `kafka/consumer/buff/consumer.go:99,161,246,452` (4 `NewBuff` call sites carrying the flag — more than the plan's predicted 2, per progress.md's Task 9 note); commit `5babe2cb9`. |
| 10 | CancelByTypes + ApplyNoExpiry buff commands | DONE | `character/buff/producer.go:42-89` (`ApplyNoExpiryCommandProvider`, `CancelByTypesCommandProvider`); `character/buff/processor.go:22,25,64,81` (interface + impl); commit `4f5831ffb`. Per progress.md, `CancelByTypes` already existed on `main` pre-task; only `ApplyNoExpiry` was net-new — agent checked `git diff main` rather than blindly re-adding. |
| 11 | Beacon mirror registry | DONE | `character/buff/beacon.go` (`BeaconEntry`, `BeaconMirror`, `GetBeaconMirror`, `Set`/`Clear`/`Get`) matches the plan's singleton-idiom spec verbatim; `beacon_test.go` present; commit `f5f7228ec`. Reviewer independently confirmed lock discipline (Get returns `BeaconEntry` by value, no map escape) per progress.md. |
| 12 | Local-give merge + foreign suppression | DONE | `kafka/consumer/buff/consumer.go:422` (`beaconChange`), `:434` (`isBeaconOnly`), `:451` (`mergeBeacon`); wiring at `:153` (`Set` on APPLIED), `:169` (merge into `localBs`), `:238` (`Clear` on EXPIRED); commit `46f79ec25`. Aliasing risk flagged and verified safe by review (bs len==cap==1 forces `mergeBeacon`'s append to reallocate). |
| 13 | Attack-handler hook | DONE | `socket/handler/character_attack_common.go:641-699` (`beaconApplyDeps`, `beaconTargetMonsterId`, `beaconTryApply`); wired at `:969-983` inside the `AttackTypeRanged && ai.SkillId() > 0` branch of `processAttack`; commit `320945690`. `tools/skill-job-id-guard.sh` confirmed clean re: the raw `sid != skill3.OutlawHomingBeaconId` comparison (5211006/5220011 not in task-187's divergences.csv, cited in-code at `:673-676`). |
| 14 | Full verification + acceptance checklist | PARTIAL | Steps 1-5 done (see Build & Test Results below and Skipped/Deferred section for the checklist-staleness items this audit re-adjudicated). Step 6 (live v83-tenant manual acceptance) was **not performed** — no live-test record exists anywhere in the task folder; this requires a running tenant and is correctly reported as outstanding, not silently marked done. Step 7 (commit checklist/doc updates) was effectively superseded by the JMS-section fix commit `fa278d8be`, but `plan.md`'s own checkboxes were never ticked. |

**Completion Rate:** 13/14 tasks fully DONE, 1/14 (Task 14) PARTIAL on one sub-step (live acceptance) — substance-wise this is ~99% complete (82 checklist steps; only the manual live-tenant walkthrough is outstanding).
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 14 Step 6 — live acceptance, requires infrastructure this audit does not have)

## Skipped / Deferred Tasks

### Task 14 Step 6 — Live acceptance (v83 tenant, manual)

Not performed. The plan explicitly scopes this as "manual — record results in the task
folder" and no such record exists (`grep -rn "Live acceptance"` across the task folder
turns up only the plan's own instruction text and PRD/design caveats, never a completed
walkthrough). This is a genuine gap against the plan's letter, but it is an infrastructure
dependency (a running v83 tenant + manual play-testing) that cannot be produced by static
code inspection, and the plan itself frames it as the terminal manual step. It should
block a "fully verified" claim but does not indicate any code defect — every other angle
(byte fixtures, unit tests, IDA evidence, build/test gates) is independently green.

### Task 14's own checklist staleness (self-identified, correctly adjudicated)

The plan's Step 4a coverage-audit checklist predates three of the divergences above and
is stale in three specific, checkable ways — all correctly caught and reported (not
silently passed) per `.superpowers/sdd/plan/progress.md`'s "Task 14 PLAN-CHECKLIST
DIVERGENCE" entry, which this audit independently re-verified:

1. **v61 length assertion.** The checklist implies every in-scope version's empty CTS
   is `16+2+110=128`. v61 is correctly `16+2+88=106` (confirmed live:
   `libs/atlas-packet/model/character_temporary_stat_test.go:366-367` asserts
   `16+2+88`). A literal reading of the checklist would flag this correct behavior as a
   failure.
2. **The `grep CreateContext` undercounts.** Running the plan's exact command
   (`grep -oE 'CreateContext\("[A-Z]+", [0-9]+' model/character_temporary_stat_test.go |
   sort -u`) returns only 3 literals (GMS 61, GMS 83, GMS 95) because the beacon
   fixtures are table-driven (`ctx := pt.CreateContext(v.region, v.major, 1)` inside a
   loop over struct literals). Verified directly: all 9 in-scope version names (`"GMS
   v61"` … `"JMS v185"`) appear as subtest-table entries at
   `character_temporary_stat_test.go:268-274,323-330`, and `go test ./model/ -run
   TestCTS -v` produces 73 passing subtests spanning all 9 versions.
3. **v79/v84 `UNVERIFIED` block sizes.** The checklist states "Any `UNVERIFIED` verdict
   blocks completion." `evidence/two-state-group-per-version.md` sections for v79
   (`## v79 — block sizes UNVERIFIED, not verified`, line 135) and v84 (line 194) are
   genuinely unresolved — the blocker is a documented, exhausted one (IDBs lack
   constructor/RTTI/vtable symbols for these two versions across two independent
   audit passes), and 110 bytes is used as an inference bracketed by the measured
   v72/v87/v92 values (all confirmed 110), not asserted as verified fact. This is a
   known, reported limitation rather than a silent pass — the evidence file states it
   plainly and the fixture comments do not claim byte-verification for these two
   versions' trailer contents beyond the length.

None of these three are code defects; they are places where the plan's own acceptance
checklist, written before the RE evidence existed, needed judgment to apply correctly —
and that judgment was applied and documented rather than mechanically followed into a
false failure or a silently-passed false success.

## Build & Test Results

| Service/Module | Build | Vet | Tests (`-race`) | Notes |
|---|---|---|---|---|
| `services/atlas-buffs/atlas.com/buffs` | PASS | PASS | PASS | All packages ok, including `kafka/consumer/characterstatus` (Task 4's final home). |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | All packages ok, including `character/buff`, `kafka/consumer/buff`, `socket/handler`. |
| `libs/atlas-packet` | PASS | PASS | PASS | 73 `TestCTS*` subtests pass across all 9 in-scope versions; no FAIL lines anywhere in full-module `go test -race ./...`. |
| `docker buildx bake atlas-buffs` | PASS | — | — | Image built and exported (`atlas-buffs:local`). |
| `docker buildx bake atlas-channel` | PASS | — | — | Image built and exported (`atlas-channel:local`). |
| `tools/redis-key-guard.sh` | — | — | — | Exit 0 (clean). |
| `tools/goroutine-guard.sh` | — | — | — | Exit 0 (clean). |
| `tools/skill-job-id-guard.sh` | — | — | — | Exit 0, "clean (14 divergent const(s) checked)" — covers the raw skill-id comparison in `beaconTryApply`. |

`tools/lint.sh --check` was intentionally NOT run per this audit's instructions (known
false-fail in this environment: no Node 22, `ui:node-missing`).

## FR Coverage Cross-Check (PRD)

Spot-checked every FR the PRD enumerates against code, not just the plan's task list, to
catch anything the plan itself might have under-specified:

- FR-1.1/1.3/1.5/1.6 (attack-pipeline hook, multi-monster last-wins, whiff no-op,
  pipeline-unaffected on failure): `beaconTargetMonsterId`/`beaconTryApply`,
  `character_attack_common.go:652-699`, exercised by
  `TestBeaconTargetMonsterId`/`TestBeaconTryApply`
  (`character_attack_common_test.go:347-434`) including a "cancel failure blocks apply"
  case.
- FR-2.2/2.3/2.6 (validation, reap-immunity, finite-buff regression): Task 1/2 tests
  `TestNewBuff_StillRejectsNonPositiveDuration`, `TestRegistry_GetExpiredKeepsNoExpiry`;
  full existing suite stays green.
- FR-3.2/3.4 (client-side clear on cancel, whole-character cancel paths):
  `TestBuffCancelBeaconOnlyV83`/`V95`, `TestRegistry_CancelAllRemovesNoExpiry`.
- FR-3.3 (kill does NOT cancel the lock): verified by absence — `grep -rl
  "HOMING_BEACON"` across both services' non-test Go files returns only
  `buff/model.go`, `kafka/consumer/buff/consumer.go`,
  `kafka/consumer/characterstatus/consumer.go` (MAP_CHANGED only), `beacon.go`, and
  `character_attack_common.go` — no monster-death consumer references the stat.
- FR-4.2/4.3 (byte-verified BuffGive, v95 conditional group): Task 5/6 fixtures.
- FR-4.4 (re-entry mid-lock carries the populated block): not a dedicated task item, but
  verified by inspection — `socket/writer/character_spawn.go:31-36`
  (`CharacterSpawnBody`) builds the full CTS from the character's fetched `[]buff.Model`
  via the same `AddStat` path Tasks 5/6/9 wired, so a no-expiry HOMING_BEACON buff
  surviving in atlas-buffs' registry automatically re-populates the GuidedBullet block on
  re-entry with no extra code needed — the design's layering makes this fall out for
  free rather than requiring a dedicated Task.
- FR-4.5 (foreign suppression): `isBeaconOnly` gate at
  `kafka/consumer/buff/consumer.go:169` and the foreign-encode regression test
  `TestCTSForeignEmptyV95StaysStatusQuo`.
- FR-4.6/4.7 (version-coverage honesty, byte-identical no-beacon encodes): covered by
  the Task 14 discussion above.
- FR-5.1 (Outlaw/Corsair skill ids handled identically): both ids gated together at
  `character_attack_common.go:679` (`sid != skill3.OutlawHomingBeaconId && sid !=
  skill3.CorsairBullseyeId`), single code path.

No FR was found unimplemented.

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (FULL on substance; one manual/infrastructure step
  outstanding, and plan.md's own checkboxes were never ticked despite the work being
  done and independently code-reviewed per-task via `.superpowers/sdd/plan/task-*-review.md`)
- **Recommendation:** NEEDS_REVIEW — not for code defects (none found), but because (a)
  Task 14 Step 6's live acceptance is genuinely outstanding and the plan treats it as a
  completion gate, and (b) `plan.md` checkboxes should be ticked (or the plan
  explicitly annotated with the 5 divergences) before calling the document itself
  "faithfully executed" in the literal sense, even though the underlying engineering is
  sound.

## Action Items

1. Perform (or explicitly and permanently defer with owner sign-off) Task 14 Step 6's
   live v83-tenant acceptance walkthrough; record the result in the task folder.
2. Tick the 82 checkboxes in `plan.md` (or add a short "Divergences from this plan" note
   at the top pointing to the 5 documented deviations) so the plan document itself
   reflects what actually shipped — currently 0/82 boxes are checked despite the work
   being done and reviewed.
3. No code changes required — all three modules build, vet, and test clean under
   `-race`, both mandatory Docker images bake successfully, and all three custom guard
   scripts (redis-key, goroutine, skill-job-id) pass.
