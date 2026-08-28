# Backend Audit Re-check — task-122-attack-path-snapshots

Scoped re-check of the 11 blocking findings (1x DOM-01, 10x DOM-20) from
`backend-audit.md`, against fix commits `633a3afb0`, `e4ee83ffe`,
`79432268b`, `5f9e1f390`. Only these two rule families were re-run; all other
families from the original audit are unchanged and not re-litigated here.

- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS — `go build ./character/... ./movement/... ./data/skill/... ./kafka/consumer/...` (atlas-channel) and `go build ./character/...` (atlas-character), both exit 0.
- **Result:** All 11 findings resolved. 0 remaining FAIL.

## DOM-01 — `character/skill/builder.go`

Rule: `builder.go` exists with `NewBuilder()`, fluent setters, and a
validating `Build()`.

| Check | Status | Evidence |
|---|---|---|
| `NewBuilder()` exists | PASS | `services/atlas-channel/atlas.com/channel/character/skill/builder.go:26` — `func NewBuilder(id skillconst.Id) *modelBuilder { return NewModelBuilder(id) }`, delegating to the pre-existing `NewModelBuilder` (line 21). |
| Fluent setters | PASS | `builder.go:41-51` — `SetLevel`, `SetMasterLevel`, `SetExpiration`, `SetCooldownExpiresAt` all return `*modelBuilder`. |
| Validating `Build()` | PASS | `builder.go:53-56` — rejects `b.id == 0` with `ErrInvalidSkillId` before constructing the `Model`. |

**Verdict: PASS.** No production caller of `NewBuilder` exists, matching the
already-ruled-acceptable `compartment.NewBuilder`/`inventory.NewBuilder`
precedent — not re-litigated per the task brief.

## DOM-20 — table-driven tests

Rule: tests use `tests := []struct{...}` (or equivalent) + `t.Run`. Each file
graded on its own merits — a single-row table wrapping one irreducible
scenario is an explicitly permitted outcome; a partial conversion leaving
most functions unconverted is not.

