# Validation Domain

## Responsibility

Validates character state against specified conditions by aggregating data from multiple external services. This is the primary domain of the service.

## Core Models

### Condition

Represents a validation condition.

| Field | Type | Description |
|-------|------|-------------|
| conditionType | ConditionType | Type of condition to validate |
| operator | Operator | Comparison operator |
| value | int | Expected value |
| values | []int | Values for "in" operator |
| referenceId | uint32 | Reference for quest, item, map, transport, skill, buff, or inventory space conditions |
| step | string | Quest progress step key |
| worldId | world.Id | World ID for map capacity conditions |
| channelId | channel.Id | Channel ID for map capacity conditions |
| includeEquipped | bool | Include equipped items in item quantity checks |

### ConditionType

| Value | Description |
|-------|-------------|
| jobId | Character job ID |
| meso | Character currency |
| mapId | Character map ID |
| fame | Character fame |
| item | Item quantity in inventory |
| gender | Character gender |
| level | Character level |
| reborns | Character rebirth count |
| dojoPoints | Dojo points |
| vanquisherKills | Vanquisher kill count |
| gmLevel | GM privilege level |
| guildId | Guild membership ID |
| guildLeader | Guild leader status |
| guildRank | Guild rank |
| questStatus | Quest state |
| questProgress | Quest progress value |
| hasUnclaimedMarriageGifts | Marriage gift availability |
| strength | Strength stat |
| dexterity | Dexterity stat |
| intelligence | Intelligence stat |
| luck | Luck stat |
| buddyCapacity | Buddy list capacity |
| petCount | Spawned pet count |
| mapCapacity | Player count in map |
| inventorySpace | Available inventory slots for an item |
| transportAvailable | Transport route availability |
| skillLevel | Skill level |
| hp | Current HP |
| maxHp | Maximum HP |
| buff | Active buff status |
| excessSp | Excess SP beyond expected for job tier |
| partyId | Party membership ID |
| partyLeader | Party leader status |
| partySize | Party member count |
| pqCustomData | Party quest custom data value by key |

### Operator

| Value | Description |
|-------|-------------|
| = | Equals |
| > | Greater than |
| < | Less than |
| >= | Greater than or equal |
| <= | Less than or equal |
| in | Value in list |

### ConditionResult

Represents the result of a condition evaluation.

| Field | Type | Description |
|-------|------|-------------|
| Passed | bool | Whether condition passed |
| Description | string | Human-readable description |
| Type | ConditionType | Condition type evaluated |
| Operator | Operator | Operator used |
| Value | int | Expected value |
| ItemId | uint32 | Item ID for item conditions |
| ActualValue | int | Actual value from character state |

### ValidationResult

Represents the result of validating multiple conditions.

| Field | Type | Description |
|-------|------|-------------|
| passed | bool | Whether all conditions passed |
| details | []string | Human-readable result details |
| results | []ConditionResult | Individual condition results |
| characterId | uint32 | Character ID validated |

### ValidationContext

Aggregated data context for validation. Supports lazy loading of external data via embedded processors.

| Field | Type | Description |
|-------|------|-------------|
| character | character.Model | Character data with inventory and equipment |
| quests | map[uint32]quest.Model | Quest data by quest ID |
| skills | map[uint32]skill.Model | Skill data by skill ID (local cache) |
| marriage | marriage.Model | Marriage gift data |
| buddyList | buddy.Model | Buddy list data |
| petCount | int | Spawned pet count |
| mapP | map.Processor | Lazy-loaded map player count queries |
| itemP | item.Processor | Lazy-loaded item slot max queries |
| transportP | transport.Processor | Lazy-loaded transport route queries |
| skillP | skill.Processor | Lazy-loaded skill level queries |
| buffP | buff.Processor | Lazy-loaded buff status queries |
| party | party.Model | Party data |
| partyP | party.Processor | Lazy-loaded party queries |
| partyQuestP | party_quest.Processor | Lazy-loaded party quest instance queries |

## Invariants

- Condition type must be a supported ConditionType value
- Operator must be a supported Operator value
- Item conditions require a referenceId (item template ID)
- Quest status conditions require a referenceId (quest ID)
- Quest progress conditions require a referenceId (quest ID) and step (progress key)
- Map capacity conditions require a referenceId (map ID)
- Transport available conditions require a referenceId (start map ID)
- Skill level conditions require a referenceId (skill ID)
- Buff conditions require a referenceId (source ID)
- Inventory space conditions require a referenceId (item ID)
- ExcessSP conditions require a referenceId (base level for job tier)
- PQ custom data conditions require a step (custom data key)
- Context-dependent conditions (questStatus, questProgress, marriageGifts, buddyCapacity, petCount, mapCapacity, transportAvailable, skillLevel, buff, inventorySpace, partyId, partyLeader, partySize, pqCustomData) return failure when evaluated without a ValidationContext
- A single failing condition causes the entire ValidationResult to fail

