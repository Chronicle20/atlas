package saved_location

import (
	"atlas-character/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/locations").Subrouter()
			r.HandleFunc("/{type}", registerGet("get_saved_location", handleGetSavedLocation)).Methods(http.MethodGet)
			r.HandleFunc("/{type}", rest.RegisterInputHandler[RestModel](l)(db)(si)("put_saved_location", handlePutSavedLocation)).Methods(http.MethodPut)
			r.HandleFunc("/{type}", registerGet("delete_saved_location", handleDeleteSavedLocation)).Methods(http.MethodDelete)
		}
	}
}

func parseLocationType(l logrus.FieldLogger, next func(locationType string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationType, ok := mux.Vars(r)["type"]
		if !ok || locationType == "" {
			l.Errorf("Unable to properly parse location type from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(locationType)(w, r)
	}
}

func handleGetSavedLocation(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return parseLocationType(d.Logger(), func(locationType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Get(characterId, locationType)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
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
	})
}

func handlePutSavedLocation(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return parseLocationType(d.Logger(), func(locationType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m := NewBuilder().
					SetCharacterId(characterId).
					SetLocationType(locationType).
					SetMapId(input.MapId).
					SetPortalId(input.PortalId).
					Build()

				result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Put(m)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(result))()
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
	})
}

func handleDeleteSavedLocation(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return parseLocationType(d.Logger(), func(locationType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), d.DB())

				m, err := p.Get(characterId, locationType)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				err = p.Delete(characterId, locationType)
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
	})
}
