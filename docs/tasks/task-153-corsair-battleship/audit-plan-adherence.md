# Plan Audit — task-153-corsair-battleship

**Plan Path:** docs/tasks/task-153-corsair-battleship/plan.md
**Audit Date:** 2026-07-28
**Branch:** task-153-corsair-battleship
**Base Branch:** main (range `f5aaced5ca7527f6d5e32d5cca59e1ab0f0020b7..HEAD`, 12 implementation commits + 3 pre-existing doc commits)

## Executive Summary

All 12 plan tasks are fully implemented with direct file:line evidence, including all 8 project-owner-approved deviations from the plan's literal text (lazy re-init via `InitIfMissingAndDecrBy`, best-effort `clearShipHP` on init failure, no-op go.mod edit, reused Redis client, wrapper-based buff hook, `shouldAnnounceGauge` predicate extraction, pre-completed v92 opcode derivation, and cancellation of the redundant code-review step). Every module (`libs/atlas-constants`, `libs/atlas-redis`, `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-configurations`) builds, vets, and passes `-race` tests cleanly. All four repo guards required by the plan (`redis-key-guard.sh`, `goroutine-guard.sh`, `template-opcode-order-guard.sh`, `lint.sh --check`) exit 0, and `docker buildx bake atlas-channel` succeeds. The plan's own "Acceptance criteria traceability" table was independently re-verified line by line against the code and holds up in every row. No silent skips, no unresolved TODOs, no scope narrowing found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-constants: `IsBattleshipMountSkill` classification | DONE | `libs/atlas-constants/skill/mount.go:35-36`; test `TestIsBattleshipMountSkill` at `mount_test.go:59`, table cases renamed at `:16`/`:44` per plan Step 1 |
| 2 | atlas-redis: `TenantCounter` atomic decrement-if-exists | DONE | `libs/atlas-redis/counter.go` — `Set`/`DecrByIfExists`/`Remove` all present; plus the **approved deviation** `InitIfMissingAndDecrBy` (`:83-97`) with its own Lua script and 4 dedicated tests (`counter_test.go:149-243`) |
| 3 | atlas-packet: `ResolveValue` uint32 config resolver | DONE | `libs/atlas-packet/resolve.go:103-133`; tests `TestResolveValueValid`/`TestResolveValueMisses` at `resolve_test.go:160,178`, with the R-10-corrected hex constant `0x1D7AE0` |
| 4 | atlas-channel: per-tenant writer-options registry | DONE | `<ch>/socket/writer/options_registry.go` (`RegisterTenantWriterOptions`/`TenantWriterOptions`/`EvictTenantWriterOptions`); wired in `main.go:395` (register) and `main.go:292` (evict) |
| 5 | atlas-channel: battleship ride mirror | DONE | `<ch>/battleship/mirror.go` (`GetRideMirror`, `Put`/`Get`/`Remove`/`EvictTenant`); eviction wired `main.go:293`; `InitRegistry` wired `main.go:191` |
| 6 | atlas-channel: battleship processor (HP pool, drain, break) | DONE | `<ch>/battleship/processor.go` — `ShipHP` version-gated formula (`:59-75`), `Drain` (`:167-232`), `breakShip` (`:234-262`) with corrected non-idempotency comment (**approved deviation 1**: lazy re-init uses `InitIfMissingAndDecrBy` at `:211`, not a bare `Set`); `go.mod` require/replace pre-existing (**approved deviation 3**, confirmed at `go.mod:11,97`, no edit needed) |
| 7 | atlas-channel: ride lifecycle hooks (buff events + session destroy) | DONE | `<ch>/kafka/consumer/buff/consumer.go` — `isBattleshipRide` (`:346`), APPLIED hook `:134-141`, EXPIRED hook `:179-181` via `newBattleshipProcessor` seam (**approved deviation 5**: dedicated wrapper, not an inline `IfPresentByCharacterId` callback edit); `<ch>/session/processor.go` `Destroy` calls `clearBattleshipOnDestroy` (`:411`, defined `:444-449`) |
| 8 | atlas-channel: cast path (carve-out, mount arm, cooldown rejection) | DONE | `common.go` `shouldApplyCastCooldown` (`:92-93`) + gate at `:150-157`; `mount.go` battleship arm (`:142-160`) including **approved deviation 2** (`clearShipHP` best-effort call in the `initShipHP` error branch, `:156-159`, tested by `TestHandleMountBattleshipInitFailureClearsStalePool` at `mount_test.go:397`); `character_skill_use.go` `battleshipCastBlocked` (`:200-202`) wired at `:81` |
| 9 | atlas-channel: damage drain + HP gauge announce | DONE | `character_damage.go` `Drain` call `:54`, gate `:55` via **approved deviation 6** `shouldAnnounceGauge` predicate (`:73-75`), `announceShipHpGauge`/`gaugeCooldownValue` (`:82-107`); R-5's corrected 29000 ceiling present in both `character_damage_test.go:32` and `processor_test.go:165,169` |
| 10 | atlas-channel: Cannon/Torpedo riding gate | DONE | `character_attack_common.go` `battleshipAttackPermitted` (`:913-919`), call site `:670`; test `TestBattleshipAttackPermitted` (`character_attack_battleship_gate_test.go:22`) |
| 11 | seed templates: config tables, v92 writers, missing cast/damage handlers | DONE | All 9 templates carry both options tables (script-verified below); v92's 5 writer entries and the 4 versions' cast/damage handler pairs verified opcode-exact against the plan's tables (gms_87 `0x5E`/`0x32`, gms_95 `0x67`/`0x34`, jms_185 `0x56`/`0x27`, gms_92 `0x66`/`0x35`, v92 writers `0x21/0x22/0xE3/0xE4/0x112`); `gms_12`/`gms_48` confirmed battleship-free; `tools/template-opcode-order-guard.sh` exits 0 |
| 12 | verification suite + live-tenant backfill runbook | DONE | `docs/tasks/task-153-corsair-battleship/backfill.md` (92 lines) created; all module test/vet/build clean (see Build & Test below); all 4 guards clean; `docker buildx bake atlas-channel` succeeds; Step 6 (`superpowers:requesting-code-review`) correctly **not** run in this task per **approved deviation 8** (coordinator runs review at a higher level) |

