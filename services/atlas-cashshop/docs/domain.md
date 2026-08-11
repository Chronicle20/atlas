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
- `AdjustCurrency`/`AdjustCurrencyWithTransaction` reject an adjustment whose deduction would exceed the current balance for the given currency type, or whose currency type is not 1/2/3, without changing state
- A failed transactional adjustment (non-nil transaction ID) emits a wallet ERROR status event so the waiting saga step fails fast instead of timing out; a non-transactional adjustment (nil transaction ID) does not emit on failure

### State Transitions
- `Purchase(currency, amount)`: Returns a new Model with the specified currency reduced by amount. Does not validate balance; caller is responsible for checking before calling.

### Processors

#### Processor
- `ByAccountIdProvider`: Provides wallet by account ID
- `GetByAccountId`: Retrieves wallet for an account
- `Create`/`CreateAndEmit`: Creates a new wallet with initial balances
- `Update`/`UpdateAndEmit`: Updates wallet balances
- `UpdateWithTransaction`/`UpdateAndEmitWithTransaction`: Updates wallet with transaction ID for saga coordination
- `AdjustCurrency`: Adjusts a specific currency type by amount, validates sufficient balance (delegates to `AdjustCurrencyWithTransaction` with a nil transaction ID)
- `AdjustCurrencyWithTransaction`: Adjusts currency with transaction ID, validates sufficient balance
- `Delete`/`DeleteAndEmit`: Deletes a wallet
- `EmitAdjustFailure`: Emits a wallet ERROR status event for a failed transactional currency adjustment
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
- `ByCharacterIdPagedProvider`: Provides one page of a character's wishlist
- `Add`/`AddAndEmit`: Adds an item to the wishlist
- `Delete`/`DeleteAndEmit`: Removes an item from the wishlist
- `DeleteAll`/`DeleteAllAndEmit`: Clears all items from a character's wishlist

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
- `Create`/`CreateAndEmit`: Creates inventory with three default compartments (Explorer, Cygnus, Legend) at default capacity
- `Delete`/`DeleteAndEmit`: Deletes all compartments for the account
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
- `FindById`/`FindByTemplateId`: Lookup helpers over `assets`

#### DefaultCapacity
- Constant value: 55

### Invariants
- Default capacity is 55
- Assets count cannot exceed capacity (enforced during purchase)
- Assets are lazily decorated onto the model when fetched via `DecorateAssets`
- `Accept` defaults `quantity` to 1 when 0 is supplied, and `purchasedBy` to the account ID when 0 is supplied
- Compartment error events use one of the codes: `UNKNOWN_ERROR` (compartment lookup failure), `ASSET_CREATION_FAILED` (Accept could not create the asset), `ITEM_NOT_FOUND` (Release target asset not found in the compartment)

### State Transitions
- `Accept`: Creates a new flattened asset in the compartment using `CreateWithCashId` (idempotent by cashId; petId is always 0). Emits ACCEPTED on success, ERROR on failure.
- `Release`: Validates the asset exists in the compartment via `FindById`, then soft-deletes it. Emits RELEASED on success, ERROR on failure.

### Processors

#### Processor
- `GetById`: Retrieves compartment by ID (with asset decoration)
- `ByIdProvider`: Provides compartment by ID
- `GetByAccountIdAndType`: Retrieves compartment by account and type
- `ByAccountIdAndTypeProvider`: Provides compartment by account and type
- `AllByAccountIdProvider`: Provides all compartments for an account
- `GetByAccountId`: Retrieves all compartments for an account
- `AllByAccountIdPagedProvider`: Provides one page of an account's compartments
- `Create`/`CreateAndEmit`: Creates a new compartment
- `UpdateCapacity`/`UpdateCapacityAndEmit`: Updates compartment capacity
- `Delete`/`DeleteAndEmit`: Deletes a compartment
- `DeleteAllByAccountId`/`DeleteAllByAccountIdAndEmit`: Deletes all compartments for an account
- `Accept`/`AcceptAndEmit`: Accepts an asset into a compartment (creates flattened asset with preserved cashId)
- `Release`/`ReleaseAndEmit`: Releases an asset from a compartment (validates existence, then deletes)
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
- `petId` (uint32): Associated pet ID (0 if the asset is not a pet)
- `purchasedBy` (uint32): Character ID that purchased the item
- `expiration` (time.Time): Item expiration time (zero time means permanent)
- `createdAt` (time.Time): Timestamp of creation

