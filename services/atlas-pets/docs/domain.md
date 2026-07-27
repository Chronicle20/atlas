# Pet Domain

## pet

### Responsibility

Manages pet lifecycle, attributes, and state within the game. Pets are companion entities owned by characters that can be spawned, despawned, fed, commanded, and have their attributes modified over time.

### Core Models

#### Pet Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique pet identifier (auto-generated) |
| cashId | uint64 | Cash shop identifier |
| templateId | uint32 | Pet template reference |
| name | string | Pet name (max 13 characters) |
| level | byte | Pet level (1-30) |
| closeness | uint16 | Pet closeness value (0-30000) |
| fullness | byte | Pet fullness value (0-100) |
| expiration | time.Time | Pet expiration timestamp |
| ownerId | uint32 | Owning character identifier |
| slot | int8 | Spawn slot (-1 = not spawned, 0-2 = spawned) |
| excludes | []Exclude | Items excluded from pet auto-loot |
| flag | uint16 | Pet flags |
| purchaseBy | uint32 | Character who purchased the pet |

#### Exclude Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Unique exclude identifier (auto-generated) |
| itemId | uint32 | Excluded item identifier |

#### Temporal Data

Redis-backed tracking for pet position and stance, managed by a singleton TemporalRegistry.

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| stance | byte | Pet stance |
| fh | int16 | Foothold identifier |

### Invariants

- `templateId` is required (enforced by builder)
- `ownerId` is required (enforced by builder)
- `name` is required and non-empty (enforced by builder)
- `level` must be between 1 and 30 (enforced by builder; returns error if out of range)
- `fullness` must be between 0 and 100 (enforced by builder; returns error if out of range)
- `slot` must be -1 (despawned) or between 0 and 2 (spawned) (enforced by builder; returns error if out of range)
- `itemId` is required for excludes (enforced by exclude builder)
- Maximum closeness is 30000
- Maximum level is 30
- Maximum fullness is 100
- A pet is considered "spawned" when slot >= 0
- A pet is considered "lead" when slot == 0
- A pet is considered "hungry" when fullness < 100
- On create, level is clamped to 1 if out of range, fullness is clamped to 100 if out of range, slot is clamped to -1 if out of range

### State Transitions

#### Spawn States

| From | To | Condition |
|------|----|-----------|
| slot = -1 | slot = 0 | Spawn as lead; existing spawned pets shift to higher slots |
| slot = -1 | slot = 0-2 | Spawn as non-lead; takes the lowest available slot |
| slot = 0-2 | slot = -1 | Despawn command received or fullness <= 5 |

#### Multi-Pet Spawning

- Maximum 3 pets can be spawned simultaneously (slots 0, 1, 2)
- Spawning more than 1 pet requires the multi-pet skill (BeginnerMultiPet or NoblesseMultiPet)
- When a pet spawns as lead (slot 0), existing spawned pets shift to higher slots
- When a pet despawns, remaining pets at higher slots shift to lower slots to fill the gap

#### Hunger Mechanics

- Fullness decreases over time based on pet template hunger value
- The hunger task runs every 3 minutes for all logged-in characters
- When fullness reaches 5 or below after hunger evaluation, the pet is automatically despawned with reason "HUNGER"
- Only spawned pets (slot >= 0) are affected by hunger evaluation

#### Closeness and Leveling

- Closeness is awarded through commands, interactions, and direct awards
- Level increases when closeness reaches experience thresholds defined by the pet experience table
- Multiple levels can be gained from a single closeness award
- At max level (30), closeness is capped at 30000
- Experience thresholds per level: 1, 1, 3, 6, 14, 31, 60, 108, 181, 287, 434, 632, 891, 1224, 1642, 2161, 2793, 3557, 4467, 5542, 6801, 8263, 9950, 11882, 14084, 16578, 19391, 22547, 26074, 30000

#### Command Execution

- A pet must be spawned (slot >= 0) to execute a command
- Command success is determined probabilistically based on the pet template skill's probability value
- Closeness is awarded regardless of command success, based on the skill's increase value
- A command response event is emitted indicating success or failure

#### Evolution

