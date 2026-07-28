# Legacy Merchant Audit Remediation — Disposition

- **Assessed:** 2026-07-14
- **Branch:** `task-127-owl-shop-search`
- **Source plan:** `docs/tasks/legacy-merchant-audit-remediation/` (context.md, plan.md, tasks.md) — 0/58 tasks, never executed
- **Assessed against:** `services/atlas-merchant/atlas.com/merchant` (+ `deploy/shared/routes.conf`) at HEAD
- **Method:** Read-only inspection of current source. `go build ./...` verified clean (covers all "go build passes" gates). Full `go test` suite NOT executed (test gates left UNVERIFIABLE).

The legacy plan was written against a much older codebase. In the interim the service absorbed Gen3 processor unification (task-116), notification-gaps / shop-interactions work, outbox adoption (task-114), and this branch's merchant-lifecycle remediation. The net effect: the overwhelming majority of the plan's asks already exist, often implemented via a superior pattern (e.g. the outbox-backed `AndEmit` methods instead of a bespoke result struct).

## Summary

| Class | Count |
|-------|------:|
| DONE-BY-SUPERSESSION | 42 |
| MOOT | 7 |
| STILL-OPEN | 2 |
| UNVERIFIABLE | 7 |
| **Total** | **58** |

- **UNVERIFIABLE** are all process gates I did not execute: the six `go test ./... -count=1` gates (1.8, 2.14, 3.8, 4.9, 5.9, 6.9) and the "re-run /backend-audit" gate (6.10). Test files now exist in ≥6 packages (shop, listing, message, frederick, searchcount, visitor) and `go build ./...` is clean, so these are expected to pass but were not run.

## STILL-OPEN items

| Task | Gap | Effort | Risk |
|------|-----|--------|------|
| **6.1** (partial) | `shop/validation_test.go` has `TestIsFreemarketRoom_*` and `TestManhattanDistance`, but the `TestIsListableItem_*` cases (Normal / Pet / Cash / Untradeable) are absent even though `IsListableItem` still exists (`shop/validation.go:121`). | trivial mechanical | Low. Pure-function, table-driven test; no fixtures. Only care needed: `IsListableItem(itemId, flag)` signature takes an item id + flag `uint16`, so classification cases must use real item ids / flag bits, not invented ones. |
| **6.3** | No listing-package tests for the write operations the task named (`createListing`/`getByShopId` round-trip, `updateBundles` correct-vs-wrong version, `decrementDisplayOrderAfter`, `deleteByShopId`). These functions moved from `listing/provider.go` to `listing/administrator.go` (task 1.3), and only `listing/builder_test.go` exists. Behavior IS covered indirectly through `shop/processor_test.go` (`TestUpdateListing`, `TestRemoveListing_DisplayOrderCollapse`, `TestPurchaseBundle_SoldOut` exercises optimistic-lock `updateBundles`), but there are no direct package-level tests. | small | Low–medium. Needs the SQLite-in-memory + `RegisterTenantCallbacks` harness the shop tests already use; the file name in the task (`listing/provider_test.go`) is stale — writes now live in `administrator.go`, so a `listing/administrator_test.go` is the correct target. |

Everything else is either already implemented or no longer applicable — details below.

---

## Per-item disposition

### Phase 1 — Blocking Issues

- **1.1 — DONE-BY-SUPERSESSION.** `shop/administrator.go:9,19` hold `create()` / `update()`; `shop/provider.go` no longer defines them.
- **1.2 — DONE-BY-SUPERSESSION.** `shop/provider.go` is read-only (getById, getByCharacterId, getActiveByCharacterIdAndType, getByField, getAllOpen, getExpired, searchListingsByItemId — all SELECT).
- **1.3 — DONE-BY-SUPERSESSION.** `listing/administrator.go` holds all six writes: `createListing:10`, `deleteListing:20`, `updateBundles:30`, `deleteByShopId:46`, `decrementDisplayOrderAfter:56`, `updateListingFields:68`.
- **1.4 — DONE-BY-SUPERSESSION.** `listing/provider.go` is read-only (`getByShopId:14`, `getByShopIdAndDisplayOrder:25`, `countByShopId:39`, plus a new read `countByShopIds:55`).
- **1.5 — MOOT.** `listing/exports.go` does not exist in the current tree. Read/write separation is already achieved by the provider/administrator split (1.3/1.4), so there is nothing to "split" and no exports file to edit.
- **1.6 — DONE-BY-SUPERSESSION.** Ingress routing moved out of `atlas-ingress.yml` into a generated ConfigMap sourced from `deploy/shared/routes.conf:1` — `location ~ ^/api/merchants(/.*)?$` → `atlas-merchant:8080`.
- **1.7 — DONE-BY-SUPERSESSION.** `go build ./...` verified clean.
- **1.8 — UNVERIFIABLE.** Test suite not executed (see Summary note).

### Phase 2 — Subdomain Layering

