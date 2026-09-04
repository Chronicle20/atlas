# Evidence — `mobTime == -1` spawn points are never spawned

Collected 2026-09-03 against the PR environment `atlas-pr-1566`
(tenant `d1defcd0-9994-4ee2-bcad-9eaaa1377839`, GMS 83.1, world 0 / channel 0).
The defect is **not** specific to PR #1566 — the filter and the missing one-time path are
both on `main`; the deployed env is only where it was observed.

## Reproduction

Character in map `920010920` (Hidden Street / Tower of Goddess \<Jail II\>).
Expected 9300044 ("Lucida in Tower of Goddess", level 65) to be present. None spawn, ever.

## Code path

1. `services/atlas-maps/atlas.com/maps/data/map/monster/processor.go:58`
   ```go
   func (p *ProcessorImpl) Spawnable(point SpawnPoint) bool {
       return point.MobTime >= 0
   }
   ```
   applied by `SpawnableSpawnPointProvider` at `:46`.
2. `services/atlas-maps/atlas.com/maps/map/monster/registry.go:205` —
   `InitializeForMap` seeds Redis from `dp.GetSpawnableSpawnPoints(...)`, the *filtered*
   set. For `920010920` that is the empty set.
3. `services/atlas-maps/atlas.com/maps/map/monster/processor.go` — `SpawnMonsters` then hits
   `if totalCount == 0 { return nil }` and returns with no log output.
4. `GetSpawnPoints` / `SpawnPointProvider` (the unfiltered variants) have **no non-test
   consumer** in the repository — verified by
   `grep -rn "GetSpawnPoints\|SpawnPointProvider(" --include="*.go" services libs`.
   There is no field-creation or one-time spawn path anywhere in Atlas.

## Live log confirmation

`kubectl logs -n atlas-pr-1566 deploy/atlas-maps`, map entry at 19:05:37.998Z:

```
Executing spawn mechanism for Tenant [... Version [83.1]] Field [0:0:920010920:00000000-...].
Issuing [GET] request to [.../api/data/maps/920010920/monsters?page[number]=1&page[size]=250].
```

…followed by **no** `Spawning monster at spawn point ...` and **no**
`Spawned N monsters out of M needed ...` line. Same at the 19:05:47 periodic pass. (The
`Spawn for map [920010920] ...: existing=[6], issuing=[0] CREATE commands.` line in the same
window is the *reactor* spawner, not the monster spawner.)

## Data is correct at the source

`GET /api/data/maps/920010920/monsters` returns 10 points, all
`{"template": 9300044, "mob_time": -1, "hide": false}`. `920010910` returns 8 of the same.

`services/atlas-data/atlas.com/data/map/reader.go:415-416` defaults both `hide` and
`mobTime` to `0`, so `-1` is genuine WZ data and not an ingest artifact. A control map,
`104040000`, returns `mob_time: 0` for all 39 of its points.

## Dataset sweep

All 5,261 maps swept via `GET /api/data/maps/{id}/monsters?page[size]=250`
(`xargs -P 24`), two passes: one collecting `mob_time < 0 or hide == true`, one collecting
total vs. one-time counts per affected map. Results in `affected-maps.tsv`.

| Class | Maps |
|---|---|
| Every spawn point is `mobTime = -1` (map can never hold a monster today) | **991** |
| Mixed: some one-time, some recurring | **61** |
| Has a `hide = true` point but no one-time points | **2** |
| Unaffected | 4,207 |

- `-1` is the **only** negative `mobTime` value in the dataset (1,093 template groups).
- Only **two** maps in all of GMS 83.1 carry a `hide = true` monster life entry:
  - `600020300` MesoGears / Wolf Spider Cavern — `9400545` Wolf Spider, lvl 80, `mobTime = 0`
  - `800020130` Zipangu / Encounter with the Buddha — `9400013` Dreamy Ghost, lvl 100, `mobTime = 0`

  Both are currently spawned by Atlas, because `Extract`
  (`services/atlas-maps/atlas.com/maps/data/map/monster/rest.go`) drops the `hide` field
  before it reaches `SpawnPoint`.

### Largest all-one-time maps

| Map | Name | One-time points |
|---|---|---|
| `930000100` | Forest of Poison Haze / Mouth of the Forest | 50 |
| `925100301` | Hidden Street / The Area of 100yrOld Bellflower II | 48 |
| `925100201` | Hidden Street / The Area of 100yrOld Bellflower | 48 |
| `926110001` | Hidden Street / Dark Tunnel (Nett's Pyramid) | 42 |
| `926100001` | Hidden Street (Nett's Pyramid) | 42 |
| `920010200` | Hidden Street / Tower of Goddess \<Walkway\> | 30 |
| `914030000` | Hidden Street / Wolf's Agony | 30 |
| `925100000` | Hidden Street | 26 |
| `922010100` | Hidden Street / Abandoned Tower \<Stage 1\> | 25 |
| `610030520` | Party Quest / Mage Mastery Room (Amoria PQ) | 24 |

### Not affected: script-spawned stages

Kerning PQ stages (`910040100`, `910040200`) and the Orbis PQ lobby (`920010900`) return
**zero** monster life entries. Those are populated by scripts, not by static life, and are
outside this task.