#### ModelBuilder
- Builder pattern via `NewBuilder(compartmentId, templateId)` and `Clone(model)`
- Setters: `SetId`, `SetCompartmentId`, `SetCashId`, `SetTemplateId`, `SetCommodityId`, `SetQuantity`, `SetFlag`, `SetPetId`, `SetPurchasedBy`, `SetExpiration`, `SetCreatedAt`

### Invariants
- Cash ID is unique within a tenant; generated randomly on creation or accepted from external source
- `CreateWithCashId` uses find-or-create semantics: if an asset with the given cashId already exists, it returns the existing one (idempotent)
- Flag defaults to 0 on creation
- Expiration period defaults to 30 days when `commodityId` is 0 or the commodity lookup fails; otherwise uses the commodity's period (see Expiration Calculation)

### State Transitions
- `Create`: Generates a unique cashId, calculates expiration from commodity period and hourly configuration, creates the asset, emits CREATED status
- `CreateWithCashId`: Find-or-create by cashId. Used during compartment Accept to preserve the cashId from external systems
- `UpdateQuantity`: Updates quantity in-place
- `Release`: Soft-deletes the asset
- `Delete`: Soft-deletes the asset
- `Expire`: Deletes the asset, emits EXPIRED status, optionally creates a replacement asset (via `Create`, with quantity 1 and no commodity/pet) with the given replaceItemId

### Processors

#### Processor
- `ByIdProvider`: Provides asset by ID
- `GetById`: Retrieves asset by ID
- `ByCompartmentIdProvider`: Provides all assets for a compartment
- `GetByCompartmentId`: Retrieves all assets for a compartment
- `Create`: Creates a new asset (generates cashId, calculates expiration), parameterized by compartmentId, templateId, commodityId, quantity, petId, purchasedBy
- `CreateAndEmit`: Creates asset and emits Kafka event
- `CreateWithCashId`: Creates or finds asset by cashId (idempotent), same parameters plus cashId
- `CreateWithCashIdAndEmit`: Creates or finds asset by cashId and emits Kafka event
- `UpdateQuantity`: Updates asset quantity
- `Delete`: Soft-deletes an asset
- `DeleteAndEmit`: Deletes asset (puts no event)
- `Release`: Soft-deletes an asset (alias for delete with logging)
- `ReleaseAndEmit`: Releases asset (puts no event)
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
- Additional fields: name, gender, skinColor, face, hair, level, strength, dexterity, intelligence, luck, hp, maxHp, mp, maxMp, hpMpUsed, ap, sp, experience, fame, gachaponExperience, spawnPoint, gm, meso, x, y, stance

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
- `PetReferenceData`: CashId, ownerId, flag, purchaseBy, name, level, closeness, fullness, expiration, slot, attribute, skill, remainingLife, attribute2

### Invariants
- Quantity defaults to 1 unless the reference data implements `HasQuantity`
- Type checks via `IsEquipable()`, `IsConsumable()`, `IsSetup()`, `IsEtc()`, `IsCash()`, `IsPet()`

---

## Cash Shop (Purchase Orchestration)

### Responsibility
Coordinates purchase flows: validates funds, determines compartment type from character job, creates a pet when the purchased item is pet-classified, creates flattened assets, and deducts currency.

### Invariants
- Insufficient funds result in `ErrInsufficientFunds` and an ERROR event with code `NOT_ENOUGH_CASH`
- Full compartment (assets count >= capacity) results in an ERROR event with code `INVENTORY_FULL`; no state is changed
- Any other failure (commodity lookup, character lookup, wallet lookup, pet creation, asset creation) results in an ERROR event with code `UNKNOWN_ERROR`
- Compartment type is derived from character job type: Explorer, Cygnus, or Legend
- When the purchased item's classification is Pet, a pet is created via the Pet (REST Client) processor before the asset is created; the pet's name is resolved from the Pet Data (REST Client) processor, defaulting to `"Pet"` if that lookup fails; the created pet's ID is stored on the asset's `petId`
- `PurchaseInventoryIncreaseByItem` resolves the target inventory type from the commodity's item ID and grants 4 slots; `PurchaseInventoryIncreaseByType` grants 8 slots for a fixed cost of 4000 currency
- Character inventory capacity increase is capped at 96 slots; exceeding produces `ErrMaxSlots`

