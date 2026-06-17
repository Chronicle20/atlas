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

// InitResource registers the read-only listing routes:
//   - GET /worlds/{worldId}/listings           — browse/search active listings
//   - GET /worlds/{worldId}/listings/{listingId} — listing detail
//
// The POST (create -> custody saga) and DELETE (cancel) routes are intentionally
// absent in this phase; they initiate custody flows that do not exist yet.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/worlds/{worldId}/listings").Subrouter()
			r.HandleFunc("", registerGet("browse_listings", handleBrowseListings)).Methods(http.MethodGet)
			r.HandleFunc("/{listingId}", registerGet("get_listing", handleGetListing)).Methods(http.MethodGet)
		}
	}
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
