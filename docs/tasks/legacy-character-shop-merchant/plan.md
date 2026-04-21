# Character Shop & Hired Merchant — Implementation Plan

Last Updated: 2026-02-25

## Executive Summary

Implement a new `atlas-merchant` service that supports both **Character Shops** and **Hired Merchants** — player-operated storefronts within Free Market rooms. Both shop types share 90%+ of their domain logic (listings, visitors, purchases, bundles, real-time updates) and are differentiated primarily by lifecycle rules (online-required vs persistent, direct-delivery vs Frederick storage). A single service with a shop-type discriminator avoids code duplication while cleanly separating behavioral differences via strategy patterns.

The service introduces:
- A formal **state machine** (Draft → Open ↔ Maintenance → Closed)
- A **bundle-based listing model** for stackable items
- **Real-time visitor synchronization** via Kafka events
- **Concurrent purchase safety** via optimistic locking
- **Frederick integration** for hired merchant item/meso recovery
- **Proximity-based placement** validation in Free Market maps
- **24-hour expiration** with reaper for hired merchants
- **100-day Frederick cleanup** reaper

---

## Current State Analysis

### Existing Infrastructure
| Component | Current State | Relevance |
|-----------|--------------|-----------|
| `atlas-npc-shops` | Fully operational NPC shop service | **Primary reference template** — shares patterns for shop browsing, transactions, inventory integration |
| `atlas-inventory` | Compartment-based item management | Source/destination for listed items; Kafka command integration |
| `atlas-storage` | Character storage service | **Not used** — Frederick is merchant-specific, handled within atlas-merchant |
| `atlas-character` | Character session/meso tracking | Meso validation, online status for character shop lifecycle |
| `atlas-cashshop` | Cash item management | Permit verification (Store Permit 514, Hired Merchant 503) |
| `atlas-maps` | Map instance management | Placement validation, proximity rules, Free Market identification |
| `atlas-drops` | Ground item reservations | Reference for distributed lock + atomic counter patterns |
| `atlas-redis` | TenantRegistry, TTLRegistry, locks | Session state, active shops, visitor tracking, placement registry |
| `atlas-database` | PostgreSQL + auto tenant filtering | Shop persistence, listing storage, Frederick tracking |
| `atlas-kafka` | At-least-once message delivery | Command/event bus for all operations |

### Gap Analysis
- **No player-to-player shop service exists** — only NPC shops
- **No bundle-based listing model** — NPC shops use fixed commodity definitions
- **No player-operated state machine** — NPC shops are always open
- **No Frederick notification/reaper** — storage service has no TTL management
- **No proximity-based placement** — maps track positions but no shop collision detection

---

## Proposed Future State

### Service: `atlas-merchant`

A new Go microservice following established Atlas DDD patterns, managing the complete lifecycle of player-operated shops.

### Domain Model

```
Shop (Aggregate Root)
├── Id (UUID)
├── TenantId (UUID)
├── CharacterId (uint32) — owner
├── ShopType (enum: CharacterShop | HiredMerchant)
├── State (enum: Draft | Open | Maintenance | Closed)
├── Title (string)
├── MapId (uint32)
├── Position (x, y int16)
├── PermitItemId (uint32) — cash item template that authorized this shop
├── CreatedAt (time)
├── ExpiresAt (*time) — nil for character shops, +24h for hired merchants
├── ClosedAt (*time)
├── CloseReason (enum: SoldOut | ManualClose | Disconnect | Expired | ServerRestart | Empty)
├── Listings []Listing — ordered, max 16
├── Visitors []Visitor — max 3
├── Messages []Message — chat/visit history
└── MesoBalance (uint32) — accumulated sale proceeds (hired merchant)

Listing (Value Object)
├── Id (UUID)
├── ItemId (uint32) — inventory item template ID
├── ItemType (inventory.Type) — equip/use/setup/etc
├── Quantity (uint16) — total remaining items in this listing
├── BundleSize (uint16) — items per bundle
├── BundlesRemaining (uint16)
├── PricePerBundle (uint32)
├── ItemSnapshot (json) — full item attribute copy at list time (stats, scrolls, flags, etc.)
└── ListedAt (time)

Visitor (Value Object)
├── CharacterId (uint32)
└── EnteredAt (time)

Message (Value Object)
├── CharacterId (uint32)
├── Content (string)
└── SentAt (time)
```

