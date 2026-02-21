# Domain

## Wallet

### Responsibility
Manages cash shop currency balances for accounts.

### Core Models

#### Model
- `id` (uuid.UUID): Unique identifier
- `accountId` (uint32): Associated account
- `credit` (uint32): Credit currency balance
- `points` (uint32): Points currency balance
- `prepaid` (uint32): Prepaid currency balance

### Invariants
- Each account has one wallet
- Currency balances cannot go negative
- Currency type 1 = credit, 2 = points, 3 = prepaid

### State Transitions
- `Purchase(currency, amount)`: Returns a new Model with the specified currency reduced by amount. Does not validate balance; caller is responsible for checking before calling.

### Processors

#### Processor
- `ByAccountIdProvider`: Provides wallet by account ID
- `GetByAccountId`: Retrieves wallet for an account
- `Create`: Creates a new wallet with initial balances
- `Update`: Updates wallet balances
- `UpdateWithTransaction`: Updates wallet with transaction ID for saga coordination
- `AdjustCurrency`: Adjusts a specific currency type by amount, validates sufficient balance
- `AdjustCurrencyWithTransaction`: Adjusts currency with transaction ID
- `Delete`: Deletes a wallet
- `WithTransaction`: Returns a new processor scoped to a database transaction

---

## Wishlist

### Responsibility
Manages character wishlists for cash shop commodities.

### Core Models

#### Model
- `id` (uuid.UUID): Unique identifier
- `characterId` (uint32): Owner character
- `serialNumber` (uint32): Serial number of the wished commodity

### Invariants
- Wishlist items are associated with a character

### Processors

#### Processor
- `ByCharacterIdProvider`: Provides wishlist items by character ID
- `GetByCharacterId`: Retrieves all wishlist items for a character
- `Add`: Adds an item to the wishlist
- `Delete`: Removes an item from the wishlist
- `DeleteAll`: Clears all items from a character's wishlist

---

## Inventory

### Responsibility
Represents a cash shop inventory containing compartments organized by character type.

### Core Models

#### Model
- `accountId` (uint32): Associated account
- `compartments` (map[CompartmentType]Compartment): Compartments indexed by type

### Invariants
- Each account has one inventory
- Inventory contains three compartments: Explorer, Cygnus, Legend

### Processors

#### Processor
- `ByAccountIdProvider`: Provides inventory by account ID, assembling compartments from the database
- `GetByAccountId`: Retrieves inventory for an account
- `Create`: Creates inventory with three default compartments (Explorer, Cygnus, Legend) at default capacity
- `Delete`: Deletes all compartments for the account
- `WithTransaction`: Returns a new processor scoped to a database transaction

---

## Compartment

### Responsibility
Represents a section of cash inventory for a specific character type. Contains assets and mediates accept/release operations that coordinate with saga orchestration.

### Core Models

#### CompartmentType
- `TypeExplorer` (1): Explorer character type
- `TypeCygnus` (2): Cygnus character type
- `TypeLegend` (3): Legend character type

#### Model
- `id` (uuid.UUID): Unique identifier
- `accountId` (uint32): Associated account
- `type_` (CompartmentType): Compartment type
- `capacity` (uint32): Maximum number of assets
- `assets` ([]asset.Model): Assets in the compartment

#### DefaultCapacity
- Constant value: 55

### Invariants
- Default capacity is 55
- Assets count cannot exceed capacity (enforced during purchase)
- Assets are lazily decorated onto the model when fetched via `DecorateAssets`

### State Transitions
- `Accept`: Creates a new flattened asset in the compartment using `CreateWithCashId` (idempotent by cashId). Emits ACCEPTED on success, ERROR on failure.
- `Release`: Validates the asset exists in the compartment via `FindById`, then soft-deletes it. Emits RELEASED on success, ERROR on failure.

### Processors

#### Processor
- `GetById`: Retrieves compartment by ID (with asset decoration)
- `ByIdProvider`: Provides compartment by ID
- `GetByAccountIdAndType`: Retrieves compartment by account and type
- `ByAccountIdAndTypeProvider`: Provides compartment by account and type
- `AllByAccountIdProvider`: Provides all compartments for an account
- `GetByAccountId`: Retrieves all compartments for an account
- `Create`: Creates a new compartment
- `UpdateCapacity`: Updates compartment capacity
- `Delete`: Deletes a compartment
- `DeleteAllByAccountId`: Deletes all compartments for an account
- `Accept`: Accepts an asset into a compartment (creates flattened asset with preserved cashId)
- `Release`: Releases an asset from a compartment (validates existence, then deletes)
- `WithTransaction`: Returns a new processor scoped to a database transaction

---

## Asset

### Responsibility
Represents a cash shop item stored in a compartment. The asset model is flattened: all item data (template, quantity, cash ID, expiration, etc.) is stored directly on the asset rather than referencing a separate item entity.

### Core Models