| File | Test functions | Table-driven (`[]struct{...}`/`t.Run`) | Verdict | Evidence |
|---|---|---|---|---|
| `character/snapshot/registry_test.go` | 17 | 17/17 | PASS | Every `func Test...` (e.g. `registry_test.go:83,106,128,154,184,208,229,249,...`) declares a `tests :=`/`cases := []struct{...}` and dispatches via `t.Run`. `TestRegistry_ApplyStatChanged_MissingOrUnappliableValueInvalidates` (`registry_test.go:184-206`) is a genuine 4-row value table; single-scenario functions (e.g. `registry_test.go:208-227`) keep one-row tables per the permitted single-row exception. |
| `character/snapshot/processor_test.go` | 10 | 10/10 | PASS | Every function (`processor_test.go:68,102,146,177,211,254,298,338,365,...`) follows the `tests := []struct{ name string; fn func(t *testing.T) }` + `t.Run` shape. |
| `character/snapshot/shadow_test.go` | 3 | 3/3 | PASS | `shadow_test.go:16,43,95` — all three use `tests :=`/`t.Run`; `TestCompareProjection_PositionToleranceBanded` (`shadow_test.go:95-111`) is a genuine multi-row value table. |
| `data/skill/cache_test.go` | 1 | 1/1 (7 rows) | PASS | `cache_test.go:52-271` — `TestSkillCache` collapses the 7 original scenarios (`PositiveHitAvoidsSecondFetch`, `ExpiredEntryRefetches`, `NegativeCachesNotFound`, `TransientErrorNotCached`, `DisabledBypasses`, `TenantIsolation`, `ConcurrentAccess`) into a `name`/`run` table dispatched via `t.Run` at line 268-270; each row's assertions are the original test body verbatim. |
| `kafka/consumer/buff/consumer_test.go` | 1 | 1/1 (3 rows) | PASS | `consumer_test.go:49-147` — `TestHandleSnapshotBuff` genuinely decomposes into `AppliedAndExpired`, `Applied_NoSnapshot_NoOp`, `Expired_NoSnapshot_NoOp`, preserving the original negative-predicate assertions (`BuffsGen`/`BuffsValid`/`len(Buffs)`, not merely "no error") at lines 117 and 137. |
| `kafka/consumer/character/consumer_test.go` | 1 | 1/1 (6 rows) | PASS | `consumer_test.go:54-201` — `TestSnapshotHandlers` decomposes into 6 scenarios (`StatChanged_RichValuesApplyInPlace`, `StatChanged_NilValuesInvalidates`, `LevelAndExperience_ApplyAbsolute`, `MapChanged_TargetPositionSetsOverlay`, `MapChanged_PortalWarpInvalidatesPositionAndCore`, `IgnoreOtherWorlds`), dispatched via `t.Run` at line 198-200. |
| `kafka/consumer/skill/consumer_test.go` | 1 | 1/1 (3 rows) | PASS | `consumer_test.go:52-138` — `TestHandleSnapshotSkill` decomposes into `CreatedAndUpdated_Upsert`, `Deleted_Removes`, `Deleted_NoSnapshot_NoOp`, preserving the negative-predicate assertion at line 128. |
| `movement/processor_test.go` | 5 | 5/5 | PASS | `TestNarrowSkillBytes` (`processor_test.go:49-76`) is a genuine 6-row value table preserving the byte-boundary literals (255 at line 63) and negative/overflow rejections (lines 59-62); `TestComputeAckMp` (78-123) is a genuine 4-row value table; `TestResolveLiveMonster` (150-247), `TestForCharacter` (252-309), `TestForMonster` (314-344) use the `name`/`run` closure-row shape, each dispatched via `t.Run`. |
| `movement/teleport_test.go` | 1 | 1/1 (4 rows) | PASS | `teleport_test.go:25-183` — `TestTeleportAndForCharacterPosition` decomposes into 4 scenarios, including `TeleportCharacter_EmitsFhZeroOnWire` (124-178) which preserves the wire-level `Fh == 0` boundary check (line 161) called out as load-bearing in the original audit. |
| `character/stat_values_test.go` (atlas-character) | 1 | 1/1 (10 rows) | PASS | `stat_values_test.go:233-424` — `TestStatChanged_ValuesCompleteOnHotPaths` is a genuine per-row-data table (`entity`, `run`, `requiredUpdates`, `checkModel` fields), covering all 10 original call sites (`ChangeMP`, `ChangeHP`, `AwardExperience`, `AwardLevel`, `RequestChangeMeso`, `APDistributeSuccess`, `LevelUpGrowth`, `JobChangeGrowth`, `ResetStats`, `RebalanceAP`), dispatched via `t.Run` at line 407-423. |

**Verdict: all 10 files PASS.** Every file's test functions were converted,
not a subset; multi-scenario functions decompose into genuine multi-row
tables (value tables where the checklist example applies, or `name`/`run`
closure tables where setups differ too much to share a value shape); the
handful of intentionally single-row tables (e.g.
`TestRegistry_ApplyStatChanged_EmptyUpdatesIsNoOp`,
`TestRegistry_ApplyStatChanged_PetSnIsSkipped`, `TestForMonster`) each wrap
one genuinely irreducible scenario, matching the permitted-outcome carve-out
in the task brief.

## Build & Test

```
cd services/atlas-channel/atlas.com/channel && go build ./character/... ./movement/... ./data/skill/... ./kafka/consumer/...   -> exit 0
cd services/atlas-character/atlas.com/character && go build ./character/...   -> exit 0
```

(`tools/verify.sh` deliberately not run here — running concurrently
elsewhere in this worktree per instruction.)

## Summary

### Blocking (must fix)

None. Both DOM-01 and DOM-20 are resolved.

### Non-Blocking (should fix)

Carried over unchanged from `backend-audit.md` (not re-touched by these four
commits, not re-evaluated here):
- `character/snapshot/shadow.go:73` — `func(ctx context.Context)` instead of the service-wide `func(_ context.Context)` convention.
- `character/snapshot/shadow.go:105` — buffs-skip note logs at Debug level.

### Not evaluable from the diff

None for this re-check — DOM-01 and DOM-20 were fully evaluable from the
changed files.
