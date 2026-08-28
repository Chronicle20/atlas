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
- `Award(currency, amount)`: Returns a new Model with the specified currency increased by amount, saturating at `math.MaxUint32` instead of wrapping

### Processors

#### Processor
- `ByAccountIdProvider`: Provides wallet by account ID
- `GetByAccountId`: Retrieves wallet for an account
- `Create`/`CreateAndEmit`: Creates a new wallet with initial balances
- `Update`/`UpdateAndEmit`: Updates wallet balances
- `UpdateWithTransaction`/`UpdateAndEmitWithTransaction`: Updates wallet with transaction ID for saga coordination
- `UpdateWithTransactionSceneRefreshOwned`: Same as `UpdateWithTransaction`, but the emitted wallet UPDATED event carries a `SceneRefreshOwned` flag for callers (currently only the gift purchase flow) whose own status handler announces the scene refresh itself
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
- `currency` (uint32): Wallet bucket this asset was purchased with (1=credit/NX, 2=Maple Points, other=prepaid; 0 means a legacy row predating this column, or an asset never bought with currency)
- `quantity` (uint32): Item quantity
- `flag` (uint16): Item flags
- `petId` (uint32): Associated pet ID (0 if the asset is not a pet)
- `purchasedBy` (uint32): Character ID that purchased the item
- `expiration` (time.Time): Item expiration time (zero time means permanent)
- `createdAt` (time.Time): Timestamp of creation
- `giftFrom` (string): Sender's character name for a GIFT purchase; empty for every other asset
- `giftMessage` (string): Sender's message for a GIFT purchase; empty for every other asset
- `giftAcknowledged` (bool): Whether the gift list carrying this asset has been presented to the recipient (a LOAD_GIFT_SUCCESS announce has fired for it); this is not "the recipient clicked OK"
- `giftNoteSent` (bool): Whether the gift-forward note for this asset has already been sent to the gifter; independent of `giftAcknowledged`

#### ModelBuilder
- Builder pattern via `NewBuilder(compartmentId, templateId)` and `Clone(model)`
- Setters: `SetId`, `SetCompartmentId`, `SetCashId`, `SetTemplateId`, `SetCommodityId`, `SetCurrency`, `SetQuantity`, `SetFlag`, `SetPetId`, `SetPurchasedBy`, `SetExpiration`, `SetCreatedAt`, `SetGiftFrom`, `SetGiftMessage`, `SetGiftAcknowledged`, `SetGiftNoteSent`

### Invariants
- Cash ID is unique within a tenant; generated randomly on creation or accepted from external source
- `CreateWithCashId` uses find-or-create semantics: if an asset with the given cashId already exists, it returns the existing one (idempotent)
- Flag defaults to 0 on creation
- Expiration period defaults to 30 days when `commodityId` is 0 or the commodity lookup fails; otherwise uses the commodity's period (see Expiration Calculation)
- Purchase flows record `currency` as `effectivePurchaseCurrency(currency)` (see Cash Shop below), never the raw wire value, so a stored 0 unambiguously means "not purchased with currency" rather than "purchased with the default bucket"

### State Transitions
- `Create`: Generates a unique cashId, calculates expiration from commodity period and hourly configuration, creates the asset, emits CREATED status
- `CreateGift`: Same as `Create`, additionally carrying the sender's `giftFrom`/`giftMessage` onto the created row
- `CreateWithCashId`: Find-or-create by cashId. Used during compartment Accept to preserve the cashId from external systems
- `NextCashId`: Reserves a fresh, collision-checked cash serial without creating an asset; used when a pet row and its cash asset must share one serial, reserved before either row exists
- `UpdateQuantity`: Updates quantity in-place
- `AcknowledgeGifts`: Marks every asset in a compartment whose cashId is in a given list as `giftAcknowledged`
- `MarkGiftNoteSent`: Marks the asset in a compartment identified by cashId as `giftNoteSent`
- `Release`: Soft-deletes the asset
- `Delete`: Soft-deletes the asset
- `Expire`: Deletes the asset, emits EXPIRED status, optionally creates a replacement asset (via `Create`, with quantity 1 and no commodity/pet) with the given replaceItemId

