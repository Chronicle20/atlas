# MTS Marketplace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a per-world player marketplace (`atlas-mts`) where characters list inventory items for cash currency, buy/bid, and take-home, with strict single-custody dupe-safety, across all five templated client versions.

**Architecture:** A new `atlas-mts` Go service owns `listed`/`holding` custody as first-class states and is the sole custodian for the middle of the item journey. Every cross-service item/currency move is a `transactionId`-idempotent saga (mirroring the cash-shop custody family); cancel/expire are atlas-mts-local DB transitions. atlas-channel wires version-parameterized serverbound handlers + the existing task-096 clientbound writers; atlas-tenants holds the economic knobs and per-version `operations` mode tables; atlas-ui surfaces config + a read-only browser.

**Tech Stack:** Go 1.25, GORM (Postgres + sqlite for tests), JSON:API (`api2go/jsonapi`), Kafka (`libs/atlas-kafka`), `libs/atlas-saga`, multi-tenant context (`libs/atlas-tenant`); Next.js/React/TypeScript + Zod for atlas-ui.

**Read `context.md` first** — it holds every file:line anchor this plan references.

---

## Conventions used by every Go task

- **Paths** are repo-relative to the worktree root `.worktrees/task-102-mts-marketplace/`.
- **Service root** for `atlas-mts` is `services/atlas-mts/atlas.com/mts/`. Per-task paths below are relative to that root unless prefixed otherwise.
- **TDD**: write the failing test, run it red, implement minimally, run it green, commit. Tests use the Builder + sqlite-in-memory pattern from `gachapon/processor_test.go` and `test/processor.go` — **never** create `*_testhelpers.go`.
- **Models are immutable**: private fields + getters + Builder (see `context.md`).
- **Run all Go test/build/vet from the module dir** (`services/atlas-mts/atlas.com/mts/`) unless a step says otherwise.
- **Commit** after every green step with a Conventional Commit message.

---

# PHASE 0 — Serverbound packet verification (HARD GATE)

> No task in Phases 1–9 that **decodes a serverbound MTS packet body** may be implemented until that packet × version cell promotes in `docs/packets/audits/STATUS.md`. Service skeleton, entities, models, sagas, and config (Phases 1–3, 8-config) do NOT decode packets and may proceed in parallel with Phase 0.

### Task 0.1: Verify standalone serverbound packets

**Mechanism:** dispatch the `packet-verifier` agent (one cell per packet × version) following `docs/packets/audits/VERIFYING_A_PACKET.md`. Batch per IDB; select the IDA instance matching each version (`select_instance(port)`).

- [ ] **Step 1: Verify `ENTER_MTS`** for gms_v83, v84, v87, v95, and jms_v185 (where the opcode exists). One byte fixture per version.
- [ ] **Step 2: Verify `ITC_STATUS_CHARGE`** (bodiless "open NX recharge" hook — confirm it is bodiless) per version.
- [ ] **Step 3: Verify `ITC_QUERY_CASH_REQUEST`** per version.
- [ ] **Step 4:** Confirm each cell promoted in `docs/packets/audits/STATUS.md`. Expected: green/verified cells, evidence pinned.

### Task 0.2: Verify every `ITC_OPERATION` mode arm

Per the dispatcher-family rule ([[feedback_dispatcher_mode_byte_is_false_pass]]), **each mode arm needs its own byte fixture** — enumerating mode bytes is a false pass. The three arms already in the scaffold (register-fixed-sale mode 2, sale-current mode 3, register-auction mode 0x12) are **re-checked, not assumed**.

- [ ] **Step 1: Verify each arm** (one fixture per arm × version): register-fixed-sale (2), sale-current (3), register-auction (0x12), buy, buy-now-on-auction, place-bid, set-zzim, buy-zzim, delete-zzim, view-wish, buy-wish, cancel-wish, register-wish, cancel-sale, move-ITC-purchase-LtoS (take-home), change-category, change-category-sub, change-page.
- [ ] **Step 2:** Record each verified read order in the promoted evidence record (the implementer of Phases 4–6 reads these, never invents).

### Task 0.3: Resolve the two design questions

- [ ] **Step 1 (§9.1):** Determine whether the verified clientbound set includes a **server-pushed** auction-state/outbid packet (a mode the server emits unsolicited). Inspect `MTS_OPERATION` cases 60–62.
- [ ] **Step 2 (§9.4):** Record the supported jms surface (which MTS opcodes/clientbound results exist).
- [ ] **Step 3: Append a decision note** to `design.md` recording the §9.1 outcome (live-push vs escrow-at-expiry) and the §9.4 jms surface.

```bash
git add docs/packets/ docs/tasks/task-102-mts-marketplace/design.md
git commit -m "verify(task-102): serverbound MTS/ITC packet cells + bidding-push determination"
```

**GATE:** Phases 4, 5, 6, and 8-channel are blocked until Tasks 0.1–0.3 are complete.

---

# PHASE 1 — `atlas-mts` service skeleton + data model + REST reads

> No packet decoding here — proceed in parallel with Phase 0. Mirror `atlas-gachapons` (REST) and `atlas-cashshop` (Kafka bootstrap).

### Task 1.1: Module scaffold + service registration

**Files:**
- Create: `services/atlas-mts/atlas.com/mts/go.mod`
- Create: `services/atlas-mts/atlas.com/mts/logger/init.go`
- Modify: `go.work` (repo root)
- Modify: `.github/config/services.json`
- Modify: `docker-bake.hcl`

- [ ] **Step 1: Create `go.mod`** mirroring `services/atlas-gachapons/atlas.com/gachapons/go.mod` — module name **`atlas-mts`**, `go 1.25.5`, the same `replace` directives for `libs/atlas-*`. Add the libs this service uses: `atlas-database`, `atlas-model`, `atlas-rest`, `atlas-service`, `atlas-tenant`, `atlas-tracing`, `atlas-kafka`, `atlas-saga`, `atlas-constants`, `github.com/google/uuid`, `github.com/sirupsen/logrus`, `gorm.io/gorm`, `gorm.io/driver/sqlite` (test).

- [ ] **Step 2: Create `logger/init.go`** copied from `services/atlas-gachapons/atlas.com/gachapons/logger/init.go` (rename package var to service name `atlas-mts`).

- [ ] **Step 3: Add to `go.work`** — append the line `	./services/atlas-mts/atlas.com/mts` in the same block as the other services (keep alphabetical/grouped ordering consistent with neighbors).

- [ ] **Step 4: Register in `.github/config/services.json`** — add an `atlas-mts` entry mirroring the `atlas-gachapons` entry shape.

