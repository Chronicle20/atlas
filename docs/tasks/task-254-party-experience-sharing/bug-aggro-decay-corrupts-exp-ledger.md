# Bug — aggro decay and aggro-clear corrupt the EXP damage ledger

**Environment:** `atlas-pr-1456` (namespace `atlas-pr-1456`)
**Service at fault:** `atlas-monsters` (NOT a task-254 regression — task-254's
party split is correct)

## Not the originally reported symptom

The reported kill (Chronicle `640 + 64`, Atlas `960 + 96`) turned out to be
**correct behaviour**. The KILLED event at `2026-08-26T20:36:30.789Z` for
monster `8140101` carried:

```
damageEntries: [{characterId: 2, damage: 7906}, {characterId: 1, damage: 29094}]
```

Sum `37000` = Black Kentaurus's full HP (`atlas-data`: `hp 37000,
experience 1600, level 88, boss false`). Atlas dealt 79% of the damage and
correctly took the `+0.2` MVP share; both level-200 members split
`0.6 / 0.4` of the 1600 pool. That kill shows no decay at all.

## The real defect, found in the same logs

`atlas-monsters` keeps ONE `DamageEntries` list per monster
(`registry.go:83-87`, `model.go:81-85`) and uses it for two unrelated jobs:
aggro targeting and the EXP damage ledger. Two aggro-side mutations therefore
destroy EXP state.

### 1. Decay

`MonsterAggroDecayTask` sweeps every 1.5s (`aggro.go:22`). An entry idle for
more than 10s (`aggro.go:10`) is multiplied by 0.85 on each subsequent tick
(`aggro.go:14`) and pruned below 1 (`registry.go:846-852`, called from
`aggro_task.go:112`). Bosses are skipped; Black Kentaurus is not one.

The KILLED/DAMAGED events publish that mutated list: `processor.go:633`
passes `m.DamageSummary()` (`model.go:179-181`, a direct return of
`m.damageEntries`) to `killedStatusEventProvider` (`producer.go:149-156`).

Observed in `atlas-monster-death` logs — kills of the same 37000-HP monster:

| Time (UTC) | Entries | Sum | Missing |
|---|---|---|---|
| 20:36:30.789 | `2:7906`, `1:29094` | 37,000 | — |
| 20:44:41.485 | `2:5530`, `1:26402` | 31,932 | 5,068 |
| 20:47:53.873 | `2:3680`, `1:29636` | 33,316 | 3,684 |

Caught mid-fight in consecutive DAMAGED events: character 2 goes
`7656 -> 5530` while character 1 *grows* `26174 -> 26402`. That is exactly
`floor(floor(7656 * 0.85) * 0.85) == 5530` — two decay ticks.

### 2. Aggro clear

`ClearDamageEntries` (`registry.go:910`, reached from `ProcessorImpl.ClearAggro`,
`processor.go:2018`) wipes **every** entry outright. An aggro wipe mid-fight
therefore erases the entire EXP damage history, not just the targeting table.

## Consequences

- Under-awarded EXP whenever a contributor idles 10s+ before the kill: the
  pool itself is unaffected (`epd = monsterExp / totalDamage` and
  `participationExp = partyDamage * epd` shrink together), but the *relative*
  shares are wrong.
- MVP (`+0.2`) can flip to the wrong character
  (`atlas-monster-death monster/experience.go:299-308`).
- `DamageLeader()` / drop ownership (`atlas-monster-death processor.go:60-64`).
- White vs yellow EXP, which keys on `personalRatio` against `totalDamage`.
- Solo EXP is unaffected — the ratio and `epd` cancel.

## Reference divergence

Cosmic keeps the two ledgers strictly separate:

- `Monster.takenDamage` (`<cosmic>/src/main/java/server/life/Monster.java:113`,
  written `:470-473`) is the EXP ledger. Never decayed, never cleared by aggro;
  read for distribution at `:613` and `:982`.
- `MonsterAggroCoordinator`'s `PlayerAggroEntry`
  (`<cosmic>/.../net/server/coordinator/world/MonsterAggroCoordinator.java:150`)
  is the aggro ledger and the only one that expires (`:140-143`).
  `Monster.aggroClearDamages()` (`:2172-2174`) clears **only** the coordinator
  entries.

## Fix direction

Split the ledger inside `storedDamageEntry` / `entry`: keep the existing
decaying `Damage` for aggro, and add a never-decayed cumulative total.

- `ApplyDamage` (`registry.go:524`) already computes the clamped `actual` both
  ledgers need — write it to both.
- `DecayDamageEntries` and `ClearDamageEntries` mutate only the aggro value,
  and prune an entry from the *aggro* view only.
- `Model.DamageSummary()` (feeding DAMAGED/KILLED) returns the EXP totals.
- `Model.DamageEntries()` (feeding the `IsAggroIdle` check at
  `aggro_task.go:84`) keeps returning the aggro view.
- Decide explicitly what `rest.go:31` exposes.

Monster state is ephemeral Redis runtime state, so no data migration is needed.
The change crosses a service seam into `atlas-monster-death` (EXP, MVP, drop
ownership) and must be traced into those consumers by hand per CLAUDE.md.