- A pet evolves when its template's `IsEvolvable()` is true and the pet's level is at least the template's `ReqPetLevel()`
- The resulting template is selected via weighted-random selection among the template's evolution outcomes, weighted by each outcome's probability
- On evolution, the pet's templateId changes to the selected outcome and its expiration resets to now + 2160 hours (90 days)
- The corresponding cash inventory asset's template is updated to match, keyed by petId
- If the pet was spawned, it is despawned and respawned (to the same lead/non-lead slot) to refresh its appearance

#### Egg Hatching

- Spawning a pet whose template's `IsEgg()` is true does not spawn the pet; the egg hatches in place into its single evolution outcome instead
- Hatching resets level to 1, closeness to 0, and fullness to 100, and preserves the original expiration
- The corresponding cash inventory asset's template is updated to match, keyed by petId
- Hatching is refused (no-op, no spawn) if the character already owns the resulting baby template in their cash inventory

### Processors

#### Pet Processor

| Method | Description |
|--------|-------------|
| GetById | Retrieves a pet by identifier |
| GetByOwner | Retrieves all pets for an owner |
| ByOwnerIdPagedProvider | Returns one page of a character's pets, used by the REST list endpoint |
| SpawnedByOwnerProvider | Returns spawned pets (slot >= 0) for an owner |
| HungryByOwnerProvider | Returns spawned pets with fullness < 100 |
| HungriestByOwnerProvider | Returns the spawned pet with the lowest fullness |
| Create | Creates a new pet with validation and default clamping |
| Delete | Deletes a pet by identifier |
| DeleteForCharacter | Deletes all pets for a character |
| DeleteOnRemove | Deletes a pet when matching cash inventory asset is removed by slot and templateId |
| Move | Updates pet position in the temporal registry via foothold calculation |
| Spawn | Spawns a pet to an active slot with multi-pet skill validation |
| Despawn | Despawns a pet and shifts remaining pets to lower slots |
| AttemptCommand | Executes a pet command with probability-based success and closeness award |
| EvaluateHunger | Evaluates and decreases fullness for all spawned pets of an owner; auto-despawns at fullness <= 5 |
| ClearPositions | Clears temporal position data for all of an owner's pets |
| AwardCloseness | Awards closeness to a pet, triggering level-ups as thresholds are reached |
| AwardClosenessWithTransaction | Awards closeness with an associated transaction identifier |
| AwardFullness | Awards fullness to a pet, capped at 100 |
| AwardLevel | Awards levels to a pet, capped at 30 |
| Evolve | Evolves a pet to a new template selected via weighted-random roll among its evolution outcomes; despawns/respawns a spawned pet to refresh its appearance |
| SetExclude | Replaces the set of excluded items for pet auto-loot |

#### Temporal Registry

| Method | Description |
|--------|-------------|
| Update | Updates position, stance, and foothold for a pet |
| UpdatePosition | Updates position and foothold for a pet |
| UpdateStance | Updates stance for a pet |
| GetById | Retrieves temporal data for a pet; returns default (fh=1) if not found |
| Remove | Removes temporal data for a pet |

#### Hunger Task

Background task that runs every 3 minutes. Iterates over all logged-in characters (tracked via the Redis-backed character registry) and evaluates hunger for each character's spawned pets concurrently.

## asset

### Responsibility

Read-only projection of the unified inventory asset model. Used to locate pet assets in a character's cash inventory compartment when processing pet deletion on item removal.

### Core Models

#### Asset Model

