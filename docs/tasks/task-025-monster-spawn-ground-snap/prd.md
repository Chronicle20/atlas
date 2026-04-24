# PRD: Monster Spawn Ground-Snap

## Problem

Some maps have monster spawn points whose Y coordinate is slightly above the foothold the mob is intended to stand on. Non-flying monsters spawn in midair and fall to the actual foothold. Reproduces on map `910310002`.

The misalignment is in the WZ source data, not in atlas's code — the source is sloppy in places. Atlas currently passes `(X, Y, Fh)` from the spawn point straight to monster creation in `atlas-maps/map/monster/processor.go:143` with no correction.

## Goal

Non-flying, non-swimming monsters spawn at the correct Y for their declared foothold. Flying and swimming monsters are left untouched — their Y is intentional.

## Non-Goals

- Don't change client behavior, network protocol, or anything visible to other services beyond `atlas-data`'s `/data/maps/{mapId}/monsters` payload.
- No retroactive fix for already-spawned mobs.
- No changes to the foothold tree algorithm (`findBelow` / `calcPointBelow`) — already ported from Cosmic and correct.

## Constraints

- Snap must distinguish flying/swimming mobs from ground mobs. The user explicitly chose to derive this from monster-template animation data rather than rely solely on the spawn point's `Fh` field.
- Atlas-data already pulls `AnimationTimes` from `Mob.wz` and exposes it on the monster `RestModel`. Cosmic's flying detection (`MonsterStats.isMobile` at `MonsterStats.java:142`) keys off the presence of a `"fly"` animation key. We will adopt the same signal.
- Swimming detection: WZ mobs that swim have a `"hover"`/`"swim"` animation key (or a movement-type marker on the info node). Use the same animation-key approach.
- The fix should live in atlas-data, not atlas-maps — keep cross-service chatter out of the spawn hot path.

## Success Criteria

1. Map `910310002` spawns walkable mobs flush with their foothold (no fall-in).
2. Flying mobs (e.g., bats on cave maps) spawn at unchanged Y.
3. Swimming mobs (e.g., fish on aqua maps) spawn at unchanged Y.
4. Spawn points whose Y already aligns with the foothold are unchanged (snap is idempotent).
5. Unit tests cover: flat foothold snap, slanted foothold snap, missing-Fh fallback to `findBelow`, flying skipped, swimming skipped, already-aligned no-op.

## Approach (decided in brainstorm)

Approach **B**: Fh-driven snap when `sp.Fh != 0`, fallback to `findBelow` when `sp.Fh == 0` and the monster is non-flying/non-swimming.

- `Fh != 0` → look up the named foothold by id, recompute Y on it via the same slope formula `calcPointBelow` already uses.
- `Fh == 0` and ground mob → `findBelow` from `(X, Y - 1)`, then `calcPointBelow`.
- `Fh == 0` and flying/swimming → leave Y alone.
- `Fh != 0` but the foothold id can't be found (data corruption, link-map quirks) → leave Y alone, log warning.

The change is contained to atlas-data:

1. Surface `Flying` and `Swimming` booleans on the monster `RestModel`, derived from `AnimationTimes` keys at parse time.
2. Add a `findById(uint32)` lookup to `FootholdTreeRestModel`.
3. Apply the snap at `GetMonsters(mapId)` serve time, reading the monster template via the existing in-process monster `Storage`.

No changes to atlas-maps, atlas-monsters, or any consumer.
