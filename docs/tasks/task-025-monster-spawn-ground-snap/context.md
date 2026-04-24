# Context — Monster Spawn Ground-Snap

## Why this fix exists

Some maps (notably `910310002`) have monster spawn entries in the WZ source whose `y` is a few pixels above the platform the mob is supposed to stand on. Atlas serves the raw value, atlas-maps spawns the monster at that `y`, and the mob falls onto the platform on first physics tick. Visually awkward.

Cosmic has the geometry helpers (`MapleMap.calcPointBelow`, `FootholdTree.findBelow`, `MapleMap.spawnMonsterOnGroundBelow`) but does **not** wire them into the regular `SpawnPoint` path either — so Cosmic has the same bug for static spawns. We're going one step further than upstream.

## Where the fix lives

All changes are in `atlas-data`. The fix runs at `GetMonsters(mapId)` serve time. atlas-maps, atlas-monsters, and any other consumer get corrected `Y` automatically through the existing REST contract (`GET /data/maps/{mapId}/monsters`).

## Decision points (locked from brainstorm)

- **Snap location:** `atlas-data` at serve time, not at parse time and not at consumer.
- **Flying/swimming gate:** template-driven, derived from `AnimationTimes` keys. Not just `Fh == 0` — the user explicitly wants the template-flag check. Cosmic's `MonsterStats.isMobile()` (`MonsterStats.java:142`) is the reference — it checks `move` and `fly` keys.
- **Detection rule (this design):**
  - `Flying` = `"fly"` key present.
  - `Swimming` = `"hover"` or `"swim"` key present.
- **Fh-driven snap is preferred over `findBelow` when `Fh != 0`** — it uses the named foothold from the data, avoiding any chance of `findBelow` selecting a different platform on stacked-foothold maps.

## Pieces already in place

- `services/atlas-data/atlas.com/data/map/model.go:35` — `findBelow(p) *FootholdRestModel`
- `services/atlas-data/atlas.com/data/map/processor.go:109` — `calcPointBelow(tree, initial) (point.RestModel, bool)`
- `services/atlas-data/atlas.com/data/monster/reader.go:88` — `m.AnimationTimes = getAnimationTimes(exml)`
- `services/atlas-data/atlas.com/data/monster/storage.go:22` — `monster.NewStorage(l, db)` constructor
- `services/atlas-data/atlas.com/data/map/resource.go:39` — `handleGetMapMonstersRequest` (this is the wiring point for the new monster `Storage`)

## What is NOT being changed

- `bSearchDropPos`, `calcDropPos` (drop landing) — unrelated path.
- atlas-maps spawn processor — passes through whatever atlas-data returns.
- atlas-monsters monster creation — accepts whatever Y atlas-maps provides.
- Cosmic-style dynamic spawn (`spawnMonsterOnGroundBelow`) — atlas doesn't have one and we're not adding it.

## Edge cases worth keeping in mind

1. **Spawn point's `Fh` references a wall foothold** (`x1 == x2`): wall is unwalkable; leave `Y` alone, log warning.
2. **Spawn point's `X` falls outside the named foothold's `[x1, x2]` span**: data is internally inconsistent; leave `Y` alone, log warning.
3. **Idempotency**: snapping an already-correct point returns the same `Y`. Verify with a test that runs the snap twice.
4. **Linked maps** (`<int name="link" value="..."/>`): `Read` follows the link recursively; the linked map's spawn points feed through the same `GetMonsters` and get snapped. Nothing extra needed.
5. **Empty foothold tree** (extremely unusual — empty `MapleStory` map): `findById` returns nil for any id; falls through to "leave alone" branch. Safe.

## Validation references

- WZ life entry shape: `services/atlas-data/atlas.com/data/map/reader.go:374 getLife`. `fh` is parsed as `uint16`.
- Foothold geometry: `services/atlas-data/atlas.com/data/map/reader.go:220 getFootholdTree`. Insertion is into a quadtree (`InsertSingle` at `model.go:171`).
- Cosmic reference (read-only): `~/source/Cosmic/src/main/java/server/maps/MapleMap.java:497-514` (`calcPointBelow`) and `MapleMap.java:1802-1808` (`spawnMonsterOnGroundBelow`).

## Manual smoke tests (post-deploy)

1. `910310002` — non-flying mobs spawn flush on platforms.
2. Any cave map with bats (e.g., `100020000` Lith Harbor sky/cave) — bats still spawn at intended Y.
3. Aquarium `230000000` — fish still spawn at intended Y.
4. Henesys hunting ground `100000000` — snails/orange mushrooms unchanged (their Y is already correct in clean data).