### Processors

#### Processor
- `Purchase`/`PurchaseAndEmit`: Validates balance, determines compartment, creates a pet if applicable, creates flattened asset directly, deducts currency, emits PURCHASE event
- `PurchaseInventoryIncreaseByType`/`PurchaseInventoryIncreaseByTypeAndEmit`: Purchases inventory capacity increase by type (8 slots for 4000 currency)
- `PurchaseInventoryIncreaseByItem`/`PurchaseInventoryIncreaseByItemAndEmit`: Purchases inventory capacity increase using a commodity item (4 slots)
- `PurchaseInventoryIncrease`: Core logic for inventory capacity increase with configurable cost and amount

---

## Pet Data (REST Client)

### Responsibility
Fetches pet template data from the external atlas-data service. Used during purchase flows to resolve a pet's display name.

### Core Models

#### Model
- `id` (uint32): Template ID
- `name` (string): Pet template name

### Processors

#### Processor
- `GetById`: Fetches pet template data by template ID from atlas-data via REST

---

## Pet (REST Client)

### Responsibility
Creates pets via an external pet service. Used during purchase flows when the purchased item is pet-classified.

### Core Models

#### Model
- `id` (uint32): Pet ID
- `templateId` (uint32): Item template ID
- `name` (string): Pet name
- `ownerId` (uint32): Owning character ID

### Processors

#### Processor
- `Create`: Creates a pet for a character via REST, given owner character ID, template ID, and name

---

## Surprise (task-207)

### Responsibility
Orchestrates opening a Cash Shop Surprise box: resolves and validates the box asset, checks locker capacity, rolls a reward from the box's configured reward pool (via the Reward Pool REST client), resolves the reward's commodity, then commits a single database transaction that inserts the openings ledger row, consumes (or decrements) the box, and grants the reward asset.

### Open Sequence
1. **Resolve** — look up the character's job-typed compartment (Explorer/Cygnus/Legend, same mapping as Purchase), then find the box asset within it by `cashId`. Ownership is enforced structurally: the compartment is looked up by `accountId`, so an asset belonging to another account is simply absent from the scanned set — `BOX_NOT_FOUND` covers both "no such box" and "not yours." The resolved template ID must be one of the tenant's configured Surprise-box template IDs (`NOT_A_SURPRISE_BOX` otherwise).
2. **Capacity** — `HasRoomForSwap(assetCount, capacity, boxQuantity)` decides whether the locker can absorb the swap. It checks the PEAK slot count, not the net: when the box's stack quantity is > 1, the box row survives the decrement so the reward needs its own free slot (`assetCount < capacity`); when the stack is exactly 1, the box row is released so the grant is slot-neutral and an exactly-full locker is fine (`assetCount <= capacity`). Fails closed as `LOCKER_FULL`.
3. **Roll** — mutates nothing. `rewardpool.Processor.SelectReward(boxTemplateId)` calls atlas-reward-pools and classifies `404` as `POOL_MISSING` and `409` as `POOL_EMPTY` (see Reward Pool (REST Client) below). This step's statelessness is the reason the whole flow needs no saga: ordering (roll, *then* consume+grant) makes partial application structurally impossible — a failed roll can never leave a half-consumed box.
4. **Resolve commodity** — the rolled reward's `commodityId` must be non-zero (`COMMODITY_MISSING` otherwise: a zero commodity carries no price/period basis to derive an expiration from), then the commodity is fetched from atlas-data via the existing Commodity (REST Client).
5. **Commit** — the only transactional step. `opening.Insert` runs FIRST so a duplicate `transactionId` (a Kafka redelivery) aborts before anything is consumed or granted. Then the box is either decremented (`UpdateQuantity`) or released (`Release`, when quantity reaches 0), and the reward asset is created (`asset.Create`) in the same transaction. All writes rebuild their processors against the transaction handle rather than `p.db`, so nothing escapes the transaction (mirrors the `cashshop.Purchase` precedent).

