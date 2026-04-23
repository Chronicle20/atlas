# Merchant Audit Remediation Plan

- **Last Updated:** 2026-02-24
- **Source:** `docs/audits/atlas-merchant/audit.md`
- **Service:** `services/atlas-merchant/atlas.com/merchant`
- **Branch:** `redis-registry-migration`
- **Target:** Resolve all 13 non-passing audit checks (2 fail, 11 warn)

---

## Executive Summary

The atlas-merchant audit identified 2 blocking failures and 11 warnings across architecture, structure, Kafka, testing, and documentation. This plan addresses all findings in 6 phases ordered by dependency and priority. Phase 1 resolves the blocking issues (administrator.go + ingress). Phases 2-3 address medium-impact structural and functional gaps. Phases 4-5 tackle the larger efforts (AndEmit pattern, test coverage). Phase 6 handles remaining low-impact cleanups.

**Estimated total effort:** ~L (broken into individually manageable tasks)

---

## Current State

- **Build:** PASS
- **Tests:** PASS (2/19 packages have tests)
- **Passing checks:** 15 (KAFKA-001/002, TENANT-001/002/003, MODEL-001/002/003, REST-001/002/003, INFRA-001/002)
- **Failing checks:** 2 (ARCH-001, STRUCT-001)
- **Warning checks:** 11 (ARCH-002/003/004/005/006, KAFKA-003/004, MODEL-004, STRUCT-002/003/004/005, REST-004, TEST-001)

---

## Proposed Future State

All 28 audit checks passing. Specifically:
- Read/write separation enforced via `administrator.go` in all domain packages
- Ingress route active for `/api/merchants`
- Frederick and message subdomains have proper model/entity/administrator/provider layering
- Kafka emissions use atomic message buffer pattern
- Auto-close and disconnect emit correct events
- Comprehensive test coverage for processor, validation, and consumer logic
- README with endpoint/Kafka documentation tables
- Bruno collection for REST API testing
- Code aligned with standard `server.RegisterHandler` and `InitializeRoutes` conventions

---

## Implementation Phases

### Phase 1: Blocking Issues (P0)
**Goal:** Resolve the 2 fail-status checks. Must complete before any feature work.

Resolves: ARCH-001, STRUCT-001

**1.1 Create shop/administrator.go**
- Move `create(entity)` and `update(entity)` from `shop/provider.go` to `shop/administrator.go`
- Follow reference pattern from `atlas-notes/note/administrator.go`
- Keep function signatures identical (private functions, same `database.EntityProvider[Entity]` return type)
- Update `shop/processor.go` imports if needed (same package, no import change)
- **Acceptance:** `shop/provider.go` contains only read queries; `shop/administrator.go` contains all writes; `go build` passes

**1.2 Create listing/administrator.go**
- Move `createListing`, `deleteListing`, `updateBundles`, `deleteByShopId`, `decrementDisplayOrderAfter`, `updateListingFields` from `listing/provider.go` to `listing/administrator.go`
- Keep `getByShopId`, `getByShopIdAndDisplayOrder`, `countByShopId` in `listing/provider.go`
- Update `listing/exports.go` imports if needed (same package, no import change)
- **Acceptance:** `listing/provider.go` contains only reads; `listing/administrator.go` contains all writes; `go build` passes

**1.3 Add ingress route**
- Add to `atlas-ingress.yml` (alphabetically placed):
  ```nginx
  location ~ ^/api/merchants(/.*)?$ {
    proxy_pass http://atlas-merchant.atlas.svc.cluster.local:8080;
  }
  ```
- **Acceptance:** Route present in `atlas-ingress.yml`; matches existing service routes pattern

**1.4 Verify**
- `go test ./... -count=1` — all tests pass
- `go build` — clean build

---

### Phase 2: Subdomain Layering (P1)
**Goal:** Bring Frederick and message subdomains into compliance with model/entity separation and administrator/provider pattern.

Resolves: ARCH-002, ARCH-003, STRUCT-004, STRUCT-005

**2.1 Create frederick/model.go**
- Define immutable `ItemModel` and `MesoModel` with private fields and value-receiver getters
- Add `MakeItem(ItemEntity) (ItemModel, error)` and `MakeMeso(MesoEntity) (MesoModel, error)` functions
- **Acceptance:** Models are immutable; Make functions use builder or direct construction