- **2.1 — DONE-BY-SUPERSESSION.** `frederick/model.go:9` `ItemModel`, `:36` `MesoModel` — private fields, value-receiver getters.
- **2.2 — DONE-BY-SUPERSESSION.** `MakeItem` (`frederick/model.go:25`), `MakeMeso` (`:46`).
- **2.3 — DONE-BY-SUPERSESSION.** `frederick/administrator.go` holds `storeItems:13`, `storeMesos:41`, `clearItems:61`, `clearMesos:71`, `createNotification:81`, `clearNotifications:101` (signatures use the `EntityProvider` curried-`db` form rather than the literal `storeItems(db, tenantId, …)` shape, per current convention).
- **2.4 — DONE-BY-SUPERSESSION.** `frederick/processor.go:69-76` `GetItems`/`GetMesos` return `[]ItemModel` / `[]MesoModel` via `model.SliceMap`; provider is read-only.
- **2.5 — DONE-BY-SUPERSESSION.** Frederick retrieval now flows through `kafka/consumer/merchant/consumer.go:269` → `RetrieveFrederickAndEmit`; the consumer no longer touches entities.
- **2.6 — DONE-BY-SUPERSESSION.** `shop/processor.go:553` `storeToFrederick` calls `fp.StoreItems`/`fp.StoreMesos` with the new model/`StoredItem` types.
- **2.7 — DONE-BY-SUPERSESSION.** Build clean.
- **2.8 — DONE-BY-SUPERSESSION.** `message/model.go:9` immutable `Model` (id, shopId, characterId, content, sentAt) with getters.
- **2.9 — DONE-BY-SUPERSESSION.** `message/model.go:23` `Make(Entity)`.
- **2.10 — DONE-BY-SUPERSESSION.** `message/administrator.go:12` `create(tenantId, shopId, characterId, content)`.
- **2.11 — DONE-BY-SUPERSESSION.** `message/provider.go:10` `getByShopId` (ORDER BY sent_at ASC).
- **2.12 — DONE-BY-SUPERSESSION.** `message/processor.go:43` `GetMessages` returns `[]Model`; `SendMessage:37` delegates to `create`.
- **2.13 — DONE-BY-SUPERSESSION.** Build clean.
- **2.14 — UNVERIFIABLE.** Test suite not executed. (Note `message/processor_test.go` exists with 4 tests.)

### Phase 3 — Kafka Event Correctness

- **3.1 — MOOT.** The `ExitMaintenanceResult{Closed, CloseReason}` struct is unnecessary under the current design. `ExitMaintenance` now takes a `*message.Buffer` and emits the correct event itself; the auto-close signal never has to travel back to the consumer.
- **3.2 — MOOT.** Same reason. Auto-close is handled inside the method: `shop/processor.go:439-464` closes the shop when listing count is 0 and puts `StatusEventShopClosedProvider(..., CloseReasonEmpty)` on the buffer.
- **3.3 — MOOT.** The interface signature changed to the buffer pattern, not a result struct: `ExitMaintenance(mb *message.Buffer) func(shopId, characterId) error` (`shop/processor.go:55`).
- **3.4 — DONE-BY-SUPERSESSION.** Correct event emission is in place: `shop/processor.go:464` emits `StatusEventShopClosed` on auto-close, `:466` emits `StatusEventMaintenanceExited` otherwise. `kafka/consumer/merchant/consumer.go:127` just calls `ExitMaintenanceAndEmit`.
- **3.5 — DONE-BY-SUPERSESSION.** `kafka/consumer/character/consumer.go:54` calls `CloseShopAndEmit(..., CloseReasonDisconnect)`, which emits `StatusEventShopClosed` (`shop/processor.go:543`) per closed shop. (Logout also handles the maintenance case via `ExitMaintenanceAndEmit:58`.)
- **3.6 — MOOT.** No `producer.ProviderImpl` import is needed in the character consumer; emission now happens via the outbox inside the processor's `AndEmit` methods, not a producer created in the consumer.
- **3.7 — DONE-BY-SUPERSESSION.** Build clean.
- **3.8 — UNVERIFIABLE.** Test suite not executed.

### Phase 4 — REST & Resource Convention Alignment

- **4.1 — DONE-BY-SUPERSESSION.** `rest/handler.go:26` is now `var RegisterHandler = server.RegisterHandler` — the custom handler collapsed into a thin re-export/alias over `atlas-rest/server`.
- **4.2 — DONE-BY-SUPERSESSION.** `shop/resource.go:25` and `frederick/resource.go:17` use `rest.RegisterHandler(l)(si)` (the standard pattern).
- **4.3 — DONE-BY-SUPERSESSION.** `shop/resource.go:22` is `InitializeRoutes` (no `InitResource`).
- **4.4 — DONE-BY-SUPERSESSION.** `main.go:114` calls `shop.InitializeRoutes(GetServer())(db)` (and `:115` for frederick).
- **4.5 — DONE-BY-SUPERSESSION.** `shop/rest.go` has no `Extract()` (now purely JSON:API `RestModel`/`Transform*`); no dead `fmt` import.
- **4.6 — DONE-BY-SUPERSESSION.** `shop/state.go` holds the enums: `ShopType:8` and `State:15` (now type aliases to `libs/atlas-constants` `merchantconst`), `CloseReason:24` + constants `:27-33`.
- **4.7 — DONE-BY-SUPERSESSION.** `shop/model.go` defines only `Model` (`:11`); no enum type definitions remain.
- **4.8 — DONE-BY-SUPERSESSION.** Build clean.
- **4.9 — UNVERIFIABLE.** Test suite not executed.

