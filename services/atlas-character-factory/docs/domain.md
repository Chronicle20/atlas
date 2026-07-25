# Character Factory Domain

## Responsibility

The character factory domain handles character creation through saga-based orchestration. It supports two creation paths — validating requests against tenant-configured templates, and creating characters from tenant-configured presets — and builds a unified saga containing character creation, item awards, equipment creation, and skill creation steps for either path.

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
| Gm           | int      |
| Meso         | uint32   |

Gm and Meso are only populated by preset-based creation (`CreateFromPreset`); template-based creation (`Create`) always sends zero values.

### AwaitInventoryCreatedPayload

| Field       | Type   |
|-------------|--------|
| CharacterId | uint32 |

### AwardItemActionPayload

| Field       | Type        |
|-------------|-------------|
| CharacterId | uint32      |
| Item        | ItemPayload |

### CreateAndEquipAssetPayload

| Field           | Type        |
|-----------------|-------------|
| CharacterId     | uint32      |
| Item            | ItemPayload |
| UseAverageStats | bool        |

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

### PresetCreateRestModel

Input model for preset-based character creation requests.

| Field     | Type   |
|-----------|--------|
| PresetId  | string |
| AccountId | uint32 |
| WorldId   | byte   |
| Name      | string |

### Preset RestModel

Tenant-configured character-creation preset.

| Field      | Type       |
|------------|------------|
| Id         | string     |
| Attributes | Attributes |

Attributes:

| Field       | Type            |
|-------------|-----------------|
| Name        | string          |
| Description | string          |
| Tags        | []string        |
| JobId       | uint32          |
| Gender      | byte            |
| Face        | uint32          |
| Hair        | uint32          |
| HairColor   | uint32          |
| SkinColor   | byte            |
| MapId       | uint32          |
| Level       | byte            |
| Meso        | uint32          |
| Gm          | int             |
| Stats       | StatBlock       |
| DefaultName | string          |
| Equipment   | []EquipmentEntry |
| Inventory   | []InventoryEntry |
| Skills      | []SkillEntry    |

StatBlock:

| Field | Type   |
|-------|--------|
| Str   | uint16 |
| Dex   | uint16 |
| Int   | uint16 |
| Luk   | uint16 |
| Hp    | uint16 |
| Mp    | uint16 |

EquipmentEntry:

| Field           | Type   |
|-----------------|--------|
| TemplateId      | uint32 |
| UseAverageStats | bool   |

InventoryEntry:

| Field      | Type   |
|------------|--------|
| TemplateId | uint32 |
| Quantity   | uint32 |

SkillEntry:

| Field   | Type  |
|---------|-------|
| SkillId | uint32 |
| Level   | uint8 |

### NameValidityResult

Result of a character name-validity check against atlas-character.

| Field  | Type   |
|--------|--------|
| Valid  | bool   |
| Reason | string |
| Detail | string |

### ItemInfo

Result of an item existence/attribute lookup against atlas-data.

| Field     | Type   |
|-----------|--------|
| Id        | uint32 |
| Equipable | bool   |

### SkillInfo

Result of a skill lookup against atlas-data.

| Field    | Type   |
|----------|--------|
| Id       | uint32 |
| Name     | string |
| MaxLevel | uint8  |

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
- Template-based creation (`Create`) rejects the request if no template matches the chosen (jobIndex, subJobIndex, gender) combination
- Template-based creation maps (jobIndex, subJobIndex) to a `job.Id` via `JobFromIndex`

### Preset-based creation invariants

