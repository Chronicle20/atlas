# Consumable Domain

## asset

### Responsibility

Represents any inventory item within the consumables service as a unified flat model. A single Asset carries all possible fields across equipment, stackable, cash, and pet item types. The asset's actual type is derived at runtime from its templateId using `inventory.TypeFromItemId` from the constants library.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset identifier |
| compartmentId | uuid.UUID | Owning compartment |
| slot | int16 | Inventory slot position |
| templateId | uint32 | Item template identifier |
| expiration | time.Time | Expiration timestamp |
| createdAt | time.Time | Creation timestamp |
| quantity | uint32 | Stack quantity (stackable/cash items) |
| ownerId | uint32 | Owner character ID |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Rechargeable amount |
| strength | uint16 | STR stat (equipment) |
| dexterity | uint16 | DEX stat (equipment) |
| intelligence | uint16 | INT stat (equipment) |
| luck | uint16 | LUK stat (equipment) |
| hp | uint16 | HP stat (equipment) |
| mp | uint16 | MP stat (equipment) |
| weaponAttack | uint16 | Weapon attack (equipment) |
| magicAttack | uint16 | Magic attack (equipment) |
| weaponDefense | uint16 | Weapon defense (equipment) |
| magicDefense | uint16 | Magic defense (equipment) |
| accuracy | uint16 | Accuracy (equipment) |
| avoidability | uint16 | Avoidability (equipment) |
| hands | uint16 | Hands stat (equipment) |
| speed | uint16 | Speed stat (equipment) |
| jump | uint16 | Jump stat (equipment) |
| slots | uint16 | Upgrade slots remaining (equipment) |
| locked | bool | Lock state (equipment) |
| spikes | bool | Spike scroll applied (equipment) |
| karmaUsed | bool | Karma used (equipment) |
| cold | bool | Cold protection applied (equipment) |
| canBeTraded | bool | Trade eligibility (equipment) |
| levelType | byte | Equipment level type |
| level | byte | Equipment scroll level |
| experience | uint32 | Equipment experience |
| hammersApplied | uint32 | Hammers applied count |
| equippedSince | *time.Time | Equip timestamp |
| cashId | int64 | Cash shop identifier |
| commodityId | uint32 | Cash commodity identifier |
| purchaseBy | uint32 | Purchaser identifier |
| petId | uint32 | Pet reference identifier |

#### ModelBuilder

Fluent builder for constructing and cloning Asset models. Created via `NewBuilder(compartmentId, templateId)` for new instances or `Clone(model)` for copies. Provides `Set*` methods for absolute value assignment and `Add*` methods for delta-based stat modifications used by scroll enhancement. Delta methods clamp values to their type range (0 to max).

`Add*` methods available: `AddStrength`, `AddDexterity`, `AddIntelligence`, `AddLuck`, `AddHP`, `AddMP`, `AddWeaponAttack`, `AddMagicAttack`, `AddWeaponDefense`, `AddMagicDefense`, `AddAccuracy`, `AddAvoidability`, `AddHands`, `AddSpeed`, `AddJump`, `AddSlots`, `AddLevel`, `AddExperience`, `AddHammersApplied`.

### Invariants

- `Quantity()` returns the stored quantity for stackable items and non-pet cash items; returns 1 for equipment and pet items.
- `HasQuantity()` is true for stackable types (Use, Setup, ETC) and non-pet cash items.
- `IsStackable()` is true for Use, Setup, and ETC inventory types.
- `InventoryType()` is derived from `templateId` using `inventory.TypeFromItemId`.
- `IsEquipment()` is true when inventory type is Equip.
- `IsCashEquipment()` is true when equipment and cashId is non-zero.
- `IsConsumable()` is true when inventory type is Use.
- `IsCash()` is true when inventory type is Cash.
- `IsPet()` requires both cash inventory type and a non-zero petId.
- `Add*` builder methods clamp results to `[0, max]` for the target type (uint16, uint32, byte).

### State Transitions

Assets are read-only projections within this service. Stat modifications are applied locally via the builder pattern and then emitted as MODIFY_EQUIPMENT commands to the compartment service.

### Processors

None. The asset package provides models, builder, and REST transform/extract functions only.

## compartment

### Responsibility

