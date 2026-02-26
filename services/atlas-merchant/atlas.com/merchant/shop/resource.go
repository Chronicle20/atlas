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

			router.HandleFunc("/merchants", registerHandler("get_merchants_by_map", handleGetMerchantsByMap(db))).Methods(http.MethodGet)

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

				res, err := TransformWithListings(listings)(m)
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

func handleGetMerchantsByMap(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mapIdStr := r.URL.Query().Get("mapId")
			if mapIdStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			v, err := strconv.ParseUint(mapIdStr, 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), db)
			shops, err := p.GetByMapId(uint32(v))
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving merchants by map.")
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
