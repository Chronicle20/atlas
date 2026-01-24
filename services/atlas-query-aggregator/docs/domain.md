# Validation Domain

## Responsibility

Validates character state against specified conditions by aggregating data from multiple external services.

## Core Models

### Condition

Represents a validation condition.

| Field | Type | Description |
|-------|------|-------------|
| conditionType | ConditionType | Type of condition to validate |
| operator | Operator | Comparison operator |
| value | int | Expected value |
| values | []int | Values for "in" operator |
| referenceId | uint32 | Reference for quest, item, map, transport, skill, or buff conditions |
| step | string | Quest progress step key |
| worldId | byte | World ID for map capacity conditions |
| channelId | byte | Channel ID for map capacity conditions |
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
| inventorySpace | Available inventory slots |
| transportAvailable | Transport route availability |
| skillLevel | Skill level |
| hp | Current HP |
| maxHp | Maximum HP |
| buff | Active buff status |

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

Aggregated data context for validation.

| Field | Type | Description |
|-------|------|-------------|
| character | character.Model | Character data |
| quests | map[uint32]quest.Model | Quest data by quest ID |
| skills | map[uint32]skill.Model | Skill data by skill ID |
| marriage | marriage.Model | Marriage gift data |
| buddyList | buddy.Model | Buddy list data |
| petCount | int | Spawned pet count |

## Invariants

- Condition type must be a supported ConditionType value
- Operator must be a supported Operator value
- Item conditions require a referenceId
- Quest status conditions require a referenceId
- Quest progress conditions require a referenceId and step
- Map capacity conditions require a referenceId (map ID)
- Transport available conditions require a referenceId (start map ID)
- Skill level conditions require a referenceId (skill ID)
- Buff conditions require a referenceId (source ID)
- Inventory space conditions require a referenceId (item ID)

## Processors

### Processor

Handles validation logic by aggregating data from external services.

| Method | Description |
|--------|-------------|
| ValidateStructured | Validates conditions against a character by ID |
| ValidateWithContext | Validates conditions using a pre-built ValidationContext |

---

# Character Domain

## Responsibility

Represents character data retrieved from the Character service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character ID |
| accountId | uint32 | Account ID |
| worldId | world.Id | World ID |
| name | string | Character name |
| gender | byte | Gender (0=male, 1=female) |
| level | byte | Character level |
| jobId | uint16 | Job ID |
| strength | uint16 | Strength stat |
| dexterity | uint16 | Dexterity stat |
| intelligence | uint16 | Intelligence stat |
| luck | uint16 | Luck stat |
| hp | uint16 | Current HP |
| maxHp | uint16 | Maximum HP |
| mp | uint16 | Current MP |
| maxMp | uint16 | Maximum MP |
| meso | uint32 | Currency |
| fame | int16 | Fame |
| mapId | uint32 | Current map ID |
| gm | int | GM level |
| reborns | uint32 | Rebirth count |
| dojoPoints | uint32 | Dojo points |
| vanquisherKills | uint32 | Vanquisher kill count |
| equipment | equipment.Model | Equipped items |
| inventory | inventory.Model | Inventory data |
| guild | guild.Model | Guild data |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetById | Retrieves character by ID with optional decorators |
| InventoryDecorator | Decorates character with inventory data |
| GuildDecorator | Decorates character with guild data |

---

# Inventory Domain

## Responsibility

Represents character inventory data retrieved from the Inventory service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| compartments | map[inventory.Type]compartment.Model | Inventory compartments by type |

### Compartment Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment ID |
| characterId | uint32 | Character ID |
| inventoryType | inventory.Type | Inventory type |
| capacity | uint32 | Slot capacity |
| assets | []asset.Model | Items in compartment |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetByCharacterId | Retrieves inventory for a character |

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
| progress | []ProgressModel | Progress entries |

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
| GetQuestsByCharacter | Retrieves all quests for a character |

---

# Guild Domain

## Responsibility

Represents guild data retrieved from the Guild service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Guild ID |
| worldId | byte | World ID |
| name | string | Guild name |
| notice | string | Guild notice |
| points | uint32 | Guild points |
| capacity | uint32 | Member capacity |
| leaderId | uint32 | Leader character ID |
| members | []member.Model | Guild members |
| titles | []title.Model | Guild titles |

### Member Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Member character ID |
| rank | byte | Member rank |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetByMemberId | Retrieves guild by member character ID |

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

---

# Pet Domain

## Responsibility

Represents pet data retrieved from the Pet service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Pet ID |
| slot | int8 | Spawn slot (-1 if not spawned) |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetSpawnedPetCount | Returns count of spawned pets for a character |

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
| GetSkillLevel | Returns skill level for a character and skill ID |

---

# Buff Domain

## Responsibility

Represents active buff data retrieved from the Buff service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| sourceId | int32 | Buff source ID |
| duration | int32 | Duration in seconds |
| createdAt | time.Time | Application time |
| expiresAt | time.Time | Expiration time |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| HasActiveBuff | Returns whether character has an active buff with given source ID |

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
| state | string | Route state |
| startMapId | map.Id | Starting map ID |

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetRouteByStartMap | Returns transport route by starting map ID |

---

# Map Domain

## Responsibility

Represents map data retrieved from the Map service.

## Processors

### Processor

| Method | Description |
|--------|-------------|
| GetPlayerCountInMap | Returns player count for a map |
