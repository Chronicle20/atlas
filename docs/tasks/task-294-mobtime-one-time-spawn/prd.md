# One-Time Spawn for `mobTime == -1` Spawn Points — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-03
---

## 1. Overview

`atlas-maps` owns monster spawning. Its spawn-point registry is seeded exclusively from
`SpawnableSpawnPointProvider`, which applies the predicate
`Spawnable(point) == point.MobTime >= 0`
(`services/atlas-maps/atlas.com/maps/data/map/monster/processor.go:46,58`). Every WZ life
entry carrying `mobTime = -1` is therefore discarded before it ever reaches Redis, and
`SpawnMonsters` returns early at `if totalCount == 0` with no log line
(`services/atlas-maps/atlas.com/maps/map/monster/processor.go`). The unfiltered
`GetSpawnPoints` / `SpawnPointProvider` methods exist but have **no non-test consumer**
anywhere in the repository — there is no other code path that could spawn these points.

In the original game `mobTime = -1` does not mean "never spawn". It means "place this
monster once when the field is created, and never respawn it". It is how a field with a
fixed, finite monster layout is authored: party-quest stages, boss antechambers, jump-quest
rooms. Atlas implements the "never respawn" half and silently drops the "place it once"
half, so those fields are permanently empty.

This is not a small corner. Across the GMS 83.1 dataset (5,261 maps swept via
`GET /api/data/maps/{id}/monsters`), **1,052 maps carry at least one `mobTime = -1` spawn
point, and 991 of them consist of nothing but `mobTime = -1` points** — those 991 maps can
never contain a monster on Atlas today. `-1` is the only negative `mobTime` value in the
entire dataset (1,093 template groups, all `-1`). Affected content includes Orbis PQ
(`920010910`, `920010920` — 8 and 10 × Lucida `9300044`; `920010200` Tower of Goddess
\<Walkway\> — 30), Nett's Pyramid (`926110001` Dark Tunnel — 42; `926100001` — 42),
Amoria PQ (`610030510/520/521` — 21/24/22), Ludibrium Maze PQ (`922010100` Abandoned
Tower \<Stage 1\> — 25), and `930000100` Mouth of the Forest (50).

## 2. Goals

Primary goals:

- Implement the `mobTime == -1` semantic in `atlas-maps`: place every such spawn point
  exactly once per field arming, and never respawn it while that arming holds.
- Re-arm a field's one-time spawns when the field empties, so the next party entering the
  same `world:channel:map:instance` finds a full layout.
- Plumb the `hide` flag, already served by `atlas-data` and already parsed into
  `map/monster.RestModel`, through `atlas-maps` and honor it, instead of silently dropping
  it in `Extract`.
- Make the currently-silent "zero spawn points" path observable.

Non-goals:

- Reworking `mobTime > 0` cooldown behavior or the `defaultSpawnCooldown` (5s) default.
- Changing the `getMonsterMax` rate formula for ordinary (`mobTime >= 0`) spawn points.
- Script-driven spawning. Maps whose stages are populated by scripts (Kerning PQ stages
  `910040100`/`910040200` have zero static monster life) are unaffected and out of scope.
- Any change to `atlas-monsters`, `atlas-channel`, or `atlas-data`. The data is already
  correct and already served; the defect is entirely on the `atlas-maps` consumption side.
- Instance/field lifecycle redesign. This task uses the existing `field.Model` key.

## 3. User Stories

- As a player entering Tower of Goddess \<Jail II\> (`920010920`), I want the 10 Lucidas
  (`9300044`) to be present on arrival, so that the Orbis PQ jail stage is completable.
- As a player who has cleared a one-time field, I want the monsters to stay dead for as long
  as my party is in the field, so that a cleared stage does not repopulate under me.
- As the next party to enter that same field on the same channel after it empties, I want a
  full, fresh monster layout, so that the PQ is repeatable.
