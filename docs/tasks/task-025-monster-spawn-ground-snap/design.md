# Design: Monster Spawn Ground-Snap

## Problem

Some maps (e.g. `910310002`) have monster spawn points whose `Y` coordinate is slightly above the foothold the mob is intended to stand on. Non-flying mobs spawn in midair and fall on first tick. The misalignment is in the WZ source data; atlas currently passes raw `(X, Y, Fh)` from the spawn point to monster creation with no correction.

## Root Cause

`atlas-maps/atlas.com/maps/map/monster/processor.go:143` spawns monsters using `sp.X, sp.Y, sp.Fh` straight from the data layer:

```go
go func(sp monster2.SpawnPoint) {
    p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
}(sp)
```

The spawn point comes from `atlas-data/atlas.com/data/map/processor.go:337 GetMonsters(...)`, which returns `m.Monsters` unchanged. `m.Monsters` is populated by `getLife(...)` at `atlas-data/atlas.com/data/map/reader.go:374`, where each `life` entry's `x`, `y`, `fh` come straight from the WZ map node.

If WZ has the wrong `y`, atlas serves the wrong `y`, atlas-maps spawns at the wrong `y`, and the mob falls.

## Mechanism Already Available

`atlas-data` already implements the foothold geometry helpers ported from Cosmic:

- `services/atlas-data/atlas.com/data/map/model.go:35` — `FootholdTreeRestModel.findBelow(point.RestModel) *FootholdRestModel`
- `services/atlas-data/atlas.com/data/map/processor.go:109` — `calcPointBelow(tree, initial)` returns the corrected `(X, Y)` on the foothold below `initial`, handling slanted footholds via the same trig as Cosmic's `MapleMap.calcPointBelow`.
- `services/atlas-data/atlas.com/data/map/resource.go:41` — `POST /data/maps/{mapId}/footholds/below` exposes `findBelow` over REST.

The pieces exist; they're just not wired into the spawn-point output path.

## Approach (B: Fh-driven snap with template-flag gate)

Apply the snap inside `atlas-data` when serving spawn points. For each spawn point on a map:

