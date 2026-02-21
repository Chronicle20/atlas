# Domain

## Storage

### Responsibility

Represents an account-level storage container within a world. Holds mesos and references to stored assets.

### Core Models

**Model**
- `id`: UUID - Unique identifier
- `worldId`: world.Id - World identifier
- `accountId`: uint32 - Account identifier
- `capacity`: uint32 - Maximum number of assets
- `mesos`: uint32 - Stored currency
- `assets`: []asset.Model - Stored assets

**ModelBuilder**
- Constructs Model instances with builder pattern
- `validate()` enforces: id is required, accountId is required, capacity must be greater than 0
- `Build()` returns error on validation failure
- `MustBuild()` panics on validation failure (used for trusted data from database entities)

### Invariants

- Storage is unique per tenant, world, and account combination
- Capacity defaults to 4
- Mesos defaults to 0
- `HasCapacity()` returns true when asset count is less than capacity
- `NextFreeSlot()` finds the first unoccupied 0-indexed slot up to capacity

### State Transitions

- Nonexistent -> Created: via `GetOrCreateStorage` or `CreateStorage`
- Created -> Deposited: via `Deposit` (creates new asset in storage)
- Deposited -> Withdrawn: via `Withdraw` (deletes asset or reduces stackable quantity)
- Mesos updated: via `UpdateMesos` with SET, ADD, or SUBTRACT operations
- Arranged: via `MergeAndSort` (merges stackable items and sorts by templateId)
- Accepted: via `Accept` (receives item from transfer saga, merges stackables when possible)
- Released: via `Release` (sends item out in transfer saga, supports partial for stackables)
- Expired: via `ExpireAndEmit` (deletes expired asset, optionally creates replacement item)
- Deleted: via `DeleteByAccountId` (cascade deletes storage and all associated assets)

### Processors

**Processor**
- `GetOrCreateStorage`: Retrieves existing storage or creates new storage for world and account
- `GetStorageByWorldAndAccountId`: Retrieves storage by world and account
- `CreateStorage`: Creates new storage, returns error if already exists
- `Deposit`: Deposits an item into storage by creating a new asset entity with all fields inline
- `DepositAndEmit`: Deposits an item and emits DEPOSITED event
- `Withdraw`: Withdraws an item from storage; for stackable items where quantity is greater than 0 and less than the current quantity, reduces the quantity; otherwise deletes the asset
- `WithdrawAndEmit`: Withdraws an item and emits WITHDRAWN event
- `UpdateMesos`: Updates mesos using SET, ADD, or SUBTRACT operations; SUBTRACT clamps to 0 on underflow
- `UpdateMesosAndEmit`: Updates mesos and emits MESOS_UPDATED event; emits error event if SUBTRACT would underflow
- `DepositRollback`: Rolls back a deposit by deleting the asset
- `Accept`: Accepts an item into storage as part of a transfer saga; for stackable items, attempts to merge with existing stacks before creating a new asset (see Accept Merge Rules below)
- `AcceptAndEmit`: Accepts an item and emits ACCEPTED compartment status event
- `Release`: Releases an item from storage as part of a transfer saga; for non-stackable items or when quantity is 0, deletes the asset; for stackable items where quantity is less than the current quantity, reduces the quantity; otherwise deletes the asset
- `ReleaseAndEmit`: Releases an item and emits RELEASED compartment status event
- `MergeAndSort`: Groups stackable items by (templateId, ownerId, flag), merges quantities up to slotMax, deletes excess assets, then sorts by templateId within each inventory type
- `ArrangeAndEmit`: Arranges storage and emits ARRANGED event
- `ExpireAndEmit`: Deletes an expired asset from storage; if a replacement item ID is provided, creates a new asset for the replacement; emits EXPIRED event
- `DeleteByAccountId`: Deletes all storage records and associated assets for an account across all worlds
- `EmitProjectionCreatedEvent`: Emits PROJECTION_CREATED event with character, account, world, channel, and NPC identifiers
- `EmitProjectionDestroyedEvent`: Emits PROJECTION_DESTROYED event

**Accept Merge Rules**
- Only stackable items are candidates for merging
- Quantity defaults to 1 if zero
- Existing assets with the same templateId in the same storage are checked
- Rechargeable assets (rechargeable > 0) are skipped
- Only assets with matching ownerId and flag can merge
- Merge occurs if the combined quantity fits within slotMax
- slotMax is looked up from atlas-data; defaults to 100 if lookup fails
- On successful merge, returns the existing asset ID and slot
- If no merge is possible, a new asset is created at the next slot

**Merge Rules (MergeAndSort)**
- Equipment assets are never merged
- Rechargeable consumables (rechargeable > 0) are never merged
- Only assets with the same templateId, ownerId, and flag can merge
- Merged stacks respect slotMax from atlas-data; defaults to 100 if lookup fails
- After merging, assets are sorted by templateId within each inventory type compartment
- Excess assets (empty after merge) are deleted