### State Machine

```
         ┌─────────┐
         │  Draft  │
         └────┬────┘
              │ addListing + open()
              ▼
         ┌─────────┐  enterMaintenance()  ┌──────────────┐
         │  Open   │ ──────────────────▶ │ Maintenance  │
         └────┬────┘ ◀────────────────── └──────┬───────┘
              │        exitMaintenance()         │
              │                                  │ exitMaintenance(0 listings)
              │ soldOut/close/disconnect/expire   │ close()
              ▼                                  ▼
         ┌─────────┐
         │ Closed  │ (terminal)
         └─────────┘
```

### Storage Architecture

| Data | Store | Rationale |
|------|-------|-----------|
| Shop entity (persistent) | PostgreSQL | Survives restarts; hired merchants persist across sessions |
| Listings | PostgreSQL | Part of shop aggregate; transactional with purchase operations |
| Messages/history | PostgreSQL | Audit trail, survives restart |
| Active shop registry | Redis TenantRegistry | Fast lookup: characterId → shopId for online checks |
| Visitor sessions | Redis TenantRegistry | Ephemeral; auto-cleanup on disconnect |
| Map placement index | Redis SET | Fast proximity checks; `atlas:merchant-map:{tenantId}:{mapId}` |
| Frederick storage | PostgreSQL | Hired merchant unsold items + mesos; TTL-based with reaper; notification scheduling; entirely within atlas-merchant (not atlas-storage) |

### Kafka Topics

| Topic | Direction | Purpose |
|-------|-----------|---------|
| `COMMAND_TOPIC_MERCHANT` | Inbound | Place, open, close, maintain, list, delist, purchase, enter, exit |
| `EVENT_TOPIC_MERCHANT_STATUS` | Outbound | Shop state changes (opened, closed, maintenance), visitor events |
| `EVENT_TOPIC_MERCHANT_LISTING` | Outbound | Listing added, removed, purchased (broadcast to visitors) |
| `COMMAND_TOPIC_INVENTORY_COMPARTMENT` | Outbound | Item create/destroy in character inventory |
| `EVENT_TOPIC_INVENTORY_COMPARTMENT_STATUS` | Inbound | Inventory operation confirmations |
| `EVENT_TOPIC_MERCHANT_FREDERICK` | Outbound | Frederick notifications (items/mesos available for retrieval) |
| `EVENT_TOPIC_CHARACTER_STATUS` | Inbound | Disconnect detection (character shop auto-close) |

### REST Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/merchants` | List active shops (by map, by owner) |
| GET | `/merchants/{shopId}` | Shop details + listings |
| GET | `/merchants/{shopId}/relationships/listings` | Listing details |
| GET | `/merchants/{shopId}/relationships/visitors` | Current visitors |
| GET | `/characters/{characterId}/merchants` | Character's active shops |
| GET | `/characters/{characterId}/frederick` | Frederick stored items/mesos |

---

## Implementation Phases

### Phase 1: Core Domain & Persistence (Foundation)
Establish the service skeleton, domain models, database entities, and basic CRUD. No Kafka integration yet — just the domain layer that everything builds on.

**Deliverables**: Service compiles, migrations run, models/entities/builders work, basic REST endpoints serve data.

### Phase 2: Shop Lifecycle State Machine
Implement the state machine (Draft → Open ↔ Maintenance → Closed) with all transition rules, validation, and closure reasons. Wire up Redis registries for active shop tracking.

**Deliverables**: Shops can be created, opened, maintained, and closed with proper state enforcement.

### Phase 3: Listing Management & Bundle Model
Implement the bundle-based listing system: add/remove listings, bundle definition, inventory integration (item escrow on list, return on delist).