- [ ] **Step 5: Register in `docker-bake.hcl`** — add `"atlas-mts"` to the hardcoded `go_services` list (HCL can't read JSON — [[reference_docker_bake_hand_synced]]).

- [ ] **Step 6: Verify the module resolves**

Run (from worktree root): `go work sync && (cd services/atlas-mts/atlas.com/mts && go build ./...)`
Expected: no module-resolution errors (empty package builds clean, or "no Go files" until first source lands — acceptable at this step).

- [ ] **Step 7: Commit**

```bash
git add go.work .github/config/services.json docker-bake.hcl services/atlas-mts
git commit -m "feat(atlas-mts): module scaffold + service registration"
```

### Task 1.2: Shared test harness (sqlite, tenant ctx) — no testhelpers files

**Files:**
- Create: `test/database.go`, `test/tenant.go`, `test/context.go`

- [ ] **Step 1: Copy the test harness** from `services/atlas-gachapons/atlas.com/gachapons/test/` (`database.go` providing `SetupTestDB(t, migrations...)`/`CleanupTestDB`, plus the `CreateTestContext()`/`TestTenantId` helpers). Adjust the package import path to `atlas-mts/...`. These are `package test` infrastructure (sqlite + tenant context), **not** `*_testhelpers.go` constructors — that is the sanctioned pattern. Do not add domain-processor constructors here yet; add them per domain as those land (Task 1.3+).

- [ ] **Step 2: Verify it compiles** — `go build ./test/...`. Expected: clean.

- [ ] **Step 3: Commit** — `git commit -am "test(atlas-mts): sqlite + tenant-context test harness"`

### Task 1.3: `Listing` domain — model + builder

**Files:**
- Create: `listing/model.go`, `listing/builder.go`
- Test: `listing/builder_test.go`

The `Listing` model carries the explicit item snapshot (template id, quantity, and the full equip stat block) plus sale/auction/state fields per `design.md §3.2`. Item-id/inventory types come from `libs/atlas-constants` ([CLAUDE.md DOM-21] — do not reinvent; use `item.Id` etc. where applicable).

- [ ] **Step 1: Write the failing builder test** `listing/builder_test.go`:

```go
package listing_test

import (
	"atlas-mts/listing"
	"testing"

	"github.com/google/uuid"
)

func TestBuilder_RequiresTenantAndWorld(t *testing.T) {
	_, err := listing.NewBuilder(uuid.Nil, 0, 1001).Build()
	if err == nil {
		t.Fatal("expected error when tenantId is nil")
	}
}

func TestBuilder_BuildsFixedListing(t *testing.T) {
	tid := uuid.New()
	m, err := listing.NewBuilder(tid, 0, 1001).
		SetSellerName("alice").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(110).
		SetCommissionRate(0.10).
		SetCategory("equip").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SaleType() != listing.SaleTypeFixed || m.State() != listing.StateActive {
		t.Fatalf("unexpected sale/state: %v/%v", m.SaleType(), m.State())
	}
	if m.ListValue() != 110 || m.SellerId() != 0 || m.WorldId() != 0 {
		t.Fatalf("unexpected fields")
	}
}
```

- [ ] **Step 2: Run red** — `go test ./listing/ -run TestBuilder -v`. Expected: FAIL (package `listing` not defined).

- [ ] **Step 3: Implement `listing/model.go`** — immutable struct + getters, mirroring `gachapon/model.go`. Fields per `design.md §3.2`: `id uuid.UUID`, `tenantId uuid.UUID`, `worldId world.Id`, `sellerId uint32`, `sellerName string`, `saleType SaleType`, `state State`, item snapshot (`templateId uint32`, `quantity uint32`, equip stat block — str/dex/int/luk/hp/mp/watk/matk/wdef/mdef/acc/avoid/hands/speed/jump/slots/level/itemLevel/itemExp/ringId/viciousCount/flags as typed fields), `listValue uint32`, `buyNowPrice *uint32`, `commissionRate float64`, `category string`, `subCategory string`, auction (`endsAt *time.Time`, `currentBid uint32`, `highBidderId uint32`, `minIncrement uint32`), `createdAt`, `updatedAt`. Define exported enum types:

```go
type SaleType string
const (
	SaleTypeFixed   SaleType = "fixed"
	SaleTypeAuction SaleType = "auction"
)

type State string
const (
	StateActive    State = "active"
	StateSold      State = "sold"
	StateCancelled State = "cancelled"
	StateExpired   State = "expired"
)
```

- [ ] **Step 4: Implement `listing/builder.go`** — mirror `gachapon/builder.go`: `NewBuilder(tenantId uuid.UUID, worldId world.Id, sellerId uint32) *Builder`, fluent `Set*`, `Build()` validating `tenantId != uuid.Nil`. (id is assigned at create time in the administrator, like gachapon's Uid.)

- [ ] **Step 5: Run green** — `go test ./listing/ -run TestBuilder -v`. Expected: PASS.

- [ ] **Step 6: Commit** — `git commit -am "feat(atlas-mts): Listing immutable model + builder"`

### Task 1.4: `Listing` entity + Migration + provider + administrator

**Files:**
- Create: `listing/entity.go`, `listing/provider.go`, `listing/administrator.go`
- Test: `listing/administrator_test.go`

- [ ] **Step 1: Write the failing test** `listing/administrator_test.go` asserting create→getById round-trips, cross-tenant same-world isolation, and that the three design indexes exist. Mirror `gachapon/processor_test.go` structure (sqlite via the Task 1.2 harness, two tenants, `Build()` + create + read back). Assert a created `Listing` is retrievable by id and scoped to its tenant.

- [ ] **Step 2: Run red** — `go test ./listing/ -run TestAdministrator -v`. Expected: FAIL.

- [ ] **Step 3: Implement `listing/entity.go`** — GORM entity with **surrogate UUID PK** + `(tenant_id, id)` unique index + **explicit name-keyed columns** (no JSON blob for the snapshot) per `design.md §3.2`, mirroring `gachapon/entity.go`'s tag style. Add the three indexes: `(tenant_id, world_id, state, category)`, `(tenant_id, seller_id, state)`, `(tenant_id, world_id, ends_at)`. Provide `Migration(db)` = `db.AutoMigrate(&entity{})` (no legacy-PK rewrite needed — this is a brand-new table, so the simpler gachapon `Migration` minus `migrateToSurrogatePK` is correct). `TableName()` returns `"listings"`.

- [ ] **Step 4: Implement `listing/provider.go`** — `getById(id string)`, `getAll()`, and a browse query provider `getBrowse(worldId, state, category, ...)` using `database.Query`/`SliceQuery`; `modelFromEntity(e)`.

- [ ] **Step 5: Implement `listing/administrator.go`** — `CreateListing(db, m)` (assigns `uuid.New()` id, explicit column struct), `UpdateState(db, id, from, to State)` as a conditional `UPDATE ... WHERE state = from` (returns rows-affected for the cancel-vs-buy race), and the auction-update mutators. Use `database.ExecuteTransaction` for multi-statement transitions.

- [ ] **Step 6: Run green** — `go test ./listing/ -v`. Expected: PASS.

- [ ] **Step 7: Commit** — `git commit -am "feat(atlas-mts): Listing entity, migration, provider, administrator"`

### Task 1.5: `Listing` processor

**Files:**
- Create: `listing/processor.go`
- Test: `listing/processor_test.go` + extend `test/processor.go` with `CreateListingProcessor`

- [ ] **Step 1: Add `CreateListingProcessor(t)`** to `test/processor.go` mirroring `CreateGachaponProcessor` (uses `SetupTestDB(t, listing.Migration)`).
- [ ] **Step 2: Write the failing processor test** covering Create, GetById, Browse (filter by world/state/category), and the conditional `TransitionState` (active→cancelled succeeds once; a second call returns 0 rows / not-active error).
- [ ] **Step 3: Run red** — `go test ./listing/ -run TestProcessor -v`. Expected: FAIL.
- [ ] **Step 4: Implement `listing/processor.go`** — `Interface` + `Impl` with `NewProcessor(l, ctx, db)`, `db.WithContext(p.ctx)`, mirroring `gachapon/processor.go`. Methods: `Create(m) error`, `GetById(id) (Model, error)`, `Browse(worldId, filters, page, pageSize) ([]Model, error)`, `TransitionState(id, from, to State) (bool, error)` (wraps the conditional update), plus the auction mutators. Follow the [[project memory]] processor convention: pure `Method(mb)` and side-effecting `MethodAndEmit()` once Kafka lands (Task 3.x) — for now REST-only methods are fine.
- [ ] **Step 5: Run green** — `go test ./listing/ -v`. Expected: PASS.
- [ ] **Step 6: Commit** — `git commit -am "feat(atlas-mts): Listing processor"`

### Task 1.6: `Holding`, `Bid`, `WishEntry` domains

Repeat the Task 1.3–1.5 model→builder→entity→provider→administrator→processor pattern for each, per `design.md §3.2`. Each is its own package with its own `Migration`.

- [ ] **Step 1: `holding/`** — model fields: `id`, `tenantId`, `worldId`, `ownerId`, item snapshot (same columns as Listing), `origin` (`purchased|unsold|cancelled|expired`), `createdAt`. Administrator supports soft-delete by id (for idempotent take-home). TDD as Task 1.4–1.5. Commit `feat(atlas-mts): Holding domain`.
- [ ] **Step 2: `bid/`** — model fields: `id`, `tenantId`, `listingId`, `bidderId`, `amount`, `escrowTxnId uuid.UUID`, `state` (`held|released|won`), `createdAt`. TDD. Commit `feat(atlas-mts): Bid domain`.
- [ ] **Step 3: `wish/`** — model fields: `id`, `tenantId`, `characterId`, `itemId`/criteria, `createdAt`. TDD. Commit `feat(atlas-mts): WishEntry domain`.

### Task 1.7: REST resources (reads + create stub) — JSON:API

**Files:**
- Create: `listing/rest.go`, `listing/resource.go`, `holding/rest.go`, `holding/resource.go`, `wish/rest.go`, `wish/resource.go`, `rest/handler.go`

- [ ] **Step 1: Create `rest/handler.go`** copied from `services/atlas-gachapons/atlas.com/gachapons/rest/handler.go` (the `HandlerDependency`/`HandlerContext`/`RegisterHandler`/`RegisterInputHandler[M]`/`ParseInput` infra). Add path parsers `ParseListingId`, `ParseHoldingId`, `ParseCharacterId`, `ParseWorldId` mirroring `ParseGachaponId`.
- [ ] **Step 2: Write `listing/rest.go`** — `RestModel` with JSON:API tags, `GetName()` returns `"listings"`, `GetID()/SetID()`, `Transform(m)`. Cover the browse/detail attributes from `prd.md §5.1`.
- [ ] **Step 3: Write `listing/resource.go`** — `InitResource(si)(db)` mirroring `gachapon/resource.go`. Routes per `prd.md §5.1`:
  - `GET /worlds/{worldId}/listings` → browse/search (query: category, subCategory, type, page, pageSize, itemId, sellerName, saleType).
  - `GET /worlds/{worldId}/listings/{listingId}` → detail.
  - **Register only the GET routes in this task.** The `POST /worlds/{worldId}/listings` (create → `TransferToMts` saga) and `DELETE /worlds/{worldId}/listings/{listingId}` (cancel) routes are added in Phase 4 (Tasks 4.1 and 4.2) because they initiate custody flows that don't exist until then. No stub, no 501 — the routes simply don't exist yet ([[feedback_no_todos_in_deliverables]]).
- [ ] **Step 4: Write `holding/resource.go`** and `wish/resource.go`** — the read endpoints (`GET /characters/{characterId}/mts/holding`, `GET /characters/{characterId}/mts/wishlist`) and wish CRUD (`POST`/`DELETE` wishlist — no saga, safe to land now). Wallet read endpoint `GET /characters/{characterId}/mts/wallet` is a passthrough added in Phase 5 (Task 5.5); omit now.
- [ ] **Step 5: Write `listing/resource_test.go`** (and holding/wish) — httptest-driven, asserting browse filters and JSON:API envelope shape. Use the Task 1.2 harness.
- [ ] **Step 6: Run** — `go test ./... -v`. Expected: PASS.
- [ ] **Step 7: Commit** — `git commit -am "feat(atlas-mts): JSON:API REST reads (listings browse/detail, holding, wishlist CRUD)"`

### Task 1.8: Config registry (lazy per-tenant cache)

**Files:**
- Create: `configuration/registry.go`, `configuration/requests.go`, `configuration/model.go`
- Test: `configuration/registry_test.go`

- [ ] **Step 1: Write the failing test** asserting `GetTenantConfig` returns defaults on a fetch miss and caches a fetched config (mirror cash-shop `configuration/registry.go` behavior; inject a stub fetcher).
- [ ] **Step 2: Run red.**
- [ ] **Step 3: Implement** the registry mirroring `atlas-cashshop` `configuration/registry.go` (`sync.RWMutex` double-check, default-on-miss). The config `Model` holds the `design.md §8` knobs (`listingFee`, `commissionRate`, `maxActiveListings`, `minLevel`, `auctionMinHours`, `auctionMaxHours`, `priceFloor`, `pageSize`, `minBidIncrement`), fetched from atlas-tenants `GET /tenants/{tenantId}/configurations/mts-configs` (Phase 8). Defaults per `context.md` "Economic knobs."
- [ ] **Step 4: Run green. Commit** — `git commit -am "feat(atlas-mts): lazy per-tenant config registry with defaults"`

### Task 1.9: `main.go` (REST + DB AutoMigrate) + k8s manifest

**Files:**
- Create: `main.go`, `deploy/k8s/base/atlas-mts.yaml` (repo root path)

- [ ] **Step 1: Write `main.go`** mirroring `services/atlas-gachapons/atlas.com/gachapons/main.go` — `serviceName = "atlas-mts"`, `database.Connect(l, database.SetMigrations(listing.Migration, holding.Migration, bid.Migration, wish.Migration))`, `server.New(l)...AddRouteInitializer(listing.InitResource(...))...` for each domain, tracing + teardown. (Kafka consumers/producers + ticker are added in Phases 3 and 7.)
- [ ] **Step 2: Create `deploy/k8s/base/atlas-mts.yaml`** mirroring `deploy/k8s/base/atlas-gachapons.yaml` — Deployment + Service, `DB_NAME=atlas-mts`, `atlas-env`. **No socket ports.** Add the manifest to the base `kustomization.yaml` resource list if that's how siblings are wired (check `atlas-gachapons` registration there).
- [ ] **Step 3: Build + vet** — `go build ./... && go vet ./...`. Expected: clean.
- [ ] **Step 4: Bake** (from worktree root): `docker buildx bake atlas-mts`. Expected: image builds (catches any missing Dockerfile COPY — none expected since no new lib).
- [ ] **Step 5: Commit** — `git commit -am "feat(atlas-mts): main bootstrap + k8s manifest"`

**PHASE 1 GATE:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `atlas-mts`; `docker buildx bake atlas-mts` green.

---

# PHASE 2 — Custody saga actions (shared lib + orchestrator)

> Mirror the cash-shop custody family. New actions: `TransferToMts`, `WithdrawFromMts` (composites), `AcceptToMtsListing`, `ReleaseFromMtsHolding` (atomic), `MtsSettlePurchase` (composite money-mover), `MtsBidEscrow` (single-step wallet). All touch `libs/atlas-saga` + `atlas-saga-orchestrator` + a new `atlas-mts` custody consumer.

### Task 2.1: Shared saga Action constants + payloads

**Files:**
- Modify: `libs/atlas-saga/model.go`, `libs/atlas-saga/payloads.go`, `libs/atlas-saga/unmarshal.go`
- Test: `libs/atlas-saga/unmarshal_test.go`

- [ ] **Step 1: Write the failing unmarshal test** in `libs/atlas-saga/unmarshal_test.go` asserting a `TransferToMts` step JSON unmarshals into `TransferToMtsPayload`, and likewise for `MtsSettlePurchasePayload`, `MtsBidEscrowPayload`, `ReleaseFromMtsHoldingPayload`, `AcceptToMtsListingPayload` (mirror the existing cash-shop cases in that file).
- [ ] **Step 2: Run red** — `(cd libs/atlas-saga && go test ./... -run TestUnmarshal -v)`. Expected: FAIL.
- [ ] **Step 3: Add Action constants** to `libs/atlas-saga/model.go` (in a new MTS block after the cash-shop block at line ~126):

```go
// MTS marketplace
TransferToMts        Action = "transfer_to_mts"
WithdrawFromMts      Action = "withdraw_from_mts"
AcceptToMtsListing   Action = "accept_to_mts_listing"
ReleaseFromMtsHolding Action = "release_from_mts_holding"
MtsSettlePurchase    Action = "mts_settle_purchase"
MtsMoveListingToHolding Action = "mts_move_listing_to_holding"
MtsBidEscrow         Action = "mts_bid_escrow"
```

Add a `SagaType`: `MtsOperation Type = "mts_operation"` near line 19.

- [ ] **Step 4: Add payload structs** to `libs/atlas-saga/payloads.go` mirroring `TransferToCashShopPayload` (line 518) and `ReleaseFromCharacterPayload` (line 540):

```go
// TransferToMtsPayload — expanded into release_from_character + accept_to_mts_listing.
type TransferToMtsPayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`
	CharacterId         uint32    `json:"characterId"`
	WorldId             world.Id  `json:"worldId"`
	SourceInventoryType byte      `json:"sourceInventoryType"`
	AssetId             uint32    `json:"assetId"`
	Quantity            uint32    `json:"quantity"`
	ListingId           uuid.UUID `json:"listingId"`   // pre-allocated by initiator
}

// WithdrawFromMtsPayload — expanded into release_from_mts_holding + accept_to_character.
type WithdrawFromMtsPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	WorldId       world.Id  `json:"worldId"`
	HoldingId     uuid.UUID `json:"holdingId"`
	InventoryType byte      `json:"inventoryType"`
}

