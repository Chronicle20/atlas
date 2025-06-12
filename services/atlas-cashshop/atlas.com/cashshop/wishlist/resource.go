package wishlist

import (
	"atlas-cashshop/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters/{characterId}/cash-shop/wishlist").Subrouter()
			r.HandleFunc("", registerGet("get_wishlist", handleGetWishlist(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("add_to_wishlist", handleAddToWishlist(db))).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterHandler(l)(si)("clear_wishlist", handleClearWishlist(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{itemId}", rest.RegisterHandler(l)(si)("remove_from_wishlist", handleRemoveFromWishlist(db))).Methods(http.MethodDelete)
		}
	}
}

func handleGetWishlist(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ms, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
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

func handleAddToWishlist(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).AddAndEmit(characterId, input.SerialNumber)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
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

func handleClearWishlist(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAllAndEmit(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleRemoveFromWishlist(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseItemId(d.Logger(), func(itemId uuid.UUID) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(characterId, itemId)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}