A unified asset model representing any inventory item type. All fields are carried for compatibility with the shared REST representation from the inventory service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset identifier |
| compartmentId | uuid.UUID | Parent compartment identifier |
| slot | int16 | Inventory slot position |
| templateId | uint32 | Item template identifier |
| expiration | time.Time | Asset expiration timestamp |
| createdAt | time.Time | Asset creation timestamp |
| quantity | uint32 | Item quantity (stackable items) |
| ownerId | uint32 | Owner character identifier |
| flag | uint16 | Asset flags |
| rechargeable | uint64 | Rechargeable value |
| strength | uint16 | Equipment stat |
| dexterity | uint16 | Equipment stat |
| intelligence | uint16 | Equipment stat |
| luck | uint16 | Equipment stat |
| hp | uint16 | Equipment stat |
| mp | uint16 | Equipment stat |
| weaponAttack | uint16 | Equipment stat |
| magicAttack | uint16 | Equipment stat |
| weaponDefense | uint16 | Equipment stat |
| magicDefense | uint16 | Equipment stat |
| accuracy | uint16 | Equipment stat |
| avoidability | uint16 | Equipment stat |
| hands | uint16 | Equipment stat |
| speed | uint16 | Equipment stat |
| jump | uint16 | Equipment stat |
| slots | uint16 | Equipment upgrade slots |
| locked | bool | Equipment locked flag |
| spikes | bool | Equipment spikes flag |
| karmaUsed | bool | Equipment karma used flag |
| cold | bool | Equipment cold flag |
| canBeTraded | bool | Equipment tradeable flag |
| levelType | byte | Equipment level type |
| level | byte | Equipment level |
| experience | uint32 | Equipment experience |
| hammersApplied | uint32 | Equipment hammers applied |
| equippedSince | *time.Time | Equipment equipped timestamp |
| cashId | int64 | Cash shop identifier |
| commodityId | uint32 | Cash shop commodity identifier |
| purchaseBy | uint32 | Purchaser character identifier |
| petId | uint32 | Associated pet identifier (non-zero for pet items) |

Helper methods:

- `InventoryType()` - derives inventory type from templateId
- `IsStackable()` - true for Use, Setup, ETC types
- `IsCash()` - true for Cash type
- `IsPet()` - true for cash items with petId > 0
- `HasQuantity()` - true for stackable items or non-pet cash items
- `Quantity()` - returns quantity for items that have quantity, otherwise 1

#### Asset Builder

Constructs asset models. Created via `NewBuilder(compartmentId, templateId)` or `Clone(model)`.

| Method | Description |
|--------|-------------|
| SetId | Sets the asset identifier |
| SetSlot | Sets the inventory slot position |
| SetExpiration | Sets the expiration timestamp |
| SetPetId | Sets the associated pet identifier |
| Build | Returns the constructed Model |

### Invariants

- Asset is a read-only projection; no persistence in this service
- Pet items are identified by `IsCash() && petId > 0`

### Processors

No processors. Asset data is obtained via REST from the inventory service and used as part of the character's inventory model.

## compartment

### Responsibility

Read-only projection of an inventory compartment. Groups assets by inventory type (Equip, Use, Setup, ETC, Cash) within a character's inventory.

### Core Models

#### Compartment Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment identifier |
| characterId | uint32 | Owning character identifier |
| inventoryType | inventory.Type | Inventory type (Equip, Use, Setup, ETC, Cash) |
| capacity | uint32 | Maximum number of slots |
| assets | []asset.Model | Assets in this compartment |

Helper methods:

- `FindBySlot(slot)` - returns the asset at the given slot, or nil and false if not found
- `FindFirstByItemId(templateId)` - returns the first asset matching the template identifier, or nil and false if not found

### Invariants

- Compartment is a read-only projection; no persistence in this service

### Processors

No processors. Compartment data is obtained via REST from the inventory service.

## inventory

### Responsibility

Read-only projection of a character's full inventory. Aggregates compartments by inventory type for convenient access.

### Core Models

#### Inventory Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Owning character identifier |
| compartments | map[inventory.Type]compartment.Model | Compartments keyed by inventory type |

Accessor methods: `Equipable()`, `Consumable()`, `Setup()`, `ETC()`, `Cash()`, `CompartmentByType()`, `Compartments()`.

### Processors

#### Inventory Processor

| Method | Description |
|--------|-------------|
| GetByCharacterId | Fetches inventory via REST from the atlas-inventory service |
| ChangeTemplate | Requests an inventory item template change for a pet, keyed by petId (used during pet evolution and egg hatching) |

## character

### Responsibility

Read-only projection of character data. Maintains a Redis-backed registry of logged-in characters for the hunger task. Provides an inventory decorator that enriches character models with inventory data.

### Core Models

