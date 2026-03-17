package shop

import (
	"atlas-merchant/listing"
	"atlas-merchant/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)

			router.HandleFunc("/merchants", registerHandler("get_merchants", handleGetMerchants(db))).Methods(http.MethodGet)
			router.HandleFunc("/merchants/search/listings", registerHandler("search_listings", handleSearchListings(db))).Methods(http.MethodGet)

			r := router.PathPrefix("/merchants/{shopId}").Subrouter()
			r.HandleFunc("", registerHandler("get_merchant", handleGetMerchant(db))).Methods(http.MethodGet)
			r.HandleFunc("/relationships/listings", registerHandler("get_merchant_listings", handleGetMerchantListings(db))).Methods(http.MethodGet)

			cr := router.PathPrefix("/characters/{characterId}").Subrouter()
			cr.HandleFunc("/merchants", registerHandler("get_character_merchants", handleGetCharacterMerchants(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetMerchant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseShopId(d.Logger(), func(shopId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)

				m, err := p.GetById(shopId)
				if err != nil {
					if errors.Is(err, ErrNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Retrieving merchant.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				listings, err := p.GetListings(shopId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving listings.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				visitors, err := p.GetVisitors(shopId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving visitors.")
					visitors = nil
				}

				res, err := TransformWithListingsAndVisitors(listings, visitors)(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleGetMerchantListings(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseShopId(d.Logger(), func(shopId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)

				listings, err := p.GetListings(shopId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving listings.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(listing.Transform)(model.FixedProvider(listings))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST models.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]listing.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleGetMerchants(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), db)

			var shops []Model
			var err error

			mapIdStr := r.URL.Query().Get("mapId")
			if mapIdStr != "" {
				v, parseErr := strconv.ParseUint(mapIdStr, 10, 32)
				if parseErr != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				shops, err = p.GetByMapId(uint32(v))
			} else {
				shops, err = p.GetAllOpen()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving merchants.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			shopIds := make([]uuid.UUID, 0, len(shops))
			for _, s := range shops {
				shopIds = append(shopIds, s.Id())
			}

			counts, err := p.GetListingCounts(shopIds)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving listing counts.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(TransformWithListingCount(counts))(model.FixedProvider(shops))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST models.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleSearchListings(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			itemIdStr := r.URL.Query().Get("itemId")
			if itemIdStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			v, err := strconv.ParseUint(itemIdStr, 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), db)
			results, err := p.SearchListingsByItemId(uint32(v))
			if err != nil {
				d.Logger().WithError(err).Errorf("Searching listings by item.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(TransformSearchResult)(model.FixedProvider(results))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST models.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]ListingSearchRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetCharacterMerchants(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)
				shops, err := p.GetByCharacterId(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving merchants for character.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(shops))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST models.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
