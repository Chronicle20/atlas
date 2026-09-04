# Plan Audit — task-290-cosmic-map-action-parity (Tasks C5-C15, plus BC and SP)

**Plan Path:** docs/tasks/task-290-cosmic-map-action-parity/plan-c2.md
**Audit Date:** 2026-09-03
**Branch:** task-290-cosmic-map-action-parity
**Base Branch:** main (merge base 9613e7259)
**Scope:** Tasks C5-C15 of plan-c2.md, plus the two out-of-band tasks tracked in the same
workspace: BC (spawn_npc broadcast wiring, USER RULING #4) and SP (SpawnPoint predicate
widen, USER RULING #3). C1-C4, C16-C23, and Plan A/Plan B are out of scope for this shard
(already covered by audit.md / audit-final.md).

## Executive Summary

All 11 in-scope plan tasks (C5-C15) plus both out-of-band tasks (BC, SP) landed on the
branch, each as its own commit(s), each independently reviewed by a `task-reviewer` agent
per the controller ledger, with zero blocking findings across every review. Build and test
re-verification for this audit (independent of the ledger's own gate runs) is clean across
all six affected modules (atlas-maps, atlas-channel, atlas-saga-orchestrator,
atlas-map-actions, atlas-query-aggregator, libs/atlas-saga): `go build ./...` and
`go test ./... -count=1` pass with no failures, and `tools/catalog-lint` exits 0 against the
full `deploy/seed` tree. Two tasks (C9, C14) split into out-of-plan sub-tasks
(C9b/C9c-abandoned/C9d; C14 continuation) under explicit user rulings, and both converged to
a fully reviewed, working end state — this is documented mid-flight, not silently dropped.
No task in this range is SKIPPED or unresolved; three carry disclosed non-blocking gaps
(BC's rx0/rx1/facing/cy placeholder and its SpawnForSelf resync gap; C12/SP's fix already
lands the flagged parity gap) that were explicitly raised to and ruled on by the user rather
than hidden.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| C5 | G6 seeds — `925040100` and `910510000` | DONE | Commits 4278bded9..b58fe439a. Reviewed APPROVED, 0/0/0. Reviewer confirmed 11 roots x 2 maps byte-identical, jobId band partition gap-free, first-match-wins at script/processor.go:144-169 (progress.md:19-22). |
| C6 | G7 seed — `pepeking_effect` | DONE | Commit da77df6ad. Reviewed APPROVED, 0/1/0. Non-blocking: `spawnIfAbsent="true"` deviates from Cosmic's unconditional spawn but is brief-mandated by PRD FR-2.2, reasoned in commit body (progress.md:23-27). |
| C7 | `play_sound` end to end (G8, live part) | DONE | Commits da77df6ad..40becf6ff. Reviewed APPROVED, 0/2/0. Two files outside the brief's Files inventory (producer.go, kafka.go) were required to compile and ruled in-scope as under-listed mirror pairs, not scope creep (progress.md:29-58). Non-blocking carry-forwards are pre-existing, not introduced here. |
| C8 | Two convertible cannon-tutorial seeds (G8, live part) | DONE | Commit f8f4cb815. Reviewed APPROVED (corrected to 0/0/0 after the reviewer located the Cosmic checkout and confirmed `playSound(...)` is literally the first statement in both scripts) (progress.md:59-70). |
| C9 | `change_music`/`boat_effect` end to end (G1c) | DONE | Commit b341a8775 (core action). Reviewed APPROVED_WITH_FINDINGS, 0 blocking / 1 non-blocking. The non-blocking finding (7 tenant/writer cells missing → sentinel-99 crash risk) was escalated to the user and resolved via C9b (derive real opcodes, 4/5 templates) + C9d (runtime guard for the one genuine external blocker, gms_12_1 — no v12 IDB/client binary exists). Final state: catalog-lint 0 issues, `gms-12-1-dock-arrival-seed-exclusion.md` documents the guard. See "Out-of-band sub-tasks" below. |
| C10 | `transportInTransit` aggregator condition (G1b) | DONE | Commit c3d74493e. Reviewed APPROVED, 0/1/0. Reviewer independently confirmed `state == "in_transit"` literal comparison (not the rejected `!open_entry` predicate) and traced Cosmic `Boats.js` docked semantics directly (progress.md:152-164). |
| C11 | Dock-arrival seeds `200090000`/`200090010` (G1c) | DONE | Commit 61307f074. Reviewed APPROVED, 0/0/1 not-evaluable (runtime packet delivery deferred to C9's tests, correctly out of scope for a seed-only commit). Pairwise-diffed all 11 copies against the gms/83_1 source; zero diffs (progress.md:178-190). |
| C12 | `warp_to_map` saga action (G1a) | DONE | Commit 32ac1aa1d. Reviewed APPROVED_WITH_FINDINGS, 0 blocking. Both controller-flagged risks (spawn-point-vs-portal semantics; `transactionId` arg ordering) independently confirmed against Cosmic source and sibling code. Raised a genuine pre-existing parity gap (SpawnPoint predicate too narrow) as a carry-forward, which the user ruled to fix — see task SP below (progress.md:342-361). |
| C13 | Twelve transport catch-up seeds (G1a/G1b) | DONE | Commit cc64b0358, 132 seed files across 11 roots. Reviewed APPROVED, 0/0/0. Reviewer independently verified all 12 scriptName→toMap pairs, md5sum-matched all 132 files, traced `params.mapId` to the executor's actual `map[string]string`+`ParseUint` handling (progress.md:363-373). |
| C14 | `spawn_npc` field NPC registry in atlas-maps (G2) | DONE | Commits f03fd8a3e, c35e9e446, ec4b3b7ee (initial pass PARTIAL at the 126-tool-call cap, closed by a continuation brief). Full range cc64b0358..ec4b3b7ee reviewed as one task, APPROVED, 0 blocking / 2 non-blocking (both pre-existing patterns, not introduced here). Four-module sweep clean: atlas-maps 320/320, libs/atlas-saga 69/69, saga-orchestrator 822/822, atlas-map-actions 108/108 (progress.md:388-448). |
| C15 | Five explorer-route NPC seeds (G2) | DONE | Commit 541fdc71c, 55 seed files (5 docs x 11 roots) + catalog-lint spawn-guard extension to `spawn_npc`. Reviewed APPROVED, 0 blocking / 1 informational. Reviewer mutation-tested the new lint rule (reverted it, confirmed the fixture genuinely fails, restored) rather than trusting it existence-only (progress.md:449-459, 512-521). |
| BC | Broadcast `spawn_npc` placements to clients (out-of-band, USER RULING #4) | DONE | Commit c26bdf00a. Reviewed APPROVED_WITH_FINDINGS, 0 blocking / 3 non-blocking / 1 not-evaluable. See "Disclosed gaps" below — both concerns were explicitly disclosed by the implementer, confirmed by the reviewer against real code, and correctly scoped out of this brief's Files list rather than hidden. |
| SP | Widen `SpawnPoint` portal predicate to types 0 and 1 (out-of-band, USER RULING #3) | DONE | Commit 2e50db2ab. Reviewed APPROVED, 0/0/0. Cosmic semantics confirmed from source (`MapleMap.java:2532-2540`); blast radius independently traced to a single production consumer chain; new `data/portal/model_test.go` added (progress.md:462-476, 553-559). |

**Completion Rate:** 13/13 tasks in scope (100%) — DONE
**Skipped without approval:** 0
**Partial implementations:** 0 (C9's initial commit and C14's initial pass were transiently
PARTIAL mid-session but both closed to DONE within the same body of work, tracked openly in
the ledger, not left incomplete)

## Skipped / Deferred Tasks

None. Every task in this range reached a fully reviewed DONE state. Two tasks generated
disclosed, non-blocking gaps that were surfaced to the user rather than silently accepted or
hidden — recorded here for completeness, not as audit failures:

### Out-of-band sub-tasks under C9 (change_music/boat_effect)

C9's review flagged a genuine pre-existing writer-binding gap (7 tenant×writer cells
missing → `libs/atlas-packet/resolve.go` sentinel-99 "will likely cause a client crash").
The user ruled to derive real opcodes in-branch. This produced:
- **C9b** (out-of-plan, user-authorised): derived and landed real ContiMove/FieldEffect
  opcodes for 4 of 5 template gaps from client binaries. **gms_12_1 is a genuine external
  blocker** — no v12 IDB session, no client, no registry entry exists anywhere in the repo
  or its ~20 prior gms_12-exclusion precedents. Nothing was guessed or invented.
- **C9c** (abandoned): tried to delete the two gms_12_1 seed documents; blocked by
  `tools/catalog-lint`'s byte-identical replication invariant, which has no exemption
  mechanism. Correctly reverted rather than adding an ad hoc exemption.
- **C9d** (out-of-plan, user-ruled, replaces C9c): added a runtime guard in
  `atlas-channel/kafka/consumer/system_message/consumer.go` that skips
  `change_music`/`boat_effect` on any tenant with no writer binding (keyed off the actual
  binding data, no `gms_12_1` literal), logging a skip instead of emitting sentinel-99.
  Added that package's first handler test file, pinning both the skip path and the
  unchanged-write path for bound tenants. Reviewed APPROVED, 0 blocking.

Net effect: `change_music`/`boat_effect` work correctly on 10 of 11 tenants; on `gms_12_1`
they are a logged no-op instead of a crash-risk packet. This is the correct, fully-landed
resolution of a real gap — not a skipped task.

### BC's disclosed non-blocking gaps (reviewed, not blocking)

1. **rx0/rx1/facing/cy placeholder values** — Cosmic's `spawnNpc` also sets facing and a
   walk-range on the live NPC, but no source data for these reaches `spawn_npc`'s saga
   payload (confirmed: `map/npc.Model` only carries `NpcId`/`X`/`Y`/`Fh`, and C14's plan
   never threads them through). `ScriptedNpcSpawn` substitutes `Y`→`cy`, `X`→both
   `rx0`/`rx1` (stationary, non-roaming), `0`→facing — disclosed in three places (two doc
   comments plus the report), not guessed at. Reviewer confirmed this is a real,
   conservative-but-visible divergence, not a forbidden placeholder. **Impact:** any
   `spawn_npc`-placed NPC will render with facing 0 and zero walk range regardless of
   Cosmic's real value.
2. **No `SpawnForSelf` resync** — a character entering a field *after* an NPC was already
   placed via `spawn_npc` will not see it; the map-enter fast path has no `map/npc` REST
   client to re-sync against. Confirmed by the reviewer at the code level (no such call
   exists in `SpawnForSelf`, and the `SpawnIfAbsent`-suppressed re-trigger path emits zero
   Kafka events). Explicitly out of BC's Files list; the reviewer's own not-evaluable notes
   that the actual C15 seed documents were not read to confirm how often they re-trigger.
   **Impact, per the reviewer:** C15/G2 seeds are functional for the first entrant to a
   field only, until a follow-up resync task lands.

Both gaps are non-blocking per the reviewer's verdict (correctly scoped out of the BC
brief's Files list, not silently dropped) but both are real functional facts that should be
tracked as follow-up work before calling G2 fully complete end-to-end. Note: the branch's
later history (`19ffc9a4b`, "resync scripted NPCs to a character entering the field
(task-BC2)") appears to address gap 2, but that commit is outside this shard's C5-C15/BC/SP
scope and was not audited here.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-maps | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` all `ok` (map/npc, map/monster, etc.), no FAIL lines. |
| atlas-channel | PASS | PASS | Clean build; all touched packages (`kafka/consumer/map`, `kafka/consumer/system_message`) pass. |
| atlas-saga-orchestrator | PASS | PASS | Clean build; `saga`, `saga/mock`, `transport`, `validation/mock` all `ok`. |
| atlas-map-actions | PASS | PASS | Clean build; `atlas-map-actions`, `script` packages `ok`. |
| atlas-query-aggregator | PASS | PASS | Clean build; `transport`, `quest`, `skill`, etc. all `ok`. |
| libs/atlas-saga | PASS | PASS | Clean build; module test `ok`. |
| tools/catalog-lint | PASS | — | `GOWORK=off go run . ../../deploy/seed` exit 0 over the full seed tree (includes all C5/C6/C8/C11/C13/C15 seed additions). |

All results independently re-run for this audit (not merely trusted from the ledger). No
FAIL lines found in any `go test` output across the six modules.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this shard's scope; final merge readiness depends
  on the other shards' audits and the flagless whole-branch `tools/verify.sh` run, which is
  outside this shard's scope per the review protocol)

## Action Items

1. None blocking within C5-C15/BC/SP. The two BC-disclosed gaps (NPC facing/walk-range
   placeholder values; pre-BC2 `SpawnForSelf` resync) were already ruled on by the user as
   accepted scope boundaries for the BC commit itself; confirm at final whole-branch review
   that the later `task-BC2` commit (`19ffc9a4b`, outside this shard) actually closes gap 2
   before declaring G2 end-to-end complete.
2. No code changes required in this range. The `gms_12_1` change_music/boat_effect no-op is
   a deliberate, reviewed, user-ruled resolution to a genuine external blocker (no v12
   client/IDB exists) and should not be revisited unless a v12 client surfaces.
