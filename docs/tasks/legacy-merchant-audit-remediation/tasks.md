# Merchant Audit Remediation — Tasks

- **Last Updated:** 2026-02-24
- **Plan:** `plan.md`
- **Context:** `context.md`

---

## Phase 1: Blocking Issues (P0)
> Resolves: ARCH-001, STRUCT-001

- [ ] **1.1** Create `shop/administrator.go` — move `create()` and `update()` from `shop/provider.go`
- [ ] **1.2** Verify `shop/provider.go` contains only read operations
- [ ] **1.3** Create `listing/administrator.go` — move `createListing`, `deleteListing`, `updateBundles`, `deleteByShopId`, `decrementDisplayOrderAfter`, `updateListingFields` from `listing/provider.go`
- [ ] **1.4** Verify `listing/provider.go` contains only read operations (`getByShopId`, `getByShopIdAndDisplayOrder`, `countByShopId`)
- [ ] **1.5** Update `listing/exports.go` — split exports: read exports reference provider, write exports reference administrator
- [ ] **1.6** Add ingress route to `atlas-ingress.yml`: `location ~ ^/api/merchants(/.*)?$`
- [ ] **1.7** `go build` passes
- [ ] **1.8** `go test ./... -count=1` passes

---

## Phase 2: Subdomain Layering (P1)
> Resolves: ARCH-002, ARCH-003, STRUCT-004, STRUCT-005

### Frederick Subdomain
- [ ] **2.1** Create `frederick/model.go` — `ItemModel` and `MesoModel` immutable types with private fields and getters
- [ ] **2.2** Add `MakeItem(ItemEntity) (ItemModel, error)` and `MakeMeso(MesoEntity) (MesoModel, error)` functions
- [ ] **2.3** Create `frederick/administrator.go` — move all write operations out of processor:
  - `storeItems(db *gorm.DB, tenantId uuid.UUID, characterId uint32, items []StoredItem) error`
  - `storeMesos(db *gorm.DB, tenantId uuid.UUID, characterId uint32, amount uint32) error`
  - `clearItems(db *gorm.DB, characterId uint32) error`
  - `clearMesos(db *gorm.DB, characterId uint32) error`
  - `createNotification(db *gorm.DB, tenantId uuid.UUID, t tenant.Model, characterId uint32) error`
  - `clearNotifications(db *gorm.DB, characterId uint32) error`
- [ ] **2.4** Refactor `frederick/processor.go` — delegate to administrator/provider; return `[]ItemModel` and `[]MesoModel` instead of entities
- [ ] **2.5** Update `kafka/consumer/merchant/consumer.go:handleRetrieveFrederickCommand` for new model return types
- [ ] **2.6** Update `shop/processor.go:storeToFrederick` for new model types (if impacted)
- [ ] **2.7** `go build` passes after Frederick refactor

### Message Subdomain
- [ ] **2.8** Create `message/model.go` — immutable `Model` with id, shopId, characterId, content, sentAt
- [ ] **2.9** Add `Make(Entity) (Model, error)` function
- [ ] **2.10** Create `message/administrator.go` — `create(db, tenantId, shopId, characterId, content) error`
- [ ] **2.11** Create `message/provider.go` — `getByShopId(shopId) database.EntityProvider[[]Entity]`
- [ ] **2.12** Refactor `message/processor.go` — delegate to administrator/provider; return `[]Model`
- [ ] **2.13** `go build` passes after message refactor
- [ ] **2.14** `go test ./... -count=1` passes

---

## Phase 3: Kafka Event Correctness (P1)
> Resolves: KAFKA-004

- [ ] **3.1** Define `ExitMaintenanceResult` struct: `{Closed bool, CloseReason CloseReason}` or change `ExitMaintenance` return to `(bool, error)`
- [ ] **3.2** Update `shop/processor.go:ExitMaintenance` to return auto-close information
- [ ] **3.3** Update `shop/processor.go:Processor` interface with new return type
- [ ] **3.4** Fix `kafka/consumer/merchant/consumer.go:handleExitMaintenanceCommand` — emit `StatusEventShopClosed` when auto-closed, `StatusEventMaintenanceExited` otherwise
- [ ] **3.5** Fix `kafka/consumer/character/consumer.go:handleLogout` — emit `StatusEventShopClosed(characterId, shopId, CloseReasonDisconnect)` after each successful `CloseShop()`
- [ ] **3.6** Import `producer.ProviderImpl` in character consumer if not already present
- [ ] **3.7** `go build` passes
- [ ] **3.8** `go test ./... -count=1` passes

---

## Phase 4: REST & Resource Convention Alignment (P2)
> Resolves: ARCH-004, ARCH-005, MODEL-004, REST-004

- [ ] **4.1** Evaluate `server.RegisterHandler` compatibility — read atlas-rest library, compare with custom `rest/handler.go`
- [ ] **4.2** If compatible: migrate `rest/handler.go` to use standard pattern; if not: add comment documenting accepted deviation
- [ ] **4.3** Rename `shop/resource.go:InitResource` to `InitializeRoutes`
- [ ] **4.4** Update `main.go:83` — `shop.InitializeRoutes(GetServer())(db)`
- [ ] **4.5** Delete `shop/rest.go:Extract()` function and unused `fmt` import
- [ ] **4.6** Create `shop/state.go` — move `ShopType`, `State`, `CloseReason` types and constants from `model.go`
- [ ] **4.7** Verify `shop/model.go` no longer has enum definitions
- [ ] **4.8** `go build` passes
- [ ] **4.9** `go test ./... -count=1` passes