// AcceptToMtsListingPayload (atomic, dispatched to atlas-mts custody consumer).
type AcceptToMtsListingPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ListingId     uuid.UUID `json:"listingId"`
	// asset snapshot fields populated during expansion
}

// ReleaseFromMtsHoldingPayload (atomic, dispatched to atlas-mts custody consumer).
type ReleaseFromMtsHoldingPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	HoldingId     uuid.UUID `json:"holdingId"`
}

// MtsSettlePurchasePayload (composite money-mover): debit buyer prepaid, credit seller points, move custody.
type MtsSettlePurchasePayload struct {
	TransactionId  uuid.UUID `json:"transactionId"`
	ListingId      uuid.UUID `json:"listingId"`
	BuyerId        uint32    `json:"buyerId"`
	BuyerAccountId uint32    `json:"buyerAccountId"`
	SellerId       uint32    `json:"sellerId"`
	SellerAccountId uint32   `json:"sellerAccountId"`
	MarkedUpPrice  int32     `json:"markedUpPrice"`
	ListValue      int32     `json:"listValue"`
}

// MtsBidEscrowPayload (single-step wallet hold).
type MtsBidEscrowPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ListingId     uuid.UUID `json:"listingId"`
	BidderId      uint32    `json:"bidderId"`
	BidderAccountId uint32  `json:"bidderAccountId"`
	Amount        int32     `json:"amount"` // negative to hold, positive to release
}
```

- [ ] **Step 5: Register the new payload types** in `libs/atlas-saga/unmarshal.go` (the Action→payload switch) and in `libs/atlas-saga/validation.go` if it enumerates actions.
- [ ] **Step 6: Run green** — `(cd libs/atlas-saga && go test ./... -v)`. Expected: PASS.
- [ ] **Step 7: Commit** — `git commit -am "feat(atlas-saga): MTS custody + settlement saga actions and payloads"`

### Task 2.2: atlas-mts custody Kafka consumer + events

**Files:**
- Create: `kafka/message/message.go`, `kafka/message/custody/kafka.go`, `kafka/consumer/consumer.go`, `kafka/consumer/custody/consumer.go`, `kafka/producer/producer.go`, `kafka/producer/custody/producer.go`
- Test: `kafka/consumer/custody/consumer_test.go`

- [ ] **Step 1: Create the generic envelope** `kafka/message/message.go` (the `Buffer`/`Emit`/`EmitWithResult` pattern) copied from `services/atlas-cashshop/atlas.com/cashshop/kafka/message/message.go`.
- [ ] **Step 2: Define custody messages** `kafka/message/custody/kafka.go`: topic constants `EnvCommandTopic = "COMMAND_TOPIC_MTS_CUSTODY"`, `EnvStatusEventTopic = "EVENT_TOPIC_MTS_CUSTODY_STATUS"`; command types `AcceptToMtsListing` / `ReleaseFromMtsHolding`; the `Command` envelope (carrying `TransactionId`) and a `StatusEvent[E]` ack body carrying `transactionId` + result. Model on `kafka/message/wallet/kafka.go`.
- [ ] **Step 3: Write the failing consumer test** asserting: an `AcceptToMtsListing` command flips the pre-allocated listing row to `active` and emits an ack with the same `transactionId`; a **replayed** delivery is a no-op (row already active for that txn) and re-acks. A `ReleaseFromMtsHolding` soft-deletes the holding and re-ack on replay (idempotent — [[design §4.2]]).
- [ ] **Step 4: Run red.**
- [ ] **Step 5: Implement** the consumer (`InitConsumers(l)(cmf)(groupId)` + `InitHandlers(l)(db)(register)` curried registration, mirroring `atlas-cashshop` `kafka/consumer/wallet/consumer.go`) and producer (`ProviderImpl(l)(ctx)(token)`). The handler performs the row transition inside one local DB tx and emits the ack via `message.Emit`.
- [ ] **Step 6: Run green. Commit** — `git commit -am "feat(atlas-mts): custody command consumer + status events (idempotent)"`

### Task 2.3: Orchestrator — handlers, expansion, acceptance, compensation

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`, `saga/event_acceptance.go`, `saga/compensator.go`, `saga/processor.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/mts/processor.go`
- Test: `saga/mts_expansion_test.go`, `saga/mts_integration_test.go`

