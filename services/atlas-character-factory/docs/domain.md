# Character Factory Domain

## Responsibility

The character factory domain handles character creation through saga-based orchestration. It validates character creation requests against tenant-configured templates, builds saga transactions for character creation, and coordinates follow-up sagas for awarding items, equipment, and skills.

## Core Models

### Factory RestModel

Input model for character creation requests.

| Field        | Type    |
|--------------|---------|
| AccountId    | uint32  |
| WorldId      | byte    |
| Name         | string  |
| Gender       | byte    |
| JobIndex     | uint32  |
| SubJobIndex  | uint32  |
| Face         | uint32  |
| Hair         | uint32  |
| HairColor    | uint32  |
| SkinColor    | byte    |
| Top          | uint32  |
| Bottom       | uint32  |
| Shoes        | uint32  |
| Weapon       | uint32  |
| Level        | byte    |
| Strength     | uint16  |
| Dexterity    | uint16  |
| Intelligence | uint16  |
| Luck         | uint16  |
| Hp           | uint16  |
| Mp           | uint16  |
| MapId        | map.Id  |

### Character Model

Domain model representing a character (used for REST communication with character service).

| Field              | Type   |
|--------------------|--------|
| id                 | uint32 |
| accountId          | uint32 |
| worldId            | byte   |
| name               | string |
| level              | byte   |
| experience         | uint32 |
| gachaponExperience | uint32 |
| strength           | uint16 |
| dexterity          | uint16 |
| intelligence       | uint16 |
| luck               | uint16 |
| hp                 | uint16 |
| mp                 | uint16 |
| maxHp              | uint16 |
| maxMp              | uint16 |
| meso               | uint32 |
| hpMpUsed           | int    |
| jobId              | uint16 |
| skinColor          | byte   |
| gender             | byte   |
| fame               | int16  |
| hair               | uint32 |
| face               | uint32 |
| ap                 | uint16 |
| sp                 | string |
| mapId              | uint32 |
| spawnPoint         | uint32 |
| gm                 | int    |

### ItemGained

| Field  | Type   |
|--------|--------|
| ItemId | uint32 |
| Slot   | int16  |

### Saga Model

| Field         | Type        |
|---------------|-------------|
| TransactionId | uuid.UUID   |
| SagaType      | Type        |
| InitiatedBy   | string      |
| Steps         | []Step[any] |

### Saga Step

| Field     | Type      |
|-----------|-----------|
| StepId    | string    |
| Status    | Status    |
| Action    | Action    |
| Payload   | T         |
| CreatedAt | time.Time |
| UpdatedAt | time.Time |

### FollowUpSagaTemplate

Stores template information for creating follow-up sagas after character creation.

| Field                          | Type                   |
|--------------------------------|------------------------|
| TenantId                       | uuid.UUID              |
| Input                          | RestModel              |
| Template                       | template.RestModel     |
| CharacterCreationTransactionId | uuid.UUID              |

### SagaCompletionTracker

Tracks completion status for character creation saga pairs.

| Field                          | Type      |
|--------------------------------|-----------|
| TenantId                       | uuid.UUID |
| AccountId                      | uint32    |
| CharacterId                    | uint32    |
| CharacterCreationTransactionId | uuid.UUID |
| FollowUpSagaTransactionId      | uuid.UUID |
| CharacterCreationCompleted     | bool      |
| FollowUpSagaCompleted          | bool      |

## Invariants

- Character name must be 1-12 characters containing only alphanumeric characters, underscores, or hyphens
- Gender must be 0 or 1
- Face, hair, hair color, skin color, top, bottom, shoes, and weapon must be valid for the job/gender template
- Job index and sub-job index must be valid

## Processors

### Factory Processor

Creates character creation sagas with validation.

- Validates character creation input against tenant configuration
- Builds `character_creation_only` saga with single `create_character` step
- Stores follow-up saga template for later use
- Stores saga completion tracking information
- Emits saga to orchestrator via Kafka

### Saga Processor

Emits saga commands to the orchestrator.

- Creates saga commands via Kafka producer

## Saga Types

| Type                        | Description                                              |
|-----------------------------|----------------------------------------------------------|
| character_creation_only     | Creates the character only                               |
| character_creation_followup | Awards items, equipment, and skills after character created |

## Saga Actions

| Action              | Description                                |
|---------------------|--------------------------------------------|
| create_character    | Creates a new character                    |
| award_asset         | Awards an item to character inventory      |
| create_and_equip_asset | Creates and equips an equipment item    |
| create_skill        | Creates a skill for the character          |

## Saga Step Statuses

| Status    | Description          |
|-----------|----------------------|
| pending   | Step not yet started |
| completed | Step finished        |
| failed    | Step failed          |