**Deliverables**: Items can be listed with bundle configuration, delisted, and inventory is properly managed.

### Phase 4: Visitor Management & Real-Time Updates
Implement visitor entry/exit, capacity enforcement (with character shop vs hired merchant differences), and real-time broadcast of shop state changes to all visitors.

**Deliverables**: Players can enter/exit shops, see other visitors, receive live listing updates.

### Phase 5: Purchase Flow
Implement the complete purchase flow: meso validation, inventory space check, concurrent purchase safety (optimistic locking), fee calculation, bundle decrement, and sold-out detection.

**Deliverables**: Players can purchase bundles with proper validation, concurrency safety, and automatic sold-out closure.

### Phase 6: Shop Placement & Proximity Rules
Implement Free Market placement validation: map restriction, portal proximity, shop-to-shop proximity, duplicate shop type prevention.

**Deliverables**: Shops can only be placed in valid Free Market locations with proper spacing.

### Phase 7: Character Shop Specifics
Implement character-shop-specific behavior: online requirement, disconnect auto-close, direct meso delivery to owner, owner not counting as visitor.

**Deliverables**: Character shops work end-to-end with proper online lifecycle.

### Phase 8: Hired Merchant Specifics & Frederick
Implement hired-merchant-specific behavior: 24-hour expiration, owner-as-visitor, Frederick transfers on close, Frederick notification scheduling, 100-day cleanup reaper.

**Deliverables**: Hired merchants work end-to-end with Frederick integration and expiration.

### Phase 9: Kafka Command/Event Integration
Wire all operations through Kafka command handlers and status event emitters. Integrate with external services (inventory, character, storage).

**Deliverables**: Full event-driven integration; all operations triggered via Kafka commands.

### Phase 10: Maintenance Mode & Chat
Implement maintenance mode (visitor ejection, restricted entry, listing management) and visitor chat/message system.

**Deliverables**: Owners can manage shops in maintenance mode; visitors can chat.

---

## Detailed Tasks

### Phase 1: Core Domain & Persistence

**1.1 Create service skeleton** [M]
- Create `services/atlas-merchant/atlas.com/merchant/` directory structure
- Create `main.go` with standard Atlas service initialization (logger, redis, database, kafka, rest server, teardown manager)
- Create `go.mod` with required dependencies
- Add service to `go.work`
- Create `Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug`
- Create `atlas-merchant.yml` Docker Compose config
- **Acceptance**: `go build` succeeds; service starts and connects to Postgres/Redis/Kafka

**1.2 Create shop domain model** [M]
- `shop/model.go` — immutable Model with private fields + accessors for all Shop fields
- `shop/builder.go` — fluent ModelBuilder with `SetShopType()`, `SetState()`, `SetTitle()`, `SetMapId()`, `SetPosition()`, `SetOwner()`, `SetPermitItemId()`, `SetExpiresAt()`, etc.
- Include `ShopType` enum (CharacterShop=1, HiredMerchant=2)
- Include `State` enum (Draft=1, Open=2, Maintenance=3, Closed=4)
- Include `CloseReason` enum (SoldOut=1, ManualClose=2, Disconnect=3, Expired=4, ServerRestart=5, Empty=6)
- **Acceptance**: Model compiles; all accessors return correct values; builder validates required fields

**1.3 Create listing domain model** [M]
- `listing/model.go` — immutable Model with accessors for all Listing fields
- `listing/builder.go` — fluent ModelBuilder with `SetItemId()`, `SetItemType()`, `SetQuantity()`, `SetBundleSize()`, `SetBundlesRemaining()`, `SetPricePerBundle()`, `SetSlotMetadata()`
- Include bundle validation: `BundleSize * BundlesRemaining == Quantity`
- **Acceptance**: Model compiles; bundle math is correct; builder enforces constraints

