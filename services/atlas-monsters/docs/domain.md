# Monster Domain

## Responsibility

Manages monster instances within game maps, including creation, movement, damage tracking, control assignment, skill execution, status effects, and destruction.

## Core Models

### Model

Represents an active monster instance in a map.

| Field | Type | Description |
|-------|------|-------------|
| uniqueId | uint32 | Unique identifier for this monster instance |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| monsterId | uint32 | Monster type identifier |
| maxHp | uint32 | Maximum hit points |
| hp | uint32 | Current hit points |
| maxMp | uint32 | Maximum magic points |
| mp | uint32 | Current magic points |
| controlCharacterId | uint32 | Character ID controlling this monster (0 if uncontrolled) |
| controllerHasAggro | bool | Whether the controller is actively engaged with the monster (vs. idle/passive) |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| fh | int16 | Foothold |
| stance | byte | Animation stance |
| team | int8 | Team assignment |
| damageEntries | []entry | List of damage dealt by characters |
| statusEffects | []StatusEffect | Active status effects on this monster |
| nextSkillDecision | nextSkillDecision | Picker's current next-skill decision (skill choice is in-memory only; see Skill Picker) |
| lastDamageTakenMs | int64 | Unix millis of the last damage applied to this monster (drives HP recovery gating) |

### StatusEffect

Represents an active status effect on a monster.

| Field | Type | Description |
|-------|------|-------------|
| effectId | uuid.UUID | Unique effect identifier |
| sourceType | string | Source type: "PLAYER_SKILL" or "MONSTER_SKILL" |
| sourceCharacterId | uint32 | Character who applied the effect (0 for monster skills) |
| sourceSkillId | uint32 | Skill that applied the effect |
| sourceSkillLevel | uint32 | Level of the skill that applied the effect |
| statuses | map[string]int32 | Status type to value mapping |
| duration | time.Duration | Total effect duration |
| tickInterval | time.Duration | Interval between DoT ticks (0 for non-ticking effects) |
| lastTick | time.Time | Time of last DoT tick |
| createdAt | time.Time | When the effect was applied |
| expiresAt | time.Time | When the effect expires |
| reflectKind | string | Reflect classification ("PHYSICAL"/"WEAPON_COUNTER" or "MAGICAL"/"MAGIC_COUNTER" kind); empty for non-reflect effects |
| reflectPercent | int32 | Reflect damage percentage |
| reflectLtX | int16 | Reflect bounding box left-top X |
| reflectLtY | int16 | Reflect bounding box left-top Y |
| reflectRbX | int16 | Reflect bounding box right-bottom X |
| reflectRbY | int16 | Reflect bounding box right-bottom Y |
| reflectMaxDamage | int32 | Maximum damage the reflect can return |

A status effect is a reflect effect when `reflectKind` is non-empty (`IsReflect()`).

### DamageSummary

Represents the result of applying damage to a monster.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Monster | Model | Updated monster state |
| VisibleDamage | uint32 | Damage shown to clients |
| ActualDamage | int64 | Actual damage applied |
| Killed | bool | Whether the monster was killed |
| WasFirstHit | bool | Whether this is the first hit landed on a controlled monster (flips controllerHasAggro) |

### entry

Tracks damage dealt by a character.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Damage | uint32 | Amount of damage |
| LastHitMs | int64 | Unix millis of this character's last hit (drives aggro decay) |

### MapKey

Composite key for map-scoped monster lookups.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | byte | World identifier |
| ChannelId | byte | Channel identifier |
| MapId | uint32 | Map identifier |
| Instance | uuid.UUID | Map instance identifier |

### MonsterKey

Composite key for monster lookups.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| MonsterId | uint32 | Monster unique identifier |

### information.Model

Monster type information retrieved from atlas-data.