Represents an inventory compartment (a typed section of inventory) containing assets. Issues Kafka commands to the inventory service for item reservation, consumption, destruction, cancellation, and equipment modification.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment identifier |
| characterId | uint32 | Owning character ID |
| inventoryType | inventory.Type | Inventory type (Equip, Use, Setup, ETC, Cash) |
| capacity | uint32 | Maximum slot count |
| assets | []asset.Model | Contained assets |

#### ModelBuilder

Fluent builder created via `NewBuilder(id, characterId, inventoryType, capacity)` or `Clone(model)`. Supports `SetCapacity`, `AddAsset`, `SetAssets`.

#### Reserves

| Field | Type | Description |
|-------|------|-------------|
| Slot | int16 | Source slot |
| ItemId | uint32 | Item template ID |
| Quantity | int16 | Quantity to reserve |

### Invariants

- `FindBySlot` returns the first asset matching the given slot position.
- `FindFirstByItemId` returns the first asset matching the given template ID.

### State Transitions

None locally. State changes are delegated via Kafka commands.

### Processors

- `RequestReserve`: Sends REQUEST_RESERVE command to compartment topic
- `ConsumeItem`: Sends CONSUME command to compartment topic
- `DestroyItem`: Sends DESTROY command to compartment topic
- `CancelItemReservation`: Sends CANCEL_RESERVATION command to compartment topic
- `Consume`: Creates a handler for compartment RESERVED events that delegates to an ItemConsumer callback

## consumable

### Responsibility

Core domain processor orchestrating all consumable item usage. Routes consumption requests by item classification, manages scroll enhancement logic, applies item effects (buffs, HP/MP recovery), and handles the reservation transaction lifecycle.

### Core Models

#### ItemConsumer

Function type: `func(l logrus.FieldLogger) func(ctx context.Context) error`

Callback invoked after item reservation is confirmed.

### Invariants

- Item consumption requires a valid item in the specified inventory slot
- Scroll usage requires available equipment upgrade slots (except clean slate, spike, and cold protection scrolls)
- Clean slate scrolls cannot add slots beyond the original equipment slot count
- Pet food consumption requires a spawned pet with hunger (fullness < 100)
- Cash pet food applies only to matching pet templates (filtered by cash item indexes)
- Equipment cursed by a failed scroll is destroyed
- White scrolls prevent slot loss on scroll failure (but do not prevent curse)

### State Transitions

1. `RequestItemConsume` / `RequestScroll` -> registers one-time event handler -> sends REQUEST_RESERVE
2. On RESERVED event -> executes item-specific logic -> sends CONSUME
3. On error -> sends CANCEL_RESERVATION -> emits ERROR event

### Processors

- `RequestItemConsume`: Routes to appropriate consumer based on item classification
  - Classifications 200-202: Standard consumables (ConsumeStandard)
  - ClassificationConsumableTownWarp: Town scrolls (ConsumeTownScroll)
  - ClassificationConsumablePetFood: Pet food (ConsumePetFood)
  - ClassificationPetConsumable: Cash pet food (ConsumeCashPetFood)
  - ClassificationConsumableSummoningSack: Summoning sacks (ConsumeSummoningSack)
- `RequestScroll`: Processes equipment enhancement scroll usage
- `ApplyItemEffects`: Applies stat buffs and HP/MP recovery from consumable data
- `ApplyConsumableEffect`: Applies effects without consuming an inventory item (NPC-initiated)
- `CancelConsumableEffect`: Cancels buff effects using sourceId = -int32(itemId)
- `ConsumeError`: Cancels reservation and emits error event
- `PassScroll` / `FailScroll`: Emit SCROLL events with result

### Scroll Processing

- Success roll: random(0-100) vs success rate
- Clean slate: Adds 1 slot (up to original max)
- Spike/Cold scrolls: Set special boolean properties
- Chaos scrolls: Randomize stats using weighted distribution
- Regular scrolls: Add stat increases, consume 1 slot, increment level
- Failed scrolls: Consume 1 slot unless white scroll used
- Cursed items: Destroyed on curse roll

### Chaos Scroll Stat Adjustment Distribution

| Adjustment | Probability |
|------------|-------------|
| -5 | 4.94% |
| -4 | 2.97% |
| -3 | 3.65% |
| -2 | 8.00% |
| -1 | 13.70% |
| 0 | 18.38% |
| +1 | 19.31% |
| +2 | 15.87% |
| +3 | 10.21% |
| +4 | 1.98% |
| +5 | 0.99% |