- [ ] **Step 1: Create the MTS dispatch processor** `mts/processor.go` mirroring `cashshop/processor.go` — `AcceptToMtsListingAndEmit`, `ReleaseFromMtsHoldingAndEmit`, `MoveListingToHoldingAndEmit` dispatch commands to `COMMAND_TOPIC_MTS_CUSTODY`. Settlement currency steps reuse the existing `cashshop` `AwardCurrencyAndEmit` (prepaid debit / points credit). The listing fee reuses `AwardMesos` (negative amount).
- [ ] **Step 2: Write the failing expansion test** `saga/mts_expansion_test.go`: `TransferToMts` expands to `[release_from_character, accept_to_mts_listing]`; `WithdrawFromMts` expands to `[release_from_mts_holding, accept_to_character]`; `MtsSettlePurchase` expands to `[award_currency(buyer,prepaid,−markedUp), award_currency(seller,points,+listValue), mts_move_listing_to_holding]` in that order (debit-first — [[design §12-E]]). Mirror the cash-shop expansion test assertions.
- [ ] **Step 3: Run red.**
- [ ] **Step 4: Implement expansion** in `saga/processor.go` — add `expandTransferToMts`, `expandWithdrawFromMts`, `expandMtsSettlePurchase` mirroring `expandTransferToCashShop` (~line 1217). For `TransferToMts`, look up the source asset and populate the `ReleaseFromCharacterPayload` + `AcceptToMtsListingPayload` snapshot, exactly as the cash-shop expansion populates its accept payload.
- [ ] **Step 5: Add handlers** to `saga/handler.go` `GetHandler` switch: `AcceptToMtsListing`→`h.mtsP.AcceptToMtsListingAndEmit`, `ReleaseFromMtsHolding`→`h.mtsP.ReleaseFromMtsHoldingAndEmit`, `MtsMoveListingToHolding`→`h.mtsP.MoveListingToHoldingAndEmit`, `MtsBidEscrow`→`h.cashshopP.AwardCurrencyAndEmit` (reuses wallet). Composite `TransferToMts`/`WithdrawFromMts`/`MtsSettlePurchase` are expanded, not directly handled.
- [ ] **Step 6: Add event-acceptance rows** to `saga/event_acceptance.go` `acceptanceTable`: composites → `{}`; `AcceptToMtsListing`/`ReleaseFromMtsHolding`/`MtsMoveListingToHolding` → new `EventKindMtsCustody*` kinds (define them; the orchestrator consumes `EVENT_TOPIC_MTS_CUSTODY_STATUS`); `MtsBidEscrow` → `EventKindCashShopWalletUpdated` (reused). Add a consumer subscription for `EVENT_TOPIC_MTS_CUSTODY_STATUS` mirroring the cash-compartment status consumer.
- [ ] **Step 7: Add compensators** to `saga/compensator.go`: inverse of `AcceptToMtsListing` is re-grant to character (`AcceptToCharacter`); inverse of `ReleaseFromMtsHolding` is re-create the holding; settlement compensation re-credits buyer, debits seller, restores listing to `active`. Idempotent.
- [ ] **Step 8: Set timeout explicitly** in the saga builders for MTS sagas: `base + perStep*N` (record N: TransferToMts N=2, MtsSettlePurchase N=3, WithdrawFromMts N=2, MtsBidEscrow N=1) — never the default ([[bug_preset_creation_saga_flat_timeout]]).
- [ ] **Step 9: Write the failing integration test** `saga/mts_integration_test.go` modeled on `preset_integration_test.go`: drive a `MtsSettlePurchase` to a forced mid-saga failure and assert the reverse-walk compensation re-credits buyer, debits seller, and restores the listing — single-custody invariant holds.
- [ ] **Step 10: Run green** — `(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... -v)`. Expected: PASS.
- [ ] **Step 11: Bake** — `docker buildx bake atlas-saga-orchestrator atlas-mts`. Expected: green.
- [ ] **Step 12: Commit** — `git commit -am "feat(saga-orchestrator): MTS custody + settlement actions, expansion, compensation"`