## State Transitions

Conditions and validation results are stateless. Each validation request produces a fresh result based on current character state.

## Processors

### Processor

Handles validation logic by aggregating data from external services.

| Method | Description |
|--------|-------------|
| ValidateStructured | Validates conditions against a character by ID. Automatically detects whether a ValidationContext is needed and builds one if required. |
| ValidateWithContext | Validates conditions using a pre-built ValidationContext |
| GetValidationContextProvider | Returns a provider that builds a ValidationContext by fetching character, quest, marriage, buddy, pet, and party data |

### Inventory Space Calculation

The `CalculateInventorySpace` function determines whether a character can hold a specified quantity of an item. It fills existing partial stacks first, then calculates new slots needed for the remainder. It queries the item processor for slot max data (with caching).

---

# Character Domain

## Responsibility

Represents character data retrieved from the Character service. The character model is the central aggregation point, decorated with inventory, equipment, and guild data.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character ID |
| accountId | uint32 | Account ID |
| worldId | world.Id | World ID |
| name | string | Character name |
| gender | byte | Gender (0=male, 1=female) |
| skinColor | byte | Skin color |
| face | uint32 | Face ID |
| hair | uint32 | Hair ID |
| level | byte | Character level |
| jobId | job.Id | Job ID |
| strength | uint16 | Strength stat |
| dexterity | uint16 | Dexterity stat |
| intelligence | uint16 | Intelligence stat |
| luck | uint16 | Luck stat |
| hp | uint16 | Current HP |
| maxHp | uint16 | Maximum HP |
| mp | uint16 | Current MP |
| maxMp | uint16 | Maximum MP |
| hpMpUsed | int | HP/MP usage tracker |
| ap | uint16 | Ability points |
| sp | string | Skill points (comma-separated) |
| experience | uint32 | Experience points |
| fame | int16 | Fame |
| gachaponExperience | uint32 | Gachapon experience |
| mapId | map.Id | Current map ID |
| spawnPoint | uint32 | Spawn point |
| gm | int | GM level |
| reborns | uint32 | Rebirth count |
| dojoPoints | uint32 | Dojo points |
| vanquisherKills | uint32 | Vanquisher kill count |
| x | int16 | X position |
| y | int16 | Y position |
| stance | byte | Current stance |
| meso | uint32 | Currency |
| equipment | equipment.Model | Equipped items (derived from equipable compartment) |
| inventory | inventory.Model | Inventory data |
| guild | guild.Model | Guild data |

### SetInventory Logic

When `SetInventory` is called, the equip compartment assets are split into two views:

- Assets with **positive slots** remain in the equipable compartment (unequipped equips in inventory).
- Assets with **negative slots** are placed into the equipment model, mapped to equipment slot types. Slots below -100 are treated as cash equipment (slot += 100 to find the base slot type).

### SP Table

Characters with Evan job IDs (2210-2218) use an SP table (multiple SP values), otherwise SP is a single value at index 0.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetById | Retrieves character by ID with optional decorators |
| InventoryDecorator | Decorates character with inventory data by fetching from the inventory service |
| GuildDecorator | Decorates character with guild data by fetching from the guild service |

---

# Inventory Domain

## Responsibility

Represents character inventory data retrieved from the Inventory service. An inventory consists of typed compartments (equip, use, setup, etc, cash), each containing assets.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| compartments | map[inventory.Type]compartment.Model | Inventory compartments by type |

Provides named accessors: `Equipable()`, `Consumable()`, `Setup()`, `ETC()`, `Cash()`, as well as `CompartmentByType(inventory.Type)` and `CompartmentById(uuid.UUID)`.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| ByCharacterIdProvider | Returns a provider for inventory data |
| GetByCharacterId | Retrieves inventory for a character |

---

# Compartment Domain

## Responsibility

Represents an inventory compartment -- a typed container of assets with a capacity.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment ID |
| characterId | uint32 | Character ID |
| inventoryType | inventory.Type | Inventory type (equip, use, setup, etc, cash) |
| capacity | uint32 | Slot capacity |
| assets | []asset.Model | Assets in compartment |

### ModelBuilder

Constructed via `NewBuilder(id, characterId, inventoryType, capacity)` or `Clone(model)`. Supports `SetCapacity`, `AddAsset`, `SetAssets`, and `Build`.

---

# Asset Domain

## Responsibility

Represents a unified inventory asset. All asset types (equipment, stackable, cash) share a single model. The asset's inventory type is derived from its template ID.

