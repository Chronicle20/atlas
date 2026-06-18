# Plan Audit — task-093-mystic-door (plan-reconciliation.md)

**Plan Path:** docs/tasks/task-093-mystic-door/plan-reconciliation.md
**Audit Date:** 2026-06-18
**Branch:** task-093-mystic-door
**Base Branch:** main
**Commit range audited:** 7abb70e42..6cefdc004

## Executive Summary

All 5 plan tasks plus the post-final-review fix (commit 6cefdc004) are faithfully
implemented with the prescribed behaviors, file:line evidence, and non-vacuous tests.
No `// TODO`, stub, 501, or deferred work was found in the landed commits. `go vet ./...`
and `go test -race ./...` are clean in all three changed modules (atlas-doors,
atlas-parties, atlas-channel). One intentional, behavior-preserving deviation from the
plan text (the `handleSlotChanged` guard) is noted below and does not change behavior.
**Overall adherence: PASS.**

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-parties: DISBAND carries departing leader (capture before RemoveMember) | DONE | `party/processor.go:429-433` captures `formerMembers` before `Update`/`RemoveMember`; `:467` emits `disbandEventProvider(..., formerMembers)`. Test `TestLeaderLeaveDisbandEventIncludesLeader` (`processor_test.go:344-385`) asserts Members==[1,5], passes. |
| 2 | atlas-doors: `ReconcileParty` engine + contract behaviors + test suite | DONE | `door/reconcile.go:17-167` — adopt (no owner area re-send: `reconcile.go:100-104`), reslot (`:82-89`), drop-to-solo + cross-removal (`:117-134`), joiner/leaver visibility (`:136-167`), idempotent (no-op branches `:38-39`, `:83-84`), orphan self-heal (`:38` membership-driven), slot ≤ 5 via `ComputeSlot` (`:79`). Test suite `reconcile_test.go` — 7 tests incl. `TestReconcileNeverEmitsSlotAbove5`. |
| 3 | atlas-doors: 5 handlers rewired to `ReconcileParty`; old methods + ReslotParty deleted; obsolete tests removed | DONE | `kafka/consumer/party/consumer.go:75-157` all 5 handlers call `enginedoor.ReconcileParty`. `reslot.go`/`reslot_test.go` deleted (confirmed absent). No references to `JoinPartyDoor`/`LeavePartyDoor`/`DisbandPartyDoors`/`ShowPartyDoorsToCharacter`/`HidePartyDoorsFromCharacter`/`ReslotParty` remain. 3 obsolete tests removed (`TestLeavePartyDoor`/`TestJoinPartyDoor`/`TestDisbandPartyDoors` absent). processor.go -191 lines, processor_test.go -170 lines. |
| 4 | atlas-channel: `handleSlotChanged` reconciles party town-portal array; `SlotChangedBody.AreaX/AreaY` both sides | DONE | channel `consumer.go:289-298` clear-old/set-new via `announceTownPortalToParty`. `AreaX/AreaY` added on channel `kafka.go:80-81` and doors `kafka.go:81-82`; producer `producer.go:50-51` populates `m.AreaX()/m.AreaY()`. Test `TestHandleSlotChanged_PartyTownPortalReconciled` (`consumer_test.go:340-393`) asserts clear(slot0)+set(slot1, AreaX=300/AreaY=400), passes. |
| 5 | Verification gate (vet/test/build/bake) | DONE (re-verified here) | `go vet ./...` clean and `go test -race ./...` green in atlas-doors, atlas-parties, atlas-channel (run during this audit). redis-key-guard / docker bake noted environmental per task description; not re-run here. |
| + | Post-final-review fix (6cefdc004): adopt emits OldSlot==NewSlot | DONE | `reconcile.go:106-110` emits `slotChangedEventProvider(n, n.Slot())` (OldSlot==NewSlot) with explanatory comment. Regression test `TestReconcileAdoptDoesNotClearAnotherMembersSlot` (`reconcile_test.go:209-243`) asserts every emitted SLOT_CHANGED has OldSlot==NewSlot. |