**PHASE 2 GATE:** orchestrator + atlas-mts + atlas-saga modules: `go test -race`, `go vet`, `go build`, bake all green.

---

# PHASE 3 — atlas-mts command consumer + status events

### Task 3.1: `COMMAND_TOPIC_MTS` consumer + `EVENT_TOPIC_MTS_STATUS` producer

**Files:**
- Create: `kafka/message/mts/kafka.go`, `kafka/consumer/mts/consumer.go`, `kafka/producer/mts/producer.go`
- Modify: `main.go` (register consumers + producer teardown)
- Test: `kafka/consumer/mts/consumer_test.go`

- [ ] **Step 1: Define `kafka/message/mts/kafka.go`** — `EnvCommandTopic = "COMMAND_TOPIC_MTS"`, `EnvStatusEventTopic = "EVENT_TOPIC_MTS_STATUS"`; command types CreateListing, CancelListing, PlaceBid, Buy, TakeHome, ExpireListing, RegisterWish, RemoveWish; event types ListingCreated, ListingCancelled, BidPlaced, Outbid, ListingSold, ListingExpired, ItemMovedToHolding, ItemTakenHome, WishAdded, WishRemoved (all carry `transactionId`, `worldId`). Envelopes use the generic `Command`/`StatusEvent[E]`.
- [ ] **Step 2: Write the failing consumer test** asserting a `CancelListing` command runs the local active→holding(seller) transition and emits `ListingCancelled`; a `RegisterWish`/`RemoveWish` updates the wish table and emits.
- [ ] **Step 3: Run red.**
- [ ] **Step 4: Implement** the consumer + producer mirroring Task 2.2's structure. Wire `main.go` to register them (`consumer.GetManager().AddConsumer`, `InitConsumers`/`InitHandlers`, producer teardown) — mirror `atlas-cashshop/main.go`.
- [ ] **Step 5: Run green. Bake. Commit** — `git commit -am "feat(atlas-mts): MTS command consumer + status events"`

