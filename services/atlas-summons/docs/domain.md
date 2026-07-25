# Summon Domain

## Responsibility

Manages the lifecycle of summon instances cast by characters — puppets,
attacker summons, and the Beholder buff-aura summon — including spawning,
movement, attack relay with a faithful damage ceiling, damage absorption
(puppets), and despawning (manual, expiry, or owner logout/channel-change/map-change).

## Core Models

### Model

Represents an active summon instance.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique identifier for this summon instance |
| ownerCharacterId | uint32 | Character who cast the summon |
| skillId | uint32 | Skill id that spawned the summon |
| skillLevel | byte | Level of the casting skill |
| summonType | SummonType | PUPPET, ATTACKER, or BUFF_AURA |
| movementType | MovementType | Stationary, Follow, or CircleFollow |
| fld | field.Model | World/channel/map/instance the summon occupies |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| stance | byte | Animation stance |
| hp | int32 | Current hit points (puppets and the Beholder only; 0 for other types) |
| maxHp | int32 | Maximum hit points |
| animated | bool | Whether the despawn plays an animation |
| spawnTime | time.Time | Time the summon was created |
| expiresAt | time.Time | Time the summon's duration elapses |
| nextHealAt | time.Time | Beholder-only: next scheduled heal tick |
| nextBuffAt | time.Time | Beholder-only: next scheduled buff tick |
| healAmount | int16 | Beholder-only: HP restored to the owner per heal tick |
| healInterval | time.Duration | Beholder-only: interval between heal ticks |
| buffInterval | time.Duration | Beholder-only: interval between buff ticks |
| buffSourceId | int32 | Beholder-only: the buff's source skill id (HEX_OF_THE_BEHOLDER) |
| buffLevel | byte | Beholder-only: level of the buff skill |
| buffDuration | int32 | Beholder-only: duration applied to each buff tick |
| buffChanges | []StatChange | Beholder-only: pool of stat deltas the buff can apply |

All non-Beholder-only fields apply to every summon type; the Beholder-only
fields are zero-valued for PUPPET and ATTACKER summons.

### StatChange

One `{stat-type, amount}` buff delta snapshotted from the Beholder's
HEX_OF_THE_BEHOLDER effect at spawn.

| Field | Type | Description |
|-------|------|-------------|
| Type | string | Stat type |
| Amount | int32 | Stat delta |

### SummonType

Enumerates the summon classification: `PUPPET`, `ATTACKER`, `BUFF_AURA`.

### MovementType

Enumerates the on-the-wire movement classification: `MovementStationary` (0),
`MovementFollow` (1), `MovementCircleFollow` (3).

### AttackTarget

One `{monster, reported damage}` pair from a summon-attack relay.

| Field | Type | Description |
|-------|------|-------------|
| MonsterId | uint32 | Target monster's unique id |
| Damage | uint32 | Client-reported damage (server-clamped before use) |

### skill.Model

Projection of an atlas-data skill resource.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill id |
| action | bool | Action flag |
| element | string | Elemental attribute |
| animationTime | uint32 | Animation duration in milliseconds |
| effects | []effect.Model | Per-level effect data |

### effect.Model

Projection of a skill-effect resource; supplies the summon's HP/duration,
the damage ceiling's weapon/magic attack values, the proc chance and monster
status map applied on hit, and the Beholder's heal/buff snapshot inputs.

| Field | Type | Description |
|-------|------|-------------|
| weaponAttack | int16 | Effect's weapon attack attribute |
| magicAttack | int16 | Effect's magic attack attribute |
| hp | uint16 | Heal amount (used by AURA_OF_THE_BEHOLDER) |
| duration | int32 | Effect duration in milliseconds (-1 = no duration) |
| x | int16 | Integer X attribute (puppet HP; AURA_OF_THE_BEHOLDER heal interval seconds) |
| y | int16 | Integer Y attribute |
| prop | float64 | Proc chance, 0.0-1.0 |
| monsterStatus | map[string]uint32 | Status-effect map applied by the skill on hit |
| statups | []StatChange | Buff stat deltas (HEX_OF_THE_BEHOLDER) |