**2.2 Create frederick/administrator.go**
- Move all write operations from `frederick/processor.go`:
  - `storeItems(db, tenantId, characterId, items)` — INSERT items
  - `storeMesos(db, tenantId, characterId, amount)` — INSERT meso record
  - `clearItems(db, characterId)` — DELETE items for character
  - `clearMesos(db, characterId)` — DELETE mesos for character
  - `createNotification(db, tenantId, ...)` — INSERT notification
  - `clearNotifications(db, characterId)` — DELETE notifications
- Functions take `*gorm.DB` (already contextualized) instead of being methods on Processor
- **Acceptance:** `frederick/processor.go` delegates all DB writes to administrator functions

**2.3 Refactor frederick/provider.go**
- Move `GetItems` and `GetMesos` query logic to proper `database.EntityProvider` functions
- Processor calls providers via `model.SliceMap(MakeItem)(getItemsByCharacterId(cid)(p.db.WithContext(p.ctx)))(model.ParallelMap())()`
- Processor returns `[]ItemModel` and `[]MesoModel` instead of raw entities
- **Acceptance:** Processor returns models; provider contains only reads; no direct DB access in processor

**2.4 Update callers of Frederick processor**
- Update `kafka/consumer/merchant/consumer.go:handleRetrieveFrederickCommand` to work with new model types
- Update `shop/processor.go:storeToFrederick` to work with new model types
- **Acceptance:** All callers compile; `go build` passes

**2.5 Create message/model.go**
- Define immutable `Model` with private fields (id, shopId, characterId, content, sentAt)
- Add `Make(Entity) (Model, error)` function

**2.6 Create message/administrator.go and message/provider.go**
- `message/administrator.go`: `create(db, tenantId, shopId, characterId, content)` — INSERT
- `message/provider.go`: `getByShopId(shopId)` — SELECT ORDER BY sent_at
- Refactor `message/processor.go` to delegate

**2.7 Verify**
- `go test ./... -count=1` — all tests pass
- `go build` — clean build

---

### Phase 3: Kafka Event Correctness (P1)
**Goal:** Fix missing/incorrect Kafka event emissions on auto-close and disconnect scenarios.

Resolves: KAFKA-004

**3.1 Fix handleExitMaintenanceCommand auto-close event**
- `ExitMaintenance` processor method currently returns `error` — no way to know if shop was auto-closed
- Option A: Return a result struct `ExitMaintenanceResult{Closed bool, CloseReason CloseReason}` instead of just `error`
- Option B: Return `(bool, error)` where bool indicates auto-close
- Update consumer to check return value and emit `StatusEventShopClosed` when auto-closed instead of `StatusEventMaintenanceExited`
- **Acceptance:** When ExitMaintenance auto-closes due to empty listings, `StatusEventShopClosed` is emitted (not `StatusEventMaintenanceExited`)

**3.2 Fix handleLogout missing closure event**
- `kafka/consumer/character/consumer.go:handleLogout` closes character shops but emits no events
- After each successful `p.CloseShop()`, emit `StatusEventShopClosed` with `CloseReasonDisconnect`
- Requires creating a Kafka producer in the character consumer (import `producer.ProviderImpl`)
- **Acceptance:** Character disconnect emits `StatusEventShopClosed` for each closed shop

**3.3 Verify**
- `go test ./... -count=1` — all tests pass
- `go build` — clean build

---

### Phase 4: REST & Resource Convention Alignment (P2)
**Goal:** Align REST handler registration with standard patterns and clean up naming.

Resolves: ARCH-004, ARCH-005, REST-004

**4.1 Evaluate server.RegisterHandler compatibility**
- Read the current `rest/handler.go` custom implementation
- Compare with standard `server.RegisterHandler` from atlas-rest library (used by atlas-buddies, atlas-notes)
- The atlas-merchant custom `rest.RegisterHandler` passes `db` as a curried parameter — check if the standard pattern does the same
- If compatible: migrate. If not: document why the deviation exists and mark as accepted deviation
- **Acceptance:** Either migrated to standard pattern OR documented as accepted deviation with rationale

