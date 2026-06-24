package listing

import (
	"atlas-mts/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource registers the listing routes:
//   - GET  /worlds/{worldId}/listings              — browse/search active listings
//   - POST /worlds/{worldId}/listings              — initiate a list (TransferToMts saga)
//   - GET  /worlds/{worldId}/listings/{listingId}  — listing detail
//
// The POST initiates the custody/fee saga; it does NOT create the listing row
// (that happens on the custody consumer's AcceptToMtsListing). The DELETE
// performs the seller's race-safe cancel (active->holding(seller)).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[CreateListingRestModel](l)(db)(si)

			r := router.PathPrefix("/worlds/{worldId}/listings").Subrouter()
			r.HandleFunc("", registerGet("browse_listings", handleBrowseListings)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_listing", handleCreateListing)).Methods(http.MethodPost)
			r.HandleFunc("/{listingId}", registerGet("get_listing", handleGetListing)).Methods(http.MethodGet)
			r.HandleFunc("/{listingId}", registerGet("cancel_listing", handleCancelListing)).Methods(http.MethodDelete)
		}
	}
}

// handleCancelListing performs the seller's cancel of an active listing, moving
// the item to the seller's holding (origin=cancelled) so it can be taken home.
//
// Seller-only owner check (prd §8.4): the requesting character id is read from
// the ?characterId= query param (the atlas-portals ParseCharacterId precedent —
// path var, else query param), the listing is loaded, and the cancel proceeds
// only when the requester is the listing's seller; otherwise 403. The DELETE path
// carries no characterId path var, so the query param is the caller's identity.
//
// The cancel itself runs through the shared, race-safe listing.Processor.Cancel:
//   - 204 No Content on a win (row moved to cancelled, seller holding created);
//   - 409 Conflict on a non-active listing (race loser / already settled) — a
//     clean no-op, never a 500.
//
// Event emission stays in the Kafka command path (handleCancelListing in the mts
// consumer emits LISTING_CANCELLED). This REST entry point is the direct (e.g.
// channel-driven) cancel; it does not emit, keeping a single producer of the
// status event per the command-path-owns-emission convention.
func handleCancelListing(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseListingId(d.Logger(), func(listingId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			characterIdStr := r.URL.Query().Get("characterId")
			if characterIdStr == "" {
				d.Logger().Errorf("Cancel listing [%s] missing characterId query param for the seller-only check.", listingId)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			characterId64, err := strconv.ParseUint(characterIdStr, 10, 32)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to parse characterId query for cancel of listing [%s].", listingId)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			characterId := uint32(characterId64)

			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			// Load the listing for the seller-only owner check.
			m, err := p.GetById(listingId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving listing [%s] for cancel.", listingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Seller-only: only the listing's seller may cancel it (prd §8.4).
			if m.SellerId() != characterId {
				d.Logger().Errorf("Character [%d] attempted to cancel listing [%s] owned by seller [%d]; forbidden.", characterId, listingId, m.SellerId())
				w.WriteHeader(http.StatusForbidden)
				return
			}

			res, err := p.Cancel(listingId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Cancelling listing [%s].", listingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !res.Won {
				// Race loser / already-not-active: a clean conflict, not a 500.
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// handleCreateListing initiates a list: it validates the request against the
// tenant config and, on success, emits a TransferToMts saga. The response is
// 202 Accepted carrying the pre-allocated listing id — the listing row does not
// exist yet (it is created when the custody saga's AcceptToMtsListing lands).
func handleCreateListing(d *rest.HandlerDependency, c *rest.HandlerContext, rm CreateListingRestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			req := ListRequest{
				WorldId:             worldId,
				SellerId:            rm.SellerId,
				SellerAccountId:     rm.SellerAccountId,
				SellerName:          rm.SellerName,
				SaleType:            SaleType(rm.SaleType),
				SourceInventoryType: rm.SourceInventoryType,
				AssetId:             rm.AssetId,
				Quantity:            rm.Quantity,
				ListValue:           rm.ListValue,
				BuyNowPrice:         rm.BuyNowPrice,
				DurationHours:       rm.DurationHours,
				Category:            rm.Category,
				SubCategory:         rm.SubCategory,
			}

			listingId, err := NewProcessor(d.Logger(), d.Context(), d.DB()).List(req)
			if err != nil {
				d.Logger().WithError(err).Errorf("Initiating listing for world [%d] seller [%d].", byte(worldId), rm.SellerId)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			res := CreateListingRestModel{
				Id:                  listingId.String(),
				SellerId:            rm.SellerId,
				SellerName:          rm.SellerName,
				SaleType:            rm.SaleType,
				SourceInventoryType: rm.SourceInventoryType,
				AssetId:             rm.AssetId,
				Quantity:            rm.Quantity,
				ListValue:           rm.ListValue,
				BuyNowPrice:         rm.BuyNowPrice,
				DurationHours:       rm.DurationHours,
				Category:            rm.Category,
				SubCategory:         rm.SubCategory,
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusAccepted)
			server.MarshalResponse[CreateListingRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleBrowseListings(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			f := BrowseFilter{
				Category:    query.Get("category"),
				SubCategory: query.Get("subCategory"),
				SaleType:    SaleType(firstNonEmpty(query.Get("saleType"), query.Get("type"))),
				SellerName:  query.Get("sellerName"),
			}
			if v := query.Get("itemId"); v != "" {
				if itemId, err := strconv.ParseUint(v, 10, 32); err == nil {
					f.ItemId = uint32(itemId)
				}
			}
			if v := query.Get("serial"); v != "" {
				if serial, err := strconv.ParseUint(v, 10, 32); err == nil {
					f.Serial = uint32(serial)
				}
			}
			if v := query.Get("sellerId"); v != "" {
				if sellerId, err := strconv.ParseUint(v, 10, 32); err == nil {
					f.SellerId = uint32(sellerId)
				}
			}
			if v := query.Get("excludeSellerId"); v != "" {
				if excludeSellerId, err := strconv.ParseUint(v, 10, 32); err == nil {
					f.ExcludeSellerId = uint32(excludeSellerId)
				}
			}
			if v := query.Get("page"); v != "" {
				if page, err := strconv.Atoi(v); err == nil {
					f.Page = page
				}
			}
			if v := query.Get("pageSize"); v != "" {
				if pageSize, err := strconv.Atoi(v); err == nil {
					f.PageSize = pageSize
				}
			}

			// Public browse only ever shows active listings; sold/cancelled/
			// expired listings are never surfaced here.
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Browse(worldId, StateActive, f)
			if err != nil {
				d.Logger().WithError(err).Errorf("Browsing listings for world [%d].", byte(worldId))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetListing(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseListingId(d.Logger(), func(listingId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(listingId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving listing [%s].", listingId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
