# Domain

## Storage

### Responsibility

Represents an account-level storage container within a world. Holds mesos and references to stored assets.

### Core Models

**Model**
- `id`: UUID - Unique identifier
- `worldId`: byte - World identifier
- `accountId`: uint32 - Account identifier
- `capacity`: uint32 - Maximum number of assets
- `mesos`: uint32 - Stored currency
- `assets`: []asset.Model[any] - Stored assets (decorated)

### Invariants

- Storage is unique per tenant, world, and account combination
- Capacity defaults to 4
- Mesos defaults to 0
- Assets cannot exceed capacity

### Processors

**Processor**
- `GetOrCreateStorage`: Retrieves existing storage or creates new storage for world and account
- `GetStorageByWorldAndAccountId`: Retrieves storage by world and account
- `CreateStorage`: Creates new storage, returns error if already exists
- `Deposit`: Deposits an item into storage, creates stackable data for stackable items
- `DepositAndEmit`: Deposits an item and emits DEPOSITED event
- `Withdraw`: Withdraws an item from storage, supports partial withdrawal for stackables
- `WithdrawAndEmit`: Withdraws an item and emits WITHDRAWN event
- `UpdateMesos`: Updates mesos using SET, ADD, or SUBTRACT operations
- `UpdateMesosAndEmit`: Updates mesos and emits MESOS_UPDATED event
- `DepositRollback`: Rolls back a deposit operation
- `Accept`: Accepts an item into storage as part of transfer saga, merges stackables when possible
- `AcceptAndEmit`: Accepts an item and emits ACCEPTED status event
- `Release`: Releases an item from storage as part of transfer saga, supports partial release
- `ReleaseAndEmit`: Releases an item and emits RELEASED status event
- `MergeAndSort`: Merges stackable items with same templateId/ownerId/flag and sorts by templateId
- `ArrangeAndEmit`: Arranges storage and emits ARRANGED event
- `EmitProjectionCreatedEvent`: Emits PROJECTION_CREATED event
- `EmitProjectionDestroyedEvent`: Emits PROJECTION_DESTROYED event

---

## Asset

### Responsibility

Represents a stored item within storage. Assets are generic containers that reference type-specific data (equipable, stackable, pet).

### Core Models

**Model[E any]**
- `id`: uint32 - Unique identifier (auto-increment)
- `storageId`: UUID - Parent storage identifier
- `inventoryType`: InventoryType - Category (equip, use, setup, etc, cash)
- `slot`: int16 - Position within storage
- `templateId`: uint32 - Item template identifier
- `expiration`: time.Time - Expiration timestamp
- `referenceId`: uint32 - Reference to type-specific data
- `referenceType`: ReferenceType - Type of referenced data
- `referenceData`: E - Decorated reference data (generic)

**ReferenceType**
- `equipable`: Standard equipment
- `cash_equipable`: Cash shop equipment
- `consumable`: Use items (stackable)
- `setup`: Setup items (stackable)
- `etc`: Etc items (stackable)
- `cash`: Cash items
- `pet`: Pet items

**InventoryType**
- `1` (Equip)
- `2` (Use)
- `3` (Setup)
- `4` (Etc)
- `5` (Cash)

### Invariants

- InventoryType is derived from templateId (templateId / 1000000)
- Stackable types: consumable, setup, etc
- Assets belong to exactly one storage

### Processors

**Processor**
- `GetAssetById`: Retrieves an asset by ID
- `GetAssetsByStorageId`: Retrieves all assets for a storage
- `GetOrCreateStorageId`: Retrieves or creates storage and returns its ID
- `DecorateAsset`: Adds reference data based on asset type
- `DecorateEquipable`: Loads equipable data from atlas-equipables service
- `DecorateStackable`: Loads stackable data from local table
- `DecoratePet`: Loads pet data from atlas-pets service
- `DecorateAll`: Decorates multiple assets
- `GetByStorageIdDecorated`: Retrieves and decorates all assets for a storage

---

## Stackable

### Responsibility

Represents quantity and ownership data for stackable items (consumables, setup, etc).

### Core Models

**Model**
- `assetId`: uint32 - Parent asset identifier
- `quantity`: uint32 - Item count
- `ownerId`: uint32 - Owner character identifier
- `flag`: uint16 - Item flags

### Invariants

- AssetId is the primary key
- Quantity defaults to 1
- OwnerId defaults to 0
- Flag defaults to 0

---

## Projection

### Responsibility

Represents an in-memory projection of storage state for an active character session. Used to provide fast read access during storage UI interactions.

### Core Models

**Model**
- `characterId`: uint32 - Active character identifier
- `accountId`: uint32 - Account identifier
- `worldId`: byte - World identifier
- `storageId`: UUID - Storage identifier
- `capacity`: uint32 - Storage capacity
- `mesos`: uint32 - Stored mesos
- `npcId`: uint32 - NPC that opened storage
- `compartments`: map[InventoryType][]asset.Model[any] - Assets organized by inventory type

### Invariants

- CharacterId is required
- AccountId is required
- StorageId is required
- Projections are keyed by characterId in the manager
- Projections are removed on logout, channel change, or map change

### Processors

**Manager (Singleton)**
- `Get`: Retrieves a projection for a character
- `Create`: Stores a projection for a character
- `Delete`: Removes a projection for a character
- `Update`: Atomically updates a projection using a function

**BuildProjection**
- Creates a new projection from storage data
- Initializes all compartments with all assets