**Completion Rate:** 12/12 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 12 tasks are fully implemented. The two intentionally-out-of-scope items (`gms_12`/`gms_48` templates staying battleship-free, per R-3) are correctly documented content-absence findings, not gaps — confirmed both templates contain zero "BATTLESHIP" references.

## Acceptance Criteria Traceability — independently re-verified

| PRD acceptance criterion | Plan's claimed coverage | Verification |
|---|---|---|
| Cast mounts vehicle 1932000 via MONSTER_RIDING, self + foreign | Tasks 1, 8, 11 | Confirmed: `mount.go:142-160` resolves + applies; templates carry `CharacterBuffGive` options on all 9 versions |
| No cooldown on cast; cast-while-cooling rejected | Task 8 | Confirmed: `shouldApplyCastCooldown` excludes battleship; `battleshipCastBlocked` gates re-cast |
| HP init, version-gated formula | Tasks 6, 8 | Confirmed: `ShipHP`/`isPostBigBangDurability` in processor.go; unit tests cover both arms across all versions incl. the R-4 v87 crossover |
| Drain + parallel HP + gauge per drain | Tasks 6, 9 | Confirmed: `Drain` in processor.go; `announceShipHpGauge` in character_damage.go; `ChangeHP` call untouched (character HP flow independent) |
| Break exactly-once → dismount + cooldown + state cleared | Tasks 2, 6 | Confirmed: atomic `InitIfMissingAndDecrBy`/`DecrByIfExists` Lua scripts; `TestDrainBreakExactlyOnceUnderConcurrency` present and passing |
| Manual dismount/expiry/logout: no cooldown, state cleared | Task 7 | Confirmed: EXPIRED hook and session Destroy both call `.Clear()` only, no cooldown func invoked on those paths |
| Cannon/Torpedo rejected on foot, normal while riding | Task 10 | Confirmed: `battleshipAttackPermitted` gate + test matrix covering both skills, on/off ship, tenant isolation |
| Wire values config-resolved, live tenants backfilled | Tasks 3, 4, 11, 12 | Confirmed: no literal 5221999/1932000 found in `services/` or `libs/atlas-constants` Go code (only in JSON templates and test fixtures); `backfill.md` runbook present |
| Feature reachable on every version with the skill | Task 11 Steps 3-4 | Confirmed: all 4 missing handler pairs wired at plan-specified opcodes; gms_12/48 correctly excluded |
| mount_test.go flips + new unit tests | Tasks 1-10 | Confirmed: every listed test file exists with the named test functions, all passing under `-race` |
| test/vet/build/bake clean; all 4 guards clean | Task 12 | Confirmed below |

## Build & Test Results

| Service/Module | Build | Vet | Tests (`-race`) | Notes |
|---|---|---|---|---|
| libs/atlas-constants | PASS | PASS | PASS | |
| libs/atlas-redis | PASS | PASS | PASS | |
| libs/atlas-packet | PASS | PASS | PASS | |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS | full package list clean; `battleship` and `kafka/consumer/buff` re-run with `-count=1` to bypass cache, still PASS |
| services/atlas-configurations/atlas.com/configurations | PASS | PASS | PASS | template-consuming tests all PASS |

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/template-opcode-order-guard.sh` | exit 0 (22 arrays in ascending order) |
| `tools/lint.sh --check` | exit 0 (both a full-repo run and a scoped run against the 5 changed modules) |
| `docker buildx bake atlas-channel` | success (image `atlas-channel:local` built, mostly cached layers) |
| `tools/service-registration-guard.sh` | correctly not required — confirmed zero diff in `.github/config/services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, `tools/db-bootstrap.sh` |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the parallel backend-guidelines-reviewer pass, which this agent does not substitute for)

## Action Items

None required. No gaps, no unresolved deviations, no unverified acceptance criteria found.
