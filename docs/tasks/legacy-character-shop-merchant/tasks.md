# Character Shop & Hired Merchant — Task Checklist

Last Updated: 2026-02-25

## Phase 1: Core Domain & Persistence

- [x] **1.1** Create service skeleton (`services/atlas-merchant/`) with main.go, go.mod, Dockerfiles, docker-compose yml
- [x] **1.2** Add service to `go.work`
- [x] **1.3** Create shop domain model (`shop/model.go`) — immutable model with ShopType, State, CloseReason enums
- [x] **1.4** Create shop model builder (`shop/builder.go`) — fluent builder with validation
- [x] **1.5** Create listing domain model (`listing/model.go`) — immutable model with bundle fields + ItemSnapshot (JSONB)
- [x] **1.6** Create listing model builder (`listing/builder.go`) — bundle math validation
- [x] **1.7** Create GORM entities and migrations
  - [x] `shop/entity.go` — shops table with all fields, TenantId, tenant info for reaper, indexes
  - [x] `listing/entity.go` — listings table with ShopId FK, display_order, version, item_snapshot JSONB
  - [x] `message/entity.go` — messages table with ShopId FK
  - [x] `frederick/entity.go` — frederick_items + frederick_mesos tables (internal to merchant service)
- [x] **1.8** Create database providers
  - [x] `shop/provider.go` — ByCharacterId, ByMapId, ById, Active, Create, Update, GetExpired
  - [x] `listing/provider.go` — ByShopId, Create, Delete, UpdateBundles (with optimistic lock), CountByShopId, ByShopIdAndDisplayOrder, DecrementDisplayOrderAfter, UpdateListingFields
  - [x] `frederick/provider.go` — ByCharacterId, HasItemsOrMesos (exported)
- [x] **1.9** Create REST models and basic GET endpoints
  - [x] `shop/rest.go` — JSON:API RestModel with Transform, TransformWithListings, GetReferences
  - [x] `listing/rest.go` — JSON:API RestModel with Transform
  - [x] `rest/handler.go` — register GET /merchants, GET /merchants/{id}, GET /characters/{id}/merchants
- [x] **1.10** Create logger, tracing, service lifecycle boilerplate
- [x] **1.11** Tests pass: `go test ./... -count=1`
- [x] **1.12** Build succeeds: `go build`

## Phase 2: Shop Lifecycle State Machine

- [x] **2.1** Implement `shop/processor.go` — Processor interface with state machine methods
  - [x] `CreateShop(characterId, shopType, title, mapId, position, permitItemId) → Shop`
  - [x] `OpenShop(shopId) → error` (Draft→Open, requires ≥1 listing)
  - [x] `EnterMaintenance(shopId) → error` (Open→Maintenance, ejects visitors)
  - [x] `ExitMaintenance(shopId) → error` (Maintenance→Open or Maintenance→Closed if 0 listings)
  - [x] `CloseShop(shopId, reason) → error` (Open/Maintenance/Draft→Closed)
- [x] **2.2** Implement invalid state transition rejection with typed errors
- [x] **2.3** Initialize Redis registries (`shop/registry.go`)
  - [x] `TenantRegistry[uint32, ActiveShopEntry]` — characterId → {shopId, shopType, mapId}
  - [x] Map placement index: mapId → set of shopIds (via atlas.Index)
- [x] **2.4** Wire state transitions to DB persistence + Redis registry sync
- [x] **2.5** Enforce active shop limits (max 1 per type per character)
- [x] **2.6** Tests pass: `go test ./... -count=1`
- [x] **2.7** Build succeeds: `go build`

## Phase 3: Listing Management & Bundle Model

- [x] **3.1** Implement listing management on shop processor
  - [x] `AddListing(shopId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot) → Listing`
  - [x] `RemoveListing(shopId, listingIndex) → (Listing, error)` — returns removed listing for item return
  - [x] `UpdateListing(shopId, listingIndex, newPrice, newBundleSize, newBundleCount) → error`
- [x] **3.2** Enforce listing limits (max 16, price ≥ 1, bundleSize ≥ 1, bundleCount ≥ 1)
- [x] **3.3** Implement ordered dynamic list behavior (display_order, collapse on removal)
- [ ] **3.4** Implement trade restriction validation (untradeable, cash items, pets rejected) — deferred to external data
- [ ] **3.5** Implement item ownership transfer via acquire/release — Kafka flow for inventory integration
- [x] **3.6** Tests pass: `go test ./... -count=1`
- [x] **3.7** Build succeeds: `go build`