| Field | Type | Description |
|-------|------|-------------|
| hp | uint32 | Base hit points |
| mp | uint32 | Base magic points |
| boss | bool | Whether this is a boss monster |
| undead | bool | Whether this is an undead monster |
| friendly | bool | Whether this is a friendly monster |
| weaponAttack | uint32 | Base weapon attack |
| dropPeriod | uint32 | Drop period in milliseconds (friendly monsters only) |
| resistances | map[string]string | Elemental resistances (element code to resistance level) |
| animationTimes | map[string]uint32 | Animation name to duration in milliseconds |
| skills | []Skill | Mob skills available to this monster |
| revives | []uint32 | Monster IDs to spawn when this monster dies |
| banish | Banish | Banish target configuration |
| attacks | []AttackInfo | Basic-attack metadata per attack position |
| hpRecovery | uint32 | HP recovered per recovery task tick |
| mpRecovery | uint32 | MP recovered per recovery task tick |

Resistance values: "1"=immune, "2"=strong, "3"=normal, "4"=weak.

### information.Banish

Banish target configuration for a monster.

| Field | Type | Description |
|-------|------|-------------|
| Message | string | Banish message |
| MapId | uint32 | Target map ID |
| PortalName | string | Target portal name |

### information.AttackInfo

Basic-attack metadata for one attack position, retrieved from atlas-data.

| Field | Type | Description |
|-------|------|-------------|
| Pos | uint8 | Attack position (1-indexed in atlas-data; wire/registry attackPos is 0-indexed) |
| ConMP | int32 | MP cost of the basic attack |
| AttackAfter | int32 | Cooldown in milliseconds before the attack position can be used again |

### drop.Model

Monster drop definition retrieved from atlas-drops.

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Item ID (0 for meso drops) |
| minimumQuantity | uint32 | Minimum drop quantity |
| maximumQuantity | uint32 | Maximum drop quantity |
| questId | uint32 | Associated quest ID (0 for non-quest drops) |
| chance | uint32 | Drop chance out of 999999 |

### DropTimerEntry

Tracks drop timer state for a friendly monster.

| Field | Type | Description |
|-------|------|-------------|
| monsterId | uint32 | Monster type identifier |
| field | field.Model | Field where the monster resides |
| dropPeriod | time.Duration | Interval between drops (dropPeriod / 3) |
| weaponAttack | uint32 | Monster weapon attack (for friendly damage calculation) |
| maxHp | uint32 | Monster max HP (for friendly damage calculation) |
| lastDropAt | time.Time | Time of last drop emission |
| lastHitAt | time.Time | Time of last hit received |

### mobskill.Model

Mob skill definition retrieved from atlas-data.

| Field | Type | Description |
|-------|------|-------------|
| skillId | uint16 | Skill type identifier |
| level | uint16 | Skill level |
| mpCon | uint32 | MP cost |
| duration | uint32 | Effect duration in seconds |
| hp | uint32 | Max HP percentage threshold to use skill |
| x | int32 | Skill value (damage, heal amount, stat modifier) |
| y | int32 | Secondary skill value |
| prop | uint32 | Activation probability (0-100) |
| interval | uint32 | Cooldown interval in seconds |
| count | uint32 | Target count limit |
| limit | uint32 | Summon limit (max monsters in field) |
| ltX | int32 | Bounding box left-top X |
| ltY | int32 | Bounding box left-top Y |
| rbX | int32 | Bounding box right-bottom X |
| rbY | int32 | Bounding box right-bottom Y |
| summonEffect | uint32 | Summon visual effect |
| summons | []uint32 | Monster IDs to summon |

## Invariants