**PHASE 3 GATE:** atlas-mts module green (test/vet/build/bake).

---

# PHASE 4 — List + cancel/expire + take-home (custody flows)

> **Blocked on Phase 0** for the channel handlers (Task 4.5); the saga-side and REST-side pieces (4.1–4.4) are not.

### Task 4.1: List flow — `TransferToMts` initiation + fee + caps + floor

**Files:**
- Modify: `listing/processor.go` (validation + saga initiation), `listing/resource.go` (POST route)
- Test: `listing/list_flow_test.go`

- [ ] **Step 1: Write the failing test** asserting: list rejects `listValue < priceFloor (110)`; rejects when the seller already has `maxActiveListings` active; auction rejects duration outside 24–168h / non-1h step; a valid fixed list pre-allocates a `listingId`, charges the listing fee via `AwardMesos(-listingFee)`, and emits a `TransferToMts` saga.
- [ ] **Step 2: Run red.**
- [ ] **Step 3: Implement** the validation (server-authoritative floor/cap/level/duration from the config registry) and saga construction in the processor: build a `saga.Saga{SagaType: saga.MtsOperation, Steps: [AwardMesos(-fee), TransferToMts{ListingId: preAllocated}]}` and `Create` it (mirror `storage_operation.go handleRetrieveAsset`). The listing row is created in `active` only on the custody consumer's `AcceptToMtsListing` (Task 2.2) — item leaves inventory first.
- [ ] **Step 4: Wire the `POST /worlds/{worldId}/listings` route** in `listing/resource.go` to call the processor (JSON:API envelope — [[bug_ui_jsonapi_envelope_required_for_input_handlers]]).
- [ ] **Step 5: Run green. Commit** — `git commit -am "feat(atlas-mts): list flow (TransferToMts + fee + floor/cap/duration validation)"`

### Task 4.2: Cancel — atlas-mts-local active→holding(seller)

- [ ] **Step 1: Write the failing test** asserting cancel of an `active` listing moves the row to seller `holding` in one DB tx and emits `ListingCancelled`; cancel of a non-active listing is a clean no-op/failure (conditional `WHERE state='active'`); a concurrent settle wins exactly one (cancel-vs-buy — [[design §4.2]]).
- [ ] **Step 2: Run red. Step 3: Implement** the local transition in the processor (conditional state update + holding insert in one `ExecuteTransaction`). **Step 4: Wire `DELETE /worlds/{worldId}/listings/{listingId}`** (seller-only owner check — [[prd §8.4]]). **Step 5: Run green. Commit** — `feat(atlas-mts): cancel-sale local transition (race-safe)`.

### Task 4.3: Take-home — `WithdrawFromMts` (idempotent)

- [ ] **Step 1: Write the failing test** asserting take-home emits `WithdrawFromMts` and a **replayed** take-home is a no-op (holding already released for that txn — [[design §4.2]]). **Step 2: red. Step 3: Implement** the processor method + `POST /characters/{characterId}/mts/holding/{holdingId}/take-home` (attributes: inventoryType, slot). **Step 4: green. Commit** — `feat(atlas-mts): take-home (WithdrawFromMts, idempotent)`.

### Task 4.4: Expiration ticker (DB-driven sweep)

**Files:**
- Create: `task/periodic.go`
- Modify: `main.go`
- Test: `task/periodic_test.go`

- [ ] **Step 1: Write the failing test** asserting the sweep selects `(tenant_id, world_id, ends_at < now, state='active')` and runs the cancel-equivalent transition to seller holding, and that the swept count is returned/logged (bounded, never silently truncated — [[feedback_no_todos_in_deliverables]] / NFR 8.3).
- [ ] **Step 2: Run red.**
- [ ] **Step 3: Implement `task/periodic.go`** mirroring `atlas-asset-expiration/task/periodic.go` (`PeriodicTask` + `time.Ticker` + `stopCh` + `sync.WaitGroup`, env interval). **DB-driven**: enumerate active tenants (from the config registry / tenant list), reconstruct tenant context per iteration (`tenant.Create` + `tenant.WithContext`), query each tenant's expired listings, run the local expire transition. Register in `main.go` (`task.NewPeriodicTask(...).Start()` + `tdm.TeardownFunc(task.Stop)`).
- [ ] **Step 4: Run green. Bake. Commit** — `git commit -am "feat(atlas-mts): DB-driven expiration ticker"`

### Task 4.5: Channel — `ENTER_MTS` migration + list/cancel/take-home handler arms

> **Blocked on Phase 0.** Decode read orders come from the promoted fixtures — do not invent.

**Files:**
- Create: `services/atlas-channel/.../socket/handler/mts_entry.go`, `.../socket/handler/itc_operation.go`