- PresetId must be a valid UUID
- The preset must exist in the tenant's configured presets
- Character name must pass the atlas-character name-validity check; a `"duplicate"` reason is a distinct error from other invalid-name reasons
- Each preset equipment entry's item must be equipable per atlas-data
- Preset equipment entries may not collide on equipment slot, where slot is derived as `TemplateId / 10000`
- Each preset inventory entry's item must exist in atlas-data
- Each preset skill entry's skill must exist in atlas-data; its MaxLevel is resolved from atlas-data and used as the saga step's MasterLevel
- Preset-based creation uses the preset's JobId directly (not mapped via `JobFromIndex`)
- Preset-based creation's legacy top/bottom/shoes/weapon fields in `CharacterCreatePayload` are always 0; equipment is conveyed entirely through `create_and_equip_asset` steps

## Processors

### Factory Processor

Creates unified character creation sagas with validation. Exposes two operations:

`Create` (template-based, `POST /api/characters/seed`):

- Validates character creation input against tenant configuration
- Matches input to a character template by job index, sub-job index, and gender; rejects the request if no template matches
- Builds a single `CharacterCreation` saga containing all steps
- Step ordering: `create_character`, then `await_inventory_created`, then `award_item_N` for template items, then `equip_<slot>` for equipment (top/bottom/shoes/weapon, skipping zero-value slots), then `create_skill_N` for template skills
- Saga timeout is a fixed 10 seconds
- Emits saga to orchestrator via Kafka

`CreateFromPreset` (preset-based, `POST /api/factory/characters/from-preset`):

- Resolves the preset by ID from tenant configuration
- Validates the character name via the atlas-character name-validity check
- Validates each preset equipment/inventory item and skill against atlas-data (see Invariants)
- Builds a single `CharacterCreation` saga containing all steps
- Step ordering: `create_character`, then `await_inventory_created`, then `award_asset_N` for inventory entries, then `create_and_equip_asset_N` for equipment entries, then `create_skill_N` for skill entries
- Saga timeout scales with step count: `10s + 1s * (2 + inventory count + equipment count + skill count)`
- Emits saga to orchestrator via Kafka

For both operations, all steps after `create_character` use `CharacterId=0` as a sentinel value; the saga orchestrator injects the actual character ID via result forwarding. The `await_inventory_created` step is a passive step advanced by the orchestrator once the character's inventory compartments are committed.

### Data Processor

Validates item and skill existence/attributes against atlas-data for preset-based creation.

- `GetItemById` resolves an item's inventory type via `atlas-constants`; equip-type items are additionally checked for existence against atlas-data, non-equip items are presumed to exist
- `GetSkillsByIds` batch-fetches skill name and max level from atlas-data

### Saga Processor

Emits saga commands to the orchestrator.

- Creates saga commands via Kafka producer

### Saga Status Handler

Reacts to saga status events for the `CharacterCreation` saga type.

- On a COMPLETED event, extracts `accountId` and `characterId` from the event results and emits a seed CREATED event; missing/zero values are logged and dropped
- On a FAILED event, re-emits a seed FAILED event carrying the failure reason; an event with `accountId` = 0 is logged and dropped
- Events for other saga types are ignored

## Job Mapping

`JobFromIndex` maps job index and sub-job index to a `job.Id`.

| JobIndex | SubJobIndex | Result          |
|----------|-------------|-----------------|
| 0        | any         | NoblesseId      |
| 1        | any         | BeginnerId      |
| 2        | any         | LegendId        |
| 3        | any         | EvanId          |
| other    | any         | BeginnerId      |

`JobFromIndex` is used only by template-based creation (`Create`). Preset-based creation (`CreateFromPreset`) uses the preset's configured JobId directly.

## Saga Actions

| Action                  | Description                                    |
|-------------------------|-------------------------------------------------|
| create_character        | Creates a new character                        |
| await_inventory_created | Waits for the character's inventory compartments to be created |
| award_asset             | Awards an item to character inventory          |
| create_and_equip_asset  | Creates and equips an equipment item           |
| create_skill            | Creates a skill for the character              |

## Saga Step Statuses

| Status    | Description          |
|-----------|----------------------|
| pending   | Step not yet started |
| completed | Step finished        |
| failed    | Step failed          |