### Phase 5 — Kafka AndEmit Pattern

- **5.1 — DONE-BY-SUPERSESSION.** `kafka/message/message.go:11` `Buffer`, `:22` `Buffer.Put`, `:43` `Emit`. (`EmitWithResult` was never added — result-returning operations capture the result via a closure variable inside `Emit`, e.g. `AddListingAndEmit`/`PurchaseBundleAndEmit`. The sub-ask for a distinct `EmitWithResult` is moot; the goal is met.)
- **5.2 — DONE-BY-SUPERSESSION.** `shop/ProcessorImpl` has a `producer` field (`shop/processor.go:133`), initialized in `NewProcessor` (`:142`).
- **5.3 — MOOT.** No separate `NewProcessorWithProducer` is needed; the producer is constructed unconditionally in `NewProcessor` (`shop/processor.go:142`), so REST and consumer paths share one constructor.
- **5.4 — DONE-BY-SUPERSESSION.** `AndEmit` variants on the interface: `shop/processor.go:69-79` (OpenShop, CloseShop, EnterMaintenance, ExitMaintenance, EnterShop, ExitShop, AddListing, RemoveListing, PurchaseBundle, SendMessage, RetrieveFrederick, plus CreateShop) — a superset of the planned five.
- **5.5 — DONE-BY-SUPERSESSION.** Implementations at `shop/processor.go:1124+`, wrapping operations in `message.Emit(outbox.EmitProvider(...))` inside a DB transaction (outbox-backed, exceeds the plan's in-memory buffer).
- **5.6 — DONE-BY-SUPERSESSION.** All `kafka/consumer/merchant/consumer.go` handlers call the `AndEmit` variants (lines 58, 76, 93, 110, 127, 145, 163, 198, 223, 240, 257, 269).
- **5.7 — DONE-BY-SUPERSESSION.** Consumer handlers are thin (parse ids → single `AndEmit` call → log error).
- **5.8 — DONE-BY-SUPERSESSION.** Build clean.
- **5.9 — UNVERIFIABLE.** Test suite not executed.

### Phase 6 — Documentation, Testing & Polish

- **6.1 — STILL-OPEN (partial).** `shop/validation_test.go` has `TestIsFreemarketRoom_ValidRooms/_InvalidRooms` and `TestManhattanDistance`, but the four `TestIsListableItem_*` cases are missing; `IsListableItem` still exists (`shop/validation.go:121`). Effort: trivial mechanical. Risk: low (pure function; use real item ids/flag bits).
- **6.2 — DONE-BY-SUPERSESSION.** `shop/processor_test.go` has an extensive state-machine suite (CreateShop happy/validation, OpenShop ±listings, CloseShop from Open/Maintenance/Draft + invalid-state, EnterMaintenance ±, ExitMaintenance reopen/close-when-empty/invalid, PurchaseBundle happy/sold-out/insufficient/zero, fee tiers, meso accumulation, GetShopForCharacter ownership, expiry). Supporting suites: `create_feedback_test.go`, `logout_policy_test.go`, `provider_search_test.go`, `provider_tenant_test.go`.
- **6.3 — STILL-OPEN.** No listing-package tests for the write ops (createListing round-trip, updateBundles right/wrong version, decrementDisplayOrderAfter, deleteByShopId). Only `listing/builder_test.go` exists. Effort: small. Risk: low–medium (needs the SQLite+tenant-callbacks harness; correct target is `listing/administrator_test.go`, not the stale `provider_test.go` name — writes moved to administrator.go). Partially covered indirectly by `shop/processor_test.go`.
- **6.4 — DONE-BY-SUPERSESSION.** Tests exist in ≥6 packages (shop, listing, message, frederick, searchcount, visitor), up from 2.
- **6.5 — DONE-BY-SUPERSESSION.** `services/atlas-merchant/README.md` has REST Endpoints, Kafka Commands Consumed, Kafka Events Produced, and Kafka Events Consumed tables; the four documentation links (`docs/domain.md`, `docs/kafka.md`, `docs/rest.md`, `docs/storage.md`) all resolve to existing files.
- **6.6 — DONE-BY-SUPERSESSION.** `services/atlas-merchant/.bruno/` exists with `bruno.json`, `collection.bru`, `environments/`, and the four request files (Get Merchants By Map, Get Merchant By Id, Get Merchant Listings, Get Character Merchants).
- **6.7 — MOOT.** `listing/exports.go` does not exist; there is no header to annotate. The cross-package concern (ARCH-006) is moot under the provider/administrator split.
- **6.8 — DONE-BY-SUPERSESSION.** Build clean.
- **6.9 — UNVERIFIABLE.** Test suite not executed.
- **6.10 — UNVERIFIABLE.** Re-running `/backend-audit atlas-merchant` requires dispatching the reviewer agent; not performed in this read-only assessment.