- [ ] **Step 1: Implement `MtsEntryHandleFunc`** (`ENTER_MTS`) mirroring `cash_shop_entry.go CashShopEntryHandleFunc`: gate on min level (config) + map/event eligibility; save character; leave channel/map; mark entered; announce initial browse page + character's active listings + holding + wallet via the task-096 `MTS_OPERATION2` / `MTS_OPERATION` writers.
- [ ] **Step 2: Implement the `ITC_OPERATION` mode dispatcher** `itc_operation.go` mirroring `MessengerOperationHandleFunc` + `isMessengerShopOperation` — resolve the sub-op from `options["operations"][KEY]`. Implement the list (modes 2/3/0x12), cancel-sale, and take-home (move-ITC-purchase-LtoS) arms. **Each arm decodes its body using the Phase-0-verified read order** for that mode × version; each arm initiates the corresponding saga via `channel/saga` `Create(...)` (→ `COMMAND_TOPIC_SAGA`).
- [ ] **Step 3:** Byte-fixture each implemented serverbound arm (the verification deliverable; one fixture per arm, per the dispatcher-family rule).
- [ ] **Step 4: Bake** `atlas-channel`. **Commit** — `feat(atlas-channel): ENTER_MTS migration + ITC_OPERATION list/cancel/take-home arms`.

---

# PHASE 5 — Buy + settlement + wallet query

> **Blocked on Phase 0** for the channel arms.

### Task 5.1: Buy / buy-now — `MtsSettlePurchase`

- [ ] **Step 1: Write the failing test** asserting buy validates buyer NX Prepaid ≥ `listValue × (1 + commission)` (read from wallet), then emits `MtsSettlePurchase`; item lands in buyer holding, never inventory; commission is never credited (sink). Assert the debit-first ordering ([[design §12-E]]). **Step 2: red. Step 3: Implement** the processor method. **Step 4: green. Commit** — `feat(atlas-mts): buy/buy-now settlement (MtsSettlePurchase)`.

### Task 5.2: Dupe-safety suite (acceptance-critical)

**Files:**
- Test: `listing/dupe_safety_test.go` (+ orchestrator-side where the saga crosses)

- [ ] **Step 1: Write the failing tests** (NFR 8.1) — each asserts the single-custody invariant (exactly one copy, currency balanced) after compensation/replay: **crash-mid-list**, **grant-before-debit**, **double-grant replay**, **cancel-racing-purchase**, **take-home replay**. Model on `preset_integration_test.go` reverse-walk assertions.
- [ ] **Step 2: Run red. Step 3:** Fix any invariant violations surfaced (this is the risk core). **Step 4: Run green. Commit** — `test(atlas-mts): dupe-safety suite (crash/grant/replay/race)`.

### Task 5.3: Channel buy arm + wallet query

- [ ] **Step 1: Implement** the `ITC_OPERATION` buy / buy-now-on-auction arms (Phase-0 read orders) and the `ITC_QUERY_CASH_REQUEST` handler → read two-bucket wallet (`Prepaid()`/`Points()`) → `MTS_OPERATION2` (2× i32). `ITC_STATUS_CHARGE` re-opens the existing NX recharge hook (bodiless; `NoOpValidator`). **Step 2:** Byte-fixture each arm. **Step 3: Add `GET /characters/{characterId}/mts/wallet`** passthrough in `holding`/`wallet` resource. **Step 4: Bake `atlas-channel`. Commit** — `feat(atlas-channel): ITC buy arm + wallet query (ITC_QUERY_CASH_REQUEST/ITC_STATUS_CHARGE)`.

---

# PHASE 6 — Auction + bidding

> Path chosen by Phase 0 (Task 0.3). The custody/settlement core is identical to buy-now; only the notification edge differs.

### Task 6.1: Bid escrow + outbid release + settle-at-expiry

- [ ] **Step 1: Write the failing test** asserting: a bid above `currentBid + minIncrement` emits `MtsBidEscrow` (`AdjustCurrency(bidder, prepaid, −bid)`), records the `Bid` `held`, updates `currentBid`/`highBidder` under the listing row lock; an outbid releases the prior escrow (`+amount`, mark `released`); settle-at-expiry credits seller points, marks bid `won`, moves custody to winner holding; no-bids returns the listing to seller holding (the expire path). Escrow the **marked-up** amount at bid time ([[design §5.6]]).
- [ ] **Step 2: red. Step 3: Implement** bid in the processor + the `PlaceBid` command + the ticker's settle-at-expiry branch (extend Task 4.4). **Step 4: green. Commit** — `feat(atlas-mts): auction bidding escrow + settle-at-expiry`.

### Task 6.2: Channel bid arm + (conditional) live outbid push

- [ ] **Step 1: Implement** the `ITC_OPERATION` place-bid / buy-now-on-auction arms (Phase-0 read orders) + byte fixtures.
- [ ] **Step 2 (escrow path — default):** no server push; bidder gets no live toast. Done.
- [ ] **Step 2-alt (live path — ONLY if Task 0.3 found a server-push packet):** on outbid, emit the verified `MTS_OPERATION` push mode to the prior bidder; anti-snipe window-extension is a config knob, **off by default**. Do **not** build this path if Phase 0 did not verify a push packet ([[feedback_dispatcher_mode_byte_is_false_pass]] / [[design §12-D]]).
- [ ] **Step 3: Bake `atlas-channel`. Commit** — `feat(atlas-channel): ITC bid arm (+ live outbid push iff Phase 0 unlocked it)`.

---

# PHASE 7 — Wish-list (zzim)

### Task 7.1: Wish CRUD + buy-from-wish + channel arms

- [ ] **Step 1: Write the failing test** for wish add/view/remove/register and buy-from-wish (routes into the buy flow). No saga for add/view/remove. This is the **only** saved-items mechanism — no "cart" ([[research-scaffold §1]], [[design §5.7]]). **Step 2: red. Step 3: Implement** processor + REST (the wish CRUD landed read-side in Task 1.7; add buy-from-wish). **Step 4:** Implement the `ITC_OPERATION` zzim/wish arms in channel (Phase-0 read orders) + byte fixtures. **Step 5: green. Bake. Commit** — `feat(atlas-mts+channel): wish-list (zzim) CRUD + buy-from-wish`.

---

# PHASE 8 — atlas-tenants config resource + per-version socket seeds