**Completion Rate:** 5/5 tasks + 1 follow-up fix (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None.

## Deviations from Plan Text (non-blocking)

1. **`handleSlotChanged` guard simplified.** Plan Task 4 Step 5 specified
   `if e.ForCharacterId == 0 && e.PartyId != 0`. The implementation
   (`consumer.go:294`, commit 5c1a839a5) uses only `if e.ForCharacterId == 0`.
   This is behavior-preserving: `announceTownPortalToParty` early-returns when
   `partyId == 0` (`consumer.go` party-announce body, `if partyId == 0 { ... return }`),
   so a solo (partyId==0) reslot announces nothing. Commit message documents the
   intent ("align handleSlotChanged party guard with siblings"). The pre-existing
   solo test was re-pointed to `PartyId: 0` to isolate the solo path. No functional gap.

## Build & Test Results

| Service | Vet | Tests (-race) | Notes |
|---------|-----|---------------|-------|
| atlas-doors | PASS | PASS | door pkg incl. 7 reconcile tests; reslot tests removed |
| atlas-parties | PASS | PASS | party pkg incl. disband-includes-leader test |
| atlas-channel | PASS | PASS | door consumer incl. party-town-portal-reconciled test; full suite green |

## Stub / TODO / Vacuous-Test Scan

- No `TODO`/`FIXME`/`panic("not implemented")`/`501` in any changed file.
- All added tests assert concrete emitted-event sets / registry state — none vacuous.
  The slot-bound test (`TestReconcileNeverEmitsSlotAbove5`) and the adopt regression
  test decode the JSON body and assert real invariants.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. (Optional: re-run `tools/redis-key-guard.sh` and `docker buildx bake
atlas-doors atlas-parties atlas-channel` from the worktree root as the final gate
before PR, per the task-description note that redis-key-guard fails identically on
main under local GOWORK=off — environmental, not introduced here.)

---

# Plan Audit — task-093-mystic-door (Cosmic Visibility Rework)

**Audit Date:** 2026-06-18
**Branch:** task-093-mystic-door @ 763ea8623
**Base Branch:** main
**Governing design:** `design-cosmic-visibility.md` (supersedes the party-filtered
visibility + `ReconcileParty` area-door re-sends from `design-reconciliation.md`)

## Executive Summary

The five correctness goals of the cosmic-visibility rework are all implemented and
guarded by passing tests. The area door is now a plain map object broadcast to every
session in the map (no party filter); `ReconcileParty` (the area-door re-send / toggle
bug) is deleted; party reslot is town-only via `ReslotParty` and never re-sends the area
door; non-party entry yields `BLOCKED_MAP(6)`; in-party town render is owned by the
PARTYDATA refresh (`applyMemberDoor`) with no incremental TOWN_PORTAL on reslot; and the
Mystic Door buff is applied via the SOUL_ARROW stat and wired bidirectionally to the door
lifecycle. atlas-doors and atlas-channel both build/vet/test -race clean.

## Behavior Verification (the design's correctness goals)

| # | Behavior | Status | Evidence (file:line) |
|---|----------|--------|----------------------|
| 1 | Area door (SpawnDoor + area/town SpawnPortal) broadcast to ALL sessions in the map, no party filter | DONE | `kafka/consumer/door/consumer.go:75-98` (`broadcastDoorToEligible` enumerates every session via `ForSessionsInMap`, only `forCharacterId != 0` narrows to one char) + `:167-176` (handleCreated). Late-join: `kafka/consumer/map/consumer.go:247` (`ForEachInMap` → `spawnDoorsForSession:573-584`, no party gate). Test: `consumer_test.go:104` `TestHandleCreated_AreaSpawnDoorPlusPortal_TownPortalOnly` (asserts area SpawnDoor=1, SpawnPortal=1, town SpawnPortal=1, town SpawnDoor=0). |
| 2 | Area door NOT re-sent on party changes; `ReconcileParty` deleted; reslot is town-only | DONE | `ReconcileParty` removed repo-wide (only a removal-explaining comment remains at `consumer.go:220`). Replacement `door/reslot.go:20` `ReslotParty` calls only `p.Reslot` → `processor.go:203-222` emits `SLOT_CHANGED` (town-portal move), never CREATED/REMOVED for the area door. doors party consumer (`kafka/consumer/party/consumer.go:77-140`) routes every party event through `reslotForParty`/`ReslotParty` only. |
| 3 | Entry gate: owner/party → warp; non-party → BLOCKED_MAP(6), no warp | DONE | `socket/handler/mystic_door_enter.go:84-96` (`authorizeDoorEntry`), `:147-158` (`!authorized` → `BlockedMapBody(6)` + return, no warp; `onSide && authorized` → warp). Writer `socket/writer/blocked_map.go` → `libs/atlas-packet/field/clientbound/blocked_map.go` (packet-audit:verify across v83/v84/v87/v95/jms). Tests in `mystic_door_enter_test.go`. |
| 4 | In-party town render owned by PARTYDATA refresh (`applyMemberDoor`); `handleSlotChanged` emits no incremental TOWN_PORTAL | DONE | `kafka/consumer/party/consumer.go:57-72` (`applyMemberDoor` populates `aTownPortal` from each member's live door), used by `toPartyMembers:31-55` in PartyJoin/Left/Expel (`:230,:281,:383`). `kafka/consumer/door/consumer.go:280-292` (handleSlotChanged deliberately does NOT touch the party town-portal array). Test: `consumer_test.go:364` `TestHandleSlotChanged_PartyTownPortalNotTouched` (asserts `townPortals == 0`). |
| 5 | Recast sends area RemoveDoor (no toggle); buff via SOUL_ARROW stat; buff cancel ↔ door removal both directions | DONE | Recast remove: `consumer.go:222-231` (RemoveDoor to all map sessions, returns before town/buff clear on `RemoveReasonRecast`); doors-side `processor.go:95` emits REMOVED/RECAST on recast. Buff apply (cast): `skill/handler/mysticdoor/mysticdoor.go:66-72,125` (SOUL_ARROW statup, duration = door lifetime). Buff→door: `socket/handler/character_buff_cancel.go:23-24` (cancel → `door.Remove`). Door→buff: `consumer.go:248` (removal → `buff.Cancel`, RECAST excluded). Tests: `consumer_test.go:270` `TestHandleRemoved_Recast_AreaRemoveOnly`. |

## Build & Test Results

| Service | Build | Vet | Tests (`-race -count=1`) | Notes |
|---------|-------|-----|--------------------------|-------|
| atlas-doors | PASS | PASS | PASS | `ok atlas-doors`, `door`, `data/map`, `data/skill`, `party` |
| atlas-channel | PASS | PASS | PASS | full suite green incl. `kafka/consumer/door` (8.5s), `consumer/map`, `consumer/party`, `socket/handler`, `skill/handler/mysticdoor` |

## Notes / Non-blocking observations

- **redis-key-guard:** `GOWORK=off tools/redis-key-guard.sh` reports FAIL, but the failure
  is environmental: ~26 of the scanned modules (incl. atlas-channel AND atlas-doors) emit
  `./... matched no packages` (package resolution fails under GOWORK=off in this worktree),
  so the static pass never analyzed door code. atlas-doors' registry/allocator use only
  `libs/atlas-redis` wrapper types (`r.reg.Get`, `atlasredis.NewRegistry`/`NewKeyedSet`) —
  no raw keyed go-redis client calls. The prior audit section above records the same FAIL
  reproduces identically on `main`. Re-run via CI (proper GOWORK) before PR to get a clean
  signal; not a defect in this branch.
- **`reslot.go`/`reslot_test.go` retained:** the reconciliation plan called for deleting
  these, but the cosmic-visibility rework reinstated `ReslotParty` as the town-only reslot
  primitive (no area re-send). This is consistent with the governing design §4.1/§4.2
  (reslot stays town-only) — not a gap.
- **`docker buildx bake atlas-doors atlas-parties atlas-channel`** not run in this audit
  (no docker daemon invoked here); run from the worktree root before PR per CLAUDE.md.

## Overall Assessment

- **Plan Adherence (cosmic-visibility design):** FULL
- **Recommendation:** READY_TO_MERGE (after the standard `docker buildx bake` gate + a
  CI-run redis-key-guard for a clean signal)
