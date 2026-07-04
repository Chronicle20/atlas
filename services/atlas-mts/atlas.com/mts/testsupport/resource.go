package testsupport

import (
	"atlas-mts/listing"
	"atlas-mts/rest"
	"atlas-mts/task"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
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

			registerGet := rest.RegisterHandler(l)(db)(si)
			r.HandleFunc("/listings/{listingId}/expire", registerGet("test_expire_listing", handleExpireListing)).Methods(http.MethodPost)
			r.HandleFunc("/sweep", registerGet("test_run_sweep", handleRunSweep)).Methods(http.MethodPost)
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

		// First pass: validate every entry (saleType, templateId) and compute
		// the total-vs-cap BEFORE creating anything, so any 400 happens with
		// zero rows created — never a partial commit across entries.
		total := 0
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
		}
		if total == 0 || total > seedMaxListings {
			d.Logger().Errorf("Seed request wants [%d] listings (allowed 1..%d).", total, seedMaxListings)
			w.WriteHeader(http.StatusBadRequest)
			return
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
			w.WriteHeader(http.StatusInternalServerError)
			return
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
