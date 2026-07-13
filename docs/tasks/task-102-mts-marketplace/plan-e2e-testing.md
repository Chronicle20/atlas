# MTS E2E Test Endpoints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Env-gated test endpoints in `atlas-mts` (seed / expire / sweep / simulated purchase / simulated bid) plus an E2E playbook, so time-dependent and two-actor MTS flows are verifiable in minutes against a deployed env with the real client.

**Architecture:** One new package `testsupport/` in the atlas-mts module holding the five REST routes under a `/test` prefix, registered by `main.go` only when `MTS_TEST_ROUTES_ENABLED=true`. Seeding writes rows through the existing `listing` administrator (real ITC serials via `CreateListing`); expire is a new conditional `listing.BackdateEndsAt`; sweep calls the production `task.Sweep`; purchase/bid emit the byte-identical Kafka commands the channel emits, consumed by the service's own consumer (full saga fidelity). No ingress route is added — access is port-forward only.

**Tech Stack:** Go, gorilla/mux + api2go JSON:API (existing `rest` package infra), GORM/sqlite tests via the `test` harness, atlas-kafka producer.

**Design doc:** `docs/tasks/task-102-mts-marketplace/design-e2e-testing.md` (approved).

## Global Constraints

- All work happens in the task-102 worktree `.worktrees/task-102-mts-marketplace/`, module `services/atlas-mts/atlas.com/mts` (import prefix `atlas-mts/`), on branch `task-102-mts-marketplace`.
- Env flag name: `MTS_TEST_ROUTES_ENABLED`, exact string compare to `"true"`.
- Routes live under `/test` (full path `/api/test/...` behind the service's `/api/` base path). Do NOT add them to `deploy/shared/routes.conf`.
- Seed cap: at most **200** listings per call → 400 above that.
- Builder-pattern test setup only; no `*_testhelpers.go` files (CLAUDE.md).
- Tenant headers on every request/test: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` (see `listing/resource_test.go` `withTenant`).
- Commands run from the module dir: `cd services/atlas-mts/atlas.com/mts`.

---

### Task 1: `listing.BackdateEndsAt` — conditional ends_at rewrite

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/listing/administrator.go` (append after `AdvanceAuctionBid`)
- Test: `services/atlas-mts/atlas.com/mts/listing/administrator_test.go` (append)

**Interfaces:**
- Produces: `listing.BackdateEndsAt(db *gorm.DB, id string, to time.Time) (int64, error)` — rows-affected conditional update; Task 4's expire handler consumes it.

- [ ] **Step 1: Write the failing test** (append to `administrator_test.go`, mirroring its existing style — sqlite via `test.SetupTestDB(t, listing.Migration)`, models built with `listing.NewBuilder`):

```go
func TestBackdateEndsAt(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	future := time.Now().Add(24 * time.Hour)
	auction, err := listing.NewBuilder(test.TestTenantId, 0, 1001).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeAuction).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("3").
		SetSubCategory("1").
		SetEndsAt(&future).
		Build()
	if err != nil {
		t.Fatalf("build auction: %v", err)
	}
	created, err := listing.CreateListing(db, auction)
	if err != nil {
		t.Fatalf("create auction: %v", err)
	}

	fixed, err := listing.NewBuilder(test.TestTenantId, 0, 1002).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("1").
		SetSubCategory("1").
		Build()
	if err != nil {
		t.Fatalf("build fixed: %v", err)
	}
	createdFixed, err := listing.CreateListing(db, fixed)
	if err != nil {
		t.Fatalf("create fixed: %v", err)
	}

	past := time.Now().Add(-time.Second)

	// Active auction: backdates, 1 row.
	rows, err := listing.BackdateEndsAt(db, created.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate auction: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}
	got, err := listing.GetById(created.Id().String())(db)()
	if err != nil {
		t.Fatalf("reload auction: %v", err)
	}
	if got.EndsAt() == nil || !got.EndsAt().Before(time.Now()) {
		t.Fatalf("expected backdated endsAt, got %v", got.EndsAt())
	}

	// Fixed-sale listing: refused, 0 rows.
	rows, err = listing.BackdateEndsAt(db, createdFixed.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate fixed: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows for fixed sale, got %d", rows)
	}

	// Non-active auction: refused, 0 rows.
	if _, err := listing.UpdateState(db, created.Id().String(), listing.StateActive, listing.StateExpired); err != nil {
		t.Fatalf("transition: %v", err)
	}
	rows, err = listing.BackdateEndsAt(db, created.Id().String(), past)
	if err != nil {
		t.Fatalf("backdate expired: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows for non-active listing, got %d", rows)
	}
}
```

If the existing test file's imports lack `time` or `test`, add them; check whether it is `package listing` or `package listing_test` and match the existing package clause (adjust `listing.` qualifiers accordingly).

- [ ] **Step 2: Run red** — `go test ./listing/ -run TestBackdateEndsAt -v`. Expected: FAIL `undefined: listing.BackdateEndsAt` (or `BackdateEndsAt`).

- [ ] **Step 3: Implement** (append to `administrator.go`):

```go
// BackdateEndsAt rewrites ends_at on an ACTIVE AUCTION listing only — the
// test-route time-travel primitive (design-e2e-testing.md §4.2). The state and
// sale-type guards live in the WHERE clause so the update is race-safe: a
// listing settled between the caller's read and this write is left untouched
// (0 rows affected), exactly like UpdateState. Everything downstream of the
// rewritten timestamp (sweep discovery, settle/expire arms) is production code.
func BackdateEndsAt(db *gorm.DB, id string, to time.Time) (int64, error) {
	res := db.Model(&entity{}).
		Where("id = ? AND state = ? AND sale_type = ?", parseId(id), string(StateActive), string(SaleTypeAuction)).
		Update("ends_at", to)
	return res.RowsAffected, res.Error
}
```

(`parseId`, `entity`, `StateActive`, `SaleTypeAuction` already exist in the package. If `parseId(id)` returns `uuid.Nil` for garbage input the WHERE simply matches nothing — 0 rows, no error, which is the desired behavior.)

- [ ] **Step 4: Run green** — `go test ./listing/ -v`. Expected: PASS (all listing tests, not just the new one).

- [ ] **Step 5: Commit**

```bash
git add listing/administrator.go listing/administrator_test.go
git commit -m "feat(atlas-mts): BackdateEndsAt conditional rewrite for test-route time travel"
```

---

### Task 2: `testsupport` command providers (fidelity-pinned)

**Files:**
- Create: `services/atlas-mts/atlas.com/mts/testsupport/producer.go`
- Test: `services/atlas-mts/atlas.com/mts/testsupport/producer_test.go`

**Interfaces:**
- Produces: `testsupport.BuyCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool) model.Provider[[]kafka.Message]` and `testsupport.PlaceBidCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) model.Provider[[]kafka.Message]` — consumed by Task 5's handlers.
- These MUST stay field-for-field identical to the channel's `services/atlas-channel/atlas.com/channel/mts/producer.go` `BuyCommandProvider`/`PlaceBidCommandProvider` (same key derivation `producer.CreateKey(int(buyerId))` / `(int(bidderId))`, same `Command[...]` envelope from `atlas-mts/kafka/message/mts`).

- [ ] **Step 1: Write the failing test** `testsupport/producer_test.go`:

```go
package testsupport

import (
	"encoding/json"
	"testing"

	mtsmsg "atlas-mts/kafka/message/mts"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/google/uuid"
)

// TestBuyCommandProviderShape pins the simulated buy command to the exact
// envelope the channel emits and the mts consumer decodes: same Command struct,
// CommandBuy type tag, and buyer-keyed partition key.
func TestBuyCommandProviderShape(t *testing.T) {
	txn := uuid.New()
	msgs, err := BuyCommandProvider(txn, 0, 42, 2001, 3001, true)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if want := string(kprod.CreateKey(int(uint32(2001)))); string(msgs[0].Key) != want {
		t.Fatalf("expected buyer-derived key %q, got %q", want, string(msgs[0].Key))
	}
	var c mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("decode as consumer would: %v", err)
	}
	if c.TransactionId != txn || c.Type != mtsmsg.CommandBuy {
		t.Fatalf("bad envelope: %+v", c)
	}
	if c.Body.WorldId != 0 || c.Body.Serial != 42 || c.Body.BuyerId != 2001 || c.Body.BuyerAccountId != 3001 || !c.Body.BuyNow {
		t.Fatalf("bad body: %+v", c.Body)
	}
}

// TestPlaceBidCommandProviderShape pins the simulated bid command likewise.
func TestPlaceBidCommandProviderShape(t *testing.T) {
	txn := uuid.New()
	msgs, err := PlaceBidCommandProvider(txn, 0, 42, 2001, 3001, 7500)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if want := string(kprod.CreateKey(int(uint32(2001)))); string(msgs[0].Key) != want {
		t.Fatalf("expected bidder-derived key %q, got %q", want, string(msgs[0].Key))
	}
	var c mtsmsg.Command[mtsmsg.PlaceBidCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("decode as consumer would: %v", err)
	}
	if c.TransactionId != txn || c.Type != mtsmsg.CommandPlaceBid {
		t.Fatalf("bad envelope: %+v", c)
	}
	if c.Body.WorldId != 0 || c.Body.Serial != 42 || c.Body.BidderId != 2001 || c.Body.BidderAccountId != 3001 || c.Body.Amount != 7500 {
		t.Fatalf("bad body: %+v", c.Body)
	}
}
```

- [ ] **Step 2: Run red** — `go test ./testsupport/ -v`. Expected: FAIL (package does not exist / undefined providers).

- [ ] **Step 3: Implement** `testsupport/producer.go` — a line-for-line mirror of the channel's providers, built on atlas-mts's own message structs:

```go
// Package testsupport holds the env-gated MTS test routes
// (design-e2e-testing.md): data seeding, listing time travel, an on-demand
// expiration sweep, and simulated buyer/bidder actions. The simulated actions
// emit the SAME Kafka commands the channel emits for a real client, so the
// full production path (consumer -> processor -> saga -> orchestrator ->
// wallet/custody) runs. Routes are registered only when
// MTS_TEST_ROUTES_ENABLED=true and are never routed through ingress.
package testsupport

import (
	mtsmsg "atlas-mts/kafka/message/mts"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// BuyCommandProvider mirrors the channel's BuyCommandProvider
// (services/atlas-channel/.../mts/producer.go) field-for-field: buyer-keyed,
// CommandBuy envelope. Keep the two in lockstep — the fidelity of /test
// purchases rests on this being indistinguishable from a client-driven buy.
func BuyCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, buyerId uint32, buyerAccountId uint32, buyNow bool) model.Provider[[]kafka.Message] {
	key := kprod.CreateKey(int(buyerId))
	value := &mtsmsg.Command[mtsmsg.BuyCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandBuy,
		Body: mtsmsg.BuyCommandBody{
			WorldId:        byte(worldId),
			Serial:         serial,
			BuyerId:        buyerId,
			BuyerAccountId: buyerAccountId,
			BuyNow:         buyNow,
		},
	}
	return kprod.SingleMessageProvider(key, value)
}

// PlaceBidCommandProvider mirrors the channel's PlaceBidCommandProvider:
// bidder-keyed, CommandPlaceBid envelope carrying the raw bid amount (the
// consumer applies the commission mark-up at escrow time, same as for a
// client-driven bid).
func PlaceBidCommandProvider(transactionId uuid.UUID, worldId world.Id, serial uint32, bidderId uint32, bidderAccountId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := kprod.CreateKey(int(bidderId))
	value := &mtsmsg.Command[mtsmsg.PlaceBidCommandBody]{
		TransactionId: transactionId,
		Type:          mtsmsg.CommandPlaceBid,
		Body: mtsmsg.PlaceBidCommandBody{
			WorldId:         byte(worldId),
			Serial:          serial,
			BidderId:        bidderId,
			BidderAccountId: bidderAccountId,
			Amount:          amount,
		},
	}
	return kprod.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Run green** — `go test ./testsupport/ -v`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add testsupport/producer.go testsupport/producer_test.go
git commit -m "feat(atlas-mts): testsupport buy/bid command providers (channel-identical envelopes)"
```

---

### Task 3: Seed endpoint — `POST /test/listings/seed`

**Files:**
- Create: `services/atlas-mts/atlas.com/mts/testsupport/rest.go`
- Create: `services/atlas-mts/atlas.com/mts/testsupport/resource.go`
- Test: `services/atlas-mts/atlas.com/mts/testsupport/resource_test.go`

**Interfaces:**
- Consumes: `listing.NewBuilder/CreateListing` (serial auto-assigned inside `CreateListing` via `serial.Next` — do NOT call `serial.Next` yourself), `inventory.TypeFromItemId(item.Id(templateId))` for sub-category, `rest.RegisterInputHandler`, `tenant.MustFromContext`.
- Produces: `testsupport.InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer` — Tasks 4–6 add routes inside it; Task 7 registers it in `main.go`. Also `SeedRestModel` (input) and `seededListingsResponse` marshaling via `listing.Transform` + `server.MarshalResponse[[]listing.RestModel]`.

- [ ] **Step 1: Write the failing test** `testsupport/resource_test.go`. Copy the scaffolding idiom from `listing/resource_test.go` (`testServerInfo`, `newListingServer`, `withTenant`) — reproduce it locally in this package (it is test scaffolding, not a shared helper):

```go
package testsupport_test

import (
	"atlas-mts/listing"
	"atlas-mts/test"
	"atlas-mts/testsupport"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

func newTestServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	testsupport.InitResource(testServerInfo{})(db)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", test.TestTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedBody(t *testing.T, attributes map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"data": map[string]any{"type": "test-seeds", "attributes": attributes},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return b
}

// TestSeedListings asserts a mixed seed call creates active rows with real
// serials and derived categories, and that they surface in a normal browse.
func TestSeedListings(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{
			{"saleType": "fixed", "count": 3, "templateId": 1302000, "listValue": 1000},
			{"saleType": "auction", "count": 2, "templateId": 2000000, "quantity": 50, "listValue": 500, "durationSeconds": 30},
		},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var envelope struct {
		Data []struct {
			Id         string `json:"id"`
			Attributes struct {
				ItcSn    uint32 `json:"itcSn"`
				SaleType string `json:"saleType"`
				State    string `json:"state"`
				Category string `json:"category"`
				EndsAt   *string `json:"endsAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) != 5 {
		t.Fatalf("expected 5 seeded listings, got %d", len(envelope.Data))
	}
	serials := map[uint32]bool{}
	auctions := 0
	for _, d := range envelope.Data {
		if d.Attributes.State != "active" {
			t.Fatalf("expected active, got %s", d.Attributes.State)
		}
		if d.Attributes.ItcSn == 0 || serials[d.Attributes.ItcSn] {
			t.Fatalf("expected distinct non-zero serials, got %d twice or zero", d.Attributes.ItcSn)
		}
		serials[d.Attributes.ItcSn] = true
		switch d.Attributes.SaleType {
		case "auction":
			auctions++
			if d.Attributes.Category != "3" {
				t.Fatalf("auction category = %q, want \"3\"", d.Attributes.Category)
			}
			if d.Attributes.EndsAt == nil {
				t.Fatal("auction missing endsAt")
			}
		case "fixed":
			if d.Attributes.Category != "1" {
				t.Fatalf("fixed category = %q, want \"1\"", d.Attributes.Category)
			}
		default:
			t.Fatalf("unexpected saleType %q", d.Attributes.SaleType)
		}
	}
	if auctions != 2 {
		t.Fatalf("expected 2 auctions, got %d", auctions)
	}

	// Seeded rows are real: the production browse sees all 5.
	ms, err := listing.NewProcessor(logrus.New(), test.CreateTestContext(), db).Browse(0, listing.StateActive, listing.BrowseFilter{})
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(ms) != 5 {
		t.Fatalf("browse found %d listings, want 5", len(ms))
	}
}