### Processors

#### Processor
- `ByIdProvider`: Provides asset by ID
- `GetById`: Retrieves asset by ID
- `ByCompartmentIdProvider`: Provides all assets for a compartment
- `GetByCompartmentId`: Retrieves all assets for a compartment
- `Create`: Creates a new asset (generates cashId, calculates expiration), parameterized by compartmentId, templateId, commodityId, currency, quantity, petId, purchasedBy
- `CreateAndEmit`: Creates asset and emits Kafka event
- `CreateGift`: Creates a gift asset carrying `giftFrom`/`giftMessage`
- `NextCashId`: Reserves a collision-checked cash serial without creating an asset
- `CreateWithCashId`: Creates or finds asset by cashId (idempotent), same parameters plus cashId
- `CreateWithCashIdAndEmit`: Creates or finds asset by cashId and emits Kafka event
- `UpdateQuantity`: Updates asset quantity
- `AcknowledgeGifts`: Marks a set of assets in a compartment as presented
- `MarkGiftNoteSent`: Marks a single asset in a compartment as having had its gift-forward note sent
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
- `CashShop.Surprise.BoxTemplateIds` ([]uint32): Cash item template ids that open as a Cash Shop Surprise box for this tenant
- `CashShop.Coupons.RateLimit`: Coupon redemption rate limit (`Attempts` uint32, `WindowSeconds` uint32)

### Invariants
- Configurations are cached per tenant ID after first fetch
- Cache uses double-checked locking (RWMutex)
- If fetch fails, defaults to empty configuration
- `GetSurpriseBoxTemplateIds` falls back to the stock box id (`DefaultSurpriseBoxTemplateId`, 5222000) when the tenant configures no box ids
- `GetCouponRateLimit` falls back to `DefaultCouponAttempts` (10) / `DefaultCouponWindow` (1 hour) when the tenant configures a zero threshold or window; zero is treated as "unset", never as "zero allowed"

### Processors
- `GetTenantConfig`: Retrieves cached or fetches tenant configuration
- `GetHourlyExpirations`: Returns a map of templateId to hours from tenant config
- `GetSurpriseBoxTemplateIds`: Returns the cash item template ids that open as a Surprise box for the tenant
- `GetCouponRateLimit`: Returns the coupon redemption attempt budget and window for the tenant

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
- `PurchaseInventoryIncreaseByItem` resolves the target inventory type from the commodity's item ID and grants 4 slots; `PurchaseInventoryIncreaseByType` grants 4 slots for a fixed cost of 4000 currency
- Character inventory capacity increase is capped at 96 slots; exceeding produces `ErrMaxSlots`

### Processors

#### Processor
- `Purchase`/`PurchaseAndEmit`: Validates balance, determines compartment, creates a pet if applicable, creates flattened asset directly, deducts currency, emits PURCHASE event
- `PurchaseInventoryIncreaseByType`/`PurchaseInventoryIncreaseByTypeAndEmit`: Purchases inventory capacity increase by type (4 slots for 4000 currency)
- `PurchaseInventoryIncreaseByItem`/`PurchaseInventoryIncreaseByItemAndEmit`: Purchases inventory capacity increase using a commodity item (4 slots)
- `PurchaseInventoryIncrease`: Core logic for inventory capacity increase with configurable cost and amount
- `RebateAndEmit`: Refunds one locker asset's purchase price to the currency bucket it was purchased with and removes it
- `GiftAndEmit`: Charges the sender's credit/NX balance and creates the commodity in the recipient's locker
- `PurchasePackageAndEmit`: Resolves a cash package's member commodities, charges once for the package's own price, and creates one asset per member (buy-for-self or gift, discriminated by `recipientCharacterId`)
- `PurchaseRingAndEmit`: Charges the buyer once and mints a ring pair (one asset in the buyer's locker, one in the partner's), recorded via the Ring domain
- `PurchaseEquipSlotAndEmit`: Charges the buyer and queues a deferred equip-slot extension command (see `CompleteEquipSlotExtension`)
- `CompleteEquipSlotExtension`: Performs the deferred atlas-character equip-slot write for a completed equip-slot purchase and emits EQUIP_SLOT_INCREASED
- `AcknowledgeGiftsAndEmit`: Marks a set of locker assets across an account's compartments as gift-list presented
- `MarkGiftNoteSentAndEmit`: Marks a single locker asset's gift-forward note as sent