## Phase 4: Visitor Management & Real-Time Updates

- [x] **4.1** Implement visitor Redis registry (`visitor/registry.go`) with Index-based SETs
  - [x] shopId → set of characterIds (forward lookup)
  - [x] characterId → shopId (reverse lookup)
- [x] **4.2** Implement `EnterShop(characterId, shopId) → error` — validate Open state, capacity ≤ 3
- [x] **4.3** Implement `ExitShop(characterId, shopId) → error`
- [x] **4.4** Implement `EjectAllVisitors(shopId)` — used by maintenance and close
- [x] **4.5** Capacity handling: protocol layer distinguishes character shop (owner excluded) vs hired merchant (owner counted)
- [x] **4.6** Kafka event emission for visitor enter/exit/eject/capacity full
- [x] **4.7** Tests pass: `go test ./... -count=1`
- [x] **4.8** Build succeeds: `go build`

## Phase 5: Purchase Flow

- [x] **5.1** Implement `PurchaseBundle(buyerCharacterId, shopId, listingIndex, bundleCount) → PurchaseResult`
  - [x] Validate shop is Open
  - [x] Validate listing exists (by display order)
  - [x] Validate sufficient bundles remain
- [x] **5.2** Implement DB-gated purchase execution
  - [x] `database.ExecuteTransaction()` wrapping listing update as serialization point
  - [x] Decrement bundles remaining with optimistic lock (`WHERE version = ?`)
  - [x] On conflict (RowsAffected == 0): return ErrVersionConflict — Kafka emits purchase failed
  - [x] Post-commit: fire-and-forget Kafka for listing purchased event
- [x] **5.3** Implement sold-out detection
  - [x] If listing bundles = 0: remove listing + collapse display order
  - [x] If all listings empty: close shop (SoldOut reason)
- [x] **5.4** Version column on listings entity for optimistic locking (in Phase 1)
- [x] **5.5** Concurrent purchase safety: DB gate ensures first wins, others get version conflict
- [x] **5.6** Fee calculation (tiered: 0%/0.8%/1.8%/3%/4%/5%/6% by sale amount, integer division)
- [x] **5.7** PurchaseResult includes fee, net amount for Kafka side effects
- [x] **5.8** Tests pass: `go test ./... -count=1`
- [x] **5.9** Build succeeds: `go build`

## Phase 6: Shop Placement & Proximity Rules

- [ ] **6.1** Implement Free Market room validation (mapId check) — deferred to external map data
- [ ] **6.2** Implement portal proximity check — deferred to external portal data
- [ ] **6.3** Implement shop-to-shop proximity check — deferred to external map data
- [x] **6.4** Implement pending Frederick check (block placement if items/mesos waiting at Frederick)
- [x] **6.5** Implement duplicate shop type check (via DB getActiveByCharacterIdAndType)
- [x] **6.6** Implement map placement index (Redis Index per map, add on open, remove on close)
- [x] **6.7** Tests pass: `go test ./... -count=1`
- [x] **6.8** Build succeeds: `go build`

## Phase 7: Character Shop Specifics

- [x] **7.1** Subscribe to `EVENT_TOPIC_CHARACTER_STATUS` for disconnect events
  - [x] `kafka/message/character/kafka.go` — StatusEvent with logout body
  - [x] `kafka/consumer/character/consumer.go` — InitConsumers + InitHandlers
- [x] **7.2** Implement auto-close on owner disconnect (reason=Disconnect)
  - [x] Close all active character shops for the disconnecting character
- [ ] **7.3** Implement direct meso delivery to owner on purchase — deferred to inventory Kafka integration
- [x] **7.4** Owner-excluded visitor capacity handled at protocol layer
- [x] **7.5** Tests pass: `go test ./... -count=1`
- [x] **7.6** Build succeeds: `go build`

## Phase 8: Hired Merchant Specifics & Frederick

- [x] **8.1** Set `ExpiresAt` = CreatedAt + 24h on hired merchant creation
- [x] **8.2** Implement expiration reaper goroutine (`shop/reaper.go`)
  - [x] Query expired Open/Maintenance shops across all tenants (WithoutTenantFilter)
  - [x] Close shop (reason=Expired) via processor with reconstructed tenant context
  - [x] Default 30s interval
  - [x] Graceful shutdown via context cancellation