- As an operator reading `atlas-maps` logs, I want a field that resolves to zero spawnable
  points to say so, so that "no monsters here" is diagnosable without reading source.

## 4. Functional Requirements

### FR-1 — Classify spawn points instead of discarding them

`data/map/monster.SpawnPoint` gains the notion of a one-time point. The `Spawnable`
predicate must no longer be the sole gate on registry seeding.

- FR-1.1 — `Extract` (`services/atlas-maps/atlas.com/maps/data/map/monster/rest.go`) must
  carry `RestModel.Hide` into `SpawnPoint`. It currently drops it.
- FR-1.2 — A spawn point is **one-time** iff `MobTime < 0`. `-1` is the only negative value
  present in the dataset, but the predicate is `< 0` so an unexpected `-2` behaves
  identically rather than falling through to the recurring path.
- FR-1.3 — A spawn point is **recurring** iff `MobTime >= 0`. Recurring behavior is
  unchanged from today in every respect: same eligibility, same `defaultSpawnCooldown`,
  same `getMonsterMax` rate formula, same Redis reservation.
- FR-1.4 — A spawn point with `Hide == true` is **never auto-spawned**, whether one-time or
  recurring. It is excluded from the registry entirely. Exactly two maps in GMS 83.1 carry
  a hidden monster life entry — `600020300` MesoGears / Wolf Spider Cavern (`9400545`
  Wolf Spider, `mobTime = 0`) and `800020130` Zipangu / Encounter with the Buddha
  (`9400013` Dreamy Ghost, `mobTime = 0`) — so this requirement removes two spawn points
  from two maps and changes nothing else. Both are currently spawned by Atlas.

### FR-2 — Arm and fire one-time spawn points

- FR-2.1 — A field's one-time spawn points are fired as a single batch the first time
  `SpawnMonsters` runs for that field key while the field is **armed**.
- FR-2.2 — **Every** one-time point in the field fires, ignoring `getMonsterMax` and
  ignoring the character count. A one-time spawn is a fixed field layout, not a population
  target: entering `920010920` solo must place all 10 Lucidas, not 8.
- FR-2.3 — Firing must be atomic across `atlas-maps` replicas. Two pods (or the
  character-enter path and the 10s `NewRespawn` task) racing on the same field must result
  in exactly one batch, never two. Use a Redis-atomic claim in the same style as
  `ReserveEligibleSpawnPoints`' Lua script — a read-then-write from Go is not sufficient.
- FR-2.4 — Once fired, the field is **disarmed**: one-time points do not respawn, are not
  eligible for the recurring cooldown mechanism, and are not re-fired on subsequent
  `SpawnMonsters` passes, on subsequent character entries, or after the monsters are killed.
- FR-2.5 — A field containing both one-time and recurring points (61 maps in the dataset)
  must fire its one-time batch **and** continue to serve its recurring points under the
  existing rate formula. The recurring `getMonsterMax` computation must be based on the
  recurring spawn-point count only, so one-time points do not inflate the recurring
  population target.
- FR-2.6 — `SpawnMonsters` must not return early on `totalCount == 0` when the field has
  one-time points to fire. The existing early return is the direct cause of the defect.

### FR-3 — Re-arm on field empty

- FR-3.1 — When the last character leaves a field, that field's one-time spawn state is
  cleared and the field returns to **armed**. The next entrant fires a fresh full batch.
- FR-3.2 — The clear happens in `map.ProcessorImpl.Exit`
  (`services/atlas-maps/atlas.com/maps/map/processor.go`), which is the single funnel for
  logout, warp (`TransitionMap`), and channel change (`TransitionChannel`). Emptiness is
  determined by `p.cp.GetCharactersInMap(...)` returning zero remaining characters — the
  same check PR #1566 / task-278 introduces at this site for environment state. If #1566
  has merged, extend that block; if not, introduce the block, and the two must be reconciled
  at rebase rather than duplicated.
