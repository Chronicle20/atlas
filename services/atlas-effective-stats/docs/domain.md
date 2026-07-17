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
| bonuses | []stat.Bonus | Non-equipment bonuses (`buff:*` and `passive:*` sources) |
| wearer | WearerProfile | Wearer's level and jobId; input to equipment requirement checks |
| equipped | map[uint32]EquippedAsset | Per-asset equipment snapshots keyed by asset id; source of truth for equipment bonuses |
| qualifiedSnapshot | map[uint32]bool | Cached set of equipped asset ids that satisfied their template requirements as of the most recent recompute |
| computed | stat.Computed | Cached computed totals |
| lastUpdated | time.Time | Timestamp of last computation |
| initialized | bool | Whether lazy initialization has completed |

`MaxHpMpCap` (30000) is the ceiling applied to effective MaxHp and MaxMp.

#### EquippedAsset

Per-asset equipment snapshot held on the character Model; source of truth for equipment bonuses.

| Field | Type | Description |
|-------|------|-------------|
| assetId | uint32 | Equipment asset identifier |
| templateId | uint32 | Item template identifier |
| bonuses | []stat.Bonus | Flat stat bonuses extracted from the asset |

#### WearerProfile

Non-numeric inputs to equipment requirement checks.

| Field | Type | Description |
|-------|------|-------------|
| level | byte | Character level |
| jobId | job.Id | Character job identifier |

### Invariants

- Models are immutable; all modifications return new instances via `With*` methods.
- Duplicate bonuses (same source and stat type) are replaced, not accumulated.
- `bonuses` holds only `buff:*` and `passive:*` entries; equipment bonuses live in the `equipped` map, keyed by asset id.
- `Bonuses()` returns `bonuses` plus the bonuses of every equipped asset present in `qualifiedSnapshot` — only currently-qualifying equipment contributes to the bonus list returned to callers.
- Computed stats are recomputed after any bonus change.
- Effective MaxHp and MaxMp are capped at `MaxHpMpCap` (30000).
- The initialized flag prevents recursive initialization during lazy load.
- `Bonuses()` and `EquippedAsset.Bonuses()` return defensive copies to protect internal state.
- When MaxHP or MaxMP decreases due to bonus removal, clamp commands are published to the character command topic.

### State Transitions

- **Uninitialized -> Initialized**: On session CREATED event (channel issuer) or on first `GetEffectiveStats` call (lazy initialization). Fetches base stats and wearer profile from atlas-character, equipped-asset snapshots from atlas-inventory, buff bonuses from atlas-buffs, and passive skill bonuses from atlas-skills and atlas-data, then runs equipment qualification.
- **Initialized -> Updated**: On any bonus add/remove/change, base stat update, wearer profile change (level or job), or equipped-asset add/remove. Recomputes effective stats immediately.
- **Initialized -> Removed**: On session DESTROYED event. Character entry is deleted from the registry.

### Processors

#### ProcessorImpl

Provides operations for managing character effective stats.

| Method | Description |
|--------|-------------|
| GetEffectiveStats | Retrieves computed effective stats and bonuses; performs lazy initialization if needed |
| AddBonus | Adds or updates a flat stat bonus; recomputes assuming all currently-equipped assets qualify |
| AddMultiplierBonus | Adds or updates a percentage stat bonus; recomputes assuming all currently-equipped assets qualify |
| RemoveBonus | Removes a specific stat bonus; recomputes assuming all currently-equipped assets qualify; publishes clamp commands if MaxHP/MaxMP decreases |
| RemoveBonusesBySource | Removes all bonuses from a source; recomputes assuming all currently-equipped assets qualify; publishes clamp commands if MaxHP/MaxMP decreases |
| SetBaseStats | Sets base stats, then re-runs equipment qualification via RecomputeEquipmentBonuses |
| SetWearerProfile | Sets the wearer's level/jobId, then re-runs equipment qualification via RecomputeEquipmentBonuses |
| AddEquipmentBonuses | Stores an equipped-asset snapshot (source: `equipment:<id>`) with its template id and bonuses, then re-runs equipment qualification via RecomputeEquipmentBonuses |
| RemoveEquipmentBonuses | Removes an equipped-asset snapshot, then re-runs equipment qualification via RecomputeEquipmentBonuses |
| RecomputeEquipmentBonuses | Re-runs QualifiedEquipment against the current model, updates Computed and the qualified-asset cache, and publishes clamp commands if MaxHP/MaxMP decreases |
| AddBuffBonuses | Adds stat bonuses from a buff (source: `buff:<id>`); recomputes assuming all currently-equipped assets qualify |
| RemoveBuffBonuses | Removes all bonuses from a buff; recomputes assuming all currently-equipped assets qualify |
| AddPassiveBonuses | Adds stat bonuses from a passive skill (source: `passive:<id>`); recomputes assuming all currently-equipped assets qualify |
| RemovePassiveBonuses | Removes all bonuses from a passive skill; recomputes assuming all currently-equipped assets qualify |
| RemoveCharacter | Removes a character from the registry |

