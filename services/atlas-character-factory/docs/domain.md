# Character Factory Domain

## Responsibility

The character factory domain handles character creation through saga-based orchestration. It supports three creation paths — validating requests against tenant-configured templates, creating characters from tenant-configured presets, and creating characters from a tenant-configured Maple Life class table — and builds a unified saga containing character creation, item awards, equipment creation, and skill creation steps for all three paths. Maple Life creation resolves the player's class, look, and SP choices against the tenant's Maple Life class table and projects the result onto the same preset shape used by preset-based creation.

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
| AP           | uint16   |
| SP           | string   |

Gm, Meso, AP, and SP are only populated by preset-based creation (`CreateFromPreset`, `CreateMapleLife`); template-based creation (`Create`) always sends zero/empty values. SP is the ten-slot comma-separated skill-point pool in the form atlas-character persists it.

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

### MapleLifeCreateRestModel

Input model for Maple Life class-based character creation requests.

| Field        | Type   |
|--------------|--------|
| AccountId    | uint32 |
| WorldId      | byte   |
| Name         | string |
| ClassOrdinal | uint32 |
| Gender       | byte   |
| Face         | uint32 |
| Hair         | uint32 |
| HairColor    | uint32 |
| SkinColor    | byte   |
| SP           | byte   |

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
| AP          | uint16          |
| SP          | string          |
| DefaultName | string          |
| Equipment   | []EquipmentEntry |
| Inventory   | []InventoryEntry |
| Skills      | []SkillEntry    |

AP and SP are the unspent ability/skill points the created character starts with. SP is the ten-slot comma-separated pool in the form atlas-character persists it.

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

### Maple Life RestModel

Tenant-configured Maple Life class table.

| Field   | Type          |
|---------|---------------|
| Looks   | []LookOptions |
| Classes | []ClassEntry  |

LookOptions is one gender's set of selectable appearance values:

| Field      | Type     |
|------------|----------|
| Gender     | byte     |
| Faces      | []uint32 |
| Hairs      | []uint32 |
| HairColors | []uint32 |
| SkinColors | []uint32 |

ClassEntry is one (class ordinal, gender) row of the Maple Life creation table:

| Field     | Type            |
|-----------|-----------------|
| Ordinal   | uint32          |
| Gender    | byte            |
| JobId     | uint32          |
| Level     | byte            |
| MapId     | uint32          |
| Stats     | StatBlock       |
| AP        | uint16          |
| SP        | string          |
| SpSkillId | uint32          |
| Meso      | uint32          |
| Equipment | []EquipmentEntry |
| Inventory | []InventoryEntry |

StatBlock (Maple Life):

| Field | Type   |
|-------|--------|
| Str   | uint16 |
| Dex   | uint16 |
| Int   | uint16 |
| Luk   | uint16 |
| Hp    | uint16 |
| Mp    | uint16 |

EquipmentEntry and InventoryEntry have the same shape as the Preset RestModel's EquipmentEntry and InventoryEntry above.

SpSkillId is zero when the class offers no SP step at creation. SP is the ten-slot comma-separated pool in the form atlas-character persists it, representing what the class has left unspent at Level. AP is the ability points left unspent at Level; StatBlock carries the AP already spent to meet the class's first-job requirement.

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

| Field    | Type    |
|----------|---------|
| Id       | uint32  |
| Name     | string  |
| MaxLevel | uint8   |
| EffectX  | []int16 |

EffectX is the per-level HP/MP gain bonus (index 0 == level 1).

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

### Maple Life creation invariants

- The tenant must have at least one configured Maple Life class; otherwise the request is rejected
- The (ClassOrdinal, Gender) combination must match a configured class entry
- Gender must be 0 or 1
- The tenant must have look options configured for the chosen gender
- Face, hair, hair color, and skin color must each be present in the matched look options' lists for the chosen gender; a selection of 0 is always valid
- If the class has no SP skill (`SpSkillId` is 0), the request's SP must be 0
- If the class has an SP skill, requested SP must not exceed 10 and must not exceed the class's remaining SP pool (slot 0), including a +5 cost when the SP skill has a level-5 prerequisite skill
- Character name must pass the atlas-character name-validity check; a `"duplicate"` reason is a distinct error from other invalid-name reasons
- Class equipment and inventory entries are re-validated against atlas-data using the same rules as preset-based creation (equipable, no slot collisions, existence)
- The Warrior SP skill (Improved Max HP Increase) requires the Improved HP Recovery skill as a level-5 prerequisite; the Magician SP skill (Improved Max MP Increase) requires Improved MP Recovery. Other SP skills have no known prerequisite.
- When SP is spent on a skill with a known prerequisite, both the prerequisite (level 5) and the chosen skill (level = requested SP) are added as saga skill steps, and the prerequisite's cost (5) is added to the SP spent from the pool
- The resulting character's Hp or Mp (for the Warrior/Magician SP skills only) is increased by `29 * effectX` where `effectX` is the SP skill's atlas-data-sourced per-level effect value at the requested SP level; other classes' seeded stats pass through unchanged
- The class's SP pool (slot 0 of the ten-slot string) is reduced by the total spent SP before being carried into the projected preset
- Maple Life creation projects the resolved class entry and the player's look/SP choices onto a `preset.RestModel` and is built into a saga using the same construction as preset-based creation; the `characters.Templates` matching rules do not apply to this path

## Processors

### Factory Processor

Creates unified character creation sagas with validation. Exposes three operations:

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

`CreateMapleLife` (Maple Life class-based, `POST /api/factory/characters/maple-life`):

- Resolves the tenant's Maple Life class table and validates the player's class ordinal, gender, look, and SP choices against it (see Maple Life creation invariants)
- Validates the character name via the atlas-character name-validity check
- Validates the resolved class's equipment/inventory items against atlas-data, and its SP skill (if any) against atlas-data for its per-level effect value
- Projects the resolved class entry and the player's choices onto a `preset.RestModel` via `toPreset`
- Builds and emits the same `CharacterCreation` saga structure as `CreateFromPreset`, using the step-count-scaled timeout
- Emits saga to orchestrator via Kafka

For all three operations, all steps after `create_character` use `CharacterId=0` as a sentinel value; the saga orchestrator injects the actual character ID via result forwarding. The `await_inventory_created` step is a passive step advanced by the orchestrator once the character's inventory compartments are committed.

### Data Processor

Validates item and skill existence/attributes against atlas-data for preset-based and Maple Life class-based creation.

- `GetItemById` resolves an item's inventory type via `atlas-constants`; equip-type items are additionally checked for existence against atlas-data, non-equip items are presumed to exist
- `GetSkillsByIds` batch-fetches skill name, max level, and per-level HP/MP effect values (`EffectX`) from atlas-data

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

`JobFromIndex` is used only by template-based creation (`Create`). Preset-based creation (`CreateFromPreset`) and Maple Life creation (`CreateMapleLife`) use the preset's/class's configured JobId directly.

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
