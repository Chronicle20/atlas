# Domain

## Inventory

### Responsibility

Aggregates compartments for a character. Provides access to compartments by type.

### Core Models

- `Model` - Contains characterId and a map of compartments keyed by inventory type

### Invariants

- A character has one inventory
- An inventory contains compartments for each inventory type (Equip, Use, Setup, ETC, Cash)

### Processors

- `ProcessorImpl.Create` - Creates an inventory with compartments for all inventory types (default capacity 24)
- `ProcessorImpl.Delete` - Deletes an inventory and all contained compartments
- `ProcessorImpl.GetByCharacterId` - Retrieves inventory for a character

---

## Compartment

### Responsibility

Manages a typed inventory slot container with capacity limits. Handles asset operations including equipping, moving, dropping, reserving, and consuming.

### Core Models

- `Model` - Contains id, characterId, inventoryType, capacity, and assets
- `ReservationRequest` - Contains Slot, ItemId, and Quantity for reservation operations

### Invariants

- Capacity cannot exceed 96
- Slot numbers are 1-indexed for regular slots; negative slots indicate equipped positions
- Assets with negative slot values are equipped items
- Rechargeable assets cannot be stacked
- Assets with active reservations cannot be merged

### Processors

- `Processor.Create` - Creates a compartment with specified type and capacity
- `Processor.DeleteByModel` - Deletes a compartment and all contained assets
- `Processor.EquipItem` - Moves an asset from inventory slot to equipment slot
- `Processor.RemoveEquip` - Moves an asset from equipment slot to inventory slot
- `Processor.Move` - Moves or swaps assets between slots; merges stackable assets with same templateId
- `Processor.IncreaseCapacity` - Increases compartment capacity (max 96)
- `Processor.Drop` - Removes asset from compartment and creates map drop
- `Processor.RequestReserve` - Reserves assets for a transaction with timeout
- `Processor.CancelReservation` - Cancels a reservation
- `Processor.ConsumeAsset` - Consumes a reserved asset
- `Processor.DestroyAsset` - Destroys an asset or reduces quantity
- `Processor.CreateAsset` - Creates a new asset in the next free slot
- `Processor.RechargeAsset` - Adds quantity to a rechargeable asset (Use compartment only)
- `Processor.MergeAndCompact` - Merges stackable assets and compacts slots
- `Processor.CompactAndSort` - Compacts slots and sorts by templateId
- `Processor.Accept` - Accepts a cash item into the compartment
- `Processor.Release` - Releases an asset from the compartment
- `Processor.AttemptEquipmentPickUp` - Picks up equipment from a drop
- `Processor.AttemptItemPickUp` - Picks up an item from a drop

---

## Asset

### Responsibility

Represents an item in a compartment slot with type-specific reference data.

### Core Models

- `Model[E]` - Generic model containing id, compartmentId, slot, templateId, expiration, referenceId, referenceType, and referenceData
- `ReferenceType` - String enum: equipable, cash-equipable, consumable, setup, etc, cash, pet

### Reference Data Types

- `EquipableReferenceData` - Contains statistics, slots, owner, and equipment flags
- `CashEquipableReferenceData` - Contains cashId
- `ConsumableReferenceData` - Contains quantity, owner, flag, rechargeable
- `SetupReferenceData` - Contains quantity, owner, flag
- `EtcReferenceData` - Contains quantity, owner, flag
- `CashReferenceData` - Contains cashId, quantity, owner, flag, purchasedBy
- `PetReferenceData` - Contains cashId, owner, flag, purchasedBy, name, level, closeness, fullness, expiration, slot

### Invariants

- Assets with HasQuantity interface can have quantity updated
- Equipable types do not support quantity (always 1)

### Processors

- `Processor.Create` - Creates an asset with appropriate reference type based on inventory type
- `Processor.Delete` - Deletes an asset and its reference data
- `Processor.Drop` - Deletes an asset without deleting reference data
- `Processor.UpdateSlot` - Updates an asset's slot position
- `Processor.UpdateQuantity` - Updates quantity for stackable assets
- `Processor.Acquire` - Acquires an asset from an existing reference
- `Processor.Accept` - Accepts a cash item asset
- `Processor.Release` - Releases an asset without deleting reference
- `Processor.RelayUpdate` - Emits an update event for an asset

---

## Stackable

### Responsibility

Stores quantity and metadata for stackable items (consumables, setup, etc).

### Core Models

- `Model` - Contains id, quantity, ownerId, flag, rechargeable

### Processors

- `Processor.Create` - Creates a stackable with initial quantity
- `Processor.Delete` - Deletes a stackable by id
- `Processor.UpdateQuantity` - Updates the quantity

---

## Drop

### Responsibility

Coordinates with the drop service for item drops and pickups.

### Processors

- `Processor.CreateForEquipment` - Emits command to create equipment drop
- `Processor.CreateForItem` - Emits command to create item drop
- `Processor.CancelReservation` - Emits command to cancel drop reservation
- `Processor.RequestPickUp` - Emits command to request drop pickup