- FR-3.3 — Re-arming clears the one-time state only. It must not delete or reset the
  recurring spawn points' cooldown state, and it must not despawn monsters. (In practice the
  field is empty of players; monsters in an empty field are handled by existing behavior,
  which this task does not change.)
- FR-3.4 — Re-arming is per field key (`tenant` + `world:channel:map:instance`), matching
  the existing `character.MapKey`. Emptying channel 0's copy of `920010920` must not re-arm
  channel 1's copy.
- FR-3.5 — The existing `FlushTenant` path
  (`services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler.go:37`), which drops
  spawn-point state on a data-version change, must also drop one-time arming state, so a WZ
  re-ingest does not leave a stale disarmed field.

### FR-4 — Observability

- FR-4.1 — When a field resolves to zero spawn points of any kind, `SpawnMonsters` logs at
  debug with the field id and the counts (`one-time`, `recurring`) before returning. Today
  it returns silently, which is why this defect required source reading to diagnose.
- FR-4.2 — Firing a one-time batch logs at debug: field id, number of points fired.
- FR-4.3 — Re-arming a field on empty logs at debug: field id.

## 5. API Surface

No new or modified HTTP endpoints. `atlas-maps` consumes
`GET /api/data/maps/{mapId}/monsters` from `atlas-data`, whose response already carries both
`mob_time` and `hide` (`services/atlas-data/atlas.com/data/map/monster/rest.go`). No Kafka
message type is added or changed.

## 6. Data Model

No relational schema change; no migration.

Redis state changes only, under the existing `atlas-maps` spawn-point keyspace:

- The spawn-point hash seeded by `SpawnPointRegistry.InitializeForMap`
  (`services/atlas-maps/atlas.com/maps/map/monster/registry.go:205`) must now be seeded from
  a provider that retains one-time points and excludes hidden points, rather than from
  `GetSpawnableSpawnPoints`.
- A per-field one-time arming marker is required, keyed by the same `MapKey` and covered by
  `FlushTenant`'s key sweep (FR-3.5). Whether this is a distinct key or a field on the
  existing hash is a design-phase decision; the constraints are FR-2.3 (atomic claim) and
  FR-3.5 (swept by `FlushTenant`).