### Calculation Formula

Effective stats are computed as:

```
effective = floor((base + flat_bonuses) * (1.0 + multiplier_bonuses))
```

Where:
- `base` = Character's base stat from atlas-character (primary stats and MaxHP/MaxMP); secondary stats (WATK, MATK, etc.) have a base of 0
- `flat_bonuses` = Sum of all flat bonus amounts for that stat type, including non-equipment bonuses and the flat bonuses of equipped assets in the qualifying set (all equipped assets, when using the naive `Recompute` path)
- `multiplier_bonuses` = Sum of all percentage multipliers for that stat type (equipment contributes only flat bonuses, never multipliers)

Effective MaxHp and MaxMp are clamped to `MaxHpMpCap` (30000) after this computation.

### Equipment Qualification

`QualifiedEquipment` determines which equipped assets currently satisfy their template's requirements, using a fixed-point iteration: an asset qualifies once the wearer's applied stats (base stats + non-equipment flat bonuses + flat bonuses from assets already found to qualify) meet the asset's requirements. The iteration repeats until no additional asset qualifies in a pass. An asset whose template requirement lookup fails (the requirements provider returns false) is treated as not qualifying for that evaluation.

`meetsRequirements` checks `reqLevel`, `reqJob`, `reqStr`, `reqDex`, `reqInt`, and `reqLuk` against the wearer's level, jobId, and applied stats; a zero requirement field means no restriction.

`wearerClassMask` maps an atlas internal job id to the v83 `reqJob` bitmask (Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16; Beginner/Noblesse/Legend branches map to 0), since atlas internal job ids are not raw v83 client bitmasks.

`AppliedStats` (Strength, Dexterity, Intelligence, Luck) is the wearer's numeric snapshot used for the requirement checks: base stat + flat non-equipment bonus + flat bonus from equipment that has already qualified earlier in the same iteration.

Requirement data (`EquipmentRequirements`: ReqLevel, ReqJob, ReqStr, ReqDex, ReqInt, ReqLuk) is obtained from a `Provider` (`equipment.GetProvider`), which consults a per-tenant, per-template in-process cache before fetching from atlas-data.

### Initialization

Lazy initialization occurs via `InitializeCharacter` when a character's stats are first requested or when a session CREATED event is received. The initialization process:

1. Creates or retrieves the character model from the registry and marks it as initialized (prevents recursive initialization).
2. Fetches base stats and the wearer profile (level, jobId) from atlas-character via REST. On failure, defaults to zero base stats and an empty wearer profile.
3. Fetches the equip compartment from atlas-inventory via REST and builds an equipped-asset snapshot (asset id, template id, flat stat bonuses) for every asset in an equipped slot (negative slot position). Equipment stats are read from the asset's flat fields (strength, dexterity, etc.).
4. Fetches active buffs from atlas-buffs via REST and converts stat changes to bonuses.
5. Fetches character skills from atlas-skills and skill data from atlas-data via REST. Extracts bonuses from passive skills (non-action skills) at the character's skill level, including both direct effect fields and statups arrays.
6. Runs equipment qualification (`RecomputeWith`) against the assembled model, which determines the qualifying equipment subset and computes effective stats.

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
| AddBonus | Adds a bonus and recomputes, assuming all currently-equipped assets qualify |
| AddBonuses | Adds multiple bonuses and recomputes, assuming all currently-equipped assets qualify |
| RemoveBonus | Removes a specific bonus and recomputes, assuming all currently-equipped assets qualify; returns ErrNotFound if absent |
| RemoveBonusesBySource | Removes all bonuses from a source and recomputes, assuming all currently-equipped assets qualify; returns ErrNotFound if absent |
| SetBaseStats | Sets base stats and recomputes, assuming all currently-equipped assets qualify |
| SetWearerProfile | Sets the wearer's level/jobId; does not recompute |
| PutEquippedAsset | Writes (or overwrites) an equipped-asset snapshot, get-or-creating the model; does not recompute |
| RemoveEquippedAsset | Removes an equipped-asset snapshot; returns ErrNotFound if the character is absent; does not recompute |
| MarkInitialized | Marks a character as initialized |
| IsInitialized | Checks if a character has been initialized |
| GetAll | Returns all characters for a tenant |
| GetAllForWorld | Returns all characters in a specific world |
| Delete | Removes a character from the registry |
