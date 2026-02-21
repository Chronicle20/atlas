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
| maxHp | uint16 | Base maximum HP |
| maxMp | uint16 | Base maximum MP |

#### Computed

Holds all computed effective stats for a character.

| Field | Type | Description |
|-------|------|-------------|
| strength | uint32 | Effective strength |
| dexterity | uint32 | Effective dexterity |
| luck | uint32 | Effective luck |
| intelligence | uint32 | Effective intelligence |
| maxHp | uint32 | Effective maximum HP |
| maxMp | uint32 | Effective maximum MP |
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

### Stat Mapping

The `MapBuffStatType` function maps buff stat type strings to stat types. Most buff types map to flat bonuses. `HYPER_BODY_HP`, `HYPER_BODY_MP`, and `MAPLE_WARRIOR` map to multiplier bonuses.

The `MapStatupType` function maps passive skill statup type strings to stat types. It accepts both short-form (e.g., `PAD`, `STR`) and long-form (e.g., `WEAPON_ATTACK`, `STRENGTH`) identifiers.

---

## character

### Responsibility

Manages the effective stats model for characters, including bonus tracking, computation, registry storage, and lazy initialization from external services.

### Core Models

#### Model

Holds all stat bonuses and computed effective stats for a character.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| ch | channel.Model | Channel model (contains world ID and channel ID) |
| characterId | uint32 | Character identifier |
| baseStats | stat.Base | Base stats from character service |
| bonuses | []stat.Bonus | All active bonuses |
| computed | stat.Computed | Cached computed totals |
| lastUpdated | time.Time | Timestamp of last computation |
| initialized | bool | Whether lazy initialization has completed |

### Invariants

- Models are immutable; all modifications return new instances via `With*` methods.
- Duplicate bonuses (same source and stat type) are replaced, not accumulated.
- Computed stats are recomputed after any bonus change.
- The initialized flag prevents recursive initialization during lazy load.
- `Bonuses()` returns a defensive copy to protect internal state.
- When MaxHP or MaxMP decreases due to bonus removal, clamp commands are published to the character command topic.

### State Transitions

- **Uninitialized -> Initialized**: On session CREATED event (channel issuer) or on first `GetEffectiveStats` call (lazy initialization). Fetches base stats from atlas-character, equipment bonuses from atlas-inventory, buff bonuses from atlas-buffs, and passive skill bonuses from atlas-skills and atlas-data.
- **Initialized -> Updated**: On any bonus add/remove/change or base stat update. Recomputes effective stats immediately.
- **Initialized -> Removed**: On session DESTROYED event. Character entry is deleted from the registry.

### Processors

#### ProcessorImpl

Provides operations for managing character effective stats.

| Method | Description |
|--------|-------------|
| GetEffectiveStats | Retrieves computed effective stats and bonuses; performs lazy initialization if needed |
| AddBonus | Adds or updates a flat stat bonus |
| AddMultiplierBonus | Adds or updates a percentage stat bonus |
| RemoveBonus | Removes a specific stat bonus; publishes clamp commands if MaxHP/MaxMP decreases |
| RemoveBonusesBySource | Removes all bonuses from a source; publishes clamp commands if MaxHP/MaxMP decreases |
| SetBaseStats | Sets base stats and recomputes |
| AddEquipmentBonuses | Adds stat bonuses from equipment (source: `equipment:<id>`) |
| RemoveEquipmentBonuses | Removes all bonuses from equipment |
| AddBuffBonuses | Adds stat bonuses from a buff (source: `buff:<id>`) |
| RemoveBuffBonuses | Removes all bonuses from a buff |
| AddPassiveBonuses | Adds stat bonuses from a passive skill (source: `passive:<id>`) |
| RemovePassiveBonuses | Removes all bonuses from a passive skill |
| RemoveCharacter | Removes a character from the registry |

### Calculation Formula

Effective stats are computed as:

```
effective = floor((base + flat_bonuses) * (1.0 + multiplier_bonuses))
```

Where:
- `base` = Character's base stat from atlas-character (primary stats and MaxHP/MaxMP); secondary stats (WATK, MATK, etc.) have a base of 0
- `flat_bonuses` = Sum of all flat bonus amounts for that stat type
- `multiplier_bonuses` = Sum of all percentage multipliers for that stat type

### Initialization

Lazy initialization occurs via `InitializeCharacter` when a character's stats are first requested or when a session CREATED event is received. The initialization process:

1. Creates or retrieves the character model from the registry.
2. Marks the character as initialized (prevents recursive initialization).
3. Fetches base stats from atlas-character via REST.
4. Fetches the equip compartment from atlas-inventory via REST and extracts equipment stat bonuses from equipped assets (negative slot positions). Equipment stats are read from the asset's flat fields (strength, dexterity, etc.).
5. Fetches active buffs from atlas-buffs via REST and converts stat changes to bonuses.
6. Fetches character skills from atlas-skills and skill data from atlas-data via REST. Extracts bonuses from passive skills (non-action skills) at the character's skill level, including both direct effect fields and statups arrays.
7. Recomputes effective stats and updates the registry.

Each fetch step is fail-safe; failures are logged as warnings and the character proceeds with partial data.

---

## Registry

### Responsibility

Singleton Redis-backed tenant-scoped cache for character effective stats models. Uses `atlas.TenantRegistry` with the `effective-stats` namespace. Character IDs are used as keys.

### Operations

| Method | Description |
|--------|-------------|
| Get | Retrieves a character's model; returns ErrNotFound if absent |
| GetOrCreate | Retrieves or creates a character's model |
| Update | Replaces a character's model |
| AddBonus | Adds a bonus and recomputes |
| AddBonuses | Adds multiple bonuses and recomputes |
| RemoveBonus | Removes a specific bonus and recomputes; returns ErrNotFound if absent |
| RemoveBonusesBySource | Removes all bonuses from a source and recomputes; returns ErrNotFound if absent |
| SetBaseStats | Sets base stats and recomputes |
| MarkInitialized | Marks a character as initialized |
| IsInitialized | Checks if a character has been initialized |
| GetAll | Returns all characters for a tenant |
| GetAllForWorld | Returns all characters in a specific world |
| Delete | Removes a character from the registry |