---

## Asset

### Responsibility

Represents a stored item within storage. The asset model is a unified flat structure that contains all fields for every item type (equipment stats, stackable quantities, cash item data, pet references). The item type is determined by the templateId.

### Core Models

**Model**
- `id`: uint32 - Unique identifier (auto-increment)
- `storageId`: UUID - Parent storage identifier
- `slot`: int16 - Position within storage
- `templateId`: uint32 - Item template identifier
- `expiration`: time.Time - Expiration timestamp
- Stackable fields: `quantity` (uint32), `ownerId` (uint32), `flag` (uint16), `rechargeable` (uint64)
- Equipment fields: `strength`, `dexterity`, `intelligence`, `luck`, `hp`, `mp`, `weaponAttack`, `magicAttack`, `weaponDefense`, `magicDefense`, `accuracy`, `avoidability`, `hands`, `speed`, `jump`, `slots` (all uint16); `levelType`, `level` (byte); `experience`, `hammersApplied` (uint32)
- Cash fields: `cashId` (int64), `commodityId` (uint32), `purchaseBy` (uint32)
- Pet reference: `petId` (uint32)
- Flag-derived helpers: `Locked()`, `Spikes()`, `KarmaUsed()`, `Cold()`, `CanBeTraded()` — computed from `flag` bitmask

**ModelBuilder**
- Constructs Model instances with builder pattern via `NewBuilder(storageId, templateId)`
- `Clone(m)` creates a builder pre-populated from an existing model

### Invariants

- InventoryType is derived from templateId at runtime (`templateId / 1000000`): 1=Equip, 2=Use, 3=Setup, 4=Etc, 5=Cash
- `IsEquipment()`: inventory type is Equip
- `IsCashEquipment()`: inventory type is Equip and cashId is nonzero
- `IsConsumable()`: inventory type is Use
- `IsSetup()`: inventory type is Setup
- `IsEtc()`: inventory type is Etc
- `IsCash()`: inventory type is Cash
- `IsPet()`: inventory type is Cash and petId is greater than 0
- `IsStackable()`: inventory type is Use, Setup, or Etc
- `HasQuantity()`: stackable items or non-pet cash items
- `Quantity()` returns the quantity field for items that have quantity, otherwise returns 1
- Assets belong to exactly one storage

### Processors

**Processor**
- `GetAssetById`: Retrieves an asset by ID
- `GetAssetsByStorageId`: Retrieves all assets for a storage, ordered by inventory_type then template_id; assigns dynamic sequential slots
- `GetOrCreateStorageId`: Retrieves or creates storage and returns its UUID

**Transform**
- `Transform`: Converts a Model to a RestModel with all fields
- `TransformAll`: Transforms a slice of Models

**Extract**
- `Extract`: Converts a RestModel back to a Model

---

## Projection

### Responsibility

Represents an in-memory projection of storage state for an active character session. Assets are organized into compartments by inventory type (equip, use, setup, etc, cash) for fast read access during storage UI interactions.

### Core Models

**Model**
- `characterId`: uint32 - Active character identifier
- `accountId`: uint32 - Account identifier
- `worldId`: world.Id - World identifier
- `storageId`: UUID - Storage identifier
- `capacity`: uint32 - Storage capacity
- `mesos`: uint32 - Stored mesos
- `npcId`: uint32 - NPC that opened storage
- `compartments`: map[inventory.Type][]asset.Model - Assets organized by inventory type

**Builder**
- Constructs Model instances with builder pattern
- `validate()` enforces: characterId is required, accountId is required, storageId is required
- `Clone(m)` creates a builder from an existing model with deep-copied compartments

### Invariants

- CharacterId is required
- AccountId is required
- StorageId is required
- Projections are keyed by characterId in the manager
- Projections are removed on logout, channel change, or map change
- Compartment commands (ACCEPT/RELEASE) trigger projection updates by refreshing the affected inventory type compartment from the database

### Processors

**Manager (Singleton)**
- `Get`: Retrieves a projection for a character
- `Create`: Stores a projection for a character
- `Delete`: Removes a projection for a character
- `Update`: Atomically updates a projection using a provided function; returns true if the projection existed and was updated

**BuildProjection**
- Creates a new projection from storage data
- Groups all storage assets by their inventory type into compartments

---

## NpcContextCache

### Responsibility

A legacy singleton cache that tracks which NPC a character is interacting with for storage. Entries have a TTL-based expiration.

### Core Models

**NpcContextCache**
- Redis-backed cache keyed by `atlas:npc-context:{characterId}`
- `Get`: Retrieves NPC ID for a character if not expired
- `Put`: Stores NPC context with TTL
- `Remove`: Clears NPC context for a character

### Invariants

- Entries expire based on Redis TTL
- Cleaned up on storage close, logout, channel change, and map change