### Invariants (Gift, Package, Ring, Rebate, Equip Slot)
- `RebateAndEmit`, `GiftAndEmit`, `PurchasePackageAndEmit`, `PurchaseRingAndEmit`, and `PurchaseEquipSlotAndEmit` each claim their `transactionId` via the Ledger domain as the first step of their transaction; a redelivery of an already-claimed transaction id is treated as success-without-effect (no error, no event)
- A rebate resolves the target asset by cashId scoped to the requesting account's own compartments; an asset owned by another account is indistinguishable from a missing one and reports the same rejection
- A rebate refuses an asset with `commodityId == 0` (never bought with currency), an expired asset, or one whose commodity no longer resolves
- A gift purchase is always charged to the sender's credit/NX bucket; the target compartment and capacity check are the recipient's, not the sender's
- A package purchase charges once for the package commodity's own price (never the sum of member commodity prices) and checks capacity against the full member count before charging
- A ring purchase mints both halves from the same resolved commodity item id; both compartments (buyer and partner) must have room before either asset is created, and both halves are recorded as one pair in a single insert so a partial pair cannot persist
- An equip-slot purchase creates no locker asset; it charges the buyer and extends the character's fixed pendant2 equip slot by the commodity's period (in days) via atlas-character, deferred behind the outbox so the cross-service write only happens after the wallet debit and purchase record have durably committed
- Every purchase-path write in this domain (Purchase, Gift, Package, Ring, Equip Slot) also records the purchase (and, for gifts/packages, every member serial) against the buyer's/sender's account via the Purchase Record domain

---

## Purchase Record

### Responsibility
Tracks the durable, non-soft-deletable history of which commodity serial numbers an account has ever purchased, independent of whether the resulting asset still exists in the locker.

### Core Models
- One row per `(tenantId, accountId, serialNumber)`: `count` (number of purchases), `firstAt`, `lastAt`

### Invariants
- A rebate or withdrawal does not remove a purchase record; "purchased" is a historical fact
- `Record` upserts atomically on the `(tenant_id, account_id, serial_number)` unique index, so two concurrent purchases of the same serial cannot land as separate rows
- On boot, `Backfill` seeds records from existing `cash_assets` history (grouped by tenant/account/commodity) exactly once; it is a no-op once any record exists, and assets already hard-deleted before the backfill ran are unrecoverable

### Processors

#### Processor
- `Record`: Upserts one purchase on a caller-supplied database handle (called inside the purchasing transaction)
- `Get`: Returns the purchase count for an account/serial number pair (0 if never purchased)

---

## Ledger

### Responsibility
Claims transaction-scoped idempotency for cash shop commands (gift, package, ring, equip-slot, and locker-rebate purchases) using the shared `atlas-database` idempotency table.

### Invariants
- `Claim` must be the first statement inside the claiming command's transaction, so a Kafka redelivery aborts before any state changes
- The claim keys on the bare transaction id, not `(transaction id, command type)`: a replay of the same transaction id under a different command type is still rejected as the same redelivery
- Claiming the zero transaction id (`uuid.Nil`) is refused with an error, since `uuid.Nil` is documented elsewhere as "no correlation" and must never become a shared uniqueness claim
- `ErrAlreadyProcessed` signals success-without-effect (a redelivery), not a failure to report to the client

### Processors
- `Claim(ctx, tx, transactionId, commandType, characterId)`: Writes the uniqueness claim, returning `ErrAlreadyProcessed` when the transaction id was already claimed

