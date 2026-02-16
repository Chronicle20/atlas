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
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| fh | int16 | Foothold |
| stance | byte | Animation stance |
| team | int8 | Team assignment |
| damageEntries | []entry | List of damage dealt by characters |
| statusEffects | []StatusEffect | Active status effects on this monster |

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

### DamageSummary

Represents the result of applying damage to a monster.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Monster | Model | Updated monster state |
| VisibleDamage | uint32 | Damage shown to clients |
| ActualDamage | int64 | Actual damage applied |
| Killed | bool | Whether the monster was killed |

### entry

Tracks damage dealt by a character.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character who dealt damage |
| Damage | uint32 | Amount of damage |

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
| resistances | map[string]string | Elemental resistances (element code to resistance level) |
| animationTimes | map[string]uint32 | Animation name to duration in milliseconds |
| skills | []Skill | Mob skills available to this monster |
| revives | []uint32 | Monster IDs to spawn when this monster dies |
| banish | Banish | Banish target configuration |

Resistance values: "1"=immune, "2"=strong, "3"=normal, "4"=weak.

### information.Banish

Banish target configuration for a monster.

| Field | Type | Description |
|-------|------|-------------|
| Message | string | Banish message |
| MapId | uint32 | Target map ID |
| PortalName | string | Target portal name |

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

- Monster uniqueId values range from 1000000000 to 2000000000 per tenant
- A monster is alive when hp > 0
- A monster is controlled when controlCharacterId != 0
- Damage entries accumulate over the monster's lifetime
- The damage leader is the character with the highest total damage dealt
- VENOM status effects stack up to 3; at max stacks, the oldest is replaced
- Non-VENOM status effects replace any existing effect of the same type
- Player-sourced status effects are checked against elemental immunities and boss immunities
- Boss monsters are immune to most crowd-control statuses (stun, seal, freeze, poison) but allow stat modifiers (speed, attack, defense, showdown, ninja ambush, venom)
- Monsters immune to element "P" reject POISON status; immune to element "I" reject FREEZE status
- Sealed monsters cannot use skills
- Immunity and reflect skills cannot be applied if already active on the monster
- DoT damage (poison, venom) cannot kill a monster; damage is capped at currentHP - 1
- Poison damage formula: maxHP / (70 - skillLevel)
- Venom damage equals the stat value on the status effect
- DoT tick interval defaults to 1000ms for POISON and VENOM if not specified
- HP cannot exceed maxHp after healing
- MP deduction is capped at current MP

## State Transitions

### Monster Lifecycle

1. **Created**: Monster spawned in map with initial HP/MP from monster information
2. **Controlled**: Character assigned as controller
3. **Damaged**: HP reduced, damage entries recorded
4. **Control Transferred**: Controller changed based on damage leadership
5. **Killed**: HP reaches 0, cooldowns cleared, active status effects cancelled, monster removed from registry; revive monsters spawned if configured
6. **Destroyed**: Monster removed from registry (manual destruction)

### Control Assignment

- When a monster is created, the service attempts to assign a controller from characters in the map
- The controller candidate is the character controlling the fewest monsters in that field
- When the current controller exits the map, control stops and a new controller is assigned
- When a character becomes the damage leader and is not the current controller, control transfers to them

### Status Effect Lifecycle

1. **Applied**: Status effect created with unique ID, expiry calculated from duration
2. **Ticking**: DoT effects (poison, venom) apply damage each tick interval
3. **Expired**: Status effect removed when current time exceeds expiresAt
4. **Cancelled**: Status effect removed explicitly (by command or on monster death)

### Skill Execution

1. Skill use validated: monster alive, not sealed, skill definition fetched from atlas-data
2. Cooldown checked; MP cost checked and deducted
3. HP threshold checked (skill only activates below configured HP percentage)
4. Probability check applied
5. Stacking check for immunity/reflect (rejected if already active)
6. Animation delay applied if configured
7. Effect executed based on skill category: stat buff, heal, debuff, or summon

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

**Commands:**
- `Create`: Creates a monster in a field, assigns controller, emits created status event
- `StartControl`: Assigns a character as controller, emits start control status event
- `StopControl`: Removes controller assignment, emits stop control status event
- `FindNextController`: Finds and assigns the next controller for a monster
- `Damage`: Applies damage to a monster; checks for damage reflection; may transfer control or kill monster; spawns revive monsters on boss death
- `Move`: Updates monster position and stance
- `UseSkill`: Validates and executes a monster skill (stat buff, heal, debuff, or summon)
- `ApplyStatusEffect`: Applies a status effect to a monster after checking elemental and boss immunities
- `CancelStatusEffect`: Cancels status effects by type from a monster
- `CancelAllStatusEffects`: Cancels all status effects from a monster
- `Destroy`: Removes monster from registry, emits destroyed status event
- `DestroyInField`: Destroys all monsters in a field

### Registry

Singleton in-memory store for monster instances.

**Operations:**
- `CreateMonster`: Creates and stores a new monster instance with an allocated unique ID
- `GetMonster`: Retrieves a monster by tenant and unique ID
- `GetMonstersInMap`: Retrieves all monsters in a field
- `MoveMonster`: Updates monster position
- `ControlMonster`: Assigns a controller to a monster
- `ClearControl`: Removes controller assignment
- `ApplyDamage`: Applies damage and returns damage summary
- `RemoveMonster`: Removes a monster from the registry and releases the unique ID
- `GetMonsters`: Returns all monsters grouped by tenant
- `ApplyStatusEffect`: Applies a status effect to a monster in the registry
- `CancelStatusEffect`: Cancels a status effect by effect ID
- `CancelStatusEffectByType`: Cancels status effects by status type
- `CancelAllStatusEffects`: Cancels all status effects on a monster
- `DeductMp`: Deducts MP from a monster
- `UpdateStatusEffectLastTick`: Updates the last tick time for a status effect
- `UpdateMonster`: Replaces a monster in the registry
- `Clear`: Clears all registry data

### CooldownRegistry

Singleton in-memory store for monster skill cooldowns.

**Operations:**
- `IsOnCooldown`: Checks if a skill is on cooldown for a monster
- `SetCooldown`: Sets a cooldown for a skill on a monster
- `ClearCooldowns`: Clears all cooldowns for a monster

### TenantIdAllocator

Per-tenant unique ID allocator for monster instances. Allocates IDs in the range 1000000000-2000000000. Reuses released IDs via a LIFO free pool.

### StatusExpirationTask

Periodic task (1-second interval) that iterates all monsters across all tenants. Expires status effects past their expiry time and processes DoT ticks for poison and venom effects.

### RegistryAudit

Periodic task (30-second interval) that logs registry statistics (maps tracked, monsters tracked).
