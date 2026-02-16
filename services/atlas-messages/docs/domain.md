# Domain

## Message

### Responsibility

Handles processing of character chat messages. Routes messages to either the command system (for GM commands) or produces chat events for relay to other services.

### Core Models

#### Message Types

- **General** - Messages broadcast to all players in the same map
- **Buddy** - Messages sent to buddy list recipients
- **Party** - Messages sent to party members
- **Guild** - Messages sent to guild members
- **Alliance** - Messages sent to alliance members
- **Whisper** - Private messages to a specific player
- **Messenger** - Messages through the messenger system
- **Pet** - Messages from pets
- **PinkText** - System messages displayed in pink text

### Processors

#### MessageProcessor

| Method | Responsibility |
|--------|---------------|
| HandleGeneral | Processes general chat messages; checks for GM commands before relaying |
| HandleMulti | Processes multi-recipient messages (buddy, party, guild, alliance); checks for GM commands before relaying |
| HandleWhisper | Processes whisper messages; validates recipient exists and is in same world |
| HandleMessenger | Processes messenger chat messages |
| HandlePet | Processes pet chat messages |
| IssuePinkText | Produces pink text chat events for system messages |

---

## Command

### Responsibility

Provides a registry and execution framework for GM commands. Commands are parsed from chat messages and executed when the character has GM privileges.

### Core Models

#### Command

A command consists of a Producer function that matches message patterns and returns an Executor function.

### Processors

#### CommandRegistry

| Method | Responsibility |
|--------|---------------|
| Add | Registers command producers to the registry |
| Get | Matches a message against registered commands and returns an executor if found |

### Registered Commands

| Command | Pattern | Description |
|---------|---------|-------------|
| HelpCommandProducer | `@help` | Displays available commands |
| WarpCommandProducer | `@warp <target> <mapId>` | Warps character(s) to a map |
| WhereAmICommandProducer | `@query map` | Displays current map ID |
| RatesCommandProducer | `@query rates` | Displays current rates (exp, meso, drop) with factor breakdowns |
| AwardExperienceCommandProducer | `@award <target> experience <amount>` | Awards experience points |
| AwardLevelCommandProducer | `@award <target> <amount> level` | Awards levels |
| AwardMesoCommandProducer | `@award <target> meso <amount>` | Awards mesos |
| AwardCurrencyCommandProducer | `@award <target> <credit\|points\|prepaid> <amount>` | Awards cash shop currency |
| AwardItemCommandProducer | `@award <target> item <itemId> [quantity]` | Awards items |
| ChangeJobCommandProducer | `@change <target> job <jobId>` | Changes character job |
| MaxSkillCommandProducer | `@skill max <skillId>` | Maximizes skill level |
| ResetSkillCommandProducer | `@skill reset <skillId>` | Resets skill to level 0 |
| BuffCommandProducer | `@buff <target> <skillName\|#skillId> [duration]` | Applies a buff by skill name or ID |
| ConsumeCommandProducer | `@consume <target> <itemId>` | Applies consumable item effects |
| MobStatusCommandProducer | `@mobstatus <skillId\|skillName> [level]` | Executes mob skill on all monsters in map |
| MobClearCommandProducer | `@mobclear [statusType]` | Clears statuses from all monsters in map |
| DiseaseCommandProducer | `@disease <target> <diseaseType> [value] [duration]` | Applies a disease effect to character(s) |

Target values:
- `me` - The command issuer
- `map` - All characters in the current map (not supported for currency commands)
- `<name>` - A specific character by name

---

## Character

### Responsibility

Provides character data retrieval for message processing and command execution.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character ID |
| accountId | uint32 | Associated account ID |
| worldId | world.Id | World the character is in |
| name | string | Character name |
| level | byte | Character level |
| jobId | uint16 | Character job ID |
| mapId | uint32 | Current map ID |
| gm | int | GM status (1 = GM) |
| skills | []skill.Model | Character's skills |

### Invariants

- A character is a GM if gm equals 1

### Processors

#### CharacterProcessor

| Method | Responsibility |
|--------|---------------|
| GetById | Retrieves a character by ID |
| GetByName | Retrieves a character by name |
| ByNameProvider | Returns a provider for characters by name |
| IdByNameProvider | Returns a provider for character ID by name |
| SkillModelDecorator | Decorates a character model with skill data |

---

## Skill

### Responsibility