### effectivestats.Model

Projection of a character's session-effective combat stats, used by the
summon damage ceiling.

| Field | Type | Description |
|-------|------|-------------|
| strength | uint32 | Effective strength |
| dexterity | uint32 | Effective dexterity |
| luck | uint32 | Effective luck |
| intelligence | uint32 | Effective intelligence |
| weaponAttack | uint32 | Effective weapon attack |
| magicAttack | uint32 | Effective magic attack |

## Invariants

- A cast skill not present in the summon roster is a graceful no-op; nothing spawns
- Casting a summon removes the owner's existing summon of the same skill, and any existing owned summon whose mobility class (stationary vs. non-stationary) conflicts with the new one
- Puppet HP/maxHp is the resolved skill effect's X attribute; Beholder HP/maxHp is the effect's X attribute plus 1; all other summon types spawn with HP/maxHp 0
- Only puppets absorb monster-reported damage; attacker summons and the Beholder are immune to damage reports and are not despawned by them
- A puppet despawns automatically when its HP reaches 0
- A summon despawns when its ExpiresAt time elapses, when its owner logs out, changes channel, or changes map, or when Spawn evicts it for a same-skill or mobility conflict
- A summon flagged one-shot in the roster (Gaviota) self-cancels after a single attack
- Move, Attack, and Damage commands are honored only for a summon the sending character owns; a missing summon or non-owner sender is a graceful no-op
- The wire summon identity is resolved as a real summon id first; if that id does not exist or its owner does not match the sender, it falls back to the sender's owned summons (v83/v87 send the owner's character id in place of a summon id) — the Move/Attack fallback prefers the first non-puppet, the Damage fallback prefers the puppet
- Summon attack damage is clamped to a faithful per-hit ceiling (weapon-type-aware for physical, INT-curve-based for magic) computed from the owner's session-effective stats and the skill effect's weapon/magic attack; a report exceeding the ceiling is clamped to the ceiling and logged, not rejected
- If the owner's effective stats are unavailable, the damage ceiling is disabled for that hit (no clamp) rather than zeroing damage
- A failed equipped-weapon-type lookup degrades to the one-handed-sword fallback rather than disabling the physical damage ceiling
- Monster status (stun/freeze) applied by a summon attack is gated by the skill effect's proc chance: a prop of 0 or unset always applies, a prop of 1.0 or greater always applies, otherwise a uniform random draw gates it
- Only a BUFF_AURA (Beholder) summon carries a heal/buff snapshot; the heal amount/interval are resolved from AURA_OF_THE_BEHOLDER at the caster-supplied aura level, and the buff stat pool/interval/duration/level are resolved from HEX_OF_THE_BEHOLDER at the caster-supplied hex level, both at spawn time
- The Beholder buff's sourceId is the real (non-negated) HEX_OF_THE_BEHOLDER skill id
- Summon ids are allocated from the shared per-tenant object-id pool (see docs/storage.md)

## State Transitions

### Summon Lifecycle

1. **Spawned**: the cast skill is classified against the roster; any existing owned summon that conflicts (same skill or same mobility class) is despawned; the skill effect is resolved for HP/duration; for a Beholder, the aura heal and hex buff snapshots are resolved; the summon is persisted, a CREATED event is emitted, and a puppet is additionally registered with atlas-monsters (ADD_PUPPET)
2. **Moved**: ownership is verified; position/stance are updated; a MOVED event carrying the raw movement bytes is emitted
3. **Attacked**: ownership is verified; each reported target's damage is clamped to the ceiling; the owner is credited with the clamped damage via a monster DAMAGE command; monster status (stun/freeze) is applied per the roster/effect and proc chance; an ATTACKED event carrying the clamped targets is emitted; a one-shot summon (Gaviota) self-despawns afterward
4. **Damaged** (puppets only): ownership is verified; HP is decremented by the reported amount (clamped to [0, maxHp]); a DAMAGED event is emitted; the summon is despawned when HP reaches 0
5. **Despawned**: the summon is removed from the registry, its oid is released, a DESTROYED event is emitted, and a puppet additionally has its atlas-monsters registration cleared (REMOVE_PUPPET)
6. **Expired**: the periodic expiry sweep despawns (animated) any summon whose ExpiresAt has passed