#### Character Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| mapId | map.Id | Current map identifier |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| inventory | inventory.Model | Character inventory (decorated) |

The model carries additional character fields (stats, job, etc.) for compatibility with shared REST responses, but only the fields above are used by pet operations.

#### Character Registry

Redis-backed singleton tracking logged-in characters. Keyed by character ID, values contain tenant and field (world/channel/map/instance) information.

### Processors

#### Character Processor

| Method | Description |
|--------|-------------|
| GetById | Fetches character data via REST from the atlas-characters service |
| InventoryDecorator | Enriches a character model with inventory data from atlas-inventory |
| Enter | Registers a character as logged in with field information |
| Exit | Removes a character from the logged-in registry |
| TransitionMap | Updates field information on map change |
| TransitionChannel | Updates field information on channel change |

## data/pet

### Responsibility

Read-only projection of pet template reference data (hunger rate, command skills, evolution rules) fetched from the pet reference data service.

### Core Models

#### Pet Reference Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Template identifier |
| hunger | uint32 | Fullness lost per hunger-task evaluation |
| cash | bool | Whether the template is a cash-shop item |
| life | uint32 | Life value |
| skills | []SkillModel | Command/trick skills available to the template |
| reqPetLevel | uint32 | Minimum pet level required to evolve |
| reqItemId | uint32 | Item required to trigger evolution |
| evolutions | []EvolutionModel | Possible evolution outcomes |

Helper methods:

- `IsEgg()` - true when there is exactly one evolution outcome and both `reqItemId` and `reqPetLevel` are zero
- `IsEvolvable()` - true when there is at least one evolution outcome and `reqItemId` is non-zero

#### Skill Model

| Field | Type | Description |
|-------|------|-------------|
| id | string | Composite command identifier ("{templateId}-{commandId}") |
| increase | uint16 | Closeness increase on command execution, regardless of success |
| probability | uint16 | Percent chance (0-100) of command success |

#### Evolution Model

| Field | Type | Description |
|-------|------|-------------|
| templateId | uint32 | Resulting template identifier |
| probability | uint32 | Relative weight for weighted-random selection among evolutions |

### Invariants

- `IsEgg()` is true only when the template has exactly one evolution outcome and both `reqItemId` and `reqPetLevel` are zero
- `IsEvolvable()` is true only when the template has at least one evolution outcome and `reqItemId` is non-zero
- This is a read-only projection; no persistence in this service

### Processors

#### Pet Reference Data Processor

| Method | Description |
|--------|-------------|
| GetById | Fetches pet template reference data via REST from the pet reference data service |

## data/position

### Responsibility

Read-only projection of foothold reference data, used to resolve the foothold beneath a given map coordinate for pet movement and spawn positioning.

### Core Models

#### Foothold Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Foothold identifier |

#### Point Model (package `point`)

| Field | Type | Description |
|-------|------|-------------|
| X | int16 | X coordinate |
| Y | int16 | Y coordinate |

Used as the request/response coordinate pair when resolving a foothold.

### Invariants

- This is a read-only projection; no persistence in this service

### Processors

#### Foothold Processor

| Method | Description |
|--------|-------------|
| GetBelow | Fetches the foothold below a given map coordinate via REST from the foothold reference data service |

## skill

### Responsibility

Read-only projection of character skill data, used to check whether a character has the multi-pet summon skill.

### Core Models

#### Skill Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Skill identifier |
| level | byte | Skill level |
| masterLevel | byte | Skill master level |
| expiration | time.Time | Skill expiration timestamp |
| cooldownExpiresAt | time.Time | Skill cooldown expiration timestamp |

Helper methods:

- `IsFourthJob()` - true if the job resolved from the skill id is a fourth job
- `OnCooldown()` - true if the current time is before `cooldownExpiresAt`

### Invariants

- This is a read-only projection; no persistence in this service

### Processors

#### Skill Processor

| Method | Description |
|--------|-------------|
| ByCharacterIdProvider | Fetches every skill for a character via REST from the skill reference data service, draining all pages |
| GetByCharacterId | Fetches every skill for a character |
| HasSkill | Returns true if the character has any of the given skill ids |