**1.4 Create GORM entities and migrations** [L]
- `shop/entity.go` — maps to `shops` table with all fields, `TenantId`, `Migration(db)` function
- `listing/entity.go` — maps to `listings` table with `ShopId` FK, ordering index, `Migration(db)` function
- `visitor/entity.go` — maps to `visitors` table with `ShopId` FK (optional — may use Redis only)
- `message/entity.go` — maps to `messages` table with `ShopId` FK
- `frederick/entity.go` — maps to `frederick_items` table for hired merchant stored items/mesos
- Define proper indexes (shop by characterId, shop by mapId, listings by shopId, frederick by characterId)
- **Acceptance**: `db.AutoMigrate()` creates all tables; entity ↔ model transforms work correctly

**1.5 Create database providers** [M]
- `shop/provider.go` — `EntityProvider` functions: `ByCharacterId`, `ByMapId`, `ById`, `Active`, `Create`, `Update`
- `listing/provider.go` — `ByShopId`, `Create`, `Delete`, `UpdateBundles`
- `frederick/provider.go` — `ByCharacterId`, `Create`, `Delete`, `UpdateNotifiedAt`
- Use automatic tenant filtering (no manual tenant WHERE clauses)
- **Acceptance**: CRUD operations work; tenant filtering is automatic

**1.6 Create REST models and basic endpoints** [M]
- `shop/rest.go` — JSON:API RestModel with `Transform()`, `Extract()`, `GetReferences()` for listings
- `listing/rest.go` — JSON:API RestModel with `Transform()`, `Extract()`
- `rest/handler.go` — register GET endpoints for shops and listings
- **Acceptance**: REST endpoints return proper JSON:API responses; relationships work

**1.7 Tests and build verification** [S]
- `shop/processor_test.go` — basic model creation tests
- `listing/processor_test.go` — bundle math validation tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 2: Shop Lifecycle State Machine

**2.1 Implement state machine transitions** [L]
- `shop/processor.go` — define `Processor` interface and implementation
- Implement `CreateShop(characterId, shopType, title, mapId, position, permitItemId) → Shop`
- Implement `OpenShop(shopId) → error` — validates Draft→Open, requires ≥1 listing
- Implement `EnterMaintenance(shopId) → error` — validates Open→Maintenance, ejects visitors
- Implement `ExitMaintenance(shopId) → error` — validates Maintenance→Open or Maintenance→Closed (if 0 listings)
- Implement `CloseShop(shopId, reason) → error` — validates Open/Maintenance→Closed
- All transitions enforce valid state flow; return typed errors for invalid transitions
- **Acceptance**: State transitions work correctly; invalid transitions are rejected

**2.2 Initialize Redis registries** [M]
- `shop/registry.go` — `TenantRegistry[uint32, uuid.UUID]` mapping characterId → shopId (active shops per character)
- `shop/registry.go` — `TenantRegistry[uuid.UUID, ShopSummary]` mapping shopId → shop summary (for map queries)
- Add reverse index: mapId → set of shopIds (Redis SET via `IndexRegistry`)
- `InitRegistry(rc)` function called from `main.go`
- **Acceptance**: Registry operations work; character can look up active shop; map can list shops

**2.3 Wire state transitions to persistence + registry** [M]
- On `CreateShop`: insert DB row (state=Draft), register in Redis
- On `OpenShop`: update DB state, update Redis summary
- On `CloseShop`: update DB state + closedAt + closeReason, remove from Redis registries
- All operations within `database.ExecuteTransaction()` for DB + message buffer for Kafka
- **Acceptance**: State changes persist correctly; Redis stays in sync

**2.4 Active shop limit enforcement** [S]
- Before creating: query Redis for character's active shops by type
- Reject if character already has active shop of same type
- Allow character to have both types simultaneously
- **Acceptance**: Cannot create two character shops or two hired merchants; can have one of each

**2.5 Tests and build verification** [S]
- State machine transition tests (valid and invalid)
- Active shop limit tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 3: Listing Management & Bundle Model