- `storedSpawnPoint` (`registry.go:22`) gains whatever fields the classification requires;
  it is already versionless JSON in Redis, and `FlushTenant` on a data-version change gives
  a natural invalidation path for the shape change. Confirm during design that a rolling
  deploy against a pre-existing hash cannot mis-decode.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-maps` | All of it. `data/map/monster` (`Extract` carries `Hide`; new one-time / recurring / hidden classification), `map/monster/registry.go` (seeding, atomic one-time claim, re-arm, `FlushTenant` coverage), `map/monster/processor.go` (fire one-time batch; recurring `getMonsterMax` over recurring points only; remove the silent `totalCount == 0` return), `map/processor.go` (`Exit` re-arm on empty). |
| `atlas-data` | None. Serves `mob_time` and `hide` correctly today; `reader.go:415-416` defaults are `0`, so the `-1` values are genuine WZ data. |
| `atlas-monsters` | None. `CreateMonster` is already the call `atlas-maps` makes. |
| `atlas-channel` | None. |

Conflict note: PR #1566 (task-278, branch `task-278-map-environment-object-state`) adds a
field-empties block to `map.ProcessorImpl.Exit` at exactly the site FR-3.2 targets. Expect a
rebase conflict there and reconcile into one emptiness check rather than two.

## 8. Non-Functional Requirements

- **Multi-tenancy** — All Redis keys stay scoped by `character.MapKey{Tenant, Field}`. No
  cross-tenant read or write. `FlushTenant` must continue to remove every key the tenant has
  been keyed under.
- **Concurrency** — FR-2.3 is the hard requirement: the one-time claim must be atomic in
  Redis (Lua), because `atlas-maps` runs multiple replicas and two independent spawn
  triggers (character enter, and the 10s `NewRespawn` task registered in
  `services/atlas-maps/atlas.com/maps/main.go`) can fire concurrently on the same field.
  Double-firing puts 20 Lucidas in a 10-point map.
- **Performance** — The one-time batch fires at most once per field arming and is bounded by
  the field's spawn-point count; the largest in the dataset is 50 (`930000100`). The
  per-pass cost when a field is already disarmed must be a single Redis round trip, not a
  re-read of the full spawn-point hash.
- **Observability** — FR-4. The silent `return nil` is a defect in its own right.
- **Backward compatibility** — Recurring-only maps (the other 4,209) must behave
  bit-identically to today. This is the primary regression risk and the acceptance criteria
  call it out explicitly.

## 9. Open Questions

1. **Monsters left in an emptied field.** FR-3.3 deliberately does not despawn them. If a
   party clears 6 of 10 Lucidas and leaves, the field re-arms and the next party fires 10
   more on top of the surviving 4. Design must decide whether re-arming should also despawn
   the field's residual monsters, and whether that is this task's business or a
   pre-existing `atlas-monsters` field-lifecycle question. Evidence needed: what currently
   happens to monsters in a field with zero characters.
2. **`hide` semantics.** No consumer of the `Hide` flag exists anywhere in the repo today
   (the only `Hide` hits in `atlas-monsters` are the unrelated `SuperGmHide` GM-invisibility
   feature). FR-1.4 chooses "never auto-spawn" as the interpretation. The alternative — spawn
   with a hidden/invisible flag, revealed later by script — would require a new monster
   attribute and a packet change, which is out of scope. FR-1.4 removes two spawn points
   from two maps (`600020300`, `800020130`), both currently spawned. Confirm during design
   that this is the intended behavior for those two maps and not a silent content
   regression.
3. **Instance keying.** Every observation in this PRD was made against the
   all-zero instance uuid. Confirm during design that instanced fields key the one-time
   marker per instance, which FR-3.4 requires.
4. **Redis shape migration.** Whether a rolling deploy can encounter a pre-existing
   `storedSpawnPoint` hash written by the old code, and whether that decodes safely or needs
   an explicit invalidation.

## 10. Acceptance Criteria

- [ ] `SpawnPoint` carries `Hide`; `Extract` no longer drops it.
- [ ] Entering `920010920` with one character spawns exactly 10 monsters, all template
      `9300044`, at the 10 spawn-point coordinates from
      `GET /api/data/maps/920010920/monsters`.
- [ ] Entering `920010910` with one character spawns exactly 8 × `9300044`.
- [ ] Killing all 10 in `920010920` and remaining in the field produces no respawn over at
      least three `NewRespawn` ticks (>30s).
- [ ] A second character entering the field while the first is still present does not fire a
      second batch.
- [ ] Leaving the field with the last character, then re-entering, fires a fresh batch of 10.
- [ ] Two concurrent `SpawnMonsters` calls on the same armed field (unit test against a real
      or faked Redis, exercising the Lua claim) fire exactly one batch.
- [ ] A mixed map (one of the 61 with both kinds) fires its one-time batch and continues to
      spawn its recurring points; the recurring population target equals
      `ceil((0.70 + 0.05*min(6,chars)) * recurringCount)` and does not count one-time points.
- [ ] Regression: `104040000` (8/10/12/9 recurring, all `mobTime = 0`) spawns and respawns
      exactly as it does on `main`.
- [ ] `600020300` and `800020130` no longer spawn their single `hide = true` point;
      no other map's spawn count changes as a result of FR-1.4.
- [ ] `FlushTenant` on a data-version change clears one-time arming state; a field disarmed
      before the flush fires again after it.
- [ ] A field with zero spawn points of any kind emits a debug log naming the field and both
      counts.
- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] Live smoke test on a PR environment: enter `920010920`, observe 10 Lucidas client-side,
      kill them, confirm no respawn, leave, re-enter, confirm 10 again.
