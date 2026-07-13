package testsupport

import (
	"atlas-mts/listing"
	"atlas-mts/rest"
	"atlas-mts/task"
	"atlas-mts/wallet"
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	mtsmsg "atlas-mts/kafka/message/mts"
	producer2 "atlas-mts/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// providerFn matches the per-context producer factory shape the Kafka
// consumers use (producer2.ProviderImpl(l)), so tests can inject a recorder.
type providerFn = func(ctx context.Context) func(token string) kprod.MessageProducer

// seedMaxListings caps one seed call; bigger requests are a client mistake,
// not a load test (design-e2e-testing.md §4.5).
const seedMaxListings = 200

// Seed defaults — synthetic seller ids sit far above real character ids so
// they are recognizable in DB rows and logs.
const (
	defaultSeedSellerId = 999000001
	// defaultSeedSellerAccountId is the cash-shop account credited on a sale of a
	// seeded listing. It must be non-zero and wallet-backed or the buy's
	// seller-points credit fails (account 0 has no wallet) and no seeded listing
	// can be bought — the seed flow ensures this account's wallet exists.
	defaultSeedSellerAccountId = 999000001
	defaultSeedSellerName      = "TestSeller"
	defaultSeedListValue       = 1000
	defaultSeedDuration        = 300 * time.Second
	// defaultSeedFixedTerm mirrors the production fixedSaleHours default (168h):
	// seeded fixed sales carry the same era-faithful 7-day term natural ones get.
	defaultSeedFixedTerm = 168 * time.Hour
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

			registerGet := rest.RegisterHandler(l)(db)(si)
			r.HandleFunc("/listings/{listingId}/expire", registerGet("test_expire_listing", handleExpireListing)).Methods(http.MethodPost)
			r.HandleFunc("/sweep", registerGet("test_run_sweep", handleRunSweep)).Methods(http.MethodPost)

			registerSimulateRoutes(r, l, db, si, producer2.ProviderImpl(l))
		}
	}
}

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
// history / logs. Unlike handleSimulateBid, there is deliberately no SaleType
// gate here: a BUY against an auction (with or without BuyNow) is validated
// economically/semantically by the production consumer the same way a
// real client's would be, so rejecting it structurally here would diverge
// from that fidelity contract.
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
				server.WriteErrorResponse(d.Logger())(w)(err)
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
				server.WriteErrorResponse(d.Logger())(w)(err)
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
				server.WriteErrorResponse(d.Logger())(w)(err)
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
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			d.Logger().Infof("[TEST ROUTE] Emitted PLACE_BID txn [%s] — bidder [%d] listing [%s] serial [%d] amount [%d].", txn, rm.BidderId, rm.ListingId, m.Serial(), rm.Amount)
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

// effectiveSellerAccountId resolves the cash-shop account a seeded listing's sale
// credits: the entry's value, or the default synthetic test-seller account when
// omitted (0). A zero account has no wallet, so the sale settle fails — hence the
// default plus the wallet-ensure at seed time.
func effectiveSellerAccountId(entryAccountId uint32) uint32 {
	if entryAccountId == 0 {
		return defaultSeedSellerAccountId
	}
	return entryAccountId
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

		// First pass: validate every entry (saleType, templateId) and compute
		// the total-vs-cap BEFORE creating anything, so any 400 happens with
		// zero rows created — never a partial commit across entries. Also collect
		// the distinct seller accounts so their cash-shop wallets can be ensured
		// before seeding (else the buy's seller-points credit fails and no seeded
		// listing is buyable).
		total := 0
		sellerAccounts := make(map[uint32]bool)
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
			total += count
			sellerAccounts[effectiveSellerAccountId(e.SellerAccountId)] = true
		}
		if total == 0 || total > seedMaxListings {
			d.Logger().Errorf("Seed request wants [%d] listings (allowed 1..%d).", total, seedMaxListings)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Ensure each seller account has a cash-shop wallet BEFORE seeding, so
		// every seeded listing is immediately buyable (the sale credits the
		// seller's points; account 0 / a wallet-less account fails the settle).
		// A wallet that already exists is left as-is. Failing here (rather than
		// seeding un-buyable listings) keeps the endpoint honest.
		walletP := wallet.NewProcessor(d.Logger(), d.Context())
		for acct := range sellerAccounts {
			if err := walletP.EnsureWallet(acct, 0, 0, 0); err != nil {
				d.Logger().WithError(err).Errorf("Ensuring cash-shop wallet for seed seller account [%d]; seeded listings would not be buyable.", acct)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
		}

		// Second pass: everything below is validated, so all remaining errors
		// are unexpected (build/create) failures — create inside a single
		// transaction so any of them rolls back the whole seed (500, zero rows).
		db := d.DB().WithContext(d.Context())
		created := make([]listing.Model, 0, total)
		txErr := db.Transaction(func(tx *gorm.DB) error {
			for _, e := range rm.Entries {
				st := listing.SaleType(e.SaleType)

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
				sellerAccountId := effectiveSellerAccountId(e.SellerAccountId)
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
						SetSellerAccountId(sellerAccountId).
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
					// Every listing carries a sale term (era-faithful: fixed
					// sales expire back to the seller too). durationSeconds
					// overrides for both types; the defaults differ — a short
					// window for auctions (they exist to be expired in tests)
					// and the production 7-day term for fixed sales.
					duration := defaultSeedDuration
					if st == listing.SaleTypeFixed {
						duration = defaultSeedFixedTerm
					}
					if e.DurationSeconds > 0 {
						duration = time.Duration(e.DurationSeconds) * time.Second
					}
					end := time.Now().Add(duration)
					b = b.SetEndsAt(&end)
					if st == listing.SaleTypeAuction {
						b = b.SetCurrentBid(e.StartingBid)
					}
					m, err := b.Build()
					if err != nil {
						d.Logger().WithError(err).Errorf("Building seed listing.")
						return err
					}
					cm, err := listing.CreateListing(tx, m)
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating seed listing.")
						return err
					}
					created = append(created, cm)
				}
			}
			return nil
		})
		if txErr != nil {
			server.WriteErrorResponse(d.Logger())(w)(txErr)
			return
		}

		res, err := model.SliceMap(listing.Transform)(model.FixedProvider(created))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model for seeded listings.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		d.Logger().Infof("[TEST ROUTE] Seeded [%d] listings in world [%d] for tenant [%s].", len(created), rm.WorldId, t.Id())
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[[]listing.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

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
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			rows, err := listing.BackdateEndsAt(d.DB().WithContext(d.Context()), listingId, time.Now().Add(-time.Second))
			if err != nil {
				d.Logger().WithError(err).Errorf("Backdating listing [%s].", listingId)
				server.WriteErrorResponse(d.Logger())(w)(err)
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
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		d.Logger().Infof("[TEST ROUTE] Sweep settled/expired [%d] listings.", swept)
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[SweepResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(SweepResultRestModel{Id: uuid.NewString(), Swept: swept})
	}
}
