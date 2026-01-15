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

### Processors

#### Processor
- `ByAccountIdProvider`: Provides wallet by account ID
- `GetByAccountId`: Retrieves wallet for an account
- `Create`: Creates a new wallet with initial balances
- `Update`: Updates wallet balances
- `UpdateWithTransaction`: Updates wallet with transaction ID for saga coordination
- `AdjustCurrency`: Adjusts a specific currency type by amount
- `AdjustCurrencyWithTransaction`: Adjusts currency with transaction ID
- `Delete`: Deletes a wallet

---

## Wishlist

### Responsibility
Manages character wishlists for cash shop items.

### Core Models

#### Model
- `id` (uuid.UUID): Unique identifier
- `characterId` (uint32): Owner character
- `serialNumber` (uint32): Serial number of the wished item

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
- `compartments` (map[CompartmentType]Compartment): Compartments by type

### Invariants
- Each account has one inventory
- Inventory contains three compartments: Explorer, Cygnus, Legend

### Processors

#### Processor
- `ByAccountIdProvider`: Provides inventory by account ID
- `GetByAccountId`: Retrieves inventory for an account
- `Create`: Creates inventory with default compartments
- `Delete`: Deletes inventory and all compartments

---

## Compartment

### Responsibility
Represents a section of cash inventory for a specific character type.

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
- `assets` ([]Asset): Assets in the compartment

### Invariants
- Default capacity is 55
- Assets count cannot exceed capacity

### Processors

#### Processor
- `GetById`: Retrieves compartment by ID
- `ByIdProvider`: Provides compartment by ID
- `GetByAccountIdAndType`: Retrieves compartment by account and type
- `ByAccountIdAndTypeProvider`: Provides compartment by account and type
- `AllByAccountIdProvider`: Provides all compartments for an account
- `GetByAccountId`: Retrieves all compartments for an account
- `Create`: Creates a new compartment
- `UpdateCapacity`: Updates compartment capacity
- `Delete`: Deletes a compartment
- `DeleteAllByAccountId`: Deletes all compartments for an account
- `Accept`: Accepts an asset into a compartment
- `Release`: Releases an asset from a compartment

---

## Asset

### Responsibility
Represents a cash shop item stored in a compartment.

### Core Models

#### Model
- `id` (uuid.UUID): Unique identifier
- `compartmentId` (uuid.UUID): Parent compartment
- `item` (Item): Associated item

### Invariants
- Each asset belongs to exactly one compartment
- Asset references an item by ID

### Processors

#### Processor
- `ByIdProvider`: Provides asset by ID
- `GetById`: Retrieves asset by ID
- `ByCompartmentIdProvider`: Provides all assets for a compartment
- `GetByCompartmentId`: Retrieves all assets for a compartment
- `Create`: Creates a new asset
- `Release`: Removes an asset from a compartment

---

## Item

### Responsibility
Represents a cash shop item with template and ownership information.

### Core Models

#### Model
- `id` (uint32): Unique identifier
- `cashId` (int64): Cash item identifier
- `templateId` (uint32): Item template ID
- `quantity` (uint32): Item quantity
- `flag` (uint16): Item flags
- `purchasedBy` (uint32): Character ID that purchased the item
- `expiration` (time.Time): Item expiration time

### Invariants
- Cash ID is generated on creation

### Processors

#### Processor
- `ByIdProvider`: Provides item by ID
- `GetById`: Retrieves item by ID
- `Create`: Creates a new item
