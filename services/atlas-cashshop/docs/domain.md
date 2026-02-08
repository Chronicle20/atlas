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