| Case | Action |
|---|---|
| `Fh != 0` and foothold lookup succeeds | Recompute `Y` on that foothold via the slope formula. Replace `sp.Y` with the result. |
| `Fh != 0` but the named foothold is missing or is a wall | Leave `sp.Y` alone. Log warning. |
| `Fh == 0`, monster template is **flying** or **swimming** | Leave `sp.Y` alone. Free-placement is intentional. |
| `Fh == 0`, monster template is ground | `findBelow` from `(sp.X, sp.Y - 1)`, then `calcPointBelow`. Replace `sp.Y` with the result (minus 1 to match Cosmic's `spawnMonsterOnGroundBelow` offset). |
| `Fh == 0`, monster template lookup fails | Leave `sp.Y` alone. Log warning. |

The snap is **idempotent** — re-running it on already-correct data returns the same `Y`. Cheap to apply on every fetch.

### Why apply at serve time, not at parse time

Spawn points are parsed in `getLife(...)` (`reader.go:374`); the foothold tree is built later in the same `Read(...)`. We could mutate `m.Monsters` right after `m.FootholdTree = getFootholdTree(...)`, but we'd still need monster template data — which lives in a different package, in a different storage instance, parsed at a different time. Cross-coupling parsers is messy.

Doing it at serve time inside `GetMonsters(...)` keeps the dependency graph clean: `Storage` (maps) and the new `monster.Storage` (templates) are both already constructed in the resource handler. We add a single dependency edge there.

### Why animation-key gate, not just `Fh == 0`

The user explicitly asked us to surface a flying flag from monster templates so the gate is correct even when the WZ data has `Fh == 0` for a ground mob. Cosmic's `MonsterStats.isMobile()` (`MonsterStats.java:142`) checks `animationTimes.containsKey("move") || animationTimes.containsKey("fly")`. Atlas already parses `AnimationTimes` into `monster.RestModel` (`monster/reader.go:88`) — we just need to expose convenience booleans derived from those keys.

### Detection rule

- `Flying`: animation key `"fly"` present (matches Cosmic).
- `Swimming`: animation key `"hover"` or `"swim"` present.
- A mob with both `move` and `fly` (e.g., some bosses) is treated as flying — its WZ data placed it intentionally in the air.

## Architecture & Scope

| Service | Change |
|---|---|
| `atlas-data` — monster reader | Derive `Flying` and `Swimming` booleans from `AnimationTimes` at parse time. Add the two fields to `monster.RestModel`. |
| `atlas-data` — map model | Add `FootholdTreeRestModel.findById(uint32) *FootholdRestModel` and `calcYOnFoothold(*FootholdRestModel, x int16) (int16, bool)`. |
| `atlas-data` — map processor | Modify `GetMonsters(...)` to apply ground-snap before returning. Cross-references monster template storage. |
| `atlas-data` — map resource | `handleGetMapMonstersRequest` injects monster `Storage` into the new `GetMonsters` signature. |
| `atlas-maps` | No changes. |
| `atlas-monsters` | No changes. |

No new Kafka topics, no new endpoints, no schema changes. The REST contract gains two boolean fields on `/data/monsters/{id}` (`flying`, `swimming`) — additive, backward compatible.

## Data Model Changes

### `services/atlas-data/atlas.com/data/monster/rest.go`

Add two fields to `RestModel`:

```go
type RestModel struct {
    // ... existing fields ...
    AnimationTimes map[string]uint32 `json:"animation_times"`
    Flying         bool              `json:"flying"`
    Swimming       bool              `json:"swimming"`
    // ... rest ...
}
```

### `services/atlas-data/atlas.com/data/map/monster/rest.go`

No struct changes. Spawn point `RestModel` keeps `X`, `Y`, `FH` as-is — only the served `Y` is corrected.

## Algorithm Reference

### `findById` (new)

Walks the tree's quadrants returning the first matching foothold. Linear in the number of footholds for the queried map (footholds are bounded; few hundred per map).

```go
func (f *FootholdTreeRestModel) findById(id uint32) *FootholdRestModel {
    for i := range f.Footholds {
        if f.Footholds[i].Id == id {
            return &f.Footholds[i]
        }
    }
    for _, child := range []*FootholdTreeRestModel{f.NorthWest, f.NorthEast, f.SouthWest, f.SouthEast} {
        if child == nil {
            continue
        }
        if r := child.findById(id); r != nil {
            return r
        }
    }
    return nil
}
```

### `calcYOnFoothold` (new)

Pure helper. Given a foothold and an `X`, returns the `Y` on that foothold's line. Mirrors the slope branch of `calcPointBelow`. Returns `false` if the foothold is a wall (`isWall`) or `X` falls outside the foothold's span.

```go
func calcYOnFoothold(fh *FootholdRestModel, x int16) (int16, bool) {
    if fh.isWall() {
        return 0, false
    }
    if x < fh.First.X || x > fh.Second.X {
        return 0, false
    }
    if fh.First.Y == fh.Second.Y {
        return fh.First.Y, true
    }
    s1 := math.Abs(float64(fh.Second.Y - fh.First.Y))
    s2 := math.Abs(float64(fh.Second.X - fh.First.X))
    s4 := math.Abs(float64(x - fh.First.X))
    alpha := math.Atan(s2 / s1)
    beta := math.Atan(s1 / s2)
    s5 := math.Cos(alpha) * (s4 / math.Cos(beta))
    if fh.Second.Y < fh.First.Y {
        return fh.First.Y - int16(s5), true
    }
    return fh.First.Y + int16(s5), true
}
```

### `snapToGround` (new, in `processor.go`)

```go
func snapToGround(tree FootholdTreeRestModel, sp monster.RestModel, lookup func(uint32) (monster_tpl.RestModel, error)) monster.RestModel {
    if sp.FH != 0 {
        if fh := tree.findById(uint32(sp.FH)); fh != nil {
            if y, ok := calcYOnFoothold(fh, sp.X); ok {
                sp.Y = y
                return sp
            }
        }
        return sp // foothold missing or wall; leave alone
    }
    tpl, err := lookup(sp.Template)
    if err != nil {
        return sp
    }
    if tpl.Flying || tpl.Swimming {
        return sp
    }
    if pt, ok := calcPointBelow(tree, point.RestModel{X: sp.X, Y: sp.Y - 1}); ok {
        sp.Y = pt.Y - 1
    }
    return sp
}
```

### `GetMonsters` (modified)

The signature gains a second `Storage` parameter — the monster template storage:

```go
func GetMonsters(s *Storage, ms *monster_tpl.Storage) func(ctx context.Context) func(mapId _map.Id) ([]monster.RestModel, error)
```

Implementation iterates `m.Monsters`, applies `snapToGround` for each, and returns the new slice. Original `m.Monsters` is not mutated (it's the cached parse result).

## Testing Strategy

### Unit tests

1. `services/atlas-data/atlas.com/data/monster/reader_test.go`
   - `Flying` derived: animation map with key `"fly"` → `true`.
   - `Flying` not derived: only `"move"` → `false`.
   - `Swimming` derived: `"hover"` or `"swim"` → `true`.
   - Both flying and swimming: `"fly"` + `"swim"` → both `true`.

2. `services/atlas-data/atlas.com/data/map/model_test.go` (new file)
   - `findById`: returns matching foothold from root quadrant.
   - `findById`: returns matching foothold from a deep child quadrant.
   - `findById`: returns `nil` for unknown id.
   - `calcYOnFoothold`: flat foothold returns `Y1`.
   - `calcYOnFoothold`: down-slope (`Y2 > Y1`) returns intermediate `Y`.
   - `calcYOnFoothold`: up-slope (`Y2 < Y1`) returns intermediate `Y`.
   - `calcYOnFoothold`: wall returns `(0, false)`.
   - `calcYOnFoothold`: `X` outside span returns `(0, false)`.

3. `services/atlas-data/atlas.com/data/map/processor_test.go` (new file or extension)
   - `snapToGround` with `Fh != 0` and valid foothold → `Y` corrected.
   - `snapToGround` with `Fh != 0` and missing foothold → `Y` unchanged.
   - `snapToGround` with `Fh == 0` and flying mob → `Y` unchanged.
   - `snapToGround` with `Fh == 0` and swimming mob → `Y` unchanged.
   - `snapToGround` with `Fh == 0` and ground mob with foothold below → `Y` corrected to `result.Y - 1`.
   - `snapToGround` with `Fh == 0` and ground mob with no foothold below → `Y` unchanged.
   - Idempotency: applying `snapToGround` twice yields the same `Y`.

### Integration / manual test

After deploy, log into map `910310002` and watch a non-flying spawn — should appear flush with the platform, no fall. Compare against a flying-mob map (e.g., a Kerning/Sleepywood cave with bats) — flying mobs should still spawn at their air `Y`.

## Risk & Mitigation

- **`findById` cost on map serve.** Linear walk on each spawn point; tens of spawn points × few hundred footholds is sub-millisecond. Acceptable.
- **Cosmic doesn't snap regular `SpawnPoint.getMonster()` either.** This is a divergence — atlas will be more correct than Cosmic. Intentional.
- **`AnimationTimes` semantics for `"swim"`/`"hover"`.** If WZ uses different keys for swim mobs in some versions, we'd miss them and snap fish to the seabed. Mitigation: smoke-test on an aqua map (e.g., Aquarium 230000000) before merging. If broken, expand the key set.
- **Spawn points authored in midair on purpose** (e.g., a chained boss summon). With `Fh == 0` and a non-flying template these will get snapped down. If we hit a regression we expand the gate to also skip when the spawn point's `Y` is more than N pixels above the nearest foothold (proxy for "intentional midair"). Not implementing this guard preemptively — wait for a real example.

## Out of Scope

- Player NPCs, drops, reactor positions — already use `calcPointBelow`/`calcDropPos`. Untouched.
- The `bSearchDropPos` heuristic for off-foothold drop landing. Untouched.
- Cosmic-style `spawnMonsterOnGroundBelow` for dynamic spawns (boss summons, reactor mobs). Untouched — those paths don't pass through `GetMonsters`.

## Definition of Done

1. `Flying`, `Swimming` booleans on `monster.RestModel`, populated from `AnimationTimes`, with reader tests.
2. `findById` and `calcYOnFoothold` on `FootholdTreeRestModel`, with model tests.
3. `snapToGround` applied to every spawn point returned by `GetMonsters`, with processor tests covering all five branches.
4. `handleGetMapMonstersRequest` wired with both storages.
5. `go test ./...` green in `atlas-data`.
6. `docker compose build atlas-data` succeeds.
7. Manual smoke on map `910310002` confirms ground mobs spawn flush.