### Beholder Aura Sweep

Runs once per due interval per deployed Beholder summon, independently for heal and buff:

1. **Heal tick**: when NextHealAt is due, a CHANGE_HP command heals the owner by HealAmount and a SKILL status event (stance 6) is emitted for client-side visual playback; NextHealAt advances by HealInterval
2. **Buff tick**: when NextBuffAt is due, one randomly-chosen stat delta from the snapshot pool is applied to the owner via an atlas-buffs APPLY command with per-stat accumulation, and a SKILL status event (stance 6) is emitted; NextBuffAt advances by BuffInterval
3. A zero-valued heal or buff interval is never due and is skipped
4. A despawned Beholder (removed from the registry) is never swept again

### Owner Despawn Cascade

Logout, channel-change, and map-change character-status events each despawn
every summon owned by the character (unanimated); summons do not follow their
owner across these transitions.

## Processors

### Processor (summon)

Interface defining summon processing operations.

**Operations:**
- `GetById`: retrieves a summon by id
- `GetInField`: retrieves all summons in a field
- `Spawn`: classifies the cast skill, evicts conflicting owned summons, resolves skill/aura/hex effect data, persists the summon, and emits CREATED (and ADD_PUPPET for puppets)
- `Move`: relays an owner's move, persists the new position/stance, emits MOVED
- `Attack`: relays an owner's attack, clamps and credits damage, applies monster status, emits ATTACKED, self-despawns one-shot summons
- `Damage`: applies monster-reported damage to a puppet, emits DAMAGED, despawns at 0 HP
- `Despawn`: removes a summon, releases its oid, emits DESTROYED (and REMOVE_PUPPET for puppets)
- `DespawnAllForOwner`: despawns every summon owned by a character

### Registry (summon)

Singleton Redis-backed store for summon instances.

**Operations:**
- `Put`: stores a summon and indexes it by field and by owner
- `Get`: retrieves a summon by id
- `GetInField`: retrieves all summons indexed under a field
- `GetByOwner`: retrieves all summons indexed under an owner
- `Update`: applies a mutation function to a stored summon
- `Remove`: removes a summon and its field/owner index entries
- `GetAll`: returns every stored summon grouped by tenant

### IdAllocator (summon)

Wraps the shared per-tenant object-id allocator (`libs/atlas-object-id`) so
summons share the per-tenant oid namespace with monsters, reactors, and drops.

### skill.Processor (data/skill)

Fetches skill data and per-level skill-effect data from atlas-data.

**Operations:**
- `GetById`: retrieves a skill by id
- `GetEffect`: retrieves a skill's effect data for a given level

### effectivestats.Processor

Fetches a character's session-effective combat stats from atlas-effective-stats.

**Operations:**
- `GetByCharacter`: retrieves effective stats for a character in a world/channel

### inventory.Processor

Resolves a character's equipped main-weapon type from atlas-inventory.

**Operations:**
- `GetEquippedWeaponType`: returns the weapon type in the equip compartment's weapon slot, or `item.WeaponTypeNone` if unequipped or the lookup fails

### FaithfulMaxPerHit (damage ceiling)

Computes the v83-era per-hit summon damage ceiling from the owner's
session-effective stats, equipped weapon type, and the summon skill effect's
weapon/magic attack, branching on magic vs. physical and, for physical,
weapon-type-specific main/secondary stat selection and damage multiplier.

### BeholderTask

Periodic task (1-second interval) that runs the Beholder Aura Sweep (see
State Transitions) across every deployed Beholder summon for every tenant.
Runs only on the leader-elected pod.

### ExpiryTask

Periodic task (1-second interval) that despawns every summon across every
tenant whose ExpiresAt has passed. Runs only on the leader-elected pod.
