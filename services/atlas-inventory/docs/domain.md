# Domain

## Inventory

### Responsibility

Aggregates compartments for a character. Provides access to compartments by type.

### Core Models

- `Model` - Contains characterId and a map of compartments keyed by inventory type (Equip, Use, Setup, ETC, Cash)
- `ModelBuilder` - Builder with `SetCompartment`, `SetEquipable`, `SetConsumable`, `SetSetup`, `SetEtc`, `SetCash`

### Invariants

- A character has one inventory
- An inventory contains one compartment for each of the five inventory types (Equip, Use, Setup, ETC, Cash)

### State Transitions

- Character created -> Inventory created (with five compartments, each capacity 24)
- Character deleted -> Inventory deleted (all compartments and assets deleted)

### Processors

- `ProcessorImpl.Create` - Creates an inventory with compartments for all inventory types (default capacity 24). Fails if inventory already exists for the character.
- `ProcessorImpl.Delete` - Deletes an inventory and all contained compartments and assets
- `ProcessorImpl.GetByCharacterId` - Retrieves inventory for a character by folding compartments into the model

---

## Compartment

### Responsibility

Manages a typed inventory slot container with capacity limits. Handles asset operations including equipping, moving, dropping, reserving, consuming, merging, sorting, accepting, and releasing.

### Core Models

- `Model` - Contains id (UUID), characterId, inventoryType, capacity, and assets ([]asset.Model)
- `ModelBuilder` - Builder with `SetCapacity`, `AddAsset`, `SetAssets`
- `ReservationRequest` - Contains Slot, ItemId, and Quantity for reservation operations
- `Reservation` - Contains id (UUID), itemId, quantity, expiry for tracking reserved assets
- `ReservationKey` - Composite key of tenant, characterId, inventoryType, slot

### Invariants

- Capacity cannot exceed 96
- Slot numbers are 1-indexed for regular slots; negative slots indicate equipped positions
- Assets with negative slot values are equipped items
- Rechargeable assets in Use compartments cannot be merged
- Assets with active reservations cannot be merged
- Only assets with the same templateId and HasQuantity can be merged
- Destination asset must not already be at slotMax to be eligible for merge
- Recharge operations are restricted to Use compartment type only
- Equipment slot conflicts (overall vs. pants, top) are resolved during equip operations
- Reservations have a 30-second timeout and are tracked in a Redis-backed registry
- Inventory operations acquire per-character, per-inventory-type Redis-backed distributed locks to prevent concurrent modification

### State Transitions

- Created -> assets added/removed/moved
- Asset equip: source slot -> equipment slot (negative); displaced equipment -> source slot
- Asset unequip: equipment slot -> next free slot (or specified destination if available)
- Asset move: source slot <-> destination slot (swap), or merge if stackable with same templateId
- Asset drop: slot freed, drop command emitted
- Asset consume: reservation removed, quantity decremented or asset deleted
- Asset destroy: quantity decremented or asset deleted
- Asset expire: asset deleted, optional replacement item created
- Accept: asset created in next free slot, or merged into existing stack
- Release: asset deleted (full release) or quantity reduced (partial release)

### Processors

- `Processor.Create` - Creates a compartment with specified type and capacity
- `Processor.DeleteByModel` - Deletes a compartment and all contained assets
- `Processor.EquipItem` - Moves an asset from inventory slot to equipment slot, handling overall/pants/top conflicts
- `Processor.RemoveEquip` - Moves an asset from equipment slot to inventory slot
- `Processor.Move` - Moves or swaps assets between slots; merges stackable assets with same templateId when eligible
- `Processor.IncreaseCapacity` - Increases compartment capacity (capped at 96)
- `Processor.Drop` - Removes asset from compartment (respecting reservations) and emits a drop command for equipment or items
- `Processor.RequestReserve` - Reserves assets for a transaction with 30-second timeout
- `Processor.CancelReservation` - Cancels a reservation
- `Processor.ConsumeAsset` - Consumes a reserved asset (decrements quantity or deletes)
- `Processor.DestroyAsset` - Destroys an asset or reduces quantity
- `Processor.ExpireAsset` - Expires an asset and optionally creates a replacement item
- `Processor.CreateAsset` - Creates a new asset in the next free slot; merges into existing stack for non-equip, non-rechargeable stackable items when possible
- `Processor.CreateAssetAndLock` - Same as CreateAsset but acquires the inventory lock first
- `Processor.RechargeAsset` - Adds quantity to a rechargeable asset (Use compartment only)
- `Processor.MergeAndCompact` - Merges stackable assets with same templateId, then compacts slots to fill gaps
- `Processor.CompactAndSort` - Compacts slots to fill gaps, then sorts by templateId using selection sort
- `Processor.Accept` - Accepts an asset into the compartment, merging into existing stack when possible; emits ERROR event on failure
- `Processor.Release` - Releases an asset from the compartment (full or partial by quantity); emits ERROR event on failure
- `Processor.AttemptEquipmentPickUp` - Picks up equipment from a drop, building asset from EquipmentData; cancels drop reservation on failure
- `Processor.AttemptItemPickUp` - Picks up a stackable item from a drop, merging into existing stacks when possible; cancels drop reservation on failure
- `Processor.ModifyEquipment` - Updates equipment stats for an existing asset