- [x] **8.3** Implement Frederick storage on close (internal to atlas-merchant DB)
  - [x] `frederick/processor.go` — StoreItems, StoreMesos
  - [x] `shop/processor.go` — storeToFrederick called from CloseShop for hired merchants
  - [x] Move unsold listing item snapshots into `frederick_items` table
  - [x] Move accumulated meso balance into `frederick_mesos` record
- [x] **8.4** Implement Frederick retrieval
  - [x] GetItems, GetMesos, ClearItems, ClearMesos in frederick processor
  - [x] RetrieveFrederick Kafka command handler
- [ ] **8.5** Implement Frederick notification scheduler — deferred (notification timing logic)
- [x] **8.6** Implement 100-day Frederick cleanup reaper
  - [x] `frederick/processor.go` — StartCleanupReaper (6h interval, 100-day cutoff)
  - [x] Permanently delete items and mesos, log deletions
- [x] **8.7** Owner-as-visitor behavior — handled at protocol layer (EnterShop counts toward capacity)
- [x] **8.8** Tests pass: `go test ./... -count=1`
- [x] **8.9** Build succeeds: `go build`

## Phase 9: Kafka Command/Event Integration

- [x] **9.1** Define all Kafka message types (`kafka/message/merchant/kafka.go`)
  - [x] Commands: PlaceShop, OpenShop, CloseShop, EnterMaintenance, ExitMaintenance, AddListing, RemoveListing, UpdateListing, PurchaseBundle, EnterShop, ExitShop, SendMessage, RetrieveFrederick
  - [x] Events: ShopOpened, ShopClosed, MaintenanceEntered/Exited, VisitorEntered/Exited/Ejected, CapacityFull, PurchaseFailed, ListingPurchased
- [x] **9.2** Implement Kafka consumers (`kafka/consumer/merchant/consumer.go`)
  - [x] Route all 13 command types to processor methods
  - [x] Tenant + span header parsing via standard pattern
- [x] **9.3** Implement character disconnect consumer (`kafka/consumer/character/consumer.go`)
- [ ] **9.4** Implement inventory confirmation consumer — deferred to inventory integration
- [x] **9.5** Implement Kafka producers
  - [x] `shop/producer.go` — Status event providers (opened, closed, maintenance, visitor, capacity, purchase failed)
  - [x] `shop/producer.go` — Listing event providers (purchased)
  - [x] `kafka/producer/producer.go` — ProviderImpl with span + tenant header decorators
- [x] **9.6** Tests pass: `go test ./... -count=1`
- [x] **9.7** Build succeeds: `go build`

## Phase 10: Maintenance Mode & Chat

- [x] **10.1** Maintenance mode entry ejects all visitors, blocks entry (in EnterMaintenance)
- [x] **10.2** Maintenance operations: add/remove/update listings (validated for Draft/Maintenance states)
- [x] **10.3** Maintenance mode exit: 0 listings → close; else → open (in ExitMaintenance)
- [x] **10.4** Implement visitor chat messaging (`message/processor.go`)
  - [x] Persist to DB (SendMessage)
  - [x] GetMessages query
  - [x] SendMessage Kafka command handler
- [x] **10.5** Tests pass: `go test ./... -count=1`
- [x] **10.6** Build succeeds: `go build`

## Deferred Items

- [ ] Trade restriction validation (needs external item data service integration)
- [ ] Item ownership acquire/release via Kafka (inventory service integration)
- [ ] Free Market room validation (needs map data)
- [ ] Portal/shop proximity checks (needs portal/map data)
- [ ] Direct meso delivery to owner (inventory service integration)
- [ ] Frederick notification scheduler (notification timing logic)
- [ ] Inventory confirmation consumer (inventory service integration)

## Post-Implementation

- [ ] End-to-end test: character shop open → purchase → sold out → close
- [ ] End-to-end test: hired merchant persist across logout → relogin → retrieve from Frederick
- [ ] End-to-end test: hired merchant expiration after 24h → Frederick storage
- [ ] End-to-end test: concurrent purchases on last bundle → exactly one succeeds (DB gate)
- [ ] End-to-end test: maintenance mode → modify listings → reopen
- [ ] End-to-end test: placement proximity rules in Free Market
- [ ] End-to-end test: Frederick 100-day cleanup
- [ ] Update `docs/architectural-improvements.md` with merchant service documentation
- [ ] Update `MEMORY.md` with merchant service status