**4.2 Rename InitResource to InitializeRoutes**
- Rename `shop/resource.go:InitResource` to `InitializeRoutes`
- Update `main.go:83` caller: `shop.InitializeRoutes(GetServer())(db)`
- **Acceptance:** Function named `InitializeRoutes`; builds clean

**4.3 Remove dead code Extract function**
- Delete `shop/rest.go:Extract()` and its `fmt` import (if no longer needed)
- Search for any callers first — expected none
- **Acceptance:** No dead code; `go build` passes

**4.4 Extract state enums to shop/state.go**
- Move `ShopType`, `State`, `CloseReason` type definitions and their constants from `shop/model.go` to new `shop/state.go`
- Keep `Model` struct and accessors in `model.go`
- **Acceptance:** Enums in `state.go`; model in `model.go`; `go build` passes

**4.5 Verify**
- `go test ./... -count=1` — all tests pass
- `go build` — clean build

---

### Phase 5: Kafka AndEmit Pattern (P1)
**Goal:** Introduce atomic message buffering for Kafka emissions to prevent partial-failure inconsistencies.

Resolves: KAFKA-003

**5.1 Create kafka/message/message.go**
- Implement `Buffer` and `Emit`/`EmitWithResult` following atlas-notes pattern
- `Buffer.Put(topic, provider)` accumulates messages
- `Emit(producer)` flushes all buffered messages after successful operation
- **Acceptance:** Buffer, Emit, EmitWithResult functions compile

**5.2 Add producer field to ProcessorImpl**
- Add `p producer.Provider` field to `shop.ProcessorImpl`
- Initialize in `NewProcessor` — requires Kafka producer to be available at processor creation
- This means consumer handlers must pass the producer when creating the processor
- Alternative: Create a `NewProcessorWithProducer` constructor, keep existing `NewProcessor` for REST-only use
- **Acceptance:** ProcessorImpl can hold a producer reference

**5.3 Add AndEmit variants to Processor interface**
- Focus on the highest-value operations:
  - `OpenShopAndEmit(shopId) error` — emits StatusEventShopOpened
  - `CloseShopAndEmit(shopId, reason) error` — emits StatusEventShopClosed
  - `EnterMaintenanceAndEmit(shopId) error` — emits StatusEventMaintenanceEntered
  - `ExitMaintenanceAndEmit(shopId) (ExitMaintenanceResult, error)` — emits correct event
  - `PurchaseBundleAndEmit(buyerCharacterId, shopId, listingIndex, bundleCount, worldId) (PurchaseResult, error)` — emits all 4 messages atomically
- Keep non-AndEmit variants for internal use (reaper, tests)
- **Acceptance:** Interface extended; AndEmit methods wrap operations with `message.Emit`

**5.4 Migrate consumer handlers to use AndEmit**
- Replace individual `kp(topic)(provider)` calls in each handler with processor `AndEmit` calls
- Purchase handler becomes single `p.PurchaseBundleAndEmit(...)` call
- Significantly simplifies consumer handler code
- **Acceptance:** Consumer handlers are thin (create processor, call AndEmit, handle errors)

**5.5 Verify**
- `go test ./... -count=1` — all tests pass
- `go build` — clean build

---

### Phase 6: Documentation, Testing & Polish (P1/P2)
**Goal:** Comprehensive test coverage, complete documentation, and final structural cleanup.

Resolves: TEST-001, STRUCT-002, STRUCT-003, ARCH-006

**6.1 Add validation tests**
- `shop/validation_test.go`:
  - `TestIsFreemarketRoom` — known valid/invalid map IDs
  - `TestIsListableItem` — pet, cash, untradeable, normal items
  - `TestManhattanDistance` — edge cases
- **Acceptance:** All validation functions have table-driven tests

**6.2 Add processor state machine tests**
- `shop/processor_test.go` (extend existing):
  - Test `CreateShop` — happy path, shop limit, Frederick pending, proximity
  - Test `OpenShop` — requires Draft + listings, rejects other states
  - Test `CloseShop` — valid from Open/Maintenance/Draft, rejects Closed
  - Test `EnterMaintenance` — requires Open
  - Test `ExitMaintenance` — auto-close when empty
  - Test `PurchaseBundle` — optimistic lock, sold-out auto-close, fee calculation integration