**3.1 Implement listing operations** [L]
- `listing/processor.go` — Processor interface
- Implement `AddListing(shopId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, metadata) → Listing`
  - Validate shop is in Draft or Maintenance state
  - Validate ≤16 listings
  - Validate price range (1 to max int)
  - Compute total quantity = bundleSize × bundleCount
  - Append to ordered listing list
- Implement `RemoveListing(shopId, listingIndex) → error`
  - Validate shop is in Maintenance state
  - Remove listing at index; collapse list
  - Return escrowed items to inventory
- Implement `UpdateListing(shopId, listingIndex, newPrice, newBundleSize, newBundleCount) → error`
  - Validate shop is in Maintenance state
  - Revalidate bundle constraints against remaining quantity
- **Acceptance**: Listings added/removed/updated correctly; list collapses on removal; limits enforced

**3.2 Trade restriction validation** [M]
- Before listing: verify item is tradeable
- Reject: untradeable items, drop-restricted items (unless trade-enabled), cash items, pets
- Query item data service or use cached item definitions
- **Acceptance**: Non-tradeable items cannot be listed

**3.3 Item ownership transfer via acquire/release** [M]
- On `AddListing`: **release** item from character inventory, **acquire** full item snapshot into merchant listing storage (JSONB)
  - Item snapshot includes all attributes (stats, scrolls, flags, expiration, etc.)
  - Snapshot stored in listing's `ItemSnapshot` column — this is what visitors see
- On `RemoveListing`: **release** from merchant listing, **acquire** back into character inventory
- On `Purchase`: **release** from merchant listing, **acquire** into buyer inventory
- Follow the standard acquire/release ownership transfer pattern — item exists in exactly one place at all times
- **Acceptance**: Items transfer cleanly; no duplication or loss; full item data preserved across transfers

**3.4 Tests and build verification** [S]
- Bundle math tests (various sizes, boundary conditions)
- Listing limit tests (16 max, 0 min for open)
- Trade restriction tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 4: Visitor Management & Real-Time Updates

**4.1 Implement visitor tracking** [M]
- `visitor/registry.go` — Redis TenantRegistry for active visitors per shop
- Implement `EnterShop(characterId, shopId) → error`
  - Validate shop is Open state
  - Validate same map
  - Validate capacity (≤3 visitors)
  - Character shop: owner doesn't count; Hired merchant: owner counts
- Implement `ExitShop(characterId, shopId) → error`
  - Remove from visitor list
  - No departure broadcast
- Implement `EjectAllVisitors(shopId)` — used by maintenance mode and shop closure
- **Acceptance**: Visitors can enter/exit; capacity enforced; ejection works

**4.2 Capacity enforcement by shop type** [S]
- Character shop: `visitors.Count()` ≤ 3 (owner excluded from count)
- Hired merchant: `visitors.Count()` ≤ 3 (owner included if present)
- Return capacity-full event on rejection
- **Acceptance**: Correct capacity rules per shop type

**4.3 Real-time visitor broadcast** [M]
- On listing change: emit event to all current visitors
- On visitor enter/exit: update visitor list for all current visitors
- On purchase: broadcast updated listing to all visitors
- Events published via Kafka status topic; socket layer consumes and pushes to clients
- **Acceptance**: All visitors see synchronized state

**4.4 Tests and build verification** [S]
- Visitor capacity tests (character shop vs hired merchant)
- Enter/exit flow tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 5: Purchase Flow

**5.1 Implement purchase processor** [XL]
- `shop/processor.go` — `PurchaseBundle(buyerCharacterId, shopId, listingIndex, bundleCount) → error`
- Validation chain:
  1. Buyer is currently visiting the shop
  2. Listing exists at given index
  3. Sufficient bundles remain
  4. Buyer has sufficient mesos (bundleCount × pricePerBundle)
  5. Buyer has sufficient inventory space
  6. For character shop: owner can receive mesos (not at cap)