#### Model
- `id` (uint32): Unique identifier (auto-incremented)
- `compartmentId` (uuid.UUID): Parent compartment
- `cashId` (int64): Unique cash item identifier (randomly generated or externally provided)
- `templateId` (uint32): Item template ID
- `commodityId` (uint32): Commodity catalog entry ID (0 if not from commodity purchase)
- `quantity` (uint32): Item quantity
- `flag` (uint16): Item flags
- `purchasedBy` (uint32): Character ID that purchased the item
- `expiration` (time.Time): Item expiration time (zero time means permanent)
- `createdAt` (time.Time): Timestamp of creation

#### ModelBuilder
- Builder pattern via `NewBuilder(compartmentId, templateId)` and `Clone(model)`
- Setters: `SetId`, `SetCompartmentId`, `SetCashId`, `SetTemplateId`, `SetCommodityId`, `SetQuantity`, `SetFlag`, `SetPurchasedBy`, `SetExpiration`, `SetCreatedAt`

### Invariants
- Cash ID is unique within a tenant; generated randomly on creation or accepted from external source
- `CreateWithCashId` uses find-or-create semantics: if an asset with the given cashId already exists, it returns the existing one (idempotent)
- Flag defaults to 0 on creation

### State Transitions
- `Create`: Generates a unique cashId, calculates expiration from commodity period and hourly configuration, creates the asset, emits CREATED status
- `CreateWithCashId`: Find-or-create by cashId. Used during compartment Accept to preserve the cashId from external systems
- `UpdateQuantity`: Updates quantity in-place
- `Release`: Soft-deletes the asset
- `Delete`: Soft-deletes the asset
- `Expire`: Deletes the asset, emits EXPIRED status, optionally creates a replacement asset with the given replaceItemId

### Processors

#### Processor
- `ByIdProvider`: Provides asset by ID
- `GetById`: Retrieves asset by ID
- `ByCompartmentIdProvider`: Provides all assets for a compartment
- `GetByCompartmentId`: Retrieves all assets for a compartment
- `Create`: Creates a new asset (generates cashId, calculates expiration)
- `CreateAndEmit`: Creates asset and emits Kafka event
- `CreateWithCashId`: Creates or finds asset by cashId (idempotent)
- `CreateWithCashIdAndEmit`: Creates or finds asset by cashId and emits Kafka event
- `UpdateQuantity`: Updates asset quantity
- `Delete`: Soft-deletes an asset
- `DeleteAndEmit`: Deletes asset and emits Kafka event
- `Release`: Soft-deletes an asset (alias for delete with logging)
- `ReleaseAndEmit`: Releases asset and emits Kafka event
- `Expire`: Expires an asset, optionally creating a replacement
- `ExpireAndEmit`: Expires asset and emits Kafka events

---

## Asset Reservation

### Responsibility
In-memory thread-safe cache for tracking temporary asset reservations during purchase flows.

### Core Models

#### ReservationCache (singleton)
- `reservations` (map[uint32]uint32): Maps item ID to reserving character ID
- `expirations` (map[uint32]time.Time): Maps item ID to reservation expiry

### Invariants
- Reservations expire after 5 minutes
- Only one character can reserve a given item at a time
- A background goroutine cleans up expired reservations every minute
- Singleton instance via `GetInstance()`

### State Transitions
- `Reserve(itemID, characterID)`: Attempts to reserve; returns false if already reserved by another character (unless expired)
- `Release(itemID)`: Immediately releases the reservation
- `IsReserved(itemID)`: Checks reservation status, auto-clears if expired

---

## Expiration Calculation

### Responsibility
Determines expiration timestamps for newly created cash assets based on commodity period and per-template hourly configuration.

### Invariants
- `period == 0`: Permanent item, returns zero time (no expiration)
- `period != 1`: Standard day-based expiration, returns `now + period days`
- `period == 1`: Checks hourly config map for the template ID; if found, returns `now + hours`; otherwise returns `now + 1 day`

---

## Character (REST Client)

### Responsibility
Fetches character data from the external atlas-characters service. Used during purchase flows to resolve account ID, job type, and inventory state.

### Core Models

#### Model
- `id` (uint32): Character ID
- `accountId` (uint32): Associated account
- `worldId` (world.Id): World
- `jobId` (job.Id): Character job
- `inventory` (inventory.Model): Character inventory (lazily decorated)
- `equipment` (equipment.Model): Equipment slots (derived from inventory)
- Additional fields: name, gender, skinColor, face, hair, level, stats, ap, sp, experience, fame, mapId, meso, x, y, stance

#### Equipment Model
- `slots` (map[slot.Type]slot.Model): Equipment slots indexed by type
- Each slot holds a `Position`, optional `Equipable`, and optional `CashEquipable`

### Processors

#### Processor
- `GetById`: Fetches character from atlas-characters via REST, applies optional decorators
- `InventoryDecorator`: Decorates a character model with inventory data fetched from atlas-inventory

---

## Character Inventory (REST Client)

### Responsibility
Fetches character inventory data from the external atlas-inventory service.

### Core Models