## Core Models

### Model

A unified model containing fields for all asset categories:

**Common fields:**

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset ID |
| compartmentId | uuid.UUID | Parent compartment ID |
| slot | int16 | Slot position |
| templateId | uint32 | Item template ID |
| expiration | time.Time | Expiration time |
| createdAt | time.Time | Creation time |

**Stackable fields:**

| Field | Type | Description |
|-------|------|-------------|
| quantity | uint32 | Stack quantity |
| ownerId | uint32 | Owner character ID |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Rechargeable amount |

**Equipment fields:**

| Field | Type | Description |
|-------|------|-------------|
| strength | uint16 | Strength bonus |
| dexterity | uint16 | Dexterity bonus |
| intelligence | uint16 | Intelligence bonus |
| luck | uint16 | Luck bonus |
| hp | uint16 | HP bonus |
| mp | uint16 | MP bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| hands | uint16 | Hands bonus |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Upgrade slots remaining |
| locked | bool | Whether locked |
| spikes | bool | Whether has spikes |
| karmaUsed | bool | Whether karma scissors used |
| cold | bool | Cold protection flag |
| canBeTraded | bool | Trade flag |
| levelType | byte | Level type |
| level | byte | Item level |
| experience | uint32 | Item experience |
| hammersApplied | uint32 | Hammers applied count |
| equippedSince | *time.Time | When equipped (nil if not equipped) |

**Cash fields:**

| Field | Type | Description |
|-------|------|-------------|
| cashId | int64 | Cash item serial number |
| commodityId | uint32 | Commodity ID |
| purchaseBy | uint32 | Purchasing character ID |

**Pet reference:**

| Field | Type | Description |
|-------|------|-------------|
| petId | uint32 | Pet ID (0 if not a pet) |

### Type Classification Methods

The model provides methods to classify asset type based on template ID:

| Method | Description |
|--------|-------------|
| InventoryType() | Returns inventory type derived from template ID |
| IsEquipment() | True if equip type |
| IsCashEquipment() | True if equip type with non-zero cashId |
| IsConsumable() | True if use type |
| IsSetup() | True if setup type |
| IsEtc() | True if etc type |
| IsCash() | True if cash type |
| IsPet() | True if cash type with non-zero petId |
| IsStackable() | True if use, setup, or etc type |
| HasQuantity() | True if stackable, or cash (non-pet) |
| Quantity() | Returns quantity (1 for non-stackable items) |

### ModelBuilder

Constructed via `NewBuilder(compartmentId, templateId)` or `Clone(model)`. Provides setter methods for all fields and a `Build()` method to produce a Model.

---

# Equipment Domain

## Responsibility

Represents the equipment slots on a character, derived from the equip compartment assets when inventory is loaded.

## Core Models

### Model

A map of slot types to slot models.

| Method | Description |
|--------|-------------|
| Get(slotType) | Returns the slot model for the given type |
| Set(slotType, model) | Sets the slot model for the given type |
| Slots() | Returns all slot models |

### Slot Model

| Field | Type | Description |
|-------|------|-------------|
| Position | slot.Position | Slot position value |
| Equipable | *asset.Model | Normal equipped asset (nil if empty) |
| CashEquipable | *asset.Model | Cash equipped asset (nil if empty) |

---

# Quest Domain

## Responsibility

Represents quest data retrieved from the Quest service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| questId | uint32 | Quest ID |
| state | State | Quest state |
| startedAt | time.Time | Start time |
| completedAt | time.Time | Completion time |
| progress | []ProgressModel | Progress entries by info number |
| progressByKey | map[string]int | Progress entries by string key |

### State

| Value | Description |
|-------|-------------|
| 0 | Not Started |
| 1 | Started |
| 2 | Completed |

### ProgressModel

| Field | Type | Description |
|-------|------|-------------|
| infoNumber | uint32 | Progress info number |
| progress | string | Progress value |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetQuestState | Returns quest state for a character and quest ID |
| GetQuestProgress | Returns quest progress for a specific info number |
| GetQuest | Returns complete quest model |
| GetQuestsByCharacter | Returns all quests for a character |
| GetStartedQuestsByCharacter | Returns started quests for a character |
| GetCompletedQuestsByCharacter | Returns completed quests for a character |

---

# Guild Domain

## Responsibility

Represents guild data retrieved from the Guild service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Guild ID |
| worldId | world.Id | World ID |
| name | string | Guild name |
| notice | string | Guild notice |
| points | uint32 | Guild points |
| capacity | uint32 | Member capacity |
| logo | uint16 | Logo ID |
| logoColor | byte | Logo color |
| logoBackground | uint16 | Logo background ID |
| logoBackgroundColor | byte | Logo background color |
| leaderId | uint32 | Leader character ID |
| members | []member.Model | Guild members |
| titles | []title.Model | Guild titles |