- **DB transaction as serialization point** (critical — unlike NPC shops, merchant listings have shared mutable state):
  1. `database.ExecuteTransaction()` wrapping the listing update
  2. Decrement `BundlesRemaining` with optimistic lock (`WHERE version = ?`)
  3. If conflict (RowsAffected == 0): fail with item-unavailable event — do NOT fire Kafka
  4. COMMIT — only after this point is the purchase "real"
- **Post-commit Kafka side-effects** (fire-and-forget, follows NPC shop pattern):
  1. Deduct buyer mesos
  2. Calculate fee, credit owner (character shop: direct meso; hired merchant: meso balance in DB)
  3. Release items from listing, acquire into buyer inventory
  4. If listing bundles = 0: remove listing
  5. If all listings empty: close shop (SoldOut)
  6. Broadcast update to all visitors
- **Acceptance**: Complete purchase flow; all validation; DB gate prevents overselling

**5.2 Optimistic locking for concurrent purchases** [M]
- Add `version` column to listings entity
- `UPDATE listings SET bundles_remaining = ?, version = version + 1 WHERE id = ? AND version = ?`
- On conflict (RowsAffected == 0): re-read and retry or fail with unavailable
- First successful purchase wins; subsequent attempts get item-unavailable event
- **Acceptance**: Two simultaneous purchases for last bundle — exactly one succeeds

**5.3 Fee calculation** [S]
- Implement tiered fee schedule based on total sale price:
  - ≥100,000,000 mesos → 6%
  - ≥25,000,000 mesos → 5%
  - ≥10,000,000 mesos → 4%
  - ≥5,000,000 mesos → 3%
  - ≥1,000,000 mesos → 1.8%
  - ≥100,000 mesos → 0.8%
  - <100,000 mesos → 0% (no fee)
- Fee uses integer division (truncation, no rounding)
- Fee deducted from sale proceeds before crediting owner
- **Acceptance**: `GetFee(100_000_000)` = 6,000,000; `GetFee(25_000_000)` = 1,250,000; `GetFee(50_000)` = 0

**5.4 Tests and build verification** [M]
- Purchase flow tests (happy path, insufficient mesos, insufficient space, sold out)
- Concurrent purchase tests (optimistic locking)
- Fee calculation tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 6: Shop Placement & Proximity Rules

**6.1 Implement placement validation** [L]
- `placement/processor.go` — validate shop placement
- Must be in Free Market room (check mapId against known FM map IDs)
- Must not be too close to a portal (check position against map portal data)
- Must not be too close to another shop (check against Redis map placement index)
- Must not have pending Frederick items/mesos
- Must not already have active shop of same type
- **Acceptance**: Invalid placements rejected with specific reason; valid placements succeed

**6.2 Map placement index** [M]
- Redis SET per map: `atlas:merchant-placement:{tenantId}:{mapId}` stores `{shopId}:{x}:{y}`
- On shop open: add to index
- On shop close: remove from index
- Proximity check: scan set, compute distances
- **Acceptance**: Placement index accurately tracks shop positions; proximity checks work

**6.3 Tests and build verification** [S]
- Placement validation tests (Free Market, proximity, duplicates)
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 7: Character Shop Specifics

**7.1 Online requirement & disconnect detection** [M]
- Subscribe to `EVENT_TOPIC_CHARACTER_STATUS` for disconnect events
- On owner disconnect: close shop immediately (reason=Disconnect)
- Eject all visitors with Shop Closed event
- Return escrowed items to owner inventory
- **Acceptance**: Character shop closes on owner disconnect

**7.2 Direct meso delivery** [S]
- On purchase: credit mesos directly to owner via character meso update command
- No intermediate meso balance storage
- **Acceptance**: Owner receives mesos in real-time on each sale

**7.3 Tests and build verification** [S]
- Disconnect auto-close tests
- Direct delivery tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 8: Hired Merchant Specifics & Frederick

**8.1 24-hour expiration & reaper** [L]
- Set `ExpiresAt` = CreatedAt + 24h on hired merchant creation
- Background reaper goroutine:
  - Query shops where `expires_at < NOW() AND state IN (Open, Maintenance)`
  - Close shop (reason=Expired)
  - Transfer items/mesos to Frederick