#### Model
- `characterId` (uint32): Owner character
- `compartments` (map[inventory.Type]compartment.Model): Compartments indexed by type (Equip, Use, Setup, ETC, Cash)

### Processors

#### Processor
- `ByCharacterIdProvider`: Provides character inventory by character ID via REST
- `GetByCharacterId`: Retrieves inventory for a character

---

## Character Compartment (Command Emitter)

### Responsibility
Emits commands to increase character inventory compartment capacity. Used during inventory capacity increase purchases.

### Core Models

#### Model
- `id` (uuid.UUID): Compartment ID
- `characterId` (uint32): Owner character
- `inventoryType` (inventory.Type): Inventory type
- `capacity` (uint32): Current capacity
- `assets` ([]asset.Model[any]): Assets in the compartment

### Processors

#### Processor
- `IncreaseCapacity`: Emits an INCREASE_CAPACITY command to the character compartment command topic

---

## Commodity (REST Client)

### Responsibility
Fetches commodity catalog data from the external atlas-data service. Used during purchases to resolve item ID, price, count, and period.

### Core Models

#### Model
- `id` (uint32): Commodity serial number
- `itemId` (uint32): Item template ID
- `count` (uint32): Item quantity
- `price` (uint32): Cost in currency units
- `period` (uint32): Expiration period in days (0 = permanent, 1 = check hourly config)
- `priority` (uint32): Display priority
- `gender` (byte): Gender restriction
- `onSale` (bool): Whether currently on sale

### Processors

#### Processor
- `GetById`: Fetches commodity by serial number from atlas-data via REST

---

## Configuration

### Responsibility
Thread-safe registry for tenant-specific configuration. Caches tenant config fetched from the configurations service. Provides hourly expiration mappings for asset expiration calculation.

### Core Models

#### Tenant Configuration
- `CashShop.Commodities.HourlyExpirations` ([]HourlyExpiration): Per-template hourly expiration overrides
  - `TemplateId` (uint32): Item template ID
  - `Hours` (uint32): Expiration in hours

### Invariants
- Configurations are cached per tenant ID after first fetch
- Cache uses double-checked locking (RWMutex)
- If fetch fails, defaults to empty configuration

### Processors
- `GetTenantConfig`: Retrieves cached or fetches tenant configuration
- `GetHourlyExpirations`: Returns a map of templateId to hours from tenant config

---

## Asset (Generic Model)

### Responsibility
Generic polymorphic asset model used to represent character inventory items fetched from external services. Parameterized by reference data type.

### Core Models

#### Model[E]
- `id` (uint32): Asset ID
- `slot` (int16): Inventory slot position
- `templateId` (uint32): Item template ID
- `expiration` (time.Time): Expiration time
- `referenceId` (uint32): Reference identifier
- `referenceType` (ReferenceType): Type discriminator
- `referenceData` (E): Type-specific data

#### ReferenceType
- `ReferenceTypeEquipable` ("equipable")
- `ReferenceTypeConsumable` ("consumable")
- `ReferenceTypeSetup` ("setup")
- `ReferenceTypeEtc` ("etc")
- `ReferenceTypeCash` ("cash")
- `ReferenceTypePet` ("pet")

#### Reference Data Types
- `EquipableReferenceData`: Stats, flags, level info, hammers, expiration
- `CashEquipableReferenceData`: Same as equipable plus cashId
- `ConsumableReferenceData`: Quantity, ownerId, flag, rechargeable
- `SetupReferenceData`: Quantity, ownerId, flag
- `EtcReferenceData`: Quantity, ownerId, flag
- `CashReferenceData`: CashId, quantity, ownerId, flag, purchaseBy
- `PetReferenceData`: CashId, ownerId, flag, purchaseBy, name, level, closeness, fullness, expiration, slot, attributes

### Invariants
- Quantity defaults to 1 unless the reference data implements `HasQuantity`
- Type checks via `IsEquipable()`, `IsConsumable()`, `IsSetup()`, `IsEtc()`, `IsCash()`, `IsPet()`

---

## Cash Shop (Purchase Orchestration)

### Responsibility
Coordinates purchase flows: validates funds, determines compartment type from character job, creates flattened assets, and deducts currency.

### Invariants
- Insufficient funds result in `ErrInsufficientFunds` and an ERROR event with code `NOT_ENOUGH_CASH`
- Full compartment (assets count >= capacity) results in an ERROR event with code `INVENTORY_FULL`
- Compartment type is derived from character job type: Explorer, Cygnus, or Legend
- Character inventory capacity increase is capped at 96 slots; exceeding produces `ErrMaxSlots`

### Processors

#### Processor
- `Purchase`: Validates balance, determines compartment, creates flattened asset directly, deducts currency, emits PURCHASE event
- `PurchaseInventoryIncreaseByType`: Purchases inventory capacity increase by type (8 slots for 4000 currency)
- `PurchaseInventoryIncreaseByItem`: Purchases inventory capacity increase using a commodity item (4 slots)
- `PurchaseInventoryIncrease`: Core logic for inventory capacity increase with configurable cost and amount
