# Plan Audit — task-252-jukebox-cash-item

**Plan Path:** docs/tasks/task-252-jukebox-cash-item/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-252-jukebox-cash-item
**Base Branch:** main (merge base d17404dbc)

## Executive Summary

All 8 plan tasks were implemented faithfully and match the plan's prescribed
files, interfaces, and wire-level details almost verbatim. All four affected
Go modules (`libs/atlas-packet`, `libs/atlas-saga`, `atlas-saga-orchestrator`,
`atlas-maps`, `atlas-channel`) build cleanly and all tests pass. Every global
constraint was checked directly against code and evidence found: the stop
signal is a named `int32(-1)` constant asserted byte-exact (`ff ff ff ff`) in
a test, durations stay in milliseconds end to end, no `FieldEffect` BGM
packet is constructed anywhere in the jukebox code path, the arm is keyed on
cash-slot type 20, and `libs/atlas-packet/field/clientbound/play_jukebox.go`
is untouched. One post-plan fix commit (`6360d7538`) improves on the plan's
literal instructions by removing a dead store flagged by `ineffassign`; it is
a legitimate, in-scope correction, not a deviation of concern.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `ItemUseSongPlayer` serverbound codec | DONE | `libs/atlas-packet/cash/serverbound/item_use_song_player.go` and `_test.go` match the plan's struct, accessors, doc comment, and wire-order test table exactly (commit 650bf48db). |
| 2 | `play_jukebox` saga action in `libs/atlas-saga` | DONE | `PlayJukebox` const, `PlayJukeboxPayload`, unmarshal case, `otherActions` entry, and decode test all present at the plan's named locations (commit ba81811b9). |
| 3 | `PLAY_JUKEBOX` command in atlas-saga-orchestrator | DONE | `CommandTypePlayJukebox`/`PlayJukeboxCommandBody`, `PlayJukeboxCommandProvider`, `Processor.PlayJukebox`, `saga/model.go` aliases + unmarshal case, `event_acceptance.go` fire-and-forget entry, `handlePlayJukebox`, and both required tests (`TestHandlePlayJukebox_InvalidPayload`, `TestPlayJukeboxCommandProvider`) all present and matching the plan's field/error contract (commit 8fba7b65e). |
| 4 | atlas-maps jukebox registry, processor, REST model | DONE | `map/jukebox/registry.go` is a line-for-line transcription of the weather registry with `PlayerName` in place of `Message`; `processor.go`/`rest.go` match; all four named registry tests present and passing (commit 3278606dd). |
| 5 | atlas-maps jukebox command consumer, expiry sweep, REST resource | DONE | `producer.go`, `resource.go`, `consumer.go`'s `handlePlayJukeboxCommand` (with package-scoped `maxJukeboxDuration = 10*time.Minute` and millisecond conversion), `tasks/jukebox.go`'s `processExpiredJukebox`/`emitJukeboxEnd`, `main.go` registrations, and all named tests present (commit fab90abc4). |
| 6 | atlas-channel jukebox REST client | DONE | `jukebox/{rest,requests,processor}.go` and `jukebox/mock/processor.go` match the plan's transcription of the weather REST client; `requests_test.go` covers decode, instance-scoped path, and 404-as-error exactly as specified (commit 74d22004a). |
| 7 | atlas-channel type-20 item-use arm | DONE | `CashSlotItemTypeSongPlayer = CashSlotItemType(20)` and the two-step saga arm (`DestroyAsset` then `PlayJukebox`) are present in `character_cash_item_use.go`; all four named tests in `character_cash_item_use_jukebox_test.go` present, including the zero-length and unresolvable-character rejections (commit 1886ade90, refined by fix commit 6360d7538 — see Skipped/Deferred section below for the one intentional plan deviation). |
| 8 | atlas-channel jukebox broadcast and map-enter replay | DONE | `kafka/message/map/kafka.go` adds `JUKEBOX_START`/`JUKEBOX_END` + bodies; `consumer.go` adds `jukeboxStopItemId = int32(-1)`, `handleStatusEventJukeboxStart`/`End` (routed through the testable `doorAnnounce` seam, not a direct `session.Announce`), `announceActiveJukebox` extracted as a named function, and both handlers registered in `InitHandlers`. All seven named tests present, including the byte-exact `ff ff ff ff` assertion in `TestHandleStatusEventJukeboxEnd_BroadcastsExactlyMinusOne` (commit 720374ab5). |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None skipped. One intentional, in-scope deviation from the plan's literal
Step 3 code, made after Task 7 landed:

- **Commit `6360d7538` ("fix(channel): drop ineffectual updateTime
  assignment in jukebox arm")** — the plan's Task 7 Step 3 snippet included
  `if !updateTimeFirst { updateTime = sp.UpdateTime() }` after decoding the
  sub-body. This was a dead store: neither `DestroyAssetPayload` nor
  `PlayJukeboxPayload` ever consumes `updateTime` in this arm (unlike the
  kite/morph-coupon arms), and `ineffassign` flagged it. The fix removes the
  assignment and replaces it with a comment explaining that `Decode` still
  consumes the trailing wire field on GMS ≤ v84 regardless of whether the
  caller captures the return value. This does not change wire behavior,
  saga output, or test assertions — `character_cash_item_use_jukebox_test.go`
  is unchanged by the fix commit. No impact.

The plan's own explicitly-logged deferred item is confirmed as the only gap
of its kind: `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go`'s
`Get`/`GetActive` do not filter on `ExpiresAt` (only `GetExpired` does),
inherited verbatim from the weather registry template. This creates a
bounded (~1s, one sweep-tick) stale-read window between an entry's expiry and
the sweep task's `DeleteEntry` call. This was scoped as an accepted
inheritance from the weather pattern, not a new gap introduced by this
plan, and is not re-litigated here.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | All packages, including `cash/serverbound` (jukebox codec tests). |
| `libs/atlas-saga` | PASS | PASS | Single package, includes `TestUnmarshalPlayJukeboxStep`. |
| `atlas-saga-orchestrator` (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`) | PASS | PASS | `saga` and `map_command` packages both green, including the two jukebox-specific tests. |
| `atlas-maps` (`services/atlas-maps/atlas.com/maps`) | PASS | PASS | `map/jukebox`, `tasks`, `kafka/consumer/map`, `kafka/message/map` all green. |
| `atlas-channel` (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `jukebox`, `jukebox/mock`, `socket/handler`, `kafka/consumer/map` all green. |

No `atlas-ui` changes in this branch (confirmed via `git diff --stat
main...HEAD`), so no `npm run build`/`npm test` step applies.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the controller's own flagless
  `tools/verify.sh` run, per the plan's "Final gate" section — not run here
  per instruction, since a flagless run was already in flight in this
  worktree)

## Action Items

None. All plan tasks are complete, all global constraints hold with direct
evidence, and all module-local builds/tests pass. The controller should
confirm the in-flight flagless `tools/verify.sh` run exits 0 before
proceeding to code review / PR, per plan's Final Gate and CLAUDE.md's "Done
means verified" rule.