- If owner is visiting when expired: eject like any other visitor
- Reaper interval: configurable (default 30s)
- Integrate with teardown manager for graceful shutdown
- **Acceptance**: Hired merchants auto-close after 24 hours; reaper runs reliably

**8.2 Frederick storage on close** [L]
- On hired merchant close:
  - Move all unsold listing items into `frederick_items` table (internal to atlas-merchant DB)
  - Move accumulated meso balance into `frederick_mesos` record
  - Item snapshots preserved as-is from listings (already stored as JSONB)
  - No external service call — Frederick is entirely within atlas-merchant
- On Frederick retrieval: release items/mesos from Frederick, acquire into character inventory via Kafka
- **Acceptance**: All items/mesos preserved on hired merchant close; retrievable from Frederick NPC

**8.3 Frederick notification scheduler** [M]
- Background goroutine checks Frederick items with pending notifications
- Notification schedule: 2, 5, 10, 15, 30, 60, 90 days after storage
- Emit notification events for the socket layer to deliver
- **Acceptance**: Notifications sent at correct intervals

**8.4 Frederick 100-day cleanup reaper** [M]
- Background goroutine (or extend existing reaper):
  - Query Frederick items where `stored_at + 100 days < NOW()`
  - Permanently delete items and mesos
- Log deletions for audit trail
- **Acceptance**: Items/mesos deleted after 100 days; audit log maintained

**8.5 Owner-as-visitor behavior** [S]
- When owner enters hired merchant UI: counted as visitor (consumes slot)
- When owner exits: visitor slot freed
- Owner can be ejected like any visitor on expiration
- **Acceptance**: Owner correctly occupies visitor slot in hired merchant

**8.6 Tests and build verification** [M]
- Expiration tests
- Frederick transfer tests
- Notification schedule tests
- 100-day cleanup tests
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 9: Kafka Command/Event Integration

**9.1 Define Kafka message types** [M]
- `kafka/message/merchant/kafka.go` — all command types:
  - `PlaceShop`, `OpenShop`, `CloseShop`, `EnterMaintenance`, `ExitMaintenance`
  - `AddListing`, `RemoveListing`, `UpdateListing`
  - `PurchaseBundle`
  - `EnterShop`, `ExitShop`
  - `SendMessage`
- Status event types:
  - `ShopOpened`, `ShopClosed`, `MaintenanceEntered`, `MaintenanceExited`
  - `ListingAdded`, `ListingRemoved`, `ListingUpdated`, `ListingPurchased`
  - `VisitorEntered`, `VisitorExited`, `VisitorEjected`
  - `CapacityFull`, `PurchaseFailed`
- **Acceptance**: All message types compile with proper JSON tags