### Member Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Member character ID |
| name | string | Member name |
| jobId | uint16 | Member job ID |
| level | byte | Member level |
| rank | byte | Member rank |
| online | bool | Online status |
| allianceRank | byte | Alliance rank |

The guild model provides a `MemberRank(characterId)` method that returns a member's rank value, or 0 if the character is not a member.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetByMemberId | Retrieves guild by member character ID |
| IsLeader | Checks if a character is the guild leader |
| HasGuild | Checks if a character belongs to a guild |

---

# Marriage Domain

## Responsibility

Represents marriage gift data retrieved from the Marriage service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| hasUnclaimedGifts | bool | Whether unclaimed gifts exist |
| unclaimedGiftCount | int | Number of unclaimed gifts |
| lastGiftClaimedTime | int64 | Timestamp of last gift claim |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetMarriageGifts | Retrieves marriage gift data for a character |
| HasUnclaimedGifts | Returns whether a character has unclaimed gifts |
| GetUnclaimedGiftCount | Returns count of unclaimed gifts |

---

# Buddy Domain

## Responsibility

Represents buddy list data retrieved from the Buddy service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| capacity | byte | Buddy list capacity |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetBuddyList | Retrieves buddy list for a character |
| GetBuddyCapacity | Returns buddy list capacity for a character |

---

# Pet Domain

## Responsibility

Represents pet data retrieved from the Pet service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Pet ID |
| slot | int8 | Spawn slot (-1 if not spawned, >= 0 if spawned) |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetPets | Returns all pets for a character |
| GetSpawnedPetCount | Returns count of spawned pets (slot >= 0) for a character |

---

# Skill Domain

## Responsibility

Represents skill data retrieved from the Skill service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill ID |
| level | byte | Current skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Skill expiration |
| cooldownExpiresAt | time.Time | Cooldown expiration |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetSkillLevel | Returns skill level for a character and skill ID (0 if not found) |
| GetSkill | Returns complete skill model |
| GetSkillsByCharacter | Returns all skills for a character |
| GetSkillsMap | Returns all skills as a map keyed by skill ID |

---

# Buff Domain

## Responsibility

Represents active buff data retrieved from the Buff service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| sourceId | int32 | Buff source ID (skill/item that applied the buff) |
| duration | int32 | Duration in seconds |
| createdAt | time.Time | Application time |
| expiresAt | time.Time | Expiration time |

The model provides an `IsActive()` method that returns true if the current time is before `expiresAt`.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| HasActiveBuff | Returns whether a character has an active buff with the given source ID. Returns false on error (graceful degradation). |
| GetBuffsByCharacter | Returns all active buffs for a character. Returns empty slice on error. |

---

# Transport Domain

## Responsibility

Represents transport route data retrieved from the Transport service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route ID |
| name | string | Route name |
| state | string | Route state (e.g., "open_entry") |
| startMapId | map.Id | Starting map ID |

The model provides an `IsOpenEntry()` method that returns true if the state is "open_entry".

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetRouteByStartMap | Returns transport route by starting map ID. Returns first route if multiple exist. |

---

# Map Domain

## Responsibility

Represents map data retrieved from the Map service. Used to query player counts for map capacity conditions.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetPlayerCountInMap | Returns player count for a map (world/channel/map/instance). Returns 0 on error (graceful degradation). |

---

# Item Domain

## Responsibility

Provides item metadata (slot max) for inventory space calculations. Uses a singleton in-memory cache with TTL-based expiration.

## Core Models

### ItemData

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Item ID |
| slotMax | uint32 | Maximum stack size |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetSlotMax | Returns maximum stack size for an item. Checks cache first, then fetches from the data service based on item type. Equipment always returns 1. Values capped at 1000. Falls back to defaults on error. |

---

# Party Domain

## Responsibility

Represents party data retrieved from the Party service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Party ID (0 if not in party) |
| leaderId | uint32 | Leader character ID |
| members | []uint32 | Member character IDs |

The model provides a `MemberCount()` method that returns the number of members.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetPartyByCharacter | Returns the party for a character. Returns zero-value model if not in a party. |

---

# Party Quest Domain

## Responsibility

Represents party quest instance data retrieved from the Party Quest service. Used to validate PQ custom data conditions.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Instance ID |
| customData | map[string]string | Custom data key-value pairs |

The model provides `GetCustomDataValue(key)` for string values and `GetCustomDataInt(key)` for integer values (returns 0 if not found or not numeric).

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetInstanceByCharacter | Returns the active PQ instance for a character. Returns zero-value model if no active PQ. |
