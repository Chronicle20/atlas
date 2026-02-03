# Domain

## stat

### Responsibility

Defines stat types, bonuses, base stats, and computed effective stats.

### Core Models

#### Type

Identifies which stat a bonus affects.

| Value | Description |
|-------|-------------|
| `strength` | Strength |
| `dexterity` | Dexterity |
| `luck` | Luck |
| `intelligence` | Intelligence |
| `max_hp` | Maximum HP |
| `max_mp` | Maximum MP |
| `weapon_attack` | Physical attack |
| `weapon_defense` | Physical defense |
| `magic_attack` | Magic attack |
| `magic_defense` | Magic defense |
| `accuracy` | Accuracy |
| `avoidability` | Avoidability |
| `speed` | Movement speed |
| `jump` | Jump height |

#### Bonus

Represents a single contribution to a stat from a source.

| Field | Type | Description |
|-------|------|-------------|
| source | string | Source identifier (e.g., `equipment:12345`, `passive:1000001`, `buff:2311003`) |
| statType | Type | Which stat this bonus affects |
| amount | int32 | Flat bonus value |
| multiplier | float64 | Percentage bonus (additive multipliers, e.g., 0.10 = +10%) |

#### Base

Holds base stats fetched from atlas-character.

| Field | Type | Description |
|-------|------|-------------|
| strength | uint16 | Base strength |
| dexterity | uint16 | Base dexterity |
| luck | uint16 | Base luck |
| intelligence | uint16 | Base intelligence |
| maxHP | uint16 | Base maximum HP |
| maxMP | uint16 | Base maximum MP |

#### Computed

Holds all computed effective stats for a character.

| Field | Type | Description |
|-------|------|-------------|
| strength | uint32 | Effective strength |
| dexterity | uint32 | Effective dexterity |
| luck | uint32 | Effective luck |
| intelligence | uint32 | Effective intelligence |
| maxHP | uint32 | Effective maximum HP |
| maxMP | uint32 | Effective maximum MP |
| weaponAttack | uint32 | Effective physical attack |
| weaponDefense | uint32 | Effective physical defense |
| magicAttack | uint32 | Effective magic attack |
| magicDefense | uint32 | Effective magic defense |
| accuracy | uint32 | Effective accuracy |
| avoidability | uint32 | Effective avoidability |
| speed | uint32 | Effective movement speed |
| jump | uint32 | Effective jump height |

### Invariants

- Bonus source strings follow the pattern `<type>:<id>` where type is `equipment`, `buff`, or `passive`.
- Multipliers are additive (e.g., two +10% buffs = +20% total).
- Computed stats are non-negative; negative results are clamped to zero.

---

## character

### Responsibility

Manages the in-memory effective stats model for characters, including bonus tracking, computation, and registry storage.

### Core Models

#### Model

Holds all stat bonuses and computed effective stats for a character.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| characterId | uint32 | Character identifier |
| baseStats | stat.Base | Base stats from character service |
| bonuses | []stat.Bonus | All active bonuses |
| computed | stat.Computed | Cached computed totals |
| lastUpdated | time.Time | Timestamp of last computation |
| initialized | bool | Whether lazy initialization has completed |

### Invariants

- Models are immutable; all modifications return new instances.
- Duplicate bonuses (same source and stat type) are replaced, not accumulated.
- Computed stats are recomputed after any bonus change.
- The initialized flag prevents recursive initialization during lazy load.

### Processors

#### ProcessorImpl

Provides operations for managing character effective stats.

| Method | Description |
|--------|-------------|
| GetEffectiveStats | Retrieves computed effective stats and bonuses; performs lazy initialization if needed |
| AddBonus | Adds or updates a flat stat bonus |
| AddMultiplierBonus | Adds or updates a percentage stat bonus |
| RemoveBonus | Removes a specific stat bonus |
| RemoveBonusesBySource | Removes all bonuses from a source |
| SetBaseStats | Sets base stats and recomputes |
| AddEquipmentBonuses | Adds stat bonuses from equipment |
| RemoveEquipmentBonuses | Removes all bonuses from equipment |
| AddBuffBonuses | Adds stat bonuses from a buff |
| RemoveBuffBonuses | Removes all bonuses from a buff |
| AddPassiveBonuses | Adds stat bonuses from a passive skill |
| RemovePassiveBonuses | Removes all bonuses from a passive skill |
| RemoveCharacter | Removes a character from the registry |

### Calculation Formula

Effective stats are computed as:

```
effective = floor((base + flat_bonuses) * (1.0 + multiplier_bonuses))
```

Where:
- `base` = Character's base stat from atlas-character
- `flat_bonuses` = Sum of all additive bonuses
- `multiplier_bonuses` = Sum of all percentage bonuses

---

## Registry

### Responsibility

Thread-safe singleton in-memory cache for character effective stats, organized by tenant.

### Operations

| Method | Description |
|--------|-------------|
| Get | Retrieves a character's model |
| GetOrCreate | Retrieves or creates a character's model |
| Update | Replaces a character's model |
| AddBonus | Adds a bonus and recomputes |
| AddBonuses | Adds multiple bonuses and recomputes |
| RemoveBonus | Removes a specific bonus and recomputes |
| RemoveBonusesBySource | Removes all bonuses from a source and recomputes |
| SetBaseStats | Sets base stats and recomputes |
| MarkInitialized | Marks a character as initialized |
| IsInitialized | Checks if a character has been initialized |
| GetAll | Returns all characters for a tenant |
| GetAllForWorld | Returns all characters in a specific world |
| Delete | Removes a character from the registry |