---

## Ring

### Responsibility
Records couple/friendship ring pairs purchased through the Cash Shop domain's `PurchaseRingAndEmit`, and serves a read-only query surface over them.

### Core Models

#### Type
- `TypeCouple` (`"COUPLE"`), `TypeFriendship` (`"FRIENDSHIP"`)

#### State
- `StateActive`, `StateBroken`, `StateExpired`: records what happened to a pair half without deleting its history

#### Model (one half of a pair)
- `id` (uuid.UUID), `pairId` (uuid.UUID, shared by both halves), `characterId`, `partnerCharacterId`, `assetId`, `itemTemplateId`, `ringType` (Type), `state` (State), `createdAt`
- `cashId` (int64): This half's own asset's cash id, captured at purchase time (survives the asset leaving the locker/being equipped, unlike `assetId`)
- `partnerCashId` (int64): The sibling half's cash id, resolved at read time; zero if unresolvable
- `partnerName` (string): The partner character's name, resolved at read time; empty if unresolvable

#### Half
- Purchase-time input to `CreatePair`: `CharacterId`, `AssetId`, `CashId`, `ItemTemplateId`

### Invariants
- Both halves of a pair are inserted in a single multi-row `INSERT`, so a partial pair cannot be persisted
- `enrich` (read-time decoration of `cashId`/`partnerCashId`/`partnerName`) fails soft to the zero value on every lookup; a lookup failure never turns into an error for a caller that only wants the pair rows
- `cashId`/`partnerCashId` prefer the value persisted on the row at purchase time; falling back to an `assetId` lookup is reserved for rows written before the `cashId` column existed, since a locker asset id stops resolving once the ring is equipped

### Processors

#### Processor
- `CreatePair`: Inserts both halves of a pair on a caller-supplied transaction handle
- `GetByCharacterId`/`GetByCharacterIdPaged`: Returns every ring half a character holds, enriched with `cashId`/`partnerCashId`/`partnerName`
- `GetById`: Returns a single ring half by ID, enriched, scoped to the calling tenant

---

## Coupon

### Responsibility
Redeems a one-time-per-account coupon code for a bundle of currency and/or cash-item rewards, and exposes admin CRUD over coupon definitions. There is no REST redeem route by design; redemption is triggered only by the `REQUEST_COUPON_REDEMPTION` command.

### Core Models

#### Model
- `id` (uuid.UUID), `batchId` (uuid.UUID, nil if not bulk-generated), `code` (string, stored normalized: trimmed + uppercased), `description`, `active` (bool), `startsAt`/`expiresAt` (*time.Time, nil = unbounded), `maxUses` (*uint32, nil = unlimited), `redemptionCount` (uint32), `rewards` (reward.Rewards), `createdAt`, `updatedAt`

#### RedemptionError
- Carries a client-facing key (one of `ErrorKeyInvalidCode`, `ErrorKeyNotRegistered`, `ErrorKeyExpired`, `ErrorKeyAlreadyUsed`, `ErrorKeyUsageLimit`, `ErrorKeyInventoryFull`, `ErrorKeyUnknown`) and a detail string

### Invariants
- `RedeemableAt(now)` runs the validation ladder answerable from the coupon row alone (active, window, usage-limit fast path), first failure wins, in that exact order
- The one-time-per-account rule is enforced at the database level by a unique index on `(tenant_id, coupon_id, account_id)` in `coupon_redemptions`; the ladder check is only a friendly-error fast path
- A coupon must grant at least one reward; each reward must validate for its type
- `expiresAt` must be after `startsAt` when both are set; `maxUses` cannot be set below `redemptionCount`
- Coupon codes are secrets: never logged or used as a metric label; only their length is ever recorded
- Redemption attempts are rate-limited per account via the Limiter (Redis-backed, fails open on a Redis outage); a rate-limited attempt is reported to the player as `ErrorKeyInvalidCode` (not a distinct "rate limited" result) so an attacker cannot distinguish a real code from a blocked attempt
- Deleting a coupon that has redemptions is refused (`ErrHasRedemptions`)
- A coupon created inactive stays inactive: the entity carries no GORM column default for `active`

