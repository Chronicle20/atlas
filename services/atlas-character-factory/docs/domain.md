# Character Factory Domain

## Responsibility

The character factory domain handles character creation through saga-based orchestration. It validates character creation requests against tenant-configured templates and builds a unified saga containing character creation, item awards, equipment creation, and skill creation steps.

## Core Models

### Factory RestModel

Input model for character creation requests.

| Field        | Type     |
|--------------|----------|
| AccountId    | uint32   |
| WorldId      | world.Id |
| Name         | string   |
| Gender       | byte     |
| JobIndex     | uint32   |
| SubJobIndex  | uint32   |
| Face         | uint32   |
| Hair         | uint32   |
| HairColor    | uint32   |
| SkinColor    | byte     |
| Top          | uint32   |
| Bottom       | uint32   |
| Shoes        | uint32   |
| Weapon       | uint32   |
| Level        | byte     |
| Strength     | uint16   |
| Dexterity    | uint16   |
| Intelligence | uint16   |
| Luck         | uint16   |
| Hp           | uint16   |
| Mp           | uint16   |
| MapId        | map.Id   |

### CreateCharacterResponse

Response model for character creation requests.

| Field         | Type   |
|---------------|--------|
| TransactionId | string |

### Saga Model

Re-exported from `atlas-saga` shared library.

| Field         | Type        |
|---------------|-------------|
| TransactionId | uuid.UUID   |
| SagaType      | Type        |
| InitiatedBy   | string      |
| Steps         | []Step[any] |

### Saga Step

Re-exported from `atlas-saga` shared library.

| Field     | Type      |
|-----------|-----------|
| StepId    | string    |
| Status    | Status    |
| Action    | Action    |
| Payload   | T         |
| CreatedAt | time.Time |
| UpdatedAt | time.Time |

### CharacterCreatePayload

| Field        | Type     |
|--------------|----------|
| AccountId    | uint32   |
| WorldId      | world.Id |
| Name         | string   |
| Gender       | byte     |
| Level        | byte     |
| Strength     | uint16   |
| Dexterity    | uint16   |
| Intelligence | uint16   |
| Luck         | uint16   |
| JobId        | job.Id   |
| Hp           | uint16   |
| Mp           | uint16   |
| Face         | uint32   |
| Hair         | uint32   |
| Skin         | byte     |
| Top          | uint32   |
| Bottom       | uint32   |
| Shoes        | uint32   |
| Weapon       | uint32   |
| MapId        | map.Id   |

### AwardItemActionPayload

| Field       | Type        |
|-------------|-------------|
| CharacterId | uint32      |
| Item        | ItemPayload |

### CreateAndEquipAssetPayload

| Field       | Type        |
|-------------|-------------|
| CharacterId | uint32      |
| Item        | ItemPayload |

### ItemPayload

| Field      | Type   |
|------------|--------|
| TemplateId | uint32 |
| Quantity   | int    |

### CreateSkillPayload

| Field       | Type      |
|-------------|-----------|
| CharacterId | uint32    |
| SkillId     | uint32    |
| Level       | int       |
| MasterLevel | int       |
| Expiration  | time.Time |

### Template RestModel

Tenant-configured character creation template.

| Field       | Type     |
|-------------|----------|
| JobIndex    | uint32   |
| SubJobIndex | uint32   |
| MapId       | uint32   |
| Gender      | byte     |
| Faces       | []uint32 |
| Hairs       | []uint32 |
| HairColors  | []uint32 |
| SkinColors  | []uint32 |
| Tops        | []uint32 |
| Bottoms     | []uint32 |
| Shoes       | []uint32 |
| Weapons     | []uint32 |
| Items       | []uint32 |
| Skills      | []uint32 |

### Validation ConditionInput

| Field    | Type   |
|----------|--------|
| Type     | string |
| Operator | string |
| Value    | int    |
| ItemId   | uint32 |

### ValidateCharacterStatePayload

| Field       | Type               |
|-------------|--------------------|
| CharacterId | uint32             |
| Conditions  | []ConditionInput   |

## Invariants

- Character name must be 1-12 characters containing only alphanumeric characters, underscores, or hyphens
- Gender must be 0 or 1
- Face, hair, hair color, skin color, top, bottom, shoes, and weapon must be valid for the job/gender template
- A selection of 0 is always valid for template-validated fields
- If MapId is 0 in the request, the template's configured MapId is used
- Hair value in the saga payload is computed as `Hair + HairColor`

## Processors

### Factory Processor

Creates unified character creation sagas with validation.

- Validates character creation input against tenant configuration
- Matches input to a character template by job index, sub-job index, and gender
- Builds a single `CharacterCreation` saga containing all steps
- Step ordering: `create_character`, then `award_item_N` for template items, then `equip_<slot>` for equipment, then `create_skill_N` for template skills
- All steps after `create_character` use `CharacterId=0` as a sentinel value; the saga orchestrator injects the actual character ID via result forwarding
- Emits saga to orchestrator via Kafka

### Saga Processor

Emits saga commands to the orchestrator.

- Creates saga commands via Kafka producer

## Job Mapping

`JobFromIndex` maps job index and sub-job index to a `job.Id`.

| JobIndex | SubJobIndex | Result          |
|----------|-------------|-----------------|
| 0        | any         | NoblesseId      |
| 1        | 0           | BeginnerId      |
| 2        | any         | LegendId        |
| 3        | any         | EvanId          |
| other    | any         | BeginnerId      |

## Saga Actions

| Action              | Description                           |
|---------------------|---------------------------------------|
| create_character    | Creates a new character               |
| award_asset         | Awards an item to character inventory |
| create_and_equip_asset | Creates and equips an equipment item |
| create_skill        | Creates a skill for the character     |

## Saga Step Statuses

| Status    | Description          |
|-----------|----------------------|
| pending   | Step not yet started |
| completed | Step finished        |
| failed    | Step failed          |