HP/MP adjustments are multiplied by 10.

## equipable

### Responsibility

Handles equipment stat modifications via the Change function type. Computes modified stats locally using the asset ModelBuilder, then emits a MODIFY_EQUIPMENT command.

### Core Models

#### Change

Function type: `func(b *asset.ModelBuilder)`

Applied to a cloned asset builder to modify equipment stats.

### Invariants

- Stat modifications are clamped to valid type ranges by the asset builder's `Add*` methods.
- ChangeStat clones the original asset, applies all changes, then emits the full modified stat set.

### State Transitions

None locally. Emits MODIFY_EQUIPMENT command to compartment topic.

### Processors

- `ChangeStat`: Clones asset, applies changes, emits MODIFY_EQUIPMENT command
- Factory functions for changes: `AddStrength`, `AddDexterity`, `AddIntelligence`, `AddLuck`, `AddHP`, `AddMP`, `AddWeaponAttack`, `AddMagicAttack`, `AddWeaponDefense`, `AddMagicDefense`, `AddAccuracy`, `AddAvoidability`, `AddHands`, `AddSpeed`, `AddJump`, `AddSlots`, `AddLevel`, `SetSpike`, `SetCold`

## character

### Responsibility

Fetches character data from the character service via REST and decorates it with inventory data. Issues character commands (change map, change HP/MP) via Kafka.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character ID |
| accountId | uint32 | Account ID |
| worldId | world.Id | World identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | job.Id | Job identifier |
| hp, maxHp | uint16 | Current and max HP |
| mp, maxMp | uint16 | Current and max MP |
| strength, dexterity, intelligence, luck | uint16 | Base stats |
| meso | uint32 | Meso balance |
| mapId | map.Id | Current map |
| pets | []pet.Model | Character's pets |
| equipment | equipment.Model | Equipped items |
| inventory | inventory.Model | Full inventory |

#### ModelBuilder

Fluent builder for character model construction. Created via `NewModelBuilder()` or `Clone(model)`.

### Invariants

- `SetInventory` partitions equip compartment assets by slot position: positive slots remain in the compartment, negative slots are mapped to equipment slots using `asset.Clone`. Slots below -100 are cash equipment.
- `HasSPTable` returns true only for Evan job line characters.
- `RemainingSp` indexes into the SP table based on the Evan skill book mapping.

### State Transitions

None locally. Character data is read via REST, mutations are issued via Kafka commands.

### Processors

- `GetById`: Fetches character from REST, applies optional decorators
- `InventoryDecorator`: Fetches inventory and calls `SetInventory` to populate equipment and inventory
- `ChangeMap`: Emits CHANGE_MAP command
- `ChangeHP`: Emits CHANGE_HP command
- `ChangeMP`: Emits CHANGE_MP command

## character/buff

### Responsibility

Issues buff apply and cancel commands to the character buff service via Kafka.

### Core Models

#### stat.Model

| Field | Type | Description |
|-------|------|-------------|
| Type | character.TemporaryStatType | Stat type (e.g., Accuracy, Speed) |
| Amount | int32 | Buff amount |

### Invariants

None.

### State Transitions

None locally. Emits commands to the buff topic.

### Processors

- `Apply`: Emits APPLY buff command with stat changes, duration, and sourceId
- `Cancel`: Emits CANCEL buff command with sourceId

## inventory

### Responsibility

Fetches full inventory data from the inventory service via REST. Provides a map of compartments keyed by inventory type.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Owning character ID |
| compartments | map[inventory.Type]compartment.Model | Compartments by type |

#### ModelBuilder

Fluent builder with `SetEquipable`, `SetConsumable`, `SetSetup`, `SetEtc`, `SetCash`, and generic `SetCompartment`. Also provides `FoldCompartment` for use with model fold operations.

### Invariants

- Accessor methods (`Equipable()`, `Consumable()`, `Setup()`, `ETC()`, `Cash()`) index into the compartments map by fixed inventory type constants.

### State Transitions

None. Read-only projection fetched via REST.

### Processors

- `GetByCharacterId`: Fetches inventory from REST
- `ByCharacterIdProvider`: Returns a model provider for inventory

## equipment

### Responsibility

Represents the character's currently equipped items organized by slot type.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| slots | map[slot.Type]slot.Model | Equipment slots by type |