---

## Phase 5: Kafka AndEmit Pattern (P1)
> Resolves: KAFKA-003
> Depends on: Phase 3

- [ ] **5.1** Create `kafka/message/message.go` — `Buffer`, `Emit`, `EmitWithResult` types following atlas-notes pattern
- [ ] **5.2** Add `producer producer.Provider` field to `shop.ProcessorImpl`
- [ ] **5.3** Create `NewProcessorWithProducer(l, ctx, db, p)` constructor (keep `NewProcessor` for REST/reaper use)
- [ ] **5.4** Add `AndEmit` variants to `Processor` interface:
  - `OpenShopAndEmit(shopId uuid.UUID) error`
  - `CloseShopAndEmit(shopId uuid.UUID, reason CloseReason) error`
  - `EnterMaintenanceAndEmit(shopId uuid.UUID) error`
  - `ExitMaintenanceAndEmit(shopId uuid.UUID) (ExitMaintenanceResult, error)`
  - `PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error)`
  - `EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error`
  - `ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error`
- [ ] **5.5** Implement `AndEmit` methods using `message.Emit`/`message.EmitWithResult` + `message.Buffer`
- [ ] **5.6** Migrate `kafka/consumer/merchant/consumer.go` handlers to use `AndEmit` variants
- [ ] **5.7** Verify consumer handlers are now thin (create processor, call AndEmit, handle error)
- [ ] **5.8** `go build` passes
- [ ] **5.9** `go test ./... -count=1` passes

---

## Phase 6: Documentation, Testing & Polish (P1/P2)
> Resolves: TEST-001, STRUCT-002, STRUCT-003, ARCH-006
> Depends on: All prior phases

### Testing
- [ ] **6.1** Create `shop/validation_test.go`:
  - `TestIsFreemarketRoom_ValidRooms` — spot-check known free market map IDs
  - `TestIsFreemarketRoom_InvalidRooms` — non-free-market maps return false
  - `TestIsListableItem_Normal` — normal tradeable item passes
  - `TestIsListableItem_Pet` — pet classification blocked
  - `TestIsListableItem_Cash` — cash item blocked
  - `TestIsListableItem_Untradeable` — untradeable flag blocked
  - `TestManhattanDistance` — various coordinate pairs
- [ ] **6.2** Extend `shop/processor_test.go` with state machine tests:
  - Test CreateShop happy path (character shop + hired merchant)
  - Test CreateShop validation failures (not free market, too close, shop limit, Frederick pending)
  - Test OpenShop (requires Draft + listings; rejects other states)
  - Test CloseShop (valid from Open/Maintenance/Draft; rejects Closed)
  - Test EnterMaintenance (requires Open; ejects visitors)
  - Test ExitMaintenance (returns to Open; auto-closes when empty)
  - Test PurchaseBundle (happy path, insufficient bundles, version conflict, sold-out auto-close)
  - Test EnterShop/ExitShop (capacity limit)
  - Setup: SQLite in-memory DB + RegisterTenantCallbacks + miniredis
- [ ] **6.3** Create `listing/provider_test.go`:
  - Test createListing / getByShopId round-trip
  - Test updateBundles with correct version (succeeds)
  - Test updateBundles with wrong version (returns 0 rows)
  - Test decrementDisplayOrderAfter
  - Test deleteByShopId
- [ ] **6.4** Verify test count: `go test ./... -count=1` reports tests in >= 4 packages (up from 2)

### Documentation
- [ ] **6.5** Update `services/atlas-merchant/README.md`:
  - Add REST Endpoints table (4 endpoints)
  - Add Kafka Commands table (13 commands)
  - Add Kafka Events table (status + listing events)
  - Remove broken doc links OR create the linked files
- [ ] **6.6** Create Bruno collection:
  - `services/atlas-merchant/.bruno/bruno.json`
  - `services/atlas-merchant/.bruno/collection.bru`
  - `services/atlas-merchant/.bruno/environments/local.bru`
  - Request files for 4 GET endpoints

### Final
- [ ] **6.7** Add rationale comment to `listing/exports.go` header explaining cross-package access pattern
- [ ] **6.8** Final `go build` passes
- [ ] **6.9** Final `go test ./... -count=1` passes
- [ ] **6.10** Re-run `/backend-audit atlas-merchant` — all checks pass

---

## Progress Summary

| Phase | Tasks | Done | Status |
|-------|-------|------|--------|
| 1 — Blocking Issues | 8 | 0 | Pending |
| 2 — Subdomain Layering | 14 | 0 | Pending |
| 3 — Kafka Event Correctness | 8 | 0 | Pending |
| 4 — REST Convention | 9 | 0 | Pending |
| 5 — Kafka AndEmit | 9 | 0 | Pending |
| 6 — Docs, Tests & Polish | 10 | 0 | Pending |
| **Total** | **58** | **0** | **Not Started** |