- Requires SQLite in-memory DB with `RegisterTenantCallbacks` + miniredis for registry
- **Acceptance:** All state transitions tested including error paths; `go test ./... -count=1` passes

**6.3 Add listing provider tests**
- `listing/provider_test.go`:
  - Test CRUD operations
  - Test optimistic locking (updateBundles with version mismatch)
  - Test display order decrement
- **Acceptance:** Provider functions have table-driven tests

**6.4 Update README**
- Add REST Endpoints table:
  ```
  | Method | Path | Description |
  |--------|------|-------------|
  | GET | /api/merchants?mapId={mapId} | Get shops on a map |
  | GET | /api/merchants/{shopId} | Get shop with listings |
  | GET | /api/merchants/{shopId}/relationships/listings | Get shop listings |
  | GET | /api/characters/{characterId}/merchants | Get character's shops |
  ```
- Add Kafka Commands table (13 commands on COMMAND_TOPIC_MERCHANT)
- Add Kafka Events table (status events on EVENT_TOPIC_MERCHANT_STATUS, listing events on EVENT_TOPIC_MERCHANT_LISTING)
- Remove broken links to non-existent docs/ files OR create the linked documentation files
- **Acceptance:** README has accurate endpoint/Kafka tables; no broken links

**6.5 Create Bruno collection**
- `services/atlas-merchant/.bruno/bruno.json` — collection config
- `services/atlas-merchant/.bruno/collection.bru` — collection definition
- `services/atlas-merchant/.bruno/environments/local.bru` — local environment (localhost:8080)
- Request files for each endpoint:
  - `Get Merchants By Map.bru`
  - `Get Merchant.bru`
  - `Get Merchant Listings.bru`
  - `Get Character Merchants.bru`
- **Acceptance:** Bruno collection present with all 4 endpoints

**6.6 Document listing exports pattern decision**
- ARCH-006 notes the listing exports pattern creates cross-package coupling
- Decision: **Accept as-is** for this service — listing is a subdomain of the shop aggregate, not an independent service boundary. The exports pattern is appropriate for intra-service cross-package access.
- Add a brief comment at the top of `listing/exports.go` explaining the rationale
- **Acceptance:** Pattern documented; no code change needed

**6.7 Final verification**
- `go test ./... -count=1` — all packages with test files pass
- `go build` — clean build
- Re-run `/backend-audit atlas-merchant` to verify all checks pass

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Administrator.go refactor breaks callers | Low | Medium | Same-package move; no import changes needed |
| Frederick model refactor breaks consumer | Medium | Medium | Update callers in same phase; build after each step |
| server.RegisterHandler incompatibility | Medium | Low | Evaluate first; accept deviation if incompatible |
| AndEmit refactor is larger than estimated | Medium | High | Phase 5 is self-contained; can be deferred without blocking other phases |
| Test infrastructure (SQLite + miniredis) setup complexity | Medium | Medium | Follow existing test patterns from other services |

---

## Success Metrics

- All 28 audit checks pass (0 fail, 0 warn)
- `go build` passes
- `go test ./... -count=1` passes with tests in >= 6 packages (up from 2)
- README has complete API documentation
- Ingress route is active

---

## Dependencies

- **atlas-rest** library: Need to verify `server.RegisterHandler` signature compatibility (Phase 4)
- **atlas-kafka** library: Need `message.Buffer`, `message.Emit` types (Phase 5 — may need to create locally if not in shared lib)
- **miniredis**: Test dependency for Redis registry tests (Phase 6)
- **testify**: Already in go.mod for existing tests

---

## Phase Execution Order

```
Phase 1 (Blocking)
  └─→ Phase 2 (Subdomain Layering)
  └─→ Phase 3 (Kafka Events)      ──→ Phase 5 (AndEmit Pattern)
  └─→ Phase 4 (REST Convention)
                                    ──→ Phase 6 (Testing + Docs)
```

Phases 2, 3, and 4 can proceed in parallel after Phase 1.
Phase 5 depends on Phase 3 (event correctness must be right before wrapping in AndEmit).
Phase 6 depends on all prior phases (tests should cover the final code state).