### Task 8.1: `mts-configs` JSONB resource

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go`, `processor.go`, `resource.go`, `kafka.go`, `provider.go`, `seed.go`, `mock/processor.go`, `rest/handler.go`
- Test: the existing configuration test suite + a new `mts_config` case

Mirror the **routes/vessels** resource exactly (see `context.md`). The generic `Model`/`Entity`/`administrator` need **no change**.

- [ ] **Step 1: Write the failing test** asserting create/get/update/seed of an `mts-configs` resource round-trips through the JSONB `configurations` table (mirror an existing routes test).
- [ ] **Step 2: red. Step 3: Implement** each touch-point mirroring `routes`:
  - `rest.go`: `MtsConfigRestModel` (the §8 knobs) + `TransformMtsConfig`/`ExtractMtsConfig` + `CreateMtsConfigJsonData`.
  - `processor.go`: CRUD + provider + `SeedMtsConfigs` (interface + impl).
  - `resource.go`: 6 handlers + routes `/tenants/{tenantId}/configurations/mts-configs[...]` + `/seed`.
  - `kafka.go`: `EventTypeMtsConfig{Created,Updated,Deleted}` + `CreateMtsConfigStatusEventProvider`.
  - `provider.go`: `GetMtsConfigByIdProvider`, `GetAllMtsConfigsProvider`.
  - `seed.go`: path constant + getter + `LoadMtsConfigFiles`.
  - `rest/handler.go`: `ParseMtsConfigId`.
  - `mock/processor.go`: add the new Func fields + delegations.
- [ ] **Step 4: green. Bake `atlas-tenants`. Commit** — `feat(atlas-tenants): mts-configs JSONB resource`.

### Task 8.2: Per-version socket opcode + `operations` mode-table seeds

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`, `_gms_84_1.json`, `_gms_87_1.json`, `_gms_95_1.json`, `_jms_185_1.json`

- [ ] **Step 1: For each of the five templates**, add the four serverbound MTS handler entries (`ENTER_MTS`, `ITC_STATUS_CHARGE`, `ITC_QUERY_CASH_REQUEST`, `ITC_OPERATION`) to `socket.handlers[]` with the **correct per-version opcode** (research-scaffold §4; v84 + jms from Phase 0) and a **validator on every entry** (`LoggedInValidator`, except `NoOpValidator` on bodiless `ITC_STATUS_CHARGE`) — a validator-less entry is silently dropped ([[bug_socket_handler_missing_validator_silently_dropped]]). Confirm the task-096 `MTS_OPERATION`/`MTS_OPERATION2` writer entries exist in `socket.writers[]`; add if missing.
- [ ] **Step 2: Add the per-version `operations` mode table** to the `ITC_OPERATION` handler `options` for each version, populated from **that version's** dispatcher switch (IDA-verified in Phase 0) — **not copied from v83** (modes are version-dependent; a missing/wrong table makes `ResolveCode` return 99 → client crash — [[bug_operations_mode_tables_missing_v87_v95_jms]]).
- [ ] **Step 3: Verify no opcode collisions** within each version (grep the template for the chosen opcodes).
- [ ] **Step 4: Commit** — `feat(atlas-configurations): MTS socket handler/writer + operations mode tables (5 versions)`.

### Task 8.3: Rollout-checklist note (operational, not code)

- [ ] **Step 1: Document** in `docs/tasks/task-102-mts-marketplace/` a short rollout note: existing tenants must have the live config patched + channel restarted to pick up the new handler/writer opcodes and `operations` tables ([[bug_new_opcodes_not_in_live_tenant_config]]). Commit — `docs(task-102): MTS live-config rollout checklist`.

---

# PHASE 9 — atlas-ui

> `npm run build` type-checks tests too — update test call sites in the same commit; gate on build+test + no-new-lint-errors ([[reference_atlas_ui_build_typechecks_tests]], [[reference_atlas_ui_npm_nvm_and_lint_baseline]]; source nvm 22 first).

### Task 9.1: `mts-config` service client + Zod schema

- [ ] **Step 1: Write the failing test** for the `mts-config` JSON:API service module (fetch + PATCH the §8 knobs; POST/PATCH use the `{data:{type,attributes}}` envelope — [[bug_ui_jsonapi_envelope_required_for_input_handlers]]). **Step 2: red. Step 3: Implement** the service module + Zod schema mirroring an existing tenant-config service. **Step 4: green. Commit** — `feat(atlas-ui): mts-config service client + Zod schema`.

### Task 9.2: Tenant config page (react-hook-form + Zod)

- [ ] **Step 1:** Implement the config page (react-hook-form + Zod over the knobs) mirroring an existing tenant-config page; update any test call sites the build type-checks. **Step 2: build+test green. Commit** — `feat(atlas-ui): MTS tenant config page`.

### Task 9.3: Read-only listings browser

- [ ] **Step 1:** Implement a per-world listings browser (search/paginate) over `atlas-mts` REST `GET /worlds/{worldId}/listings`. **Step 2: build+test green. Commit** — `feat(atlas-ui): MTS listings browser (read-only)`.

---

# PHASE 10 — Final verification + review

### Task 10.1: Full verification gates (from worktree root)

- [ ] **Step 1:** `go test -race ./...` clean in every changed module (atlas-mts, atlas-saga, atlas-saga-orchestrator, atlas-channel, atlas-tenants).
- [ ] **Step 2:** `go vet ./...` clean in every changed module.
- [ ] **Step 3:** `go build ./...` clean in every changed service.
- [ ] **Step 4:** `docker buildx bake atlas-mts atlas-saga-orchestrator atlas-channel atlas-tenants` green.
- [ ] **Step 5:** `tools/redis-key-guard.sh` clean (run with `GOWORK=off` per [[reference_rediskeyguard_invariant]]).
- [ ] **Step 6:** atlas-ui `npm run build` + `npm test` green, no new lint errors.
- [ ] **Step 7:** Confirm all Phase-0 matrix cells still promoted in `docs/packets/audits/STATUS.md`.

### Task 10.2: Code review before PR

- [ ] **Step 1:** Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` + `frontend-guidelines-reviewer`). Address findings on this branch — do NOT fork ([[feedback_one_worktree_per_task]]). **Then** open the PR.

---

## Acceptance-criteria coverage (from `prd.md §10`)

| PRD AC | Plan task(s) |
|---|---|
| Phase 0 packets byte-verified, each mode arm fixtured | 0.1, 0.2 |
| §9.1 real-time-bidding resolved & recorded | 0.3 |
| `atlas-mts` service (models/processors/REST/Kafka/ticker), registered, bakes | 1.1–1.9, 3.1, 4.4 |
| List (fixed+auction), item leaves inventory atomically, fee/cap/floor | 4.1, 4.5 |
| Buy-now + bid; settlement (debit buyer / credit seller / commission sink / holding); one saga + compensation | 5.1, 6.1, 2.3 |
| Take-home idempotent; replay no-op | 4.3, 5.2 |
| Cancel + expiration return to seller holding; cancel can't race | 4.2, 4.4, 5.2 |
| Wish-list add/view/remove/buy | 7.1 |
| Economic knobs tenant-configurable + atlas-ui; listings browser | 8.1, 9.1–9.3 |
| All five versions; opcodes/modes from config; every handler has a validator | 8.2, 4.5, 5.3, 6.2 |
| Dupe-safety suite passes | 5.2 |
| All verification gates clean | 10.1 |
```