---

## Asset

### Responsibility

Represents a unified inventory item in a compartment slot. All item types (equipment, consumable, setup, etc, cash, pet) are stored in a single flattened model containing fields for all possible item attributes.

### Core Models

- `Model` - Contains:
  - Identity: id (uint32), compartmentId (UUID), slot (int16), templateId (uint32), expiration, createdAt
  - Stackable fields: quantity, ownerId, flag, rechargeable
  - Equipment fields: strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots, levelType, level, experience, hammersApplied, equippedSince
  - Cash fields: cashId, commodityId, purchaseBy
  - Pet reference: petId
- `ModelBuilder` - Builder with setters for all fields; constructed via `NewBuilder(compartmentId, templateId)` or `Clone(model)`

### Invariants

- Inventory type is derived from templateId via `inventory.TypeFromItemId`
- `IsEquipment()` is true when inventory type is Equip
- `IsCashEquipment()` is true when equipment and cashId is non-zero
- `IsStackable()` is true for Use, Setup, and ETC inventory types
- `HasQuantity()` is true for stackable items and non-pet cash items
- `Quantity()` returns 1 for items where HasQuantity is false
- `IsPet()` is true for cash items with petId > 0
- Equipment stats are randomized within +/-10% of base stats (capped at a per-stat maximum range) when created via `Processor.Create`
- Quantity cannot be updated for non-HasQuantity assets

### State Transitions

- Created -> slot updated, quantity updated, equipment stats updated, deleted
- Equipment creation: base stats fetched from data service, randomized within tolerance
- Cash pet creation: pet created via pet service, asset populated with pet reference data
- Stackable/cash creation: quantity, ownerId, flag, rechargeable set from input

### Processors

- `Processor.Create` - Creates an asset with type-specific initialization based on inventory type:
  - Equip: fetches base stats from equipment statistics data service, randomizes within tolerance
  - Use/Setup/ETC: sets quantity, ownerId, flag, rechargeable
  - Cash (pet): creates pet via pet service, sets petId, cashId, ownerId, flag, expiration, purchaseBy
  - Cash (non-pet): sets quantity, ownerId, flag
- `Processor.CreateFromModel` - Creates an asset from a pre-built Model (used for equipment pickup)
- `Processor.Delete` - Soft-deletes an asset and emits DELETED event
- `Processor.Drop` - Soft-deletes an asset and emits DELETED event (same as Delete)
- `Processor.Expire` - Soft-deletes an asset and emits EXPIRED event
- `Processor.UpdateSlot` - Updates an asset's slot position; emits MOVED event unless moving to/from temporary slot
- `Processor.UpdateQuantity` - Updates quantity for assets with HasQuantity; emits QUANTITY_CHANGED event
- `Processor.UpdateEquipmentStats` - Updates all equipment stat fields on an asset; emits UPDATED event
- `Processor.Accept` - Creates an asset in a specific compartment and slot; emits ACCEPTED event
- `Processor.Release` - Soft-deletes an asset; emits RELEASED event
- `Processor.DeleteAndEmit` - Looks up asset by ID and deletes with event emission
- `Processor.GetSlotMax` - Retrieves maximum slot capacity for a templateId from the appropriate data service (consumable, setup, or etc); returns 1 for equipment and other types

---

## Drop

### Responsibility

Coordinates with the drop service for item drops and pickups via Kafka commands.

### Processors

- `Processor.CreateForEquipment` - Emits SPAWN_FROM_CHARACTER command with equipment data
- `Processor.CreateForItem` - Emits SPAWN_FROM_CHARACTER command with item quantity
- `Processor.CancelReservation` - Emits CANCEL_RESERVATION command
- `Processor.RequestPickUp` - Emits REQUEST_PICK_UP command