#### slot.Model

| Field | Type | Description |
|-------|------|-------------|
| Position | slot.Position | Slot position value |
| Equipable | *asset.Model | Normal equipped item (pointer, may be nil) |
| CashEquipable | *asset.Model | Cash equipped item (pointer, may be nil) |

### Invariants

- `NewModel` initializes all slot types from the canonical slot definitions.
- `Get` returns the slot model and a boolean indicating presence.
- Equipment slot model uses pointers to the unified asset.Model for both regular and cash equipment.

### State Transitions

Equipment model is built locally during `character.SetInventory` by partitioning equip compartment assets.

### Processors

None.

## map

### Responsibility

Handles character teleportation by resolving portal spawn points and issuing map change commands.

### Core Models

None.

### Invariants

- `WarpRandom` selects a random spawn point portal (type 0, no target) from the destination map.

### State Transitions

None locally. Delegates to character.ChangeMap.

### Processors

- `WarpRandom`: Teleports character to random spawn point
- `WarpToPortal`: Teleports character to a specific portal

## map/character

### Responsibility

Manages a Redis-backed character location registry. Tracks which field (world, channel, map, instance) each character is currently in. This registry is populated by consuming character status events.

### Core Models

#### Registry

Redis-backed tenant registry mapping character IDs to their current field context. Initialized via `InitRegistry(client)` with a Redis client. Uses the `atlas-redis` `TenantRegistry` with key prefix `consumable-map-character`.

### Invariants

- The registry is initialized once via `InitRegistry` with a Redis client.
- All access is scoped per tenant via `tenant.MustFromContext`.
- `TransitionMap` and `TransitionChannel` both call `Enter`, overwriting the previous entry.

### State Transitions

- LOGIN event -> `Enter` (add to registry)
- LOGOUT event -> `Exit` (remove from registry)
- MAP_CHANGED event -> `TransitionMap` (overwrite entry)
- CHANNEL_CHANGED event -> `TransitionChannel` (overwrite entry)

### Processors

- `GetMap`: Retrieves character's current field context
- `Enter`: Registers character in the field
- `Exit`: Removes character from registry
- `TransitionMap`: Updates registry on map change
- `TransitionChannel`: Updates registry on channel change

## pet

### Responsibility

Fetches pet data from the pet service via REST, provides filtering (spawned, hungry, template matching), and issues pet commands via Kafka.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint64 | Pet identifier |
| inventoryItemId | uint32 | Inventory item ID |
| templateId | uint32 | Pet template ID |
| fullness | byte | Current fullness (0-100) |
| slot | int8 | Pet slot (-1 = despawned, 0+ = spawned) |

### Invariants

- `Spawned` filter: slot >= 0
- `Hungry` filter: fullness < 100
- `HungriestByOwnerProvider` returns the spawned hungry pet with the lowest fullness
- `IsTemplateFilter` matches pets by one or more template IDs

### State Transitions

None locally. Emits AWARD_FULLNESS command to pet topic.

### Processors

- `GetById`: Fetches pet from REST
- `GetByOwner`: Fetches all pets for a character
- `SpawnedByOwnerProvider`: Filters to spawned pets
- `HungryByOwnerProvider`: Filters to hungry spawned pets
- `HungriestByOwnerProvider`: Selects hungriest pet
- `AwardFullness`: Emits AWARD_FULLNESS command

## portal

### Responsibility

Fetches portal data from the data service via REST. Provides spawn point filtering and random selection for teleportation.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Portal identifier |
| name | string | Portal name |
| target | string | Target portal name |
| portalType | uint8 | Portal type (0 = spawn point) |
| x, y | int16 | Portal position |
| targetMapId | map.Id | Target map ID (999999999 = no target) |
| scriptName | string | Associated script |

### Invariants

- `SpawnPoint` filter: portalType == 0
- `NoTarget` filter: targetMapId == 999999999

### State Transitions

None. Read-only data.

### Processors

- `InMapProvider`: Fetches all portals in a map
- `RandomSpawnPointProvider`: Selects random spawn point portal (type 0, no target)
- `RandomSpawnPointIdProvider`: Returns the ID of a random spawn point

## data/consumable

### Responsibility

Fetches consumable item template data from the data service via REST.

### Core Models

#### Model

Contains consumable template properties including success/cursed rates, stat increases, spec map, monster summons, rewards, and various flags.