**9.2 Implement Kafka consumers** [L]
- `kafka/consumer/merchant/consumer.go` — command handler registration
- Route each command type to appropriate processor method
- Parse tenant headers via standard consumer pattern
- Handle errors gracefully (log + emit failure events, don't block consumer)
- **Acceptance**: All commands properly consumed and dispatched

**9.3 Implement Kafka producers** [M]
- `kafka/producer/producer.go` — event emission functions
- Use `message.Buffer` pattern for atomic batch sends
- Emit status events on all state changes
- Emit listing events on all listing mutations
- **Acceptance**: Events emitted correctly for all operations

**9.4 External service integration** [M]
- Consume `EVENT_TOPIC_CHARACTER_STATUS` — disconnect detection (character shop auto-close)
- Consume `EVENT_TOPIC_INVENTORY_COMPARTMENT_STATUS` — inventory acquire/release confirmations
- Produce to `COMMAND_TOPIC_INVENTORY_COMPARTMENT` — item acquire/release for listing and purchase flows
- Frederick is internal — no external service integration needed for Frederick storage
- **Acceptance**: Cross-service integration works; events flow correctly

**9.5 Tests and build verification** [S]
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds

### Phase 10: Maintenance Mode & Chat

**10.1 Maintenance mode operations** [M]
- On enter maintenance: eject all visitors immediately
- Block entry attempts during maintenance (return maintenance event)
- Allow all listing operations (add, remove, update, reorder)
- Allow title change
- Allow history viewing
- On exit maintenance: if 0 listings → close shop; else → transition to Open
- **Acceptance**: Full maintenance mode workflow works correctly

**10.2 Visitor chat/messaging** [S]
- `message/processor.go` — handle chat messages within shop
- Persist messages to DB (audit trail during shop lifetime)
- Broadcast to all current visitors
- **Acceptance**: Messages sent and received by all visitors

**10.3 Final integration tests and build verification** [M]
- End-to-end flow tests for both shop types
- Run `go test ./... -count=1` and `go build`
- **Acceptance**: All tests pass; build succeeds; service is production-ready

---

## Risk Assessment and Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **Item duplication via race conditions** | Critical | Medium | Optimistic locking on listings; inventory escrow via Kafka (at-least-once); idempotent handlers |
| **Meso duplication/loss** | Critical | Medium | Transactional purchase flow; fee calculated atomically with transfer |
| **Orphaned hired merchants (no cleanup)** | High | Low | Expiration reaper with configurable interval; graceful shutdown integration |
| **Frederick item loss** | High | Low | DB persistence before item transfer; retry on Kafka delivery failure |
| **Redis/DB inconsistency** | Medium | Medium | DB is source of truth; Redis rebuilt from DB on startup; eventual consistency acceptable for visitor lists |
| **High concurrent purchases** | Medium | Medium | Optimistic locking; clear winner/loser semantics; no phantom reads |
| **Free Market placement spam** | Low | Medium | Proximity rules; one-shop-per-type limit; permit consumption |

---

## Success Metrics

- Character shops open/close correctly with online requirement
- Hired merchants persist across logout and expire after 24 hours
- Bundle purchases are concurrency-safe (no over-selling)
- Frederick stores and returns items/mesos correctly
- 100-day cleanup removes expired Frederick items
- State machine enforces all valid/invalid transitions
- All tests pass; zero item/meso duplication

---

## Required Resources and Dependencies

### Shared Libraries (existing, no changes)
- `atlas-database`, `atlas-redis`, `atlas-kafka`, `atlas-tenant`, `atlas-model`, `atlas-rest`, `atlas-constants`

### External Services (existing, consume/produce events)
- `atlas-inventory` — item acquire/release for listing and purchase transfers
- `atlas-character` — disconnect events, meso operations
- `atlas-cashshop` — permit verification
- `atlas-maps` — placement validation, Free Market identification

### New Infrastructure
- PostgreSQL database: `atlas_merchant` (shops, listings, messages, frederick_items, frederick_mesos)
- Kafka topics: `COMMAND_TOPIC_MERCHANT`, `EVENT_TOPIC_MERCHANT_STATUS`, `EVENT_TOPIC_MERCHANT_LISTING`, `EVENT_TOPIC_MERCHANT_FREDERICK`

---

## Effort Estimates

| Phase | Effort | Dependencies |
|-------|--------|-------------|
| Phase 1: Core Domain | L | None |
| Phase 2: State Machine | L | Phase 1 |
| Phase 3: Listing Management | L | Phase 2 |
| Phase 4: Visitor Management | M | Phase 2 |
| Phase 5: Purchase Flow | XL | Phase 3, Phase 4 |
| Phase 6: Placement Rules | M | Phase 2 |
| Phase 7: Character Shop | M | Phase 5 |
| Phase 8: Hired Merchant & Frederick | XL | Phase 5 |
| Phase 9: Kafka Integration | L | Phase 5 |
| Phase 10: Maintenance & Chat | M | Phase 4, Phase 9 |
| **Total** | **XL** | |

Phases 4 and 6 can be developed in parallel with Phase 3.
Phases 7 and 8 can be developed in parallel after Phase 5.