- Monster uniqueId values are allocated from the shared per-tenant object-id pool, range 1,000,000 to 2,147,483,647 (see docs/storage.md ID Allocation)
- Friendly monsters with a non-zero dropPeriod are registered in the drop timer on creation
- A monster is alive when hp > 0
- A monster is controlled when controlCharacterId != 0
- controllerHasAggro flips true on the first damage hit landed on a controlled monster; it is not cleared when a controller is assigned (spawn/control-change) and is only cleared by aggro decay or monster death
- Damage entries accumulate over the monster's lifetime, aggregated per character, and record each character's last-hit timestamp
- The damage leader is the character with the highest total damage dealt
- VENOM status effects stack up to 3; at max stacks, the oldest is replaced
- Non-VENOM status effects replace any existing effect of the same type
- Player-sourced status effects are checked against elemental immunities and boss immunities; DOOM bypasses elemental immunity
- Boss monsters are immune to most crowd-control statuses (stun, seal, freeze, poison) but allow stat modifiers (speed, attack, defense, showdown, ninja ambush, venom)
- Monsters immune to element "P" reject POISON status; immune to element "I" reject FREEZE status
- Sealed monsters cannot use skills
- Immunity and reflect skills cannot be applied if already active on the monster
- WEAPON_ATTACK_IMMUNE and MAGIC_ATTACK_IMMUNE are mutually exclusive; applying one cancels the other if currently active on the target
- Stat-buff and heal skills with a bounding box apply to the caster plus every other monster in the same field whose offset from the caster falls within the box (AoE)
- Debuff skills target the controlling character when the skill has no bounding box and a count of at most 1; otherwise they target every character in the field, capped and randomly sampled to the skill's count when set
- DISPEL (skill type Dispel) cancels all buffs on its targets instead of applying a status; BANISH (skill type Banish) warps its targets to the monster's configured banish map instead of applying a status, and is a no-op when no banish map is configured
- AREA_POISON is dispatched as a MIST_CREATE command to atlas-maps rather than a direct status apply; its duration is capped at 60,000ms server-side
- A monster reflects damage back to the attacking character when it holds an active WEAPON_COUNTER (non-magic attacks) or MAGIC_COUNTER (magic attacks) status; reflect is checked once per attack, not once per damage line
- A CANCEL_STATUS/CANCEL_STATUS_FIELD command carrying a non-empty sourceSkillClass is refused entirely if the monster has an active same-kind reflect (WEAPON_COUNTER for "PHYSICAL", MAGIC_COUNTER for "MAGICAL"), unless every requested status type is itself a reflect status
- DoT damage (poison, venom) cannot kill a monster; damage is capped at currentHP - 1
- Poison damage formula: maxHP / (70 - skillLevel)
- Venom damage equals the stat value on the status effect
- DoT tick interval defaults to 1000ms for POISON and VENOM if not specified
- HP cannot exceed maxHp after healing
- MP deduction is capped at current MP
- Basic monster attacks (UseBasicAttack) deduct MP and register a per-attack-position cooldown from atlas-data attack metadata (ConMP, AttackAfter); rejected silently if the monster has no attack info for the position, is on cooldown for that position, or has insufficient MP
- Drain MP (MP_EATER) is a no-op for boss monsters and for monsters with MaxMp == 0
- Friendly monster damage formula: rand.Intn(((maxHp/13 + weaponAttack*10) * 2) + 500) / 10, minimum 1
- Friendly drops skip quest-specific drops (questId != 0)
- Drop timer next eligible time is lastHitAt + dropPeriod if hit since last drop, otherwise lastDropAt + dropPeriod
- A player's puppet biases controller-candidate selection toward the puppet's owner when the puppet lies within squared-distance 177777 of the monster being assigned
- HP recovery applies only when more than 10 seconds (AggroIdleThresholdMs) have elapsed since the monster's last damage taken; MP recovery is unconditional; recovery is skipped entirely for dead monsters (hp == 0)
- Non-boss monsters' idle damage entries (no hit for 10 seconds) decay by 15% per 1.5-second sweep tick and are pruned once their value falls below 1; boss monsters are excluded from aggro decay and retain their damage table until death

## State Transitions

### Monster Lifecycle