Provides character skill data retrieval.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill ID |
| level | byte | Current skill level |
| masterLevel | byte | Maximum skill level |
| expiration | time.Time | Skill expiration time |
| cooldownExpiresAt | time.Time | Cooldown expiration time |

### Processors

#### SkillProcessor

| Method | Responsibility |
|--------|---------------|
| ByCharacterIdProvider | Returns a provider for skills by character ID |
| GetByCharacterId | Retrieves skills for a character |

---

## Saga

### Responsibility

Builds and submits saga transactions for command execution. Commands produce sagas that are processed by the saga orchestrator.

### Core Models

#### Saga

| Field | Type | Description |
|-------|------|-------------|
| TransactionId | uuid.UUID | Unique transaction identifier |
| SagaType | Type | Type of saga (inventory_transaction, quest_reward, trade_transaction) |
| InitiatedBy | string | Initiator of the saga |
| Steps | []Step | Steps in the saga |

#### Step

| Field | Type | Description |
|-------|------|-------------|
| StepId | string | Unique step identifier |
| Status | Status | Step status (pending, completed, failed) |
| Action | Action | Action to execute |
| Payload | any | Action-specific payload |
| CreatedAt | time.Time | Creation timestamp |
| UpdatedAt | time.Time | Last update timestamp |

#### Actions

| Action | Payload Type | Description |
|--------|-------------|-------------|
| AwardInventory | AwardItemActionPayload | Awards items to inventory |
| AwardExperience | AwardExperiencePayload | Awards experience points |
| AwardLevel | AwardLevelPayload | Awards character levels |
| AwardMesos | AwardMesosPayload | Awards mesos |
| AwardCurrency | AwardCurrencyPayload | Awards cash shop currency |
| WarpToRandomPortal | WarpToRandomPortalPayload | Warps to random portal in field |
| WarpToPortal | WarpToPortalPayload | Warps to specific portal |
| DestroyAsset | DestroyAssetPayload | Destroys inventory asset |
| ChangeJob | ChangeJobPayload | Changes character job |
| CreateSkill | CreateSkillPayload | Creates a skill for character |
| UpdateSkill | UpdateSkillPayload | Updates a character skill |
| ApplyConsumableEffect | ApplyConsumableEffectPayload | Applies consumable item effects to character |

### Invariants

- A saga must have at least one step
- Saga type is required
- InitiatedBy is required
- Each step must have a valid action

### Processors

#### SagaProcessor

| Method | Responsibility |
|--------|---------------|
| Create | Submits a saga for processing via Kafka |

---

## Buff

### Responsibility

Applies skill buff effects to characters. Retrieves skill effect data, builds stat changes, and emits buff commands via Kafka.

### Processors

#### BuffProcessor

| Method | Responsibility |
|--------|---------------|
| Apply | Applies a buff to a character by skill ID and level; resolves skill effect data and emits buff command |

---

## Rate

### Responsibility

Retrieves experience, meso, drop, and quest experience rates for characters, including factor breakdowns.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character ID |
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |
| factors | []Factor | Rate factor breakdowns |

#### Factor

| Field | Type | Description |
|-------|------|-------------|
| source | string | Factor source identifier |
| rateType | string | Rate type (exp, meso, item_drop, quest_exp) |
| multiplier | float64 | Multiplier value |

### Processors

#### RateProcessor

| Method | Responsibility |
|--------|---------------|
| GetByCharacter | Retrieves rates and factors for a character |

---

## Map

### Responsibility

Provides map existence validation and character lookup within maps.

### Processors

#### MapProcessor

| Method | Responsibility |
|--------|---------------|
| Exists | Checks if a map exists |
| CharacterIdsInFieldProvider | Returns a provider for character IDs in a field (map instance) |
| CharacterIdsInMapStringProvider | Returns a provider for character IDs in a map (string input) |

---

## Data

### Responsibility

Provides read-only access to game data for validation.

### Processors

#### AssetProcessor

| Method | Responsibility |
|--------|---------------|
| Exists | Checks if an item exists (validates equipable items) |

#### EquipableProcessor

| Method | Responsibility |
|--------|---------------|
| GetById | Retrieves equipable item data by ID |

#### SkillProcessor (data)

| Method | Responsibility |
|--------|---------------|
| GetById | Retrieves skill data by ID |
| GetByName | Retrieves skills matching a name |
| GetEffect | Retrieves skill effect for a specific level |

#### MapProcessor (data)

| Method | Responsibility |
|--------|---------------|
| GetById | Retrieves map data by ID |