### State Transitions
- `RedeemAndEmit`: Resolves the redeeming account, checks the rate limiter, runs the redemption ladder inside one database transaction (code exists, redeemable, no prior redemption, locker capacity pre-flight, atomic use reservation, redemption row insert, reward grants), and emits `COUPON_REDEEMED` on success (via the outbox) or `COUPON_FAILED` on rejection (via the direct producer path)
- Reservation (`reserveUse`) is one atomic conditional `UPDATE` on `redemption_count`, so two concurrent redemptions of a `maxUses = 1` coupon cannot both succeed

### Processors

#### Processor
- `RedeemAndEmit`: Runs one redemption attempt end to end
- `GetById`/`GetByCode`: Retrieves a coupon
- `AllProvider`: Pages coupons with optional filters (code, active, batchId, expiresBefore/After)
- `ValidateRewards`: Confirms every `CASH_ITEM` reward's serial number resolves to a real commodity
- `Create`/`Update`/`Delete`: Admin CRUD

---

## Coupon Reward

### Responsibility
Represents the discriminated reward bundle a coupon grants.

### Core Models

#### Reward
- `Type`: `TypeCurrency` (`"CURRENCY"`) or `TypeCashItem` (`"CASH_ITEM"`)
- Currency reward: `currency` (uint32, following wallet's 1=credit/2=points/other=prepaid convention), `amount`
- Cash item reward: `serialNumber` (commodity serial), `quantity`

#### Rewards
- `[]Reward`, persisted as one jsonb document; `CashItemCount()` returns the number of locker slots the bundle needs

### Invariants
- A currency reward's amount must be positive; a cash item reward's serial number and quantity must both be non-zero
- A bundle must contain at least one reward

### Granters (reward application)
- `currencyGranter`: Credits the redeeming account's wallet via `wallet.Award`
- `cashItemGranter`: Re-checks locker capacity inside the transaction, resolves the commodity, and creates a flattened asset; refuses to grant a pet-classified commodity (a coupon cannot create the accompanying pet row)

---

## Coupon Batch

### Responsibility
Generates a bulk set of single-use coupons sharing a common reward bundle and description, in one all-or-nothing operation.

### Core Models

#### Model
- `id`, `description`, `requestedCount`, `generatedCount`, `redeemedCount` (computed at read time from `coupon_redemptions`, never stored), `createdAt`

### Invariants
- `generatedCount` always equals `requestedCount` on a successful batch: a code collision retries (up to a bound) rather than being skipped, so a short batch would indicate a bug
- Every generated coupon gets `maxUses = 1`
- The batch row and every generated coupon are inserted in one transaction; the whole batch rolls back if any code cannot be produced

### Processors

#### Processor
- `GetById`/`AllProvider`: Retrieves batches, decorated with `redeemedCount`
- `Generate`: Creates the batch row and its coupons, returning the batch and the plaintext generated codes (returned only on this call; never re-served by a later read)

---

## Coupon Redemption

### Responsibility
Read-only audit trail of successful coupon redemptions. No route creates, edits, or deletes a redemption; redemptions are written only by `Coupon.RedeemAndEmit`.

### Core Models

#### Model
- `id`, `couponId`, `accountId`, `characterId`, `transactionId`, `rewardsGranted` (reward.Rewards snapshot at redemption time, never re-derived from the coupon's current bundle), `redeemedAt`

### Invariants
- Uniqueness on `(tenant_id, coupon_id, account_id)` is the database-level one-time-per-account enforcement
- `rewardsGranted` is a snapshot: a later edit to the coupon's bundle never rewrites redemption history

### Processors

#### Processor
- `ByCouponId`/`ByAccountId`: Paged listing of redemptions
- `CountByBatchId`: Counts redemption rows across every coupon in a batch (a row count, not a sum of `redemption_count`, since `releaseUse` can decrement that column without deleting a row)

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