1. **Created**: Monster spawned in map with initial HP/MP from monster information; friendly monsters with a configured drop period are registered in the drop timer
2. **Controlled**: Character assigned as controller (initial assignment is applied in-place without emitting START_CONTROL, so the channel's Spawn packet always precedes Control; subsequent control changes go through StartControl/StopControl and do emit)
3. **Damaged**: HP reduced, per-character damage entries updated; a DAMAGED event is always emitted, and AGGRO_CHANGED is emitted on a monster's first hit when the controller does not change
4. **Control Transferred**: Controller changed when a character other than the current controller becomes the damage leader while present in the monster's field
5. **Killed**: HP reaches 0; cooldowns (skill and basic-attack) and the drop timer are cleared, active status effects are cancelled (each emitting STATUS_CANCELLED), monster removed from registry; monsters configured with revives spawn their revive monster IDs at the same position (friendly-monster deaths via DamageFriendly do not spawn revives)
6. **Destroyed**: Monster removed from registry (manual destruction); drop timer and attack cooldowns cleared

### Control Assignment

- When a monster is created, the service attempts to assign a controller from characters in the map
- The controller candidate is the owner of an in-vicinity puppet if one exists among the field's characters; otherwise it is the character controlling the fewest monsters in that field
- When the current controller exits the map, control stops and a new controller is assigned
- When a character becomes the damage leader and is not the current controller, control transfers to them, provided the character is currently present in the monster's field
- A controller-change (StartControl) triggers a picker re-pick only when the new controller has aggro; a spawn-time controller assignment does not trigger a re-pick (controllerHasAggro is always false at spawn)

### Status Effect Lifecycle

1. **Applied**: Status effect created with unique ID, expiry calculated from duration
2. **Ticking**: DoT effects (poison, venom) apply damage each tick interval
3. **Expired**: Status effect removed when current time exceeds expiresAt
4. **Cancelled**: Status effect removed explicitly (by command or on monster death)

### Skill Execution

1. Skill use validated: monster alive, not sealed, skill definition fetched from atlas-data
2. Cooldown checked; MP cost checked and deducted (deduction emits MP_CHANGED with reason SKILL_CAST)
3. HP threshold checked (skill only activates below configured HP percentage)
4. Cooldown registered for the skill if it defines an interval
5. Stacking check for immunity/reflect (rejected if already active)
6. Animation delay applied if configured; the effect and post-execute picker re-pick run after the delay only if the monster is still alive
7. Effect executed: AREA_POISON dispatches a MIST_CREATE command regardless of category; otherwise stat-buff/immunity/reflect, heal, debuff (including the Dispel and Banish special cases), and summon are dispatched by skill category
8. After execution, the picker re-picks and emits a new decision if the monster still has aggro (see Skill Picker)

`UseSkillGM` runs the same category dispatch without the cooldown/MP/HP-threshold/probability/seal checks (used for field-wide GM skill commands).

### Skill Picker

The picker predicts which skill a monster will cast next so atlas-channel can pre-stage the animation, without waiting for a live cast. It is pure (no side effects) and re-run by `RepickAndEmit` on every trigger below, always emitting a NEXT_SKILL_DECIDED event even when the decision is unchanged or the sentinel (SkillId == 0, "no skill"):

1. If the monster's template has no skills, or the monster is sealed, the sentinel decision is returned immediately.
2. Skills are evaluated in template order; the first eligible skill whose probability roll (`prop`, out of 100) succeeds wins. A skill is eligible only if: it is not on cooldown, the monster's HP% is at or below the skill's HP threshold (0 = always eligible), the monster has enough MP, and — for immunity/reflect skills — the matching status is not already active.
3. If no skill is chosen, `nextEligibleRepickAtMs` is set to the soonest cooldown expiry among cooldown-gated candidates; if at least one candidate passed every other gate but failed its probability roll, a sweep-cadence-based fallback repick time (`now + sweep interval`) is also considered, and the minimum of the two is kept.
4. Triggers: spawn (only if the monster already has aggro), post-use-skill (after the animation delay), damaged (on first hit, or when HP% changes and the monster wasn't killed), status-applied/status-expired/status-cancelled (only for picker-relevant statuses: SEAL, SEAL_SKILL, WEAPON_ATTACK_IMMUNE, MAGIC_ATTACK_IMMUNE, WEAPON_COUNTER, MAGIC_COUNTER), control-change (only if the new controller has aggro), and sweep (see MonsterSkillPickerSweepTask).
5. Only `nextEligibleRepickAtMs` persists to Redis across a decision; the chosen skillId/skillLevel/decidedAtMs are in-memory only and rebuilt by the next picker run.

## Processors

### Processor

Interface defining monster processing operations.

**Providers:**
- `ByIdProvider`: Provides a monster by unique ID
- `ByFieldProvider`: Provides all monsters in a field
- `ControlledInFieldProvider`: Provides controlled monsters in a field
- `NotControlledInFieldProvider`: Provides uncontrolled monsters in a field
- `ControlledByCharacterInFieldProvider`: Provides monsters controlled by a specific character in a field

**Queries:**
- `GetById`: Retrieves a monster by unique ID
- `GetInField`: Retrieves all monsters in a field
- `GetInFieldRect`: Retrieves monsters in a field within a rectangle, sorted by ascending squared distance from the rectangle center, optionally capped to a limit

**Commands:**
- `Create`: Creates a monster in a field, assigns controller, emits created status event; registers a drop timer for friendly monsters with a configured drop period; fires the picker if the monster spawns with aggro
- `StartControl`: Assigns a character as controller, emits start control status event; re-picks the skill decision if the new controller has aggro
- `StopControl`: Removes controller assignment, emits stop control status event
- `FindNextController`: Finds and assigns the next controller for a monster
- `Damage`: Applies a sequence of damage lines to a monster; checks for damage reflection once per attack; may transfer control, flip controllerHasAggro, or kill the monster; spawns configured revive monsters on death
- `DamageFriendly`: Applies damage from a hostile monster to a friendly monster; resets the drop timer hit timestamp; uses attacker's info for damage calculation
- `Move`: Updates monster position and stance
- `UseSkill`: Validates and executes a monster skill (stat buff, immunity, reflect, heal, debuff/dispel/banish, summon, or area-effect mist)
- `UseSkillGM`: Executes a mob skill on a monster without validation checks (no cooldown, MP, HP threshold, probability, or seal checks)
- `UseBasicAttack`: Applies the post-conditions of a basic monster attack (MP deduction, per-position cooldown registration) after atlas-channel has already optimistically applied the attack
- `ApplyStatusEffect`: Applies a status effect to a monster after checking elemental and boss immunities (player-sourced effects only); triggers a picker re-pick if the effect is picker-relevant
- `CancelStatusEffect`: Cancels status effects by type from a monster
- `CancelStatusEffectGuarded`: Cancels status effects, refusing the cancel when a non-empty sourceSkillClass targets a monster with an active same-kind reflect (unless every requested type is itself a reflect status)
- `CancelAllStatusEffects`: Cancels all status effects from a monster
- `RepickAndEmit`: Re-runs the skill picker for a monster and emits a NEXT_SKILL_DECIDED event (see Skill Picker)
- `DrainMp`: Emits an MP_CHANGED event for a player MP-Eater proc, deducting MP from the monster when possible; no-op for boss monsters or monsters with MaxMp == 0
- `Destroy`: Removes monster from registry, clears its drop timer and attack cooldowns, emits destroyed status event
- `DestroyInField`: Destroys all monsters in a field

### Registry

Singleton Redis-backed store for monster instances.

**Operations:**
- `CreateMonster`: Creates and stores a new monster instance with an allocated unique ID
- `GetMonster`: Retrieves a monster by tenant and unique ID
- `GetMonstersInMap`: Retrieves all monsters in a field
- `MoveMonster`: Updates monster position
- `ControlMonster`: Assigns a controller to a monster
- `ClearControl`: Removes controller assignment
- `ApplyDamage`: Applies damage, aggregates the per-character damage entry, stamps lastDamageTakenMs, and flips controllerHasAggro on first hit; returns a damage summary
- `ApplyRecovery`: Applies HP recovery (gated by the idle-since-last-damage window) and MP recovery (unconditional) to a monster; returns the updated monster and per-stat applied flags
- `DecayDamageEntries`: Decays and prunes idle damage entries for aggro decay; flips controllerHasAggro false when the entry list empties
- `RemoveMonster`: Removes a monster from the registry and releases the unique ID
- `GetMonsters`: Returns all monsters grouped by tenant
- `ApplyStatusEffect`: Applies a status effect to a monster in the registry
- `CancelStatusEffect`: Cancels a status effect by effect ID
- `CancelStatusEffectByType`: Cancels status effects by status type
- `CancelAllStatusEffects`: Cancels all status effects on a monster
- `DeductMp`: Deducts MP from a monster
- `UpdateStatusEffectLastTick`: Updates the last tick time for a status effect
- `SetNextSkillDecision`: Replaces a monster's picker decision (only nextEligibleRepickAtMs persists)
- `UpdateMonster`: Replaces a monster in the registry
- `Clear`: Clears all registry data

### CooldownRegistry

Singleton Redis-backed store for monster skill cooldowns.

**Operations:**
- `IsOnCooldown`: Checks if a skill is on cooldown for a monster
- `SetCooldown`: Sets a cooldown for a skill on a monster
- `Remaining`: Returns the time remaining on a skill's cooldown (used by the picker to schedule a repick)
- `ClearCooldowns`: Clears all cooldowns for a monster

### AttackCooldownRegistry

Singleton Redis-backed store for per-attack-position basic-attack cooldowns, keyed by (monsterId, attackPos).

**Operations:**
- `IsOnCooldown`: Checks if an attack position is on cooldown for a monster
- `SetCooldown`: Sets a cooldown for an attack position (no-op for a zero duration)
- `ClearCooldowns`: Clears all attack-position cooldowns for a monster

### PuppetRegistry

Singleton Redis-backed store, per field, of player puppets' owner and position — used to bias controller-candidate selection toward a puppet's owner.

**Operations:**
- `Add`: Records (or replaces) the puppet for an owner in a field
- `Remove`: Deletes the puppet for an owner in a field
- `GetInField`: Returns every puppet registered in a field
- `VicinityOwner`: Returns the nearest puppet owner within the vicinity distance threshold of a position, if any
- `Clear`: Removes all puppet state

### DropTimerRegistry

Singleton Redis-backed store for friendly monster drop timers.

**Operations:**
- `Register`: Registers a friendly monster for periodic drop emission
- `Unregister`: Removes a friendly monster from the drop timer
- `RecordHit`: Updates the last hit timestamp for a friendly monster
- `UpdateLastDrop`: Updates the last drop timestamp for a friendly monster
- `GetAll`: Returns all registered drop timer entries

### IdAllocator

Wraps the shared per-tenant object-id allocator (`libs/atlas-object-id`) used for monster unique IDs. Allocates sequential IDs starting at 1,000,000, reuses released IDs via a LIFO free pool once the counter approaches the 2,147,483,647 ceiling (see docs/storage.md ID Allocation).

### StatusExpirationTask

Periodic task (1-second interval) that iterates all monsters across all tenants. Expires status effects past their expiry time (emitting STATUS_EXPIRED and, for picker-relevant statuses, re-running the picker) and processes DoT ticks for poison and venom effects.

### DropTimerTask

Periodic task (1-second interval) that iterates all registered drop timer entries. For each entry whose drop period has elapsed, fetches the monster's drop table from atlas-drops, rolls for drops, emits spawn drop commands for successful rolls, and emits a FRIENDLY_DROP status event.

### MonsterSkillPickerSweepTask

Periodic task (1.5-second interval, `MonsterSkillPickerSweepInterval`) that scans all live monsters and re-runs the skill picker (see Skill Picker) for any monster whose `nextEligibleRepickAtMs` has elapsed, that currently has aggro, and whose template has at least one skill.

### MonsterAggroDecayTask

Periodic task (1.5-second interval, `AggroSweepInterval`) that decays idle damage entries on non-boss monsters (see Invariants) and emits AGGRO_CHANGED when decay empties a monster's entry list while its controller had aggro.

### MonsterRecoveryTask

Periodic task (10-second interval, `MonsterRecoveryInterval`) that applies HP/MP recovery to all live monsters whose HP or MP is below maximum, using atlas-data's hpRecovery/mpRecovery values, per the gating rules in Invariants.

### RegistryAudit

Periodic task (30-second interval) that logs registry statistics (maps tracked, monsters tracked).