Rejections found during steps 1–4 (nothing mutated yet) emit `SURPRISE_FAILED` directly on the Kafka producer path and are swallowed (the command handler returns nil) — retrying the identical command would fail identically, and the client has already been told. A genuine failure inside the transaction (step 5) is propagated as a real error with no event fired.

### Openings Ledger (Idempotency)
- One row per successfully committed open, table `cash_surprise_openings`, primary key `(tenant_id, transaction_id)`.
- `transactionId` is minted by atlas-channel per click and is the idempotency key: a Kafka redelivery replays the same id and is rejected by the primary-key constraint; a genuine second click gets a new id.
- Detection is by constraint violation, not a read-then-write check — a SELECT-then-INSERT has a race window where two concurrent redeliveries both observe "not present" and both insert. `isDuplicateKeyError` recognizes the violation under both drivers this service runs against: Postgres SQLSTATE `23505` in production, sqlite extended codes `1555`/`2067` in tests.
- A redelivery of an already-committed transaction (`ErrAlreadyOpened`) is treated as success-without-effect: no further state changes, no event — the original open already told the client.

### Capacity Rule
`HasRoomForSwap(assetCount, capacity, boxQuantity uint32) bool` — see Open Sequence step 2. An over-capacity locker (`assetCount > capacity`, possible through data drift) is rejected in both branches: the neutral case (stack quantity 1) permits equality, not excess.

### Invariants
- `OPEN_SURPRISE` command carries `transactionId` (idempotency key), `accountId`, and `cashId` (the box's cash locker identity) — the server resolves and re-validates both the box and its ownership; the edge does not own the locker.
- Reward pool for a box template id is looked up in atlas-reward-pools by pool id == box template id (`cash-surprise` kind pools).
- A reward pool configured to award another Surprise box (recursive box) is honored by configuration, not blocked in code — logged loudly so an operator notices, since it produces an infinite box.
- Closed set of `SURPRISE_FAILED` reasons: `BOX_NOT_FOUND`, `NOT_A_SURPRISE_BOX`, `LOCKER_FULL`, `POOL_EMPTY`, `POOL_MISSING`, `COMMODITY_MISSING`, `INTERNAL`. The reason never reaches the client — the FAILED arm of the client packet has an empty body and no error-code field; the reason exists for logs/operators only.

### Dependency: Reward Pool (REST Client)

#### Responsibility
Selects a reward from an atlas-reward-pools `cash-surprise`-kind pool for a given box template ID.

#### Core Models
- `Model`: `itemId` (uint32), `quantity` (uint32), `commodityId` (uint32) — the rolled reward.

#### Invariants
- `ErrPoolMissing`: no `cash-surprise` pool configured for the box template id (atlas-reward-pools responds 404).
- `ErrPoolEmpty`: the pool exists but has no eligible entries (atlas-reward-pools responds 409, `requests.ErrConflict`).
- Any other error (e.g. transport failure) is returned unclassified rather than misreported as a configuration fault, so an infrastructure outage isn't logged as if an operator misconfigured a pool.

#### Processors
- `SelectReward(boxTemplateId)`: POSTs to atlas-reward-pools' gachapon-rewards select endpoint for the pool whose id equals `boxTemplateId`, decodes `itemId`/`quantity`/`commodityId`.

### Kafka
- Command: `OPEN_SURPRISE` on `COMMAND_TOPIC_CASH_SHOP` (see `services/atlas-cashshop/docs/kafka.md`).
- Status events on `EVENT_TOPIC_CASH_SHOP_STATUS`: `SURPRISE_OPENED` (success) and `SURPRISE_FAILED` (rejection). `SURPRISE_OPENED` carries `boxRemaining` (the box's quantity AFTER the decrement — the client removes the locker row when it reaches 0) and the granted `rewardAssetId`/`rewardTemplateId`/`rewardCount` (`rewardCount` comes from the commodity catalog, not the pool entry).
