package inventory

import (
	"atlas-inventory/rest"
	"errors"
	"net/http"

	"github.com/google/uuid"

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
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters/{characterId}/inventory").Subrouter()
			r.HandleFunc("", registerGet("get_inventory", handleGetInventory(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerGet("create_default_inventory", handleCreateInventory(db))).Methods(http.MethodPost)
			r.HandleFunc("", registerGet("delete_inventory", handleDeleteInventory(db))).Methods(http.MethodDelete)
		}
	}
}

func handleGetInventory(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					d.Logger().WithError(err).Errorf("Unable to retrieve characters inventory.")
					return
				}

				rm, err := model.Map(Transform)(model.FixedProvider(m))()
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
}

func handleCreateInventory(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(uuid.New(), characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to create inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := model.Map(Transform)(model.FixedProvider(m))()
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
}

func handleDeleteInventory(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(uuid.New(), characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to create inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		})
	}
}