// TestSeedCap asserts the 200-listing cap returns 400.
func TestSeedCap(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{{"saleType": "fixed", "count": 201, "templateId": 1302000, "listValue": 100}},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
```

Adjust the `Browse` call signature if it differs (check `listing/processor.go` — it is `Browse(worldId world.Id, state State, f BrowseFilter)`; fix the filter literal to whatever `BrowseFilter` requires, zero value is fine). If `fmt` ends up unused, drop it.

- [ ] **Step 2: Run red** — `go test ./testsupport/ -v`. Expected: FAIL (undefined `InitResource`, `SeedRestModel`).

- [ ] **Step 3: Implement `testsupport/rest.go`** (seed input model only for now; Tasks 5–6 append theirs):

```go
package testsupport

// SeedEntry describes one batch of listings to fabricate. Zero-valued fields
// take defaults in handleSeedListings (count 1, quantity 1, sellerId
// 999000001, sellerName "TestSeller", listValue 1000, durationSeconds 300 for
// auctions).
type SeedEntry struct {
	SaleType        string  `json:"saleType"` // "fixed" | "auction"
	Count           int     `json:"count,omitempty"`
	TemplateId      uint32  `json:"templateId"`
	Quantity        uint32  `json:"quantity,omitempty"`
	ListValue       uint32  `json:"listValue,omitempty"`
	BuyNowPrice     *uint32 `json:"buyNowPrice,omitempty"`
	StartingBid     uint32  `json:"startingBid,omitempty"`
	DurationSeconds int     `json:"durationSeconds,omitempty"`
	SellerId        uint32  `json:"sellerId,omitempty"`
	SellerAccountId uint32  `json:"sellerAccountId,omitempty"`
	SellerName      string  `json:"sellerName,omitempty"`
}

// SeedRestModel is the input envelope for POST /test/listings/seed.
type SeedRestModel struct {
	Id      string      `json:"-"`
	WorldId byte        `json:"worldId"`
	Entries []SeedEntry `json:"entries"`
}

func (r SeedRestModel) GetName() string { return "test-seeds" }

func (r SeedRestModel) GetID() string { return r.Id }

func (r *SeedRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

- [ ] **Step 4: Implement `testsupport/resource.go`** with the route registration and the seed handler:

```go
package testsupport

import (
	"atlas-mts/listing"
	"atlas-mts/rest"
	"net/http"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// seedMaxListings caps one seed call; bigger requests are a client mistake,
// not a load test (design-e2e-testing.md §4.5).
const seedMaxListings = 200

// Seed defaults — synthetic seller ids sit far above real character ids so
// they are recognizable in DB rows and logs.
const (
	defaultSeedSellerId   = 999000001
	defaultSeedSellerName = "TestSeller"
	defaultSeedListValue  = 1000
	defaultSeedDuration   = 300 * time.Second
)

// InitResource registers the env-gated MTS test routes (main.go only wires
// this when MTS_TEST_ROUTES_ENABLED=true; there is deliberately no ingress
// route — port-forward to the service to use these):
//   - POST /test/listings/seed                — fabricate active listings (real serials)
//   - POST /test/listings/{listingId}/expire  — backdate an active auction's ends_at
//   - POST /test/sweep                        — run one expiration sweep now
//   - POST /test/purchases                    — emit a channel-identical BUY command
//   - POST /test/bids                         — emit a channel-identical PLACE_BID command
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerSeed := rest.RegisterInputHandler[SeedRestModel](l)(db)(si)

			r := router.PathPrefix("/test").Subrouter()
			r.HandleFunc("/listings/seed", registerSeed("test_seed_listings", handleSeedListings)).Methods(http.MethodPost)
		}
	}
}

// handleSeedListings fabricates active listings through the production
// listing administrator: CreateListing assigns each row a real per-(tenant,
// world) ITC serial, so the client renders and interacts with seeded rows
// exactly like organic ones. Category/sub-category are derived the same way
// the custody consumer derives them (section from sale type, item tab from
// the template id) so seeded rows land under the right client tabs. The item
// snapshot is synthetic — see design-e2e-testing.md §4.3 for the fidelity
// ledger.
func handleSeedListings(d *rest.HandlerDependency, c *rest.HandlerContext, rm SeedRestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())

		total := 0
		for _, e := range rm.Entries {
			count := e.Count
			if count <= 0 {
				count = 1
			}
			total += count
		}
		if total == 0 || total > seedMaxListings {
			d.Logger().Errorf("Seed request wants [%d] listings (allowed 1..%d).", total, seedMaxListings)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		db := d.DB().WithContext(d.Context())
		created := make([]listing.Model, 0, total)
		for _, e := range rm.Entries {
			st := listing.SaleType(e.SaleType)
			if st != listing.SaleTypeFixed && st != listing.SaleTypeAuction {
				d.Logger().Errorf("Seed entry has invalid saleType [%s].", e.SaleType)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if e.TemplateId == 0 {
				d.Logger().Errorf("Seed entry missing templateId.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			count := e.Count
			if count <= 0 {
				count = 1
			}
			quantity := e.Quantity
			if quantity == 0 {
				quantity = 1
			}
			listValue := e.ListValue
			if listValue == 0 {
				listValue = defaultSeedListValue
			}
			sellerId := e.SellerId
			if sellerId == 0 {
				sellerId = defaultSeedSellerId
			}
			sellerName := e.SellerName
			if sellerName == "" {
				sellerName = defaultSeedSellerName
			}

			// Category mirrors the custody consumer's derivation: the section tab
			// from the sale type ("1" For Sale, "3" Auction), the item sub-tab
			// from the template id's inventory type.
			category := "1"
			if st == listing.SaleTypeAuction {
				category = "3"
			}
			subCategory := ""
			if it, ok := inventory.TypeFromItemId(item.Id(e.TemplateId)); ok {
				subCategory = strconv.Itoa(int(it))
			}

			for i := 0; i < count; i++ {
				b := listing.NewBuilder(t.Id(), world.Id(rm.WorldId), sellerId).
					SetSellerAccountId(e.SellerAccountId).
					SetSellerName(sellerName).
					SetSaleType(st).
					SetState(listing.StateActive).
					SetTemplateId(e.TemplateId).
					SetQuantity(quantity).
					SetListValue(listValue).
					SetBuyNowPrice(e.BuyNowPrice).
					SetCommissionRate(0.10).
					SetCategory(category).
					SetSubCategory(subCategory).
					SetMinIncrement(1)
				if st == listing.SaleTypeAuction {
					duration := defaultSeedDuration
					if e.DurationSeconds > 0 {
						duration = time.Duration(e.DurationSeconds) * time.Second
					}
					end := time.Now().Add(duration)
					b = b.SetEndsAt(&end).SetCurrentBid(e.StartingBid)
				}
				m, err := b.Build()
				if err != nil {
					d.Logger().WithError(err).Errorf("Building seed listing.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				cm, err := listing.CreateListing(db, m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating seed listing.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				created = append(created, cm)
			}
		}

		res, err := model.SliceMap(listing.Transform)(model.FixedProvider(created))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model for seeded listings.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		d.Logger().Infof("[TEST ROUTE] Seeded [%d] listings in world [%d] for tenant [%s].", len(created), rm.WorldId, t.Id())
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[[]listing.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
```

Check whether `server.MarshalResponse` writes its own status header (read `libs/atlas-rest/server` marshal code or mirror how `listing/resource.go` returns 200s): if it calls `w.WriteHeader` itself, drop the explicit `w.WriteHeader(http.StatusCreated)` and accept 200, updating the test's expected status to `http.StatusOK`. Match whatever the lib actually does — do not double-write the header.

- [ ] **Step 5: Run green** — `go test ./testsupport/ -v`. Expected: PASS.

- [ ] **Step 6: Vet + build** — `go vet ./... && go build ./...`. Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add testsupport/
git commit -m "feat(atlas-mts): test route — seed sample listings with real ITC serials"
```

---

### Task 4: Expire + sweep endpoints — `POST /test/listings/{listingId}/expire`, `POST /test/sweep`

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/testsupport/resource.go` (add routes + handlers)
- Modify: `services/atlas-mts/atlas.com/mts/testsupport/rest.go` (add `SweepResultRestModel`)
- Test: `services/atlas-mts/atlas.com/mts/testsupport/resource_test.go` (append)

**Interfaces:**
- Consumes: `listing.BackdateEndsAt` (Task 1), `task.Sweep(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, opts ...listing.Option) (int, error)`, `rest.ParseListingId`.
- Produces: the two routes; `SweepResultRestModel{Id string, Swept int}` with `GetName() "test-sweeps"`.

- [ ] **Step 1: Write the failing test** (append to `resource_test.go`; note `task.Sweep`'s no-bid expire arm creates a seller holding, so migrate `holding.Migration` too — mirror how `task/periodic_test.go` sets up its DB and, if it stubs processor options via `listing.Option`, reuse the same stubbing here):

```go
// TestExpireAndSweep drives the full test-route time-travel loop: seed a
// 1-hour auction, backdate it via the expire route, run one sweep via the
// sweep route, and assert the no-bids arm returned it to the seller
// (state=expired).
func TestExpireAndSweep(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration, bid.Migration, transaction.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{
			{"saleType": "auction", "templateId": 1302000, "listValue": 1000, "durationSeconds": 3600},
		},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var envelope struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode seed response: %v", err)
	}
	if len(envelope.Data) != 1 {
		t.Fatalf("expected 1 seeded auction, got %d", len(envelope.Data))
	}
	id := envelope.Data[0].Id

	// Not yet expired: a sweep finds nothing.
	res, err = ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/sweep", nil))
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sweep status = %d, want 200", res.StatusCode)
	}

	// Backdate.
	res, err = ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/"+id+"/expire", nil))
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expire status = %d, want 204", res.StatusCode)
	}

	// Second expire on the same (now-backdated but still active) listing is
	// still a 204 (row still matches the guard); after settling it becomes 409.
	res, err = ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/sweep", nil))
	if err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sweep 2 status = %d, want 200", res.StatusCode)
	}

	got, err := listing.GetById(id)(db)()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.State() != listing.StateExpired {
		t.Fatalf("state = %s, want expired", got.State())
	}

	// Expire after settle: 409.
	res, err = ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/"+id+"/expire", nil))
	if err != nil {
		t.Fatalf("expire 2: %v", err)
	}
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expire-after-settle status = %d, want 409", res.StatusCode)
	}
}

// TestExpireRejectsFixedSale asserts /expire 409s on a non-auction listing.
func TestExpireRejectsFixedSale(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	ts := newTestServer(t, db)
	defer ts.Close()

	body := seedBody(t, map[string]any{
		"worldId": 0,
		"entries": []map[string]any{{"saleType": "fixed", "templateId": 1302000, "listValue": 100}},
	})
	res, err := ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/seed", body))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var envelope struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res, err = ts.Client().Do(withTenant(t, http.MethodPost, ts.URL+"/test/listings/"+envelope.Data[0].Id+"/expire", nil))
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
}
```

Imports to add: `atlas-mts/holding`, `atlas-mts/bid`, `atlas-mts/transaction`. `withTenant` gains a nil-body call — it already accepts `body []byte`; `bytes.NewReader(nil)` is a valid empty reader, no change needed. If `task.Sweep`'s settle path needs a saga-producer stub for the no-bid arm, check `task/periodic_test.go` first and thread the same `listing.Option` through a package-level `sweepOptions` variable in `resource.go` that tests can set — only if the plain call fails; the no-bids expire arm is expected to be saga-free.

- [ ] **Step 2: Run red** — `go test ./testsupport/ -v`. Expected: FAIL (404s — routes not registered).

- [ ] **Step 3: Implement.** In `rest.go`, append:

```go
// SweepResultRestModel is the response envelope for POST /test/sweep.
type SweepResultRestModel struct {
	Id    string `json:"-"`
	Swept int    `json:"swept"`
}

func (r SweepResultRestModel) GetName() string { return "test-sweeps" }

func (r SweepResultRestModel) GetID() string { return r.Id }

func (r *SweepResultRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

In `resource.go`, register the routes inside `InitResource` (after the seed route):

```go
			registerGet := rest.RegisterHandler(l)(db)(si)
			r.HandleFunc("/listings/{listingId}/expire", registerGet("test_expire_listing", handleExpireListing)).Methods(http.MethodPost)
			r.HandleFunc("/sweep", registerGet("test_run_sweep", handleRunSweep)).Methods(http.MethodPost)
```

and add the handlers (new imports: `atlas-mts/task`, `errors`, `github.com/google/uuid`):

```go
// handleExpireListing backdates an ACTIVE AUCTION's ends_at to one second ago
// so the next sweep settles it. 404 unknown listing, 409 when the row is not
// an active auction (already settled / fixed sale), 204 on success. Only the
// timestamp is synthetic — discovery and settlement stay production code.
func handleExpireListing(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseListingId(d.Logger(), func(listingId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := listing.NewProcessor(d.Logger(), d.Context(), d.DB())
			if _, err := p.GetById(listingId); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving listing [%s] for test expire.", listingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			rows, err := listing.BackdateEndsAt(d.DB().WithContext(d.Context()), listingId, time.Now().Add(-time.Second))
			if err != nil {
				d.Logger().WithError(err).Errorf("Backdating listing [%s].", listingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if rows == 0 {
				// Not an active auction (fixed sale, or already settled).
				w.WriteHeader(http.StatusConflict)
				return
			}
			d.Logger().Infof("[TEST ROUTE] Backdated ends_at on listing [%s].", listingId)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// handleRunSweep runs one production expiration sweep on demand — the same
// task.Sweep the 60s ticker calls, cross-tenant like the ticker (the sweep
// itself applies WithoutTenantFilter; the row's own tenant_id scopes each
// settle). Returns the number of listings settled/expired this pass.
func handleRunSweep(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		swept, err := task.Sweep(d.Logger(), d.Context(), d.DB())
		if err != nil {
			d.Logger().WithError(err).Errorf("Test-route sweep failed.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		d.Logger().Infof("[TEST ROUTE] Sweep settled/expired [%d] listings.", swept)
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[SweepResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(SweepResultRestModel{Id: uuid.NewString(), Swept: swept})
	}
}
```

- [ ] **Step 4: Run green** — `go test ./testsupport/ -v`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add testsupport/
git commit -m "feat(atlas-mts): test routes — force-expire auction + on-demand sweep"
```

---

### Task 5: Simulated purchase + bid endpoints — `POST /test/purchases`, `POST /test/bids`

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/testsupport/resource.go` (routes + handlers, injectable producer)
- Modify: `services/atlas-mts/atlas.com/mts/testsupport/rest.go` (input models)
- Test: `services/atlas-mts/atlas.com/mts/testsupport/resource_test.go` (append; needs an in-package variant — see Step 1 note)

**Interfaces:**
- Consumes: `BuyCommandProvider`/`PlaceBidCommandProvider` (Task 2), `producer2 "atlas-mts/kafka/producer"` `ProviderImpl(l)(ctx)(token)` with token `mtsmsg.EnvCommandTopic`, `listing.Processor.GetById`.
- Produces: `PurchaseRestModel`/`BidRestModel` inputs; handlers `handleSimulatePurchase(pf providerFn)` / `handleSimulateBid(pf providerFn)` where `type providerFn = func(ctx context.Context) func(token string) kprod.MessageProducer` (same alias the consumers use).

- [ ] **Step 1: Write the failing test.** The emission assertion needs to inject a recording producer, which requires an in-package test. Create the recording producer + handler tests in a NEW file `testsupport/simulate_test.go` with `package testsupport` (not `_test` suffix), mirroring the `recordingProducer` in `kafka/consumer/mts/consumer_test.go` but decoding commands:

```go
package testsupport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"atlas-mts/listing"
	"atlas-mts/rest"
	"atlas-mts/test"

	mtsmsg "atlas-mts/kafka/message/mts"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// recordedCommand captures one emitted MTS command for assertions.
type recordedCommand struct {
	topicToken string
	key        []byte
	value      []byte
}

// recordingProducer is a providerFn that captures instead of publishing —
// the same stand-in idiom kafka/consumer/mts/consumer_test.go uses.
type recordingProducer struct {
	mu       sync.Mutex
	commands []recordedCommand
}

func (r *recordingProducer) provider() providerFn {
	return func(ctx context.Context) func(token string) kprod.MessageProducer {
		return func(token string) kprod.MessageProducer {
			return func(p model.Provider[[]kafka.Message]) error {
				ms, err := p()
				if err != nil {
					return err
				}
				r.mu.Lock()
				defer r.mu.Unlock()
				for _, m := range ms {
					r.commands = append(r.commands, recordedCommand{topicToken: token, key: m.Key, value: m.Value})
				}
				return nil
			}
		}
	}
}

// newSimulateHandler builds an httptest server with ONLY the purchase/bid
// routes wired to the recording producer.
func newSimulateServer(t *testing.T, db *gorm.DB, rec *recordingProducer) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := muxRouterWithSimulateRoutes(l, db, rec.provider())
	return httptest.NewServer(router)
}

func TestSimulatePurchaseEmitsBuyCommand(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedFixedListing(t, db, 1000)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      m.Id().String(),
		"buyerId":        2001,
		"buyerAccountId": 3001,
		"buyNow":         false,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	if len(rec.commands) != 1 {
		t.Fatalf("expected 1 emitted command, got %d", len(rec.commands))
	}
	if rec.commands[0].topicToken != mtsmsg.EnvCommandTopic {
		t.Fatalf("topic token = %s, want %s", rec.commands[0].topicToken, mtsmsg.EnvCommandTopic)
	}
	var c mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(rec.commands[0].value, &c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.Type != mtsmsg.CommandBuy || c.Body.Serial != m.Serial() || c.Body.BuyerId != 2001 || c.Body.BuyerAccountId != 3001 || c.Body.BuyNow {
		t.Fatalf("bad command: %+v", c)
	}
}

func TestSimulatePurchaseUnknownListing404s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)
	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-purchases", map[string]any{
		"listingId":      "00000000-0000-0000-0000-000000000001",
		"buyerId":        2001,
		"buyerAccountId": 3001,
	})
	res := doPost(t, ts, "/test/purchases", body)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 404, got %d", len(rec.commands))
	}
}

func TestSimulateBidEmitsPlaceBidCommand(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedAuctionListing(t, db, 1000, time.Hour)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-bids", map[string]any{
		"listingId":       m.Id().String(),
		"bidderId":        2001,
		"bidderAccountId": 3001,
		"amount":          1500,
	})
	res := doPost(t, ts, "/test/bids", body)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	var c mtsmsg.Command[mtsmsg.PlaceBidCommandBody]
	if err := json.Unmarshal(rec.commands[0].value, &c); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if c.Type != mtsmsg.CommandPlaceBid || c.Body.Serial != m.Serial() || c.Body.Amount != 1500 {
		t.Fatalf("bad command: %+v", c)
	}
}

func TestSimulateBidOnFixedSale409s(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration)
	defer test.CleanupTestDB(t, db)

	m := seedFixedListing(t, db, 1000)

	rec := &recordingProducer{}
	ts := newSimulateServer(t, db, rec)
	defer ts.Close()

	body := jsonApiBody(t, "test-bids", map[string]any{
		"listingId":       m.Id().String(),
		"bidderId":        2001,
		"bidderAccountId": 3001,
		"amount":          1500,
	})
	res := doPost(t, ts, "/test/bids", body)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	if len(rec.commands) != 0 {
		t.Fatalf("expected no emission on 409, got %d", len(rec.commands))
	}
}
```

Also add (same file) the small helpers the tests need — `seedFixedListing` / `seedAuctionListing` build + `listing.CreateListing` a listing exactly like Task 1's test fixtures (`SetSaleType(listing.SaleTypeFixed)` / `SetSaleType(listing.SaleTypeAuction)` + `SetEndsAt(&end)` where `end := time.Now().Add(d)`); `jsonApiBody(t, resourceType, attributes)` marshals `{"data":{"type":resourceType,"attributes":attributes}}`; `doPost(t, ts, path, body)` posts with the four tenant headers (`TENANT_ID` = `test.TestTenantId.String()`, `REGION` GMS, `MAJOR_VERSION` 83, `MINOR_VERSION` 1) via `ts.Client().Do` and returns the response. `muxRouterWithSimulateRoutes(l, db, pf)` is implemented in Step 3 as a small non-test helper in `resource.go` so tests and `InitResource` share the exact same wiring.

- [ ] **Step 2: Run red** — `go test ./testsupport/ -v`. Expected: FAIL (undefined models/handlers/helpers).

- [ ] **Step 3: Implement.** In `rest.go`, append:

```go
// PurchaseRestModel is the input for POST /test/purchases — simulate another
// character buying a listing. BuyNow=true is the auction immediate-buyout arm.
type PurchaseRestModel struct {
	Id             string `json:"-"`
	ListingId      string `json:"listingId"`
	BuyerId        uint32 `json:"buyerId"`
	BuyerAccountId uint32 `json:"buyerAccountId"`
	BuyNow         bool   `json:"buyNow,omitempty"`
}

func (r PurchaseRestModel) GetName() string { return "test-purchases" }

func (r PurchaseRestModel) GetID() string { return r.Id }

func (r *PurchaseRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// BidRestModel is the input for POST /test/bids — simulate a competing bidder.
type BidRestModel struct {
	Id              string `json:"-"`
	ListingId       string `json:"listingId"`
	BidderId        uint32 `json:"bidderId"`
	BidderAccountId uint32 `json:"bidderAccountId"`
	Amount          uint32 `json:"amount"`
}

func (r BidRestModel) GetName() string { return "test-bids" }

func (r BidRestModel) GetID() string { return r.Id }

func (r *BidRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

In `resource.go`: add imports (`context`, `producer2 "atlas-mts/kafka/producer"`, `mtsmsg "atlas-mts/kafka/message/mts"`, `kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"`), the alias + shared wiring helper, route registration inside `InitResource`, and the two handlers:

```go
// providerFn matches the per-context producer factory shape the Kafka
// consumers use (producer2.ProviderImpl(l)), so tests can inject a recorder.
type providerFn = func(ctx context.Context) func(token string) kprod.MessageProducer
```

Inside `InitResource`'s route block, replace direct registration with the shared helper so tests wire identically:

```go
			registerSimulateRoutes(r, l, db, si, producer2.ProviderImpl(l))
```

and add:

```go
// registerSimulateRoutes wires the simulated-actor routes onto sub-router r.
// Split out so simulate_test.go can mount the identical wiring with a
// recording producer.
func registerSimulateRoutes(r *mux.Router, l logrus.FieldLogger, db *gorm.DB, si jsonapi.ServerInformation, pf providerFn) {
	registerPurchase := rest.RegisterInputHandler[PurchaseRestModel](l)(db)(si)
	registerBid := rest.RegisterInputHandler[BidRestModel](l)(db)(si)
	r.HandleFunc("/purchases", registerPurchase("test_simulate_purchase", handleSimulatePurchase(pf))).Methods(http.MethodPost)
	r.HandleFunc("/bids", registerBid("test_simulate_bid", handleSimulateBid(pf))).Methods(http.MethodPost)
}

// muxRouterWithSimulateRoutes builds a standalone router holding only the
// simulate routes — test scaffolding for simulate_test.go kept beside the
// production wiring so the two can't drift.
func muxRouterWithSimulateRoutes(l logrus.FieldLogger, db *gorm.DB, pf providerFn) *mux.Router {
	router := mux.NewRouter()
	registerSimulateRoutes(router.PathPrefix("/test").Subrouter(), l, db, testsupportServerInfo{}, pf)
	return router
}

// testsupportServerInfo is minimal ServerInformation for the standalone router.
type testsupportServerInfo struct{}

func (testsupportServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (testsupportServerInfo) GetPrefix() string  { return "/api" }

// handleSimulatePurchase emits the channel-identical BUY command for the
// supplied buyer against an existing listing. Structural pre-checks only
// (listing exists + purchasable state) — economic validation (wallet balance,
// buy-now price) belongs to the production consumer path, which emits
// BUY_FAILED exactly as it would for a real client. 202 = command emitted,
// NOT purchase completed; observe the outcome via listing state / transaction
// history / logs.
func handleSimulatePurchase(pf providerFn) func(d *rest.HandlerDependency, c *rest.HandlerContext, rm PurchaseRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm PurchaseRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if rm.ListingId == "" || rm.BuyerId == 0 || rm.BuyerAccountId == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			m, err := listing.NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(rm.ListingId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving listing [%s] for simulated purchase.", rm.ListingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if m.State() != listing.StateActive {
				d.Logger().Errorf("Simulated purchase of listing [%s] in state [%s]; conflict.", rm.ListingId, m.State())
				w.WriteHeader(http.StatusConflict)
				return
			}
			txn := uuid.New()
			if err := pf(d.Context())(mtsmsg.EnvCommandTopic)(BuyCommandProvider(txn, m.WorldId(), m.Serial(), rm.BuyerId, rm.BuyerAccountId, rm.BuyNow)); err != nil {
				d.Logger().WithError(err).Errorf("Emitting simulated BUY for listing [%s].", rm.ListingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			d.Logger().Infof("[TEST ROUTE] Emitted BUY txn [%s] — buyer [%d] listing [%s] serial [%d] buyNow [%t].", txn, rm.BuyerId, rm.ListingId, m.Serial(), rm.BuyNow)
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

// handleSimulateBid emits the channel-identical PLACE_BID command. Structural
// pre-checks only (active auction) — increment/escrow validation stays in the
// production consumer, which emits BID_FAILED as for a real client.
func handleSimulateBid(pf providerFn) func(d *rest.HandlerDependency, c *rest.HandlerContext, rm BidRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm BidRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if rm.ListingId == "" || rm.BidderId == 0 || rm.BidderAccountId == 0 || rm.Amount == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			m, err := listing.NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(rm.ListingId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving listing [%s] for simulated bid.", rm.ListingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if m.SaleType() != listing.SaleTypeAuction || m.State() != listing.StateActive {
				d.Logger().Errorf("Simulated bid on listing [%s] (saleType [%s], state [%s]); conflict.", rm.ListingId, m.SaleType(), m.State())
				w.WriteHeader(http.StatusConflict)
				return
			}
			txn := uuid.New()
			if err := pf(d.Context())(mtsmsg.EnvCommandTopic)(PlaceBidCommandProvider(txn, m.WorldId(), m.Serial(), rm.BidderId, rm.BidderAccountId, rm.Amount)); err != nil {
				d.Logger().WithError(err).Errorf("Emitting simulated PLACE_BID for listing [%s].", rm.ListingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			d.Logger().Infof("[TEST ROUTE] Emitted PLACE_BID txn [%s] — bidder [%d] listing [%s] serial [%d] amount [%d].", txn, rm.BidderId, rm.ListingId, m.Serial(), rm.Amount)
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
```

(`m.WorldId()` already returns `world.Id`; if the provider signature wants it, pass as-is. `errors` and `gorm` are already imported from Task 4.)

- [ ] **Step 4: Run green** — `go test ./testsupport/ -v`. Expected: PASS.

- [ ] **Step 5: Vet** — `go vet ./...`. Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add testsupport/
git commit -m "feat(atlas-mts): test routes — simulated buyer purchase + competing bidder"
```

---

### Task 6: Env-gated registration in `main.go` + bake

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/main.go`

**Interfaces:**
- Consumes: `testsupport.InitResource` (Task 3).
- Produces: the `MTS_TEST_ROUTES_ENABLED` gate — the ONLY activation point for everything above.

- [ ] **Step 1: Modify `main.go`.** Add `"atlas-mts/testsupport"` to imports and restructure the fluent server chain (currently one expression ending `.Run()`) into a variable so the test resource is appended conditionally:

```go
	srv := server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(listing.InitResource(GetServer())(db)).
		AddRouteInitializer(holding.InitResource(GetServer())(db)).
		AddRouteInitializer(wish.InitResource(GetServer())(db)).
		AddRouteInitializer(transaction.InitResource(GetServer())(db)).
		AddRouteInitializer(wallet.InitResource(GetServer())(db)).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler()))

	// E2E test routes (seed/expire/sweep/simulated purchase+bid) — env-gated,
	// never routed through ingress, never enabled in any overlay. Enable ad hoc:
	//   kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
	// See docs/tasks/task-102-mts-marketplace/design-e2e-testing.md.
	if os.Getenv("MTS_TEST_ROUTES_ENABLED") == "true" {
		l.Warnln("MTS TEST ROUTES ENABLED — /api/test/* is live. This must never be set in production.")
		srv = srv.AddRouteInitializer(testsupport.InitResource(GetServer())(db))
	}

	srv.Run()
```

If `AddRouteInitializer` returns a non-assignable type (check `libs/atlas-rest/server`'s builder — it returns the builder itself), adjust to whatever the builder API allows; the observable requirement is: flag unset → `/api/test/*` returns 404, flag `"true"` → routes live, warning logged once at startup.

- [ ] **Step 2: Build + vet + race** — `go build ./... && go vet ./... && go test -race ./...`. Expected: all clean/PASS.

- [ ] **Step 3: Bake** (from the worktree root `/…/.worktrees/task-102-mts-marketplace`): `docker buildx bake atlas-mts`. Expected: image builds clean (no new libs, so no Dockerfile COPY changes — but the bake is mandatory per CLAUDE.md).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-mts/atlas.com/mts/main.go
git commit -m "feat(atlas-mts): gate E2E test routes behind MTS_TEST_ROUTES_ENABLED"
```

---

### Task 7: E2E playbook document

**Files:**
- Create: `docs/tasks/task-102-mts-marketplace/e2e-test-playbook.md`

**Interfaces:**
- Consumes: every endpoint above, the existing reads (`GET /api/worlds/{worldId}/listings`, `GET /api/characters/{characterId}/mts/holding|wallet|transactions`), atlas-cashshop `PATCH /api/accounts/{accountId}/wallet`.

- [ ] **Step 1: Write the playbook.** Structure (write the full document — concrete curl per scenario, one `$H` header block reused throughout; verify the exact transaction-history route path in `transaction/resource.go` and the wallet PATCH attribute names in `services/atlas-cashshop/atlas.com/cashshop/wallet/rest.go` before writing those curls — do not guess field names):

````markdown
# MTS E2E test playbook (task-102)

Prereqs
- PR env deployed with atlas-mts; real v83 client logged in as your main character.
- Enable routes:  kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
- Port-forward:   kubectl port-forward svc/atlas-mts 8080:8080
- Header block (fill in the tenant id):
  H=(-H "TENANT_ID: <uuid>" -H "REGION: GMS" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" -H "Content-Type: application/json")
- An alt character on the tenant = the "fake actor" (note its characterId + accountId).
- Top up the alt's NX (existing endpoint, via ingress or port-forward to atlas-cashshop):
  curl "${H[@]}" -X PATCH .../api/accounts/<altAccountId>/wallet -d '<envelope per wallet/rest.go>'
- Disable when done: kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED-

Scenario 1 — browse volume: seed 60 mixed listings, browse every tab/page in client.
Scenario 2 — I buy a seeded fixed listing (real client): full saga incl. wallet debit.
Scenario 3 — fake buyer buys MY listing: list an item in client, /test/purchases with alt ids,
             watch: listing sold in client, my points credited, transaction row.
Scenario 4 — outbid escrow release: seed auction, I bid in client, /test/bids higher with alt,
             watch: my escrow returns (wallet), OUTBID status.
Scenario 5 — I win an auction: seed auction (durationSeconds 60), I bid in client,
             /test/listings/{id}/expire + /test/sweep, watch: item in my Transfer Inventory,
             seller credited.
Scenario 6 — fake bidder wins MY auction: list auction in client, /test/bids with alt,
             expire + sweep, watch: my points credited, alt holding row.
Scenario 7 — no-bid expiry: seed or list auction, expire + sweep with zero bids,
             watch: item back in seller Transfer Inventory (origin=expired).
Scenario 8 — take-home + history: after each settle, take home in client; verify
             GET .../mts/transactions rows for every scenario above.

Each scenario lists: curl(s), expected client observation, expected REST observation
(GET listings/holding/wallet/transactions), expected atlas-mts/orchestrator log lines.
````

Flesh every scenario out with its real curl bodies (JSON:API envelopes exactly as in the tests: `{"data":{"type":"test-seeds"|"test-purchases"|"test-bids","attributes":{...}}}`) and expected outputs. Use repo-relative/service-relative URLs (`http://localhost:8080/api/...` for the port-forward) — never hardcode a cluster hostname.

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-102-mts-marketplace/e2e-test-playbook.md
git commit -m "docs(task-102): E2E test playbook (8 scenarios over the test routes)"
```

---

### Task 8: Full-module verification

- [ ] **Step 1:** From `services/atlas-mts/atlas.com/mts`: `go test -race ./... && go vet ./...`. Expected: PASS/clean.
- [ ] **Step 2:** From the worktree root: `GOWORK=off` NOT set — run `tools/redis-key-guard.sh` from the repo root per its own docs. Expected: clean (no new redis usage).
- [ ] **Step 3:** `docker buildx bake atlas-mts` (worktree root). Expected: clean.
- [ ] **Step 4:** Update `docs/tasks/task-102-mts-marketplace/rollout-checklist.md` — append a short section 7: test routes exist, how to enable/disable, pointer to the playbook. Commit:

```bash
git add docs/tasks/task-102-mts-marketplace/rollout-checklist.md
git commit -m "docs(task-102): rollout checklist — test-route enable/disable note"
```

---

## Out of scope (explicitly)

- No ingress/`routes.conf` changes, no overlay env vars, no UI surface for test routes.
- No CI-automated E2E harness (design non-goal; can layer on these endpoints later).
- The playbook EXECUTION (deploy + client pass) happens after this plan lands, per design §6 — followed by the branch's final verification phase, code review, and PR.