#### SummonModel

| Field | Type | Description |
|-------|------|-------------|
| templateId | uint32 | Monster template ID |
| probability | uint32 | Spawn probability (0-100) |

#### RewardModel

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Reward item ID |
| count | uint32 | Reward count |
| prob | uint32 | Reward probability |

### Spec Types

| Type | Key | Description |
|------|-----|-------------|
| SpecTypeHP | hp | Direct HP recovery |
| SpecTypeMP | mp | Direct MP recovery |
| SpecTypeHPRecovery | hpR | Percentage HP recovery |
| SpecTypeMPRecovery | mpR | Percentage MP recovery |
| SpecTypeMoveTo | moveTo | Town scroll destination map |
| SpecTypeWeaponAttack | pad | Weapon attack buff |
| SpecTypeMagicAttack | mad | Magic attack buff |
| SpecTypeWeaponDefense | pdd | Weapon defense buff |
| SpecTypeMagicDefense | mdd | Magic defense buff |
| SpecTypeSpeed | speed | Speed buff |
| SpecTypeEvasion | eva | Evasion buff |
| SpecTypeAccuracy | acc | Accuracy buff |
| SpecTypeJump | jump | Jump buff |
| SpecTypeTime | time | Buff duration (milliseconds) |
| SpecTypeMorph | morph | Morph transformation |
| SpecTypeThaw | thaw | Thaw status cure |
| SpecTypePoison | poison | Poison status cure |
| SpecTypeDarkness | darkness | Darkness status cure |
| SpecTypeWeakness | weakness | Weakness status cure |
| SpecTypeSeal | seal | Seal status cure |
| SpecTypeCurse | curse | Curse status cure |
| SpecTypeReturnMap | returnMapQR | Return map via quick return |
| SpecTypeIgnoreContinent | ignoreContinent | Ignore continent restriction |
| SpecTypeRandomMoveInFieldSet | randomMoveInFieldSet | Random move in field set |
| SpecTypeExperienceBuff | expBuff | Experience buff |
| SpecTypeInc | inc | Pet food fullness increment |
| SpecTypeOnlyPickup | onlyPickup | Pickup-only flag |

### Invariants

None.

### State Transitions

None. Read-only data.

### Processors

- `GetById`: Fetches consumable template from REST

## data/equipable

### Responsibility

Fetches equipable item template data from the data service via REST. Used for scroll validation (original slot count).

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| strength, dexterity, intelligence, luck | uint16 | Base stats |
| hp, mp | uint16 | Base HP/MP |
| weaponAttack, magicAttack | uint16 | Base attack stats |
| weaponDefense, magicDefense | uint16 | Base defense stats |
| accuracy, avoidability | uint16 | Base accuracy/evasion |
| speed, jump | uint16 | Base movement stats |
| slots | uint16 | Original upgrade slot count |

### Invariants

None.

### State Transitions

None. Read-only data.

### Processors

- `GetById`: Fetches equipable template from REST

## data/map

### Responsibility

Fetches map data from the data service via REST. Used for town scroll return map resolution.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| returnMapId | map.Id | Return map for town scrolls |

### Invariants

None.

### State Transitions

None. Read-only data.

### Processors

- `GetById`: Fetches map data from REST

## cash

### Responsibility

Fetches cash item data from the data service via REST. Used for cash pet food to determine fullness increment and pet template indexes.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Item identifier |
| slotMax | uint32 | Maximum slot count |
| spec | map[SpecType]int32 | Effect specifications |

### Invariants

- `Indexes()` extracts numbered index values (0-9) from the spec map, used for pet template matching.

### State Transitions

None. Read-only data.

### Processors

- `GetById`: Fetches cash item data from REST

## monster

### Responsibility

Creates monster instances via REST for summoning sack consumables.

### Core Models

None used as domain models.

### Invariants

None.

### State Transitions

Sends POST request to monster service.

### Processors

- `CreateMonster`: Creates a monster instance via REST POST

## monster/drop/position

### Responsibility

Resolves valid drop positions in a map via REST. Used by summoning sack logic to place spawned monsters.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X coordinate |
| y | int16 | Y coordinate |

### Invariants

None.

### State Transitions

None. Read-only data.

### Processors

- `GetInMap`: Fetches drop position from REST given map ID and initial/fallback coordinates
